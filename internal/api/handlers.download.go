package api

import (
	"crypto/sha256"
	"encoding/hex"
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
		var maxDownloads int 

		// 1. Fetch metadata from Postgres (added max_downloads to the query)
		query := `
			SELECT file_name, file_size, encryption_salt, expires_at, max_downloads 
			FROM drops WHERE id = $1`
		
		err := s.DB.QueryRow(query, dropID).Scan(
			&resp.FileName, &resp.FileSize, &resp.EncryptionSalt, &expiresAt, &maxDownloads,
		)

		if err != nil {
			http.Error(w, "Drop not found", http.StatusNotFound)
			return
		}

		// 2. Check Time Expiry
		if time.Now().After(expiresAt) {
			http.Error(w, "Drop has expired", http.StatusGone)
			return
		}

		// 3. Atomic download count check using Redis
		allowed, err := s.Cache.IncrementAndCheck(r.Context(), dropID, maxDownloads)
		if err != nil {
			http.Error(w, "Internal server error checking limits", http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "Download limit reached", http.StatusGone)
			return
		}

		// 4. Get chunk count
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
		chunkIndex := chi.URLParam(r, "chunkIndex")

		// 1. Look up the hash from Postgres
		var chunkHash string
		err := s.DB.QueryRow(`
			SELECT chunk_hash FROM chunks 
			WHERE drop_id = $1 AND chunk_index = $2`, 
			dropID, chunkIndex).Scan(&chunkHash)
			
		if err != nil {
			http.Error(w, "Chunk metadata not found", http.StatusNotFound)
			return
		}

		// 2. Construct CAS S3 Key
		key := fmt.Sprintf("chunks/%s", chunkHash)

		// 3. Fetch from S3
		data, err := s.Store.DownloadChunk(key)
		if err != nil {
			http.Error(w, "Chunk data not found in storage", http.StatusNotFound)
			return
		}

		// 4. Integrity Check: Verify the data hasn't been corrupted in S3!
		downloadedHash := sha256.Sum256(data)
		if hex.EncodeToString(downloadedHash[:]) != chunkHash {
			// If S3 flipped a bit, we detect it instantly and refuse to serve bad data.
			http.Error(w, "CRITICAL: Data integrity verification failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)
	}
}