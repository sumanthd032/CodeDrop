package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// We redefine the models here to keep the CLI decoupled from the Server package
type CreateDropRequest struct {
	FileName       string `json:"file_name"`
	FileSize       int64  `json:"file_size"`
	EncryptionSalt string `json:"encryption_salt"`
	ExpiresIn      string `json:"expires_in"`
	MaxDownloads   int    `json:"max_downloads"`
}

type CreateDropResponse struct {
	DropID    string    `json:"drop_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateDrop initializes the upload session
func (c *APIClient) CreateDrop(req CreateDropRequest) (*CreateDropResponse, error) {
	body, _ := json.Marshal(req)
	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/v1/drop", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(msg))
	}

	var dropResp CreateDropResponse
	if err := json.NewDecoder(resp.Body).Decode(&dropResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &dropResp, nil
}

// UploadChunk sends a single encrypted binary chunk
func (c *APIClient) UploadChunk(dropID string, chunkIndex int, data []byte) error {
	url := fmt.Sprintf("%s/api/v1/drop/%s/chunk", c.BaseURL, dropID)
	
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	
	// Set our custom header so the server knows which piece this is
	req.Header.Set("X-Chunk-Index", fmt.Sprintf("%d", chunkIndex))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(msg))
	}

	return nil
}