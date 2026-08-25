package server

import (
	"log"
	"net/http"
)

func Start(addr string, lb http.Handler, fs http.Handler) {
	mux := http.NewServeMux()

	mux.Handle("/static/", fs)
	mux.Handle("/", lb)

	log.Printf("server listening on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
