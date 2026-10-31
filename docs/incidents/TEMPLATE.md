# Incident: (short name, such as "Site down after upgrade, 2026-11-02")

Copy this file somewhere private for each incident (see docs/INCIDENTS.md). Fill in "During" as you go and "Review" afterwards. Leave nothing blank: write "unknown" or "none" instead, so it's clear the question was asked.

## During

- **Incident lead:** (name; add a line for each handover, with the time)
- **Started:** (when it began, if known, in UTC)
- **Noticed:** (when someone first noticed, and how: alert, reader report, by chance)
- **Status updates posted:** (times and where)

### Log

Every entry gets a time (UTC), who, and what. Write what was seen as well as what was done.

| Time (UTC) | Who | What |
|---|---|---|
| | | |

### Preserved

What was saved for later, and where it is (logs, backups, disk snapshots).

## Review

Fill this in within two weeks. It's about how the system allowed this, not about who did what.

### Summary

Two or three sentences a reader could follow.

### Impact

- **Who was affected:** (all readers, logged-in users, some accounts, creators of some works)
- **For how long:** (from when to when)
- **Data:** (was any lost, changed or seen by someone who shouldn't have? which, and how much?)

### How we found out

Did an alert, a person or luck find it? How long after it started? If an alert should have caught it and didn't, that's an action below.

### What happened

The timeline from the log, cleaned up. Include the parts where people were unsure or tried something that didn't work.

### Why it was possible

Ask "why?" until the answer is something that can be changed. Write each step.

1. Why did it happen?
2. Why was that possible?
3. Why wasn't it caught sooner?
4. (keep going)

### What helped

What went well and should be kept (an alert that fired, a runbook entry that worked, a backup that restored).

### What made it worse

What slowed the response (missing access, an unclear runbook step, a backup that was too old, nobody knowing who decides).

### The guardrail question

Which automatic check, refusal or alert would have stopped this, or caught it within minutes? If none exists, building one is the first action.

### Actions

| Action | Guardrail (automatic) or instruction (manual)? | Owner | Due |
|---|---|---|---|
| | | | |

### For readers

The version to publish: what happened, what it meant for them, what's changing. Hold back details that would help someone repeat it until they're fixed.
