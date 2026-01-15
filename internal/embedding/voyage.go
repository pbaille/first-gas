package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
)

const voyageAPI = "https://api.voyageai.com/v1/embeddings"

// Service handles embedding generation via Voyage AI
type Service struct {
	apiKey string
	model  string
}

// New creates a new embedding Service
func New() (*Service, error) {
	apiKey := os.Getenv("VOYAGE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("VOYAGE_API_KEY environment variable not set")
	}

	return &Service{
		apiKey: apiKey,
		model:  "voyage-3-lite",
	}, nil
}

// Embed generates an embedding vector for the given text
func (s *Service) Embed(text string) ([]float64, error) {
	vectors, err := s.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	return vectors[0], nil
}

// EmbedBatch generates embeddings for multiple texts
func (s *Service) EmbedBatch(texts []string) ([][]float64, error) {
	reqBody := embeddingRequest{
		Input: texts,
		Model: s.model,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", voyageAPI, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp embeddingResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	vectors := make([][]float64, len(apiResp.Data))
	for i, d := range apiResp.Data {
		vectors[i] = d.Embedding
	}

	return vectors, nil
}

// CosineSimilarity computes similarity between two vectors
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

type embeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}
