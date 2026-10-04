package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"github.com/shule360/api/internal/academic"
	"github.com/shule360/api/internal/academic/assessment"
	"github.com/shule360/api/internal/academic/attendance"
	"github.com/shule360/api/internal/academic/curriculum"
	"github.com/shule360/api/internal/auth"
	"github.com/shule360/api/internal/comms"
	"github.com/shule360/api/internal/comms/contacts"
	"github.com/shule360/api/internal/comms/sms"
	"github.com/shule360/api/internal/comms/whatsapp"
	"github.com/shule360/api/internal/config"
	"github.com/shule360/api/internal/finance"
	"github.com/shule360/api/internal/guardian_auth"
	"github.com/shule360/api/internal/hr"
	"github.com/shule360/api/internal/intelligence"
	"github.com/shule360/api/internal/learner"
	"github.com/shule360/api/internal/learnerimport"
	appmiddleware "github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/internal/nemis"
	"github.com/shule360/api/internal/onboarding"
	"github.com/shule360/api/internal/parent"
	"github.com/shule360/api/internal/platform"
	"github.com/shule360/api/internal/procurement"
	"github.com/shule360/api/internal/reports"
	"github.com/shule360/api/internal/security"
	"github.com/shule360/api/internal/settings"
	"github.com/shule360/api/internal/teacher"
	"github.com/shule360/api/internal/tenant"
	"github.com/shule360/api/internal/transport"
	"github.com/shule360/api/pkg/backblaze"
	"github.com/shule360/api/pkg/groq"
	"github.com/shule360/api/pkg/httputil"
	supabaseclient "github.com/shule360/api/pkg/supabase"
	"github.com/shule360/api/pkg/upstash"
)

func main() {
	_ = godotenv.Load(".env")

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

	// Sentry error tracking (no-op unless SENTRY_DSN is configured).
	sentryDsn := os.Getenv("SENTRY_DSN")
	if sentryDsn != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentryDsn,
			Environment:      cfg.AppEnv,
			Release:          "shule360-api@" + version,
			AttachStacktrace: true,
		}); err != nil {
			slog.Warn("failed to initialize Sentry; continuing without error tracking", "error", err)
		} else {
			defer sentry.Flush(2 * time.Second)
			appmiddleware.EnableSentryCapture(func(message string, r *http.Request) {
				hub := sentry.GetHubFromContext(r.Context())
				if hub == nil {
					hub = sentry.CurrentHub()
				}
				hub.CaptureException(errors.New(message))
			})
			slog.Info("sentry error tracking enabled")
		}
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
	if cfg.UpstashRedisURL == "" || cfg.UpstashRedisToken == "" {
		slog.Warn("UPSTASH_REDIS_REST_URL/TOKEN not set: login rate limiting fails closed, so every login attempt will return 503")
	}
	vectorClient := upstash.NewVectorClient(cfg.UpstashVectorURL, cfg.UpstashVectorToken)
	_ = vectorClient // Reserved for Phase 2 template suggestion
	searchClient := upstash.NewSearchClient(cfg.UpstashSearchURL, cfg.UpstashSearchToken)
	_ = searchClient // Reserved for learner/supplier full-text search

	// Initialize Backblaze B2
	b2Client, err := backblaze.NewB2Client(ctx, cfg.B2AccountID, cfg.B2ApplicationKey, cfg.B2BucketName, cfg.B2Endpoint)
	if err != nil {
		slog.Warn("Backblaze B2 not initialized; document uploads unavailable", "error", err)
	}
	_ = b2Client

	// Initialize SMS service (templates; sending is the dispatcher below)
	smsService := sms.NewSMSService(sb.Pool)

	// Initialize WhatsApp Cloud API client
	waClient := whatsapp.NewWAClient(cfg.MetaWAToken, cfg.MetaWAPhoneNumberID)

	// Initialize NEMIS client (sandbox for development)
	nemisClient := nemis.NewSandboxNEMISClient()

	// Initialize chatbot
	chatbot := whatsapp.NewChatbot()

	// Tenant settings come first: the SMS dispatcher resolves each school's
	// own Africa's Talking credentials through them. Credentials are sealed
	// with a key derived from SETTINGS_ENCRYPTION_KEY, falling back to
	// JWT_SECRET.
	settingsSecret := cfg.SettingsEncryptionKey
	if settingsSecret == "" {
		settingsSecret = cfg.JWTSecret
	}
	settingsSealer, sealErr := settings.NewSealer(settingsSecret)
	if sealErr != nil {
		slog.Warn("credential encryption unavailable; settings can be read but not configured", "error", sealErr)
	}
	settingsService := settings.NewService(sb.Pool, settingsSealer)

	// The SMS dispatcher sends every recorded message: at once when notified,
	// on a 30s sweep for scheduled messages, and at startup for anything a
	// restart interrupted.
	smsDispatcher := comms.NewDispatcher(sb.Pool, comms.NewSenderFactory(sb.Pool, settingsService, comms.PlatformSMS{
		APIKey:   cfg.ATAPIKey,
		Username: cfg.ATUsername,
		SenderID: cfg.ATSenderID,
		BaseURL:  cfg.ATBaseURL,
	}))
	dispatchCtx, stopDispatch := context.WithCancel(context.Background())
	defer stopDispatch()
	go smsDispatcher.Run(dispatchCtx)

	smsDLR := sms.NewDLRHandler(sb.Pool, cfg.ATDLRToken)
	if cfg.ATDLRToken == "" {
		slog.Warn("AT_DLR_TOKEN not set: SMS delivery reports are refused, so messages stay at 'sent' and never show 'delivered'")
	}

	// Initialize comms service
	commsService := comms.NewCommsService(sb.Pool, smsDispatcher)
	commsHandler := comms.NewHandlerWithSMS(commsService, smsService)

	// Initialize the contact book (Communications → Contacts). Staff curate
	// this before sending: it is the address book behind the "contacts"
	// audience in the SMS/WhatsApp composer.
	contactsService := contacts.NewService(sb.Pool)
	contactsHandler := contacts.NewHandler(contactsService)
	commsHandler.SetContactsHandler(contactsHandler)

	// Initialize WhatsApp webhook handler
	waWebhook := whatsapp.NewWebhookHandler(cfg.MetaWAWebhookVerifyToken, sb.Pool, chatbot, waClient)

	// Initialize academic services
	curriculumSvc := curriculum.NewService(sb.Pool)
	assessmentSvc := assessment.NewService(sb.Pool)
	attendanceSvc := attendance.NewService(sb.Pool)
	absenceAlertSvc := attendance.NewAbsenceAlertService(sb.Pool, commsService)
	academicHandler := academic.NewHandler(curriculumSvc, assessmentSvc, attendanceSvc, absenceAlertSvc)

	// Initialize learner services (EPIC 3)
	learnerSvc := learner.NewService(sb.Pool, nemisClient)
	learnerHandler := learner.NewHandler(learnerSvc)

	// Staging area for CSV learner imports. A roster is learner data, so it
	// mounts alongside the learner routes and under the same role group.
	learnerImportSvc := learnerimport.NewService(sb.Pool)
	learnerImportHandler := learnerimport.NewHandler(learnerImportSvc)

	// Initialize transport services (EPIC 4)
	transportSvc := transport.NewService(sb.Pool)
	transportHandler := transport.NewHandler(transportSvc)

	// Initialize finance services (EPIC 5)
	financeSvc := finance.NewService(sb.Pool)
	// Each school collects into its own Daraja account when it has entered
	// one; a request is recorded before Safaricom is called, and anything
	// Safaricom never reports back on is asked about every minute.
	financeMpesa := finance.NewMpesaService(sb.Pool, finance.NewGatewayFactory(settingsService, finance.PlatformMpesa{
		ConsumerKey:    cfg.MpesaConsumerKey,
		ConsumerSecret: cfg.MpesaConsumerSecret,
		Passkey:        cfg.MpesaPasskey,
		ShortCode:      cfg.MpesaShortCode,
		BaseURL:        cfg.MpesaBaseURL,
		CallbackURL:    cfg.MpesaCallbackURL,
	}))
	go financeMpesa.Run(dispatchCtx)
	financeHandler := finance.NewHandler(financeSvc, financeMpesa, cfg.MpesaWebhookToken)

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
	groqClient := groq.NewClient(cfg.GroqAPIKey, "")
	intelligenceAI := intelligence.NewAIService(sb.Pool, vectorClient, groqClient)
	intelligenceHandler := intelligence.NewHandler(intelligenceSvc, intelligenceAI)

	// Initialize security & compliance services (EPIC 9: Digital Security & Compliance)
	securitySvc := security.NewService(sb.Pool)
	securityHandler := security.NewHandler(securitySvc)

	// Initialize guardian auth (parent portal)
	guardianAuthHandler := guardian_auth.NewHandler(sb.Pool, cfg.JWTSecret, redisClient, cfg.IsProduction())

	// Initialize parent portal handler
	parentHandler := parent.NewHandler(sb.Pool)

	// Initialize teacher PWA handler
	teacherHandler := teacher.NewHandler(sb.Pool)

	// Tenant settings routes (the service is created above, with the SMS dispatcher).
	settingsHandler := settings.NewHandler(settingsService, settings.PlatformCredentials{
		MpesaConsumerKey:    cfg.MpesaConsumerKey,
		MpesaConsumerSecret: cfg.MpesaConsumerSecret,
		MpesaBaseURL:        cfg.MpesaBaseURL,
		ATAPIKey:            cfg.ATAPIKey,
		ATUsername:          cfg.ATUsername,
		MetaWAToken:         cfg.MetaWAToken,
		B2AccountID:         cfg.B2AccountID,
		B2ApplicationKey:    cfg.B2ApplicationKey,
		B2Endpoint:          cfg.B2Endpoint,
		GroqAPIKey:          cfg.GroqAPIKey,
		UpstashRedisURL:     cfg.UpstashRedisURL,
		UpstashRedisToken:   cfg.UpstashRedisToken,
	})

	// Initialize auth handler
	authHandler := auth.NewHandler(sb, cfg, redisClient)

	// Registering a school with its first administrator, the setup checklist
	// and a school's own users.
	onboardingHandler := onboarding.NewHandler(onboarding.NewService(sb.Pool, sb), redisClient, cfg.SignupMode, cfg.SignupCode)
	tenantService := tenant.NewService(sb.Pool)
	platformHandler := platform.NewHandler(platform.NewService(sb.Pool, sb))

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(appmiddleware.Metrics)
	r.Use(RecoverMiddleware)
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		// Defaults cover the production frontend, localhost dev ports, and all
		// Vercel preview deployments (*.vercel.app). CORS_ALLOWED_ORIGINS adds
		// custom origins (e.g. a dedicated app domain). Browser traffic is
		// same-origin via the web proxy, so this list mainly serves direct
		// API clients during development.
		AllowedOrigins: append([]string{
			"https://shule360.vercel.app",
			"https://*.vercel.app",
			"http://localhost:3000",
			"http://localhost:3001",
		}, cfg.CORSAllowedOrigins...),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With", "X-School-ID", "Idempotency-Key"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondOK(w, map[string]string{"status": "ok", "version": version})
	})

	// Prometheus metrics (counters + duration histograms, route-pattern labels)
	r.Get("/metrics", appmiddleware.MetricsHandler)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes (no auth required). Brute-force protection is applied
		// inside the handlers (middleware.CheckLoginRateLimit): 5 attempts per
		// IP / per account per 15-minute window, fail-closed.
		authHandler.Mount(r)
		guardianAuthHandler.Mount(r)
		onboardingHandler.MountPublic(r)

		// Webhooks (no auth). The M-Pesa callback is restricted to Daraja
		// egress IPs when MPESA_ALLOWED_IPS is configured; an empty list
		// allows all traffic (development only) and is called out at startup.
		r.Handle("/webhooks/whatsapp", waWebhook)
		smsDLR.Mount(r)
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.AllowIPs(cfg.MpesaAllowedIPs))
			financeHandler.MountWebhooks(r)
		})
		if len(cfg.MpesaAllowedIPs) == 0 {
			slog.Warn("MPESA_ALLOWED_IPS not set: M-Pesa webhook accepts any source IP (set it in production)")
		}

		// Signed in, but not necessarily inside a school: a platform or group
		// user asks these before (and in order to) choose one.
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.Auth(cfg.JWTSecret, cfg.IsProduction(), tenantService))

			// /auth/me returns the verified profile (identity from the JWT,
			// profile re-read from the DB) and the session's scope.
			authHandler.MountPrivate(r)
			r.Get("/schools", auth.Schools(tenantService))

			// Schools, groups and the users above a school: platform
			// administrators only (enforced inside Mount).
			platformHandler.Mount(r)
			r.Group(func(r chi.Router) {
				r.Use(appmiddleware.RequirePlatform)
				onboardingHandler.MountPlatform(r)
			})
		})

		// Authenticated routes inside one school
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.Auth(cfg.JWTSecret, cfg.IsProduction(), tenantService))
			r.Use(appmiddleware.TenantRequired)
			r.Use(appmiddleware.RateLimit(redisClient, 100, 10))

			// Reachable by any authenticated session: the web client asks it
			// "is this a parent session?" while hydrating.
			guardianAuthHandler.MountSession(r)

			// Role groups — mirrors the staff_role enum in
			// 002_staff_auth.sql and the frontend Sidebar ROLE_NAV map.
			allStaff := []string{"super_admin", "principal", "teacher", "bursar", "transport_manager", "hr"}
			financeRoles := []string{"super_admin", "principal", "bursar"}
			managementRoles := []string{"super_admin", "principal"}

			// Finance, procurement & financial intelligence — bursar/principal
			r.Group(func(r chi.Router) {
				r.Use(appmiddleware.RequireRole(financeRoles...))
				r.Group(func(r chi.Router) {
					r.Use(appmiddleware.RequireModule(tenantService, "finance", "Finance"))
					financeHandler.Mount(r)
				})
				procurementHandler.Mount(r)
				intelligenceHandler.Mount(r)
			})

			// HR & security/compliance — principal/super_admin only
			r.Group(func(r chi.Router) {
				r.Use(appmiddleware.RequireRole(managementRoles...))
				hrHandler.Mount(r)
				securityHandler.Mount(r)
			})

			// Academic & teaching operations — all staff roles
			r.Group(func(r chi.Router) {
				r.Use(appmiddleware.RequireRole(allStaff...))
				r.Group(func(r chi.Router) {
					r.Use(appmiddleware.RequireModule(tenantService, "communications", "Communications"))
					commsHandler.Mount(r)
				})
				onboardingHandler.MountSchool(r)
				r.Group(func(r chi.Router) {
					r.Use(appmiddleware.RequireModule(tenantService, "academic", "Academic"))
					academicHandler.Mount(r)
					reportsHandler.Mount(r)
				})
				learnerHandler.Mount(r)
				learnerImportHandler.Mount(r)
				transportHandler.Mount(r)
				teacherHandler.Mount(r)

				// Settings: every staff member can read the school's
				// configuration; only principal/super_admin can change it.
				settingsHandler.Mount(r, managementRoles...)
			})

			// Parent portal — guardians only
			r.Group(func(r chi.Router) {
				r.Use(appmiddleware.RequireRole("guardian"))
				guardianAuthHandler.MountPrivate(r) // /auth/guardian/logout
				parentHandler.Mount(r)
			})
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
// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", rec, "path", r.URL.Path)
				if sentryDsn := os.Getenv("SENTRY_DSN"); sentryDsn != "" {
					sentry.CurrentHub().PushScope()
					sentry.CaptureException(fmt.Errorf("panic: %v", rec))
				}
				httputil.RespondError(w, http.StatusInternalServerError, "PANIC", "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
