package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/shule360/api/internal/academic"
	"github.com/shule360/api/internal/academic/assessment"
	"github.com/shule360/api/internal/academic/attendance"
	"github.com/shule360/api/internal/academic/curriculum"
	"github.com/shule360/api/internal/comms"
	"github.com/shule360/api/internal/comms/sms"
	"github.com/shule360/api/internal/comms/whatsapp"
	"github.com/shule360/api/internal/config"
	"github.com/shule360/api/internal/finance"
	"github.com/shule360/api/internal/hr"
	"github.com/shule360/api/internal/intelligence"
	"github.com/shule360/api/internal/learner"
	appmiddleware "github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/internal/nemis"
	"github.com/shule360/api/internal/procurement"
	"github.com/shule360/api/internal/reports"
	"github.com/shule360/api/internal/transport"
	"github.com/shule360/api/pkg/backblaze"
	"github.com/shule360/api/pkg/httputil"
	"github.com/shule360/api/pkg/mpesa"
	supabaseclient "github.com/shule360/api/pkg/supabase"
	"github.com/shule360/api/pkg/upstash"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup logging
	if cfg.IsProduction() {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	}

	ctx := context.Background()

	// Initialize Supabase client (pgx pool + auth)
	sb, err := supabaseclient.NewClient(ctx, cfg.DatabaseURL, cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer sb.Close()
	slog.Info("database connected")

	// Initialize Upstash clients
	redisClient := upstash.NewRedisClient(cfg.UpstashRedisURL, cfg.UpstashRedisToken)
	vectorClient := upstash.NewVectorClient(cfg.UpstashVectorURL, cfg.UpstashVectorToken)
	_ = vectorClient // Reserved for Phase 2 template suggestion
	searchClient := upstash.NewSearchClient(cfg.UpstashSearchURL, cfg.UpstashSearchToken)
	_ = searchClient // Reserved for learner/supplier full-text search

	// Initialize Backblaze B2
	b2Client, err := backblaze.NewB2Client(ctx, cfg.B2AccountID, cfg.B2ApplicationKey, cfg.B2BucketName, cfg.B2Endpoint)
	if err != nil {
		slog.Error("failed to initialize Backblaze B2", "error", err)
		os.Exit(1)
	}
	_ = b2Client

	// Initialize Africa's Talking SMS client
	atClient := sms.NewATClient(cfg.ATAPIKey, cfg.ATUsername, cfg.ATSenderID, cfg.IsProduction())

	// Initialize WhatsApp Cloud API client
	waClient := whatsapp.NewWAClient(cfg.MetaWAToken, cfg.MetaWAPhoneNumberID)

	// Initialize NEMIS client (sandbox for development)
	nemisClient := nemis.NewSandboxNEMISClient()

	// Initialize chatbot
	chatbot := whatsapp.NewChatbot()

	// Initialize comms service
	commsService := comms.NewCommsService(sb.Pool, redisClient, atClient, waClient)
	commsHandler := comms.NewHandler(commsService)

	// Initialize WhatsApp webhook handler
	waWebhook := whatsapp.NewWebhookHandler(cfg.MetaWAWebhookVerifyToken, sb.Pool, chatbot, waClient)

	// Initialize academic services
	curriculumSvc := curriculum.NewService(sb.Pool)
	assessmentSvc := assessment.NewService(sb.Pool)
	attendanceSvc := attendance.NewService(sb.Pool)
	academicHandler := academic.NewHandler(curriculumSvc, assessmentSvc, attendanceSvc)

	// Initialize learner services (EPIC 3)
	learnerSvc := learner.NewService(sb.Pool, nemisClient)
	learnerHandler := learner.NewHandler(learnerSvc)

	// Initialize M-Pesa Daraja client (EPIC 5)
	mpesaClient := mpesa.NewClient(cfg.MpesaConsumerKey, cfg.MpesaConsumerSecret,
		cfg.MpesaPasskey, cfg.MpesaShortCode, cfg.MpesaBaseURL)

	// Initialize transport services (EPIC 4)
	transportSvc := transport.NewService(sb.Pool)
	transportHandler := transport.NewHandler(transportSvc)

	// Initialize finance services (EPIC 5)
	financeSvc := finance.NewService(sb.Pool)
	financeMpesa := finance.NewMpesaService(sb.Pool, mpesaClient, cfg.MpesaCallbackURL)
	financeHandler := finance.NewHandler(financeSvc, financeMpesa)

	// Initialize reports & analytics services (EPIC 6)
	reportsSvc := reports.NewService(sb.Pool)
	reportsHandler := reports.NewHandler(reportsSvc)

	// Initialize HR services (EPIC 7)
	hrSvc := hr.NewService(sb.Pool)
	hrHandler := hr.NewHandler(hrSvc)

	// Initialize procurement services (EPIC 8)
	procurementSvc := procurement.NewService(sb.Pool)
	procurementHandler := procurement.NewHandler(procurementSvc)

	// Initialize intelligence services (EPIC 8: Digital Intelligence)
	intelligenceSvc := intelligence.NewService(sb.Pool)
	intelligenceAI := intelligence.NewAIService(sb.Pool, vectorClient)
	intelligenceHandler := intelligence.NewHandler(intelligenceSvc, intelligenceAI)

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(RecoverMiddleware)
	r.Use(middleware.Logger)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondOK(w, map[string]string{"status": "ok"})
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Webhooks (no auth)
		r.Handle("/webhooks/whatsapp", waWebhook)
		r.Post("/webhooks/sms/dlr", handleSMSDLR)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.Auth(cfg.JWTSecret))
			r.Use(appmiddleware.TenantRequired)
			r.Use(appmiddleware.RateLimit(redisClient, 100, 10))

			commsHandler.Mount(r)
			academicHandler.Mount(r)
			learnerHandler.Mount(r)
			transportHandler.Mount(r)
			financeHandler.Mount(r)
			reportsHandler.Mount(r)
			hrHandler.Mount(r)
			procurementHandler.Mount(r)
			intelligenceHandler.Mount(r)
		})
	})

	// HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		slog.Info("server starting", "port", cfg.Port, "env", cfg.AppEnv)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}
	slog.Info("server stopped")
}

// RecoverMiddleware recovers from panics and returns a 500 error.
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", rec, "path", r.URL.Path)
				httputil.RespondError(w, http.StatusInternalServerError, "PANIC", "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// handleSMSDLR handles Africa's Talking delivery receipt callbacks.
func handleSMSDLR(w http.ResponseWriter, r *http.Request) {
	slog.Info("sms dlr callback received")
	// TODO: Parse AT delivery receipt and update message_logs
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
