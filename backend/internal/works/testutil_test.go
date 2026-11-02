package works

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"nuclear-ao3/internal/testdb"
)

// TestDBConfig holds database configuration for tests
type TestDBConfig struct {
	DB *sql.DB
}

// SetupTestDB gives the test its own fresh, fully migrated database (see
// internal/testdb). It is dropped when the test, or for a suite the whole
// suite, ends.
func SetupTestDB(t *testing.T) *TestDBConfig {
	return &TestDBConfig{DB: testdb.New(t)}
}

// CreateTestUser creates a test user with proper schema. The account has no
// username; penName is the name of its default pseud.
func (config *TestDBConfig) CreateTestUser(penName, email string) (uuid.UUID, uuid.UUID, error) {
	userID := uuid.New()

	// Insert user
	_, err := config.DB.Exec(`
		INSERT INTO users (id, email, password_hash, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, true, NOW(), NOW())`,
		userID, email, "$2a$10$hash")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	// Create the default pseud (what creatorships references)
	var pseudID uuid.UUID
	err = config.DB.QueryRow(`INSERT INTO pseuds (user_id, name, is_default) VALUES ($1, $2, true) RETURNING id`,
		userID, penName).Scan(&pseudID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return userID, pseudID, nil
}

// CreateTestUserWithRole creates a test user with a specific role
func (config *TestDBConfig) CreateTestUserWithRole(username, email, role string) (uuid.UUID, uuid.UUID, error) {
	userID, pseudID, err := config.CreateTestUser(username, email)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	// Roles live in user_roles (the database never had a users.role to rely on)
	_, err = config.DB.Exec(`
		INSERT INTO user_roles (user_id, role, granted_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, role) DO NOTHING`,
		userID, role)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return userID, pseudID, nil
}

// CreateTestWork creates a test work with proper schema
func (config *TestDBConfig) CreateTestWork(userID uuid.UUID, title, status string) (uuid.UUID, error) {
	workID := uuid.New()

	// Set proper status values based on what the database expects
	var isComplete bool
	var isDraft bool
	var publishedAt interface{}

	switch status {
	case "published", "posted":
		status = "published" // Use "published" which is allowed by database constraint
		isComplete = false
		isDraft = false
		publishedAt = time.Now()
	case "complete":
		status = "complete"
		isComplete = true
		isDraft = false
		publishedAt = time.Now()
	case "draft":
		isComplete = false
		isDraft = true
		publishedAt = nil
	default:
		status = "published" // Default to "published" which is allowed by constraint
		isComplete = false
		isDraft = false
		publishedAt = time.Now()
	}

	_, err := config.DB.Exec(`
		INSERT INTO works (id, title, summary, user_id, language, rating, status, 
			is_draft, is_complete, published_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())`,
		workID, title, "Test summary", userID, "en", "general",
		status, isDraft, isComplete, publishedAt)

	if err != nil {
		return workID, err
	}

	// Get user's default pseudonym from pseuds table
	var pseudID uuid.UUID
	err = config.DB.QueryRow(`
		SELECT id FROM pseuds WHERE user_id = $1 AND is_default = true LIMIT 1`,
		userID).Scan(&pseudID)
	if err != nil {
		return workID, err
	}

	// Create creatorship entry (this is what the authorization logic checks)
	_, err = config.DB.Exec(`
		INSERT INTO creatorships (creation_id, creation_type, pseud_id, approved, created_at)
		VALUES ($1, 'Work', $2, true, NOW())`,
		workID, pseudID)

	return workID, err
}

// CreateTestComment creates a test comment with proper schema
func (config *TestDBConfig) CreateTestComment(workID, userID, pseudID uuid.UUID, content, status string) (uuid.UUID, error) {
	commentID := uuid.New()

	_, err := config.DB.Exec(`
		INSERT INTO comments (id, work_id, user_id, pseudonym_id, content, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`,
		commentID, workID, userID, pseudID, content, status)

	return commentID, err
}

// CreateTestCollection creates a test collection
func (config *TestDBConfig) CreateTestCollection(userID uuid.UUID, title string) (uuid.UUID, error) {
	collectionID := uuid.New()

	// Generate a unique name from the title
	name := "test-" + uuid.New().String()[:8]

	_, err := config.DB.Exec(`
		INSERT INTO collections (id, name, title, description, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`,
		collectionID, name, title, "Test collection description", userID)

	return collectionID, err
}

// CleanupTestData removes test data based on patterns
func (config *TestDBConfig) CleanupTestData() {
	// Clean up in dependency order
	config.DB.Exec("DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'test%')")
	config.DB.Exec("DELETE FROM comments WHERE work_id IN (SELECT id FROM works WHERE title LIKE 'Test%')")
	config.DB.Exec("DELETE FROM works WHERE title LIKE 'Test%'")
	config.DB.Exec("DELETE FROM collections WHERE name LIKE 'Test%'")
	config.DB.Exec("DELETE FROM pseuds WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'test%')")
	config.DB.Exec("DELETE FROM users WHERE username LIKE 'test%'")
}

// Close closes the database connection
func (config *TestDBConfig) Close() {
	if config.DB != nil {
		config.DB.Close()
	}
}
