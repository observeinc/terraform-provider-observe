package client

import "context"

type contextKey string

var (
	// contextKeySensitive is used to mark requests as containing sensitive payloads
	contextKeySensitive = contextKey("sensitive")
	// contextKeyAuthed is used to mark requests as requiring authorization header
	contextKeyAuthed = contextKey("auth")
)

func isSensitive(ctx context.Context) bool {
	v, ok := ctx.Value(contextKeySensitive).(bool)
	if !ok {
		return false
	}
	return v
}

// requests by omission are required to be authed
func isAuthed(ctx context.Context) bool {
	v, ok := ctx.Value(contextKeyAuthed).(bool)
	if !ok {
		return true
	}
	return v
}

func setSensitive(ctx context.Context, value bool) context.Context {
	return context.WithValue(ctx, contextKeySensitive, value)
}

func setAuthed(ctx context.Context, value bool) context.Context {
	return context.WithValue(ctx, contextKeyAuthed, value)
}
