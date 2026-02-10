package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"io"

	"github.com/go-chi/chi/v5"
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



// handleUploadChunk receives a binary piece of the file
func (s *Server) handleUploadChunk() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dropID := chi.URLParam(r, "id")
		
		// 1. Read the chunk index from the header
		// We use a custom header because the body is pure binary data
		chunkIndex := r.Header.Get("X-Chunk-Index")
		if chunkIndex == "" {
			http.Error(w, "Missing X-Chunk-Index header", http.StatusBadRequest)
			return
		}

		// 2. Read the body (the binary data)
		// Limit reader protects us from someone sending a 10GB chunk
		const MaxChunkSize = 5 * 1024 * 1024 // 5MB limit per chunk
		data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxChunkSize))
		if err != nil {
			http.Error(w, "Chunk too large or read error", http.StatusRequestEntityTooLarge)
			return
		}

		// 3. Generate a unique key for S3
		// Key format: drops/<drop_id>/<chunk_index>
		s3Key := fmt.Sprintf("drops/%s/%s", dropID, chunkIndex)

		// 4. Upload to S3 (MinIO)
		if err := s.Store.UploadChunk(s3Key, data); err != nil {
			http.Error(w, "Storage failure: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 5. Record chunk metadata in Postgres
		// We store the hash later for integrity, for now just track presence
		_, err = s.DB.Exec(`
			INSERT INTO chunks (drop_id, chunk_index, chunk_hash, size)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (drop_id, chunk_index) DO NOTHING`, // If the same chunk is re-uploaded, we ignore it (idempotent)
			dropID, chunkIndex, "placeholder_hash", len(data))
		
		if err != nil {
			// If DB fails, we should ideally delete the S3 object, but for now just error
			http.Error(w, "Metadata failure: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "uploaded"})
	}
}
