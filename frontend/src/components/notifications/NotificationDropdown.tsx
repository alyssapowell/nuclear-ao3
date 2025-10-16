import React from 'react';
import { Clock, Trash2, CheckCircle, Users } from 'lucide-react';
import { Button } from '../ui/Button';
import Card from '../ui/Card';
import { formatDistanceToNow } from 'date-fns';
import type { Notification } from './NotificationBell';

interface NotificationDropdownProps {
  notifications: Notification[];
  loading: boolean;
  unreadCount: number;
  onNotificationClick: (notification: Notification) => void;
  onDeleteNotification: (notificationId: string, event: React.MouseEvent) => void;
  onMarkAllRead: () => void;
  onClose: () => void;
}

const getPriorityColor = (priority: string): string => {
  switch (priority) {
    case 'high':
      return 'border-l-red-500';
    case 'medium':
      return 'border-l-yellow-500';
    case 'low':
      return 'border-l-blue-500';
    default:
      return 'border-l-gray-300';
  }
};

const getNotificationIcon = (sourceType: string): React.ReactNode => {
  switch (sourceType) {
    case 'work':
      return <Users size={16} className="text-blue-500" />;
    case 'comment':
      return <Users size={16} className="text-green-500" />;
    default:
      return <Clock size={16} className="text-gray-500" />;
  }
};

export const NotificationDropdown: React.FC<NotificationDropdownProps> = ({
  notifications,
  loading,
  unreadCount,
  onNotificationClick,
  onDeleteNotification,
  onMarkAllRead,
  onClose,
}) => {
  return (
    <Card className="absolute right-0 top-full mt-2 w-96 max-h-96 overflow-hidden shadow-lg z-50">
      <div className="p-4 border-b border-gray-200 bg-gray-50">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900">
            Notifications {unreadCount > 0 && `(${unreadCount})`}
          </h3>
          <div className="flex items-center gap-2">
            {unreadCount > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onMarkAllRead}
                className="text-sm"
              >
                <CheckCircle size={14} className="mr-1" />
                Mark all read
              </Button>
            )}
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600 transition-colors"
              aria-label="Close notifications"
            >
              ×
            </button>
          </div>
        </div>
      </div>

      <div className="max-h-80 overflow-y-auto">
        {loading ? (
          <div className="p-4 text-center text-gray-500">
            <div className="animate-spin inline-block w-6 h-6 border-2 border-current border-t-transparent rounded-full" />
            <p className="mt-2">Loading notifications...</p>
          </div>
        ) : notifications.length === 0 ? (
          <div className="p-8 text-center text-gray-500">
            <Clock size={48} className="mx-auto mb-4 text-gray-300" />
            <p className="text-lg font-medium">No notifications</p>
            <p className="text-sm">You're all caught up!</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {notifications.map((notification) => (
              <div
                key={notification.id}
                className={`p-4 hover:bg-gray-50 cursor-pointer transition-colors relative group border-l-4 ${getPriorityColor(
                  notification.priority
                )} ${!notification.is_read ? 'bg-blue-50' : ''}`}
                onClick={() => onNotificationClick(notification)}
              >
                <div className="flex items-start gap-3">
                  <div className="flex-shrink-0 mt-1">
                    {getNotificationIcon(notification.source_type)}
                  </div>
                  
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <p className={`text-sm font-medium ${
                          !notification.is_read ? 'text-gray-900' : 'text-gray-700'
                        }`}>
                          {notification.title}
                        </p>
                        <p className="text-sm text-gray-600 mt-1 line-clamp-2">
                          {notification.description}
                        </p>
                        <div className="flex items-center gap-2 mt-2 text-xs text-gray-500">
                          <span>{notification.actor_name}</span>
                          <span>•</span>
                          <span>
                            {formatDistanceToNow(new Date(notification.created_at), {
                              addSuffix: true,
                            })}
                          </span>
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-1 ml-2">
                        {!notification.is_read && (
                          <div className="w-2 h-2 bg-blue-500 rounded-full" />
                        )}
                        <button
                          onClick={(e) => onDeleteNotification(notification.id, e)}
                          className="opacity-0 group-hover:opacity-100 p-1 text-gray-400 hover:text-red-500 transition-all"
                          aria-label="Delete notification"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {notifications.length > 0 && (
        <div className="p-3 border-t border-gray-200 bg-gray-50">
          <Button
            variant="ghost"
            size="sm"
            className="w-full text-center text-sm text-blue-600 hover:text-blue-800"
            onClick={() => {
              window.location.href = '/notifications';
              onClose();
            }}
          >
            View all notifications
          </Button>
        </div>
      )}
    </Card>
  );
};