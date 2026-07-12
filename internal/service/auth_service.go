package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/pkg/jwt"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/security"
)

// AuthService registers users and issues tokens.
type AuthService struct {
	users  repository.UserRepository
	tokens *jwt.TokenService
	cfg    config.AuthConfig
	log    *logger.Logger
}

func NewAuthService(
	users repository.UserRepository,
	tokens *jwt.TokenService,
	cfg config.AuthConfig,
	log *logger.Logger,
) *AuthService {
	return &AuthService{users: users, tokens: tokens, cfg: cfg, log: log}
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
type TokenPair struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int          `json:"expires_in"`
	User        *domain.User `json:"user"`
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
	claims, err := s.tokens.ValidateToken(token)
	if err != nil {
		return nil, domain.ErrInvalidToken
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

	return s.issueTokens(user)
}

// issueTokens mints an access token for the user.
func (s *AuthService) issueTokens(user *domain.User) (*TokenPair, error) {
	token, err := s.tokens.GenerateToken(user.ID.String(), user.Username, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	// Never let a hash leave the service, even though the field is unexported
	// from JSON's point of view only by convention.
	safe := *user
	safe.PasswordHash = ""

	return &TokenPair{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.cfg.AccessTokenTTL.Seconds()),
		User:        &safe,
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
