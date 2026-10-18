package middleware

import "net/http"

func ChainMiddleware(handler http.Handler, middlewares ...Middleware) http.Handler {
	if len(middlewares) == 0 {
		return handler
	}
	return middlewares[0](ChainMiddleware(handler, middlewares[1:]...))
}
