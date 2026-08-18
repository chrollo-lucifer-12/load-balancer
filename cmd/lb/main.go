package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	targetURL, err := url.Parse("http://localhost:8081")
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	server := http.Server{
		Addr:    ":8080",
		Handler: proxy,
	}

	log.Printf("starting load balancer on :8080")
	log.Fatal(server.ListenAndServe())
}
