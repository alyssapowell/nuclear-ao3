// Admin "Reports": abuse reports people sent about works and comments. Each
// has a link to the content, the same hide and spam actions as the Content
// screen, and a way to close the report as resolved or dismissed.
import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import AdminPage from '@/components/admin/AdminPage';
import ContentActions from '@/components/admin/ContentActions';
import ConfirmDialog, { BTN, FIELD, NEUTRAL } from '@/components/admin/ConfirmDialog';
import { closeReport, listReports, type AbuseReport } from '@/lib/admin';

const STATUSES = [
  { value: 'pending', label: 'Waiting' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'dismissed', label: 'Dismissed' },
  { value: 'escalated', label: 'Escalated' },
  { value: 'all', label: 'All' },
];
const KINDS = [
  { value: '', label: 'Works and comments' },
  { value: 'work', label: 'Works' },
  { value: 'comment', label: 'Comments' },
];
const PAGE = 25;

function contentLink(r: AbuseReport): string | null {
  if (!r.work_id) return null;
  return r.kind === 'work' ? `/works/${r.work_id}` : `/works/${r.work_id}#comment-${r.target_id}`;
}

function ReportCard({ report, onClosed }: { report: AbuseReport; onClosed: () => void }) {
  const [closing, setClosing] = useState<'resolve' | 'dismiss' | null>(null);
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const link = contentLink(report);
  const name = `${report.kind} ${report.target_id}`;

  const close = async () => {
    if (!closing) return;
    setBusy(true);
    const r = await closeReport(report, closing, note.trim() || undefined);
    setBusy(false);
    if (r.ok) {
      setClosing(null);
      onClosed();
    } else {
      setError(r.error || 'Could not close that report.');
    }
  };

  return (
    <li className="py-4">
      <h3 className="font-medium text-gray-900">
        {report.kind === 'work' ? 'Work' : 'Comment'}: {report.reason.replace(/_/g, ' ')}
        <span className="ml-2 uppercase text-xs text-orange-800">{report.status}</span>
        {report.target_hidden && <span className="ml-2 uppercase text-xs text-gray-700">hidden</span>}
      </h3>
      <dl className="text-sm text-gray-700 mt-1 space-y-1">
        <div><dt className="inline font-medium">Reported: </dt><dd className="inline">{new Date(report.created_at).toLocaleString()}</dd></div>
        {report.target_title && (
          <div><dt className="inline font-medium">{report.kind === 'work' ? 'Title' : 'Starts with'}: </dt><dd className="inline">{report.target_title}</dd></div>
        )}
        {report.description && <div><dt className="inline font-medium">What they said: </dt><dd className="inline">{report.description}</dd></div>}
        {report.resolution && <div><dt className="inline font-medium">Note: </dt><dd className="inline">{report.resolution}</dd></div>}
      </dl>
      {link && <p className="mt-1 text-sm"><Link to={link} className="text-orange-800 underline">{report.kind === 'work' ? 'Open the work' : 'Open the comment’s work'}</Link></p>}

      <div className="mt-3">
        <ContentActions kind={report.kind} id={report.target_id} name={name} />
      </div>
      {report.status === 'pending' && (
        <div className="mt-2 flex gap-2">
          <button type="button" className={NEUTRAL} aria-label={`Resolve report about ${name}`} onClick={() => { setNote(''); setError(''); setClosing('resolve'); }}>Resolve…</button>
          <button type="button" className={NEUTRAL} aria-label={`Dismiss report about ${name}`} onClick={() => { setNote(''); setError(''); setClosing('dismiss'); }}>Dismiss…</button>
        </div>
      )}
      {closing && (
        <ConfirmDialog title={closing === 'resolve' ? 'Mark this report resolved?' : 'Dismiss this report?'}
          confirmLabel={closing === 'resolve' ? 'Resolve' : 'Dismiss'} busy={busy} error={error}
          onConfirm={close} onCancel={() => setClosing(null)}>
          <p>The report leaves the waiting list. This does not hide or change the content.</p>
          <div>
            <label htmlFor={`note-${report.id}`} className="block font-medium text-gray-700">Note (optional)</label>
            <textarea id={`note-${report.id}`} rows={3} value={note} onChange={(e) => setNote(e.target.value)} className={FIELD} />
          </div>
        </ConfirmDialog>
      )}
    </li>
  );
}

function Queue() {
  const [status, setStatus] = useState('pending');
  const [kind, setKind] = useState('');
  const [offset, setOffset] = useState(0);
  const [reports, setReports] = useState<AbuseReport[] | null>(null);
  const [more, setMore] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    const r = await listReports({ status, kind, offset });
    if (!r.ok) {
      setError(r.error || 'Could not load the reports.');
      setReports([]);
      return;
    }
    setError('');
    setReports(r.data?.reports ?? []);
    setMore(!!r.data?.has_more);
  }, [status, kind, offset]);

  useEffect(() => {
    setReports(null);
    load();
  }, [load]);

  const pick = (set: (v: string) => void) => (e: React.ChangeEvent<HTMLSelectElement>) => {
    set(e.target.value);
    setOffset(0);
  };
  const select = 'mt-1 block px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500';

  return (
    <>
      <div className="flex flex-wrap gap-4 mb-4">
        <div>
          <label htmlFor="report-status" className="block text-sm font-medium text-gray-700">Show</label>
          <select id="report-status" value={status} onChange={pick(setStatus)} className={select}>
            {STATUSES.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}
          </select>
        </div>
        <div>
          <label htmlFor="report-kind" className="block text-sm font-medium text-gray-700">About</label>
          <select id="report-kind" value={kind} onChange={pick(setKind)} className={select}>
            {KINDS.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}
          </select>
        </div>
      </div>
      <p className="mb-4 text-sm text-gray-700">Reports about tags are not collected yet, so none are listed.</p>
      {error && <p role="alert" className="mb-4 text-sm text-red-600">{error}</p>}
      {reports === null && !error && <p className="text-sm text-gray-700">Loading…</p>}
      {reports && reports.length === 0 && !error && <p className="text-sm text-gray-700">Nothing here.</p>}
      {reports && reports.length > 0 && (
        <ul className="divide-y divide-gray-200">
          {reports.map((r) => <ReportCard key={`${r.kind}-${r.id}-${r.status}`} report={r} onClosed={load} />)}
        </ul>
      )}
      <nav aria-label="Pages" className="mt-4 flex gap-2">
        <button type="button" className={BTN + ' border-gray-300 bg-white text-gray-700'} disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - PAGE))}>Previous</button>
        <button type="button" className={BTN + ' border-gray-300 bg-white text-gray-700'} disabled={!more} onClick={() => setOffset(offset + PAGE)}>Next</button>
      </nav>
    </>
  );
}

export default function AdminReportsPage() {
  return (
    <AdminPage title="Reports" need="moderator" audience="the archive's admins and moderators"
      intro="Reports people sent about works and comments, newest first.">
      <Queue />
    </AdminPage>
  );
}
