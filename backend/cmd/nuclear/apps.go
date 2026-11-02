package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"nuclear-ao3/internal/audit"
	"nuclear-ao3/internal/auth"
	"nuclear-ao3/migrations"
	"nuclear-ao3/shared/email"
)

// appsCommand runs "nuclear apps ...": the review queue for apps other
// people registered, from the server. It does what the admin routes do and is
// recorded in the audit log as cli:<your operating system user>. It is for
// incidents: an app that is abusing the archive can be cut off even if the
// site itself is not reachable. Suspending or rejecting an app ends every
// token, code and consent it holds.
//
//	nuclear apps [--status pending]    list apps (default: waiting for review;
//	                                   --status all lists every app)
//	nuclear apps approve <id> --revision N
//	nuclear apps reject <id> --note "..."
//	nuclear apps suspend <id> --note "..."
//	nuclear apps approve-display <id> --revision N
//	                                    apply the display changes (name, website,
//	                                    logo, description) an owner proposed
//	nuclear apps reject-display <id> --note "..."
//	nuclear apps --status display       list approved apps with changes waiting
//
// Approving grants something, so it needs the revision shown by the listing
// (--revision N): if the owner changed the app after you looked, nothing is
// approved and you're told to look again. Rejecting and suspending only take
// access away, so they act at once, which is what an incident needs.
func appsCommand(ctx context.Context, db *sql.DB, mail *email.Config, args []string) int {
	if _, err := migrations.Up(ctx, db); err != nil {
		log.Printf("apps: %v", err)
		return 1
	}
	if len(args) == 0 || strings.HasPrefix(args[0], "--") {
		return listAppsCmd(ctx, db, args)
	}
	action := args[0]
	if action != "approve" && action != "reject" && action != "suspend" && action != "approve-display" && action != "reject-display" {
		log.Printf("apps: unknown action %q (use approve, reject, suspend, approve-display or reject-display)", action)
		return 2
	}
	grants := action == "approve" || action == "approve-display"
	usage := map[bool]string{true: ` --revision N`, false: ` --note "..."`}[grants]
	action = strings.ReplaceAll(action, "-", "_")
	plain, opts, err := parseFlags(args[1:], "note", "revision")
	if err != nil || len(plain) != 1 {
		log.Printf("usage: nuclear apps %s <id>%s", args[0], usage)
		return 2
	}
	revision := 0
	if grants {
		revision, err = strconv.Atoi(opts["revision"])
		if err != nil || revision < 1 {
			log.Printf("usage: nuclear apps %s <id> --revision N (the revision is shown by `nuclear apps`)", args[0])
			return 2
		}
	}
	id, err := uuid.Parse(plain[0])
	if err != nil {
		log.Printf("apps: %q is not an app id", plain[0])
		return 2
	}
	res, err := auth.ReviewApp(ctx, db, mail, audit.CLIActor(), id, action, opts["note"], revision)
	switch {
	case errors.Is(err, auth.ErrAppChanged):
		log.Printf("apps %s: the app changed since revision %d; run `nuclear apps` to see its new settings", args[0], revision)
		return 1
	case errors.Is(err, auth.ErrNoteRequired):
		log.Printf("apps %s: --note \"...\" is required, and the owner is told it", action)
		return 2
	case err != nil:
		log.Printf("apps %s: %v", action, err)
		return 1
	}
	msg := fmt.Sprintf("%s: %q is now %s", res.ClientID, res.Name, res.To)
	if res.Emailed {
		msg += " (the owner has been emailed)"
	}
	fmt.Println(msg)
	return 0
}

func listAppsCmd(ctx context.Context, db *sql.DB, args []string) int {
	plain, opts, err := parseFlags(args, "status")
	if err != nil || len(plain) != 0 {
		log.Printf("usage: nuclear apps [--status pending|approved|rejected|suspended|display|all]")
		return 2
	}
	status := "pending"
	if s, ok := opts["status"]; ok {
		status = s
	}
	if status == "all" {
		status = ""
	}
	apps, total, err := auth.ListApps(ctx, db, status, 100, 0)
	if err != nil {
		log.Printf("apps: %v", err)
		return 1
	}
	if len(apps) == 0 {
		fmt.Println("no apps")
		return 0
	}
	for _, a := range apps {
		owner := "(owner's account is gone)"
		if a.OwnerPseudName != "" {
			owner = a.OwnerPseudName
		}
		fmt.Printf("%s  %-9s  revision %d  %q  by %s  registered %s\n", a.ID, a.Status, a.Revision, a.Name, owner, a.CreatedAt.Format("2006-01-02"))
		if a.DisplayPending {
			fmt.Printf("    display change waiting: name %q -> %q, website %q -> %q, logo %q -> %q, description %q -> %q\n",
				a.Name, a.ProposedName, a.Website, a.ProposedWebsite, a.LogoURL, a.ProposedLogoURL, a.Description, a.ProposedDescription)
		}
		fmt.Printf("    redirects: %s\n    scopes: %s\n", strings.Join(a.RedirectURIs, " "), strings.Join(a.Scopes, " "))
		if a.Website != "" {
			fmt.Printf("    website: %s\n", a.Website)
		}
	}
	if total > len(apps) {
		fmt.Printf("(showing %d of %d)\n", len(apps), total)
	}
	return 0
}
