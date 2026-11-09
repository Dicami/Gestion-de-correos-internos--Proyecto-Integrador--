package protocol

import (
	"encoding/json"
	"time"
)

const APIPrefix = "/api/v1"

type Response struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error *ErrorBody      `json:"error,omitempty"`
}

type ErrorBody struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

type RegisterRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Site  string `json:"site"`
	Area  string `json:"area"`
}

type LoginRequest struct {
	Email string `json:"email"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserDTO   `json:"user"`
}

type UserDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Site   string `json:"site"`
	Area   string `json:"area"`
	Role   string `json:"role"`
	Active bool   `json:"active"`
}

type UserRef struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type SendRequest struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

type MessageSummary struct {
	ID      string    `json:"id"`
	Folder  string    `json:"folder"`
	From    UserRef   `json:"from"`
	To      []UserRef `json:"to"`
	Subject string    `json:"subject"`
	SentAt  time.Time `json:"sent_at"`
	Read    bool      `json:"read"`
}

type MessageDetail struct {
	MessageSummary
	Body   string     `json:"body"`
	ReadAt *time.Time `json:"read_at,omitempty"`
}

type MailboxResponse struct {
	Folder   string           `json:"folder"`
	Total    int              `json:"total"`
	Unread   int              `json:"unread"`
	Messages []MessageSummary `json:"messages"`
}

type HealthResponse struct {
	Service string    `json:"service"`
	Status  string    `json:"status"`
	Time    time.Time `json:"time"`
}
