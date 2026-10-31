package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"nuclear-ao3/internal/audit"
	"nuclear-ao3/internal/moderation"
	"nuclear-ao3/migrations"
)

// moderationCommand runs the moderation subcommands and reports whether args
// named one. Each command does what the matching admin endpoint does, on the
// server, as the operating system user who ran it ("cli:<user>" in the audit
// log). They are for incidents, when the site itself is not the right tool.
func moderationCommand(ctx context.Context, db *sql.DB, args []string) (handled bool, code int) {
	cmds := map[string]func(context.Context, *moderation.Service, []string) int{
		"hide":      func(ctx context.Context, s *moderation.Service, a []string) int { return hideCmd(ctx, s, a, true) },
		"unhide":    func(ctx context.Context, s *moderation.Service, a []string) int { return hideCmd(ctx, s, a, false) },
		"suspend":   suspendCmd,
		"unsuspend": unsuspendCmd,
		"ban":       banCmd,
		"unban":     unbanCmd,
	}
	if args[0] == "roles" && len(args) == 1 {
		return true, rolesCmd(ctx, moderation.New(db))
	}
	if args[0] == "audit" {
		return true, auditCmd(ctx, db, args[1:])
	}
	cmd, ok := cmds[args[0]]
	if !ok {
		return false, 0
	}
	// The migrations may not have run yet on a server that was just updated.
	if _, err := migrations.Up(ctx, db); err != nil {
		log.Printf("%s: %v", args[0], err)
		return true, 1
	}
	return true, cmd(ctx, moderation.New(db), args[1:])
}

// parseFlags splits args into plain arguments and --name value (or
// --name=value) options, in any order.
func parseFlags(args []string, names ...string) (plain []string, opts map[string]string, err error) {
	opts = map[string]string{}
	known := map[string]bool{}
	for _, n := range names {
		known[n] = true
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			plain = append(plain, a)
			continue
		}
		name, val, hasVal := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		if !known[name] {
			return nil, nil, fmt.Errorf("unknown option --%s", name)
		}
		if !hasVal {
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("--%s needs a value", name)
			}
			i++
			val = args[i]
		}
		opts[name] = val
	}
	return plain, opts, nil
}

func usage(format string, a ...any) int {
	log.Printf("usage: "+format, a...)
	return 2
}

func report(what string, err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, moderation.ErrNoChange):
		log.Printf("%s: nothing to do, it is already in that state", what)
		return 0
	default:
		log.Printf("%s: %v", what, err)
		return 1
	}
}

// hideCmd is "nuclear hide|unhide work|chapter|series|bookmark|comment <id>".
func hideCmd(ctx context.Context, s *moderation.Service, args []string, hide bool) int {
	verb := "unhide"
	if hide {
		verb = "hide"
	}
	plain, opts, err := parseFlags(args, "reason")
	if err != nil || len(plain) != 2 {
		return usage(`nuclear %s work|chapter|series|bookmark|comment <id> [--reason "..."]`, verb)
	}
	kind := moderation.Kind(plain[0])
	switch kind {
	case moderation.Work, moderation.Chapter, moderation.Series, moderation.Bookmark, moderation.Comment:
	default:
		return usage(`nuclear %s work|chapter|series|bookmark|comment <id> [--reason "..."]`, verb)
	}
	id, err := uuid.Parse(plain[1])
	if err != nil {
		log.Printf("%s: %q is not an id (they look like 3f2b8c1e-...)", verb, plain[1])
		return 2
	}
	if err := s.SetHidden(ctx, audit.CLIActor(), kind, id, hide, opts["reason"]); err != nil {
		return report(verb, err)
	}
	log.Printf("%s: %s %s is now %s", verb, kind, id, map[bool]string{true: "hidden", false: "visible again"}[hide])
	return 0
}

// suspendCmd is "nuclear suspend <username> --until 2026-12-01 --reason ...".
func suspendCmd(ctx context.Context, s *moderation.Service, args []string) int {
	plain, opts, err := parseFlags(args, "until", "reason")
	if err != nil || len(plain) != 1 || opts["until"] == "" {
		return usage(`nuclear suspend <username> --until 2026-12-01 --reason "..."`)
	}
	until, err := moderation.ParseUntil(opts["until"])
	if err != nil {
		log.Printf("suspend: %v", err)
		return 2
	}
	id, err := s.UserByName(ctx, plain[0])
	if err != nil {
		log.Printf("suspend: %v", err)
		return 1
	}
	if err := s.SuspendUser(ctx, audit.CLIActor(), id, until, opts["reason"]); err != nil {
		return report("suspend", err)
	}
	log.Printf("suspend: %s is suspended until %s and has been logged out", plain[0], until.UTC().Format(time.RFC3339))
	return 0
}

func unsuspendCmd(ctx context.Context, s *moderation.Service, args []string) int {
	plain, opts, err := parseFlags(args, "reason")
	if err != nil || len(plain) != 1 {
		return usage(`nuclear unsuspend <username> [--reason "..."]`)
	}
	id, err := s.UserByName(ctx, plain[0])
	if err != nil {
		log.Printf("unsuspend: %v", err)
		return 1
	}
	if err := s.LiftSuspension(ctx, audit.CLIActor(), id, opts["reason"]); err != nil {
		return report("unsuspend", err)
	}
	log.Printf("unsuspend: %s can post again", plain[0])
	return 0
}

func banCmd(ctx context.Context, s *moderation.Service, args []string) int {
	plain, opts, err := parseFlags(args, "reason")
	if err != nil || len(plain) != 1 {
		return usage(`nuclear ban <username> --reason "..."`)
	}
	id, err := s.UserByName(ctx, plain[0])
	if err != nil {
		log.Printf("ban: %v", err)
		return 1
	}
	if err := s.BanUser(ctx, audit.CLIActor(), id, opts["reason"]); err != nil {
		return report("ban", err)
	}
	log.Printf("ban: %s is banned and has been logged out", plain[0])
	return 0
}

func unbanCmd(ctx context.Context, s *moderation.Service, args []string) int {
	plain, opts, err := parseFlags(args, "reason")
	if err != nil || len(plain) != 1 {
		return usage(`nuclear unban <username> [--reason "..."]`)
	}
	id, err := s.UserByName(ctx, plain[0])
	if err != nil {
		log.Printf("unban: %v", err)
		return 1
	}
	if err := s.UnbanUser(ctx, audit.CLIActor(), id, opts["reason"]); err != nil {
		return report("unban", err)
	}
	log.Printf("unban: %s is no longer banned", plain[0])
	return 0
}

// rolesCmd is "nuclear roles": who holds a role other than the plain user.
func rolesCmd(ctx context.Context, s *moderation.Service) int {
	holders, err := s.RoleHolders(ctx)
	if err != nil {
		log.Printf("roles: %v", err)
		return 1
	}
	if len(holders) == 0 {
		fmt.Println("nobody holds a role; give the first admin with: nuclear grant <username> admin")
		return 0
	}
	fmt.Printf("%-24s %-14s %s\n", "USERNAME", "ROLE", "GRANTED")
	for _, h := range holders {
		fmt.Printf("%-24s %-14s %s\n", h.Username, h.Role, h.GrantedAt.UTC().Format("2006-01-02 15:04 UTC"))
	}
	return 0
}

// auditCmd is "nuclear audit [--since 2026-10-01] [--limit 50]".
func auditCmd(ctx context.Context, db *sql.DB, args []string) int {
	plain, opts, err := parseFlags(args, "since", "limit")
	if err != nil || len(plain) != 0 {
		return usage("nuclear audit [--since 2026-10-01] [--limit 50]")
	}
	var since time.Time
	if v := opts["since"]; v != "" {
		if since, err = moderation.ParseSince(v); err != nil {
			log.Printf("audit: %v", err)
			return 2
		}
	}
	limit := 50
	if v := opts["limit"]; v != "" {
		if limit, err = strconv.Atoi(v); err != nil || limit < 1 {
			log.Printf("audit: --limit must be a number of at least 1")
			return 2
		}
	}
	entries, err := audit.List(ctx, db, since, limit)
	if err != nil {
		log.Printf("audit: %v", err)
		return 1
	}
	if len(entries) == 0 {
		fmt.Println("no admin actions recorded")
		return 0
	}
	fmt.Printf("%-20s %-20s %-14s %-9s %-36s %s\n", "WHEN (UTC)", "WHO", "ACTION", "TARGET", "ID", "REASON")
	for _, e := range entries {
		fmt.Printf("%-20s %-20s %-14s %-9s %-36s %s\n", e.At.UTC().Format("2006-01-02 15:04:05"),
			e.Actor, e.Action, e.TargetType, e.TargetID, e.Reason)
	}
	return 0
}
