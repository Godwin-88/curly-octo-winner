package finance

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the finance API.
type Handler struct {
	service *Service
	mpesa   *MpesaService
}

// NewHandler creates a new finance handler.
func NewHandler(service *Service, mpesa *MpesaService) *Handler {
	return &Handler{service: service, mpesa: mpesa}
}

// Mount registers all finance routes under the provided router.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/fee-structures", func(r chi.Router) {
		r.Get("/", h.listFeeStructures)
		r.Post("/", h.createFeeStructure)
		r.Get("/{id}", h.getFeeStructure)
		r.Patch("/{id}", h.updateFeeStructure)
		r.Delete("/{id}", h.deleteFeeStructure)

		r.Post("/{id}/items", h.addFeeItem)
		r.Delete("/items/{itemId}", h.deleteFeeItem)
	})

	r.Route("/invoices", func(r chi.Router) {
		r.Get("/", h.listInvoices)
		r.Post("/", h.createInvoice)
		r.Get("/{id}", h.getInvoice)
		r.Patch("/{id}", h.updateInvoice)
		r.Delete("/{id}", h.deleteInvoice)

		r.Get("/{id}/payments", h.listInvoicePayments)
		r.Get("/{id}/discounts", h.listDiscounts)
		r.Post("/{id}/discounts", h.createDiscount)
		r.Delete("/discounts/{discountId}", h.deleteDiscount)
	})

	r.Route("/payments", func(r chi.Router) {
		r.Get("/", h.listPayments)
		r.Post("/", h.createPayment)
		r.Get("/{id}", h.getPayment)
		r.Post("/{id}/reverse", h.reversePayment)
		r.Post("/mpesa/stk", h.initiateSTKPush)
	})

}

// MountWebhooks registers webhook routes that require NO authentication.
// Safaricom M-Pesa callbacks send no JWT, so they must NOT be behind the
// auth middleware. Register under /api/v1 alongside the WhatsApp webhook.
func (h *Handler) MountWebhooks(r chi.Router) {
	r.Post("/webhooks/mpesa/stk", h.mpesaCallback)
}

// --- Fee structure handlers ---

func (h *Handler) listFeeStructures(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	grade := r.URL.Query().Get("grade")
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	structures, err := h.service.ListFeeStructures(r.Context(), tenantID, grade, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, structures)
}

func (h *Handler) createFeeStructure(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateFeeStructureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	structure, err := h.service.CreateFeeStructure(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, structure)
}

func (h *Handler) getFeeStructure(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid fee structure ID")
		return
	}
	structure, err := h.service.GetFeeStructure(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, structure)
}

func (h *Handler) updateFeeStructure(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid fee structure ID")
		return
	}
	var req UpdateFeeStructureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	structure, err := h.service.UpdateFeeStructure(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, structure)
}

func (h *Handler) deleteFeeStructure(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid fee structure ID")
		return
	}
	if err := h.service.DeleteFeeStructure(r.Context(), tenantID, id); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

func (h *Handler) addFeeItem(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid fee structure ID")
		return
	}
	var input FeeItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	item, err := h.service.AddFeeItem(r.Context(), tenantID, id, input, sortOrder)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, item)
}

func (h *Handler) deleteFeeItem(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid fee item ID")
		return
	}
	if err := h.service.DeleteFeeItem(r.Context(), tenantID, itemID); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// --- Invoice handlers ---

func (h *Handler) listInvoices(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	status := r.URL.Query().Get("status")
	learnerID := r.URL.Query().Get("learner_id")
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	invoices, err := h.service.ListInvoices(r.Context(), tenantID, status, learnerID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, invoices)
}

func (h *Handler) createInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	invoice, err := h.service.CreateInvoice(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, invoice)
}

func (h *Handler) getInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid invoice ID")
		return
	}
	invoice, err := h.service.GetInvoice(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, invoice)
}

func (h *Handler) updateInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid invoice ID")
		return
	}
	var req UpdateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	invoice, err := h.service.UpdateInvoice(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, invoice)
}

func (h *Handler) deleteInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid invoice ID")
		return
	}
	if err := h.service.DeleteInvoice(r.Context(), tenantID, id); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

func (h *Handler) listInvoicePayments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	invoiceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid invoice ID")
		return
	}
	payments, err := h.service.ListInvoicePayments(r.Context(), tenantID, invoiceID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, payments)
}

// --- Discount handlers ---

func (h *Handler) listDiscounts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	invoiceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid invoice ID")
		return
	}
	discounts, err := h.service.ListDiscounts(r.Context(), tenantID, invoiceID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, discounts)
}

func (h *Handler) createDiscount(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	invoiceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid invoice ID")
		return
	}
	var req CreateDiscountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	discount, err := h.service.CreateDiscount(r.Context(), tenantID, invoiceID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, discount)
}

func (h *Handler) deleteDiscount(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	discountID, err := uuid.Parse(chi.URLParam(r, "discountId"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid discount ID")
		return
	}
	if err := h.service.DeleteDiscount(r.Context(), tenantID, discountID); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// --- Payment handlers ---

func (h *Handler) listPayments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	status := r.URL.Query().Get("status")
	channel := r.URL.Query().Get("channel")
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	payments, err := h.service.ListPayments(r.Context(), tenantID, status, channel, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, payments)
}

func (h *Handler) createPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	payment, err := h.service.CreatePayment(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, payment)
}

func (h *Handler) getPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid payment ID")
		return
	}
	payment, err := h.service.GetPayment(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, payment)
}

func (h *Handler) reversePayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid payment ID")
		return
	}
	payment, err := h.service.ReversePayment(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondBadRequest(w, "REVERSE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, payment)
}

// --- M-Pesa handlers ---

func (h *Handler) initiateSTKPush(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	if h.mpesa == nil {
		httputil.RespondBadRequest(w, "MPESA_NOT_CONFIGURED", "M-Pesa is not configured")
		return
	}
	var req MpesaStkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	payment, err := h.mpesa.InitiateSTKPush(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "STK_PUSH_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, payment)
}

// mpesaCallback handles the Safaricom Daraja STK push callback (no auth).
type mpesaCallbackPayload struct {
	Body struct {
		StkCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			ResultCode        string `json:"ResultCode"`
			ResultDesc        string `json:"ResultDesc"`
			CallbackMetadata  struct {
				Item []struct {
					Name  string `json:"Name"`
					Value any    `json:"Value"`
				} `json:"Item"`
			} `json:"CallbackMetadata"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

func (h *Handler) mpesaCallback(w http.ResponseWriter, r *http.Request) {
	if h.mpesa == nil {
		httputil.RespondBadRequest(w, "MPESA_NOT_CONFIGURED", "M-Pesa is not configured")
		return
	}
	var payload mpesaCallbackPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid callback body: "+err.Error())
		return
	}

	checkoutID := payload.Body.StkCallback.CheckoutRequestID
	resultCode := payload.Body.StkCallback.ResultCode
	resultDesc := payload.Body.StkCallback.ResultDesc
	var receipt string
	for _, item := range payload.Body.StkCallback.CallbackMetadata.Item {
		if item.Name == "MpesaReceiptNumber" {
			if s, ok := item.Value.(string); ok {
				receipt = s
			}
		}
	}

	// Look up tenant from the checkout request (first matching payment)
	var tenantID uuid.UUID
	err := h.service.pool.QueryRow(r.Context(),
		`SELECT tenant_id FROM payments WHERE checkout_request_id = $1 AND channel = 'mpesa' LIMIT 1`,
		checkoutID).Scan(&tenantID)
	if err != nil {
		httputil.RespondBadRequest(w, "NOT_FOUND", "Checkout request not found")
		return
	}

	if err := h.mpesa.ConfirmSTKPush(r.Context(), tenantID, checkoutID, resultCode, resultDesc, receipt); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, map[string]string{"status": "processed"})
}
