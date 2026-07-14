package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
	"github.com/Nuu-maan/video-streaming-service/pkg/validator"
)

type SocialHandler struct {
	social *service.SocialService
	log    *logger.Logger
}

func NewSocialHandler(social *service.SocialService, log *logger.Logger) *SocialHandler {
	return &SocialHandler{social: social, log: log}
}

// requirePrincipal resolves the caller, writing the 401 itself so call sites
// stay a two-line guard.
func (h *SocialHandler) requirePrincipal(c *gin.Context) (appctx.Principal, bool) {
	principal, ok := appctx.PrincipalFrom(c.Request.Context())
	if !ok {
		response.Unauthorized(c, "Authentication required")
	}
	return principal, ok
}

// pathUUID parses a path parameter, writing the 400 itself on failure.
func (h *SocialHandler) pathUUID(c *gin.Context, param, label string) (uuid.UUID, bool) {
	id, err := validator.ValidateUUID(c.Param(param))
	if err != nil {
		response.ValidationError(c, "Invalid "+label)
		return uuid.Nil, false
	}
	return id, true
}

// viewerID is the optional caller for endpoints behind OptionalAuth, where
// authentication widens what is visible but is not required.
func viewerID(c *gin.Context) *uuid.UUID {
	if principal, ok := appctx.PrincipalFrom(c.Request.Context()); ok {
		return &principal.UserID
	}
	return nil
}

type setLikeRequest struct {
	// A pointer so an absent field is distinguishable from an explicit false:
	// false is a valid value here (a dislike).
	IsLike *bool `json:"is_like" binding:"required"`
}

// SetLike upserts the caller's rating of a video (like or dislike).
func (h *SocialHandler) SetLike(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	videoID, ok := h.pathUUID(c, "id", "video ID")
	if !ok {
		return
	}

	var req setLikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "is_like is required and must be a boolean")
		return
	}

	like, err := h.social.SetLike(ctx, principal.UserID, videoID, *req.IsLike)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrVideoNotFound):
			response.NotFound(c, "Video not found")
		case errors.Is(err, domain.ErrInvalidInput):
			response.ValidationError(c, err.Error())
		default:
			h.log.Error(ctx, "failed to set like", err, map[string]interface{}{"video_id": videoID})
			response.InternalError(c, "Failed to save rating")
		}
		return
	}

	response.Success(c, http.StatusOK, like)
}

// RemoveLike clears the caller's rating of a video.
func (h *SocialHandler) RemoveLike(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	videoID, ok := h.pathUUID(c, "id", "video ID")
	if !ok {
		return
	}

	if err := h.social.RemoveLike(ctx, principal.UserID, videoID); err != nil {
		if errors.Is(err, domain.ErrLikeNotFound) {
			response.NotFound(c, "You have not rated this video")
			return
		}
		h.log.Error(ctx, "failed to remove like", err, map[string]interface{}{"video_id": videoID})
		response.InternalError(c, "Failed to remove rating")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Rating removed", "video_id": videoID})
}

// GetLike returns the caller's current rating of a video.
func (h *SocialHandler) GetLike(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	videoID, ok := h.pathUUID(c, "id", "video ID")
	if !ok {
		return
	}

	like, err := h.social.GetLike(ctx, principal.UserID, videoID)
	if err != nil {
		if errors.Is(err, domain.ErrLikeNotFound) {
			response.NotFound(c, "You have not rated this video")
			return
		}
		h.log.Error(ctx, "failed to get like", err, map[string]interface{}{"video_id": videoID})
		response.InternalError(c, "Failed to retrieve rating")
		return
	}

	response.Success(c, http.StatusOK, like)
}

// ListComments returns a page of a video's top-level comments, pinned first.
func (h *SocialHandler) ListComments(c *gin.Context) {
	ctx := c.Request.Context()

	videoID, ok := h.pathUUID(c, "id", "video ID")
	if !ok {
		return
	}
	page := parsePage(c)

	comments, total, err := h.social.ListComments(ctx, videoID, page)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return
		}
		h.log.Error(ctx, "failed to list comments", err, map[string]interface{}{"video_id": videoID})
		response.InternalError(c, "Failed to retrieve comments")
		return
	}

	response.SuccessWithList(c, comments, paginationMeta(total, page))
}

// ListReplies returns a page of a comment's replies, oldest first.
func (h *SocialHandler) ListReplies(c *gin.Context) {
	ctx := c.Request.Context()

	commentID, ok := h.pathUUID(c, "id", "comment ID")
	if !ok {
		return
	}
	page := parsePage(c)

	replies, total, err := h.social.ListReplies(ctx, commentID, page)
	if err != nil {
		if errors.Is(err, domain.ErrCommentNotFound) {
			response.NotFound(c, "Comment not found")
			return
		}
		h.log.Error(ctx, "failed to list replies", err, map[string]interface{}{"comment_id": commentID})
		response.InternalError(c, "Failed to retrieve replies")
		return
	}

	response.SuccessWithList(c, replies, paginationMeta(total, page))
}

type createCommentRequest struct {
	Content  string `json:"content" binding:"required"`
	ParentID string `json:"parent_id"`
}

// CreateComment posts a comment on a video, or a reply when parent_id is set.
func (h *SocialHandler) CreateComment(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	videoID, ok := h.pathUUID(c, "id", "video ID")
	if !ok {
		return
	}

	var req createCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "content is required")
		return
	}

	parentID, bad := optionalUUID(c, req.ParentID, "parent_id")
	if bad {
		return
	}

	comment, err := h.social.CreateComment(ctx, principal.UserID, videoID, parentID, strings.TrimSpace(req.Content))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrVideoNotFound):
			response.NotFound(c, "Video not found")
		case errors.Is(err, domain.ErrCommentNotFound):
			response.NotFound(c, "Parent comment not found")
		case errors.Is(err, domain.ErrInvalidInput):
			response.ValidationError(c, "content must be 1-10000 characters and any parent comment must be on the same video")
		default:
			h.log.Error(ctx, "failed to create comment", err, map[string]interface{}{"video_id": videoID})
			response.InternalError(c, "Failed to post comment")
		}
		return
	}

	response.Success(c, http.StatusCreated, comment)
}

type updateCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

// UpdateComment edits a comment's content. Author only.
func (h *SocialHandler) UpdateComment(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	commentID, ok := h.pathUUID(c, "id", "comment ID")
	if !ok {
		return
	}

	var req updateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "content is required")
		return
	}

	comment, err := h.social.UpdateComment(ctx, principal.UserID, commentID, strings.TrimSpace(req.Content))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCommentNotFound):
			response.NotFound(c, "Comment not found")
		case errors.Is(err, domain.ErrForbidden):
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only the author can edit a comment")
		case errors.Is(err, domain.ErrInvalidInput):
			response.ValidationError(c, "content must be 1-10000 characters")
		default:
			h.log.Error(ctx, "failed to update comment", err, map[string]interface{}{"comment_id": commentID})
			response.InternalError(c, "Failed to update comment")
		}
		return
	}

	response.Success(c, http.StatusOK, comment)
}

// DeleteComment soft-deletes a comment. Allowed for the author, the owner of
// the video, or a caller holding moderate_content.
func (h *SocialHandler) DeleteComment(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	commentID, ok := h.pathUUID(c, "id", "comment ID")
	if !ok {
		return
	}

	canModerate := principal.HasPermission(domain.PermissionModerateContent)
	if err := h.social.DeleteComment(ctx, principal.UserID, canModerate, commentID); err != nil {
		switch {
		case errors.Is(err, domain.ErrCommentNotFound):
			response.NotFound(c, "Comment not found")
		case errors.Is(err, domain.ErrForbidden):
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "You cannot delete this comment")
		default:
			h.log.Error(ctx, "failed to delete comment", err, map[string]interface{}{"comment_id": commentID})
			response.InternalError(c, "Failed to delete comment")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Comment deleted", "comment_id": commentID})
}

// Subscribe subscribes the caller to a creator. Idempotent.
func (h *SocialHandler) Subscribe(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	creatorID, ok := h.pathUUID(c, "id", "user ID")
	if !ok {
		return
	}

	if err := h.social.Subscribe(ctx, principal.UserID, creatorID); err != nil {
		switch {
		case errors.Is(err, domain.ErrSelfSubscription):
			response.ValidationError(c, "You cannot subscribe to yourself")
		case errors.Is(err, domain.ErrUserNotFound):
			response.NotFound(c, "User not found")
		case errors.Is(err, domain.ErrInvalidInput):
			response.ValidationError(c, err.Error())
		default:
			h.log.Error(ctx, "failed to subscribe", err, map[string]interface{}{"creator_id": creatorID})
			response.InternalError(c, "Failed to subscribe")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Subscribed", "creator_id": creatorID})
}

// Unsubscribe removes the caller's subscription to a creator.
func (h *SocialHandler) Unsubscribe(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	creatorID, ok := h.pathUUID(c, "id", "user ID")
	if !ok {
		return
	}

	if err := h.social.Unsubscribe(ctx, principal.UserID, creatorID); err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			response.NotFound(c, "You are not subscribed to this user")
			return
		}
		h.log.Error(ctx, "failed to unsubscribe", err, map[string]interface{}{"creator_id": creatorID})
		response.InternalError(c, "Failed to unsubscribe")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Unsubscribed", "creator_id": creatorID})
}

// ListSubscribers returns a page of a creator's subscribers.
func (h *SocialHandler) ListSubscribers(c *gin.Context) {
	ctx := c.Request.Context()

	creatorID, ok := h.pathUUID(c, "id", "user ID")
	if !ok {
		return
	}
	page := parsePage(c)

	subscribers, total, err := h.social.ListSubscribers(ctx, creatorID, page)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.NotFound(c, "User not found")
			return
		}
		h.log.Error(ctx, "failed to list subscribers", err, map[string]interface{}{"creator_id": creatorID})
		response.InternalError(c, "Failed to retrieve subscribers")
		return
	}

	response.SuccessWithList(c, subscribers, paginationMeta(total, page))
}

// ListMySubscriptions returns the creators the caller follows.
func (h *SocialHandler) ListMySubscriptions(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	page := parsePage(c)

	subscriptions, total, err := h.social.ListSubscriptions(ctx, principal.UserID, page)
	if err != nil {
		h.log.Error(ctx, "failed to list subscriptions", err, nil)
		response.InternalError(c, "Failed to retrieve subscriptions")
		return
	}

	response.SuccessWithList(c, subscriptions, paginationMeta(total, page))
}

type createPlaylistRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

// CreatePlaylist creates a playlist owned by the caller.
func (h *SocialHandler) CreatePlaylist(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}

	var req createPlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "title is required")
		return
	}

	playlist, err := h.social.CreatePlaylist(ctx, principal.UserID,
		strings.TrimSpace(req.Title), strings.TrimSpace(req.Description), strings.TrimSpace(req.Visibility))
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			response.ValidationError(c, "title must be 1-255 characters and visibility one of: public, private, unlisted")
			return
		}
		h.log.Error(ctx, "failed to create playlist", err, nil)
		response.InternalError(c, "Failed to create playlist")
		return
	}

	response.Success(c, http.StatusCreated, playlist)
}

// GetPlaylist returns a playlist. Private playlists resolve only for their
// owner; everyone else gets a 404.
func (h *SocialHandler) GetPlaylist(c *gin.Context) {
	ctx := c.Request.Context()

	playlistID, ok := h.pathUUID(c, "id", "playlist ID")
	if !ok {
		return
	}

	playlist, err := h.social.GetPlaylist(ctx, viewerID(c), playlistID)
	if err != nil {
		if errors.Is(err, domain.ErrPlaylistNotFound) {
			response.NotFound(c, "Playlist not found")
			return
		}
		h.log.Error(ctx, "failed to get playlist", err, map[string]interface{}{"playlist_id": playlistID})
		response.InternalError(c, "Failed to retrieve playlist")
		return
	}

	response.Success(c, http.StatusOK, playlist)
}

type updatePlaylistRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Visibility  *string `json:"visibility"`
}

// UpdatePlaylist edits a playlist's metadata. Owner only.
func (h *SocialHandler) UpdatePlaylist(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	playlistID, ok := h.pathUUID(c, "id", "playlist ID")
	if !ok {
		return
	}

	var req updatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "Invalid request body")
		return
	}
	if req.Title == nil && req.Description == nil && req.Visibility == nil {
		response.ValidationError(c, "Provide at least one of: title, description, visibility")
		return
	}

	playlist, err := h.social.UpdatePlaylist(ctx, principal.UserID, playlistID, req.Title, req.Description, req.Visibility)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPlaylistNotFound):
			response.NotFound(c, "Playlist not found")
		case errors.Is(err, domain.ErrForbidden):
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only the owner can edit a playlist")
		case errors.Is(err, domain.ErrInvalidInput):
			response.ValidationError(c, "title must be 1-255 characters and visibility one of: public, private, unlisted")
		default:
			h.log.Error(ctx, "failed to update playlist", err, map[string]interface{}{"playlist_id": playlistID})
			response.InternalError(c, "Failed to update playlist")
		}
		return
	}

	response.Success(c, http.StatusOK, playlist)
}

// DeletePlaylist deletes a playlist. Owner only.
func (h *SocialHandler) DeletePlaylist(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	playlistID, ok := h.pathUUID(c, "id", "playlist ID")
	if !ok {
		return
	}

	if err := h.social.DeletePlaylist(ctx, principal.UserID, playlistID); err != nil {
		switch {
		case errors.Is(err, domain.ErrPlaylistNotFound):
			response.NotFound(c, "Playlist not found")
		case errors.Is(err, domain.ErrForbidden):
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only the owner can delete a playlist")
		default:
			h.log.Error(ctx, "failed to delete playlist", err, map[string]interface{}{"playlist_id": playlistID})
			response.InternalError(c, "Failed to delete playlist")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Playlist deleted", "playlist_id": playlistID})
}

// ListMyPlaylists returns the caller's playlists, private ones included.
func (h *SocialHandler) ListMyPlaylists(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	page := parsePage(c)

	playlists, total, err := h.social.ListMyPlaylists(ctx, principal.UserID, page)
	if err != nil {
		h.log.Error(ctx, "failed to list playlists", err, nil)
		response.InternalError(c, "Failed to retrieve playlists")
		return
	}

	response.SuccessWithList(c, playlists, paginationMeta(total, page))
}

type addPlaylistVideoRequest struct {
	VideoID string `json:"video_id" binding:"required"`
}

// AddPlaylistVideo appends a video to the end of a playlist. Owner only.
func (h *SocialHandler) AddPlaylistVideo(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	playlistID, ok := h.pathUUID(c, "id", "playlist ID")
	if !ok {
		return
	}

	var req addPlaylistVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "video_id is required")
		return
	}
	videoID, err := validator.ValidateUUID(strings.TrimSpace(req.VideoID))
	if err != nil {
		response.ValidationError(c, "Invalid video_id")
		return
	}

	entry, err := h.social.AddPlaylistVideo(ctx, principal.UserID, playlistID, videoID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPlaylistNotFound):
			response.NotFound(c, "Playlist not found")
		case errors.Is(err, domain.ErrForbidden):
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only the owner can modify a playlist")
		case errors.Is(err, domain.ErrVideoNotFound):
			response.NotFound(c, "Video not found")
		case errors.Is(err, domain.ErrVideoAlreadyInPlaylist):
			response.Error(c, http.StatusConflict, "ALREADY_IN_PLAYLIST", "The video is already in this playlist")
		default:
			h.log.Error(ctx, "failed to add video to playlist", err, map[string]interface{}{
				"playlist_id": playlistID,
				"video_id":    videoID,
			})
			response.InternalError(c, "Failed to add video to playlist")
		}
		return
	}

	response.Success(c, http.StatusCreated, entry)
}

// RemovePlaylistVideo removes a video from a playlist. Owner only.
func (h *SocialHandler) RemovePlaylistVideo(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	playlistID, ok := h.pathUUID(c, "id", "playlist ID")
	if !ok {
		return
	}
	videoID, ok := h.pathUUID(c, "videoId", "video ID")
	if !ok {
		return
	}

	if err := h.social.RemovePlaylistVideo(ctx, principal.UserID, playlistID, videoID); err != nil {
		switch {
		case errors.Is(err, domain.ErrPlaylistNotFound):
			response.NotFound(c, "Playlist not found")
		case errors.Is(err, domain.ErrForbidden):
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only the owner can modify a playlist")
		case errors.Is(err, domain.ErrPlaylistVideoNotFound):
			response.NotFound(c, "The video is not in this playlist")
		default:
			h.log.Error(ctx, "failed to remove video from playlist", err, map[string]interface{}{
				"playlist_id": playlistID,
				"video_id":    videoID,
			})
			response.InternalError(c, "Failed to remove video from playlist")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Video removed from playlist", "video_id": videoID})
}

// ListPlaylistVideos returns a playlist's videos in position order. Visibility
// follows the playlist itself.
func (h *SocialHandler) ListPlaylistVideos(c *gin.Context) {
	ctx := c.Request.Context()

	playlistID, ok := h.pathUUID(c, "id", "playlist ID")
	if !ok {
		return
	}
	page := parsePage(c)

	items, total, err := h.social.ListPlaylistVideos(ctx, viewerID(c), playlistID, page)
	if err != nil {
		if errors.Is(err, domain.ErrPlaylistNotFound) {
			response.NotFound(c, "Playlist not found")
			return
		}
		h.log.Error(ctx, "failed to list playlist videos", err, map[string]interface{}{"playlist_id": playlistID})
		response.InternalError(c, "Failed to retrieve playlist videos")
		return
	}

	response.SuccessWithList(c, items, paginationMeta(total, page))
}

// AddWatchLater saves a video to the caller's watch-later list. Idempotent.
func (h *SocialHandler) AddWatchLater(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	videoID, ok := h.pathUUID(c, "id", "video ID")
	if !ok {
		return
	}

	if err := h.social.AddWatchLater(ctx, principal.UserID, videoID); err != nil {
		switch {
		case errors.Is(err, domain.ErrVideoNotFound):
			response.NotFound(c, "Video not found")
		case errors.Is(err, domain.ErrInvalidInput):
			response.ValidationError(c, err.Error())
		default:
			h.log.Error(ctx, "failed to add to watch later", err, map[string]interface{}{"video_id": videoID})
			response.InternalError(c, "Failed to save video")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Saved to watch later", "video_id": videoID})
}

// RemoveWatchLater removes a video from the caller's watch-later list.
func (h *SocialHandler) RemoveWatchLater(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	videoID, ok := h.pathUUID(c, "id", "video ID")
	if !ok {
		return
	}

	if err := h.social.RemoveWatchLater(ctx, principal.UserID, videoID); err != nil {
		if errors.Is(err, domain.ErrWatchLaterNotFound) {
			response.NotFound(c, "The video is not in your watch later list")
			return
		}
		h.log.Error(ctx, "failed to remove from watch later", err, map[string]interface{}{"video_id": videoID})
		response.InternalError(c, "Failed to remove video")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Removed from watch later", "video_id": videoID})
}

// ListWatchLater returns the caller's watch-later list, most recently saved
// first.
func (h *SocialHandler) ListWatchLater(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	page := parsePage(c)

	items, total, err := h.social.ListWatchLater(ctx, principal.UserID, page)
	if err != nil {
		h.log.Error(ctx, "failed to list watch later", err, nil)
		response.InternalError(c, "Failed to retrieve watch later list")
		return
	}

	response.SuccessWithList(c, items, paginationMeta(total, page))
}

// ListNotifications returns the caller's notifications, newest first.
// ?unread=true narrows to unread ones.
func (h *SocialHandler) ListNotifications(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	page := parsePage(c)
	unreadOnly := c.Query("unread") == "true"

	notifications, total, err := h.social.ListNotifications(ctx, principal.UserID, unreadOnly, page)
	if err != nil {
		h.log.Error(ctx, "failed to list notifications", err, nil)
		response.InternalError(c, "Failed to retrieve notifications")
		return
	}

	response.SuccessWithList(c, notifications, paginationMeta(total, page))
}

// UnreadNotificationCount returns how many unread notifications the caller
// has, for badge rendering without pulling a page.
func (h *SocialHandler) UnreadNotificationCount(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}

	count, err := h.social.UnreadNotificationCount(ctx, principal.UserID)
	if err != nil {
		h.log.Error(ctx, "failed to count unread notifications", err, nil)
		response.InternalError(c, "Failed to retrieve unread count")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"unread_count": count})
}

// MarkNotificationRead marks one of the caller's notifications as read.
func (h *SocialHandler) MarkNotificationRead(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}
	notificationID, ok := h.pathUUID(c, "id", "notification ID")
	if !ok {
		return
	}

	if err := h.social.MarkNotificationRead(ctx, principal.UserID, notificationID); err != nil {
		if errors.Is(err, domain.ErrNotificationNotFound) {
			response.NotFound(c, "Notification not found")
			return
		}
		h.log.Error(ctx, "failed to mark notification read", err, map[string]interface{}{
			"notification_id": notificationID,
		})
		response.InternalError(c, "Failed to mark notification as read")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Notification marked as read", "notification_id": notificationID})
}

// MarkAllNotificationsRead marks every unread notification of the caller as
// read and reports how many were affected.
func (h *SocialHandler) MarkAllNotificationsRead(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := h.requirePrincipal(c)
	if !ok {
		return
	}

	marked, err := h.social.MarkAllNotificationsRead(ctx, principal.UserID)
	if err != nil {
		h.log.Error(ctx, "failed to mark all notifications read", err, nil)
		response.InternalError(c, "Failed to mark notifications as read")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "All notifications marked as read", "marked": marked})
}
