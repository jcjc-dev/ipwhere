package api

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// SetupMiddleware configures common middleware for the router
func SetupMiddleware(r *chi.Mux) {
	// Request ID
	r.Use(middleware.RequestID)

	// Real IP (for proxies)
	r.Use(middleware.RealIP)

	// Logger
	r.Use(middleware.Logger)

	// Recoverer
	r.Use(middleware.Recoverer)

	// Timeout
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
}

// NewRouter creates and configures a new chi router
func NewRouter() *chi.Mux {
	r := chi.NewRouter()
	SetupMiddleware(r)
	return r
}
