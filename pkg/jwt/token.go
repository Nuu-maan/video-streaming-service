package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType distinguishes the two kinds of token this service mints.
//
// The distinction is load-bearing, not decorative. A refresh token is long-lived
// (days) where an access token lives for minutes, so if the two were
// interchangeable a stolen refresh token would be a long-lived API credential,
// and — worse — an access token would be accepted at the refresh endpoint, which
// is how the previous implementation worked: it took the caller's access token
// and minted a new one from it. That made refresh useless for its only purpose,
// because once the access token had expired there was nothing left to refresh
// with, and the user was silently logged out however recently they had been
// active.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// ErrWrongTokenType is returned when a token is valid but is not the kind the
// caller requires — an access token presented to the refresh endpoint, or a
// refresh token presented as API credentials.
var ErrWrongTokenType = errors.New("wrong token type")

type Claims struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	Type     TokenType `json:"typ"`

	// IssuedAtMS is the issue time in milliseconds.
	//
	// The standard iat claim is defined in whole seconds, which is too coarse for
	// "log out everywhere": that works by recording a cutoff and denying every
	// token issued before it, so with second granularity any session created in
	// the same second as the logout survives it. That is not a theoretical race —
	// signing in twice and immediately revoking reproduces it every time — and a
	// surviving session is the entire failure the button exists to prevent.
	IssuedAtMS int64 `json:"iat_ms,omitempty"`

	jwt.RegisteredClaims
}

// IssuedAtTime is when the token was minted, at the best precision available.
// It falls back to the whole-second iat for tokens minted before iat_ms existed.
func (c *Claims) IssuedAtTime() time.Time {
	if c.IssuedAtMS > 0 {
		return time.UnixMilli(c.IssuedAtMS)
	}
	if c.IssuedAt != nil {
		return c.IssuedAt.Time
	}
	return time.Time{}
}

type TokenService struct {
	secretKey  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
}

func NewTokenService(secretKey string, accessTTL, refreshTTL time.Duration, issuer string) *TokenService {
	return &TokenService{
		secretKey:  []byte(secretKey),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		issuer:     issuer,
	}
}

// AccessTTL is the lifetime of the tokens GenerateToken mints, which the API
// reports to clients as expires_in so they can refresh before it lapses.
func (t *TokenService) AccessTTL() time.Duration { return t.accessTTL }

// GenerateToken mints a short-lived access token: the credential presented on
// every authenticated request.
func (t *TokenService) GenerateToken(userID, username, role string) (string, error) {
	return t.sign(userID, username, role, TokenTypeAccess, t.accessTTL)
}

// GenerateRefreshToken mints a long-lived refresh token, exchangeable for a new
// access token at the refresh endpoint and valid for nothing else.
//
// It deliberately carries the username and role even though the refresh endpoint
// re-reads the user from the database anyway: the claims are what identify the
// token's owner when it is revoked, and re-reading is what stops a refresh from
// resurrecting a role or a ban state that has since changed.
func (t *TokenService) GenerateRefreshToken(userID, username, role string) (string, error) {
	return t.sign(userID, username, role, TokenTypeRefresh, t.refreshTTL)
}

func (t *TokenService) sign(userID, username, role string, typ TokenType, ttl time.Duration) (string, error) {
	now := time.Now()

	claims := &Claims{
		UserID:     userID,
		Username:   username,
		Role:       role,
		Type:       typ,
		IssuedAtMS: now.UnixMilli(),
		RegisteredClaims: jwt.RegisteredClaims{
			// A random jti gives each token an identity of its own, so revocation
			// can denylist a single token by ID instead of storing the raw token
			// string — which would leak a usable credential into Redis and break if
			// the client re-encoded the token.
			ID:        uuid.NewString(),
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    t.issuer,
		},
	}

	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateAccessToken accepts only an access token. Every authenticated request
// goes through it, so a refresh token can never be used as API credentials.
func (t *TokenService) ValidateAccessToken(tokenString string) (*Claims, error) {
	return t.validate(tokenString, TokenTypeAccess)
}

// ValidateRefreshToken accepts only a refresh token.
func (t *TokenService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return t.validate(tokenString, TokenTypeRefresh)
}

func (t *TokenService) validate(tokenString string, want TokenType) (*Claims, error) {
	claims, err := t.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Tokens minted before the type claim existed have an empty Type. Treating
	// those as access tokens keeps existing sessions working across the upgrade;
	// they simply cannot be used to refresh, which is the safe direction to fail.
	got := claims.Type
	if got == "" {
		got = TokenTypeAccess
	}
	if got != want {
		return nil, fmt.Errorf("%w: token is a %s token, %s required", ErrWrongTokenType, got, want)
	}

	return claims, nil
}

// ValidateToken checks the signature and standard claims without caring which
// kind of token it is. Prefer ValidateAccessToken or ValidateRefreshToken;
// this is for callers that must inspect a token whose type is not yet known,
// such as logout, which revokes whatever it is handed.
func (t *TokenService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return t.secretKey, nil
	}, jwt.WithIssuer(t.issuer))

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (t *TokenService) ExtractUserID(tokenString string) (string, error) {
	claims, err := t.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (t *TokenService) ExtractClaims(tokenString string) (*Claims, error) {
	return t.ValidateToken(tokenString)
}
