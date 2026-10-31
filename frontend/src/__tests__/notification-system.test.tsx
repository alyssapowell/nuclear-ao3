import type { MockedFunction } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { NotificationBell, type Notification } from '../components/notifications/NotificationBell';
import { useNotifications } from '../hooks/useNotifications';

vi.mock('../hooks/useNotifications', () => ({ useNotifications: vi.fn() }));
vi.mock('../hooks/useWebSocket', () => ({
  useWebSocket: () => ({
    connect: vi.fn(),
    disconnect: vi.fn(),
    sendMessage: vi.fn(),
    isConnected: false,
    reconnectAttempts: 0,
  }),
}));
const mockUseNotifications = useNotifications as MockedFunction<typeof useNotifications>;

// Notifications name people by display name only (actor_name), never by account.
const mockNotifications: Notification[] = [
  {
    id: '1',
    title: 'New comment on your work',
    description: 'Someone commented on "Test Work"',
    is_read: false,
    created_at: '2023-01-01T10:00:00Z',
    actor_name: 'Commenter',
    priority: 'medium',
    source_type: 'comment',
  },
  {
    id: '2',
    title: 'New kudos received',
    description: 'Someone gave kudos to "Test Work"',
    is_read: true,
    created_at: '2023-01-01T09:00:00Z',
    actor_name: 'Anonymous Creator',
    priority: 'low',
    source_type: 'work',
  },
  {
    id: '3',
    title: 'New chapter posted',
    description: 'A work you subscribe to was updated',
    is_read: false,
    created_at: '2023-01-01T08:00:00Z',
    actor_name: 'Writer',
    priority: 'high',
    source_type: 'work',
  },
];

const hook = {
  notifications: mockNotifications,
  loading: false,
  error: null,
  fetchNotifications: vi.fn().mockResolvedValue(undefined),
  markAsRead: vi.fn().mockResolvedValue(undefined),
  deleteNotification: vi.fn().mockResolvedValue(undefined),
  getUnreadCount: vi.fn().mockResolvedValue(2),
  createTestNotification: vi.fn().mockResolvedValue(undefined),
};

// setup.ts gives every test a fake localStorage; this logs the reader in or out.
const logIn = (token: string | null) => {
  vi.mocked(localStorage.getItem).mockImplementation((key: string) => (key === 'auth_token' ? token : null));
};

const openBell = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(await screen.findByRole('button', { name: /^notifications/i }));
};

describe('Notification System', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    hook.fetchNotifications.mockResolvedValue(undefined);
    hook.markAsRead.mockResolvedValue(undefined);
    hook.deleteNotification.mockResolvedValue(undefined);
    hook.getUnreadCount.mockResolvedValue(2);
    mockUseNotifications.mockReturnValue(hook);
    logIn('real-token');
  });

  describe('NotificationBell', () => {
    it('shows nothing to a reader who is not logged in', async () => {
      logIn(null);
      const { container } = render(<NotificationBell />);

      await waitFor(() => expect(container).toBeEmptyDOMElement());
      expect(hook.fetchNotifications).not.toHaveBeenCalled();
    });

    it('loads notifications and shows the unread count', async () => {
      render(<NotificationBell />);

      const bell = await screen.findByRole('button', { name: 'Notifications (2 unread)' });
      expect(within(bell).getByText('2')).toBeInTheDocument();
      expect(hook.fetchNotifications).toHaveBeenCalled();
    });

    it('shows no badge when nothing is unread', async () => {
      hook.getUnreadCount.mockResolvedValue(0);
      render(<NotificationBell />);

      const bell = await screen.findByRole('button', { name: 'Notifications' });
      expect(within(bell).queryByText('0')).not.toBeInTheDocument();
    });

    it('caps a large count at 99+', async () => {
      hook.getUnreadCount.mockResolvedValue(250);
      render(<NotificationBell />);

      expect(await screen.findByText('99+')).toBeInTheDocument();
    });

    it('opens the dropdown with the notifications', async () => {
      const user = userEvent.setup();
      render(<NotificationBell />);

      await openBell(user);

      expect(screen.getByText('New comment on your work')).toBeInTheDocument();
      expect(screen.getByText('New kudos received')).toBeInTheDocument();
      expect(screen.getByText('Commenter')).toBeInTheDocument();
      // Notifications name people, but never link to them.
      expect(screen.queryByRole('link')).not.toBeInTheDocument();
    });

    it('closes the dropdown when the reader clicks outside', async () => {
      const user = userEvent.setup();
      render(
        <div>
          <NotificationBell />
          <div data-testid="outside">Outside element</div>
        </div>
      );

      await openBell(user);
      expect(screen.getByText('New comment on your work')).toBeInTheDocument();

      await user.click(screen.getByTestId('outside'));

      await waitFor(() => {
        expect(screen.queryByText('New comment on your work')).not.toBeInTheDocument();
      });
    });

    it('closes the dropdown with its close button', async () => {
      const user = userEvent.setup();
      render(<NotificationBell />);

      await openBell(user);
      await user.click(screen.getByRole('button', { name: 'Close notifications' }));

      expect(screen.queryByText('New comment on your work')).not.toBeInTheDocument();
    });

    it('marks an unread notification read when it is clicked', async () => {
      const user = userEvent.setup();
      render(<NotificationBell />);

      await openBell(user);
      await user.click(screen.getByText('New comment on your work'));

      expect(hook.markAsRead).toHaveBeenCalledWith('1');
      expect(await screen.findByRole('button', { name: 'Notifications (1 unread)' })).toBeInTheDocument();
    });

    it('leaves an already read notification alone when it is clicked', async () => {
      const user = userEvent.setup();
      render(<NotificationBell />);

      await openBell(user);
      await user.click(screen.getByText('New kudos received'));

      expect(hook.markAsRead).not.toHaveBeenCalled();
    });

    it('marks every unread notification read', async () => {
      const user = userEvent.setup();
      render(<NotificationBell />);

      await openBell(user);
      await user.click(screen.getByRole('button', { name: /mark all read/i }));

      expect(hook.markAsRead).toHaveBeenCalledTimes(2);
      expect(hook.markAsRead).toHaveBeenCalledWith('1');
      expect(hook.markAsRead).toHaveBeenCalledWith('3');
      expect(await screen.findByRole('button', { name: 'Notifications' })).toBeInTheDocument();
    });

    it('deletes a notification without opening it', async () => {
      const user = userEvent.setup();
      render(<NotificationBell />);

      await openBell(user);
      const deleteButtons = screen.getAllByRole('button', { name: 'Delete notification' });
      await user.click(deleteButtons[0]);

      expect(hook.deleteNotification).toHaveBeenCalledWith('1');
      expect(hook.markAsRead).not.toHaveBeenCalled();
    });
  });

  describe('NotificationDropdown states', () => {
    it('says when the reader is all caught up', async () => {
      const user = userEvent.setup();
      mockUseNotifications.mockReturnValue({ ...hook, notifications: [] });
      render(<NotificationBell />);

      await openBell(user);

      expect(screen.getByText('No notifications')).toBeInTheDocument();
      expect(screen.getByText("You're all caught up!")).toBeInTheDocument();
    });

    it('shows a loading message while notifications load', async () => {
      const user = userEvent.setup();
      mockUseNotifications.mockReturnValue({ ...hook, loading: true });
      render(<NotificationBell />);

      await openBell(user);

      expect(screen.getByText('Loading notifications...')).toBeInTheDocument();
    });
  });
});
