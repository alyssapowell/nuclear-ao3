package tags

import (
	"database/sql"

	"nuclear-ao3/internal/platform"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// Mount registers the tags routes.
func Mount(r *gin.Engine, tagService *TagService, d *platform.Deps) {

	// API endpoints
	api := r.Group("/api/v1")
	{
		// Public endpoints (no auth required)
		tags := api.Group("/tags")
		tags.Use(d.OptionalUser)
		{
			tags.GET("", tagService.SearchTags)                     // GET /api/v1/tags?q=search&type=fandom
			tags.GET("/search", tagService.SearchTags)              // GET /api/v1/tags/search?q=search&type=fandom (frontend compatibility)
			tags.GET("/:tag_id", tagService.GetTag)                 // GET /api/v1/tags/123
			tags.GET("/:tag_id/related", tagService.GetRelatedTags) // GET /api/v1/tags/123/related
			tags.GET("/:tag_id/works", tagService.GetTagWorks)      // GET /api/v1/tags/123/works
			tags.GET("/autocomplete", tagService.AutocompleteTags)  // GET /api/v1/tags/autocomplete?q=harry
		}

		// Fandoms
		fandoms := api.Group("/fandoms")
		{
			fandoms.GET("", tagService.SearchFandoms)                                   // GET /api/v1/fandoms?q=search
			fandoms.GET("/:fandom_id", tagService.GetFandom)                            // GET /api/v1/fandoms/123
			fandoms.GET("/:fandom_id/tags", tagService.GetFandomTags)                   // GET /api/v1/fandoms/123/tags
			fandoms.GET("/:fandom_id/characters", tagService.GetFandomCharacters)       // GET /api/v1/fandoms/123/characters
			fandoms.GET("/:fandom_id/relationships", tagService.GetFandomRelationships) // GET /api/v1/fandoms/123/relationships
		}

		// Characters
		characters := api.Group("/characters")
		{
			characters.GET("", tagService.SearchCharacters)                                      // GET /api/v1/characters?q=search&fandom=123
			characters.GET("/:character_id", tagService.GetCharacter)                            // GET /api/v1/characters/123
			characters.GET("/:character_id/relationships", tagService.GetCharacterRelationships) // GET /api/v1/characters/123/relationships
		}

		// Relationships
		relationships := api.Group("/relationships")
		{
			relationships.GET("", tagService.SearchRelationships)              // GET /api/v1/relationships?q=search&fandom=123
			relationships.GET("/:relationship_id", tagService.GetRelationship) // GET /api/v1/relationships/123
		}

		// Tag hierarchies and navigation
		hierarchy := api.Group("/hierarchy")
		{
			hierarchy.GET("/fandoms", tagService.GetFandomHierarchy)             // GET /api/v1/hierarchy/fandoms
			hierarchy.GET("/characters", tagService.GetCharacterHierarchy)       // GET /api/v1/hierarchy/characters
			hierarchy.GET("/relationships", tagService.GetRelationshipHierarchy) // GET /api/v1/hierarchy/relationships
		}

		// Tag statistics
		stats := api.Group("/stats")
		{
			stats.GET("/popular", tagService.GetPopularTags)   // GET /api/v1/stats/popular?type=fandom&limit=50
			stats.GET("/trending", tagService.GetTrendingTags) // GET /api/v1/stats/trending?period=week
			stats.GET("/usage", tagService.GetTagUsageStats)   // GET /api/v1/stats/usage
		}

		// Authenticated endpoints
		protected := api.Group("")
		protected.Use(d.RequireUser)
		{
			// Tag management (user submissions)
			protected.POST("/tags", tagService.CreateTag)                     // POST /api/v1/tags
			protected.PUT("/tags/:tag_id", tagService.UpdateTag)              // PUT /api/v1/tags/123
			protected.POST("/tags/:tag_id/synonym", tagService.CreateSynonym) // POST /api/v1/tags/123/synonym
			protected.POST("/tags/merge", tagService.RequestTagMerge)         // POST /api/v1/tags/merge

			// User tag relationships
			protected.POST("/user/tags/follow", tagService.FollowTag)             // POST /api/v1/user/tags/follow
			protected.DELETE("/user/tags/follow/:tag_id", tagService.UnfollowTag) // DELETE /api/v1/user/tags/follow/123
			protected.GET("/user/tags/followed", tagService.GetFollowedTags)      // GET /api/v1/user/tags/followed

			// Tag reports
			protected.POST("/tags/:tag_id/report", tagService.ReportTag) // POST /api/v1/tags/123/report
		}

		// Wrangler endpoints (tag wranglers have special permissions)
		wrangler := api.Group("/wrangling")
		wrangler.Use(d.RequireUser)
		wrangler.Use(d.RequireRole("tag_wrangler", "admin"))
		{
			wrangler.GET("/queue", tagService.GetWranglingQueue)                           // GET /api/v1/wrangling/queue
			wrangler.GET("/tags/:tag_id", tagService.GetTagForWrangling)                   // GET /api/v1/wrangling/tags/123
			wrangler.POST("/tags/:tag_id/wrangle", tagService.WrangleTag)                  // POST /api/v1/wrangling/tags/123/wrangle
			wrangler.POST("/tags/:tag_id/canonical", tagService.MakeCanonical)             // POST /api/v1/wrangling/tags/123/canonical
			wrangler.POST("/tags/:tag_id/synonym", tagService.CreateCanonicalSynonym)      // POST /api/v1/wrangling/tags/123/synonym
			wrangler.POST("/tags/:tag_id/parent", tagService.AddParentTag)                 // POST /api/v1/wrangling/tags/123/parent
			wrangler.DELETE("/tags/:tag_id/parent/:parent_id", tagService.RemoveParentTag) // DELETE /api/v1/wrangling/tags/123/parent/456
			wrangler.PUT("/merge/:merge_id", tagService.ProcessTagMerge)                   // PUT /api/v1/wrangling/merge/123
			wrangler.GET("/reports", tagService.GetTagReports)                             // GET /api/v1/wrangling/reports
			wrangler.PUT("/reports/:report_id", tagService.ProcessTagReport)               // PUT /api/v1/wrangling/reports/123
		}

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(d.RequireUser)
		admin.Use(d.RequireRole("admin"))
		{
			admin.GET("/tags", tagService.AdminListTags)                        // GET /api/v1/admin/tags
			admin.DELETE("/tags/:tag_id", tagService.AdminDeleteTag)            // DELETE /api/v1/admin/tags/123
			admin.POST("/tags/:tag_id/ban", tagService.AdminBanTag)             // POST /api/v1/admin/tags/123/ban
			admin.DELETE("/tags/:tag_id/ban", tagService.AdminUnbanTag)         // DELETE /api/v1/admin/tags/123/ban
			admin.GET("/wranglers", tagService.AdminListWranglers)              // GET /api/v1/admin/wranglers
			admin.POST("/wranglers", tagService.AdminAddWrangler)               // POST /api/v1/admin/wranglers
			admin.DELETE("/wranglers/:user_id", tagService.AdminRemoveWrangler) // DELETE /api/v1/admin/wranglers/123
			admin.GET("/tag-statistics", tagService.AdminGetTagStatistics)      // GET /api/v1/admin/tag-statistics
		}
	}
}

// TagService holds all dependencies for tag management
type TagService struct {
	db *sql.DB
}

// New builds the tag service on top of the shared database.
func New(db *sql.DB) *TagService {
	return &TagService{db: db}
}
