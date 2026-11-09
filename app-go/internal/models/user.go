package models

import (
	"strings"
	"time"
	"unicode/utf8"

	"maizap/internal/apperr"
)

type Role string

const (
	RoleUser  Role = "usuario"
	RoleAdmin Role = "administrador"
)

type Site string

const (
	SiteA Site = "A"
	SiteB Site = "B"
)

const CorporateDomain = "maizap.com"

type User struct {
	id        string
	name      string
	email     string
	site      Site
	area      string
	role      Role
	active    bool
	createdAt time.Time
}

func NewUser(id, name, email string, site Site, area string, role Role, createdAt time.Time) (*User, error) {
	var problems []string
	id = strings.TrimSpace(id)
	if id == "" {
		problems = append(problems, "el identificador del usuario es obligatorio")
	}
	problems = append(problems, userDataProblems(name, email, site, area)...)
	if !role.Valid() {
		problems = append(problems, "el rol no es válido")
	}
	if len(problems) > 0 {
		return nil, apperr.New(apperr.Validation, "datos de usuario inválidos").WithDetails(problems...)
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	normalized, _ := NormalizeEmail(email)
	return &User{
		id:        id,
		name:      strings.TrimSpace(name),
		email:     normalized,
		site:      site,
		area:      strings.TrimSpace(area),
		role:      role,
		active:    true,
		createdAt: createdAt,
	}, nil
}

func ValidateUserData(name, email string, site Site, area string) error {
	problems := userDataProblems(name, email, site, area)
	if len(problems) > 0 {
		return apperr.New(apperr.Validation, "datos de usuario inválidos").WithDetails(problems...)
	}
	return nil
}

func userDataProblems(name, email string, site Site, area string) []string {
	var problems []string
	if n := utf8.RuneCountInString(strings.TrimSpace(name)); n < 3 || n > 80 {
		problems = append(problems, "el nombre debe tener entre 3 y 80 caracteres")
	}
	if _, err := NormalizeEmail(email); err != nil {
		problems = append(problems, err.Error())
	}
	if !site.Valid() {
		problems = append(problems, "la sede debe ser A o B")
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(area)); n < 2 || n > 60 {
		problems = append(problems, "el área debe tener entre 2 y 60 caracteres")
	}
	return problems
}

func NormalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return "", apperr.New(apperr.Validation, "el correo es obligatorio")
	}
	if !validCorporateEmail(normalized) {
		return "", apperr.New(apperr.Validation, "el correo "+normalized+" no es un correo corporativo válido (@"+CorporateDomain+")")
	}
	return normalized, nil
}

func validCorporateEmail(email string) bool {
	local, domain, found := strings.Cut(email, "@")
	if !found || domain != CorporateDomain || local == "" || len(local) > 64 {
		return false
	}
	previousSeparator := true
	for _, r := range local {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			previousSeparator = false
		case r == '.' || r == '_' || r == '-':
			if previousSeparator {
				return false
			}
			previousSeparator = true
		default:
			return false
		}
	}
	return !previousSeparator
}

func ParseSite(value string) (Site, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	normalized = strings.TrimPrefix(normalized, "SEDE ")
	site := Site(normalized)
	if !site.Valid() {
		return "", apperr.New(apperr.Validation, "la sede debe ser A o B")
	}
	return site, nil
}

func (s Site) Valid() bool {
	return s == SiteA || s == SiteB
}

func (r Role) Valid() bool {
	return r == RoleUser || r == RoleAdmin
}

func (u *User) ID() string           { return u.id }
func (u *User) Name() string         { return u.name }
func (u *User) Email() string        { return u.email }
func (u *User) Site() Site           { return u.site }
func (u *User) Area() string         { return u.area }
func (u *User) Role() Role           { return u.role }
func (u *User) IsActive() bool       { return u.active }
func (u *User) IsAdmin() bool        { return u.role == RoleAdmin }
func (u *User) CreatedAt() time.Time { return u.createdAt }

func (u *User) Deactivate() error {
	if !u.active {
		return apperr.New(apperr.Conflict, "el usuario "+u.email+" ya se encuentra desactivado")
	}
	u.active = false
	return nil
}

func (u *User) ChangeArea(area string) error {
	area = strings.TrimSpace(area)
	if n := utf8.RuneCountInString(area); n < 2 || n > 60 {
		return apperr.New(apperr.Validation, "el área debe tener entre 2 y 60 caracteres")
	}
	u.area = area
	return nil
}
