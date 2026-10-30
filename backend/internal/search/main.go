// Package search finds works, using Postgres full-text search.
//
// There's no separate search index: queries read the works table directly,
// through the search_vector column that Postgres keeps current itself
// (migration 028). A work is findable the moment it's posted and gone the
// moment it's deleted or hidden, and there's nothing to rebuild.
package search

import (
	"database/sql"

	"nuclear-ao3/internal/platform"

	"github.com/gin-gonic/gin"
)

// Service answers search requests.
type Service struct {
	db *sql.DB
}

// New builds the search service.
func New(db *sql.DB) *Service {
	return &Service{db: db}
}

// Mount registers the search routes.
func Mount(r *gin.Engine, s *Service, d *platform.Deps) {
	g := r.Group("/api/v1/search")
	g.Use(d.OptionalUser) // logged-in readers also see works restricted to users
	g.GET("/works", s.SearchWorks)
	g.GET("/suggestions", s.Suggestions)
}
