import { useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { verifyEmail } from '@/lib/api';
import ResendActivation from '@/components/auth/ResendActivation';

const INPUT =
  'mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500';

type Status = 'form' | 'done' | 'invalid';

// Activating asks for the account's password as well as the emailed link,
// so an account someone else signed up with your address can't be
// activated by you clicking the link (and then logged into by them).
export default function ActivatePage() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') || '';
  const [status, setStatus] = useState<Status>(token ? 'form' : 'invalid');
  const [password, setPassword] = useState('');
  const [wrongPassword, setWrongPassword] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    setWrongPassword(false);
    try {
      const result = await verifyEmail(token, password);
      if (result.ok) setStatus('done');
      else if (result.code === 'invalid_token') setStatus('invalid');
      else if (result.code === 'wrong_password') setWrongPassword(true);
      else setError(result.error || 'Something went wrong. Please try again.');
    } catch {
      setError('Something went wrong. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-md mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Activate Your Account</h1>

      {status === 'done' && (
        <div role="status" className="space-y-4">
          <p className="text-sm text-gray-700">Your account is activated. You can now log in.</p>
          <Link to="/auth/login" className="text-orange-700 hover:text-orange-500 text-sm">
            Log in
          </Link>
        </div>
      )}

      {status === 'invalid' && (
        <div className="space-y-4">
          <p role="alert" className="text-red-600 text-sm">
            This activation link is invalid or has expired. Enter your email address and we'll send a new one.
          </p>
          <ResendActivation />
        </div>
      )}

      {status === 'form' && (
        <form onSubmit={handleSubmit} className="space-y-4">
          <p className="text-sm text-gray-700">Enter the password you chose when you signed up to activate your account.</p>
          {wrongPassword && (
            <p role="alert" className="text-red-600 text-sm">
              That isn't this account's password. If you didn't create this account, someone may have signed up with
              your email address: <Link to="/auth/forgot-password" className="underline">reset the password</Link> to
              take it over.
            </p>
          )}
          {error && <p role="alert" className="text-red-600 text-sm">{error}</p>}
          <div>
            <label htmlFor="password" className="block text-sm font-medium text-gray-700">
              Password
            </label>
            <input
              type="password"
              id="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={INPUT}
              disabled={loading}
              required
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 px-4 rounded-md text-white bg-orange-600 hover:bg-orange-700 disabled:opacity-50"
          >
            {loading ? 'Activating…' : 'Activate account'}
          </button>
        </form>
      )}
    </div>
  );
}
