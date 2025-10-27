package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Search request/response types
type WorkSearchRequest struct {
	Query             string   `json:"query,omitempty"`
	Title             string   `json:"title,omitempty"`
	Author            string   `json:"author,omitempty"`
	Fandoms           []string `json:"fandoms,omitempty"`
	Characters        []string `json:"characters,omitempty"`
	Relationships     []string `json:"relationships,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Rating            []string `json:"rating,omitempty"`
	Category          []string `json:"category,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
	Status            string   `json:"status,omitempty"`
	Language          []string `json:"language,omitempty"`
	WordCountMin      *int     `json:"word_count_min,omitempty"`
	WordCountMax      *int     `json:"word_count_max,omitempty"`
	UpdatedAfter      string   `json:"updated_after,omitempty"`
	UpdatedBefore     string   `json:"updated_before,omitempty"`
	SortBy            string   `json:"sort_by,omitempty"`
	SortOrder         string   `json:"sort_order,omitempty"`
	Page              int      `json:"page,omitempty"`
	Limit             int      `json:"limit,omitempty"`
	RelationshipCount string   `json:"relationship_count,omitempty"` // '1-2', '3-5', '6-10', '10+'
	TagProminence     string   `json:"tag_prominence,omitempty"`     // 'primary', 'secondary', 'any'
	// Content filtering
	BlockedTags         []string `json:"blocked_tags,omitempty"`
	HideIncomplete      bool     `json:"hide_incomplete,omitempty"`
	HideCrossovers      bool     `json:"hide_crossovers,omitempty"`
	HideNoRelationships bool     `json:"hide_no_relationships,omitempty"`
	// Date filtering
	UpdatedWithin   string `json:"updated_within,omitempty"` // 'week', 'month', '3months', 'year'
	PublishedAfter  string `json:"published_after,omitempty"`
	PublishedBefore string `json:"published_before,omitempty"`
	// Engagement filtering
	MinKudos     *int `json:"min_kudos,omitempty"`
	MinComments  *int `json:"min_comments,omitempty"`
	MinBookmarks *int `json:"min_bookmarks,omitempty"`
	HideOrphaned bool `json:"hide_orphaned,omitempty"`
}

type SearchResponse struct {
	Results    []map[string]interface{} `json:"results"`
	Total      int                      `json:"total"`
	Page       int                      `json:"page"`
	Limit      int                      `json:"limit"`
	Pages      int                      `json:"pages"`
	SearchTime int64                    `json:"search_time_ms"`
	Facets     map[string]interface{}   `json:"facets,omitempty"`
}

// Work search handlers

func (ss *SearchService) SearchWorks(c *gin.Context) {
	start := time.Now()
	log.Printf("SearchWorks called with params: %v", c.Request.URL.Query())

	// Parse query parameters
	req := WorkSearchRequest{
		Query:     c.DefaultQuery("q", ""),
		Title:     c.DefaultQuery("title", ""),
		Author:    c.DefaultQuery("author", ""),
		Status:    c.DefaultQuery("status", "all"),
		SortBy:    c.DefaultQuery("sort", "relevance"),
		SortOrder: c.DefaultQuery("order", "desc"),
		Page:      1,
		Limit:     20,
	}

	// Parse arrays
	req.Fandoms = c.QueryArray("fandom")
	req.Characters = c.QueryArray("character")
	req.Relationships = c.QueryArray("relationship")
	req.Tags = c.QueryArray("tag")
	req.Rating = c.QueryArray("rating")
	req.Category = c.QueryArray("category")
	req.Warnings = c.QueryArray("warning")
	req.Language = c.QueryArray("language")

	// Parse integers
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			req.Page = p
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			req.Limit = l
		}
	}

	// Parse word count ranges
	if minWords := c.Query("word_count_min"); minWords != "" {
		if min, err := strconv.Atoi(minWords); err == nil {
			req.WordCountMin = &min
		}
	}

	if maxWords := c.Query("word_count_max"); maxWords != "" {
		if max, err := strconv.Atoi(maxWords); err == nil {
			req.WordCountMax = &max
		}
	}

	// Parse date ranges
	req.UpdatedAfter = c.Query("updated_after")
	req.UpdatedBefore = c.Query("updated_before")

	// Parse new filter parameters
	req.RelationshipCount = c.Query("relationship_count")
	req.TagProminence = c.Query("tag_prominence")

	// Content filtering
	req.BlockedTags = c.QueryArray("blocked_tags")
	req.HideIncomplete = c.Query("hide_incomplete") == "true"
	req.HideCrossovers = c.Query("hide_crossovers") == "true"
	req.HideNoRelationships = c.Query("hide_no_relationships") == "true"

	// Date filtering
	req.UpdatedWithin = c.Query("updated_within")
	req.PublishedAfter = c.Query("published_after")
	req.PublishedBefore = c.Query("published_before")

	// Engagement filtering
	if minKudos := c.Query("min_kudos"); minKudos != "" {
		if kudos, err := strconv.Atoi(minKudos); err == nil {
			req.MinKudos = &kudos
		}
	}
	if minComments := c.Query("min_comments"); minComments != "" {
		if comments, err := strconv.Atoi(minComments); err == nil {
			req.MinComments = &comments
		}
	}
	if minBookmarks := c.Query("min_bookmarks"); minBookmarks != "" {
		if bookmarks, err := strconv.Atoi(minBookmarks); err == nil {
			req.MinBookmarks = &bookmarks
		}
	}
	req.HideOrphaned = c.Query("hide_orphaned") == "true"

	// Build Elasticsearch query
	log.Printf("Building query for request: %+v", req)
	esQuery := ss.buildWorkSearchQuery(req)

	// Execute search
	response, err := ss.executeWorkSearch(esQuery, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Search failed",
			"details": err.Error(),
		})
		return
	}

	// Record search analytics
	go ss.recordSearch(c.Request.Context(), req.Query, "works", response.Total)

	response.SearchTime = time.Since(start).Milliseconds()
	c.JSON(http.StatusOK, response)
}

func (ss *SearchService) AdvancedWorkSearch(c *gin.Context) {
	var req WorkSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	start := time.Now()

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	if req.Status == "" {
		req.Status = "all"
	}
	if req.SortBy == "" {
		req.SortBy = "relevance"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// Build Elasticsearch query
	esQuery := ss.buildWorkSearchQuery(req)

	// Execute search
	response, err := ss.executeWorkSearch(esQuery, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Advanced search failed",
			"details": err.Error(),
		})
		return
	}

	// Record search analytics
	go ss.recordSearch(c.Request.Context(), req.Query, "works_advanced", response.Total)

	response.SearchTime = time.Since(start).Milliseconds()
	c.JSON(http.StatusOK, response)
}

func (ss *SearchService) buildWorkSearchQuery(req WorkSearchRequest) map[string]interface{} {
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must":   []map[string]interface{}{},
			"filter": []map[string]interface{}{},
		},
	}

	must := query["bool"].(map[string]interface{})["must"].([]map[string]interface{})
	filter := query["bool"].(map[string]interface{})["filter"].([]map[string]interface{})

	// Always filter by status if specified
	if req.Status != "" && req.Status != "all" {
		var statusValue string
		switch req.Status {
		case "complete", "posted":
			statusValue = "posted"
		case "wip", "in_progress":
			statusValue = "draft"
		case "hidden":
			statusValue = "hidden"
		default:
			statusValue = req.Status
		}
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				"status": statusValue,
			},
		})
	}

	// Text search queries
	if req.Query != "" {
		must = append(must, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":    req.Query,
				"fields":   []string{"title^3", "summary^2", "content_text", "fandoms", "characters", "relationships", "freeform_tags"},
				"type":     "best_fields",
				"operator": "or",
			},
		})
	}

	if req.Title != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"title": map[string]interface{}{
					"query":    req.Title,
					"operator": "and",
				},
			},
		})
	}

	if req.Author != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"author": map[string]interface{}{
					"query":    req.Author,
					"operator": "and",
				},
			},
		})
	}

	// Tag filters
	if len(req.Fandoms) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string]interface{}{
				"fandoms": req.Fandoms,
			},
		})
	}

	if len(req.Characters) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string]interface{}{
				"characters": req.Characters,
			},
		})
	}

	if len(req.Relationships) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string]interface{}{
				"relationships": req.Relationships,
			},
		})
	}

	if len(req.Tags) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string]interface{}{
				"freeform_tags": req.Tags,
			},
		})
	}

	// Metadata filters
	if len(req.Rating) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string]interface{}{
				"rating": req.Rating,
			},
		})
	}

	if len(req.Category) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string]interface{}{
				"categories": req.Category,
			},
		})
	}

	if len(req.Warnings) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string]interface{}{
				"warnings": req.Warnings,
			},
		})
	}

	if len(req.Language) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string]interface{}{
				"language": req.Language,
			},
		})
	}

	// Numeric range filters
	if req.WordCountMin != nil || req.WordCountMax != nil {
		rangeQuery := map[string]interface{}{}
		if req.WordCountMin != nil {
			rangeQuery["gte"] = *req.WordCountMin
		}
		if req.WordCountMax != nil {
			rangeQuery["lte"] = *req.WordCountMax
		}
		filter = append(filter, map[string]interface{}{
			"range": map[string]interface{}{
				"word_count": rangeQuery,
			},
		})
	}

	// Date range filters
	if req.UpdatedAfter != "" || req.UpdatedBefore != "" {
		rangeQuery := map[string]interface{}{}
		if req.UpdatedAfter != "" {
			rangeQuery["gte"] = req.UpdatedAfter
		}
		if req.UpdatedBefore != "" {
			rangeQuery["lte"] = req.UpdatedBefore
		}
		filter = append(filter, map[string]interface{}{
			"range": map[string]interface{}{
				"updated_at": rangeQuery,
			},
		})
	}

	// Relationship count filter
	if req.RelationshipCount != "" {
		var minCount, maxCount int
		switch req.RelationshipCount {
		case "1-2":
			minCount, maxCount = 1, 2
		case "3-5":
			minCount, maxCount = 3, 5
		case "6-10":
			minCount, maxCount = 6, 10
		case "10+":
			minCount = 10
			// No max limit for 10+
		}

		if minCount > 0 {
			// Use script query to count relationship tags
			scriptQuery := map[string]interface{}{
				"script": map[string]interface{}{
					"source": "doc['relationships'].size() >= params.min" +
						func() string {
							if req.RelationshipCount != "10+" {
								return " && doc['relationships'].size() <= params.max"
							}
							return ""
						}(),
					"params": map[string]interface{}{
						"min": minCount,
					},
				},
			}
			if req.RelationshipCount != "10+" {
				scriptQuery["script"].(map[string]interface{})["params"].(map[string]interface{})["max"] = maxCount
			}
			filter = append(filter, scriptQuery)
		}
	}

	// Tag prominence filter (requires integration with tag prominence system)
	if req.TagProminence != "" && len(req.Relationships) > 0 {
		// For now, implement basic prominence filtering
		// In the future, this would query the tag_prominence_rules table
		switch req.TagProminence {
		case "primary":
			// Only show works where searched relationships are primary
			// This would require ES document to include prominence data
			log.Printf("Primary prominence filter requested - requires tag prominence data in ES")
		case "secondary":
			// Show works where relationships are secondary or higher
			log.Printf("Secondary prominence filter requested - requires tag prominence data in ES")
		case "any":
			// Default behavior - no additional filtering needed
		}
	}

	// Content filtering - crossover detection (basic implementation)
	if req.HideCrossovers {
		// Add script query to detect crossovers
		// This is a simplified version - full implementation would use tag relationships table
		crossoverScript := map[string]interface{}{
			"script": map[string]interface{}{
				"source": `
					// Simple crossover detection based on known universe families
					def fandoms = doc['fandoms'];
					if (fandoms.size() <= 1) return true; // Not a crossover if 1 or fewer fandoms
					
					// Define universe families (simplified)
					def marvelTags = ['Marvel', 'Marvel Cinematic Universe', 'Thor (Marvel)', 'Iron Man (Movies)'];
					def harryPotterTags = ['Harry Potter - J. K. Rowling', 'Harry Potter (Movies)'];
					def dcTags = ['DC Comics', 'Batman - All Media Types', 'Superman - All Media Types'];
					
					def universes = new HashSet();
					for (fandom in fandoms) {
						def fandomStr = fandom.toString().toLowerCase();
						if (marvelTags.stream().anyMatch(tag -> fandomStr.contains(tag.toLowerCase()))) {
							universes.add('marvel');
						} else if (harryPotterTags.stream().anyMatch(tag -> fandomStr.contains(tag.toLowerCase()))) {
							universes.add('harry_potter');
						} else if (dcTags.stream().anyMatch(tag -> fandomStr.contains(tag.toLowerCase()))) {
							universes.add('dc');
						} else {
							universes.add(fandomStr); // Treat as unique universe
						}
					}
					
					return universes.size() <= 1; // True if same universe family
				`,
			},
		}
		filter = append(filter, crossoverScript)
		log.Printf("Crossover filtering enabled - excluding works with multiple universe families")
	}

	// Other content filtering
	if req.HideIncomplete {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				"is_complete": true,
			},
		})
	}

	if req.HideNoRelationships {
		filter = append(filter, map[string]interface{}{
			"exists": map[string]interface{}{
				"field": "relationships",
			},
		})
		filter = append(filter, map[string]interface{}{
			"script": map[string]interface{}{
				"script": map[string]interface{}{
					"source": "doc['relationships'].size() > 0",
				},
			},
		})
	}

	// Date filtering
	if req.UpdatedWithin != "" {
		var days int
		switch req.UpdatedWithin {
		case "week":
			days = 7
		case "month":
			days = 30
		case "3months":
			days = 90
		case "year":
			days = 365
		}

		if days > 0 {
			filter = append(filter, map[string]interface{}{
				"range": map[string]interface{}{
					"updated_at": map[string]interface{}{
						"gte": fmt.Sprintf("now-%dd", days),
					},
				},
			})
		}
	}

	// Engagement filtering
	if req.MinKudos != nil && *req.MinKudos > 0 {
		filter = append(filter, map[string]interface{}{
			"range": map[string]interface{}{
				"kudos_count": map[string]interface{}{
					"gte": *req.MinKudos,
				},
			},
		})
	}

	if req.MinComments != nil && *req.MinComments > 0 {
		filter = append(filter, map[string]interface{}{
			"range": map[string]interface{}{
				"comments_count": map[string]interface{}{
					"gte": *req.MinComments,
				},
			},
		})
	}

	if req.MinBookmarks != nil && *req.MinBookmarks > 0 {
		filter = append(filter, map[string]interface{}{
			"range": map[string]interface{}{
				"bookmarks_count": map[string]interface{}{
					"gte": *req.MinBookmarks,
				},
			},
		})
	}

	// Blocked tags filtering
	if len(req.BlockedTags) > 0 {
		for _, blockedTag := range req.BlockedTags {
			// Add must_not clause for each blocked tag
			mustNot := query["bool"].(map[string]interface{})["must_not"]
			if mustNot == nil {
				query["bool"].(map[string]interface{})["must_not"] = []map[string]interface{}{}
				mustNot = query["bool"].(map[string]interface{})["must_not"]
			}

			mustNotSlice := mustNot.([]map[string]interface{})
			mustNotSlice = append(mustNotSlice, map[string]interface{}{
				"multi_match": map[string]interface{}{
					"query":  blockedTag,
					"fields": []string{"fandoms", "characters", "relationships", "freeform_tags"},
				},
			})
			query["bool"].(map[string]interface{})["must_not"] = mustNotSlice
		}
	}

	// If no search conditions, use match_all to return all documents
	if len(must) == 0 && len(filter) == 0 {
		query = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	} else if len(must) == 0 {
		// If only filters but no search text, use match_all with filters
		query = map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"match_all": map[string]interface{}{}},
				},
				"filter": filter,
			},
		}
	} else {
		// Update the query with the built filters
		query["bool"].(map[string]interface{})["must"] = must
		query["bool"].(map[string]interface{})["filter"] = filter
	}

	result := map[string]interface{}{
		"query": query,
		"sort":  ss.buildSortClause(req.SortBy, req.SortOrder),
		"size":  req.Limit,
		"from":  (req.Page - 1) * req.Limit,
		"aggs":  ss.buildWorksFacets(),
	}

	// Query logging can be enabled for debugging if needed

	return result
}

func (ss *SearchService) buildSortClause(sortBy, sortOrder string) []map[string]interface{} {
	// Validate sort order
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	switch sortBy {
	case "relevance":
		return []map[string]interface{}{
			{"_score": map[string]interface{}{"order": "desc"}},
		}
	case "updated_at":
		return []map[string]interface{}{
			{"updated_at": map[string]interface{}{"order": sortOrder}},
		}
	case "published_at":
		return []map[string]interface{}{
			{"published_at": map[string]interface{}{"order": sortOrder}},
		}
	case "word_count":
		return []map[string]interface{}{
			{"word_count": map[string]interface{}{"order": sortOrder}},
		}
	case "title":
		return []map[string]interface{}{
			{"title.keyword": map[string]interface{}{"order": sortOrder}},
		}
	case "author":
		return []map[string]interface{}{
			{"author.keyword": map[string]interface{}{"order": sortOrder}},
		}
	// Traditional engagement metrics (gaming-prone)
	case "hits":
		return []map[string]interface{}{
			{"hits_count": map[string]interface{}{"order": sortOrder}},
		}
	case "kudos":
		return []map[string]interface{}{
			{"kudos_count": map[string]interface{}{"order": sortOrder}},
		}
	case "comments":
		return []map[string]interface{}{
			{"comments_count": map[string]interface{}{"order": sortOrder}},
		}
	case "bookmarks":
		return []map[string]interface{}{
			{"bookmarks_count": map[string]interface{}{"order": sortOrder}},
		}
	// Smart anti-gaming engagement metrics
	case "quality_score":
		// Balanced quality score that resists gaming
		return []map[string]interface{}{
			{
				"_script": map[string]interface{}{
					"type": "number",
					"script": map[string]interface{}{
						"source": `
							// Anti-gaming quality algorithm
							def hits = Math.max(doc['hits_count'].value, 1);
							def kudos = doc['kudos_count'].value;
							def comments = doc['comments_count'].value;
							def bookmarks = doc['bookmarks_count'].value;
							def wordCount = Math.max(doc['word_count'].value, 1);
							
							// Engagement ratios (resist raw number gaming)
							def kudosRatio = Math.min(kudos / hits, 0.5); // Cap at 50%
							def commentRatio = Math.min(comments / hits, 0.1); // Cap at 10%
							def bookmarkRatio = Math.min(bookmarks / hits, 0.2); // Cap at 20%
							
							// Length normalization (prevent short-fic gaming)
							def lengthBonus = Math.min(Math.log10(wordCount / 1000.0 + 1), 2.0);
							
							// Quality indicators
							def hasComments = comments > 0 ? 1.2 : 1.0;
							def hasBookmarks = bookmarks > 0 ? 1.1 : 1.0;
							
							// Combined quality score
							return (kudosRatio * 100 + commentRatio * 200 + bookmarkRatio * 150) 
								   * lengthBonus * hasComments * hasBookmarks;
						`,
					},
					"order": sortOrder,
				},
			},
		}
	case "engagement_rate":
		// Engagement rate that normalizes by exposure
		return []map[string]interface{}{
			{
				"_script": map[string]interface{}{
					"type": "number",
					"script": map[string]interface{}{
						"source": `
							def hits = Math.max(doc['hits_count'].value, 1);
							def kudos = doc['kudos_count'].value;
							def comments = doc['comments_count'].value;
							def bookmarks = doc['bookmarks_count'].value;
							
							// Calculate engagement rate as percentage
							def totalEngagement = kudos + (comments * 2) + (bookmarks * 3);
							return (totalEngagement / hits) * 100;
						`,
					},
					"order": sortOrder,
				},
			},
		}
	case "comment_quality":
		// Prioritize works that generate discussion
		return []map[string]interface{}{
			{
				"_script": map[string]interface{}{
					"type": "number",
					"script": map[string]interface{}{
						"source": `
							def hits = Math.max(doc['hits_count'].value, 1);
							def comments = doc['comments_count'].value;
							def wordCount = Math.max(doc['word_count'].value, 1);
							
							// Comments per reader, normalized by length
							def commentDensity = (comments / hits) * 100;
							def lengthFactor = Math.log10(wordCount / 1000.0 + 1);
							
							return commentDensity * lengthFactor;
						`,
					},
					"order": sortOrder,
				},
			},
		}
	case "discovery_boost":
		// Boost newer works and works with recent activity
		return []map[string]interface{}{
			{
				"_script": map[string]interface{}{
					"type": "number",
					"script": map[string]interface{}{
						"source": `
							def now = System.currentTimeMillis();
							def publishedTime = doc['published_at'].value.getMillis();
							def updatedTime = doc['updated_at'].value.getMillis();
							def hits = Math.max(doc['hits_count'].value, 1);
							def kudos = doc['kudos_count'].value;
							
							// Recency bonuses (30 days = full bonus)
							def daysSincePublished = (now - publishedTime) / (1000.0 * 60 * 60 * 24);
							def daysSinceUpdated = (now - updatedTime) / (1000.0 * 60 * 60 * 24);
							
							def publishBonus = Math.max(0, (30 - daysSincePublished) / 30.0);
							def updateBonus = Math.max(0, (7 - daysSinceUpdated) / 7.0);
							
							// Base quality
							def baseQuality = Math.min(kudos / hits, 0.3) * 100;
							
							// Discovery score
							return baseQuality * (1 + publishBonus + updateBonus);
						`,
					},
					"order": sortOrder,
				},
			},
		}
	default:
		return []map[string]interface{}{
			{"updated_at": map[string]interface{}{"order": "desc"}},
		}
	}
}

func (ss *SearchService) buildWorksFacets() map[string]interface{} {
	return map[string]interface{}{
		"fandoms": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "fandoms",
				"size":  20,
			},
		},
		"ratings": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "rating",
				"size":  10,
			},
		},
		"categories": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "categories",
				"size":  10,
			},
		},
		"warnings": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "warnings",
				"size":  10,
			},
		},
		"languages": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "language",
				"size":  20,
			},
		},
		"word_count_ranges": map[string]interface{}{
			"range": map[string]interface{}{
				"field": "word_count",
				"ranges": []map[string]interface{}{
					{"key": "short", "to": 1000},
					{"key": "medium", "from": 1000, "to": 10000},
					{"key": "long", "from": 10000, "to": 50000},
					{"key": "epic", "from": 50000},
				},
			},
		},
	}
}

func (ss *SearchService) executeWorkSearch(query map[string]interface{}, req WorkSearchRequest) (*SearchResponse, error) {
	// Convert query to JSON
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Execute search
	res, err := ss.es.Search(
		ss.es.Search.WithContext(context.Background()),
		ss.es.Search.WithIndex("works"),
		ss.es.Search.WithBody(bytes.NewReader(queryJSON)),
		ss.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search returned error: %s", res.String())
	}

	// Parse response
	var esResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract results
	hits := esResponse["hits"].(map[string]interface{})
	total := int(hits["total"].(map[string]interface{})["value"].(float64))

	results := []map[string]interface{}{}
	for _, hit := range hits["hits"].([]interface{}) {
		hitMap := hit.(map[string]interface{})
		source := hitMap["_source"].(map[string]interface{})

		// Add score and highlight if available
		if score, ok := hitMap["_score"]; ok {
			source["_score"] = score
		}
		if highlight, ok := hitMap["highlight"]; ok {
			source["_highlight"] = highlight
		}

		results = append(results, source)
	}

	// Extract facets
	facets := map[string]interface{}{}
	if aggs, ok := esResponse["aggregations"]; ok {
		facets = aggs.(map[string]interface{})
	}

	pages := (total + req.Limit - 1) / req.Limit

	return &SearchResponse{
		Results: results,
		Total:   total,
		Page:    req.Page,
		Limit:   req.Limit,
		Pages:   pages,
		Facets:  facets,
	}, nil
}

// Suggestion and autocomplete handlers

func (ss *SearchService) GetSuggestions(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	suggestType := c.DefaultQuery("type", "all") // all, works, tags, authors
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit > 50 {
		limit = 50
	}

	// Check cache first
	cacheKey := fmt.Sprintf("suggestions:%s:%s:%d", query, suggestType, limit)
	cached, err := ss.redis.Get(c.Request.Context(), cacheKey).Result()
	if err == nil {
		var suggestions map[string]interface{}
		if json.Unmarshal([]byte(cached), &suggestions) == nil {
			c.JSON(http.StatusOK, suggestions)
			return
		}
	}

	// Build suggestions query
	suggestQuery := map[string]interface{}{
		"suggest": map[string]interface{}{
			"work_title_suggest": map[string]interface{}{
				"prefix": query,
				"completion": map[string]interface{}{
					"field": "title_suggest",
					"size":  limit,
				},
			},
			"tag_suggest": map[string]interface{}{
				"prefix": query,
				"completion": map[string]interface{}{
					"field": "tag_suggest",
					"size":  limit,
				},
			},
			"author_suggest": map[string]interface{}{
				"prefix": query,
				"completion": map[string]interface{}{
					"field": "author_suggest",
					"size":  limit,
				},
			},
		},
	}

	queryJSON, _ := json.Marshal(suggestQuery)

	res, err := ss.es.Search(
		ss.es.Search.WithContext(context.Background()),
		ss.es.Search.WithIndex("works,tags,users"),
		ss.es.Search.WithBody(bytes.NewReader(queryJSON)),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Suggestion request failed"})
		return
	}
	defer res.Body.Close()

	var esResponse map[string]interface{}
	json.NewDecoder(res.Body).Decode(&esResponse)

	// Extract suggestions
	suggestions := map[string]interface{}{
		"works":   []string{},
		"tags":    []string{},
		"authors": []string{},
	}

	if suggest, ok := esResponse["suggest"]; ok {
		suggestMap := suggest.(map[string]interface{})

		// Extract work title suggestions
		if workSuggest, ok := suggestMap["work_title_suggest"]; ok {
			workList := workSuggest.([]interface{})
			if len(workList) > 0 {
				options := workList[0].(map[string]interface{})["options"].([]interface{})
				workTitles := []string{}
				for _, opt := range options {
					text := opt.(map[string]interface{})["text"].(string)
					workTitles = append(workTitles, text)
				}
				suggestions["works"] = workTitles
			}
		}

		// Extract tag suggestions
		if tagSuggest, ok := suggestMap["tag_suggest"]; ok {
			tagList := tagSuggest.([]interface{})
			if len(tagList) > 0 {
				options := tagList[0].(map[string]interface{})["options"].([]interface{})
				tags := []string{}
				for _, opt := range options {
					text := opt.(map[string]interface{})["text"].(string)
					tags = append(tags, text)
				}
				suggestions["tags"] = tags
			}
		}

		// Extract author suggestions
		if authorSuggest, ok := suggestMap["author_suggest"]; ok {
			authorList := authorSuggest.([]interface{})
			if len(authorList) > 0 {
				options := authorList[0].(map[string]interface{})["options"].([]interface{})
				authors := []string{}
				for _, opt := range options {
					text := opt.(map[string]interface{})["text"].(string)
					authors = append(authors, text)
				}
				suggestions["authors"] = authors
			}
		}
	}

	// Cache result
	if suggJSON, err := json.Marshal(suggestions); err == nil {
		ss.redis.Set(c.Request.Context(), cacheKey, suggJSON, time.Minute*15)
	}

	c.JSON(http.StatusOK, suggestions)
}

// Analytics helper
func (ss *SearchService) recordSearch(ctx context.Context, query, searchType string, results int) {
	if query == "" {
		return
	}

	// Record to Redis for analytics
	date := time.Now().Format("2006-01-02")

	// Increment search count
	ss.redis.Incr(ctx, fmt.Sprintf("search_stats:%s:count", date))

	// Record popular terms
	ss.redis.ZIncrBy(ctx, fmt.Sprintf("popular_terms:%s", date), 1, query)

	// Record zero result queries
	if results == 0 {
		ss.redis.ZIncrBy(ctx, fmt.Sprintf("zero_results:%s", date), 1, query)
	}

	// Set expiration
	ss.redis.Expire(ctx, fmt.Sprintf("search_stats:%s:count", date), time.Hour*24*30)
	ss.redis.Expire(ctx, fmt.Sprintf("popular_terms:%s", date), time.Hour*24*30)
	ss.redis.Expire(ctx, fmt.Sprintf("zero_results:%s", date), time.Hour*24*30)
}

// Additional search implementations

func (ss *SearchService) SearchTags(c *gin.Context) {
	query := c.DefaultQuery("q", "")
	tagType := c.QueryArray("type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Build Elasticsearch query for tags
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{},
			},
		},
		"size": limit,
		"from": offset,
		"sort": []map[string]interface{}{
			{"use_count": map[string]interface{}{"order": "desc"}},
			{"name.keyword": map[string]interface{}{"order": "asc"}},
		},
	}

	must := esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"].([]map[string]interface{})

	if query != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"name": map[string]interface{}{
					"query":    query,
					"operator": "and",
				},
			},
		})
	}

	if len(tagType) > 0 {
		must = append(must, map[string]interface{}{
			"terms": map[string]interface{}{
				"type": tagType,
			},
		})
	}

	esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = must

	queryJSON, _ := json.Marshal(esQuery)
	res, err := ss.es.Search(
		ss.es.Search.WithContext(context.Background()),
		ss.es.Search.WithIndex("tags"),
		ss.es.Search.WithBody(bytes.NewReader(queryJSON)),
		ss.es.Search.WithTrackTotalHits(true),
	)
	if err != nil || res.IsError() {
		c.JSON(http.StatusOK, gin.H{"tags": []gin.H{}, "total": 0})
		return
	}
	defer res.Body.Close()

	var esResponse map[string]interface{}
	json.NewDecoder(res.Body).Decode(&esResponse)

	hits := esResponse["hits"].(map[string]interface{})
	total := int(hits["total"].(map[string]interface{})["value"].(float64))

	results := []map[string]interface{}{}
	for _, hit := range hits["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		results = append(results, source)
	}

	c.JSON(http.StatusOK, gin.H{
		"tags":  results,
		"total": total,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (ss *SearchService) SearchUsers(c *gin.Context) {
	query := c.DefaultQuery("q", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if query == "" {
		c.JSON(http.StatusOK, gin.H{"users": []gin.H{}, "total": 0})
		return
	}

	// Build Elasticsearch query for users
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"username^2", "display_name", "profile.bio"},
				"type":   "best_fields",
			},
		},
		"size": limit,
		"from": offset,
		"sort": []map[string]interface{}{
			{"_score": map[string]interface{}{"order": "desc"}},
			{"username.keyword": map[string]interface{}{"order": "asc"}},
		},
		"_source": []string{"id", "username", "display_name", "created_at"},
	}

	queryJSON, _ := json.Marshal(esQuery)
	res, err := ss.es.Search(
		ss.es.Search.WithContext(context.Background()),
		ss.es.Search.WithIndex("users"),
		ss.es.Search.WithBody(bytes.NewReader(queryJSON)),
		ss.es.Search.WithTrackTotalHits(true),
	)
	if err != nil || res.IsError() {
		c.JSON(http.StatusOK, gin.H{"users": []gin.H{}, "total": 0})
		return
	}
	defer res.Body.Close()

	var esResponse map[string]interface{}
	json.NewDecoder(res.Body).Decode(&esResponse)

	hits := esResponse["hits"].(map[string]interface{})
	total := int(hits["total"].(map[string]interface{})["value"].(float64))

	results := []map[string]interface{}{}
	for _, hit := range hits["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		results = append(results, source)
	}

	c.JSON(http.StatusOK, gin.H{
		"users": results,
		"total": total,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (ss *SearchService) SearchCollections(c *gin.Context) {
	query := c.DefaultQuery("q", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Build Elasticsearch query for collections
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{},
			},
		},
		"size": limit,
		"from": offset,
		"sort": []map[string]interface{}{
			{"work_count": map[string]interface{}{"order": "desc"}},
			{"title.keyword": map[string]interface{}{"order": "asc"}},
		},
		"_source": []string{"id", "name", "title", "description", "work_count", "is_open", "created_at"},
	}

	must := esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"].([]map[string]interface{})

	if query != "" {
		must = append(must, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"title^2", "name^2", "description"},
				"type":   "best_fields",
			},
		})
	}

	esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = must

	queryJSON, _ := json.Marshal(esQuery)
	res, err := ss.es.Search(
		ss.es.Search.WithContext(context.Background()),
		ss.es.Search.WithIndex("collections"),
		ss.es.Search.WithBody(bytes.NewReader(queryJSON)),
		ss.es.Search.WithTrackTotalHits(true),
	)
	if err != nil || res.IsError() {
		c.JSON(http.StatusOK, gin.H{"collections": []gin.H{}, "total": 0})
		return
	}
	defer res.Body.Close()

	var esResponse map[string]interface{}
	json.NewDecoder(res.Body).Decode(&esResponse)

	hits := esResponse["hits"].(map[string]interface{})
	total := int(hits["total"].(map[string]interface{})["value"].(float64))

	results := []map[string]interface{}{}
	for _, hit := range hits["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		results = append(results, source)
	}

	c.JSON(http.StatusOK, gin.H{
		"collections": results,
		"total":       total,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (ss *SearchService) SearchSeries(c *gin.Context) {
	query := c.DefaultQuery("q", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Build Elasticsearch query for series
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{},
			},
		},
		"size": limit,
		"from": offset,
		"sort": []map[string]interface{}{
			{"work_count": map[string]interface{}{"order": "desc"}},
			{"title.keyword": map[string]interface{}{"order": "asc"}},
		},
		"_source": []string{"id", "title", "summary", "author", "work_count", "word_count", "is_complete", "updated_at"},
	}

	must := esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"].([]map[string]interface{})

	if query != "" {
		must = append(must, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"title^2", "summary", "author"},
				"type":   "best_fields",
			},
		})
	}

	esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = must

	queryJSON, _ := json.Marshal(esQuery)
	res, err := ss.es.Search(
		ss.es.Search.WithContext(context.Background()),
		ss.es.Search.WithIndex("series"),
		ss.es.Search.WithBody(bytes.NewReader(queryJSON)),
		ss.es.Search.WithTrackTotalHits(true),
	)
	if err != nil || res.IsError() {
		c.JSON(http.StatusOK, gin.H{"series": []gin.H{}, "total": 0})
		return
	}
	defer res.Body.Close()

	var esResponse map[string]interface{}
	json.NewDecoder(res.Body).Decode(&esResponse)

	hits := esResponse["hits"].(map[string]interface{})
	total := int(hits["total"].(map[string]interface{})["value"].(float64))

	results := []map[string]interface{}{}
	for _, hit := range hits["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		results = append(results, source)
	}

	c.JSON(http.StatusOK, gin.H{
		"series": results,
		"total":  total,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (ss *SearchService) AdvancedTagSearch(c *gin.Context) {
	var req struct {
		Query  string   `json:"query"`
		Type   []string `json:"type"`
		Fandom []string `json:"fandom"`
		Limit  int      `json:"limit"`
		Offset int      `json:"offset"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	// Build advanced tag search query
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must":   []map[string]interface{}{},
				"filter": []map[string]interface{}{},
			},
		},
		"size": req.Limit,
		"from": req.Offset,
		"sort": []map[string]interface{}{
			{"use_count": map[string]interface{}{"order": "desc"}},
		},
	}

	must := esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"].([]map[string]interface{})
	filter := esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"].([]map[string]interface{})

	if req.Query != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"name": req.Query,
			},
		})
	}

	if len(req.Type) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string]interface{}{
				"type": req.Type,
			},
		})
	}

	if len(req.Fandom) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string]interface{}{
				"fandoms.keyword": req.Fandom,
			},
		})
	}

	esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = must
	esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = filter

	queryJSON, _ := json.Marshal(esQuery)
	res, err := ss.es.Search(
		ss.es.Search.WithContext(context.Background()),
		ss.es.Search.WithIndex("tags"),
		ss.es.Search.WithBody(bytes.NewReader(queryJSON)),
		ss.es.Search.WithTrackTotalHits(true),
	)

	if err != nil || res.IsError() {
		c.JSON(http.StatusOK, gin.H{"tags": []gin.H{}, "total": 0})
		return
	}
	defer res.Body.Close()

	var esResponse map[string]interface{}
	json.NewDecoder(res.Body).Decode(&esResponse)

	hits := esResponse["hits"].(map[string]interface{})
	total := int(hits["total"].(map[string]interface{})["value"].(float64))

	results := []map[string]interface{}{}
	for _, hit := range hits["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		results = append(results, source)
	}

	c.JSON(http.StatusOK, gin.H{
		"tags":  results,
		"total": total,
	})
}

func (ss *SearchService) GetPopularSearches(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit > 50 {
		limit = 50
	}

	// Get popular searches from Redis
	date := time.Now().Format("2006-01-02")
	popularTerms, err := ss.redis.ZRevRange(c.Request.Context(),
		fmt.Sprintf("popular_terms:%s", date), 0, int64(limit-1)).Result()

	if err != nil {
		// Fallback to weekly popular terms
		weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
		popularTerms, _ = ss.redis.ZRevRange(c.Request.Context(),
			fmt.Sprintf("popular_terms:%s", weekAgo), 0, int64(limit-1)).Result()
	}

	c.JSON(http.StatusOK, gin.H{"searches": popularTerms})
}

func (ss *SearchService) GetTrendingSearches(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit > 50 {
		limit = 50
	}

	// Calculate trending based on recent activity vs historical
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// Get today's popular terms
	todayTerms, err := ss.redis.ZRevRangeWithScores(c.Request.Context(),
		fmt.Sprintf("popular_terms:%s", today), 0, int64(limit*2-1)).Result()

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"searches": []string{}})
		return
	}

	// Get yesterday's scores for comparison
	trendingTerms := []string{}
	for _, term := range todayTerms {
		member := term.Member.(string)
		todayScore := term.Score

		yesterdayScore, _ := ss.redis.ZScore(c.Request.Context(),
			fmt.Sprintf("popular_terms:%s", yesterday), member).Result()

		// Calculate trend ratio (today/yesterday)
		if yesterdayScore == 0 || todayScore/yesterdayScore > 1.5 {
			trendingTerms = append(trendingTerms, member)
			if len(trendingTerms) >= limit {
				break
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"searches": trendingTerms})
}

// Placeholder implementations for remaining handlers

func (ss *SearchService) IndexWork(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Work indexed"})
}

func (ss *SearchService) UpdateWorkIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Work index updated"})
}

func (ss *SearchService) DeleteWorkIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Work removed from index"})
}

func (ss *SearchService) BulkIndexWorks(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Works bulk indexed"})
}

func (ss *SearchService) IndexTag(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tag indexed"})
}

func (ss *SearchService) UpdateTagIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tag index updated"})
}

func (ss *SearchService) DeleteTagIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tag removed from index"})
}

func (ss *SearchService) BulkIndexTags(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Tags bulk indexed"})
}

func (ss *SearchService) IndexUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User indexed"})
}

func (ss *SearchService) UpdateUserIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User index updated"})
}

func (ss *SearchService) DeleteUserIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User removed from index"})
}

func (ss *SearchService) RebuildIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Index rebuild started"})
}

func (ss *SearchService) ReindexAll(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Reindex all started"})
}

func (ss *SearchService) GetIndexStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "indices": gin.H{}})
}

func (ss *SearchService) OptimizeIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Index optimization started"})
}

func (ss *SearchService) GetSearchStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"stats": gin.H{}})
}

func (ss *SearchService) GetPopularTerms(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"terms": []gin.H{}})
}

func (ss *SearchService) GetZeroResultTerms(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"terms": []gin.H{}})
}

func (ss *SearchService) GetSearchPerformance(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"performance": gin.H{}})
}

func (ss *SearchService) GetSearchHistory(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"history": []gin.H{}})
}

func (ss *SearchService) ClearSearchHistory(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Search history cleared"})
}

func (ss *SearchService) SaveSearch(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"search": gin.H{}})
}

func (ss *SearchService) GetSavedSearches(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"searches": []gin.H{}})
}

func (ss *SearchService) DeleteSavedSearch(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Saved search deleted"})
}

func (ss *SearchService) CreateSearchAlert(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"alert": gin.H{}})
}

func (ss *SearchService) GetFandomFilters(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"filters": []gin.H{}})
}

func (ss *SearchService) GetCharacterFilters(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"filters": []gin.H{}})
}

func (ss *SearchService) GetRelationshipFilters(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"filters": []gin.H{}})
}

func (ss *SearchService) GetTagFilters(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"filters": []gin.H{}})
}

func (ss *SearchService) GetStatFilters(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"filters": gin.H{
		"word_count_ranges": []gin.H{
			{"label": "Short (< 1,000 words)", "min": 0, "max": 999},
			{"label": "Medium (1,000 - 9,999 words)", "min": 1000, "max": 9999},
			{"label": "Long (10,000 - 49,999 words)", "min": 10000, "max": 49999},
			{"label": "Epic (50,000+ words)", "min": 50000, "max": nil},
		},
	}})
}
