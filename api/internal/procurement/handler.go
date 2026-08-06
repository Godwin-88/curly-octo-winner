package procurement

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the Procurement API.
type Handler struct {
	service *Service
}

// NewHandler creates a new procurement handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Mount registers all procurement routes under the provided router.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/suppliers", func(r chi.Router) {
		r.Get("/", h.listSuppliers)
		r.Post("/", h.createSupplier)
		r.Get("/{id}", h.getSupplier)
		r.Patch("/{id}", h.updateSupplier)
		r.Delete("/{id}", h.deleteSupplier)
	})
	r.Route("/requisitions", func(r chi.Router) {
		r.Get("/", h.listRequisitions)
		r.Post("/", h.createRequisition)
		r.Get("/{id}", h.getRequisition)
		r.Post("/{id}/approve", h.approveRequisition)
		r.Post("/{id}/reject", h.rejectRequisition)
		r.Post("/{id}/cancel", h.cancelRequisition)
		r.Delete("/{id}", h.deleteRequisition)
	})
	r.Route("/purchase-orders", func(r chi.Router) {
		r.Get("/", h.listPurchaseOrders)
		r.Post("/", h.createPurchaseOrder)
		r.Get("/{id}", h.getPurchaseOrder)
		r.Patch("/{id}", h.updatePurchaseOrder)
		r.Delete("/{id}", h.deletePurchaseOrder)
	})
	r.Route("/goods-receipts", func(r chi.Router) {
		r.Get("/", h.listGoodsReceipts)
		r.Post("/", h.createGoodsReceipt)
		r.Get("/{id}", h.getGoodsReceipt)
		r.Delete("/{id}", h.deleteGoodsReceipt)
	})
	r.Route("/supplier-payments", func(r chi.Router) {
		r.Get("/", h.listSupplierPayments)
		r.Post("/", h.createSupplierPayment)
		r.Get("/{id}", h.getSupplierPayment)
		r.Patch("/{id}", h.updateSupplierPayment)
		r.Delete("/{id}", h.deleteSupplierPayment)
	})
}

// --- Supplier handlers ---

func (h *Handler) listSuppliers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	category := r.URL.Query().Get("category")
	includeInactive := r.URL.Query().Get("include_inactive") == "true"
	suppliers, err := h.service.ListSuppliers(r.Context(), tenantID, category, includeInactive)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, suppliers)
}

func (h *Handler) createSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	supplier, err := h.service.CreateSupplier(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, supplier)
}

func (h *Handler) getSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid supplier ID")
		return
	}
	supplier, err := h.service.GetSupplier(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, supplier)
}

func (h *Handler) updateSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid supplier ID")
		return
	}
	var req UpdateSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	supplier, err := h.service.UpdateSupplier(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, supplier)
}

func (h *Handler) deleteSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid supplier ID")
		return
	}
	if err := h.service.DeleteSupplier(r.Context(), tenantID, id); err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}

// --- Requisition handlers ---

func (h *Handler) listRequisitions(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	status := r.URL.Query().Get("status")
	department := r.URL.Query().Get("department")
	reqs, err := h.service.ListRequisitions(r.Context(), tenantID, status, department)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, reqs)
}

func (h *Handler) createRequisition(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateRequisitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	pr, err := h.service.CreateRequisition(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, pr)
}

func (h *Handler) getRequisition(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid requisition ID")
		return
	}
	pr, err := h.service.GetRequisition(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, pr)
}

func (h *Handler) approveRequisition(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid requisition ID")
		return
	}
	var req ApproveRequisitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	pr, err := h.service.ApproveRequisition(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "APPROVE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, pr)
}

func (h *Handler) rejectRequisition(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid requisition ID")
		return
	}
	var req RejectRequisitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	pr, err := h.service.RejectRequisition(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "REJECT_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, pr)
}

func (h *Handler) cancelRequisition(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid requisition ID")
		return
	}
	pr, err := h.service.CancelRequisition(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondBadRequest(w, "CANCEL_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, pr)
}

func (h *Handler) deleteRequisition(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid requisition ID")
		return
	}
	if err := h.service.DeleteRequisition(r.Context(), tenantID, id); err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}

// --- Purchase order handlers ---

func (h *Handler) listPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	status := r.URL.Query().Get("status")
	supplierID := r.URL.Query().Get("supplier_id")
	pos, err := h.service.ListPurchaseOrders(r.Context(), tenantID, status, supplierID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, pos)
}

func (h *Handler) createPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreatePurchaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	po, err := h.service.CreatePurchaseOrder(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, po)
}

func (h *Handler) getPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid purchase order ID")
		return
	}
	po, err := h.service.GetPurchaseOrder(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, po)
}

func (h *Handler) updatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid purchase order ID")
		return
	}
	var req UpdatePurchaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	po, err := h.service.UpdatePurchaseOrder(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, po)
}

func (h *Handler) deletePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid purchase order ID")
		return
	}
	if err := h.service.DeletePurchaseOrder(r.Context(), tenantID, id); err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}

// --- Goods receipt handlers ---

func (h *Handler) listGoodsReceipts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	status := r.URL.Query().Get("status")
	purchaseOrderID := r.URL.Query().Get("purchase_order_id")
	grns, err := h.service.ListGoodsReceipts(r.Context(), tenantID, status, purchaseOrderID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, grns)
}

func (h *Handler) createGoodsReceipt(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateGoodsReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	gr, err := h.service.CreateGoodsReceipt(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, gr)
}

func (h *Handler) getGoodsReceipt(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid goods receipt ID")
		return
	}
	gr, err := h.service.GetGoodsReceipt(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, gr)
}

func (h *Handler) deleteGoodsReceipt(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid goods receipt ID")
		return
	}
	if err := h.service.DeleteGoodsReceipt(r.Context(), tenantID, id); err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}

// --- Supplier payment handlers ---

func (h *Handler) listSupplierPayments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	status := r.URL.Query().Get("status")
	supplierID := r.URL.Query().Get("supplier_id")
	payments, err := h.service.ListSupplierPayments(r.Context(), tenantID, status, supplierID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, payments)
}

func (h *Handler) createSupplierPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateSupplierPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	payment, err := h.service.CreateSupplierPayment(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, payment)
}

func (h *Handler) getSupplierPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid supplier payment ID")
		return
	}
	payment, err := h.service.GetSupplierPayment(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, payment)
}

func (h *Handler) updateSupplierPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid supplier payment ID")
		return
	}
	var req UpdateSupplierPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	payment, err := h.service.UpdateSupplierPayment(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, payment)
}

func (h *Handler) deleteSupplierPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid supplier payment ID")
		return
	}
	if err := h.service.DeleteSupplierPayment(r.Context(), tenantID, id); err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}
