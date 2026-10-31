import { useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { confirmPasswordReset, PASSWORD_MIN, PASSWORD_MAX } from '@/lib/api';

const INPUT =
  'mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500';

export default function ResetPasswordPage() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') || '';
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [invalidToken, setInvalidToken] = useState(!token);
  const [done, setDone] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (password !== confirm) {
      setError('Passwords do not match');
      return;
    }
    if (password.length < PASSWORD_MIN || password.length > PASSWORD_MAX) {
      setError(`Password must be ${PASSWORD_MIN} to ${PASSWORD_MAX} characters long`);
      return;
    }
    setLoading(true);
    setError('');
    try {
      const result = await confirmPasswordReset(token, password);
      if (result.ok) setDone(true);
      else if (result.code === 'invalid_token') setInvalidToken(true);
      else setError(result.error || 'Something went wrong. Please try again.');
    } catch {
      setError('Something went wrong. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-md mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Reset Password</h1>

      {done ? (
        <div role="status" className="space-y-4">
          <p className="text-sm text-gray-700">Your password has been changed. Log in with your new password.</p>
          <Link to="/auth/login" className="text-orange-700 hover:text-orange-500 text-sm">
            Log in
          </Link>
        </div>
      ) : invalidToken ? (
        <div className="space-y-4">
          <p role="alert" className="text-red-600 text-sm">This reset link is invalid or has expired.</p>
          <Link to="/auth/forgot-password" className="text-orange-700 hover:text-orange-500 text-sm">
            Request a new reset link
          </Link>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <p role="alert" className="text-red-600 text-sm">{error}</p>}
          <div>
            <label htmlFor="new-password" className="block text-sm font-medium text-gray-700">
              New password
            </label>
            <input
              type="password"
              id="new-password"
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={INPUT}
              disabled={loading}
              required
              minLength={PASSWORD_MIN}
              maxLength={PASSWORD_MAX}
            />
            <p className="mt-1 text-xs text-gray-600">Password must be 6 to 40 characters long</p>
          </div>
          <div>
            <label htmlFor="confirm-password" className="block text-sm font-medium text-gray-700">
              Confirm new password
            </label>
            <input
              type="password"
              id="confirm-password"
              autoComplete="new-password"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              className={INPUT}
              disabled={loading}
              required
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-orange-600 hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Change password
          </button>
        </form>
      )}
    </div>
  );
}
