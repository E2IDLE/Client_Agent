package util

import (
	"Client_Agent/pkg/consts"
	"context"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	defaultLogFilePath   = "./logs/app.log"
	defaultLogMaxSizeMB  = 100
	defaultLogMaxBackups = 3
	defaultLogMaxAgeDays = 28
)

var (
	phoneRegex = regexp.MustCompile(`\b(01[016789])-?(\d{3,4})-?(\d{4})\b`)
	emailRegex = regexp.MustCompile(`\b([a-zA-Z0-9._%+\-])([a-zA-Z0-9._%+\-]*)@([a-zA-Z0-9.\-]+\.[a-zA-Z]{2,})\b`)
)

var appLogger *logrus.Logger

func SetDefaultLogger() {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)

	lj := &lumberjack.Logger{
		Filename:   defaultLogFilePath,
		MaxSize:    defaultLogMaxSizeMB,
		MaxBackups: defaultLogMaxBackups,
		MaxAge:     defaultLogMaxAgeDays,
		Compress:   true,
	}

	log.SetOutput(io.MultiWriter(os.Stdout, lj))
	appLogger = log
}

func GetLogger() *logrus.Logger {
	return appLogger
}

func withTrace(ctx context.Context) *logrus.Entry {
	traceID, _ := ctx.Value(consts.CtxTraceID).(string)
	return appLogger.WithField(consts.TraceIDKey, traceID)
}

func LogInfo(ctx context.Context, msg string, fields ...logrus.Fields) {
	entry := withTrace(ctx)
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Info(msg)
}

func LogWarn(ctx context.Context, msg string, fields ...logrus.Fields) {
	entry := withTrace(ctx)
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Warn(msg)
}

func LogError(ctx context.Context, msg string, fields ...logrus.Fields) {
	entry := withTrace(ctx)
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Error(msg)
}

// MaskSensitiveFormat 는 전화번호와 이메일을 마스킹한다.
// 전화번호: 010-1234-5678 → 010-****-5678
// 이메일: alice.admin@example.com → a***@example.com
func MaskSensitiveFormat(input string) string {
	result := phoneRegex.ReplaceAllString(input, "${1}-****-${3}")
	result = emailRegex.ReplaceAllStringFunc(result, maskEmail)
	return result
}

func maskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return email
	}
	local := parts[0]
	domain := parts[1]
	if len(local) <= 1 {
		return local + "***@" + domain
	}
	return string(local[0]) + "***@" + domain
}
