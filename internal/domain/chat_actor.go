package domain

import "context"

type actorContextKey struct{}

func WithChatActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}
func ChatActor(ctx context.Context) string {
	actor, _ := ctx.Value(actorContextKey{}).(string)
	return actor
}

type companyContextKey struct{}

func WithChatCompany(ctx context.Context, company string) context.Context {
	return context.WithValue(ctx, companyContextKey{}, company)
}
func ChatCompany(ctx context.Context) string {
	company, _ := ctx.Value(companyContextKey{}).(string)
	return company
}
