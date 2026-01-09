package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}

type SessionService struct {
	redisClient *redis.Client
}

func NewSessionService(redisClient *redis.Client) *SessionService {
	return &SessionService{
		redisClient: redisClient,
	}
}

func (s *SessionService) CreateSession(ctx context.Context, userID, username, role, ipAddress, userAgent string, rememberMe bool) (string, error) {
	sessionID := uuid.New().String()
	
	duration := 7 * 24 * time.Hour
	if rememberMe {
		duration = 30 * 24 * time.Hour
	}

	session := Session{
		ID:        sessionID,
		UserID:    userID,
		Username:  username,
		Role:      role,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	sessionData, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session: %w", err)
	}

	sessionKey := fmt.Sprintf("session:%s", sessionID)
	err = s.redisClient.Set(ctx, sessionKey, sessionData, duration).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store session: %w", err)
	}

	userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)
	err = s.redisClient.SAdd(ctx, userSessionsKey, sessionID).Err()
	if err != nil {
		return "", fmt.Errorf("failed to add session to user sessions: %w", err)
	}

	err = s.redisClient.Expire(ctx, userSessionsKey, duration).Err()
	if err != nil {
		return "", fmt.Errorf("failed to set expiry on user sessions: %w", err)
	}

	return sessionID, nil
}

func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	sessionKey := fmt.Sprintf("session:%s", sessionID)
	sessionData, err := s.redisClient.Get(ctx, sessionKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session Session
	err = json.Unmarshal([]byte(sessionData), &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.DeleteSession(ctx, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

func (s *SessionService) RefreshSession(ctx context.Context, sessionID string) error {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	duration := 7 * 24 * time.Hour
	session.ExpiresAt = time.Now().Add(duration)

	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	sessionKey := fmt.Sprintf("session:%s", sessionID)
	err = s.redisClient.Set(ctx, sessionKey, sessionData, duration).Err()
	if err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}

	return nil
}

func (s *SessionService) DeleteSession(ctx context.Context, sessionID string) error {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil
	}

	sessionKey := fmt.Sprintf("session:%s", sessionID)
	err = s.redisClient.Del(ctx, sessionKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	userSessionsKey := fmt.Sprintf("user_sessions:%s", session.UserID)
	err = s.redisClient.SRem(ctx, userSessionsKey, sessionID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove session from user sessions: %w", err)
	}

	return nil
}

func (s *SessionService) DeleteAllUserSessions(ctx context.Context, userID string) error {
	userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)
	sessionIDs, err := s.redisClient.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	for _, sessionID := range sessionIDs {
		sessionKey := fmt.Sprintf("session:%s", sessionID)
		err = s.redisClient.Del(ctx, sessionKey).Err()
		if err != nil {
			return fmt.Errorf("failed to delete session %s: %w", sessionID, err)
		}
	}

	err = s.redisClient.Del(ctx, userSessionsKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete user sessions set: %w", err)
	}

	return nil
}

func (s *SessionService) GetUserSessions(ctx context.Context, userID string) ([]Session, error) {
	userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)
	sessionIDs, err := s.redisClient.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	sessions := make([]Session, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		session, err := s.GetSession(ctx, sessionID)
		if err != nil {
			continue
		}
		sessions = append(sessions, *session)
	}

	return sessions, nil
}

func (s *SessionService) ValidateSession(ctx context.Context, sessionID, ipAddress, userAgent string) (*Session, error) {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session.IPAddress != ipAddress {
		return nil, fmt.Errorf("IP address mismatch")
	}

	if session.UserAgent != userAgent {
		return nil, fmt.Errorf("user agent mismatch")
	}

	return session, nil
}
