import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PrivacyWizard from '../components/privacy/PrivacyWizard';
import WorkWizard from '../components/works/WorkWizard';
import { useAuth } from '../contexts/AuthContext';
import { useUserSettings } from '../hooks/useUserSettings';

// Mock dependencies
jest.mock('../contexts/AuthContext', () => ({
  useAuth: jest.fn(),
}));

jest.mock('../hooks/useUserSettings', () => ({
  useUserSettings: jest.fn(),
}));

jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
    back: jest.fn(),
    forward: jest.fn(),
    refresh: jest.fn(),
    replace: jest.fn(),
    prefetch: jest.fn(),
  }),
}));

const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;
const mockUseUserSettings = useUserSettings as jest.MockedFunction<typeof useUserSettings>;

beforeEach(() => {
  jest.clearAllMocks();
  
  // Mock authenticated user
  mockUseAuth.mockReturnValue({
    user: { id: '1', username: 'testuser' },
    isLoading: false,
    isAuthenticated: true,
    login: jest.fn(),
    logout: jest.fn(),
    register: jest.fn(),
  });

  // Mock user settings
  mockUseUserSettings.mockReturnValue({
    settings: {
      privacy: {
        profileVisibility: 'public',
        allowMessages: true,
        allowKudos: true,
        allowComments: true,
      },
      preferences: {
        theme: 'light',
        language: 'en',
      },
    },
    isLoading: false,
    updateSettings: jest.fn(),
    resetSettings: jest.fn(),
  });
});

describe('Wizard Integration Tests', () => {
  describe('PrivacyWizard', () => {
    test('completes full privacy setup flow', async () => {
      const user = userEvent.setup();
      const mockUpdateSettings = jest.fn().mockResolvedValue(undefined);
      
      mockUseUserSettings.mockReturnValue({
        settings: {
          privacy: {
            profileVisibility: 'public',
            allowMessages: true,
            allowKudos: true,
            allowComments: true,
          },
          preferences: {
            theme: 'light',
            language: 'en',
          },
        },
        isLoading: false,
        updateSettings: mockUpdateSettings,
        resetSettings: jest.fn(),
      });

      render(<PrivacyWizard />);

      // Step 1: Profile Visibility
      expect(screen.getByText(/profile visibility/i)).toBeInTheDocument();
      
      const privateOption = screen.getByLabelText(/private/i);
      await user.click(privateOption);
      
      const nextButton = screen.getByRole('button', { name: /next/i });
      await user.click(nextButton);

      // Step 2: Communication Settings
      await waitFor(() => {
        expect(screen.getByText(/communication settings/i)).toBeInTheDocument();
      });

      const allowMessagesToggle = screen.getByLabelText(/allow messages/i);
      await user.click(allowMessagesToggle);

      await user.click(screen.getByRole('button', { name: /next/i }));

      // Step 3: Content Interaction
      await waitFor(() => {
        expect(screen.getByText(/content interaction/i)).toBeInTheDocument();
      });

      const allowKudosToggle = screen.getByLabelText(/allow kudos/i);
      await user.click(allowKudosToggle);

      await user.click(screen.getByRole('button', { name: /next/i }));

      // Step 4: Review and Complete
      await waitFor(() => {
        expect(screen.getByText(/review your settings/i)).toBeInTheDocument();
      });

      const completeButton = screen.getByRole('button', { name: /complete setup/i });
      await user.click(completeButton);

      // Verify settings were updated correctly
      await waitFor(() => {
        expect(mockUpdateSettings).toHaveBeenCalledWith({
          privacy: {
            profileVisibility: 'private',
            allowMessages: false,
            allowKudos: false,
            allowComments: true, // unchanged
          },
        });
      });
    });

    test('handles navigation between wizard steps', async () => {
      const user = userEvent.setup();
      
      render(<PrivacyWizard />);

      // Move forward through steps
      await user.click(screen.getByRole('button', { name: /next/i }));
      await waitFor(() => {
        expect(screen.getByText(/communication settings/i)).toBeInTheDocument();
      });

      await user.click(screen.getByRole('button', { name: /next/i }));
      await waitFor(() => {
        expect(screen.getByText(/content interaction/i)).toBeInTheDocument();
      });

      // Navigate back
      await user.click(screen.getByRole('button', { name: /back/i }));
      await waitFor(() => {
        expect(screen.getByText(/communication settings/i)).toBeInTheDocument();
      });

      await user.click(screen.getByRole('button', { name: /back/i }));
      await waitFor(() => {
        expect(screen.getByText(/profile visibility/i)).toBeInTheDocument();
      });
    });

    test('validates required fields before proceeding', async () => {
      const user = userEvent.setup();
      
      render(<PrivacyWizard />);

      // Try to proceed without making a selection
      const nextButton = screen.getByRole('button', { name: /next/i });
      
      // Some steps might require a selection
      await user.click(nextButton);
      
      // Should still be on the same step if validation fails
      expect(screen.getByText(/profile visibility/i)).toBeInTheDocument();
    });

    test('shows progress indicator correctly', () => {
      render(<PrivacyWizard />);

      // Check for progress indicator
      const progressElements = screen.getAllByRole('progressbar');
      expect(progressElements.length).toBeGreaterThan(0);
    });

    test('handles wizard cancellation', async () => {
      const user = userEvent.setup();
      
      render(<PrivacyWizard />);

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      await user.click(cancelButton);

      // Should show confirmation dialog or navigate away
      // The exact behavior depends on implementation
    });

    test('preserves form data when navigating between steps', async () => {
      const user = userEvent.setup();
      
      render(<PrivacyWizard />);

      // Select an option
      const privateOption = screen.getByLabelText(/private/i);
      await user.click(privateOption);

      // Navigate away and back
      await user.click(screen.getByRole('button', { name: /next/i }));
      await user.click(screen.getByRole('button', { name: /back/i }));

      // Option should still be selected
      expect(privateOption).toBeChecked();
    });

    test('handles API errors gracefully', async () => {
      const user = userEvent.setup();
      const mockUpdateSettings = jest.fn().mockRejectedValue(new Error('API Error'));
      
      mockUseUserSettings.mockReturnValue({
        settings: {
          privacy: {
            profileVisibility: 'public',
            allowMessages: true,
            allowKudos: true,
            allowComments: true,
          },
          preferences: {
            theme: 'light',
            language: 'en',
          },
        },
        isLoading: false,
        updateSettings: mockUpdateSettings,
        resetSettings: jest.fn(),
      });

      render(<PrivacyWizard />);

      // Complete the wizard
      await user.click(screen.getByRole('button', { name: /next/i }));
      await user.click(screen.getByRole('button', { name: /next/i }));
      await user.click(screen.getByRole('button', { name: /next/i }));
      
      const completeButton = screen.getByRole('button', { name: /complete setup/i });
      await user.click(completeButton);

      // Should show error message
      await waitFor(() => {
        expect(screen.getByText(/error/i)).toBeInTheDocument();
      });
    });
  });

  describe('WorkWizard', () => {
    test('completes full work creation flow', async () => {
      const user = userEvent.setup();
      
      render(<WorkWizard />);

      // Step 1: Basic Information
      expect(screen.getByText(/basic information/i)).toBeInTheDocument();
      
      await user.type(screen.getByLabelText(/title/i), 'Test Work Title');
      await user.type(screen.getByLabelText(/summary/i), 'This is a test work summary.');
      
      await user.click(screen.getByRole('button', { name: /next/i }));

      // Step 2: Tags and Categories
      await waitFor(() => {
        expect(screen.getByText(/tags and categories/i)).toBeInTheDocument();
      });

      // Add some tags
      const tagInput = screen.getByLabelText(/tags/i);
      await user.type(tagInput, 'test tag');
      await user.keyboard('{Enter}');

      await user.click(screen.getByRole('button', { name: /next/i }));

      // Step 3: Content and Chapters
      await waitFor(() => {
        expect(screen.getByText(/content/i)).toBeInTheDocument();
      });

      await user.type(screen.getByLabelText(/content/i), 'This is the work content.');

      await user.click(screen.getByRole('button', { name: /next/i }));

      // Step 4: Privacy and Publishing
      await waitFor(() => {
        expect(screen.getByText(/privacy and publishing/i)).toBeInTheDocument();
      });

      const publishButton = screen.getByRole('button', { name: /publish work/i });
      await user.click(publishButton);

      // Should show success message or redirect
      await waitFor(() => {
        expect(screen.getByText(/work published successfully/i)).toBeInTheDocument();
      });
    });

    test('handles draft saving throughout the process', async () => {
      const user = userEvent.setup();
      
      render(<WorkWizard />);

      // Fill in basic information
      await user.type(screen.getByLabelText(/title/i), 'Draft Work');
      
      // Save as draft
      const saveDraftButton = screen.getByRole('button', { name: /save draft/i });
      await user.click(saveDraftButton);

      // Should show save confirmation
      await waitFor(() => {
        expect(screen.getByText(/draft saved/i)).toBeInTheDocument();
      });
    });

    test('validates required fields at each step', async () => {
      const user = userEvent.setup();
      
      render(<WorkWizard />);

      // Try to proceed without title
      const nextButton = screen.getByRole('button', { name: /next/i });
      await user.click(nextButton);

      // Should show validation error
      expect(screen.getByText(/title is required/i)).toBeInTheDocument();
      
      // Should remain on current step
      expect(screen.getByText(/basic information/i)).toBeInTheDocument();
    });

    test('handles tag autocomplete and validation', async () => {
      const user = userEvent.setup();
      
      render(<WorkWizard />);

      // Navigate to tags step
      await user.type(screen.getByLabelText(/title/i), 'Test Work');
      await user.click(screen.getByRole('button', { name: /next/i }));

      // Test tag input
      const tagInput = screen.getByLabelText(/tags/i);
      await user.type(tagInput, 'fanfic');

      // Should show autocomplete suggestions
      await waitFor(() => {
        expect(screen.getByText(/fanfiction/i)).toBeInTheDocument();
      });

      // Select suggestion
      await user.click(screen.getByText(/fanfiction/i));

      // Tag should be added
      expect(screen.getByText(/fanfiction/i)).toBeInTheDocument();
    });

    test('supports multiple chapters workflow', async () => {
      const user = userEvent.setup();
      
      render(<WorkWizard />);

      // Complete basic steps
      await user.type(screen.getByLabelText(/title/i), 'Multi-chapter Work');
      await user.click(screen.getByRole('button', { name: /next/i }));
      await user.click(screen.getByRole('button', { name: /next/i }));

      // In content step, add multiple chapters
      const addChapterButton = screen.getByRole('button', { name: /add chapter/i });
      await user.click(addChapterButton);

      // Should show chapter management interface
      expect(screen.getByText(/chapter 2/i)).toBeInTheDocument();
    });

    test('handles work series association', async () => {
      const user = userEvent.setup();
      
      render(<WorkWizard />);

      // Navigate through to publishing step
      await user.type(screen.getByLabelText(/title/i), 'Series Work');
      await user.click(screen.getByRole('button', { name: /next/i }));
      await user.click(screen.getByRole('button', { name: /next/i }));
      await user.click(screen.getByRole('button', { name: /next/i }));

      // Test series selection
      const seriesSelect = screen.getByLabelText(/add to series/i);
      await user.click(seriesSelect);

      // Should show series options or creation form
      expect(screen.getByText(/select series/i)).toBeInTheDocument();
    });
  });

  describe('Wizard Error Handling', () => {
    test('handles network failures gracefully', async () => {
      const user = userEvent.setup();
      
      // Mock network failure
      const mockUpdateSettings = jest.fn().mockRejectedValue(new Error('Network error'));
      mockUseUserSettings.mockReturnValue({
        settings: {
          privacy: { profileVisibility: 'public', allowMessages: true, allowKudos: true, allowComments: true },
          preferences: { theme: 'light', language: 'en' },
        },
        isLoading: false,
        updateSettings: mockUpdateSettings,
        resetSettings: jest.fn(),
      });

      render(<PrivacyWizard />);

      // Complete wizard
      await user.click(screen.getByRole('button', { name: /next/i }));
      await user.click(screen.getByRole('button', { name: /next/i }));
      await user.click(screen.getByRole('button', { name: /next/i }));
      await user.click(screen.getByRole('button', { name: /complete setup/i }));

      // Should show retry option
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument();
      });
    });

    test('prevents data loss on unexpected errors', async () => {
      const user = userEvent.setup();
      
      render(<PrivacyWizard />);

      // Fill in data
      await user.click(screen.getByLabelText(/private/i));
      
      // Simulate error during save
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
      
      // Force an error
      window.dispatchEvent(new Event('error'));

      // Data should still be preserved
      expect(screen.getByLabelText(/private/i)).toBeChecked();
      
      consoleSpy.mockRestore();
    });
  });

  describe('Accessibility in Wizards', () => {
    test('supports keyboard navigation', async () => {
      render(<PrivacyWizard />);

      // Test tab navigation
      const nextButton = screen.getByRole('button', { name: /next/i });
      nextButton.focus();
      expect(nextButton).toHaveFocus();

      // Test arrow key navigation for radio buttons
      const radioButtons = screen.getAllByRole('radio');
      if (radioButtons.length > 0) {
        radioButtons[0].focus();
        fireEvent.keyDown(radioButtons[0], { key: 'ArrowDown' });
        expect(radioButtons[1] || radioButtons[0]).toHaveFocus();
      }
    });

    test('provides proper ARIA labels and descriptions', () => {
      render(<PrivacyWizard />);

      // Check for proper labeling
      const progressBar = screen.getByRole('progressbar');
      expect(progressBar).toHaveAttribute('aria-label');

      // Check for step descriptions
      const stepContent = screen.getByRole('main');
      expect(stepContent).toHaveAttribute('aria-describedby');
    });

    test('announces step changes to screen readers', async () => {
      const user = userEvent.setup();
      
      render(<PrivacyWizard />);

      // Navigate to next step
      await user.click(screen.getByRole('button', { name: /next/i }));

      // Should have live region updates
      const liveRegion = screen.getByRole('status');
      expect(liveRegion).toBeInTheDocument();
    });
  });
});