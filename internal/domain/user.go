package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// User is a registered account.
//
// Secrets carry `json:"-"`. Without it, any handler that returns a User — the
// auth response, an admin user listing — would serialize the bcrypt hash and
// the live password-reset token straight to the client.
type User struct {
	ID                     uuid.UUID  `json:"id"`
	Username               string     `json:"username"`
	Email                  string     `json:"email"`
	PasswordHash           string     `json:"-"`
	FullName               *string    `json:"full_name,omitempty"`
	Bio                    *string    `json:"bio,omitempty"`
	AvatarURL              *string    `json:"avatar_url,omitempty"`
	Role                   Role       `json:"role"`
	EmailVerified          bool       `json:"email_verified"`
	EmailVerificationToken *uuid.UUID `json:"-"`
	PasswordResetToken     *uuid.UUID `json:"-"`
	PasswordResetExpiry    *time.Time `json:"-"`
	LastLoginAt            *time.Time `json:"last_login_at,omitempty"`
	OAuthProvider          *string    `json:"oauth_provider,omitempty"`
	OAuthProviderID        *string    `json:"-"`
	OAuthAvatarURL         *string    `json:"oauth_avatar_url,omitempty"`

	// Moderation state, from migration 7. BanExpiry is nil for a permanent ban,
	// so a nil expiry must never be read as "already expired".
	IsBanned  bool       `json:"is_banned"`
	BanReason *string    `json:"ban_reason,omitempty"`
	BanExpiry *time.Time `json:"ban_expiry,omitempty"`
	BannedAt  *time.Time `json:"banned_at,omitempty"`
	BannedBy  *uuid.UUID `json:"-"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-"`
}

// IsCurrentlyBanned reports whether the user is banned right now. A temporary
// ban whose expiry has passed is no longer in force even though is_banned is
// still true in the row.
func (u *User) IsCurrentlyBanned() bool {
	if !u.IsBanned {
		return false
	}
	if u.BanExpiry == nil {
		return true // permanent
	}
	return time.Now().Before(*u.BanExpiry)
}

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

func NewUser(username, email, passwordHash string, role Role) (*User, error) {
	user := &User{
		ID:            uuid.New(),
		Username:      strings.TrimSpace(username),
		Email:         strings.TrimSpace(strings.ToLower(email)),
		PasswordHash:  passwordHash,
		Role:          role,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *User) Validate() error {
	if !usernameRegex.MatchString(u.Username) {
		return ErrInvalidUsername
	}

	if !emailRegex.MatchString(u.Email) {
		return ErrInvalidEmail
	}

	if u.PasswordHash == "" {
		return ErrInvalidPassword
	}

	if !u.Role.IsValid() {
		return ErrInvalidRole
	}

	if u.Bio != nil && len(*u.Bio) > 500 {
		return ErrBioTooLong
	}

	if u.FullName != nil && len(*u.FullName) > 100 {
		return ErrFullNameTooLong
	}

	return nil
}

func (u *User) IsEmailVerified() bool {
	return u.EmailVerified
}

func (u *User) CanUploadVideos() bool {
	return u.EmailVerified && u.HasPermission(PermissionUploadVideo)
}

func (u *User) HasRole(role Role) bool {
	return u.Role == role
}

func (u *User) HasPermission(permission Permission) bool {
	return u.Role.HasPermission(permission)
}

func (u *User) VerifyEmail() {
	u.EmailVerified = true
	u.EmailVerificationToken = nil
	u.UpdatedAt = time.Now()
}

func (u *User) ClearPasswordResetToken() {
	u.PasswordResetToken = nil
	u.PasswordResetExpiry = nil
	u.UpdatedAt = time.Now()
}

func (u *User) IsPasswordResetTokenValid() bool {
	if u.PasswordResetToken == nil || u.PasswordResetExpiry == nil {
		return false
	}
	return time.Now().Before(*u.PasswordResetExpiry)
}

func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}

func (u *User) UpdateProfile(fullName, bio *string) error {
	if fullName != nil {
		if len(*fullName) > 100 {
			return ErrFullNameTooLong
		}
		u.FullName = fullName
	}

	if bio != nil {
		if len(*bio) > 500 {
			return ErrBioTooLong
		}
		u.Bio = bio
	}

	u.UpdatedAt = time.Now()
	return nil
}

func (u *User) SetAvatarURL(url string) {
	u.AvatarURL = &url
	u.UpdatedAt = time.Now()
}

func (u *User) SoftDelete() {
	now := time.Now()
	u.DeletedAt = &now
	u.UpdatedAt = now
}

func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

func (u *User) DisplayName() string {
	if u.FullName != nil && *u.FullName != "" {
		return *u.FullName
	}
	return u.Username
}

func (u *User) GetAvatarURL() string {
	if u.OAuthAvatarURL != nil && *u.OAuthAvatarURL != "" {
		return *u.OAuthAvatarURL
	}
	if u.AvatarURL != nil && *u.AvatarURL != "" {
		return *u.AvatarURL
	}
	return fmt.Sprintf("https://ui-avatars.com/api/?name=%s&background=random", u.Username)
}
