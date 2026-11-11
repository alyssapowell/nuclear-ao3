-- Support & Feedback and Policy & Abuse requests, sent from the Contact and
-- Feedback pages by guests and by logged-in people.
--
-- AO3 has two forms (Support and Feedback, Policy and Abuse), each of which
-- opens a ticket that its volunteers answer. 'kind' tells them apart.
--
-- Tickets are for administrators and moderators only. The person who sent one
-- gets a confirmation by email and nothing else: there is no "my tickets"
-- page and no public view.
--
-- user_id is the logged-in sender, if any. It is set to NULL when the account
-- is deleted. reporter_email is what replies are sent to: the address typed
-- by a guest, or the account's own address for a logged-in person.

CREATE TABLE support_tickets (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind           TEXT NOT NULL CHECK (kind IN ('support', 'abuse')),
    subject        TEXT NOT NULL CHECK (char_length(subject) BETWEEN 1 AND 200),
    body           TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 10000),
    user_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    reporter_email TEXT NOT NULL CHECK (char_length(reporter_email) BETWEEN 3 AND 254),
    url            TEXT CHECK (url IS NULL OR char_length(url) <= 2000),
    status         TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at      TIMESTAMPTZ,
    closed_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    CHECK ((status = 'closed') = (closed_at IS NOT NULL))
);

CREATE INDEX idx_support_tickets_status ON support_tickets (status, created_at DESC);
CREATE INDEX idx_support_tickets_kind ON support_tickets (kind, status, created_at DESC);

-- What staff wrote on a ticket. A 'note' is internal; a 'reply' was also
-- emailed to the reporter. Append-only, like admin_actions: the triggers
-- refuse UPDATE, DELETE and TRUNCATE. author_id has no foreign key for the
-- same reason as admin_actions.actor_user_id; author_label keeps who it was.
CREATE TABLE support_ticket_notes (
    id           BIGSERIAL PRIMARY KEY,
    ticket_id    UUID NOT NULL REFERENCES support_tickets(id),
    kind         TEXT NOT NULL CHECK (kind IN ('note', 'reply')),
    body         TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 10000),
    author_id    UUID,
    author_label TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_support_ticket_notes_ticket ON support_ticket_notes (ticket_id, id);

CREATE OR REPLACE FUNCTION support_ticket_notes_are_append_only()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'support_ticket_notes is append-only: % is not allowed', TG_OP
        USING ERRCODE = 'integrity_constraint_violation';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER support_ticket_notes_no_update_delete
    BEFORE UPDATE OR DELETE ON support_ticket_notes
    FOR EACH ROW EXECUTE FUNCTION support_ticket_notes_are_append_only();

CREATE TRIGGER support_ticket_notes_no_truncate
    BEFORE TRUNCATE ON support_ticket_notes
    FOR EACH STATEMENT EXECUTE FUNCTION support_ticket_notes_are_append_only();
