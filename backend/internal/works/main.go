package works

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"nuclear-ao3/internal/platform"
	"nuclear-ao3/shared/notifications"
)

// Mount registers the works, chapters, comments, bookmarks, series,
// collections and user-dashboard routes.
func Mount(r *gin.Engine, workService *WorkService, d *platform.Deps) {

	// API endpoints
	api := r.Group("/api/v1")
	{
		// Public endpoints with optional auth
		// Legacy routes (plural - supports both UUID and integer with redirect)
		legacy := api.Group("/works")
		legacy.Use(d.OptionalUser)
		{
			legacy.GET("", workService.SearchWorks)                                       // GET /api/v1/works?q=search&fandom=HP (browse/search)
			legacy.GET("/:work_id", workService.GetWorkWithAuthors)                       // GET /api/v1/works/123 or /works/uuid (redirects legacy IDs)
			legacy.GET("/:work_id/chapters", workService.GetChapters)                     // GET /api/v1/works/123/chapters
			legacy.GET("/:work_id/chapters/:chapter_id", workService.GetChapter)          // GET /api/v1/works/123/chapters/1
			legacy.GET("/:work_id/comments", workService.GetComments)                     // GET /api/v1/works/123/comments
			legacy.GET("/:work_id/kudos", workService.GetKudos)                           // GET /api/v1/works/123/kudos
			legacy.GET("/:work_id/stats", workService.GetWorkStats)                       // GET /api/v1/works/123/stats
			legacy.GET("/:work_id/download.:format", workService.DownloadWork)            // GET /api/v1/works/123/download.epub (epub, html, txt)
			legacy.POST("/:work_id/comments", d.ActiveAccount, workService.CreateComment) // POST /api/v1/works/123/comments (guest + auth comments)
		}

		// Modern routes (singular - UUID-based permanent URLs)
		modern := api.Group("/work")
		modern.Use(d.OptionalUser)
		{
			modern.GET("/:work_id", workService.GetWorkWithAuthors)                       // GET /api/v1/work/{uuid} (permanent)
			modern.GET("/:work_id/chapters", workService.GetChapters)                     // GET /api/v1/work/{uuid}/chapters
			modern.GET("/:work_id/chapters/:chapter_id", workService.GetChapter)          // GET /api/v1/work/{uuid}/chapters/{uuid}
			modern.GET("/:work_id/comments", workService.GetComments)                     // GET /api/v1/work/{uuid}/comments
			modern.GET("/:work_id/kudos", workService.GetKudos)                           // GET /api/v1/work/{uuid}/kudos
			modern.GET("/:work_id/stats", workService.GetWorkStats)                       // GET /api/v1/work/{uuid}/stats
			modern.GET("/:work_id/download.:format", workService.DownloadWork)            // GET /api/v1/work/{uuid}/download.epub
			modern.POST("/:work_id/comments", d.ActiveAccount, workService.CreateComment) // POST /api/v1/work/{uuid}/comments (guest + auth comments)
		}

		// Series endpoints
		series := api.Group("/series")
		series.Use(d.OptionalUser)
		{
			series.GET("", workService.SearchSeries)                    // GET /api/v1/series?q=search
			series.GET("/:series_id", workService.GetSeries)            // GET /api/v1/series/123
			series.GET("/:series_id/works", workService.GetSeriesWorks) // GET /api/v1/series/123/works
		}

		// Collections endpoints
		collections := api.Group("/collections")
		collections.Use(d.OptionalUser)
		{
			collections.GET("", workService.SearchCollections)                       // GET /api/v1/collections
			collections.GET("/:collection_id", workService.GetCollection)            // GET /api/v1/collections/123
			collections.GET("/:collection_id/works", workService.GetCollectionWorks) // GET /api/v1/collections/123/works
		}

		// Tag search endpoints (enhanced partial matching)

		// Pseud pages (AO3: /users/:user_id/pseuds/:id/works, /series and
		// /bookmarks). Lists are keyed by pseud, never by account. AO3's
		// account-keyed pages (/users/:login/works) are not offered: they join
		// all of a person's pseuds in one list and take an account name.
		pseuds := api.Group("/pseuds")
		pseuds.Use(d.OptionalUser)
		{
			pseuds.GET("/:pseud_id", workService.GetPseud)                    // GET /api/v1/pseuds/{uuid}
			pseuds.GET("/:pseud_id/gifts", workService.GetPseudGifts)         // GET /api/v1/pseuds/{uuid}/gifts
			pseuds.GET("/:pseud_id/works", workService.GetPseudWorks)         // GET /api/v1/pseuds/{uuid}/works
			pseuds.GET("/:pseud_id/series", workService.GetPseudSeries)       // GET /api/v1/pseuds/{uuid}/series
			pseuds.GET("/:pseud_id/bookmarks", workService.GetPseudBookmarks) // GET /api/v1/pseuds/{uuid}/bookmarks
		}

		// Authenticated endpoints
		protected := api.Group("")
		protected.Use(d.RequireUser)
		{
			// Work management
			protected.POST("/works", d.ActiveAccount, workService.CreateWork)                                   // POST /api/v1/works
			protected.PUT("/works/:work_id", d.ActiveAccount, workService.UpdateWork)                           // PUT /api/v1/works/123
			protected.DELETE("/works/:work_id", d.NotSuspended, workService.DeleteWork)                         // DELETE /api/v1/works/123
			protected.POST("/works/:work_id/chapters", d.ActiveAccount, workService.CreateChapter)              // POST /api/v1/works/123/chapters
			protected.PUT("/works/:work_id/chapters/:chapter_id", d.ActiveAccount, workService.UpdateChapter)   // PUT /api/v1/works/123/chapters/1
			protected.DELETE("/works/:work_id/chapters/:chapter_id", d.NotSuspended, workService.DeleteChapter) // DELETE /api/v1/works/123/chapters/1

			// Engagement
			protected.POST("/works/:work_id/kudos", d.ActiveAccount, workService.GiveKudos) // POST /api/v1/works/123/kudos
			protected.DELETE("/works/:work_id/kudos", workService.RemoveKudos)              // DELETE /api/v1/works/123/kudos
			// Note: Comment creation moved to legacy/modern groups to support guest comments
			protected.PUT("/comments/:comment_id", d.ActiveAccount, workService.UpdateComment)    // PUT /api/v1/comments/123
			protected.DELETE("/comments/:comment_id", d.ActiveAccount, workService.DeleteComment) // DELETE /api/v1/comments/123

			// Bookmarks
			protected.POST("/works/:work_id/bookmark", d.ActiveAccount, workService.CreateBookmark) // POST /api/v1/works/123/bookmark
			protected.GET("/works/:work_id/bookmark-status", workService.GetBookmarkStatus)         // GET /api/v1/works/123/bookmark-status
			protected.PUT("/bookmarks/:bookmark_id", d.ActiveAccount, workService.UpdateBookmark)   // PUT /api/v1/bookmarks/123
			protected.DELETE("/bookmarks/:bookmark_id", workService.DeleteBookmark)                 // DELETE /api/v1/bookmarks/123
			protected.GET("/bookmarks", workService.GetMyBookmarks)                                 // GET /api/v1/bookmarks

			// Series management
			protected.POST("/series", d.ActiveAccount, workService.CreateSeries)                                     // POST /api/v1/series
			protected.PUT("/series/:series_id", d.ActiveAccount, workService.UpdateSeries)                           // PUT /api/v1/series/123
			protected.DELETE("/series/:series_id", workService.DeleteSeries)                                         // DELETE /api/v1/series/123
			protected.POST("/series/:series_id/works/:work_id", d.ActiveAccount, workService.AddWorkToSeries)        // POST /api/v1/series/123/works/456
			protected.DELETE("/series/:series_id/works/:work_id", d.ActiveAccount, workService.RemoveWorkFromSeries) // DELETE /api/v1/series/123/works/456

			// Collections management
			protected.POST("/collections", d.ActiveAccount, workService.CreateCollection)                        // POST /api/v1/collections
			protected.PUT("/collections/:collection_id", d.ActiveAccount, workService.UpdateCollection)          // PUT /api/v1/collections/123
			protected.DELETE("/collections/:collection_id", d.ActiveAccount, workService.DeleteCollection)       // DELETE /api/v1/collections/123
			protected.POST("/collections/:collection_id/works/:work_id", workService.AddWorkToCollection)        // POST /api/v1/collections/123/works/456
			protected.DELETE("/collections/:collection_id/works/:work_id", workService.RemoveWorkFromCollection) // DELETE /api/v1/collections/123/works/456

			// Comment moderation
			protected.PUT("/comments/:comment_id/moderate", d.ActiveAccount, workService.ModerateComment) // PUT /api/v1/comments/123/moderate
			protected.PUT("/comments/:comment_id/freeze", d.ActiveAccount, workService.FreezeComment)     // PUT /api/v1/comments/123/freeze
			protected.PUT("/comments/:comment_id/unfreeze", d.ActiveAccount, workService.UnfreezeComment) // PUT /api/v1/comments/123/unfreeze

			// Blocks and mutes are by pseud. A block is between two accounts
			// and works both ways; a mute hides one pseud for the muter.
			protected.POST("/blocks", workService.CreateBlock)             // POST /api/v1/blocks {pseud_id}
			protected.GET("/blocks", workService.ListBlocks)               // GET /api/v1/blocks
			protected.DELETE("/blocks/:pseud_id", workService.DeleteBlock) // DELETE /api/v1/blocks/{pseud_id}
			protected.POST("/mutes", workService.CreateMute)               // POST /api/v1/mutes {pseud_id}
			protected.GET("/mutes", workService.ListMutes)                 // GET /api/v1/mutes
			protected.DELETE("/mutes/:pseud_id", workService.DeleteMute)   // DELETE /api/v1/mutes/{pseud_id}

			// Reports
			protected.POST("/comments/:comment_id/report", workService.ReportComment) // POST /api/v1/comments/123/report
			protected.POST("/works/:work_id/report", workService.ReportWork)          // POST /api/v1/works/123/report

			// Core AO3 Features: Pseuds, Gifting, Orphaning, Co-authors
			protected.POST("/pseuds", d.ActiveAccount, workService.CreatePseud)                    // POST /api/v1/pseuds
			protected.GET("/my/pseuds", workService.GetUserPseuds)                                 // GET /api/v1/my/pseuds
			protected.PUT("/pseuds/:pseud_id", d.ActiveAccount, workService.UpdatePseud)           // PUT /api/v1/pseuds/123
			protected.DELETE("/pseuds/:pseud_id", d.ActiveAccount, workService.DeletePseud)        // DELETE /api/v1/pseuds/123
			protected.POST("/works/:work_id/gift", d.ActiveAccount, workService.GiftWork)          // POST /api/v1/works/123/gift
			protected.GET("/works/:work_id/gifts", workService.GetWorkGifts)                       // GET /api/v1/works/123/gifts
			protected.POST("/works/:work_id/orphan", d.NotSuspended, workService.OrphanWork)       // POST /api/v1/works/123/orphan
			protected.GET("/works/:work_id/authors", workService.GetWorkAuthors)                   // GET /api/v1/works/123/authors
			protected.POST("/works/:work_id/co-authors", d.ActiveAccount, workService.AddCoAuthor) // POST /api/v1/works/123/co-authors

			// User dashboard
			protected.GET("/my/works", workService.GetMyWorks)             // GET /api/v1/my/works
			protected.GET("/my/series", workService.GetMySeries)           // GET /api/v1/my/series
			protected.GET("/my/collections", workService.GetMyCollections) // GET /api/v1/my/collections
			protected.GET("/my/comments", workService.GetMyComments)       // GET /api/v1/my/comments
			protected.GET("/my/stats", workService.GetMyStats)             // GET /api/v1/my/stats

			// Reading history and Marked for Later. Under /my/, so never open to apps.
			protected.GET("/my/history", workService.GetMyHistory)                      // GET /api/v1/my/history?show=to-read
			protected.DELETE("/my/history", workService.ClearMyHistory)                 // DELETE /api/v1/my/history
			protected.GET("/my/history/:work_id", workService.GetMyHistoryEntry)        // GET /api/v1/my/history/{uuid}
			protected.DELETE("/my/history/:work_id", workService.DeleteMyHistoryEntry)  // DELETE /api/v1/my/history/{uuid}
			protected.PUT("/my/history/:work_id/marked", workService.SetMarkedForLater) // PUT /api/v1/my/history/{uuid}/marked
			protected.GET("/my/history-settings", workService.GetHistorySettings)       // GET /api/v1/my/history-settings
			protected.PUT("/my/history-settings", workService.SetHistorySettings)       // PUT /api/v1/my/history-settings

			// Subscriptions
			protected.POST("/subscriptions", workService.CreateSubscription)           // POST /api/v1/subscriptions
			protected.GET("/subscriptions", workService.GetUserSubscriptions)          // GET /api/v1/subscriptions
			protected.PUT("/subscriptions/:id", workService.UpdateSubscription)        // PUT /api/v1/subscriptions/123
			protected.DELETE("/subscriptions/:id", workService.DeleteSubscription)     // DELETE /api/v1/subscriptions/123
			protected.GET("/subscription-status", workService.CheckSubscriptionStatus) // GET /api/v1/subscription-status?type=work&target_id=123
		}

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(d.RequireUser)
		admin.Use(d.RequireRole("admin"))
		{
			admin.GET("/works", workService.AdminListWorks)                                 // GET /api/v1/admin/works
			admin.PUT("/works/:work_id/status", workService.AdminUpdateWorkStatus)          // PUT /api/v1/admin/works/123/status
			admin.DELETE("/works/:work_id", workService.AdminDeleteWork)                    // DELETE /api/v1/admin/works/123
			admin.GET("/comments", workService.AdminListComments)                           // GET /api/v1/admin/comments
			admin.PUT("/comments/:comment_id/status", workService.AdminUpdateCommentStatus) // PUT /api/v1/admin/comments/123/status
			admin.DELETE("/comments/:comment_id", workService.AdminDeleteComment)           // DELETE /api/v1/admin/comments/123
			admin.GET("/reports", workService.AdminGetReports)                              // GET /api/v1/admin/reports
			admin.GET("/statistics", workService.AdminGetStatistics)                        // GET /api/v1/admin/statistics
		}
	}
}

// WorkService holds all dependencies for work management
type WorkService struct {
	db                  *sql.DB
	hits                hitCounter
	readings            readingBuffer
	downloads           downloadLimiter
	notificationService *notifications.NotificationService
}

// New builds the works service on top of the shared database.
// It refuses to start if the database schema doesn't match what the code expects.
func New(db *sql.DB) (*WorkService, error) {
	if err := NewSchemaValidator(db).ValidateAllSchemas(); err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}
	return &WorkService{
		db:                  db,
		notificationService: nil, // set by SetNotifier
	}, nil
}

// SetNotifier connects the works package to notifications so subscribers
// hear about new chapters and comments.
func (ws *WorkService) SetNotifier(n *notifications.NotificationService) {
	ws.notificationService = n
}
