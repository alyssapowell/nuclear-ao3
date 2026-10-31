/**
 * Browser storage safety.
 *
 * The website is a client-only Vite app (there is no server rendering), but
 * localStorage can still be unavailable or throw: private browsing modes,
 * blocked cookies and full quotas. Components and hooks that read the login
 * token must carry on rather than crash the page.
 */
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () =>
  (await import('@/test/router-mock')).routerMock({
    useNavigate: () => mockNavigate,
    useLocation: () => ({ pathname: '/test', search: '', hash: '', state: null, key: 'test' }),
  }));

// setup.ts gives every test a fake localStorage; these change what it does.
const storage = () => vi.mocked(localStorage.getItem);
const logIn = (token: string | null) => {
  storage().mockImplementation((key: string) => (key === 'auth_token' ? token : null));
};
const breakStorage = () => {
  storage().mockImplementation(() => {
    throw new Error('localStorage not available');
  });
};

describe('Storage Safety', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    logIn(null);
    vi.spyOn(console, 'warn').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('AuthGuard', () => {
    it('shows the protected content to a logged in reader', async () => {
      logIn('real-token');
      const { default: AuthGuard } = await import('../components/auth/AuthGuard');

      render(<AuthGuard><div>Protected content</div></AuthGuard>);

      expect(await screen.findByText('Protected content')).toBeInTheDocument();
      expect(mockNavigate).not.toHaveBeenCalled();
    });

    it('sends a logged out reader to log in, remembering where they were', async () => {
      const { default: AuthGuard } = await import('../components/auth/AuthGuard');

      render(<AuthGuard><div>Protected content</div></AuthGuard>);

      expect(await screen.findByRole('heading', { name: 'Authentication Required' })).toBeInTheDocument();
      expect(screen.queryByText('Protected content')).not.toBeInTheDocument();
      expect(mockNavigate).toHaveBeenCalledWith(
        expect.stringMatching(/^\/auth\/login\?redirect=/),
        { replace: true }
      );
    });

    it('treats unreadable storage as logged out instead of crashing', async () => {
      breakStorage();
      const { default: AuthGuard } = await import('../components/auth/AuthGuard');

      render(<AuthGuard><div>Protected content</div></AuthGuard>);

      expect(await screen.findByRole('heading', { name: 'Authentication Required' })).toBeInTheDocument();
      expect(screen.queryByText('Protected content')).not.toBeInTheDocument();
      expect(storage()).toHaveBeenCalled();
    });
  });

  describe('Components', () => {
    it('NotificationBell stays hidden when storage is unreadable', async () => {
      breakStorage();
      const { NotificationBell } = await import('../components/notifications/NotificationBell');

      const { container } = render(<NotificationBell />);

      await waitFor(() => expect(storage()).toHaveBeenCalled());
      expect(container).toBeEmptyDOMElement();
    });

    it('Navigation renders for a logged out reader when storage is unreadable', async () => {
      breakStorage();
      const { default: Navigation } = await import('../components/Navigation');

      expect(() => render(<Navigation />)).not.toThrow();
      await waitFor(() => expect(storage()).toHaveBeenCalled());
    });
  });

  describe('Hooks', () => {
    it('useUserSettings finishes loading when storage is unreadable', async () => {
      breakStorage();
      const { useUserSettings } = await import('../hooks/useUserSettings');

      const TestComponent = () => {
        const { isLoading, settings } = useUserSettings();
        return <div>{isLoading ? 'loading' : settings?.isComplete ? 'complete' : 'unknown'}</div>;
      };
      render(<TestComponent />);

      // It assumes settings are complete rather than blocking the reader.
      expect(await screen.findByText('complete')).toBeInTheDocument();
    });

    it('useNotifications mounts when storage is unreadable', async () => {
      breakStorage();
      const { useNotifications } = await import('../hooks/useNotifications');

      const TestComponent = () => {
        const { notifications, loading } = useNotifications();
        return <div>Notifications: {loading ? 'loading' : notifications.length}</div>;
      };

      expect(() => render(<TestComponent />)).not.toThrow();
      expect(screen.getByText('Notifications: 0')).toBeInTheDocument();
    });
  });

  describe('Imports', () => {
    it('components that read storage can be imported', async () => {
      const modules = [
        () => import('../components/auth/AuthGuard'),
        () => import('../components/Navigation'),
        () => import('../components/notifications/NotificationBell'),
        () => import('../components/privacy/PrivacyWizard'),
        () => import('../pages/Profile'),
        () => import('../pages/Dashboard'),
        () => import('../pages/Login'),
      ];

      for (const load of modules) {
        await expect(load()).resolves.toBeDefined();
      }
    });

    it('hooks that read storage can be imported', async () => {
      const hooks = [
        () => import('../hooks/useUserSettings'),
        () => import('../hooks/useNotifications'),
        () => import('../hooks/useWebSocket'),
      ];

      for (const load of hooks) {
        await expect(load()).resolves.toBeDefined();
      }
    });
  });
});
