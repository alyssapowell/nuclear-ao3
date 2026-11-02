// Settings, "Your apps": for people who build something that signs in with
// this archive. Register an app, see where it stands in review, edit it,
// reset its secret and delete it.
import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  deleteMyApp,
  listMyApps,
  registerApp,
  resetMyAppSecret,
  SCOPE_CHOICES,
  updateMyApp,
  type MyApp,
} from '@/lib/oauth';

const INPUT =
  'mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500';
const PRIMARY =
  'py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-orange-600 hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed';
const SECONDARY =
  'py-1 px-3 border border-gray-300 rounded-md text-sm text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-orange-500';

const STATUS_TEXT: Record<string, string> = {
  pending: 'Waiting for review. It cannot sign anyone in until an admin approves it.',
  approved: 'Approved. People can sign in with it.',
  rejected: 'Not approved. Fix what the note says and save it to send it back for review.',
  suspended: 'Suspended by the admins. It cannot sign anyone in and cannot be edited.',
};

interface FormValues {
  name: string;
  description: string;
  website: string;
  logo: string;
  redirects: string;
  scopes: string[];
  isPublic: boolean;
}

const EMPTY: FormValues = { name: '', description: '', website: '', logo: '', redirects: '', scopes: ['openid', 'profile'], isPublic: false };

function fromApp(app: MyApp): FormValues {
  return {
    name: app.client_name,
    description: app.description,
    website: app.website,
    logo: app.logo_url,
    redirects: app.redirect_uris.join('\n'),
    scopes: app.scopes,
    isPublic: app.is_public,
  };
}

function AppForm({
  idPrefix,
  initial,
  canChooseKind,
  submitLabel,
  onSubmit,
  onCancel,
}: {
  idPrefix: string;
  initial: FormValues;
  canChooseKind: boolean;
  submitLabel: string;
  onSubmit: (values: FormValues) => Promise<string | null>;
  onCancel?: () => void;
}) {
  const [v, setV] = useState(initial);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const set = <K extends keyof FormValues>(key: K, value: FormValues[K]) => setV((prev) => ({ ...prev, [key]: value }));
  const id = (name: string) => `${idPrefix}-${name}`;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError('');
    const problem = await onSubmit(v);
    if (problem) setError(problem);
    setBusy(false);
  };

  const toggleScope = (name: string) =>
    set('scopes', v.scopes.includes(name) ? v.scopes.filter((s) => s !== name) : [...v.scopes, name]);

  return (
    <form onSubmit={submit} className="space-y-4">
      {error && <p role="alert" className="text-red-600 text-sm">{error}</p>}
      <div>
        <label htmlFor={id('name')} className="block text-sm font-medium text-gray-700">App name</label>
        <input id={id('name')} className={INPUT} value={v.name} maxLength={100} required disabled={busy}
          onChange={(e) => set('name', e.target.value)} />
      </div>
      <div>
        <label htmlFor={id('description')} className="block text-sm font-medium text-gray-700">What it does</label>
        <textarea id={id('description')} className={INPUT} rows={2} value={v.description} disabled={busy}
          onChange={(e) => set('description', e.target.value)} />
      </div>
      <div>
        <label htmlFor={id('website')} className="block text-sm font-medium text-gray-700">Website (https address)</label>
        <input id={id('website')} type="url" className={INPUT} value={v.website} disabled={busy}
          onChange={(e) => set('website', e.target.value)} />
      </div>
      <div>
        <label htmlFor={id('logo')} className="block text-sm font-medium text-gray-700">Logo (https address of an image)</label>
        <input id={id('logo')} type="url" className={INPUT} value={v.logo} disabled={busy}
          onChange={(e) => set('logo', e.target.value)} />
      </div>
      <div>
        <label htmlFor={id('redirects')} className="block text-sm font-medium text-gray-700">
          Redirect addresses, one per line
        </label>
        <textarea id={id('redirects')} className={INPUT} rows={3} value={v.redirects} required disabled={busy}
          onChange={(e) => set('redirects', e.target.value)} />
        <p className="mt-1 text-xs text-gray-600">
          After someone says yes, we send them to one of these with a sign-in code. Use https addresses you control.
        </p>
      </div>
      <fieldset>
        <legend className="block text-sm font-medium text-gray-700">What the app may ask for</legend>
        <div className="mt-1 space-y-1">
          {SCOPE_CHOICES.map((s) => (
            <div key={s.name} className="flex items-start gap-2">
              <input id={id(`scope-${s.name}`)} type="checkbox" checked={v.scopes.includes(s.name)} disabled={busy}
                onChange={() => toggleScope(s.name)}
                className="mt-1 h-4 w-4 rounded border-gray-300 text-orange-600 focus:ring-orange-500" />
              <label htmlFor={id(`scope-${s.name}`)} className="text-sm text-gray-800">
                <span className="font-mono">{s.name}</span>: {s.label}
              </label>
            </div>
          ))}
        </div>
      </fieldset>
      {canChooseKind && (
        <div className="flex items-start gap-2">
          <input id={id('public')} type="checkbox" checked={v.isPublic} disabled={busy}
            onChange={(e) => set('isPublic', e.target.checked)}
            className="mt-1 h-4 w-4 rounded border-gray-300 text-orange-600 focus:ring-orange-500" />
          <label htmlFor={id('public')} className="text-sm text-gray-800">
            This is a public app (a phone app or a page that runs in the browser). It gets no secret.
          </label>
        </div>
      )}
      <div className="flex gap-3">
        <button type="submit" disabled={busy} className={PRIMARY}>{submitLabel}</button>
        {onCancel && <button type="button" onClick={onCancel} disabled={busy} className={SECONDARY}>Cancel</button>}
      </div>
    </form>
  );
}

const lines = (text: string) => text.split('\n').map((l) => l.trim()).filter(Boolean);

export default function MyAppsPage() {
  const [apps, setApps] = useState<MyApp[] | null>(null);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [editing, setEditing] = useState<string | null>(null);
  const [secret, setSecret] = useState<{ name: string; value: string } | null>(null);

  const load = useCallback(async () => {
    const result = await listMyApps();
    if (result.ok) {
      setApps(result.data?.apps ?? []);
      setError('');
    } else {
      setError(result.error || 'Could not load your apps.');
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const handleRegister = async (v: FormValues): Promise<string | null> => {
    const result = await registerApp({
      client_name: v.name.trim(),
      description: v.description,
      website: v.website.trim(),
      logo_url: v.logo.trim(),
      redirect_uris: lines(v.redirects),
      scopes: v.scopes,
      is_public: v.isPublic,
      grant_types: ['authorization_code', 'refresh_token'],
    });
    if (!result.ok) return result.error || 'Could not register the app.';
    setNotice(`${v.name.trim()} was registered. It is waiting for an admin to review it.`);
    setSecret(result.data?.client_secret ? { name: v.name.trim(), value: result.data.client_secret } : null);
    load();
    return null;
  };

  const handleSave = (app: MyApp) => async (v: FormValues): Promise<string | null> => {
    const result = await updateMyApp(app.client_id, {
      client_name: v.name.trim(),
      description: v.description,
      website: v.website.trim(),
      logo_url: v.logo.trim(),
      redirect_uris: lines(v.redirects),
      scopes: v.scopes,
    });
    if (!result.ok) return result.error || 'Could not save the changes.';
    setEditing(null);
    setNotice(
      result.data?.status === 'pending'
        ? `${v.name.trim()} was sent back to review and everyone was signed out of it.`
        : result.data?.display_review_pending
          ? `${v.name.trim()}: the new name, website, logo or description is waiting for review. People keep seeing the old details until then.`
          : `${v.name.trim()} was saved.`,
    );
    load();
    return null;
  };

  const handleReset = async (app: MyApp) => {
    if (!window.confirm(`Reset the secret for ${app.client_name}? The old secret stops working and everyone is signed out of the app.`)) return;
    const result = await resetMyAppSecret(app.client_id);
    if (!result.ok || !result.data?.client_secret) {
      setError(result.error || 'Could not reset the secret.');
      return;
    }
    setError('');
    setNotice('');
    setSecret({ name: app.client_name, value: result.data.client_secret });
  };

  const handleDelete = async (app: MyApp) => {
    if (!window.confirm(`Delete ${app.client_name}? This signs everyone out of it and cannot be undone.`)) return;
    const result = await deleteMyApp(app.client_id);
    if (!result.ok) {
      setError(result.error || 'Could not delete the app.');
      return;
    }
    setError('');
    setNotice(`${app.client_name} was deleted.`);
    load();
  };

  return (
    <div className="max-w-3xl mx-auto mt-8 space-y-8">
      <div className="p-6 bg-white rounded-lg shadow-md">
        <h1 className="text-2xl font-bold text-gray-900 mb-2">Your apps</h1>
        <p className="mb-2 text-sm text-gray-700">
          If you build something that lets people sign in with their account here, register it below. A new app starts
          as <strong>pending</strong> and does nothing until an admin approves it.
        </p>
        <p className="text-sm text-gray-700">
          Once it is approved, changing its redirect addresses or scopes sends it back to review and signs everyone out
          of it. Changing its name, website, logo or description also needs approval, but people keep seeing the old
          details and nobody is signed out while it waits.
        </p>
        <p className="mt-2 text-sm text-gray-700">
          People who use apps can see and disconnect them under{' '}
          <Link to="/profile/apps" className="text-orange-700 underline">Connected apps</Link>.
        </p>
      </div>

      {error && <p role="alert" className="text-red-600 text-sm">{error}</p>}
      {notice && <p role="status" className="text-sm text-green-700">{notice}</p>}

      {secret && (
        <div role="alert" className="p-4 bg-yellow-50 border border-yellow-300 rounded-md text-sm text-gray-900">
          <p className="font-semibold">Secret for {secret.name}</p>
          <p className="my-2 break-all font-mono bg-white border border-yellow-200 rounded p-2" data-testid="client-secret">
            {secret.value}
          </p>
          <p>Copy it now, it won&apos;t be shown again. If you lose it, reset it to get a new one.</p>
          <button type="button" onClick={() => setSecret(null)} className={`${SECONDARY} mt-2`}>I have copied it</button>
        </div>
      )}

      <div className="p-6 bg-white rounded-lg shadow-md">
        <h2 className="text-xl font-bold text-gray-900 mb-4">Register an app</h2>
        <AppForm idPrefix="new" initial={EMPTY} canChooseKind submitLabel="Register app" onSubmit={handleRegister} />
      </div>

      <div className="p-6 bg-white rounded-lg shadow-md">
        <h2 className="text-xl font-bold text-gray-900 mb-4">Your registered apps</h2>
        {apps === null && !error && <p className="text-sm text-gray-700">Loading…</p>}
        {apps && apps.length === 0 && <p className="text-sm text-gray-700">You have not registered any apps.</p>}
        {apps && apps.length > 0 && (
          <ul className="divide-y divide-gray-200">
            {apps.map((app) => (
              <li key={app.client_id} className="py-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="text-sm text-gray-700">
                    <p className="font-medium text-gray-900">
                      {app.client_name} <span className="ml-2 uppercase text-xs text-orange-800">{app.status}</span>
                    </p>
                    <p>{STATUS_TEXT[app.status]}</p>
                    {app.review_note && <p>Note from the reviewer: {app.review_note}</p>}
                    {app.display_review_pending && (
                      <p>
                        Waiting for review: your changes to the name, website, logo or description
                        {app.proposed ? ` (new name: ${app.proposed.client_name})` : ''}. People still see the
                        old details.
                      </p>
                    )}
                    <p>Client id: <span className="font-mono">{app.client_id}</span></p>
                    <p>{app.is_public ? 'Public app (no secret)' : 'Confidential app (has a secret)'}</p>
                    <p>Scopes: {app.scopes.join(', ')}</p>
                  </div>
                  <div className="shrink-0 flex flex-col gap-2">
                    {app.status !== 'suspended' && (
                      <button type="button" className={SECONDARY} aria-label={`Edit ${app.client_name}`}
                        onClick={() => setEditing(editing === app.client_id ? null : app.client_id)}>
                        Edit
                      </button>
                    )}
                    {!app.is_public && (
                      <button type="button" className={SECONDARY} aria-label={`Reset secret for ${app.client_name}`}
                        onClick={() => handleReset(app)}>
                        Reset secret
                      </button>
                    )}
                    <button type="button" className={SECONDARY} aria-label={`Delete ${app.client_name}`}
                      onClick={() => handleDelete(app)}>
                      Delete
                    </button>
                  </div>
                </div>
                {editing === app.client_id && (
                  <div className="mt-4 p-4 bg-gray-50 rounded-md">
                    <AppForm idPrefix={`edit-${app.client_id}`} initial={fromApp(app)} canChooseKind={false}
                      submitLabel="Save changes" onSubmit={handleSave(app)} onCancel={() => setEditing(null)} />
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
