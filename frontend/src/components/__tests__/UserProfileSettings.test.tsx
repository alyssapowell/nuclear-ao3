import type { MockedFunction } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import '@testing-library/jest-dom';
import UserProfileSettings from '../UserProfileSettings';
import { getMyProfile, updateProfile, getPseudonyms, createPseudonym } from '@/lib/api';

// Mock the API functions
vi.mock('@/lib/api', () => ({
  getMyProfile: vi.fn(),
  updateProfile: vi.fn(),
  getPseudonyms: vi.fn(),
  createPseudonym: vi.fn(),
}));

const mockGetMyProfile = getMyProfile as MockedFunction<typeof getMyProfile>;
const mockUpdateProfile = updateProfile as MockedFunction<typeof updateProfile>;
const mockGetPseudonyms = getPseudonyms as MockedFunction<typeof getPseudonyms>;
const mockCreatePseudonym = createPseudonym as MockedFunction<typeof createPseudonym>;

// Helper function to render UserProfileSettings component wrapped in act
const renderUserProfileWithAct = async (props: any) => {
  let result: any;
  await act(async () => {
    result = render(<UserProfileSettings {...props} />);
  });
  return result;
};

describe('UserProfileSettings', () => {
  const mockUser = {
    user_id: 'user-1',
    username: 'testuser',
    display_name: 'Test User',
    bio: 'Test bio',
    location: 'Test City',
    website: 'https://test.com',
    email_notifications: true,
    show_adult_content: false,
    allow_friend_requests: true,
    show_stats: true,
    show_works: true,
    show_bookmarks: true,
    default_work_privacy: 'public' as const,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  };

  const mockPseudonyms = [
    {
      id: 'pseudo-1',
      user_id: 'user-1',
      name: 'TestPseudo',
      description: 'Test pseudonym',
      is_default: true,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    mockGetMyProfile.mockResolvedValue({ user: mockUser });
    mockGetPseudonyms.mockResolvedValue({ pseudonyms: mockPseudonyms });
    mockUpdateProfile.mockResolvedValue({ success: true });
    mockCreatePseudonym.mockResolvedValue({ 
      pseudonym: {
        id: 'pseudo-2',
        user_id: 'user-1',
        name: 'NewPseudo',
        description: 'New pseudonym',
        is_default: false,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }
    });
  });

  it('renders profile settings tabs', async () => {
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      expect(screen.getByText('Profile Settings')).toBeInTheDocument();
      expect(screen.getByText('Profile')).toBeInTheDocument();
      expect(screen.getByText('Privacy')).toBeInTheDocument();
      expect(screen.getByText('Pseudonyms')).toBeInTheDocument();
    });
  });

  it('loads and displays user profile data', async () => {
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      expect(screen.getByDisplayValue('testuser')).toBeInTheDocument();
      expect(screen.getByDisplayValue('Test User')).toBeInTheDocument();
      expect(screen.getByDisplayValue('Test bio')).toBeInTheDocument();
      expect(screen.getByDisplayValue('Test City')).toBeInTheDocument();
      expect(screen.getByDisplayValue('https://test.com')).toBeInTheDocument();
    });
  });

  it('shows loading state initially', async () => {
    // Delay the mock to see loading state
    mockGetMyProfile.mockImplementation(() => new Promise(resolve => setTimeout(() => resolve({ user: mockUser }), 100)));
    
    await renderUserProfileWithAct({ authToken: "test-token" });
    
    expect(screen.getByText('Loading profile settings...')).toBeInTheDocument();
  });

  it('updates profile when form is submitted', async () => {
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      expect(screen.getByDisplayValue('Test User')).toBeInTheDocument();
    });

    const displayNameInput = screen.getByDisplayValue('Test User');
    const saveButton = screen.getByText('Save Changes');

    fireEvent.change(displayNameInput, { target: { value: 'Updated Name' } });
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateProfile).toHaveBeenCalledWith(
        expect.objectContaining({
          display_name: 'Updated Name',
        }),
        'test-token'
      );
    });
  });

  it('switches to privacy tab and shows privacy settings', async () => {
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      expect(screen.getByText('Profile Settings')).toBeInTheDocument();
    });

    const privacyTab = screen.getByRole('tab', { name: /Privacy/i });
    fireEvent.click(privacyTab);

    expect(screen.getByText('Privacy Settings')).toBeInTheDocument();
    expect(screen.getByText('Show Adult Content')).toBeInTheDocument();
    expect(screen.getByText('Allow Friend Requests')).toBeInTheDocument();
    expect(screen.getByText('Show Statistics')).toBeInTheDocument();
  });

  it('toggles privacy settings', async () => {
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      expect(screen.getByText('Profile Settings')).toBeInTheDocument();
    });

    // Switch to privacy tab
    const privacyTab = screen.getByRole('tab', { name: /Privacy/i });
    fireEvent.click(privacyTab);

    // Find and toggle adult content switch - it's the first checkbox (unchecked)
    const adultContentSwitch = screen.getAllByRole('checkbox')[0];
    fireEvent.click(adultContentSwitch);

    // Save changes
    const saveButton = screen.getByText('Save Privacy Settings');
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateProfile).toHaveBeenCalledWith(
        expect.objectContaining({
          show_adult_content: true, // Should be toggled to true
        }),
        'test-token'
      );
    });
  });

  it('switches to pseudonyms tab and shows pseudonyms', async () => {
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      expect(screen.getByText('Profile Settings')).toBeInTheDocument();
    });

    const pseudonymsTab = screen.getByRole('tab', { name: /Pseudonyms/i });
    fireEvent.click(pseudonymsTab);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Pseudonyms' })).toBeInTheDocument();
      expect(screen.getByText('TestPseudo')).toBeInTheDocument();
      expect(screen.getByText('Default')).toBeInTheDocument();
    });
  });

  it('shows add pseudonym form when button is clicked', async () => {
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      expect(screen.getByText('Profile Settings')).toBeInTheDocument();
    });

    // Switch to pseudonyms tab
    const pseudonymsTab = screen.getByRole('tab', { name: /Pseudonyms/i });
    fireEvent.click(pseudonymsTab);

    // Click add pseudonym button
    const addButton = screen.getByText('Add Pseudonym');
    fireEvent.click(addButton);

    expect(screen.getByText('Create New Pseudonym')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Enter pseudonym name')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Optional description for this pseudonym')).toBeInTheDocument();
  });

  it('creates a new pseudonym', async () => {
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      expect(screen.getByText('Profile Settings')).toBeInTheDocument();
    });

    // Switch to pseudonyms tab
    const pseudonymsTab = screen.getByRole('tab', { name: /Pseudonyms/i });
    fireEvent.click(pseudonymsTab);

    // Click add pseudonym button
    const addButton = screen.getByText('Add Pseudonym');
    fireEvent.click(addButton);

    // Fill in form
    const nameInput = screen.getByPlaceholderText('Enter pseudonym name');
    const descriptionInput = screen.getByPlaceholderText('Optional description for this pseudonym');
    
    fireEvent.change(nameInput, { target: { value: 'NewPseudo' } });
    fireEvent.change(descriptionInput, { target: { value: 'New pseudonym description' } });

    // Submit form
    const createButton = screen.getByText('Create Pseudonym');
    fireEvent.click(createButton);

    await waitFor(() => {
      expect(mockCreatePseudonym).toHaveBeenCalledWith(
        {
          name: 'NewPseudo',
          description: 'New pseudonym description',
        },
        'test-token'
      );
    });
  });

  it('shows error when pseudonym name is empty', async () => {
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      expect(screen.getByText('Profile Settings')).toBeInTheDocument();
    });

    // Switch to pseudonyms tab
    const pseudonymsTab = screen.getByRole('tab', { name: /Pseudonyms/i });
    fireEvent.click(pseudonymsTab);

    // Click add pseudonym button
    const addButton = screen.getByText('Add Pseudonym');
    fireEvent.click(addButton);

    // The button is disabled while the name is empty, so submit the form itself.
    const createButton = screen.getByText('Create Pseudonym');
    expect(createButton).toBeDisabled();
    fireEvent.submit(createButton.closest('form')!);

    await waitFor(() => {
      expect(screen.getByText('Pseudonym name is required')).toBeInTheDocument();
    });
  });

  it('handles profile loading error', async () => {
    mockGetMyProfile.mockRejectedValue(new Error('Failed to load profile'));
    
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      expect(screen.getByText('Failed to load profile')).toBeInTheDocument();
    });
  });

  it('shows success message after saving', async () => {
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      expect(screen.getByDisplayValue('Test User')).toBeInTheDocument();
    });

    const saveButton = screen.getByText('Save Changes');
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(screen.getByRole('status')).toHaveTextContent('Profile updated successfully!');
    });
  });

  it('handles update profile error', async () => {
    mockUpdateProfile.mockRejectedValue(new Error('Update failed'));
    
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      expect(screen.getByDisplayValue('Test User')).toBeInTheDocument();
    });

    const saveButton = screen.getByText('Save Changes');
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(screen.getByText('Update failed')).toBeInTheDocument();
    });
  });

  it('shows disabled username field', async () => {
    await renderUserProfileWithAct({ authToken: "test-token" });

    await waitFor(() => {
      const usernameInput = screen.getByDisplayValue('testuser');
      expect(usernameInput).toBeDisabled();
      expect(screen.getByText('Username cannot be changed')).toBeInTheDocument();
    });
  });

  it('renders without auth token gracefully', async () => {
    render(<UserProfileSettings />);
    
    // Should not make API calls without auth token
    expect(mockGetMyProfile).not.toHaveBeenCalled();
    expect(mockGetPseudonyms).not.toHaveBeenCalled();
  });
});