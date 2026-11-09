package models

import (
	"strings"
	"time"

	"maizap/internal/apperr"
)

type Session struct {
	token     string
	userID    string
	startedAt time.Time
	lastSeen  time.Time
	expiresAt time.Time
}

func NewSession(token, userID string, startedAt time.Time, ttl time.Duration) (*Session, error) {
	token = strings.TrimSpace(token)
	userID = strings.TrimSpace(userID)
	if token == "" || userID == "" {
		return nil, apperr.New(apperr.Validation, "la sesión requiere token y usuario")
	}
	if ttl <= 0 {
		return nil, apperr.New(apperr.Validation, "la duración de la sesión debe ser positiva")
	}
	return &Session{
		token:     token,
		userID:    userID,
		startedAt: startedAt,
		lastSeen:  startedAt,
		expiresAt: startedAt.Add(ttl),
	}, nil
}

func (s *Session) Token() string        { return s.token }
func (s *Session) UserID() string       { return s.userID }
func (s *Session) StartedAt() time.Time { return s.startedAt }
func (s *Session) LastSeen() time.Time  { return s.lastSeen }
func (s *Session) ExpiresAt() time.Time { return s.expiresAt }

func (s *Session) IsExpired(now time.Time) bool {
	return !now.Before(s.expiresAt)
}

func (s *Session) Touch(now time.Time, ttl time.Duration) {
	s.lastSeen = now
	s.expiresAt = now.Add(ttl)
}
