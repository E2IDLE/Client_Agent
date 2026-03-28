package errors

import (
	stderrors "errors"
	"net/http"
)

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func New(status int, message string) error {
	return &Error{Status: status, Message: message}
}

// As 는 err가 *Error 타입이면 추출하여 반환한다.
func As(err error) (*Error, bool) {
	var e *Error
	if stderrors.As(err, &e) {
		return e, true
	}
	return nil, false
}

var (
	ErrUserAlreadyExists  = New(http.StatusConflict, "username already exists")
	ErrUserNotFound       = New(http.StatusNotFound, "user not found")
	ErrInvalidCredentials = New(http.StatusUnauthorized, "invalid credentials")
	ErrInvalidToken       = New(http.StatusUnauthorized, "invalid authorization token")
	ErrPasswordMismatch   = New(http.StatusUnauthorized, "password mismatch")
	ErrPostNotFound       = New(http.StatusNotFound, "post not found")
	ErrUnauthorizedAction = New(http.StatusForbidden, "unauthorized action")
)
