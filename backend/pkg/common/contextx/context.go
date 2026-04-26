package contextx

import "context"

func WithValue[T any](ctx context.Context, key any, value T) context.Context {
	return context.WithValue(ctx, key, value)
}

func Value[T any](ctx context.Context, key any) (T, bool) {
	value, ok := ctx.Value(key).(T)
	return value, ok
}
