package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

// TagProminenceMigrator handles intelligent tag prominence assignment
type TagProminenceMigrator struct {
	db        *sql.DB
	batchSize int
	dryRun    bool
}

// TagInfo holds tag information for prominence calculation
type TagInfo struct {
	TagName      string
	TagType      string
	WorkID       string
	WordCount    int
	ChapterCount int
	TotalTags    int
}

// ProminenceScore holds calculated prominence information
type ProminenceScore struct {
	Prominence   string
	Score        float64
	Reasoning    string
	AutoAssigned bool
}

// NewTagProminenceMigrator creates a new migrator instance
func NewTagProminenceMigrator(dbURL string, batchSize int, dryRun bool) (*TagProminenceMigrator, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &TagProminenceMigrator{
		db:        db,
		batchSize: batchSize,
		dryRun:    dryRun,
	}, nil
}

// IntelligentProminenceAssignment calculates tag prominence based on multiple factors
func (m *TagProminenceMigrator) IntelligentProminenceAssignment(tagInfo TagInfo) ProminenceScore {
	score := 0.0
	reasoning := []string{}

	// Factor 1: Tag count ratio (fewer tags = higher prominence for each)
	if tagInfo.TotalTags == 1 {
		score += 0.4 // Single tag gets high boost
		reasoning = append(reasoning, "only tag")
	} else if tagInfo.TotalTags <= 3 {
		score += 0.3 // Few tags get medium boost
		reasoning = append(reasoning, "few tags")
	} else if tagInfo.TotalTags > 20 {
		score -= 0.2 // Many tags get penalty
		reasoning = append(reasoning, "many tags")
	} else if tagInfo.TotalTags > 15 {
		score -= 0.1 // Medium penalty for moderate tag count
		reasoning = append(reasoning, "moderate tags")
	}

	// Factor 2: Tag type patterns
	if tagInfo.TagType == "relationship" {
		score += 0.3 // Relationships start with higher base score

		// Common primary relationship indicators
		if strings.Contains(strings.ToLower(tagInfo.TagName), "main") ||
			strings.Contains(strings.ToLower(tagInfo.TagName), "central") ||
			strings.Contains(strings.ToLower(tagInfo.TagName), "pairing") {
			score += 0.5 // Increased bonus for explicit main tags
			reasoning = append(reasoning, "marked as main")
		}

		// Gen fic should be primary if it's the only relationship
		if strings.ToLower(tagInfo.TagName) == "gen" ||
			strings.ToLower(tagInfo.TagName) == "no romantic pairings" {
			if tagInfo.TotalTags <= 5 {
				score += 0.5
				reasoning = append(reasoning, "gen fic")
			}
		}

		// Background/side relationship indicators
		if strings.Contains(strings.ToLower(tagInfo.TagName), "background") ||
			strings.Contains(strings.ToLower(tagInfo.TagName), "side") ||
			strings.Contains(strings.ToLower(tagInfo.TagName), "minor") ||
			strings.Contains(strings.ToLower(tagInfo.TagName), "mention") ||
			strings.Contains(strings.ToLower(tagInfo.TagName), "past") ||
			strings.Contains(strings.ToLower(tagInfo.TagName), "implied") {
			score -= 0.6
			reasoning = append(reasoning, "marked as background")
		}
	} else if tagInfo.TagType == "warning" {
		score += 0.2 // Warnings are important but not as much as relationships

		// Special handling for major warnings
		if strings.Contains(strings.ToLower(tagInfo.TagName), "character death") ||
			strings.Contains(strings.ToLower(tagInfo.TagName), "main character") {
			score += 0.3
			reasoning = append(reasoning, "major warning")
		}
	}

	// Factor 3: Word count ratio (more words per tag = higher prominence)
	if tagInfo.WordCount > 0 {
		wordsPerTag := float64(tagInfo.WordCount) / float64(tagInfo.TotalTags)
		if wordsPerTag > 3000 { // Lowered threshold
			score += 0.2 // Substantial content
			reasoning = append(reasoning, "substantial content")
		} else if wordsPerTag < 200 { // Very restrictive penalty threshold
			score -= 0.1 // Light content
			reasoning = append(reasoning, "light content")
		}
	} else {
		// Zero word count
		reasoning = append(reasoning, "zero words")
	}

	// Factor 4: Chapter distribution (if multi-chapter)
	if tagInfo.ChapterCount > 1 {
		// Assume primary relationships appear throughout
		bonus := 0.2
		if tagInfo.ChapterCount >= 5 {
			bonus = 0.3 // Larger bonus for longer works
		}
		score += bonus
		reasoning = append(reasoning, "multi-chapter")
	}

	// Convert score to prominence level
	prominence := "secondary" // Default
	autoAssigned := true

	if score >= 0.7 {
		prominence = "primary"
	} else if score < 0.2 {
		prominence = "micro"
	}

	// Special cases that override calculation
	if tagInfo.TotalTags == 1 && tagInfo.TagType == "relationship" {
		prominence = "primary"
		reasoning = append(reasoning, "single relationship")
	}

	if tagInfo.TotalTags > 25 {
		// Likely tag spam - needs manual review
		autoAssigned = false
		reasoning = append(reasoning, "potential tag spam")
		// For tag spam, default to secondary if it's not already primary or explicitly micro
		if prominence != "primary" && !strings.Contains(strings.Join(reasoning, ", "), "marked as background") {
			prominence = "secondary"
		}
	}

	return ProminenceScore{
		Prominence:   prominence,
		Score:        math.Min(1.0, math.Max(0.0, score)),
		Reasoning:    strings.Join(reasoning, ", "),
		AutoAssigned: autoAssigned,
	}
}

// MigrateBatch processes a batch of works for tag prominence assignment
func (m *TagProminenceMigrator) MigrateBatch(batchNumber int, offset, limit int) error {
	log.Printf("Starting batch %d: offset %d, limit %d", batchNumber, offset, limit)

	// Get works that need migration (where prominence = 'unassigned')
	query := `
		SELECT DISTINCT w.id, w.word_count, w.chapter_count,
		       COUNT(wt.tag_name) as total_tags
		FROM works w
		JOIN work_tags wt ON w.id = wt.work_id
		WHERE wt.prominence = 'unassigned'
		GROUP BY w.id, w.word_count, w.chapter_count
		ORDER BY w.created_at
		OFFSET $1 LIMIT $2
	`

	rows, err := m.db.Query(query, offset, limit)
	if err != nil {
		return fmt.Errorf("failed to query works: %w", err)
	}
	defer rows.Close()

	processed := 0
	for rows.Next() {
		var workID string
		var wordCount, chapterCount, totalTags int

		err := rows.Scan(&workID, &wordCount, &chapterCount, &totalTags)
		if err != nil {
			log.Printf("Error scanning work row: %v", err)
			continue
		}

		err = m.processWork(workID, wordCount, chapterCount, totalTags, batchNumber)
		if err != nil {
			log.Printf("Error processing work %s: %v", workID, err)
			continue
		}

		processed++
		if processed%100 == 0 {
			log.Printf("Processed %d works in batch %d", processed, batchNumber)
		}
	}

	log.Printf("Completed batch %d: processed %d works", batchNumber, processed)
	return nil
}

// processWork handles prominence assignment for a single work
func (m *TagProminenceMigrator) processWork(workID string, wordCount, chapterCount, totalTags int, batchNumber int) error {
	// Get all unassigned tags for this work
	tagQuery := `
		SELECT wt.tag_name, t.type
		FROM work_tags wt
		JOIN tags t ON wt.tag_name = t.name
		WHERE wt.work_id = $1 AND wt.prominence = 'unassigned'
	`

	rows, err := m.db.Query(tagQuery, workID)
	if err != nil {
		return fmt.Errorf("failed to query tags for work %s: %w", workID, err)
	}
	defer rows.Close()

	var updates []struct {
		tagName      string
		prominence   string
		score        float64
		autoAssigned bool
	}

	for rows.Next() {
		var tagName, tagType string
		err := rows.Scan(&tagName, &tagType)
		if err != nil {
			continue
		}

		tagInfo := TagInfo{
			TagName:      tagName,
			TagType:      tagType,
			WorkID:       workID,
			WordCount:    wordCount,
			ChapterCount: chapterCount,
			TotalTags:    totalTags,
		}

		prominenceScore := m.IntelligentProminenceAssignment(tagInfo)

		updates = append(updates, struct {
			tagName      string
			prominence   string
			score        float64
			autoAssigned bool
		}{
			tagName:      tagName,
			prominence:   prominenceScore.Prominence,
			score:        prominenceScore.Score,
			autoAssigned: prominenceScore.AutoAssigned,
		})

		if m.dryRun {
			log.Printf("DRY RUN - Work %s, Tag %s -> %s (score: %.2f, reason: %s)",
				workID, tagName, prominenceScore.Prominence, prominenceScore.Score, prominenceScore.Reasoning)
		}
	}

	if m.dryRun {
		return nil
	}

	// Apply updates in a transaction
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	updateQuery := `
		UPDATE work_tags 
		SET prominence = $1, prominence_score = $2, auto_assigned = $3, 
		    migration_batch = $4, updated_at = CURRENT_TIMESTAMP
		WHERE work_id = $5 AND tag_name = $6
	`

	for _, update := range updates {
		_, err = tx.Exec(updateQuery,
			update.prominence, update.score, update.autoAssigned,
			batchNumber, workID, update.tagName)
		if err != nil {
			return fmt.Errorf("failed to update tag prominence: %w", err)
		}
	}

	return tx.Commit()
}

// RunMigration executes the full migration process
func (m *TagProminenceMigrator) RunMigration() error {
	// Count total works needing migration
	var totalWorks int
	err := m.db.QueryRow(`
		SELECT COUNT(DISTINCT work_id) 
		FROM work_tags 
		WHERE prominence = 'unassigned'
	`).Scan(&totalWorks)
	if err != nil {
		return fmt.Errorf("failed to count works: %w", err)
	}

	log.Printf("Total works to migrate: %d", totalWorks)

	// Create migration batch record
	var batchNumber int
	err = m.db.QueryRow(`
		INSERT INTO tag_migration_batches (batch_number, works_total, migration_strategy, status)
		VALUES ((SELECT COALESCE(MAX(batch_number), 0) + 1 FROM tag_migration_batches), $1, $2, 'running')
		RETURNING batch_number
	`, totalWorks, "intelligent_prominence_assignment").Scan(&batchNumber)
	if err != nil {
		return fmt.Errorf("failed to create migration batch: %w", err)
	}

	log.Printf("Starting migration batch %d", batchNumber)

	// Process in batches
	for offset := 0; offset < totalWorks; offset += m.batchSize {
		err = m.MigrateBatch(batchNumber, offset, m.batchSize)
		if err != nil {
			// Mark batch as failed
			m.db.Exec(`
				UPDATE tag_migration_batches 
				SET status = 'failed', completed_at = CURRENT_TIMESTAMP 
				WHERE batch_number = $1
			`, batchNumber)
			return fmt.Errorf("batch migration failed: %w", err)
		}
	}

	// Mark batch as completed
	_, err = m.db.Exec(`
		UPDATE tag_migration_batches 
		SET status = 'completed', completed_at = CURRENT_TIMESTAMP,
		    works_processed = $1
		WHERE batch_number = $2
	`, totalWorks, batchNumber)
	if err != nil {
		log.Printf("Warning: failed to mark batch as completed: %v", err)
	}

	log.Printf("Migration batch %d completed successfully", batchNumber)
	return nil
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	batchSize := 1000
	if bs := os.Getenv("BATCH_SIZE"); bs != "" {
		if parsed, err := strconv.Atoi(bs); err == nil {
			batchSize = parsed
		}
	}

	dryRun := os.Getenv("DRY_RUN") == "true"

	migrator, err := NewTagProminenceMigrator(dbURL, batchSize, dryRun)
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.db.Close()

	if dryRun {
		log.Println("Running in DRY RUN mode - no changes will be made")
	}

	err = migrator.RunMigration()
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully!")
}
