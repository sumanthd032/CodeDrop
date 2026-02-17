package api

import (
	"encoding/json"
	"net/http"
)

// handleGetStats calculates and returns system metrics
func (s *Server) handleGetStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var resp StatsResponse

		// 1. Count Active Drops (Not expired)
		err := s.DB.QueryRow("SELECT COUNT(*) FROM drops WHERE expires_at > NOW()").Scan(&resp.ActiveDrops)
		if err != nil {
			http.Error(w, "Database error counting drops", http.StatusInternalServerError)
			return
		}

		// 2. Count Total Unique Chunks (The number of objects actually in S3)
		err = s.DB.QueryRow("SELECT COUNT(DISTINCT chunk_hash) FROM chunks").Scan(&resp.TotalChunks)
		if err != nil {
			http.Error(w, "Database error counting chunks", http.StatusInternalServerError)
			return
		}

		// 3. Calculate Actual Storage Used (Sum of DISTINCT chunk sizes)
		// COALESCE ensures we get 0 instead of NULL if the database is empty
		err = s.DB.QueryRow(`
			SELECT COALESCE(SUM(size), 0) 
			FROM (SELECT DISTINCT chunk_hash, size FROM chunks) AS unique_chunks
		`).Scan(&resp.StorageUsed)
		if err != nil {
			http.Error(w, "Database error calculating storage", http.StatusInternalServerError)
			return
		}

		// 4. Calculate Intended Storage (If CAS was NOT implemented)
		var intendedStorage int64
		err = s.DB.QueryRow("SELECT COALESCE(SUM(size), 0) FROM chunks").Scan(&intendedStorage)
		if err != nil {
			http.Error(w, "Database error calculating intended storage", http.StatusInternalServerError)
			return
		}

		// 5. Calculate Savings!
		resp.StorageSaved = intendedStorage - resp.StorageUsed

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}