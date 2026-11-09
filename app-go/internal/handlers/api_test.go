package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"maizap/internal/protocol"
	"maizap/internal/services"
	"maizap/internal/storage"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	store := storage.NewMemoryStore()
	users := services.NewUserService(store, time.Now)
	mail := services.NewMailService(store, users, time.Now)
	auth := services.NewLocalAuthenticator(users, time.Hour, time.Now)
	srv := httptest.NewServer(NewAPI(users, mail, auth).Routes())
	t.Cleanup(srv.Close)
	return srv
}

func call(t *testing.T, srv *httptest.Server, method, path, token string, body any) (int, protocol.Response) {
	t.Helper()
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, srv.URL+path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("error de red: %v", err)
	}
	defer resp.Body.Close()
	var out protocol.Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("respuesta no JSON en %s %s: %v", method, path, err)
	}
	return resp.StatusCode, out
}

func TestEndToEnd(t *testing.T) {
	srv := newTestServer(t)
	p := protocol.APIPrefix

	status, _ := call(t, srv, "POST", p+"/users", "", protocol.RegisterRequest{Name: "Ana Torres", Email: "ana@maizap.com", Site: "A", Area: "Compras"})
	if status != http.StatusCreated {
		t.Fatalf("registro: estado %d", status)
	}
	call(t, srv, "POST", p+"/users", "", protocol.RegisterRequest{Name: "Luis Paredes", Email: "luis@maizap.com", Site: "B", Area: "Logística"})

	status, resp := call(t, srv, "POST", p+"/users", "", map[string]string{"name": ""})
	if status != http.StatusBadRequest || resp.Error == nil {
		t.Fatalf("registro inválido: estado %d", status)
	}

	if status, _ := call(t, srv, "GET", p+"/mailbox/inbox", "", nil); status != http.StatusUnauthorized {
		t.Fatalf("sin token se esperaba 401, se obtuvo %d", status)
	}

	_, resp = call(t, srv, "POST", p+"/sessions", "", protocol.LoginRequest{Email: "ana@maizap.com"})
	var login protocol.LoginResponse
	if err := json.Unmarshal(resp.Data, &login); err != nil || login.Token == "" {
		t.Fatalf("login sin token: %v", err)
	}

	status, resp = call(t, srv, "POST", p+"/messages", login.Token, protocol.SendRequest{To: []string{"luis@maizap.com"}, Subject: "Hola", Body: "Prueba"})
	if status != http.StatusCreated {
		t.Fatalf("envío: estado %d %+v", status, resp.Error)
	}

	status, resp = call(t, srv, "POST", p+"/messages", login.Token, protocol.SendRequest{To: []string{"fantasma@maizap.com"}, Subject: "Hola", Body: "Prueba"})
	if status != http.StatusBadRequest || len(resp.Error.Details) == 0 {
		t.Fatalf("destinatario inexistente: estado %d", status)
	}

	status, resp = call(t, srv, "GET", p+"/mailbox/sent", login.Token, nil)
	var box protocol.MailboxResponse
	_ = json.Unmarshal(resp.Data, &box)
	if status != http.StatusOK || box.Total != 1 {
		t.Fatalf("enviados: estado %d total %d", status, box.Total)
	}

	if status, _ := call(t, srv, "GET", p+"/mailbox/spam", login.Token, nil); status != http.StatusNotFound {
		t.Fatalf("carpeta inexistente: se esperaba 404, se obtuvo %d", status)
	}

	if status, _ := call(t, srv, "POST", p+"/users/USR-0002/deactivate", login.Token, nil); status != http.StatusForbidden {
		t.Fatalf("desactivar sin ser admin: se esperaba 403, se obtuvo %d", status)
	}

	if status, _ := call(t, srv, "DELETE", p+"/sessions/current", login.Token, nil); status != http.StatusOK {
		t.Fatalf("logout: estado %d", status)
	}
	if status, _ := call(t, srv, "GET", p+"/me", login.Token, nil); status != http.StatusUnauthorized {
		t.Fatalf("token revocado: se esperaba 401, se obtuvo %d", status)
	}
}
