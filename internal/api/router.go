package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sumanthd032/codedrop/internal/db"
	"github.com/sumanthd032/codedrop/internal/store"
)

// Server holds the dependencies for our API
type Server struct {
	DB    *db.DB
	Store *store.Store
	Router *chi.Mux
}

// NewServer initializes the router and dependencies
func NewServer(db *db.DB, store *store.Store) *Server {
	s := &Server{
		DB:    db,
		Store: store,
		Router: chi.NewRouter(),
	}

	s.routes()
	return s
}

// routes defines the API endpoints
func (s *Server) routes() {
	// Middleware (The Pipeline for every request)
	// Logger: Logs every request (method, path, duration)
	s.Router.Use(middleware.Logger)
	// Recoverer: If code panics (crashes), this catches it and returns 500 instead of killing the server
	s.Router.Use(middleware.Recoverer)
	// Timeout: Hard limit of 60s per request to prevent hanging connections
	s.Router.Use(middleware.Timeout(60 * time.Second))

	// Routes
	s.Router.Get("/health", s.handleHealthCheck())
	
	// API Group (v1)
	s.Router.Route("/api/v1", func(r chi.Router) {
		// API endpoints will go here 
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		})
	})
}