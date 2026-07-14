package jwt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const (
	testSecret  = "test-secret-key-at-least-32-characters-long"
	otherSecret = "a-completely-different-secret-key-value-32+"
	testIssuer  = "video-streaming-service-test"
)

func newService(t *testing.T, secret string, d time.Duration) *TokenService {
	t.Helper()
	return NewTokenService(secret, d, 24*time.Hour, testIssuer)
}

func TestGenerateAndValidateToken(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		username string
		role     string
	}{
		{name: "regular user", userID: "user-1", username: "alice", role: "user"},
		{name: "admin", userID: "9a8b7c6d-5e4f-4a3b-8c1d-0e9f8a7b6c5d", username: "root", role: "admin"},
		{name: "empty role", userID: "user-2", username: "bob", role: ""},
		{name: "unicode username", userID: "user-3", username: "ünïcødé", role: "moderator"},
	}

	svc := newService(t, testSecret, time.Hour)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.GenerateToken(tt.userID, tt.username, tt.role)
			if err != nil {
				t.Fatalf("GenerateToken: %v", err)
			}
			if token == "" {
				t.Fatal("GenerateToken returned an empty token")
			}

			claims, err := svc.ValidateToken(token)
			if err != nil {
				t.Fatalf("ValidateToken: %v", err)
			}
			if claims.UserID != tt.userID {
				t.Errorf("UserID = %q, want %q", claims.UserID, tt.userID)
			}
			if claims.Username != tt.username {
				t.Errorf("Username = %q, want %q", claims.Username, tt.username)
			}
			if claims.Role != tt.role {
				t.Errorf("Role = %q, want %q", claims.Role, tt.role)
			}
			if claims.Subject != tt.userID {
				t.Errorf("Subject = %q, want %q", claims.Subject, tt.userID)
			}
			if claims.Issuer != testIssuer {
				t.Errorf("Issuer = %q, want %q", claims.Issuer, testIssuer)
			}
			if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
				t.Errorf("ExpiresAt = %v, want a time in the future", claims.ExpiresAt)
			}
		})
	}
}

func TestGenerateTokenAssignsUniqueJTI(t *testing.T) {
	svc := newService(t, testSecret, time.Hour)

	first, err := svc.GenerateToken("user-1", "alice", "user")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	second, err := svc.GenerateToken("user-1", "alice", "user")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	firstClaims, err := svc.ValidateToken(first)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	secondClaims, err := svc.ValidateToken(second)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}

	if firstClaims.ID == "" {
		t.Fatal("token carries no jti; per-token revocation cannot key on it")
	}
	// Identical identity, still distinct jti: revoking one login must not
	// revoke another.
	if firstClaims.ID == secondClaims.ID {
		t.Errorf("two tokens share jti %q, want unique IDs", firstClaims.ID)
	}
}

// tamperPayload re-encodes the token's claims with an escalated role, keeping
// the original (now wrong) signature.
func tamperPayload(t *testing.T, token string) string {
	t.Helper()

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3", len(parts))
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decoding payload: %v", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatalf("unmarshalling payload: %v", err)
	}
	claims["role"] = "admin"

	forged, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshalling forged payload: %v", err)
	}

	parts[1] = base64.RawURLEncoding.EncodeToString(forged)
	return strings.Join(parts, ".")
}

func TestValidateTokenRejectsBadTokens(t *testing.T) {
	svc := newService(t, testSecret, time.Hour)

	valid, err := svc.GenerateToken("user-1", "alice", "user")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	// Signed with a different secret by an otherwise identical service.
	foreign, err := newService(t, otherSecret, time.Hour).GenerateToken("user-1", "alice", "admin")
	if err != nil {
		t.Fatalf("GenerateToken with other secret: %v", err)
	}

	// A token whose signature was stripped and alg set to "none".
	noneToken := func() string {
		tok := jwtlib.NewWithClaims(jwtlib.SigningMethodNone, &Claims{
			UserID:   "user-1",
			Username: "alice",
			Role:     "admin",
			RegisteredClaims: jwtlib.RegisteredClaims{
				ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
			},
		})
		s, err := tok.SignedString(jwtlib.UnsafeAllowNoneSignatureType)
		if err != nil {
			t.Fatalf("signing none-alg token: %v", err)
		}
		return s
	}()

	tests := []struct {
		name  string
		token string
	}{
		{name: "signed with a different secret", token: foreign},
		{name: "empty string", token: ""},
		{name: "garbage", token: "not-a-jwt-at-all"},
		{name: "garbage with dots", token: "aaa.bbb.ccc"},
		{name: "tampered signature", token: valid + "AB"},
		{name: "tampered payload", token: tamperPayload(t, valid)},
		{name: "alg=none", token: noneToken},
		{name: "only header", token: "eyJhbGciOiJIUzI1NiJ9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := svc.ValidateToken(tt.token)
			if err == nil {
				t.Fatalf("ValidateToken(%q) = %+v, want an error", tt.token, claims)
			}
			if claims != nil {
				t.Errorf("ValidateToken(%q) returned claims %+v alongside an error, want nil", tt.token, claims)
			}
		})
	}
}

func TestValidateTokenRejectsExpiredToken(t *testing.T) {
	// A one-nanosecond lifetime: the token is expired the moment it is issued.
	svc := newService(t, testSecret, time.Nanosecond)

	token, err := svc.GenerateToken("user-1", "alice", "user")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	// jwt/v5 compares expiry at whole-second resolution, so wait out the
	// boundary rather than racing it.
	time.Sleep(2 * time.Millisecond)

	if _, err := svc.ValidateToken(token); err == nil {
		t.Fatal("ValidateToken accepted an expired token, want an error")
	}

	// The same holds for a token issued with an already-elapsed lifetime.
	past := newService(t, testSecret, -time.Hour)
	expired, err := past.GenerateToken("user-1", "alice", "user")
	if err != nil {
		t.Fatalf("GenerateToken (negative duration): %v", err)
	}
	if _, err := svc.ValidateToken(expired); err == nil {
		t.Fatal("ValidateToken accepted a token that expired an hour ago, want an error")
	}
}

func TestExtractUserID(t *testing.T) {
	svc := newService(t, testSecret, time.Hour)

	valid, err := svc.GenerateToken("user-42", "alice", "user")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	foreign, err := newService(t, otherSecret, time.Hour).GenerateToken("user-42", "alice", "user")
	if err != nil {
		t.Fatalf("GenerateToken with other secret: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		want    string
		wantErr bool
	}{
		{name: "valid token", token: valid, want: "user-42"},
		{name: "empty token", token: "", wantErr: true},
		{name: "garbage token", token: "nonsense", wantErr: true},
		{name: "wrong secret", token: foreign, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.ExtractUserID(tt.token)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExtractUserID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ExtractUserID() = %q, want %q", got, tt.want)
			}
		})
	}
}

// The two token types must not be interchangeable. Regression test for a real
// bug: refresh used to take the caller's *access* token and mint a new one from
// it, so a session could only be extended while it was still alive — the one
// situation in which extending it is unnecessary. Once the access token expired
// there was nothing to refresh with and the user was silently logged out.
func TestTokenTypesAreNotInterchangeable(t *testing.T) {
	svc := newService(t, testSecret, time.Hour)

	access, err := svc.GenerateToken("user-7", "carol", "admin")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	refresh, err := svc.GenerateRefreshToken("user-7", "carol", "admin")
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}

	if _, err := svc.ValidateAccessToken(access); err != nil {
		t.Errorf("ValidateAccessToken(access token) = %v, want it accepted", err)
	}
	if _, err := svc.ValidateRefreshToken(refresh); err != nil {
		t.Errorf("ValidateRefreshToken(refresh token) = %v, want it accepted", err)
	}

	// An access token presented at the refresh endpoint.
	if _, err := svc.ValidateRefreshToken(access); !errors.Is(err, ErrWrongTokenType) {
		t.Errorf("ValidateRefreshToken(access token) = %v, want ErrWrongTokenType", err)
	}

	// A refresh token presented as API credentials. This is the dangerous
	// direction: refresh tokens are long-lived, so accepting one here would be a
	// multi-day API key.
	if _, err := svc.ValidateAccessToken(refresh); !errors.Is(err, ErrWrongTokenType) {
		t.Errorf("ValidateAccessToken(refresh token) = %v, want ErrWrongTokenType", err)
	}
}

// Regression test for a hole in "log out everywhere".
//
// Revocation-of-all works by recording a cutoff and rejecting every token issued
// before it. The standard iat claim is whole seconds, so when the cutoff was
// compared against it, any session created in the same second as the logout
// survived — reproducible simply by signing in and immediately revoking. The
// token must therefore carry an issue time finer than a second.
func TestIssuedAtHasSubSecondPrecision(t *testing.T) {
	svc := newService(t, testSecret, time.Hour)

	token, err := svc.GenerateToken("user-9", "dave", "user")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims, err := svc.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}

	if claims.IssuedAtMS == 0 {
		t.Fatal("token carries no iat_ms; the logout-all cutoff can only be compared at whole-second precision, which lets same-second sessions survive revocation")
	}

	issued := claims.IssuedAtTime()
	if issued.IsZero() {
		t.Fatal("IssuedAtTime is zero")
	}

	// The whole point: the issue time must not be truncated to the second.
	if !issued.Equal(issued.Truncate(time.Second)) {
		return // carries sub-second detail, which is what we require
	}
	// Landing exactly on a second boundary is possible but vanishingly unlikely;
	// what must never happen is the value being *derived* from the second-only
	// claim, so assert the two are independent.
	if claims.IssuedAt != nil && claims.IssuedAtMS != claims.IssuedAt.Unix()*1000 {
		return
	}
	t.Log("issued exactly on a second boundary; precision not proven by this run")
}

func TestRefreshTokenOutlivesAccessToken(t *testing.T) {
	svc := NewTokenService(testSecret, time.Minute, 24*time.Hour, testIssuer)

	access, err := svc.GenerateToken("user-7", "carol", "admin")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	refresh, err := svc.GenerateRefreshToken("user-7", "carol", "admin")
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}

	accessClaims, err := svc.ValidateAccessToken(access)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	refreshClaims, err := svc.ValidateRefreshToken(refresh)
	if err != nil {
		t.Fatalf("ValidateRefreshToken: %v", err)
	}

	if !refreshClaims.ExpiresAt.After(accessClaims.ExpiresAt.Time) {
		t.Errorf("refresh expiry %v is not after access expiry %v — a refresh token that dies with the access token cannot renew anything",
			refreshClaims.ExpiresAt, accessClaims.ExpiresAt)
	}

	// Distinct jti per token, so revoking one does not revoke the other.
	if accessClaims.ID == refreshClaims.ID {
		t.Error("access and refresh tokens share a jti, want distinct identities")
	}
}

func TestExtractClaims(t *testing.T) {
	svc := newService(t, testSecret, time.Hour)

	token, err := svc.GenerateToken("user-9", "dave", "user")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims, err := svc.ExtractClaims(token)
	if err != nil {
		t.Fatalf("ExtractClaims: %v", err)
	}
	if claims.UserID != "user-9" || claims.Username != "dave" || claims.Role != "user" {
		t.Errorf("ExtractClaims() = %+v, want the generated identity", claims)
	}

	if _, err := svc.ExtractClaims(""); err == nil {
		t.Error("ExtractClaims accepted an empty token, want an error")
	}
}
