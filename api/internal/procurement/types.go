package procurement

import (
	"time"

	"github.com/google/uuid"
)

// Supplier is a vendor in the supplier registry (KYC).
type Supplier struct {
	ID                   uuid.UUID `json:"id"`
	TenantID             uuid.UUID `json:"tenant_id"`
	Name                 string    `json:"name"`
	BusinessRegistration *string   `json:"business_registration,omitempty"`
	KRAPin               *string   `json:"kra_pin,omitempty"`
	Category             string    `json:"category"`
	ContactPerson        *string   `json:"contact_person,omitempty"`
	Phone                *string   `json:"phone,omitempty"`
	Email                *string   `json:"email,omitempty"`
	WhatsappPhone        *string   `json:"whatsapp_phone,omitempty"`
	PhysicalAddress      *string   `json:"physical_address,omitempty"`
	BankBranch           *string   `json:"bank_branch,omitempty"`
	BankAccountName      *string   `json:"bank_account_name,omitempty"`
	BankAccountNumber    *string   `json:"bank_account_number,omitempty"`
	BankSwiftCode        *string   `json:"bank_swift_code,omitempty"`
	Notes                *string   `json:"notes,omitempty"`
	IsActive             bool      `json:"is_active"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// PurchaseRequisition is a staff request for goods/services with approval workflow.
type PurchaseRequisition struct {
	ID                 uuid.UUID         `json:"id"`
	TenantID           uuid.UUID         `json:"tenant_id"`
	RequisitionNo      string            `json:"requisition_no"`
	Title              string            `json:"title"`
	Department         *string           `json:"department,omitempty"`
	RequestedBy        *uuid.UUID        `json:"requested_by,omitempty"`
	RequestedByName    *string           `json:"requested_by_name,omitempty"`
	RequestedAt        time.Time         `json:"requested_at"`
	RequiredBy         *string           `json:"required_by,omitempty"`
	Justification      *string           `json:"justification,omitempty"`
	Status             string            `json:"status"`
	HODApprovedBy      *uuid.UUID        `json:"hod_approved_by,omitempty"`
	HODApprovedAt      *time.Time        `json:"hod_approved_at,omitempty"`
	ApprovedBy         *uuid.UUID        `json:"approved_by,omitempty"`
	ApprovedAt         *time.Time        `json:"approved_at,omitempty"`
	RejectionReason    *string           `json:"rejection_reason,omitempty"`
	TotalEstimateCents int64             `json:"total_estimate_cents"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	Items              []RequisitionItem `json:"items,omitempty"`
}

// RequisitionItem is a line item on a purchase requisition.
type RequisitionItem struct {
	ID                     uuid.UUID `json:"id"`
	TenantID               uuid.UUID `json:"tenant_id"`
	RequisitionID          uuid.UUID `json:"requisition_id"`
	ItemName               string    `json:"item_name"`
	Description            *string   `json:"description,omitempty"`
	Quantity               int       `json:"quantity"`
	Unit                   *string   `json:"unit,omitempty"`
	EstimatedUnitCostCents int64     `json:"estimated_unit_cost_cents"`
	EstimatedTotalCents    int64     `json:"estimated_total_cents"`
	CreatedAt              time.Time `json:"created_at"`
}

// PurchaseOrder is a PO generated from an approved requisition.
type PurchaseOrder struct {
	ID               uuid.UUID           `json:"id"`
	TenantID         uuid.UUID           `json:"tenant_id"`
	PONumber         string              `json:"po_number"`
	RequisitionID    *uuid.UUID          `json:"requisition_id,omitempty"`
	SupplierID       uuid.UUID           `json:"supplier_id"`
	SupplierName     string              `json:"supplier_name,omitempty"`
	OrderDate        string              `json:"order_date"`
	ExpectedDelivery *string             `json:"expected_delivery,omitempty"`
	Status           string              `json:"status"`
	TotalAmountCents int64               `json:"total_amount_cents"`
	Notes            *string             `json:"notes,omitempty"`
	CreatedBy        *uuid.UUID          `json:"created_by,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
	Items            []PurchaseOrderItem `json:"items,omitempty"`
}

// PurchaseOrderItem is a line item on a purchase order.
type PurchaseOrderItem struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	PurchaseOrderID uuid.UUID `json:"purchase_order_id"`
	ItemName        string    `json:"item_name"`
	Description     *string   `json:"description,omitempty"`
	Quantity        int       `json:"quantity"`
	Unit            *string   `json:"unit,omitempty"`
	UnitCostCents   int64     `json:"unit_cost_cents"`
	TotalCostCents  int64     `json:"total_cost_cents"`
	CreatedAt       time.Time `json:"created_at"`
}

// GoodsReceipt is a GRN confirming delivery with quantity verification.
type GoodsReceipt struct {
	ID              uuid.UUID          `json:"id"`
	TenantID        uuid.UUID          `json:"tenant_id"`
	GRNNumber       string             `json:"grn_number"`
	PurchaseOrderID uuid.UUID          `json:"purchase_order_id"`
	PONumber        string             `json:"po_number,omitempty"`
	SupplierID      uuid.UUID          `json:"supplier_id"`
	SupplierName    string             `json:"supplier_name,omitempty"`
	ReceivedDate    string             `json:"received_date"`
	ReceivedBy      *uuid.UUID         `json:"received_by,omitempty"`
	ReceivedByName  *string            `json:"received_by_name,omitempty"`
	Status          string             `json:"status"`
	Notes           *string            `json:"notes,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	Items           []GoodsReceiptItem `json:"items,omitempty"`
}

// GoodsReceiptItem is a line item on a GRN with quantity verification.
type GoodsReceiptItem struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	GoodsReceiptID   uuid.UUID  `json:"goods_receipt_id"`
	POItemID         *uuid.UUID `json:"po_item_id,omitempty"`
	ItemName         string     `json:"item_name"`
	QuantityReceived int        `json:"quantity_received"`
	QuantityRejected int        `json:"quantity_rejected"`
	Unit             *string    `json:"unit,omitempty"`
	UnitCostCents    int64      `json:"unit_cost_cents"`
	TotalCostCents   int64      `json:"total_cost_cents"`
	CreatedAt        time.Time  `json:"created_at"`
}

// SupplierPayment is a payment to a supplier after three-way match.
type SupplierPayment struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	PaymentNo       string     `json:"payment_no"`
	SupplierID      uuid.UUID  `json:"supplier_id"`
	SupplierName    string     `json:"supplier_name,omitempty"`
	PurchaseOrderID *uuid.UUID `json:"purchase_order_id,omitempty"`
	PONumber        *string    `json:"po_number,omitempty"`
	GoodsReceiptID  *uuid.UUID `json:"goods_receipt_id,omitempty"`
	GRNNumber       *string    `json:"grn_number,omitempty"`
	InvoiceNumber   *string    `json:"invoice_number,omitempty"`
	InvoiceDate     *string    `json:"invoice_date,omitempty"`
	AmountCents     int64      `json:"amount_cents"`
	PaymentMethod   string     `json:"payment_method"`
	Status          string     `json:"status"`
	AuthorisedBy    *uuid.UUID `json:"authorised_by,omitempty"`
	AuthorisedAt    *time.Time `json:"authorised_at,omitempty"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	Reference       *string    `json:"reference,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// --- Request types ---

type CreateSupplierRequest struct {
	Name                 string  `json:"name"`
	BusinessRegistration *string `json:"business_registration,omitempty"`
	KRAPin               *string `json:"kra_pin,omitempty"`
	Category             *string `json:"category,omitempty"`
	ContactPerson        *string `json:"contact_person,omitempty"`
	Phone                *string `json:"phone,omitempty"`
	Email                *string `json:"email,omitempty"`
	WhatsappPhone        *string `json:"whatsapp_phone,omitempty"`
	PhysicalAddress      *string `json:"physical_address,omitempty"`
	BankBranch           *string `json:"bank_branch,omitempty"`
	BankAccountName      *string `json:"bank_account_name,omitempty"`
	BankAccountNumber    *string `json:"bank_account_number,omitempty"`
	BankSwiftCode        *string `json:"bank_swift_code,omitempty"`
	Notes                *string `json:"notes,omitempty"`
	IsActive             *bool   `json:"is_active,omitempty"`
}

type UpdateSupplierRequest struct {
	Name                 *string `json:"name,omitempty"`
	BusinessRegistration *string `json:"business_registration,omitempty"`
	KRAPin               *string `json:"kra_pin,omitempty"`
	Category             *string `json:"category,omitempty"`
	ContactPerson        *string `json:"contact_person,omitempty"`
	Phone                *string `json:"phone,omitempty"`
	Email                *string `json:"email,omitempty"`
	WhatsappPhone        *string `json:"whatsapp_phone,omitempty"`
	PhysicalAddress      *string `json:"physical_address,omitempty"`
	BankBranch           *string `json:"bank_branch,omitempty"`
	BankAccountName      *string `json:"bank_account_name,omitempty"`
	BankAccountNumber    *string `json:"bank_account_number,omitempty"`
	BankSwiftCode        *string `json:"bank_swift_code,omitempty"`
	Notes                *string `json:"notes,omitempty"`
	IsActive             *bool   `json:"is_active,omitempty"`
}

type RequisitionItemInput struct {
	ItemName               string  `json:"item_name"`
	Description            *string `json:"description,omitempty"`
	Quantity               int     `json:"quantity"`
	Unit                   *string `json:"unit,omitempty"`
	EstimatedUnitCostCents int64   `json:"estimated_unit_cost_cents"`
}

type CreateRequisitionRequest struct {
	Title         string                 `json:"title"`
	Department    *string                `json:"department,omitempty"`
	RequestedBy   *uuid.UUID             `json:"requested_by,omitempty"`
	RequiredBy    *string                `json:"required_by,omitempty"`
	Justification *string                `json:"justification,omitempty"`
	Items         []RequisitionItemInput `json:"items"`
}

type ApproveRequisitionRequest struct {
	ApprovedBy *uuid.UUID `json:"approved_by,omitempty"`
}

type RejectRequisitionRequest struct {
	ApprovedBy      *uuid.UUID `json:"approved_by,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
}

type PurchaseOrderItemInput struct {
	ItemName      string  `json:"item_name"`
	Description   *string `json:"description,omitempty"`
	Quantity      int     `json:"quantity"`
	Unit          *string `json:"unit,omitempty"`
	UnitCostCents int64   `json:"unit_cost_cents"`
}

type CreatePurchaseOrderRequest struct {
	RequisitionID    *uuid.UUID               `json:"requisition_id,omitempty"`
	SupplierID       uuid.UUID                `json:"supplier_id"`
	OrderDate        *string                  `json:"order_date,omitempty"`
	ExpectedDelivery *string                  `json:"expected_delivery,omitempty"`
	Notes            *string                  `json:"notes,omitempty"`
	CreatedBy        *uuid.UUID               `json:"created_by,omitempty"`
	Items            []PurchaseOrderItemInput `json:"items"`
}

type UpdatePurchaseOrderRequest struct {
	ExpectedDelivery *string `json:"expected_delivery,omitempty"`
	Status           *string `json:"status,omitempty"`
	Notes            *string `json:"notes,omitempty"`
}

type GoodsReceiptItemInput struct {
	POItemID         *uuid.UUID `json:"po_item_id,omitempty"`
	ItemName         string     `json:"item_name"`
	QuantityReceived int        `json:"quantity_received"`
	QuantityRejected int        `json:"quantity_rejected"`
	Unit             *string    `json:"unit,omitempty"`
	UnitCostCents    int64      `json:"unit_cost_cents"`
}

type CreateGoodsReceiptRequest struct {
	PurchaseOrderID uuid.UUID               `json:"purchase_order_id"`
	ReceivedDate    *string                 `json:"received_date,omitempty"`
	ReceivedBy      *uuid.UUID              `json:"received_by,omitempty"`
	Notes           *string                 `json:"notes,omitempty"`
	Items           []GoodsReceiptItemInput `json:"items"`
}

type CreateSupplierPaymentRequest struct {
	SupplierID      uuid.UUID  `json:"supplier_id"`
	PurchaseOrderID *uuid.UUID `json:"purchase_order_id,omitempty"`
	GoodsReceiptID  *uuid.UUID `json:"goods_receipt_id,omitempty"`
	InvoiceNumber   *string    `json:"invoice_number,omitempty"`
	InvoiceDate     *string    `json:"invoice_date,omitempty"`
	AmountCents     int64      `json:"amount_cents"`
	PaymentMethod   *string    `json:"payment_method,omitempty"`
	Reference       *string    `json:"reference,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
}

type UpdateSupplierPaymentRequest struct {
	InvoiceNumber *string `json:"invoice_number,omitempty"`
	InvoiceDate   *string `json:"invoice_date,omitempty"`
	Reference     *string `json:"reference,omitempty"`
	Notes         *string `json:"notes,omitempty"`
	Status        *string `json:"status,omitempty"`
}
