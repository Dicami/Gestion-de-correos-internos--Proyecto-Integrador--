package services

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"maizap/internal/apperr"
	"maizap/internal/models"
)

type Authenticator interface {
	Authenticate(email string) (*models.Session, *models.User, error)
	Validate(token string) (*models.Session, error)
	Revoke(token string) error
}

type LocalAuthenticator struct {
	users    *UserService
	ttl      time.Duration
	now      func() time.Time
	mu       sync.Mutex
	sessions map[string]*models.Session
}

func NewLocalAuthenticator(users *UserService, ttl time.Duration, now func() time.Time) *LocalAuthenticator {
	if now == nil {
		now = time.Now
	}
	return &LocalAuthenticator{
		users:    users,
		ttl:      ttl,
		now:      now,
		sessions: make(map[string]*models.Session),
	}
}

func (a *LocalAuthenticator) Authenticate(email string) (*models.Session, *models.User, error) {
	user, err := a.users.ByEmail(email)
	if err != nil {
		return nil, nil, err
	}
	if !user.IsActive() {
		return nil, nil, apperr.New(apperr.Forbidden, "la cuenta "+user.Email()+" está desactivada; contacte al administrador")
	}
	token, err := newToken()
	if err != nil {
		return nil, nil, err
	}
	session, err := models.NewSession(token, user.ID(), a.now(), a.ttl)
	if err != nil {
		return nil, nil, err
	}
	a.mu.Lock()
	a.sessions[token] = session
	a.mu.Unlock()
	return session, user, nil
}

func (a *LocalAuthenticator) Validate(token string) (*models.Session, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, apperr.New(apperr.Unauthorized, "debe identificarse para realizar esta operación")
	}
	now := a.now()
	a.mu.Lock()
	defer a.mu.Unlock()
	session, ok := a.sessions[token]
	if !ok {
		return nil, apperr.New(apperr.Unauthorized, "la sesión no es válida o ya fue cerrada")
	}
	if session.IsExpired(now) {
		delete(a.sessions, token)
		return nil, apperr.New(apperr.Unauthorized, "la sesión expiró por inactividad; identifíquese de nuevo")
	}
	session.Touch(now, a.ttl)
	return session, nil
}

func (a *LocalAuthenticator) Revoke(token string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.sessions[token]; !ok {
		return apperr.New(apperr.Unauthorized, "la sesión no es válida o ya fue cerrada")
	}
	delete(a.sessions, token)
	return nil
}

func newToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", apperr.Wrap(apperr.Internal, "no se pudo generar el token de sesión", err)
	}
	return hex.EncodeToString(buf), nil
}
