package main

import (
	"encoding/json"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/objects/1", func(w http.ResponseWriter, r *http.Request) {
		identity := r.Header.Get("Authorization")
		if identity == "Bearer owner" || identity == "Bearer other" {
			_ = json.NewEncoder(w).Encode(map[string]string{"owner": "owner", "secret": "synthetic-canary"})
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	mux.HandleFunc("/objects/2", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer owner" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"owner": "owner"})
	})
	mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer valid-fixture-token" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	mux.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("url") == "http://callback.local/canary" {
			w.Header().Set("X-Fixture-Fetch", "callback.local/canary")
			w.WriteHeader(http.StatusAccepted)
			return
		}
		http.Error(w, "target rejected", http.StatusBadRequest)
	})
	mux.HandleFunc("/redirect-out", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://outside.invalid/", http.StatusFound)
	})
	address := os.Getenv("CYBERPILOT_FIXTURE_ADDR")
	if address == "" {
		address = "127.0.0.1:18080"
	}
	if err := http.ListenAndServe(address, mux); err != nil {
		panic(err)
	}
}
