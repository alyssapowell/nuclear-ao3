// Admin "Accounts": find an account by email address or id, see where it
// stands, and suspend, ban, log out or (admins only) change roles. Accounts
// have no username, so the email address and id are shown here and nowhere
// else on the site. Every action asks first and writes an audit row.
import { useState } from 'react';
import AdminPage from '@/components/admin/AdminPage';
import ConfirmDialog, { BTN, DANGER, FIELD, NEUTRAL } from '@/components/admin/ConfirmDialog';
import {
  accountAction, ASSIGNABLE_ROLES, findAccount, grantRole, hasRole, revokeRole,
  type Account, type AccountAction, type Role,
} from '@/lib/admin';
import { useRoles } from '@/hooks/useRoles';

type Pending =
  | { kind: AccountAction }
  | { kind: 'grant' | 'revoke'; role: Role };

const ROLE_LABEL: Record<Role, string> = { admin: 'admin', moderator: 'moderator', tag_wrangler: 'tag wrangler' };

function formatTime(s?: string) {
  if (!s) return '';
  const d = new Date(s);
  return Number.isNaN(d.getTime()) ? s : d.toLocaleString();
}

function tomorrow(): string {
  const d = new Date(Date.now() + 24 * 3600 * 1000);
  return d.toISOString().slice(0, 10);
}

function Status({ account }: { account: Account }) {
  const parts: string[] = [];
  if (account.banned) parts.push('Banned');
  if (account.suspended) parts.push(account.suspended_until ? `Suspended until ${formatTime(account.suspended_until)}` : 'Suspended');
  if (parts.length === 0) parts.push('Active');
  return <>{parts.join('; ')}</>;
}

function AccountView({ account, isAdmin, onChanged }: { account: Account; isAdmin: boolean; onChanged: (a: Account | null) => void }) {
  const [pending, setPending] = useState<Pending | null>(null);
  const [reason, setReason] = useState('');
  const [until, setUntil] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const open = (p: Pending) => {
    setPending(p);
    setReason('');
    setUntil('');
    setError('');
  };

  const needsReason = pending && (pending.kind === 'suspend' || pending.kind === 'ban');
  const needsDate = pending?.kind === 'suspend';
  const dateOk = !needsDate || (/^\d{4}-\d{2}-\d{2}$/.test(until) && until > new Date().toISOString().slice(0, 10));
  const canConfirm = (!needsReason || reason.trim() !== '') && dateOk;

  const run = async () => {
    if (!pending) return;
    setBusy(true);
    setError('');
    const r =
      pending.kind === 'grant' ? await grantRole(account.id, pending.role)
      : pending.kind === 'revoke' ? await revokeRole(account.id, pending.role)
      : await accountAction(account.id, pending.kind, { reason: reason.trim() || undefined, until: needsDate ? until : undefined });
    if (!r.ok) {
      setBusy(false);
      setError(r.error || 'Could not do that.');
      return;
    }
    const done = describe(pending);
    setPending(null);
    setBusy(false);
    setNotice(`Done: ${done}.`);
    const again = await findAccount(account.id);
    onChanged(again.ok && again.data ? again.data.account : null);
    // The button that opened the box may have changed or gone; land on the heading.
    setTimeout(() => document.getElementById('account-heading')?.focus(), 0);
  };

  const describe = (p: Pending) =>
    p.kind === 'grant' ? `${ROLE_LABEL[p.role]} role granted`
    : p.kind === 'revoke' ? `${ROLE_LABEL[p.role]} role revoked`
    : p.kind === 'logout-everywhere' ? 'logged out everywhere'
    : p.kind === 'suspend' ? 'suspended'
    : p.kind === 'unsuspend' ? 'suspension lifted'
    : p.kind === 'ban' ? 'banned'
    : 'ban lifted';

  const title = (p: Pending) =>
    p.kind === 'grant' ? `Give the ${ROLE_LABEL[p.role]} role?`
    : p.kind === 'revoke' ? `Take away the ${ROLE_LABEL[p.role]} role?`
    : p.kind === 'logout-everywhere' ? 'Log this account out everywhere?'
    : p.kind === 'suspend' ? 'Suspend this account?'
    : p.kind === 'unsuspend' ? 'Lift the suspension?'
    : p.kind === 'ban' ? 'Ban this account?'
    : 'Lift the ban?';

  const confirmLabel = (p: Pending) =>
    p.kind === 'grant' ? 'Give role'
    : p.kind === 'revoke' ? 'Take away role'
    : p.kind === 'logout-everywhere' ? 'Log out everywhere'
    : p.kind === 'suspend' ? 'Suspend'
    : p.kind === 'unsuspend' ? 'Lift suspension'
    : p.kind === 'ban' ? 'Ban'
    : 'Lift ban';

  const hurts = pending && ['suspend', 'ban', 'revoke', 'logout-everywhere'].includes(pending.kind);

  return (
    <section aria-labelledby="account-heading" className="mt-6 border-t border-gray-200 pt-4">
      <h2 id="account-heading" tabIndex={-1} className="text-lg font-medium text-gray-900 focus:outline-none">Account</h2>
      <dl className="mt-2 text-sm text-gray-700 space-y-1">
        <div><dt className="inline font-medium">Email: </dt><dd className="inline break-all">{account.email}</dd></div>
        <div><dt className="inline font-medium">Id: </dt><dd className="inline break-all">{account.id}</dd></div>
        <div><dt className="inline font-medium">Created: </dt><dd className="inline">{formatTime(account.created_at)}</dd></div>
        <div><dt className="inline font-medium">Email confirmed: </dt><dd className="inline">{account.is_verified ? 'yes' : 'no'}</dd></div>
        <div><dt className="inline font-medium">Status: </dt><dd className="inline"><Status account={account} /></dd></div>
        <div><dt className="inline font-medium">Roles: </dt><dd className="inline">{account.roles.length ? account.roles.map((r) => ROLE_LABEL[r as Role] ?? r).join(', ') : 'none'}</dd></div>
      </dl>

      {notice && <p role="status" className="mt-3 text-sm text-green-700">{notice}</p>}

      <div className="mt-4 flex flex-wrap gap-2">
        {account.suspended
          ? <button type="button" className={NEUTRAL} onClick={() => open({ kind: 'unsuspend' })}>Lift suspension</button>
          : <button type="button" className={NEUTRAL} onClick={() => open({ kind: 'suspend' })}>Suspend…</button>}
        {account.banned
          ? <button type="button" className={NEUTRAL} onClick={() => open({ kind: 'unban' })}>Lift ban</button>
          : <button type="button" className={DANGER} onClick={() => open({ kind: 'ban' })}>Ban…</button>}
        <button type="button" className={NEUTRAL} onClick={() => open({ kind: 'logout-everywhere' })}>Log out everywhere</button>
      </div>

      {isAdmin && (
        <fieldset className="mt-4">
          <legend className="text-sm font-medium text-gray-900">Roles (admins only)</legend>
          <ul className="mt-2 space-y-2">
            {ASSIGNABLE_ROLES.map((role) => {
              const held = account.roles.includes(role);
              return (
                <li key={role} className="flex items-center gap-3 text-sm text-gray-700">
                  <span className="w-28">{ROLE_LABEL[role]}</span>
                  <span>{held ? 'held' : 'not held'}</span>
                  <button type="button" className={held ? DANGER : NEUTRAL}
                    aria-label={`${held ? 'Revoke' : 'Grant'} the ${ROLE_LABEL[role]} role`}
                    onClick={() => open({ kind: held ? 'revoke' : 'grant', role })}>
                    {held ? 'Revoke' : 'Grant'}
                  </button>
                </li>
              );
            })}
          </ul>
        </fieldset>
      )}

      {pending && (
        <ConfirmDialog title={title(pending)} confirmLabel={confirmLabel(pending)} danger={!!hurts}
          canConfirm={canConfirm} busy={busy} error={error} onConfirm={run} onCancel={() => setPending(null)}>
          <p>
            {account.email}
            {pending.kind === 'suspend' && ' will be able to read but not post or change anything, and is logged out now.'}
            {pending.kind === 'ban' && ' will be able to read but not post or change anything, and is logged out now.'}
            {pending.kind === 'logout-everywhere' && ' will have to sign in again on every device, and every app is signed out.'}
          </p>
          {needsDate && (
            <div>
              <label htmlFor="suspend-until" className="block font-medium text-gray-700">Suspended until (the last day)</label>
              <input id="suspend-until" type="date" min={tomorrow()} value={until} onChange={(e) => setUntil(e.target.value)} className={FIELD} />
            </div>
          )}
          {(needsReason || pending.kind === 'logout-everywhere' || pending.kind === 'unsuspend' || pending.kind === 'unban') && (
            <div>
              <label htmlFor="action-reason" className="block font-medium text-gray-700">
                Reason{needsReason ? ' (required)' : ' (optional)'}
              </label>
              <textarea id="action-reason" rows={3} value={reason} onChange={(e) => setReason(e.target.value)} className={FIELD} />
            </div>
          )}
        </ConfirmDialog>
      )}
    </section>
  );
}

export default function AdminAccountsPage() {
  const { roles } = useRoles();
  const [query, setQuery] = useState('');
  const [account, setAccount] = useState<Account | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const search = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;
    setBusy(true);
    setError('');
    setAccount(null);
    const r = await findAccount(query.trim());
    setBusy(false);
    if (r.ok && r.data) setAccount(r.data.account);
    else setError(r.status === 404 ? 'No account has that email address or id.' : r.error || 'Could not look that up.');
  };

  return (
    <AdminPage title="Accounts" need="moderator" audience="the archive's admins and moderators"
      intro="Find an account by its email address or id. Moderators can suspend, ban and log out accounts; only admins can act on staff or change roles.">
      <form onSubmit={search} className="flex items-end gap-2">
        <div className="flex-1">
          <label htmlFor="account-query" className="block text-sm font-medium text-gray-700">Email address or account id</label>
          <input id="account-query" type="text" value={query} onChange={(e) => setQuery(e.target.value)} className={FIELD} />
        </div>
        <button type="submit" disabled={busy || !query.trim()} className={`${BTN} border-transparent text-white bg-orange-700 hover:bg-orange-800`}>Find</button>
      </form>
      {error && <p role="alert" className="mt-4 text-sm text-red-600">{error}</p>}
      {account && <AccountView key={account.id} account={account} isAdmin={hasRole(roles, 'admin')} onChanged={(a) => setAccount(a)} />}
    </AdminPage>
  );
}
