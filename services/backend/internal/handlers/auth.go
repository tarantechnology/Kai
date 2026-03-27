package handlers

import (
	"encoding/json"
	"net/http"
)

type authResponse struct {
	Provider string `json:"provider"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

func AuthStart(provider string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(authResponse{
			Provider: provider,
			Status:   "not_implemented",
			Message:  "OAuth start is scaffolded but not wired to provider credentials yet.",
		})
	}
}

func AuthCallback(provider string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(authResponse{
			Provider: provider,
			Status:   "not_implemented",
			Message:  "OAuth callback is scaffolded but not wired to token exchange yet.",
		})
	}
}

