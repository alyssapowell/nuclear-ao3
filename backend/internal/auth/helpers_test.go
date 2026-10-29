package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"nuclear-ao3/internal/platform"
)

// setupRouter builds a router with only the auth routes, for tests.
func setupRouter(as *AuthService) *gin.Engine {
	r := gin.New()
	Mount(r, as, platform.NewDeps(as.db, as.ValidateUserToken))
	return r
}

// NewAuthService connects to the test database and Redis, for tests.
func NewAuthService() *AuthService {
	db, err := platform.OpenDB()
	if err != nil {
		panic(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: platform.Env("REDIS_URL", "localhost:6379")})
	as, err := New(db, rdb)
	if err != nil {
		panic(err)
	}
	return as
}

func (as *AuthService) Close() {
	as.db.Close()
	as.redis.Close()
}
