package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Fetch retrieves URL content and extracts readable text
func Fetch(rawURL string) (string, error) {
	// Validate URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	// Fetch with timeout
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "kb/1.0 (knowledge-base)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read body with size limit (5MB)
	limited := io.LimitReader(resp.Body, 5*1024*1024)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	// Extract text from HTML
	text := extractText(string(body))
	if text == "" {
		return "", fmt.Errorf("no text content found")
	}

	return text, nil
}

// IsURL checks if a string looks like a URL
func IsURL(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "www.")
}

// extractText parses HTML and returns readable text content
func extractText(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return ""
	}

	var sb strings.Builder
	var extract func(*html.Node)

	// Tags to skip (non-content)
	skipTags := map[string]bool{
		"script": true, "style": true, "nav": true,
		"header": true, "footer": true, "aside": true,
		"noscript": true, "iframe": true,
	}

	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && skipTags[n.Data] {
			return
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				sb.WriteString(text)
				sb.WriteString(" ")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}

		// Add newlines after block elements
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "li", "br":
				sb.WriteString("\n")
			}
		}
	}

	extract(doc)

	// Clean up: collapse whitespace, trim
	result := sb.String()
	result = strings.Join(strings.Fields(result), " ")

	// Truncate if too long (keep first 10KB of text)
	if len(result) > 10*1024 {
		result = result[:10*1024] + "..."
	}

	return strings.TrimSpace(result)
}
