package handlers

import (
	"net/http"

	"maizap/internal/models"
	"maizap/internal/protocol"
	"maizap/internal/services"
)

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var req protocol.RegisterRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	user, err := a.users.Register(services.RegisterInput{
		Name:  req.Name,
		Email: req.Email,
		Site:  req.Site,
		Area:  req.Area,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, toUserDTO(user))
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var req protocol.LoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	session, user, err := a.auth.Authenticate(req.Email)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, protocol.LoginResponse{
		Token:     session.Token(),
		ExpiresAt: session.ExpiresAt(),
		User:      toUserDTO(user),
	})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request, user *models.User, token string) {
	if err := a.auth.Revoke(token); err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]string{"message": "sesión cerrada para " + user.Email()})
}

func (a *API) me(w http.ResponseWriter, r *http.Request, user *models.User, _ string) {
	writeData(w, http.StatusOK, toUserDTO(user))
}

func (a *API) listUsers(w http.ResponseWriter, r *http.Request, _ *models.User, _ string) {
	users, err := a.users.List()
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]protocol.UserDTO, 0, len(users))
	for _, u := range users {
		out = append(out, toUserDTO(u))
	}
	writeData(w, http.StatusOK, out)
}

func (a *API) deactivateUser(w http.ResponseWriter, r *http.Request, actor *models.User, _ string) {
	user, err := a.users.Deactivate(actor, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, toUserDTO(user))
}

func toUserDTO(u *models.User) protocol.UserDTO {
	return protocol.UserDTO{
		ID:     u.ID(),
		Name:   u.Name(),
		Email:  u.Email(),
		Site:   string(u.Site()),
		Area:   u.Area(),
		Role:   string(u.Role()),
		Active: u.IsActive(),
	}
}

func toUserRef(u *models.User) protocol.UserRef {
	return protocol.UserRef{Name: u.Name(), Email: u.Email()}
}
