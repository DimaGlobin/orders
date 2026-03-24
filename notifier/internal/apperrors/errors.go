package apperrors

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation error")
	ErrInternal   = errors.New("internal error")
	ErrConflict   = errors.New("conflict")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field %q: %s", e.Field, e.Message)
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

type AppError struct {
	Code string
	Err  error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s", e.Code, e.Err.Error())
	}
	return fmt.Sprintf("[%s]", e.Code)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code string, err error) *AppError {
	return &AppError{Code: code, Err: err}
}
