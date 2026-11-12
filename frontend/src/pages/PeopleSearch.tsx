// People search (AO3: Search > People). The people are pseuds: each result is
// one pseud, linking to its own page. Nothing here groups pseuds or says
// which of them belong to the same person.
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { searchPseuds, type PseudSearchHit } from '@/lib/searchApi';
import { getPseudUrl } from '@/lib/urls';

export default function PeopleSearch() {
  const [q, setQ] = useState('');
  const [hits, setHits] = useState<PseudSearchHit[] | null>(null);
  const [page, setPage] = useState(1);
  const [pages, setPages] = useState(1);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const run = async (target: number) => {
    setLoading(true);
    setError('');
    try {
      const out = await searchPseuds(q.trim(), target);
      setHits(out.pseuds);
      setPage(out.pagination.page);
      setPages(out.pagination.total_pages);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Search failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold text-gray-900 mb-2">People search</h1>
      <p className="mb-6 text-gray-700">
        <Link to="/search" className="text-blue-700 hover:underline">Search works</Link>
        {' · '}
        <Link to="/search/tags" className="text-blue-700 hover:underline">Search tags</Link>
      </p>
      <form className="flex gap-3 mb-6" onSubmit={(e) => { e.preventDefault(); run(1); }}>
        <label className="flex-1">
          <span className="sr-only">Name or description</span>
          <input className="w-full border rounded px-3 py-2" placeholder="Name or description" value={q} onChange={(e) => setQ(e.target.value)} />
        </label>
        <button type="submit" className="px-4 py-2 bg-orange-600 text-white rounded hover:bg-orange-700">Search people</button>
      </form>

      {loading && <p role="status">Searching...</p>}
      {error && <p role="alert" className="text-red-700">{error}</p>}
      {hits && !loading && (hits.length === 0 ? (
        <p>No people found.</p>
      ) : (
        <>
          <ul className="divide-y bg-white rounded-lg border">
            {hits.map((p) => (
              <li key={p.id} className="px-4 py-3 flex gap-4">
                {p.icon_url && <img src={p.icon_url} alt={p.icon_alt_text} className="w-12 h-12 rounded object-cover" />}
                <div className="flex-1">
                  <Link to={getPseudUrl(p.id)} className="text-blue-700 hover:underline font-medium">{p.name}</Link>
                  {p.description && <p className="text-sm text-gray-700">{p.description}</p>}
                </div>
                <span className="text-sm text-gray-700">{p.work_count} {p.work_count === 1 ? 'work' : 'works'}</span>
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
