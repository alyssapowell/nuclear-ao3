package moderation_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"nuclear-ao3/internal/audit"
	"nuclear-ao3/internal/moderation"
)

func TestSanctionsFollowTheStaffHierarchy(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	svc := moderation.New(e.db)
	adminID, adminTok := e.user("boss", "admin")
	otherAdmin, _ := e.user("boss2", "admin")
	modID, modTok := e.user("mod", "moderator")
	otherMod, _ := e.user("mod2", "moderator")
	plain, _ := e.user("plain")
	actor := func(id uuid.UUID, name string) audit.Actor { return audit.UserActor(id, name) }
	until := time.Now().Add(48 * time.Hour)

	refused := func(what string, err error) {
		t.Helper()
		if !errors.Is(err, moderation.ErrForbidden) {
			t.Errorf("%s: err = %v, want a refusal", what, err)
		}
	}
	before := e.auditRows()

	// Nobody acts on themselves, admins included.
	e.exec(`UPDATE users SET banned = true WHERE id = $1`, modID)
	refused("moderator unbans themselves", svc.UnbanUser(ctx, actor(modID, "mod"), modID, ""))
	refused("moderator bans themselves", svc.BanUser(ctx, actor(modID, "mod"), modID, "r"))
	refused("admin suspends themselves", svc.SuspendUser(ctx, actor(adminID, "boss"), adminID, until, "r"))
	refused("admin grants themselves a role", svc.GrantRole(ctx, actor(adminID, "boss"), adminID, "moderator"))
	e.exec(`UPDATE users SET banned = false WHERE id = $1`, modID)
	e.exec(`UPDATE users SET suspended = true, suspended_until = $2 WHERE id = $1`, modID, until)
	refused("moderator lifts own suspension", svc.LiftSuspension(ctx, actor(modID, "mod"), modID, ""))
	e.exec(`UPDATE users SET suspended = false, suspended_until = NULL WHERE id = $1`, modID)

	// A moderator can't touch staff, nor change roles at all.
	for _, target := range []uuid.UUID{adminID, otherMod} {
		refused("moderator suspends staff", svc.SuspendUser(ctx, actor(modID, "mod"), target, until, "r"))
		refused("moderator bans staff", svc.BanUser(ctx, actor(modID, "mod"), target, "r"))
		refused("moderator unbans staff", svc.UnbanUser(ctx, actor(modID, "mod"), target, ""))
		refused("moderator lifts staff suspension", svc.LiftSuspension(ctx, actor(modID, "mod"), target, ""))
		refused("moderator revokes a role", svc.RevokeRole(ctx, actor(modID, "mod"), target, "moderator"))
	}
	refused("moderator grants a role", svc.GrantRole(ctx, actor(modID, "mod"), plain, "moderator"))
	if e.auditRows() != before {
		t.Fatalf("refused attempts wrote %d audit rows", e.auditRows()-before)
	}
	// Over HTTP these are 403s too.
	e.expect(403, "POST", "/api/v1/admin/users/"+adminID.String()+"/suspend", modTok, map[string]string{"until": "2099-01-01", "reason": "r"})
	e.expect(403, "POST", "/api/v1/admin/users/"+otherMod.String()+"/ban", modTok, map[string]string{"reason": "r"})
	e.expect(403, "POST", "/api/v1/admin/users/"+modID.String()+"/unban", modTok, nil)
	if e.auditRows() != before {
		t.Fatalf("refused HTTP attempts wrote %d audit rows", e.auditRows()-before)
	}

	// Moderators can still act on ordinary accounts; admins can act on staff.
	if err := svc.BanUser(ctx, actor(modID, "mod"), plain, "r"); err != nil {
		t.Errorf("moderator bans a plain user: %v", err)
	}
	e.expect(200, "POST", "/api/v1/admin/users/"+otherMod.String()+"/suspend", adminTok, map[string]string{"until": "2099-01-01", "reason": "r"})
	e.expect(200, "POST", "/api/v1/admin/users/"+otherAdmin.String()+"/ban", adminTok, map[string]string{"reason": "r"})
	e.expect(200, "POST", "/api/v1/admin/users/"+otherAdmin.String()+"/unban", adminTok, nil)
	if err := svc.GrantRole(ctx, actor(adminID, "boss"), plain, "tag_wrangler"); err != nil {
		t.Errorf("admin grants a role: %v", err)
	}
	// The command line counts as an admin.
	cli := audit.CLIActor()
	if err := svc.BanUser(ctx, cli, adminID, "r"); err != nil {
		t.Errorf("CLI bans an admin: %v", err)
	}
	if err := svc.UnbanUser(ctx, cli, adminID, ""); err != nil {
		t.Errorf("CLI unbans an admin: %v", err)
	}
	if err := svc.GrantRole(ctx, cli, plain, "moderator"); err != nil {
		t.Errorf("CLI grants a role: %v", err)
	}
	// The orphan account is still protected.
	orphan := uuid.MustParse("00000000-0000-0000-0000-000000000000")
	if err := svc.BanUser(ctx, actor(adminID, "boss"), orphan, "r"); !errors.Is(err, moderation.ErrProtectedAcc) {
		t.Errorf("ban orphan_account: %v", err)
	}
}

func TestSuspendedOrBannedStaffCannotModerate(t *testing.T) {
	e := newEnv(t)
	creator, _ := e.user("creator")
	target, _ := e.user("target")
	modID, modTok := e.user("mod", "moderator")
	adminID, adminTok := e.user("boss", "admin")
	work := e.work(creator, "W")
	w, u := work.String(), target.String()
	comment := uuid.NewString()
	paths := []string{
		"/api/v1/admin/works/" + w + "/hide", "/api/v1/admin/works/" + w + "/unhide",
		"/api/v1/admin/works/" + w + "/spam", "/api/v1/admin/works/" + w + "/unspam",
		"/api/v1/admin/works/" + w + "/chapters/" + uuid.NewString() + "/hide",
		"/api/v1/admin/series/" + uuid.NewString() + "/hide",
		"/api/v1/admin/bookmarks/" + uuid.NewString() + "/hide",
		"/api/v1/admin/comments/" + comment + "/hide", "/api/v1/admin/comments/" + comment + "/spam",
		"/api/v1/admin/comments/" + comment + "/approve", "/api/v1/admin/comments/" + comment + "/freeze",
		"/api/v1/admin/users/" + u + "/suspend", "/api/v1/admin/users/" + u + "/unsuspend",
		"/api/v1/admin/users/" + u + "/ban", "/api/v1/admin/users/" + u + "/unban",
	}
	before := e.auditRows()
	for _, state := range []string{"suspended", "banned"} {
		for _, id := range []uuid.UUID{modID, adminID} {
			e.exec(`UPDATE users SET banned = false, suspended = false, suspended_until = NULL WHERE id IN ($1, $2)`, modID, adminID)
			if state == "banned" {
				e.exec(`UPDATE users SET banned = true WHERE id = $1`, id)
			} else {
				e.exec(`UPDATE users SET suspended = true, suspended_until = now() + interval '1 day' WHERE id = $1`, id)
			}
			tok := modTok
			if id == adminID {
				tok = adminTok
			}
			for _, p := range paths {
				e.expect(403, "POST", p, tok, map[string]string{"reason": "r", "until": "2099-01-01"})
			}
			if id == adminID {
				e.expect(403, "GET", "/api/v1/admin/audit", tok, nil)
			}
		}
	}
	if e.auditRows() != before {
		t.Errorf("refused requests wrote %d audit rows", e.auditRows()-before)
	}
	// Once the suspension is over the moderator works again.
	e.exec(`UPDATE users SET banned = false, suspended = true, suspended_until = now() - interval '1 minute' WHERE id = $1`, modID)
	e.expect(200, "POST", "/api/v1/admin/works/"+w+"/hide", modTok, map[string]string{"reason": "r"})
}
