package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"nuclear-ao3/internal/audit"
	"nuclear-ao3/internal/jobs"
	"nuclear-ao3/shared/email"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// App vetting (security audit, phase 3).
//
// Anyone in good standing can register an app, but it starts as "pending"
// and does nothing until an admin approves it. Only an approved app can
// authorize, exchange a code, refresh, or have its tokens accepted; the
// database keeps is_active equal to "status = approved" (migration 047).

const (
	maxAppsPerPerson    = 10
	maxPendingPerPerson = 5
	maxReviewNoteLen    = 1000
)

// The statuses an app can have.
const (
	statusPending   = "pending"
	statusApproved  = "approved"
	statusRejected  = "rejected"
	statusSuspended = "suspended"
)

var appStatuses = []string{statusPending, statusApproved, statusRejected, statusSuspended}

// ---------------------------------------------------------------------
// The review itself (used by the admin routes and by "nuclear apps")
// ---------------------------------------------------------------------

// Reasons a review can be refused.
var (
	ErrAppNotFound  = errors.New("no such app")
	ErrWrongStatus  = errors.New("the app is not in a status that allows that")
	ErrNoteRequired = errors.New("a note is required to reject or suspend an app")
	ErrNoteTooLong  = errors.New("the note is too long")
	ErrUnknownCmd   = errors.New("unknown review action")
	// ErrAppChanged means the app was edited after the reviewer looked at it.
	ErrAppChanged = errors.New("the app changed since you looked at it; look at its new settings and decide again")
)

// reviewRule says which statuses an action may start from and where it ends.
type reviewRule struct {
	from        []string
	to          string
	needsNote   bool
	revoke      bool // ends every token, code and consent the app holds
	emailsOwner bool
	display     bool // decides the owner's proposed display changes; the app stays approved
}

var reviewRules = map[string]reviewRule{
	"approve":   {from: []string{statusPending, statusRejected}, to: statusApproved, emailsOwner: true},
	"reject":    {from: []string{statusPending}, to: statusRejected, needsNote: true, revoke: true, emailsOwner: true},
	"suspend":   {from: []string{statusApproved, statusPending}, to: statusSuspended, needsNote: true, revoke: true, emailsOwner: true},
	"reinstate": {from: []string{statusSuspended}, to: statusApproved},
	// Display changes proposed by the owner of an approved app. Nobody is
	// signed out by either decision.
	"approve_display": {from: []string{statusApproved}, to: statusApproved, emailsOwner: true, display: true},
	"reject_display":  {from: []string{statusApproved}, to: statusApproved, needsNote: true, emailsOwner: true, display: true},
}

// ReviewResult is what a review changed.
type ReviewResult struct {
	ClientID uuid.UUID
	Name     string
	From, To string
	Emailed  bool
}

// ReviewApp approves, rejects, suspends or reinstates an app (action is
// "approve", "reject", "suspend" or "reinstate"), or approves or rejects an
// owner's proposed display changes ("approve_display", "reject_display").
// expectRevision is the revision the reviewer looked at; if the app has
// changed since, nothing happens and ErrAppChanged is returned. 0 skips the
// check (the command line, which shows the settings and acts at once). The status change, the
// revocation of everything the app holds (for reject and suspend), the audit
// row and the email to the owner are one transaction. mail may be nil, which
// turns the email off. The owner is not emailed about their own action.
func ReviewApp(ctx context.Context, db *sql.DB, mail *email.Config, actor audit.Actor, id uuid.UUID, action, note string, expectRevision int) (*ReviewResult, error) {
	rule, ok := reviewRules[action]
	if !ok {
		return nil, ErrUnknownCmd
	}
	note = strings.TrimSpace(note)
	if rule.needsNote && note == "" {
		return nil, ErrNoteRequired
	}
	if len([]rune(note)) > maxReviewNoteLen {
		return nil, ErrNoteTooLong
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var name, from string
	var owner *uuid.UUID
	var revision int
	var displayPending bool
	err = tx.QueryRowContext(ctx,
		`SELECT client_name, status, owner_id, revision, display_review_pending FROM oauth_clients WHERE client_id = $1 FOR UPDATE`, id).
		Scan(&name, &from, &owner, &revision, &displayPending)
	if err == sql.ErrNoRows {
		return nil, ErrAppNotFound
	}
	if err != nil {
		return nil, err
	}
	if expectRevision > 0 && revision != expectRevision {
		return nil, ErrAppChanged
	}
	if !contains(rule.from, from) {
		return nil, fmt.Errorf("%w: it is %s", ErrWrongStatus, from)
	}
	if rule.display && !displayPending {
		return nil, fmt.Errorf("%w: it has no display changes waiting", ErrWrongStatus)
	}

	var reviewer any // NULL when it was done from the command line
	if actor.UserID != nil {
		reviewer = *actor.UserID
	}
	clearProposal := `display_review_pending = false, proposed_name = NULL, proposed_description = NULL,
		proposed_website = NULL, proposed_logo_url = NULL`
	apply := ""
	if action == "approve_display" {
		apply = `client_name = COALESCE(proposed_name, client_name), description = COALESCE(proposed_description, description),
			website = COALESCE(proposed_website, website), logo_url = COALESCE(proposed_logo_url, logo_url), `
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE oauth_clients SET `+apply+clearProposalFor(rule, clearProposal)+`status = $2, reviewed_by = $3, reviewed_at = now(),
			review_note = NULLIF($4, ''), updated_at = now()
		WHERE client_id = $1`, id, rule.to, reviewer, note); err != nil {
		return nil, err
	}
	if rule.revoke {
		if err = revokeClientAccess(ctx, tx, id, true); err != nil {
			return nil, err
		}
	}
	if err = audit.Record(ctx, tx, actor, "oauth_client."+action, "oauth_client", id.String(), note,
		map[string]any{"client_name": name, "from": from, "to": rule.to}); err != nil {
		return nil, err
	}

	res := &ReviewResult{ClientID: id, Name: name, From: from, To: rule.to}
	selfAction := owner != nil && actor.UserID != nil && *owner == *actor.UserID
	if rule.emailsOwner && mail != nil && owner != nil && !selfAction {
		var to, username string
		err = tx.QueryRowContext(ctx, `SELECT email, username FROM users WHERE id = $1`, *owner).Scan(&to, &username)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if err == nil {
			outcome := rule.to
			switch action {
			case "approve_display":
				outcome = "display_approved"
			case "reject_display":
				outcome = "display_rejected"
			}
			msg, merr := mail.AppReviewed(to, username, name, outcome, note)
			if merr != nil {
				return nil, merr
			}
			if err = jobs.Enqueue(ctx, tx, "email", msg); err != nil {
				return nil, err
			}
			res.Emailed = true
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return res, nil
}

// clearProposalFor says whether a decision also clears the owner's proposed
// display changes: both display decisions do. (Other decisions leave them.)
func clearProposalFor(rule reviewRule, clause string) string {
	if rule.display {
		return clause + ", "
	}
	return ""
}

// reviewStatusCode turns a ReviewApp error into an HTTP status.
func reviewStatusCode(err error) int {
	switch {
	case errors.Is(err, ErrAppNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrWrongStatus), errors.Is(err, ErrAppChanged):
		return http.StatusConflict
	case errors.Is(err, ErrNoteRequired), errors.Is(err, ErrNoteTooLong), errors.Is(err, ErrUnknownCmd):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// AppSummary is one line of the review queue.
type AppSummary struct {
	ID           uuid.UUID
	Name         string
	Description  string
	Website      string
	LogoURL      string
	Status       string
	ReviewNote   string
	RedirectURIs []string
	Scopes       []string
	GrantTypes   []string
	IsPublic     bool
	IsTrusted    bool
	CreatedAt    time.Time
	ReviewedAt   *time.Time
	// Revision is the version the reviewer is looking at; a decision has to
	// send it back.
	Revision int
	// DisplayPending says the owner proposed new display details for this
	// approved app. Until an admin approves them, everything above is what
	// people see.
	DisplayPending                    bool
	ProposedName, ProposedDescription string
	ProposedWebsite, ProposedLogoURL  string
	// The owner is shown as their default pseud only: no account id and no
	// username. Zero when the account is gone.
	OwnerPseudID   *uuid.UUID
	OwnerPseudName string
}

// ListApps returns apps in one status (all of them when status is empty),
// oldest first, so the queue is worked in order.
func ListApps(ctx context.Context, db *sql.DB, status string, limit, offset int) ([]AppSummary, int, error) {
	if status != "" && status != "display" && !contains(appStatuses, status) {
		return nil, 0, fmt.Errorf("status must be one of %s, or display", strings.Join(appStatuses, ", "))
	}
	var total int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM oauth_clients WHERE $1 = '' OR status = $1 OR ($1 = 'display' AND display_review_pending)`, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT oc.client_id, oc.client_name, COALESCE(oc.description, ''), COALESCE(oc.website, ''), COALESCE(oc.logo_url, ''),
			oc.status, COALESCE(oc.review_note, ''), oc.redirect_uris, oc.scopes, oc.grant_types,
			oc.is_public, oc.is_trusted, oc.created_at, oc.reviewed_at, oc.revision, oc.display_review_pending,
			COALESCE(oc.proposed_name, ''), COALESCE(oc.proposed_description, ''),
			COALESCE(oc.proposed_website, ''), COALESCE(oc.proposed_logo_url, ''), p.id, p.name
		FROM oauth_clients oc
		LEFT JOIN LATERAL (
			SELECT id, name FROM pseuds WHERE user_id = oc.owner_id AND is_default ORDER BY created_at, id LIMIT 1
		) p ON true
		WHERE $1 = '' OR oc.status = $1 OR ($1 = 'display' AND oc.display_review_pending)
		ORDER BY oc.created_at, oc.client_id
		LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []AppSummary
	for rows.Next() {
		var a AppSummary
		var pseudID *uuid.UUID
		var pseudName sql.NullString
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.Website, &a.LogoURL, &a.Status, &a.ReviewNote,
			pq.Array(&a.RedirectURIs), pq.Array(&a.Scopes), pq.Array(&a.GrantTypes),
			&a.IsPublic, &a.IsTrusted, &a.CreatedAt, &a.ReviewedAt, &a.Revision, &a.DisplayPending,
			&a.ProposedName, &a.ProposedDescription, &a.ProposedWebsite, &a.ProposedLogoURL, &pseudID, &pseudName); err != nil {
			return nil, 0, err
		}
		a.OwnerPseudID, a.OwnerPseudName = pseudID, pseudName.String
		out = append(out, a)
	}
	return out, total, rows.Err()
}

// ---------------------------------------------------------------------
// Admin routes: GET /api/v1/admin/apps and the four actions
// ---------------------------------------------------------------------

// AdminListApps is the review queue. ?status= picks one status (default
// pending); ?status=all lists every app.
func (as *AuthService) AdminListApps(c *gin.Context) {
	status := c.DefaultQuery("status", statusPending)
	if status == "all" {
		status = ""
	}
	if status != "" && status != "display" && !contains(appStatuses, status) {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_request", "status must be pending, approved, rejected, suspended, display or all"))
		return
	}
	page, limit, offset := pageParams(c)
	apps, total, err := ListApps(c.Request.Context(), as.db, status, limit, offset)
	if err != nil {
		log.Printf("list apps: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list apps"})
		return
	}
	out := make([]gin.H, 0, len(apps))
	for _, a := range apps {
		var owner any // the owner's default pseud, or null
		if a.OwnerPseudID != nil {
			owner = gin.H{"id": a.OwnerPseudID, "name": a.OwnerPseudName}
		}
		out = append(out, gin.H{
			"client_id": a.ID, "client_name": a.Name, "description": a.Description,
			"website": a.Website, "logo_url": a.LogoURL, "status": a.Status, "review_note": a.ReviewNote,
			"redirect_uris": a.RedirectURIs, "scopes": a.Scopes, "grant_types": a.GrantTypes,
			"is_public": a.IsPublic, "is_trusted": a.IsTrusted,
			"created_at": a.CreatedAt, "reviewed_at": a.ReviewedAt, "owner": owner,
			"revision": a.Revision, "display_review_pending": a.DisplayPending,
			"proposed": proposedView(a.DisplayPending, a.ProposedName, a.ProposedDescription, a.ProposedWebsite, a.ProposedLogoURL),
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"apps":       out,
		"pagination": gin.H{"page": page, "limit": limit, "total": total, "pages": (total + limit - 1) / limit},
	})
}

// proposedView is the owner's proposed display details, or nil when none wait.
func proposedView(pending bool, name, desc, site, logo string) any {
	if !pending {
		return nil
	}
	return gin.H{"client_name": name, "description": desc, "website": site, "logo_url": logo}
}

// adminReview returns the handler for one review action.
func (as *AuthService) adminReview(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
			return
		}
		actor, ok := as.adminActor(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
			return
		}
		var body struct {
			Note     string `json:"note"`
			Revision int    `json:"revision"`
		}
		if c.Request.ContentLength != 0 {
			dec := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&body); err != nil {
				c.JSON(http.StatusBadRequest, problemJSON("invalid_request", "Invalid request data"))
				return
			}
		}
		// A decision applies to the version the admin looked at.
		if body.Revision < 1 {
			c.JSON(http.StatusBadRequest, problemJSON("invalid_request", "revision is required: send the revision you reviewed"))
			return
		}
		res, err := ReviewApp(c.Request.Context(), as.db, as.mail, actor, id, action, body.Note, body.Revision)
		if err != nil {
			code := reviewStatusCode(err)
			if code == http.StatusInternalServerError {
				log.Printf("app %s: %v", action, err)
				c.JSON(code, gin.H{"error": "Failed to " + action + " the app"})
				return
			}
			kind := "invalid_request"
			if errors.Is(err, ErrAppChanged) {
				kind = "app_changed"
			}
			c.JSON(code, problemJSON(kind, err.Error()))
			return
		}
		c.JSON(http.StatusOK, gin.H{"client_id": res.ClientID, "status": res.To})
	}
}

// ---------------------------------------------------------------------
// Owner routes: /api/v1/apps
// ---------------------------------------------------------------------

// appCreate is what a person may send to register an app. Any other field,
// is_trusted and is_first_party included, is refused.
type appCreate struct {
	Name            string   `json:"client_name"`
	Description     string   `json:"description"`
	Website         string   `json:"website"`
	LogoURL         string   `json:"logo_url"`
	RedirectURIs    []string `json:"redirect_uris"`
	Scopes          []string `json:"scopes"`
	GrantTypes      []string `json:"grant_types"`
	ResponseTypes   []string `json:"response_types"`
	IsPublic        bool     `json:"is_public"`
	AccessTokenTTL  int      `json:"access_token_ttl"`
	RefreshTokenTTL int      `json:"refresh_token_ttl"`
}

// appPatch is what an owner may change. A field left out is left alone.
type appPatch struct {
	Name         *string   `json:"client_name"`
	Description  *string   `json:"description"`
	Website      *string   `json:"website"`
	LogoURL      *string   `json:"logo_url"`
	RedirectURIs *[]string `json:"redirect_uris"`
	Scopes       *[]string `json:"scopes"`
}

func decodeStrict(c *gin.Context, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_request", "Invalid request data"))
		return false
	}
	return true
}

// checkAppText validates the name, description and addresses of an app and
// returns the problem found, if any.
func checkAppText(name, description, website, logo string) (code, desc string) {
	if n := len([]rune(strings.TrimSpace(name))); n == 0 || n > maxClientNameLen {
		return "invalid_client_metadata", "client_name must be 1 to 100 characters"
	}
	if len([]rune(description)) > maxDescriptionLen {
		return "invalid_client_metadata", "description is too long"
	}
	if !validOptionalURL(website) || !validOptionalURL(logo) {
		return "invalid_client_metadata", "website and logo_url must be https addresses"
	}
	return "", ""
}

// currentUser is the signed-in person (RequireUser has run).
func currentUser(c *gin.Context) (uuid.UUID, bool) {
	v, _ := c.Get("user_uuid")
	id, ok := v.(uuid.UUID)
	return id, ok
}

// RegisterApp is POST /api/v1/apps: a person registers an app. It is created
// pending, owned by them, and does nothing until an admin approves it.
func (as *AuthService) RegisterApp(c *gin.Context) {
	owner, ok := currentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	var req appCreate
	if !decodeStrict(c, &req) {
		return
	}
	if code, desc := checkAppText(req.Name, req.Description, req.Website, req.LogoURL); code != "" {
		c.JSON(http.StatusBadRequest, problemJSON(code, desc))
		return
	}
	redirectURIs, problem := cleanRedirectURIs(req.RedirectURIs, req.IsPublic)
	if problem != "" {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_redirect_uri", problem))
		return
	}
	scopes, problem := cleanScopes(req.Scopes)
	if problem != "" {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_scope", problem))
		return
	}
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code"}
	}
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}
	grantTypes, ok := cleanChoices(req.GrantTypes, allowedGrantTypes)
	if !ok || !contains(grantTypes, "authorization_code") {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_client_metadata",
			"grant_types must be authorization_code, optionally with refresh_token"))
		return
	}
	responseTypes, ok := cleanChoices(req.ResponseTypes, allowedResponseTypes)
	if !ok {
		c.JSON(http.StatusBadRequest, problemJSON("invalid_client_metadata", "response_types must be code"))
		return
	}

	clientID := uuid.New()
	var clientSecret, secretHash string
	if !req.IsPublic {
		var err error
		if clientSecret, secretHash, err = newClientSecret(); err != nil {
			c.JSON(http.StatusInternalServerError, problemJSON("server_error", "Failed to generate client secret"))
			return
		}
	}
	accessTTL, refreshTTL := clampAccessTTL(req.AccessTokenTTL), clampRefreshTTL(req.RefreshTokenTTL)
	name := strings.TrimSpace(req.Name)

	ctx := c.Request.Context()
	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, problemJSON("server_error", "Failed to register app"))
		return
	}
	defer tx.Rollback()

	// Locking the person's row makes two requests at once count one after
	// the other, so the limits can't be beaten by sending many together.
	var total, pending int
	if _, err = tx.ExecContext(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, owner); err == nil {
		err = tx.QueryRowContext(ctx, `
			SELECT count(*), count(*) FILTER (WHERE status = 'pending') FROM oauth_clients WHERE owner_id = $1`,
			owner).Scan(&total, &pending)
	}
	if err != nil {
		log.Printf("register app: %v", err)
		c.JSON(http.StatusInternalServerError, problemJSON("server_error", "Failed to register app"))
		return
	}
	if total >= maxAppsPerPerson {
		c.JSON(http.StatusConflict, problemJSON("app_limit", fmt.Sprintf("You can have at most %d apps. Delete one you no longer use.", maxAppsPerPerson)))
		return
	}
	if pending >= maxPendingPerPerson {
		c.JSON(http.StatusConflict, problemJSON("pending_limit", fmt.Sprintf("You can have at most %d apps waiting for review. Wait for a decision before adding more.", maxPendingPerPerson)))
		return
	}

	var createdAt time.Time
	err = tx.QueryRowContext(ctx, `
		INSERT INTO oauth_clients (
			client_id, client_secret, client_name, description, website, logo_url,
			redirect_uris, scopes, grant_types, response_types, is_public, is_confidential,
			is_trusted, is_first_party, owner_id, access_token_ttl, refresh_token_ttl, status
		) VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, false, false, $13, $14, $15, 'pending')
		RETURNING created_at`,
		clientID, secretHash, name, req.Description, req.Website, req.LogoURL,
		pq.Array(redirectURIs), pq.Array(scopes), pq.Array(grantTypes), pq.Array(responseTypes),
		req.IsPublic, !req.IsPublic, owner, accessTTL, refreshTTL).Scan(&createdAt)
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("register app: %v", err)
		c.JSON(http.StatusInternalServerError, problemJSON("server_error", "Failed to register app"))
		return
	}

	resp := gin.H{
		"client_id": clientID, "client_name": name, "description": req.Description,
		"website": req.Website, "logo_url": req.LogoURL, "status": statusPending,
		"redirect_uris": redirectURIs, "scopes": scopes, "grant_types": grantTypes,
		"response_types": responseTypes, "is_public": req.IsPublic,
		"access_token_ttl": accessTTL, "refresh_token_ttl": refreshTTL, "created_at": createdAt,
	}
	if !req.IsPublic {
		resp["client_secret"] = clientSecret // shown this once
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, resp)
}

// newClientSecret makes a secret and its bcrypt hash.
func newClientSecret() (secret, hash string, err error) {
	if secret, err = generateClientSecret(); err != nil {
		return "", "", err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return secret, string(h), nil
}

// ListMyApps is GET /api/v1/apps: the caller's own apps. The secret is never
// returned, and nobody else's apps are listed.
func (as *AuthService) ListMyApps(c *gin.Context) {
	owner, ok := currentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	rows, err := as.db.QueryContext(c.Request.Context(), `
		SELECT client_id, client_name, COALESCE(description, ''), COALESCE(website, ''), COALESCE(logo_url, ''),
			status, COALESCE(review_note, ''), redirect_uris, scopes, is_public, created_at, reviewed_at,
			display_review_pending, COALESCE(proposed_name, ''), COALESCE(proposed_description, ''),
			COALESCE(proposed_website, ''), COALESCE(proposed_logo_url, '')
		FROM oauth_clients WHERE owner_id = $1 ORDER BY created_at, client_id`, owner)
	if err != nil {
		log.Printf("list my apps: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list apps"})
		return
	}
	defer rows.Close()
	apps := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, desc, website, logo, status, note string
		var redirects, scopes []string
		var public bool
		var created time.Time
		var reviewed *time.Time
		var pending bool
		var pName, pDesc, pSite, pLogo string
		if err := rows.Scan(&id, &name, &desc, &website, &logo, &status, &note,
			pq.Array(&redirects), pq.Array(&scopes), &public, &created, &reviewed,
			&pending, &pName, &pDesc, &pSite, &pLogo); err != nil {
			log.Printf("list my apps: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list apps"})
			return
		}
		apps = append(apps, gin.H{
			"client_id": id, "client_name": name, "description": desc, "website": website, "logo_url": logo,
			"status": status, "review_note": note, "redirect_uris": redirects, "scopes": scopes,
			"is_public": public, "created_at": created, "reviewed_at": reviewed,
			"display_review_pending": pending, "proposed": proposedView(pending, pName, pDesc, pSite, pLogo),
		})
	}
	c.JSON(http.StatusOK, gin.H{"apps": apps})
}

// sameSet says whether two lists hold the same values, in any order.
func sameSet(a, b []string) bool {
	x, y := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

// ownedApp loads one of the caller's apps. It answers 404 for an app that
// is someone else's, so the answer doesn't say whether the id exists.
func (as *AuthService) ownedApp(c *gin.Context) (*appRow, bool) {
	owner, ok := currentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return nil, false
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return nil, false
	}
	client, err := as.getClientForAdmin(id)
	if err != nil || client.OwnerID == nil || *client.OwnerID != owner {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return nil, false
	}
	// The archive's own apps are managed by admins only.
	if client.IsFirstParty {
		c.JSON(http.StatusForbidden, gin.H{"error": "This app is managed by the archive's admins"})
		return nil, false
	}
	return &appRow{client.ID, client.Name, client.Status, client.IsPublic, client.RedirectURIs, client.Scopes}, true
}

type appRow struct {
	id           uuid.UUID
	name, status string
	isPublic     bool
	redirectURIs []string
	scopes       []string
}

// UpdateMyApp is PATCH /api/v1/apps/:id. The owner can edit the name,
// description, website, logo, redirect addresses and scopes, and nothing
// else (the status can only be changed by an admin).
//
// What an edit does depends on the app as it is when the row is locked, not
// as it was a moment earlier, so it can't slip past an admin's decision:
//
//   - Changing the redirect addresses or scopes puts the app back to pending
//     (whatever it was, except suspended, which can't be edited). If it was
//     approved, every token, code and consent it holds ends too: those were
//     given for the reviewed settings, and an old consent would let a new
//     address receive codes without asking anyone.
//   - Changing only how an approved app is shown (name, website, logo,
//     description) is kept aside as a proposal. People keep seeing the
//     approved details, and nobody is signed out, until an admin approves the
//     change. Otherwise an approved app could rename itself to look like
//     someone trusted.
//   - On an app that is not approved, display changes just apply: nobody is
//     using it yet, and an admin reviews all of it.
//   - A rejected app that is edited goes back to pending (it is resubmitted).
//   - Any edit drops the app's "trusted" mark and is written to the audit log.
func (as *AuthService) UpdateMyApp(c *gin.Context) {
	app, ok := as.ownedApp(c)
	if !ok {
		return
	}
	var up appPatch
	if !decodeStrict(c, &up) {
		return
	}
	bad := func(code, desc string) { c.JSON(http.StatusBadRequest, problemJSON(code, desc)) }

	if up.Name != nil {
		trimmed := strings.TrimSpace(*up.Name)
		if n := len([]rune(trimmed)); n == 0 || n > maxClientNameLen {
			bad("invalid_client_metadata", "client_name must be 1 to 100 characters")
			return
		}
		up.Name = &trimmed
	}
	if up.Description != nil && len([]rune(*up.Description)) > maxDescriptionLen {
		bad("invalid_client_metadata", "description is too long")
		return
	}
	if up.Website != nil && !validOptionalURL(*up.Website) {
		bad("invalid_client_metadata", "website must be an https address")
		return
	}
	if up.LogoURL != nil && !validOptionalURL(*up.LogoURL) {
		bad("invalid_client_metadata", "logo_url must be an https address")
		return
	}
	var newURIs, newScopes []string
	if up.RedirectURIs != nil {
		uris, problem := cleanRedirectURIs(*up.RedirectURIs, app.isPublic)
		if problem != "" {
			bad("invalid_redirect_uri", problem)
			return
		}
		newURIs = uris
	}
	if up.Scopes != nil {
		scopes, problem := cleanScopes(*up.Scopes)
		if problem != "" {
			bad("invalid_scope", problem)
			return
		}
		newScopes = scopes
	}
	if up.Name == nil && up.Description == nil && up.Website == nil && up.LogoURL == nil &&
		up.RedirectURIs == nil && up.Scopes == nil {
		bad("invalid_request", "Nothing to change")
		return
	}
	actor, ok := as.adminActor(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}

	ctx := c.Request.Context()
	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update app"})
		return
	}
	defer tx.Rollback()

	// Lock the row and decide from what is in it now.
	var cur struct {
		status                   string
		name, desc, site, logo   string
		uris, scopes             []string
		pending                  bool
		pName, pDesc, pSite, pLo sql.NullString
	}
	err = tx.QueryRowContext(ctx, `
		SELECT status, client_name, COALESCE(description, ''), COALESCE(website, ''), COALESCE(logo_url, ''),
			redirect_uris, scopes, display_review_pending,
			proposed_name, proposed_description, proposed_website, proposed_logo_url
		FROM oauth_clients WHERE client_id = $1 AND owner_id = $2 FOR UPDATE`, app.id, actor.UserID).Scan(
		&cur.status, &cur.name, &cur.desc, &cur.site, &cur.logo,
		pq.Array(&cur.uris), pq.Array(&cur.scopes), &cur.pending,
		&cur.pName, &cur.pDesc, &cur.pSite, &cur.pLo)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}
	if err != nil {
		log.Printf("update app: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update app"})
		return
	}
	if cur.status == statusSuspended {
		c.JSON(http.StatusConflict, problemJSON("suspended", "This app is suspended. Ask the admins to look at it."))
		return
	}

	sensitive := (newURIs != nil && !sameSet(newURIs, cur.uris)) || (newScopes != nil && !sameSet(newScopes, cur.scopes))
	newStatus := cur.status
	if sensitive || cur.status == statusRejected {
		newStatus = statusPending
	}
	resubmitted := newStatus != cur.status
	proposeOnly := cur.status == statusApproved && !sensitive

	var sets []string
	var args []any
	set := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	pick := func(patch *string, current string) string {
		if patch != nil {
			return *patch
		}
		return current
	}
	// What the display would be after this edit, starting from the earlier
	// proposal if one is waiting.
	base := struct{ name, desc, site, logo string }{cur.name, cur.desc, cur.site, cur.logo}
	if cur.pending {
		base.name, base.desc = cur.pName.String, cur.pDesc.String
		base.site, base.logo = cur.pSite.String, cur.pLo.String
	}
	next := struct{ name, desc, site, logo string }{
		pick(up.Name, base.name), pick(up.Description, base.desc), pick(up.Website, base.site), pick(up.LogoURL, base.logo)}
	displayGiven := up.Name != nil || up.Description != nil || up.Website != nil || up.LogoURL != nil

	proposed := false
	if proposeOnly {
		if displayGiven {
			same := next.name == cur.name && next.desc == cur.desc && next.site == cur.site && next.logo == cur.logo
			if same { // back to the approved details: nothing left to review
				set("display_review_pending", false)
				sets = append(sets, "proposed_name = NULL", "proposed_description = NULL", "proposed_website = NULL", "proposed_logo_url = NULL")
			} else {
				proposed = true
				set("proposed_name", next.name)
				set("proposed_description", next.desc)
				set("proposed_website", next.site)
				set("proposed_logo_url", next.logo)
				set("display_review_pending", true)
			}
		}
	} else {
		// Not live, or going back to full review: the display applies now,
		// and any proposal is folded in.
		set("client_name", next.name)
		set("description", next.desc)
		set("website", next.site)
		set("logo_url", next.logo)
		set("display_review_pending", false)
		sets = append(sets, "proposed_name = NULL", "proposed_description = NULL", "proposed_website = NULL", "proposed_logo_url = NULL")
	}
	if newURIs != nil {
		set("redirect_uris", pq.Array(newURIs))
	}
	if newScopes != nil {
		set("scopes", pq.Array(newScopes))
	}
	set("status", newStatus)
	sets = append(sets, "is_trusted = false")
	if resubmitted {
		sets = append(sets, "review_note = NULL")
	}

	args = append(args, app.id)
	_, err = tx.ExecContext(ctx, "UPDATE oauth_clients SET "+strings.Join(sets, ", ")+
		fmt.Sprintf(", updated_at = now() WHERE client_id = $%d", len(args)), args...)
	if err == nil && cur.status == statusApproved && sensitive {
		err = revokeClientAccess(ctx, tx, app.id, true)
	}
	if err == nil {
		changed := []string{}
		for name, on := range map[string]bool{
			"client_name": up.Name != nil, "description": up.Description != nil, "website": up.Website != nil,
			"logo_url": up.LogoURL != nil, "redirect_uris": up.RedirectURIs != nil, "scopes": up.Scopes != nil} {
			if on {
				changed = append(changed, name)
			}
		}
		sort.Strings(changed)
		err = audit.Record(ctx, tx, actor, "oauth_client.owner_update", "oauth_client", app.id.String(), "",
			map[string]any{"client_name": cur.name, "fields": changed, "from": cur.status, "to": newStatus,
				"sensitive": sensitive, "display_proposed": proposed})
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("update app: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update app"})
		return
	}
	waiting := proposeOnly && cur.pending
	if proposeOnly && displayGiven {
		waiting = proposed
	}
	c.JSON(http.StatusOK, gin.H{"client_id": app.id, "status": newStatus, "display_review_pending": waiting})
}

// ResetMyAppSecret is POST /api/v1/apps/:id/reset-secret. The new secret is
// shown once; tokens issued with the old one are revoked, as when an admin
// resets it.
func (as *AuthService) ResetMyAppSecret(c *gin.Context) {
	app, ok := as.ownedApp(c)
	if !ok {
		return
	}
	if app.isPublic {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Public apps don't have secrets"})
		return
	}
	secret, hash, err := newClientSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate secret"})
		return
	}
	ctx := c.Request.Context()
	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update secret"})
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `UPDATE oauth_clients SET client_secret = $1, updated_at = now() WHERE client_id = $2`, hash, app.id)
	if err == nil {
		err = revokeClientAccess(ctx, tx, app.id, false)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("reset app secret: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update secret"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"client_id": app.id, "client_secret": secret})
}

// DeleteMyApp is DELETE /api/v1/apps/:id. It ends everything the app holds
// (the same revocation an admin's delete does) and removes the app.
func (as *AuthService) DeleteMyApp(c *gin.Context) {
	app, ok := as.ownedApp(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete app"})
		return
	}
	defer tx.Rollback()
	err = revokeClientAccess(ctx, tx, app.id, true)
	if err == nil {
		_, err = tx.ExecContext(ctx, `DELETE FROM oauth_clients WHERE client_id = $1`, app.id)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("delete app: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete app"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "App deleted"})
}
