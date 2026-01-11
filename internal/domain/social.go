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
	return nil
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
