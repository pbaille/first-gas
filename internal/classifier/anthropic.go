package classifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const anthropicAPI = "https://api.anthropic.com/v1/messages"

// TagSuggestion represents a suggested tag with optional parent
type TagSuggestion struct {
	Name       string  `json:"name"`
	Parent     string  `json:"parent,omitempty"`
	Confidence float64 `json:"confidence"`
}

// ClassifyResult holds the classification output
type ClassifyResult struct {
	Tags []TagSuggestion `json:"tags"`
}

// Classifier handles content classification via Anthropic API
type Classifier struct {
	apiKey string
	model  string
}

// New creates a new Classifier
func New() (*Classifier, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	return &Classifier{
		apiKey: apiKey,
		model:  "claude-sonnet-4-20250514",
	}, nil
}

// Classify analyzes content and returns tag suggestions
func (c *Classifier) Classify(content string, existingTags []string) (*ClassifyResult, error) {
	prompt := buildPrompt(content, existingTags)

	resp, err := c.callAPI(prompt)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}

	return parseResponse(resp)
}

func buildPrompt(content string, existingTags []string) string {
	var sb strings.Builder

	sb.WriteString("Classify this content and suggest tags. Return JSON only.\n\n")
	sb.WriteString("Content:\n")
	sb.WriteString(content)
	sb.WriteString("\n\n")

	if len(existingTags) > 0 {
		sb.WriteString("Existing tags in the system (prefer reusing these when appropriate):\n")
		for _, tag := range existingTags {
			sb.WriteString("- ")
			sb.WriteString(tag)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`Return a JSON object with this structure:
{
  "tags": [
    {"name": "tag-name", "parent": "parent-tag-or-empty", "confidence": 0.9}
  ]
}

Rules:
- Use lowercase, hyphenated tag names (e.g., "machine-learning" not "Machine Learning")
- Suggest 2-5 relevant tags
- Use "parent" to build hierarchy (e.g., {"name": "golang", "parent": "programming"})
- Confidence is 0.0-1.0 based on how certain the classification is
- Reuse existing tags when they fit; create new ones when needed
- Keep tags general enough to be reusable across entries

Return ONLY the JSON, no other text.`)

	return sb.String()
}

type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	Messages  []apiMessage `json:"messages"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Classifier) callAPI(prompt string) (string, error) {
	reqBody := apiRequest{
		Model:     c.model,
		MaxTokens: 1024,
		Messages: []apiMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", anthropicAPI, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("api error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return apiResp.Content[0].Text, nil
}

func parseResponse(resp string) (*ClassifyResult, error) {
	// Clean up response - remove markdown code blocks if present
	resp = strings.TrimSpace(resp)
	resp = strings.TrimPrefix(resp, "```json")
	resp = strings.TrimPrefix(resp, "```")
	resp = strings.TrimSuffix(resp, "```")
	resp = strings.TrimSpace(resp)

	var result ClassifyResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("parse json: %w (response: %s)", err, resp)
	}

	return &result, nil
}
