'use client';

import React, { useState, useEffect, useRef } from 'react';
import { Bell, X } from 'lucide-react';
import Badge from '../ui/Badge';
import { NotificationDropdown } from './NotificationDropdown';
import { useNotifications } from '../../hooks/useNotifications';
import { useWebSocket } from '../../hooks/useWebSocket';

export interface Notification {
  id: string;
  title: string;
  description: string;
  action_url?: string;
  is_read: boolean;
  created_at: string;
  actor_name: string;
  priority: 'high' | 'medium' | 'low';
  source_type: string;
}

export const NotificationBell: React.FC = () => {
  const [isOpen, setIsOpen] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isHydrated, setIsHydrated] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  
  // Check authentication status - FIXED: Safe SSR access with hydration
  useEffect(() => {
    // Mark component as hydrated
    setIsHydrated(true);
    
    // Ensure we're on the client side
    if (typeof window === 'undefined') return;
    
    const checkAuth = () => {
      try {
        const token = localStorage.getItem('auth_token');
        setIsAuthenticated(!!token);
      } catch (error) {
        // Handle localStorage access errors gracefully
        console.warn('Could not access localStorage:', error);
        setIsAuthenticated(false);
      }
    };
    
    checkAuth();
    
    // Listen for auth changes
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === 'auth_token' || e.key === 'token') {
        checkAuth();
      }
    };
    
    window.addEventListener('storage', handleStorageChange);
    return () => window.removeEventListener('storage', handleStorageChange);
  }, []);
  
  const {
    notifications,
    loading,
    fetchNotifications,
    markAsRead,
    deleteNotification,
    getUnreadCount,
  } = useNotifications();

  const { connect, disconnect, isConnected } = useWebSocket({
    url: process.env.NEXT_PUBLIC_NOTIFICATION_WS_URL || 'ws://localhost:8004/ws',
    onMessage: (data) => {
      if (data.type === 'unread_count') {
        setUnreadCount(data.payload.count);
      } else if (data.type === 'new_notification') {
        // Refresh notifications list
        fetchNotifications();
        setUnreadCount(prev => prev + 1);
      }
    },
    onError: (error) => {
      console.warn('WebSocket connection error (non-critical):', error);
    },
    maxReconnectAttempts: 3, // Reduce reconnection attempts
    reconnectInterval: 10000, // Increase reconnection interval
  });

  useEffect(() => {
    // Only initialize if user is authenticated
    if (!isAuthenticated) {
      return;
    }

    // Only connect if we have a valid auth token
    let authToken: string | null = null;
    try {
      authToken = typeof window !== 'undefined' ? localStorage.getItem('auth_token') : null;
    } catch (error) {
      console.warn('Could not access localStorage for auth token:', error);
      return;
    }
    
    if (!authToken || authToken === 'test-token') {
      return;
    }

    // Initial data fetch
    fetchNotifications();
    getUnreadCount().then(count => setUnreadCount(count));
    
    // Connect to WebSocket for real-time updates  
    connect();

    return () => {
      disconnect();
    };
  }, [isAuthenticated]); // Only depend on authentication status

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleNotificationClick = async (notification: Notification) => {
    if (!notification.is_read) {
      await markAsRead(notification.id);
      setUnreadCount(prev => Math.max(0, prev - 1));
    }
    
    if (notification.action_url) {
      window.location.href = notification.action_url;
    }
    setIsOpen(false);
  };

  const handleDeleteNotification = async (notificationId: string, event: React.MouseEvent) => {
    event.stopPropagation();
    const notification = notifications.find(n => n.id === notificationId);
    
    await deleteNotification(notificationId);
    
    if (notification && !notification.is_read) {
      setUnreadCount(prev => Math.max(0, prev - 1));
    }
  };

  const handleMarkAllRead = async () => {
    const unreadNotifications = notifications.filter(n => !n.is_read);
    
    await Promise.all(
      unreadNotifications.map(notification => markAsRead(notification.id))
    );
    
    setUnreadCount(0);
  };

  // Don't render anything until hydrated to prevent SSR mismatch
  if (!isHydrated) {
    return null;
  }
  
  // Don't render anything if user is not authenticated
  if (!isAuthenticated) {
    return null;
  }

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="relative p-2 text-gray-600 hover:text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 rounded-lg transition-colors"
        aria-label={`Notifications${unreadCount > 0 ? ` (${unreadCount} unread)` : ''}`}
      >
        <Bell size={20} />
        {unreadCount > 0 && (
          <Badge
            variant="danger"
            className="absolute -top-1 -right-1 min-w-[1.25rem] h-5 flex items-center justify-center text-xs"
          >
            {unreadCount > 99 ? '99+' : unreadCount}
          </Badge>
        )}
        {isConnected && (
          <div className="absolute -bottom-1 -right-1 w-2 h-2 bg-green-500 rounded-full border border-white" />
        )}
      </button>

      {isOpen && (
        <NotificationDropdown
          notifications={notifications}
          loading={loading}
          unreadCount={unreadCount}
          onNotificationClick={handleNotificationClick}
          onDeleteNotification={handleDeleteNotification}
          onMarkAllRead={handleMarkAllRead}
          onClose={() => setIsOpen(false)}
        />
      )}
    </div>
  );
};