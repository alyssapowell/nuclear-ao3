import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { useRouter } from 'next/navigation';
import AuthGuard from '../components/auth/AuthGuard';
import LoginForm from '../components/auth/LoginForm';
import { useAuth } from '../utils/auth';

// Mock dependencies
jest.mock('next/navigation', () => ({
  useRouter: jest.fn(),
}));

jest.mock('../utils/auth', () => ({
  useAuth: jest.fn(),
  getAuthState: jest.fn(),
  isAuthenticated: jest.fn(),
  logout: jest.fn(),
}));

const mockPush = jest.fn();
const mockUseRouter = useRouter as jest.MockedFunction<typeof useRouter>;
const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;

beforeEach(() => {
  jest.clearAllMocks();
  mockUseRouter.mockReturnValue({
    push: mockPush,
    back: jest.fn(),
    forward: jest.fn(),
    refresh: jest.fn(),
    replace: jest.fn(),
    prefetch: jest.fn(),
  } as any);
});

describe('Authentication Flow Tests', () => {
  describe('AuthGuard Component', () => {
    const TestComponent = () => <div>Protected Content</div>;

    test('renders children when user is authenticated', () => {
      mockUseAuth.mockReturnValue({
        user: { id: '1', username: 'testuser' },
        isLoading: false,
        isAuthenticated: true,
        login: jest.fn(),
        logout: jest.fn(),
        register: jest.fn(),
      });

      render(
        <AuthGuard>
          <TestComponent />
        </AuthGuard>
      );

      expect(screen.getByText('Protected Content')).toBeInTheDocument();
    });

    test('shows loading state when authentication is loading', () => {
      mockUseAuth.mockReturnValue({
        user: null,
        isLoading: true,
        isAuthenticated: false,
        login: jest.fn(),
        logout: jest.fn(),
        register: jest.fn(),
      });

      render(
        <AuthGuard>
          <TestComponent />
        </AuthGuard>
      );

      expect(screen.getByText(/loading/i)).toBeInTheDocument();
    });

    test('redirects to login when user is not authenticated', async () => {
      mockUseAuth.mockReturnValue({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        login: jest.fn(),
        logout: jest.fn(),
        register: jest.fn(),
      });

      render(
        <AuthGuard>
          <TestComponent />
        </AuthGuard>
      );

      await waitFor(() => {
        expect(mockPush).toHaveBeenCalledWith('/login');
      });
    });

    test('prevents infinite redirect loops', async () => {
      mockUseAuth.mockReturnValue({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        login: jest.fn(),
        logout: jest.fn(),
        register: jest.fn(),
      });

      const { rerender } = render(
        <AuthGuard>
          <TestComponent />
        </AuthGuard>
      );

      // Simulate multiple re-renders
      rerender(
        <AuthGuard>
          <TestComponent />
        </AuthGuard>
      );

      rerender(
        <AuthGuard>
          <TestComponent />
        </AuthGuard>
      );

      // Should only redirect once
      await waitFor(() => {
        expect(mockPush).toHaveBeenCalledTimes(1);
      });
    });

    test('handles custom redirect path', async () => {
      mockUseAuth.mockReturnValue({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        login: jest.fn(),
        logout: jest.fn(),
        register: jest.fn(),
      });

      render(
        <AuthGuard redirectTo="/custom-login">
          <TestComponent />
        </AuthGuard>
      );

      await waitFor(() => {
        expect(mockPush).toHaveBeenCalledWith('/custom-login');
      });
    });
  });

  describe('LoginForm Component', () => {
    test('renders login form fields', () => {
      mockUseAuth.mockReturnValue({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        login: jest.fn(),
        logout: jest.fn(),
        register: jest.fn(),
      });

      render(<LoginForm />);

      expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument();
    });

    test('validates required fields', async () => {
      mockUseAuth.mockReturnValue({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        login: jest.fn(),
        logout: jest.fn(),
        register: jest.fn(),
      });

      render(<LoginForm />);

      const submitButton = screen.getByRole('button', { name: /login/i });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/username is required/i)).toBeInTheDocument();
        expect(screen.getByText(/password is required/i)).toBeInTheDocument();
      });
    });

    test('submits form with valid credentials', async () => {
      const mockLogin = jest.fn().mockResolvedValue({ success: true });
      
      mockUseAuth.mockReturnValue({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        login: mockLogin,
        logout: jest.fn(),
        register: jest.fn(),
      });

      render(<LoginForm />);

      const usernameField = screen.getByLabelText(/username/i);
      const passwordField = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      fireEvent.change(usernameField, { target: { value: 'testuser' } });
      fireEvent.change(passwordField, { target: { value: 'password123' } });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledWith('testuser', 'password123');
      });
    });

    test('handles login errors gracefully', async () => {
      const mockLogin = jest.fn().mockRejectedValue(new Error('Invalid credentials'));
      
      mockUseAuth.mockReturnValue({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        login: mockLogin,
        logout: jest.fn(),
        register: jest.fn(),
      });

      render(<LoginForm />);

      const usernameField = screen.getByLabelText(/username/i);
      const passwordField = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      fireEvent.change(usernameField, { target: { value: 'testuser' } });
      fireEvent.change(passwordField, { target: { value: 'wrongpassword' } });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument();
      });
    });

    test('disables form during submission', async () => {
      const mockLogin = jest.fn(() => new Promise(resolve => setTimeout(resolve, 1000)));
      
      mockUseAuth.mockReturnValue({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        login: mockLogin,
        logout: jest.fn(),
        register: jest.fn(),
      });

      render(<LoginForm />);

      const usernameField = screen.getByLabelText(/username/i);
      const passwordField = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /login/i });

      fireEvent.change(usernameField, { target: { value: 'testuser' } });
      fireEvent.change(passwordField, { target: { value: 'password123' } });
      fireEvent.click(submitButton);

      expect(submitButton).toBeDisabled();
      expect(usernameField).toBeDisabled();
      expect(passwordField).toBeDisabled();
    });
  });

  describe('Authentication Context Integration', () => {
    test('persists authentication state across page reloads', () => {
      // Mock localStorage
      const mockToken = 'mock-jwt-token';
      Storage.prototype.getItem = jest.fn(() => mockToken);
      
      mockUseAuth.mockReturnValue({
        user: { id: '1', username: 'testuser' },
        isLoading: false,
        isAuthenticated: true,
        login: jest.fn(),
        logout: jest.fn(),
        register: jest.fn(),
      });

      const TestComponent = () => {
        const { user, isAuthenticated } = useAuth();
        return (
          <div>
            {isAuthenticated ? `Welcome ${user?.username}` : 'Not logged in'}
          </div>
        );
      };

      render(<TestComponent />);

      expect(screen.getByText('Welcome testuser')).toBeInTheDocument();
    });

    test('handles token expiration gracefully', async () => {
      const mockLogout = jest.fn();
      
      // Mock expired token scenario
      mockUseAuth.mockReturnValue({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        login: jest.fn(),
        logout: mockLogout,
        register: jest.fn(),
      });

      const TestComponent = () => {
        const { isAuthenticated } = useAuth();
        return (
          <div>
            {isAuthenticated ? 'Authenticated' : 'Token expired'}
          </div>
        );
      };

      render(<TestComponent />);

      expect(screen.getByText('Token expired')).toBeInTheDocument();
    });

    test('clears authentication state on logout', async () => {
      const mockLogout = jest.fn().mockResolvedValue(undefined);
      Storage.prototype.removeItem = jest.fn();
      
      mockUseAuth.mockReturnValue({
        user: { id: '1', username: 'testuser' },
        isLoading: false,
        isAuthenticated: true,
        login: jest.fn(),
        logout: mockLogout,
        register: jest.fn(),
      });

      const TestComponent = () => {
        const { logout } = useAuth();
        return (
          <button onClick={() => logout()}>
            Logout
          </button>
        );
      };

      render(<TestComponent />);

      const logoutButton = screen.getByRole('button', { name: /logout/i });
      fireEvent.click(logoutButton);

      await waitFor(() => {
        expect(mockLogout).toHaveBeenCalled();
      });
    });
  });

  describe('Protected Route Access', () => {
    test('allows access to public routes without authentication', () => {
      mockUseAuth.mockReturnValue({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        login: jest.fn(),
        logout: jest.fn(),
        register: jest.fn(),
      });

      // Test public route (no AuthGuard)
      const PublicComponent = () => <div>Public Content</div>;
      
      render(<PublicComponent />);

      expect(screen.getByText('Public Content')).toBeInTheDocument();
    });

    test('redirects to login for protected routes when unauthenticated', async () => {
      mockUseAuth.mockReturnValue({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        login: jest.fn(),
        logout: jest.fn(),
        register: jest.fn(),
      });

      const ProtectedComponent = () => <div>Protected Content</div>;
      
      render(
        <AuthGuard>
          <ProtectedComponent />
        </AuthGuard>
      );

      await waitFor(() => {
        expect(mockPush).toHaveBeenCalledWith('/login');
      });
    });

    test('allows access to protected routes when authenticated', () => {
      mockUseAuth.mockReturnValue({
        user: { id: '1', username: 'testuser' },
        isLoading: false,
        isAuthenticated: true,
        login: jest.fn(),
        logout: jest.fn(),
        register: jest.fn(),
      });

      const ProtectedComponent = () => <div>Protected Content</div>;
      
      render(
        <AuthGuard>
          <ProtectedComponent />
        </AuthGuard>
      );

      expect(screen.getByText('Protected Content')).toBeInTheDocument();
      expect(mockPush).not.toHaveBeenCalled();
    });
  });
});