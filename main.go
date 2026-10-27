package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	fmt.Println("server is runnning on PORT: ", 8080)
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		fmt.Println("failed to start the server: ", err)
	}
}
