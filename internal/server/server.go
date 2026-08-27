package server

import (
	"log"
	"net/http"

	"github.com/lb/internal/middleware"
)

func Start(addr string, lb http.Handler, fs http.Handler, middlewares ...middleware.Middleware) {
	mux := http.NewServeMux()

	handler := lb

	mw := middleware.Chain(middlewares...)

	handler = mw(handler)

	mux.Handle("/static/", fs)
	mux.Handle("/", lb)

	log.Printf("server listening on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
