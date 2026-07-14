package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/pkg/jwt"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/security"
)

// AuthService registers users, issues tokens, and revokes them again.
type AuthService struct {
	users    repository.UserRepository
	tokens   *jwt.TokenService
	sessions *SessionService
	cfg      config.AuthConfig
	log      *logger.Logger
}

func NewAuthService(
	users repository.UserRepository,
	tokens *jwt.TokenService,
	sessions *SessionService,
	cfg config.AuthConfig,
	log *logger.Logger,
) *AuthService {
	return &AuthService{users: users, tokens: tokens, sessions: sessions, cfg: cfg, log: log}
}

// Credentials is a login attempt. Identifier is either a username or an email.
type Credentials struct {
	Identifier string
	Password   string
}

// Registration is a sign-up request.
type Registration struct {
	Username string
	Email    string
	Password string
}

// TokenPair is what a successful authentication yields.
//
// The refresh token is the half that makes a session outlive the access token's
// few minutes. Without it a client is logged out the moment the access token
// lapses, however recently the user was active, because there is nothing left to
// authenticate the renewal with.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	// ExpiresIn is the access token's lifetime in seconds. A client should renew
	// before it elapses rather than waiting to be rejected.
	ExpiresIn        int          `json:"expires_in"`
	RefreshExpiresIn int          `json:"refresh_expires_in"`
	User             *domain.User `json:"user"`
}

// Register creates a user with the default role and returns tokens for them.
func (s *AuthService) Register(ctx context.Context, req Registration) (*TokenPair, error) {
	if err := security.ValidatePassword(req.Password); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrWeakPassword, err)
	}

	hash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user, err := domain.NewUser(req.Username, req.Email, hash, domain.RoleUser)
	if err != nil {
		return nil, err
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	s.log.Info(ctx, "user registered", map[string]interface{}{"user_id": user.ID})

	return s.issueTokens(user)
}

// Login authenticates a user and returns tokens.
//
// A missing user and a wrong password both yield ErrInvalidCredentials, and the
// password is compared even when the user does not exist, so the response does
// not reveal which usernames are registered.
func (s *AuthService) Login(ctx context.Context, creds Credentials) (*TokenPair, error) {
	user, err := s.lookup(ctx, creds.Identifier)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, err
	}

	// security.ComparePassword on a dummy hash keeps the work (and therefore
	// the response time) roughly constant whether or not the user exists.
	storedHash := dummyBcryptHash
	if user != nil {
		storedHash = user.PasswordHash
	}
	passwordMatches := security.ComparePassword(storedHash, creds.Password)

	if user == nil || !passwordMatches {
		return nil, domain.ErrInvalidCredentials
	}

	if user.IsCurrentlyBanned() {
		return nil, domain.ErrUserBanned
	}

	user.UpdateLastLogin()
	if err := s.users.Update(ctx, user); err != nil {
		// The credentials were valid; failing to record the login timestamp is
		// not a reason to deny access.
		s.log.Warn(ctx, "could not record last login", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
	}

	return s.issueTokens(user)
}

// Refresh exchanges a still-valid token for a new one.
func (s *AuthService) Refresh(ctx context.Context, token string) (*TokenPair, error) {
	// Only a refresh token is redeemable here. Accepting an access token — which
	// is what this used to do — meant refresh could only ever extend a session
	// that had not yet expired, making it useless for the one job it has.
	claims, err := s.tokens.ValidateRefreshToken(token)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// A revoked token must not be redeemable for a fresh one, or a logout would
	// only hold until the next refresh. Unlike the per-request middleware check
	// there is no fail-open switch here: minting a new token is a bigger grant
	// than serving one request, so an unreachable revocation store fails it.
	revoked, err := s.sessions.IsRevoked(ctx, claims.ID, claims.UserID, claims.IssuedAtTime())
	if err != nil {
		return nil, fmt.Errorf("checking token revocation: %w", err)
	}
	if revoked {
		return nil, domain.ErrTokenRevoked
	}

	user, err := s.lookupByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// Re-check the ban on refresh; otherwise a user banned mid-session could
	// keep extending their token indefinitely.
	if user.IsCurrentlyBanned() {
		return nil, domain.ErrUserBanned
	}

	// The role is re-read from the user record above, so a promotion or demotion
	// takes effect on the next refresh rather than being frozen into the session.
	return s.renewAccess(user, token)
}

// Logout revokes the presented access token, and the refresh token too when
// one is supplied. Both are validated first: revocation is keyed by jti, and
// an unverifiable token has no trustworthy jti to key on.
func (s *AuthService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	claims, err := s.tokens.ValidateToken(accessToken)
	if err != nil {
		return domain.ErrInvalidToken
	}
	if err := s.revokeByClaims(ctx, claims); err != nil {
		return err
	}

	if refreshToken != "" && refreshToken != accessToken {
		// The refresh token is best-effort: if it does not validate it cannot
		// be redeemed anyway, and its failure must not undo the logout of the
		// access token that already succeeded.
		if refreshClaims, err := s.tokens.ValidateToken(refreshToken); err == nil {
			if err := s.revokeByClaims(ctx, refreshClaims); err != nil {
				return err
			}
		}
	}

	return nil
}

// RevokeAllSessions invalidates every outstanding token for userID. It backs
// POST /auth/logout-all, and it is the method security-sensitive account flows
// — a password reset above all — must call so a stolen session does not
// survive the reset.
func (s *AuthService) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	return s.sessions.RevokeAllUserSessions(ctx, userID)
}

func (s *AuthService) revokeByClaims(ctx context.Context, claims *jwt.Claims) error {
	// Tokens minted before jti existed cannot be individually denylisted; they
	// age out within the access-token TTL, and logout-all still catches them
	// through the issued-at cutoff.
	if claims.ID == "" {
		return nil
	}

	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}
	return s.sessions.RevokeToken(ctx, claims.ID, expiresAt)
}

// issueTokens mints a fresh access token and a fresh refresh token. This is the
// sign-in path: register and login.
func (s *AuthService) issueTokens(user *domain.User) (*TokenPair, error) {
	refresh, err := s.tokens.GenerateRefreshToken(user.ID.String(), user.Username, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}
	return s.renewAccess(user, refresh)
}

// renewAccess mints a new access token and returns it alongside the refresh
// token the caller already holds.
//
// The refresh token is deliberately NOT rotated. Rotation would limit the replay
// window on a stolen refresh token, but it makes concurrent refreshes lose:
// two requests that race a 401 both redeem the same token, one rotation wins,
// and the loser is holding a token that has just been revoked — logging out a
// user who did nothing wrong. Sessions are already revocable through logout and
// logout-all, which is the control that actually matters here.
func (s *AuthService) renewAccess(user *domain.User, refreshToken string) (*TokenPair, error) {
	access, err := s.tokens.GenerateToken(user.ID.String(), user.Username, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	// Never let a hash leave the service, even though the field is unexported
	// from JSON's point of view only by convention.
	safe := *user
	safe.PasswordHash = ""

	return &TokenPair{
		AccessToken:      access,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        int(s.cfg.AccessTokenTTL.Seconds()),
		RefreshExpiresIn: int(s.cfg.RefreshTokenTTL.Seconds()),
		User:             &safe,
	}, nil
}

// lookup resolves an identifier that may be either an email or a username.
func (s *AuthService) lookup(ctx context.Context, identifier string) (*domain.User, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, domain.ErrUserNotFound
	}

	if strings.Contains(identifier, "@") {
		return s.users.GetByEmail(ctx, identifier)
	}
	return s.users.GetByUsername(ctx, identifier)
}

func (s *AuthService) lookupByID(ctx context.Context, rawID string) (*domain.User, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	return s.users.GetByID(ctx, id)
}

// dummyBcryptHash is a valid bcrypt hash of a value nobody knows. Comparing
// against it costs the same as comparing against a real hash, which is the
// point: it keeps login timing from revealing whether an account exists.
const dummyBcryptHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEeO.wLPvfLZBrPWvpBRDDMDDNiIVKWEP.6"
