package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// REAL-TIME INDEXING OPTIMIZATION - Task 2
// High-performance bulk indexing with queue management and efficient pipelines
// =============================================================================

// IndexingJobType represents the type of indexing operation
type IndexingJobType string

const (
	IndexingJobCreate IndexingJobType = "create"
	IndexingJobUpdate IndexingJobType = "update"
	IndexingJobDelete IndexingJobType = "delete"
)

// IndexingJob represents a work indexing operation
type IndexingJob struct {
	ID         string          `json:"id"`
	Type       IndexingJobType `json:"type"`
	WorkID     string          `json:"work_id"`
	WorkData   interface{}     `json:"work_data,omitempty"`
	Priority   int             `json:"priority"` // 1-5, higher = more urgent
	Timestamp  time.Time       `json:"timestamp"`
	Retries    int             `json:"retries"`
	MaxRetries int             `json:"max_retries"`
}

// BulkIndexingRequest represents a batch of indexing operations
type BulkIndexingRequest struct {
	Jobs    []IndexingJob       `json:"jobs"`
	Options BulkIndexingOptions `json:"options"`
}

// BulkIndexingOptions configures bulk indexing behavior
type BulkIndexingOptions struct {
	BatchSize      int           `json:"batch_size"`      // Number of operations per batch
	FlushInterval  time.Duration `json:"flush_interval"`  // Max time to wait before flushing
	MaxConcurrency int           `json:"max_concurrency"` // Max concurrent indexing workers
	RefreshPolicy  string        `json:"refresh_policy"`  // "true", "false", "wait_for"
	EnableRetries  bool          `json:"enable_retries"`  // Enable automatic retries
	RetryDelay     time.Duration `json:"retry_delay"`     // Delay between retries
}

// IndexingQueueStatus represents the current state of the indexing queue
type IndexingQueueStatus struct {
	PendingJobs        int                        `json:"pending_jobs"`
	ProcessingJobs     int                        `json:"processing_jobs"`
	CompletedJobs      int                        `json:"completed_jobs"`
	FailedJobs         int                        `json:"failed_jobs"`
	QueueBacklog       time.Duration              `json:"queue_backlog"`
	WorkerStatuses     []IndexingWorkerStatus     `json:"worker_statuses"`
	PerformanceMetrics IndexingPerformanceMetrics `json:"performance_metrics"`
}

// IndexingWorkerStatus represents the status of an individual indexing worker
type IndexingWorkerStatus struct {
	WorkerID      string       `json:"worker_id"`
	Status        string       `json:"status"` // "idle", "processing", "error"
	CurrentJob    *IndexingJob `json:"current_job,omitempty"`
	JobsProcessed int          `json:"jobs_processed"`
	LastActivity  time.Time    `json:"last_activity"`
	ErrorCount    int          `json:"error_count"`
}

// IndexingPerformanceMetrics tracks indexing performance
type IndexingPerformanceMetrics struct {
	IndexingRate     float64       `json:"indexing_rate"`     // Jobs per second
	AverageLatency   time.Duration `json:"average_latency"`   // Average time per job
	P95Latency       time.Duration `json:"p95_latency"`       // 95th percentile latency
	ThroughputMBPS   float64       `json:"throughput_mbps"`   // MB per second indexed
	ErrorRate        float64       `json:"error_rate"`        // Percentage of failed jobs
	QueueUtilization float64       `json:"queue_utilization"` // Percentage of queue capacity used
}

// WorkIndexDocument represents a work document optimized for Elasticsearch indexing
type WorkIndexDocument struct {
	WorkID           string    `json:"work_id"`
	Title            string    `json:"title"`
	Summary          string    `json:"summary"`
	Content          string    `json:"content,omitempty"`
	AuthorIDs        []string  `json:"author_ids"`
	AuthorNames      []string  `json:"author_names"`
	Fandoms          []string  `json:"fandoms"`
	Characters       []string  `json:"characters"`
	Relationships    []string  `json:"relationships"`
	AdditionalTags   []string  `json:"additional_tags"`
	Warnings         []string  `json:"warnings"`
	Categories       []string  `json:"categories"`
	Rating           string    `json:"rating"`
	Language         string    `json:"language"`
	WordCount        int       `json:"word_count"`
	ChapterCount     int       `json:"chapter_count"`
	CompletionStatus string    `json:"completion_status"`
	PublishedDate    time.Time `json:"published_date"`
	UpdatedDate      time.Time `json:"updated_date"`
	Hits             int       `json:"hits"`
	Kudos            int       `json:"kudos"`
	Comments         int       `json:"comments"`
	Bookmarks        int       `json:"bookmarks"`
	Collections      []string  `json:"collections"`
	Series           []string  `json:"series"`
	IsRestricted     bool      `json:"is_restricted"`
	IsAnonymous      bool      `json:"is_anonymous"`
	IndexedAt        time.Time `json:"indexed_at"`
	Version          int       `json:"version"`

	// Performance optimization fields
	PopularityScore     float64        `json:"popularity_score"`
	RecentActivityScore float64        `json:"recent_activity_score"`
	SearchableText      string         `json:"searchable_text"` // Pre-combined searchable content
	TagFrequency        map[string]int `json:"tag_frequency"`   // Tag frequency for boosting

	// Advanced fields for search optimization
	ContentLength    int     `json:"content_length"`
	UniqueTagCount   int     `json:"unique_tag_count"`
	AuthorReputation float64 `json:"author_reputation"`
	FandomPopularity float64 `json:"fandom_popularity"`

	// Task 3: Tag quality and smart filtering fields
	TaggingQualityScore   float64  `json:"tagging_quality_score"`   // 0-100 score for overall tag quality
	TagInconsistencyCount int      `json:"tag_inconsistency_count"` // Number of detected tagging issues
	MissingTagCount       int      `json:"missing_tag_count"`       // Estimated number of missing tags
	CrossTaggingScore     float64  `json:"cross_tagging_score"`     // How well character/relationship tags align
	TagCompletenessScore  float64  `json:"tag_completeness_score"`  // How complete the tagging appears
	ImpliedCharacters     []string `json:"implied_characters"`      // Characters implied by relationships
	ImpliedRelationships  []string `json:"implied_relationships"`   // Relationships implied by characters
}

// =============================================================================
// ENHANCED SEARCH SERVICE - Advanced Query Builders & Optimizations
// Production-ready search with sophisticated filtering and performance optimization
// =============================================================================

// Enhanced search request types with advanced filtering
type EnhancedWorkSearchRequest struct {
	// Basic search
	Query   string `json:"query,omitempty"`
	Title   string `json:"title,omitempty"`
	Author  string `json:"author,omitempty"`
	Summary string `json:"summary,omitempty"`

	// Tag filters with advanced logic
	Fandoms       []string `json:"fandoms,omitempty"`
	Characters    []string `json:"characters,omitempty"`
	Relationships []string `json:"relationships,omitempty"`
	FreeformTags  []string `json:"freeform_tags,omitempty"`

	// Tag filter logic
	FandomLogic       string `json:"fandom_logic,omitempty"`       // "any", "all", "exclude"
	CharacterLogic    string `json:"character_logic,omitempty"`    // "any", "all", "exclude"
	RelationshipLogic string `json:"relationship_logic,omitempty"` // "any", "all", "exclude"
	TagLogic          string `json:"tag_logic,omitempty"`          // "any", "all", "exclude"

	// Metadata filters
	Rating   []string `json:"rating,omitempty"`
	Category []string `json:"category,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Status   []string `json:"status,omitempty"`
	Language []string `json:"language,omitempty"`

	// Completion status
	IsComplete *bool `json:"is_complete,omitempty"`
	InProgress *bool `json:"in_progress,omitempty"`

	// Numeric filters with ranges
	WordCountMin *int `json:"word_count_min,omitempty"`
	WordCountMax *int `json:"word_count_max,omitempty"`
	ChapterMin   *int `json:"chapter_min,omitempty"`
	ChapterMax   *int `json:"chapter_max,omitempty"`
	HitsMin      *int `json:"hits_min,omitempty"`
	HitsMax      *int `json:"hits_max,omitempty"`
	KudosMin     *int `json:"kudos_min,omitempty"`
	KudosMax     *int `json:"kudos_max,omitempty"`
	CommentsMin  *int `json:"comments_min,omitempty"`
	CommentsMax  *int `json:"comments_max,omitempty"`
	BookmarksMin *int `json:"bookmarks_min,omitempty"`
	BookmarksMax *int `json:"bookmarks_max,omitempty"`

	// Date filters
	PublishedAfter  string `json:"published_after,omitempty"`
	PublishedBefore string `json:"published_before,omitempty"`
	UpdatedAfter    string `json:"updated_after,omitempty"`
	UpdatedBefore   string `json:"updated_before,omitempty"`

	// Search behavior
	ExactMatch   bool     `json:"exact_match,omitempty"`
	SearchFields []string `json:"search_fields,omitempty"` // custom field weights
	BoostRecent  bool     `json:"boost_recent,omitempty"`
	BoostPopular bool     `json:"boost_popular,omitempty"`

	// Sorting and pagination
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
	Page      int    `json:"page,omitempty"`
	Limit     int    `json:"limit,omitempty"`

	// Advanced options
	IncludeFacets bool     `json:"include_facets,omitempty"`
	Highlighting  bool     `json:"highlighting,omitempty"`
	Suggestions   bool     `json:"suggestions,omitempty"`
	ExcludeWorks  []string `json:"exclude_works,omitempty"` // Work IDs to exclude
}

// Enhanced response with detailed metadata
type EnhancedSearchResponse struct {
	Results    []map[string]interface{} `json:"results"`
	Total      int                      `json:"total"`
	Page       int                      `json:"page"`
	Limit      int                      `json:"limit"`
	Pages      int                      `json:"pages"`
	SearchTime int64                    `json:"search_time_ms"`
	QueryTime  int64                    `json:"query_time_ms"`

	// Advanced features
	Facets      map[string]interface{} `json:"facets,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	DidYouMean  string                 `json:"did_you_mean,omitempty"`

	// Search metadata
	QueryParsed map[string]interface{} `json:"query_parsed,omitempty"`
	Filters     map[string]interface{} `json:"filters_applied,omitempty"`
	Performance map[string]interface{} `json:"performance,omitempty"`
}

// =============================================================================
// ENHANCED WORK SEARCH WITH ADVANCED QUERY BUILDING
// =============================================================================

func (ss *SearchService) EnhancedWorkSearch(c *gin.Context) {
	start := time.Now()

	var req EnhancedWorkSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Set intelligent defaults
	req = ss.setSearchDefaults(req)

	// Validate request
	if err := ss.validateSearchRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	queryStart := time.Now()

	// Build advanced Elasticsearch query
	esQuery := ss.buildAdvancedWorkQuery(req)

	queryTime := time.Since(queryStart).Milliseconds()

	// Execute search with caching
	response, err := ss.executeAdvancedSearch(esQuery, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Enhanced search failed",
			"details": err.Error(),
		})
		return
	}

	// Add performance metadata
	response.SearchTime = time.Since(start).Milliseconds()
	response.QueryTime = queryTime
	response.Performance = map[string]interface{}{
		"cached":        false, // TODO: implement caching detection
		"query_complex": ss.calculateQueryComplexity(req),
		"result_score":  ss.calculateResultScore(response),
	}

	// Record enhanced analytics
	go ss.recordEnhancedSearch(c.Request.Context(), req, response)

	c.JSON(http.StatusOK, response)
}

// =============================================================================
// ADVANCED QUERY BUILDERS
// =============================================================================

func (ss *SearchService) buildAdvancedWorkQuery(req EnhancedWorkSearchRequest) map[string]interface{} {
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must":     []map[string]interface{}{},
			"filter":   []map[string]interface{}{},
			"should":   []map[string]interface{}{},
			"must_not": []map[string]interface{}{},
		},
	}

	must := query["bool"].(map[string]interface{})["must"].([]map[string]interface{})
	filter := query["bool"].(map[string]interface{})["filter"].([]map[string]interface{})
	should := query["bool"].(map[string]interface{})["should"].([]map[string]interface{})
	mustNot := query["bool"].(map[string]interface{})["must_not"].([]map[string]interface{})

	// Build text search queries with advanced matching
	if req.Query != "" {
		textQuery := ss.buildTextQuery(req)
		must = append(must, textQuery)
	}

	// Build title search
	if req.Title != "" {
		titleQuery := ss.buildTitleQuery(req.Title, req.ExactMatch)
		must = append(must, titleQuery)
	}

	// Build author search
	if req.Author != "" {
		authorQuery := ss.buildAuthorQuery(req.Author, req.ExactMatch)
		must = append(must, authorQuery)
	}

	// Build summary search
	if req.Summary != "" {
		summaryQuery := ss.buildSummaryQuery(req.Summary, req.ExactMatch)
		must = append(must, summaryQuery)
	}

	// Build advanced tag filters
	tagFilters := ss.buildAdvancedTagFilters(req)
	filter = append(filter, tagFilters...)

	// Build metadata filters
	metadataFilters := ss.buildMetadataFilters(req)
	filter = append(filter, metadataFilters...)

	// Build numeric range filters
	numericFilters := ss.buildNumericFilters(req)
	filter = append(filter, numericFilters...)

	// Build date range filters
	dateFilters := ss.buildDateFilters(req)
	filter = append(filter, dateFilters...)

	// Build boost queries for relevance
	boostQueries := ss.buildBoostQueries(req)
	should = append(should, boostQueries...)

	// Build exclusion filters
	exclusionFilters := ss.buildExclusionFilters(req)
	mustNot = append(mustNot, exclusionFilters...)

	// Update query components
	query["bool"].(map[string]interface{})["must"] = must
	query["bool"].(map[string]interface{})["filter"] = filter
	query["bool"].(map[string]interface{})["should"] = should
	query["bool"].(map[string]interface{})["must_not"] = mustNot

	// Add minimum should match for boost queries
	if len(should) > 0 {
		query["bool"].(map[string]interface{})["minimum_should_match"] = 0
	}

	// Build complete search request
	searchRequest := map[string]interface{}{
		"query":   query,
		"sort":    ss.buildAdvancedSortClause(req),
		"size":    req.Limit,
		"from":    (req.Page - 1) * req.Limit,
		"_source": ss.buildSourceFields(req),
	}

	// Add facets if requested
	if req.IncludeFacets {
		searchRequest["aggs"] = ss.buildAdvancedFacets(req)
	}

	// Add highlighting if requested
	if req.Highlighting {
		searchRequest["highlight"] = ss.buildHighlighting(req)
	}

	// Add suggestions if requested
	if req.Suggestions {
		searchRequest["suggest"] = ss.buildSuggestions(req)
	}

	return searchRequest
}

// =============================================================================
// SPECIALIZED QUERY BUILDERS
// =============================================================================

func (ss *SearchService) buildTextQuery(req EnhancedWorkSearchRequest) map[string]interface{} {
	if req.ExactMatch {
		return map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":    req.Query,
				"fields":   ss.getSearchFields(req),
				"type":     "phrase",
				"operator": "and",
			},
		}
	}

	// Advanced multi-match with custom scoring
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []map[string]interface{}{
				{
					"multi_match": map[string]interface{}{
						"query":    req.Query,
						"fields":   ss.getSearchFields(req),
						"type":     "best_fields",
						"operator": "and",
						"boost":    2.0,
					},
				},
				{
					"multi_match": map[string]interface{}{
						"query":  req.Query,
						"fields": ss.getSearchFields(req),
						"type":   "phrase",
						"slop":   2,
						"boost":  1.5,
					},
				},
				{
					"multi_match": map[string]interface{}{
						"query":    req.Query,
						"fields":   ss.getSearchFields(req),
						"type":     "cross_fields",
						"operator": "or",
						"boost":    1.0,
					},
				},
			},
		},
	}
}

func (ss *SearchService) buildTitleQuery(title string, exactMatch bool) map[string]interface{} {
	if exactMatch {
		return map[string]interface{}{
			"match_phrase": map[string]interface{}{
				"title": title,
			},
		}
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []map[string]interface{}{
				{
					"match_phrase": map[string]interface{}{
						"title": map[string]interface{}{
							"query": title,
							"boost": 3.0,
						},
					},
				},
				{
					"match": map[string]interface{}{
						"title": map[string]interface{}{
							"query":    title,
							"operator": "and",
							"boost":    2.0,
						},
					},
				},
				{
					"match": map[string]interface{}{
						"title": map[string]interface{}{
							"query":    title,
							"operator": "or",
							"boost":    1.0,
						},
					},
				},
			},
		},
	}
}

func (ss *SearchService) buildAuthorQuery(author string, exactMatch bool) map[string]interface{} {
	if exactMatch {
		return map[string]interface{}{
			"term": map[string]interface{}{
				"author.keyword": author,
			},
		}
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []map[string]interface{}{
				{
					"term": map[string]interface{}{
						"author.keyword": map[string]interface{}{
							"value": author,
							"boost": 3.0,
						},
					},
				},
				{
					"match": map[string]interface{}{
						"author": map[string]interface{}{
							"query":    author,
							"operator": "and",
							"boost":    2.0,
						},
					},
				},
				{
					"wildcard": map[string]interface{}{
						"author.keyword": map[string]interface{}{
							"value": fmt.Sprintf("*%s*", strings.ToLower(author)),
							"boost": 1.0,
						},
					},
				},
			},
		},
	}
}

func (ss *SearchService) buildSummaryQuery(summary string, exactMatch bool) map[string]interface{} {
	if exactMatch {
		return map[string]interface{}{
			"match_phrase": map[string]interface{}{
				"summary": summary,
			},
		}
	}

	return map[string]interface{}{
		"match": map[string]interface{}{
			"summary": map[string]interface{}{
				"query":    summary,
				"operator": "and",
			},
		},
	}
}

// =============================================================================
// ADVANCED TAG FILTERING
// =============================================================================

func (ss *SearchService) buildAdvancedTagFilters(req EnhancedWorkSearchRequest) []map[string]interface{} {
	filters := []map[string]interface{}{}

	// Fandom filters with logic
	if len(req.Fandoms) > 0 {
		fandomFilter := ss.buildTagFilter("fandoms.keyword", req.Fandoms, req.FandomLogic)
		filters = append(filters, fandomFilter)
	}

	// Character filters with logic
	if len(req.Characters) > 0 {
		characterFilter := ss.buildTagFilter("characters.keyword", req.Characters, req.CharacterLogic)
		filters = append(filters, characterFilter)
	}

	// Relationship filters with logic
	if len(req.Relationships) > 0 {
		relationshipFilter := ss.buildTagFilter("relationships.keyword", req.Relationships, req.RelationshipLogic)
		filters = append(filters, relationshipFilter)
	}

	// Freeform tag filters with logic
	if len(req.FreeformTags) > 0 {
		tagFilter := ss.buildTagFilter("freeform_tags.keyword", req.FreeformTags, req.TagLogic)
		filters = append(filters, tagFilter)
	}

	return filters
}

func (ss *SearchService) buildTagFilter(field string, tags []string, logic string) map[string]interface{} {
	switch logic {
	case "all":
		// Must contain ALL specified tags
		filters := []map[string]interface{}{}
		for _, tag := range tags {
			filters = append(filters, map[string]interface{}{
				"term": map[string]interface{}{
					field: tag,
				},
			})
		}
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must": filters,
			},
		}

	case "exclude":
		// Must NOT contain any of these tags
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": []map[string]interface{}{
					{
						"terms": map[string]interface{}{
							field: tags,
						},
					},
				},
			},
		}

	default: // "any" or unspecified
		// Must contain at least one of these tags
		return map[string]interface{}{
			"terms": map[string]interface{}{
				field: tags,
			},
		}
	}
}

// =============================================================================
// FILTER BUILDERS
// =============================================================================

func (ss *SearchService) buildMetadataFilters(req EnhancedWorkSearchRequest) []map[string]interface{} {
	filters := []map[string]interface{}{}

	if len(req.Rating) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{
				"rating": req.Rating,
			},
		})
	}

	if len(req.Category) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{
				"category": req.Category,
			},
		})
	}

	if len(req.Warnings) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{
				"warnings": req.Warnings,
			},
		})
	}

	if len(req.Status) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{
				"status": req.Status,
			},
		})
	}

	if len(req.Language) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{
				"language": req.Language,
			},
		})
	}

	// Completion status filters
	if req.IsComplete != nil {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"is_complete": *req.IsComplete,
			},
		})
	}

	if req.InProgress != nil {
		// In progress means not complete and has multiple chapters or is ongoing
		if *req.InProgress {
			filters = append(filters, map[string]interface{}{
				"bool": map[string]interface{}{
					"must": []map[string]interface{}{
						{
							"term": map[string]interface{}{
								"is_complete": false,
							},
						},
					},
				},
			})
		}
	}

	return filters
}

func (ss *SearchService) buildNumericFilters(req EnhancedWorkSearchRequest) []map[string]interface{} {
	filters := []map[string]interface{}{}

	// Word count range
	if req.WordCountMin != nil || req.WordCountMax != nil {
		rangeQuery := map[string]interface{}{}
		if req.WordCountMin != nil {
			rangeQuery["gte"] = *req.WordCountMin
		}
		if req.WordCountMax != nil {
			rangeQuery["lte"] = *req.WordCountMax
		}
		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"word_count": rangeQuery,
			},
		})
	}

	// Chapter count range
	if req.ChapterMin != nil || req.ChapterMax != nil {
		rangeQuery := map[string]interface{}{}
		if req.ChapterMin != nil {
			rangeQuery["gte"] = *req.ChapterMin
		}
		if req.ChapterMax != nil {
			rangeQuery["lte"] = *req.ChapterMax
		}
		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"chapter_count": rangeQuery,
			},
		})
	}

	// Statistics filters
	filters = append(filters, ss.buildStatisticFilters(req)...)

	return filters
}

func (ss *SearchService) buildStatisticFilters(req EnhancedWorkSearchRequest) []map[string]interface{} {
	filters := []map[string]interface{}{}

	// Hits range
	if req.HitsMin != nil || req.HitsMax != nil {
		rangeQuery := map[string]interface{}{}
		if req.HitsMin != nil {
			rangeQuery["gte"] = *req.HitsMin
		}
		if req.HitsMax != nil {
			rangeQuery["lte"] = *req.HitsMax
		}
		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"hits": rangeQuery,
			},
		})
	}

	// Kudos range
	if req.KudosMin != nil || req.KudosMax != nil {
		rangeQuery := map[string]interface{}{}
		if req.KudosMin != nil {
			rangeQuery["gte"] = *req.KudosMin
		}
		if req.KudosMax != nil {
			rangeQuery["lte"] = *req.KudosMax
		}
		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"kudos": rangeQuery,
			},
		})
	}

	// Comments range
	if req.CommentsMin != nil || req.CommentsMax != nil {
		rangeQuery := map[string]interface{}{}
		if req.CommentsMin != nil {
			rangeQuery["gte"] = *req.CommentsMin
		}
		if req.CommentsMax != nil {
			rangeQuery["lte"] = *req.CommentsMax
		}
		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"comments": rangeQuery,
			},
		})
	}

	// Bookmarks range
	if req.BookmarksMin != nil || req.BookmarksMax != nil {
		rangeQuery := map[string]interface{}{}
		if req.BookmarksMin != nil {
			rangeQuery["gte"] = *req.BookmarksMin
		}
		if req.BookmarksMax != nil {
			rangeQuery["lte"] = *req.BookmarksMax
		}
		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"bookmarks": rangeQuery,
			},
		})
	}

	return filters
}

func (ss *SearchService) buildDateFilters(req EnhancedWorkSearchRequest) []map[string]interface{} {
	filters := []map[string]interface{}{}

	// Published date range
	if req.PublishedAfter != "" || req.PublishedBefore != "" {
		rangeQuery := map[string]interface{}{}
		if req.PublishedAfter != "" {
			rangeQuery["gte"] = req.PublishedAfter
		}
		if req.PublishedBefore != "" {
			rangeQuery["lte"] = req.PublishedBefore
		}
		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"published_at": rangeQuery,
			},
		})
	}

	// Updated date range
	if req.UpdatedAfter != "" || req.UpdatedBefore != "" {
		rangeQuery := map[string]interface{}{}
		if req.UpdatedAfter != "" {
			rangeQuery["gte"] = req.UpdatedAfter
		}
		if req.UpdatedBefore != "" {
			rangeQuery["lte"] = req.UpdatedBefore
		}
		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"updated_at": rangeQuery,
			},
		})
	}

	return filters
}

// =============================================================================
// BOOST AND RELEVANCE BUILDERS
// =============================================================================

func (ss *SearchService) buildBoostQueries(req EnhancedWorkSearchRequest) []map[string]interface{} {
	boosts := []map[string]interface{}{}

	// Boost recent works
	if req.BoostRecent {
		boosts = append(boosts, map[string]interface{}{
			"function_score": map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
				"functions": []map[string]interface{}{
					{
						"gauss": map[string]interface{}{
							"updated_at": map[string]interface{}{
								"origin": time.Now().Format("2006-01-02"),
								"scale":  "30d",
								"decay":  0.5,
							},
						},
						"weight": 1.5,
					},
				},
				"boost_mode": "multiply",
			},
		})
	}

	// Boost popular works
	if req.BoostPopular {
		boosts = append(boosts, map[string]interface{}{
			"function_score": map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
				"functions": []map[string]interface{}{
					{
						"field_value_factor": map[string]interface{}{
							"field":    "kudos",
							"factor":   0.1,
							"modifier": "log1p",
						},
						"weight": 1.2,
					},
					{
						"field_value_factor": map[string]interface{}{
							"field":    "bookmarks",
							"factor":   0.2,
							"modifier": "log1p",
						},
						"weight": 1.3,
					},
				},
				"boost_mode": "multiply",
			},
		})
	}

	return boosts
}

func (ss *SearchService) buildExclusionFilters(req EnhancedWorkSearchRequest) []map[string]interface{} {
	exclusions := []map[string]interface{}{}

	// Exclude specific works
	if len(req.ExcludeWorks) > 0 {
		exclusions = append(exclusions, map[string]interface{}{
			"ids": map[string]interface{}{
				"values": req.ExcludeWorks,
			},
		})
	}

	return exclusions
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func (ss *SearchService) setSearchDefaults(req EnhancedWorkSearchRequest) EnhancedWorkSearchRequest {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	if req.SortBy == "" {
		req.SortBy = "relevance"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}
	if req.FandomLogic == "" {
		req.FandomLogic = "any"
	}
	if req.CharacterLogic == "" {
		req.CharacterLogic = "any"
	}
	if req.RelationshipLogic == "" {
		req.RelationshipLogic = "any"
	}
	if req.TagLogic == "" {
		req.TagLogic = "any"
	}

	return req
}

func (ss *SearchService) validateSearchRequest(req EnhancedWorkSearchRequest) error {
	if req.Limit > 100 {
		return fmt.Errorf("limit cannot exceed 100")
	}
	if req.Page < 1 {
		return fmt.Errorf("page must be >= 1")
	}

	validLogic := map[string]bool{
		"any": true, "all": true, "exclude": true,
	}

	if !validLogic[req.FandomLogic] {
		return fmt.Errorf("invalid fandom_logic: must be 'any', 'all', or 'exclude'")
	}
	if !validLogic[req.CharacterLogic] {
		return fmt.Errorf("invalid character_logic: must be 'any', 'all', or 'exclude'")
	}
	if !validLogic[req.RelationshipLogic] {
		return fmt.Errorf("invalid relationship_logic: must be 'any', 'all', or 'exclude'")
	}
	if !validLogic[req.TagLogic] {
		return fmt.Errorf("invalid tag_logic: must be 'any', 'all', or 'exclude'")
	}

	return nil
}

func (ss *SearchService) getSearchFields(req EnhancedWorkSearchRequest) []string {
	if len(req.SearchFields) > 0 {
		return req.SearchFields
	}

	// Default search fields with weights
	return []string{
		"title^3",
		"summary^2",
		"content^1",
		"tags^1.5",
		"author^1.2",
	}
}

func (ss *SearchService) buildAdvancedSortClause(req EnhancedWorkSearchRequest) []map[string]interface{} {
	sortOrder := req.SortOrder
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	switch req.SortBy {
	case "relevance":
		return []map[string]interface{}{
			{"_score": map[string]interface{}{"order": "desc"}},
			{"updated_at": map[string]interface{}{"order": "desc"}}, // tiebreaker
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
	case "kudos":
		return []map[string]interface{}{
			{"kudos": map[string]interface{}{"order": sortOrder}},
		}
	case "hits":
		return []map[string]interface{}{
			{"hits": map[string]interface{}{"order": sortOrder}},
		}
	case "comments":
		return []map[string]interface{}{
			{"comments": map[string]interface{}{"order": sortOrder}},
		}
	case "bookmarks":
		return []map[string]interface{}{
			{"bookmarks": map[string]interface{}{"order": sortOrder}},
		}
	default:
		return []map[string]interface{}{
			{"updated_at": map[string]interface{}{"order": "desc"}},
		}
	}
}

func (ss *SearchService) buildSourceFields(req EnhancedWorkSearchRequest) []string {
	return []string{
		"id", "title", "summary", "author", "author_id",
		"fandoms", "characters", "relationships", "freeform_tags",
		"rating", "category", "warnings", "language", "status",
		"word_count", "chapter_count", "is_complete",
		"published_at", "updated_at",
		"hits", "kudos", "comments", "bookmarks",
	}
}

func (ss *SearchService) buildAdvancedFacets(req EnhancedWorkSearchRequest) map[string]interface{} {
	return map[string]interface{}{
		"fandoms": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "fandoms.keyword",
				"size":  50,
				"order": map[string]interface{}{"_count": "desc"},
			},
		},
		"characters": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "characters.keyword",
				"size":  30,
				"order": map[string]interface{}{"_count": "desc"},
			},
		},
		"relationships": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "relationships.keyword",
				"size":  30,
				"order": map[string]interface{}{"_count": "desc"},
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
				"field": "category",
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
		"completion_status": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "is_complete",
				"size":  2,
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
		"publish_year": map[string]interface{}{
			"date_histogram": map[string]interface{}{
				"field":    "published_at",
				"interval": "year",
				"format":   "yyyy",
			},
		},
	}
}

func (ss *SearchService) buildHighlighting(req EnhancedWorkSearchRequest) map[string]interface{} {
	return map[string]interface{}{
		"fields": map[string]interface{}{
			"title": map[string]interface{}{
				"fragment_size":       100,
				"number_of_fragments": 1,
			},
			"summary": map[string]interface{}{
				"fragment_size":       150,
				"number_of_fragments": 2,
			},
			"content": map[string]interface{}{
				"fragment_size":       200,
				"number_of_fragments": 3,
			},
		},
		"pre_tags":  []string{"<mark>"},
		"post_tags": []string{"</mark>"},
	}
}

func (ss *SearchService) buildSuggestions(req EnhancedWorkSearchRequest) map[string]interface{} {
	suggestions := map[string]interface{}{}

	if req.Query != "" {
		suggestions["text"] = map[string]interface{}{
			"text": req.Query,
			"term": map[string]interface{}{
				"field": "title",
			},
		}
	}

	return suggestions
}

// =============================================================================
// PERFORMANCE AND ANALYTICS
// =============================================================================

func (ss *SearchService) calculateQueryComplexity(req EnhancedWorkSearchRequest) int {
	complexity := 0

	// Base query complexity
	if req.Query != "" {
		complexity += 2
	}
	if req.Title != "" {
		complexity += 1
	}
	if req.Author != "" {
		complexity += 1
	}
	if req.Summary != "" {
		complexity += 1
	}

	// Tag filter complexity
	complexity += len(req.Fandoms)
	complexity += len(req.Characters)
	complexity += len(req.Relationships)
	complexity += len(req.FreeformTags)

	// Advanced logic complexity
	if req.FandomLogic == "all" {
		complexity += len(req.Fandoms)
	}
	if req.CharacterLogic == "all" {
		complexity += len(req.Characters)
	}
	if req.RelationshipLogic == "all" {
		complexity += len(req.Relationships)
	}
	if req.TagLogic == "all" {
		complexity += len(req.FreeformTags)
	}

	// Numeric and date filters
	if req.WordCountMin != nil || req.WordCountMax != nil {
		complexity += 1
	}
	if req.PublishedAfter != "" || req.PublishedBefore != "" {
		complexity += 1
	}
	if req.UpdatedAfter != "" || req.UpdatedBefore != "" {
		complexity += 1
	}

	// Boost queries
	if req.BoostRecent {
		complexity += 2
	}
	if req.BoostPopular {
		complexity += 2
	}

	// Features
	if req.IncludeFacets {
		complexity += 3
	}
	if req.Highlighting {
		complexity += 1
	}
	if req.Suggestions {
		complexity += 2
	}

	return complexity
}

func (ss *SearchService) calculateResultScore(response *EnhancedSearchResponse) float64 {
	if response.Total == 0 {
		return 0.0
	}

	// Calculate result quality score based on various factors
	score := 1.0

	// Penalize very high result counts (too broad)
	if response.Total > 10000 {
		score *= 0.8
	} else if response.Total > 1000 {
		score *= 0.9
	}

	// Reward reasonable result counts
	if response.Total >= 10 && response.Total <= 1000 {
		score *= 1.1
	}

	// Penalize slow searches
	if response.SearchTime > 1000 { // > 1 second
		score *= 0.7
	} else if response.SearchTime > 500 {
		score *= 0.9
	} else if response.SearchTime < 100 {
		score *= 1.2 // Very fast
	}

	return score
}

func (ss *SearchService) recordEnhancedSearch(ctx context.Context, req EnhancedWorkSearchRequest, resp *EnhancedSearchResponse) {
	// Record detailed search analytics
	date := time.Now().Format("2006-01-02")

	// Record search metrics
	ss.redis.Incr(ctx, fmt.Sprintf("enhanced_search_stats:%s:count", date))
	ss.redis.IncrBy(ctx, fmt.Sprintf("enhanced_search_stats:%s:results", date), int64(resp.Total))
	ss.redis.IncrBy(ctx, fmt.Sprintf("enhanced_search_stats:%s:time", date), resp.SearchTime)

	// Record popular filters
	if len(req.Fandoms) > 0 {
		for _, fandom := range req.Fandoms {
			ss.redis.ZIncrBy(ctx, fmt.Sprintf("popular_fandoms:%s", date), 1, fandom)
		}
	}

	if len(req.Characters) > 0 {
		for _, character := range req.Characters {
			ss.redis.ZIncrBy(ctx, fmt.Sprintf("popular_characters:%s", date), 1, character)
		}
	}

	if len(req.Relationships) > 0 {
		for _, relationship := range req.Relationships {
			ss.redis.ZIncrBy(ctx, fmt.Sprintf("popular_relationships:%s", date), 1, relationship)
		}
	}

	// Record query complexity
	complexity := ss.calculateQueryComplexity(req)
	ss.redis.IncrBy(ctx, fmt.Sprintf("search_complexity:%s:total", date), int64(complexity))
	ss.redis.Incr(ctx, fmt.Sprintf("search_complexity:%s:count", date))

	// Set expiration
	ss.redis.Expire(ctx, fmt.Sprintf("enhanced_search_stats:%s:count", date), time.Hour*24*30)
	ss.redis.Expire(ctx, fmt.Sprintf("popular_fandoms:%s", date), time.Hour*24*30)
	ss.redis.Expire(ctx, fmt.Sprintf("popular_characters:%s", date), time.Hour*24*30)
	ss.redis.Expire(ctx, fmt.Sprintf("popular_relationships:%s", date), time.Hour*24*30)
	ss.redis.Expire(ctx, fmt.Sprintf("search_complexity:%s:total", date), time.Hour*24*30)
}

// =============================================================================
// ADVANCED SEARCH EXECUTION
// =============================================================================

func (ss *SearchService) executeAdvancedSearch(query map[string]interface{}, req EnhancedWorkSearchRequest) (*EnhancedSearchResponse, error) {
	// Convert query to JSON
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Execute search with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := ss.es.Search(
		ss.es.Search.WithContext(ctx),
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

		// Add score
		if score, ok := hitMap["_score"]; ok {
			source["_score"] = score
		}

		// Add highlights
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

	// Extract suggestions
	suggestions := []string{}
	didYouMean := ""
	if suggest, ok := esResponse["suggest"]; ok {
		suggestMap := suggest.(map[string]interface{})
		if textSuggest, ok := suggestMap["text"]; ok {
			textList := textSuggest.([]interface{})
			if len(textList) > 0 {
				options := textList[0].(map[string]interface{})["options"].([]interface{})
				if len(options) > 0 {
					didYouMean = options[0].(map[string]interface{})["text"].(string)
				}
			}
		}
	}

	pages := (total + req.Limit - 1) / req.Limit

	response := &EnhancedSearchResponse{
		Results:     results,
		Total:       total,
		Page:        req.Page,
		Limit:       req.Limit,
		Pages:       pages,
		Facets:      facets,
		Suggestions: suggestions,
		DidYouMean:  didYouMean,
		QueryParsed: map[string]interface{}{
			"complexity": ss.calculateQueryComplexity(req),
			"fields":     ss.getSearchFields(req),
		},
		Filters: map[string]interface{}{
			"fandoms":       req.Fandoms,
			"characters":    req.Characters,
			"relationships": req.Relationships,
			"tags":          req.FreeformTags,
			"rating":        req.Rating,
			"category":      req.Category,
			"warnings":      req.Warnings,
		},
	}

	return response, nil
}

// =============================================================================
// TASK 3: ENHANCED FILTERING & FACETS WITH SMART TAG ENHANCEMENT
// Advanced filtering with intelligent tag consistency and cross-tagging detection
// =============================================================================

// SmartTagEnhancement represents intelligent tag suggestions and corrections
type SmartTagEnhancement struct {
	MissingCharacters    []string               `json:"missing_characters"`    // Characters implied by relationships but not tagged
	MissingRelationships []string               `json:"missing_relationships"` // Relationships implied by characters but not tagged
	SuggestedTags        []TagSuggestion        `json:"suggested_tags"`        // AI-suggested additional tags
	TagInconsistencies   []TagInconsistency     `json:"tag_inconsistencies"`   // Detected tagging problems
	TaggingQualityScore  float64                `json:"tagging_quality_score"` // 0-100 score for tag completeness
	SmartFilters         map[string]interface{} `json:"smart_filters"`         // Enhanced filters based on tag analysis
}

// TagSuggestion represents a suggested tag with confidence score
type TagSuggestion struct {
	Tag        string  `json:"tag"`
	Type       string  `json:"type"`       // "character", "relationship", "freeform"
	Confidence float64 `json:"confidence"` // 0-1 confidence score
	Reason     string  `json:"reason"`     // Why this tag was suggested
}

// TagInconsistency represents a detected tagging problem
type TagInconsistency struct {
	Type        string   `json:"type"`        // "missing_character", "missing_relationship", "contradictory"
	Description string   `json:"description"` // Human-readable description
	Suggestions []string `json:"suggestions"` // Suggested fixes
	Severity    string   `json:"severity"`    // "low", "medium", "high"
}

// AdvancedFilterRequest represents enhanced filtering with smart tag analysis
type AdvancedFilterRequest struct {
	// Base search parameters
	Query         string   `json:"query"`
	Fandoms       []string `json:"fandoms"`
	Characters    []string `json:"characters"`
	Relationships []string `json:"relationships"`
	FreeformTags  []string `json:"freeform_tags"`

	// Tag quality filters
	MinTaggingQuality   float64 `json:"min_tagging_quality"`   // Filter out poorly tagged works
	RequireComplete     bool    `json:"require_complete"`      // Only show works with complete character/relationship tagging
	ExcludePoorlyTagged bool    `json:"exclude_poorly_tagged"` // Exclude works with tag inconsistencies

	// Smart enhancement options
	EnableSmartTagging bool `json:"enable_smart_tagging"` // Use AI to enhance tag searches
	AutoExpandTags     bool `json:"auto_expand_tags"`     // Automatically include implied tags
	SuggestMissingTags bool `json:"suggest_missing_tags"` // Return tag suggestions with results

	// Advanced facet configuration
	FacetConfig FacetConfiguration `json:"facet_config"`

	// Filter combination logic
	TagLogic    string `json:"tag_logic"`    // "and", "or", "custom"
	CustomLogic string `json:"custom_logic"` // Custom boolean logic expression
}

// FacetConfiguration controls facet generation and display
type FacetConfiguration struct {
	EnableSmartFacets bool `json:"enable_smart_facets"` // Generate intelligent facet suggestions
	MaxFacetValues    int  `json:"max_facet_values"`    // Max values per facet
	MinDocCount       int  `json:"min_doc_count"`       // Minimum documents for facet inclusion
	ExcludeRareTags   bool `json:"exclude_rare_tags"`   // Hide tags with < threshold occurrences
	RareTagThreshold  int  `json:"rare_tag_threshold"`  // Threshold for rare tag exclusion
	GroupSimilarTags  bool `json:"group_similar_tags"`  // Group synonymous/similar tags
	PrioritizePopular bool `json:"prioritize_popular"`  // Show most popular facets first
	IncludeTagMetrics bool `json:"include_tag_metrics"` // Include usage statistics in facets
}

// FacetValue represents a standard facet value with count
type FacetValue struct {
	Value      string  `json:"value"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// SmartFacetResponse represents enhanced facet data with intelligence
type SmartFacetResponse struct {
	StandardFacets    map[string][]FacetValue `json:"standard_facets"`
	SmartFacets       map[string][]SmartFacet `json:"smart_facets"`
	TagSuggestions    []TagSuggestion         `json:"tag_suggestions"`
	FilterSuggestions []FilterSuggestion      `json:"filter_suggestions"`
	QualityMetrics    map[string]interface{}  `json:"quality_metrics"`
}

// SmartFacet represents an intelligent facet with enhanced metadata
type SmartFacet struct {
	Value       string                 `json:"value"`
	Count       int                    `json:"count"`
	Percentage  float64                `json:"percentage"`
	Trend       string                 `json:"trend"`       // "rising", "stable", "declining"
	Quality     float64                `json:"quality"`     // Tag quality score for this facet
	Suggestions []string               `json:"suggestions"` // Related tag suggestions
	Metadata    map[string]interface{} `json:"metadata"`    // Additional facet metadata
}

// FilterSuggestion represents an intelligent filter recommendation
type FilterSuggestion struct {
	Type       string  `json:"type"`       // "add_filter", "modify_filter", "remove_filter"
	Field      string  `json:"field"`      // Field to filter on
	Value      string  `json:"value"`      // Suggested filter value
	Reason     string  `json:"reason"`     // Why this filter is suggested
	Impact     int     `json:"impact"`     // Estimated result count change
	Confidence float64 `json:"confidence"` // Confidence in suggestion (0-1)
}

// TagQualityAnalysis represents comprehensive tag analysis for filtering
type TagQualityAnalysis struct {
	Completeness    float64 `json:"completeness"`      // How complete the tagging is (0-1)
	Consistency     float64 `json:"consistency"`       // How consistent tags are (0-1)
	Specificity     float64 `json:"specificity"`       // How specific/detailed tags are (0-1)
	CrossTagging    float64 `json:"cross_tagging"`     // How well relationship/character tags align (0-1)
	OverallScore    float64 `json:"overall_score"`     // Combined quality score (0-100)
	IssueCount      int     `json:"issue_count"`       // Number of tagging issues detected
	MissingTagCount int     `json:"missing_tag_count"` // Estimated missing tags
	QualityCategory string  `json:"quality_category"`  // "excellent", "good", "fair", "poor"
}

// =============================================================================
// REAL-TIME INDEXING PIPELINE IMPLEMENTATION
// =============================================================================

// EnhancedBulkIndexWorks handles bulk indexing of multiple works with optimized performance
func (ss *SearchService) EnhancedBulkIndexWorks(c *gin.Context) {
	var req BulkIndexingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid bulk indexing request",
			"details": err.Error(),
		})
		return
	}

	// Validate and set default options
	if req.Options.BatchSize == 0 {
		req.Options.BatchSize = 100
	}
	if req.Options.MaxConcurrency == 0 {
		req.Options.MaxConcurrency = 5
	}
	if req.Options.FlushInterval == 0 {
		req.Options.FlushInterval = 30 * time.Second
	}
	if req.Options.RefreshPolicy == "" {
		req.Options.RefreshPolicy = "false" // Don't refresh immediately for performance
	}

	// Process bulk indexing with optimized pipeline
	result, err := ss.processBulkIndexing(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Bulk indexing failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bulk indexing completed",
		"result":  result,
	})
}

// EnhancedIndexWork handles single work indexing with real-time updates
func (ss *SearchService) EnhancedIndexWork(c *gin.Context) {
	workID := c.Param("id")
	if workID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Work ID is required",
		})
		return
	}

	var workDoc WorkIndexDocument
	if err := c.ShouldBindJSON(&workDoc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid work document",
			"details": err.Error(),
		})
		return
	}

	// Enhance document with derived fields
	ss.enhanceWorkDocument(&workDoc)

	// Index the work
	err := ss.indexSingleWork(workDoc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to index work",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Work indexed successfully",
		"work_id":    workDoc.WorkID,
		"indexed_at": workDoc.IndexedAt,
	})
}

// DeleteWorkFromIndex removes a work from the search index
func (ss *SearchService) DeleteWorkFromIndex(c *gin.Context) {
	workID := c.Param("id")
	if workID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Work ID is required",
		})
		return
	}

	err := ss.deleteWorkFromIndex(workID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete work from index",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Work deleted from index",
		"work_id": workID,
	})
}

// GetIndexingStatus returns the current status of the indexing queue
func (ss *SearchService) GetIndexingStatus(c *gin.Context) {
	status := ss.getIndexingQueueStatus()
	c.JSON(http.StatusOK, status)
}

// EnhancedRebuildIndex triggers a complete index rebuild (admin operation)
func (ss *SearchService) EnhancedRebuildIndex(c *gin.Context) {
	// This should be protected with proper authentication in production
	go ss.rebuildSearchIndex()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Enhanced index rebuild started",
		"status":  "processing",
	})
}

// =============================================================================
// INDEXING PIPELINE CORE IMPLEMENTATION
// =============================================================================

// processBulkIndexing handles the bulk indexing pipeline with optimized performance
func (ss *SearchService) processBulkIndexing(req BulkIndexingRequest) (map[string]interface{}, error) {
	startTime := time.Now()

	// Prepare bulk request body
	var bulkBody strings.Builder
	successCount := 0
	errorCount := 0

	for _, job := range req.Jobs {
		// Add action and metadata
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": "works",
				"_id":    job.WorkID,
			},
		}

		if job.Type == IndexingJobDelete {
			action = map[string]interface{}{
				"delete": map[string]interface{}{
					"_index": "works",
					"_id":    job.WorkID,
				},
			}
		}

		actionJSON, _ := json.Marshal(action)
		bulkBody.Write(actionJSON)
		bulkBody.WriteString("\n")

		// Add document body for non-delete operations
		if job.Type != IndexingJobDelete && job.WorkData != nil {
			// Enhance work document if it's a WorkIndexDocument
			if workDoc, ok := job.WorkData.(WorkIndexDocument); ok {
				ss.enhanceWorkDocument(&workDoc)
				job.WorkData = workDoc
			}

			docJSON, _ := json.Marshal(job.WorkData)
			bulkBody.Write(docJSON)
			bulkBody.WriteString("\n")
		}
	}

	// Execute bulk request
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	res, err := ss.es.Bulk(
		bytes.NewReader([]byte(bulkBody.String())),
		ss.es.Bulk.WithContext(ctx),
		ss.es.Bulk.WithIndex("works"),
		ss.es.Bulk.WithRefresh(req.Options.RefreshPolicy),
	)
	if err != nil {
		return nil, fmt.Errorf("bulk request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("bulk request returned error: %s", res.String())
	}

	// Parse bulk response
	var bulkResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&bulkResponse); err != nil {
		return nil, fmt.Errorf("failed to parse bulk response: %w", err)
	}

	// Count successes and errors
	if items, ok := bulkResponse["items"].([]interface{}); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				for _, operation := range itemMap {
					if opMap, ok := operation.(map[string]interface{}); ok {
						if status, ok := opMap["status"].(float64); ok {
							if status >= 200 && status < 300 {
								successCount++
							} else {
								errorCount++
							}
						}
					}
				}
			}
		}
	}

	duration := time.Since(startTime)

	return map[string]interface{}{
		"total_jobs":    len(req.Jobs),
		"successful":    successCount,
		"failed":        errorCount,
		"duration_ms":   duration.Milliseconds(),
		"throughput":    float64(len(req.Jobs)) / duration.Seconds(),
		"bulk_response": bulkResponse,
	}, nil
}

// indexSingleWork indexes a single work document
func (ss *SearchService) indexSingleWork(doc WorkIndexDocument) error {
	doc.IndexedAt = time.Now()
	doc.Version = doc.Version + 1

	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := ss.es.Index(
		"works",
		bytes.NewReader(docJSON),
		ss.es.Index.WithContext(ctx),
		ss.es.Index.WithDocumentID(doc.WorkID),
		ss.es.Index.WithRefresh("true"), // Immediate refresh for single documents
	)
	if err != nil {
		return fmt.Errorf("index request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("index request returned error: %s", res.String())
	}

	return nil
}

// deleteWorkFromIndex removes a work from the search index
func (ss *SearchService) deleteWorkFromIndex(workID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := ss.es.Delete(
		"works",
		workID,
		ss.es.Delete.WithContext(ctx),
		ss.es.Delete.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("delete request returned error: %s", res.String())
	}

	return nil
}

// enhanceWorkDocument adds derived fields and optimizations to work documents
func (ss *SearchService) enhanceWorkDocument(doc *WorkIndexDocument) {
	// Calculate popularity score based on engagement metrics
	doc.PopularityScore = ss.calculatePopularityScore(doc)

	// Calculate recent activity score
	doc.RecentActivityScore = ss.calculateRecentActivityScore(doc)

	// Create combined searchable text
	doc.SearchableText = ss.createSearchableText(doc)

	// Calculate tag frequency for boosting
	doc.TagFrequency = ss.calculateTagFrequency(doc)

	// Calculate tag quality metrics (NEW for Task 3)
	ss.calculateTagQualityMetrics(doc)

	// Set derived metrics
	doc.ContentLength = len(doc.Content)
	doc.UniqueTagCount = len(doc.AdditionalTags) + len(doc.Characters) + len(doc.Relationships)

	// Set default values if missing
	if doc.IndexedAt.IsZero() {
		doc.IndexedAt = time.Now()
	}

	// Increment version for tracking
	if doc.Version == 0 {
		doc.Version = 1
	}
}

// calculatePopularityScore computes a popularity score based on engagement metrics
func (ss *SearchService) calculatePopularityScore(doc *WorkIndexDocument) float64 {
	// Weighted scoring based on different engagement types
	score := float64(doc.Kudos)*2.0 + float64(doc.Comments)*3.0 + float64(doc.Bookmarks)*4.0 + float64(doc.Hits)*0.1

	// Apply word count normalization (longer works tend to get more engagement)
	if doc.WordCount > 0 {
		score = score / (1.0 + float64(doc.WordCount)/10000.0)
	}

	// Apply time decay (newer works get slight boost)
	daysSincePublished := time.Since(doc.PublishedDate).Hours() / 24
	if daysSincePublished > 0 {
		timeDecay := 1.0 / (1.0 + daysSincePublished/365.0) // Decay over a year
		score *= (0.8 + 0.2*timeDecay)                      // Minimum 80% of original score
	}

	return score
}

// calculateRecentActivityScore gives higher scores to recently active works
func (ss *SearchService) calculateRecentActivityScore(doc *WorkIndexDocument) float64 {
	now := time.Now()
	daysSinceUpdate := now.Sub(doc.UpdatedDate).Hours() / 24

	if daysSinceUpdate <= 1 {
		return 1.0 // Maximum score for works updated within 24 hours
	} else if daysSinceUpdate <= 7 {
		return 0.8 // High score for works updated within a week
	} else if daysSinceUpdate <= 30 {
		return 0.6 // Medium score for works updated within a month
	} else if daysSinceUpdate <= 365 {
		return 0.3 // Lower score for works updated within a year
	} else {
		return 0.1 // Minimum score for older works
	}
}

// createSearchableText combines all searchable fields into a single field for efficient searching
func (ss *SearchService) createSearchableText(doc *WorkIndexDocument) string {
	var parts []string

	parts = append(parts, doc.Title)
	parts = append(parts, doc.Summary)
	parts = append(parts, strings.Join(doc.AuthorNames, " "))
	parts = append(parts, strings.Join(doc.Fandoms, " "))
	parts = append(parts, strings.Join(doc.Characters, " "))
	parts = append(parts, strings.Join(doc.Relationships, " "))
	parts = append(parts, strings.Join(doc.AdditionalTags, " "))

	// Include first 1000 characters of content if available
	if len(doc.Content) > 0 {
		contentSnippet := doc.Content
		if len(contentSnippet) > 1000 {
			contentSnippet = contentSnippet[:1000]
		}
		parts = append(parts, contentSnippet)
	}

	return strings.Join(parts, " ")
}

// calculateTagFrequency calculates frequency scores for tags to boost popular combinations
func (ss *SearchService) calculateTagFrequency(doc *WorkIndexDocument) map[string]int {
	frequency := make(map[string]int)

	// Count fandom occurrences
	for _, fandom := range doc.Fandoms {
		frequency[strings.ToLower(fandom)]++
	}

	// Count character occurrences
	for _, character := range doc.Characters {
		frequency[strings.ToLower(character)]++
	}

	// Count relationship occurrences
	for _, relationship := range doc.Relationships {
		frequency[strings.ToLower(relationship)]++
	}

	// Count additional tag occurrences
	for _, tag := range doc.AdditionalTags {
		frequency[strings.ToLower(tag)]++
	}

	return frequency
}

// getIndexingQueueStatus returns the current status of the indexing system
func (ss *SearchService) getIndexingQueueStatus() IndexingQueueStatus {
	// In a real implementation, this would query actual queue metrics
	// For now, return mock data that demonstrates the structure
	return IndexingQueueStatus{
		PendingJobs:    0,
		ProcessingJobs: 0,
		CompletedJobs:  0,
		FailedJobs:     0,
		QueueBacklog:   0,
		WorkerStatuses: []IndexingWorkerStatus{},
		PerformanceMetrics: IndexingPerformanceMetrics{
			IndexingRate:     0.0,
			AverageLatency:   0,
			P95Latency:       0,
			ThroughputMBPS:   0.0,
			ErrorRate:        0.0,
			QueueUtilization: 0.0,
		},
	}
}

// rebuildSearchIndex performs a complete rebuild of the search index
func (ss *SearchService) rebuildSearchIndex() {
	// In a real implementation, this would:
	// 1. Create a new index with updated mappings
	// 2. Reindex all works from the database
	// 3. Switch to the new index atomically
	// 4. Delete the old index

	// For now, this is a placeholder that demonstrates the concept
	fmt.Println("Index rebuild started...")
	time.Sleep(5 * time.Second) // Simulate rebuild time
	fmt.Println("Index rebuild completed")
}

// =============================================================================
// ENHANCED FILTERING & FACETS IMPLEMENTATION - TASK 3
// =============================================================================

// SmartFilteredSearch handles advanced filtering with tag quality analysis
func (ss *SearchService) SmartFilteredSearch(c *gin.Context) {
	var req AdvancedFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid advanced filter request",
			"details": err.Error(),
		})
		return
	}

	// Set defaults for smart filtering
	if req.FacetConfig.MaxFacetValues == 0 {
		req.FacetConfig.MaxFacetValues = 20
	}
	if req.FacetConfig.MinDocCount == 0 {
		req.FacetConfig.MinDocCount = 1
	}
	if req.FacetConfig.RareTagThreshold == 0 {
		req.FacetConfig.RareTagThreshold = 5
	}

	// Build enhanced query with smart tag analysis
	query := ss.buildSmartFilterQuery(req)

	// Execute search with enhanced facets
	response, err := ss.executeSmartFilteredSearch(query, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Smart filtered search failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// AnalyzeTagQuality analyzes the tag quality of works matching the search criteria
func (ss *SearchService) AnalyzeTagQuality(c *gin.Context) {
	var req AdvancedFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid tag quality analysis request",
			"details": err.Error(),
		})
		return
	}

	// Analyze tag quality for the search results
	analysis, err := ss.performTagQualityAnalysis(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Tag quality analysis failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// GetSmartFacets returns intelligent facets with tag enhancement suggestions
func (ss *SearchService) GetSmartFacets(c *gin.Context) {
	var req AdvancedFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid smart facets request",
			"details": err.Error(),
		})
		return
	}

	// Generate smart facets
	facets, err := ss.generateSmartFacets(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Smart facets generation failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, facets)
}

// SuggestTagEnhancements provides tag suggestions for poorly tagged works
func (ss *SearchService) SuggestTagEnhancements(c *gin.Context) {
	workID := c.Param("id")
	if workID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Work ID is required",
		})
		return
	}

	// Get work and analyze its tagging
	enhancement, err := ss.analyzeWorkTagging(workID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Tag enhancement analysis failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, enhancement)
}

// =============================================================================
// SMART FILTERING CORE IMPLEMENTATION
// =============================================================================

// buildSmartFilterQuery constructs an Elasticsearch query with enhanced tag filtering
func (ss *SearchService) buildSmartFilterQuery(req AdvancedFilterRequest) map[string]interface{} {
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must":   []map[string]interface{}{},
			"filter": []map[string]interface{}{},
			"should": []map[string]interface{}{},
		},
	}

	boolQuery := query["bool"].(map[string]interface{})

	// Add base text query if provided
	if req.Query != "" {
		boolQuery["must"] = append(boolQuery["must"].([]map[string]interface{}), map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  req.Query,
				"fields": []string{"title^3", "summary^2", "searchable_text", "author_names^2"},
				"type":   "best_fields",
			},
		})
	}

	// Add smart tag filters with auto-expansion
	if req.AutoExpandTags {
		ss.addExpandedTagFilters(boolQuery, req)
	} else {
		ss.addStandardTagFilters(boolQuery, req)
	}

	// Add tag quality filters
	if req.MinTaggingQuality > 0 {
		boolQuery["filter"] = append(boolQuery["filter"].([]map[string]interface{}), map[string]interface{}{
			"range": map[string]interface{}{
				"tagging_quality_score": map[string]interface{}{
					"gte": req.MinTaggingQuality,
				},
			},
		})
	}

	// Exclude poorly tagged works if requested
	if req.ExcludePoorlyTagged {
		boolQuery["must_not"] = []map[string]interface{}{
			{
				"range": map[string]interface{}{
					"tag_inconsistency_count": map[string]interface{}{
						"gt": 0,
					},
				},
			},
		}
	}

	// Add facet aggregations
	query["aggs"] = ss.buildSmartFacetAggregations(req.FacetConfig)

	return map[string]interface{}{
		"query": query,
		"size":  req.FacetConfig.MaxFacetValues,
	}
}

// addExpandedTagFilters adds tag filters with smart expansion (e.g., "Reader" implied by "Character/Reader")
func (ss *SearchService) addExpandedTagFilters(boolQuery map[string]interface{}, req AdvancedFilterRequest) {
	// Expand character tags based on relationships
	expandedCharacters := ss.expandCharacterTags(req.Characters, req.Relationships)

	// Expand relationship tags based on characters
	expandedRelationships := ss.expandRelationshipTags(req.Relationships, req.Characters)

	// Add expanded character filters
	if len(expandedCharacters) > 0 {
		boolQuery["filter"] = append(boolQuery["filter"].([]map[string]interface{}), map[string]interface{}{
			"terms": map[string]interface{}{
				"characters.keyword": expandedCharacters,
			},
		})
	}

	// Add expanded relationship filters
	if len(expandedRelationships) > 0 {
		boolQuery["filter"] = append(boolQuery["filter"].([]map[string]interface{}), map[string]interface{}{
			"terms": map[string]interface{}{
				"relationships.keyword": expandedRelationships,
			},
		})
	}

	// Add standard filters for other tags
	ss.addStandardTagFilters(boolQuery, req)
}

// expandCharacterTags extracts characters from relationship tags and adds them
func (ss *SearchService) expandCharacterTags(characters []string, relationships []string) []string {
	expanded := make([]string, len(characters))
	copy(expanded, characters)

	// Extract characters from relationship tags
	for _, rel := range relationships {
		// Handle common relationship patterns
		if strings.Contains(rel, "/") {
			parts := strings.Split(rel, "/")
			for _, part := range parts {
				character := strings.TrimSpace(part)
				if character != "" && !ss.containsString(expanded, character) {
					expanded = append(expanded, character)
				}
			}
		}

		// Handle "Character & Character" patterns
		if strings.Contains(rel, " & ") {
			parts := strings.Split(rel, " & ")
			for _, part := range parts {
				character := strings.TrimSpace(part)
				if character != "" && !ss.containsString(expanded, character) {
					expanded = append(expanded, character)
				}
			}
		}

		// Handle "Character x Character" patterns
		if strings.Contains(rel, " x ") {
			parts := strings.Split(rel, " x ")
			for _, part := range parts {
				character := strings.TrimSpace(part)
				if character != "" && !ss.containsString(expanded, character) {
					expanded = append(expanded, character)
				}
			}
		}
	}

	return expanded
}

// expandRelationshipTags suggests relationships based on character combinations
func (ss *SearchService) expandRelationshipTags(relationships []string, characters []string) []string {
	expanded := make([]string, len(relationships))
	copy(expanded, relationships)

	// Generate potential relationship combinations from characters
	if len(characters) >= 2 {
		for i := 0; i < len(characters); i++ {
			for j := i + 1; j < len(characters); j++ {
				// Create potential romantic relationship
				potentialRel := characters[i] + "/" + characters[j]
				if !ss.containsString(expanded, potentialRel) {
					// This would be checked against known relationships in a real implementation
					// For now, we'll add it as a potential suggestion
					expanded = append(expanded, potentialRel)
				}

				// Create potential platonic relationship
				potentialPlatonic := characters[i] + " & " + characters[j]
				if !ss.containsString(expanded, potentialPlatonic) {
					expanded = append(expanded, potentialPlatonic)
				}
			}
		}
	}

	return expanded
}

// addStandardTagFilters adds regular tag filters without expansion
func (ss *SearchService) addStandardTagFilters(boolQuery map[string]interface{}, req AdvancedFilterRequest) {
	if len(req.Fandoms) > 0 {
		boolQuery["filter"] = append(boolQuery["filter"].([]map[string]interface{}), map[string]interface{}{
			"terms": map[string]interface{}{
				"fandoms.keyword": req.Fandoms,
			},
		})
	}

	if len(req.Characters) > 0 {
		boolQuery["filter"] = append(boolQuery["filter"].([]map[string]interface{}), map[string]interface{}{
			"terms": map[string]interface{}{
				"characters.keyword": req.Characters,
			},
		})
	}

	if len(req.Relationships) > 0 {
		boolQuery["filter"] = append(boolQuery["filter"].([]map[string]interface{}), map[string]interface{}{
			"terms": map[string]interface{}{
				"relationships.keyword": req.Relationships,
			},
		})
	}

	if len(req.FreeformTags) > 0 {
		boolQuery["filter"] = append(boolQuery["filter"].([]map[string]interface{}), map[string]interface{}{
			"terms": map[string]interface{}{
				"additional_tags.keyword": req.FreeformTags,
			},
		})
	}
}

// buildSmartFacetAggregations creates intelligent facet aggregations
func (ss *SearchService) buildSmartFacetAggregations(config FacetConfiguration) map[string]interface{} {
	aggs := map[string]interface{}{}

	// Standard facets with smart filtering
	facetFields := map[string]string{
		"fandoms":       "fandoms.keyword",
		"characters":    "characters.keyword",
		"relationships": "relationships.keyword",
		"tags":          "additional_tags.keyword",
		"ratings":       "rating.keyword",
		"warnings":      "warnings.keyword",
		"categories":    "categories.keyword",
	}

	for facetName, fieldName := range facetFields {
		facetAgg := map[string]interface{}{
			"terms": map[string]interface{}{
				"field": fieldName,
				"size":  config.MaxFacetValues,
			},
		}

		// Add minimum document count filter
		if config.MinDocCount > 1 {
			facetAgg["terms"].(map[string]interface{})["min_doc_count"] = config.MinDocCount
		}

		// Exclude rare tags if configured
		if config.ExcludeRareTags && config.RareTagThreshold > 0 {
			facetAgg["terms"].(map[string]interface{})["min_doc_count"] = config.RareTagThreshold
		}

		aggs[facetName] = facetAgg
	}

	// Add tag quality distribution facet
	aggs["tag_quality_distribution"] = map[string]interface{}{
		"range": map[string]interface{}{
			"field": "tagging_quality_score",
			"ranges": []map[string]interface{}{
				{"from": 0, "to": 25, "key": "poor"},
				{"from": 25, "to": 50, "key": "fair"},
				{"from": 50, "to": 75, "key": "good"},
				{"from": 75, "to": 100, "key": "excellent"},
			},
		},
	}

	// Add word count distribution
	aggs["word_count_distribution"] = map[string]interface{}{
		"range": map[string]interface{}{
			"field": "word_count",
			"ranges": []map[string]interface{}{
				{"from": 0, "to": 1000, "key": "flash_fiction"},
				{"from": 1000, "to": 5000, "key": "short"},
				{"from": 5000, "to": 20000, "key": "medium"},
				{"from": 20000, "to": 50000, "key": "long"},
				{"from": 50000, "key": "epic"},
			},
		},
	}

	// Add completion status facet
	aggs["completion_status"] = map[string]interface{}{
		"terms": map[string]interface{}{
			"field": "completion_status.keyword",
			"size":  10,
		},
	}

	return aggs
}

// executeSmartFilteredSearch executes the search with enhanced processing
func (ss *SearchService) executeSmartFilteredSearch(query map[string]interface{}, req AdvancedFilterRequest) (*SmartFacetResponse, error) {
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := ss.es.Search(
		ss.es.Search.WithContext(ctx),
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
	var searchResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	// Process facets and add intelligence
	return ss.processSmartFacetResponse(searchResponse, req)
}

// processSmartFacetResponse converts Elasticsearch response to smart facet response
func (ss *SearchService) processSmartFacetResponse(response map[string]interface{}, req AdvancedFilterRequest) (*SmartFacetResponse, error) {
	smartResponse := &SmartFacetResponse{
		StandardFacets:    make(map[string][]FacetValue),
		SmartFacets:       make(map[string][]SmartFacet),
		TagSuggestions:    []TagSuggestion{},
		FilterSuggestions: []FilterSuggestion{},
		QualityMetrics:    make(map[string]interface{}),
	}

	// Process aggregations if they exist
	if aggs, ok := response["aggregations"].(map[string]interface{}); ok {
		// Process standard facets
		ss.processStandardFacets(aggs, smartResponse)

		// Generate smart facets with intelligence
		ss.generateIntelligentFacets(aggs, smartResponse, req)

		// Add tag suggestions if enabled
		if req.SuggestMissingTags {
			smartResponse.TagSuggestions = ss.generateTagSuggestions(aggs, req)
		}

		// Add filter suggestions
		smartResponse.FilterSuggestions = ss.generateFilterSuggestions(aggs, req)
	}

	return smartResponse, nil
}

// Additional helper methods for smart filtering
func (ss *SearchService) containsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

// Placeholder implementations for remaining methods
func (ss *SearchService) performTagQualityAnalysis(req AdvancedFilterRequest) (*TagQualityAnalysis, error) {
	// Mock implementation - would analyze actual tag quality in production
	return &TagQualityAnalysis{
		Completeness:    0.75,
		Consistency:     0.80,
		Specificity:     0.70,
		CrossTagging:    0.65,
		OverallScore:    72.5,
		IssueCount:      3,
		MissingTagCount: 5,
		QualityCategory: "good",
	}, nil
}

func (ss *SearchService) generateSmartFacets(req AdvancedFilterRequest) (*SmartFacetResponse, error) {
	// This would generate intelligent facets based on the request
	return &SmartFacetResponse{
		StandardFacets:    make(map[string][]FacetValue),
		SmartFacets:       make(map[string][]SmartFacet),
		TagSuggestions:    []TagSuggestion{},
		FilterSuggestions: []FilterSuggestion{},
		QualityMetrics:    make(map[string]interface{}),
	}, nil
}

func (ss *SearchService) analyzeWorkTagging(workID string) (*SmartTagEnhancement, error) {
	// Mock implementation - would analyze specific work's tagging
	return &SmartTagEnhancement{
		MissingCharacters:    []string{"Reader"},
		MissingRelationships: []string{},
		SuggestedTags: []TagSuggestion{
			{
				Tag:        "Reader",
				Type:       "character",
				Confidence: 0.95,
				Reason:     "Implied by relationship tag 'Agatha Harkness/Reader'",
			},
		},
		TagInconsistencies:  []TagInconsistency{},
		TaggingQualityScore: 75.0,
		SmartFilters:        make(map[string]interface{}),
	}, nil
}

func (ss *SearchService) processStandardFacets(aggs map[string]interface{}, response *SmartFacetResponse) {
	// Process standard Elasticsearch aggregations into facet values
}

func (ss *SearchService) generateIntelligentFacets(aggs map[string]interface{}, response *SmartFacetResponse, req AdvancedFilterRequest) {
	// Generate smart facets with trend analysis and quality metrics
}

func (ss *SearchService) generateTagSuggestions(aggs map[string]interface{}, req AdvancedFilterRequest) []TagSuggestion {
	// Generate intelligent tag suggestions based on search context
	return []TagSuggestion{}
}

func (ss *SearchService) generateFilterSuggestions(aggs map[string]interface{}, req AdvancedFilterRequest) []FilterSuggestion {
	// Generate intelligent filter suggestions to improve search results
	return []FilterSuggestion{}
}

// =============================================================================
// TAG QUALITY ANALYSIS IMPLEMENTATION - Core of Task 3
// =============================================================================

// calculateTagQualityMetrics analyzes and scores the tag quality of a work document
func (ss *SearchService) calculateTagQualityMetrics(doc *WorkIndexDocument) {
	// Extract implied characters from relationships
	doc.ImpliedCharacters = ss.extractImpliedCharacters(doc.Relationships)

	// Extract implied relationships from characters
	doc.ImpliedRelationships = ss.extractImpliedRelationships(doc.Characters)

	// Calculate cross-tagging score (how well relationship and character tags align)
	doc.CrossTaggingScore = ss.calculateCrossTaggingScore(doc)

	// Calculate tag completeness score
	doc.TagCompletenessScore = ss.calculateTagCompletenessScore(doc)

	// Count missing tags
	doc.MissingTagCount = ss.countMissingTags(doc)

	// Count tag inconsistencies
	doc.TagInconsistencyCount = ss.countTagInconsistencies(doc)

	// Calculate overall tagging quality score (0-100)
	doc.TaggingQualityScore = ss.calculateOverallTagQuality(doc)
}

// extractImpliedCharacters finds characters that should be tagged based on relationships
func (ss *SearchService) extractImpliedCharacters(relationships []string) []string {
	var implied []string

	for _, rel := range relationships {
		// Parse relationship patterns
		characters := ss.parseCharactersFromRelationship(rel)
		for _, char := range characters {
			if char != "" && !ss.containsString(implied, char) {
				implied = append(implied, char)
			}
		}
	}

	return implied
}

// parseCharactersFromRelationship extracts character names from relationship strings
func (ss *SearchService) parseCharactersFromRelationship(relationship string) []string {
	var characters []string

	// Handle "/" for romantic relationships
	if strings.Contains(relationship, "/") {
		parts := strings.Split(relationship, "/")
		for _, part := range parts {
			char := strings.TrimSpace(part)
			if char != "" {
				characters = append(characters, char)
			}
		}
	}

	// Handle "&" for platonic relationships
	if strings.Contains(relationship, " & ") {
		parts := strings.Split(relationship, " & ")
		for _, part := range parts {
			char := strings.TrimSpace(part)
			if char != "" {
				characters = append(characters, char)
			}
		}
	}

	// Handle "x" notation
	if strings.Contains(relationship, " x ") {
		parts := strings.Split(relationship, " x ")
		for _, part := range parts {
			char := strings.TrimSpace(part)
			if char != "" {
				characters = append(characters, char)
			}
		}
	}

	return characters
}

// extractImpliedRelationships suggests relationships based on character combinations
func (ss *SearchService) extractImpliedRelationships(characters []string) []string {
	var implied []string

	// Generate potential relationships from character pairs
	for i := 0; i < len(characters); i++ {
		for j := i + 1; j < len(characters); j++ {
			// Add romantic possibility
			romantic := characters[i] + "/" + characters[j]
			implied = append(implied, romantic)

			// Add platonic possibility
			platonic := characters[i] + " & " + characters[j]
			implied = append(implied, platonic)
		}
	}

	return implied
}

// calculateCrossTaggingScore measures how well character and relationship tags align
func (ss *SearchService) calculateCrossTaggingScore(doc *WorkIndexDocument) float64 {
	if len(doc.Relationships) == 0 {
		return 100.0 // No relationships to cross-check
	}

	totalImplied := len(doc.ImpliedCharacters)
	if totalImplied == 0 {
		return 100.0 // No implied characters
	}

	matchingTags := 0
	for _, implied := range doc.ImpliedCharacters {
		if ss.containsString(doc.Characters, implied) {
			matchingTags++
		}
	}

	return (float64(matchingTags) / float64(totalImplied)) * 100.0
}

// calculateTagCompletenessScore estimates how complete the tagging appears
func (ss *SearchService) calculateTagCompletenessScore(doc *WorkIndexDocument) float64 {
	score := 100.0

	// Basic tags present check
	if len(doc.Fandoms) == 0 {
		score -= 20 // Major penalty for missing fandom
	}

	if doc.Rating == "" {
		score -= 10 // Penalty for missing rating
	}

	if len(doc.Categories) == 0 {
		score -= 5 // Minor penalty for missing category
	}

	// Relationship/character consistency check
	if len(doc.Relationships) > 0 && len(doc.Characters) == 0 {
		score -= 15 // Relationships without characters
	}

	// Additional tags based on word count
	expectedTagCount := ss.estimateExpectedTagCount(doc.WordCount)
	actualTagCount := len(doc.AdditionalTags)

	if actualTagCount < expectedTagCount/2 {
		score -= 10 // Under-tagged work
	}

	// Ensure score doesn't go below 0
	if score < 0 {
		score = 0
	}

	return score
}

// estimateExpectedTagCount estimates how many additional tags a work should have
func (ss *SearchService) estimateExpectedTagCount(wordCount int) int {
	if wordCount < 1000 {
		return 3 // Short works need fewer tags
	} else if wordCount < 5000 {
		return 5
	} else if wordCount < 20000 {
		return 8
	} else {
		return 12 // Longer works typically need more descriptive tags
	}
}

// countMissingTags estimates the number of missing tags
func (ss *SearchService) countMissingTags(doc *WorkIndexDocument) int {
	missing := 0

	// Count missing characters implied by relationships
	for _, implied := range doc.ImpliedCharacters {
		if !ss.containsString(doc.Characters, implied) {
			missing++
		}
	}

	// Basic required tags
	if len(doc.Fandoms) == 0 {
		missing++
	}
	if doc.Rating == "" {
		missing++
	}
	if len(doc.Categories) == 0 {
		missing++
	}

	return missing
}

// countTagInconsistencies detects various tagging problems
func (ss *SearchService) countTagInconsistencies(doc *WorkIndexDocument) int {
	inconsistencies := 0

	// Relationships without corresponding characters
	for _, implied := range doc.ImpliedCharacters {
		if !ss.containsString(doc.Characters, implied) {
			inconsistencies++
		}
	}

	// Check for contradictory tags (would require domain knowledge)
	// For now, just basic structure checks

	// Multiple contradictory ratings (would need parsing)
	// Multiple contradictory completion statuses, etc.

	return inconsistencies
}

// calculateOverallTagQuality computes the final quality score
func (ss *SearchService) calculateOverallTagQuality(doc *WorkIndexDocument) float64 {
	// Weighted combination of different quality metrics
	completenessWeight := 0.4
	crossTaggingWeight := 0.3
	consistencyWeight := 0.3

	// Consistency score based on inconsistency count
	consistencyScore := 100.0
	if doc.TagInconsistencyCount > 0 {
		consistencyScore = 100.0 - float64(doc.TagInconsistencyCount*10) // 10 point penalty per inconsistency
		if consistencyScore < 0 {
			consistencyScore = 0
		}
	}

	// Calculate weighted average
	overallScore := (doc.TagCompletenessScore * completenessWeight) +
		(doc.CrossTaggingScore * crossTaggingWeight) +
		(consistencyScore * consistencyWeight)

	// Apply penalty for missing tags
	if doc.MissingTagCount > 0 {
		overallScore -= float64(doc.MissingTagCount * 5) // 5 point penalty per missing tag
	}

	// Ensure score is within bounds
	if overallScore < 0 {
		overallScore = 0
	}
	if overallScore > 100 {
		overallScore = 100
	}

	return overallScore
}

// =============================================================================
// TASK 5: SEARCH ANALYTICS DASHBOARD
// Comprehensive analytics and insights for search performance and user behavior
// =============================================================================

// SearchAnalyticsDashboard represents the main dashboard data structure
type SearchAnalyticsDashboard struct {
	Overview           AnalyticsOverview        `json:"overview"`
	PerformanceMetrics PerformanceAnalytics     `json:"performance_metrics"`
	SearchTrends       SearchTrendAnalytics     `json:"search_trends"`
	TagQualityInsights TagQualityAnalytics      `json:"tag_quality_insights"`
	UserBehavior       UserBehaviorAnalytics    `json:"user_behavior"`
	PopularContent     PopularContentAnalytics  `json:"popular_content"`
	SystemHealth       SystemHealthAnalytics    `json:"system_health"`
	RealtimeMetrics    RealtimeAnalytics        `json:"realtime_metrics"`
	Recommendations    AnalyticsRecommendations `json:"recommendations"`
	GeneratedAt        time.Time                `json:"generated_at"`
}

// AnalyticsOverview provides high-level summary metrics
type AnalyticsOverview struct {
	TotalSearches24h       int64   `json:"total_searches_24h"`
	UniqueUsers24h         int64   `json:"unique_users_24h"`
	AverageResponseTime    float64 `json:"average_response_time_ms"`
	SearchSuccessRate      float64 `json:"search_success_rate"`
	MostPopularSearchTerm  string  `json:"most_popular_search_term"`
	TagQualityTrend        string  `json:"tag_quality_trend"` // "improving", "stable", "declining"
	IndexingHealth         string  `json:"indexing_health"`   // "excellent", "good", "needs_attention"
	TotalWorksIndexed      int64   `json:"total_works_indexed"`
	AverageTagQualityScore float64 `json:"average_tag_quality_score"`
}

// PerformanceAnalytics tracks search performance metrics
type PerformanceAnalytics struct {
	ResponseTimeHistogram []TimeHistogramBucket `json:"response_time_histogram"`
	ThroughputMetrics     ThroughputAnalytics   `json:"throughput_metrics"`
	ErrorRateAnalysis     ErrorAnalytics        `json:"error_rate_analysis"`
	IndexingPerformance   IndexingAnalytics     `json:"indexing_performance"`
	CacheEfficiency       CacheAnalytics        `json:"cache_efficiency"`
}

// SearchTrendAnalytics provides insights into search patterns
type SearchTrendAnalytics struct {
	PopularSearchTerms []SearchTermTrend    `json:"popular_search_terms"`
	EmergingTrends     []EmergingTrend      `json:"emerging_trends"`
	SeasonalPatterns   []SeasonalPattern    `json:"seasonal_patterns"`
	FandomTrends       []FandomTrend        `json:"fandom_trends"`
	ZeroResultQueries  []ZeroResultQuery    `json:"zero_result_queries"`
	SearchVolumeByHour []HourlySearchVolume `json:"search_volume_by_hour"`
}

// TagQualityAnalytics analyzes tagging patterns and quality
type TagQualityAnalytics struct {
	QualityDistribution      []QualityDistribution `json:"quality_distribution"`
	CommonTaggingIssues      []TaggingIssue        `json:"common_tagging_issues"`
	TagCompletionSuggestions []TagSuggestion       `json:"tag_completion_suggestions"`
	CrossTaggingAnalysis     CrossTagAnalysis      `json:"cross_tagging_analysis"`
	TagUsagePatterns         []TagUsagePattern     `json:"tag_usage_patterns"`
	QualityTrendsByFandom    []FandomQualityTrend  `json:"quality_trends_by_fandom"`
}

// UserBehaviorAnalytics tracks user search behavior
type UserBehaviorAnalytics struct {
	SearchSessionAnalysis    SessionAnalytics     `json:"search_session_analysis"`
	UserJourneyPatterns      []UserJourneyPattern `json:"user_journey_patterns"`
	SearchRefinementPatterns []RefinementPattern  `json:"search_refinement_patterns"`
	FilterUsageStatistics    FilterUsageStats     `json:"filter_usage_statistics"`
	DeviceAndLocationStats   DeviceLocationStats  `json:"device_location_stats"`
}

// PopularContentAnalytics identifies trending content
type PopularContentAnalytics struct {
	TrendingWorks       []TrendingWork        `json:"trending_works"`
	PopularTags         []PopularTag          `json:"popular_tags"`
	EmergingFandoms     []EmergingFandom      `json:"emerging_fandoms"`
	CharacterPopularity []CharacterPopularity `json:"character_popularity"`
	RelationshipTrends  []RelationshipTrend   `json:"relationship_trends"`
}

// SystemHealthAnalytics monitors system performance
type SystemHealthAnalytics struct {
	ElasticsearchHealth ElasticsearchMetrics `json:"elasticsearch_health"`
	DatabasePerformance DatabaseMetrics      `json:"database_performance"`
	RedisPerformance    RedisMetrics         `json:"redis_performance"`
	ServerResourceUsage ResourceMetrics      `json:"server_resource_usage"`
	AlertsAndWarnings   []SystemAlert        `json:"alerts_warnings"`
}

// RealtimeAnalytics provides live metrics
type RealtimeAnalytics struct {
	CurrentActiveUsers      int64             `json:"current_active_users"`
	SearchesPerMinute       float64           `json:"searches_per_minute"`
	AverageResponseTimeLive float64           `json:"average_response_time_live_ms"`
	TopSearchesRightNow     []LiveSearchTrend `json:"top_searches_right_now"`
	SystemLoadMetrics       LiveSystemMetrics `json:"system_load_metrics"`
}

// AnalyticsRecommendations provides actionable insights
type AnalyticsRecommendations struct {
	PerformanceOptimizations   []Recommendation `json:"performance_optimizations"`
	TagQualityImprovements     []Recommendation `json:"tag_quality_improvements"`
	UserExperienceEnhancements []Recommendation `json:"user_experience_enhancements"`
	SystemMaintenance          []Recommendation `json:"system_maintenance"`
	ContentCurationTips        []Recommendation `json:"content_curation_tips"`
}

// Supporting data structures for analytics
type TimeHistogramBucket struct {
	Range      string  `json:"range"` // "0-50ms", "50-100ms", etc.
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

type ThroughputAnalytics struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	PeakThroughput    float64 `json:"peak_throughput"`
	ThroughputTrend   string  `json:"throughput_trend"`
}

type ErrorAnalytics struct {
	ErrorRate    float64          `json:"error_rate"`
	CommonErrors []ErrorFrequency `json:"common_errors"`
	ErrorTrend   string           `json:"error_trend"`
}

type IndexingAnalytics struct {
	IndexingRate   float64 `json:"indexing_rate"`
	QueueBacklog   int64   `json:"queue_backlog"`
	IndexingErrors int64   `json:"indexing_errors"`
}

type CacheAnalytics struct {
	HitRate   float64 `json:"hit_rate"`
	MissRate  float64 `json:"miss_rate"`
	CacheSize int64   `json:"cache_size"`
}

type SearchTermTrend struct {
	Term             string  `json:"term"`
	SearchCount      int64   `json:"search_count"`
	TrendDirection   string  `json:"trend_direction"`
	PercentageChange float64 `json:"percentage_change"`
}

type EmergingTrend struct {
	Term              string  `json:"term"`
	GrowthRate        float64 `json:"growth_rate"`
	FirstSeenDays     int     `json:"first_seen_days"`
	CurrentPopularity int64   `json:"current_popularity"`
}

type FandomTrend struct {
	Fandom         string  `json:"fandom"`
	SearchVolume   int64   `json:"search_volume"`
	QualityScore   float64 `json:"quality_score"`
	TrendDirection string  `json:"trend_direction"`
}

type ZeroResultQuery struct {
	Query                 string   `json:"query"`
	Frequency             int64    `json:"frequency"`
	SuggestedAlternatives []string `json:"suggested_alternatives"`
}

type HourlySearchVolume struct {
	Hour                int     `json:"hour"`
	SearchCount         int64   `json:"search_count"`
	AverageResponseTime float64 `json:"average_response_time"`
}

type QualityDistribution struct {
	QualityRange string  `json:"quality_range"`
	WorkCount    int64   `json:"work_count"`
	Percentage   float64 `json:"percentage"`
}

type TaggingIssue struct {
	IssueType    string   `json:"issue_type"`
	Frequency    int64    `json:"frequency"`
	ExampleWorks []string `json:"example_works"`
	SuggestedFix string   `json:"suggested_fix"`
}

type Recommendation struct {
	Type                 string `json:"type"`
	Priority             string `json:"priority"`
	Title                string `json:"title"`
	Description          string `json:"description"`
	EstimatedImpact      string `json:"estimated_impact"`
	ImplementationEffort string `json:"implementation_effort"`
}

// Additional supporting types (abbreviated for brevity)
type CrossTagAnalysis struct {
	ConsistencyScore float64  `json:"consistency_score"`
	CommonMismatches []string `json:"common_mismatches"`
}

type TagUsagePattern struct {
	Pattern       string  `json:"pattern"`
	Frequency     int64   `json:"frequency"`
	QualityImpact float64 `json:"quality_impact"`
}

type FandomQualityTrend struct {
	Fandom         string  `json:"fandom"`
	AverageQuality float64 `json:"average_quality"`
	QualityTrend   string  `json:"quality_trend"`
}

type SessionAnalytics struct {
	AverageSessionLength float64 `json:"average_session_length"`
	SearchesPerSession   float64 `json:"searches_per_session"`
	BounceRate           float64 `json:"bounce_rate"`
}

type UserJourneyPattern struct {
	Pattern     string  `json:"pattern"`
	Frequency   int64   `json:"frequency"`
	SuccessRate float64 `json:"success_rate"`
}

type RefinementPattern struct {
	InitialSearch string `json:"initial_search"`
	RefinedSearch string `json:"refined_search"`
	Frequency     int64  `json:"frequency"`
}

type FilterUsageStats struct {
	MostUsedFilters    []string `json:"most_used_filters"`
	FilterCombinations []string `json:"filter_combinations"`
}

type DeviceLocationStats struct {
	TopDevices   []string `json:"top_devices"`
	TopLocations []string `json:"top_locations"`
}

type TrendingWork struct {
	WorkID       string  `json:"work_id"`
	Title        string  `json:"title"`
	SearchCount  int64   `json:"search_count"`
	QualityScore float64 `json:"quality_score"`
}

type PopularTag struct {
	Tag            string `json:"tag"`
	UsageCount     int64  `json:"usage_count"`
	TrendDirection string `json:"trend_direction"`
}

type EmergingFandom struct {
	Fandom     string  `json:"fandom"`
	GrowthRate float64 `json:"growth_rate"`
	WorkCount  int64   `json:"work_count"`
}

type CharacterPopularity struct {
	Character         string  `json:"character"`
	SearchCount       int64   `json:"search_count"`
	CrossTaggingScore float64 `json:"cross_tagging_score"`
}

type RelationshipTrend struct {
	Relationship    string  `json:"relationship"`
	PopularityScore float64 `json:"popularity_score"`
	TagQualityScore float64 `json:"tag_quality_score"`
}

type ElasticsearchMetrics struct {
	ClusterHealth string  `json:"cluster_health"`
	IndexCount    int64   `json:"index_count"`
	DocumentCount int64   `json:"document_count"`
	QueryLatency  float64 `json:"query_latency"`
}

type DatabaseMetrics struct {
	ConnectionCount int64   `json:"connection_count"`
	QueryLatency    float64 `json:"query_latency"`
	LockWaitTime    float64 `json:"lock_wait_time"`
}

type RedisMetrics struct {
	UsedMemory int64   `json:"used_memory"`
	HitRate    float64 `json:"hit_rate"`
	KeyCount   int64   `json:"key_count"`
}

type ResourceMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
}

type SystemAlert struct {
	Type      string    `json:"type"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type LiveSearchTrend struct {
	Term           string `json:"term"`
	CurrentCount   int64  `json:"current_count"`
	TrendDirection string `json:"trend_direction"`
}

type LiveSystemMetrics struct {
	ActiveConnections int64   `json:"active_connections"`
	QueueDepth        int64   `json:"queue_depth"`
	ResponseTime      float64 `json:"response_time"`
}

type ErrorFrequency struct {
	ErrorType string    `json:"error_type"`
	Count     int64     `json:"count"`
	LastSeen  time.Time `json:"last_seen"`
}

type SeasonalPattern struct {
	Period       string   `json:"period"`
	SearchVolume int64    `json:"search_volume"`
	PopularTags  []string `json:"popular_tags"`
}

// =============================================================================
// SEARCH ANALYTICS DASHBOARD HANDLERS
// =============================================================================

// GetAnalyticsDashboard returns the complete analytics dashboard
func (ss *SearchService) GetAnalyticsDashboard(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "24h") // 1h, 24h, 7d, 30d

	dashboard, err := ss.generateAnalyticsDashboard(timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate analytics dashboard",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// GetPerformanceMetrics returns detailed performance analytics
func (ss *SearchService) GetPerformanceMetrics(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "1h")

	metrics, err := ss.generatePerformanceMetrics(timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate performance metrics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetSearchTrends returns search trend analysis
func (ss *SearchService) GetSearchTrends(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "7d")

	trends, err := ss.generateSearchTrends(timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate search trends",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, trends)
}

// GetTagQualityInsights returns tag quality analytics
func (ss *SearchService) GetTagQualityInsights(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "30d")

	insights, err := ss.generateTagQualityInsights(timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate tag quality insights",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, insights)
}

// GetRealtimeMetrics returns live system metrics
func (ss *SearchService) GetRealtimeMetrics(c *gin.Context) {
	metrics, err := ss.generateRealtimeMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate realtime metrics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetAnalyticsRecommendations returns actionable insights
func (ss *SearchService) GetAnalyticsRecommendations(c *gin.Context) {
	recommendations, err := ss.generateAnalyticsRecommendations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate recommendations",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, recommendations)
}

// =============================================================================
// ANALYTICS GENERATION IMPLEMENTATION
// =============================================================================

// generateAnalyticsDashboard creates the complete dashboard data
func (ss *SearchService) generateAnalyticsDashboard(timeRange string) (*SearchAnalyticsDashboard, error) {
	// This would integrate with actual analytics data sources
	// For now, returning a comprehensive mock dashboard

	dashboard := &SearchAnalyticsDashboard{
		Overview: AnalyticsOverview{
			TotalSearches24h:       25847,
			UniqueUsers24h:         8923,
			AverageResponseTime:    45.2,
			SearchSuccessRate:      94.8,
			MostPopularSearchTerm:  "Harry Potter",
			TagQualityTrend:        "improving",
			IndexingHealth:         "excellent",
			TotalWorksIndexed:      1250000,
			AverageTagQualityScore: 82.3,
		},
		PerformanceMetrics: PerformanceAnalytics{
			ResponseTimeHistogram: []TimeHistogramBucket{
				{Range: "0-25ms", Count: 15234, Percentage: 58.9},
				{Range: "25-50ms", Count: 7823, Percentage: 30.2},
				{Range: "50-100ms", Count: 2156, Percentage: 8.3},
				{Range: "100ms+", Count: 634, Percentage: 2.6},
			},
			ThroughputMetrics: ThroughputAnalytics{
				RequestsPerSecond: 125.4,
				PeakThroughput:    478.2,
				ThroughputTrend:   "stable",
			},
		},
		SearchTrends: SearchTrendAnalytics{
			PopularSearchTerms: []SearchTermTrend{
				{Term: "Harry Potter", SearchCount: 3421, TrendDirection: "up", PercentageChange: 12.5},
				{Term: "Marvel", SearchCount: 2876, TrendDirection: "stable", PercentageChange: 1.2},
				{Term: "Agatha Harkness", SearchCount: 1234, TrendDirection: "up", PercentageChange: 45.6},
			},
			EmergingTrends: []EmergingTrend{
				{Term: "Wednesday Addams", GrowthRate: 89.2, FirstSeenDays: 7, CurrentPopularity: 892},
				{Term: "House of the Dragon", GrowthRate: 67.8, FirstSeenDays: 14, CurrentPopularity: 654},
			},
		},
		TagQualityInsights: TagQualityAnalytics{
			QualityDistribution: []QualityDistribution{
				{QualityRange: "90-100 (Excellent)", WorkCount: 425000, Percentage: 34.0},
				{QualityRange: "75-89 (Good)", WorkCount: 500000, Percentage: 40.0},
				{QualityRange: "50-74 (Fair)", WorkCount: 250000, Percentage: 20.0},
				{QualityRange: "0-49 (Poor)", WorkCount: 75000, Percentage: 6.0},
			},
			CommonTaggingIssues: []TaggingIssue{
				{
					IssueType:    "Missing Character Tags",
					Frequency:    15234,
					ExampleWorks: []string{"work123", "work456"},
					SuggestedFix: "Add characters implied by relationships",
				},
				{
					IssueType:    "Inconsistent Rating",
					Frequency:    8923,
					ExampleWorks: []string{"work789"},
					SuggestedFix: "Review content for appropriate rating",
				},
			},
		},
		Recommendations: AnalyticsRecommendations{
			TagQualityImprovements: []Recommendation{
				{
					Type:                 "tag_quality",
					Priority:             "high",
					Title:                "Implement Smart Tag Suggestions",
					Description:          "15,234 works missing implied character tags. Enable smart tag completion.",
					EstimatedImpact:      "20% improvement in search relevance",
					ImplementationEffort: "medium",
				},
			},
			PerformanceOptimizations: []Recommendation{
				{
					Type:                 "performance",
					Priority:             "medium",
					Title:                "Optimize Popular Query Cache",
					Description:          "Top 10 queries account for 45% of traffic. Implement specialized caching.",
					EstimatedImpact:      "15% reduction in response time",
					ImplementationEffort: "low",
				},
			},
		},
		GeneratedAt: time.Now(),
	}

	return dashboard, nil
}

// generatePerformanceMetrics creates detailed performance analytics
func (ss *SearchService) generatePerformanceMetrics(timeRange string) (*PerformanceAnalytics, error) {
	// Mock implementation - would integrate with actual monitoring systems
	return &PerformanceAnalytics{
		ResponseTimeHistogram: []TimeHistogramBucket{
			{Range: "0-25ms", Count: 15234, Percentage: 58.9},
			{Range: "25-50ms", Count: 7823, Percentage: 30.2},
			{Range: "50-100ms", Count: 2156, Percentage: 8.3},
			{Range: "100ms+", Count: 634, Percentage: 2.6},
		},
		ThroughputMetrics: ThroughputAnalytics{
			RequestsPerSecond: 125.4,
			PeakThroughput:    478.2,
			ThroughputTrend:   "stable",
		},
		ErrorRateAnalysis: ErrorAnalytics{
			ErrorRate: 2.1,
			CommonErrors: []ErrorFrequency{
				{ErrorType: "timeout", Count: 234, LastSeen: time.Now().Add(-time.Hour)},
				{ErrorType: "index_unavailable", Count: 45, LastSeen: time.Now().Add(-time.Minute * 30)},
			},
			ErrorTrend: "decreasing",
		},
	}, nil
}

// generateSearchTrends creates search trend analysis
func (ss *SearchService) generateSearchTrends(timeRange string) (*SearchTrendAnalytics, error) {
	// Mock implementation - would analyze actual search data
	return &SearchTrendAnalytics{
		PopularSearchTerms: []SearchTermTrend{
			{Term: "Harry Potter", SearchCount: 3421, TrendDirection: "up", PercentageChange: 12.5},
			{Term: "Marvel", SearchCount: 2876, TrendDirection: "stable", PercentageChange: 1.2},
			{Term: "Agatha Harkness/Reader", SearchCount: 1234, TrendDirection: "up", PercentageChange: 45.6},
		},
		EmergingTrends: []EmergingTrend{
			{Term: "Wednesday Addams", GrowthRate: 89.2, FirstSeenDays: 7, CurrentPopularity: 892},
			{Term: "House of the Dragon", GrowthRate: 67.8, FirstSeenDays: 14, CurrentPopularity: 654},
		},
		ZeroResultQueries: []ZeroResultQuery{
			{
				Query:                 "Hagatha Harkness",
				Frequency:             123,
				SuggestedAlternatives: []string{"Agatha Harkness"},
			},
		},
	}, nil
}

// generateTagQualityInsights creates tag quality analytics
func (ss *SearchService) generateTagQualityInsights(timeRange string) (*TagQualityAnalytics, error) {
	// Mock implementation showcasing tag quality insights
	return &TagQualityAnalytics{
		QualityDistribution: []QualityDistribution{
			{QualityRange: "90-100 (Excellent)", WorkCount: 425000, Percentage: 34.0},
			{QualityRange: "75-89 (Good)", WorkCount: 500000, Percentage: 40.0},
			{QualityRange: "50-74 (Fair)", WorkCount: 250000, Percentage: 20.0},
			{QualityRange: "0-49 (Poor)", WorkCount: 75000, Percentage: 6.0},
		},
		CommonTaggingIssues: []TaggingIssue{
			{
				IssueType:    "Missing Character Tags (Reader)",
				Frequency:    15234,
				ExampleWorks: []string{"work123", "work456"},
				SuggestedFix: "Add 'Reader' character tag when 'Character/Reader' relationship exists",
			},
			{
				IssueType:    "Incomplete Cross-Tagging",
				Frequency:    8923,
				ExampleWorks: []string{"work789"},
				SuggestedFix: "Ensure all relationship characters are listed in character tags",
			},
		},
		CrossTaggingAnalysis: CrossTagAnalysis{
			ConsistencyScore: 73.5,
			CommonMismatches: []string{
				"Agatha Harkness/Reader relationship without Reader character",
				"Harry Potter/Ginny Weasley relationship without Ginny character",
			},
		},
	}, nil
}

// generateRealtimeMetrics creates live system metrics
func (ss *SearchService) generateRealtimeMetrics() (*RealtimeAnalytics, error) {
	// Mock implementation - would connect to live monitoring
	return &RealtimeAnalytics{
		CurrentActiveUsers:      1247,
		SearchesPerMinute:       89.3,
		AverageResponseTimeLive: 42.8,
		TopSearchesRightNow: []LiveSearchTrend{
			{Term: "Wednesday Addams", CurrentCount: 23, TrendDirection: "up"},
			{Term: "Harry Potter", CurrentCount: 18, TrendDirection: "stable"},
			{Term: "Marvel", CurrentCount: 15, TrendDirection: "down"},
		},
		SystemLoadMetrics: LiveSystemMetrics{
			ActiveConnections: 2341,
			QueueDepth:        45,
			ResponseTime:      42.8,
		},
	}, nil
}

// generateAnalyticsRecommendations creates actionable insights
func (ss *SearchService) generateAnalyticsRecommendations() (*AnalyticsRecommendations, error) {
	// Mock implementation showcasing smart recommendations
	return &AnalyticsRecommendations{
		TagQualityImprovements: []Recommendation{
			{
				Type:                 "tag_quality",
				Priority:             "high",
				Title:                "Fix Missing Reader Character Tags",
				Description:          "15,234 works have 'Character/Reader' relationships but missing 'Reader' character tag. This affects search discoverability.",
				EstimatedImpact:      "20% improvement in Reader-focused search results",
				ImplementationEffort: "low",
			},
			{
				Type:                 "tag_quality",
				Priority:             "medium",
				Title:                "Implement Cross-Tag Validation",
				Description:          "8,923 works have relationship/character inconsistencies. Enable validation during work submission.",
				EstimatedImpact:      "15% improvement in overall tag quality",
				ImplementationEffort: "medium",
			},
		},
		PerformanceOptimizations: []Recommendation{
			{
				Type:                 "performance",
				Priority:             "medium",
				Title:                "Cache Popular Character Searches",
				Description:          "Harry Potter, Marvel, and Agatha Harkness searches account for 40% of traffic.",
				EstimatedImpact:      "25% reduction in response time for popular searches",
				ImplementationEffort: "low",
			},
		},
		UserExperienceEnhancements: []Recommendation{
			{
				Type:                 "ux",
				Priority:             "high",
				Title:                "Smart Search Suggestions",
				Description:          "123 users searched for 'Hagatha Harkness' with zero results. Implement auto-correct.",
				EstimatedImpact:      "Reduce zero-result searches by 30%",
				ImplementationEffort: "medium",
			},
		},
	}, nil
}
