import { useState } from 'react';
import { resendVerification } from '@/lib/api';

const INPUT =
  'mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500';
const BUTTON =
  'w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-orange-600 hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed';

/**
 * "Resend activation email". If we already know the address it is just a
 * button; otherwise it asks for the address first. The server always answers
 * the same way, so the confirmation never says whether the address exists.
 */
export default function ResendActivation({ email }: { email?: string }) {
  const [typed, setTyped] = useState('');
  const [sending, setSending] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');
  const address = (email ?? typed).trim();

  const send = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!address) {
      setError('Please enter your email address');
      return;
    }
    setSending(true);
    setError('');
    try {
      const result = await resendVerification(address);
      if (result.ok) setSent(true);
      else setError(result.error || 'Something went wrong. Please try again.');
    } catch {
      setError('Something went wrong. Please try again.');
    } finally {
      setSending(false);
    }
  };

  if (sent) {
    return (
      <p role="status" className="text-sm text-gray-700">
        If an account uses that address and still needs activating, we've sent a new activation email.
      </p>
    );
  }

  return (
    <form onSubmit={send} className="space-y-3">
      {error && <p role="alert" className="text-red-600 text-sm">{error}</p>}
      {email === undefined && (
        <div>
          <label htmlFor="resend-email" className="block text-sm font-medium text-gray-700">
            Email address
          </label>
          <input
            type="email"
            id="resend-email"
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            className={INPUT}
            disabled={sending}
            required
          />
        </div>
      )}
      <button type="submit" disabled={sending} className={BUTTON}>
        Resend activation email
      </button>
    </form>
  );
}
