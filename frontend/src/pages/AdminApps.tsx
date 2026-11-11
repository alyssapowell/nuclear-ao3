// Admin "App review": the queue of apps people registered. Admins approve,
// reject or suspend an app, and approve or reject changes to how an approved
// app is shown. Each decision names the version the admin looked at, so a
// change made in between is caught instead of approved unseen.
import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { listReviewApps, reviewApp, type ReviewAction, type ReviewApp, type ReviewFilter } from '@/lib/oauth';

const BTN =
  'py-1 px-3 border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500 disabled:opacity-50';
const GOOD = `${BTN} border-transparent text-white bg-green-700 hover:bg-green-800`;
const NEUTRAL = `${BTN} border-gray-300 text-gray-700 bg-white hover:bg-gray-50`;

type Filter = 'queue' | ReviewFilter;
const FILTERS: { value: Filter; label: string }[] = [
  { value: 'queue', label: 'Waiting for review' },
  { value: 'pending', label: 'New apps only' },
  { value: 'display', label: 'Display changes only' },
  { value: 'approved', label: 'Approved' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'suspended', label: 'Suspended' },
];

function DisplayCompare({ app }: { app: ReviewApp }) {
  if (!app.display_review_pending || !app.proposed) return null;
  const rows: [string, string, string][] = [
    ['Name', app.client_name, app.proposed.client_name],
    ['Website', app.website, app.proposed.website],
    ['Logo', app.logo_url, app.proposed.logo_url],
    ['Description', app.description, app.proposed.description],
  ];
  return (
    <table className="mt-2 text-sm border border-gray-200 w-full">
      <caption className="text-left font-medium text-gray-900 py-1">
        The owner wants to change how this app is shown
      </caption>
      <thead>
        <tr className="bg-gray-50">
          <th scope="col" className="text-left p-2">Field</th>
          <th scope="col" className="text-left p-2">Now shown</th>
          <th scope="col" className="text-left p-2">Proposed</th>
        </tr>
      </thead>
      <tbody>
        {rows.map(([label, now, next]) => (
          <tr key={label} className={now === next ? '' : 'bg-yellow-50'}>
            <th scope="row" className="text-left p-2 font-normal">{label}</th>
            <td className="p-2 break-all">{now || '(none)'}</td>
            <td className="p-2 break-all">{next || '(none)'}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function AppCard({ app, onDecide }: { app: ReviewApp; onDecide: (app: ReviewApp, action: ReviewAction, note: string) => Promise<void> }) {
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const noteId = `note-${app.client_id}`;

  const decide = async (action: ReviewAction) => {
    setBusy(true);
    await onDecide(app, action, note.trim());
    setBusy(false);
  };

  return (
    <li className="py-4">
      <h3 className="font-medium text-gray-900">
        {app.client_name} <span className="ml-2 uppercase text-xs text-orange-800">{app.status}</span>
      </h3>
      <dl className="text-sm text-gray-700 mt-1 space-y-1">
        <div><dt className="inline font-medium">Registered by: </dt><dd className="inline">{app.owner ? app.owner.name : '(account is gone)'}</dd></div>
        {app.description && <div><dt className="inline font-medium">What it does: </dt><dd className="inline">{app.description}</dd></div>}
        {app.website && <div><dt className="inline font-medium">Website: </dt><dd className="inline break-all">{app.website}</dd></div>}
        <div><dt className="inline font-medium">Redirect addresses: </dt><dd className="inline break-all">{app.redirect_uris.join(' ')}</dd></div>
        <div><dt className="inline font-medium">Scopes: </dt><dd className="inline">{app.scopes.join(', ')}</dd></div>
        <div><dt className="inline font-medium">Kind: </dt><dd className="inline">{app.is_public ? 'public (no secret)' : 'confidential'}</dd></div>
        {app.review_note && <div><dt className="inline font-medium">Last note: </dt><dd className="inline">{app.review_note}</dd></div>}
      </dl>
      <DisplayCompare app={app} />

      <div className="mt-3">
        <label htmlFor={noteId} className="block text-sm font-medium text-gray-700">
          Note for the owner (needed to reject or suspend)
        </label>
        <textarea id={noteId} rows={2} value={note} onChange={(e) => setNote(e.target.value)} disabled={busy}
          className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm text-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500" />
      </div>
      <div className="mt-2 flex flex-wrap gap-2">
        {(app.status === 'pending' || app.status === 'rejected') && (
          <button type="button" className={GOOD} disabled={busy} aria-label={`Approve ${app.client_name}`} onClick={() => decide('approve')}>Approve</button>
        )}
        {app.status === 'pending' && (
          <button type="button" className={NEUTRAL} disabled={busy} aria-label={`Reject ${app.client_name}`} onClick={() => decide('reject')}>Reject</button>
        )}
        {(app.status === 'pending' || app.status === 'approved') && (
          <button type="button" className={NEUTRAL} disabled={busy} aria-label={`Suspend ${app.client_name}`} onClick={() => decide('suspend')}>Suspend</button>
        )}
        {app.status === 'suspended' && (
          <button type="button" className={GOOD} disabled={busy} aria-label={`Reinstate ${app.client_name}`} onClick={() => decide('reinstate')}>Reinstate</button>
        )}
        {app.status === 'approved' && app.display_review_pending && (
          <>
            <button type="button" className={GOOD} disabled={busy} aria-label={`Approve display changes for ${app.client_name}`} onClick={() => decide('approve-display')}>
              Approve display changes
            </button>
            <button type="button" className={NEUTRAL} disabled={busy} aria-label={`Reject display changes for ${app.client_name}`} onClick={() => decide('reject-display')}>
              Reject display changes
            </button>
          </>
        )}
      </div>
    </li>
  );
}

export default function AdminAppsPage() {
  const [filter, setFilter] = useState<Filter>('queue');
  const [apps, setApps] = useState<ReviewApp[] | null>(null);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [forbidden, setForbidden] = useState(false);

  const load = useCallback(async (which: Filter) => {
    const wanted: ReviewFilter[] = which === 'queue' ? ['pending', 'display'] : [which];
    const results = await Promise.all(wanted.map((w) => listReviewApps(w)));
    const failed = results.find((r) => !r.ok);
    if (failed) {
      if (failed.status === 401 || failed.status === 403) setForbidden(true);
      else setError(failed.error || 'Could not load the apps.');
      setApps([]);
      return;
    }
    const seen = new Set<string>();
    const merged: ReviewApp[] = [];
    for (const r of results) {
      for (const a of r.data?.apps ?? []) {
        if (!seen.has(a.client_id)) {
          seen.add(a.client_id);
          merged.push(a);
        }
      }
    }
    setApps(merged);
  }, []);

  useEffect(() => {
    setError('');
    load(filter);
  }, [filter, load]);

  const decide = async (app: ReviewApp, action: ReviewAction, note: string) => {
    setNotice('');
    const result = await reviewApp(app.client_id, action, app.revision, note || undefined);
    if (result.ok) {
      setError('');
      setNotice(`Done: ${app.client_name}.`);
    } else if (result.code === 'app_changed') {
      setError(`${app.client_name} was changed while you were looking at it. The list was reloaded: check its new settings and decide again.`);
    } else {
      setError(result.error || 'Could not save that decision.');
    }
    await load(filter);
  };

  if (forbidden) {
    return (
      <div className="max-w-2xl mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
        <h1 className="text-2xl font-bold text-gray-900 mb-2">App review</h1>
        <p role="alert" className="text-sm text-red-700">This page is for the archive&apos;s admins.</p>
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
      <h1 className="text-2xl font-bold text-gray-900 mb-2">App review</h1>
      <p className="mb-4 text-sm text-gray-700">
        Apps other people registered. An app does nothing until you approve it. Rejecting or suspending an app signs
        everyone out of it, and the owner is emailed your note. See also{' '}
        <Link to="/admin/support" className="underline text-orange-800">Support requests</Link>.
      </p>
      <div className="mb-4">
        <label htmlFor="filter" className="block text-sm font-medium text-gray-700">Show</label>
        <select id="filter" value={filter} onChange={(e) => setFilter(e.target.value as Filter)}
          className="mt-1 block px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500">
          {FILTERS.map((f) => <option key={f.value} value={f.value}>{f.label}</option>)}
        </select>
      </div>
      {error && <p role="alert" className="mb-4 text-red-600 text-sm">{error}</p>}
      {notice && <p role="status" className="mb-4 text-sm text-green-700">{notice}</p>}
      {apps === null && !error && <p className="text-sm text-gray-700">Loading…</p>}
      {apps && apps.length === 0 && !error && <p className="text-sm text-gray-700">Nothing here.</p>}
      {apps && apps.length > 0 && (
        <ul className="divide-y divide-gray-200">
          {apps.map((app) => <AppCard key={`${app.client_id}-${app.revision}`} app={app} onDecide={decide} />)}
        </ul>
      )}
    </div>
  );
}
