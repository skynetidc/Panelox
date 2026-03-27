package core

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/config"
	"github.com/jakub/panelox/internal/core/nodes"
	"github.com/jakub/panelox/internal/core/proxy"
	"github.com/jakub/panelox/internal/server/middleware"
)

// Core is the central panel that routes requests to agents.
type Core struct {
	cfg        *config.Config
	router     *chi.Mux
	pool       *pgxpool.Pool
	localProxy *httputil.ReverseProxy
}

func New(cfg *config.Config, pool *pgxpool.Pool) *Core {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.CORS(cfg.CORSOrigins))

	c := &Core{
		cfg:        cfg,
		router:     r,
		pool:       pool,
		localProxy: proxy.NewLocalProxy(),
	}

	c.registerRoutes()
	return c
}

func (c *Core) registerRoutes() {
	// Health check
	c.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","role":"core"}`))
	})

	// Serve frontend static files
	frontendDir := "/opt/panelox/frontend"
	if _, err := os.Stat(frontendDir); err == nil {
		fileServer := http.FileServer(http.Dir(frontendDir))
		c.router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			// Don't serve API routes as files
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			// Try serving file, fall back to index.html (SPA)
			path := frontendDir + r.URL.Path
			if _, err := os.Stat(path); os.IsNotExist(err) {
				http.ServeFile(w, r, frontendDir+"/index.html")
				return
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	// Node management (core-only endpoints)
	nodeHandler := nodes.NewHandler(c.pool)
	c.router.Route("/api/v1", func(r chi.Router) {
		nodes.RegisterRoutes(r, nodeHandler)
		nodes.RegisterAdvancedRoutes(r, nodeHandler)
	})

	// ALL /api/v1/* requests (except /nodes) → proxy to agent
	c.router.HandleFunc("/api/v1/*", c.proxyHandler)
}

// proxyHandler routes API requests to the correct agent (local or remote).
func (c *Core) proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Determine target node from query param or header
	nodeName := r.URL.Query().Get("node")
	if nodeName == "" {
		nodeName = r.Header.Get("X-Panelox-Node")
	}
	if nodeName == "" {
		nodeName = "local"
	}

	// Local node → Unix socket proxy
	if nodeName == "local" {
		c.localProxy.ServeHTTP(w, r)
		return
	}

	// Remote node → SSH tunnel proxy
	node, err := nodes.GetNodeByName(r.Context(), c.pool, nodeName)
	if err != nil {
		http.Error(w, `{"error":{"code":"node_not_found","message":"unknown node"}}`, http.StatusBadRequest)
		return
	}

	sshClient, err := nodes.ConnectToNode(node)
	if err != nil {
		log.Printf("SSH connection to %s failed: %v", nodeName, err)
		http.Error(w, `{"error":{"code":"node_offline","message":"cannot connect to node"}}`, http.StatusBadGateway)
		return
	}
	defer sshClient.Close()

	remoteProxy := proxy.NewRemoteProxy(sshClient, node.AgentPort)
	remoteProxy.ServeHTTP(w, r)
}

func (c *Core) Start() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", c.cfg.Port),
		Handler:      c.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Panelox Core starting on :%d", c.cfg.Port)
		log.Printf("Proxying API requests to local agent via Unix socket")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Core error: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down core...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
