package procurement

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// --- Purchase order operations ---

const poColumns = `po.id, po.tenant_id, po.po_number, po.requisition_id, po.supplier_id, sup.name,
	po.order_date::text, po.expected_delivery::text, po.status, po.total_amount_cents,
	po.notes, po.created_by, po.created_at, po.updated_at`

const poJoin = `FROM purchase_orders po
	JOIN suppliers sup ON sup.id = po.supplier_id`

func scanPO(row pgx.Row) (*PurchaseOrder, error) {
	var po PurchaseOrder
	err := row.Scan(
		&po.ID, &po.TenantID, &po.PONumber, &po.RequisitionID, &po.SupplierID, &po.SupplierName,
		&po.OrderDate, &po.ExpectedDelivery, &po.Status, &po.TotalAmountCents,
		&po.Notes, &po.CreatedBy, &po.CreatedAt, &po.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &po, nil
}

const poItemColumns = `id, tenant_id, purchase_order_id, item_name, description, quantity,
	unit, unit_cost_cents, total_cost_cents, created_at`

func scanPOItem(row pgx.Row) (*PurchaseOrderItem, error) {
	var it PurchaseOrderItem
	err := row.Scan(&it.ID, &it.TenantID, &it.PurchaseOrderID, &it.ItemName, &it.Description, &it.Quantity,
		&it.Unit, &it.UnitCostCents, &it.TotalCostCents, &it.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (s *Service) listPOItems(ctx context.Context, tenantID, poID uuid.UUID) ([]PurchaseOrderItem, error) {
	query := fmt.Sprintf(`SELECT %s FROM purchase_order_items WHERE tenant_id = $1 AND purchase_order_id = $2 ORDER BY created_at`, poItemColumns)
	rows, err := s.pool.Query(ctx, query, tenantID, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PurchaseOrderItem
	for rows.Next() {
		it, err := scanPOItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *it)
	}
	return items, rows.Err()
}

// ListPurchaseOrders returns POs optionally filtered by status/supplier.
func (s *Service) ListPurchaseOrders(ctx context.Context, tenantID uuid.UUID, status, supplierID string) ([]PurchaseOrder, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE po.tenant_id = $1`, poColumns, poJoin)
	args := []any{tenantID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(` AND po.status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}
	if supplierID != "" {
		query += fmt.Sprintf(` AND po.supplier_id = $%d`, argIdx)
		args = append(args, supplierID)
		argIdx++
	}
	query += ` ORDER BY po.order_date DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pos []PurchaseOrder
	for rows.Next() {
		po, err := scanPO(rows)
		if err != nil {
			return nil, err
		}
		pos = append(pos, *po)
	}
	return pos, rows.Err()
}

// GetPurchaseOrder returns a PO with items.
func (s *Service) GetPurchaseOrder(ctx context.Context, tenantID, id uuid.UUID) (*PurchaseOrder, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE po.tenant_id = $1 AND po.id = $2`, poColumns, poJoin)
	po, err := scanPO(s.pool.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return nil, err
	}
	items, err := s.listPOItems(ctx, tenantID, po.ID)
	if err != nil {
		return nil, err
	}
	po.Items = items
	return po, nil
}

// CreatePurchaseOrder creates a PO with items and computes total.
func (s *Service) CreatePurchaseOrder(ctx context.Context, tenantID uuid.UUID, req CreatePurchaseOrderRequest) (*PurchaseOrder, error) {
	if req.SupplierID == uuid.Nil || len(req.Items) == 0 {
		return nil, errors.New("supplier_id and at least one item are required")
	}
	var total int64
	for _, it := range req.Items {
		if it.Quantity <= 0 || it.UnitCostCents < 0 {
			return nil, errors.New("item quantity must be positive and cost cannot be negative")
		}
		total += int64(it.Quantity) * it.UnitCostCents
	}

	poNo := fmt.Sprintf("PO-%s", strings.ToUpper(uuid.NewString()[:8]))

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var po *PurchaseOrder
	if req.OrderDate != nil {
		query := fmt.Sprintf(`INSERT INTO purchase_orders (tenant_id, po_number, requisition_id, supplier_id,
			order_date, expected_delivery, total_amount_cents, notes, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING %s %s`, poColumns, poJoin)
		po, err = scanPO(tx.QueryRow(ctx, query,
			tenantID, poNo, req.RequisitionID, req.SupplierID, *req.OrderDate, req.ExpectedDelivery,
			total, req.Notes, req.CreatedBy,
		))
	} else {
		query := fmt.Sprintf(`INSERT INTO purchase_orders (tenant_id, po_number, requisition_id, supplier_id,
			expected_delivery, total_amount_cents, notes, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING %s %s`, poColumns, poJoin)
		po, err = scanPO(tx.QueryRow(ctx, query,
			tenantID, poNo, req.RequisitionID, req.SupplierID, req.ExpectedDelivery,
			total, req.Notes, req.CreatedBy,
		))
	}
	if err != nil {
		return nil, err
	}

	for _, it := range req.Items {
		itemTotal := int64(it.Quantity) * it.UnitCostCents
		itemQuery := fmt.Sprintf(`INSERT INTO purchase_order_items (tenant_id, purchase_order_id, item_name, description,
			quantity, unit, unit_cost_cents, total_cost_cents)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING %s`, poItemColumns)
		item, err := scanPOItem(tx.QueryRow(ctx, itemQuery,
			tenantID, po.ID, it.ItemName, it.Description, it.Quantity, it.Unit, it.UnitCostCents, itemTotal,
		))
		if err != nil {
			return nil, err
		}
		po.Items = append(po.Items, *item)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return po, nil
}

// UpdatePurchaseOrder partially updates a PO.
func (s *Service) UpdatePurchaseOrder(ctx context.Context, tenantID, id uuid.UUID, req UpdatePurchaseOrderRequest) (*PurchaseOrder, error) {
	if _, err := s.GetPurchaseOrder(ctx, tenantID, id); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE purchase_orders po SET
		expected_delivery = COALESCE($3, po.expected_delivery),
		status = COALESCE($4, po.status),
		notes = COALESCE($5, po.notes)
		WHERE po.tenant_id = $1 AND po.id = $2
		RETURNING %s %s`, poColumns, poJoin)
	po, err := scanPO(s.pool.QueryRow(ctx, query, tenantID, id, req.ExpectedDelivery, req.Status, req.Notes))
	if err != nil {
		return nil, err
	}
	items, err := s.listPOItems(ctx, tenantID, po.ID)
	if err != nil {
		return nil, err
	}
	po.Items = items
	return po, nil
}

// DeletePurchaseOrder removes a PO.
func (s *Service) DeletePurchaseOrder(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM purchase_orders WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Goods receipt operations ---

const grnColumns = `gr.id, gr.tenant_id, gr.grn_number, gr.purchase_order_id, po.po_number,
	gr.supplier_id, sup.name, gr.received_date::text, gr.received_by, recv_staff.full_name,
	gr.status, gr.notes, gr.created_at, gr.updated_at`

const grnJoin = `FROM goods_receipts gr
	JOIN purchase_orders po ON po.id = gr.purchase_order_id
	JOIN suppliers sup ON sup.id = gr.supplier_id
	LEFT JOIN staff recv_staff ON recv_staff.id = gr.received_by`

func scanGRN(row pgx.Row) (*GoodsReceipt, error) {
	var gr GoodsReceipt
	err := row.Scan(
		&gr.ID, &gr.TenantID, &gr.GRNNumber, &gr.PurchaseOrderID, &gr.PONumber,
		&gr.SupplierID, &gr.SupplierName, &gr.ReceivedDate, &gr.ReceivedBy, &gr.ReceivedByName,
		&gr.Status, &gr.Notes, &gr.CreatedAt, &gr.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &gr, nil
}

const grnItemColumns = `id, tenant_id, goods_receipt_id, po_item_id, item_name, quantity_received,
	quantity_rejected, unit, unit_cost_cents, total_cost_cents, created_at`

func scanGRNItem(row pgx.Row) (*GoodsReceiptItem, error) {
	var it GoodsReceiptItem
	err := row.Scan(&it.ID, &it.TenantID, &it.GoodsReceiptID, &it.POItemID, &it.ItemName, &it.QuantityReceived,
		&it.QuantityRejected, &it.Unit, &it.UnitCostCents, &it.TotalCostCents, &it.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (s *Service) listGRNItems(ctx context.Context, tenantID, grnID uuid.UUID) ([]GoodsReceiptItem, error) {
	query := fmt.Sprintf(`SELECT %s FROM goods_receipt_items WHERE tenant_id = $1 AND goods_receipt_id = $2 ORDER BY created_at`, grnItemColumns)
	rows, err := s.pool.Query(ctx, query, tenantID, grnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GoodsReceiptItem
	for rows.Next() {
		it, err := scanGRNItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *it)
	}
	return items, rows.Err()
}

// ListGoodsReceipts returns GRNs optionally filtered by status/PO.
func (s *Service) ListGoodsReceipts(ctx context.Context, tenantID uuid.UUID, status, purchaseOrderID string) ([]GoodsReceipt, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE gr.tenant_id = $1`, grnColumns, grnJoin)
	args := []any{tenantID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(` AND gr.status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}
	if purchaseOrderID != "" {
		query += fmt.Sprintf(` AND gr.purchase_order_id = $%d`, argIdx)
		args = append(args, purchaseOrderID)
		argIdx++
	}
	query += ` ORDER BY gr.received_date DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grns []GoodsReceipt
	for rows.Next() {
		gr, err := scanGRN(rows)
		if err != nil {
			return nil, err
		}
		grns = append(grns, *gr)
	}
	return grns, rows.Err()
}

// GetGoodsReceipt returns a GRN with items.
func (s *Service) GetGoodsReceipt(ctx context.Context, tenantID, id uuid.UUID) (*GoodsReceipt, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE gr.tenant_id = $1 AND gr.id = $2`, grnColumns, grnJoin)
	gr, err := scanGRN(s.pool.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return nil, err
	}
	items, err := s.listGRNItems(ctx, tenantID, gr.ID)
	if err != nil {
		return nil, err
	}
	gr.Items = items
	return gr, nil
}

// CreateGoodsReceipt creates a GRN, verifies quantities, and updates PO status.
func (s *Service) CreateGoodsReceipt(ctx context.Context, tenantID uuid.UUID, req CreateGoodsReceiptRequest) (*GoodsReceipt, error) {
	if req.PurchaseOrderID == uuid.Nil || len(req.Items) == 0 {
		return nil, errors.New("purchase_order_id and at least one item are required")
	}
	po, err := s.GetPurchaseOrder(ctx, tenantID, req.PurchaseOrderID)
	if err != nil {
		return nil, err
	}
	if po.Status == "cancelled" {
		return nil, errors.New("cannot receive goods for a cancelled purchase order")
	}

	grnNo := fmt.Sprintf("GRN-%s", strings.ToUpper(uuid.NewString()[:8]))
	status := "received"
	hasPartial := false
	for _, it := range req.Items {
		if it.QuantityReceived <= 0 {
			return nil, errors.New("quantity_received must be positive")
		}
		if it.QuantityRejected > 0 {
			hasPartial = true
		}
	}
	if hasPartial {
		status = "partial"
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var gr *GoodsReceipt
	if req.ReceivedDate != nil {
		query := fmt.Sprintf(`INSERT INTO goods_receipts (tenant_id, grn_number, purchase_order_id, supplier_id,
			received_date, received_by, status, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING %s %s`, grnColumns, grnJoin)
		gr, err = scanGRN(tx.QueryRow(ctx, query,
			tenantID, grnNo, req.PurchaseOrderID, po.SupplierID, *req.ReceivedDate, req.ReceivedBy, status, req.Notes,
		))
	} else {
		query := fmt.Sprintf(`INSERT INTO goods_receipts (tenant_id, grn_number, purchase_order_id, supplier_id,
			received_by, status, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING %s %s`, grnColumns, grnJoin)
		gr, err = scanGRN(tx.QueryRow(ctx, query,
			tenantID, grnNo, req.PurchaseOrderID, po.SupplierID, req.ReceivedBy, status, req.Notes,
		))
	}
	if err != nil {
		return nil, err
	}

	for _, it := range req.Items {
		total := int64(it.QuantityReceived) * it.UnitCostCents
		itemQuery := fmt.Sprintf(`INSERT INTO goods_receipt_items (tenant_id, goods_receipt_id, po_item_id, item_name,
			quantity_received, quantity_rejected, unit, unit_cost_cents, total_cost_cents)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING %s`, grnItemColumns)
		item, err := scanGRNItem(tx.QueryRow(ctx, itemQuery,
			tenantID, gr.ID, it.POItemID, it.ItemName, it.QuantityReceived, it.QuantityRejected,
			it.Unit, it.UnitCostCents, total,
		))
		if err != nil {
			return nil, err
		}
		gr.Items = append(gr.Items, *item)
	}

	// Update PO status
	poStatus := "received"
	if status == "partial" {
		poStatus = "partially_received"
	}
	if _, err := tx.Exec(ctx,
		`UPDATE purchase_orders SET status = $3 WHERE tenant_id = $1 AND id = $2`,
		tenantID, req.PurchaseOrderID, poStatus,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return gr, nil
}

// DeleteGoodsReceipt removes a GRN.
func (s *Service) DeleteGoodsReceipt(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM goods_receipts WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Supplier payment operations ---

const paymentColumns = `sp.id, sp.tenant_id, sp.payment_no, sp.supplier_id, sup.name,
	sp.purchase_order_id, po.po_number, sp.goods_receipt_id, gr.grn_number,
	sp.invoice_number, sp.invoice_date::text, sp.amount_cents, sp.payment_method, sp.status,
	sp.authorised_by, sp.authorised_at, sp.paid_at, sp.reference, sp.notes, sp.created_by,
	sp.created_at, sp.updated_at`

const paymentJoin = `FROM supplier_payments sp
	JOIN suppliers sup ON sup.id = sp.supplier_id
	LEFT JOIN purchase_orders po ON po.id = sp.purchase_order_id
	LEFT JOIN goods_receipts gr ON gr.id = sp.goods_receipt_id`

func scanPayment(row pgx.Row) (*SupplierPayment, error) {
	var sp SupplierPayment
	err := row.Scan(
		&sp.ID, &sp.TenantID, &sp.PaymentNo, &sp.SupplierID, &sp.SupplierName,
		&sp.PurchaseOrderID, &sp.PONumber, &sp.GoodsReceiptID, &sp.GRNNumber,
		&sp.InvoiceNumber, &sp.InvoiceDate, &sp.AmountCents, &sp.PaymentMethod, &sp.Status,
		&sp.AuthorisedBy, &sp.AuthorisedAt, &sp.PaidAt, &sp.Reference, &sp.Notes, &sp.CreatedBy,
		&sp.CreatedAt, &sp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sp, nil
}

// ListSupplierPayments returns payments optionally filtered by status/supplier.
func (s *Service) ListSupplierPayments(ctx context.Context, tenantID uuid.UUID, status, supplierID string) ([]SupplierPayment, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE sp.tenant_id = $1`, paymentColumns, paymentJoin)
	args := []any{tenantID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(` AND sp.status = $%d`, argIdx)
		args = append(args, status)
		argIdx++
	}
	if supplierID != "" {
		query += fmt.Sprintf(` AND sp.supplier_id = $%d`, argIdx)
		args = append(args, supplierID)
		argIdx++
	}
	query += ` ORDER BY sp.created_at DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []SupplierPayment
	for rows.Next() {
		sp, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, *sp)
	}
	return payments, rows.Err()
}

// GetSupplierPayment returns a single supplier payment.
func (s *Service) GetSupplierPayment(ctx context.Context, tenantID, id uuid.UUID) (*SupplierPayment, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE sp.tenant_id = $1 AND sp.id = $2`, paymentColumns, paymentJoin)
	return scanPayment(s.pool.QueryRow(ctx, query, tenantID, id))
}

// CreateSupplierPayment creates a payment (three-way match: PO → GRN → Invoice).
func (s *Service) CreateSupplierPayment(ctx context.Context, tenantID uuid.UUID, req CreateSupplierPaymentRequest) (*SupplierPayment, error) {
	if req.SupplierID == uuid.Nil || req.AmountCents <= 0 {
		return nil, errors.New("supplier_id and a positive amount are required")
	}
	paymentMethod := "bank"
	if req.PaymentMethod != nil {
		paymentMethod = *req.PaymentMethod
	}
	paymentNo := fmt.Sprintf("PAY-%s", strings.ToUpper(uuid.NewString()[:8]))

	query := fmt.Sprintf(`INSERT INTO supplier_payments (tenant_id, payment_no, supplier_id, purchase_order_id,
		goods_receipt_id, invoice_number, invoice_date, amount_cents, payment_method, reference, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING %s %s`, paymentColumns, paymentJoin)
	return scanPayment(s.pool.QueryRow(ctx, query,
		tenantID, paymentNo, req.SupplierID, req.PurchaseOrderID, req.GoodsReceiptID,
		req.InvoiceNumber, req.InvoiceDate, req.AmountCents, paymentMethod, req.Reference, req.Notes, req.CreatedBy,
	))
}

// UpdateSupplierPayment partially updates a payment.
func (s *Service) UpdateSupplierPayment(ctx context.Context, tenantID, id uuid.UUID, req UpdateSupplierPaymentRequest) (*SupplierPayment, error) {
	if _, err := s.GetSupplierPayment(ctx, tenantID, id); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE supplier_payments sp SET
		invoice_number = COALESCE($3, sp.invoice_number),
		invoice_date = COALESCE($4, sp.invoice_date),
		reference = COALESCE($5, sp.reference),
		notes = COALESCE($6, sp.notes),
		status = COALESCE($7, sp.status),
		authorised_by = CASE WHEN $7 = 'authorised' THEN COALESCE(sp.authorised_by, sp.created_by) ELSE sp.authorised_by END,
		authorised_at = CASE WHEN $7 = 'authorised' THEN COALESCE(sp.authorised_at, now()) ELSE sp.authorised_at END,
		paid_at = CASE WHEN $7 = 'paid' THEN COALESCE(sp.paid_at, now()) ELSE sp.paid_at END
		WHERE sp.tenant_id = $1 AND sp.id = $2
		RETURNING %s %s`, paymentColumns, paymentJoin)
	return scanPayment(s.pool.QueryRow(ctx, query,
		tenantID, id, req.InvoiceNumber, req.InvoiceDate, req.Reference, req.Notes, req.Status,
	))
}

// DeleteSupplierPayment removes a payment.
func (s *Service) DeleteSupplierPayment(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM supplier_payments WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
