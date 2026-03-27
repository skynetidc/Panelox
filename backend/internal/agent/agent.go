package agent

import (
	"context"
	"fmt"
	"log"
	"net"
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
	"github.com/jakub/panelox/internal/config"
	"github.com/jakub/panelox/internal/container"
	"github.com/jakub/panelox/internal/cronjob"
	"github.com/jakub/panelox/internal/database"
	"github.com/jakub/panelox/internal/dashboard"
	"github.com/jakub/panelox/internal/dns"
	"github.com/jakub/panelox/internal/fail2ban"
	"github.com/jakub/panelox/internal/filemanager"
	"github.com/jakub/panelox/internal/firewall"
	"github.com/jakub/panelox/internal/ftp"
	"github.com/jakub/panelox/internal/gitdeploy"
	"github.com/jakub/panelox/internal/loginlog"
	"github.com/jakub/panelox/internal/mail"
	"github.com/jakub/panelox/internal/monitoring"
	"github.com/jakub/panelox/internal/notification"
	"github.com/jakub/panelox/internal/phpruntime"
	"github.com/jakub/panelox/internal/quota"
	"github.com/jakub/panelox/internal/pkg/claims"
	"github.com/jakub/panelox/internal/server/middleware"
	"github.com/jakub/panelox/internal/settings"
	"github.com/jakub/panelox/internal/snapshot"
	"github.com/jakub/panelox/internal/ssl"
	"github.com/jakub/panelox/internal/staging"
	"github.com/jakub/panelox/internal/terminal"
	"github.com/jakub/panelox/internal/totp"
	"github.com/jakub/panelox/internal/webhook"
	"github.com/jakub/panelox/internal/website"
)

const SockPath = "/var/run/panelox-agent.sock"

// Agent runs all the actual server management logic.
// It listens on both a Unix socket (for local core) and a TCP port (for remote core).
type Agent struct {
	cfg    *config.Config
	router *chi.Mux
	pool   *pgxpool.Pool
}

func New(cfg *config.Config, pool *pgxpool.Pool) *Agent {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	a := &Agent{cfg: cfg, router: r, pool: pool}
	a.registerRoutes()
	return a
}

func (a *Agent) registerRoutes() {
	// Health check (no auth needed for core to ping)
	a.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","role":"agent"}`))
	})

	// Agent API key auth (simple bearer token from core)
	agentKey := a.cfg.JWTSecret // Reuse JWT secret as agent API key

	validate := func(tokenStr string) (*claims.Claims, error) {
		return auth.ValidateAccessToken(a.cfg.JWTSecret, tokenStr)
	}
	getClaims := func(r *http.Request) *claims.Claims {
		return middleware.GetClaims(r.Context())
	}

	// Auth services (agent has its own user DB for token validation)
	authRepo := auth.NewRepository(a.pool)
	authSvc := auth.NewService(authRepo, a.cfg.JWTSecret)
	authHandler := auth.NewHandler(authSvc, getClaims)

	dashHandler := dashboard.NewHandler(a.pool)
	websiteRepo := website.NewRepository(a.pool)
	websiteSvc := website.NewService(websiteRepo)
	websiteHandler := website.NewHandler(websiteSvc, getClaims)
	sslRepo := ssl.NewRepository(a.pool)
	sslSvc := ssl.NewService(sslRepo)
	sslHandler := ssl.NewHandler(sslSvc, getClaims)
	fmSvc := filemanager.NewService()
	fmHandler := filemanager.NewHandler(fmSvc)
	termHandler := terminal.NewHandler(getClaims)
	dbRepo := database.NewRepository(a.pool)
	dbSvc := database.NewService(dbRepo)
	dbHandler := database.NewHandler(dbSvc, getClaims)
	dnsRepo := dns.NewRepository(a.pool)
	dnsSvc := dns.NewService(dnsRepo)
	dnsHandler := dns.NewHandler(dnsSvc, getClaims)
	fwSvc := firewall.NewService(a.pool)
	fwHandler := firewall.NewHandler(fwSvc)
	mailSvc := mail.NewService(a.pool)
	mailHandler := mail.NewHandler(mailSvc, getClaims)
	containerSvc := container.NewService()
	containerHandler := container.NewHandler(containerSvc)
	settingsHandler := settings.NewHandler(a.pool)
	cronHandler := cronjob.NewHandler(a.pool, getClaims)
	backupHandler := backup.NewHandler(a.pool, getClaims)
	monitorHandler := monitoring.NewHandler(a.pool)
	appHandler := appstore.NewHandler(a.pool, getClaims)
	phpHandler := phpruntime.NewHandler(a.pool)
	ftpHandler := ftp.NewHandler(a.pool)
	loginLogHandler := loginlog.NewHandler(a.pool)
	totpHandler := totp.NewHandler(a.pool, getClaims)
	fail2banHandler := fail2ban.NewHandler(a.pool)
	notifHandler := notification.NewHandler(a.pool)
	alertHandler := alert.NewHandler(a.pool)
	snapshotHandler := snapshot.NewHandler(a.pool)
	gitDeployHandler := gitdeploy.NewHandler(a.pool)
	webhookHandler := webhook.NewHandler(a.pool)
	billingHandler := billing.NewHandler(a.pool)
	stagingHandler := staging.NewHandler(a.pool)
	analyticsHandler := analytics.NewHandler(a.pool)
	apikeyHandler := apikey.NewHandler(a.pool, getClaims)
	quotaHandler := quota.NewHandler(a.pool, getClaims)

	// Background services
	cronScheduler := cronjob.NewScheduler(a.pool)
	cronScheduler.Start()
	metricsCollector := monitoring.NewCollector(a.pool)
	metricsCollector.Start()
	alertChecker := alert.NewChecker(a.pool)
	alertChecker.Start()
	backupRunner := backup.NewScheduleRunner(a.pool)
	backupRunner.Start()
	_ = cronScheduler
	_ = metricsCollector
	_ = alertChecker
	_ = backupRunner
	_ = agentKey

	authRL := middleware.NewRateLimiter(10, time.Minute)

	a.router.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authRL.Middleware)
			auth.RegisterPublicRoutes(r, authHandler)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuthWithAPIKey(validate, a.pool))
			r.Use(middleware.Audit(a.pool))
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
			phpruntime.RegisterRoutes(r, phpHandler)
			ftp.RegisterRoutes(r, ftpHandler)
			website.RegisterAdvancedRoutes(r, websiteHandler)
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
			cronjob.RegisterAdvancedRoutes(r, cronHandler)
		})

		// Public webhook endpoint (no JWT auth)
		gitdeploy.RegisterWebhookRoute(r, gitDeployHandler)
	})
}

// Start starts both Unix socket and TCP listeners.
func (a *Agent) Start() error {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// 1. Unix socket listener (for local core)
	os.Remove(SockPath) // Clean up old socket
	unixListener, err := net.Listen("unix", SockPath)
	if err != nil {
		log.Printf("Warning: could not create Unix socket at %s: %v", SockPath, err)
	} else {
		os.Chmod(SockPath, 0660)
		unixSrv := &http.Server{Handler: a.router}
		go func() {
			log.Printf("Agent listening on Unix socket: %s", SockPath)
			if err := unixSrv.Serve(unixListener); err != nil && err != http.ErrServerClosed {
				log.Printf("Unix socket error: %v", err)
			}
		}()
		defer unixSrv.Close()
	}

	// 2. TCP listener (for remote core connections + direct access)
	agentPort := 9090
	if a.cfg.Port > 0 {
		agentPort = a.cfg.Port
	}
	tcpSrv := &http.Server{
		Addr:         fmt.Sprintf(":%d", agentPort),
		Handler:      a.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	go func() {
		log.Printf("Agent listening on TCP :%d", agentPort)
		if err := tcpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Agent TCP error: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down agent...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	os.Remove(SockPath)
	return tcpSrv.Shutdown(ctx)
}
