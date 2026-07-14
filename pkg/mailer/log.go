package mailer

import (
	"context"

	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
)

// LogMailer writes messages to the log instead of delivering them. It is the
// default sender when SMTP is not configured.
type LogMailer struct {
	log *logger.Logger
}

func NewLogMailer(log *logger.Logger) *LogMailer {
	return &LogMailer{log: log}
}

// Send logs the whole text body at info level. Verification and password-reset
// links land in that body, so a developer can copy them straight out of the
// console without any mail server running.
func (m *LogMailer) Send(ctx context.Context, msg Message) error {
	m.log.Info(ctx, "outgoing email (log mailer, message not sent)", map[string]interface{}{
		"to":      msg.To,
		"subject": msg.Subject,
		"body":    msg.TextBody,
	})
	return nil
}
