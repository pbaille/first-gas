package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pbaille/kb/internal/classifier"
	"github.com/pbaille/kb/internal/domain"
	"github.com/pbaille/kb/internal/embedding"
	"github.com/pbaille/kb/internal/store"
)

// Server handles HTTP requests for the knowledge base API
type Server struct {
	store *store.Store
	addr  string
}

// New creates a new API server
func New(s *store.Store, addr string) *Server {
	return &Server{store: s, addr: addr}
}

// Run starts the HTTP server
func (s *Server) Run() error {
	mux := http.NewServeMux()

	// Entries
	mux.HandleFunc("GET /entries", s.listEntries)
	mux.HandleFunc("POST /entries", s.addEntry)
	mux.HandleFunc("GET /entries/{id}", s.getEntry)

	// Tags
	mux.HandleFunc("GET /tags", s.listTags)

	// Search
	mux.HandleFunc("GET /search", s.searchEntries)

	// Health check
	mux.HandleFunc("GET /health", s.health)

	fmt.Printf("Starting server on %s\n", s.addr)
	return http.ListenAndServe(s.addr, withCORS(mux))
}

// withCORS adds CORS headers for frontend development
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// AddEntryRequest is the request body for adding an entry
type AddEntryRequest struct {
	Content    string `json:"content"`
	NoClassify bool   `json:"no_classify,omitempty"`
}

// AddEntryResponse is the response for adding an entry
type AddEntryResponse struct {
	Entry   *domain.Entry        `json:"entry"`
	Tags    []TagWithParent      `json:"tags,omitempty"`
	Similar []store.SimilarEntry `json:"similar,omitempty"`
}

// TagWithParent includes parent info for API response
type TagWithParent struct {
	Name       string  `json:"name"`
	Parent     string  `json:"parent,omitempty"`
	Confidence float64 `json:"confidence"`
}

func (s *Server) addEntry(w http.ResponseWriter, r *http.Request) {
	var req AddEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	entry, err := s.store.AddEntry(req.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := AddEntryResponse{Entry: entry}

	// Classify unless disabled
	if !req.NoClassify {
		clf, err := classifier.New()
		if err == nil {
			existingTags, _ := s.store.ListTags()
			tagNames := make([]string, len(existingTags))
			for i, t := range existingTags {
				tagNames[i] = t.Name
			}

			result, err := clf.Classify(req.Content, tagNames)
			if err == nil {
				for _, suggestion := range result.Tags {
					var parentID *string

					if suggestion.Parent != "" {
						parentTag, err := s.store.GetOrCreateTag(suggestion.Parent, nil)
						if err == nil {
							parentID = &parentTag.ID
						}
					}

					tag, err := s.store.GetOrCreateTag(suggestion.Name, parentID)
					if err != nil {
						continue
					}

					s.store.LinkEntryTag(entry.ID, tag.ID, suggestion.Confidence)

					resp.Tags = append(resp.Tags, TagWithParent{
						Name:       suggestion.Name,
						Parent:     suggestion.Parent,
						Confidence: suggestion.Confidence,
					})
				}

				// Refresh entry with tags
				entry, _ = s.store.GetEntry(entry.ID)
				resp.Entry = entry
			}
		}
	}

	// Compute embedding and find similar entries
	if embSvc, err := embedding.New(); err == nil {
		if vector, err := embSvc.Embed(req.Content); err == nil {
			// Find similar before saving (so we don't match ourselves)
			similar, _ := s.store.FindSimilar(vector, 5, entry.ID)
			resp.Similar = similar

			// Save embedding for future similarity searches
			s.store.SaveEmbedding(entry.ID, vector, "voyage-3-lite")
		}
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) getEntry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Support prefix matching
	entries, err := s.store.ListEntries(100, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var fullID string
	for _, e := range entries {
		if strings.HasPrefix(e.ID, id) {
			fullID = e.ID
			break
		}
	}

	if fullID == "" {
		writeError(w, http.StatusNotFound, "entry not found")
		return
	}

	entry, err := s.store.GetEntry(fullID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) listEntries(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	entries, err := s.store.ListEntries(limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"limit":   limit,
		"offset":  offset,
	})
}

// TagNode represents a tag with its children for hierarchical display
type TagNode struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Children []TagNode `json:"children,omitempty"`
}

func (s *Server) listTags(w http.ResponseWriter, r *http.Request) {
	tags, err := s.store.ListTags()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build hierarchy
	tagMap := make(map[string]domain.Tag)
	children := make(map[string][]string)
	var rootIDs []string

	for _, t := range tags {
		tagMap[t.ID] = t
		if t.ParentID == nil {
			rootIDs = append(rootIDs, t.ID)
		} else {
			children[*t.ParentID] = append(children[*t.ParentID], t.ID)
		}
	}

	var buildNode func(id string) TagNode
	buildNode = func(id string) TagNode {
		t := tagMap[id]
		node := TagNode{ID: t.ID, Name: t.Name}
		for _, childID := range children[id] {
			node.Children = append(node.Children, buildNode(childID))
		}
		return node
	}

	var tree []TagNode
	for _, rootID := range rootIDs {
		tree = append(tree, buildNode(rootID))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tags": tree,
		"flat": tags,
	})
}

func (s *Server) searchEntries(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	entries, err := s.store.SearchEntries(query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"query":   query,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
