import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach, afterEach, beforeAll } from 'vitest';
import { usePathname } from 'next/navigation';
import MobileNavigation from '../../components/MobileNavigation';

// Import our test utilities and mocks
import { mockServiceWorkerRegistration, mockPWAEvents } from '../test-utils';

// Mock Next.js navigation
vi.mock('next/navigation', () => ({
  usePathname: vi.fn(),
}));

// Mock the PWA hook
const mockPWAState = {
  isInstalled: false,
  isInstallable: false,
  updateAvailable: false,
  isOnline: true,
};

const mockPWAActions = {
  installPWA: vi.fn(),
  dismissInstallPrompt: vi.fn(),
  skipWaiting: vi.fn(),
};

vi.mock('../../hooks/usePWAState', () => ({
  usePWAState: () => [mockPWAState, mockPWAActions],
}));

// Mock the offline reading manager
const mockOfflineManager = {
  getOfflineWorks: vi.fn(),
  onMessage: vi.fn(),
  offMessage: vi.fn(),
};

vi.mock('../../utils/offlineReadingManager', () => ({
  useOfflineReading: () => mockOfflineManager,
}));

// Mock OfflineReadingManager component
vi.mock('../../components/OfflineReadingManager', () => ({
  default: ({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) => (
    isOpen ? (
      <div data-testid="offline-reading-manager">
        <button onClick={onClose}>Close Manager</button>
      </div>
    ) : null
  ),
}));

describe('MobileNavigation', () => {
  const mockUser = {
    id: 'user-1',
    username: 'testuser',
    avatar: '/avatar.jpg',
  };

  beforeAll(() => {
    // Set up service worker mock
    Object.defineProperty(navigator, 'serviceWorker', {
      value: mockServiceWorkerRegistration,
      configurable: true,
    });
  });

  beforeEach(() => {
    vi.clearAllMocks();
    (usePathname as any).mockReturnValue('/');
    
    mockOfflineManager.getOfflineWorks.mockResolvedValue([]);
    
    // Reset PWA state
    Object.assign(mockPWAState, {
      isInstalled: false,
      isInstallable: false,
      updateAvailable: false,
      isOnline: true,
    });
  });

  afterEach(() => {
    vi.clearAllTimers();
  });

  describe('Basic Rendering', () => {
    it('renders top status bar and bottom navigation', () => {
      render(<MobileNavigation />);
      
      expect(screen.getByText('Nuclear AO3')).toBeInTheDocument();
      expect(screen.getByText('Online')).toBeInTheDocument();
      expect(screen.getByText('Home')).toBeInTheDocument();
      expect(screen.getByText('Browse')).toBeInTheDocument();
      expect(screen.getByText('Bookmarks')).toBeInTheDocument();
      expect(screen.getByText('Profile')).toBeInTheDocument();
    });

    it('shows user information when user is provided', () => {
      render(<MobileNavigation user={mockUser} />);
      
      // Open menu to see user info
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      expect(screen.getByText('Welcome, testuser')).toBeInTheDocument();
    });

    it('shows sign in buttons when no user is provided', () => {
      render(<MobileNavigation />);
      
      // Open menu to see sign in options
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      expect(screen.getByText('Sign In')).toBeInTheDocument();
      expect(screen.getByText('Create Account')).toBeInTheDocument();
    });
  });

  describe('Navigation States', () => {
    it('highlights active navigation item', () => {
      (usePathname as any).mockReturnValue('/search');
      
      render(<MobileNavigation />);
      
      const browseItem = screen.getByText('Browse').closest('a');
      expect(browseItem).toHaveClass('text-blue-600', 'bg-blue-50');
    });

    it('shows correct navigation state for home page', () => {
      (usePathname as any).mockReturnValue('/');
      
      render(<MobileNavigation />);
      
      const homeItem = screen.getByText('Home').closest('a');
      expect(homeItem).toHaveClass('text-blue-600', 'bg-blue-50');
    });

    it('shows correct navigation state for nested paths', () => {
      (usePathname as any).mockReturnValue('/dashboard/works');
      
      render(<MobileNavigation />);
      
      const profileItem = screen.getByText('Profile').closest('a');
      expect(profileItem).toHaveClass('text-blue-600', 'bg-blue-50');
    });
  });

  describe('Online/Offline Status', () => {
    it('shows online status correctly', () => {
      mockPWAState.isOnline = true;
      
      render(<MobileNavigation />);
      
      expect(screen.getByText('Online')).toBeInTheDocument();
    });

    it('shows offline status correctly', () => {
      mockPWAState.isOnline = false;
      
      render(<MobileNavigation />);
      
      expect(screen.getByText('Offline')).toBeInTheDocument();
    });
  });

  describe('Offline Works Integration', () => {
    it('loads offline works count on mount', async () => {
      mockOfflineManager.getOfflineWorks.mockResolvedValue([
        { workId: 'work-1', isExpired: false },
        { workId: 'work-2', isExpired: false },
        { workId: 'work-3', isExpired: true }, // Expired work should not count
      ]);
      
      render(<MobileNavigation />);
      
      await waitFor(() => {
        expect(mockOfflineManager.getOfflineWorks).toHaveBeenCalled();
      });
      
      expect(screen.getByText('2 works offline')).toBeInTheDocument();
    });

    it('does not show offline works count when none are cached', async () => {
      mockOfflineManager.getOfflineWorks.mockResolvedValue([]);
      
      render(<MobileNavigation />);
      
      await waitFor(() => {
        expect(mockOfflineManager.getOfflineWorks).toHaveBeenCalled();
      });
      
      expect(screen.queryByText(/works offline/)).not.toBeInTheDocument();
    });

    it('shows offline works section in menu when works are cached', async () => {
      mockOfflineManager.getOfflineWorks.mockResolvedValue([
        { workId: 'work-1', isExpired: false },
        { workId: 'work-2', isExpired: false },
      ]);
      
      render(<MobileNavigation />);
      
      await waitFor(() => {
        expect(mockOfflineManager.getOfflineWorks).toHaveBeenCalled();
      });
      
      // Open menu
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      expect(screen.getByText('Offline Works')).toBeInTheDocument();
      expect(screen.getByText('2 works available')).toBeInTheDocument();
    });

    it('opens offline reading manager when offline works button is clicked', async () => {
      mockOfflineManager.getOfflineWorks.mockResolvedValue([
        { workId: 'work-1', isExpired: false },
      ]);
      
      render(<MobileNavigation />);
      
      await waitFor(() => {
        expect(screen.getByText('1 work offline')).toBeInTheDocument();
      });
      
      fireEvent.click(screen.getByText('1 work offline'));
      
      expect(screen.getByTestId('offline-reading-manager')).toBeInTheDocument();
    });
  });

  describe('Menu Functionality', () => {
    it('opens and closes menu correctly', () => {
      render(<MobileNavigation />);
      
      // Menu should not be visible initially
      expect(screen.queryByText('Menu')).not.toBeInTheDocument();
      
      // Open menu
      fireEvent.click(screen.getByLabelText('Open menu'));
      expect(screen.getByText('Menu')).toBeInTheDocument();
      
      // Close menu
      fireEvent.click(screen.getByRole('button', { name: /close/i }));
      expect(screen.queryByText('Menu')).not.toBeInTheDocument();
    });

    it('closes menu when backdrop is clicked', () => {
      render(<MobileNavigation />);
      
      // Open menu
      fireEvent.click(screen.getByLabelText('Open menu'));
      expect(screen.getByText('Menu')).toBeInTheDocument();
      
      // Click backdrop
      const backdrop = screen.getByText('Menu').closest('.fixed')?.querySelector('.absolute.inset-0');
      fireEvent.click(backdrop!);
      
      expect(screen.queryByText('Menu')).not.toBeInTheDocument();
    });

    it('shows menu items based on authentication status', () => {
      render(<MobileNavigation user={mockUser} />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      expect(screen.getByText('My Works')).toBeInTheDocument();
      expect(screen.getByText('Reading History')).toBeInTheDocument();
      expect(screen.getByText('Settings')).toBeInTheDocument();
    });

    it('hides auth-required menu items when not authenticated', () => {
      render(<MobileNavigation />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      expect(screen.queryByText('My Works')).not.toBeInTheDocument();
      expect(screen.queryByText('Reading History')).not.toBeInTheDocument();
      expect(screen.queryByText('Settings')).not.toBeInTheDocument();
      
      // But should show non-auth items
      expect(screen.getByText('Collections')).toBeInTheDocument();
      expect(screen.getByText('Help')).toBeInTheDocument();
    });
  });

  describe('PWA Install Banner', () => {
    it('shows install banner when PWA is installable but not installed', () => {
      mockPWAState.isInstallable = true;
      mockPWAState.isInstalled = false;
      
      render(<MobileNavigation />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      expect(screen.getByText('Install Nuclear AO3')).toBeInTheDocument();
      expect(screen.getByText('Install as an app for better offline reading and faster access')).toBeInTheDocument();
    });

    it('does not show install banner when PWA is not installable', () => {
      mockPWAState.isInstallable = false;
      mockPWAState.isInstalled = false;
      
      render(<MobileNavigation />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      expect(screen.queryByText('Install Nuclear AO3')).not.toBeInTheDocument();
    });

    it('does not show install banner when PWA is already installed', () => {
      mockPWAState.isInstallable = true;
      mockPWAState.isInstalled = true;
      
      render(<MobileNavigation />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      expect(screen.queryByText('Install Nuclear AO3')).not.toBeInTheDocument();
    });

    it('handles install button click', async () => {
      mockPWAState.isInstallable = true;
      mockPWAState.isInstalled = false;
      mockPWAActions.installPWA.mockResolvedValue(true);
      
      render(<MobileNavigation />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      fireEvent.click(screen.getByText('Install'));
      
      await waitFor(() => {
        expect(mockPWAActions.installPWA).toHaveBeenCalled();
      });
    });

    it('handles dismiss button click', () => {
      mockPWAState.isInstallable = true;
      mockPWAState.isInstalled = false;
      
      render(<MobileNavigation />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      fireEvent.click(screen.getByText('Later'));
      
      expect(mockPWAActions.dismissInstallPrompt).toHaveBeenCalled();
    });
  });

  describe('PWA Update Banner', () => {
    it('shows update banner when update is available', () => {
      mockPWAState.updateAvailable = true;
      
      render(<MobileNavigation />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      expect(screen.getByText('Update Available')).toBeInTheDocument();
      expect(screen.getByText('A new version of Nuclear AO3 is ready to install')).toBeInTheDocument();
    });

    it('does not show update banner when no update is available', () => {
      mockPWAState.updateAvailable = false;
      
      render(<MobileNavigation />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      expect(screen.queryByText('Update Available')).not.toBeInTheDocument();
    });

    it('handles update button click', () => {
      mockPWAState.updateAvailable = true;
      
      render(<MobileNavigation />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      fireEvent.click(screen.getByText('Update Now'));
      
      expect(mockPWAActions.skipWaiting).toHaveBeenCalled();
    });
  });

  describe('User Profile Section', () => {
    it('shows user avatar when provided', () => {
      render(<MobileNavigation user={mockUser} />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      const avatar = screen.getByAltText('testuser');
      expect(avatar).toBeInTheDocument();
      expect(avatar).toHaveAttribute('src', '/avatar.jpg');
    });

    it('shows default avatar when no avatar is provided', () => {
      const userWithoutAvatar = { ...mockUser, avatar: undefined };
      
      render(<MobileNavigation user={userWithoutAvatar} />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      // Should show default avatar container
      expect(screen.getByText('testuser')).toBeInTheDocument();
      expect(screen.queryByAltText('testuser')).not.toBeInTheDocument();
    });

    it('shows sign out button for authenticated users', () => {
      render(<MobileNavigation user={mockUser} />);
      
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      expect(screen.getByText('Sign Out')).toBeInTheDocument();
    });
  });

  describe('Event Listeners', () => {
    it('sets up offline works event listeners', () => {
      render(<MobileNavigation />);
      
      expect(mockOfflineManager.onMessage).toHaveBeenCalledWith('WORK_CACHED_WITH_CONSENT', expect.any(Function));
      expect(mockOfflineManager.onMessage).toHaveBeenCalledWith('WORK_DELETED', expect.any(Function));
    });

    it('reloads offline works count when work is cached', async () => {
      let cachedHandler: Function;
      mockOfflineManager.onMessage.mockImplementation((event, handler) => {
        if (event === 'WORK_CACHED_WITH_CONSENT') {
          cachedHandler = handler;
        }
      });
      
      mockOfflineManager.getOfflineWorks
        .mockResolvedValueOnce([]) // Initial load
        .mockResolvedValueOnce([{ workId: 'work-1', isExpired: false }]); // After cache event
      
      render(<MobileNavigation />);
      
      await waitFor(() => {
        expect(mockOfflineManager.getOfflineWorks).toHaveBeenCalledTimes(1);
      });
      
      // Trigger cached event
      act(() => {
        cachedHandler!();
      });
      
      await waitFor(() => {
        expect(mockOfflineManager.getOfflineWorks).toHaveBeenCalledTimes(2);
      });
    });
  });

  describe('Cleanup', () => {
    it('removes event listeners on unmount', () => {
      const { unmount } = render(<MobileNavigation />);
      
      unmount();
      
      expect(mockOfflineManager.offMessage).toHaveBeenCalledWith('WORK_CACHED_WITH_CONSENT');
      expect(mockOfflineManager.offMessage).toHaveBeenCalledWith('WORK_DELETED');
    });
  });

  describe('Offline Reading Manager Integration', () => {
    it('opens offline reading manager from menu', async () => {
      mockOfflineManager.getOfflineWorks.mockResolvedValue([
        { workId: 'work-1', isExpired: false },
      ]);
      
      render(<MobileNavigation />);
      
      await waitFor(() => {
        expect(screen.getByText('1 work offline')).toBeInTheDocument();
      });
      
      // Open menu
      fireEvent.click(screen.getByLabelText('Open menu'));
      
      // Click offline works button in menu
      fireEvent.click(screen.getByText('Offline Works'));
      
      expect(screen.getByTestId('offline-reading-manager')).toBeInTheDocument();
    });

    it('closes offline reading manager', async () => {
      mockOfflineManager.getOfflineWorks.mockResolvedValue([
        { workId: 'work-1', isExpired: false },
      ]);
      
      render(<MobileNavigation />);
      
      await waitFor(() => {
        expect(screen.getByText('1 work offline')).toBeInTheDocument();
      });
      
      // Open offline manager
      fireEvent.click(screen.getByText('1 work offline'));
      expect(screen.getByTestId('offline-reading-manager')).toBeInTheDocument();
      
      // Close offline manager
      fireEvent.click(screen.getByText('Close Manager'));
      expect(screen.queryByTestId('offline-reading-manager')).not.toBeInTheDocument();
    });
  });

  describe('Responsive Design', () => {
    it('renders both top and bottom navigation elements', () => {
      render(<MobileNavigation />);
      
      // Top navigation elements
      expect(screen.getByText('Nuclear AO3')).toBeInTheDocument();
      expect(screen.getByLabelText('Open menu')).toBeInTheDocument();
      
      // Bottom navigation elements
      expect(screen.getByText('Home')).toBeInTheDocument();
      expect(screen.getByText('Browse')).toBeInTheDocument();
      expect(screen.getByText('Bookmarks')).toBeInTheDocument();
      expect(screen.getByText('Profile')).toBeInTheDocument();
    });

    it('renders spacer elements for fixed positioning', () => {
      const { container } = render(<MobileNavigation />);
      
      const spacers = container.querySelectorAll('.h-14, .h-16');
      expect(spacers).toHaveLength(2); // Top and bottom spacers
    });
  });
});