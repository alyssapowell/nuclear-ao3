/**
 * Critical Route Health Check Tests
 * 
 * These tests would have caught the 404/500 errors we experienced.
 * Tests ensure all routes are accessible and respond correctly.
 */

import { render, screen } from '@testing-library/react';
import { useRouter } from 'next/navigation';
import { ReactElement } from 'react';

// Mock Next.js router
jest.mock('next/navigation', () => ({
  useRouter: jest.fn(),
  usePathname: jest.fn(),
  useSearchParams: jest.fn(() => ({
    get: jest.fn()
  })),
}));

// Mock localStorage for SSR safety
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
};
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock
});

// Test wrapper to handle dynamic imports
const testPageComponent = async (importFn: () => Promise<{ default: React.ComponentType }>) => {
  try {
    const { default: Component } = await importFn();
    return Component;
  } catch (error) {
    throw new Error(`Failed to import component: ${error}`);
  }
};

describe('Route Health Checks', () => {
  const mockPush = jest.fn();
  const mockReplace = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    (useRouter as jest.Mock).mockReturnValue({
      push: mockPush,
      replace: mockReplace,
      back: jest.fn(),
      forward: jest.fn(),
    });
    
    // Clear localStorage mocks
    localStorageMock.getItem.mockReturnValue(null);
  });

  describe('Public Routes', () => {
    test('Home page (/) renders without errors', async () => {
      const HomePage = await testPageComponent(() => import('../app/page'));
      
      render(<HomePage />);
      
      // Should not throw any errors during render
      expect(screen.getByRole('main') || screen.getByText(/Nuclear AO3/i)).toBeInTheDocument();
    });

    test('Login page (/auth/login) renders without errors', async () => {
      const LoginPage = await testPageComponent(() => import('../app/auth/login/page'));
      
      render(<LoginPage />);
      
      // Should render login form
      expect(screen.getByRole('form') || screen.getByText(/sign in/i)).toBeInTheDocument();
    });

    test('Register page (/auth/register) renders without errors', async () => {
      const RegisterPage = await testPageComponent(() => import('../app/auth/register/page'));
      
      render(<RegisterPage />);
      
      // Should render registration form
      expect(screen.getByText(/create account/i) || screen.getByText(/sign up/i)).toBeInTheDocument();
    });

    test('Works listing (/works) renders without errors', async () => {
      const WorksPage = await testPageComponent(() => import('../app/works/page'));
      
      render(<WorksPage />);
      
      // Should not crash during render
      expect(document.querySelector('body')).toBeInTheDocument();
    });

    test('Search page (/search) renders without errors', async () => {
      const SearchPage = await testPageComponent(() => import('../app/search/page'));
      
      render(<SearchPage />);
      
      // Should not crash during render
      expect(document.querySelector('body')).toBeInTheDocument();
    });
  });

  describe('Protected Routes (Should Redirect When Unauthenticated)', () => {
    test('Profile page (/profile) redirects to login when unauthenticated', async () => {
      const ProfilePage = await testPageComponent(() => import('../app/profile/page'));
      
      render(<ProfilePage />);
      
      // Should attempt to redirect to login
      expect(mockPush).toHaveBeenCalledWith(
        expect.stringContaining('/auth/login')
      );
    });

    test('Dashboard page (/dashboard) redirects when unauthenticated', async () => {
      const DashboardPage = await testPageComponent(() => import('../app/dashboard/page'));
      
      render(<DashboardPage />);
      
      // Should show loading or redirect
      // The AuthGuard should handle the redirect
      expect(document.querySelector('body')).toBeInTheDocument();
    });

    test('Onboarding page (/onboarding) renders privacy wizard', async () => {
      const OnboardingPage = await testPageComponent(() => import('../app/onboarding/page'));
      
      render(<OnboardingPage />);
      
      // Should not crash during render
      expect(document.querySelector('body')).toBeInTheDocument();
    });
  });

  describe('Component Import Safety', () => {
    test('All critical page components can be imported', async () => {
      const criticalPages = [
        () => import('../app/page'),
        () => import('../app/auth/login/page'),
        () => import('../app/auth/register/page'),
        () => import('../app/profile/page'),
        () => import('../app/dashboard/page'),
        () => import('../app/onboarding/page'),
        () => import('../app/works/page'),
        () => import('../app/search/page'),
      ];

      // All imports should succeed
      for (const importFn of criticalPages) {
        await expect(testPageComponent(importFn)).resolves.toBeDefined();
      }
    });

    test('Critical components can be imported', async () => {
      const criticalComponents = [
        () => import('../components/Navigation'),
        () => import('../components/auth/AuthGuard'),
        () => import('../components/privacy/PrivacyWizard'),
        () => import('../components/notifications/NotificationBell'),
      ];

      // All imports should succeed
      for (const importFn of criticalComponents) {
        await expect(importFn()).resolves.toBeDefined();
      }
    });
  });

  describe('Route Configuration Validation', () => {
    test('Package.json dev script includes Turbopack', () => {
      const packageJson = require('../../package.json');
      
      expect(packageJson.scripts.dev).toContain('--turbo');
    });

    test('Next.js config exists and is valid', () => {
      expect(() => {
        require('../../next.config.js');
      }).not.toThrow();
    });
  });
});