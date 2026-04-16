package log

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type contextKey string

const (
	correlationIDKey contextKey = "correlation_id"
	operationKey     contextKey = "operation"
	startTimeKey     contextKey = "start_time"
)

var (
	correlationIDGenerator = &idGenerator{}
)

type idGenerator struct {
	mu  sync.Mutex
	seq uint64
}

func (g *idGenerator) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.seq++
	return time.Now().Format("20060102-150405") + "-" + fmt.Sprint(g.seq)
}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	if id == "" {
		id = correlationIDGenerator.Generate()
	}
	return context.WithValue(ctx, correlationIDKey, id)
}

func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, operationKey, operation)
}

func WithStartTime(ctx context.Context) context.Context {
	return context.WithValue(ctx, startTimeKey, time.Now())
}

func GetCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

func GetOperation(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if op, ok := ctx.Value(operationKey).(string); ok {
		return op
	}
	return ""
}

func GetStartTime(ctx context.Context) time.Time {
	if ctx == nil {
		return time.Time{}
	}
	if t, ok := ctx.Value(startTimeKey).(time.Time); ok {
		return t
	}
	return time.Time{}
}

func Duration(ctx context.Context) time.Duration {
	start := GetStartTime(ctx)
	if start.IsZero() {
		return 0
	}
	return time.Since(start)
}
