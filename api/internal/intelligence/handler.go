package intelligence

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the intelligence API.
type Handler struct {
	service *Service
	ai      *AIService
}

// NewHandler creates a new intelligence handler.
func NewHandler(service *Service, ai *AIService) *Handler {
	return &Handler{service: service, ai: ai}
}

// Mount registers all intelligence routes under the provided router.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/intelligence", func(r chi.Router) {
		// Financial analytics
		r.Get("/financial/fee-collection", h.feeCollectionSummary)
		r.Get("/financial/payment-channels", h.paymentChannelBreakdown)
		r.Get("/financial/fee-defaulters", h.feeDefaulters)
		r.Get("/financial/monthly-trend", h.monthlyCollectionTrend)

		// Communication analytics
		r.Get("/communications/campaigns", h.campaignDeliverySummary)
		r.Get("/communications/channel-reach", h.channelReach)
		r.Get("/communications/failed-numbers", h.failedNumbers)

		// AI knowledge base: FAQ
		r.Get("/ai/faq", h.listFAQEntries)
		r.Post("/ai/faq", h.createFAQEntry)
		r.Get("/ai/faq/{id}", h.getFAQEntry)
		r.Patch("/ai/faq/{id}", h.updateFAQEntry)
		r.Delete("/ai/faq/{id}", h.deleteFAQEntry)

		// AI knowledge base: template embeddings
		r.Get("/ai/templates", h.listTemplateEmbeddings)
		r.Post("/ai/templates", h.createTemplateEmbedding)
		r.Delete("/ai/templates/{id}", h.deleteTemplateEmbedding)

		// AI features
		r.Get("/ai/suggest-templates", h.suggestTemplates)
		r.Post("/ai/auto-respond", h.autoRespond)
		r.Get("/ai/portfolio-summary/{learnerId}", h.portfolioSummary)
	})
}

// --- Financial analytics handlers ---

func (h *Handler) feeCollectionSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	rows, err := h.service.FeeCollectionSummary(r.Context(), tenantID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) paymentChannelBreakdown(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	rows, err := h.service.PaymentChannelBreakdown(r.Context(), tenantID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) feeDefaulters(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	rows, err := h.service.FeeDefaulters(r.Context(), tenantID, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) monthlyCollectionTrend(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	rows, err := h.service.MonthlyCollectionTrend(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

// --- Communication analytics handlers ---

func (h *Handler) campaignDeliverySummary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	rows, err := h.service.CampaignDeliverySummary(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) channelReach(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	rows, err := h.service.ChannelReach(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) failedNumbers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	rows, err := h.service.FailedNumbers(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

// --- FAQ handlers ---

func (h *Handler) listFAQEntries(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	category := r.URL.Query().Get("category")

	rows, err := h.service.ListFAQEntries(r.Context(), tenantID, category)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) getFAQEntry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid FAQ entry ID")
		return
	}
	entry, err := h.service.GetFAQEntry(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, entry)
}

func (h *Handler) createFAQEntry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateFAQEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	if req.Question == "" || req.Answer == "" {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "question and answer are required")
		return
	}
	entry, err := h.service.CreateFAQEntry(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondCreated(w, entry)
}

func (h *Handler) updateFAQEntry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid FAQ entry ID")
		return
	}
	var req UpdateFAQEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	entry, err := h.service.UpdateFAQEntry(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, entry)
}

func (h *Handler) deleteFAQEntry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid FAQ entry ID")
		return
	}
	if err := h.service.DeleteFAQEntry(r.Context(), tenantID, id); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// --- Template embedding handlers ---

func (h *Handler) listTemplateEmbeddings(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	rows, err := h.service.ListTemplateEmbeddings(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) createTemplateEmbedding(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateTemplateEmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	if req.Content == "" {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "content is required")
		return
	}
	entry, err := h.service.CreateTemplateEmbedding(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondCreated(w, entry)
}

func (h *Handler) deleteTemplateEmbedding(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid template embedding ID")
		return
	}
	if err := h.service.DeleteTemplateEmbedding(r.Context(), tenantID, id); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// --- AI feature handlers ---

func (h *Handler) suggestTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	purpose := r.URL.Query().Get("purpose")
	tone := r.URL.Query().Get("tone")
	language := r.URL.Query().Get("language")
	topK, _ := strconv.Atoi(r.URL.Query().Get("top_k"))

	rows, err := h.ai.SuggestTemplates(r.Context(), tenantID, purpose, tone, language, topK)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

func (h *Handler) autoRespond(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	resp, err := h.ai.AutoRespond(r.Context(), tenantID, req.Query)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, resp)
}

func (h *Handler) portfolioSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	learnerID, err := uuid.Parse(chi.URLParam(r, "learnerId"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	if term < 1 || term > 3 || year <= 0 {
		httputil.RespondBadRequest(w, "INVALID_PARAM", "term (1-3) and year are required")
		return
	}
	summary, err := h.service.PortfolioSummary(r.Context(), tenantID, learnerID, term, year)
	if err != nil {
		httputil.RespondBadRequest(w, "SUMMARY_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, summary)
}
