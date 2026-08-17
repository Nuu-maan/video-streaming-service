package domain

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID            uuid.UUID `json:"id"`
	SubscriberID  uuid.UUID `json:"subscriber_id"`
	CreatorID     uuid.UUID `json:"creator_id"`
	NotifyUploads bool      `json:"notify_uploads"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *Subscription) Validate() error {
	if s.SubscriberID == uuid.Nil {
		return ErrInvalidInput
	}
	if s.CreatorID == uuid.Nil {
		return ErrInvalidInput
	}
	if s.SubscriberID == s.CreatorID {
		return ErrInvalidInput
	}
	return nil
}

type Like struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	VideoID   uuid.UUID `json:"video_id"`
	IsLike    bool      `json:"is_like"`
	CreatedAt time.Time `json:"created_at"`
}

func (l *Like) Validate() error {
	if l.UserID == uuid.Nil {
		return ErrInvalidInput
	}
	if l.VideoID == uuid.Nil {
		return ErrInvalidInput
	}
	return nil
}

type Comment struct {
	ID         uuid.UUID  `json:"id"`
	VideoID    uuid.UUID  `json:"video_id"`
	UserID     uuid.UUID  `json:"user_id"`
	ParentID   *uuid.UUID `json:"parent_id,omitempty"`
	Content    string     `json:"content"`
	LikeCount  int64      `json:"like_count"`
	ReplyCount int64      `json:"reply_count"`
	Pinned     bool       `json:"pinned"`
	EditedAt   *time.Time `json:"edited_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`

	// VideoTimestamp anchors the comment to a moment in the video, in seconds
	// from the start. Nil means the comment is about the video as a whole —
	// which is not the same as second zero, so this is a pointer rather than an
	// int defaulting to 0. A reply may carry one too: a thread can legitimately
	// move on to a different moment.
	VideoTimestamp *int `json:"video_timestamp,omitempty"`

	Username  string `json:"username,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

func (c *Comment) Validate() error {
	if c.UserID == uuid.Nil {
		return ErrInvalidInput
	}
	if c.VideoID == uuid.Nil {
		return ErrInvalidInput
	}
	if len(c.Content) == 0 {
		return ErrInvalidInput
	}
	if len(c.Content) > 10000 {
		return ErrInvalidInput
	}
	// The upper bound is the video's duration, which this type does not know.
	// The service checks it where the video is already loaded.
	if c.VideoTimestamp != nil && *c.VideoTimestamp < 0 {
		return ErrInvalidInput
	}
	return nil
}

// CommentSort names the orderings the comment listing supports.
type CommentSort string

const (
	// CommentSortNewest is the conversational order: most recent first.
	CommentSortNewest CommentSort = "newest"
	// CommentSortTimestamp is the review order: earliest moment in the video
	// first, with unanchored comments after every anchored one.
	CommentSortTimestamp CommentSort = "timestamp"
)

func (s CommentSort) IsValid() bool {
	switch s {
	case CommentSortNewest, CommentSortTimestamp:
		return true
	default:
		return false
	}
}

type Playlist struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Visibility  string    `json:"visibility"`
	VideoCount  int64     `json:"video_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (p *Playlist) Validate() error {
	if p.UserID == uuid.Nil {
		return ErrInvalidInput
	}
	if len(p.Title) == 0 || len(p.Title) > 255 {
		return ErrInvalidInput
	}
	validVisibility := map[string]bool{
		"public":   true,
		"private":  true,
		"unlisted": true,
	}
	if !validVisibility[p.Visibility] {
		return ErrInvalidInput
	}
	return nil
}

type PlaylistVideo struct {
	ID         uuid.UUID `json:"id"`
	PlaylistID uuid.UUID `json:"playlist_id"`
	VideoID    uuid.UUID `json:"video_id"`
	Position   int32     `json:"position"`
	AddedAt    time.Time `json:"added_at"`
}

func (pv *PlaylistVideo) Validate() error {
	if pv.PlaylistID == uuid.Nil {
		return ErrInvalidInput
	}
	if pv.VideoID == uuid.Nil {
		return ErrInvalidInput
	}
	if pv.Position < 0 {
		return ErrInvalidInput
	}
	return nil
}

type WatchHistory struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	VideoID       uuid.UUID `json:"video_id"`
	WatchedAt     time.Time `json:"watched_at"`
	WatchDuration int32     `json:"watch_duration"`
	Completed     bool      `json:"completed"`
	LastPosition  int32     `json:"last_position"`
}

func (wh *WatchHistory) Validate() error {
	if wh.UserID == uuid.Nil {
		return ErrInvalidInput
	}
	if wh.VideoID == uuid.Nil {
		return ErrInvalidInput
	}
	if wh.WatchDuration < 0 {
		return ErrInvalidInput
	}
	if wh.LastPosition < 0 {
		return ErrInvalidInput
	}
	return nil
}

type WatchLater struct {
	ID      uuid.UUID `json:"id"`
	UserID  uuid.UUID `json:"user_id"`
	VideoID uuid.UUID `json:"video_id"`
	AddedAt time.Time `json:"added_at"`
}

func (wl *WatchLater) Validate() error {
	if wl.UserID == uuid.Nil {
		return ErrInvalidInput
	}
	if wl.VideoID == uuid.Nil {
		return ErrInvalidInput
	}
	return nil
}

// SubscriptionEntry is one row of a subscriber or subscription listing: the
// counterpart user plus the relationship metadata. For a subscriber listing the
// user is the subscriber; for a subscription listing it is the creator.
type SubscriptionEntry struct {
	UserID          uuid.UUID `json:"user_id"`
	Username        string    `json:"username"`
	AvatarURL       string    `json:"avatar_url,omitempty"`
	SubscriberCount int64     `json:"subscriber_count"`
	NotifyUploads   bool      `json:"notify_uploads"`
	SubscribedAt    time.Time `json:"subscribed_at"`
}

// PlaylistItem is a video inside a playlist together with its ordering
// metadata. Positions may have gaps: removals never renumber.
type PlaylistItem struct {
	Position int32     `json:"position"`
	AddedAt  time.Time `json:"added_at"`
	Video    *Video    `json:"video"`
}

// WatchLaterItem is a saved video together with when it was saved.
type WatchLaterItem struct {
	AddedAt time.Time `json:"added_at"`
	Video   *Video    `json:"video"`
}

type NotificationType string

const (
	NotificationNewVideo   NotificationType = "new_video"
	NotificationComment    NotificationType = "comment"
	NotificationReply      NotificationType = "reply"
	NotificationLike       NotificationType = "like"
	NotificationSubscriber NotificationType = "subscriber"
	NotificationMention    NotificationType = "mention"
)

type Notification struct {
	ID        uuid.UUID        `json:"id"`
	UserID    uuid.UUID        `json:"user_id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	ActionURL *string          `json:"action_url,omitempty"`
	ActorID   *uuid.UUID       `json:"actor_id,omitempty"`
	VideoID   *uuid.UUID       `json:"video_id,omitempty"`
	CommentID *uuid.UUID       `json:"comment_id,omitempty"`
	Read      bool             `json:"read"`
	CreatedAt time.Time        `json:"created_at"`
}

func (n *Notification) Validate() error {
	if n.UserID == uuid.Nil {
		return ErrInvalidInput
	}
	if len(n.Title) == 0 || len(n.Title) > 255 {
		return ErrInvalidInput
	}
	validTypes := map[NotificationType]bool{
		NotificationNewVideo:   true,
		NotificationComment:    true,
		NotificationReply:      true,
		NotificationLike:       true,
		NotificationSubscriber: true,
		NotificationMention:    true,
	}
	if !validTypes[n.Type] {
		return ErrInvalidInput
	}
	return nil
}
