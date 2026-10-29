package search

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"nuclear-ao3/internal/platform"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// Mount registers the search routes.
func Mount(r *gin.Engine, searchService *SearchService, d *platform.Deps) {

	// API endpoints
	api := r.Group("/api/v1")
	{
		// Search endpoints
		search := api.Group("/search")
		{
			// General search
			search.GET("/works", searchService.SearchWorks)             // GET /api/v1/search/works?q=harry+potter
			search.GET("/tags", searchService.SearchTags)               // GET /api/v1/search/tags?q=angst
			search.GET("/users", searchService.SearchUsers)             // GET /api/v1/search/users?q=author_name
			search.GET("/collections", searchService.SearchCollections) // GET /api/v1/search/collections?q=prompt_fest
			search.GET("/series", searchService.SearchSeries)           // GET /api/v1/search/series?q=trilogy

			// Advanced/filtered search
			search.POST("/works/advanced", searchService.AdvancedWorkSearch) // POST /api/v1/search/works/advanced
			search.POST("/tags/advanced", searchService.AdvancedTagSearch)   // POST /api/v1/search/tags/advanced

			// Enhanced smart filtering (Task 3)
			search.POST("/works/smart", searchService.SmartFilteredSearch)   // POST /api/v1/search/works/smart
			search.POST("/facets/smart", searchService.GetSmartFacets)       // POST /api/v1/search/facets/smart
			search.POST("/quality/analyze", searchService.AnalyzeTagQuality) // POST /api/v1/search/quality/analyze

			// Autocomplete/suggestions
			search.GET("/suggestions", searchService.GetSuggestions)   // GET /api/v1/search/suggestions?q=har
			search.GET("/popular", searchService.GetPopularSearches)   // GET /api/v1/search/popular
			search.GET("/trending", searchService.GetTrendingSearches) // GET /api/v1/search/trending
		}

		// Indexing operations (internal/admin only)
		index := api.Group("/index")
		index.Use(d.RequireUser)
		index.Use(d.RequireRole("admin", "indexer"))
		{
			// Enhanced work indexing operations
			index.POST("/works", searchService.EnhancedIndexWork)           // POST /api/v1/index/works
			index.PUT("/works/:id", searchService.EnhancedIndexWork)        // PUT /api/v1/index/works/123
			index.DELETE("/works/:id", searchService.DeleteWorkFromIndex)   // DELETE /api/v1/index/works/123
			index.POST("/works/bulk", searchService.EnhancedBulkIndexWorks) // POST /api/v1/index/works/bulk

			// Legacy tag indexing (to be enhanced)
			index.POST("/tags", searchService.IndexTag)                 // POST /api/v1/index/tags
			index.PUT("/tags/:tag_id", searchService.UpdateTagIndex)    // PUT /api/v1/index/tags/123
			index.DELETE("/tags/:tag_id", searchService.DeleteTagIndex) // DELETE /api/v1/index/tags/123
			index.POST("/tags/bulk", searchService.BulkIndexTags)       // POST /api/v1/index/tags/bulk

			// Legacy user indexing (to be enhanced)
			index.POST("/users", searchService.IndexUser)                  // POST /api/v1/index/users
			index.PUT("/users/:user_id", searchService.UpdateUserIndex)    // PUT /api/v1/index/users/123
			index.DELETE("/users/:user_id", searchService.DeleteUserIndex) // DELETE /api/v1/index/users/123

			// Enhanced index management
			index.POST("/rebuild", searchService.EnhancedRebuildIndex) // POST /api/v1/index/rebuild
			index.GET("/status", searchService.GetIndexingStatus)      // GET /api/v1/index/status
			index.POST("/optimize", searchService.OptimizeIndex)       // POST /api/v1/index/optimize

			// Tag enhancement suggestions
			index.GET("/works/:id/suggest-tags", searchService.SuggestTagEnhancements) // GET /api/v1/index/works/123/suggest-tags
		}

		// Analytics and insights
		analytics := api.Group("/analytics")
		analytics.Use(d.RequireUser)
		analytics.Use(d.RequireRole("admin"))
		{
			// Legacy analytics endpoints
			analytics.GET("/search-stats", searchService.GetSearchStats)      // GET /api/v1/analytics/search-stats
			analytics.GET("/popular-terms", searchService.GetPopularTerms)    // GET /api/v1/analytics/popular-terms
			analytics.GET("/zero-results", searchService.GetZeroResultTerms)  // GET /api/v1/analytics/zero-results
			analytics.GET("/performance", searchService.GetSearchPerformance) // GET /api/v1/analytics/performance

			// Enhanced analytics dashboard (Task 5)
			analytics.GET("/dashboard", searchService.GetAnalyticsDashboard)             // GET /api/v1/analytics/dashboard
			analytics.GET("/metrics/performance", searchService.GetPerformanceMetrics)   // GET /api/v1/analytics/metrics/performance
			analytics.GET("/trends", searchService.GetSearchTrends)                      // GET /api/v1/analytics/trends
			analytics.GET("/tag-quality", searchService.GetTagQualityInsights)           // GET /api/v1/analytics/tag-quality
			analytics.GET("/realtime", searchService.GetRealtimeMetrics)                 // GET /api/v1/analytics/realtime
			analytics.GET("/recommendations", searchService.GetAnalyticsRecommendations) // GET /api/v1/analytics/recommendations
		}

		// Search history and saved searches (authenticated users)
		protected := api.Group("")
		protected.Use(d.RequireUser)
		{
			protected.GET("/history", searchService.GetSearchHistory)                           // GET /api/v1/history
			protected.DELETE("/history", searchService.ClearSearchHistory)                      // DELETE /api/v1/history
			protected.POST("/saved-searches", searchService.SaveSearch)                         // POST /api/v1/saved-searches
			protected.GET("/saved-searches", searchService.GetSavedSearches)                    // GET /api/v1/saved-searches
			protected.DELETE("/saved-searches/:search_id", searchService.DeleteSavedSearch)     // DELETE /api/v1/saved-searches/123
			protected.POST("/saved-searches/:search_id/alert", searchService.CreateSearchAlert) // POST /api/v1/saved-searches/123/alert
		}

		// Search filters and facets
		filters := api.Group("/filters")
		{
			filters.GET("/fandoms", searchService.GetFandomFilters)             // GET /api/v1/filters/fandoms
			filters.GET("/characters", searchService.GetCharacterFilters)       // GET /api/v1/filters/characters
			filters.GET("/relationships", searchService.GetRelationshipFilters) // GET /api/v1/filters/relationships
			filters.GET("/tags", searchService.GetTagFilters)                   // GET /api/v1/filters/tags
			filters.GET("/stats", searchService.GetStatFilters)                 // GET /api/v1/filters/stats (word count ranges, etc)
		}
	}
}

// SearchService holds all dependencies for search functionality
type SearchService struct {
	db    *sql.DB
	redis *redis.Client
	es    *elasticsearch.Client
}

// New builds the search service. It needs Elasticsearch (ELASTICSEARCH_URL)
// and returns an error if it can't reach it.
func New(db *sql.DB, rdb *redis.Client) (*SearchService, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{platform.Env("ELASTICSEARCH_URL", "http://localhost:9200")},
		Username:  platform.Env("ELASTICSEARCH_USERNAME", ""),
		Password:  platform.Env("ELASTICSEARCH_PASSWORD", ""),
		CloudID:   platform.Env("ELASTICSEARCH_CLOUD_ID", ""),
		APIKey:    platform.Env("ELASTICSEARCH_API_KEY", ""),
	})
	if err != nil {
		return nil, fmt.Errorf("create Elasticsearch client: %w", err)
	}
	if _, err := es.Ping(); err != nil {
		return nil, fmt.Errorf("Elasticsearch not reachable: %w", err)
	}
	return &SearchService{db: db, redis: rdb, es: es}, nil
}

// IndexWorkDoc adds or updates one work in the search index. doc is the work
// as JSON (the WorkIndexDocument shape).
func (ss *SearchService) IndexWorkDoc(workID string, doc []byte) error {
	var workDoc WorkIndexDocument
	if err := json.Unmarshal(doc, &workDoc); err != nil {
		return fmt.Errorf("invalid work document: %w", err)
	}
	workDoc.WorkID = workID
	ss.enhanceWorkDocument(&workDoc)
	return ss.indexSingleWork(workDoc)
}
