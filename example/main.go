package main

import (
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/livebud/mux"
	"github.com/matthewmueller/jig"
	"github.com/matthewmueller/jig/example/dashboard"
	"github.com/matthewmueller/logs"
)

//go:generate go tool jig

//go:embed all:dist
var distfs embed.FS

func main() {
	log := logs.Default()
	if err := run(log); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	viewer, err := jig.Load(
		jig.WithFS(distfs),
		jig.WithLog(log),
	)
	if err != nil {
		return err
	}
	defer viewer.Close()
	router := mux.New()
	router.Use(viewer)
	router.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		defer func() {
			fmt.Printf("Request %s took %s\n", r.URL.Path, time.Since(now))
		}()
		if err := viewer.Render(r.Context(), w, "index.tsx", map[string]any{
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
		if err := viewer.Render(r.Context(), w, "dashboard/dashboard.tsx", dashboard.Props{
			Name: "Matt",
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	router.Get("/about", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		defer func() {
			fmt.Printf("Request %s took %s\n", r.URL.Path, time.Since(now))
		}()
		if err := viewer.Render(r.Context(), w, "about.tsx", map[string]any{
			"name": "Max",
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	fmt.Println("Listening on http://localhost:8000")
	return http.ListenAndServe(":8000", router)
}
