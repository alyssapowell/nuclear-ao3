import { useState, useCallback } from 'react';
import type { Notification } from '../components/notifications/NotificationBell';

const API_BASE_URL = process.env.NEXT_PUBLIC_NOTIFICATION_API_URL || 'http://localhost:8004/api/v1';

interface UseNotificationsReturn {
  notifications: Notification[];
  loading: boolean;
  error: string | null;
  fetchNotifications: () => Promise<void>;
  markAsRead: (notificationId: string) => Promise<void>;
  deleteNotification: (notificationId: string) => Promise<void>;
  getUnreadCount: () => Promise<number>;
  createTestNotification: () => Promise<void>;
}

export const useNotifications = (): UseNotificationsReturn => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const getAuthHeaders = (): HeadersInit => {
    // Safe localStorage access for SSR compatibility
    let token = 'test-token';
    let userId = '550e8400-e29b-41d4-a716-446655440001';
    
    if (typeof window !== 'undefined') {
      try {
        token = localStorage.getItem('auth_token') || token;
        userId = localStorage.getItem('user_id') || userId;
      } catch (error) {
        console.warn('Could not access localStorage in getAuthHeaders:', error);
      }
    }
    
    return {
      'Authorization': `Bearer ${token}`,
      'X-User-ID': userId,
      'Content-Type': 'application/json',
    };
  };

  const fetchNotifications = useCallback(async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(`${API_BASE_URL}/notifications?limit=20`, {
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch notifications: ${response.status}`);
      }

      const data = await response.json();
      setNotifications(data.notifications || []);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch notifications';
      setError(errorMessage);
      console.error('Error fetching notifications:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  const markAsRead = useCallback(async (notificationId: string) => {
    try {
      const response = await fetch(`${API_BASE_URL}/notifications/${notificationId}/read`, {
        method: 'PUT',
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`Failed to mark notification as read: ${response.status}`);
      }

      // Update local state
      setNotifications(prev =>
        prev.map(notification =>
          notification.id === notificationId
            ? { ...notification, is_read: true }
            : notification
        )
      );
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to mark notification as read';
      setError(errorMessage);
      console.error('Error marking notification as read:', err);
    }
  }, []);

  const deleteNotification = useCallback(async (notificationId: string) => {
    try {
      const response = await fetch(`${API_BASE_URL}/notifications/${notificationId}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`Failed to delete notification: ${response.status}`);
      }

      // Update local state
      setNotifications(prev =>
        prev.filter(notification => notification.id !== notificationId)
      );
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to delete notification';
      setError(errorMessage);
      console.error('Error deleting notification:', err);
    }
  }, []);

  const getUnreadCount = useCallback(async (): Promise<number> => {
    try {
      const response = await fetch(`${API_BASE_URL}/notifications/unread-count`, {
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`Failed to get unread count: ${response.status}`);
      }

      const data = await response.json();
      return data.count || 0;
    } catch (err) {
      console.error('Error fetching unread count:', err);
      return 0;
    }
  }, []);

  const createTestNotification = useCallback(async () => {
    try {
      const testEvent = {
        type: 'comment_received',
        source_id: '550e8400-e29b-41d4-a716-446655440000',
        source_type: 'work',
        title: 'Test notification from frontend',
        description: 'This is a test notification created from the frontend',
        action_url: '/works/550e8400-e29b-41d4-a716-446655440000/comments',
        actor_id: '550e8400-e29b-41d4-a716-446655440001',
        actor_name: 'Test User',
        extra_data: {
          test: true,
          created_from: 'frontend'
        }
      };

      const response = await fetch(`${API_BASE_URL}/test-notification`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify(testEvent),
      });

      if (!response.ok) {
        throw new Error(`Failed to create test notification: ${response.status}`);
      }

      // Refresh notifications after creating test notification
      await fetchNotifications();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to create test notification';
      setError(errorMessage);
      console.error('Error creating test notification:', err);
    }
  }, [fetchNotifications]);

  return {
    notifications,
    loading,
    error,
    fetchNotifications,
    markAsRead,
    deleteNotification,
    getUnreadCount,
    createTestNotification,
  };
};