package finance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles finance domain operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a finance service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
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

// listFeeItems returns all items for a fee structure ordered by sort_order.
func (s *Service) listFeeItems(ctx context.Context, tenantID, feeStructureID uuid.UUID) ([]FeeStructureItem, error) {
	query := fmt.Sprintf(`SELECT %s FROM fee_structure_items
		WHERE tenant_id = $1 AND fee_structure_id = $2 ORDER BY sort_order, created_at`, feeItemColumns)
	rows, err := s.pool.Query(ctx, query, tenantID, feeStructureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []FeeStructureItem
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
	argIdx := 2

	if grade != "" {
		query += fmt.Sprintf(` AND grade = $%d`, argIdx)
		args = append(args, grade)
		argIdx++
	}
	if term > 0 {
		query += fmt.Sprintf(` AND term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	query += ` ORDER BY year DESC, term DESC, grade`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var structures []FeeStructure
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

	// Attach items for each structure
	for i := range structures {
		items, err := s.listFeeItems(ctx, tenantID, structures[i].ID)
		if err != nil {
			return nil, err
		}
		structures[i].Items = items
	}
	return structures, nil
}

// GetFeeStructure returns a fee structure with items.
func (s *Service) GetFeeStructure(ctx context.Context, tenantID, id uuid.UUID) (*FeeStructure, error) {
	query := fmt.Sprintf(`SELECT %s FROM fee_structures WHERE tenant_id = $1 AND id = $2`, feeStructureColumns)
	fs, err := scanFeeStructure(s.pool.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return nil, err
	}
	items, err := s.listFeeItems(ctx, tenantID, fs.ID)
	if err != nil {
		return nil, err
	}
	fs.Items = items
	return fs, nil
}

// CreateFeeStructure inserts a fee structure and its items in a transaction.
func (s *Service) CreateFeeStructure(ctx context.Context, tenantID uuid.UUID, req CreateFeeStructureRequest) (*FeeStructure, error) {
	if req.Name == "" || req.Grade == "" || req.Term < 1 || req.Term > 3 || req.Year <= 0 {
		return nil, errors.New("name, grade, term (1-3), and year are required")
	}

	var total int64
	for _, item := range req.Items {
		if item.AmountCents < 0 {
			return nil, errors.New("item amounts cannot be negative")
		}
		total += item.AmountCents
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := fmt.Sprintf(`INSERT INTO fee_structures (tenant_id, name, grade, term, year, total_cents, active, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING %s`, feeStructureColumns)
	fs, err := scanFeeStructure(tx.QueryRow(ctx, query,
		tenantID, req.Name, req.Grade, req.Term, req.Year, total, active, req.Notes, req.CreatedBy,
	))
	if err != nil {
		return nil, err
	}

	for idx, item := range req.Items {
		itemType := item.ItemType
		if itemType == "" {
			itemType = "other"
		}
		isOptional := false
		if item.IsOptional != nil {
			isOptional = *item.IsOptional
		}
		sortOrder := idx
		if item.SortOrder != nil {
			sortOrder = *item.SortOrder
		}
		itemQuery := fmt.Sprintf(`INSERT INTO fee_structure_items (tenant_id, fee_structure_id, name, amount_cents, item_type, is_optional, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING %s`, feeItemColumns)
		it, err := scanFeeItem(tx.QueryRow(ctx, itemQuery,
			tenantID, fs.ID, item.Name, item.AmountCents, itemType, isOptional, sortOrder,
		))
		if err != nil {
			return nil, err
		}
		fs.Items = append(fs.Items, *it)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return fs, nil
}

// UpdateFeeStructure partially updates a fee structure.
func (s *Service) UpdateFeeStructure(ctx context.Context, tenantID, id uuid.UUID, req UpdateFeeStructureRequest) (*FeeStructure, error) {
	if _, err := s.GetFeeStructure(ctx, tenantID, id); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE fee_structures SET
		name = COALESCE($3, name),
		grade = COALESCE($4, grade),
		term = COALESCE($5, term),
		year = COALESCE($6, year),
		active = COALESCE($7, active),
		notes = COALESCE($8, notes)
		WHERE tenant_id = $1 AND id = $2
		RETURNING %s`, feeStructureColumns)
	fs, err := scanFeeStructure(s.pool.QueryRow(ctx, query,
		tenantID, id, req.Name, req.Grade, req.Term, req.Year, req.Active, req.Notes,
	))
	if err != nil {
		return nil, err
	}
	items, err := s.listFeeItems(ctx, tenantID, fs.ID)
	if err != nil {
		return nil, err
	}
	fs.Items = items
	return fs, nil
}

// DeleteFeeStructure removes a fee structure.
func (s *Service) DeleteFeeStructure(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM fee_structures WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// AddFeeItem adds an item to an existing fee structure.
func (s *Service) AddFeeItem(ctx context.Context, tenantID, structureID uuid.UUID, input FeeItemInput, sortOrder int) (*FeeStructureItem, error) {
	if _, err := s.GetFeeStructure(ctx, tenantID, structureID); err != nil {
		return nil, err
	}
	itemType := input.ItemType
	if itemType == "" {
		itemType = "other"
	}
	isOptional := false
	if input.IsOptional != nil {
		isOptional = *input.IsOptional
	}
	query := fmt.Sprintf(`INSERT INTO fee_structure_items (tenant_id, fee_structure_id, name, amount_cents, item_type, is_optional, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING %s`, feeItemColumns)
	// Recompute the structure total after adding the item
	if err := s.recomputeFeeTotal(ctx, tenantID, structureID); err != nil {
		return nil, err
	}
	return scanFeeItem(s.pool.QueryRow(ctx, query,
		tenantID, structureID, input.Name, input.AmountCents, itemType, isOptional, sortOrder,
	))
}

// DeleteFeeItem removes an item from a fee structure.
func (s *Service) DeleteFeeItem(ctx context.Context, tenantID, itemID uuid.UUID) error {
	var structureID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT fee_structure_id FROM fee_structure_items WHERE tenant_id = $1 AND id = $2`,
		tenantID, itemID).Scan(&structureID)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM fee_structure_items WHERE tenant_id = $1 AND id = $2`, tenantID, itemID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return s.recomputeFeeTotal(ctx, tenantID, structureID)
}

// recomputeFeeTotal recalculates and updates total_cents for a fee structure.
func (s *Service) recomputeFeeTotal(ctx context.Context, tenantID, structureID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE fee_structures fs SET total_cents = COALESCE((
			SELECT SUM(amount_cents) FROM fee_structure_items WHERE fee_structure_id = fs.id
		), 0)
		WHERE fs.tenant_id = $1 AND fs.id = $2`, tenantID, structureID)
	return err
}

// --- Invoice operations ---

const invoiceColumns = `i.id, i.tenant_id, i.learner_id, i.fee_structure_id, i.invoice_number,
	i.term, i.year, i.issue_date::text, i.due_date::text, i.total_cents, i.discount_cents,
	i.paid_cents, i.status, i.notes, i.created_by, i.created_at, i.updated_at,
	l.full_name, l.grade, l.stream`

func scanInvoice(row pgx.Row) (*Invoice, error) {
	var inv Invoice
	err := row.Scan(
		&inv.ID, &inv.TenantID, &inv.LearnerID, &inv.FeeStructureID, &inv.InvoiceNumber,
		&inv.Term, &inv.Year, &inv.IssueDate, &inv.DueDate, &inv.TotalCents, &inv.DiscountCents,
		&inv.PaidCents, &inv.Status, &inv.Notes, &inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.LearnerName, &inv.Grade, &inv.Stream,
	)
	if err != nil {
		return nil, err
	}
	inv.BalanceCents = inv.TotalCents - inv.DiscountCents - inv.PaidCents
	if inv.BalanceCents < 0 {
		inv.BalanceCents = 0
	}
	return &inv, nil
}

const invoiceItemColumns = `id, tenant_id, invoice_id, name, amount_cents, item_type, is_optional, sort_order, created_at`

func scanInvoiceItem(row pgx.Row) (*InvoiceItem, error) {
	var it InvoiceItem
	err := row.Scan(
		&it.ID, &it.TenantID, &it.InvoiceID, &it.Name, &it.AmountCents,
		&it.ItemType, &it.IsOptional, &it.SortOrder, &it.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// listInvoiceItems returns all snapshot items for an invoice.
func (s *Service) listInvoiceItems(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]InvoiceItem, error) {
	query := fmt.Sprintf(`SELECT %s FROM invoice_items
		WHERE tenant_id = $1 AND invoice_id = $2 ORDER BY sort_order, created_at`, invoiceItemColumns)
	rows, err := s.pool.Query(ctx, query, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []InvoiceItem
	for rows.Next() {
		it, err := scanInvoiceItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *it)
	}
	return items, rows.Err()
}

// ListInvoices returns invoices optionally filtered by status/learner/term/year.
func (s *Service) ListInvoices(ctx context.Context, tenantID uuid.UUID, status, learnerID string, term, year int) ([]Invoice, error) {
	query := fmt.Sprintf(`SELECT %s FROM invoices i
		JOIN learners l ON l.id = i.learner_id
		WHERE i.tenant_id = $1`, invoiceColumns)
	args := []any{tenantID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(` AND i.status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}
	if learnerID != "" {
		query += fmt.Sprintf(` AND i.learner_id = $%d`, argIdx)
		args = append(args, learnerID)
		argIdx++
	}
	if term > 0 {
		query += fmt.Sprintf(` AND i.term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND i.year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	query += ` ORDER BY i.created_at DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []Invoice
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
	query := fmt.Sprintf(`SELECT %s FROM invoices i
		JOIN learners l ON l.id = i.learner_id
		WHERE i.tenant_id = $1 AND i.id = $2`, invoiceColumns)
	inv, err := scanInvoice(s.pool.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return nil, err
	}
	items, err := s.listInvoiceItems(ctx, tenantID, inv.ID)
	if err != nil {
		return nil, err
	}
	inv.Items = items
	return inv, nil
}

// CreateInvoice creates an invoice for a learner.
// If fee_structure_id is provided, items are copied from it; otherwise direct items are used.
func (s *Service) CreateInvoice(ctx context.Context, tenantID uuid.UUID, req CreateInvoiceRequest) (*Invoice, error) {
	if req.LearnerID == uuid.Nil || req.Term < 1 || req.Term > 3 || req.Year <= 0 {
		return nil, errors.New("learner_id, term (1-3), and year are required")
	}

	// Fetch learner grade to validate fee structure compatibility
	var learnerGrade string
	err := s.pool.QueryRow(ctx,
		`SELECT grade FROM learners WHERE tenant_id = $1 AND id = $2`, tenantID, req.LearnerID).
		Scan(&learnerGrade)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Determine items and total
	var items []FeeItemInput
	if req.FeeStructureID != nil && *req.FeeStructureID != uuid.Nil {
		fs, err := s.GetFeeStructure(ctx, tenantID, *req.FeeStructureID)
		if err != nil {
			return nil, err
		}
		if fs.Grade != learnerGrade {
			return nil, errors.New("fee structure grade does not match learner grade")
		}
		for _, it := range fs.Items {
			items = append(items, FeeItemInput{
				Name:        it.Name,
				AmountCents: it.AmountCents,
				ItemType:    it.ItemType,
				IsOptional:  &it.IsOptional,
				SortOrder:   &it.SortOrder,
			})
		}
	} else {
		items = req.Items
	}
	if len(items) == 0 {
		return nil, errors.New("no fee items provided")
	}

	var total int64
	for _, it := range items {
		if it.AmountCents < 0 {
			return nil, errors.New("item amounts cannot be negative")
		}
		total += it.AmountCents
	}

	// Generate invoice number: INV-{YEAR}-{TERM}-{last 6 of uuid}
	invoiceNumber := fmt.Sprintf("INV-%d-%d-%s", req.Year, req.Term, uuid.NewString()[:6])
	issueDate := time.Now().Format("2006-01-02")
	if req.IssueDate != nil {
		issueDate = *req.IssueDate
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO invoices (tenant_id, learner_id, fee_structure_id, invoice_number, term, year, issue_date, due_date, total_cents, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		tenantID, req.LearnerID, req.FeeStructureID, invoiceNumber, req.Term, req.Year,
		issueDate, req.DueDate, total, req.Notes, req.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	// Fetch the inserted invoice id
	var invoiceID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT id FROM invoices WHERE tenant_id = $1 AND invoice_number = $2`, tenantID, invoiceNumber).
		Scan(&invoiceID)
	if err != nil {
		return nil, err
	}

	for idx, it := range items {
		itemType := it.ItemType
		if itemType == "" {
			itemType = "other"
		}
		isOptional := false
		if it.IsOptional != nil {
			isOptional = *it.IsOptional
		}
		sortOrder := idx
		if it.SortOrder != nil {
			sortOrder = *it.SortOrder
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO invoice_items (tenant_id, invoice_id, name, amount_cents, item_type, is_optional, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			tenantID, invoiceID, it.Name, it.AmountCents, itemType, isOptional, sortOrder,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetInvoice(ctx, tenantID, invoiceID)
}

// UpdateInvoice partially updates an invoice (due date, status, notes).
func (s *Service) UpdateInvoice(ctx context.Context, tenantID, id uuid.UUID, req UpdateInvoiceRequest) (*Invoice, error) {
	if _, err := s.GetInvoice(ctx, tenantID, id); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE invoices i SET
		due_date = COALESCE($3, i.due_date),
		status = COALESCE($4, i.status),
		notes = COALESCE($5, i.notes)
		WHERE i.tenant_id = $1 AND i.id = $2
		RETURNING %s`, invoiceColumns+" FROM learners l WHERE l.id = i.learner_id")
	inv, err := scanInvoice(s.pool.QueryRow(ctx, query,
		tenantID, id, req.DueDate, req.Status, req.Notes,
	))
	if err != nil {
		return nil, err
	}
	items, err := s.listInvoiceItems(ctx, tenantID, inv.ID)
	if err != nil {
		return nil, err
	}
	inv.Items = items
	return inv, nil
}

// DeleteInvoice removes an invoice (hard delete).
func (s *Service) DeleteInvoice(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM invoices WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Discount operations ---

const discountColumns = `id, tenant_id, invoice_id, amount_cents, discount_type, reason, approved_by, created_at`

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

	var discounts []Discount
	for rows.Next() {
		d, err := scanDiscount(rows)
		if err != nil {
			return nil, err
		}
		discounts = append(discounts, *d)
	}
	return discounts, rows.Err()
}

// CreateDiscount adds a discount and updates the invoice's discount_cents and status.
func (s *Service) CreateDiscount(ctx context.Context, tenantID, invoiceID uuid.UUID, req CreateDiscountRequest) (*Discount, error) {
	if req.AmountCents <= 0 {
		return nil, errors.New("amount_cents must be positive")
	}
	discountType := req.DiscountType
	if discountType == "" {
		discountType = "other"
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := fmt.Sprintf(`INSERT INTO discounts (tenant_id, invoice_id, amount_cents, discount_type, reason, approved_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING %s`, discountColumns)
	d, err := scanDiscount(tx.QueryRow(ctx, query,
		tenantID, invoiceID, req.AmountCents, discountType, req.Reason, req.ApprovedBy,
	))
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
	var invoiceID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT invoice_id FROM discounts WHERE tenant_id = $1 AND id = $2`,
		tenantID, discountID).Scan(&invoiceID)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `DELETE FROM discounts WHERE tenant_id = $1 AND id = $2`, tenantID, discountID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	if err := refreshInvoiceFinance(ctx, tx, tenantID, invoiceID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// refreshInvoiceFinance recomputes discount_cents, paid_cents, and status for an invoice.
// It must be called inside a transaction.
func refreshInvoiceFinance(ctx context.Context, tx pgx.Tx, tenantID, invoiceID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		UPDATE invoices i SET
			discount_cents = COALESCE((SELECT SUM(amount_cents) FROM discounts WHERE invoice_id = i.id), 0),
			paid_cents = COALESCE((SELECT SUM(amount_cents) FROM payments WHERE invoice_id = i.id AND status = 'completed'), 0),
			status = CASE
				WHEN i.status = 'void' THEN 'void'
				WHEN COALESCE((SELECT SUM(amount_cents) FROM payments WHERE invoice_id = i.id AND status = 'completed'), 0) >= i.total_cents - COALESCE((SELECT SUM(amount_cents) FROM discounts WHERE invoice_id = i.id), 0)
					THEN 'paid'
				WHEN COALESCE((SELECT SUM(amount_cents) FROM payments WHERE invoice_id = i.id AND status = 'completed'), 0) > 0
					THEN 'partially_paid'
				WHEN i.due_date IS NOT NULL AND i.due_date < CURRENT_DATE THEN 'overdue'
				ELSE 'unpaid'
			END
		WHERE i.tenant_id = $1 AND i.id = $2`, tenantID, invoiceID)
	return err
}

// --- Payment operations ---

const paymentColumns = `p.id, p.tenant_id, p.invoice_id, p.amount_cents, p.channel, p.status,
	p.reference, p.paid_by, p.phone, p.paid_at, p.received_by, p.notes,
	p.checkout_request_id, p.merchant_request_id, p.mpesa_receipt, p.mpesa_result_code, p.mpesa_result_desc,
	p.created_at, p.updated_at,
	i.invoice_number, l.full_name, l.grade`

func scanPayment(row pgx.Row) (*Payment, error) {
	var pay Payment
	err := row.Scan(
		&pay.ID, &pay.TenantID, &pay.InvoiceID, &pay.AmountCents, &pay.Channel, &pay.Status,
		&pay.Reference, &pay.PaidBy, &pay.Phone, &pay.PaidAt, &pay.ReceivedBy, &pay.Notes,
		&pay.CheckoutRequestID, &pay.MerchantRequestID, &pay.MpesaReceipt, &pay.MpesaResultCode, &pay.MpesaResultDesc,
		&pay.CreatedAt, &pay.UpdatedAt,
		&pay.InvoiceNumber, &pay.LearnerName, &pay.Grade,
	)
	if err != nil {
		return nil, err
	}
	return &pay, nil
}

// ListPayments returns payments optionally filtered by status/channel/term/year.
func (s *Service) ListPayments(ctx context.Context, tenantID uuid.UUID, status, channel string, term, year int) ([]Payment, error) {
	query := fmt.Sprintf(`SELECT %s FROM payments p
		JOIN invoices i ON i.id = p.invoice_id
		JOIN learners l ON l.id = i.learner_id
		WHERE p.tenant_id = $1`, paymentColumns)
	args := []any{tenantID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(` AND p.status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}
	if channel != "" {
		query += fmt.Sprintf(` AND p.channel = $%d`, argIdx)
		args = append(args, channel)
		argIdx++
	}
	if term > 0 {
		query += fmt.Sprintf(` AND i.term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND i.year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	query += ` ORDER BY p.created_at DESC LIMIT 500`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []Payment
	for rows.Next() {
		pay, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, *pay)
	}
	return payments, rows.Err()
}

// GetPayment returns a single payment.
func (s *Service) GetPayment(ctx context.Context, tenantID, id uuid.UUID) (*Payment, error) {
	query := fmt.Sprintf(`SELECT %s FROM payments p
		JOIN invoices i ON i.id = p.invoice_id
		JOIN learners l ON l.id = i.learner_id
		WHERE p.tenant_id = $1 AND p.id = $2`, paymentColumns)
	return scanPayment(s.pool.QueryRow(ctx, query, tenantID, id))
}

// CreatePayment records a payment (cash, bank, cheque, or manual M-Pesa entry).
// For non-M-Pesa channels it is immediately completed and refreshes the invoice.
func (s *Service) CreatePayment(ctx context.Context, tenantID uuid.UUID, req CreatePaymentRequest) (*Payment, error) {
	if req.InvoiceID == uuid.Nil || req.AmountCents <= 0 {
		return nil, errors.New("invoice_id and a positive amount_cents are required")
	}
	channel := req.Channel
	if channel == "" {
		channel = "cash"
	}

	// Verify invoice belongs to tenant
	var invoiceExists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM invoices WHERE tenant_id = $1 AND id = $2)`,
		tenantID, req.InvoiceID).Scan(&invoiceExists)
	if err != nil {
		return nil, err
	}
	if !invoiceExists {
		return nil, pgx.ErrNoRows
	}

	status := "completed"
	paidAt := time.Now()
	if req.PaidAt != nil {
		paidAt = *req.PaidAt
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := fmt.Sprintf(`INSERT INTO payments (tenant_id, invoice_id, amount_cents, channel, status, reference, paid_by, phone, paid_at, received_by, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING %s`, paymentColumns+`
		FROM invoices i JOIN learners l ON l.id = i.learner_id WHERE i.id = payments.invoice_id`)
	pay, err := scanPayment(tx.QueryRow(ctx, query,
		tenantID, req.InvoiceID, req.AmountCents, channel, status, req.Reference, req.PaidBy, req.Phone,
		paidAt, req.ReceivedBy, req.Notes,
	))
	if err != nil {
		return nil, err
	}

	if err := refreshInvoiceFinance(ctx, tx, tenantID, req.InvoiceID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return pay, nil
}

// ReversePayment reverses a payment (status -> reversed) and refreshes the invoice.
func (s *Service) ReversePayment(ctx context.Context, tenantID, id uuid.UUID) (*Payment, error) {
	var invoiceID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT invoice_id FROM payments WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&invoiceID)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := fmt.Sprintf(`UPDATE payments p SET status = 'reversed'
		WHERE p.tenant_id = $1 AND p.id = $2 AND p.status = 'completed'
		RETURNING %s`, paymentColumns+`
		FROM invoices i JOIN learners l ON l.id = i.learner_id WHERE i.id = p.invoice_id`)
	pay, err := scanPayment(tx.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return nil, err
	}

	if err := refreshInvoiceFinance(ctx, tx, tenantID, invoiceID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return pay, nil
}
