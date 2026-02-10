package api

import "time"

// CreateDropRequest is what the CLI sends to start an upload
type CreateDropRequest struct {
	FileName       string `json:"file_name"`
	FileSize       int64  `json:"file_size"`
	EncryptionSalt string `json:"encryption_salt"` // The salt used for client-side encryption
	ExpiresIn      string `json:"expires_in"`      // e.g., "1h", "30m"
	MaxDownloads   int    `json:"max_downloads"`
}

// CreateDropResponse is what the server sends back
type CreateDropResponse struct {
	DropID    string    `json:"drop_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ChunkUploadResponse confirms a chunk was saved
type ChunkUploadResponse struct {
	ChunkIndex int    `json:"chunk_index"`
	Status     string `json:"status"`
}

// The flow
// 1. CLI sends CreateDropRequest to /api/v1/drops (POST)
// 2. Server responds with CreateDropResponse (drop ID and expiration time)
// 3. CLI uploads file chunks to /api/v1/drops/{drop_id}/chunks (POST) with ChunkUploadResponse confirming each chunk

