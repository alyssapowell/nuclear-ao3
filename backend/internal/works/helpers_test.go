package works

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"nuclear-ao3/internal/platform"
)

// setupRouter builds a router with only the works routes, for tests.
// Tests log in by sending "Authorization: Bearer <user uuid>".
func setupRouter(ws *WorkService) *gin.Engine {
	r := gin.New()
	Mount(r, ws, platform.NewDeps(ws.db, uuid.Parse))
	return r
}

// NewWorkService connects to the test database and Redis, for tests.
func NewWorkService() *WorkService {
	db, err := platform.OpenDB()
	if err != nil {
		panic(err)
	}
	ws, err := New(db, redis.NewClient(&redis.Options{Addr: platform.Env("REDIS_URL", "localhost:6379"), DB: 1}))
	if err != nil {
		panic(err)
	}
	return ws
}

func (ws *WorkService) Close() {
	ws.db.Close()
	ws.redis.Close()
}
