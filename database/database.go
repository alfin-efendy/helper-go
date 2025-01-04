package database

import "context"

func Init() {
	ctx := context.Background()

	initSql(ctx)
	initRedis(ctx)
}
