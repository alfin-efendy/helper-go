package database

import (
	"context"

	"github.com/alfin-efendy/helper-go/otel"
)

func Init(ctx context.Context) {
	ctx, span := otel.Trace(ctx)
	defer span.End()

	initSql(ctx)
	initRedis(ctx)
}
