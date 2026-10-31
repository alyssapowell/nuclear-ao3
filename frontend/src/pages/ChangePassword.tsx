import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { changePassword, getSessions, revokeSession, PASSWORD_MIN, PASSWORD_MAX, type Session } from '@/lib/api';
import { getAuthToken, logout } from '@/utils/auth';

const INPUT =
  'mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500';

function formatTime(value: string) {
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString();
}

export default function ChangePasswordPage() {
  const navigate = useNavigate();
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [confirm, setConfirm] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);

  const [sessions, setSessions] = useState<Session[] | null>(null);
  const [sessionsError, setSessionsError] = useState('');

  const loadSessions = useCallback(async () => {
    try {
      const result = await getSessions(getAuthToken());
      if (result.ok) {
        setSessions(result.data ?? []);
        setSessionsError('');
      } else {
        setSessionsError(result.error || 'Could not load your sessions.');
      }
    } catch {
      setSessionsError('Could not load your sessions.');
    }
  }, []);

  useEffect(() => {
    loadSessions();
  }, [loadSessions]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaved(false);
    if (next !== confirm) {
      setError('New passwords do not match');
      return;
    }
    if (next.length < PASSWORD_MIN || next.length > PASSWORD_MAX) {
      setError(`Password must be ${PASSWORD_MIN} to ${PASSWORD_MAX} characters long`);
      return;
    }
    setSaving(true);
    setError('');
    try {
      const result = await changePassword(current, next, getAuthToken());
      if (result.ok) {
        setSaved(true);
        setCurrent('');
        setNext('');
        setConfirm('');
        loadSessions(); // every other session was just logged out
      } else {
        setError(result.error || 'Something went wrong. Please try again.');
      }
    } catch {
      setError('Something went wrong. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const handleRevoke = async (session: Session) => {
    if (session.current) {
      await logout();
      navigate('/auth/login');
      return;
    }
    try {
      const result = await revokeSession(session.id, getAuthToken());
      if (!result.ok && result.code !== 'not_found') {
        setSessionsError(result.error || 'Could not log out that session.');
        return;
      }
    } catch {
      setSessionsError('Could not log out that session.');
      return;
    }
    loadSessions();
  };

  return (
    <div className="max-w-2xl mx-auto mt-8 space-y-8">
      <div className="p-6 bg-white rounded-lg shadow-md">
        <h1 className="text-2xl font-bold text-gray-900 mb-6">Change Password</h1>

        {saved && (
          <p role="status" className="mb-4 text-sm text-green-700">
            Your password has been changed. Your other sessions were logged out.
          </p>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <p role="alert" className="text-red-600 text-sm">{error}</p>}
          <div>
            <label htmlFor="current-password" className="block text-sm font-medium text-gray-700">
              Current password
            </label>
            <input
              type="password"
              id="current-password"
              autoComplete="current-password"
              value={current}
              onChange={(e) => setCurrent(e.target.value)}
              className={INPUT}
              disabled={saving}
              required
            />
          </div>
          <div>
            <label htmlFor="new-password" className="block text-sm font-medium text-gray-700">
              New password
            </label>
            <input
              type="password"
              id="new-password"
              autoComplete="new-password"
              value={next}
              onChange={(e) => setNext(e.target.value)}
              className={INPUT}
              disabled={saving}
              required
              minLength={PASSWORD_MIN}
              maxLength={PASSWORD_MAX}
            />
            <p className="mt-1 text-xs text-gray-600">Password must be 6 to 40 characters long</p>
          </div>
          <div>
            <label htmlFor="confirm-new-password" className="block text-sm font-medium text-gray-700">
              Confirm new password
            </label>
            <input
              type="password"
              id="confirm-new-password"
              autoComplete="new-password"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              className={INPUT}
              disabled={saving}
              required
            />
          </div>
          <button
            type="submit"
            disabled={saving}
            className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-orange-600 hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Change password
          </button>
        </form>
      </div>

      <div className="p-6 bg-white rounded-lg shadow-md">
        <h2 className="text-xl font-bold text-gray-900 mb-4">Sessions</h2>
        {sessionsError && <p role="alert" className="mb-4 text-red-600 text-sm">{sessionsError}</p>}
        {sessions === null && !sessionsError && <p className="text-sm text-gray-700">Loading…</p>}
        {sessions && sessions.length === 0 && <p className="text-sm text-gray-700">No active sessions.</p>}
        {sessions && sessions.length > 0 && (
          <ul className="divide-y divide-gray-200">
            {sessions.map((s) => (
              <li key={s.id} className="py-3 flex items-start justify-between gap-4">
                <div className="text-sm text-gray-700">
                  <p className="font-medium text-gray-900">
                    {s.user_agent || 'Unknown device'}
                    {s.current && <span className="ml-2 text-orange-700">(this session)</span>}
                  </p>
                  <p>{s.ip_address}</p>
                  <p>Signed in {formatTime(s.created_at)}; last active {formatTime(s.last_seen)}</p>
                </div>
                <button
                  type="button"
                  onClick={() => handleRevoke(s)}
                  aria-label={`Log out ${s.user_agent || 'session'}${s.current ? ' (this session)' : ''}`}
                  className="shrink-0 py-1 px-3 border border-gray-300 rounded-md text-sm text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-orange-500"
                >
                  Log out
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
