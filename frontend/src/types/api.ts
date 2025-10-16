// Nuclear AO3 API Types
export interface Work {
  id: string;
  title: string;
  summary: string;
  notes: string;
  author: string;
  language: string;
  rating: string;
  word_count: number;
  chapter_count: number;
  is_complete: boolean;
  published_at?: string;
  created_at: string;
  updated_at: string;
}

export interface WorkSearchResponse {
  works: Work[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
  };
}

export interface WorkSearchParams {
  q?: string;
  rating?: string[];
  language?: string;
  fandom?: string[];
  character?: string[];
  relationship?: string[];
  tag?: string[];
  page?: number;
  limit?: number;
  sort?: 'updated_at' | 'title' | 'word_count' | 'kudos' | 'hits';
  order?: 'asc' | 'desc';
}

export interface User {
  id: string;
  username: string;
  email: string;
  created_at: string;
  updated_at: string;
}

export interface AuthResponse {
  user: User;
  token: string;
  refreshToken: string;
}

export interface Chapter {
  id: string;
  work_id: string;
  number: number;
  title: string;
  summary: string;
  content: string;
  word_count: number;
  published_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ApiError {
  error: string;
  details?: string;
}

// API Response wrapper
export type ApiResponse<T> = T | ApiError;

export function isApiError(response: unknown): response is ApiError {
  return response !== null && typeof response === 'object' && 'error' in response && typeof (response as { error: unknown }).error === 'string';
}