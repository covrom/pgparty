package pgparty

import "context"

const AdminSchema = "admin"

type ctxCurrentSchema struct{}

func WithCurrentSchema(ctx context.Context, shemaName string) context.Context {
	return context.WithValue(ctx, ctxCurrentSchema{}, shemaName)
}

func CurrentSchemaFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxCurrentSchema{}).(string)
	return v, ok
}

func WithAdminSchema(ctx context.Context) context.Context {
	return WithCurrentSchema(ctx, AdminSchema)
}
