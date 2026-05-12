// Package ctxkey provides shared context keys to avoid import cycles
// between the transport (which sets values) and the repository / outbox
// layers (which read them, e.g. to attach request_id to outbox metadata).
package ctxkey

import "context"

type key int

const (
	requestID key = iota
)

// WithRequestID returns a copy of ctx carrying the given request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestID, id)
}

// RequestIDFrom returns the request ID stored in ctx, or "" if absent.
func RequestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestID).(string); ok {
		return v
	}
	return ""
}
