package auth

import (
	"github.com/gin-gonic/gin"

	"nuclear-ao3/internal/platform"
)

// setupRouter builds a router with only the auth routes, for tests.
func setupRouter(as *AuthService) *gin.Engine {
	r := gin.New()
	Mount(r, as, platform.NewDeps(as.db, as.ValidateUserToken))
	return r
}
