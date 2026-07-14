package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/mailer"
	"github.com/Nuu-maan/video-streaming-service/pkg/security"
)

const (
	// tokenSecretBytes is the entropy of every mailed token: 256 bits from
	// crypto/rand, never a UUID and never derived from the user or the clock.
	tokenSecretBytes = 32

	// defaultResetTokenTTL bounds how long a password-reset link works.
	defaultResetTokenTTL = time.Hour

	// mailSendTimeout bounds a detached send, which no longer has a request
	// deadline to inherit.
	mailSendTimeout = 30 * time.Second
)

// SessionRevoker force-logs-out every session a user holds. Session revocation
// is built elsewhere; this interface is the seam where the wiring layer plugs
// it in. A nil revoker is tolerated so this service does not block on that
// work landing.
type SessionRevoker interface {
	RevokeAllSessions(ctx context.Context, userID uuid.UUID) error
}

// EmailService owns email verification and password reset.
//
// Only a digest of each token is stored: sha256 of the random secret,
// truncated to 16 bytes so it fits the UUID-typed token columns (and the
// *uuid.UUID fields on domain.User), and compared in constant time. A leaked
// database dump therefore contains nothing that works in a link. 128 bits of
// preimage resistance over a 256-bit random secret is far beyond brute force.
type EmailService struct {
	users    repository.UserRepository
	mail     mailer.Mailer
	baseURL  string
	resetTTL time.Duration
	revoker  SessionRevoker
	log      *logger.Logger
}

// NewEmailService wires the service. baseURL is the frontend origin links
// point at (the API is consumed cross-origin, so links must open the frontend,
// not this server). A nil revoker and a non-positive resetTokenTTL both fall
// back to safe defaults.
func NewEmailService(
	users repository.UserRepository,
	mail mailer.Mailer,
	baseURL string,
	resetTokenTTL time.Duration,
	revoker SessionRevoker,
	log *logger.Logger,
) *EmailService {
	if resetTokenTTL <= 0 {
		resetTokenTTL = defaultResetTokenTTL
	}
	for len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	return &EmailService{
		users:    users,
		mail:     mail,
		baseURL:  baseURL,
		resetTTL: resetTokenTTL,
		revoker:  revoker,
		log:      log,
	}
}

// SendVerificationEmail (re)issues a verification token for the user and mails
// the link to their registered address. Re-sending replaces the stored digest,
// so only the newest link works.
func (s *EmailService) SendVerificationEmail(ctx context.Context, userID uuid.UUID) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.EmailVerified {
		return domain.ErrEmailAlreadyVerified
	}

	plaintext, digest, err := newEmailToken(user.ID)
	if err != nil {
		return err
	}

	user.EmailVerificationToken = &digest
	if err := s.users.Update(ctx, user); err != nil {
		return fmt.Errorf("storing verification token: %w", err)
	}

	link := s.baseURL + "/verify-email?token=" + plaintext
	msg := mailer.Message{
		To:      user.Email,
		Subject: "Verify your email address",
		TextBody: fmt.Sprintf(
			"Hi %s,\n\nConfirm your email address by opening this link:\n\n%s\n\nIf you did not create this account, you can ignore this message.\n",
			user.Username, link),
		HTMLBody: fmt.Sprintf(
			`<p>Hi %s,</p><p>Confirm your email address by opening this link:</p><p><a href="%s">%s</a></p><p>If you did not create this account, you can ignore this message.</p>`,
			user.Username, link, link),
	}
	if err := s.mail.Send(ctx, msg); err != nil {
		return fmt.Errorf("sending verification email: %w", err)
	}
	return nil
}

// VerifyEmail consumes a verification token. The stored digest is cleared on
// success, so a token works exactly once.
func (s *EmailService) VerifyEmail(ctx context.Context, token string) error {
	user, digest, err := s.lookupByToken(ctx, token)
	if err != nil {
		return err
	}

	if !digestMatches(user.EmailVerificationToken, digest) {
		return domain.ErrInvalidToken
	}

	user.VerifyEmail()
	if err := s.users.Update(ctx, user); err != nil {
		return fmt.Errorf("marking email verified: %w", err)
	}
	return nil
}

// RequestPasswordReset issues a reset token for the given address, if it is
// registered. It reports success either way, generates the token on both
// paths, and mails asynchronously, so neither the response body nor its
// timing tells a caller whether the address exists.
func (s *EmailService) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return err
	}

	tokenID := uuid.Nil
	if user != nil {
		tokenID = user.ID
	}
	plaintext, digest, err := newEmailToken(tokenID)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	expiry := time.Now().Add(s.resetTTL)
	user.PasswordResetToken = &digest
	user.PasswordResetExpiry = &expiry
	if err := s.users.Update(ctx, user); err != nil {
		return fmt.Errorf("storing password reset token: %w", err)
	}

	link := s.baseURL + "/reset-password?token=" + plaintext
	s.sendAsync(ctx, mailer.Message{
		To:      user.Email,
		Subject: "Reset your password",
		TextBody: fmt.Sprintf(
			"Hi %s,\n\nA password reset was requested for your account. Open this link to choose a new password:\n\n%s\n\nThe link expires in %s. If you did not request this, you can ignore this message; your password is unchanged.\n",
			user.Username, link, s.resetTTL),
		HTMLBody: fmt.Sprintf(
			`<p>Hi %s,</p><p>A password reset was requested for your account. Open this link to choose a new password:</p><p><a href="%s">%s</a></p><p>The link expires in %s. If you did not request this, you can ignore this message; your password is unchanged.</p>`,
			user.Username, link, link, s.resetTTL),
	})
	return nil
}

// ResetPassword consumes a reset token and sets a new password. A missing,
// malformed, expired, and simply wrong token all yield the same
// domain.ErrInvalidToken so the endpoint cannot be used to probe token state.
func (s *EmailService) ResetPassword(ctx context.Context, token, newPassword string) error {
	user, digest, err := s.lookupByToken(ctx, token)
	if err != nil {
		return err
	}

	if !user.IsPasswordResetTokenValid() || !digestMatches(user.PasswordResetToken, digest) {
		return domain.ErrInvalidToken
	}

	if err := security.ValidatePassword(newPassword); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrWeakPassword, err)
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	user.PasswordHash = hash
	user.ClearPasswordResetToken()
	if err := s.users.Update(ctx, user); err != nil {
		return fmt.Errorf("resetting password: %w", err)
	}

	s.revokeSessions(ctx, user.ID)
	return nil
}

// ChangePassword sets a new password for an authenticated user after verifying
// the current one. Any outstanding reset token is cleared too: a live reset
// link must not outlast a deliberate password change.
func (s *EmailService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !security.ComparePassword(user.PasswordHash, currentPassword) {
		return domain.ErrInvalidCredentials
	}

	if err := security.ValidatePassword(newPassword); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrWeakPassword, err)
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	user.PasswordHash = hash
	user.ClearPasswordResetToken()
	if err := s.users.Update(ctx, user); err != nil {
		return fmt.Errorf("changing password: %w", err)
	}

	s.revokeSessions(ctx, user.ID)
	return nil
}

// lookupByToken resolves the user a token addresses. Every failure — bad
// encoding, unknown user, deleted account — collapses into ErrInvalidToken so
// callers cannot distinguish them.
func (s *EmailService) lookupByToken(ctx context.Context, token string) (*domain.User, uuid.UUID, error) {
	userID, digest, ok := parseEmailToken(token)
	if !ok {
		return nil, uuid.Nil, domain.ErrInvalidToken
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, uuid.Nil, domain.ErrInvalidToken
		}
		return nil, uuid.Nil, err
	}
	return user, digest, nil
}

// revokeSessions ends every session the user holds, if a revoker is wired in.
// The password change has already been committed at this point, so a
// revocation failure is logged rather than surfaced: failing the request would
// leave the user believing the change did not happen.
func (s *EmailService) revokeSessions(ctx context.Context, userID uuid.UUID) {
	if s.revoker == nil {
		return
	}
	if err := s.revoker.RevokeAllSessions(ctx, userID); err != nil {
		s.log.Error(ctx, "password changed but session revocation failed", err, map[string]interface{}{
			"user_id": userID,
		})
	}
}

// sendAsync delivers mail detached from the request. On forgot-password the
// SMTP round-trip only happens for real accounts, so keeping it out of the
// request would otherwise turn response latency into an enumeration oracle.
func (s *EmailService) sendAsync(ctx context.Context, msg mailer.Message) {
	ctx = context.WithoutCancel(ctx)
	go func() {
		ctx, cancel := context.WithTimeout(ctx, mailSendTimeout)
		defer cancel()
		if err := s.mail.Send(ctx, msg); err != nil {
			s.log.Error(ctx, "failed to send email", err, map[string]interface{}{
				"to":      msg.To,
				"subject": msg.Subject,
			})
		}
	}()
}

// newEmailToken builds the plaintext token to mail and the digest to store.
// The plaintext is base64url(userID || secret): the ID prefix is only
// addressing — it lets the public endpoints find the row without a
// token-indexed query — while all the secrecy lives in the 32 random bytes.
func newEmailToken(userID uuid.UUID) (plaintext string, digest uuid.UUID, err error) {
	secret := make([]byte, tokenSecretBytes)
	if _, err := rand.Read(secret); err != nil {
		return "", uuid.Nil, fmt.Errorf("generating token: %w", err)
	}

	payload := make([]byte, 0, len(userID)+tokenSecretBytes)
	payload = append(payload, userID[:]...)
	payload = append(payload, secret...)

	return base64.RawURLEncoding.EncodeToString(payload), hashTokenSecret(secret), nil
}

// parseEmailToken splits a mailed token back into the user it addresses and
// the digest of its secret. ok is false for anything malformed.
func parseEmailToken(token string) (userID uuid.UUID, digest uuid.UUID, ok bool) {
	payload, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(payload) != 16+tokenSecretBytes {
		return uuid.Nil, uuid.Nil, false
	}

	userID, err = uuid.FromBytes(payload[:16])
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}
	return userID, hashTokenSecret(payload[16:]), true
}

// hashTokenSecret is the one-way mapping from a token's secret to what the
// database holds. The sha256 digest is truncated to 16 bytes purely because
// the token columns (and domain.User fields) are UUID-typed; 128 bits of
// preimage resistance is not the weak link here.
func hashTokenSecret(secret []byte) uuid.UUID {
	sum := sha256.Sum256(secret)
	var digest uuid.UUID
	copy(digest[:], sum[:16])
	return digest
}

// digestMatches compares the stored digest against the one computed from the
// presented token. Constant-time out of habit: both sides are already hashes,
// but the comparison costs nothing and never becomes a refactoring hazard.
func digestMatches(stored *uuid.UUID, computed uuid.UUID) bool {
	if stored == nil {
		return false
	}
	return subtle.ConstantTimeCompare(stored[:], computed[:]) == 1
}
