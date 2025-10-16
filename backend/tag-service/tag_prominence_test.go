package main

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagProminenceDatabase(t *testing.T) {
	// Use test database - in real implementation this would be a test instance
	db := setupTestDB(t)
	defer db.Close()

	t.Run("Test work_tags prominence columns", func(t *testing.T) {
		// Create test work and tags
		workID := uuid.New()
		tagName := "Test Ship/Test Ship 2"

		// Insert test data
		_, err := db.Exec(`
			INSERT INTO works (id, title, summary, word_count, chapter_count) 
			VALUES ($1, 'Test Work', 'Test summary', 5000, 1)
		`, workID)
		require.NoError(t, err)

		_, err = db.Exec(`
			INSERT INTO tags (name, type) 
			VALUES ($1, 'relationship')
		`, tagName)
		require.NoError(t, err)

		_, err = db.Exec(`
			INSERT INTO work_tags (work_id, tag_name, prominence, prominence_score, auto_assigned) 
			VALUES ($1, $2, 'primary', 0.85, true)
		`, workID, tagName)
		require.NoError(t, err)

		// Test retrieval
		var prominence string
		var score float64
		var autoAssigned bool
		err = db.QueryRow(`
			SELECT prominence, prominence_score, auto_assigned 
			FROM work_tags 
			WHERE work_id = $1 AND tag_name = $2
		`, workID, tagName).Scan(&prominence, &score, &autoAssigned)

		require.NoError(t, err)
		assert.Equal(t, "primary", prominence)
		assert.Equal(t, 0.85, score)
		assert.True(t, autoAssigned)
	})

	t.Run("Test work_tag_summaries trigger", func(t *testing.T) {
		workID := uuid.New()

		// Insert test work
		_, err := db.Exec(`
			INSERT INTO works (id, title, summary, word_count, chapter_count) 
			VALUES ($1, 'Test Work 2', 'Test summary', 3000, 1)
		`, workID)
		require.NoError(t, err)

		// Insert test tags
		tags := []struct {
			name       string
			tagType    string
			prominence string
		}{
			{"Primary Ship", "relationship", "primary"},
			{"Secondary Ship", "relationship", "secondary"},
			{"Background Ship", "relationship", "micro"},
			{"Angst", "additional_tags", "secondary"},
		}

		for _, tag := range tags {
			// Insert tag if not exists
			_, err = db.Exec(`
				INSERT INTO tags (name, type) VALUES ($1, $2) 
				ON CONFLICT (name) DO NOTHING
			`, tag.name, tag.tagType)
			require.NoError(t, err)

			// Insert work_tag
			_, err = db.Exec(`
				INSERT INTO work_tags (work_id, tag_name, prominence, auto_assigned) 
				VALUES ($1, $2, $3, true)
			`, workID, tag.name, tag.prominence)
			require.NoError(t, err)
		}

		// Check if summary was created by trigger
		var primaryCount, secondaryCount, microCount, totalCount int
		err = db.QueryRow(`
			SELECT primary_relationship_count, secondary_relationship_count, 
			       micro_relationship_count, total_tag_count
			FROM work_tag_summaries 
			WHERE work_id = $1
		`, workID).Scan(&primaryCount, &secondaryCount, &microCount, &totalCount)

		require.NoError(t, err)
		assert.Equal(t, 1, primaryCount)   // Primary Ship
		assert.Equal(t, 1, secondaryCount) // Secondary Ship
		assert.Equal(t, 1, microCount)     // Background Ship
		assert.Equal(t, 4, totalCount)     // All tags
	})

	t.Run("Test tag_prominence_rules", func(t *testing.T) {
		// Insert test rule
		_, err := db.Exec(`
			INSERT INTO tag_prominence_rules (tag_name, tag_type, default_prominence, min_word_threshold)
			VALUES ('Gen', 'relationship', 'primary', 0)
		`)
		require.NoError(t, err)

		// Query rule
		var prominence string
		var threshold int
		err = db.QueryRow(`
			SELECT default_prominence, min_word_threshold 
			FROM tag_prominence_rules 
			WHERE tag_name = 'Gen'
		`).Scan(&prominence, &threshold)

		require.NoError(t, err)
		assert.Equal(t, "primary", prominence)
		assert.Equal(t, 0, threshold)
	})

	t.Run("Test tag_migration_batches tracking", func(t *testing.T) {
		// Insert migration batch
		var batchNumber int
		err := db.QueryRow(`
			INSERT INTO tag_migration_batches 
			(batch_number, works_total, migration_strategy, status)
			VALUES ((SELECT COALESCE(MAX(batch_number), 0) + 1 FROM tag_migration_batches), 
			        100, 'test_migration', 'running')
			RETURNING batch_number
		`).Scan(&batchNumber)
		require.NoError(t, err)

		// Update batch as completed
		_, err = db.Exec(`
			UPDATE tag_migration_batches 
			SET status = 'completed', completed_at = CURRENT_TIMESTAMP, works_processed = 100
			WHERE batch_number = $1
		`, batchNumber)
		require.NoError(t, err)

		// Verify batch
		var status string
		var processed int
		err = db.QueryRow(`
			SELECT status, works_processed 
			FROM tag_migration_batches 
			WHERE batch_number = $1
		`, batchNumber).Scan(&status, &processed)

		require.NoError(t, err)
		assert.Equal(t, "completed", status)
		assert.Equal(t, 100, processed)
	})

	t.Run("Test work_tag_metrics", func(t *testing.T) {
		workID := uuid.New()

		// Insert test work
		_, err := db.Exec(`
			INSERT INTO works (id, title, summary, word_count, chapter_count) 
			VALUES ($1, 'Spam Work', 'Test summary', 1000, 1)
		`, workID)
		require.NoError(t, err)

		// Insert metrics
		_, err = db.Exec(`
			INSERT INTO work_tag_metrics 
			(work_id, tags_per_1k_words, relationship_tag_ratio, unique_fandom_count, potential_tag_spam)
			VALUES ($1, 25.0, 0.8, 3, true)
		`, workID)
		require.NoError(t, err)

		// Query metrics
		var tagsPerK float64
		var ratio float64
		var fandoms int
		var isSpam bool
		err = db.QueryRow(`
			SELECT tags_per_1k_words, relationship_tag_ratio, unique_fandom_count, potential_tag_spam
			FROM work_tag_metrics 
			WHERE work_id = $1
		`, workID).Scan(&tagsPerK, &ratio, &fandoms, &isSpam)

		require.NoError(t, err)
		assert.Equal(t, 25.0, tagsPerK)
		assert.Equal(t, 0.8, ratio)
		assert.Equal(t, 3, fandoms)
		assert.True(t, isSpam)
	})
}

func TestTagProminenceQueries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Setup test data
	setupTestData(t, db)

	t.Run("Query works by primary relationships only", func(t *testing.T) {
		query := `
			SELECT w.id, w.title
			FROM works w
			JOIN work_tag_summaries wts ON w.id = wts.work_id
			WHERE wts.primary_relationship_count > 0
			ORDER BY w.title
		`

		rows, err := db.Query(query)
		require.NoError(t, err)
		defer rows.Close()

		var count int
		for rows.Next() {
			var id uuid.UUID
			var title string
			err := rows.Scan(&id, &title)
			require.NoError(t, err)
			count++
		}

		// Should find works with primary relationships
		assert.Greater(t, count, 0)
	})

	t.Run("Filter out tag spam works", func(t *testing.T) {
		query := `
			SELECT w.id, w.title
			FROM works w
			JOIN work_tag_summaries wts ON w.id = wts.work_id
			LEFT JOIN work_tag_metrics wtm ON w.id = wtm.work_id
			WHERE wts.total_tag_count <= 15 
			   OR (wtm.potential_tag_spam = false OR wtm.potential_tag_spam IS NULL)
			ORDER BY w.title
		`

		rows, err := db.Query(query)
		require.NoError(t, err)
		defer rows.Close()

		var count int
		for rows.Next() {
			var id uuid.UUID
			var title string
			err := rows.Scan(&id, &title)
			require.NoError(t, err)
			count++
		}

		// Should find non-spam works
		assert.Greater(t, count, 0)
	})

	t.Run("Get tag prominence in search results", func(t *testing.T) {
		query := `
			SELECT wt.tag_name, wt.prominence, wt.prominence_score, wt.auto_assigned
			FROM work_tags wt
			JOIN tags t ON wt.tag_name = t.name
			WHERE t.type = 'relationship'
			   AND wt.prominence IN ('primary', 'secondary')
			ORDER BY wt.prominence_score DESC
		`

		rows, err := db.Query(query)
		require.NoError(t, err)
		defer rows.Close()

		var count int
		for rows.Next() {
			var tagName, prominence string
			var score float64
			var autoAssigned bool
			err := rows.Scan(&tagName, &prominence, &score, &autoAssigned)
			require.NoError(t, err)

			// Validate data
			assert.Contains(t, []string{"primary", "secondary"}, prominence)
			assert.GreaterOrEqual(t, score, 0.0)
			assert.LessOrEqual(t, score, 1.0)

			count++
		}

		assert.Greater(t, count, 0)
	})
}

// Test helper functions
func setupTestDB(t *testing.T) *sql.DB {
	// In a real test environment, this would connect to a test database
	// For now, we'll simulate with a mock or skip if no test DB available

	// Check if test database URL is available
	testDBURL := "postgres://localhost/nuclear_ao3_test?sslmode=disable"

	db, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Skip("Requires test database - skipping integration tests")
		return nil
	}

	// Test connection
	if err := db.Ping(); err != nil {
		t.Skip("Cannot connect to test database - skipping integration tests")
		return nil
	}

	// Clean up any existing test data
	cleanupTestData(t, db)

	return db
}

func cleanupTestData(t *testing.T, db *sql.DB) {
	// Clean up test data in reverse dependency order
	queries := []string{
		"DELETE FROM work_tag_metrics WHERE work_id IN (SELECT id FROM works WHERE title LIKE 'Test%')",
		"DELETE FROM work_tag_summaries WHERE work_id IN (SELECT id FROM works WHERE title LIKE 'Test%')",
		"DELETE FROM work_tags WHERE work_id IN (SELECT id FROM works WHERE title LIKE 'Test%')",
		"DELETE FROM tag_migration_batches WHERE migration_strategy LIKE 'test%'",
		"DELETE FROM tag_prominence_rules WHERE tag_name LIKE 'Test%'",
		"DELETE FROM works WHERE title LIKE 'Test%'",
		"DELETE FROM tags WHERE name LIKE 'Test%'",
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			// Log but don't fail - cleanup is best effort
			t.Logf("Cleanup warning: %v", err)
		}
	}
}

func setupTestData(t *testing.T, db *sql.DB) {
	// Insert test works, tags, and prominence data
	testWorks := []struct {
		id           uuid.UUID
		title        string
		wordCount    int
		chapterCount int
	}{
		{uuid.New(), "Primary Ship Focus", 5000, 1},
		{uuid.New(), "Multi-Ship Epic", 50000, 20},
		{uuid.New(), "Tag Spam Work", 2000, 1},
	}

	for _, work := range testWorks {
		_, err := db.Exec(`
			INSERT INTO works (id, title, summary, word_count, chapter_count) 
			VALUES ($1, $2, 'Test summary', $3, $4)
		`, work.id, work.title, work.wordCount, work.chapterCount)
		require.NoError(t, err)
	}

	// Add test tags and prominence data
	testTags := []struct {
		workIndex  int
		tagName    string
		tagType    string
		prominence string
		score      float64
	}{
		{0, "Harry/Draco", "relationship", "primary", 0.9},
		{1, "Harry/Draco", "relationship", "primary", 0.8},
		{1, "Ron/Hermione", "relationship", "secondary", 0.6},
		{2, "Random Ship 1", "relationship", "micro", 0.1},
		{2, "Random Ship 2", "relationship", "micro", 0.1},
		// ... many more for spam work
	}

	for _, tag := range testTags {
		workID := testWorks[tag.workIndex].id

		// Insert tag
		_, err := db.Exec(`
			INSERT INTO tags (name, type) VALUES ($1, $2) 
			ON CONFLICT (name) DO NOTHING
		`, tag.tagName, tag.tagType)
		require.NoError(t, err)

		// Insert work_tag
		_, err = db.Exec(`
			INSERT INTO work_tags (work_id, tag_name, prominence, prominence_score, auto_assigned) 
			VALUES ($1, $2, $3, $4, true)
		`, workID, tag.tagName, tag.prominence, tag.score)
		require.NoError(t, err)
	}
}
