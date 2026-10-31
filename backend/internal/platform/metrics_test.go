package platform

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"nuclear-ao3/internal/jobs"
	"nuclear-ao3/internal/testdb"
)

func TestRouteLabelUsesPatternOrUnmatched(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Metrics())
	r.GET("/works/:id", func(c *gin.Context) { c.Status(200) })

	get := func(method, path string) {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, path, nil))
	}
	get("GET", "/works/123")
	get("GET", "/works/456")
	get("GET", "/random/attacker/path-1")
	get("GET", "/random/attacker/path-2")
	get("BREW", "/works/1")

	count := func(method, route, status string) float64 {
		return testutil.ToFloat64(httpRequests.WithLabelValues(method, route, status))
	}
	if got := count("GET", "/works/:id", "200"); got != 2 {
		t.Errorf("pattern route counted %v, want 2", got)
	}
	if got := count("GET", "unmatched", "404"); got != 2 {
		t.Errorf("unmatched counted %v, want 2", got)
	}
	if got := count("OTHER", "/works/:id", "404"); got < 1 {
		// an unknown method is not routed, so it may also be "unmatched"
		if count("OTHER", "unmatched", "404") < 1 {
			t.Errorf("unknown method was not folded into OTHER")
		}
	}
}

func TestJobGaugesReflectTheQueue(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()

	if err := jobs.Enqueue(ctx, db, "email", map[string]string{"to": "a"}); err != nil {
		t.Fatal(err)
	}
	if err := jobs.EnqueueAt(ctx, db, "email", map[string]string{"to": "b"}, time.Now().Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := jobs.Enqueue(ctx, db, "export", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	// A job that gave up.
	if _, err := db.Exec(`UPDATE jobs SET failed_at = now() WHERE kind = 'export'`); err != nil {
		t.Fatal(err)
	}

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(newDBCollector(db, 3*time.Second))
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]float64{}
	for _, f := range families {
		for _, m := range f.Metric {
			key := f.GetName()
			for _, l := range m.Label {
				key += "{" + l.GetName() + "=" + l.GetValue() + "}"
			}
			got[key] = m.GetGauge().GetValue()
		}
	}
	want := map[string]float64{
		"nuclear_db_up":                     1,
		"nuclear_migrations_pending":        0,
		"nuclear_jobs_waiting{kind=email}":  2,
		"nuclear_jobs_running{kind=email}":  0,
		"nuclear_jobs_failed{kind=email}":   0,
		"nuclear_jobs_failed{kind=export}":  1,
		"nuclear_jobs_waiting{kind=export}": 0,
	}
	for k, v := range want {
		if g, ok := got[k]; !ok || g != v {
			t.Errorf("%s = %v (present %v), want %v", k, g, ok, v)
		}
	}
	if age := got["nuclear_jobs_oldest_waiting_seconds{kind=email}"]; age < 590 || age > 700 {
		t.Errorf("oldest waiting email = %vs, want about 600", age)
	}
	if age := got["nuclear_jobs_oldest_waiting_seconds{kind=export}"]; age != 0 {
		t.Errorf("failed job counted as waiting: %v", age)
	}

	// A closed database reports down, and nothing else that needs it.
	db.Close()
	reg = prometheus.NewPedanticRegistry()
	reg.MustRegister(newDBCollector(db, time.Second))
	body := &strings.Builder{}
	families, _ = reg.Gather()
	for _, f := range families {
		body.WriteString(f.GetName() + " ")
		if f.GetName() == "nuclear_db_up" && f.Metric[0].GetGauge().GetValue() != 0 {
			t.Errorf("nuclear_db_up = 1 with a closed database")
		}
	}
	if strings.Contains(body.String(), "nuclear_jobs_") {
		t.Errorf("job metrics reported for a dead database: %s", body)
	}
}
