package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"maizap/internal/apperr"
	"maizap/internal/models"
	"maizap/internal/protocol"
	"maizap/internal/services"
)

type API struct {
	users *services.UserService
	mail  *services.MailService
	auth  services.Authenticator
	now   func() time.Time
}

type authedHandler func(w http.ResponseWriter, r *http.Request, user *models.User, token string)

func NewAPI(users *services.UserService, mail *services.MailService, auth services.Authenticator) *API {
	return &API{users: users, mail: mail, auth: auth, now: time.Now}
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	p := protocol.APIPrefix

	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("POST "+p+"/users", a.register)
	mux.HandleFunc("POST "+p+"/sessions", a.login)
	mux.HandleFunc("DELETE "+p+"/sessions/current", a.requireSession(a.logout))
	mux.HandleFunc("GET "+p+"/me", a.requireSession(a.me))
	mux.HandleFunc("GET "+p+"/users", a.requireSession(a.listUsers))
	mux.HandleFunc("POST "+p+"/users/{id}/deactivate", a.requireSession(a.deactivateUser))
	mux.HandleFunc("POST "+p+"/messages", a.requireSession(a.sendMessage))
	mux.HandleFunc("GET "+p+"/mailbox/{folder}", a.requireSession(a.mailbox))
	mux.HandleFunc("GET "+p+"/messages/{id}", a.requireSession(a.readMessage))
	mux.HandleFunc("DELETE "+p+"/messages/{id}", a.requireSession(a.deleteMessage))
	mux.HandleFunc("/", a.notFound)

	return recoverer(logRequests(mux))
}

func (a *API) requireSession(next authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := bearerToken(r)
		if err != nil {
			writeError(w, err)
			return
		}
		session, err := a.auth.Validate(token)
		if err != nil {
			writeError(w, err)
			return
		}
		user, err := a.users.ByID(session.UserID())
		if err != nil {
			writeError(w, apperr.New(apperr.Unauthorized, "el usuario de la sesión ya no existe"))
			return
		}
		if !user.IsActive() {
			_ = a.auth.Revoke(token)
			writeError(w, apperr.New(apperr.Forbidden, "la cuenta está desactivada; contacte al administrador"))
			return
		}
		next(w, r, user, token)
	}
}

func bearerToken(r *http.Request) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", apperr.New(apperr.Unauthorized, "debe identificarse para realizar esta operación")
	}
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", apperr.New(apperr.Unauthorized, "el encabezado Authorization debe tener el formato 'Bearer <token>'")
	}
	return strings.TrimSpace(token), nil
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	writeData(w, http.StatusOK, protocol.HealthResponse{Service: "maizap-server", Status: "ok", Time: a.now()})
}

func (a *API) notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, apperr.New(apperr.NotFound, "la ruta "+r.Method+" "+r.URL.Path+" no existe"))
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %s -> %d (%s)", r.RemoteAddr, r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Microsecond))
	})
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				log.Printf("fallo inesperado atendiendo %s %s: %v", r.Method, r.URL.Path, v)
				writeError(w, apperr.New(apperr.Internal, "error interno del servidor"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}
