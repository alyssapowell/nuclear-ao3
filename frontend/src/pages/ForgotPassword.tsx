import { useState } from 'react';
import { Link } from 'react-router-dom';
import { requestPasswordReset } from '@/lib/api';

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email.trim()) {
      setError('Please enter your email address');
      return;
    }
    setLoading(true);
    setError('');
    try {
      await requestPasswordReset(email.trim());
    } catch {
      // Same answer whatever happens, so this page never reveals which addresses have accounts.
    } finally {
      setLoading(false);
      setSent(true);
    }
  };

  return (
    <div className="max-w-md mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Forgot Password</h1>

      {sent ? (
        <p role="status" className="text-sm text-gray-700">
          If an account uses that address, we've sent a reset link.
        </p>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <p role="alert" className="text-red-600 text-sm">{error}</p>}
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-gray-700">
              Email
            </label>
            <input
              type="email"
              id="email"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value);
                if (error) setError('');
              }}
              className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500"
              placeholder="Enter your email"
              disabled={loading}
              required
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-orange-600 hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Send reset link
          </button>
        </form>
      )}

      <p className="mt-4 text-center text-sm text-gray-700">
        <Link to="/auth/login" className="text-orange-700 hover:text-orange-500">
          Back to log in
        </Link>
      </p>
    </div>
  );
}
