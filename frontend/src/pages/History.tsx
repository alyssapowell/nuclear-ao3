// /my/history: the works the person has read, newest first, and with
// ?show=to-read the ones marked for later. Only the person sees it.
import { useCallback, useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useAuth } from '@/utils/auth';
import {
  clearMyHistory, deleteHistoryEntry, getMyHistory, setHistoryEnabled, setMarkedForLater,
  type HistoryReading, type PseudListPagination,
} from '@/lib/api';
import { getWorkUrl } from '@/lib/urls';
import { SafeHtml } from '@/lib/safeHtml';
import { PseudList } from '@/components/PseudLink';

export default function History() {
  const { token } = useAuth();
  const [params, setParams] = useSearchParams();
  const toRead = params.get('show') === 'to-read';
  const page = Math.max(1, parseInt(params.get('page') || '1', 10) || 1);

  const [readings, setReadings] = useState<HistoryReading[] | null>(null);
  const [pagination, setPagination] = useState<PseudListPagination | null>(null);
  const [enabled, setEnabled] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    try {
      const r = await getMyHistory({ show: toRead ? 'to-read' : undefined, page }, token || undefined);
      setReadings(r.readings);
      setPagination(r.pagination);
      setEnabled(r.history_enabled);
      setError('');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load your history');
    }
  }, [toRead, page, token]);

  useEffect(() => { load(); }, [load]);

  const act = async (fn: () => Promise<unknown>) => {
    try {
      await fn();
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'That did not work');
    }
  };

  const clearAll = () => {
    if (!confirm('Clear your whole reading history, including works marked for later? This cannot be undone.')) return;
    act(() => clearMyHistory(token || undefined));
  };

  const totalPages = pagination?.total_pages ?? 1;
  const go = (p: number) => {
    const next = new URLSearchParams(params);
    if (p > 1) next.set('page', String(p)); else next.delete('page');
    setParams(next);
  };

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <h1 className="text-2xl font-bold mb-4">{toRead ? 'Marked for Later' : 'History'}</h1>

      <nav aria-label="History views" className="flex gap-4 mb-4 text-sm">
        {toRead
          ? <Link to="/my/history" className="text-blue-700 hover:underline">Full History</Link>
          : <Link to="/my/history?show=to-read" className="text-blue-700 hover:underline">Marked for Later</Link>}
      </nav>

      <section aria-label="History settings" className="mb-6 border border-slate-200 rounded-lg p-3 text-sm">
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => act(() => setHistoryEnabled(e.target.checked, token || undefined))}
          />
          Keep my reading history
        </label>
        {!enabled && <p className="mt-1 text-slate-600">Nothing you read is being recorded, and Mark for Later is off.</p>}
        <button type="button" onClick={clearAll} className="mt-2 text-red-700 hover:underline">
          Clear History
        </button>
      </section>

      {error && <div role="alert" className="mb-4 text-red-700">{error}</div>}
      {readings === null && !error && <p>Loading...</p>}
      {readings !== null && readings.length === 0 && (
        <p>{toRead ? 'No works are marked for later.' : 'Your history is empty.'}</p>
      )}

      <ol className="space-y-4">
        {(readings ?? []).map((r) => (
          <li key={r.work_id}>
            <article className="bg-white border border-slate-200 rounded-lg p-4">
              {r.available && r.work ? (
                <>
                  <h2 className="text-lg font-semibold">
                    <Link to={getWorkUrl(r.work_id)} className="text-blue-700 hover:underline">{r.work.title}</Link>
                  </h2>
                  {r.work.authors && r.work.authors.length > 0 && (
                    <p className="text-sm text-slate-700">by <PseudList pseuds={r.work.authors} /></p>
                  )}
                  {r.work.fandoms && r.work.fandoms.length > 0 && (
                    <p className="text-sm text-slate-600">{r.work.fandoms.join(' • ')}</p>
                  )}
                  {r.work.summary && <SafeHtml html={r.work.summary} className="mt-2 text-sm text-slate-800" />}
                </>
              ) : (
                <p className="text-slate-700">This work is no longer available</p>
              )}
              <p className="text-xs text-slate-600 mt-2">
                Last visited: {new Date(r.last_visited).toLocaleDateString()}
                {' · '}Visited {r.visit_count} time{r.visit_count === 1 ? '' : 's'}
                {r.marked_for_later && <strong> (Marked for Later)</strong>}
              </p>
              <div className="mt-2 flex gap-4 text-sm">
                {r.available && enabled && (
                  <button
                    type="button"
                    className="text-blue-700 hover:underline"
                    onClick={() => act(() => setMarkedForLater(r.work_id, !r.marked_for_later, token || undefined))}
                  >
                    {r.marked_for_later ? 'Mark as Read' : 'Mark for Later'}
                  </button>
                )}
                <button
                  type="button"
                  className="text-red-700 hover:underline"
                  onClick={() => act(() => deleteHistoryEntry(r.work_id, token || undefined))}
                >
                  Delete from History
                </button>
              </div>
            </article>
          </li>
        ))}
      </ol>

      {totalPages > 1 && (
        <nav aria-label="Pages" className="mt-6 flex gap-4 items-center">
          <button type="button" disabled={page <= 1} onClick={() => go(page - 1)}>Previous</button>
          <span>Page {page} of {totalPages}</span>
          <button type="button" disabled={page >= totalPages} onClick={() => go(page + 1)}>Next</button>
        </nav>
      )}
    </div>
  );
}
