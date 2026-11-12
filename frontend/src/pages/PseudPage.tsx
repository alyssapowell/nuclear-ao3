// A pseud's public page: its name, description and icon, and what it has
// made, bookmarked and been given. The page is about one pseud only. It never
// lists or hints at any other pseud of the same person, and the address holds
// only the pseud's own id.
import { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  getPseud,
  getPseudWorks,
  getPseudSeries,
  getPseudBookmarks,
  getPseudGifts,
  type PseudPageHead,
  type PseudPageWork,
  type PseudListPagination,
  type PseudRef,
} from '@/lib/api';
import { useAuth } from '@/utils/auth';
import { getSeriesUrl, getWorkUrl } from '@/lib/urls';
import { SafeHtml } from '@/lib/safeHtml';
import { PseudList } from '@/components/PseudLink';
import { SubscriptionButton } from '@/components/SubscriptionButton';

type TabId = 'works' | 'series' | 'bookmarks' | 'gifts';

const TABS: Array<{ id: TabId; label: string; empty: string }> = [
  { id: 'works', label: 'Works', empty: 'No works yet.' },
  { id: 'series', label: 'Series', empty: 'No series yet.' },
  { id: 'bookmarks', label: 'Bookmarks', empty: 'No public bookmarks yet.' },
  { id: 'gifts', label: 'Gifts', empty: 'No gifts yet.' },
];

const PAGE_SIZE = 20;

interface SeriesItem {
  id: string;
  title: string;
  summary?: string;
  authors: PseudRef[];
  work_count: number;
  word_count?: number;
  is_complete: boolean;
}

interface BookmarkItem {
  id: string;
  notes?: string;
  tags?: string[];
  work: PseudPageWork;
}

interface GiftItem {
  work: PseudPageWork;
  created_at: string;
}

type Loaded =
  | { tab: 'works'; items: PseudPageWork[] }
  | { tab: 'series'; items: SeriesItem[] }
  | { tab: 'bookmarks'; items: BookmarkItem[] }
  | { tab: 'gifts'; items: GiftItem[] };

function WorkBlurb({ work }: { work: PseudPageWork }) {
  const tags = [...(work.fandoms ?? []), ...(work.relationships ?? [])].slice(0, 6);
  return (
    <article className="bg-white border border-slate-200 rounded-lg p-4" aria-labelledby={`work-${work.id}-title`}>
      <h3 id={`work-${work.id}-title`} className="text-lg font-semibold">
        <Link to={getWorkUrl(work.id)} className="text-blue-700 hover:text-blue-800 hover:underline">
          {work.title}
        </Link>
      </h3>
      <p className="text-sm text-slate-700">
        by <PseudList pseuds={work.authors} />
      </p>
      {tags.length > 0 && <p className="text-sm text-slate-600 mt-1">{tags.join(' • ')}</p>}
      {work.summary && <SafeHtml html={work.summary} className="mt-2 text-sm text-slate-800" />}
      <p className="text-xs text-slate-600 mt-2">
        {work.rating ? `${work.rating} · ` : ''}
        {work.word_count.toLocaleString()} words · {work.chapter_count} chapter{work.chapter_count === 1 ? '' : 's'}
        {work.is_complete ? ' · Complete' : ''}
      </p>
    </article>
  );
}

function SeriesBlurb({ series }: { series: SeriesItem }) {
  return (
    <article className="bg-white border border-slate-200 rounded-lg p-4" aria-labelledby={`series-${series.id}-title`}>
      <h3 id={`series-${series.id}-title`} className="text-lg font-semibold">
        <Link to={getSeriesUrl(series.id)} className="text-blue-700 hover:text-blue-800 hover:underline">
          {series.title}
        </Link>
      </h3>
      <p className="text-sm text-slate-700">
        by <PseudList pseuds={series.authors} />
      </p>
      {series.summary && <SafeHtml html={series.summary} className="mt-2 text-sm text-slate-800" />}
      <p className="text-xs text-slate-600 mt-2">
        {series.work_count} work{series.work_count === 1 ? '' : 's'}
        {series.word_count !== undefined ? ` · ${series.word_count.toLocaleString()} words` : ''}
        {series.is_complete ? ' · Complete' : ''}
      </p>
    </article>
  );
}

export default function PseudPage() {
  const { id = '' } = useParams<{ id: string }>();
  const { user, token } = useAuth();
  const [head, setHead] = useState<PseudPageHead | null>(null);
  const [headError, setHeadError] = useState('');
  const [tab, setTab] = useState<TabId>('works');
  const [page, setPage] = useState(1);
  const [loaded, setLoaded] = useState<Loaded | null>(null);
  const [pagination, setPagination] = useState<PseudListPagination | null>(null);
  const [listError, setListError] = useState('');
  const [loading, setLoading] = useState(false);
  const tabRefs = useRef<Record<TabId, HTMLButtonElement | null>>({ works: null, series: null, bookmarks: null, gifts: null });

  useEffect(() => {
    let cancelled = false;
    setHead(null);
    setHeadError('');
    setTab('works');
    setPage(1);
    (async () => {
      try {
        const h = await getPseud(id, token || undefined);
        if (!cancelled) setHead(h);
      } catch (e) {
        if (!cancelled) setHeadError(e instanceof Error ? e.message : 'Failed to load pseud');
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id, token]);

  const loadList = useCallback(async () => {
    setLoading(true);
    setListError('');
    try {
      const params = { page, limit: PAGE_SIZE };
      const t = token || undefined;
      if (tab === 'works') {
        const r = await getPseudWorks(id, params, t);
        setLoaded({ tab, items: r.works ?? [] });
        setPagination(r.pagination);
      } else if (tab === 'series') {
        const r = await getPseudSeries(id, params, t);
        setLoaded({ tab, items: r.series ?? [] });
        setPagination(r.pagination);
      } else if (tab === 'bookmarks') {
        const r = await getPseudBookmarks(id, params, t);
        setLoaded({ tab, items: r.bookmarks ?? [] });
        setPagination(r.pagination);
      } else {
        const r = await getPseudGifts(id, params, t);
        setLoaded({ tab, items: r.gifts ?? [] });
        setPagination(r.pagination);
      }
    } catch (e) {
      setListError(e instanceof Error ? e.message : 'Failed to load this list');
      setLoaded(null);
    } finally {
      setLoading(false);
    }
  }, [id, tab, page, token]);

  useEffect(() => {
    if (head) loadList();
  }, [head, loadList]);

  const selectTab = (next: TabId) => {
    setTab(next);
    setPage(1);
    setLoaded(null);
  };

  const onTabKeyDown = (e: React.KeyboardEvent, index: number) => {
    let next = index;
    if (e.key === 'ArrowRight') next = (index + 1) % TABS.length;
    else if (e.key === 'ArrowLeft') next = (index - 1 + TABS.length) % TABS.length;
    else if (e.key === 'Home') next = 0;
    else if (e.key === 'End') next = TABS.length - 1;
    else return;
    e.preventDefault();
    selectTab(TABS[next].id);
    tabRefs.current[TABS[next].id]?.focus();
  };

  if (headError) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-8">
        <h1 className="text-2xl font-bold text-slate-900 mb-2">Pseud not found</h1>
        <p role="alert" className="text-slate-700">{headError}</p>
      </div>
    );
  }
  if (!head) {
    return <p className="p-8 text-center text-slate-700" role="status">Loading…</p>;
  }

  const current = TABS.find((t) => t.id === tab)!;

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <header className="bg-white rounded-lg shadow-sm border border-slate-200 p-6 mb-6">
        <div className="flex items-start gap-4">
          {head.icon_url && (
            <img
              src={head.icon_url}
              alt={head.icon_alt_text}
              title={head.icon_comment_text || undefined}
              className="w-24 h-24 rounded object-cover"
            />
          )}
          <div className="flex-1">
            <h1 className="text-3xl font-bold text-slate-900">{head.name}</h1>
            {head.description && (
              <SafeHtml html={head.description} className="mt-3 text-slate-800" aria-label="About this pseud" />
            )}
          </div>
          <div>
            {head.owned ? (
              <Link to="/my/pseuds" className="text-sm text-blue-700 hover:underline">
                Manage my pseuds
              </Link>
            ) : user ? (
              <SubscriptionButton type="author" targetId={head.id} targetName={head.name} />
            ) : (
              <Link to={`/auth/login?redirect=${encodeURIComponent(`/pseuds/${head.id}`)}`} className="text-sm text-blue-700 hover:underline">
                Sign in to subscribe
              </Link>
            )}
          </div>
        </div>
      </header>

      <div role="tablist" aria-label={`${head.name}'s page`} className="flex gap-2 border-b border-slate-200 mb-4">
        {TABS.map((t, i) => (
          <button
            key={t.id}
            ref={(el) => { tabRefs.current[t.id] = el; }}
            role="tab"
            id={`pseud-tab-${t.id}`}
            aria-selected={tab === t.id}
            aria-controls="pseud-tabpanel"
            tabIndex={tab === t.id ? 0 : -1}
            onClick={() => selectTab(t.id)}
            onKeyDown={(e) => onTabKeyDown(e, i)}
            className={`px-4 py-2 -mb-px border-b-2 text-sm font-medium ${
              tab === t.id ? 'border-orange-600 text-orange-700' : 'border-transparent text-slate-700 hover:text-slate-900'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div role="tabpanel" id="pseud-tabpanel" aria-labelledby={`pseud-tab-${tab}`} aria-busy={loading}>
        <h2 className="sr-only">{current.label}</h2>
        {listError && <p role="alert" className="text-red-700 mb-4">{listError}</p>}
        {loading && !loaded && <p role="status" className="text-slate-700">Loading…</p>}
        {loaded && loaded.tab === tab && (
          loaded.items.length === 0 ? (
            <p className="text-slate-700">{current.empty}</p>
          ) : (
            <ul className="space-y-4">
              {loaded.tab === 'works' && loaded.items.map((w) => <li key={w.id}><WorkBlurb work={w} /></li>)}
              {loaded.tab === 'series' && loaded.items.map((s) => <li key={s.id}><SeriesBlurb series={s} /></li>)}
              {loaded.tab === 'bookmarks' && loaded.items.map((b) => (
                <li key={b.id}>
                  <WorkBlurb work={{ ...b.work, id: b.work.id }} />
                  {(b.notes || (b.tags && b.tags.length > 0)) && (
                    <div className="mt-1 ml-4 text-sm text-slate-700">
                      {b.notes && <SafeHtml html={b.notes} />}
                      {b.tags && b.tags.length > 0 && <p>Tags: {b.tags.join(', ')}</p>}
                    </div>
                  )}
                </li>
              ))}
              {loaded.tab === 'gifts' && loaded.items.map((g) => <li key={g.work.id}><WorkBlurb work={g.work} /></li>)}
            </ul>
          )
        )}
        {pagination && pagination.total_pages > 1 && (
          <nav aria-label="Pages" className="flex items-center justify-between mt-6">
            <button
              type="button"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="px-3 py-1 border border-slate-300 rounded disabled:opacity-50"
            >
              Previous
            </button>
            <span className="text-sm text-slate-700">
              Page {pagination.page} of {pagination.total_pages}
            </span>
            <button
              type="button"
              onClick={() => setPage((p) => Math.min(pagination.total_pages, p + 1))}
              disabled={page >= pagination.total_pages}
              className="px-3 py-1 border border-slate-300 rounded disabled:opacity-50"
            >
              Next
            </button>
          </nav>
        )}
      </div>
    </div>
  );
}
