package platform

import (
	"fmt"
	"net/url"
	"strings"
)

// Problem is one thing wrong with the server's settings.
type Problem struct {
	Setting string
	Message string
	Fatal   bool // the server must not start with this
}

func (p Problem) String() string {
	return p.Setting + ": " + p.Message
}

// weakPasswords are passwords that ship in examples and tutorials, so
// they are the first ones anyone tries.
var weakPasswords = map[string]bool{
	"": true, "ao3_password": true, "password": true, "postgres": true, "changeme": true,
	"secret": true, "admin": true, "root": true, "example": true,
	"generate-a-secure-password-here": true,
}

// Preflight checks the settings for mistakes that would make a public
// archive unsafe or broken, such as an example password or email links that
// point at localhost. getenv is os.Getenv, or a stand-in for tests.
//
// Unless ENVIRONMENT is "development", every problem is fatal, so a server
// set up from the examples and never finished refuses to start instead of
// running insecurely. In development the same problems are only reported.
func Preflight(getenv func(string) string) []Problem {
	env := getenv("ENVIRONMENT")
	if env == "" {
		env = "production"
	}
	dev := env == "development"
	var out []Problem
	add := func(setting, msg string) {
		out = append(out, Problem{Setting: setting, Message: msg, Fatal: !dev})
	}
	if env != "production" && env != "development" {
		out = append(out, Problem{"ENVIRONMENT", fmt.Sprintf("is %q; use production or development", env), true})
	}

	dbURL := getenv("DATABASE_URL")
	if dbURL == "" {
		add("DATABASE_URL", "is not set, so the app would use the built-in example database password")
	} else if u, err := url.Parse(dbURL); err == nil {
		pw, _ := u.User.Password()
		switch {
		case weakPasswords[strings.ToLower(pw)]:
			add("DATABASE_URL", "uses an example or empty database password; set POSTGRES_PASSWORD in .env to a long random one (for example: openssl rand -base64 24)")
		case len(pw) < 16:
			add("DATABASE_URL", "uses a database password shorter than 16 characters; use a long random one (for example: openssl rand -base64 24)")
		}
	}

	switch site := strings.TrimRight(getenv("SITE_URL"), "/"); {
	case site == "":
		add("SITE_URL", "is not set, so links in emails would point at localhost; set it to the archive's public address, such as https://archive.example.org")
	case !strings.HasPrefix(site, "https://"):
		add("SITE_URL", fmt.Sprintf("is %q; it must start with https:// so links in emails are encrypted", site))
	default:
		if u, err := url.Parse(site); err != nil || isLocalHost(u.Hostname()) {
			add("SITE_URL", fmt.Sprintf("is %q, which only works on this machine; use the archive's public address", site))
		}
	}

	switch strings.ToLower(getenv("SMTP_HOST")) {
	case "", "smtp", "mailhog":
		add("SMTP_HOST", "points at the development mail catcher, which keeps every email instead of sending it; set the SMTP_* settings to a real mail service")
	}

	if gm := getenv("GIN_MODE"); gm == "test" {
		out = append(out, Problem{"GIN_MODE", "is test, which turns off rate limits and approves OAuth app access without asking; unset it", true})
	}
	return out
}

func isLocalHost(host string) bool {
	host = strings.ToLower(host)
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0" ||
		strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local")
}
