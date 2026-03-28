package consts

import "time"

type ContextKey string

const (
	AuthorizationCookieName = "authorization"
	AuthorizationHeader     = "Authorization"
	CookieMaxAge            = 60 * 60 * 8
	CookiePath              = "/"

	TraceIDKey    = "traceId"
	TraceIDHeader = "X-Trace-ID"
	CtxTraceID    = ContextKey("trace_id")

	ContextUserKey = "user"

	KeyMessage   = "message"
	KeyUser      = "user"
	KeyPost      = "post"
	KeyAmount    = "amount"
	KeyTarget    = "target"
	KeySessionId = "sessionId"

	MsgInvalidRequest        = "invalid request"
	MsgInvalidEmail          = "invalid email format"
	MsgInvalidPhone          = "invalid phone format"
	MsgMissingToken          = "missing authorization token"
	MsgFailedToLoadUser      = "failed to load user"
	MsgFailedToCreateSession = "failed to create session"
	MsgTooManyRequests       = "too many requests"

	MsgInvalidUsername = "username must be 3-20 characters"
	MsgInvalidName     = "name must be 2-50 characters"
	MsgInvalidPassword = "password must be 8-72 characters"
	MsgRequiredField   = "required field is missing"
	MsgInvalidTitle    = "title must be 1-100 characters"
	MsgInvalidContent  = "content must be 1-5000 characters"

	MsgInvalidAmount       = "invalid amount"
	MsgInsufficientBalance = "insufficient balance"
	MsgCannotTransferSelf  = "cannot transfer to self"
	MsgInvalidPostID       = "invalid post id"
	MsgInternalServerError = "internal server error"

	MinUsernameLen = 3
	MaxUsernameLen = 20
	MinNameLen     = 2
	MaxNameLen     = 50
	MinPasswordLen = 8
	MaxPasswordLen = 72
	MaxTitleLen    = 100
	MaxContentLen  = 5000

	MsgRegisterSuccess = "register success"
	MsgLogoutSuccess   = "logout success"
	MsgWithdrawSuccess = "account withdraw success"
	MsgDepositSuccess  = "deposit success"
	MsgBalanceWithdraw = "balance withdraw success"
	MsgTransferSuccess = "transfer success"
	MsgPostCreated     = "post created"
	MsgPostUpdated     = "post updated"
	MsgPostDeleted     = "post deleted"

	TimeFormat = time.RFC3339
)
