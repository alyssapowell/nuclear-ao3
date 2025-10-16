package main

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDatabaseConnection(t *testing.T) {
	// Test database connection
	dbURL := "postgres://ao3_user:ao3_password@localhost/ao3_nuclear?sslmode=disable"
	db, err := sql.Open("postgres", dbURL)
	assert.NoError(t, err)
	defer db.Close()

	err = db.Ping()
	assert.NoError(t, err)

	// Test user creation
	testUserID := uuid.New()
	username := "testuser_debug"
	email := "debug@example.com"

	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, NOW(), NOW())
	`, testUserID, username, email, "hashed_password")
	assert.NoError(t, err)

	// Verify user exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", testUserID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Test pseudonym creation
	testPseudID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO user_pseudonyms (id, user_id, name, is_default, created_at)
		VALUES ($1, $2, $3, true, NOW())
	`, testPseudID, testUserID, "TestPseud")
	assert.NoError(t, err)

	// Test work creation
	testWorkID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO works (id, title, summary, user_id, language, rating, status, word_count, chapter_count, is_complete, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'en', 'General Audiences', 'published', 1000, 1, true, NOW(), NOW())
	`, testWorkID, "Debug Work", "Debug Summary", testUserID)
	assert.NoError(t, err)

	// Test comment creation with proper constraint compliance
	commentID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO comments (
			id, work_id, user_id, pseudonym_id, content, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, NOW(), NOW()
		)
	`, commentID, testWorkID, testUserID, testPseudID, "Test comment content")
	assert.NoError(t, err)

	// Verify comment exists
	err = db.QueryRow("SELECT COUNT(*) FROM comments WHERE id = $1", commentID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Clean up
	db.Exec("DELETE FROM comments WHERE id = $1", commentID)
	db.Exec("DELETE FROM works WHERE id = $1", testWorkID)
	db.Exec("DELETE FROM user_pseudonyms WHERE id = $1", testPseudID)
	db.Exec("DELETE FROM users WHERE id = $1", testUserID)
}
