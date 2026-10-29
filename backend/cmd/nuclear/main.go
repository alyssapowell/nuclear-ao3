// Command nuclear runs the whole Nuclear AO3 backend: accounts, works, tags,
// search, notifications and exports, all in one program on one port.
//
// Configuration comes from environment variables (see .env.example). The
// important ones:
//
//	DATABASE_URL       Postgres connection string
//	REDIS_URL          Redis host:port
//	ELASTICSEARCH_URL  search backend (search is disabled if unreachable)
//	PORT               where the API listens (default 8080)
//	METRICS_ADDR       where Prometheus metrics are served (default :9091);
//	                   keep this port private
//	STATIC_DIR         optional folder of built frontend files to serve
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"nuclear-ao3/internal/auth"
	"nuclear-ao3/internal/export"
	"nuclear-ao3/internal/notify"
	"nuclear-ao3/internal/platform"
	"nuclear-ao3/internal/search"
	"nuclear-ao3/internal/tags"
	"nuclear-ao3/internal/works"
)

func main() {
	_ = godotenv.Load() // a local .env file is optional

	db, err := platform.OpenDB()
	if err != nil {
		log.Fatalf("startup: %v", err)
	}

	// Auth comes first: everything else checks logins through it.
	// Redis databases are numbered per area so cache keys can't collide.
	var authSvc *auth.AuthService
	deps := platform.NewDeps(db, func(token string) (uuid.UUID, error) { return authSvc.ValidateUserToken(token) })
	defer deps.Close()

	if err := deps.Redis(0).Ping(context.Background()).Err(); err != nil {
		log.Fatalf("startup: Redis not reachable: %v", err)
	}
	if authSvc, err = auth.New(db, deps.Redis(0)); err != nil {
		log.Fatalf("startup: %v", err)
	}
	worksSvc, err := works.New(db, deps.Redis(1))
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	notifySvc := notify.New(db)
	worksSvc.SetNotifier(notifySvc.Notifier())
	authSvc.SetEventHandler(notifySvc.HandleEvent)
	notifySvc.Start()
	exportSvc := export.New(db)
	exportSvc.Start()
	searchSvc, err := search.New(db, deps.Redis(3))
	if err != nil {
		log.Printf("WARNING: search is disabled: %v", err)
		searchSvc = nil
	} else {
		worksSvc.SetSearchIndexer(searchSvc.IndexWorkDoc)
	}

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	// Only these proxies (Caddy, on the private Docker network) may tell us
	// the visitor's real IP via X-Forwarded-For.
	if err := r.SetTrustedProxies(strings.Split(platform.Env("TRUSTED_PROXIES",
		"127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"), ",")); err != nil {
		log.Fatalf("startup: TRUSTED_PROXIES: %v", err)
	}
	r.Use(gin.Recovery(), platform.Logger(), platform.SecurityHeaders(), platform.CORS(),
		authSvc.RateLimitMiddleware())
	mountRoutes(r, deps, services{
		auth:   authSvc,
		works:  worksSvc,
		tags:   tags.New(db, deps.Redis(2)),
		notify: notifySvc,
		export: exportSvc,
		search: searchSvc,
	})

	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		r.NoRoute(serveFrontend(dir))
	}

	go serveMetrics(platform.Env("METRICS_ADDR", ":9091"))

	srv := &http.Server{
		Addr:              ":" + platform.Env("PORT", "8080"),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	go func() {
		log.Printf("Nuclear AO3 listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// services are the areas of the app. search may be nil when Elasticsearch
// is unreachable.
type services struct {
	auth   *auth.AuthService
	works  *works.WorkService
	tags   *tags.TagService
	notify *notify.NotificationService
	export *export.ExportService
	search *search.SearchService
}

// mountRoutes registers every API route on r.
func mountRoutes(r *gin.Engine, deps *platform.Deps, s services) {
	r.GET("/health", deps.Health)
	auth.Mount(r, s.auth, deps)
	works.Mount(r, s.works, deps)
	tags.Mount(r, s.tags, deps)
	notify.Mount(r, s.notify, deps)
	export.Mount(r, s.export, deps)
	if s.search != nil {
		search.Mount(r, s.search, deps)
	}
}

// serveMetrics exposes Prometheus metrics on their own port so they are never
// reachable through the public site.
func serveMetrics(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	log.Printf("Metrics on %s/metrics", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("metrics server stopped: %v", err)
	}
}

// serveFrontend serves the built single-page app from dir. Unknown paths get
// index.html so the browser-side router can handle them; unknown /api paths
// get a JSON 404.
func serveFrontend(dir string) gin.HandlerFunc {
	fs := http.FileServer(http.Dir(dir))
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if _, err := os.Stat(filepath.Join(dir, filepath.Clean(c.Request.URL.Path))); err != nil {
			c.File(filepath.Join(dir, "index.html"))
			return
		}
		fs.ServeHTTP(c.Writer, c.Request)
	}
}
