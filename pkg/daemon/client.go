package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient is a client for the daemon API
type APIClient struct {
	baseURL string
	client  *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(addr string) *APIClient {
	return &APIClient{
		baseURL: fmt.Sprintf("http://%s", addr),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetStatus gets the daemon status
func (c *APIClient) GetStatus() (*Status, error) {
	resp, err := c.client.Get(c.baseURL + "/api/v1/status")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// AddChartSubstitution adds a chart substitution
func (c *APIClient) AddChartSubstitution(original, localPath string) error {
	req := AddChartRequest{
		Original:  original,
		LocalPath: localPath,
	}

	return c.post("/api/v1/charts", req)
}

// AddImageSubstitution adds an image substitution
func (c *APIClient) AddImageSubstitution(original, replacement string) error {
	req := AddImageRequest{
		Original:    original,
		Replacement: replacement,
	}

	return c.post("/api/v1/images", req)
}

// RemoveChartSubstitution removes a chart substitution
func (c *APIClient) RemoveChartSubstitution(original string) error {
	req := RemoveChartRequest{
		Original: original,
	}

	return c.post("/api/v1/charts/remove", req)
}

// RemoveImageSubstitution removes an image substitution
func (c *APIClient) RemoveImageSubstitution(original string) error {
	req := RemoveImageRequest{
		Original: original,
	}

	return c.post("/api/v1/images/remove", req)
}

// GetSubstitutions gets all substitutions
func (c *APIClient) GetSubstitutions() (*SubstitutionsResponse, error) {
	resp, err := c.client.Get(c.baseURL + "/api/v1/substitutions")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var subs SubstitutionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&subs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &subs, nil
}

// Shutdown sends shutdown request to daemon
func (c *APIClient) Shutdown() error {
	return c.post("/api/v1/shutdown", nil)
}

// post sends a POST request
func (c *APIClient) post(path string, data interface{}) error {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	resp, err := c.client.Post(c.baseURL+path, "application/json", body)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("%s", errResp.Error)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var successResp SuccessResponse
	if err := json.NewDecoder(resp.Body).Decode(&successResp); err != nil {
		return nil // Success even if we can't decode response
	}

	return nil
}

// IsHealthy checks if the daemon is healthy
func (c *APIClient) IsHealthy() bool {
	resp, err := c.client.Get(c.baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
