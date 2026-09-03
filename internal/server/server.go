package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lb/internal/middleware"
)

func Start(addr string, lb http.Handler, middlewares ...middleware.Middleware) {
	mux := http.NewServeMux()

	handler := lb

	mw := middleware.Chain(middlewares...)

	handler = mw(handler)

	mux.Handle("/", handler)

	srv := &http.Server{
		Handler: mux,
		Addr:    addr,
	}

	go func() {
		log.Printf("server listening on %s", addr)

		if err := srv.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)

	signal.Notify(
		sig,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-sig

	log.Println("shutdown signal received")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)

		if err := srv.Close(); err != nil {
			log.Printf("server close failed: %v", err)
		}
	}

	log.Println("server stopped")
}
