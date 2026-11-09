package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"maizap/internal/apperr"
	"maizap/internal/protocol"
)

const maxBodyBytes = 64 * 1024

type envelope struct {
	OK    bool                `json:"ok"`
	Data  any                 `json:"data,omitempty"`
	Error *protocol.ErrorBody `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("error al escribir respuesta: %v", err)
	}
}

func writeData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, envelope{OK: true, Data: data})
}

func writeError(w http.ResponseWriter, err error) {
	appErr := apperr.From(err)
	status := statusFor(appErr.Kind())
	message := appErr.Message()
	if status >= http.StatusInternalServerError {
		log.Printf("error interno: %v", err)
	}
	writeJSON(w, status, envelope{
		OK: false,
		Error: &protocol.ErrorBody{
			Code:    string(appErr.Kind()),
			Message: message,
			Details: appErr.Details(),
		},
	})
}

func statusFor(kind apperr.Kind) int {
	switch kind {
	case apperr.Validation:
		return http.StatusBadRequest
	case apperr.NotFound:
		return http.StatusNotFound
	case apperr.Conflict:
		return http.StatusConflict
	case apperr.Forbidden:
		return http.StatusForbidden
	case apperr.Unauthorized:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		return apperr.New(apperr.Validation, "el contenido debe enviarse como application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		switch {
		case errors.Is(err, io.EOF):
			return apperr.New(apperr.Validation, "el cuerpo de la solicitud está vacío")
		case errors.As(err, &tooLarge):
			return apperr.New(apperr.Validation, "la solicitud supera el tamaño máximo permitido")
		default:
			return apperr.New(apperr.Validation, "el cuerpo de la solicitud no es JSON válido").WithDetails(err.Error())
		}
	}
	if decoder.More() {
		return apperr.New(apperr.Validation, "la solicitud debe contener un único objeto JSON")
	}
	return nil
}
