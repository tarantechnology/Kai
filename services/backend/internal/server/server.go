package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/taran/kai/services/backend/internal/config"
	"github.com/taran/kai/services/backend/internal/handlers"
)

func New(cfg config.Config) http.Handler {
	router := chi.NewRouter()
	authHandler := handlers.NewAuthHandler(cfg)

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(localCORS)

	router.Get("/", authHandler.AuthLanding())
	router.Get("/health", handlers.Health())

	router.Route("/auth", func(r chi.Router) {
		r.Get("/google/start", authHandler.GoogleStart())
		r.Get("/google/callback", authHandler.GoogleCallback())
		r.Post("/google/session", authHandler.GoogleFinalize())
		r.Get("/google/status", authHandler.GoogleStatus())
		r.Get("/email/confirmed", authHandler.EmailConfirmed())
		r.Post("/email/sign-up", authHandler.EmailSignUp())
		r.Post("/email/sign-in", authHandler.EmailSignIn())
		r.Get("/me", authHandler.Me())
		r.Post("/logout", authHandler.Logout())
	})

	return router
}

func localCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
