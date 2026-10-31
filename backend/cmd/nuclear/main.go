// Command nuclear runs the whole Nuclear AO3 backend: accounts, works, tags,
// search, notifications and exports, all in one program on one port.
//
// Configuration comes from environment variables (see .env.example). The
// important ones:
//
//	DATABASE_URL       Postgres connection string
//	PORT               where the API listens (default 8080)
//	METRICS_ADDR       where Prometheus metrics are served (default :9091);
//	                   keep this port private
//	STATIC_DIR         optional folder of built frontend files to serve
//	SITE_URL           the archive's public address, for links in emails
//	SMTP_HOST, SMTP_PORT, SMTP_FROM, SMTP_USERNAME, SMTP_PASSWORD
//	                   the mail server emails are sent through
//	SIGNING_KEY_FILE   the PEM file with the key that signs logins (required;
//	                   create it with "nuclear keys generate")
//	SIGNING_KEY_PREVIOUS_FILES
//	                   optional comma-separated list of old key files, accepted
//	                   only to check logins made before a planned key change
//	JOB_WORKERS        how many background jobs run at once (default 2)
//	MIGRATE_ON_START   set to "false" to skip applying database migrations
//	                   at startup (then run "nuclear migrate" yourself)
//
// Subcommands:
//
//	nuclear                   run the server (applies pending migrations first)
//	nuclear migrate           apply pending migrations and exit
//	nuclear migrate status    list migrations that haven't been applied
//	nuclear jobs              show background jobs: how many are waiting,
//	                          running and failed, and the latest failures
//	nuclear jobs retry [id]   put a failed job (or, with no id, every failed
//	                          job) back in the queue
//	nuclear keys generate [path]
//	                          create a new signing key file (default
//	                          secrets/signing-key-DATE.pem) and print what to
//	                          do with it. Needs no database.
//	nuclear keys show         print the ID of the current signing key and of
//	                          each previous key, to match against a saved copy
//	nuclear sessions revoke-all
//	                          end every login session and revoke every OAuth
//	                          token, unused code and app consent, so everyone has to log in again
//	nuclear sessions revoke <username>
//	                          the same for one account
//	nuclear grant <username> <role>
//	                          give an existing account a role: admin,
//	                          moderator or tag_wrangler. This is how a new
//	                          archive gets its first admin: register normally,
//	                          then run this on the server.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"nuclear-ao3/internal/auth"
	"nuclear-ao3/internal/export"
	"nuclear-ao3/internal/html"
	"nuclear-ao3/internal/jobs"
	"nuclear-ao3/internal/notify"
	"nuclear-ao3/internal/platform"
	"nuclear-ao3/internal/search"
	"nuclear-ao3/internal/tags"
	"nuclear-ao3/internal/works"
	"nuclear-ao3/migrations"
	"nuclear-ao3/shared/email"
)

func main() {
	_ = godotenv.Load() // a local .env file is optional

	// Key commands work without a database.
	if len(os.Args) > 1 && os.Args[1] == "keys" {
		os.Exit(runKeys(os.Args[2:]))
	}

	db, err := platform.OpenDB()
	if err != nil {
		log.Fatalf("startup: %v", err)
	}

	if len(os.Args) > 1 {
		os.Exit(runCommand(db, os.Args[1:]))
	}
	// Refuse to run a public archive on example settings: example passwords,
	// email links to localhost, the development mail catcher.
	if problems := platform.Preflight(os.Getenv); len(problems) > 0 {
		fatal := false
		for _, p := range problems {
			if p.Fatal {
				fatal = true
				log.Printf("startup: %s", p)
			} else {
				log.Printf("warning (allowed because ENVIRONMENT=development): %s", p)
			}
		}
		if fatal {
			log.Fatalf("startup: fix the settings above in .env and start again. On your own computer, set ENVIRONMENT=development to allow them.")
		}
	}
	if platform.Env("MIGRATE_ON_START", "true") != "false" {
		if _, err := migrations.Up(context.Background(), db); err != nil {
			log.Fatalf("startup: %v", err)
		}
	}

	// Auth comes first: everything else checks logins through it.
	var authSvc *auth.AuthService
	deps := platform.NewDeps(db, func(token string) (uuid.UUID, error) { return authSvc.ValidateUserToken(token) })
	defer deps.Close()

	if authSvc, err = auth.New(db); err != nil {
		log.Fatalf("startup: %v", err)
	}
	worksSvc, err := works.New(db)
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	mail := email.LoadConfigFromEnv()
	notifySvc := notify.New(db, mail)
	worksSvc.SetNotifier(notifySvc.Notifier())
	authSvc.SetMail(mail)
	notifySvc.Start()
	exportSvc := export.New(db)
	exportSvc.Start()
	searchSvc := search.New(db)

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
	platform.RegisterDBMetrics(db)
	r.Use(platform.Metrics(), gin.Recovery(), platform.Logger(), platform.SecurityHeaders(), platform.CORS(),
		authSvc.RateLimitMiddleware())
	mountRoutes(r, deps, services{
		auth:   authSvc,
		works:  worksSvc,
		tags:   tags.New(db),
		notify: notifySvc,
		export: exportSvc,
		search: searchSvc,
	})

	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		r.NoRoute(serveFrontend(dir))
	}

	go serveMetrics(platform.Env("METRICS_ADDR", ":9091"))

	stopWorkers := startWorkers(db, mail)
	stopHits := worksSvc.StartHitFlusher()

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
	// After the server stops, so views from the last requests are counted.
	stopHits()
	stopWorkers()
}

// startWorkers runs background jobs until the returned function is called.
// That function waits for jobs that are running to finish.
func startWorkers(db *sql.DB, mail *email.Config) (stop func()) {
	q := jobs.New(db)
	q.Handle("email", func(ctx context.Context, payload json.RawMessage) error {
		var m email.Message
		if err := json.Unmarshal(payload, &m); err != nil {
			return err
		}
		return mail.Send(ctx, m)
	})
	// Queued once by migration 031: cleans HTML that was saved before it was
	// cleaned on write. Safe to run again.
	q.Handle("clean-html", func(ctx context.Context, _ json.RawMessage) error {
		changed, err := html.CleanStored(ctx, db)
		log.Printf("clean-html: rewrote %d rows", changed)
		return err
	})

	n, err := strconv.Atoi(platform.Env("JOB_WORKERS", "2"))
	if err != nil || n < 1 {
		log.Fatalf("startup: JOB_WORKERS must be a number of at least 1")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	for i := 0; i < n; i++ {
		go func() {
			q.Run(ctx)
			done <- struct{}{}
		}()
	}
	return func() {
		cancel()
		for i := 0; i < n; i++ {
			<-done
		}
	}
}

// runCommand runs a subcommand and returns the process exit code.
func runCommand(db *sql.DB, args []string) int {
	ctx := context.Background()
	switch {
	case len(args) == 1 && args[0] == "migrate":
		applied, err := migrations.Up(ctx, db)
		if err != nil {
			log.Printf("migrate: %v", err)
			return 1
		}
		if len(applied) == 0 {
			log.Println("migrate: database is up to date")
		}
		return 0
	case len(args) == 2 && args[0] == "migrate" && args[1] == "status":
		pending, err := migrations.Pending(ctx, db)
		if err != nil {
			log.Printf("migrate: %v", err)
			return 1
		}
		if len(pending) == 0 {
			fmt.Println("up to date")
		}
		for _, v := range pending {
			fmt.Println("pending:", v)
		}
		return 0
	case args[0] == "jobs" && len(args) == 1:
		return showJobs(ctx, db)
	case args[0] == "jobs" && args[1] == "retry" && len(args) <= 3:
		var id int64
		if len(args) == 3 {
			var err error
			if id, err = strconv.ParseInt(args[2], 10, 64); err != nil {
				log.Printf("jobs retry: %q is not a job id", args[2])
				return 2
			}
		}
		n, err := jobs.Retry(ctx, db, id)
		if err != nil {
			log.Printf("jobs retry: %v", err)
			return 1
		}
		log.Printf("jobs retry: %d job(s) queued again", n)
		return 0
	case len(args) == 2 && args[0] == "sessions" && args[1] == "revoke-all":
		return revokeSessions(ctx, db, "")
	case len(args) == 3 && args[0] == "sessions" && args[1] == "revoke":
		return revokeSessions(ctx, db, args[2])
	case len(args) == 3 && args[0] == "grant":
		if err := grantRole(ctx, db, args[1], args[2]); err != nil {
			log.Printf("grant: %v", err)
			return 1
		}
		log.Printf("grant: %s now has the %s role", args[1], args[2])
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\nusage: nuclear [migrate [status] | jobs [retry [id]] | sessions revoke-all | sessions revoke <username> | grant <username> <role> | keys generate|show]\n", strings.Join(args, " "))
		return 2
	}
}

// showJobs prints what the background job queue is doing.
func showJobs(ctx context.Context, db *sql.DB) int {
	summary, err := jobs.Summarize(ctx, db)
	if err != nil {
		log.Printf("jobs: %v", err)
		return 1
	}
	if len(summary) == 0 {
		fmt.Println("no jobs")
	}
	fmt.Printf("%-12s %8s %8s %8s  %s\n", "KIND", "WAITING", "RUNNING", "FAILED", "OLDEST WAITING")
	for _, s := range summary {
		oldest := "-"
		if s.Oldest != nil {
			oldest = time.Since(*s.Oldest).Round(time.Second).String() + " ago"
			if s.Oldest.After(time.Now()) {
				oldest = "retrying in " + time.Until(*s.Oldest).Round(time.Second).String()
			}
		}
		fmt.Printf("%-12s %8d %8d %8d  %s\n", s.Kind, s.Waiting, s.Running, s.Failed, oldest)
	}
	failures, err := jobs.Failures(ctx, db, 20)
	if err != nil {
		log.Printf("jobs: %v", err)
		return 1
	}
	if len(failures) > 0 {
		fmt.Println("\nlatest failures (retry with: nuclear jobs retry <id>, or all with: nuclear jobs retry):")
	}
	for _, f := range failures {
		fmt.Printf("  #%d %s, gave up %s after %d attempts: %s\n", f.ID, f.Kind,
			f.FailedAt.Format(time.RFC3339), f.Attempts, firstLine(f.Error))
	}
	return 0
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// revokeSessions ends login sessions and revokes OAuth tokens: for every
// account when username is empty, otherwise for that one account. It does
// all of it or none of it, and prints how many of each it ended.
func revokeSessions(ctx context.Context, db *sql.DB, username string) int {
	n, err := RevokeSessions(ctx, db, username)
	if err != nil {
		log.Printf("sessions: %v", err)
		return 1
	}
	who := "every account"
	if username != "" {
		who = username
	}
	fmt.Printf("revoked %d login session(s), %d access token(s), %d refresh token(s), %d unused authorization code(s) and %d app consent(s) for %s\n",
		n.Sessions, n.AccessTokens, n.RefreshTokens, n.Codes, n.Consents, who)
	return 0
}

// Revoked says how many of each thing RevokeSessions ended.
type Revoked struct{ Sessions, AccessTokens, RefreshTokens, Codes, Consents int64 }

// RevokeSessions does the work of "nuclear sessions revoke[-all]". Only
// sessions and tokens that were still live are counted.
func RevokeSessions(ctx context.Context, db *sql.DB, username string) (Revoked, error) {
	var r Revoked
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return r, err
	}
	defer tx.Rollback()

	// With no username, "$1 = ''" is true for every row.
	if username != "" {
		var id string
		if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE username = $1`, username).Scan(&id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return r, fmt.Errorf("no account named %q", username)
			}
			return r, err
		}
	}
	for _, q := range []struct {
		dst *int64
		sql string
	}{
		{&r.Sessions, `UPDATE user_sessions SET is_active = false
			WHERE is_active AND ($1 = '' OR user_id = (SELECT id FROM users WHERE username = $1))`},
		{&r.AccessTokens, `UPDATE oauth_access_tokens SET is_revoked = true, revoked_at = now()
			WHERE NOT is_revoked AND ($1 = '' OR user_id = (SELECT id FROM users WHERE username = $1))`},
		{&r.RefreshTokens, `UPDATE oauth_refresh_tokens SET is_revoked = true, revoked_at = now()
			WHERE NOT is_revoked AND ($1 = '' OR user_id = (SELECT id FROM users WHERE username = $1))`},
		// An unused code could still be swapped for new tokens.
		{&r.Codes, `UPDATE authorization_codes SET used_at = now()
			WHERE used_at IS NULL AND ($1 = '' OR user_id = (SELECT id FROM users WHERE username = $1))`},
		// Apps must ask the user for permission again.
		{&r.Consents, `UPDATE user_consents SET is_revoked = true, revoked_at = now()
			WHERE NOT is_revoked AND ($1 = '' OR user_id = (SELECT id FROM users WHERE username = $1))`},
	} {
		res, err := tx.ExecContext(ctx, q.sql, username)
		if err != nil {
			return r, err
		}
		*q.dst, _ = res.RowsAffected()
	}
	return r, tx.Commit()
}

// grantRole gives an existing user a role, or restores it if it was revoked.
func grantRole(ctx context.Context, db *sql.DB, username, role string) error {
	switch role {
	case "admin", "moderator", "tag_wrangler":
	default:
		return fmt.Errorf("unknown role %q (use admin, moderator or tag_wrangler)", role)
	}
	if _, err := migrations.Up(ctx, db); err != nil {
		return err
	}
	res, err := db.ExecContext(ctx, `
		INSERT INTO user_roles (user_id, role)
		SELECT id, $2 FROM users WHERE username = $1
		ON CONFLICT (user_id, role) DO UPDATE SET revoked_at = NULL, revoked_by = NULL, granted_at = NOW()`,
		username, role)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("no account named %q; register it on the site first", username)
	}
	return nil
}

// services are the areas of the app.
type services struct {
	auth   *auth.AuthService
	works  *works.WorkService
	tags   *tags.TagService
	notify *notify.NotificationService
	export *export.ExportService
	search *search.Service
}

// mountRoutes registers every API route on r.
func mountRoutes(r *gin.Engine, deps *platform.Deps, s services) {
	r.GET("/health", deps.Health)
	auth.Mount(r, s.auth, deps)
	works.Mount(r, s.works, deps)
	tags.Mount(r, s.tags, deps)
	notify.Mount(r, s.notify, deps)
	export.Mount(r, s.export, deps)
	search.Mount(r, s.search, deps)
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
