package works

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/platform"
)

// setupRouter builds a router with only the works routes, for tests.
// Tests log in by sending "Authorization: Bearer <user uuid>".
func setupRouter(ws *WorkService) *gin.Engine {
	r := gin.New()
	Mount(r, ws, platform.NewDeps(ws.db, uuid.Parse))
	return r
}

// NewWorkService connects to the test database, for tests.
func NewWorkService() *WorkService {
	db, err := platform.OpenDB()
	if err != nil {
		panic(err)
	}
	ws, err := New(db)
	if err != nil {
		panic(err)
	}
	return ws
}

func (ws *WorkService) Close() {
	ws.db.Close()
}
