package requestcontext

import "context"

type userIDKey struct{}

const legacyUserIDKey = "user_id"

func WithUserID(ctx context.Context, userID int64) context.Context {
	ctx = context.WithValue(ctx, userIDKey{}, userID)
	//lint:ignore SA1029 Existing handlers still read the legacy string key.
	return context.WithValue(ctx, legacyUserIDKey, userID)
}

func UserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey{}).(int64)
	if ok {
		return userID, true
	}
	userID, ok = ctx.Value(legacyUserIDKey).(int64)
	return userID, ok
}
