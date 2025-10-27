package main

import (
	"testing"
)

func TestIntelligentProminenceAssignment(t *testing.T) {
	migrator := &TagProminenceMigrator{}

	tests := []struct {
		name     string
		tagInfo  TagInfo
		expected ProminenceScore
	}{
		{
			name: "Single relationship tag should be primary",
			tagInfo: TagInfo{
				TagName:      "Hermione Granger/Ron Weasley",
				TagType:      "relationship",
				WorkID:       "test-1",
				WordCount:    5000,
				ChapterCount: 1,
				TotalTags:    1,
			},
			expected: ProminenceScore{
				Prominence:   "primary",
				AutoAssigned: true,
			},
		},
		{
			name: "Gen tag with few total tags should be primary",
			tagInfo: TagInfo{
				TagName:      "Gen",
				TagType:      "relationship",
				WorkID:       "test-2",
				WordCount:    3000,
				ChapterCount: 1,
				TotalTags:    3,
			},
			expected: ProminenceScore{
				Prominence:   "primary",
				AutoAssigned: true,
			},
		},
		{
			name: "Background relationship should be micro",
			tagInfo: TagInfo{
				TagName:      "Background Harry Potter/Ginny Weasley",
				TagType:      "relationship",
				WorkID:       "test-3",
				WordCount:    8000,
				ChapterCount: 1,
				TotalTags:    5,
			},
			expected: ProminenceScore{
				Prominence:   "micro",
				AutoAssigned: true,
			},
		},
		{
			name: "Side relationship should be micro",
			tagInfo: TagInfo{
				TagName:      "Side Draco Malfoy/Pansy Parkinson",
				TagType:      "relationship",
				WorkID:       "test-4",
				WordCount:    6000,
				ChapterCount: 1,
				TotalTags:    8,
			},
			expected: ProminenceScore{
				Prominence:   "micro",
				AutoAssigned: true,
			},
		},
		{
			name: "Many tags should default to secondary",
			tagInfo: TagInfo{
				TagName:      "Harry Potter/Draco Malfoy",
				TagType:      "relationship",
				WorkID:       "test-5",
				WordCount:    4000,
				ChapterCount: 1,
				TotalTags:    12,
			},
			expected: ProminenceScore{
				Prominence:   "secondary",
				AutoAssigned: true,
			},
		},
		{
			name: "Tag spam should require manual review",
			tagInfo: TagInfo{
				TagName:      "Some Random Ship",
				TagType:      "relationship",
				WorkID:       "test-6",
				WordCount:    2000,
				ChapterCount: 1,
				TotalTags:    30,
			},
			expected: ProminenceScore{
				Prominence:   "secondary",
				AutoAssigned: false, // Should require manual review
			},
		},
		{
			name: "Main character tag with high word count should be primary",
			tagInfo: TagInfo{
				TagName:      "Main Character Death",
				TagType:      "warning",
				WorkID:       "test-7",
				WordCount:    15000,
				ChapterCount: 3,
				TotalTags:    4,
			},
			expected: ProminenceScore{
				Prominence:   "primary",
				AutoAssigned: true,
			},
		},
		{
			name: "Relationship with low words per tag should be micro",
			tagInfo: TagInfo{
				TagName:      "Minor Character A/Minor Character B",
				TagType:      "relationship",
				WorkID:       "test-8",
				WordCount:    800,
				ChapterCount: 1,
				TotalTags:    10,
			},
			expected: ProminenceScore{
				Prominence:   "micro",
				AutoAssigned: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.IntelligentProminenceAssignment(tt.tagInfo)

			if result.Prominence != tt.expected.Prominence {
				t.Errorf("Expected prominence %s, got %s", tt.expected.Prominence, result.Prominence)
			}

			if result.AutoAssigned != tt.expected.AutoAssigned {
				t.Errorf("Expected AutoAssigned %v, got %v", tt.expected.AutoAssigned, result.AutoAssigned)
			}

			// Score should be between 0 and 1
			if result.Score < 0 || result.Score > 1 {
				t.Errorf("Score should be between 0 and 1, got %f", result.Score)
			}
		})
	}
}

func TestTagSpamDetection(t *testing.T) {
	migrator := &TagProminenceMigrator{}

	tagSpamCase := TagInfo{
		TagName:      "Random Ship",
		TagType:      "relationship",
		WorkID:       "spam-work",
		WordCount:    1000,
		ChapterCount: 1,
		TotalTags:    50, // Way too many tags
	}

	result := migrator.IntelligentProminenceAssignment(tagSpamCase)

	if result.AutoAssigned {
		t.Error("Works with excessive tags should not be auto-assigned")
	}
}

func TestScoreCalculation(t *testing.T) {
	migrator := &TagProminenceMigrator{}

	// Test that fewer tags = higher scores
	singleTag := TagInfo{
		TagName:      "Main Ship",
		TagType:      "relationship",
		WorkID:       "single",
		WordCount:    5000,
		ChapterCount: 1,
		TotalTags:    1,
	}

	manyTags := TagInfo{
		TagName:      "Main Ship",
		TagType:      "relationship",
		WorkID:       "many",
		WordCount:    5000,
		ChapterCount: 1,
		TotalTags:    20,
	}

	singleResult := migrator.IntelligentProminenceAssignment(singleTag)
	manyResult := migrator.IntelligentProminenceAssignment(manyTags)

	if singleResult.Score <= manyResult.Score {
		t.Error("Single tag should have higher prominence score than many tags")
	}
}

func TestGenFicHandling(t *testing.T) {
	migrator := &TagProminenceMigrator{}

	genCases := []string{
		"Gen",
		"gen",
		"No Romantic Pairings",
		"no romantic pairings",
	}

	for _, genTag := range genCases {
		tagInfo := TagInfo{
			TagName:      genTag,
			TagType:      "relationship",
			WorkID:       "gen-test",
			WordCount:    3000,
			ChapterCount: 1,
			TotalTags:    3,
		}

		result := migrator.IntelligentProminenceAssignment(tagInfo)

		if result.Prominence != "primary" {
			t.Errorf("Gen tag '%s' should be primary, got %s", genTag, result.Prominence)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}

// Test edge cases and error conditions
func TestEdgeCasesAndErrorHandling(t *testing.T) {
	migrator := &TagProminenceMigrator{}

	t.Run("Zero word count", func(t *testing.T) {
		tagInfo := TagInfo{
			TagName:      "Test Tag",
			TagType:      "relationship",
			WorkID:       "zero-words",
			WordCount:    0,
			ChapterCount: 1,
			TotalTags:    5,
		}

		result := migrator.IntelligentProminenceAssignment(tagInfo)

		// Should still assign prominence despite zero words
		validProminence := result.Prominence == "primary" || result.Prominence == "secondary" || result.Prominence == "micro"
		if !validProminence {
			t.Errorf("Invalid prominence %s for zero word count", result.Prominence)
		}
		if !result.AutoAssigned {
			t.Error("Zero word count should still be auto-assigned")
		}
	})

	t.Run("Very high tag count", func(t *testing.T) {
		tagInfo := TagInfo{
			TagName:      "Random Tag",
			TagType:      "relationship",
			WorkID:       "high-tag-count",
			WordCount:    5000,
			ChapterCount: 1,
			TotalTags:    100, // Extremely high
		}

		result := migrator.IntelligentProminenceAssignment(tagInfo)

		// Should not be auto-assigned due to potential spam
		if result.AutoAssigned {
			t.Error("High tag count should not be auto-assigned")
		}
		if !contains(result.Reasoning, "potential tag spam") {
			t.Errorf("Expected reasoning to contain 'potential tag spam', got: %s", result.Reasoning)
		}
	})

	t.Run("Character tags", func(t *testing.T) {
		tagInfo := TagInfo{
			TagName:      "Hermione Granger",
			TagType:      "character",
			WorkID:       "char-test",
			WordCount:    3000,
			ChapterCount: 1,
			TotalTags:    8,
		}

		result := migrator.IntelligentProminenceAssignment(tagInfo)

		// Character tags shouldn't get relationship bonuses
		if result.Score < 0.0 || result.Score > 1.0 {
			t.Errorf("Score should be between 0 and 1, got %f", result.Score)
		}
	})

	t.Run("Multi-chapter works", func(t *testing.T) {
		tagInfo := TagInfo{
			TagName:      "Harry Potter/Draco Malfoy",
			TagType:      "relationship",
			WorkID:       "multi-chapter",
			WordCount:    20000,
			ChapterCount: 10,
			TotalTags:    5,
		}

		result := migrator.IntelligentProminenceAssignment(tagInfo)

		// Multi-chapter should get bonus
		if !contains(result.Reasoning, "multi-chapter") {
			t.Errorf("Expected reasoning to contain 'multi-chapter', got: %s", result.Reasoning)
		}
		if result.Prominence != "primary" {
			t.Errorf("Expected primary prominence for multi-chapter work, got: %s", result.Prominence)
		}
	})

	t.Run("Various warning tags", func(t *testing.T) {
		warningTags := []string{
			"Major Character Death",
			"Graphic Depictions Of Violence",
			"Rape/Non-Con",
			"Underage",
		}

		for _, tagName := range warningTags {
			tagInfo := TagInfo{
				TagName:      tagName,
				TagType:      "warning",
				WorkID:       "warning-test",
				WordCount:    8000,
				ChapterCount: 1,
				TotalTags:    6,
			}

			result := migrator.IntelligentProminenceAssignment(tagInfo)

			// Warning tags should be properly handled
			validProminence := result.Prominence == "primary" || result.Prominence == "secondary" || result.Prominence == "micro"
			if !validProminence {
				t.Errorf("Invalid prominence %s for warning tag %s", result.Prominence, tagName)
			}
			if !result.AutoAssigned {
				t.Errorf("Warning tag %s should be auto-assigned", tagName)
			}
		}
	})
}

func TestSpecialRelationshipPatterns(t *testing.T) {
	migrator := &TagProminenceMigrator{}

	testCases := []struct {
		name          string
		tagName       string
		expectedLevel string
		shouldContain string
	}{
		{
			name:          "Past relationship",
			tagName:       "Past Harry Potter/Ginny Weasley",
			expectedLevel: "micro",
			shouldContain: "marked as background",
		},
		{
			name:          "Implied relationship",
			tagName:       "Implied Draco Malfoy/Harry Potter",
			expectedLevel: "micro",
			shouldContain: "marked as background",
		},
		{
			name:          "One-sided relationship",
			tagName:       "One-Sided Ron Weasley/Hermione Granger",
			expectedLevel: "micro",
			shouldContain: "marked as background",
		},
		{
			name:          "Central relationship",
			tagName:       "Central Relationship Harry Potter/Draco Malfoy",
			expectedLevel: "primary",
			shouldContain: "marked as main",
		},
		{
			name:          "Main pairing",
			tagName:       "Main Pairing: Hermione Granger/Severus Snape",
			expectedLevel: "primary",
			shouldContain: "marked as main",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tagInfo := TagInfo{
				TagName:      tc.tagName,
				TagType:      "relationship",
				WorkID:       "pattern-test",
				WordCount:    5000,
				ChapterCount: 1,
				TotalTags:    8,
			}

			result := migrator.IntelligentProminenceAssignment(tagInfo)

			if result.Prominence != tc.expectedLevel {
				t.Errorf("Expected prominence %s for tag %s, got %s", tc.expectedLevel, tc.tagName, result.Prominence)
			}
			if !contains(result.Reasoning, tc.shouldContain) {
				t.Errorf("Expected reasoning to contain '%s' for tag %s, got: %s", tc.shouldContain, tc.tagName, result.Reasoning)
			}
		})
	}
}

func TestScoreValidation(t *testing.T) {
	migrator := &TagProminenceMigrator{}

	// Test various tag combinations to ensure scores are always valid
	testCases := []TagInfo{
		{TagName: "Single Tag", TagType: "relationship", WorkID: "test", WordCount: 1, ChapterCount: 1, TotalTags: 1},
		{TagName: "Many Tags Test", TagType: "additional_tags", WorkID: "test", WordCount: 500, ChapterCount: 1, TotalTags: 50},
		{TagName: "High Words", TagType: "relationship", WorkID: "test", WordCount: 100000, ChapterCount: 50, TotalTags: 3},
		{TagName: "Zero Words", TagType: "character", WorkID: "test", WordCount: 0, ChapterCount: 1, TotalTags: 10},
	}

	for _, tagInfo := range testCases {
		result := migrator.IntelligentProminenceAssignment(tagInfo)

		// Score must always be between 0 and 1
		if result.Score < 0.0 {
			t.Errorf("Score below 0 for tag: %s, got %f", tagInfo.TagName, result.Score)
		}
		if result.Score > 1.0 {
			t.Errorf("Score above 1 for tag: %s, got %f", tagInfo.TagName, result.Score)
		}

		// Prominence must be valid
		validProminence := result.Prominence == "primary" || result.Prominence == "secondary" || result.Prominence == "micro"
		if !validProminence {
			t.Errorf("Invalid prominence %s for tag: %s", result.Prominence, tagInfo.TagName)
		}

		// Reasoning should not be empty
		if result.Reasoning == "" {
			t.Errorf("Empty reasoning for tag: %s", tagInfo.TagName)
		}
	}
}

// Benchmark tests for performance
func BenchmarkProminenceAssignment(b *testing.B) {
	migrator := &TagProminenceMigrator{}

	tagInfo := TagInfo{
		TagName:      "Test Ship",
		TagType:      "relationship",
		WorkID:       "bench-test",
		WordCount:    5000,
		ChapterCount: 1,
		TotalTags:    8,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		migrator.IntelligentProminenceAssignment(tagInfo)
	}
}

func BenchmarkBatchProcessing(b *testing.B) {
	migrator := &TagProminenceMigrator{}

	// Simulate batch processing multiple tags
	tags := make([]TagInfo, 100)
	for i := range tags {
		tags[i] = TagInfo{
			TagName:      "Batch Test Tag",
			TagType:      "relationship",
			WorkID:       "batch-test",
			WordCount:    5000,
			ChapterCount: 1,
			TotalTags:    10,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tag := range tags {
			migrator.IntelligentProminenceAssignment(tag)
		}
	}
}
