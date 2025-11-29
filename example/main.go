package main

import (
	"embed"
	"fmt"
	"net/http"
	"time"

	"github.com/livebud/mux"
	"github.com/matthewmueller/logs"
	"github.com/matthewmueller/views/react"
)

//go:generate go run github.com/matthewmueller/views/cmd/react *.tsx

//go:embed dist/**.js
var distfs embed.FS

func main() {
	log := logs.Default()
	react := react.New(log)
	react.FS = distfs
	router := mux.New()
	router.Use(react.Middleware())
	router.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		defer func() {
			fmt.Printf("Request %s took %s\n", r.URL.Path, time.Since(now))
		}()
		if err := react.Render(w, "index.jsx", map[string]any{
			"name": "Matt",
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	router.Get("/dashboard", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		defer func() {
			fmt.Printf("Request %s took %s\n", r.URL.Path, time.Since(now))
		}()
		if err := react.Render(w, "dashboard/dashboard.tsx", map[string]any{
			"name": "Matt",
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	router.Get("/about", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		defer func() {
			fmt.Printf("Request %s took %s\n", r.URL.Path, time.Since(now))
		}()
		react.Render(w, "about.jsx", map[string]any{
			"name": "Max",
		})
	}))
	fmt.Println("Listening on http://localhost:8000")
	if err := http.ListenAndServe(":8000", router); err != nil {
		log.Error(err.Error())
	}
}
