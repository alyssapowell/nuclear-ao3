// Admin "Support requests": what people sent through the Contact and Feedback
// forms. Moderators and admins read a ticket, add notes (or a reply that is
// also emailed to the sender) and close or reopen it. All text is shown as
// plain text.
import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  addNote, getTicket, listTickets, setClosed,
  type Ticket, type TicketKind, type TicketStatus,
} from '@/lib/support';

const BTN =
  'py-1 px-3 border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500 disabled:opacity-50';
const NEUTRAL = `${BTN} border-gray-300 text-gray-700 bg-white hover:bg-gray-50`;
const GOOD = `${BTN} border-transparent text-white bg-green-700 hover:bg-green-800`;
const KIND_LABEL: Record<TicketKind, string> = { support: 'Support & Feedback', abuse: 'Policy & Abuse' };

function TicketView({ id, onChanged }: { id: string; onChanged: () => void }) {
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [text, setText] = useState('');
  const [reply, setReply] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    const r = await getTicket(id);
    if (r.ok && r.data) setTicket(r.data);
    else setError(r.error || 'Could not load the request.');
  }, [id]);
  useEffect(() => { load(); }, [load]);

  const run = async (f: () => Promise<{ ok: boolean; error?: string }>) => {
    setBusy(true);
    setError('');
    const r = await f();
    setBusy(false);
    if (!r.ok) {
      setError(r.error || 'Could not save that.');
      return false;
    }
    await load();
    onChanged();
    return true;
  };

  if (!ticket) return error ? <p role="alert" className="text-sm text-red-700">{error}</p> : <p className="text-sm text-gray-700">Loading…</p>;

  return (
    <div className="mt-2 p-4 border border-gray-200 rounded-md bg-gray-50">
      <dl className="text-sm text-gray-700 space-y-1">
        <div><dt className="inline font-medium">From: </dt><dd className="inline break-all">{ticket.reporter_email}</dd></div>
        {ticket.url && <div><dt className="inline font-medium">Page: </dt><dd className="inline break-all">{ticket.url}</dd></div>}
        <div><dt className="inline font-medium">Sent: </dt><dd className="inline">{new Date(ticket.created_at).toLocaleString()}</dd></div>
      </dl>
      <p className="mt-3 whitespace-pre-wrap text-sm text-gray-900">{ticket.body}</p>

      {ticket.notes && ticket.notes.length > 0 && (
        <ul className="mt-4 space-y-2">
          {ticket.notes.map((n) => (
            <li key={n.id} className="text-sm border-l-4 border-gray-300 pl-3">
              <span className="font-medium">{n.kind === 'reply' ? 'Reply sent' : 'Note'}</span>{' '}
              <span className="text-gray-600">by {n.author}, {new Date(n.created_at).toLocaleString()}</span>
              <p className="whitespace-pre-wrap text-gray-900">{n.body}</p>
            </li>
          ))}
        </ul>
      )}

      <div className="mt-4">
        <label htmlFor={`note-${id}`} className="block text-sm font-medium text-gray-700">Add a note</label>
        <textarea id={`note-${id}`} rows={3} value={text} onChange={(e) => setText(e.target.value)} disabled={busy}
          className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm text-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500" />
        <label className="mt-1 flex items-center gap-2 text-sm text-gray-700">
          <input type="checkbox" checked={reply} onChange={(e) => setReply(e.target.checked)} disabled={busy} />
          Also email this to the sender
        </label>
      </div>
      {error && <p role="alert" className="mt-2 text-sm text-red-700">{error}</p>}
      <div className="mt-2 flex flex-wrap gap-2">
        <button type="button" className={NEUTRAL} disabled={busy || !text.trim()}
          onClick={async () => { if (await run(() => addNote(id, text.trim(), reply))) { setText(''); setReply(false); } }}>
          {reply ? 'Send reply' : 'Add note'}
        </button>
        {ticket.status === 'open' ? (
          <button type="button" className={GOOD} disabled={busy} onClick={() => run(() => setClosed(id, true))}>Close request</button>
        ) : (
          <button type="button" className={NEUTRAL} disabled={busy} onClick={() => run(() => setClosed(id, false))}>Reopen request</button>
        )}
      </div>
    </div>
  );
}

export default function AdminSupportPage() {
  const [status, setStatus] = useState<TicketStatus | ''>('open');
  const [kind, setKind] = useState<TicketKind | ''>('');
  const [tickets, setTickets] = useState<Ticket[] | null>(null);
  const [error, setError] = useState('');
  const [forbidden, setForbidden] = useState(false);
  const [openId, setOpenId] = useState('');

  const load = useCallback(async () => {
    const r = await listTickets(status, kind);
    if (!r.ok) {
      if (r.status === 401 || r.status === 403) setForbidden(true);
      else setError(r.error || 'Could not load the requests.');
      setTickets([]);
      return;
    }
    setError('');
    setTickets(r.data?.tickets ?? []);
  }, [status, kind]);
  useEffect(() => { load(); }, [load]);

  if (forbidden) {
    return (
      <div className="max-w-2xl mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
        <h1 className="text-2xl font-bold text-gray-900 mb-2">Support requests</h1>
        <p role="alert" className="text-sm text-red-700">This page is for the archive&apos;s admins and moderators.</p>
      </div>
    );
  }

  const select = 'mt-1 block px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500';
  return (
    <div className="max-w-3xl mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
      <h1 className="text-2xl font-bold text-gray-900 mb-2">Support requests</h1>
      <p className="mb-4 text-sm text-gray-700">
        Messages sent from the Contact and Feedback pages. Only admins and moderators can see them. Every note, reply and
        change is recorded in the audit log. See also <Link to="/admin/apps" className="underline text-orange-800">App review</Link>.
      </p>
      <div className="mb-4 flex flex-wrap gap-4">
        <div>
          <label htmlFor="status" className="block text-sm font-medium text-gray-700">Status</label>
          <select id="status" value={status} onChange={(e) => setStatus(e.target.value as TicketStatus | '')} className={select}>
            <option value="open">Open</option>
            <option value="closed">Closed</option>
            <option value="">All</option>
          </select>
        </div>
        <div>
          <label htmlFor="kind" className="block text-sm font-medium text-gray-700">Kind</label>
          <select id="kind" value={kind} onChange={(e) => setKind(e.target.value as TicketKind | '')} className={select}>
            <option value="">All</option>
            <option value="support">Support & Feedback</option>
            <option value="abuse">Policy & Abuse</option>
          </select>
        </div>
      </div>
      {error && <p role="alert" className="mb-4 text-red-600 text-sm">{error}</p>}
      {tickets === null && !error && <p className="text-sm text-gray-700">Loading…</p>}
      {tickets && tickets.length === 0 && !error && <p className="text-sm text-gray-700">Nothing here.</p>}
      {tickets && tickets.length > 0 && (
        <ul className="divide-y divide-gray-200">
          {tickets.map((t) => (
            <li key={t.id} className="py-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <h2 className="font-medium text-gray-900">{t.subject}</h2>
                  <p className="text-xs text-gray-600">
                    {KIND_LABEL[t.kind]} · {t.status} · {new Date(t.created_at).toLocaleString()}
                  </p>
                </div>
                <button type="button" className={NEUTRAL} aria-expanded={openId === t.id} aria-label={`${openId === t.id ? 'Hide' : 'View'} ${t.subject}`}
                  onClick={() => setOpenId(openId === t.id ? '' : t.id)}>
                  {openId === t.id ? 'Hide' : 'View'}
                </button>
              </div>
              {openId === t.id && <TicketView id={t.id} onChanged={load} />}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
