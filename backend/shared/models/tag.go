package models

import (
	"time"

	"github.com/google/uuid"
)

// Tag represents a tag in the system (matches database schema)
type Tag struct {
	ID            uuid.UUID `json:"id" db:"id"`
	Name          string    `json:"name" db:"name" validate:"required,min=1,max=100"`
	CanonicalName *string   `json:"canonical_name" db:"canonical_name"` // Points to canonical tag name if this is a synonym
	Type          string    `json:"type" db:"type" validate:"oneof=fandom character relationship freeform warning category rating additional"`
	Description   *string   `json:"description" db:"description"`
	IsCanonical   bool      `json:"is_canonical" db:"is_canonical"`   // Whether this tag is canonical
	IsFilterable  bool      `json:"is_filterable" db:"is_filterable"` // Whether this tag can be used for filtering
	UseCount      int       `json:"use_count" db:"use_count"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// TagRelationship represents relationships between tags (matches database schema)
type TagRelationship struct {
	ParentTagID      uuid.UUID  `json:"parent_tag_id" db:"parent_tag_id"`
	ChildTagID       uuid.UUID  `json:"child_tag_id" db:"child_tag_id"`
	RelationshipType string     `json:"relationship_type" db:"relationship_type" validate:"oneof=parent_child synonym related"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	CreatedBy        *uuid.UUID `json:"created_by" db:"created_by"`
}

// TagSynonym represents alternative names for tags
type TagSynonym struct {
	ID          uuid.UUID `json:"id" db:"id"`
	CanonicalID uuid.UUID `json:"canonical_id" db:"canonical_id"`
	SynonymName string    `json:"synonym_name" db:"synonym_name" validate:"required,min=1,max=100"`
	CreatedBy   uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// TagWrangling represents the wrangling status of tags
type TagWrangling struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TagID        uuid.UUID  `json:"tag_id" db:"tag_id"`
	Status       string     `json:"status" db:"status" validate:"oneof=unwrangled wrangled disputed banned"`
	WrangledBy   *uuid.UUID `json:"wrangled_by" db:"wrangled_by"`
	WrangledAt   *time.Time `json:"wrangled_at" db:"wrangled_at"`
	ReviewedBy   *uuid.UUID `json:"reviewed_by" db:"reviewed_by"`
	ReviewedAt   *time.Time `json:"reviewed_at" db:"reviewed_at"`
	Notes        string     `json:"notes" db:"notes"`
	ReportsCount int        `json:"reports_count" db:"reports_count"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// TagMerge represents a request to merge tags
type TagMerge struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	FromTagID   uuid.UUID  `json:"from_tag_id" db:"from_tag_id"`
	ToTagID     uuid.UUID  `json:"to_tag_id" db:"to_tag_id"`
	RequestedBy uuid.UUID  `json:"requested_by" db:"requested_by"`
	Status      string     `json:"status" db:"status" validate:"oneof=pending approved rejected"`
	ReviewedBy  *uuid.UUID `json:"reviewed_by" db:"reviewed_by"`
	ReviewedAt  *time.Time `json:"reviewed_at" db:"reviewed_at"`
	Reason      string     `json:"reason" db:"reason"`
	Notes       string     `json:"notes" db:"notes"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// Fandom represents a fandom/media source
type Fandom struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name" validate:"required,min=1,max=200"`
	Canonical   string    `json:"canonical" db:"canonical"` // Canonical name
	Type        string    `json:"type" db:"type" validate:"oneof=tv movie book game comic anime manga podcast other"`
	External    bool      `json:"external" db:"external"` // From external database
	WorkCount   int       `json:"work_count" db:"work_count"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Character represents a character within a fandom
type Character struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	Name      string     `json:"name" db:"name" validate:"required,min=1,max=100"`
	Canonical string     `json:"canonical" db:"canonical"`
	FandomID  *uuid.UUID `json:"fandom_id" db:"fandom_id"`
	WorkCount int        `json:"work_count" db:"work_count"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// Relationship represents a relationship between characters
type Relationship struct {
	ID         uuid.UUID   `json:"id" db:"id"`
	Name       string      `json:"name" db:"name" validate:"required,min=1,max=200"`
	Canonical  string      `json:"canonical" db:"canonical"`
	Type       string      `json:"type" db:"type" validate:"oneof=romantic platonic family other"`
	Characters []uuid.UUID `json:"characters" db:"characters"` // Array of character IDs
	FandomID   *uuid.UUID  `json:"fandom_id" db:"fandom_id"`
	WorkCount  int         `json:"work_count" db:"work_count"`
	CreatedAt  time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at" db:"updated_at"`
}

// TagSearchRequest represents a tag search query
type TagSearchRequest struct {
	Query    string   `json:"query"`
	Type     []string `json:"type,omitempty"` // Filter by tag types
	Fandom   []string `json:"fandom,omitempty"`
	Common   *bool    `json:"common,omitempty"`
	External *bool    `json:"external,omitempty"`
	Limit    int      `json:"limit,omitempty"`
	Offset   int      `json:"offset,omitempty"`
}

// TagSearchResult represents search results
type TagSearchResult struct {
	Tags  []Tag `json:"tags"`
	Total int   `json:"total"`
}

// CreateTagRequest represents a request to create a new tag
type CreateTagRequest struct {
	Name          string  `json:"name" validate:"required,min=1,max=100"`
	Type          string  `json:"type" validate:"required,oneof=fandom character relationship freeform warning category rating additional"`
	CanonicalName *string `json:"canonical_name,omitempty"`
	Description   *string `json:"description,omitempty"`
	IsCanonical   bool    `json:"is_canonical,omitempty"`
	IsFilterable  bool    `json:"is_filterable,omitempty"`
}

// WrangleTagRequest represents a request to wrangle a tag
type WrangleTagRequest struct {
	Status     string      `json:"status" validate:"required,oneof=unwrangled wrangled disputed banned"`
	Canonical  *uuid.UUID  `json:"canonical,omitempty"`
	Synonyms   []string    `json:"synonyms,omitempty"`
	ParentTags []uuid.UUID `json:"parent_tags,omitempty"`
	ChildTags  []uuid.UUID `json:"child_tags,omitempty"`
	Notes      string      `json:"notes,omitempty"`
}

// TagAutoCompleteRequest represents an autocomplete request
type TagAutoCompleteRequest struct {
	Query      string      `json:"query" validate:"required,min=1"`
	Type       []string    `json:"type,omitempty"`
	FandomID   *uuid.UUID  `json:"fandom_id,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	ExcludeIDs []uuid.UUID `json:"exclude_ids,omitempty"`
}

// TagAutoCompleteResponse represents autocomplete results
type TagAutoCompleteResponse struct {
	Suggestions []TagSuggestion `json:"suggestions"`
}

// TagSuggestion represents a single tag suggestion
type TagSuggestion struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	UseCount  int       `json:"use_count"`
	Canonical bool      `json:"canonical"`
}
