package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"maizap/internal/config"
	"maizap/internal/protocol"
)

type ConnectionError struct {
	Address string
	Cause   error
}

func (e *ConnectionError) Error() string {
	var netErr net.Error
	if errors.As(e.Cause, &netErr) && netErr.Timeout() {
		return fmt.Sprintf("el servidor %s no respondió a tiempo", e.Address)
	}
	return fmt.Sprintf("no se pudo conectar con el servidor %s; verifique que esté en ejecución y que la IP y el puerto sean correctos", e.Address)
}

func (e *ConnectionError) Unwrap() error {
	return e.Cause
}

type APIError struct {
	Status  int
	Code    string
	Message string
	Details []string
}

func (e *APIError) Error() string {
	return e.Message
}

type APIClient struct {
	cfg   config.ClientConfig
	http  *http.Client
	token string
}

func NewAPIClient(cfg config.ClientConfig) *APIClient {
	return &APIClient{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

func (c *APIClient) Address() string {
	return c.cfg.Address()
}

func (c *APIClient) HasSession() bool {
	return c.token != ""
}

func (c *APIClient) ClearSession() {
	c.token = ""
}

func (c *APIClient) Health() (protocol.HealthResponse, error) {
	var out protocol.HealthResponse
	err := c.do(http.MethodGet, "/health", nil, &out)
	return out, err
}

func (c *APIClient) Register(req protocol.RegisterRequest) (protocol.UserDTO, error) {
	var out protocol.UserDTO
	err := c.do(http.MethodPost, protocol.APIPrefix+"/users", req, &out)
	return out, err
}

func (c *APIClient) Login(email string) (protocol.LoginResponse, error) {
	var out protocol.LoginResponse
	if err := c.do(http.MethodPost, protocol.APIPrefix+"/sessions", protocol.LoginRequest{Email: email}, &out); err != nil {
		return out, err
	}
	c.token = out.Token
	return out, nil
}

func (c *APIClient) Logout() error {
	err := c.do(http.MethodDelete, protocol.APIPrefix+"/sessions/current", nil, nil)
	c.token = ""
	return err
}

func (c *APIClient) Users() ([]protocol.UserDTO, error) {
	var out []protocol.UserDTO
	err := c.do(http.MethodGet, protocol.APIPrefix+"/users", nil, &out)
	return out, err
}

func (c *APIClient) DeactivateUser(id string) (protocol.UserDTO, error) {
	var out protocol.UserDTO
	err := c.do(http.MethodPost, protocol.APIPrefix+"/users/"+url.PathEscape(id)+"/deactivate", nil, &out)
	return out, err
}

func (c *APIClient) Send(req protocol.SendRequest) (protocol.MessageDetail, error) {
	var out protocol.MessageDetail
	err := c.do(http.MethodPost, protocol.APIPrefix+"/messages", req, &out)
	return out, err
}

func (c *APIClient) Mailbox(folder string) (protocol.MailboxResponse, error) {
	var out protocol.MailboxResponse
	err := c.do(http.MethodGet, protocol.APIPrefix+"/mailbox/"+url.PathEscape(folder), nil, &out)
	return out, err
}

func (c *APIClient) Message(id string) (protocol.MessageDetail, error) {
	var out protocol.MessageDetail
	err := c.do(http.MethodGet, protocol.APIPrefix+"/messages/"+url.PathEscape(id), nil, &out)
	return out, err
}

func (c *APIClient) DeleteMessage(id string) error {
	return c.do(http.MethodDelete, protocol.APIPrefix+"/messages/"+url.PathEscape(id), nil, nil)
}

func (c *APIClient) do(method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("no se pudo preparar la solicitud: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, c.cfg.BaseURL()+path, reader)
	if err != nil {
		return fmt.Errorf("no se pudo construir la solicitud: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return &ConnectionError{Address: c.cfg.Address(), Cause: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return &ConnectionError{Address: c.cfg.Address(), Cause: err}
	}

	var envelope protocol.Response
	if err := json.Unmarshal(raw, &envelope); err != nil {
		text := strings.TrimSpace(string(raw))
		if text == "" {
			text = http.StatusText(resp.StatusCode)
		}
		return &APIError{Status: resp.StatusCode, Code: "invalid_response", Message: "respuesta no reconocida del servidor: " + text}
	}
	if !envelope.OK {
		apiErr := &APIError{Status: resp.StatusCode, Code: "unknown", Message: "el servidor no pudo procesar la solicitud"}
		if envelope.Error != nil {
			apiErr.Code = envelope.Error.Code
			apiErr.Message = envelope.Error.Message
			apiErr.Details = envelope.Error.Details
		}
		if resp.StatusCode == http.StatusUnauthorized {
			c.token = ""
		}
		return apiErr
	}
	if out != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return &APIError{Status: resp.StatusCode, Code: "invalid_response", Message: "no se pudo interpretar la respuesta del servidor"}
		}
	}
	return nil
}
