package finance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/pkg/httputil"
	"github.com/shule360/api/pkg/pgxutil"
)

// Service handles finance domain operations.
//
// The rules it keeps:
//   - a learner is billed once per term;
//   - an invoice's status and totals follow from its payments and discounts,
//     and are never set by hand;
//   - money recorded is never deleted: an invoice is voided, a payment is
//     reversed, each with who, when and why;
//   - every confirmed payment has a receipt number, in sequence per school.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a finance service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ErrNotFound: the record does not exist in this school.
var ErrNotFound = errors.New("not found")

// ValidationError is a request the caller can correct.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

// ConflictError is a request that is well formed but contradicts what is
// already recorded.
type ConflictError struct{ Message string }

func (e *ConflictError) Error() string { return e.Message }

func conflict(format string, args ...any) error {
	return &ConflictError{Message: fmt.Sprintf(format, args...)}
}

// notFound turns the driver's "no rows" into ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// KES writes an amount in cents the way it is read aloud: "KES 17,500.00".
func KES(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	whole := fmt.Sprintf("%d", cents/100)
	var grouped []byte
	for i := 0; i < len(whole); i++ {
		if i > 0 && (len(whole)-i)%3 == 0 {
			grouped = append(grouped, ',')
		}
		grouped = append(grouped, whole[i])
	}
	return fmt.Sprintf("%sKES %s.%02d", sign, grouped, cents%100)
}

func trimmed(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// nextNumber takes the next running number for a school. It runs inside the
// caller's transaction, so a rolled-back payment leaves no gap in the sequence.
func nextNumber(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, kind string, year int) (int64, error) {
	var n int64
	err := tx.QueryRow(ctx, `
		INSERT INTO finance_counters (tenant_id, kind, year, last_value)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (tenant_id, kind, year)
		DO UPDATE SET last_value = finance_counters.last_value + 1
		RETURNING last_value`, tenantID, kind, year).Scan(&n)
	return n, err
}

// --- Fee structure operations ---

const feeStructureColumns = `id, tenant_id, name, grade, term, year, total_cents, active, notes, created_by, created_at, updated_at`

func scanFeeStructure(row pgx.Row) (*FeeStructure, error) {
	var fs FeeStructure
	err := row.Scan(
		&fs.ID, &fs.TenantID, &fs.Name, &fs.Grade, &fs.Term, &fs.Year,
		&fs.TotalCents, &fs.Active, &fs.Notes, &fs.CreatedBy, &fs.CreatedAt, &fs.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &fs, nil
}

const feeItemColumns = `id, tenant_id, fee_structure_id, name, amount_cents, item_type, is_optional, sort_order, created_at`

func scanFeeItem(row pgx.Row) (*FeeStructureItem, error) {
	var it FeeStructureItem
	err := row.Scan(
		&it.ID, &it.TenantID, &it.FeeStructureID, &it.Name, &it.AmountCents,
		&it.ItemType, &it.IsOptional, &it.SortOrder, &it.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// querier is what both the pool and a transaction can do.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func listFeeItems(ctx context.Context, q querier, tenantID, feeStructureID uuid.UUID) ([]FeeStructureItem, error) {
	query := fmt.Sprintf(`SELECT %s FROM fee_structure_items
		WHERE tenant_id = $1 AND fee_structure_id = $2 ORDER BY sort_order, created_at`, feeItemColumns)
	rows, err := q.Query(ctx, query, tenantID, feeStructureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []FeeStructureItem{}
	for rows.Next() {
		it, err := scanFeeItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *it)
	}
	return items, rows.Err()
}

// ListFeeStructures returns fee structures optionally filtered by grade/term/year.
func (s *Service) ListFeeStructures(ctx context.Context, tenantID uuid.UUID, grade string, term, year int) ([]FeeStructure, error) {
	query := fmt.Sprintf(`SELECT %s FROM fee_structures WHERE tenant_id = $1`, feeStructureColumns)
	args := []any{tenantID}

	if grade != "" {
		args = append(args, grade)
		query += fmt.Sprintf(` AND grade = $%d`, len(args))
	}
	if term > 0 {
		args = append(args, term)
		query += fmt.Sprintf(` AND term = $%d`, len(args))
	}
	if year > 0 {
		args = append(args, year)
		query += fmt.Sprintf(` AND year = $%d`, len(args))
	}
	query += ` ORDER BY year DESC, term DESC, grade`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	structures := []FeeStructure{}
	for rows.Next() {
		fs, err := scanFeeStructure(rows)
		if err != nil {
			return nil, err
		}
		structures = append(structures, *fs)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range structures {
		items, err := listFeeItems(ctx, s.pool, tenantID, structures[i].ID)
		if err != nil {
			return nil, err
		}
		structures[i].Items = items
	}
	return structures, nil
}

// GetFeeStructure returns a fee structure with items.
func (s *Service) GetFeeStructure(ctx context.Context, tenantID, id uuid.UUID) (*FeeStructure, error) {
	return getFeeStructure(ctx, s.pool, tenantID, id)
}

func getFeeStructure(ctx context.Context, q querier, tenantID, id uuid.UUID) (*FeeStructure, error) {
	query := fmt.Sprintf(`SELECT %s FROM fee_structures WHERE tenant_id = $1 AND id = $2`, feeStructureColumns)
	fs, err := scanFeeStructure(q.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return nil, notFound(err)
	}
	items, err := listFeeItems(ctx, q, tenantID, fs.ID)
	if err != nil {
		return nil, err
	}
	fs.Items = items
	return fs, nil
}

// checkItem validates one line of a fee structure. What it is called is up to
// the school: there is no fixed list of things a school may charge for.
func checkItem(item *FeeItemInput) error {
	item.Name = strings.Join(strings.Fields(item.Name), " ")
	if item.Name == "" {
		return invalid("Every fee item needs a name.")
	}
	if len(item.Name) > 100 {
		return invalid("%s… is too long a name for a fee item.", item.Name[:40])
	}
	if item.AmountCents <= 0 {
		return invalid("%s needs an amount above zero.", item.Name)
	}
	// item_type is kept for records made before schools named their own
	// items; new ones carry the name alone.
	if item.ItemType = strings.TrimSpace(item.ItemType); item.ItemType == "" || len(item.ItemType) > 100 {
		item.ItemType = "other"
	}
	return nil
}

const duplicateStructure = "This grade already has a fee structure for that term and year. Open it and change its items instead."

// CreateFeeStructure inserts a fee structure and its items in a transaction.
func (s *Service) CreateFeeStructure(ctx context.Context, tenantID uuid.UUID, actor *uuid.UUID, req CreateFeeStructureRequest) (*FeeStructure, error) {
	req.Name, req.Grade = strings.TrimSpace(req.Name), strings.TrimSpace(req.Grade)
	if req.Name == "" || req.Grade == "" {
		return nil, invalid("A fee structure needs a name and a grade.")
	}
	if req.Term < 1 || req.Term > 3 {
		return nil, invalid("Term must be 1, 2 or 3.")
	}
	if req.Year < 2000 || req.Year > 2100 {
		return nil, invalid("Enter the year in full, for example 2026.")
	}
	if len(req.Items) == 0 {
		return nil, invalid("Add at least one fee item.")
	}

	var total int64
	seen := map[string]bool{}
	for i := range req.Items {
		if err := checkItem(&req.Items[i]); err != nil {
			return nil, err
		}
		key := strings.ToLower(req.Items[i].Name)
		if seen[key] {
			return nil, invalid("%s is listed twice.", req.Items[i].Name)
		}
		seen[key] = true
		total += req.Items[i].AmountCents
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO fee_structures (tenant_id, name, grade, term, year, total_cents, active, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		tenantID, req.Name, req.Grade, req.Term, req.Year, total, active, trimmed(req.Notes), actor,
	).Scan(&id)
	if err != nil {
		if httputil.IsUniqueViolation(err) {
			return nil, conflict(duplicateStructure)
		}
		return nil, err
	}

	for idx, item := range req.Items {
		isOptional := item.IsOptional != nil && *item.IsOptional
		sortOrder := idx
		if item.SortOrder != nil {
			sortOrder = *item.SortOrder
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO fee_structure_items (tenant_id, fee_structure_id, name, amount_cents, item_type, is_optional, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			tenantID, id, item.Name, item.AmountCents, item.ItemType, isOptional, sortOrder,
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetFeeStructure(ctx, tenantID, id)
}

// UpdateFeeStructure partially updates a fee structure.
func (s *Service) UpdateFeeStructure(ctx context.Context, tenantID, id uuid.UUID, req UpdateFeeStructureRequest) (*FeeStructure, error) {
	if req.Term != nil && (*req.Term < 1 || *req.Term > 3) {
		return nil, invalid("Term must be 1, 2 or 3.")
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return nil, invalid("A fee structure needs a name.")
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE fee_structures SET
			name = COALESCE($3, name),
			grade = COALESCE($4, grade),
			term = COALESCE($5, term),
			year = COALESCE($6, year),
			active = COALESCE($7, active),
			notes = COALESCE($8, notes)
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, trimmed(req.Name), trimmed(req.Grade), req.Term, req.Year, req.Active, req.Notes,
	)
	if err != nil {
		if httputil.IsUniqueViolation(err) {
			return nil, conflict(duplicateStructure)
		}
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetFeeStructure(ctx, tenantID, id)
}

// DeleteFeeStructure removes a fee structure. Invoices already issued from it
// keep their own copy of the items, so they are unaffected.
func (s *Service) DeleteFeeStructure(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM fee_structures WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AddFeeItem adds an item to an existing fee structure.
func (s *Service) AddFeeItem(ctx context.Context, tenantID, structureID uuid.UUID, input FeeItemInput) (*FeeStructure, error) {
	if err := checkItem(&input); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var count int
	if err := tx.QueryRow(ctx, `
		SELECT (SELECT COUNT(*) FROM fee_structure_items WHERE fee_structure_id = fs.id)
		FROM fee_structures fs WHERE fs.tenant_id = $1 AND fs.id = $2 FOR UPDATE`,
		tenantID, structureID).Scan(&count); err != nil {
		return nil, notFound(err)
	}

	isOptional := input.IsOptional != nil && *input.IsOptional
	if _, err := tx.Exec(ctx, `
		INSERT INTO fee_structure_items (tenant_id, fee_structure_id, name, amount_cents, item_type, is_optional, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tenantID, structureID, input.Name, input.AmountCents, input.ItemType, isOptional, count,
	); err != nil {
		if httputil.IsUniqueViolation(err) {
			return nil, conflict("This fee structure already has an item called %s.", input.Name)
		}
		return nil, err
	}
	if err := recomputeFeeTotal(ctx, tx, tenantID, structureID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetFeeStructure(ctx, tenantID, structureID)
}

// DeleteFeeItem removes an item from a fee structure.
func (s *Service) DeleteFeeItem(ctx context.Context, tenantID, itemID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var structureID uuid.UUID
	if err := tx.QueryRow(ctx,
		`DELETE FROM fee_structure_items WHERE tenant_id = $1 AND id = $2 RETURNING fee_structure_id`,
		tenantID, itemID).Scan(&structureID); err != nil {
		return notFound(err)
	}
	if err := recomputeFeeTotal(ctx, tx, tenantID, structureID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// recomputeFeeTotal recalculates total_cents from the items as they now are.
func recomputeFeeTotal(ctx context.Context, tx pgx.Tx, tenantID, structureID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		UPDATE fee_structures fs SET total_cents = COALESCE((
			SELECT SUM(amount_cents) FROM fee_structure_items WHERE fee_structure_id = fs.id
		), 0)
		WHERE fs.tenant_id = $1 AND fs.id = $2`, tenantID, structureID)
	return err
}

// --- Invoice operations ---

// The status shown is "overdue" once the due date has passed and money is
// still owed; it is worked out when read, so it is right every day without a
// job that rewrites it.
const invoiceColumns = `i.id, i.tenant_id, i.learner_id, i.fee_structure_id, i.invoice_number,
	i.term, i.year, i.issue_date::text, i.due_date::text, i.total_cents, i.discount_cents,
	i.paid_cents,
	CASE WHEN i.status IN ('unpaid', 'partially_paid', 'overdue') AND i.due_date < CURRENT_DATE THEN 'overdue'
	     WHEN i.status = 'overdue' THEN CASE WHEN i.paid_cents > 0 THEN 'partially_paid' ELSE 'unpaid' END
	     ELSE i.status END,
	i.notes, i.created_by, i.created_at, i.updated_at, i.voided_at, i.void_reason,
	l.full_name, l.grade, COALESCE(l.stream, ''), COALESCE(l.upi, '')`

const invoiceFrom = ` FROM invoices i JOIN learners l ON l.id = i.learner_id `

func scanInvoice(row pgx.Row) (*Invoice, error) {
	var inv Invoice
	err := row.Scan(
		&inv.ID, &inv.TenantID, &inv.LearnerID, &inv.FeeStructureID, &inv.InvoiceNumber,
		&inv.Term, &inv.Year, &inv.IssueDate, &inv.DueDate, &inv.TotalCents, &inv.DiscountCents,
		&inv.PaidCents, &inv.Status, &inv.Notes, &inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.VoidedAt, &inv.VoidReason,
		&inv.LearnerName, &inv.Grade, &inv.Stream, &inv.LearnerUPI,
	)
	if err != nil {
		return nil, err
	}
	owed := inv.TotalCents - inv.DiscountCents - inv.PaidCents
	if inv.Status == "void" {
		owed = 0
	}
	if owed >= 0 {
		inv.BalanceCents = owed
	} else {
		inv.CreditCents = -owed
	}
	return &inv, nil
}

const invoiceItemColumns = `id, tenant_id, invoice_id, name, amount_cents, item_type, is_optional, sort_order, created_at`

func (s *Service) listInvoiceItems(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]InvoiceItem, error) {
	query := fmt.Sprintf(`SELECT %s FROM invoice_items
		WHERE tenant_id = $1 AND invoice_id = $2 ORDER BY sort_order, created_at`, invoiceItemColumns)
	rows, err := s.pool.Query(ctx, query, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []InvoiceItem{}
	for rows.Next() {
		var it InvoiceItem
		if err := rows.Scan(&it.ID, &it.TenantID, &it.InvoiceID, &it.Name, &it.AmountCents,
			&it.ItemType, &it.IsOptional, &it.SortOrder, &it.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// InvoiceFilter narrows a list of invoices.
type InvoiceFilter struct {
	// Status: a stored status, "overdue", or "open" for anything still owed.
	Status    string
	LearnerID string
	Grade     string
	Search    string
	Term      int
	Year      int
	Limit     int
	Offset    int
}

func pageBounds(limit, offset int) (int, int) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

const owing = `i.status IN ('unpaid', 'partially_paid', 'overdue')`

// ListInvoices returns invoices, newest first.
func (s *Service) ListInvoices(ctx context.Context, tenantID uuid.UUID, f InvoiceFilter) ([]Invoice, error) {
	query := `SELECT ` + invoiceColumns + invoiceFrom + ` WHERE i.tenant_id = $1`
	args := []any{tenantID}

	switch f.Status {
	case "":
	case "open":
		query += ` AND ` + owing
	case "overdue":
		query += ` AND ` + owing + ` AND i.due_date < CURRENT_DATE`
	case "unpaid", "partially_paid":
		args = append(args, f.Status)
		query += fmt.Sprintf(` AND i.status = $%d AND (i.due_date IS NULL OR i.due_date >= CURRENT_DATE)`, len(args))
	default:
		args = append(args, f.Status)
		query += fmt.Sprintf(` AND i.status = $%d`, len(args))
	}
	if f.LearnerID != "" {
		id, err := uuid.Parse(f.LearnerID)
		if err != nil {
			return nil, invalid("That is not a learner.")
		}
		args = append(args, id)
		query += fmt.Sprintf(` AND i.learner_id = $%d`, len(args))
	}
	if f.Grade != "" {
		args = append(args, f.Grade)
		query += fmt.Sprintf(` AND l.grade = $%d`, len(args))
	}
	if search := strings.TrimSpace(f.Search); search != "" {
		args = append(args, "%"+search+"%")
		query += fmt.Sprintf(` AND (l.full_name ILIKE $%d OR i.invoice_number ILIKE $%d OR l.upi ILIKE $%d)`, len(args), len(args), len(args))
	}
	if f.Term > 0 {
		args = append(args, f.Term)
		query += fmt.Sprintf(` AND i.term = $%d`, len(args))
	}
	if f.Year > 0 {
		args = append(args, f.Year)
		query += fmt.Sprintf(` AND i.year = $%d`, len(args))
	}
	limit, offset := pageBounds(f.Limit, f.Offset)
	args = append(args, limit, offset)
	query += fmt.Sprintf(` ORDER BY i.created_at DESC, i.invoice_number DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invoices := []Invoice{}
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, *inv)
	}
	return invoices, rows.Err()
}

// GetInvoice returns an invoice with items.
func (s *Service) GetInvoice(ctx context.Context, tenantID, id uuid.UUID) (*Invoice, error) {
	inv, err := scanInvoice(s.pool.QueryRow(ctx,
		`SELECT `+invoiceColumns+invoiceFrom+` WHERE i.tenant_id = $1 AND i.id = $2`, tenantID, id))
	if err != nil {
		return nil, notFound(err)
	}
	items, err := s.listInvoiceItems(ctx, tenantID, inv.ID)
	if err != nil {
		return nil, err
	}
	inv.Items = items
	return inv, nil
}

func checkDate(label string, value *string) (*string, error) {
	value = trimmed(value)
	if value == nil {
		return nil, nil
	}
	// A date-time from a form is accepted for its date.
	day := *value
	if len(day) > 10 {
		day = day[:10]
	}
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return nil, invalid("%s is not a date.", label)
	}
	return &day, nil
}

// billable is the part of a fee structure that goes on an invoice.
func billable(fs *FeeStructure, includeOptional bool) []FeeItemInput {
	var items []FeeItemInput
	for _, it := range fs.Items {
		if it.IsOptional && !includeOptional {
			continue
		}
		optional, order := it.IsOptional, it.SortOrder
		items = append(items, FeeItemInput{
			Name: it.Name, AmountCents: it.AmountCents, ItemType: it.ItemType,
			IsOptional: &optional, SortOrder: &order,
		})
	}
	return items
}

type newInvoice struct {
	learnerID      uuid.UUID
	feeStructureID *uuid.UUID
	term, year     int
	dueDate        *string
	notes          *string
	items          []FeeItemInput
}

// insertInvoice writes one invoice and its items. created is false when the
// learner already has a live invoice for that term: nothing was written.
func insertInvoice(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, actor *uuid.UUID, in newInvoice) (id uuid.UUID, created bool, err error) {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM invoices
			WHERE tenant_id = $1 AND learner_id = $2 AND term = $3 AND year = $4 AND status <> 'void')`,
		tenantID, in.learnerID, in.term, in.year).Scan(&exists); err != nil {
		return uuid.Nil, false, err
	}
	if exists {
		return uuid.Nil, false, nil
	}

	var total int64
	for _, it := range in.items {
		total += it.AmountCents
	}
	n, err := nextNumber(ctx, tx, tenantID, "invoice", in.year)
	if err != nil {
		return uuid.Nil, false, err
	}
	number := fmt.Sprintf("INV-%d-%05d", in.year, n)

	// The unique index is the guard against two requests billing the same
	// learner at once; the check above only saves a number in the usual case.
	err = tx.QueryRow(ctx, `
		INSERT INTO invoices (tenant_id, learner_id, fee_structure_id, invoice_number, term, year, due_date, total_cents, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9, $10)
		ON CONFLICT (tenant_id, learner_id, term, year) WHERE status <> 'void' DO NOTHING
		RETURNING id`,
		tenantID, in.learnerID, in.feeStructureID, number, in.term, in.year, in.dueDate, total, in.notes, actor,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}

	for idx, it := range in.items {
		isOptional := it.IsOptional != nil && *it.IsOptional
		sortOrder := idx
		if it.SortOrder != nil {
			sortOrder = *it.SortOrder
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO invoice_items (tenant_id, invoice_id, name, amount_cents, item_type, is_optional, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			tenantID, id, it.Name, it.AmountCents, it.ItemType, isOptional, sortOrder,
		); err != nil {
			return uuid.Nil, false, err
		}
	}
	return id, true, nil
}

// CreateInvoice bills one learner for a term, from a fee structure or from
// items given directly.
func (s *Service) CreateInvoice(ctx context.Context, tenantID uuid.UUID, actor *uuid.UUID, req CreateInvoiceRequest) (*Invoice, error) {
	if req.LearnerID == uuid.Nil {
		return nil, invalid("Choose the learner to bill.")
	}
	dueDate, err := checkDate("The due date", req.DueDate)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var learnerGrade, learnerName string
	var active bool
	if err := tx.QueryRow(ctx,
		`SELECT grade, full_name, is_active FROM learners WHERE tenant_id = $1 AND id = $2`,
		tenantID, req.LearnerID).Scan(&learnerGrade, &learnerName, &active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, invalid("That learner is not in this school.")
		}
		return nil, err
	}
	if !active {
		return nil, invalid("%s is no longer an active learner.", learnerName)
	}

	in := newInvoice{learnerID: req.LearnerID, term: req.Term, year: req.Year, dueDate: dueDate, notes: trimmed(req.Notes)}
	if req.FeeStructureID != nil && *req.FeeStructureID != uuid.Nil {
		fs, err := getFeeStructure(ctx, tx, tenantID, *req.FeeStructureID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, invalid("That fee structure is not in this school.")
			}
			return nil, err
		}
		if fs.Grade != learnerGrade {
			return nil, invalid("%s is in %s; this fee structure is for %s.", learnerName, learnerGrade, fs.Grade)
		}
		if !fs.Active {
			return nil, invalid("%s is switched off. Make it active before billing from it.", fs.Name)
		}
		// The term and year are the fee structure's: billing Term 1 fees as a
		// Term 2 invoice would defeat the once-per-term rule.
		in.feeStructureID, in.term, in.year = &fs.ID, fs.Term, fs.Year
		in.items = billable(fs, req.IncludeOptional)
	} else {
		if req.Term < 1 || req.Term > 3 {
			return nil, invalid("Term must be 1, 2 or 3.")
		}
		if req.Year < 2000 || req.Year > 2100 {
			return nil, invalid("Enter the year in full, for example 2026.")
		}
		for i := range req.Items {
			if err := checkItem(&req.Items[i]); err != nil {
				return nil, err
			}
		}
		in.items = req.Items
	}
	if len(in.items) == 0 {
		return nil, invalid("There is nothing to bill: the invoice has no fee items.")
	}

	id, created, err := insertInvoice(ctx, tx, tenantID, actor, in)
	if err != nil {
		return nil, err
	}
	if !created {
		return nil, conflict("%s already has an invoice for Term %d %d. Open that one, or void it first.", learnerName, in.term, in.year)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetInvoice(ctx, tenantID, id)
}

// BulkInvoice bills every active learner of the fee structure's grade who has
// not been billed for that term yet. Run twice, it creates nothing the second
// time.
func (s *Service) BulkInvoice(ctx context.Context, tenantID uuid.UUID, actor *uuid.UUID, req BulkInvoiceRequest) (*BulkInvoiceResult, error) {
	dueDate, err := checkDate("The due date", req.DueDate)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	fs, err := getFeeStructure(ctx, tx, tenantID, req.FeeStructureID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, invalid("Choose the fee structure to bill from.")
		}
		return nil, err
	}
	if !fs.Active {
		return nil, invalid("%s is switched off. Make it active before billing from it.", fs.Name)
	}
	items := billable(fs, req.IncludeOptional)
	if len(items) == 0 {
		return nil, invalid("%s has no items to bill.", fs.Name)
	}
	var each int64
	for _, it := range items {
		each += it.AmountCents
	}

	query := `SELECT id FROM learners WHERE tenant_id = $1 AND grade = $2 AND is_active = true`
	args := []any{tenantID, fs.Grade}
	if stream := strings.TrimSpace(req.Stream); stream != "" {
		args = append(args, stream)
		query += ` AND stream = $3`
	}
	rows, err := tx.Query(ctx, query+` ORDER BY full_name`, args...)
	if err != nil {
		return nil, err
	}
	var learners []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		learners = append(learners, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &BulkInvoiceResult{Learners: len(learners), EachCents: each}
	if req.DryRun {
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM invoices
			WHERE tenant_id = $1 AND term = $2 AND year = $3 AND status <> 'void' AND learner_id = ANY($4::uuid[])`,
			tenantID, fs.Term, fs.Year, pgxutil.UUIDArray(learners)).Scan(&result.AlreadyInvoiced); err != nil {
			return nil, err
		}
		result.Created = result.Learners - result.AlreadyInvoiced
		result.TotalCents = int64(result.Created) * each
		return result, nil
	}

	for _, learnerID := range learners {
		_, created, err := insertInvoice(ctx, tx, tenantID, actor, newInvoice{
			learnerID: learnerID, feeStructureID: &fs.ID, term: fs.Term, year: fs.Year,
			dueDate: dueDate, items: items,
		})
		if err != nil {
			return nil, err
		}
		if created {
			result.Created++
		} else {
			result.AlreadyInvoiced++
		}
	}
	result.TotalCents = int64(result.Created) * each
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateInvoice changes the due date or notes of an invoice.
func (s *Service) UpdateInvoice(ctx context.Context, tenantID, id uuid.UUID, req UpdateInvoiceRequest) (*Invoice, error) {
	dueDate, err := checkDate("The due date", req.DueDate)
	if err != nil {
		return nil, err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE invoices SET
			due_date = COALESCE($3::date, due_date),
			notes = COALESCE($4, notes)
		WHERE tenant_id = $1 AND id = $2 AND status <> 'void'`,
		tenantID, id, dueDate, req.Notes)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.GetInvoice(ctx, tenantID, id); err != nil {
			return nil, err
		}
		return nil, conflict("A voided invoice cannot be changed.")
	}
	return s.GetInvoice(ctx, tenantID, id)
}

// VoidInvoice cancels an invoice that was raised in error. An invoice with
// money against it cannot be voided: reverse the payments first, so the
// record shows where the money went.
func (s *Service) VoidInvoice(ctx context.Context, tenantID uuid.UUID, actor *uuid.UUID, id uuid.UUID, reason string) (*Invoice, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, invalid("Say why this invoice is being voided.")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var status string
	if err := tx.QueryRow(ctx,
		`SELECT status FROM invoices WHERE tenant_id = $1 AND id = $2 FOR UPDATE`, tenantID, id).Scan(&status); err != nil {
		return nil, notFound(err)
	}
	if status == "void" {
		return nil, conflict("This invoice is already void.")
	}
	var completed, pending int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE status = 'completed'), COUNT(*) FILTER (WHERE status = 'pending')
		FROM payments WHERE tenant_id = $1 AND invoice_id = $2`, tenantID, id).Scan(&completed, &pending); err != nil {
		return nil, err
	}
	if completed > 0 {
		return nil, conflict("This invoice has %d payment(s) against it. Reverse them before voiding it.", completed)
	}
	if pending > 0 {
		return nil, conflict("An M-Pesa request for this invoice is still waiting for the parent. Try again in a few minutes.")
	}

	if _, err := tx.Exec(ctx, `
		UPDATE invoices SET status = 'void', voided_at = now(), voided_by = $3, void_reason = $4
		WHERE tenant_id = $1 AND id = $2`, tenantID, id, actor, reason); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetInvoice(ctx, tenantID, id)
}

// --- Discount operations ---

const discountColumns = `id, tenant_id, invoice_id, amount_cents, discount_type, reason, approved_by, created_at`

var discountTypes = map[string]bool{"scholarship": true, "sibling": true, "waiver": true, "other": true}

func scanDiscount(row pgx.Row) (*Discount, error) {
	var d Discount
	err := row.Scan(&d.ID, &d.TenantID, &d.InvoiceID, &d.AmountCents, &d.DiscountType, &d.Reason, &d.ApprovedBy, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ListDiscounts returns discounts for an invoice.
func (s *Service) ListDiscounts(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]Discount, error) {
	query := fmt.Sprintf(`SELECT %s FROM discounts WHERE tenant_id = $1 AND invoice_id = $2 ORDER BY created_at`, discountColumns)
	rows, err := s.pool.Query(ctx, query, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	discounts := []Discount{}
	for rows.Next() {
		d, err := scanDiscount(rows)
		if err != nil {
			return nil, err
		}
		discounts = append(discounts, *d)
	}
	return discounts, rows.Err()
}

// lockInvoice locks an invoice for the rest of the transaction and returns
// its status and what is still owed on it.
func lockInvoice(ctx context.Context, tx pgx.Tx, tenantID, invoiceID uuid.UUID) (status string, balance int64, err error) {
	err = tx.QueryRow(ctx, `
		SELECT status, total_cents - discount_cents - paid_cents
		FROM invoices WHERE tenant_id = $1 AND id = $2 FOR UPDATE`, tenantID, invoiceID).Scan(&status, &balance)
	return status, balance, notFound(err)
}

// CreateDiscount takes an amount off an invoice. The signed-in user is
// recorded as the approver.
func (s *Service) CreateDiscount(ctx context.Context, tenantID uuid.UUID, actor *uuid.UUID, invoiceID uuid.UUID, req CreateDiscountRequest) (*Discount, error) {
	if req.AmountCents <= 0 {
		return nil, invalid("Enter the amount to take off.")
	}
	if req.DiscountType == "" {
		req.DiscountType = "other"
	}
	if !discountTypes[req.DiscountType] {
		return nil, invalid("%s is not a kind of discount.", req.DiscountType)
	}
	reason := trimmed(req.Reason)
	if reason == nil {
		return nil, invalid("Say why this discount is given.")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	status, balance, err := lockInvoice(ctx, tx, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	if status == "void" {
		return nil, conflict("A voided invoice cannot be discounted.")
	}
	if req.AmountCents > balance {
		return nil, invalid("Only %s is still owed on this invoice; the discount cannot be more than that.", KES(max(balance, 0)))
	}

	query := fmt.Sprintf(`INSERT INTO discounts (tenant_id, invoice_id, amount_cents, discount_type, reason, approved_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING %s`, discountColumns)
	d, err := scanDiscount(tx.QueryRow(ctx, query, tenantID, invoiceID, req.AmountCents, req.DiscountType, reason, actor))
	if err != nil {
		return nil, err
	}
	if err := refreshInvoiceFinance(ctx, tx, tenantID, invoiceID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

// DeleteDiscount removes a discount and refreshes the invoice.
func (s *Service) DeleteDiscount(ctx context.Context, tenantID, discountID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var invoiceID uuid.UUID
	if err := tx.QueryRow(ctx,
		`SELECT invoice_id FROM discounts WHERE tenant_id = $1 AND id = $2`, tenantID, discountID).Scan(&invoiceID); err != nil {
		return notFound(err)
	}
	status, _, err := lockInvoice(ctx, tx, tenantID, invoiceID)
	if err != nil {
		return err
	}
	if status == "void" {
		return conflict("A voided invoice cannot be changed.")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM discounts WHERE tenant_id = $1 AND id = $2`, tenantID, discountID); err != nil {
		return err
	}
	if err := refreshInvoiceFinance(ctx, tx, tenantID, invoiceID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// refreshInvoiceFinance recomputes discount_cents, paid_cents and status for
// an invoice from its discounts and completed payments, the only source of
// truth for them. It must be called inside a transaction.
func refreshInvoiceFinance(ctx context.Context, tx pgx.Tx, tenantID, invoiceID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		UPDATE invoices i SET
			discount_cents = t.discounts,
			paid_cents = t.paid,
			status = CASE
				WHEN i.status = 'void' THEN 'void'
				WHEN t.paid >= i.total_cents - t.discounts THEN 'paid'
				WHEN t.paid > 0 THEN 'partially_paid'
				ELSE 'unpaid'
			END
		FROM (
			SELECT
				COALESCE((SELECT SUM(amount_cents) FROM discounts WHERE tenant_id = $1 AND invoice_id = $2), 0) AS discounts,
				COALESCE((SELECT SUM(amount_cents) FROM payments WHERE tenant_id = $1 AND invoice_id = $2 AND status = 'completed'), 0) AS paid
		) t
		WHERE i.tenant_id = $1 AND i.id = $2`, tenantID, invoiceID)
	return err
}

// --- Payment operations ---

const paymentColumns = `p.id, p.tenant_id, p.invoice_id, p.amount_cents, p.channel, p.status,
	p.reference, p.paid_by, p.phone, p.paid_at, p.received_by, p.notes,
	p.checkout_request_id, p.merchant_request_id, p.mpesa_receipt, p.mpesa_result_code, p.mpesa_result_desc,
	p.created_at, p.updated_at,
	p.receipt_number, p.failure_code, p.reversed_at, p.reversal_reason,
	i.invoice_number, i.learner_id, l.full_name, l.grade`

const paymentFrom = ` FROM payments p
	JOIN invoices i ON i.id = p.invoice_id
	JOIN learners l ON l.id = i.learner_id `

func scanPayment(row pgx.Row) (*Payment, error) {
	var pay Payment
	err := row.Scan(
		&pay.ID, &pay.TenantID, &pay.InvoiceID, &pay.AmountCents, &pay.Channel, &pay.Status,
		&pay.Reference, &pay.PaidBy, &pay.Phone, &pay.PaidAt, &pay.ReceivedBy, &pay.Notes,
		&pay.CheckoutRequestID, &pay.MerchantRequestID, &pay.MpesaReceipt, &pay.MpesaResultCode, &pay.MpesaResultDesc,
		&pay.CreatedAt, &pay.UpdatedAt,
		&pay.ReceiptNumber, &pay.FailureCode, &pay.ReversedAt, &pay.ReversalReason,
		&pay.InvoiceNumber, &pay.LearnerID, &pay.LearnerName, &pay.Grade,
	)
	if err != nil {
		return nil, err
	}
	return &pay, nil
}

func scanPayments(rows pgx.Rows) ([]Payment, error) {
	defer rows.Close()
	payments := []Payment{}
	for rows.Next() {
		pay, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, *pay)
	}
	return payments, rows.Err()
}

// PaymentFilter narrows a list of payments.
type PaymentFilter struct {
	Status  string
	Channel string
	Search  string
	Term    int
	Year    int
	Limit   int
	Offset  int
}

// ListPayments returns payments, newest first.
func (s *Service) ListPayments(ctx context.Context, tenantID uuid.UUID, f PaymentFilter) ([]Payment, error) {
	query := `SELECT ` + paymentColumns + paymentFrom + ` WHERE p.tenant_id = $1`
	args := []any{tenantID}

	if f.Status != "" {
		args = append(args, f.Status)
		query += fmt.Sprintf(` AND p.status = $%d`, len(args))
	}
	if f.Channel != "" {
		args = append(args, f.Channel)
		query += fmt.Sprintf(` AND p.channel = $%d`, len(args))
	}
	if search := strings.TrimSpace(f.Search); search != "" {
		args = append(args, "%"+search+"%")
		n := len(args)
		query += fmt.Sprintf(` AND (l.full_name ILIKE $%d OR p.receipt_number ILIKE $%d OR p.reference ILIKE $%d
			OR p.mpesa_receipt ILIKE $%d OR i.invoice_number ILIKE $%d)`, n, n, n, n, n)
	}
	if f.Term > 0 {
		args = append(args, f.Term)
		query += fmt.Sprintf(` AND i.term = $%d`, len(args))
	}
	if f.Year > 0 {
		args = append(args, f.Year)
		query += fmt.Sprintf(` AND i.year = $%d`, len(args))
	}
	limit, offset := pageBounds(f.Limit, f.Offset)
	args = append(args, limit, offset)
	query += fmt.Sprintf(` ORDER BY p.created_at DESC, p.id LIMIT $%d OFFSET $%d`, len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return scanPayments(rows)
}

// ListInvoicePayments returns all payments for an invoice.
func (s *Service) ListInvoicePayments(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]Payment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+paymentColumns+paymentFrom+` WHERE p.tenant_id = $1 AND p.invoice_id = $2 ORDER BY p.created_at DESC`,
		tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	return scanPayments(rows)
}

// GetPayment returns a single payment.
func (s *Service) GetPayment(ctx context.Context, tenantID, id uuid.UUID) (*Payment, error) {
	pay, err := scanPayment(s.pool.QueryRow(ctx,
		`SELECT `+paymentColumns+paymentFrom+` WHERE p.tenant_id = $1 AND p.id = $2`, tenantID, id))
	return pay, notFound(err)
}

// paymentByKey returns the payment an earlier request with the same
// idempotency key created, if there is one.
func (s *Service) paymentByKey(ctx context.Context, tenantID uuid.UUID, key string) (*Payment, error) {
	if key == "" {
		return nil, nil
	}
	pay, err := scanPayment(s.pool.QueryRow(ctx,
		`SELECT `+paymentColumns+paymentFrom+` WHERE p.tenant_id = $1 AND p.idempotency_key = $2`, tenantID, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return pay, err
}

// receiptNumber takes the next receipt number for a school.
func receiptNumber(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, at time.Time) (string, error) {
	year := at.In(nairobi).Year()
	n, err := nextNumber(ctx, tx, tenantID, "receipt", year)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("RCT-%d-%05d", year, n), nil
}

var nairobi = time.FixedZone("EAT", 3*60*60)

var manualChannels = map[string]string{"cash": "cash", "bank": "bank", "cheque": "cheque", "mpesa": "M-Pesa"}

// CreatePayment records money the school has already received: cash, a bank
// deposit, a cheque, or an M-Pesa payment entered from its confirmation code.
// It is completed at once, with a receipt number.
func (s *Service) CreatePayment(ctx context.Context, tenantID uuid.UUID, actor *uuid.UUID, req CreatePaymentRequest) (*Payment, error) {
	if req.InvoiceID == uuid.Nil {
		return nil, invalid("Choose the invoice this payment is for.")
	}
	if req.AmountCents <= 0 {
		return nil, invalid("Enter the amount received.")
	}
	if req.Channel == "" {
		req.Channel = "cash"
	}
	label, ok := manualChannels[req.Channel]
	if !ok {
		return nil, invalid("%s is not a way of paying.", req.Channel)
	}
	reference := trimmed(req.Reference)
	if reference != nil {
		upper := strings.ToUpper(*reference)
		reference = &upper
	}
	if req.Channel != "cash" && reference == nil {
		return nil, invalid("Enter the %s reference, so this payment can be traced.", label)
	}
	paidAt := time.Now()
	if req.PaidAt != nil {
		if req.PaidAt.After(paidAt.Add(5 * time.Minute)) {
			return nil, invalid("The payment date is in the future.")
		}
		paidAt = *req.PaidAt
	}

	if existing, err := s.paymentByKey(ctx, tenantID, req.IdempotencyKey); err != nil || existing != nil {
		return existing, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	status, balance, err := lockInvoice(ctx, tx, tenantID, req.InvoiceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, invalid("That invoice is not in this school.")
		}
		return nil, err
	}
	if status == "void" {
		return nil, conflict("This invoice is void; a payment cannot be recorded against it.")
	}
	if balance <= 0 {
		return nil, conflict("This invoice is already paid in full.")
	}
	if req.AmountCents > balance {
		return nil, invalid("Only %s is still owed on this invoice. Record at most that; put the rest against the learner's next invoice.", KES(balance))
	}

	if reference != nil {
		var receipt *string
		err := tx.QueryRow(ctx, `
			SELECT receipt_number FROM payments
			WHERE tenant_id = $1 AND status = 'completed' AND channel = $2
			  AND (upper(reference) = $3 OR upper(mpesa_receipt) = $3)
			LIMIT 1`, tenantID, req.Channel, *reference).Scan(&receipt)
		if err == nil {
			return nil, conflict("%s reference %s is already recorded%s.", label, *reference, receiptSuffix(receipt))
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if req.Channel == "mpesa" {
			var inInbox bool
			if err := tx.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM mpesa_inbox WHERE upper(trans_id) = $1)`, *reference).Scan(&inInbox); err != nil {
				return nil, err
			}
			if inInbox {
				return nil, conflict("M-Pesa payment %s has already arrived from Safaricom. Allocate it under Paybill payments instead of entering it by hand.", *reference)
			}
		}
	}

	receipt, err := receiptNumber(ctx, tx, tenantID, paidAt)
	if err != nil {
		return nil, err
	}
	var key *string
	if req.IdempotencyKey != "" {
		key = &req.IdempotencyKey
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO payments (tenant_id, invoice_id, amount_cents, channel, status, reference, paid_by, phone,
			paid_at, received_by, notes, receipt_number, idempotency_key)
		VALUES ($1, $2, $3, $4, 'completed', $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		tenantID, req.InvoiceID, req.AmountCents, req.Channel, reference, trimmed(req.PaidBy), trimmed(req.Phone),
		paidAt, actor, trimmed(req.Notes), receipt, key,
	).Scan(&id)
	if err != nil {
		if httputil.IsUniqueViolation(err) {
			// Two submissions of the same form raced; the other one won.
			if existing, lookupErr := s.paymentByKey(ctx, tenantID, req.IdempotencyKey); lookupErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}

	if err := refreshInvoiceFinance(ctx, tx, tenantID, req.InvoiceID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetPayment(ctx, tenantID, id)
}

func receiptSuffix(receipt *string) string {
	if receipt == nil || *receipt == "" {
		return ""
	}
	return " as receipt " + *receipt
}

// ReversePayment undoes a confirmed payment: a bounced cheque, money entered
// against the wrong learner. The payment stays on record as reversed, with
// who reversed it and why, and the invoice is owed again.
func (s *Service) ReversePayment(ctx context.Context, tenantID uuid.UUID, actor *uuid.UUID, id uuid.UUID, reason string) (*Payment, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, invalid("Say why this payment is being reversed.")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var invoiceID uuid.UUID
	var status string
	var inboxID *uuid.UUID
	var amount int64
	if err := tx.QueryRow(ctx, `
		SELECT invoice_id, status, inbox_id, amount_cents FROM payments
		WHERE tenant_id = $1 AND id = $2 FOR UPDATE`, tenantID, id).Scan(&invoiceID, &status, &inboxID, &amount); err != nil {
		return nil, notFound(err)
	}
	switch status {
	case "completed":
	case "reversed":
		return nil, conflict("This payment has already been reversed.")
	default:
		return nil, conflict("Only a confirmed payment can be reversed; this one is %s.", status)
	}
	if _, _, err := lockInvoice(ctx, tx, tenantID, invoiceID); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE payments SET status = 'reversed', reversed_at = now(), reversed_by = $3, reversal_reason = $4
		WHERE tenant_id = $1 AND id = $2`, tenantID, id, actor, reason); err != nil {
		return nil, err
	}
	// Money that came from the paybill goes back to be allocated again: it was
	// received, whatever invoice it was first put against.
	if inboxID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE mpesa_inbox SET
				allocated_cents = allocated_cents - $3,
				status = CASE WHEN allocated_cents - $3 <= 0 THEN 'unmatched' ELSE 'part_allocated' END
			WHERE tenant_id = $1 AND id = $2`, tenantID, *inboxID, amount); err != nil {
			return nil, err
		}
	}
	if err := refreshInvoiceFinance(ctx, tx, tenantID, invoiceID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetPayment(ctx, tenantID, id)
}

// --- Position: arrears, statement, summary ---

// Arrears lists the learners who still owe fees, largest balance first.
func (s *Service) Arrears(ctx context.Context, tenantID uuid.UUID, grade, search string, limit, offset int) ([]ArrearsRow, error) {
	query := `
		SELECT l.id, l.full_name, l.grade, COALESCE(l.stream, ''), COUNT(i.id),
		       SUM(i.total_cents - i.discount_cents - i.paid_cents), MIN(i.due_date)::text
		FROM invoices i JOIN learners l ON l.id = i.learner_id
		WHERE i.tenant_id = $1 AND ` + owing + ` AND i.total_cents - i.discount_cents - i.paid_cents > 0`
	args := []any{tenantID}
	if grade != "" {
		args = append(args, grade)
		query += fmt.Sprintf(` AND l.grade = $%d`, len(args))
	}
	if search = strings.TrimSpace(search); search != "" {
		args = append(args, "%"+search+"%")
		query += fmt.Sprintf(` AND l.full_name ILIKE $%d`, len(args))
	}
	limit, offset = pageBounds(limit, offset)
	args = append(args, limit, offset)
	query += fmt.Sprintf(` GROUP BY l.id, l.full_name, l.grade, l.stream
		ORDER BY 6 DESC, l.full_name LIMIT $%d OFFSET $%d`, len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ArrearsRow{}
	for rows.Next() {
		var r ArrearsRow
		if err := rows.Scan(&r.LearnerID, &r.LearnerName, &r.Grade, &r.Stream, &r.Invoices, &r.BalanceCents, &r.OldestDue); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Statement returns everything billed to and received for one learner.
func (s *Service) Statement(ctx context.Context, tenantID, learnerID uuid.UUID) (*Statement, error) {
	st := &Statement{LearnerID: learnerID}
	if err := s.pool.QueryRow(ctx,
		`SELECT full_name, grade, COALESCE(stream, '') FROM learners WHERE tenant_id = $1 AND id = $2`,
		tenantID, learnerID).Scan(&st.LearnerName, &st.Grade, &st.Stream); err != nil {
		return nil, notFound(err)
	}

	invoices, err := s.ListInvoices(ctx, tenantID, InvoiceFilter{LearnerID: learnerID.String(), Limit: 200})
	if err != nil {
		return nil, err
	}
	st.Invoices = invoices
	for _, inv := range invoices {
		if inv.Status == "void" {
			continue
		}
		st.BilledCents += inv.TotalCents - inv.DiscountCents
		st.PaidCents += inv.PaidCents
	}
	st.BalanceCents = st.BilledCents - st.PaidCents

	rows, err := s.pool.Query(ctx,
		`SELECT `+paymentColumns+paymentFrom+`
		 WHERE p.tenant_id = $1 AND i.learner_id = $2 AND p.status IN ('completed', 'reversed')
		 ORDER BY p.paid_at DESC NULLS LAST, p.created_at DESC`, tenantID, learnerID)
	if err != nil {
		return nil, err
	}
	st.Payments, err = scanPayments(rows)
	return st, err
}

// Summary returns the school's fee position, for one term when term and year
// are given.
func (s *Service) Summary(ctx context.Context, tenantID uuid.UUID, term, year int) (*Summary, error) {
	out := &Summary{ByChannel: map[string]int64{}}
	period := ``
	args := []any{tenantID}
	if term > 0 {
		args = append(args, term)
		period += fmt.Sprintf(` AND i.term = $%d`, len(args))
	}
	if year > 0 {
		args = append(args, year)
		period += fmt.Sprintf(` AND i.year = $%d`, len(args))
	}

	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*),
		       COALESCE(SUM(i.total_cents), 0),
		       COALESCE(SUM(i.discount_cents), 0),
		       COALESCE(SUM(i.paid_cents), 0),
		       COALESCE(SUM(GREATEST(i.total_cents - i.discount_cents - i.paid_cents, 0)), 0),
		       COALESCE(SUM(GREATEST(i.total_cents - i.discount_cents - i.paid_cents, 0)) FILTER (WHERE i.due_date < CURRENT_DATE), 0),
		       COUNT(DISTINCT i.learner_id) FILTER (WHERE i.total_cents - i.discount_cents - i.paid_cents > 0)
		FROM invoices i
		WHERE i.tenant_id = $1 AND i.status <> 'void'`+period, args...).Scan(
		&out.Invoices, &out.BilledCents, &out.DiscountCents, &out.CollectedCents,
		&out.OutstandingCents, &out.OverdueCents, &out.LearnersOwing); err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT p.channel, COALESCE(SUM(p.amount_cents), 0)
		FROM payments p JOIN invoices i ON i.id = p.invoice_id
		WHERE p.tenant_id = $1 AND p.status = 'completed' AND i.status <> 'void'`+period+`
		GROUP BY p.channel`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var channel string
		var cents int64
		if err := rows.Scan(&channel, &cents); err != nil {
			rows.Close()
			return nil, err
		}
		out.ByChannel[channel] = cents
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(amount_cents - allocated_cents), 0)
		FROM mpesa_inbox WHERE tenant_id = $1 AND status <> 'allocated'`, tenantID).Scan(&out.UnmatchedCount, &out.UnmatchedCents); err != nil {
		return nil, err
	}
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM payments WHERE tenant_id = $1 AND channel = 'mpesa' AND status = 'pending'`,
		tenantID).Scan(&out.PendingMpesa); err != nil {
		return nil, err
	}
	return out, nil
}
