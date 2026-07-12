package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
)

// pgUniqueViolation is the SQLSTATE class for a unique-constraint violation.
// The previous implementation detected this by substring-matching the driver's
// error text, which breaks under any locale or driver-wording change.
const pgUniqueViolation = "23505"

// userColumns is the single source of truth for the SELECT list. Each of the
// six read methods previously repeated it, along with its own 19-line Scan.
const userColumns = `
	id, username, email, password_hash, full_name, bio, avatar_url, role,
	email_verified, email_verification_token, password_reset_token,
	password_reset_expiry, last_login_at, oauth_provider, oauth_provider_id,
	oauth_avatar_url, is_banned, ban_reason, ban_expiry, banned_at, banned_by,
	created_at, updated_at, deleted_at`

// UserRepository is the PostgreSQL implementation of repository.UserRepository.
//
// It was previously written against database/sql while the application only
// ever constructs a *pgxpool.Pool, so it could not be wired in at all.
type UserRepository struct {
	pool *pgxpool.Pool
}

var _ repository.UserRepository = (*UserRepository)(nil)

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// scanUser reads one row in userColumns order.
func scanUser(row scanner) (*domain.User, error) {
	var user domain.User
	err := row.Scan(
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
		&user.IsBanned,
		&user.BanReason,
		&user.BanExpiry,
		&user.BannedAt,
		&user.BannedBy,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// getBy runs a single-row lookup with the shared column list.
func (r *UserRepository) getBy(ctx context.Context, whereClause string, arg any) (*domain.User, error) {
	query := `SELECT` + userColumns + ` FROM users WHERE ` + whereClause + ` AND deleted_at IS NULL`

	user, err := scanUser(r.pool.QueryRow(ctx, query, arg))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("getting user: %w", err)
	}
	return user, nil
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	const query = `
		INSERT INTO users (
			id, username, email, password_hash, full_name, bio, avatar_url, role,
			email_verified, email_verification_token, oauth_provider,
			oauth_provider_id, oauth_avatar_url, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err := r.pool.Exec(ctx, query,
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
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %s", domain.ErrUserAlreadyExists, conflictingField(err))
		}
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.getBy(ctx, "id = $1", id)
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return r.getBy(ctx, "username = $1", strings.TrimSpace(username))
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.getBy(ctx, "email = $1", strings.ToLower(strings.TrimSpace(email)))
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	const query = `
		UPDATE users SET
			username = $2, email = $3, password_hash = $4, full_name = $5,
			bio = $6, avatar_url = $7, role = $8, email_verified = $9,
			email_verification_token = $10, password_reset_token = $11,
			password_reset_expiry = $12, last_login_at = $13,
			oauth_provider = $14, oauth_provider_id = $15, oauth_avatar_url = $16,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, query,
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
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %s", domain.ErrUserAlreadyExists, conflictingField(err))
		}
		return fmt.Errorf("updating user %s: %w", user.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// Delete soft-deletes the user. Rows are retained because videos reference them.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.execUser(ctx,
		`UPDATE users SET deleted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`, id)
}

// BanUser bans the user until the given time; a nil until means permanently.
func (r *UserRepository) BanUser(ctx context.Context, id uuid.UUID, reason string, until *time.Time) error {
	return r.execUser(ctx,
		`UPDATE users
		 SET is_banned = TRUE, ban_reason = $2, ban_expiry = $3,
		     banned_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`, id, reason, until)
}

func (r *UserRepository) UnbanUser(ctx context.Context, id uuid.UUID) error {
	return r.execUser(ctx,
		`UPDATE users
		 SET is_banned = FALSE, ban_reason = NULL, ban_expiry = NULL,
		     banned_at = NULL, banned_by = NULL, updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`, id)
}

func (r *UserRepository) List(ctx context.Context, page repository.Page) ([]*domain.User, error) {
	query := `SELECT` + userColumns + `
		FROM users WHERE deleted_at IS NULL
		ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	users := make([]*domain.User, 0, page.Limit)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating users: %w", err)
	}
	return users, nil
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}
	return count, nil
}

// execUser runs a statement keyed by user ID and reports ErrUserNotFound when
// it matches no row.
func (r *UserRepository) execUser(ctx context.Context, query string, id uuid.UUID, args ...any) error {
	tag, err := r.pool.Exec(ctx, query, append([]any{id}, args...)...)
	if err != nil {
		return fmt.Errorf("updating user %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation, identified by SQLSTATE code rather than by error text.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}

// conflictingField names the column a unique violation was raised on, so the
// caller can tell the user which field to change.
func conflictingField(err error) string {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return "field is already in use"
	}
	switch {
	case strings.Contains(pgErr.ConstraintName, "username"):
		return "username is taken"
	case strings.Contains(pgErr.ConstraintName, "email"):
		return "email is already registered"
	default:
		return "field is already in use"
	}
}
