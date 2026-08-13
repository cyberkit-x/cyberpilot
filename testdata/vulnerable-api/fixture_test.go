package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFixturePositiveNegativeAndFalsePositiveCases(t *testing.T) {
	mux := http.NewServeMux()
	_ = mux
	tests := []struct {
		name, path, auth string
		want             int
	}{{"positive IDOR", "/objects/1", "Bearer other", http.StatusOK}, {"negative authorization", "/objects/2", "Bearer other", http.StatusForbidden}, {"valid auth", "/auth", "Bearer valid-fixture-token", http.StatusNoContent}, {"invalid auth", "/auth", "bad", http.StatusUnauthorized}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/objects/1":
			if r.Header.Get("Authorization") == "Bearer other" {
				w.WriteHeader(http.StatusOK)
				return
			}
		case "/objects/2":
			w.WriteHeader(http.StatusForbidden)
			return
		case "/auth":
			if r.Header.Get("Authorization") == "Bearer valid-fixture-token" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	for _, tt := range tests {
		request, _ := http.NewRequest(http.MethodGet, server.URL+tt.path, nil)
		request.Header.Set("Authorization", tt.auth)
		response, err := server.Client().Do(request)
		if err != nil || response.StatusCode != tt.want {
			t.Fatalf("%s status=%v err=%v", tt.name, response, err)
		}
		response.Body.Close()
	}
}
func TestAutomatedFixtureTargetsMustBeLoopback(t *testing.T) {
	for _, value := range []string{"127.0.0.1:8080", "[::1]:8080"} {
		host, _, err := net.SplitHostPort(value)
		if err != nil || !net.ParseIP(host).IsLoopback() {
			t.Fatalf("fixture target %q is not loopback", value)
		}
	}
}
