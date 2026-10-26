package main


import (
	"fmt"
	"net/http"
	"encoding/json"

	"github.com/go-chi/chi/v5"
)

func main(){
	r := chi.NewRouter()

	r.Get("/test", func(w http.ResponseWriter, r *http.Request){
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"msg":"done"})
	})

	fmt.Println("server is runnning on PORT: ", 8080)
	http.ListenAndServe(":8080", r)
}