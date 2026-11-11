// The website's side of "sign in with this archive" and the pages around it:
// connected apps, the apps a person registers, and the admin review. Every
// call sends the login as a Bearer header (a browser redirect can't).
import { getAuthToken } from '@/utils/auth';

export interface Result<T> {
  ok: boolean;
  status: number;
  data?: T;
  /** The server's error code, e.g. "invalid_client" or "app_changed". */
  code?: string;
  /** A plain sentence to show. */
  error?: string;
}

/** Where the browser goes next. Kept here so tests can watch it. */
export const nav = {
  to(url: string) {
    window.location.assign(url);
  },
};

export async function call<T>(path: string, method: 'GET' | 'POST' | 'PATCH' | 'DELETE', body?: object): Promise<Result<T>> {
  const headers: Record<string, string> = {};
  const token = getAuthToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;
  if (body) headers['Content-Type'] = 'application/json';
  let response: Response;
  try {
    response = await fetch(path, { method, headers, body: body ? JSON.stringify(body) : undefined });
  } catch {
    return { ok: false, status: 0, error: 'Could not reach the server. Please try again.' };
  }
  const data = await response.json().catch(() => undefined);
  if (!response.ok) {
    const code = typeof data?.error === 'string' ? (data.error as string) : undefined;
    // OAuth routes say error_description; the admin routes say message.
    const description =
      typeof data?.error_description === 'string'
        ? (data.error_description as string)
        : typeof data?.message === 'string'
          ? (data.message as string)
          : undefined;
    return {
      ok: false,
      status: response.status,
      code,
      error: description || (code && !/^[a-z_]+$/.test(code) ? code : undefined) || 'Something went wrong. Please try again.',
    };
  }
  return { ok: true, status: response.status, data: data as T };
}

// ---- Authorize and consent ------------------------------------------------

export interface ConsentScope {
  name: string;
  description: string;
}

export interface Pseud {
  id: string;
  name: string;
}

export interface ConsentScreen {
  consent_id: string;
  app: { name: string; website?: string; logo_url?: string };
  scopes: ConsentScope[];
  pseuds?: Pseud[];
  default_pseud_id?: string;
}

export type AuthorizeAnswer = { redirect?: string; consent?: ConsentScreen };

export function authorize(query: string): Promise<Result<AuthorizeAnswer>> {
  return call(`/api/v1/oauth/authorize?${query}`, 'GET');
}

export function answerConsent(
  consentId: string,
  answer: { approved: boolean; pseud_id?: string; share_email?: boolean },
): Promise<Result<{ redirect?: string }>> {
  return call(`/api/v1/oauth/consent/${encodeURIComponent(consentId)}`, 'POST', answer);
}

// ---- Connected apps -------------------------------------------------------

export interface ConnectedApp {
  client_id: string;
  client_name: string;
  description?: string;
  website?: string;
  logo_url?: string;
  scopes: string[];
  pseud: Pseud | null;
  share_email: boolean;
  granted_at: string | null;
  last_used: string | null;
}

export function listConnectedApps(): Promise<Result<{ applications: ConnectedApp[] }>> {
  return call('/api/v1/oauth/authorized-applications', 'GET');
}

export function disconnectApp(clientId: string): Promise<Result<{ message: string }>> {
  return call(`/api/v1/oauth/authorized-applications/${encodeURIComponent(clientId)}`, 'DELETE');
}

// ---- Your apps --------------------------------------------------------------

export type AppStatus = 'pending' | 'approved' | 'rejected' | 'suspended';

export interface DisplayDetails {
  client_name: string;
  description: string;
  website: string;
  logo_url: string;
}

export interface MyApp {
  client_id: string;
  client_name: string;
  description: string;
  website: string;
  logo_url: string;
  status: AppStatus;
  review_note: string;
  redirect_uris: string[];
  scopes: string[];
  is_public: boolean;
  created_at: string;
  reviewed_at: string | null;
  display_review_pending: boolean;
  proposed: DisplayDetails | null;
}

export interface NewApp {
  client_name: string;
  description: string;
  website: string;
  logo_url: string;
  redirect_uris: string[];
  scopes: string[];
  is_public: boolean;
  grant_types: string[];
}

export function listMyApps(): Promise<Result<{ apps: MyApp[] }>> {
  return call('/api/v1/apps', 'GET');
}

export function registerApp(app: NewApp): Promise<Result<MyApp & { client_secret?: string }>> {
  return call('/api/v1/apps', 'POST', app);
}

export function updateMyApp(
  id: string,
  changes: Partial<Pick<NewApp, 'client_name' | 'description' | 'website' | 'logo_url' | 'redirect_uris' | 'scopes'>>,
): Promise<Result<{ client_id: string; status: AppStatus; display_review_pending: boolean }>> {
  return call(`/api/v1/apps/${encodeURIComponent(id)}`, 'PATCH', changes);
}

export function resetMyAppSecret(id: string): Promise<Result<{ client_id: string; client_secret: string }>> {
  return call(`/api/v1/apps/${encodeURIComponent(id)}/reset-secret`, 'POST');
}

export function deleteMyApp(id: string): Promise<Result<{ message: string }>> {
  return call(`/api/v1/apps/${encodeURIComponent(id)}`, 'DELETE');
}

// ---- Admin review -----------------------------------------------------------

export interface ReviewApp extends MyApp {
  grant_types: string[];
  is_trusted: boolean;
  revision: number;
  owner: Pseud | null;
}

export type ReviewFilter = 'pending' | 'display' | 'approved' | 'rejected' | 'suspended';
export type ReviewAction = 'approve' | 'reject' | 'suspend' | 'reinstate' | 'approve-display' | 'reject-display';

export function listReviewApps(status: ReviewFilter): Promise<Result<{ apps: ReviewApp[] }>> {
  return call(`/api/v1/admin/apps?status=${encodeURIComponent(status)}`, 'GET');
}

export function reviewApp(id: string, action: ReviewAction, revision: number, note?: string): Promise<Result<{ status: AppStatus }>> {
  return call(`/api/v1/admin/apps/${encodeURIComponent(id)}/${action}`, 'POST', note ? { revision, note } : { revision });
}

/** The scopes an app may ask for, with a plain sentence for each. */
export const SCOPE_CHOICES: { name: string; label: string }[] = [
  { name: 'openid', label: 'Sign in (an id that is different for every app)' },
  { name: 'profile', label: 'See the one pseud (name and id) the person picks' },
  { name: 'email', label: 'See the email address, if the person agrees' },
  { name: 'read', label: 'Read their works, bookmarks and other content' },
  { name: 'write', label: 'Create and edit their works' },
  { name: 'works:manage', label: 'Fully manage their works (create, edit, delete)' },
  { name: 'comments:write', label: 'Post comments on works' },
  { name: 'bookmarks:manage', label: 'Manage their bookmarks' },
  { name: 'collections:manage', label: 'Manage collections they own' },
];
