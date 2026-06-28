package errs

// Code is a stable machine-readable error code returned in the API envelope.
type Code string

const (
	CodeEmailTaken         Code = "AUTH_EMAIL_TAKEN"
	CodeInvalidCredentials Code = "AUTH_INVALID_CREDENTIALS"
	CodeUserNotFound       Code = "AUTH_USER_NOT_FOUND"
	CodeUserInactive       Code = "AUTH_USER_INACTIVE"
	CodeInvalidToken       Code = "AUTH_INVALID_TOKEN"
	CodeTokenExpired       Code = "AUTH_TOKEN_EXPIRED"

	CodeBadRequest Code = "BAD_REQUEST"

	CodeNotFound  Code = "NOT_FOUND"
	CodeForbidden Code = "FORBIDDEN"
	CodeConflict  Code = "CONFLICT"

	CodeAIDisabled            Code = "AI_DISABLED"
	CodeAIDailyBudgetExceeded  Code = "AI_DAILY_BUDGET_EXCEEDED"
	CodeAIMonthlyBudgetExceeded Code = "AI_MONTHLY_BUDGET_EXCEEDED"

	CodeRunTimeout   Code = "RUN_TIMEOUT"
	CodeRunCancelled Code = "RUN_CANCELLED"

	CodeMockNoMatch Code = "MOCK_NO_MATCH"
)

// CodeFor maps a sentinel error to its stable code. Returns "" for unknown errors.
func CodeFor(err error) Code {
	m := map[error]Code{
		ErrEmailTaken:         CodeEmailTaken,
		ErrInvalidCredentials: CodeInvalidCredentials,
		ErrUserNotFound:       CodeUserNotFound,
		ErrUserInactive:       CodeUserInactive,
		ErrInvalidToken:       CodeInvalidToken,
		ErrTokenExpired:       CodeTokenExpired,
		ErrBadRequest:         CodeBadRequest,
		ErrNotFound:           CodeNotFound,
		ErrForbidden:          CodeForbidden,
		ErrConflict:           CodeConflict,
	}
	return m[err]
}
