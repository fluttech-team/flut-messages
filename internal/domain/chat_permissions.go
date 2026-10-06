package domain

import "context"

type chatPermissionsKey struct{}

func WithChatPermissions(ctx context.Context, permissions *[]string) context.Context {
	return context.WithValue(ctx, chatPermissionsKey{}, permissions)
}

func HasChatPermission(ctx context.Context, permission string) bool {
	permissions, _ := ctx.Value(chatPermissionsKey{}).(*[]string)
	if permissions == nil {
		return false
	}
	for _, granted := range *permissions {
		if granted == permission {
			return true
		}
	}
	return false
}
