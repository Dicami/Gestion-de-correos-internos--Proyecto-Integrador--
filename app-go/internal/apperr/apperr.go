package apperr

import "errors"

type Kind string

const (
	Validation   Kind = "validation"
	NotFound     Kind = "not_found"
	Conflict     Kind = "conflict"
	Forbidden    Kind = "forbidden"
	Unauthorized Kind = "unauthorized"
	Storage      Kind = "storage"
	Internal     Kind = "internal"
)

type Error struct {
	kind    Kind
	message string
	details []string
	cause   error
}

func New(kind Kind, message string) *Error {
	return &Error{kind: kind, message: message}
}

func Wrap(kind Kind, message string, cause error) *Error {
	return &Error{kind: kind, message: message, cause: cause}
}

func (e *Error) WithDetails(details ...string) *Error {
	e.details = append(e.details, details...)
	return e
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.message + ": " + e.cause.Error()
	}
	return e.message
}

func (e *Error) Unwrap() error {
	return e.cause
}

func (e *Error) Kind() Kind {
	return e.kind
}

func (e *Error) Message() string {
	return e.message
}

func (e *Error) Details() []string {
	out := make([]string, len(e.details))
	copy(out, e.details)
	return out
}

func KindOf(err error) Kind {
	var target *Error
	if errors.As(err, &target) {
		return target.kind
	}
	return Internal
}

func Is(err error, kind Kind) bool {
	return err != nil && KindOf(err) == kind
}

func From(err error) *Error {
	var target *Error
	if errors.As(err, &target) {
		return target
	}
	return Wrap(Internal, "error interno del servidor", err)
}
