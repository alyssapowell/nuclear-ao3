import { getAuthToken } from '@/utils/auth';
// Legacy REST API client for Nuclear AO3 (updated to use API Gateway)
// Note: Consider migrating to GraphQL client in graphql.ts for enhanced features

// API Gateway endpoint configuration - fixed for production  
const getApiGatewayUrl = () => {
  // Always use empty string (relative URLs) to avoid SSR hydration mismatches
  return '';
};

const API_GATEWAY_URL = getApiGatewayUrl();

export interface SearchParams {
  page?: number;
  limit?: number;
  sort?: 'relevance' | 'updated_at' | 'published_at' | 'title' | 'word_count' | 'hits' | 'kudos' | 'comments' | 'bookmarks';
  order?: 'asc' | 'desc';
  rating?: string[];         // "General Audiences" or "general" both work
  fandoms?: string[];        // works must have all of these
  characters?: string[];
  relationships?: string[];
  tags?: string[];           // additional (freeform) tags
  blockedTags?: string[];    // works with any of these tags are left out
  category?: string[];
  warnings?: string[];
  language?: string[];
  wordCountMin?: number;
  wordCountMax?: number;
  kudosMin?: number;
  commentsMin?: number;
  status?: string;           // 'complete', 'in-progress' or 'all'
  updatedWithin?: string;    // 'week', 'month', '3months', 'year'
  hideCrossovers?: boolean;
  creator?: string;          // a pseud name
}

// A work's creator, as the API shows it: by pseud only.
export interface WorkAuthor {
  id: string;
  name: string;
}

export interface WorkSearchResult {
  id: string;
  title: string;
  summary: string;
  authors: WorkAuthor[];     // empty for anonymous works
  anonymous: boolean;
  rating: string;
  warnings: string[];
  categories: string[];
  language: string;
  fandoms: string[];
  characters: string[];
  relationships: string[];
  freeform_tags: string[];
  word_count: number;
  chapter_count: number;
  max_chapters: number | null;
  is_complete: boolean;
  kudos_count: number;
  hit_count: number;
  comment_count: number;
  bookmark_count: number;
  published_at: string | null;
  updated_at: string;
}

export interface WorkSearchResponse {
  works: WorkSearchResult[];
  total: number;
  pagination: { page: number; limit: number; total: number; pages: number };
  facets: Record<string, { name: string; count: number }[]>;
}

// Tag suggestion and autocomplete types
export interface TagSuggestion {
  id: string;
  name: string;
  type: string;
  use_count: number;
  canonical: boolean;
}

export interface TagAutoCompleteRequest {
  query: string;
  type?: string[];
  fandom_id?: string;
  limit?: number;
  exclude_ids?: string[];
}

export interface TagAutoCompleteResponse {
  suggestions: TagSuggestion[];
}

// Tag autocomplete API
export async function getTagSuggestions(request: TagAutoCompleteRequest): Promise<TagAutoCompleteResponse> {
  try {
    const params = new URLSearchParams();
    params.append('q', request.query);
    
    if (request.type) {
      request.type.forEach(t => params.append('type', t));
    }
    if (request.fandom_id) {
      params.append('fandom_id', request.fandom_id);
    }
    if (request.limit) {
      params.append('limit', request.limit.toString());
    }
    if (request.exclude_ids) {
      request.exclude_ids.forEach(id => params.append('exclude_id', id));
    }
    
    const response = await fetch(`${API_GATEWAY_URL}/api/v1/tags/autocomplete?${params.toString()}`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
      }
    });
    
    if (!response.ok) {
      throw new Error(`Tag autocomplete API responded with status ${response.status}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Tag autocomplete API error:', error);
    throw error;
  }
}

// Logged-in readers send their token, so search includes works that are
// restricted to logged-in users.
function authHeaders(): Record<string, string> {
  const headers: Record<string, string> = { 'Accept': 'application/json' };
  const token = getAuthToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;
  return headers;
}

// Searches works (GET /api/v1/search/works). The search runs in Postgres
// and only returns works the reader may see; with no matches, `works` is
// empty. Filters left out aren't applied.
export async function searchWorks(query: string, searchParams: SearchParams = {}): Promise<WorkSearchResponse> {
  const params = new URLSearchParams();
  const add = (name: string, value: string | number | undefined) => {
    if (value !== undefined && value !== '') params.append(name, String(value));
  };
  const addAll = (name: string, values: string[] | undefined) => values?.forEach(v => add(name, v));

  add('q', query);
  add('page', searchParams.page);
  add('limit', searchParams.limit);
  add('sort', searchParams.sort);
  add('order', searchParams.order);
  addAll('rating', searchParams.rating);
  addAll('fandoms', searchParams.fandoms);
  addAll('characters', searchParams.characters);
  addAll('relationships', searchParams.relationships);
  addAll('tags', searchParams.tags);
  addAll('exclude_tags', searchParams.blockedTags);
  addAll('category', searchParams.category);
  addAll('warnings', searchParams.warnings);
  addAll('languages', searchParams.language);
  add('word_count_min', searchParams.wordCountMin);
  add('word_count_max', searchParams.wordCountMax);
  add('kudos_min', searchParams.kudosMin);
  add('comments_min', searchParams.commentsMin);
  if (searchParams.status && searchParams.status !== 'all') add('status', searchParams.status);
  add('updated_within', searchParams.updatedWithin);
  if (searchParams.hideCrossovers) add('crossovers', 'exclude');
  add('creator', searchParams.creator);

  const response = await fetch(`${API_GATEWAY_URL}/api/v1/search/works?${params.toString()}`, {
    method: 'GET',
    headers: authHeaders(),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || `Search failed (${response.status})`);
  }
  return response.json();
}

// Filters for the plain works listing (GET /api/v1/works).
export interface WorkListParams {
  q?: string;
  fandom?: string[];
  character?: string[];
  relationship?: string[];
  tag?: string[];
  rating?: string[];
  category?: string[];
  warning?: string[];
  sort?: string;
  order?: 'asc' | 'desc';
  page?: number;
  limit?: number;
}

export async function getWorks(searchParams?: WorkListParams) {
  try {
    // Build query string
    const params = new URLSearchParams();
    
    if (searchParams) {
      if (searchParams.q) params.append('q', searchParams.q);
      if (searchParams.fandom) searchParams.fandom.forEach(f => params.append('fandom', f));
      if (searchParams.character) searchParams.character.forEach(c => params.append('character', c));
      if (searchParams.relationship) searchParams.relationship.forEach(r => params.append('relationship', r));
      if (searchParams.tag) searchParams.tag.forEach(t => params.append('tag', t));
      if (searchParams.rating) searchParams.rating.forEach(r => params.append('rating', r));
      if (searchParams.category) searchParams.category.forEach(c => params.append('category', c));
      if (searchParams.warning) searchParams.warning.forEach(w => params.append('warning', w));
      if (searchParams.sort) params.append('sort', searchParams.sort);
      if (searchParams.order) params.append('order', searchParams.order);
      if (searchParams.page) params.append('page', searchParams.page.toString());
      if (searchParams.limit) params.append('limit', searchParams.limit.toString());
    }
    
    const queryString = params.toString();
    const url = `${API_GATEWAY_URL}/api/v1/works/${queryString ? '?' + queryString : ''}`;
    
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
      }
    });
    
    if (!response.ok) {
      throw new Error(`API responded with status ${response.status}: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    // Map backend response to frontend format
    return {
      ...data,
      works: data.works?.map((work: Record<string, unknown>) => ({
        ...work,
        author: work.username // Map username to author for frontend compatibility
      })) || []
    };
  } catch (error) {
    console.error('API error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

export async function getWork(id: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${id}`, {
      method: 'GET',
      headers
    });
    
    if (!response.ok) {
      throw new Error(`API responded with status ${response.status}: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    // The GetWork endpoint now returns { work: {...}, authors: [...] }
    // If it's the old format (direct work object), wrap it
    if (data.work && data.authors) {
      return data;
    } else {
      // Handle legacy format - wrap in new structure
      return {
        work: data,
        authors: []
      };
    }
  } catch (error) {
    console.error('API error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Work creation and management interfaces
export interface CreateWorkRequest {
  title: string;
  summary: string;
  notes?: string;
  language: string;
  rating: string;
  category?: string[];
  warnings?: string[];
  fandoms: string[];
  characters?: string[];
  relationships?: string[];
  freeform_tags?: string[];
  max_chapters?: number;
  chapter_title?: string;
  chapter_summary?: string;
  chapter_notes?: string;
  chapter_end_notes?: string;
  chapter_content: string;
  // Privacy settings
  restricted_to_users?: boolean;
  restricted_to_adults?: boolean;
  comment_policy?: 'open' | 'users_only' | 'disabled';
  moderate_comments?: boolean;
  disable_comments?: boolean;
  is_anonymous?: boolean;
}

export interface UpdateWorkRequest {
  title?: string;
  summary?: string;
  notes?: string;
  rating?: string;
  category?: string[];
  warnings?: string[];
  fandoms?: string[];
  characters?: string[];
  relationships?: string[];
  freeform_tags?: string[];
  max_chapters?: number;
  is_complete?: boolean;
  status?: 'draft' | 'posted' | 'hidden';
  // Privacy settings
  restricted_to_users?: boolean;
  restricted_to_adults?: boolean;
  comment_policy?: 'open' | 'users_only' | 'disabled';
  moderate_comments?: boolean;
  disable_comments?: boolean;
  is_anonymous?: boolean;
  in_anon_collection?: boolean;
  in_unrevealed_collection?: boolean;
}

// Create a new work
export async function createWork(workData: CreateWorkRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works`, {
      method: 'POST',
      headers,
      body: JSON.stringify(workData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Create work error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Update an existing work
export async function updateWork(id: string, workData: UpdateWorkRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${id}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(workData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Update work error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Delete a work
export async function deleteWork(id: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${id}`, {
      method: 'DELETE',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Delete work error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get user's works (dashboard)
export async function getMyWorks(authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/my/works`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get my works error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get work authors (for co-authorship)
export async function getWorkAuthors(id: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${id}/authors`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get work authors error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Chapter management interfaces and functions
export interface Chapter {
  id: string;
  work_id: string;
  number: number;
  title?: string;
  summary?: string;
  notes?: string;
  end_notes?: string;
  content: string;
  word_count: number;
  status: 'draft' | 'posted';
  published_at?: string;
  updated_at: string;
  created_at: string;
}

export interface CreateChapterRequest {
  title?: string;
  summary?: string;
  notes?: string;
  end_notes?: string;
  content: string;
}

export interface UpdateChapterRequest {
  title?: string;
  summary?: string;
  notes?: string;
  end_notes?: string;
  content?: string;
  status?: 'draft' | 'posted';
}

// Get work chapters
export async function getWorkChapters(workId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/chapters`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get work chapters error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get single chapter
export async function getChapter(workId: string, chapterId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/chapters/${chapterId}`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get chapter error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Create new chapter
export async function createChapter(workId: string, chapterData: CreateChapterRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/chapters`, {
      method: 'POST',
      headers,
      body: JSON.stringify(chapterData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Create chapter error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Update chapter
export async function updateChapter(workId: string, chapterId: string, chapterData: UpdateChapterRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/chapters/${chapterId}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(chapterData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Update chapter error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Delete chapter
export async function deleteChapter(workId: string, chapterId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/chapters/${chapterId}`, {
      method: 'DELETE',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Delete chapter error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Comment management interfaces and functions
export interface WorkComment {
  id: string;
  work_id: string;
  chapter_id?: string;
  user_id?: string;
  pseudonym_id?: string;
  parent_comment_id?: string;
  content: string;
  status: 'published' | 'pending' | 'rejected';
  is_anonymous: boolean;
  username?: string;
  author_name?: string;
  author_type?: 'user' | 'guest' | 'unknown';
  author_user_id?: string;
  is_deleted?: boolean;
  is_moderated?: boolean;
  is_spam?: boolean;
  kudos_count?: number;
  has_kudos?: boolean;
  created_at: string;
  updated_at: string;
  edited_at?: string;
}

export interface CreateCommentRequest {
  content: string;
  chapter_id?: string;
  parent_comment_id?: string;
  is_anonymous?: boolean;
}

// Get work comments
export interface CommentUpdateRequest {
  content: string;
}

export async function getWorkComments(workId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/comments`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get work comments error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Create comment
export async function createComment(workId: string, commentData: CreateCommentRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/comments`, {
      method: 'POST',
      headers,
      body: JSON.stringify(commentData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Create comment error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Give kudos to a comment
export async function giveCommentKudos(workId: string, commentId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/comments/${commentId}/kudos`, {
      method: 'POST',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Give comment kudos error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Remove kudos from a comment
export async function removeCommentKudos(workId: string, commentId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/comments/${commentId}/kudos`, {
      method: 'DELETE',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Remove comment kudos error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Update comment
export async function updateComment(workId: string, commentId: string, commentData: CommentUpdateRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/comments/${commentId}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(commentData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Update comment error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Delete comment
export async function deleteComment(workId: string, commentId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/comments/${commentId}`, {
      method: 'DELETE',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Delete comment error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Give kudos
export async function giveKudos(workId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/kudos`, {
      method: 'POST',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Give kudos error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get work kudos info
export async function getWorkKudos(workId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/kudos`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get kudos error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Gift management interfaces and functions
export interface Gift {
  id: string;
  work_id: string;
  pseud_id?: string;
  recipient_name: string;
  rejected: boolean;
  created_at: string;
  updated_at: string;
  recipient?: {
    pseud_id: string;
    pseud_name: string;
    username: string;
  };
}

export interface CreateGiftRequest {
  pseud_id?: string;
  recipient_name: string;
}

// Create gift for work (author-only)
export async function giftWork(workId: string, giftData: CreateGiftRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/gift`, {
      method: 'POST',
      headers,
      body: JSON.stringify(giftData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Gift work error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get work gifts
export async function getWorkGifts(workId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/gifts`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get work gifts error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Subscription management interfaces and functions
export interface Subscription {
  id: string;
  type: 'work' | 'author' | 'series' | 'tag' | 'collection';
  target_id: string;
  target_name: string;
  events: string[];
  frequency: 'immediate' | 'daily' | 'weekly' | 'never';
  filter_tags?: string[];
  filter_rating?: string[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateSubscriptionRequest {
  type: 'work' | 'author' | 'series' | 'tag' | 'collection';
  target_id: string;
  target_name?: string;
  events?: string[];
  frequency?: 'immediate' | 'daily' | 'weekly' | 'never';
  filter_tags?: string[];
  filter_rating?: string[];
}

export interface UpdateSubscriptionRequest {
  events?: string[];
  frequency?: 'immediate' | 'daily' | 'weekly' | 'never';
  filter_tags?: string[];
  filter_rating?: string[];
  is_active?: boolean;
}

// Create subscription
export async function createSubscription(subscriptionData: CreateSubscriptionRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/subscriptions`, {
      method: 'POST',
      headers,
      body: JSON.stringify(subscriptionData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Create subscription error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get user subscriptions
export async function getUserSubscriptions(authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/subscriptions`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get subscriptions error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Update subscription
export async function updateSubscription(subscriptionId: string, subscriptionData: UpdateSubscriptionRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/subscriptions/${subscriptionId}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(subscriptionData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Update subscription error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Delete subscription
export async function deleteSubscription(subscriptionId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/subscriptions/${subscriptionId}`, {
      method: 'DELETE',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Delete subscription error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Check subscription status
export async function checkSubscriptionStatus(type: string, targetId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/subscription-status?type=${encodeURIComponent(type)}&target_id=${encodeURIComponent(targetId)}`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Check subscription status error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Series management interfaces and functions
export interface Series {
  id: string;
  title: string;
  summary?: string;
  notes?: string;
  user_id: string;
  username?: string;
  is_complete: boolean;
  work_count: number;
  word_count?: number;
  created_at: string;
  updated_at: string;
}

export interface CreateSeriesRequest {
  title: string;
  summary?: string;
  notes?: string;
  is_complete?: boolean;
  work_ids?: string[];
}

export interface UpdateSeriesRequest {
  title: string;
  summary?: string;
  notes?: string;
  is_complete?: boolean;
}

// Get series
export async function getSeries(seriesId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/series/${seriesId}`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get series error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get series works
export async function getSeriesWorks(seriesId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/series/${seriesId}/works`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get series works error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Create series
export async function createSeries(seriesData: CreateSeriesRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/series`, {
      method: 'POST',
      headers,
      body: JSON.stringify(seriesData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Create series error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Update series
export async function updateSeries(seriesId: string, seriesData: UpdateSeriesRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/series/${seriesId}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(seriesData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Update series error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Delete series
export async function deleteSeries(seriesId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/series/${seriesId}`, {
      method: 'DELETE',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Delete series error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}



// Get user's series
export async function getUserSeries(userId: string, params?: { page?: number; limit?: number }, authToken?: string) {
  try {
    const queryParams = new URLSearchParams();
    
    if (params?.page) queryParams.append('page', params.page.toString());
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/users/${userId}/series?${queryParams.toString()}`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get user series error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get my series
export async function getMySeries(params?: { page?: number; limit?: number }, authToken?: string) {
  try {
    const queryParams = new URLSearchParams();
    
    if (params?.page) queryParams.append('page', params.page.toString());
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/my/series?${queryParams.toString()}`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get my series error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Add work to series
export async function addWorkToSeries(seriesId: string, workId: string, position?: number, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const body = position ? JSON.stringify({ position }) : undefined;

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/series/${seriesId}/works/${workId}`, {
      method: 'POST',
      headers,
      body,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Add work to series error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Remove work from series
export async function removeWorkFromSeries(seriesId: string, workId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/series/${seriesId}/works/${workId}`, {
      method: 'DELETE',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Remove work from series error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Collection management interfaces and functions
export interface Collection {
  id: string;
  name: string;
  title: string;
  description?: string;
  user_id: string;
  is_open: boolean;
  is_moderated: boolean;
  is_anonymous: boolean;
  is_unrevealed?: boolean;
  header_image_url?: string;
  icon_url?: string;
  work_count: number;
  created_at: string;
  updated_at: string;
}

export interface CreateCollectionRequest {
  name: string;
  title: string;
  description?: string;
  is_open?: boolean;
  is_moderated?: boolean;
  is_anonymous?: boolean;
  is_unrevealed?: boolean;
  header_image_url?: string;
  icon_url?: string;
}

// Get collection
export async function getCollection(collectionId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/collections/${collectionId}`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get collection error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Create collection
export async function createCollection(collectionData: CreateCollectionRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/collections`, {
      method: 'POST',
      headers,
      body: JSON.stringify(collectionData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Create collection error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}



// Get my collections
export async function getMyCollections(params: {
  page?: number;
  limit?: number;
}, authToken: string) {
  try {
    const queryParams = new URLSearchParams();
    if (params.page) queryParams.append('page', params.page.toString());
    if (params.limit) queryParams.append('limit', params.limit.toString());

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/my/collections?${queryParams.toString()}`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
        'Authorization': `Bearer ${authToken}`,
      },
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get my collections error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Update collection
export type UpdateCollectionRequest = Partial<CreateCollectionRequest>;

export async function updateCollection(collectionId: string, updateData: UpdateCollectionRequest, authToken: string) {
  try {
    const response = await fetch(`${API_GATEWAY_URL}/api/v1/collections/${collectionId}`, {
      method: 'PUT',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authToken}`,
      },
      body: JSON.stringify(updateData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Update collection error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Delete collection
export async function deleteCollection(collectionId: string, authToken: string) {
  try {
    const response = await fetch(`${API_GATEWAY_URL}/api/v1/collections/${collectionId}`, {
      method: 'DELETE',
      headers: {
        'Accept': 'application/json',
        'Authorization': `Bearer ${authToken}`,
      },
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Delete collection error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get collection works
export async function getCollectionWorks(collectionId: string, params: {
  page?: number;
  limit?: number;
} = {}, authToken?: string) {
  try {
    const queryParams = new URLSearchParams();
    if (params.page) queryParams.append('page', params.page.toString());
    if (params.limit) queryParams.append('limit', params.limit.toString());

    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/collections/${collectionId}/works?${queryParams.toString()}`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get collection works error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Bookmark management interfaces and functions
export interface Bookmark {
  id: string;
  work_id: string;
  notes: string;
  tags: string[];
  is_private: boolean;
  created_at: string;
  updated_at: string;
  work?: Record<string, unknown>; // Work details when fetching bookmarks
}

export interface CreateBookmarkRequest {
  notes?: string;
  tags?: string[];
  is_private?: boolean;
}

export interface UpdateBookmarkRequest {
  notes?: string;
  tags?: string[];
  is_private?: boolean;
}

export interface BookmarkSearchParams {
  q?: string;     // Search query
  tag?: string;   // Filter by bookmark tag
  page?: number;
  limit?: number;
}

// Create bookmark
export async function createBookmark(workId: string, bookmarkData: CreateBookmarkRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/bookmark`, {
      method: 'POST',
      headers,
      body: JSON.stringify(bookmarkData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Create bookmark error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Update bookmark
export async function updateBookmark(bookmarkId: string, bookmarkData: UpdateBookmarkRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/bookmarks/${bookmarkId}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(bookmarkData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Update bookmark error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Delete bookmark
export async function deleteBookmark(bookmarkId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/bookmarks/${bookmarkId}`, {
      method: 'DELETE',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Delete bookmark error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get my bookmarks
export async function getMyBookmarks(searchParams?: BookmarkSearchParams, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    // Build query string
    const params = new URLSearchParams();
    if (searchParams) {
      if (searchParams.q) params.append('q', searchParams.q);
      if (searchParams.tag) params.append('tag', searchParams.tag);
      if (searchParams.page) params.append('page', searchParams.page.toString());
      if (searchParams.limit) params.append('limit', searchParams.limit.toString());
    }

    const queryString = params.toString();
    const url = `${API_GATEWAY_URL}/api/v1/bookmarks${queryString ? '?' + queryString : ''}`;

    const response = await fetch(url, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get my bookmarks error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get user's public bookmarks
export async function getUserBookmarks(userId: string, searchParams?: BookmarkSearchParams, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    // Build query string
    const params = new URLSearchParams();
    if (searchParams) {
      if (searchParams.q) params.append('q', searchParams.q);
      if (searchParams.tag) params.append('tag', searchParams.tag);
      if (searchParams.page) params.append('page', searchParams.page.toString());
      if (searchParams.limit) params.append('limit', searchParams.limit.toString());
    }

    const queryString = params.toString();
    const url = `${API_GATEWAY_URL}/api/v1/users/${userId}/bookmarks${queryString ? '?' + queryString : ''}`;

    const response = await fetch(url, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get user bookmarks error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Check if work is bookmarked by current user
export async function isWorkBookmarked(workId: string, authToken?: string) {
  try {
    if (!authToken) {
      return false; // Not authenticated, can't have bookmarks
    }

    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Authorization': `Bearer ${authToken}`,
    };

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/works/${workId}/bookmark-status`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      return false; // Error or not bookmarked
    }
    
    const data = await response.json();
    return data.is_bookmarked === true;
  } catch (error) {
    console.error('Check bookmark status error:', error instanceof Error ? error.message : String(error));
    return false;
  }
}

// Tag suggestion functions
export async function searchTags(query: string, type?: 'fandom' | 'character' | 'relationship' | 'freeform') {
  try {
    const params = new URLSearchParams();
    params.append('q', query);
    if (type) params.append('type', type);
    params.append('limit', '10');

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/tags/search?${params.toString()}`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
    });
    
    if (!response.ok) {
      return [];
    }
    
    const data = await response.json();
    console.log('Search completed successfully', data)
    return data.tags || [];
  } catch (error) {
    console.error('Search failed:', error)
    throw error
  }
}

export async function getSearchSuggestions(query: string) {
  if (!query || query.length < 2) {
    return { authors: [], tags: [], works: [] }
  }

  try {
    const response = await fetch(`${API_GATEWAY_URL}/api/v1/search/suggestions?q=${encodeURIComponent(query)}`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      }
    })
    
    if (!response.ok) {
      throw new Error(`Suggestions failed: ${response.status}`)
    }
    
    return await response.json()
  } catch (error) {
    console.error('Suggestions failed:', error)
    return { authors: [], tags: [], works: [] }
  }
}





// Browse collections (uses work-service collections endpoint)
export async function browseCollections(params: {
  q?: string;
  page?: number;
  limit?: number;
} = {}) {
  try {
    const queryParams = new URLSearchParams();
    if (params.q) queryParams.append('q', params.q);
    if (params.page) queryParams.append('page', params.page.toString());
    if (params.limit) queryParams.append('limit', params.limit.toString());

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/collections?${queryParams.toString()}`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Browse collections error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}



export async function getPopularTags(type?: 'fandom' | 'character' | 'relationship' | 'freeform') {
  try {
    const params = new URLSearchParams();
    if (type) params.append('type', type);
    params.append('limit', '20');

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/tags/popular?${params.toString()}`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
    });
    
    if (!response.ok) {
      return [];
    }
    
    const data = await response.json();
    return data.tags || [];
  } catch (error) {
    console.error('Get popular tags error:', error instanceof Error ? error.message : String(error));
    return [];
  }
}

// User Profile Management interfaces and functions
export interface UserProfile {
  user_id: string;
  username: string;
  display_name?: string;
  bio?: string;
  profile_image_url?: string;
  header_image_url?: string;
  location?: string;
  website?: string;
  date_of_birth?: string;
  preferred_categories?: string[];
  preferred_tags?: string[];
  email_notifications?: boolean;
  show_adult_content?: boolean;
  allow_guest_downloads?: boolean;
  default_work_privacy?: 'public' | 'users_only' | 'private';
  disable_work_skins?: boolean;
  hide_warnings?: boolean;
  minimize_warnings?: boolean;
  show_stats?: boolean;
  allow_friend_requests?: boolean;
  show_bookmarks?: boolean;
  show_works?: boolean;
  created_at: string;
  updated_at: string;
}

export interface UserPseudonym {
  id: string;
  user_id: string;
  name: string;
  description?: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface UserRelationship {
  id: string;
  requester_id: string;
  requested_id: string;
  status: 'pending' | 'accepted' | 'blocked';
  type: 'friend' | 'block_user' | 'block_comments' | 'block_works';
  created_at: string;
  updated_at: string;
  requester?: UserProfile;
  requested?: UserProfile;
}

export interface UpdateProfileRequest {
  display_name?: string;
  bio?: string;
  profile_image_url?: string;
  header_image_url?: string;
  location?: string;
  website?: string;
  date_of_birth?: string;
  preferred_categories?: string[];
  preferred_tags?: string[];
  email_notifications?: boolean;
  show_adult_content?: boolean;
  allow_guest_downloads?: boolean;
  default_work_privacy?: 'public' | 'users_only' | 'private';
  disable_work_skins?: boolean;
  hide_warnings?: boolean;
  minimize_warnings?: boolean;
  show_stats?: boolean;
  allow_friend_requests?: boolean;
  show_bookmarks?: boolean;
  show_works?: boolean;
}

export interface CreatePseudonymRequest {
  name: string;
  description?: string;
  is_default?: boolean;
}

export interface FriendRequestRequest {
  user_id: string;
}

export interface BlockUserRequest {
  user_id: string;
  type: 'block_user' | 'block_comments' | 'block_works';
}

// Get user profile
export async function getUserProfile(userId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/users/${userId}/profile`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get user profile error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get current user's profile
export async function getMyProfile(authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/profile`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get my profile error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Update user profile
export async function updateProfile(profileData: UpdateProfileRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/profile`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(profileData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Update profile error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Privacy Settings Management
export interface PrivacySettingsRequest {
  // Content Filtering
  show_explicit_content?: boolean;
  show_mature_content?: boolean;
  hide_unrated_content?: boolean;
  hide_creator_chose_not_to_warn?: boolean;
  hide_archive_warnings?: string[];
  
  // Profile Privacy
  profile_visibility?: 'public' | 'logged_in_only' | 'private';
  show_stats_publicly?: boolean;
  show_email_to_users?: boolean;
  allow_user_contact?: boolean;
  
  // Reading & Interaction Privacy
  show_reading_history?: boolean;
  show_bookmarks_publicly?: boolean;
  enable_comment_notifications?: boolean;
  enable_kudos_notifications?: boolean;
  
  // Work Posting Defaults
  default_work_privacy?: 'public' | 'unrevealed' | 'anonymous';
  default_comment_policy?: 'open' | 'registered_users' | 'disabled';
  allow_concrit?: boolean;
  
  // Email & Notifications
  email_on_comments?: boolean;
  email_on_kudos?: boolean;
  email_on_bookmarks?: boolean;
  email_on_subscriptions?: boolean;
  email_digest_frequency?: 'none' | 'daily' | 'weekly';
}

export interface PrivacySettingsResponse {
  // Content Filtering
  show_explicit_content: boolean;
  show_mature_content: boolean;
  hide_unrated_content: boolean;
  hide_creator_chose_not_to_warn: boolean;
  hide_archive_warnings: string[];
  
  // Profile Privacy
  profile_visibility: 'public' | 'logged_in_only' | 'private';
  show_stats_publicly: boolean;
  show_email_to_users: boolean;
  allow_user_contact: boolean;
  
  // Reading & Interaction Privacy
  show_reading_history: boolean;
  show_bookmarks_publicly: boolean;
  enable_comment_notifications: boolean;
  enable_kudos_notifications: boolean;
  
  // Work Posting Defaults
  default_work_privacy: 'public' | 'unrevealed' | 'anonymous';
  default_comment_policy: 'open' | 'registered_users' | 'disabled';
  allow_concrit: boolean;
  
  // Email & Notifications
  email_on_comments: boolean;
  email_on_kudos: boolean;
  email_on_bookmarks: boolean;
  email_on_subscriptions: boolean;
  email_digest_frequency: 'none' | 'daily' | 'weekly';
  
  updated_at: string;
}

// Get user privacy settings
export async function getPrivacySettings(authToken?: string): Promise<PrivacySettingsResponse> {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/profile/privacy`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get privacy settings error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Update user privacy settings
export async function updatePrivacySettings(settingsData: PrivacySettingsRequest, authToken?: string): Promise<PrivacySettingsResponse> {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/profile/privacy`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(settingsData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Update privacy settings error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get user pseudonyms
export async function getPseudonyms(authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/profile/pseudonyms`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get pseudonyms error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Create pseudonym
export async function createPseudonym(pseudonymData: CreatePseudonymRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/profile/pseudonyms`, {
      method: 'POST',
      headers,
      body: JSON.stringify(pseudonymData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Create pseudonym error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Send friend request
export async function sendFriendRequest(requestData: FriendRequestRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/profile/friends/request`, {
      method: 'POST',
      headers,
      body: JSON.stringify(requestData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Send friend request error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Accept friend request
export async function acceptFriendRequest(requestId: string, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/profile/friends/${requestId}/accept`, {
      method: 'POST',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Accept friend request error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Block user
export async function blockUser(blockData: BlockUserRequest, authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/profile/block`, {
      method: 'POST',
      headers,
      body: JSON.stringify(blockData),
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Block user error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}

// Get user dashboard with statistics
export async function getUserDashboard(authToken?: string) {
  try {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    };
    
    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_GATEWAY_URL}/api/v1/profile/dashboard`, {
      method: 'GET',
      headers,
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `API responded with status ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Get user dashboard error:', error instanceof Error ? error.message : String(error));
    throw error;
  }
}
// =============================================================================
// AUTHENTICATION
// =============================================================================

export interface AuthUser {
  id: string;
  username: string;
  email: string;
}

export interface AuthResult {
  token?: string;
  user?: AuthUser;
  error?: string;
  /** The server's error code, e.g. "not_activated". */
  code?: string;
}

/** Result of any account call: ok, or an error with its code and a message to show. */
export interface AccountResult<T = undefined> {
  ok: boolean;
  data?: T;
  code?: string;
  error?: string;
}

export interface Session {
  id: string;
  created_at: string;
  last_seen: string;
  ip_address: string;
  user_agent: string;
  current: boolean;
}

export const PASSWORD_MIN = 6;
export const PASSWORD_MAX = 40;

const AUTH_ERRORS: Record<string, string> = {
  invalid_credentials: 'Incorrect username/email or password.',
  invalid_request: 'Please check the form and try again.',
  not_activated: "This account hasn't been activated yet. Check your email for the activation link.",
  user_exists: 'That username or email is already in use.',
  weak_password: `Passwords must be ${PASSWORD_MIN} to ${PASSWORD_MAX} characters long.`,
  wrong_password: 'Your current password is not correct.',
  invalid_token: 'This link is invalid or has expired.',
  not_found: 'That session no longer exists.',
};

async function accountCall<T = undefined>(
  path: string,
  method: 'GET' | 'POST' | 'DELETE',
  body?: object,
  token?: string | null,
): Promise<AccountResult<T>> {
  const headers: Record<string, string> = {};
  if (body) headers['Content-Type'] = 'application/json';
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const response = await fetch(`${API_GATEWAY_URL}/api/v1/auth/${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!response.ok) {
    const data = await response.json().catch(() => ({}));
    const code = typeof data.error === 'string' ? (data.error as string) : undefined;
    return {
      ok: false,
      code,
      error: (code && AUTH_ERRORS[code]) || data.message || 'Something went wrong. Please try again.',
    };
  }
  const data = response.status === 204 ? undefined : await response.json().catch(() => undefined);
  return { ok: true, data };
}

/** `login` is a username or an email address. */
export async function login(loginName: string, password: string): Promise<AuthResult> {
  const r = await accountCall<{ access_token?: string; user?: AuthUser }>('login', 'POST', { login: loginName, password });
  if (!r.ok) return { error: r.error, code: r.code };
  return { token: r.data?.access_token, user: r.data?.user };
}

/** On success the account still has to be activated from the email; there is no token. */
export function register(username: string, email: string, password: string): Promise<AccountResult> {
  return accountCall('register', 'POST', { username, email, password });
}

export function verifyEmail(token: string): Promise<AccountResult> {
  return accountCall('verify-email', 'POST', { token });
}

export function resendVerification(email: string): Promise<AccountResult> {
  return accountCall('resend-verification', 'POST', { email });
}

export function requestPasswordReset(email: string): Promise<AccountResult> {
  return accountCall('reset-password', 'POST', { email });
}

export function confirmPasswordReset(token: string, password: string): Promise<AccountResult> {
  return accountCall('reset-password/confirm', 'POST', { token, password });
}

export function changePassword(currentPassword: string, newPassword: string, token: string | null): Promise<AccountResult> {
  return accountCall('change-password', 'POST', { current_password: currentPassword, new_password: newPassword }, token);
}

export function getSessions(token: string | null): Promise<AccountResult<Session[]>> {
  return accountCall<Session[]>('sessions', 'GET', undefined, token);
}

export function revokeSession(id: string, token: string | null): Promise<AccountResult> {
  return accountCall(`sessions/${encodeURIComponent(id)}`, 'DELETE', undefined, token);
}
