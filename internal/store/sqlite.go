package store

import (
	"database/sql"
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pbaille/kb/internal/domain"
)

//go:embed schema.sql
var schema string

// Store handles database operations
type Store struct {
	db *sql.DB
}

// New creates a new Store with the given database path
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Initialize schema
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// AddEntry creates a new entry and returns it
func (s *Store) AddEntry(content string) (*domain.Entry, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := s.db.Exec(
		"INSERT INTO entries (id, content, created_at) VALUES (?, ?, ?)",
		id, content, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert entry: %w", err)
	}

	return &domain.Entry{
		ID:        id,
		Content:   content,
		CreatedAt: now,
	}, nil
}

// DeleteEntry removes an entry by ID
func (s *Store) DeleteEntry(id string) error {
	result, err := s.db.Exec("DELETE FROM entries WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check delete result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("entry not found")
	}

	return nil
}

// GetEntry retrieves an entry by ID with its tags
func (s *Store) GetEntry(id string) (*domain.Entry, error) {
	var entry domain.Entry
	err := s.db.QueryRow(
		"SELECT id, content, created_at, last_viewed_at FROM entries WHERE id = ?",
		id,
	).Scan(&entry.ID, &entry.Content, &entry.CreatedAt, &entry.LastViewedAt)
	if err != nil {
		return nil, fmt.Errorf("get entry: %w", err)
	}

	// Get associated tags
	tags, err := s.GetEntryTags(id)
	if err != nil {
		return nil, err
	}
	entry.Tags = tags

	return &entry, nil
}

// ListEntries returns recent entries with pagination
func (s *Store) ListEntries(limit, offset int) ([]domain.Entry, error) {
	rows, err := s.db.Query(
		"SELECT id, content, created_at, last_viewed_at FROM entries ORDER BY created_at DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()

	var entries []domain.Entry
	for rows.Next() {
		var e domain.Entry
		if err := rows.Scan(&e.ID, &e.Content, &e.CreatedAt, &e.LastViewedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, nil
}

// GetOrCreateTag finds a tag by name or creates it
func (s *Store) GetOrCreateTag(name string, parentID *string) (*domain.Tag, error) {
	// Try to find existing tag
	var tag domain.Tag
	err := s.db.QueryRow(
		"SELECT id, name, parent_id, created_at FROM tags WHERE name = ?",
		name,
	).Scan(&tag.ID, &tag.Name, &tag.ParentID, &tag.CreatedAt)

	if err == nil {
		return &tag, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("find tag: %w", err)
	}

	// Create new tag
	id := uuid.New().String()
	now := time.Now()

	_, err = s.db.Exec(
		"INSERT INTO tags (id, name, parent_id, created_at) VALUES (?, ?, ?, ?)",
		id, name, parentID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert tag: %w", err)
	}

	return &domain.Tag{
		ID:        id,
		Name:      name,
		ParentID:  parentID,
		CreatedAt: now,
	}, nil
}

// LinkEntryTag associates a tag with an entry
func (s *Store) LinkEntryTag(entryID, tagID string, confidence float64) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO entry_tags (entry_id, tag_id, confidence) VALUES (?, ?, ?)",
		entryID, tagID, confidence,
	)
	if err != nil {
		return fmt.Errorf("link entry tag: %w", err)
	}
	return nil
}

// GetEntryTags returns all tags for an entry
func (s *Store) GetEntryTags(entryID string) ([]domain.Tag, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.name, t.parent_id, t.created_at
		FROM tags t
		JOIN entry_tags et ON t.id = et.tag_id
		WHERE et.entry_id = ?
	`, entryID)
	if err != nil {
		return nil, fmt.Errorf("get entry tags: %w", err)
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.ParentID, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}

	return tags, nil
}

// ListTags returns all tags
func (s *Store) ListTags() ([]domain.Tag, error) {
	rows, err := s.db.Query(
		"SELECT id, name, parent_id, created_at FROM tags ORDER BY name",
	)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.ParentID, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}

	return tags, nil
}

// FindSimilar finds entries sharing tags with the given entry, excluding the entry itself
func (s *Store) FindSimilar(entryID string, limit int) ([]domain.Entry, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT e.id, e.content, e.created_at, e.last_viewed_at
		FROM entries e
		JOIN entry_tags et ON e.id = et.entry_id
		WHERE et.tag_id IN (
			SELECT tag_id FROM entry_tags WHERE entry_id = ?
		)
		AND e.id != ?
		ORDER BY e.last_viewed_at ASC NULLS FIRST, e.created_at DESC
		LIMIT ?
	`, entryID, entryID, limit)
	if err != nil {
		return nil, fmt.Errorf("find similar: %w", err)
	}
	defer rows.Close()

	var entries []domain.Entry
	for rows.Next() {
		var e domain.Entry
		if err := rows.Scan(&e.ID, &e.Content, &e.CreatedAt, &e.LastViewedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// GetSuggestions returns entries the user hasn't viewed recently
func (s *Store) GetSuggestions(limit int) ([]domain.Entry, error) {
	rows, err := s.db.Query(`
		SELECT id, content, created_at, last_viewed_at
		FROM entries
		ORDER BY last_viewed_at ASC NULLS FIRST, created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("get suggestions: %w", err)
	}
	defer rows.Close()

	var entries []domain.Entry
	for rows.Next() {
		var e domain.Entry
		if err := rows.Scan(&e.ID, &e.Content, &e.CreatedAt, &e.LastViewedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// SearchEntries performs a simple text search
func (s *Store) SearchEntries(query string) ([]domain.Entry, error) {
	rows, err := s.db.Query(
		"SELECT id, content, created_at, last_viewed_at FROM entries WHERE content LIKE ? ORDER BY created_at DESC",
		"%"+query+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("search entries: %w", err)
	}
	defer rows.Close()

	var entries []domain.Entry
	for rows.Next() {
		var e domain.Entry
		if err := rows.Scan(&e.ID, &e.Content, &e.CreatedAt, &e.LastViewedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, nil
}

// SaveEmbedding stores an embedding vector for an entry
func (s *Store) SaveEmbedding(entryID string, vector []float64, model string) error {
	blob := vectorToBlob(vector)
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO embeddings (entry_id, vector, model, created_at) VALUES (?, ?, ?, ?)",
		entryID, blob, model, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("save embedding: %w", err)
	}
	return nil
}

// SimilarEntry represents an entry with a similarity score
type SimilarEntry struct {
	Entry      domain.Entry `json:"entry"`
	Similarity float64      `json:"similarity"`
}

// FindSimilar returns entries most similar to the given vector
func (s *Store) FindSimilar(vector []float64, limit int, excludeID string) ([]SimilarEntry, error) {
	rows, err := s.db.Query(`
		SELECT e.id, e.content, e.created_at, em.vector
		FROM entries e
		JOIN embeddings em ON e.id = em.entry_id
		WHERE e.id != ?
	`, excludeID)
	if err != nil {
		return nil, fmt.Errorf("find similar: %w", err)
	}
	defer rows.Close()

	var results []SimilarEntry
	for rows.Next() {
		var e domain.Entry
		var blob []byte
		if err := rows.Scan(&e.ID, &e.Content, &e.CreatedAt, &blob); err != nil {
			return nil, fmt.Errorf("scan similar: %w", err)
		}

		storedVec := blobToVector(blob)
		sim := cosineSimilarity(vector, storedVec)

		results = append(results, SimilarEntry{Entry: e, Similarity: sim})
	}

	// Sort by similarity descending
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[i].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func vectorToBlob(v []float64) []byte {
	buf := make([]byte, len(v)*8)
	for i, f := range v {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(f))
	}
	return buf
}

func blobToVector(b []byte) []float64 {
	v := make([]float64, len(b)/8)
	for i := range v {
		v[i] = math.Float64frombits(binary.LittleEndian.Uint64(b[i*8:]))
	}
	return v
}

func cosineSimilarity(a, b []float64) float64 {
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
