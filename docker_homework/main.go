package main

import (
	"log"
	"net/http"
)

func main() {
	s := http.NewServeMux()
	s.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	if err := http.ListenAndServe(":8000", s); err != nil {
		log.Fatal(err)
	}
}
