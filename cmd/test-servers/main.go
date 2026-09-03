package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := os.Args[1]

	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if port == "8081" {
			http.Error(w, "backend 8081 failed", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Hello from backend %s\n", port)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Println("Backend running on :" + port)
	http.ListenAndServe(":"+port, nil)
}
