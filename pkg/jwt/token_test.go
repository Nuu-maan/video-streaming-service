package jwt

import (
	"encoding/base64"
	"encoding/json"
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
	return NewTokenService(secret, d, testIssuer)
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

func TestRefreshToken(t *testing.T) {
	svc := newService(t, testSecret, time.Hour)

	token, err := svc.GenerateToken("user-7", "carol", "admin")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	refreshed, err := svc.RefreshToken(token)
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}

	claims, err := svc.ValidateToken(refreshed)
	if err != nil {
		t.Fatalf("ValidateToken(refreshed): %v", err)
	}
	if claims.UserID != "user-7" || claims.Username != "carol" || claims.Role != "admin" {
		t.Errorf("refreshed claims = %+v, want the original identity preserved", claims)
	}

	if _, err := svc.RefreshToken("garbage"); err == nil {
		t.Error("RefreshToken accepted a garbage token, want an error")
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
