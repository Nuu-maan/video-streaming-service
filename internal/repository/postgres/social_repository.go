package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
)

// commentColumns is the single source of truth for comment SELECTs. Every
// comment read joins users, because the API contract promises a username and
// avatar on each comment. The avatar precedence (OAuth first) mirrors
// domain.User.GetAvatarURL.
const commentColumns = `
	c.id, c.video_id, c.user_id, c.parent_id, c.content, c.like_count,
	c.reply_count, c.pinned, c.edited_at, c.created_at, c.updated_at,
	c.deleted_at, u.username, COALESCE(u.oauth_avatar_url, u.avatar_url, '')`

const notificationColumns = `
	id, user_id, type, title, COALESCE(message, ''), action_url, actor_id,
	video_id, comment_id, read, created_at`

// SocialRepository is the PostgreSQL store behind likes, comments,
// subscriptions, playlists, watch-later and notifications.
//
// The counter columns touched by these tables (videos.like_count,
// videos.comment_count, comments.reply_count, users.subscriber_count,
// playlists.video_count) are maintained by database triggers installed in
// migration 9. No method here may update them: doing so double-counts.
type SocialRepository struct {
	pool *pgxpool.Pool
}

func NewSocialRepository(pool *pgxpool.Pool) *SocialRepository {
	return &SocialRepository{pool: pool}
}

func scanComment(row scanner) (*domain.Comment, error) {
	var c domain.Comment
	err := row.Scan(
		&c.ID,
		&c.VideoID,
		&c.UserID,
		&c.ParentID,
		&c.Content,
		&c.LikeCount,
		&c.ReplyCount,
		&c.Pinned,
		&c.EditedAt,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.DeletedAt,
		&c.Username,
		&c.AvatarURL,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// scanVideoWith reads one row that begins with videoColumns and continues with
// extra destinations, for listings that join videos to a membership table.
// scanVideo cannot be reused there because pgx scans a row in a single call.
func scanVideoWith(row scanner, extra ...any) (*domain.Video, error) {
	var v domain.Video
	if err := row.Scan(append(videoScanDest(&v), extra...)...); err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *SocialRepository) count(ctx context.Context, query string, args ...any) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting rows: %w", err)
	}
	return count, nil
}

// UpsertLike records a rating, converting an existing opposite rating in
// place. The likes trigger adjusts videos.like_count on both the INSERT and
// the UPDATE branch, so this deliberately touches no counter. RETURNING hands
// back the surviving row's identity so the caller does not report the
// discarded insert candidate.
func (r *SocialRepository) UpsertLike(ctx context.Context, like *domain.Like) error {
	query := `
	INSERT INTO likes (id, user_id, video_id, is_like, created_at)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (user_id, video_id) DO UPDATE SET is_like = EXCLUDED.is_like
	RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		like.ID, like.UserID, like.VideoID, like.IsLike, like.CreatedAt,
	).Scan(&like.ID, &like.CreatedAt)
	if err != nil {
		return fmt.Errorf("upserting like: %w", err)
	}
	return nil
}

func (r *SocialRepository) DeleteLike(ctx context.Context, userID, videoID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM likes WHERE user_id = $1 AND video_id = $2`, userID, videoID)
	if err != nil {
		return fmt.Errorf("deleting like: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrLikeNotFound
	}
	return nil
}

func (r *SocialRepository) GetLike(ctx context.Context, userID, videoID uuid.UUID) (*domain.Like, error) {
	query := `
	SELECT id, user_id, video_id, is_like, created_at
	FROM likes
	WHERE user_id = $1 AND video_id = $2`

	like := &domain.Like{}
	err := r.pool.QueryRow(ctx, query, userID, videoID).Scan(
		&like.ID, &like.UserID, &like.VideoID, &like.IsLike, &like.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrLikeNotFound
		}
		return nil, fmt.Errorf("getting like: %w", err)
	}
	return like, nil
}

func (r *SocialRepository) CreateComment(ctx context.Context, comment *domain.Comment) error {
	query := `
	INSERT INTO comments (id, video_id, user_id, parent_id, content, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		comment.ID, comment.VideoID, comment.UserID, comment.ParentID,
		comment.Content, comment.CreatedAt, comment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating comment: %w", err)
	}
	return nil
}

// GetCommentByID returns the comment even when soft-deleted: the service needs
// the deleted row to distinguish "never existed" from "deleted", and to keep a
// deleted comment from being deleted (and trigger-decremented) twice.
func (r *SocialRepository) GetCommentByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	query := `SELECT` + commentColumns + ` FROM comments c JOIN users u ON u.id = c.user_id WHERE c.id = $1`

	comment, err := scanComment(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCommentNotFound
		}
		return nil, fmt.Errorf("getting comment %s: %w", id, err)
	}
	return comment, nil
}

func (r *SocialRepository) ListComments(ctx context.Context, videoID uuid.UUID, page repository.Page) ([]*domain.Comment, error) {
	query := `SELECT` + commentColumns + `
	FROM comments c JOIN users u ON u.id = c.user_id
	WHERE c.video_id = $1 AND c.parent_id IS NULL AND c.deleted_at IS NULL
	ORDER BY c.pinned DESC, c.created_at DESC
	LIMIT $2 OFFSET $3`

	return r.listComments(ctx, query, videoID, page)
}

func (r *SocialRepository) CountComments(ctx context.Context, videoID uuid.UUID) (int, error) {
	return r.count(ctx,
		`SELECT COUNT(*) FROM comments WHERE video_id = $1 AND parent_id IS NULL AND deleted_at IS NULL`,
		videoID,
	)
}

// ListReplies is ordered oldest-first: a thread reads top-down, unlike the
// top-level listing which surfaces the newest conversation.
func (r *SocialRepository) ListReplies(ctx context.Context, parentID uuid.UUID, page repository.Page) ([]*domain.Comment, error) {
	query := `SELECT` + commentColumns + `
	FROM comments c JOIN users u ON u.id = c.user_id
	WHERE c.parent_id = $1 AND c.deleted_at IS NULL
	ORDER BY c.created_at ASC
	LIMIT $2 OFFSET $3`

	return r.listComments(ctx, query, parentID, page)
}

func (r *SocialRepository) CountReplies(ctx context.Context, parentID uuid.UUID) (int, error) {
	return r.count(ctx,
		`SELECT COUNT(*) FROM comments WHERE parent_id = $1 AND deleted_at IS NULL`,
		parentID,
	)
}

func (r *SocialRepository) listComments(ctx context.Context, query string, id uuid.UUID, page repository.Page) ([]*domain.Comment, error) {
	rows, err := r.pool.Query(ctx, query, id, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("listing comments: %w", err)
	}
	defer rows.Close()

	comments := make([]*domain.Comment, 0, page.Limit)
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning comment: %w", err)
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating comments: %w", err)
	}
	return comments, nil
}

func (r *SocialRepository) UpdateCommentContent(ctx context.Context, id uuid.UUID, content string) error {
	query := `
	UPDATE comments SET content = $2, edited_at = NOW()
	WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, id, content)
	if err != nil {
		return fmt.Errorf("updating comment %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCommentNotFound
	}
	return nil
}

// SoftDeleteComment marks the comment deleted rather than removing the row:
// the comments trigger fixes videos.comment_count and the parent's reply_count
// on exactly this NULL -> NOT NULL transition, and a hard DELETE would also
// cascade away the reply subtree.
func (r *SocialRepository) SoftDeleteComment(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE comments SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting comment %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCommentNotFound
	}
	return nil
}

// CreateSubscription is idempotent and reports whether a row was actually
// inserted, so the caller can notify the creator only on a genuinely new
// subscription and not on every repeated click.
func (r *SocialRepository) CreateSubscription(ctx context.Context, sub *domain.Subscription) (bool, error) {
	query := `
	INSERT INTO subscriptions (id, subscriber_id, creator_id, notify_uploads, created_at)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (subscriber_id, creator_id) DO NOTHING`

	tag, err := r.pool.Exec(ctx, query,
		sub.ID, sub.SubscriberID, sub.CreatorID, sub.NotifyUploads, sub.CreatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("creating subscription: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *SocialRepository) DeleteSubscription(ctx context.Context, subscriberID, creatorID uuid.UUID) error {
	query := `DELETE FROM subscriptions WHERE subscriber_id = $1 AND creator_id = $2`

	tag, err := r.pool.Exec(ctx, query, subscriberID, creatorID)
	if err != nil {
		return fmt.Errorf("deleting subscription: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSubscriptionNotFound
	}
	return nil
}

func (r *SocialRepository) ListSubscribers(ctx context.Context, creatorID uuid.UUID, page repository.Page) ([]*domain.SubscriptionEntry, error) {
	query := `
	SELECT u.id, u.username, COALESCE(u.oauth_avatar_url, u.avatar_url, ''),
	       u.subscriber_count, s.notify_uploads, s.created_at
	FROM subscriptions s JOIN users u ON u.id = s.subscriber_id
	WHERE s.creator_id = $1
	ORDER BY s.created_at DESC
	LIMIT $2 OFFSET $3`

	return r.listSubscriptionEntries(ctx, query, creatorID, page)
}

func (r *SocialRepository) CountSubscribers(ctx context.Context, creatorID uuid.UUID) (int, error) {
	return r.count(ctx, `SELECT COUNT(*) FROM subscriptions WHERE creator_id = $1`, creatorID)
}

func (r *SocialRepository) ListSubscriptions(ctx context.Context, subscriberID uuid.UUID, page repository.Page) ([]*domain.SubscriptionEntry, error) {
	query := `
	SELECT u.id, u.username, COALESCE(u.oauth_avatar_url, u.avatar_url, ''),
	       u.subscriber_count, s.notify_uploads, s.created_at
	FROM subscriptions s JOIN users u ON u.id = s.creator_id
	WHERE s.subscriber_id = $1
	ORDER BY s.created_at DESC
	LIMIT $2 OFFSET $3`

	return r.listSubscriptionEntries(ctx, query, subscriberID, page)
}

func (r *SocialRepository) CountSubscriptions(ctx context.Context, subscriberID uuid.UUID) (int, error) {
	return r.count(ctx, `SELECT COUNT(*) FROM subscriptions WHERE subscriber_id = $1`, subscriberID)
}

func (r *SocialRepository) listSubscriptionEntries(ctx context.Context, query string, id uuid.UUID, page repository.Page) ([]*domain.SubscriptionEntry, error) {
	rows, err := r.pool.Query(ctx, query, id, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("listing subscriptions: %w", err)
	}
	defer rows.Close()

	entries := make([]*domain.SubscriptionEntry, 0, page.Limit)
	for rows.Next() {
		entry := &domain.SubscriptionEntry{}
		err := rows.Scan(
			&entry.UserID, &entry.Username, &entry.AvatarURL,
			&entry.SubscriberCount, &entry.NotifyUploads, &entry.SubscribedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning subscription entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating subscriptions: %w", err)
	}
	return entries, nil
}

const playlistColumns = `
	id, user_id, title, COALESCE(description, ''), visibility, video_count,
	created_at, updated_at`

func scanPlaylist(row scanner) (*domain.Playlist, error) {
	var p domain.Playlist
	err := row.Scan(
		&p.ID, &p.UserID, &p.Title, &p.Description, &p.Visibility,
		&p.VideoCount, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *SocialRepository) CreatePlaylist(ctx context.Context, playlist *domain.Playlist) error {
	query := `
	INSERT INTO playlists (id, user_id, title, description, visibility, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		playlist.ID, playlist.UserID, playlist.Title, playlist.Description,
		playlist.Visibility, playlist.CreatedAt, playlist.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating playlist: %w", err)
	}
	return nil
}

func (r *SocialRepository) GetPlaylistByID(ctx context.Context, id uuid.UUID) (*domain.Playlist, error) {
	query := `SELECT` + playlistColumns + ` FROM playlists WHERE id = $1`

	playlist, err := scanPlaylist(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPlaylistNotFound
		}
		return nil, fmt.Errorf("getting playlist %s: %w", id, err)
	}
	return playlist, nil
}

// UpdatePlaylist persists the mutable fields; updated_at is bumped by the
// playlists trigger.
func (r *SocialRepository) UpdatePlaylist(ctx context.Context, playlist *domain.Playlist) error {
	query := `UPDATE playlists SET title = $2, description = $3, visibility = $4 WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		playlist.ID, playlist.Title, playlist.Description, playlist.Visibility,
	)
	if err != nil {
		return fmt.Errorf("updating playlist %s: %w", playlist.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPlaylistNotFound
	}
	return nil
}

func (r *SocialRepository) DeletePlaylist(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM playlists WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting playlist %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPlaylistNotFound
	}
	return nil
}

func (r *SocialRepository) ListPlaylistsByUser(ctx context.Context, userID uuid.UUID, page repository.Page) ([]*domain.Playlist, error) {
	query := `SELECT` + playlistColumns + `
	FROM playlists WHERE user_id = $1
	ORDER BY updated_at DESC
	LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("listing playlists: %w", err)
	}
	defer rows.Close()

	playlists := make([]*domain.Playlist, 0, page.Limit)
	for rows.Next() {
		playlist, err := scanPlaylist(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning playlist: %w", err)
		}
		playlists = append(playlists, playlist)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating playlists: %w", err)
	}
	return playlists, nil
}

func (r *SocialRepository) CountPlaylistsByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.count(ctx, `SELECT COUNT(*) FROM playlists WHERE user_id = $1`, userID)
}

// AddPlaylistVideo appends the video at MAX(position)+1. The playlist row is
// locked first: UNIQUE(playlist_id, position) means two concurrent appends
// that read the same MAX would collide, so appends to one playlist must be
// serialized. The lock also confirms the playlist still exists inside the same
// transaction. playlists.video_count is maintained by trigger.
func (r *SocialRepository) AddPlaylistVideo(ctx context.Context, playlistID, videoID uuid.UUID) (*domain.PlaylistVideo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var locked uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM playlists WHERE id = $1 FOR UPDATE`, playlistID).Scan(&locked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPlaylistNotFound
		}
		return nil, fmt.Errorf("locking playlist %s: %w", playlistID, err)
	}

	entry := &domain.PlaylistVideo{
		ID:         uuid.New(),
		PlaylistID: playlistID,
		VideoID:    videoID,
	}

	query := `
	INSERT INTO playlist_videos (id, playlist_id, video_id, position, added_at)
	SELECT $1, $2, $3, COALESCE(MAX(position), -1) + 1, NOW()
	FROM playlist_videos WHERE playlist_id = $2
	RETURNING position, added_at`

	err = tx.QueryRow(ctx, query, entry.ID, playlistID, videoID).Scan(&entry.Position, &entry.AddedAt)
	if err != nil {
		// With the playlist locked, a position collision is impossible, so the
		// only unique constraint left to trip is (playlist_id, video_id).
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil, domain.ErrVideoAlreadyInPlaylist
		}
		return nil, fmt.Errorf("adding video %s to playlist %s: %w", videoID, playlistID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing playlist add: %w", err)
	}
	return entry, nil
}

// RemovePlaylistVideo leaves a gap in the position sequence on purpose:
// ordering only needs relative positions, and renumbering the tail would
// rewrite rows under UNIQUE(playlist_id, position) for no observable benefit.
func (r *SocialRepository) RemovePlaylistVideo(ctx context.Context, playlistID, videoID uuid.UUID) error {
	query := `DELETE FROM playlist_videos WHERE playlist_id = $1 AND video_id = $2`

	tag, err := r.pool.Exec(ctx, query, playlistID, videoID)
	if err != nil {
		return fmt.Errorf("removing video from playlist: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPlaylistVideoNotFound
	}
	return nil
}

// ListPlaylistVideos joins through a derived table that exposes only
// video_id/position/added_at, so every videoColumns name stays unambiguous
// against the join without qualifying the shared column list.
func (r *SocialRepository) ListPlaylistVideos(ctx context.Context, playlistID uuid.UUID, page repository.Page) ([]*domain.PlaylistItem, error) {
	query := `SELECT` + videoColumns + `, pv.position, pv.added_at
	FROM (SELECT video_id, position, added_at FROM playlist_videos WHERE playlist_id = $1) pv
	JOIN videos v ON v.id = pv.video_id
	ORDER BY pv.position
	LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, playlistID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("listing playlist videos: %w", err)
	}
	defer rows.Close()

	items := make([]*domain.PlaylistItem, 0, page.Limit)
	for rows.Next() {
		item := &domain.PlaylistItem{}
		video, err := scanVideoWith(rows, &item.Position, &item.AddedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning playlist video: %w", err)
		}
		item.Video = video
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating playlist videos: %w", err)
	}
	return items, nil
}

func (r *SocialRepository) CountPlaylistVideos(ctx context.Context, playlistID uuid.UUID) (int, error) {
	return r.count(ctx, `SELECT COUNT(*) FROM playlist_videos WHERE playlist_id = $1`, playlistID)
}

func (r *SocialRepository) UpsertWatchLater(ctx context.Context, entry *domain.WatchLater) error {
	query := `
	INSERT INTO watch_later (id, user_id, video_id, added_at)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (user_id, video_id) DO NOTHING`

	_, err := r.pool.Exec(ctx, query, entry.ID, entry.UserID, entry.VideoID, entry.AddedAt)
	if err != nil {
		return fmt.Errorf("adding to watch later: %w", err)
	}
	return nil
}

func (r *SocialRepository) DeleteWatchLater(ctx context.Context, userID, videoID uuid.UUID) error {
	query := `DELETE FROM watch_later WHERE user_id = $1 AND video_id = $2`

	tag, err := r.pool.Exec(ctx, query, userID, videoID)
	if err != nil {
		return fmt.Errorf("removing from watch later: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrWatchLaterNotFound
	}
	return nil
}

func (r *SocialRepository) ListWatchLater(ctx context.Context, userID uuid.UUID, page repository.Page) ([]*domain.WatchLaterItem, error) {
	query := `SELECT` + videoColumns + `, wl.added_at
	FROM (SELECT video_id, added_at FROM watch_later WHERE user_id = $1) wl
	JOIN videos v ON v.id = wl.video_id
	ORDER BY wl.added_at DESC
	LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("listing watch later: %w", err)
	}
	defer rows.Close()

	items := make([]*domain.WatchLaterItem, 0, page.Limit)
	for rows.Next() {
		item := &domain.WatchLaterItem{}
		video, err := scanVideoWith(rows, &item.AddedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning watch later entry: %w", err)
		}
		item.Video = video
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating watch later: %w", err)
	}
	return items, nil
}

func (r *SocialRepository) CountWatchLater(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.count(ctx, `SELECT COUNT(*) FROM watch_later WHERE user_id = $1`, userID)
}

func (r *SocialRepository) CreateNotification(ctx context.Context, n *domain.Notification) error {
	query := `
	INSERT INTO notifications (id, user_id, type, title, message, action_url, actor_id, video_id, comment_id, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		n.ID, n.UserID, n.Type, n.Title, n.Message,
		n.ActionURL, n.ActorID, n.VideoID, n.CommentID, n.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating notification: %w", err)
	}
	return nil
}

func (r *SocialRepository) ListNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool, page repository.Page) ([]*domain.Notification, error) {
	query := `SELECT` + notificationColumns + ` FROM notifications WHERE user_id = $1`
	if unreadOnly {
		query += ` AND read = false`
	}
	query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("listing notifications: %w", err)
	}
	defer rows.Close()

	notifications := make([]*domain.Notification, 0, page.Limit)
	for rows.Next() {
		n := &domain.Notification{}
		err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message,
			&n.ActionURL, &n.ActorID, &n.VideoID, &n.CommentID, &n.Read, &n.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning notification: %w", err)
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating notifications: %w", err)
	}
	return notifications, nil
}

func (r *SocialRepository) CountNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1`
	if unreadOnly {
		query += ` AND read = false`
	}
	return r.count(ctx, query, userID)
}

// MarkNotificationRead scopes the update to the owner, so a caller can never
// mark another user's notification (or probe whether it exists).
func (r *SocialRepository) MarkNotificationRead(ctx context.Context, id, userID uuid.UUID) error {
	query := `UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2`

	tag, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("marking notification %s read: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}

func (r *SocialRepository) MarkAllNotificationsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `UPDATE notifications SET read = true WHERE user_id = $1 AND read = false`

	tag, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("marking notifications read: %w", err)
	}
	return tag.RowsAffected(), nil
}
