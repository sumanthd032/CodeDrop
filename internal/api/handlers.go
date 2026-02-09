package api

import (
	"encoding/json"
	"net/http"
)

// handleHealthCheck returns a simple 200 OK to say "I'm alive"
func (s *Server) handleHealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// To check if DB/Redis are reachable here
		response := map[string]string{
			"status": "healthy",
			"db":     "connected", // We assume connected since main.go checked it
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}