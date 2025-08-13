package http

import (
	"net/http"
)

type HandlerFunc func(*Ctx)

func Handler(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := GetCtx(r)
		ctx.ResponseWriter = w
		ctx.Request = r
		h(ctx)
	}
}
