package finance

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the finance API.
type Handler struct {
	service *Service
	mpesa   *MpesaService
	// webhookToken is the secret part of the address Safaricom reports to.
	webhookToken string
}

// NewHandler creates a new finance handler.
func NewHandler(service *Service, mpesa *MpesaService, webhookToken string) *Handler {
	return &Handler{service: service, mpesa: mpesa, webhookToken: strings.TrimSpace(webhookToken)}
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

	r.Route("/fee-categories", func(r chi.Router) {
		r.Get("/", h.listFeeCategories)
		r.Post("/", h.createFeeCategory)
		r.Get("/{id}", h.getFeeCategory)
		r.Patch("/{id}", h.updateFeeCategory)
	})

	r.Route("/invoices", func(r chi.Router) {
		r.Get("/", h.listInvoices)
		r.Post("/", h.createInvoice)
		r.Post("/bulk", h.bulkInvoice)
		r.Get("/{id}", h.getInvoice)
		r.Patch("/{id}", h.updateInvoice)
		r.Post("/{id}/void", h.voidInvoice)

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

	r.Route("/finance", func(r chi.Router) {
		r.Get("/summary", h.summary)
		r.Get("/arrears", h.arrears)
		r.Get("/learners/{id}/statement", h.statement)
		r.Get("/paybill", h.listInbox)
		r.Get("/paybill/{id}", h.getInbox)
		r.Post("/paybill/{id}/allocate", h.allocateInbox)
	})
}

// MountWebhooks registers the addresses Safaricom reports to. They carry no
// session; see webhookAllowed.
func (h *Handler) MountWebhooks(r chi.Router) {
	r.Post("/webhooks/mpesa/stk", h.mpesaCallback)
	r.Post("/webhooks/mpesa/{token}/stk", h.mpesaCallback)
	r.Post("/webhooks/mpesa/{token}/c2b/validation", h.c2bValidation)
	r.Post("/webhooks/mpesa/{token}/c2b/confirmation", h.c2bConfirmation)
}

// webhookAllowed checks the secret in the address. Safaricom does not sign
// what it sends, so the address itself is the credential (compared in
// constant time), on top of the source-address list the router applies.
//
// Once a token is configured the address without one stops working. Paybill
// confirmations create money records, so they are never accepted without one.
func (h *Handler) webhookAllowed(r *http.Request, tokenRequired bool) bool {
	given := chi.URLParam(r, "token")
	if h.webhookToken == "" {
		return !tokenRequired && given == ""
	}
	return subtle.ConstantTimeCompare([]byte(given), []byte(h.webhookToken)) == 1
}

// respond maps a service error to the response the caller can act on.
func respond(w http.ResponseWriter, err error, what string) {
	var (
		validation    *ValidationError
		conflicting   *ConflictError
		notConfigured *NotConfiguredError
	)
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.RespondNotFound(w, "NOT_FOUND", what+" not found")
	case errors.As(err, &validation):
		httputil.RespondBadRequest(w, "INVALID", validation.Message)
	case errors.As(err, &conflicting):
		httputil.RespondConflict(w, "CONFLICT", conflicting.Message)
	case errors.As(err, &notConfigured):
		httputil.RespondBadRequest(w, "MPESA_NOT_CONFIGURED", notConfigured.Reason)
	default:
		httputil.RespondInternalError(w, err)
	}
}

// actor is the staff member making the request. A platform or group user has
// a staff row in the school they have opened, so this is set for them too.
func actor(r *http.Request) *uuid.UUID {
	if id, ok := middleware.GetStaffID(r); ok && id != uuid.Nil {
		return &id
	}
	return nil
}

// school returns the school the request is for, or answers 401.
func school(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
	}
	return tenantID, ok
}

// pathID reads a record id from the address, or answers 404: an id that is
// not an id names nothing.
func pathID(w http.ResponseWriter, r *http.Request, name, what string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", what+" not found")
		return uuid.Nil, false
	}
	return id, true
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(into); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "The request could not be read.")
		return false
	}
	return true
}

func queryInt(r *http.Request, name string) int {
	n, _ := strconv.Atoi(r.URL.Query().Get(name))
	return n
}

// --- Fee structure handlers ---

func (h *Handler) listFeeStructures(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	structures, err := h.service.ListFeeStructures(r.Context(), tenantID,
		r.URL.Query().Get("grade"), queryInt(r, "term"), queryInt(r, "year"))
	if err != nil {
		respond(w, err, "Fee structure")
		return
	}
	httputil.RespondOK(w, structures)
}

func (h *Handler) createFeeStructure(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	var req CreateFeeStructureRequest
	if !decode(w, r, &req) {
		return
	}
	fs, err := h.service.CreateFeeStructure(r.Context(), tenantID, actor(r), req)
	if err != nil {
		respond(w, err, "Fee structure")
		return
	}
	httputil.RespondCreated(w, fs)
}

func (h *Handler) getFeeStructure(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Fee structure")
	if !ok {
		return
	}
	fs, err := h.service.GetFeeStructure(r.Context(), tenantID, id)
	if err != nil {
		respond(w, err, "Fee structure")
		return
	}
	httputil.RespondOK(w, fs)
}

func (h *Handler) updateFeeStructure(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Fee structure")
	if !ok {
		return
	}
	var req UpdateFeeStructureRequest
	if !decode(w, r, &req) {
		return
	}
	fs, err := h.service.UpdateFeeStructure(r.Context(), tenantID, id, req)
	if err != nil {
		respond(w, err, "Fee structure")
		return
	}
	httputil.RespondOK(w, fs)
}

func (h *Handler) deleteFeeStructure(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Fee structure")
	if !ok {
		return
	}
	if err := h.service.DeleteFeeStructure(r.Context(), tenantID, id); err != nil {
		respond(w, err, "Fee structure")
		return
	}
	httputil.RespondNoContent(w)
}

func (h *Handler) addFeeItem(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Fee structure")
	if !ok {
		return
	}
	var req FeeItemInput
	if !decode(w, r, &req) {
		return
	}
	fs, err := h.service.AddFeeItem(r.Context(), tenantID, id, req)
	if err != nil {
		respond(w, err, "Fee structure")
		return
	}
	httputil.RespondCreated(w, fs)
}

func (h *Handler) deleteFeeItem(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "itemId", "Fee item")
	if !ok {
		return
	}
	if err := h.service.DeleteFeeItem(r.Context(), tenantID, id); err != nil {
		respond(w, err, "Fee item")
		return
	}
	httputil.RespondNoContent(w)
}

// --- Fee item handlers ---

func (h *Handler) listFeeCategories(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListFeeCategories(r.Context(), tenantID, r.URL.Query().Get("status"))
	if err != nil {
		respond(w, err, "Fee item")
		return
	}
	httputil.RespondOK(w, items)
}

func (h *Handler) getFeeCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Fee item")
	if !ok {
		return
	}
	item, err := h.service.GetFeeCategory(r.Context(), tenantID, id)
	if err != nil {
		respond(w, err, "Fee item")
		return
	}
	httputil.RespondOK(w, item)
}

func (h *Handler) createFeeCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	var req FeeCategoryInput
	if !decode(w, r, &req) {
		return
	}
	item, err := h.service.CreateFeeCategory(r.Context(), tenantID, req)
	if err != nil {
		respond(w, err, "Fee item")
		return
	}
	httputil.RespondCreated(w, item)
}

func (h *Handler) updateFeeCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Fee item")
	if !ok {
		return
	}
	var req FeeCategoryInput
	if !decode(w, r, &req) {
		return
	}
	item, err := h.service.UpdateFeeCategory(r.Context(), tenantID, id, req)
	if err != nil {
		respond(w, err, "Fee item")
		return
	}
	httputil.RespondOK(w, item)
}

// --- Invoice handlers ---

func (h *Handler) listInvoices(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	invoices, err := h.service.ListInvoices(r.Context(), tenantID, InvoiceFilter{
		Status: q.Get("status"), LearnerID: q.Get("learner_id"), Grade: q.Get("grade"), Search: q.Get("search"),
		Term: queryInt(r, "term"), Year: queryInt(r, "year"), Limit: queryInt(r, "limit"), Offset: queryInt(r, "offset"),
	})
	if err != nil {
		respond(w, err, "Invoice")
		return
	}
	httputil.RespondOK(w, invoices)
}

func (h *Handler) createInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	var req CreateInvoiceRequest
	if !decode(w, r, &req) {
		return
	}
	invoice, err := h.service.CreateInvoice(r.Context(), tenantID, actor(r), req)
	if err != nil {
		respond(w, err, "Invoice")
		return
	}
	httputil.RespondCreated(w, invoice)
}

func (h *Handler) bulkInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	var req BulkInvoiceRequest
	if !decode(w, r, &req) {
		return
	}
	result, err := h.service.BulkInvoice(r.Context(), tenantID, actor(r), req)
	if err != nil {
		respond(w, err, "Fee structure")
		return
	}
	httputil.RespondOK(w, result)
}

func (h *Handler) getInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Invoice")
	if !ok {
		return
	}
	invoice, err := h.service.GetInvoice(r.Context(), tenantID, id)
	if err != nil {
		respond(w, err, "Invoice")
		return
	}
	httputil.RespondOK(w, invoice)
}

func (h *Handler) updateInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Invoice")
	if !ok {
		return
	}
	var req UpdateInvoiceRequest
	if !decode(w, r, &req) {
		return
	}
	invoice, err := h.service.UpdateInvoice(r.Context(), tenantID, id, req)
	if err != nil {
		respond(w, err, "Invoice")
		return
	}
	httputil.RespondOK(w, invoice)
}

func (h *Handler) voidInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Invoice")
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if !decode(w, r, &req) {
		return
	}
	invoice, err := h.service.VoidInvoice(r.Context(), tenantID, actor(r), id, req.Reason)
	if err != nil {
		respond(w, err, "Invoice")
		return
	}
	httputil.RespondOK(w, invoice)
}

func (h *Handler) listInvoicePayments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Invoice")
	if !ok {
		return
	}
	payments, err := h.service.ListInvoicePayments(r.Context(), tenantID, id)
	if err != nil {
		respond(w, err, "Invoice")
		return
	}
	httputil.RespondOK(w, payments)
}

func (h *Handler) listDiscounts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Invoice")
	if !ok {
		return
	}
	discounts, err := h.service.ListDiscounts(r.Context(), tenantID, id)
	if err != nil {
		respond(w, err, "Invoice")
		return
	}
	httputil.RespondOK(w, discounts)
}

func (h *Handler) createDiscount(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Invoice")
	if !ok {
		return
	}
	var req CreateDiscountRequest
	if !decode(w, r, &req) {
		return
	}
	discount, err := h.service.CreateDiscount(r.Context(), tenantID, actor(r), id, req)
	if err != nil {
		respond(w, err, "Invoice")
		return
	}
	httputil.RespondCreated(w, discount)
}

func (h *Handler) deleteDiscount(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "discountId", "Discount")
	if !ok {
		return
	}
	if err := h.service.DeleteDiscount(r.Context(), tenantID, id); err != nil {
		respond(w, err, "Discount")
		return
	}
	httputil.RespondNoContent(w)
}

// --- Payment handlers ---

func (h *Handler) listPayments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	payments, err := h.service.ListPayments(r.Context(), tenantID, PaymentFilter{
		Status: q.Get("status"), Channel: q.Get("channel"), Search: q.Get("search"),
		Term: queryInt(r, "term"), Year: queryInt(r, "year"), Limit: queryInt(r, "limit"), Offset: queryInt(r, "offset"),
	})
	if err != nil {
		respond(w, err, "Payment")
		return
	}
	httputil.RespondOK(w, payments)
}

func (h *Handler) createPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	var req CreatePaymentRequest
	if !decode(w, r, &req) {
		return
	}
	payment, err := h.service.CreatePayment(r.Context(), tenantID, actor(r), req)
	if err != nil {
		respond(w, err, "Invoice")
		return
	}
	httputil.RespondCreated(w, payment)
}

func (h *Handler) getPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Payment")
	if !ok {
		return
	}
	payment, err := h.service.GetPayment(r.Context(), tenantID, id)
	if err != nil {
		respond(w, err, "Payment")
		return
	}
	httputil.RespondOK(w, payment)
}

func (h *Handler) reversePayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Payment")
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if !decode(w, r, &req) {
		return
	}
	payment, err := h.service.ReversePayment(r.Context(), tenantID, actor(r), id, req.Reason)
	if err != nil {
		respond(w, err, "Payment")
		return
	}
	httputil.RespondOK(w, payment)
}

func (h *Handler) initiateSTKPush(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	var req MpesaStkRequest
	if !decode(w, r, &req) {
		return
	}
	payment, err := h.mpesa.InitiateSTKPush(r.Context(), tenantID, actor(r), req)
	if err != nil {
		respond(w, err, "Invoice")
		return
	}
	httputil.RespondCreated(w, payment)
}

// --- Position handlers ---

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	out, err := h.service.Summary(r.Context(), tenantID, queryInt(r, "term"), queryInt(r, "year"))
	if err != nil {
		respond(w, err, "Summary")
		return
	}
	httputil.RespondOK(w, out)
}

func (h *Handler) arrears(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	rows, err := h.service.Arrears(r.Context(), tenantID, q.Get("grade"), q.Get("search"), queryInt(r, "limit"), queryInt(r, "offset"))
	if err != nil {
		respond(w, err, "Arrears")
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) statement(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Learner")
	if !ok {
		return
	}
	st, err := h.service.Statement(r.Context(), tenantID, id)
	if err != nil {
		respond(w, err, "Learner")
		return
	}
	httputil.RespondOK(w, st)
}

// --- Paybill handlers ---

func (h *Handler) listInbox(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	rows, err := h.service.ListInbox(r.Context(), tenantID, q.Get("status"), q.Get("search"), queryInt(r, "limit"), queryInt(r, "offset"))
	if err != nil {
		respond(w, err, "Paybill payment")
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) getInbox(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Paybill payment")
	if !ok {
		return
	}
	payment, err := h.service.GetInboxPayment(r.Context(), tenantID, id)
	if err != nil {
		respond(w, err, "Paybill payment")
		return
	}
	allocations, err := h.service.InboxAllocations(r.Context(), tenantID, id)
	if err != nil {
		respond(w, err, "Paybill payment")
		return
	}
	httputil.RespondOK(w, map[string]any{"payment": payment, "allocations": allocations})
}

func (h *Handler) allocateInbox(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id", "Paybill payment")
	if !ok {
		return
	}
	var req AllocateRequest
	if !decode(w, r, &req) {
		return
	}
	payment, err := h.service.Allocate(r.Context(), tenantID, actor(r), id, req)
	if err != nil {
		respond(w, err, "Paybill payment")
		return
	}
	httputil.RespondOK(w, payment)
}

// --- Safaricom webhooks ---

// loose reads a JSON value Safaricom sends sometimes as a number and
// sometimes as text (ResultCode, TransAmount, MSISDN).
type loose string

func (l *loose) UnmarshalJSON(data []byte) error {
	*l = loose(strings.Trim(strings.TrimSpace(string(data)), `"`))
	if *l == "null" {
		*l = ""
	}
	return nil
}

type stkCallbackPayload struct {
	Body struct {
		StkCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			ResultCode        loose  `json:"ResultCode"`
			ResultDesc        string `json:"ResultDesc"`
			CallbackMetadata  struct {
				Item []struct {
					Name  string `json:"Name"`
					Value loose  `json:"Value"`
				} `json:"Item"`
			} `json:"CallbackMetadata"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

// accepted is the answer Safaricom expects; anything else makes it retry.
func accepted(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Accepted"}`))
}

// mpesaCallback receives the outcome of an STK request.
func (h *Handler) mpesaCallback(w http.ResponseWriter, r *http.Request) {
	if !h.webhookAllowed(r, false) {
		http.NotFound(w, r)
		return
	}
	var payload stkCallbackPayload
	if !decode(w, r, &payload) {
		return
	}
	callback := payload.Body.StkCallback
	if callback.CheckoutRequestID == "" {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "CheckoutRequestID is required.")
		return
	}
	result := STKResult{
		CheckoutRequestID: callback.CheckoutRequestID,
		ResultCode:        string(callback.ResultCode),
		ResultDesc:        callback.ResultDesc,
	}
	for _, item := range callback.CallbackMetadata.Item {
		if item.Name == "MpesaReceiptNumber" {
			result.Receipt = string(item.Value)
		}
	}

	changed, err := h.mpesa.ConfirmSTKPush(r.Context(), result)
	if err != nil {
		// A 5xx makes Safaricom retry, which is what we want when the
		// database is briefly unavailable.
		slog.Error("mpesa callback: could not record the result", "checkout_request_id", result.CheckoutRequestID, "error", err)
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	if !changed {
		// Unknown here, or settled already. A paid result nobody is waiting
		// for is money to look into, so it is logged loudly.
		level := slog.LevelInfo
		if result.ResultCode == "0" {
			level = slog.LevelWarn
		}
		slog.Log(r.Context(), level, "mpesa callback: nothing to update",
			"checkout_request_id", result.CheckoutRequestID, "result_code", result.ResultCode, "receipt", result.Receipt)
	}
	accepted(w)
}

type c2bPayload struct {
	TransID           string `json:"TransID"`
	TransTime         loose  `json:"TransTime"`
	TransAmount       loose  `json:"TransAmount"`
	BusinessShortCode loose  `json:"BusinessShortCode"`
	BillRefNumber     string `json:"BillRefNumber"`
	MSISDN            loose  `json:"MSISDN"`
	FirstName         string `json:"FirstName"`
	MiddleName        string `json:"MiddleName"`
	LastName          string `json:"LastName"`
}

// c2bValidation is asked before Safaricom completes a paybill payment. Every
// payment is accepted: refusing a parent's money because an account number is
// mistyped helps nobody, and the bursar can allocate it afterwards.
func (h *Handler) c2bValidation(w http.ResponseWriter, r *http.Request) {
	if !h.webhookAllowed(r, true) {
		http.NotFound(w, r)
		return
	}
	accepted(w)
}

// c2bConfirmation receives a completed paybill payment.
func (h *Handler) c2bConfirmation(w http.ResponseWriter, r *http.Request) {
	if !h.webhookAllowed(r, true) {
		http.NotFound(w, r)
		return
	}
	var payload c2bPayload
	if !decode(w, r, &payload) {
		return
	}
	cents, err := ParseAmount(string(payload.TransAmount))
	if err != nil || payload.TransID == "" {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "TransID and TransAmount are required.")
		return
	}
	err = h.service.ReceiveC2B(r.Context(), C2BConfirmation{
		TransID:           payload.TransID,
		TransTime:         string(payload.TransTime),
		AmountCents:       cents,
		BusinessShortCode: string(payload.BusinessShortCode),
		BillRefNumber:     payload.BillRefNumber,
		MSISDN:            string(payload.MSISDN),
		PayerName:         strings.Join(strings.Fields(payload.FirstName+" "+payload.MiddleName+" "+payload.LastName), " "),
	})
	var validation *ValidationError
	switch {
	case err == nil:
	case errors.Is(err, ErrUnknownPaybill):
		// Retrying will not help; a person has to look at it.
		slog.Error("mpesa c2b: payment to a paybill no single school owns",
			"trans_id", payload.TransID, "short_code", string(payload.BusinessShortCode), "bill_ref", payload.BillRefNumber, "amount_cents", cents)
	case errors.As(err, &validation):
		httputil.RespondBadRequest(w, "INVALID_REQUEST", validation.Message)
		return
	default:
		slog.Error("mpesa c2b: could not record the payment", "trans_id", payload.TransID, "error", err)
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	accepted(w)
}
