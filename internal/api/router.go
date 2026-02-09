package api

import (

	"github.com/go-chi/chi/v5"
	"github.com/sumanthd032/codedrop/internal/db"
	"github.com/sumanthd032/codedrop/internal/store"
)

// Server holds the dependencies for our API
type Server struct {
	DB     *db.DB
	Store  *store.Store
	Router *chi.Mux
}

// NewServer initializes the router and dependencies
func NewServer(db *db.DB, store *store.Store) *Server {
	s := &Server{
		DB:     db,
		Store:  store,
		Router: chi.NewRouter(),
	}

	return s
}
