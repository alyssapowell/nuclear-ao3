package platform

import (
	"strings"
	"testing"
)

func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// good is a finished production setup.
func good() map[string]string {
	return map[string]string{
		"DATABASE_URL": "postgres://ao3_user:Zq8vN3r7Lw2pXk9sTt4m@postgres:5432/ao3_nuclear?sslmode=disable",
		"SITE_URL":     "https://archive.example.org",
		"SMTP_HOST":    "smtp.mailprovider.example",
	}
}

func TestAFinishedSetupPasses(t *testing.T) {
	if p := Preflight(envOf(good())); len(p) != 0 {
		t.Errorf("unexpected problems: %v", p)
	}
}

func TestExampleSettingsStopAProductionServer(t *testing.T) {
	cases := map[string]struct{ key, value, want string }{
		"example db password": {"DATABASE_URL", "postgres://ao3_user:ao3_password@postgres:5432/db", "example or empty"},
		"short db password":   {"DATABASE_URL", "postgres://ao3_user:hunter2xyz@postgres:5432/db", "shorter than 16"},
		"no db url":           {"DATABASE_URL", "", "not set"},
		"no site url":         {"SITE_URL", "", "not set"},
		"http site url":       {"SITE_URL", "http://archive.example.org", "https://"},
		"localhost site url":  {"SITE_URL", "https://localhost:3000", "only works on this machine"},
		"mail catcher":        {"SMTP_HOST", "smtp", "mail catcher"},
		"no mail host":        {"SMTP_HOST", "", "mail catcher"},
		"test mode":           {"GIN_MODE", "test", "rate limits"},
		"typo in environment": {"ENVIRONMENT", "prod", "production or development"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			env := good()
			env[tc.key] = tc.value
			problems := Preflight(envOf(env))
			if len(problems) != 1 || !problems[0].Fatal || !strings.Contains(problems[0].Message, tc.want) {
				t.Errorf("got %v, want one fatal problem mentioning %q", problems, tc.want)
			}
		})
	}
}

func TestDevelopmentOnlyWarns(t *testing.T) {
	env := map[string]string{
		"ENVIRONMENT":  "development",
		"DATABASE_URL": "postgres://ao3_user:ao3_password@postgres:5432/db",
		"SITE_URL":     "http://localhost",
		"SMTP_HOST":    "smtp",
	}
	problems := Preflight(envOf(env))
	if len(problems) != 3 {
		t.Fatalf("want 3 warnings, got %v", problems)
	}
	for _, p := range problems {
		if p.Fatal {
			t.Errorf("%v should only be a warning in development", p)
		}
	}
	// Test mode is never acceptable outside the test suite.
	env["GIN_MODE"] = "test"
	fatal := 0
	for _, p := range Preflight(envOf(env)) {
		if p.Fatal {
			fatal++
		}
	}
	if fatal != 1 {
		t.Errorf("GIN_MODE=test should stop even a development server")
	}
}

func TestUnsetEnvironmentMeansProduction(t *testing.T) {
	env := good()
	env["SITE_URL"] = "http://localhost"
	if p := Preflight(envOf(env)); len(p) != 1 || !p[0].Fatal {
		t.Errorf("a server with no ENVIRONMENT must be held to production rules: %v", p)
	}
}
