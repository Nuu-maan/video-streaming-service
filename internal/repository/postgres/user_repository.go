package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"orchids-video-streaming/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (
			id, username, email, password_hash, full_name, bio, avatar_url, role,
			email_verified, email_verification_token, oauth_provider, oauth_provider_id,
			oauth_avatar_url, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Bio,
		user.AvatarURL,
		user.Role,
		user.EmailVerified,
		user.EmailVerificationToken,
		user.OAuthProvider,
		user.OAuthProviderID,
		user.OAuthAvatarURL,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			if strings.Contains(err.Error(), "username") {
				return fmt.Errorf("username already exists")
			}
			if strings.Contains(err.Error(), "email") {
				return fmt.Errorf("email already exists")
			}
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, bio, avatar_url, role,
			   email_verified, email_verification_token, password_reset_token, password_reset_expiry,
			   last_login_at, oauth_provider, oauth_provider_id, oauth_avatar_url,
			   created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Bio,
		&user.AvatarURL,
		&user.Role,
		&user.EmailVerified,
		&user.EmailVerificationToken,
		&user.PasswordResetToken,
		&user.PasswordResetExpiry,
		&user.LastLoginAt,
		&user.OAuthProvider,
		&user.OAuthProviderID,
		&user.OAuthAvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, bio, avatar_url, role,
			   email_verified, email_verification_token, password_reset_token, password_reset_expiry,
			   last_login_at, oauth_provider, oauth_provider_id, oauth_avatar_url,
			   created_at, updated_at, deleted_at
		FROM users
		WHERE username = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Bio,
		&user.AvatarURL,
		&user.Role,
		&user.EmailVerified,
		&user.EmailVerificationToken,
		&user.PasswordResetToken,
		&user.PasswordResetExpiry,
		&user.LastLoginAt,
		&user.OAuthProvider,
		&user.OAuthProviderID,
		&user.OAuthAvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, bio, avatar_url, role,
			   email_verified, email_verification_token, password_reset_token, password_reset_expiry,
			   last_login_at, oauth_provider, oauth_provider_id, oauth_avatar_url,
			   created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, strings.ToLower(email)).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Bio,
		&user.AvatarURL,
		&user.Role,
		&user.EmailVerified,
		&user.EmailVerificationToken,
		&user.PasswordResetToken,
		&user.PasswordResetExpiry,
		&user.LastLoginAt,
		&user.OAuthProvider,
		&user.OAuthProviderID,
		&user.OAuthAvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByEmailVerificationToken(ctx context.Context, token uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, bio, avatar_url, role,
			   email_verified, email_verification_token, password_reset_token, password_reset_expiry,
			   last_login_at, oauth_provider, oauth_provider_id, oauth_avatar_url,
			   created_at, updated_at, deleted_at
		FROM users
		WHERE email_verification_token = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Bio,
		&user.AvatarURL,
		&user.Role,
		&user.EmailVerified,
		&user.EmailVerificationToken,
		&user.PasswordResetToken,
		&user.PasswordResetExpiry,
		&user.LastLoginAt,
		&user.OAuthProvider,
		&user.OAuthProviderID,
		&user.OAuthAvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByPasswordResetToken(ctx context.Context, token uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, bio, avatar_url, role,
			   email_verified, email_verification_token, password_reset_token, password_reset_expiry,
			   last_login_at, oauth_provider, oauth_provider_id, oauth_avatar_url,
			   created_at, updated_at, deleted_at
		FROM users
		WHERE password_reset_token = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Bio,
		&user.AvatarURL,
		&user.Role,
		&user.EmailVerified,
		&user.EmailVerificationToken,
		&user.PasswordResetToken,
		&user.PasswordResetExpiry,
		&user.LastLoginAt,
		&user.OAuthProvider,
		&user.OAuthProviderID,
		&user.OAuthAvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByOAuth(ctx context.Context, provider, providerID string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, bio, avatar_url, role,
			   email_verified, email_verification_token, password_reset_token, password_reset_expiry,
			   last_login_at, oauth_provider, oauth_provider_id, oauth_avatar_url,
			   created_at, updated_at, deleted_at
		FROM users
		WHERE oauth_provider = $1 AND oauth_provider_id = $2 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, provider, providerID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Bio,
		&user.AvatarURL,
		&user.Role,
		&user.EmailVerified,
		&user.EmailVerificationToken,
		&user.PasswordResetToken,
		&user.PasswordResetExpiry,
		&user.LastLoginAt,
		&user.OAuthProvider,
		&user.OAuthProviderID,
		&user.OAuthAvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users SET
			username = $2,
			email = $3,
			password_hash = $4,
			full_name = $5,
			bio = $6,
			avatar_url = $7,
			role = $8,
			email_verified = $9,
			email_verification_token = $10,
			password_reset_token = $11,
			password_reset_expiry = $12,
			last_login_at = $13,
			oauth_provider = $14,
			oauth_provider_id = $15,
			oauth_avatar_url = $16,
			updated_at = $17
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Bio,
		user.AvatarURL,
		user.Role,
		user.EmailVerified,
		user.EmailVerificationToken,
		user.PasswordResetToken,
		user.PasswordResetExpiry,
		user.LastLoginAt,
		user.OAuthProvider,
		user.OAuthProviderID,
		user.OAuthAvatarURL,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, bio, avatar_url, role,
			   email_verified, email_verification_token, password_reset_token, password_reset_expiry,
			   last_login_at, oauth_provider, oauth_provider_id, oauth_avatar_url,
			   created_at, updated_at, deleted_at
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.FullName,
			&user.Bio,
			&user.AvatarURL,
			&user.Role,
			&user.EmailVerified,
			&user.EmailVerificationToken,
			&user.PasswordResetToken,
			&user.PasswordResetExpiry,
			&user.LastLoginAt,
			&user.OAuthProvider,
			&user.OAuthProviderID,
			&user.OAuthAvatarURL,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return users, nil
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}
