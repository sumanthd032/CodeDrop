package api

import (
	"crypto/sha256"
	"encoding/hex"
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
		
		chunkIndex := r.Header.Get("X-Chunk-Index")
		if chunkIndex == "" {
			http.Error(w, "Missing X-Chunk-Index header", http.StatusBadRequest)
			return
		}

		const MaxChunkSize = 5 * 1024 * 1024
		data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxChunkSize))
		if err != nil {
			http.Error(w, "Chunk too large or read error", http.StatusRequestEntityTooLarge)
			return
		}

		// 1. Calculate the SHA-256 Hash of the chunk (CONTENT-ADDRESSED STORAGE)
		hash := sha256.Sum256(data)
		chunkHash := hex.EncodeToString(hash[:])

		// 2. Generate CAS key for S3
		s3Key := fmt.Sprintf("chunks/%s", chunkHash)

		// 3. Upload to S3
		// Because it's CAS, if the file already exists, overwriting it is harmless 
		// (it's the exact same data). In a super-optimized system, we'd check if it exists first.
		if err := s.Store.UploadChunk(s3Key, data); err != nil {
			http.Error(w, "Storage failure: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 4. Record chunk metadata in Postgres
		// We now store the ACTUAL hash instead of "placeholder_hash"
		_, err = s.DB.Exec(`
			INSERT INTO chunks (drop_id, chunk_index, chunk_hash, size)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (drop_id, chunk_index) DO NOTHING`,
			dropID, chunkIndex, chunkHash, len(data))
		
		if err != nil {
			http.Error(w, "Metadata failure: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "uploaded",
			"hash":   chunkHash,
		})
	}
}