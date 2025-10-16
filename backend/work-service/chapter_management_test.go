package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
	"nuclear-ao3/shared/models"
)

// =============================================================================
// CHAPTER MANAGEMENT TESTS
// Comprehensive test suite for chapter management functionality
// =============================================================================

type ChapterManagementTestSuite struct {
	suite.Suite
	config *TestDBConfig
	db     *sql.DB
	ws     *WorkService
	router *gin.Engine
}

func (suite *ChapterManagementTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup database using test utilities
	suite.config = SetupTestDB(suite.T())
	suite.db = suite.config.DB

	// Clean up any existing test data
	suite.config.CleanupTestData()

	suite.ws = &WorkService{db: suite.db}
	suite.router = gin.New()

	// Register routes
	api := suite.router.Group("/api/v1")
	{
		api.DELETE("/works/:work_id/chapters/:chapter_id", suite.ws.DeleteChapter)
	}
}

func (suite *ChapterManagementTestSuite) TearDownSuite() {
	if suite.config != nil {
		suite.config.CleanupTestData()
	}
}

func (suite *ChapterManagementTestSuite) SetupTest() {
	// Clean up test data before each test using test utilities
	if suite.config != nil {
		suite.config.CleanupTestData()
	}
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (suite *ChapterManagementTestSuite) createTestUser(name string) uuid.UUID {
	// Generate unique username to avoid conflicts
	uniqueName := fmt.Sprintf("%s_%d", name, time.Now().UnixNano())
	userID, _, err := suite.config.CreateTestUser(uniqueName, uniqueName+"@test.com")
	suite.Require().NoError(err)
	return userID
}

func (suite *ChapterManagementTestSuite) createTestWork(userID uuid.UUID, title string, chapterCount int) (uuid.UUID, []uuid.UUID) {
	// Use test utilities to create work
	workID, err := suite.config.CreateTestWork(userID, title, "published")
	suite.Require().NoError(err)

	// For this test, we'll create chapters manually since the test utilities don't have chapter creation
	// Note: This is a simplified approach - in production you might want to add chapter creation to test_utils.go
	chapterIDs := make([]uuid.UUID, chapterCount)
	for i := 0; i < chapterCount; i++ {
		chapterID := uuid.New()
		_, err = suite.db.Exec(`
			INSERT INTO chapters (id, work_id, chapter_number, title, summary, content, word_count, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
			chapterID, workID, i+1, fmt.Sprintf("Chapter %d", i+1),
			"Chapter summary", "Chapter content goes here.", 1000)
		suite.Require().NoError(err)
		chapterIDs[i] = chapterID
	}

	return workID, chapterIDs
}

func (suite *ChapterManagementTestSuite) makeRequestWithAuth(method, url string, userID uuid.UUID) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, url, nil)
	w := httptest.NewRecorder()

	// Mock JWT middleware by setting user_id in context
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID.String())
		c.Next()
	})

	api := router.Group("/api/v1")
	{
		api.DELETE("/works/:work_id/chapters/:chapter_id", suite.ws.DeleteChapter)
	}

	router.ServeHTTP(w, req)
	return w
}

func (suite *ChapterManagementTestSuite) verifyChapterCount(workID uuid.UUID, expectedCount int) {
	var count int
	err := suite.db.QueryRow("SELECT COUNT(*) FROM chapters WHERE work_id = $1", workID).Scan(&count)
	suite.Require().NoError(err)
	suite.Equal(expectedCount, count, "Chapter count should match expected value")
}

func (suite *ChapterManagementTestSuite) verifyChapterNumbersSequential(workID uuid.UUID) {
	rows, err := suite.db.Query("SELECT chapter_number FROM chapters WHERE work_id = $1 ORDER BY chapter_number", workID)
	suite.Require().NoError(err)
	defer rows.Close()

	expectedNum := 1
	for rows.Next() {
		var chapterNum int
		err = rows.Scan(&chapterNum)
		suite.Require().NoError(err)
		suite.Equal(expectedNum, chapterNum, "Chapter numbers should be sequential")
		expectedNum++
	}
}

func (suite *ChapterManagementTestSuite) verifyWorkStatistics(workID uuid.UUID, expectedChapters, expectedWords int) {
	var work models.Work
	err := suite.db.QueryRow(`
		SELECT chapter_count, word_count FROM works WHERE id = $1`, workID).Scan(
		&work.ChapterCount, &work.WordCount)
	suite.Require().NoError(err)

	suite.Equal(expectedChapters, work.ChapterCount, "Work chapter count should be updated")
	suite.Equal(expectedWords, work.WordCount, "Work word count should be updated")
}

// =============================================================================
// DELETE CHAPTER TESTS
// =============================================================================

func (suite *ChapterManagementTestSuite) TestDeleteChapter_Success() {
	// Create user and multi-chapter work
	userID := suite.createTestUser("testuser")
	workID, chapterIDs := suite.createTestWork(userID, "Multi-Chapter Work", 3)

	// Delete the middle chapter
	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", workID, chapterIDs[1]), userID)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Equal("Chapter deleted successfully", response["message"])

	// Verify chapter count reduced
	suite.verifyChapterCount(workID, 2)

	// Verify chapter numbers are sequential (renumbered)
	suite.verifyChapterNumbersSequential(workID)

	// Verify work statistics updated
	suite.verifyWorkStatistics(workID, 2, 2000) // 2 chapters * 1000 words each

	// Verify the deleted chapter is gone
	var exists bool
	err = suite.db.QueryRow("SELECT EXISTS(SELECT 1 FROM chapters WHERE id = $1)", chapterIDs[1]).Scan(&exists)
	suite.NoError(err)
	suite.False(exists, "Deleted chapter should not exist")
}

func (suite *ChapterManagementTestSuite) TestDeleteChapter_FirstChapter() {
	// Create user and multi-chapter work
	userID := suite.createTestUser("testuser")
	workID, chapterIDs := suite.createTestWork(userID, "Multi-Chapter Work", 3)

	// Delete the first chapter
	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", workID, chapterIDs[0]), userID)

	suite.Equal(http.StatusOK, w.Code)

	// Verify chapter count reduced
	suite.verifyChapterCount(workID, 2)

	// Verify remaining chapters are renumbered starting from 1
	suite.verifyChapterNumbersSequential(workID)

	// Verify the originally chapter 2 is now chapter 1
	var chapterNum int
	err := suite.db.QueryRow("SELECT chapter_number FROM chapters WHERE id = $1", chapterIDs[1]).Scan(&chapterNum)
	suite.NoError(err)
	suite.Equal(1, chapterNum, "Second chapter should become chapter 1")
}

func (suite *ChapterManagementTestSuite) TestDeleteChapter_LastChapter() {
	// Create user and multi-chapter work
	userID := suite.createTestUser("testuser")
	workID, chapterIDs := suite.createTestWork(userID, "Multi-Chapter Work", 3)

	// Delete the last chapter
	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", workID, chapterIDs[2]), userID)

	suite.Equal(http.StatusOK, w.Code)

	// Verify chapter count reduced
	suite.verifyChapterCount(workID, 2)

	// Verify remaining chapters maintain their numbers
	suite.verifyChapterNumbersSequential(workID)

	// Verify work statistics updated
	suite.verifyWorkStatistics(workID, 2, 2000)
}

func (suite *ChapterManagementTestSuite) TestDeleteChapter_CannotDeleteOnlyChapter() {
	// Create user and single-chapter work
	userID := suite.createTestUser("testuser")
	workID, chapterIDs := suite.createTestWork(userID, "Single-Chapter Work", 1)

	// Attempt to delete the only chapter
	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", workID, chapterIDs[0]), userID)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Equal("Cannot delete the only chapter of a work. Delete the work instead.", response["error"])

	// Verify chapter still exists
	suite.verifyChapterCount(workID, 1)
}

func (suite *ChapterManagementTestSuite) TestDeleteChapter_InvalidWorkID() {
	userID := suite.createTestUser("testuser")
	_, chapterIDs := suite.createTestWork(userID, "Test Work", 2)

	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/invalid-uuid/chapters/%s", chapterIDs[0]), userID)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Equal("Invalid work ID", response["error"])
}

func (suite *ChapterManagementTestSuite) TestDeleteChapter_InvalidChapterID() {
	userID := suite.createTestUser("testuser")
	workID, _ := suite.createTestWork(userID, "Test Work", 2)

	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/invalid-uuid", workID), userID)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Equal("Invalid chapter ID", response["error"])
}

func (suite *ChapterManagementTestSuite) TestDeleteChapter_Unauthorized() {
	// Create work without authentication
	req, _ := http.NewRequest("DELETE", "/api/v1/works/some-id/chapters/some-id", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Equal("User not authenticated", response["error"])
}

func (suite *ChapterManagementTestSuite) TestDeleteChapter_NotAuthor() {
	// Create two users
	authorID := suite.createTestUser("author")
	otherUserID := suite.createTestUser("otheruser")

	// Create work by author
	workID, chapterIDs := suite.createTestWork(authorID, "Author's Work", 2)

	// Try to delete chapter as other user
	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", workID, chapterIDs[0]), otherUserID)

	suite.Equal(http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Equal("Not authorized to delete chapters from this work", response["error"])

	// Verify chapter still exists
	suite.verifyChapterCount(workID, 2)
}

func (suite *ChapterManagementTestSuite) TestDeleteChapter_ChapterNotFound() {
	userID := suite.createTestUser("testuser")
	workID, _ := suite.createTestWork(userID, "Test Work", 2)

	// Try to delete non-existent chapter
	nonExistentChapterID := uuid.New()
	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", workID, nonExistentChapterID), userID)

	suite.Equal(http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Equal("Chapter not found", response["error"])
}

func (suite *ChapterManagementTestSuite) TestDeleteChapter_ChapterNotBelongToWork() {
	userID := suite.createTestUser("testuser")

	// Create two works
	workID1, chapterIDs1 := suite.createTestWork(userID, "Work 1", 2)
	workID2, _ := suite.createTestWork(userID, "Work 2", 2)

	// Try to delete chapter from work1 but reference it as belonging to work2
	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", workID2, chapterIDs1[0]), userID)

	suite.Equal(http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Equal("Chapter not found", response["error"])

	// Verify chapters still exist in their original work
	suite.verifyChapterCount(workID1, 2)
}

// =============================================================================
// EDGE CASES AND COMPLEX SCENARIOS
// =============================================================================

func (suite *ChapterManagementTestSuite) TestDeleteChapter_RenumberingComplex() {
	// Create a work with many chapters and delete from various positions
	userID := suite.createTestUser("testuser")
	workID, chapterIDs := suite.createTestWork(userID, "Complex Work", 5)

	// Delete chapter 3 (middle)
	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", workID, chapterIDs[2]), userID)
	suite.Equal(http.StatusOK, w.Code)

	// Verify renumbering: chapters should now be 1, 2, 3, 4
	suite.verifyChapterCount(workID, 4)
	suite.verifyChapterNumbersSequential(workID)

	// Delete chapter 1 (now first)
	w = suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", workID, chapterIDs[0]), userID)
	suite.Equal(http.StatusOK, w.Code)

	// Verify renumbering: chapters should now be 1, 2, 3
	suite.verifyChapterCount(workID, 3)
	suite.verifyChapterNumbersSequential(workID)

	// Verify work statistics are correct
	suite.verifyWorkStatistics(workID, 3, 3000)
}

func (suite *ChapterManagementTestSuite) TestDeleteChapter_WorkStatisticsAccuracy() {
	userID := suite.createTestUser("testuser")
	workID, chapterIDs := suite.createTestWork(userID, "Statistics Test", 4)

	// Modify one chapter to have different word count
	_, err := suite.db.Exec("UPDATE chapters SET word_count = 500 WHERE id = $1", chapterIDs[1])
	suite.Require().NoError(err)

	// Update work total
	_, err = suite.db.Exec("UPDATE works SET word_count = 3500 WHERE id = $1", workID) // 1000+500+1000+1000
	suite.Require().NoError(err)

	// Delete the chapter with 500 words
	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", workID, chapterIDs[1]), userID)
	suite.Equal(http.StatusOK, w.Code)

	// Verify statistics: should be 3 chapters, 3000 words (3500 - 500)
	suite.verifyWorkStatistics(workID, 3, 3000)
}

func (suite *ChapterManagementTestSuite) TestDeleteChapter_MultipleUsersWorks() {
	// Create multiple users and works to ensure isolation
	user1ID := suite.createTestUser("user1")
	user2ID := suite.createTestUser("user2")

	work1ID, chapter1IDs := suite.createTestWork(user1ID, "User1 Work", 3)
	work2ID, _ := suite.createTestWork(user2ID, "User2 Work", 3)

	// User1 deletes chapter from their work
	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", work1ID, chapter1IDs[1]), user1ID)
	suite.Equal(http.StatusOK, w.Code)

	// Verify only user1's work is affected
	suite.verifyChapterCount(work1ID, 2)
	suite.verifyChapterCount(work2ID, 3) // User2's work unchanged

	// User2 cannot delete from user1's work
	w = suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", work1ID, chapter1IDs[0]), user2ID)
	suite.Equal(http.StatusForbidden, w.Code)
}

func (suite *ChapterManagementTestSuite) TestDeleteChapter_TransactionIntegrity() {
	userID := suite.createTestUser("testuser")
	workID, chapterIDs := suite.createTestWork(userID, "Transaction Test", 3)

	// Delete a chapter
	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", workID, chapterIDs[1]), userID)
	suite.Equal(http.StatusOK, w.Code)

	// Verify all transaction operations completed successfully:
	// 1. Chapter deleted
	var exists bool
	err := suite.db.QueryRow("SELECT EXISTS(SELECT 1 FROM chapters WHERE id = $1)", chapterIDs[1]).Scan(&exists)
	suite.NoError(err)
	suite.False(exists, "Chapter should be deleted")

	// 2. Chapters renumbered
	suite.verifyChapterNumbersSequential(workID)

	// 3. Work statistics updated
	suite.verifyWorkStatistics(workID, 2, 2000)

	// 4. Remaining chapters have correct titles
	var titles []string
	rows, err := suite.db.Query("SELECT title FROM chapters WHERE work_id = $1 ORDER BY chapter_number", workID)
	suite.Require().NoError(err)
	defer rows.Close()

	for rows.Next() {
		var title string
		err = rows.Scan(&title)
		suite.Require().NoError(err)
		titles = append(titles, title)
	}

	// Should still have original titles (not renumbered in title)
	suite.Len(titles, 2)
}

// =============================================================================
// PERFORMANCE AND STRESS TESTS
// =============================================================================

func (suite *ChapterManagementTestSuite) TestDeleteChapter_ManyChapters() {
	// Test performance with a work that has many chapters
	userID := suite.createTestUser("testuser")
	workID, chapterIDs := suite.createTestWork(userID, "Large Work", 50)

	// Delete a chapter from the middle
	startTime := time.Now()
	w := suite.makeRequestWithAuth("DELETE",
		fmt.Sprintf("/api/v1/works/%s/chapters/%s", workID, chapterIDs[25]), userID)
	duration := time.Since(startTime)

	suite.Equal(http.StatusOK, w.Code)
	suite.Less(duration, 5*time.Second, "Chapter deletion should complete in reasonable time")

	// Verify correct renumbering
	suite.verifyChapterCount(workID, 49)
	suite.verifyChapterNumbersSequential(workID)
	suite.verifyWorkStatistics(workID, 49, 49000)
}

// =============================================================================
// TEST SUITE REGISTRATION
// =============================================================================

func TestChapterManagementTestSuite(t *testing.T) {
	suite.Run(t, new(ChapterManagementTestSuite))
}
