package domain

import "time"

// Entry represents a captured piece of content
type Entry struct {
	ID           string     `json:"id"`
	Content      string     `json:"content"`
	Tags         []Tag      `json:"tags,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	LastViewedAt *time.Time `json:"last_viewed_at,omitempty"`
}

// Tag represents a classification label with optional hierarchy
type Tag struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// EntryTag represents the relationship between an entry and a tag
type EntryTag struct {
	EntryID    string  `json:"entry_id"`
	TagID      string  `json:"tag_id"`
	Confidence float64 `json:"confidence"`
}
