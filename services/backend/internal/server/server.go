package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/taran/kai/services/backend/internal/handlers"
)

func New() http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/health", handlers.Health())

	router.Route("/auth", func(r chi.Router) {
		r.Get("/google/start", handlers.AuthStart("google"))
		r.Get("/google/callback", handlers.AuthCallback("google"))
		r.Get("/canvas/start", handlers.AuthStart("canvas"))
		r.Get("/canvas/callback", handlers.AuthCallback("canvas"))
	})

	return router
}

