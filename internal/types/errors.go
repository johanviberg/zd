package types

import "fmt"

const (
	ExitSuccess   = 0
	ExitGeneral   = 1
	ExitArgError  = 2
	ExitAuthError = 3
	ExitRetryable = 4
	ExitNotFound  = 5
)

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	ExitCode   int    `json:"exitCode"`
	RetryAfter int    `json:"retryAfter,omitempty"`
	Cause      error  `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func NewAuthError(msg string) *AppError {
	return &AppError{Code: "auth_error", Message: msg, ExitCode: ExitAuthError}
}

func NewNotFoundError(msg string) *AppError {
	return &AppError{Code: "not_found", Message: msg, ExitCode: ExitNotFound}
}

func NewRetryableError(msg string, retryAfter int) *AppError {
	return &AppError{Code: "rate_limited", Message: msg, ExitCode: ExitRetryable, RetryAfter: retryAfter}
}

func NewArgError(msg string) *AppError {
	return &AppError{Code: "invalid_argument", Message: msg, ExitCode: ExitArgError}
}

func NewGeneralError(msg string) *AppError {
	return &AppError{Code: "error", Message: msg, ExitCode: ExitGeneral}
}
