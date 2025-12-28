package main

import (
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/livebud/mux"
	"github.com/matthewmueller/logs"
	"github.com/matthewmueller/views/react"
)

//go:generate go run github.com/matthewmueller/views/cmd/react **.tsx

//go:embed dist/**
var distfs embed.FS

func main() {
	log := logs.Default()
	if err := run(log); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	viewer, err := react.Load(
		react.WithFS(distfs),
		react.WithLog(log),
	)
	if err != nil {
		return err
	}
	defer viewer.Close()
	router := mux.New()
	router.Use(viewer.Middleware())
	router.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		defer func() {
			fmt.Printf("Request %s took %s\n", r.URL.Path, time.Since(now))
		}()
		if err := viewer.Render(w, "index.tsx", map[string]any{
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
		if err := viewer.Render(w, "dashboard/dashboard.tsx", map[string]any{
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
		viewer.Render(w, "about.tsx", map[string]any{
			"name": "Max",
		})
	}))
	fmt.Println("Listening on http://localhost:8000")
	return http.ListenAndServe(":8000", router)
}
