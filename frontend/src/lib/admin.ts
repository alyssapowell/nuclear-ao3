// The website's side of the admin routes: accounts, content, reports and the
// audit log. The server checks the role on every request; the website only
// uses roles to decide which links and pages to show.
import { call, type Result } from '@/lib/oauth';

export type Role = 'admin' | 'moderator' | 'tag_wrangler';
export const ASSIGNABLE_ROLES: Role[] = ['admin', 'moderator', 'tag_wrangler'];

/** An admin passes every role check; a moderator passes "moderator". */
export function hasRole(roles: string[], need: Role): boolean {
  return roles.includes('admin') || roles.includes(need);
}

export function getMyRoles(): Promise<Result<{ roles?: string[] }>> {
  return call('/api/v1/auth/me', 'GET');
}

// ---- Accounts ---------------------------------------------------------------

export interface Account {
  id: string;
  email: string;
  created_at: string;
  is_verified: boolean;
  banned: boolean;
  suspended: boolean;
  suspended_until?: string;
  roles: string[];
}

export function findAccount(query: string): Promise<Result<{ account: Account }>> {
  return call(`/api/v1/admin/accounts?q=${encodeURIComponent(query)}`, 'GET');
}

export type AccountAction = 'suspend' | 'unsuspend' | 'ban' | 'unban' | 'logout-everywhere';

export function accountAction(
  id: string,
  action: AccountAction,
  body: { reason?: string; until?: string } = {},
): Promise<Result<unknown>> {
  return call(`/api/v1/admin/users/${encodeURIComponent(id)}/${action}`, 'POST', body);
}

export function grantRole(id: string, role: Role): Promise<Result<unknown>> {
  return call(`/api/v1/auth/admin/users/${encodeURIComponent(id)}/roles`, 'POST', { role });
}

export function revokeRole(id: string, role: Role): Promise<Result<unknown>> {
  return call(`/api/v1/auth/admin/users/${encodeURIComponent(id)}/roles/${encodeURIComponent(role)}`, 'DELETE');
}

// ---- Content ----------------------------------------------------------------

export type ContentKind = 'work' | 'chapter' | 'series' | 'bookmark' | 'comment';
export const CONTENT_KINDS: { value: ContentKind; label: string }[] = [
  { value: 'work', label: 'Work' },
  { value: 'chapter', label: 'Chapter' },
  { value: 'series', label: 'Series' },
  { value: 'bookmark', label: 'Bookmark' },
  { value: 'comment', label: 'Comment' },
];

const UUID = '[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}';
const UUID_ONLY = new RegExp(`^${UUID}$`);

export function isId(s: string): boolean {
  return UUID_ONLY.test(s);
}

/**
 * Works out what an admin pasted: a bare id, or the address of a page.
 * A bare id has no type (the caller uses the one picked); an address says
 * what it is. Returns null if no id can be found.
 */
export function parseContentRef(input: string): { kind?: ContentKind; id: string } | null {
  const text = input.trim();
  if (isId(text)) return { id: text };
  const patterns: [ContentKind, RegExp][] = [
    ['comment', new RegExp(`(?:/comments/|#comment[-_:]?)(${UUID})`)],
    ['chapter', new RegExp(`/chapters/(${UUID})`)],
    ['bookmark', new RegExp(`/bookmarks/(${UUID})`)],
    ['series', new RegExp(`/series/(${UUID})`)],
    ['work', new RegExp(`/works?/(${UUID})`)],
  ];
  for (const [kind, re] of patterns) {
    const m = re.exec(text);
    if (m) return { kind, id: m[1] };
  }
  return null;
}

const PLURAL: Record<ContentKind, string> = {
  work: 'works',
  chapter: 'chapters',
  series: 'series',
  bookmark: 'bookmarks',
  comment: 'comments',
};

export type ContentAction = 'hide' | 'unhide' | 'spam' | 'unspam';

/** Spam marks exist for works and comments only. */
export function actionsFor(kind: ContentKind): ContentAction[] {
  return kind === 'work' || kind === 'comment' ? ['hide', 'unhide', 'spam', 'unspam'] : ['hide', 'unhide'];
}

export function contentAction(kind: ContentKind, id: string, action: ContentAction, reason?: string): Promise<Result<unknown>> {
  return call(`/api/v1/admin/${PLURAL[kind]}/${encodeURIComponent(id)}/${action}`, 'POST', reason ? { reason } : {});
}

// ---- Reports ----------------------------------------------------------------

export interface AbuseReport {
  kind: 'work' | 'comment';
  id: string;
  target_id: string;
  work_id?: string;
  chapter_id?: string;
  target_title?: string;
  target_hidden: boolean;
  reason: string;
  description: string;
  status: string;
  created_at: string;
  reviewed_at?: string;
  resolution?: string;
}

export function listReports(params: { status: string; kind: string; offset: number }): Promise<Result<{ reports: AbuseReport[]; has_more: boolean }>> {
  const q = new URLSearchParams({ status: params.status, limit: '25', offset: String(params.offset) });
  if (params.kind) q.set('kind', params.kind);
  return call(`/api/v1/admin/abuse-reports?${q}`, 'GET');
}

export function closeReport(report: Pick<AbuseReport, 'kind' | 'id'>, how: 'resolve' | 'dismiss', note?: string): Promise<Result<unknown>> {
  return call(`/api/v1/admin/abuse-reports/${report.kind}/${encodeURIComponent(report.id)}/${how}`, 'POST', note ? { reason: note } : {});
}

// ---- Audit log --------------------------------------------------------------

export interface AuditEntry {
  id: number;
  at: string;
  actor: string;
  action: string;
  target_type: string;
  target_id: string;
  reason?: string;
  details?: Record<string, unknown>;
}

export interface AuditQuery {
  since: string;
  until: string;
  action: string;
  actor: string;
  offset: number;
}

export const AUDIT_PAGE = 50;

export function listAudit(f: AuditQuery): Promise<Result<{ actions: AuditEntry[]; has_more: boolean }>> {
  const q = new URLSearchParams({ limit: String(AUDIT_PAGE), offset: String(f.offset) });
  if (f.since) q.set('since', f.since);
  if (f.until) q.set('until', f.until);
  if (f.action) q.set('action', f.action);
  if (f.actor) q.set('actor', f.actor);
  return call(`/api/v1/admin/audit?${q}`, 'GET');
}
