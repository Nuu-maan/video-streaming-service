package mailer

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// dialTimeout bounds the TCP connect when the request context carries no
// deadline of its own.
const dialTimeout = 15 * time.Second

// SMTPMailer sends mail through one SMTP server using only the standard
// library.
type SMTPMailer struct {
	cfg Config
}

func NewSMTPMailer(cfg Config) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

func (m *SMTPMailer) Send(ctx context.Context, msg Message) error {
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))

	dialer := &net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dialing smtp server %s: %w", addr, err)
	}

	// net/smtp has no context support past the dial, so the request deadline
	// is pushed down onto the socket instead.
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	// Port 465 is implicit TLS: the handshake happens before the SMTP banner,
	// so STARTTLS never applies there.
	if m.cfg.Port == 465 {
		conn = tls.Client(conn, &tls.Config{ServerName: m.cfg.Host})
	}

	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp handshake with %s: %w", addr, err)
	}
	defer client.Close()

	if m.cfg.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: m.cfg.Host}); err != nil {
				return fmt.Errorf("starttls with %s: %w", addr, err)
			}
		} else if !m.cfg.AllowInsecure {
			// Refusing here rather than sending means a misconfigured server
			// can never leak reset links or credentials over cleartext.
			return fmt.Errorf("smtp server %s does not offer STARTTLS; refusing to send in cleartext", addr)
		}
	}

	if m.cfg.Username != "" {
		auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth as %s: %w", m.cfg.Username, err)
		}
	}

	if err := client.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}

	body, err := buildMIME(m.cfg.From, msg)
	if err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		w.Close()
		return fmt.Errorf("writing message body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("finishing message body: %w", err)
	}

	return client.Quit()
}

// buildMIME renders the message as an RFC 5322 payload. When both bodies are
// present they are sent as multipart/alternative so a text-only client still
// gets the link.
func buildMIME(from string, msg Message) ([]byte, error) {
	var b strings.Builder
	header := func(k, v string) {
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(v)
		b.WriteString("\r\n")
	}

	header("From", from)
	header("To", msg.To)
	header("Subject", mime.QEncoding.Encode("utf-8", msg.Subject))
	header("Date", time.Now().Format(time.RFC1123Z))
	header("MIME-Version", "1.0")

	part := func(contentType, body string) {
		header("Content-Type", contentType)
		header("Content-Transfer-Encoding", "8bit")
		b.WriteString("\r\n")
		b.WriteString(body)
		b.WriteString("\r\n")
	}

	switch {
	case msg.TextBody != "" && msg.HTMLBody != "":
		// A fixed boundary would corrupt any message whose body happened to
		// contain it; a random one per message rules that out.
		raw := make([]byte, 16)
		if _, err := rand.Read(raw); err != nil {
			return nil, fmt.Errorf("generating mime boundary: %w", err)
		}
		boundary := hex.EncodeToString(raw)

		header("Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
		b.WriteString("\r\n")

		b.WriteString("--" + boundary + "\r\n")
		part("text/plain; charset=utf-8", msg.TextBody)
		b.WriteString("--" + boundary + "\r\n")
		part("text/html; charset=utf-8", msg.HTMLBody)
		b.WriteString("--" + boundary + "--\r\n")
	case msg.HTMLBody != "":
		part("text/html; charset=utf-8", msg.HTMLBody)
	default:
		part("text/plain; charset=utf-8", msg.TextBody)
	}

	return []byte(b.String()), nil
}
