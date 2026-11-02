// Package email sends mail over SMTP.
//
// Nothing calls Send directly from a web request: emails are queued as
// "email" jobs (see internal/jobs), so a slow or broken mail server delays
// mail instead of breaking pages, and a message that couldn't be sent is
// retried and, if it keeps failing, shows up in `nuclear jobs`.
package email

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"net/url"
	"os"
	"strings"
	"time"
)

// Config is where mail is sent from and through.
type Config struct {
	Host     string
	Port     string
	From     string
	FromName string
	Username string
	Password string
	// SiteURL is the archive's public address, for links in emails, such
	// as https://archive.example.org (no trailing slash).
	SiteURL string
}

// LoadConfigFromEnv reads SMTP_HOST, SMTP_PORT, SMTP_FROM, SMTP_FROM_NAME,
// SMTP_USERNAME, SMTP_PASSWORD and SITE_URL.
func LoadConfigFromEnv() *Config {
	return &Config{
		Host:     getEnv("SMTP_HOST", "mailhog"),
		Port:     getEnv("SMTP_PORT", "1025"),
		From:     getEnv("SMTP_FROM", "noreply@nuclear-ao3.local"),
		FromName: getEnv("SMTP_FROM_NAME", "Nuclear AO3"),
		Username: getEnv("SMTP_USERNAME", ""),
		Password: getEnv("SMTP_PASSWORD", ""),
		SiteURL:  strings.TrimRight(getEnv("SITE_URL", "http://localhost:8080"), "/"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Message is one email. Text is required; HTML is optional and is sent as
// an alternative to it.
type Message struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	HTML    string `json:"html,omitempty"`
}

// timeout bounds a whole send, from connecting to the last byte, when the
// caller's context has no deadline of its own.
const timeout = 30 * time.Second

// Send delivers m. It uses TLS when the server offers it (STARTTLS), or
// throughout on port 465, and only sends a password over TLS (or to a
// server on this machine).
func (c *Config) Send(ctx context.Context, m Message) error {
	to, err := mail.ParseAddress(m.To)
	if err != nil {
		return fmt.Errorf("bad recipient %q: %w", m.To, err)
	}
	from, err := mail.ParseAddress(c.From)
	if err != nil {
		return fmt.Errorf("bad SMTP_FROM %q: %w", c.From, err)
	}
	body, err := c.build(from.Address, to.Address, m)
	if err != nil {
		return err
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	addr := net.JoinHostPort(c.Host, c.Port)
	var conn net.Conn
	if c.Port == "465" {
		conn, err = (&tls.Dialer{Config: &tls.Config{ServerName: c.Host}}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("connect to mail server %s: %w", addr, err)
	}
	defer conn.Close()
	deadline, _ := ctx.Deadline()
	conn.SetDeadline(deadline)

	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return fmt.Errorf("mail server %s: %w", addr, err)
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok && c.Port != "465" {
		if err := client.StartTLS(&tls.Config{ServerName: c.Host}); err != nil {
			return fmt.Errorf("mail server %s: STARTTLS: %w", addr, err)
		}
	}
	if c.Username != "" {
		// PlainAuth refuses to send the password without TLS unless the
		// server is localhost.
		if err := client.Auth(smtp.PlainAuth("", c.Username, c.Password, c.Host)); err != nil {
			return fmt.Errorf("mail server %s: login: %w", addr, err)
		}
	}
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("mail server %s refused sender: %w", addr, err)
	}
	if err := client.Rcpt(to.Address); err != nil {
		return fmt.Errorf("mail server %s refused recipient: %w", addr, err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail server %s: %w", addr, err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("mail server %s: %w", addr, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail server %s didn't accept the message: %w", addr, err)
	}
	return client.Quit()
}

// build writes the message with its headers. Every header value is either
// a parsed address or encoded, so nothing in a subject or name can add
// headers of its own.
func (c *Config) build(fromAddr, toAddr string, m Message) ([]byte, error) {
	if strings.TrimSpace(m.Text) == "" {
		return nil, errors.New("email has no text")
	}
	var buf bytes.Buffer
	header := func(k, v string) { fmt.Fprintf(&buf, "%s: %s\r\n", k, v) }
	header("From", (&mail.Address{Name: c.FromName, Address: fromAddr}).String())
	header("To", (&mail.Address{Address: toAddr}).String())
	header("Subject", mime.QEncoding.Encode("utf-8", oneLine(m.Subject)))
	header("Date", time.Now().Format(time.RFC1123Z))
	header("Message-ID", "<"+randomID()+"@"+domain(fromAddr)+">")
	header("MIME-Version", "1.0")

	if m.HTML == "" {
		header("Content-Type", "text/plain; charset=utf-8")
		header("Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		err := writeQP(&buf, m.Text)
		return buf.Bytes(), err
	}

	mw := multipart.NewWriter(&buf)
	header("Content-Type", "multipart/alternative; boundary="+mw.Boundary())
	buf.WriteString("\r\n")
	for _, part := range []struct{ kind, body string }{{"text/plain", m.Text}, {"text/html", m.HTML}} {
		w, err := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {part.kind + "; charset=utf-8"},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return nil, err
		}
		if err := writeQP(w, part.body); err != nil {
			return nil, err
		}
	}
	err := mw.Close()
	return buf.Bytes(), err
}

func writeQP(w io.Writer, s string) error {
	qp := quotedprintable.NewWriter(w)
	if _, err := qp.Write([]byte(s)); err != nil {
		return err
	}
	return qp.Close()
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func randomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func domain(addr string) string {
	if i := strings.LastIndex(addr, "@"); i >= 0 {
		return addr[i+1:]
	}
	return "localhost"
}

// linkHTML is the body of the two account emails: a greeting, a sentence,
// a button and the plain link. html/template escapes everything in it.
var linkHTML = template.Must(template.New("link").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2>Hello {{.Username}}!</h2>
  <p>{{.Intro}}</p>
  <p><a href="{{.Link}}" style="display: inline-block; padding: 12px 24px; background: #990000; color: white; text-decoration: none; border-radius: 6px;">{{.Button}}</a></p>
  <p>If the button doesn't work, copy this address into your browser:<br>{{.Link}}</p>
  <p>{{.Outro}}</p>
  <p style="color: #6b7280; font-size: 12px;">This is an automated message. Please do not reply to it.</p>
</body>
</html>`))

// linkEmail builds an email whose point is one link carrying a token.
func (c *Config) linkEmail(to, username, subject, intro, button, path, token, outro string) (Message, error) {
	return c.linkMessage(to, username, subject, intro, button, c.SiteURL+path+"?token="+url.QueryEscape(token), outro)
}

// linkMessage builds an email whose point is one link.
func (c *Config) linkMessage(to, username, subject, intro, button, link, outro string) (Message, error) {
	username = strings.TrimSpace(username)
	var html bytes.Buffer
	err := linkHTML.Execute(&html, map[string]string{
		"Username": username, "Intro": intro, "Button": button, "Link": link, "Outro": outro,
	})
	if err != nil {
		return Message{}, err
	}
	return Message{
		To:      to,
		Subject: subject,
		Text: fmt.Sprintf("Hello %s!\n\n%s\n\n%s\n\n%s\n\nThis is an automated message. Please do not reply to it.\n",
			username, intro, link, outro),
		HTML: html.String(),
	}, nil
}

// Activation is the email that lets a new account be activated. The link is
// good for 14 days.
func (c *Config) Activation(to, username, token string) (Message, error) {
	return c.linkEmail(to, username, "Activate your account",
		"Welcome to "+c.FromName+"! Use the link below to activate your account. It works for 14 days.",
		"Activate my account", "/auth/activate", token,
		"If you didn't sign up, you can ignore this email.")
}

// PasswordReset is the email that lets someone choose a new password. The
// link is good for 6 hours.
func (c *Config) PasswordReset(to, username, token string) (Message, error) {
	return c.linkEmail(to, username, "Reset your password",
		"Someone asked to reset the password for your account. Use the link below to choose a new one. It works for 6 hours, once.",
		"Choose a new password", "/auth/reset-password", token,
		"If you didn't ask for this, you can ignore this email; your password hasn't changed.")
}

// AccountLocked tells the owner that an account refused logins for a while
// after many wrong passwords. Resetting the password is safe, and ends the
// lock. The link goes to the page where a reset is asked for.
func (c *Config) AccountLocked(to, username string) (Message, error) {
	return c.linkMessage(to, username, "Repeated failed logins on your account",
		"There were many failed attempts to log in to your account, so we have paused logins for it for a few minutes. If that was you, you can wait and try again.",
		"Reset my password", c.SiteURL+"/auth/forgot-password",
		"If it wasn't you, resetting your password is safe and ends the pause straight away.")
}

// AppReviewed tells the person who registered an app what an admin decided:
// outcome is "approved", "rejected" or "suspended". The admin's note, if
// there is one, is quoted. It never says which admin decided.
func (c *Config) AppReviewed(to, username, appName, outcome, note string) (Message, error) {
	var intro string
	switch outcome {
	case "approved":
		intro = fmt.Sprintf("Your app %q has been approved. People can now sign in to it with their %s account.", appName, c.FromName)
	case "rejected":
		intro = fmt.Sprintf("Your app %q was not approved, so it cannot be used yet.", appName)
	case "suspended":
		intro = fmt.Sprintf("Your app %q has been suspended. It cannot sign anyone in, and every access it held has been ended.", appName)
	default:
		return Message{}, fmt.Errorf("email: unknown app outcome %q", outcome)
	}
	outro := "You can see the status of your apps on the site."
	if note = strings.TrimSpace(note); note != "" {
		outro = "Note from the reviewer: " + note
	}
	return c.linkMessage(to, username, "Your app was "+outcome, intro, "Open "+c.FromName, c.SiteURL, outro)
}
