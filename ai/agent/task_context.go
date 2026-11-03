package agent

import "context"

type originalUserRequestContextKey struct{}

func WithOriginalUserRequest(ctx context.Context, request string) context.Context {
	if ctx == nil || request == "" {
		return ctx
	}
	return context.WithValue(ctx, originalUserRequestContextKey{}, request)
}

func OriginalUserRequestFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	request, _ := ctx.Value(originalUserRequestContextKey{}).(string)
	return request
}
