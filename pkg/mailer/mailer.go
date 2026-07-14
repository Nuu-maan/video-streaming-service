// Package mailer sends transactional email. The concrete sender is chosen from
// configuration: a real SMTP client when a server is configured, otherwise a
// logger-backed no-op so local development needs no mail server at all.
package mailer

import (
	"context"

	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
)

// Message is one outgoing email. TextBody and HTMLBody are alternative
// renderings of the same content; a sender includes whichever are non-empty.
type Message struct {
	To       string
	Subject  string
	TextBody string
	HTMLBody string
}

// Mailer delivers a message to a single recipient.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// Config selects and configures the sender. An empty Host means "no SMTP
// server": New then falls back to the log mailer rather than failing, so an
// unconfigured environment still boots.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	// From is the envelope sender and From header on every message.
	From string
	// AllowInsecure permits sending over cleartext when the server offers no
	// STARTTLS. Credentials and reset links would cross the wire unencrypted,
	// so this exists only for local relays such as MailHog.
	AllowInsecure bool
}

// New picks the sender for the given configuration. An unset SMTP host is not
// an error — the log mailer keeps the service bootable with zero mail setup —
// but the single startup line below is what tells an operator their mail is
// going to the log rather than to users.
func New(cfg Config, log *logger.Logger) Mailer {
	if cfg.Host == "" {
		log.Info(context.Background(), "SMTP not configured; outgoing mail will be written to the log instead of sent", nil)
		return NewLogMailer(log)
	}
	return NewSMTPMailer(cfg)
}
