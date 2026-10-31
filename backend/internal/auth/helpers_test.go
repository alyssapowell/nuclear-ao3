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

// NewAuthService connects to the test database, for tests.
func NewAuthService() *AuthService {
	db, err := platform.OpenDB()
	if err != nil {
		panic(err)
	}
	as, err := newTestAuthService(db)
	if err != nil {
		panic(err)
	}
	return as
}

func (as *AuthService) Close() {
	as.db.Close()
}
