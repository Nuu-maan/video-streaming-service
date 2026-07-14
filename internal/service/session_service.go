package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
)

const (
	revokedTokenKeyPrefix = "auth:revoked:jti:"
	// The key is versioned because the value's unit changed from seconds to
	// milliseconds. A stale seconds-valued marker read as milliseconds is a
	// number ~1000x too small, which would silently deny nothing — a revocation
	// that quietly stops revoking is worse than one that fails loudly, so the old
	// keys are abandoned rather than reinterpreted.
	minIssuedAtKeyPrefix = "auth:revoked:user:ms:"
)

// SessionService tracks revoked JWTs in Redis.
//
// Tokens are stateless, so "logging out" cannot delete anything by itself: a
// signed token stays cryptographically valid until it expires. Revocation is
// therefore a denylist. A single token is denied by its jti claim; logout-all
// stores a per-user minimum issued-at instead, so every token minted before
// that moment is denied in O(1) without enumerating outstanding tokens. Every
// entry carries a TTL no longer than the tokens it denies, so the denylist
// never outgrows the set of still-live tokens.
type SessionService struct {
	redis *redis.Client
	// maxTokenTTL is the longest lifetime any issued token can have. It bounds
	// the min-issued-at marker: once maxTokenTTL has passed, every token the
	// marker could deny has already expired on its own.
	maxTokenTTL time.Duration
}

func NewSessionService(redisClient *redis.Client, maxTokenTTL time.Duration) *SessionService {
	return &SessionService{redis: redisClient, maxTokenTTL: maxTokenTTL}
}

// RevokeToken denylists one token by its jti until the moment the token would
// have expired anyway, after which the entry evicts itself.
func (s *SessionService) RevokeToken(ctx context.Context, jti string, expiresAt time.Time) error {
	if jti == "" {
		return domain.ErrInvalidToken
	}

	ttl := time.Until(expiresAt)
	if expiresAt.IsZero() {
		// A token without an exp claim never expires on its own, so the entry
		// falls back to the longest lifetime this service ever issues.
		ttl = s.maxTokenTTL
	}
	if ttl <= 0 {
		// Already expired; signature validation rejects it without our help.
		return nil
	}

	if err := s.redis.Set(ctx, revokedTokenKeyPrefix+jti, "1", ttl).Err(); err != nil {
		return fmt.Errorf("denylisting token: %w", err)
	}
	return nil
}

// RevokeAllUserSessions invalidates every token issued to userID up to now.
// This is the hook for logout-all and for security-sensitive account changes
// such as a password reset, where every outstanding session must die at once.
//
// The cutoff is nudged one millisecond into the future so that a token minted in
// the same millisecond is still caught: the comparison in IsRevoked is a strict
// less-than, and a session that survives "log out everywhere" defeats the point
// of the call.
func (s *SessionService) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	key := minIssuedAtKeyPrefix + userID.String()
	cutoff := time.Now().UnixMilli() + 1
	if err := s.redis.Set(ctx, key, strconv.FormatInt(cutoff, 10), s.maxTokenTTL).Err(); err != nil {
		return fmt.Errorf("revoking all sessions for user %s: %w", userID, err)
	}
	return nil
}

// IsRevoked reports whether the token identified by jti, issued to userID at
// issuedAt, has been revoked. It runs on every authenticated request, so both
// keys are fetched in a single MGET round trip.
//
// issuedAt must carry millisecond precision — see jwt.Claims.IssuedAtTime. The
// whole-second iat claim is too coarse to compare against the logout-all cutoff:
// every session created in the same second as the revocation would outlive it.
func (s *SessionService) IsRevoked(ctx context.Context, jti, userID string, issuedAt time.Time) (bool, error) {
	vals, err := s.redis.MGet(ctx, revokedTokenKeyPrefix+jti, minIssuedAtKeyPrefix+userID).Result()
	if err != nil {
		return false, fmt.Errorf("checking token revocation: %w", err)
	}

	if vals[0] != nil {
		return true, nil
	}

	if raw, ok := vals[1].(string); ok {
		minIssuedAt, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return false, fmt.Errorf("parsing min issued-at %q: %w", raw, err)
		}
		if issuedAt.UnixMilli() < minIssuedAt {
			return true, nil
		}
	}

	return false, nil
}
