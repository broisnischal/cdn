package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

func main() {
	publicDir := filepath.Join(".", "public")
	staticFS := http.FileServer(http.Dir(publicDir))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Cache-Control", "public, max-age=15")
			http.ServeFile(w, r, filepath.Join(publicDir, "index.html"))
			return
		case "/healthz":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		default:
			// Static assets are immutable and long-lived.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			staticFS.ServeHTTP(w, r)
			if r.URL.Path != "/favicon.ico" && r.URL.Path != "/robots.txt" {
				log.Printf("origin static request path=%s served_at=%s", r.URL.Path, time.Now().Format(time.RFC3339))
			}
		}
	})

	http.HandleFunc("/api/echo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=30")
		fmt.Fprintf(w, "Hello from origin! You requested: %s (served at %s)\n",
			r.URL.Path, time.Now().Format(time.RFC3339))
	})

	log.Println("Origin static server listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
