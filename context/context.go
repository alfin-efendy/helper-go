package context

import (
	"context"
)

const (
	userIdKey   = "userId"
	fullNameKey = "fullName"
	realmKey    = "realm"
)

func SetUserId(ctx context.Context, userId string) context.Context {
	return context.WithValue(ctx, userIdKey, userId)
}

func GetUserId(ctx context.Context) string {
	if userId, ok := ctx.Value(userIdKey).(string); ok {
		return userId
	}
	return ""
}

func SetFullName(ctx context.Context, fullName string) context.Context {
	return context.WithValue(ctx, fullNameKey, fullName)
}

func GetFullName(ctx context.Context) string {
	if fullName, ok := ctx.Value(fullNameKey).(string); ok {
		return fullName
	}
	return ""
}

func SetRealm(ctx context.Context, realm string) context.Context {
	return context.WithValue(ctx, realmKey, realm)
}

func GetRealm(ctx context.Context) string {
	if realm, ok := ctx.Value(realmKey).(string); ok {
		return realm
	}
	return ""
}
