package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %s %s", r.Method, r.URL.Path)
		fmt.Fprintf(w, "Hello, World!")
	})

	log.Println("Starting server on :8090")
	log.Fatal(http.ListenAndServe(":8090", nil))
}
