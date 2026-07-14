package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
)

// SocialRepository is the slice of the social store this service needs.
// Satisfied by *postgres.SocialRepository.
type SocialRepository interface {
	UpsertLike(ctx context.Context, like *domain.Like) error
	DeleteLike(ctx context.Context, userID, videoID uuid.UUID) error
	GetLike(ctx context.Context, userID, videoID uuid.UUID) (*domain.Like, error)

	CreateComment(ctx context.Context, comment *domain.Comment) error
	GetCommentByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error)
	ListComments(ctx context.Context, videoID uuid.UUID, page repository.Page) ([]*domain.Comment, error)
	CountComments(ctx context.Context, videoID uuid.UUID) (int, error)
	ListReplies(ctx context.Context, parentID uuid.UUID, page repository.Page) ([]*domain.Comment, error)
	CountReplies(ctx context.Context, parentID uuid.UUID) (int, error)
	UpdateCommentContent(ctx context.Context, id uuid.UUID, content string) error
	SoftDeleteComment(ctx context.Context, id uuid.UUID) error

	CreateSubscription(ctx context.Context, sub *domain.Subscription) (bool, error)
	DeleteSubscription(ctx context.Context, subscriberID, creatorID uuid.UUID) error
	ListSubscribers(ctx context.Context, creatorID uuid.UUID, page repository.Page) ([]*domain.SubscriptionEntry, error)
	CountSubscribers(ctx context.Context, creatorID uuid.UUID) (int, error)
	ListSubscriptions(ctx context.Context, subscriberID uuid.UUID, page repository.Page) ([]*domain.SubscriptionEntry, error)
	CountSubscriptions(ctx context.Context, subscriberID uuid.UUID) (int, error)

	CreatePlaylist(ctx context.Context, playlist *domain.Playlist) error
	GetPlaylistByID(ctx context.Context, id uuid.UUID) (*domain.Playlist, error)
	UpdatePlaylist(ctx context.Context, playlist *domain.Playlist) error
	DeletePlaylist(ctx context.Context, id uuid.UUID) error
	ListPlaylistsByUser(ctx context.Context, userID uuid.UUID, page repository.Page) ([]*domain.Playlist, error)
	CountPlaylistsByUser(ctx context.Context, userID uuid.UUID) (int, error)
	AddPlaylistVideo(ctx context.Context, playlistID, videoID uuid.UUID) (*domain.PlaylistVideo, error)
	RemovePlaylistVideo(ctx context.Context, playlistID, videoID uuid.UUID) error
	ListPlaylistVideos(ctx context.Context, playlistID uuid.UUID, page repository.Page) ([]*domain.PlaylistItem, error)
	CountPlaylistVideos(ctx context.Context, playlistID uuid.UUID) (int, error)

	UpsertWatchLater(ctx context.Context, entry *domain.WatchLater) error
	DeleteWatchLater(ctx context.Context, userID, videoID uuid.UUID) error
	ListWatchLater(ctx context.Context, userID uuid.UUID, page repository.Page) ([]*domain.WatchLaterItem, error)
	CountWatchLater(ctx context.Context, userID uuid.UUID) (int, error)

	CreateNotification(ctx context.Context, n *domain.Notification) error
	ListNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool, page repository.Page) ([]*domain.Notification, error)
	CountNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool) (int, error)
	MarkNotificationRead(ctx context.Context, id, userID uuid.UUID) error
	MarkAllNotificationsRead(ctx context.Context, userID uuid.UUID) (int64, error)
}

// SocialVideoRepository is the slice of the video store this service needs.
// Satisfied by *postgres.PostgresVideoRepository.
type SocialVideoRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Video, error)
}

// SocialUserRepository is the slice of the user store this service needs.
// Satisfied by *postgres.UserRepository.
type SocialUserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type SocialService struct {
	repo   SocialRepository
	videos SocialVideoRepository
	users  SocialUserRepository
	log    *logger.Logger
}

func NewSocialService(
	repo SocialRepository,
	videos SocialVideoRepository,
	users SocialUserRepository,
	log *logger.Logger,
) *SocialService {
	return &SocialService{
		repo:   repo,
		videos: videos,
		users:  users,
		log:    log,
	}
}

// SetLike records or flips the caller's rating of a video. The database
// trigger reconciles videos.like_count on both the insert and the flip.
func (s *SocialService) SetLike(ctx context.Context, userID, videoID uuid.UUID, isLike bool) (*domain.Like, error) {
	if _, err := s.videos.GetByID(ctx, videoID); err != nil {
		return nil, err
	}

	like := &domain.Like{
		ID:        uuid.New(),
		UserID:    userID,
		VideoID:   videoID,
		IsLike:    isLike,
		CreatedAt: time.Now(),
	}
	if err := like.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.UpsertLike(ctx, like); err != nil {
		return nil, err
	}
	return like, nil
}

func (s *SocialService) RemoveLike(ctx context.Context, userID, videoID uuid.UUID) error {
	return s.repo.DeleteLike(ctx, userID, videoID)
}

func (s *SocialService) GetLike(ctx context.Context, userID, videoID uuid.UUID) (*domain.Like, error) {
	return s.repo.GetLike(ctx, userID, videoID)
}

func (s *SocialService) ListComments(ctx context.Context, videoID uuid.UUID, page repository.Page) ([]*domain.Comment, int, error) {
	if _, err := s.videos.GetByID(ctx, videoID); err != nil {
		return nil, 0, err
	}

	comments, err := s.repo.ListComments(ctx, videoID, page)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountComments(ctx, videoID)
	if err != nil {
		return nil, 0, err
	}
	return comments, total, nil
}

func (s *SocialService) ListReplies(ctx context.Context, parentID uuid.UUID, page repository.Page) ([]*domain.Comment, int, error) {
	parent, err := s.repo.GetCommentByID(ctx, parentID)
	if err != nil {
		return nil, 0, err
	}
	if parent.DeletedAt != nil {
		return nil, 0, domain.ErrCommentNotFound
	}

	replies, err := s.repo.ListReplies(ctx, parentID, page)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountReplies(ctx, parentID)
	if err != nil {
		return nil, 0, err
	}
	return replies, total, nil
}

func (s *SocialService) CreateComment(ctx context.Context, userID, videoID uuid.UUID, parentID *uuid.UUID, content string) (*domain.Comment, error) {
	video, err := s.videos.GetByID(ctx, videoID)
	if err != nil {
		return nil, err
	}

	var parent *domain.Comment
	if parentID != nil {
		parent, err = s.repo.GetCommentByID(ctx, *parentID)
		if err != nil {
			return nil, err
		}
		if parent.DeletedAt != nil {
			return nil, domain.ErrCommentNotFound
		}
		if parent.VideoID != videoID {
			return nil, fmt.Errorf("%w: parent comment belongs to a different video", domain.ErrInvalidInput)
		}
	}

	now := time.Now()
	comment := &domain.Comment{
		ID:        uuid.New(),
		VideoID:   videoID,
		UserID:    userID,
		ParentID:  parentID,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := comment.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return nil, err
	}

	s.notifyForComment(ctx, comment, parent, video)

	// Re-read for the username/avatar join. The comment is already durable, so
	// a failed re-read must not fail the request.
	created, err := s.repo.GetCommentByID(ctx, comment.ID)
	if err != nil {
		s.log.Error(ctx, "failed to reload created comment", err, map[string]interface{}{
			"comment_id": comment.ID,
		})
		return comment, nil
	}
	return created, nil
}

func (s *SocialService) UpdateComment(ctx context.Context, userID, commentID uuid.UUID, content string) (*domain.Comment, error) {
	comment, err := s.repo.GetCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if comment.DeletedAt != nil {
		return nil, domain.ErrCommentNotFound
	}
	if comment.UserID != userID {
		return nil, domain.ErrForbidden
	}
	if len(content) == 0 || len(content) > 10000 {
		return nil, domain.ErrInvalidInput
	}

	if err := s.repo.UpdateCommentContent(ctx, commentID, content); err != nil {
		return nil, err
	}
	return s.repo.GetCommentByID(ctx, commentID)
}

// DeleteComment soft-deletes. Allowed for the author, the owner of the video
// the comment sits on, and anyone holding moderate_content (canModerate — the
// handler resolves the permission because the service has no principal).
func (s *SocialService) DeleteComment(ctx context.Context, actorID uuid.UUID, canModerate bool, commentID uuid.UUID) error {
	comment, err := s.repo.GetCommentByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment.DeletedAt != nil {
		return domain.ErrCommentNotFound
	}

	allowed := canModerate || comment.UserID == actorID
	if !allowed {
		video, err := s.videos.GetByID(ctx, comment.VideoID)
		if err != nil && !errors.Is(err, domain.ErrVideoNotFound) {
			return err
		}
		allowed = err == nil && video.IsOwnedBy(actorID)
	}
	if !allowed {
		return domain.ErrForbidden
	}

	return s.repo.SoftDeleteComment(ctx, commentID)
}

// Subscribe is idempotent: subscribing twice succeeds and does not notify the
// creator again. The self-subscription check runs here so the caller gets a
// 400 instead of the CHECK constraint surfacing as a 500.
func (s *SocialService) Subscribe(ctx context.Context, subscriberID, creatorID uuid.UUID) error {
	if subscriberID == creatorID {
		return domain.ErrSelfSubscription
	}
	creator, err := s.users.GetByID(ctx, creatorID)
	if err != nil {
		return err
	}

	sub := &domain.Subscription{
		ID:            uuid.New(),
		SubscriberID:  subscriberID,
		CreatorID:     creatorID,
		NotifyUploads: true,
		CreatedAt:     time.Now(),
	}
	if err := sub.Validate(); err != nil {
		return err
	}

	inserted, err := s.repo.CreateSubscription(ctx, sub)
	if err != nil {
		return err
	}

	if inserted {
		s.notify(ctx, &domain.Notification{
			UserID:  creator.ID,
			Type:    domain.NotificationSubscriber,
			Title:   "You have a new subscriber",
			ActorID: &subscriberID,
		})
	}
	return nil
}

func (s *SocialService) Unsubscribe(ctx context.Context, subscriberID, creatorID uuid.UUID) error {
	return s.repo.DeleteSubscription(ctx, subscriberID, creatorID)
}

func (s *SocialService) ListSubscribers(ctx context.Context, creatorID uuid.UUID, page repository.Page) ([]*domain.SubscriptionEntry, int, error) {
	if _, err := s.users.GetByID(ctx, creatorID); err != nil {
		return nil, 0, err
	}

	subscribers, err := s.repo.ListSubscribers(ctx, creatorID, page)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountSubscribers(ctx, creatorID)
	if err != nil {
		return nil, 0, err
	}
	return subscribers, total, nil
}

func (s *SocialService) ListSubscriptions(ctx context.Context, subscriberID uuid.UUID, page repository.Page) ([]*domain.SubscriptionEntry, int, error) {
	subscriptions, err := s.repo.ListSubscriptions(ctx, subscriberID, page)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountSubscriptions(ctx, subscriberID)
	if err != nil {
		return nil, 0, err
	}
	return subscriptions, total, nil
}

func (s *SocialService) CreatePlaylist(ctx context.Context, userID uuid.UUID, title, description, visibility string) (*domain.Playlist, error) {
	if visibility == "" {
		visibility = "public"
	}

	now := time.Now()
	playlist := &domain.Playlist{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       title,
		Description: description,
		Visibility:  visibility,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := playlist.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.CreatePlaylist(ctx, playlist); err != nil {
		return nil, err
	}
	return playlist, nil
}

// GetPlaylist enforces visibility: a private playlist is a 404 for anyone but
// its owner, never a 403, so its existence is not leaked. Unlisted playlists
// are readable by anyone holding the link.
func (s *SocialService) GetPlaylist(ctx context.Context, viewerID *uuid.UUID, id uuid.UUID) (*domain.Playlist, error) {
	playlist, err := s.repo.GetPlaylistByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if playlist.Visibility == "private" && (viewerID == nil || *viewerID != playlist.UserID) {
		return nil, domain.ErrPlaylistNotFound
	}
	return playlist, nil
}

// ownedPlaylist fetches a playlist for a mutation. A private playlist owned by
// someone else reads as not-found (no existence leak); a visible one reads as
// forbidden.
func (s *SocialService) ownedPlaylist(ctx context.Context, userID, id uuid.UUID) (*domain.Playlist, error) {
	playlist, err := s.repo.GetPlaylistByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if playlist.UserID != userID {
		if playlist.Visibility == "private" {
			return nil, domain.ErrPlaylistNotFound
		}
		return nil, domain.ErrForbidden
	}
	return playlist, nil
}

func (s *SocialService) UpdatePlaylist(ctx context.Context, userID, id uuid.UUID, title, description, visibility *string) (*domain.Playlist, error) {
	playlist, err := s.ownedPlaylist(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if title != nil {
		playlist.Title = *title
	}
	if description != nil {
		playlist.Description = *description
	}
	if visibility != nil {
		playlist.Visibility = *visibility
	}
	if err := playlist.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.UpdatePlaylist(ctx, playlist); err != nil {
		return nil, err
	}
	// Re-read so the response carries the trigger-maintained updated_at.
	return s.repo.GetPlaylistByID(ctx, id)
}

func (s *SocialService) DeletePlaylist(ctx context.Context, userID, id uuid.UUID) error {
	if _, err := s.ownedPlaylist(ctx, userID, id); err != nil {
		return err
	}
	return s.repo.DeletePlaylist(ctx, id)
}

func (s *SocialService) ListMyPlaylists(ctx context.Context, userID uuid.UUID, page repository.Page) ([]*domain.Playlist, int, error) {
	playlists, err := s.repo.ListPlaylistsByUser(ctx, userID, page)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountPlaylistsByUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return playlists, total, nil
}

func (s *SocialService) AddPlaylistVideo(ctx context.Context, userID, playlistID, videoID uuid.UUID) (*domain.PlaylistVideo, error) {
	if _, err := s.ownedPlaylist(ctx, userID, playlistID); err != nil {
		return nil, err
	}
	if _, err := s.videos.GetByID(ctx, videoID); err != nil {
		return nil, err
	}
	return s.repo.AddPlaylistVideo(ctx, playlistID, videoID)
}

func (s *SocialService) RemovePlaylistVideo(ctx context.Context, userID, playlistID, videoID uuid.UUID) error {
	if _, err := s.ownedPlaylist(ctx, userID, playlistID); err != nil {
		return err
	}
	return s.repo.RemovePlaylistVideo(ctx, playlistID, videoID)
}

func (s *SocialService) ListPlaylistVideos(ctx context.Context, viewerID *uuid.UUID, playlistID uuid.UUID, page repository.Page) ([]*domain.PlaylistItem, int, error) {
	if _, err := s.GetPlaylist(ctx, viewerID, playlistID); err != nil {
		return nil, 0, err
	}

	items, err := s.repo.ListPlaylistVideos(ctx, playlistID, page)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountPlaylistVideos(ctx, playlistID)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *SocialService) AddWatchLater(ctx context.Context, userID, videoID uuid.UUID) error {
	if _, err := s.videos.GetByID(ctx, videoID); err != nil {
		return err
	}

	entry := &domain.WatchLater{
		ID:      uuid.New(),
		UserID:  userID,
		VideoID: videoID,
		AddedAt: time.Now(),
	}
	if err := entry.Validate(); err != nil {
		return err
	}
	return s.repo.UpsertWatchLater(ctx, entry)
}

func (s *SocialService) RemoveWatchLater(ctx context.Context, userID, videoID uuid.UUID) error {
	return s.repo.DeleteWatchLater(ctx, userID, videoID)
}

func (s *SocialService) ListWatchLater(ctx context.Context, userID uuid.UUID, page repository.Page) ([]*domain.WatchLaterItem, int, error) {
	items, err := s.repo.ListWatchLater(ctx, userID, page)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountWatchLater(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *SocialService) ListNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool, page repository.Page) ([]*domain.Notification, int, error) {
	notifications, err := s.repo.ListNotifications(ctx, userID, unreadOnly, page)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountNotifications(ctx, userID, unreadOnly)
	if err != nil {
		return nil, 0, err
	}
	return notifications, total, nil
}

func (s *SocialService) UnreadNotificationCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountNotifications(ctx, userID, true)
}

func (s *SocialService) MarkNotificationRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	return s.repo.MarkNotificationRead(ctx, notificationID, userID)
}

func (s *SocialService) MarkAllNotificationsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.MarkAllNotificationsRead(ctx, userID)
}

// notifyForComment fans a new comment out to whoever should hear about it: the
// video owner for a top-level comment, the parent's author for a reply. A user
// is never notified about their own action, and legacy ownerless videos have
// nobody to notify.
func (s *SocialService) notifyForComment(ctx context.Context, comment *domain.Comment, parent *domain.Comment, video *domain.Video) {
	if parent != nil {
		if parent.UserID == comment.UserID {
			return
		}
		s.notify(ctx, &domain.Notification{
			UserID:    parent.UserID,
			Type:      domain.NotificationReply,
			Title:     "New reply to your comment",
			Message:   truncateForNotification(comment.Content),
			ActorID:   &comment.UserID,
			VideoID:   &comment.VideoID,
			CommentID: &comment.ID,
		})
		return
	}

	if video.UserID == nil || *video.UserID == comment.UserID {
		return
	}
	s.notify(ctx, &domain.Notification{
		UserID:    *video.UserID,
		Type:      domain.NotificationComment,
		Title:     "New comment on your video",
		Message:   truncateForNotification(comment.Content),
		ActorID:   &comment.UserID,
		VideoID:   &comment.VideoID,
		CommentID: &comment.ID,
	})
}

// notify writes a notification on a best-effort basis: losing one is
// tolerable, failing the comment/subscription that produced it is not.
func (s *SocialService) notify(ctx context.Context, n *domain.Notification) {
	n.ID = uuid.New()
	n.CreatedAt = time.Now()

	if err := n.Validate(); err != nil {
		s.log.Error(ctx, "invalid notification", err, map[string]interface{}{
			"notification_type": n.Type,
			"recipient_id":      n.UserID,
		})
		return
	}
	if err := s.repo.CreateNotification(ctx, n); err != nil {
		s.log.Error(ctx, "failed to create notification", err, map[string]interface{}{
			"notification_type": n.Type,
			"recipient_id":      n.UserID,
		})
	}
}

// truncateForNotification bounds a comment excerpt carried in a notification
// message, cutting on a rune boundary so multi-byte text is never split.
func truncateForNotification(content string) string {
	const maxRunes = 200
	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}
	return string(runes[:maxRunes-1]) + "…"
}
