package platform

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"nuclear-ao3/internal/jobs"
	"nuclear-ao3/migrations"
)

// Metrics for Prometheus. They use the default registry, which is the one
// the metrics port (METRICS_ADDR) serves.
var (
	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Requests handled, by method, route pattern and status code.",
	}, []string{"method", "route", "status"})

	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "How long requests took, by method and route pattern.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"method", "route"})
)

func init() {
	prometheus.MustRegister(httpRequests, httpDuration)
}

// unmatchedRoute is the route label for requests no route handled. Using one
// fixed word keeps strangers from creating endless label values by asking
// for random URLs.
const unmatchedRoute = "unmatched"

// RouteLabel returns the label for a request's route: the route pattern
// (such as "/api/v1/works/:id"), never the real URL.
func RouteLabel(c *gin.Context) string {
	if p := c.FullPath(); p != "" {
		return p
	}
	return unmatchedRoute
}

// methodLabel keeps clients from inventing label values with made-up methods.
func methodLabel(m string) string {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return m
	}
	return "OTHER"
}

// Metrics records request counts and durations. Register it first, so it
// also times the other middleware.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		method, route := methodLabel(c.Request.Method), RouteLabel(c)
		httpRequests.WithLabelValues(method, route, strconv.Itoa(c.Writer.Status())).Inc()
		httpDuration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
	}
}

// dbCollector asks the database its questions when Prometheus scrapes, so the
// numbers are never stale. Each scrape is limited to a few seconds; a slow or
// dead database shows up as nuclear_db_up 0 rather than a hung scrape.
type dbCollector struct {
	db      *sql.DB
	timeout time.Duration

	up, waiting, running, failed, oldest, pending *prometheus.Desc
	open, inUse, idle, maxOpen                    *prometheus.Desc
	waitCount, waitSeconds                        *prometheus.Desc
}

// RegisterDBMetrics adds the database and job-queue metrics to the default
// registry.
func RegisterDBMetrics(db *sql.DB) {
	prometheus.MustRegister(newDBCollector(db, 3*time.Second))
}

func newDBCollector(db *sql.DB, timeout time.Duration) *dbCollector {
	d := func(name, help string, labels ...string) *prometheus.Desc {
		return prometheus.NewDesc(name, help, labels, nil)
	}
	return &dbCollector{
		db: db, timeout: timeout,
		up:          d("nuclear_db_up", "1 if the database answered, 0 if not."),
		waiting:     d("nuclear_jobs_waiting", "Background jobs waiting to run (including ones waiting to retry).", "kind"),
		running:     d("nuclear_jobs_running", "Background jobs running now.", "kind"),
		failed:      d("nuclear_jobs_failed", "Background jobs that gave up after too many tries.", "kind"),
		oldest:      d("nuclear_jobs_oldest_waiting_seconds", "How long the longest-waiting job has been due. 0 if none is due.", "kind"),
		pending:     d("nuclear_migrations_pending", "Database migrations that exist but have not been applied."),
		open:        d("nuclear_db_connections_open", "Open database connections (in use plus idle)."),
		inUse:       d("nuclear_db_connections_in_use", "Database connections in use."),
		idle:        d("nuclear_db_connections_idle", "Database connections idle."),
		maxOpen:     d("nuclear_db_connections_max", "Most connections the app may open."),
		waitCount:   d("nuclear_db_connection_waits_total", "Times a request had to wait for a free database connection."),
		waitSeconds: d("nuclear_db_connection_wait_seconds_total", "Total time spent waiting for a free database connection."),
	}
}

// Describe sends nothing: the job metrics have labels that depend on the data,
// so this is an "unchecked" collector.
func (c *dbCollector) Describe(chan<- *prometheus.Desc) {}

func (c *dbCollector) Collect(ch chan<- prometheus.Metric) {
	gauge := func(d *prometheus.Desc, v float64, labels ...string) {
		ch <- prometheus.MustNewConstMetric(d, prometheus.GaugeValue, v, labels...)
	}
	st := c.db.Stats()
	gauge(c.open, float64(st.OpenConnections))
	gauge(c.inUse, float64(st.InUse))
	gauge(c.idle, float64(st.Idle))
	gauge(c.maxOpen, float64(st.MaxOpenConnections))
	ch <- prometheus.MustNewConstMetric(c.waitCount, prometheus.CounterValue, float64(st.WaitCount))
	ch <- prometheus.MustNewConstMetric(c.waitSeconds, prometheus.CounterValue, st.WaitDuration.Seconds())

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	if err := c.db.PingContext(ctx); err != nil {
		gauge(c.up, 0)
		return
	}
	gauge(c.up, 1)

	// If a query below fails, its metrics are left out rather than reported
	// as zero, so a gap is never mistaken for "all clear".
	if summary, err := jobs.Summarize(ctx, c.db); err == nil {
		for _, s := range summary {
			gauge(c.waiting, float64(s.Waiting), s.Kind)
			gauge(c.running, float64(s.Running), s.Kind)
			gauge(c.failed, float64(s.Failed), s.Kind)
			var age float64
			if s.Oldest != nil && s.Waiting > 0 {
				if age = time.Since(*s.Oldest).Seconds(); age < 0 {
					age = 0 // not due yet (waiting to retry)
				}
			}
			gauge(c.oldest, age, s.Kind)
		}
	}
	if pending, err := migrations.Pending(ctx, c.db); err == nil {
		gauge(c.pending, float64(len(pending)))
	}
}
