import { useEffect, useRef, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { verifyEmail } from '@/lib/api';
import ResendActivation from '@/components/auth/ResendActivation';

type Status = 'working' | 'done' | 'invalid' | 'failed';

export default function ActivatePage() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') || '';
  const [status, setStatus] = useState<Status>(token ? 'working' : 'invalid');
  const [error, setError] = useState('');
  // The token can only be used once, so make sure React's StrictMode double
  // run in development doesn't spend it twice.
  const started = useRef(false);

  useEffect(() => {
    if (!token || started.current) return;
    started.current = true;
    verifyEmail(token)
      .then((result) => {
        if (result.ok) setStatus('done');
        else if (result.code === 'invalid_token') setStatus('invalid');
        else {
          setError(result.error || 'Something went wrong. Please try again.');
          setStatus('failed');
        }
      })
      .catch(() => {
        setError('Something went wrong. Please try again.');
        setStatus('failed');
      });
  }, [token]);

  return (
    <div className="max-w-md mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Activate Your Account</h1>

      {status === 'working' && <p role="status" className="text-sm text-gray-700">Activating your account…</p>}

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

      {status === 'failed' && <p role="alert" className="text-red-600 text-sm">{error}</p>}
    </div>
  );
}
