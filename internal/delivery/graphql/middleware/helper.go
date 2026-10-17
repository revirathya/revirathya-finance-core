package middleware

import "net/http"

func ChainMiddleware(handler http.Handler, middlewares ...Middleware) http.Handler {
	if len(middlewares) > 1 {
		wrappedHandler := middlewares[0](handler)
		return ChainMiddleware(wrappedHandler, middlewares[1:]...)
	}
	return handler
}
