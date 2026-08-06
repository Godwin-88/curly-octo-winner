package procurement

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles procurement domain operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a procurement service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// --- Supplier operations ---

const supplierColumns = `id, tenant_id, name, business_registration, kra_pin, category,
	contact_person, phone, email, whatsapp_phone, physical_address, bank_branch,
	bank_account_name, bank_account_number, bank_swift_code, notes, is_active, created_at, updated_at`

func scanSupplier(row pgx.Row) (*Supplier, error) {
	var s Supplier
	err := row.Scan(
		&s.ID, &s.TenantID, &s.Name, &s.BusinessRegistration, &s.KRAPin, &s.Category,
		&s.ContactPerson, &s.Phone, &s.Email, &s.WhatsappPhone, &s.PhysicalAddress, &s.BankBranch,
		&s.BankAccountName, &s.BankAccountNumber, &s.BankSwiftCode, &s.Notes, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListSuppliers returns suppliers optionally filtered by category/active.
func (s *Service) ListSuppliers(ctx context.Context, tenantID uuid.UUID, category string, includeInactive bool) ([]Supplier, error) {
	query := fmt.Sprintf(`SELECT %s FROM suppliers WHERE tenant_id = $1`, supplierColumns)
	args := []any{tenantID}
	argIdx := 2

	if category != "" {
		query += fmt.Sprintf(` AND category = $%d`, argIdx)
		args = append(args, category)
		argIdx++
	}
	if !includeInactive {
		query += ` AND is_active = true`
	}
	query += ` ORDER BY name`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suppliers []Supplier
	for rows.Next() {
		sup, err := scanSupplier(rows)
		if err != nil {
			return nil, err
		}
		suppliers = append(suppliers, *sup)
	}
	return suppliers, rows.Err()
}

// GetSupplier returns a single supplier.
func (s *Service) GetSupplier(ctx context.Context, tenantID, id uuid.UUID) (*Supplier, error) {
	query := fmt.Sprintf(`SELECT %s FROM suppliers WHERE tenant_id = $1 AND id = $2`, supplierColumns)
	return scanSupplier(s.pool.QueryRow(ctx, query, tenantID, id))
}

// CreateSupplier inserts a new supplier.
func (s *Service) CreateSupplier(ctx context.Context, tenantID uuid.UUID, req CreateSupplierRequest) (*Supplier, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	category := "general"
	if req.Category != nil {
		category = *req.Category
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	query := fmt.Sprintf(`INSERT INTO suppliers (tenant_id, name, business_registration, kra_pin, category,
		contact_person, phone, email, whatsapp_phone, physical_address, bank_branch,
		bank_account_name, bank_account_number, bank_swift_code, notes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING %s`, supplierColumns)
	return scanSupplier(s.pool.QueryRow(ctx, query,
		tenantID, req.Name, req.BusinessRegistration, req.KRAPin, category,
		req.ContactPerson, req.Phone, req.Email, req.WhatsappPhone, req.PhysicalAddress, req.BankBranch,
		req.BankAccountName, req.BankAccountNumber, req.BankSwiftCode, req.Notes, isActive,
	))
}

// UpdateSupplier partially updates a supplier.
func (s *Service) UpdateSupplier(ctx context.Context, tenantID, id uuid.UUID, req UpdateSupplierRequest) (*Supplier, error) {
	if _, err := s.GetSupplier(ctx, tenantID, id); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE suppliers SET
		name = COALESCE($3, name),
		business_registration = COALESCE($4, business_registration),
		kra_pin = COALESCE($5, kra_pin),
		category = COALESCE($6, category),
		contact_person = COALESCE($7, contact_person),
		phone = COALESCE($8, phone),
		email = COALESCE($9, email),
		whatsapp_phone = COALESCE($10, whatsapp_phone),
		physical_address = COALESCE($11, physical_address),
		bank_branch = COALESCE($12, bank_branch),
		bank_account_name = COALESCE($13, bank_account_name),
		bank_account_number = COALESCE($14, bank_account_number),
		bank_swift_code = COALESCE($15, bank_swift_code),
		notes = COALESCE($16, notes),
		is_active = COALESCE($17, is_active)
		WHERE tenant_id = $1 AND id = $2
		RETURNING %s`, supplierColumns)
	return scanSupplier(s.pool.QueryRow(ctx, query,
		tenantID, id, req.Name, req.BusinessRegistration, req.KRAPin, req.Category,
		req.ContactPerson, req.Phone, req.Email, req.WhatsappPhone, req.PhysicalAddress, req.BankBranch,
		req.BankAccountName, req.BankAccountNumber, req.BankSwiftCode, req.Notes, req.IsActive,
	))
}

// DeleteSupplier deactivates a supplier (soft delete).
func (s *Service) DeleteSupplier(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE suppliers SET is_active = false WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Requisition operations ---

const requisitionColumns = `pr.id, pr.tenant_id, pr.requisition_no, pr.title, pr.department,
	pr.requested_by, req_staff.full_name, pr.requested_at, pr.required_by::text, pr.justification,
	pr.status, pr.hod_approved_by, pr.hod_approved_at, pr.approved_by, pr.approved_at,
	pr.rejection_reason, pr.total_estimate_cents, pr.created_at, pr.updated_at`

const requisitionJoin = `FROM purchase_requisitions pr
	LEFT JOIN staff req_staff ON req_staff.id = pr.requested_by`

func scanRequisition(row pgx.Row) (*PurchaseRequisition, error) {
	var pr PurchaseRequisition
	err := row.Scan(
		&pr.ID, &pr.TenantID, &pr.RequisitionNo, &pr.Title, &pr.Department,
		&pr.RequestedBy, &pr.RequestedByName, &pr.RequestedAt, &pr.RequiredBy, &pr.Justification,
		&pr.Status, &pr.HODApprovedBy, &pr.HODApprovedAt, &pr.ApprovedBy, &pr.ApprovedAt,
		&pr.RejectionReason, &pr.TotalEstimateCents, &pr.CreatedAt, &pr.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

const requisitionItemColumns = `id, tenant_id, requisition_id, item_name, description, quantity,
	unit, estimated_unit_cost_cents, estimated_total_cents, created_at`

func scanRequisitionItem(row pgx.Row) (*RequisitionItem, error) {
	var it RequisitionItem
	err := row.Scan(&it.ID, &it.TenantID, &it.RequisitionID, &it.ItemName, &it.Description, &it.Quantity,
		&it.Unit, &it.EstimatedUnitCostCents, &it.EstimatedTotalCents, &it.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (s *Service) listRequisitionItems(ctx context.Context, tenantID, reqID uuid.UUID) ([]RequisitionItem, error) {
	query := fmt.Sprintf(`SELECT %s FROM requisition_items WHERE tenant_id = $1 AND requisition_id = $2 ORDER BY created_at`, requisitionItemColumns)
	rows, err := s.pool.Query(ctx, query, tenantID, reqID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []RequisitionItem
	for rows.Next() {
		it, err := scanRequisitionItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *it)
	}
	return items, rows.Err()
}

// ListRequisitions returns requisitions optionally filtered by status/department.
func (s *Service) ListRequisitions(ctx context.Context, tenantID uuid.UUID, status, department string) ([]PurchaseRequisition, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE pr.tenant_id = $1`, requisitionColumns, requisitionJoin)
	args := []any{tenantID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(` AND pr.status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}
	if department != "" {
		query += fmt.Sprintf(` AND pr.department = $%d`, argIdx)
		args = append(args, department)
		argIdx++
	}
	query += ` ORDER BY pr.requested_at DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reqs []PurchaseRequisition
	for rows.Next() {
		pr, err := scanRequisition(rows)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, *pr)
	}
	return reqs, rows.Err()
}

// GetRequisition returns a requisition with items.
func (s *Service) GetRequisition(ctx context.Context, tenantID, id uuid.UUID) (*PurchaseRequisition, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE pr.tenant_id = $1 AND pr.id = $2`, requisitionColumns, requisitionJoin)
	pr, err := scanRequisition(s.pool.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return nil, err
	}
	items, err := s.listRequisitionItems(ctx, tenantID, pr.ID)
	if err != nil {
		return nil, err
	}
	pr.Items = items
	return pr, nil
}

// CreateRequisition creates a requisition with items and computes total estimate.
func (s *Service) CreateRequisition(ctx context.Context, tenantID uuid.UUID, req CreateRequisitionRequest) (*PurchaseRequisition, error) {
	if req.Title == "" || len(req.Items) == 0 {
		return nil, errors.New("title and at least one item are required")
	}
	var total int64
	for _, it := range req.Items {
		if it.Quantity <= 0 || it.EstimatedUnitCostCents < 0 {
			return nil, errors.New("item quantity must be positive and cost cannot be negative")
		}
		total += int64(it.Quantity) * it.EstimatedUnitCostCents
	}

	reqNo := fmt.Sprintf("REQ-%s", strings.ToUpper(uuid.NewString()[:8]))

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := fmt.Sprintf(`INSERT INTO purchase_requisitions (tenant_id, requisition_no, title, department,
		requested_by, required_by, justification, total_estimate_cents)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING %s %s`, requisitionColumns, requisitionJoin)
	pr, err := scanRequisition(tx.QueryRow(ctx, query,
		tenantID, reqNo, req.Title, req.Department, req.RequestedBy, req.RequiredBy, req.Justification, total,
	))
	if err != nil {
		return nil, err
	}

	for _, it := range req.Items {
		itemTotal := int64(it.Quantity) * it.EstimatedUnitCostCents
		itemQuery := fmt.Sprintf(`INSERT INTO requisition_items (tenant_id, requisition_id, item_name, description,
			quantity, unit, estimated_unit_cost_cents, estimated_total_cents)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING %s`, requisitionItemColumns)
		item, err := scanRequisitionItem(tx.QueryRow(ctx, itemQuery,
			tenantID, pr.ID, it.ItemName, it.Description, it.Quantity, it.Unit, it.EstimatedUnitCostCents, itemTotal,
		))
		if err != nil {
			return nil, err
		}
		pr.Items = append(pr.Items, *item)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return pr, nil
}

// ApproveRequisition advances the approval workflow (pending → hod_approved → approved).
func (s *Service) ApproveRequisition(ctx context.Context, tenantID, id uuid.UUID, req ApproveRequisitionRequest) (*PurchaseRequisition, error) {
	pr, err := s.GetRequisition(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	var newStatus string
	switch pr.Status {
	case "pending":
		newStatus = "hod_approved"
	case "hod_approved":
		newStatus = "approved"
	default:
		return nil, errors.New("requisition is not in a pending approval state")
	}

	query := fmt.Sprintf(`UPDATE purchase_requisitions pr SET
		status = $3,
		hod_approved_by = CASE WHEN $3 = 'hod_approved' THEN COALESCE($4, pr.hod_approved_by) ELSE pr.hod_approved_by END,
		hod_approved_at = CASE WHEN $3 = 'hod_approved' THEN COALESCE(pr.hod_approved_at, now()) ELSE pr.hod_approved_at END,
		approved_by = CASE WHEN $3 = 'approved' THEN COALESCE($4, pr.approved_by) ELSE pr.approved_by END,
		approved_at = CASE WHEN $3 = 'approved' THEN COALESCE(pr.approved_at, now()) ELSE pr.approved_at END
		WHERE pr.tenant_id = $1 AND pr.id = $2
		RETURNING %s %s`, requisitionColumns, requisitionJoin)
	updated, err := scanRequisition(s.pool.QueryRow(ctx, query, tenantID, id, newStatus, req.ApprovedBy))
	if err != nil {
		return nil, err
	}
	items, err := s.listRequisitionItems(ctx, tenantID, updated.ID)
	if err != nil {
		return nil, err
	}
	updated.Items = items
	return updated, nil
}

// RejectRequisition rejects a requisition.
func (s *Service) RejectRequisition(ctx context.Context, tenantID, id uuid.UUID, req RejectRequisitionRequest) (*PurchaseRequisition, error) {
	query := fmt.Sprintf(`UPDATE purchase_requisitions pr SET
		status = 'rejected',
		rejection_reason = COALESCE($3, pr.rejection_reason),
		approved_by = COALESCE($4, pr.approved_by)
		WHERE pr.tenant_id = $1 AND pr.id = $2
		RETURNING %s %s`, requisitionColumns, requisitionJoin)
	pr, err := scanRequisition(s.pool.QueryRow(ctx, query, tenantID, id, req.RejectionReason, req.ApprovedBy))
	if err != nil {
		return nil, err
	}
	items, err := s.listRequisitionItems(ctx, tenantID, pr.ID)
	if err != nil {
		return nil, err
	}
	pr.Items = items
	return pr, nil
}

// CancelRequisition cancels a requisition.
func (s *Service) CancelRequisition(ctx context.Context, tenantID, id uuid.UUID) (*PurchaseRequisition, error) {
	query := fmt.Sprintf(`UPDATE purchase_requisitions pr SET status = 'cancelled'
		WHERE pr.tenant_id = $1 AND pr.id = $2
		RETURNING %s %s`, requisitionColumns, requisitionJoin)
	pr, err := scanRequisition(s.pool.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return nil, err
	}
	items, err := s.listRequisitionItems(ctx, tenantID, pr.ID)
	if err != nil {
		return nil, err
	}
	pr.Items = items
	return pr, nil
}

// DeleteRequisition removes a requisition.
func (s *Service) DeleteRequisition(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM purchase_requisitions WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
