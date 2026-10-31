import { safeRedirect } from '@/utils/auth';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useState, useEffect, Suspense } from 'react';
import { register as registerRequest, PASSWORD_MIN, PASSWORD_MAX } from '@/lib/api';

interface RegisterForm {
  username: string;
  email: string;
  password: string;
  confirmPassword: string;
}

function RegisterContent() {
  const [form, setForm] = useState<RegisterForm>({ 
    username: '', 
    email: '', 
    password: '', 
    confirmPassword: '' 
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [registeredEmail, setRegisteredEmail] = useState<string | null>(null);
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  

  // Check if user is already authenticated
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const token = localStorage.getItem('auth_token');
      if (token) {
        const redirect = safeRedirect(searchParams.get('redirect'));
        navigate(redirect, { replace: true });
      }
    }
  }, [navigate, searchParams]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!form.username || !form.email || !form.password || !form.confirmPassword) {
      setError('Please fill in all fields');
      return;
    }

    if (form.password !== form.confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    if (form.password.length < PASSWORD_MIN || form.password.length > PASSWORD_MAX) {
      setError(`Password must be ${PASSWORD_MIN} to ${PASSWORD_MAX} characters long`);
      return;
    }

    try {
      setLoading(true);
      setError('');

      const result = await registerRequest(form.username, form.email.trim(), form.password);

      if (!result.ok) {
        setError(result.error || 'Registration failed. Please try again.');
        return;
      }

      // The account must be activated from the email before anyone can log in,
      // so there is no token and we don't log in here.
      setRegisteredEmail(form.email.trim());
    } catch (err) {
      console.error('Registration error:', err);
      setError('Registration failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (field: keyof RegisterForm, value: string) => {
    setForm(prev => ({ ...prev, [field]: value }));
    if (error) setError(''); // Clear error when user starts typing
  };

  if (registeredEmail) {
    return (
      <div className="max-w-md mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
        <h1 className="text-2xl font-bold text-gray-900 mb-4">Check your email to activate your account</h1>
        <p role="status" className="text-sm text-gray-700 mb-4">
          We've sent an activation link to <strong>{registeredEmail}</strong>. You need to follow it
          before you can log in. The link works for 14 days.
        </p>
        <p className="text-sm text-gray-700">
          Didn't get it? You can{' '}
          <Link to="/auth/login" className="text-orange-700 hover:text-orange-500">
            log in
          </Link>{' '}
          to ask for another one.
        </p>
      </div>
    );
  }

  return (
    <div className="max-w-md mx-auto mt-8 p-6 bg-white rounded-lg shadow-md">
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Sign Up</h1>
      
      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md">
          <p role="alert" className="text-red-600 text-sm">{error}</p>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="username" className="block text-sm font-medium text-gray-700">
            Username
          </label>
          <input
            type="text"
            id="username"
            data-testid="username"
            value={form.username}
            onChange={(e) => handleInputChange('username', e.target.value)}
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500"
            placeholder="Choose a username"
            disabled={loading}
            required
          />
        </div>
        
        <div>
          <label htmlFor="email" className="block text-sm font-medium text-gray-700">
            Email
          </label>
          <input
            type="email"
            id="email"
            data-testid="email"
            value={form.email}
            onChange={(e) => handleInputChange('email', e.target.value)}
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500"
            placeholder="Enter your email"
            disabled={loading}
            required
          />
        </div>
        
        <div>
          <label htmlFor="password" className="block text-sm font-medium text-gray-700">
            Password
          </label>
          <input
            type="password"
            id="password"
            data-testid="password"
            value={form.password}
            onChange={(e) => handleInputChange('password', e.target.value)}
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500"
            placeholder="Create a password"
            disabled={loading}
            required
            minLength={PASSWORD_MIN}
            maxLength={PASSWORD_MAX}
          />
          <p className="mt-1 text-xs text-gray-600">Password must be 6 to 40 characters long</p>
        </div>
        
        <div>
          <label htmlFor="confirmPassword" className="block text-sm font-medium text-gray-700">
            Confirm Password
          </label>
          <input
            type="password"
            id="confirmPassword"
            data-testid="confirm-password"
            value={form.confirmPassword}
            onChange={(e) => handleInputChange('confirmPassword', e.target.value)}
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500"
            placeholder="Confirm your password"
            disabled={loading}
            required
          />
        </div>
        
        <button
          type="submit"
          data-testid="register-button"
          disabled={loading}
          className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-orange-600 hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? (
            <div className="flex items-center">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
              Creating Account...
            </div>
          ) : (
            'Create Account'
          )}
        </button>
      </form>
      
      <p className="mt-4 text-center text-sm text-gray-600">
        Already have an account?{' '}
        <a 
          href={`/auth/login${searchParams.get('redirect') ? `?redirect=${encodeURIComponent(searchParams.get('redirect')!)}` : ''}`}
          className="text-orange-700 hover:text-orange-500"
        >
          Sign in here
        </a>
      </p>
    </div>
  );
}

export default function RegisterPage() {
  return (
    <Suspense fallback={<div className="min-h-screen flex items-center justify-center">Loading...</div>}>
      <RegisterContent />
    </Suspense>
  );
}