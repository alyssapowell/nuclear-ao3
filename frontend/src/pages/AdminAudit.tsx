// Admin "Audit log": everything admins and moderators did, newest first,
// filtered by date, action and who did it. The log can only be added to.
// Admins only.
import { useCallback, useEffect, useState } from 'react';
import AdminPage from '@/components/admin/AdminPage';
import { BTN, FIELD } from '@/components/admin/ConfirmDialog';
import { AUDIT_PAGE, listAudit, type AuditEntry, type AuditQuery } from '@/lib/admin';

const ACTIONS = [
  'hide', 'unhide', 'spam', 'unspam', 'approve', 'freeze', 'unfreeze', 'suspend', 'unsuspend', 'ban', 'unban',
  'grant_role', 'revoke_role', 'revoke_sessions', 'resolve_report', 'dismiss_report',
];

const EMPTY: Omit<AuditQuery, 'offset'> = { since: '', until: '', action: '', actor: '' };

function Log() {
  const [form, setForm] = useState(EMPTY);
  const [applied, setApplied] = useState<AuditQuery>({ ...EMPTY, offset: 0 });
  const [rows, setRows] = useState<AuditEntry[] | null>(null);
  const [more, setMore] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    const r = await listAudit(applied);
    if (!r.ok) {
      setError(r.error || 'Could not load the log.');
      setRows([]);
      return;
    }
    setError('');
    setRows(r.data?.actions ?? []);
    setMore(!!r.data?.has_more);
  }, [applied]);

  useEffect(() => {
    setRows(null);
    load();
  }, [load]);

  const set = (key: keyof typeof EMPTY) => (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: e.target.value });
  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    setApplied({ ...form, actor: form.actor.trim(), action: form.action.trim(), offset: 0 });
  };

  return (
    <>
      <form onSubmit={submit} className="grid gap-3 sm:grid-cols-2 mb-4">
        <div>
          <label htmlFor="audit-since" className="block text-sm font-medium text-gray-700">From date</label>
          <input id="audit-since" type="date" value={form.since} onChange={set('since')} className={FIELD} />
        </div>
        <div>
          <label htmlFor="audit-until" className="block text-sm font-medium text-gray-700">Up to and including date</label>
          <input id="audit-until" type="date" value={form.until} onChange={set('until')} className={FIELD} />
        </div>
        <div>
          <label htmlFor="audit-action" className="block text-sm font-medium text-gray-700">Action</label>
          <input id="audit-action" type="text" list="audit-actions" value={form.action} onChange={set('action')} className={FIELD} />
          <datalist id="audit-actions">{ACTIONS.map((a) => <option key={a} value={a} />)}</datalist>
        </div>
        <div>
          <label htmlFor="audit-actor" className="block text-sm font-medium text-gray-700">Done by (email or account id)</label>
          <input id="audit-actor" type="text" value={form.actor} onChange={set('actor')} className={FIELD} />
        </div>
        <div className="sm:col-span-2">
          <button type="submit" className={`${BTN} border-transparent text-white bg-orange-700 hover:bg-orange-800`}>Apply filters</button>
        </div>
      </form>
      {error && <p role="alert" className="mb-4 text-sm text-red-600">{error}</p>}
      {rows === null && !error && <p className="text-sm text-gray-700">Loading…</p>}
      {rows && rows.length === 0 && !error && <p className="text-sm text-gray-700">Nothing matches.</p>}
      {rows && rows.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <caption className="sr-only">Admin actions, newest first</caption>
            <thead>
              <tr className="bg-gray-50">
                <th scope="col" className="p-2">When</th>
                <th scope="col" className="p-2">Done by</th>
                <th scope="col" className="p-2">Action</th>
                <th scope="col" className="p-2">On</th>
                <th scope="col" className="p-2">Reason</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id} className="border-t border-gray-200 align-top">
                  <td className="p-2 whitespace-nowrap">{new Date(r.at).toLocaleString()}</td>
                  <td className="p-2 break-all">{r.actor}</td>
                  <td className="p-2">{r.action.replace(/_/g, ' ')}</td>
                  <td className="p-2 break-all">{r.target_type} {r.target_id}</td>
                  <td className="p-2">{r.reason ?? ''}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <nav aria-label="Pages" className="mt-4 flex items-center gap-2">
        <button type="button" className={BTN + ' border-gray-300 bg-white text-gray-700'} disabled={applied.offset === 0}
          onClick={() => setApplied({ ...applied, offset: Math.max(0, applied.offset - AUDIT_PAGE) })}>Previous</button>
        <span className="text-sm text-gray-700">Page {applied.offset / AUDIT_PAGE + 1}</span>
        <button type="button" className={BTN + ' border-gray-300 bg-white text-gray-700'} disabled={!more}
          onClick={() => setApplied({ ...applied, offset: applied.offset + AUDIT_PAGE })}>Next</button>
      </nav>
    </>
  );
}

export default function AdminAuditPage() {
  return (
    <AdminPage title="Audit log" need="admin" audience="the archive's admins"
      intro="Every action by an admin or moderator, with the reason they gave. Nothing here can be edited or removed.">
      <Log />
    </AdminPage>
  );
}
