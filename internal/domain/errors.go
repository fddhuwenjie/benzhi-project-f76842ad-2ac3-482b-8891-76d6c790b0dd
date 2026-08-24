package domain

import "fmt"

type ErrorCode string

const (
	CodeInvalid     ErrorCode = "INVALID_ARGUMENT"
	CodeNotFound    ErrorCode = "NOT_FOUND"
	CodeConflict    ErrorCode = "REVISION_CONFLICT"
	CodeState       ErrorCode = "INVALID_STATE"
	CodeForbidden   ErrorCode = "FORBIDDEN"
	CodeIdempotency ErrorCode = "IDEMPOTENCY_CONFLICT"
	CodeChain       ErrorCode = "CHAIN_INCOMPLETE"
	CodeClosed      ErrorCode = "DOSSIER_CLOSED"
	CodeInternal    ErrorCode = "INTERNAL"
)

type Error struct {
	Code    ErrorCode
	Message string
	Field   string
	Details any
}

func (e *Error) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Field)
}

func NewError(code ErrorCode, message string) *Error { return &Error{Code: code, Message: message} }
func FieldError(field, message string) *Error {
	return &Error{Code: CodeInvalid, Message: message, Field: field}
}

func IsCode(err error, code ErrorCode) bool {
	de, ok := err.(*Error)
	return ok && de.Code == code
}
