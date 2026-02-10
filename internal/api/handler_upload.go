package api

import (
	"encoding/json"
	"net/http"
	"time"

)

// handleCreateDrop initiates the upload session
func (s *Server) handleCreateDrop() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to decode the JSON body into our CreateDropRequest struct
		var req CreateDropRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// 1. Calculate Expiry
		duration, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			http.Error(w, "Invalid duration format (use 1h, 30m)", http.StatusBadRequest)
			return
		}
		if duration > 24*time.Hour {    // We set the max limit to 24 hours for security reasons
			http.Error(w, "Max expiry is 24 hours", http.StatusBadRequest)
			return
		}
		expiresAt := time.Now().Add(duration)

		// 2. Insert into Database
		var dropID string
		query := `
			INSERT INTO drops (file_name, file_size, encryption_salt, expires_at, max_downloads)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`
		
		err = s.DB.QueryRow(query, req.FileName, req.FileSize, req.EncryptionSalt, expiresAt, req.MaxDownloads).Scan(&dropID)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 3. Return the Drop ID
		resp := CreateDropResponse{
			DropID:    dropID,
			ExpiresAt: expiresAt,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
