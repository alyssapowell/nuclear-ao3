// Settings, "Connected apps": the apps this person has let sign in with their
// account, and a way to disconnect each one.
import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { disconnectApp, listConnectedApps, SCOPE_CHOICES, type ConnectedApp } from '@/lib/oauth';

function formatTime(value: string | null) {
  if (!value) return 'never';
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString();
}

function scopeLabel(name: string) {
  return SCOPE_CHOICES.find((s) => s.name === name)?.label ?? name;
}

export default function ConnectedAppsPage() {
  const [apps, setApps] = useState<ConnectedApp[] | null>(null);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const load = useCallback(async () => {
    const result = await listConnectedApps();
    if (result.ok) {
      setApps(result.data?.applications ?? []);
      setError('');
    } else {
      setError(result.error || 'Could not load your connected apps.');
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const handleDisconnect = async (app: ConnectedApp) => {
    if (!window.confirm(`Disconnect ${app.client_name}? It will be signed out everywhere and will have to ask you again.`)) {
      return;
    }
    setNotice('');
    const result = await disconnectApp(app.client_id);
    if (!result.ok) {
      setError(result.error || 'Could not disconnect that app.');
      return;
    }
    setError('');
    setNotice(`${app.client_name} was disconnected.`);
    load();
  };

  return (
    <div className="max-w-2xl mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
      <h1 className="text-2xl font-bold text-gray-900 mb-2">Connected apps</h1>
      <p className="mb-4 text-sm text-gray-700">
        These apps can sign in with your account. Disconnecting an app signs it out everywhere, on every device, and
        cancels what you allowed. If it wants access again, it has to ask you again.
      </p>

      {error && <p role="alert" className="mb-4 text-red-600 text-sm">{error}</p>}
      {notice && <p role="status" className="mb-4 text-sm text-green-700">{notice}</p>}
      {apps === null && !error && <p className="text-sm text-gray-700">Loading…</p>}
      {apps && apps.length === 0 && <p className="text-sm text-gray-700">No apps are connected to your account.</p>}

      {apps && apps.length > 0 && (
        <ul className="divide-y divide-gray-200">
          {apps.map((app) => (
            <li key={app.client_id} className="py-4 flex items-start justify-between gap-4">
              <div className="text-sm text-gray-700">
                <p className="font-medium text-gray-900">{app.client_name}</p>
                {app.website && app.website.startsWith('https://') && (
                  <p>
                    <a href={app.website} target="_blank" rel="noopener noreferrer" className="text-orange-700 underline">
                      {app.website}
                    </a>
                  </p>
                )}
                <p>Allowed to: {app.scopes.length ? app.scopes.map(scopeLabel).join('; ') : 'nothing yet'}</p>
                <p>Sees the pseud: {app.pseud ? app.pseud.name : 'none'}</p>
                <p>Sees your email address: {app.share_email ? 'yes' : 'no'}</p>
                <p>Allowed on {formatTime(app.granted_at)}; last used {formatTime(app.last_used)}</p>
              </div>
              <button
                type="button"
                onClick={() => handleDisconnect(app)}
                aria-label={`Disconnect ${app.client_name}`}
                className="shrink-0 py-1 px-3 border border-gray-300 rounded-md text-sm text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-orange-500"
              >
                Disconnect
              </button>
            </li>
          ))}
        </ul>
      )}

      <p className="mt-6 text-sm text-gray-700">
        Building an app of your own? See <Link to="/profile/my-apps" className="text-orange-700 underline">Your apps</Link>.
      </p>
    </div>
  );
}
