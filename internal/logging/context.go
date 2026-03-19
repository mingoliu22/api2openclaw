package logging

import (
	"context"
	"github.com/google/uuid"
)

// traceIDKey context 中 trace ID 的 key
type traceIDKey struct{}

// GenerateTraceID 生成新的 trace ID
func GenerateTraceID() string {
	return uuid.New().String()
}

// ContextWithTraceID 创建包含 trace ID 的 context
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// GetTraceIDFromContext 从 context 获取 trace ID
func getTraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey{}).(string); ok {
		return traceID
	}
	return ""
}

// GetTraceID 公开方法：从 context 获取 trace ID
func GetTraceID(ctx context.Context) string {
	return getTraceIDFromContext(ctx)
}
