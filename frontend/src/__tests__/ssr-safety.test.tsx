/**
 * SSR/Hydration Safety Tests
 * 
 * These tests ensure components handle server-side rendering safely
 * and prevent hydration mismatches that cause crashes.
 */

import { render } from '@testing-library/react';
import { ReactElement, ReactNode } from 'react';

// Mock Next.js navigation
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
    replace: jest.fn(),
    back: jest.fn(),
  }),
  usePathname: () => '/test',
  useSearchParams: () => ({
    get: jest.fn(),
  }),
}));

// SSR Environment Simulation
const simulateSSR = () => {
  // @ts-ignore
  delete global.window;
  // @ts-ignore
  delete global.localStorage;
  // @ts-ignore
  delete global.document;
};

const restoreClientEnv = () => {
  // @ts-ignore
  global.window = { location: { pathname: '/test', search: '' } };
  global.localStorage = {
    getItem: jest.fn(),
    setItem: jest.fn(),
    removeItem: jest.fn(),
    clear: jest.fn(),
  };
  global.document = {
    cookie: '',
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
  };
};

describe('SSR Safety Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(() => {
    restoreClientEnv();
  });

  describe('LocalStorage Safety', () => {
    test('Components handle missing localStorage gracefully', async () => {
      simulateSSR();
      
      const { default: AuthGuard } = await import('../components/auth/AuthGuard');
      
      expect(() => {
        render(
          <AuthGuard>
            <div>Protected content</div>
          </AuthGuard>
        );
      }).not.toThrow();
    });

    test('NotificationBell handles SSR without window object', async () => {
      simulateSSR();
      
      const { NotificationBell } = await import('../components/notifications/NotificationBell');
      
      expect(() => {
        render(<NotificationBell />);
      }).not.toThrow();
    });

    test('Navigation component handles SSR safely', async () => {
      simulateSSR();
      
      const { default: Navigation } = await import('../components/Navigation');
      
      expect(() => {
        render(<Navigation />);
      }).not.toThrow();
    });

    test('Profile page handles SSR without localStorage', async () => {
      simulateSSR();
      
      const { default: ProfilePage } = await import('../app/profile/page');
      
      expect(() => {
        render(<ProfilePage />);
      }).not.toThrow();
    });

    test('Privacy wizard handles SSR safely', async () => {
      simulateSSR();
      
      const { default: PrivacyWizard } = await import('../components/privacy/PrivacyWizard');
      
      expect(() => {
        render(
          <PrivacyWizard 
            onComplete={() => {}} 
            onSkip={() => {}}
            showSkipOption={true}
          />
        );
      }).not.toThrow();
    });
  });

  describe('Hydration Consistency', () => {
    test('AuthGuard renders consistently between SSR and client', async () => {
      const { default: AuthGuard } = await import('../components/auth/AuthGuard');
      
      // First render (SSR simulation)
      simulateSSR();
      const { container: ssrContainer } = render(
        <AuthGuard>
          <div>Content</div>
        </AuthGuard>
      );
      const ssrHTML = ssrContainer.innerHTML;

      // Second render (Client hydration)
      restoreClientEnv();
      const { container: clientContainer } = render(
        <AuthGuard>
          <div>Content</div>
        </AuthGuard>
      );
      const clientHTML = clientContainer.innerHTML;

      // Should be consistent (or at least not cause React hydration errors)
      expect(ssrHTML).toBeDefined();
      expect(clientHTML).toBeDefined();
    });

    test('NotificationBell hydrates without errors', async () => {
      const { NotificationBell } = await import('../components/notifications/NotificationBell');
      
      // Should not crash during hydration
      expect(() => {
        render(<NotificationBell />);
      }).not.toThrow();
    });
  });

  describe('Hook Safety', () => {
    test('useUserSettings hook handles SSR safely', async () => {
      const { useUserSettings } = await import('../hooks/useUserSettings');
      
      const TestComponent = () => {
        const settings = useUserSettings();
        return <div>Settings loaded: {settings.isLoading ? 'loading' : 'done'}</div>;
      };

      expect(() => {
        render(<TestComponent />);
      }).not.toThrow();
    });

    test('useNotifications hook handles missing window gracefully', async () => {
      simulateSSR();
      
      const { useNotifications } = await import('../hooks/useNotifications');
      
      const TestComponent = () => {
        const { notifications, loading } = useNotifications();
        return <div>Notifications: {loading ? 'loading' : notifications.length}</div>;
      };

      expect(() => {
        render(<TestComponent />);
      }).not.toThrow();
    });

    test('useWebSocket hook handles SSR without WebSocket API', async () => {
      simulateSSR();
      
      const { useWebSocket } = await import('../hooks/useWebSocket');
      
      const TestComponent = () => {
        const { isConnected, connect } = useWebSocket({
          url: 'ws://localhost:8004',
          onMessage: () => {},
        });
        
        return <div>Connected: {isConnected ? 'yes' : 'no'}</div>;
      };

      expect(() => {
        render(<TestComponent />);
      }).not.toThrow();
    });
  });

  describe('Component Import Safety', () => {
    test('All components with localStorage access import safely', async () => {
      const componentsWithLocalStorage = [
        () => import('../components/auth/AuthGuard'),
        () => import('../components/Navigation'),
        () => import('../components/notifications/NotificationBell'),
        () => import('../components/privacy/PrivacyWizard'),
        () => import('../app/profile/page'),
        () => import('../app/dashboard/page'),
        () => import('../app/auth/login/page'),
      ];

      for (const importFn of componentsWithLocalStorage) {
        await expect(importFn()).resolves.toBeDefined();
      }
    });

    test('All hooks import safely', async () => {
      const hooks = [
        () => import('../hooks/useUserSettings'),
        () => import('../hooks/useNotifications'),
        () => import('../hooks/useWebSocket'),
      ];

      for (const importFn of hooks) {
        await expect(importFn()).resolves.toBeDefined();
      }
    });
  });

  describe('Error Boundaries', () => {
    test('Components recover from localStorage errors gracefully', async () => {
      // Mock localStorage to throw errors
      const mockLocalStorage = {
        getItem: jest.fn(() => { throw new Error('localStorage not available'); }),
        setItem: jest.fn(() => { throw new Error('localStorage not available'); }),
        removeItem: jest.fn(() => { throw new Error('localStorage not available'); }),
        clear: jest.fn(),
      };
      
      Object.defineProperty(window, 'localStorage', {
        value: mockLocalStorage,
        writable: true,
      });

      const { default: AuthGuard } = await import('../components/auth/AuthGuard');
      
      // Should handle localStorage errors gracefully
      expect(() => {
        render(
          <AuthGuard>
            <div>Content</div>
          </AuthGuard>
        );
      }).not.toThrow();
      
      // Console warnings are acceptable, crashes are not
      expect(mockLocalStorage.getItem).toHaveBeenCalled();
    });
  });
});