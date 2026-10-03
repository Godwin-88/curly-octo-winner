package comms

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/comms/sms"
)

// Sender sends one text to many numbers. *sms.ATClient is the real one.
type Sender interface {
	SendBulk(ctx context.Context, req sms.BulkSMSRequest) ([]sms.SMSResult, error)
}

// SenderFactory returns the sender a school's messages go out through: its own
// Africa's Talking account when it has configured one, the platform's otherwise.
type SenderFactory func(ctx context.Context, tenantID uuid.UUID) (Sender, error)

// NotConfiguredError means there are no usable SMS credentials for the school.
type NotConfiguredError struct{ Reason string }

func (e *NotConfiguredError) Error() string { return e.Reason }

const (
	defaultChunkSize     = 100
	defaultSweepInterval = 30 * time.Second
	sendTimeout          = 45 * time.Second
)

// Dispatcher sends recorded messages. It is the only code that calls the SMS
// provider for a bulk message.
//
// Rules it keeps:
//
//   - It only ever sends to `pending` rows that have never been attempted.
//   - `attempted_at` is committed before the provider is called, and the call
//     happens outside any database transaction.
//   - A row found `pending` with `attempted_at` set (the process died between
//     the call and recording its result) is marked failed with
//     OUTCOME_UNKNOWN and is not sent again. Re-sending is a person's decision.
//   - One message is dispatched by one process at a time (advisory lock), so
//     a second instance or an overlapping sweep cannot double-send.
type Dispatcher struct {
	pool      *pgxpool.Pool
	senderFor SenderFactory
	chunkSize int
	interval  time.Duration
	wake      chan struct{}
}

// NewDispatcher creates a dispatcher. Call Run to start it.
func NewDispatcher(pool *pgxpool.Pool, senderFor SenderFactory) *Dispatcher {
	return &Dispatcher{
		pool:      pool,
		senderFor: senderFor,
		chunkSize: defaultChunkSize,
		interval:  defaultSweepInterval,
		wake:      make(chan struct{}, 1),
	}
}

// Notify asks for a sweep now instead of at the next tick. It never blocks.
func (d *Dispatcher) Notify() {
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

// Run sweeps until ctx is cancelled: on every Notify and every interval. The
// first sweep runs at once, which is what resumes work after a restart.
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	for {
		if err := d.Sweep(ctx); err != nil && ctx.Err() == nil {
			slog.Error("sms dispatch: sweep failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-d.wake:
		case <-ticker.C:
		}
	}
}

// Sweep starts every scheduled message that is due and dispatches every
// message that is `sending`.
func (d *Dispatcher) Sweep(ctx context.Context) error {
	if _, err := d.pool.Exec(ctx, `
		UPDATE messages
		SET status = 'sending', updated_at = now()
		WHERE status = 'scheduled' AND channel = 'sms' AND scheduled_at <= now()
	`); err != nil {
		return fmt.Errorf("start due messages: %w", err)
	}

	rows, err := d.pool.Query(ctx, `
		SELECT id, tenant_id FROM messages
		WHERE status = 'sending' AND channel = 'sms'
		ORDER BY created_at
	`)
	if err != nil {
		return fmt.Errorf("list messages to send: %w", err)
	}
	type job struct{ id, tenantID uuid.UUID }
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.id, &j.tenantID); err != nil {
			rows.Close()
			return fmt.Errorf("scan message to send: %w", err)
		}
		jobs = append(jobs, j)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, j := range jobs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := d.dispatch(ctx, j.tenantID, j.id); err != nil {
			slog.Error("sms dispatch: message failed", "message_id", j.id, "error", err)
		}
	}
	return nil
}

// lockKey derives the advisory-lock key for a message.
func lockKey(id uuid.UUID) int64 {
	return int64(binary.BigEndian.Uint64(id[:8]))
}

type pendingRow struct {
	id    uuid.UUID
	phone string
	text  string
}

// dispatch sends one message to completion.
func (d *Dispatcher) dispatch(ctx context.Context, tenantID, messageID uuid.UUID) error {
	// Hold the lock on a dedicated connection for the whole dispatch.
	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockKey(messageID)).Scan(&locked); err != nil {
		return fmt.Errorf("lock message: %w", err)
	}
	if !locked {
		return nil // another process is sending it
	}
	defer func() {
		// Unlock even when ctx is already cancelled (shutdown).
		unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_, _ = conn.Exec(unlockCtx, `SELECT pg_advisory_unlock($1)`, lockKey(messageID))
	}()

	// Rows a previous run attempted without recording a result.
	if _, err := d.pool.Exec(ctx, `
		UPDATE message_logs
		SET status = 'failed', error_code = 'OUTCOME_UNKNOWN',
		    error_message = 'Sending was interrupted before the result was recorded. This person may or may not have received the message.',
		    updated_at = now()
		WHERE message_id = $1 AND status = 'pending' AND attempted_at IS NOT NULL
	`, messageID); err != nil {
		return fmt.Errorf("resolve interrupted rows: %w", err)
	}

	sender, err := d.senderFor(ctx, tenantID)
	if err != nil {
		var notConfigured *NotConfiguredError
		if errors.As(err, &notConfigured) {
			if err := d.failRemaining(ctx, messageID, "NOT_CONFIGURED", notConfigured.Reason); err != nil {
				return err
			}
			return d.finish(ctx, messageID)
		}
		// Could not read the school's settings: leave the message for the next sweep.
		return fmt.Errorf("resolve sender: %w", err)
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err() // un-attempted rows stay pending and resume on restart
		}
		chunk, err := d.nextChunk(ctx, messageID)
		if err != nil {
			return err
		}
		if len(chunk) == 0 {
			break
		}

		// Personalised messages differ per recipient; identical texts share a call.
		var order []string
		groups := map[string][]pendingRow{}
		for _, row := range chunk {
			if _, ok := groups[row.text]; !ok {
				order = append(order, row.text)
			}
			groups[row.text] = append(groups[row.text], row)
		}
		for _, text := range order {
			stop, err := d.sendGroup(ctx, sender, messageID, text, groups[text])
			if err != nil {
				return err
			}
			if stop != nil {
				// The account itself was refused (no credit, sender id not
				// approved): every further call would fail the same way.
				if err := d.failRemaining(ctx, messageID, stop.code, stop.message); err != nil {
					return err
				}
				return d.finish(ctx, messageID)
			}
		}
	}
	return d.finish(ctx, messageID)
}

func (d *Dispatcher) nextChunk(ctx context.Context, messageID uuid.UUID) ([]pendingRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT l.id, l.phone, COALESCE(l.rendered_content, m.content)
		FROM message_logs l
		JOIN messages m ON m.id = l.message_id
		WHERE l.message_id = $1 AND l.status = 'pending' AND l.attempted_at IS NULL
		ORDER BY l.created_at, l.id
		LIMIT $2
	`, messageID, d.chunkSize)
	if err != nil {
		return nil, fmt.Errorf("load recipients: %w", err)
	}
	defer rows.Close()
	var chunk []pendingRow
	for rows.Next() {
		var row pendingRow
		if err := rows.Scan(&row.id, &row.phone, &row.text); err != nil {
			return nil, fmt.Errorf("scan recipient: %w", err)
		}
		chunk = append(chunk, row)
	}
	return chunk, rows.Err()
}

type accountStop struct{ code, message string }

// sendGroup sends one text to a group of recipients and records each result.
func (d *Dispatcher) sendGroup(ctx context.Context, sender Sender, messageID uuid.UUID, text string, group []pendingRow) (*accountStop, error) {
	ids := make([]uuid.UUID, len(group))
	phones := make([]string, len(group))
	for i, row := range group {
		ids[i] = row.id
		phones[i] = row.phone
	}

	// Record the attempt, committed, before the provider is called.
	if _, err := d.pool.Exec(ctx, `
		UPDATE message_logs SET attempted_at = now(), updated_at = now()
		WHERE id = ANY($1::uuid[])
	`, ids); err != nil {
		return nil, fmt.Errorf("record attempt: %w", err)
	}

	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	results, sendErr := sender.SendBulk(sendCtx, sms.BulkSMSRequest{To: phones, Message: text})
	cancel()

	// From here the provider has (or may have) acted: results are recorded
	// even if the server is shutting down.
	recordCtx, cancelRecord := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancelRecord()

	if sendErr != nil {
		code, message := "OUTCOME_UNKNOWN", "The SMS provider did not answer ("+sendErr.Error()+"). This person may or may not have received the message."
		var rejected *sms.RejectedError
		if errors.As(sendErr, &rejected) {
			code, message = "PROVIDER_REJECTED", rejected.Error()
		}
		if _, err := d.pool.Exec(recordCtx, `
			UPDATE message_logs
			SET status = 'failed', error_code = $2, error_message = $3, updated_at = now()
			WHERE id = ANY($1::uuid[])
		`, ids, code, message); err != nil {
			return nil, fmt.Errorf("record send failure: %w", err)
		}
		slog.Warn("sms dispatch: send failed", "message_id", messageID, "recipients", len(group), "code", code, "error", sendErr)
		if code == "PROVIDER_REJECTED" {
			// Credentials or the request itself were refused: the next call
			// would be refused too.
			return &accountStop{code: code, message: message}, nil
		}
		return nil, nil
	}

	byPhone := make(map[string]sms.SMSResult, len(results))
	for _, res := range results {
		byPhone[res.Phone] = res
	}

	var stop *accountStop
	accepted := 0
	for _, row := range group {
		res, ok := byPhone[row.phone]
		switch {
		case !ok:
			_, err := d.pool.Exec(recordCtx, `
				UPDATE message_logs
				SET status = 'failed', error_code = 'NO_RESULT',
				    error_message = 'The SMS provider returned no result for this number.', updated_at = now()
				WHERE id = $1
			`, row.id)
			if err != nil {
				return nil, fmt.Errorf("record missing result: %w", err)
			}
		case res.Accepted():
			accepted++
			_, err := d.pool.Exec(recordCtx, `
				UPDATE message_logs
				SET status = 'sent', sent_at = now(), provider_message_id = NULLIF($2, ''),
				    cost_cents = $3, error_code = NULL, error_message = NULL, updated_at = now()
				WHERE id = $1
			`, row.id, res.MessageID, res.CostCents())
			if err != nil {
				return nil, fmt.Errorf("record sent: %w", err)
			}
		default:
			detail := res.Detail
			if detail == "" {
				detail = "Refused by the SMS provider."
			}
			code := fmt.Sprintf("AT_%d", res.StatusCode)
			_, err := d.pool.Exec(recordCtx, `
				UPDATE message_logs
				SET status = 'failed', error_code = $2, error_message = $3, updated_at = now()
				WHERE id = $1
			`, row.id, code, detail)
			if err != nil {
				return nil, fmt.Errorf("record refusal: %w", err)
			}
			if res.AccountLevel() {
				stop = &accountStop{code: code, message: detail}
			}
		}
	}

	if _, err := d.pool.Exec(recordCtx, sms.RefreshCountsSQL, messageID); err != nil {
		return nil, fmt.Errorf("refresh counts: %w", err)
	}
	if accepted > 0 {
		// Some numbers went through, so the account is fine.
		stop = nil
	}
	return stop, nil
}

// failRemaining marks every not-yet-attempted recipient as failed.
func (d *Dispatcher) failRemaining(ctx context.Context, messageID uuid.UUID, code, message string) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	if _, err := d.pool.Exec(ctx, `
		UPDATE message_logs
		SET status = 'failed', error_code = $2, error_message = $3, updated_at = now()
		WHERE message_id = $1 AND status = 'pending' AND attempted_at IS NULL
	`, messageID, code, message); err != nil {
		return fmt.Errorf("fail remaining recipients: %w", err)
	}
	return nil
}

// finish closes a message once no recipient is pending: `sent` if the provider
// accepted at least one, `failed` otherwise.
func (d *Dispatcher) finish(ctx context.Context, messageID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	if _, err := d.pool.Exec(ctx, sms.RefreshCountsSQL, messageID); err != nil {
		return fmt.Errorf("refresh counts: %w", err)
	}
	if _, err := d.pool.Exec(ctx, `
		UPDATE messages m
		SET status = CASE WHEN EXISTS (
		                 SELECT 1 FROM message_logs l
		                 WHERE l.message_id = m.id AND l.status IN ('sent', 'delivered', 'read')
		             ) THEN 'sent' ELSE 'failed' END,
		    sent_at = now(),
		    updated_at = now()
		WHERE m.id = $1 AND m.status = 'sending'
		  AND NOT EXISTS (SELECT 1 FROM message_logs l WHERE l.message_id = m.id AND l.status = 'pending')
	`, messageID); err != nil {
		return fmt.Errorf("finish message: %w", err)
	}
	return nil
}
