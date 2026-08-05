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

	"github.com/shule360/api/internal/comms"
	"github.com/shule360/api/internal/comms/sms"
	"github.com/shule360/api/internal/comms/whatsapp"
	"github.com/shule360/api/internal/config"
	appmiddleware "github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/internal/nemis"
	"github.com/shule360/api/pkg/backblaze"
	"github.com/shule360/api/pkg/httputil"
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
	_ = nemisClient // Reserved for learner enrollment validation

	// Initialize chatbot
	chatbot := whatsapp.NewChatbot()

	// Initialize services
	commsService := comms.NewCommsService(sb.Pool, redisClient, atClient, waClient)
	commsHandler := comms.NewHandler(commsService)

	// Initialize WhatsApp webhook handler
	waWebhook := whatsapp.NewWebhookHandler(cfg.MetaWAWebhookVerifyToken, sb.Pool, chatbot, waClient)

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
