import type { MockedFunction } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import NotificationBell from '../components/notifications/NotificationBell';
import NotificationList from '../components/notifications/NotificationList';
import { useNotifications } from '../hooks/useNotifications';
import { useAuth } from '../utils/auth';

// Mock dependencies
vi.mock('../hooks/useNotifications', () => ({
  useNotifications: vi.fn(),
}));

vi.mock('../utils/auth', () => ({
  useAuth: vi.fn(),
  getAuthState: vi.fn(),
  isAuthenticated: vi.fn(),
}));


const mockUseNotifications = useNotifications as MockedFunction<typeof useNotifications>;
const mockUseAuth = useAuth as MockedFunction<typeof useAuth>;

const mockNotifications = [
  {
    id: '1',
    type: 'comment',
    title: 'New comment on your work',
    message: 'User commented on "Test Work"',
    timestamp: new Date('2023-01-01T10:00:00Z'),
    read: false,
    workId: 'work-1',
    userId: 'user-1',
  },
  {
    id: '2',
    type: 'kudos',
    title: 'New kudos received',
    message: 'User gave kudos to "Test Work"',
    timestamp: new Date('2023-01-01T09:00:00Z'),
    read: true,
    workId: 'work-1',
    userId: 'user-2',
  },
  {
    id: '3',
    type: 'follow',
    title: 'New follower',
    message: 'User started following you',
    timestamp: new Date('2023-01-01T08:00:00Z'),
    read: false,
    userId: 'user-3',
  },
];

beforeEach(() => {
  vi.clearAllMocks();
  
  mockUseAuth.mockReturnValue({
    user: { id: '1', username: 'testuser' },
    isLoading: false,
    isAuthenticated: true,
    login: vi.fn(),
    logout: vi.fn(),
    register: vi.fn(),
  });

  // Mock localStorage for SSR safety
  Object.defineProperty(window, 'localStorage', {
    value: {
      getItem: vi.fn(),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    },
    writable: true,
  });
});

describe('Notification System Tests', () => {
  describe('NotificationBell Component', () => {
    test('renders notification bell with unread count', () => {
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationBell />);

      expect(screen.getByRole('button', { name: /notifications/i })).toBeInTheDocument();
      expect(screen.getByText('2')).toBeInTheDocument(); // Unread count badge
    });

    test('does not show badge when no unread notifications', () => {
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications.map(n => ({ ...n, read: true })),
        unreadCount: 0,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationBell />);

      expect(screen.getByRole('button', { name: /notifications/i })).toBeInTheDocument();
      expect(screen.queryByText('0')).not.toBeInTheDocument();
    });

    test('opens notification dropdown on click', async () => {
      const user = userEvent.setup();
      
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationBell />);

      const bellButton = screen.getByRole('button', { name: /notifications/i });
      await user.click(bellButton);

      // Should show notification dropdown
      expect(screen.getByText(/new comment on your work/i)).toBeInTheDocument();
      expect(screen.getByText(/new kudos received/i)).toBeInTheDocument();
    });

    test('closes dropdown when clicking outside', async () => {
      const user = userEvent.setup();
      
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(
        <div>
          <NotificationBell />
          <div data-testid="outside">Outside element</div>
        </div>
      );

      // Open dropdown
      const bellButton = screen.getByRole('button', { name: /notifications/i });
      await user.click(bellButton);

      expect(screen.getByText(/new comment on your work/i)).toBeInTheDocument();

      // Click outside
      await user.click(screen.getByTestId('outside'));

      // Dropdown should close
      await waitFor(() => {
        expect(screen.queryByText(/new comment on your work/i)).not.toBeInTheDocument();
      });
    });

    test('handles SSR safety when localStorage is undefined', () => {
      // Mock SSR environment
      const originalWindow = global.window;
      delete (global as any).window;

      mockUseNotifications.mockReturnValue({
        notifications: [],
        unreadCount: 0,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      expect(() => render(<NotificationBell />)).not.toThrow();

      global.window = originalWindow;
    });

    test('shows loading state correctly', () => {
      mockUseNotifications.mockReturnValue({
        notifications: [],
        unreadCount: 0,
        isLoading: true,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationBell />);

      const bellButton = screen.getByRole('button', { name: /notifications/i });
      expect(bellButton).toBeInTheDocument();
      // Loading indicator should be present
      expect(screen.getByRole('progressbar')).toBeInTheDocument();
    });

    test('handles keyboard navigation', async () => {
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationBell />);

      const bellButton = screen.getByRole('button', { name: /notifications/i });
      bellButton.focus();

      // Press Enter to open
      fireEvent.keyDown(bellButton, { key: 'Enter' });
      
      expect(screen.getByText(/new comment on your work/i)).toBeInTheDocument();

      // Press Escape to close
      fireEvent.keyDown(document, { key: 'Escape' });
      
      await waitFor(() => {
        expect(screen.queryByText(/new comment on your work/i)).not.toBeInTheDocument();
      });
    });
  });

  describe('NotificationList Component', () => {
    test('renders list of notifications correctly', () => {
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationList />);

      expect(screen.getByText(/new comment on your work/i)).toBeInTheDocument();
      expect(screen.getByText(/new kudos received/i)).toBeInTheDocument();
      expect(screen.getByText(/new follower/i)).toBeInTheDocument();
    });

    test('distinguishes between read and unread notifications', () => {
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationList />);

      const notifications = screen.getAllByRole('listitem');
      
      // First notification (unread) should have distinct styling
      expect(notifications[0]).toHaveClass('unread');
      
      // Second notification (read) should not have unread styling
      expect(notifications[1]).not.toHaveClass('unread');
    });

    test('marks notification as read when clicked', async () => {
      const user = userEvent.setup();
      const mockMarkAsRead = vi.fn();
      
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: mockMarkAsRead,
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationList />);

      const firstNotification = screen.getByText(/new comment on your work/i);
      await user.click(firstNotification);

      expect(mockMarkAsRead).toHaveBeenCalledWith('1');
    });

    test('handles mark all as read action', async () => {
      const user = userEvent.setup();
      const mockMarkAllAsRead = vi.fn();
      
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: mockMarkAllAsRead,
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationList />);

      const markAllButton = screen.getByRole('button', { name: /mark all as read/i });
      await user.click(markAllButton);

      expect(mockMarkAllAsRead).toHaveBeenCalled();
    });

    test('handles notification deletion', async () => {
      const user = userEvent.setup();
      const mockDeleteNotification = vi.fn();
      
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: mockDeleteNotification,
        fetchNotifications: vi.fn(),
      });

      render(<NotificationList />);

      const deleteButtons = screen.getAllByRole('button', { name: /delete/i });
      await user.click(deleteButtons[0]);

      expect(mockDeleteNotification).toHaveBeenCalledWith('1');
    });

    test('shows empty state when no notifications', () => {
      mockUseNotifications.mockReturnValue({
        notifications: [],
        unreadCount: 0,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationList />);

      expect(screen.getByText(/no notifications/i)).toBeInTheDocument();
    });

    test('shows loading state', () => {
      mockUseNotifications.mockReturnValue({
        notifications: [],
        unreadCount: 0,
        isLoading: true,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationList />);

      expect(screen.getByText(/loading notifications/i)).toBeInTheDocument();
    });

    test('handles error state', () => {
      mockUseNotifications.mockReturnValue({
        notifications: [],
        unreadCount: 0,
        isLoading: false,
        error: 'Failed to load notifications',
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationList />);

      expect(screen.getByText(/failed to load notifications/i)).toBeInTheDocument();
    });

    test('groups notifications by date', () => {
      const todayNotifications = [
        {
          ...mockNotifications[0],
          timestamp: new Date(),
        },
        {
          ...mockNotifications[1],
          timestamp: new Date(Date.now() - 2 * 60 * 60 * 1000), // 2 hours ago
        },
      ];

      mockUseNotifications.mockReturnValue({
        notifications: todayNotifications,
        unreadCount: 1,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationList />);

      expect(screen.getByText(/today/i)).toBeInTheDocument();
    });

    test('handles pagination for large notification lists', async () => {
      const user = userEvent.setup();
      const manyNotifications = Array.from({ length: 25 }, (_, i) => ({
        ...mockNotifications[0],
        id: `notification-${i}`,
        title: `Notification ${i}`,
      }));

      mockUseNotifications.mockReturnValue({
        notifications: manyNotifications.slice(0, 20), // First page
        unreadCount: 5,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationList />);

      // Should show first 20 notifications
      expect(screen.getAllByRole('listitem')).toHaveLength(20);

      // Should have load more button
      const loadMoreButton = screen.getByRole('button', { name: /load more/i });
      expect(loadMoreButton).toBeInTheDocument();

      await user.click(loadMoreButton);
      // Implementation would load more notifications
    });
  });

  describe('Notification System Integration', () => {
    test('real-time notification updates', async () => {
      const mockFetchNotifications = vi.fn();
      
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: mockFetchNotifications,
      });

      render(<NotificationBell />);

      // Simulate real-time update
      const event = new CustomEvent('notification-update', {
        detail: { newNotification: mockNotifications[0] }
      });
      window.dispatchEvent(event);

      expect(mockFetchNotifications).toHaveBeenCalled();
    });

    test('handles WebSocket connection for real-time updates', () => {
      // Mock WebSocket
      const mockWebSocket = {
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        close: vi.fn(),
      };
      
      global.WebSocket = vi.fn(() => mockWebSocket) as any;

      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationBell />);

      expect(global.WebSocket).toHaveBeenCalled();
      expect(mockWebSocket.addEventListener).toHaveBeenCalledWith('message', expect.any(Function));
    });

    test('persists notification preferences', async () => {
      const user = userEvent.setup();
      const mockSetItem = vi.spyOn(Storage.prototype, 'setItem');
      
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationBell />);

      // Open settings menu
      const bellButton = screen.getByRole('button', { name: /notifications/i });
      await user.click(bellButton);

      const settingsButton = screen.getByRole('button', { name: /settings/i });
      await user.click(settingsButton);

      // Toggle notification preference
      const emailToggle = screen.getByLabelText(/email notifications/i);
      await user.click(emailToggle);

      expect(mockSetItem).toHaveBeenCalledWith(
        'notification-preferences',
        expect.stringContaining('email')
      );
    });

    test('handles notification permission requests', async () => {
      const user = userEvent.setup();
      
      // Mock Notification API
      global.Notification = {
        permission: 'default',
        requestPermission: vi.fn().mockResolvedValue('granted'),
      } as any;

      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationBell />);

      const bellButton = screen.getByRole('button', { name: /notifications/i });
      await user.click(bellButton);

      const enableButton = screen.getByRole('button', { name: /enable browser notifications/i });
      await user.click(enableButton);

      expect(Notification.requestPermission).toHaveBeenCalled();
    });

    test('filters notifications by type', async () => {
      const user = userEvent.setup();
      
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationList />);

      // Filter by comments only
      const filterDropdown = screen.getByRole('combobox', { name: /filter by type/i });
      await user.selectOptions(filterDropdown, 'comment');

      // Should only show comment notifications
      expect(screen.getByText(/new comment on your work/i)).toBeInTheDocument();
      expect(screen.queryByText(/new kudos received/i)).not.toBeInTheDocument();
    });
  });

  describe('Notification Accessibility', () => {
    test('provides proper ARIA labels and announcements', () => {
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationBell />);

      const bellButton = screen.getByRole('button');
      expect(bellButton).toHaveAttribute('aria-label');
      expect(bellButton).toHaveAttribute('aria-expanded', 'false');
    });

    test('announces new notifications to screen readers', () => {
      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationBell />);

      // Should have live region for announcements
      const liveRegion = screen.getByRole('status');
      expect(liveRegion).toBeInTheDocument();
    });

    test('supports high contrast mode', () => {
      // Mock high contrast mode
      Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: vi.fn().mockImplementation(query => ({
          matches: query === '(prefers-contrast: high)',
          media: query,
          onchange: null,
          addListener: vi.fn(),
          removeListener: vi.fn(),
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
          dispatchEvent: vi.fn(),
        })),
      });

      mockUseNotifications.mockReturnValue({
        notifications: mockNotifications,
        unreadCount: 2,
        isLoading: false,
        error: null,
        markAsRead: vi.fn(),
        markAllAsRead: vi.fn(),
        deleteNotification: vi.fn(),
        fetchNotifications: vi.fn(),
      });

      render(<NotificationBell />);

      const bellButton = screen.getByRole('button');
      expect(bellButton).toHaveClass('high-contrast');
    });
  });
});