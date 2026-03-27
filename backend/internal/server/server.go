package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/alert"
	"github.com/jakub/panelox/internal/analytics"
	"github.com/jakub/panelox/internal/apikey"
	"github.com/jakub/panelox/internal/appstore"
	"github.com/jakub/panelox/internal/auth"
	"github.com/jakub/panelox/internal/billing"
	"github.com/jakub/panelox/internal/backup"
	"github.com/jakub/panelox/internal/clamav"
	"github.com/jakub/panelox/internal/cleanup"
	"github.com/jakub/panelox/internal/config"
	"github.com/jakub/panelox/internal/container"
	"github.com/jakub/panelox/internal/core/nodes"
	"github.com/jakub/panelox/internal/cronjob"
	"github.com/jakub/panelox/internal/database"
	"github.com/jakub/panelox/internal/dashboard"
	"github.com/jakub/panelox/internal/dns"
	"github.com/jakub/panelox/internal/fail2ban"
	"github.com/jakub/panelox/internal/filemanager"
	"github.com/jakub/panelox/internal/firewall"
	"github.com/jakub/panelox/internal/license"
	"github.com/jakub/panelox/internal/ftp"
	"github.com/jakub/panelox/internal/gitdeploy"
	"github.com/jakub/panelox/internal/loginlog"
	"github.com/jakub/panelox/internal/mail"
	"github.com/jakub/panelox/internal/monitoring"
	"github.com/jakub/panelox/internal/notification"
	"github.com/jakub/panelox/internal/phpruntime"
	"github.com/jakub/panelox/internal/process"
	"github.com/jakub/panelox/internal/sshmanage"
	"github.com/jakub/panelox/internal/quota"
	"github.com/jakub/panelox/internal/pkg/claims"
	"github.com/jakub/panelox/internal/server/middleware"
	"github.com/jakub/panelox/internal/settings"
	"github.com/jakub/panelox/internal/snapshot"
	"github.com/jakub/panelox/internal/ssl"
	"github.com/jakub/panelox/internal/staging"
	"github.com/jakub/panelox/internal/totp"
	"github.com/jakub/panelox/internal/webhook"
	"github.com/jakub/panelox/internal/terminal"
	"github.com/jakub/panelox/internal/website"
)

type Server struct {
	cfg    *config.Config
	router *chi.Mux
	pool   *pgxpool.Pool
}

func New(cfg *config.Config, pool *pgxpool.Pool) *Server {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.CORS(cfg.CORSOrigins))

	s := &Server{
		cfg:    cfg,
		router: r,
		pool:   pool,
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Token validator closure
	validate := func(tokenStr string) (*claims.Claims, error) {
		return auth.ValidateAccessToken(s.cfg.JWTSecret, tokenStr)
	}

	// Helper to get claims from request
	getClaims := func(r *http.Request) *claims.Claims {
		return middleware.GetClaims(r.Context())
	}

	// Services
	authRepo := auth.NewRepository(s.pool)
	authSvc := auth.NewService(authRepo, s.cfg.JWTSecret)
	authHandler := auth.NewHandler(authSvc, getClaims)

	dashHandler := dashboard.NewHandler(s.pool)

	websiteRepo := website.NewRepository(s.pool)
	websiteSvc := website.NewService(websiteRepo)
	websiteHandler := website.NewHandler(websiteSvc, getClaims)

	sslRepo := ssl.NewRepository(s.pool)
	sslSvc := ssl.NewService(sslRepo)
	sslHandler := ssl.NewHandler(sslSvc, getClaims)

	fmSvc := filemanager.NewService()
	fmHandler := filemanager.NewHandler(fmSvc)

	termHandler := terminal.NewHandler(getClaims)

	dbRepo := database.NewRepository(s.pool)
	dbSvc := database.NewService(dbRepo)
	dbHandler := database.NewHandler(dbSvc, getClaims)

	dnsRepo := dns.NewRepository(s.pool)
	dnsSvc := dns.NewService(dnsRepo)
	dnsHandler := dns.NewHandler(dnsSvc, getClaims)

	fwSvc := firewall.NewService(s.pool)
	fwHandler := firewall.NewHandler(fwSvc)

	mailSvc := mail.NewService(s.pool)
	mailHandler := mail.NewHandler(mailSvc, getClaims)

	containerSvc := container.NewService()
	containerHandler := container.NewHandler(containerSvc)

	settingsHandler := settings.NewHandler(s.pool)

	cronHandler := cronjob.NewHandler(s.pool, getClaims)
	backupHandler := backup.NewHandler(s.pool, getClaims)
	monitorHandler := monitoring.NewHandler(s.pool)
	appHandler := appstore.NewHandler(s.pool, getClaims)
	phpHandler := phpruntime.NewHandler(s.pool)
	ftpHandler := ftp.NewHandler(s.pool)
	loginLogHandler := loginlog.NewHandler(s.pool)
	totpHandler := totp.NewHandler(s.pool, getClaims)
	fail2banHandler := fail2ban.NewHandler(s.pool)
	notifHandler := notification.NewHandler(s.pool)
	alertHandler := alert.NewHandler(s.pool)
	snapshotHandler := snapshot.NewHandler(s.pool)
	gitDeployHandler := gitdeploy.NewHandler(s.pool)
	webhookHandler := webhook.NewHandler(s.pool)
	billingHandler := billing.NewHandler(s.pool)
	stagingHandler := staging.NewHandler(s.pool)
	analyticsHandler := analytics.NewHandler(s.pool)
	apikeyHandler := apikey.NewHandler(s.pool, getClaims)
	processHandler := process.NewHandler()
	clamavHandler := clamav.NewHandler()
	cleanupHandler := cleanup.NewHandler()
	sshHandler := sshmanage.NewHandler()
	nodeHandler := nodes.NewHandler(s.pool)
	quotaHandler := quota.NewHandler(s.pool, getClaims)
	licenseHandler := license.NewHandler(s.pool)
	licenseHandler.StartPeriodicVerification()

	alertChecker := alert.NewChecker(s.pool)
	alertChecker.Start()
	alertChecker.StartProChecks()
	backupRunner := backup.NewScheduleRunner(s.pool)
	backupRunner.Start()
	autoUpdater := appstore.NewAutoUpdateRunner(appHandler)
	autoUpdater.Start()
	_ = alertChecker
	_ = backupRunner
	_ = autoUpdater

	// Start background services
	cronScheduler := cronjob.NewScheduler(s.pool)
	cronScheduler.Start()
	metricsCollector := monitoring.NewCollector(s.pool)
	metricsCollector.Start()
	_ = cronScheduler
	_ = metricsCollector

	// Rate limiter for auth endpoints
	authRL := middleware.NewRateLimiter(10, time.Minute)

	s.router.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			r.Use(authRL.Middleware)
			auth.RegisterPublicRoutes(r, authHandler)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuthWithAPIKey(validate, s.pool))
			r.Use(middleware.Audit(s.pool))
			r.Use(license.EnforceLimits(s.pool))
			auth.RegisterRoutes(r, authHandler, middleware.RequireRole("admin"))
			dashboard.RegisterRoutes(r, dashHandler)
			website.RegisterRoutes(r, websiteHandler)
			ssl.RegisterRoutes(r, sslHandler)
			filemanager.RegisterRoutes(r, fmHandler)
			filemanager.RegisterAdvancedRoutes(r, fmHandler)
			terminal.RegisterRoutes(r, termHandler)
			database.RegisterRoutes(r, dbHandler)
			dns.RegisterRoutes(r, dnsHandler)
			firewall.RegisterRoutes(r, fwHandler)
			mail.RegisterRoutes(r, mailHandler)
			container.RegisterRoutes(r, containerHandler)
			settings.RegisterRoutes(r, settingsHandler)
			cronjob.RegisterRoutes(r, cronHandler)
			backup.RegisterRoutes(r, backupHandler)
			monitoring.RegisterRoutes(r, monitorHandler)
			monitoring.RegisterAdvancedRoutes(r, monitorHandler)
			appstore.RegisterRoutes(r, appHandler)
			appstore.RegisterAdvancedRoutes(r, appHandler)
			appstore.RegisterAutoUpdateRoutes(r, appHandler)
			phpruntime.RegisterRoutes(r, phpHandler)
			ftp.RegisterRoutes(r, ftpHandler)
			website.RegisterAdvancedRoutes(r, websiteHandler)
			website.RegisterProRoutes(r, websiteHandler)
			loginlog.RegisterRoutes(r, loginLogHandler)
			totp.RegisterRoutes(r, totpHandler)
			fail2ban.RegisterRoutes(r, fail2banHandler)
			notification.RegisterRoutes(r, notifHandler)
			alert.RegisterRoutes(r, alertHandler)
			snapshot.RegisterRoutes(r, snapshotHandler)
			container.RegisterComposeRoutes(r, containerHandler)
			container.RegisterResourceRoutes(r, containerHandler)
			database.RegisterRedisRoutes(r, dbHandler)
			backup.RegisterStorageRoutes(r, backupHandler)
			gitdeploy.RegisterRoutes(r, gitDeployHandler)
			webhook.RegisterRoutes(r, webhookHandler)
			billing.RegisterRoutes(r, billingHandler)
			staging.RegisterRoutes(r, stagingHandler)
			analytics.RegisterRoutes(r, analyticsHandler)
			quota.RegisterRoutes(r, quotaHandler)
			apikey.RegisterRoutes(r, apikeyHandler)
			process.RegisterRoutes(r, processHandler)
			clamav.RegisterRoutes(r, clamavHandler)
			cleanup.RegisterRoutes(r, cleanupHandler)
			sshmanage.RegisterRoutes(r, sshHandler)
			website.RegisterRewriteRoutes(r, websiteHandler)
			nodes.RegisterRoutes(r, nodeHandler)
			nodes.RegisterAdvancedRoutes(r, nodeHandler)
			cronjob.RegisterAdvancedRoutes(r, cronHandler)
			license.RegisterRoutes(r, licenseHandler)
		})

		// Public webhook (no auth)
		gitdeploy.RegisterWebhookRoute(r, gitDeployHandler)
	})
}

func (s *Server) Start() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.Port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Panelox server starting on :%d", s.cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return srv.Shutdown(ctx)
}
