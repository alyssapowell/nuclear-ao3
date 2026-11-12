// Tag search and people (pseud) search. People search finds pseuds: a result
// is one pseud's name, description excerpt, icon and public work count, and
// says nothing about any other pseud of the same person.
import { getAuthToken } from '@/utils/auth';

export interface TagSearchHit {
  id: string;
  name: string;
  type: string;
  is_canonical: boolean;
  canonical_name?: string | null;
  use_count: number;
}

export interface TagSearchParams {
  q?: string;
  type?: string;
  canonical?: boolean;
  sort?: 'relevance' | 'uses' | 'name' | 'newest';
  page?: number;
}

export interface PseudSearchHit {
  id: string;
  name: string;
  description: string;
  icon_url: string;
  icon_alt_text: string;
  work_count: number;
}

async function get<T>(path: string): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  const token = getAuthToken();
  if (token) headers.Authorization = `Bearer ${token}`;
  const response = await fetch(path, { headers });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || `Search failed (${response.status})`);
  }
  return response.json();
}

export function searchTagsPage(params: TagSearchParams) {
  const q = new URLSearchParams();
  if (params.q) q.set('q', params.q);
  if (params.type) q.set('type', params.type);
  if (params.canonical) q.set('canonical', 'true');
  if (params.sort) q.set('sort', params.sort);
  if (params.page && params.page > 1) q.set('page', String(params.page));
  return get<{ tags: TagSearchHit[]; total: number; page: number; pages: number }>(`/api/v1/tags/search?${q.toString()}`);
}

export function searchPseuds(query: string, page = 1) {
  const q = new URLSearchParams();
  if (query) q.set('q', query);
  if (page > 1) q.set('page', String(page));
  return get<{
    pseuds: PseudSearchHit[];
    pagination: { page: number; limit: number; total: number; total_pages: number };
  }>(`/api/v1/pseuds/search?${q.toString()}`);
}
