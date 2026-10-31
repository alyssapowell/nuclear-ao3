package email

import (
	"bufio"
	"context"
	"io"
	"mime"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("SMTP_HOST", "test-smtp.local")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_FROM", "test@example.com")
	t.Setenv("SITE_URL", "https://archive.example.org/")

	c := LoadConfigFromEnv()
	if c.Host != "test-smtp.local" || c.Port != "2525" || c.From != "test@example.com" {
		t.Errorf("config %+v", c)
	}
	if c.SiteURL != "https://archive.example.org" {
		t.Errorf("SiteURL %q should lose its trailing slash", c.SiteURL)
	}
}

// fakeServer is just enough of an SMTP server to receive one message.
type fakeServer struct {
	addr string
	got  chan received
}

type received struct {
	to   string
	data string
}

func newFakeServer(t *testing.T) *fakeServer {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	s := &fakeServer{addr: ln.Addr().String(), got: make(chan received, 1)}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		say := func(line string) { io.WriteString(conn, line+"\r\n") }
		say("220 fake")
		var msg received
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "RCPT TO:"):
				msg.to = strings.TrimSpace(line[len("RCPT TO:"):])
				say("250 ok")
			case cmd == "DATA":
				say("354 go")
				var b strings.Builder
				for {
					l, err := r.ReadString('\n')
					if err != nil || l == ".\r\n" {
						break
					}
					b.WriteString(l)
				}
				msg.data = b.String()
				say("250 queued")
				s.got <- msg
			case cmd == "QUIT":
				say("221 bye")
				return
			default: // EHLO, MAIL FROM
				say("250 ok")
			}
		}
	}()
	return s
}

func (s *fakeServer) config() *Config {
	host, port, _ := net.SplitHostPort(s.addr)
	return &Config{Host: host, Port: port, From: "noreply@archive.example.org", FromName: "Test Archive",
		SiteURL: "https://archive.example.org"}
}

func TestSend(t *testing.T) {
	s := newFakeServer(t)
	err := s.config().Send(context.Background(), Message{
		To:      "reader@example.org",
		Subject: "New chapter\r\nBcc: everyone@example.org",
		Text:    "Chapter 2 is up.",
		HTML:    "<p>Chapter 2 is up.</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := <-s.got
	if got.to != "<reader@example.org>" {
		t.Errorf("RCPT TO %q", got.to)
	}
	msg, err := mail.ReadMessage(strings.NewReader(got.data))
	if err != nil {
		t.Fatalf("not a valid message: %v\n%s", err, got.data)
	}
	if msg.Header.Get("Bcc") != "" {
		t.Error("a line break in the subject added a header")
	}
	if subj, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject")); subj != "New chapter Bcc: everyone@example.org" {
		t.Errorf("subject %q", subj)
	}
	if !strings.HasPrefix(msg.Header.Get("Content-Type"), "multipart/alternative") {
		t.Errorf("content type %q", msg.Header.Get("Content-Type"))
	}
}

func TestSendRefusesBadAddresses(t *testing.T) {
	c := &Config{Host: "127.0.0.1", Port: "1", From: "noreply@archive.example.org"}
	for _, to := range []string{"", "not an address", "a@example.org\r\nRCPT TO:<b@example.org>"} {
		if err := c.Send(context.Background(), Message{To: to, Text: "hi"}); err == nil || !strings.Contains(err.Error(), "recipient") {
			t.Errorf("%q: %v", to, err)
		}
	}
}

func TestSendGivesUpOnASilentServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			defer conn.Close()
			time.Sleep(5 * time.Second) // never says hello
		}
	}()
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := &Config{Host: host, Port: port, From: "noreply@archive.example.org"}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := c.Send(ctx, Message{To: "reader@example.org", Text: "hi"}); err == nil {
		t.Fatal("expected a timeout")
	}
	if time.Since(start) > 2*time.Second {
		t.Error("Send waited too long for a server that never answered")
	}
}

func TestWelcomeEscapesTheUsername(t *testing.T) {
	c := &Config{FromName: "Test Archive", SiteURL: "https://archive.example.org"}
	m, err := c.Welcome("new@example.org", `<script>alert(1)</script>`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.HTML, "<script>") {
		t.Error("the username went into the HTML unescaped")
	}
	if !strings.Contains(m.HTML, "https://archive.example.org/dashboard") || !strings.Contains(m.Text, "https://archive.example.org/dashboard") {
		t.Error("the dashboard link should use SITE_URL")
	}
	if m.To != "new@example.org" || m.Subject != "Welcome to Test Archive!" {
		t.Errorf("%+v", m)
	}
}
