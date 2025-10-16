// Comment-related types for Nuclear AO3

export interface Comment {
  id: string;
  work_id: string;
  chapter_id?: string;
  user_id?: string;
  pseudonym_id?: string;
  parent_comment_id?: string;
  content: string;
  guest_name?: string;
  guest_email?: string;
  author_name: string;
  author_type: 'user' | 'guest' | 'unknown';
  is_deleted: boolean;
  is_moderated: boolean;
  is_spam: boolean;
  thread_level: number;
  kudos_count: number;
  reply_count: number;
  has_kudos?: boolean;
  created_at: string;
  updated_at: string;
  edited_at?: string;
  replies?: Comment[];
}

export interface CommentCreateRequest {
  work_id?: string;
  chapter_id?: string;
  parent_comment_id?: string;
  content: string;
  pseudonym_id?: string;
  guest_name?: string;
  guest_email?: string;
}

export interface CommentUpdateRequest {
  content: string;
}

export interface CommentResponse {
  comments: Comment[];
  total_count: number;
  page: number;
  limit: number;
  total_pages: number;
  threaded: boolean;
}

export interface CommentKudosResponse {
  message: string;
  kudos_id: string;
}