package services

import (
	"strings"
	"time"

	"maizap/internal/apperr"
	"maizap/internal/models"
	"maizap/internal/storage"
)

type RegisterInput struct {
	Name  string
	Email string
	Site  string
	Area  string
}

type UserService struct {
	store *storage.JSONStore
	now   func() time.Time
}

func NewUserService(store *storage.JSONStore, now func() time.Time) *UserService {
	if now == nil {
		now = time.Now
	}
	return &UserService{store: store, now: now}
}

func (s *UserService) Register(in RegisterInput) (*models.User, error) {
	site, _ := models.ParseSite(in.Site)
	if err := models.ValidateUserData(in.Name, in.Email, site, in.Area); err != nil {
		return nil, err
	}
	if existing, err := s.ByEmail(in.Email); err == nil {
		return nil, apperr.New(apperr.Conflict, "ya existe un usuario registrado con el correo "+existing.Email())
	}
	user, err := models.NewUser(s.store.NextUserID(), in.Name, in.Email, site, in.Area, models.RoleUser, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.store.CreateUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) ByID(id string) (*models.User, error) {
	return s.store.UserByID(strings.TrimSpace(id))
}

func (s *UserService) ByEmail(email string) (*models.User, error) {
	normalized, err := models.NormalizeEmail(email)
	if err != nil {
		return nil, err
	}
	return s.store.UserByEmail(normalized)
}

func (s *UserService) List() ([]*models.User, error) {
	return s.store.ListUsers()
}

func (s *UserService) Deactivate(actor *models.User, targetID string) (*models.User, error) {
	if !actor.IsAdmin() {
		return nil, apperr.New(apperr.Forbidden, "solo un administrador puede desactivar usuarios")
	}
	target, err := s.ByID(targetID)
	if err != nil {
		return nil, err
	}
	if target.ID() == actor.ID() {
		return nil, apperr.New(apperr.Validation, "un administrador no puede desactivar su propia cuenta")
	}
	if err := target.Deactivate(); err != nil {
		return nil, err
	}
	if err := s.store.UpdateUser(target); err != nil {
		return nil, err
	}
	return target, nil
}

func (s *UserService) ResolveRecipients(emails []string) ([]*models.User, error) {
	var problems []string
	seen := make(map[string]bool)
	recipients := make([]*models.User, 0, len(emails))
	for _, raw := range emails {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		email, err := models.NormalizeEmail(raw)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		if seen[email] {
			continue
		}
		seen[email] = true
		user, err := s.store.UserByEmail(email)
		if err != nil {
			problems = append(problems, "el destinatario "+email+" no existe")
			continue
		}
		if !user.IsActive() {
			problems = append(problems, "el destinatario "+email+" está desactivado")
			continue
		}
		recipients = append(recipients, user)
	}
	if len(problems) > 0 {
		return nil, apperr.New(apperr.Validation, "hay destinatarios inválidos").WithDetails(problems...)
	}
	if len(recipients) == 0 {
		return nil, apperr.New(apperr.Validation, "debe indicar al menos un destinatario")
	}
	return recipients, nil
}
