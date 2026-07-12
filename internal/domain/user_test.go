package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		email        string
		passwordHash string
		role         Role
		wantErr      error
	}{
		{
			name:         "valid user",
			username:     "gopher_1",
			email:        "gopher@example.com",
			passwordHash: "$2a$10$hash",
			role:         RoleUser,
			wantErr:      nil,
		},
		{
			name:         "username too short",
			username:     "ab",
			email:        "gopher@example.com",
			passwordHash: "$2a$10$hash",
			role:         RoleUser,
			wantErr:      ErrInvalidUsername,
		},
		{
			name:         "username too long",
			username:     strings.Repeat("a", 31),
			email:        "gopher@example.com",
			passwordHash: "$2a$10$hash",
			role:         RoleUser,
			wantErr:      ErrInvalidUsername,
		},
		{
			name:         "username with illegal characters",
			username:     "go pher!",
			email:        "gopher@example.com",
			passwordHash: "$2a$10$hash",
			role:         RoleUser,
			wantErr:      ErrInvalidUsername,
		},
		{
			name:         "empty username",
			username:     "",
			email:        "gopher@example.com",
			passwordHash: "$2a$10$hash",
			role:         RoleUser,
			wantErr:      ErrInvalidUsername,
		},
		{
			name:         "email without at sign",
			username:     "gopher",
			email:        "not-an-email",
			passwordHash: "$2a$10$hash",
			role:         RoleUser,
			wantErr:      ErrInvalidEmail,
		},
		{
			name:         "email without domain tld",
			username:     "gopher",
			email:        "gopher@example",
			passwordHash: "$2a$10$hash",
			role:         RoleUser,
			wantErr:      ErrInvalidEmail,
		},
		{
			name:         "empty email",
			username:     "gopher",
			email:        "",
			passwordHash: "$2a$10$hash",
			role:         RoleUser,
			wantErr:      ErrInvalidEmail,
		},
		{
			name:         "empty password hash",
			username:     "gopher",
			email:        "gopher@example.com",
			passwordHash: "",
			role:         RoleUser,
			wantErr:      ErrInvalidPassword,
		},
		{
			name:         "unknown role",
			username:     "gopher",
			email:        "gopher@example.com",
			passwordHash: "$2a$10$hash",
			role:         Role("wizard"),
			wantErr:      ErrInvalidRole,
		},
		{
			name:         "empty role",
			username:     "gopher",
			email:        "gopher@example.com",
			passwordHash: "$2a$10$hash",
			role:         Role(""),
			wantErr:      ErrInvalidRole,
		},
		{
			name:         "admin role is valid",
			username:     "root_admin",
			email:        "admin@example.com",
			passwordHash: "$2a$10$hash",
			role:         RoleAdmin,
			wantErr:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := NewUser(tt.username, tt.email, tt.passwordHash, tt.role)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NewUser() error = %v, want %v", err, tt.wantErr)
				}
				if user != nil {
					t.Errorf("NewUser() returned a user alongside an error: %+v", user)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewUser() unexpected error: %v", err)
			}
			if user.ID == uuid.Nil {
				t.Error("ID was not generated")
			}
			if user.EmailVerified {
				t.Error("EmailVerified = true, want false for a new user")
			}
			if user.Role != tt.role {
				t.Errorf("Role = %q, want %q", user.Role, tt.role)
			}
		})
	}
}

func TestNewUserNormalizesInput(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		email        string
		wantUsername string
		wantEmail    string
	}{
		{
			name:         "uppercase email is lowercased",
			username:     "gopher",
			email:        "GOPHER@EXAMPLE.COM",
			wantUsername: "gopher",
			wantEmail:    "gopher@example.com",
		},
		{
			name:         "mixed case email is lowercased",
			username:     "gopher",
			email:        "Gopher.Dev@Example.Com",
			wantUsername: "gopher",
			wantEmail:    "gopher.dev@example.com",
		},
		{
			name:         "surrounding whitespace is trimmed",
			username:     "  gopher  ",
			email:        "  GOPHER@example.com  ",
			wantUsername: "gopher",
			wantEmail:    "gopher@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := NewUser(tt.username, tt.email, "$2a$10$hash", RoleUser)
			if err != nil {
				t.Fatalf("NewUser() unexpected error: %v", err)
			}
			if user.Email != tt.wantEmail {
				t.Errorf("Email = %q, want %q", user.Email, tt.wantEmail)
			}
			if user.Username != tt.wantUsername {
				t.Errorf("Username = %q, want %q", user.Username, tt.wantUsername)
			}
		})
	}
}

func TestUserIsCurrentlyBanned(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	tests := []struct {
		name      string
		isBanned  bool
		banExpiry *time.Time
		want      bool
	}{
		{
			name:      "not banned",
			isBanned:  false,
			banExpiry: nil,
			want:      false,
		},
		{
			// A stale expiry on a user who is not banned must not resurrect a ban.
			name:      "not banned, stale expiry present",
			isBanned:  false,
			banExpiry: &future,
			want:      false,
		},
		{
			// A nil expiry means permanent, never "already expired".
			name:      "banned permanently (nil expiry)",
			isBanned:  true,
			banExpiry: nil,
			want:      true,
		},
		{
			name:      "banned, expiry in the future",
			isBanned:  true,
			banExpiry: &future,
			want:      true,
		},
		{
			name:      "banned, expiry in the past",
			isBanned:  true,
			banExpiry: &past,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{IsBanned: tt.isBanned, BanExpiry: tt.banExpiry}
			if got := user.IsCurrentlyBanned(); got != tt.want {
				t.Errorf("IsCurrentlyBanned() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestUserJSONDoesNotLeakSecrets guards the `json:"-"` tags on the credential
// fields. Any handler returning a User would otherwise ship the bcrypt hash and
// the live reset token to the client.
func TestUserJSONDoesNotLeakSecrets(t *testing.T) {
	verificationToken := uuid.New()
	resetToken := uuid.New()
	resetExpiry := time.Now().Add(time.Hour)
	bannedBy := uuid.New()
	deletedAt := time.Now()

	user := &User{
		ID:                     uuid.New(),
		Username:               "gopher",
		Email:                  "gopher@example.com",
		PasswordHash:           "$2a$10$SUPERSECRETBCRYPTHASHVALUE",
		Role:                   RoleUser,
		EmailVerificationToken: &verificationToken,
		PasswordResetToken:     &resetToken,
		PasswordResetExpiry:    &resetExpiry,
		BannedBy:               &bannedBy,
		DeletedAt:              &deletedAt,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("json.Marshal() unexpected error: %v", err)
	}
	payload := string(data)

	secrets := []struct {
		name  string
		value string
	}{
		{"password hash", user.PasswordHash},
		{"email verification token", verificationToken.String()},
		{"password reset token", resetToken.String()},
	}
	for _, secret := range secrets {
		if strings.Contains(payload, secret.value) {
			t.Errorf("marshalled User leaks the %s: %s", secret.name, payload)
		}
	}

	// Belt and braces: the field names must be absent too, not merely their
	// current values.
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() unexpected error: %v", err)
	}
	for _, field := range []string{
		"PasswordHash", "password_hash",
		"EmailVerificationToken", "email_verification_token",
		"PasswordResetToken", "password_reset_token",
		"PasswordResetExpiry", "password_reset_expiry",
	} {
		if _, ok := decoded[field]; ok {
			t.Errorf("marshalled User contains secret field %q", field)
		}
	}

	// Sanity check that the public fields do survive, so a test that passes
	// because marshalling silently produced nothing would still fail.
	if _, ok := decoded["username"]; !ok {
		t.Error("marshalled User is missing the public username field")
	}
	if _, ok := decoded["email"]; !ok {
		t.Error("marshalled User is missing the public email field")
	}
}
