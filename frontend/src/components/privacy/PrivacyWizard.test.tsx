import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import PrivacyWizard from './PrivacyWizard';

// Mock the API module
jest.mock('@/lib/api', () => ({
  updatePrivacySettings: jest.fn(() => Promise.resolve({ updated_at: '2023-01-01T00:00:00Z' })),
  getPrivacySettings: jest.fn(() => Promise.reject(new Error('API not available'))),
}));

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  clear: jest.fn(),
};
Object.defineProperty(window, 'localStorage', { value: localStorageMock });

describe('PrivacyWizard', () => {
  beforeEach(() => {
    localStorageMock.getItem.mockClear();
    localStorageMock.setItem.mockClear();
    localStorageMock.clear.mockClear();
  });

  test('renders privacy wizard with correct title', async () => {
    render(
      <PrivacyWizard 
        onComplete={(settings) => {}}
        title="Privacy Setup"
        subtitle="Configure your privacy and safety preferences"
      />
    );
    
    expect(screen.getByText('Privacy Setup')).toBeInTheDocument();
    expect(screen.getByText('Configure your privacy and safety preferences')).toBeInTheDocument();
  });

  test('shows step 1 content filtering options', async () => {
    render(<PrivacyWizard onComplete={(settings) => {}} />);
    
    expect(screen.getByText('Content Filtering')).toBeInTheDocument();
    expect(screen.getByText('Show Explicit Content')).toBeInTheDocument();
    expect(screen.getByText('Show Mature Content')).toBeInTheDocument();
    expect(screen.getByText('Hide Unrated Content')).toBeInTheDocument();
  });

  test('can navigate through all steps', async () => {
    render(<PrivacyWizard onComplete={(settings) => {}} />);
    
    // Start at step 1
    expect(screen.getByText('Content Filtering')).toBeInTheDocument();
    
    // Navigate to step 2
    fireEvent.click(screen.getByText('Continue'));
    await waitFor(() => {
      expect(screen.getByText('Profile Privacy')).toBeInTheDocument();
    });
    
    // Navigate to step 3
    fireEvent.click(screen.getByText('Continue'));
    await waitFor(() => {
      expect(screen.getByText('Reading & Interaction Privacy')).toBeInTheDocument();
    });
    
    // Navigate to step 4
    fireEvent.click(screen.getByText('Continue'));
    await waitFor(() => {
      expect(screen.getByText('Work Posting Defaults')).toBeInTheDocument();
    });
    
    // Navigate to step 5
    fireEvent.click(screen.getByText('Continue'));
    await waitFor(() => {
      expect(screen.getByText('Email & Notifications')).toBeInTheDocument();
    });
  });

  test('can navigate backwards through steps', async () => {
    render(<PrivacyWizard onComplete={(settings) => {}} />);
    
    // Go to step 2
    fireEvent.click(screen.getByText('Continue'));
    await waitFor(() => {
      expect(screen.getByText('Profile Privacy')).toBeInTheDocument();
    });
    
    // Go back to step 1
    fireEvent.click(screen.getByText('Previous'));
    await waitFor(() => {
      expect(screen.getByText('Content Filtering')).toBeInTheDocument();
    });
  });

  test('shows progress indicators correctly', async () => {
    render(<PrivacyWizard onComplete={(settings) => {}} />);
    
    expect(screen.getByText('Step 1 of 5')).toBeInTheDocument();
    expect(screen.getByText('20% Complete')).toBeInTheDocument();
  });

  test('calls onComplete when wizard is finished', async () => {
    const mockOnComplete = jest.fn();
    localStorageMock.getItem.mockReturnValue('fake-auth-token');
    
    render(<PrivacyWizard onComplete={mockOnComplete} />);
    
    // Navigate through all steps
    for (let i = 0; i < 4; i++) {
      fireEvent.click(screen.getByText('Continue'));
      await new Promise(resolve => setTimeout(resolve, 100));
    }
    
    // Complete the wizard
    await waitFor(() => {
      expect(screen.getByText('Complete Setup')).toBeInTheDocument();
    });
    
    fireEvent.click(screen.getByText('Complete Setup'));
    
    await waitFor(() => {
      expect(mockOnComplete).toHaveBeenCalled();
    });
  });

  test('saves settings to localStorage when API is not available', async () => {
    const mockOnComplete = jest.fn();
    localStorageMock.getItem.mockReturnValue(null); // No auth token
    
    render(<PrivacyWizard onComplete={mockOnComplete} />);
    
    // Navigate through all steps quickly
    for (let i = 0; i < 4; i++) {
      fireEvent.click(screen.getByText('Continue'));
      await new Promise(resolve => setTimeout(resolve, 50));
    }
    
    // Complete the wizard
    await waitFor(() => {
      expect(screen.getByText('Complete Setup')).toBeInTheDocument();
    });
    
    fireEvent.click(screen.getByText('Complete Setup'));
    
    await waitFor(() => {
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'user_privacy_settings',
        expect.any(String)
      );
      expect(mockOnComplete).toHaveBeenCalled();
    });
  });

  test('can toggle privacy settings', async () => {
    render(<PrivacyWizard onComplete={(settings) => {}} />);
    
    // Find the toggle for "Show Explicit Content" by getting all checkboxes and selecting the first one
    const toggles = screen.getAllByRole('checkbox');
    const explicitContentToggle = toggles[0]; // First toggle is "Show Explicit Content"
    
    // Should be unchecked by default (safe defaults)
    expect(explicitContentToggle).not.toBeChecked();
    
    // Toggle it on
    fireEvent.click(explicitContentToggle);
    expect(explicitContentToggle).toBeChecked();
  });

  test('shows skip option when enabled', async () => {
    render(<PrivacyWizard onComplete={(settings) => {}} showSkipOption={true} onSkip={() => {}} />);
    
    expect(screen.getByText('Skip for Now')).toBeInTheDocument();
  });

  test('hides skip option when disabled', async () => {
    render(<PrivacyWizard onComplete={(settings) => {}} showSkipOption={false} />);
    
    expect(screen.queryByText('Skip for Now')).not.toBeInTheDocument();
  });
});