package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                      uuid.UUID
	Username                string
	Email                   string
	PasswordHash            string
	FullName                *string
	Bio                     *string
	AvatarURL               *string
	Role                    Role
	EmailVerified           bool
	EmailVerificationToken  *uuid.UUID
	PasswordResetToken      *uuid.UUID
	PasswordResetExpiry     *time.Time
	LastLoginAt             *time.Time
	OAuthProvider           *string
	OAuthProviderID         *string
	OAuthAvatarURL          *string
	CreatedAt               time.Time
	UpdatedAt               time.Time
	DeletedAt               *time.Time
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

func (u *User) GenerateEmailVerificationToken() uuid.UUID {
	token := uuid.New()
	u.EmailVerificationToken = &token
	u.UpdatedAt = time.Now()
	return token
}

func (u *User) GeneratePasswordResetToken(expiryDuration time.Duration) uuid.UUID {
	token := uuid.New()
	expiry := time.Now().Add(expiryDuration)
	u.PasswordResetToken = &token
	u.PasswordResetExpiry = &expiry
	u.UpdatedAt = time.Now()
	return token
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
