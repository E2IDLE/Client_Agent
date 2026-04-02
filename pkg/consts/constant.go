package consts

import "time"

type ContextKey string

const (
	ReleaseMode = false

	TraceIDKey    = "traceId"
	TraceIDHeader = "X-Trace-ID"
	CtxTraceID    = ContextKey("trace_id")

	Version = "0.0.0"

	FileTransferProtocol = "/file-transfer/1.0.0"
	FileSaveDir          = "."

	KeyECode   = "code"
	KeyMessage = "message"
	KeyDetail  = "detail"

	ErrCodeInvalidParam = "E1001"
	ErrCodeNotFound     = "E3001"
	ErrCodeInternal     = "E6001"

	MsgInvalidRequest      = "잘못된 요청 파라미터"
	MsgTooManyRequests     = "too many requests"
	MsgRequiredField       = "필수 필드 누락"
	MsgInternalServerError = "에이전트 내부 오류"

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
