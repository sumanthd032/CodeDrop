package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// handleGetDropMetadata returns info about the file (name, size, salt)
// The CLI needs this *before* it starts downloading to set up decryption.
func (s *Server) handleGetDropMetadata() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dropID := chi.URLParam(r, "id")

		var resp GetDropMetadataResponse
		var expiresAt time.Time

		// 1. Fetch metadata from Postgres
		query := `
			SELECT file_name, file_size, encryption_salt, expires_at 
			FROM drops WHERE id = $1`
		
		err := s.DB.QueryRow(query, dropID).Scan(
			&resp.FileName, &resp.FileSize, &resp.EncryptionSalt, &expiresAt,
		)

		if err != nil {
			http.Error(w, "Drop not found", http.StatusNotFound)
			return
		}

		// 2. Check Expiry
		if time.Now().After(expiresAt) {
			http.Error(w, "Drop has expired", http.StatusGone)
			return
		}

		// 3. Get chunk count (to know how many pieces to expect)
		err = s.DB.QueryRow("SELECT COUNT(*) FROM chunks WHERE drop_id = $1", dropID).Scan(&resp.ChunkCount)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// handleDownloadChunk retrieves a specific piece of binary data
func (s *Server) handleDownloadChunk() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dropID := chi.URLParam(r, "id")
		chunkIndex := chi.URLParam(r, "chunkIndex") // We'll add this param to router

		// 1. Construct S3 Key
		key := fmt.Sprintf("drops/%s/%s", dropID, chunkIndex)

		// 2. Fetch from S3
		data, err := s.Store.DownloadChunk(key)
		if err != nil {
			http.Error(w, "Chunk not found or storage error", http.StatusNotFound)
			return
		}

		// 3. Stream binary back
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)
	}
}