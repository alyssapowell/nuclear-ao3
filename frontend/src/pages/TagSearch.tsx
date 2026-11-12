// Tag search: find tags by name, type and canonical status. A result links to
// the tag's works.
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { searchTagsPage, type TagSearchHit } from '@/lib/searchApi';

const TYPES = ['fandom', 'character', 'relationship', 'freeform'];

export default function TagSearch() {
  const [q, setQ] = useState('');
  const [type, setType] = useState('');
  const [canonical, setCanonical] = useState(false);
  const [sort, setSort] = useState<'relevance' | 'uses' | 'name' | 'newest'>('relevance');
  const [tags, setTags] = useState<TagSearchHit[] | null>(null);
  const [page, setPage] = useState(1);
  const [pages, setPages] = useState(1);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const run = async (target: number) => {
    setLoading(true);
    setError('');
    try {
      const out = await searchTagsPage({ q, type, canonical, sort, page: target });
      setTags(out.tags);
      setPage(out.page);
      setPages(out.pages);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Search failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold text-gray-900 mb-2">Tag search</h1>
      <p className="mb-6 text-gray-700">
        <Link to="/search" className="text-blue-700 hover:underline">Search works</Link>
        {' · '}
        <Link to="/search/people" className="text-blue-700 hover:underline">Search people</Link>
      </p>
      <form
        className="grid grid-cols-1 md:grid-cols-2 gap-4 bg-white rounded-lg border p-4 mb-6"
        onSubmit={(e) => { e.preventDefault(); run(1); }}
      >
        <label className="block">
          <span className="text-sm font-medium text-gray-800">Tag name</span>
          <input className="mt-1 w-full border rounded px-3 py-2" value={q} onChange={(e) => setQ(e.target.value)} />
        </label>
        <label className="block">
          <span className="text-sm font-medium text-gray-800">Type</span>
          <select className="mt-1 w-full border rounded px-3 py-2" value={type} onChange={(e) => setType(e.target.value)}>
            <option value="">Any</option>
            {TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
          </select>
        </label>
        <label className="block">
          <span className="text-sm font-medium text-gray-800">Sort by</span>
          <select className="mt-1 w-full border rounded px-3 py-2" value={sort} onChange={(e) => setSort(e.target.value as typeof sort)}>
            <option value="relevance">Best match</option>
            <option value="uses">Most used</option>
            <option value="name">Name</option>
            <option value="newest">Newest</option>
          </select>
        </label>
        <label className="flex items-center gap-2 mt-6">
          <input type="checkbox" checked={canonical} onChange={(e) => setCanonical(e.target.checked)} />
          <span className="text-sm text-gray-800">Canonical tags only</span>
        </label>
        <div className="md:col-span-2">
          <button type="submit" className="px-4 py-2 bg-orange-600 text-white rounded hover:bg-orange-700">Search tags</button>
        </div>
      </form>

      {loading && <p role="status">Searching...</p>}
      {error && <p role="alert" className="text-red-700">{error}</p>}
      {tags && !loading && (tags.length === 0 ? (
        <p>No tags found.</p>
      ) : (
        <>
          <ul className="divide-y bg-white rounded-lg border">
            {tags.map((t) => (
              <li key={t.id} className="px-4 py-3 flex justify-between gap-4">
                <span>
                  <Link to={`/works?tag=${encodeURIComponent(t.name)}`} className="text-blue-700 hover:underline">{t.name}</Link>
                  <span className="ml-2 text-sm text-gray-600">{t.type}{t.is_canonical ? ', canonical' : ''}</span>
                </span>
                <span className="text-sm text-gray-700">{t.use_count} {t.use_count === 1 ? 'work' : 'works'}</span>
              </li>
            ))}
          </ul>
          {pages > 1 && (
            <nav className="mt-4 flex gap-4" aria-label="Pages">
              <button disabled={page <= 1} onClick={() => run(page - 1)} className="underline disabled:opacity-50">Previous</button>
              <span>Page {page} of {pages}</span>
              <button disabled={page >= pages} onClick={() => run(page + 1)} className="underline disabled:opacity-50">Next</button>
            </nav>
          )}
        </>
      ))}
    </div>
  );
}
