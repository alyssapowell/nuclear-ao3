package works

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/shared/models"
)

// Blocks and mutes.
//
// Both are chosen by pseud, and neither answer ever contains an account id,
// a username or another pseud of the same person. A person can only see the
// pseuds they picked themselves.
//
// A block (migration 041) is between two accounts and works in both
// directions. The works of the person on the other side stay readable; only
// comments, kudos and gifts close. A mute only hides one pseud's works and
// comments from the muter's own view.

// maxMutes is how many pseuds one account can mute.
const maxMutes = 100

// pseudRef is a pseud as a block or mute list shows it.
type pseudRef struct {
	Pseud     models.PseudSummary `json:"pseud"`
	CreatedAt time.Time           `json:"created_at"`
}

// pseudOwner finds the account behind a pseud and its name. ok is false when
// there is no such pseud.
func (ws *WorkService) pseudOwner(c *gin.Context, pseudID uuid.UUID) (owner uuid.UUID, name string, ok bool, err error) {
	err = ws.db.QueryRowContext(c.Request.Context(),
		`SELECT user_id, name FROM pseuds WHERE id = $1`, pseudID).Scan(&owner, &name)
	if err == sql.ErrNoRows {
		return uuid.Nil, "", false, nil
	}
	return owner, name, err == nil, err
}

// requestPseud reads {"pseud_id": ...} from the body and the signed-in user.
// It writes the error answer itself and returns false when the request can't go on.
func (ws *WorkService) requestPseud(c *gin.Context) (me *uuid.UUID, pseudID uuid.UUID, ok bool) {
	me = ws.getUserIDFromContext(c)
	if me == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return nil, uuid.Nil, false
	}
	var req struct {
		PseudID uuid.UUID `json:"pseud_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PseudID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pseud_required", "message": "Say which pseud with pseud_id."})
		return nil, uuid.Nil, false
	}
	return me, req.PseudID, true
}

// CreateBlock blocks the account behind a pseud. POST /api/v1/blocks {pseud_id}
//
// Blocking a second pseud of an account that is already blocked adds a second
// entry to the blocker's list, exactly as blocking a stranger would, so the
// answer never says whether two pseuds are the same person.
func (ws *WorkService) CreateBlock(c *gin.Context) {
	me, pseudID, ok := ws.requestPseud(c)
	if !ok {
		return
	}
	owner, name, found, err := ws.pseudOwner(c, pseudID)
	if err != nil {
		log.Printf("block: look up pseud %s: %v", pseudID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "pseud_not_found"})
		return
	}
	if owner == *me {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot_block_yourself", "message": "That pseud is yours."})
		return
	}

	var created time.Time
	err = ws.db.QueryRowContext(c.Request.Context(), `
		INSERT INTO user_blocks (blocker_id, blocked_id, pseud_id, pseud_name)
		VALUES ($1, $2, $3, left($4, 40))
		ON CONFLICT (blocker_id, pseud_id) DO UPDATE SET pseud_name = EXCLUDED.pseud_name
		RETURNING created_at`, *me, owner, pseudID, name).Scan(&created)
	if err != nil {
		log.Printf("block: insert: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"block": pseudRef{
		Pseud: models.PseudSummary{ID: pseudID.String(), Name: name}, CreatedAt: created}})
}

// ListBlocks lists the pseuds the signed-in user blocked. GET /api/v1/blocks
func (ws *WorkService) ListBlocks(c *gin.Context) {
	me := ws.getUserIDFromContext(c)
	if me == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	// The name is the pseud's current name, or the name it had when it was
	// blocked if the pseud has since been deleted.
	rows, err := ws.db.QueryContext(c.Request.Context(), `
		SELECT b.pseud_id, COALESCE(p.name, b.pseud_name), b.created_at
		FROM user_blocks b LEFT JOIN pseuds p ON p.id = b.pseud_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC, b.id`, *me)
	if err != nil {
		log.Printf("list blocks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list blocks"})
		return
	}
	defer rows.Close()
	out := []pseudRef{}
	for rows.Next() {
		var id uuid.UUID
		var r pseudRef
		var name string
		if err := rows.Scan(&id, &name, &r.CreatedAt); err != nil {
			log.Printf("list blocks: scan: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list blocks"})
			return
		}
		r.Pseud = models.PseudSummary{ID: id.String(), Name: name}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{"blocks": out})
}

// DeleteBlock removes a block. DELETE /api/v1/blocks/:pseud_id
//
// It only matches the pseud the blocker picked, never any other pseud of the
// same account, so it can't be used to test whether two pseuds are linked.
func (ws *WorkService) DeleteBlock(c *gin.Context) {
	me := ws.getUserIDFromContext(c)
	if me == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	pseudID, err := uuid.Parse(c.Param("pseud_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pseud ID"})
		return
	}
	res, err := ws.db.ExecContext(c.Request.Context(),
		`DELETE FROM user_blocks WHERE blocker_id = $1 AND pseud_id = $2`, *me, pseudID)
	if err != nil {
		log.Printf("unblock: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unblock"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_blocked"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Unblocked"})
}

// CreateMute mutes one pseud. POST /api/v1/mutes {pseud_id}
//
// The mute covers that pseud only, never the account's other pseuds.
func (ws *WorkService) CreateMute(c *gin.Context) {
	me, pseudID, ok := ws.requestPseud(c)
	if !ok {
		return
	}
	owner, name, found, err := ws.pseudOwner(c, pseudID)
	if err != nil {
		log.Printf("mute: look up pseud %s: %v", pseudID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mute"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "pseud_not_found"})
		return
	}
	if owner == *me {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot_mute_yourself", "message": "That pseud is yours."})
		return
	}

	var count int
	if err := ws.db.QueryRowContext(c.Request.Context(),
		`SELECT count(*) FROM user_mutes WHERE muter_id = $1 AND muted_pseud_id <> $2`, *me, pseudID).Scan(&count); err != nil {
		log.Printf("mute: count: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mute"})
		return
	}
	if count >= maxMutes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mute_limit", "message": "You have muted as many pseuds as is allowed."})
		return
	}

	var created time.Time
	err = ws.db.QueryRowContext(c.Request.Context(), `
		INSERT INTO user_mutes (muter_id, muted_pseud_id)
		VALUES ($1, $2)
		ON CONFLICT (muter_id, muted_pseud_id) DO UPDATE SET updated_at = now()
		RETURNING created_at`, *me, pseudID).Scan(&created)
	if err != nil {
		log.Printf("mute: insert: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mute"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"mute": pseudRef{
		Pseud: models.PseudSummary{ID: pseudID.String(), Name: name}, CreatedAt: created}})
}

// ListMutes lists the pseuds the signed-in user muted. GET /api/v1/mutes
func (ws *WorkService) ListMutes(c *gin.Context) {
	me := ws.getUserIDFromContext(c)
	if me == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	rows, err := ws.db.QueryContext(c.Request.Context(), `
		SELECT p.id, p.name, m.created_at
		FROM user_mutes m JOIN pseuds p ON p.id = m.muted_pseud_id
		WHERE m.muter_id = $1
		ORDER BY m.created_at DESC, m.id`, *me)
	if err != nil {
		log.Printf("list mutes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list mutes"})
		return
	}
	defer rows.Close()
	out := []pseudRef{}
	for rows.Next() {
		var id uuid.UUID
		var r pseudRef
		var name string
		if err := rows.Scan(&id, &name, &r.CreatedAt); err != nil {
			log.Printf("list mutes: scan: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list mutes"})
			return
		}
		r.Pseud = models.PseudSummary{ID: id.String(), Name: name}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{"mutes": out})
}

// DeleteMute removes a mute. DELETE /api/v1/mutes/:pseud_id
func (ws *WorkService) DeleteMute(c *gin.Context) {
	me := ws.getUserIDFromContext(c)
	if me == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	pseudID, err := uuid.Parse(c.Param("pseud_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pseud ID"})
		return
	}
	res, err := ws.db.ExecContext(c.Request.Context(),
		`DELETE FROM user_mutes WHERE muter_id = $1 AND muted_pseud_id = $2`, *me, pseudID)
	if err != nil {
		log.Printf("unmute: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmute"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_muted"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Unmuted"})
}

// blockedFromWork says whether a block closes comments and kudos on the work
// for this viewer. See blocked_from_work in migrations 041 and 042. It is
// always false on a work that hides its creators: there the block is silent
// (blockedSilently), because refusing would tell the viewer who wrote it.
func (ws *WorkService) blockedFromWork(c *gin.Context, workID uuid.UUID, viewer *uuid.UUID) (bool, error) {
	if viewer == nil {
		return false, nil
	}
	var blocked bool
	err := ws.db.QueryRowContext(c.Request.Context(),
		`SELECT blocked_from_work($1, $2)`, workID, *viewer).Scan(&blocked)
	return blocked, err
}

// blockedSilently says whether the viewer is blocked by a creator of a work
// that hides its creators. Nothing is refused and nothing looks different to
// them; what they post is stored as shadowed (migration 042), so only they
// and moderators can see it, it isn't counted, and it notifies nobody.
func (ws *WorkService) blockedSilently(c *gin.Context, workID uuid.UUID, viewer *uuid.UUID) (bool, error) {
	if viewer == nil {
		return false, nil
	}
	var silent bool
	err := ws.db.QueryRowContext(c.Request.Context(),
		`SELECT blocked_silently($1, $2)`, workID, *viewer).Scan(&silent)
	return silent, err
}

// commentsDisabledBody is the 403 answer for a comment, reply or kudos the
// viewer is blocked from. It says the same as a work whose comments are off.
func commentsDisabledBody() gin.H {
	return gin.H{"error": "comments_disabled", "message": "Comments are disabled."}
}
