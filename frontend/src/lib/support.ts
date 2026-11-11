// The Contact and Feedback forms, and the admin page that answers them.
// A ticket's text is plain text: it is only ever shown as text, never as HTML.
import { getAuthToken } from '@/utils/auth';

export type TicketKind = 'support' | 'abuse';
export type TicketStatus = 'open' | 'closed';

// The limits the server enforces, so the form can say so before sending.
export const LIMITS = { subject: 200, body: 10000, url: 2000 };

export interface Result<T> {
  ok: boolean;
  status: number;
  data?: T;
  /** A plain sentence to show. */
  error?: string;
}

async function call<T>(path: string, method: 'GET' | 'POST', body?: object): Promise<Result<T>> {
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
    const message = typeof data?.message === 'string' ? (data.message as string) : undefined;
    return { ok: false, status: response.status, error: message || 'Something went wrong. Please try again.' };
  }
  return { ok: true, status: response.status, data: data as T };
}

export interface NewTicket {
  kind: TicketKind;
  subject: string;
  body: string;
  /** Needed from guests; a logged-in person's account address is used. */
  email?: string;
  url?: string;
}

export function sendTicket(ticket: NewTicket): Promise<Result<{ received: boolean }>> {
  return call('/api/v1/support', 'POST', ticket);
}

export interface Note {
  id: number;
  kind: 'note' | 'reply';
  body: string;
  author: string;
  created_at: string;
}

export interface Ticket {
  id: string;
  kind: TicketKind;
  subject: string;
  status: TicketStatus;
  created_at: string;
  closed_at?: string;
  reporter_email: string;
  url?: string;
  body?: string;
  notes?: Note[];
}

export function listTickets(status: TicketStatus | '', kind: TicketKind | ''): Promise<Result<{ tickets: Ticket[]; total: number }>> {
  const q = new URLSearchParams();
  if (status) q.set('status', status);
  if (kind) q.set('kind', kind);
  return call(`/api/v1/admin/support?${q.toString()}`, 'GET');
}

export function getTicket(id: string): Promise<Result<Ticket>> {
  return call(`/api/v1/admin/support/${encodeURIComponent(id)}`, 'GET');
}

export function addNote(id: string, body: string, reply: boolean): Promise<Result<{ added: boolean }>> {
  return call(`/api/v1/admin/support/${encodeURIComponent(id)}/notes`, 'POST', { body, reply });
}

export function setClosed(id: string, closed: boolean, reason?: string): Promise<Result<{ status: TicketStatus }>> {
  return call(`/api/v1/admin/support/${encodeURIComponent(id)}/${closed ? 'close' : 'reopen'}`, 'POST', reason ? { reason } : undefined);
}
