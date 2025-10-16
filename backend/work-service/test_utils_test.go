package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestUtils(t *testing.T) {
	// Setup database
	config := SetupTestDB(t)
	defer config.Close()

	// Clean up any existing test data
	config.CleanupTestData()

	// Test user creation
	userID, pseudID, err := config.CreateTestUser("testutils", "testutils@test.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	assert.NotEmpty(t, userID)
	assert.NotEmpty(t, pseudID)

	// Test user with role creation
	adminID, adminPseudID, err := config.CreateTestUserWithRole("testadmin", "testadmin@test.com", "admin")
	assert.NoError(t, err)
	assert.NotEmpty(t, adminID)
	assert.NotEmpty(t, adminPseudID)

	// Test work creation
	workID, err := config.CreateTestWork(userID, "Test Work Utils", "published")
	if err != nil {
		t.Fatalf("Failed to create test work: %v", err)
	}
	assert.NotEmpty(t, workID)

	// Test comment creation
	t.Logf("Creating comment with userID=%s, pseudID=%s, workID=%s", userID, pseudID, workID)
	commentID, err := config.CreateTestComment(workID, userID, pseudID, "Test comment", "published")
	if err != nil {
		t.Fatalf("Failed to create test comment: %v", err)
	}
	assert.NotEmpty(t, commentID)

	// Test collection creation
	collectionID, err := config.CreateTestCollection(userID, "Test Collection")
	assert.NoError(t, err)
	assert.NotEmpty(t, collectionID)

	// Clean up
	config.CleanupTestData()
}
