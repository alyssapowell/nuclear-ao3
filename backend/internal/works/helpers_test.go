package works

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"nuclear-ao3/internal/testdb"

	"nuclear-ao3/internal/platform"
)

// setupRouter builds a router with only the works routes, for tests.
// Tests log in by sending "Authorization: Bearer <user uuid>".
func setupRouter(ws *WorkService) *gin.Engine {
	r := gin.New()
	Mount(r, ws, platform.NewDeps(ws.db, uuid.Parse))
	return r
}

// newTestWorkService builds the service on a fresh test database.
func newTestWorkService(t *testing.T) *WorkService {
	t.Helper()
	ws, err := New(testdb.New(t))
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func (ws *WorkService) Close() {
	ws.db.Close()
}
