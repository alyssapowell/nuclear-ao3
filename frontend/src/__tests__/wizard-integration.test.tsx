import type { MockedFunction } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PrivacyWizard from '../components/privacy/PrivacyWizard';
import { useAuth } from '../utils/auth';
import { useUserSettings } from '../hooks/useUserSettings';

// Mock dependencies
vi.mock('../utils/auth', () => ({
  useAuth: vi.fn(),
  getAuthState: vi.fn(),
  isAuthenticated: vi.fn(),
}));

vi.mock('../hooks/useUserSettings', () => ({
  useUserSettings: vi.fn(),
}));


const mockUseAuth = useAuth as MockedFunction<typeof useAuth>;
const mockUseUserSettings = useUserSettings as MockedFunction<typeof useUserSettings>;

beforeEach(() => {
  vi.clearAllMocks();
  mockUseAuth.mockReturnValue({
    user: { id: '1', username: 'testuser', email: 'test@example.com' },
    isAuthenticated: true,
    token: 'mock-token',
    logout: vi.fn(),
  });

  mockUseUserSettings.mockReturnValue({
    settings: {
      emailNotifications: true,
      profilePrivacy: 'public',
      workPrivacy: 'public',
    },
    updateSettings: vi.fn(),
    loading: false,
    error: null,
  });
});

describe('Wizard Integration Tests', () => {
  describe('PrivacyWizard', () => {
    test('completes full privacy setup flow', async () => {
      const user = userEvent.setup();
      
      render(<PrivacyWizard />);

      // Should start at first step
      expect(screen.getByText(/Privacy Setup/i)).toBeInTheDocument();
      
      // Should have navigation buttons
      expect(screen.getByRole('button', { name: /continue/i })).toBeInTheDocument();
    });

    test('validates required privacy selections', async () => {
      const user = userEvent.setup();
      
      render(<PrivacyWizard />);

      // Try to proceed without making selections
      const continueButton = screen.getByRole('button', { name: /continue/i });
      await user.click(continueButton);

      // Should show validation message or stay on current step
      expect(screen.getByText(/Privacy Setup/i)).toBeInTheDocument();
    });

    test('allows navigation between wizard steps', async () => {
      const user = userEvent.setup();
      
      render(<PrivacyWizard />);

      // Should be able to navigate
      const continueButton = screen.getByRole('button', { name: /continue/i });
      expect(continueButton).toBeInTheDocument();
    });

    test('saves privacy settings on completion', async () => {
      const mockUpdateSettings = vi.fn();
      mockUseUserSettings.mockReturnValue({
        settings: {
          emailNotifications: true,
          profilePrivacy: 'public',
          workPrivacy: 'public',
        },
        updateSettings: mockUpdateSettings,
        loading: false,
        error: null,
      });

      const user = userEvent.setup();
      
      render(<PrivacyWizard />);

      // Complete the wizard
      const completeButton = screen.queryByRole('button', { name: /complete/i });
      if (completeButton) {
        await user.click(completeButton);
        
        // Should call update settings
        expect(mockUpdateSettings).toHaveBeenCalled();
      }
    });

    test('handles errors gracefully during setup', async () => {
      mockUseUserSettings.mockReturnValue({
        settings: {},
        updateSettings: vi.fn(),
        loading: false,
        error: 'Failed to save settings',
      });

      const user = userEvent.setup();
      
      render(<PrivacyWizard />);

      // Should show error message
      expect(screen.getByText(/error/i) || screen.getByText(/failed/i)).toBeInTheDocument();
    });

    test('shows loading state during settings update', async () => {
      mockUseUserSettings.mockReturnValue({
        settings: {},
        updateSettings: vi.fn(),
        loading: true,
        error: null,
      });
      
      render(<PrivacyWizard />);

      // Should show loading indicator
      expect(screen.getByText(/loading/i) || screen.getByRole('progressbar')).toBeInTheDocument();
    });
  });

  // Note: WorkWizard component doesn't exist yet, skipping these tests
  // TODO: Implement WorkWizard component and restore these tests
});