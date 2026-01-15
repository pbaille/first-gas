package store

import (
	"database/sql"
	_ "embed"
	"fmt"
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

// GetEntry retrieves an entry by ID with its tags
func (s *Store) GetEntry(id string) (*domain.Entry, error) {
	var entry domain.Entry
	err := s.db.QueryRow(
		"SELECT id, content, created_at FROM entries WHERE id = ?",
		id,
	).Scan(&entry.ID, &entry.Content, &entry.CreatedAt)
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
		"SELECT id, content, created_at FROM entries ORDER BY created_at DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()

	var entries []domain.Entry
	for rows.Next() {
		var e domain.Entry
		if err := rows.Scan(&e.ID, &e.Content, &e.CreatedAt); err != nil {
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

// SearchEntries performs a simple text search
func (s *Store) SearchEntries(query string) ([]domain.Entry, error) {
	rows, err := s.db.Query(
		"SELECT id, content, created_at FROM entries WHERE content LIKE ? ORDER BY created_at DESC",
		"%"+query+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("search entries: %w", err)
	}
	defer rows.Close()

	var entries []domain.Entry
	for rows.Next() {
		var e domain.Entry
		if err := rows.Scan(&e.ID, &e.Content, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, nil
}
