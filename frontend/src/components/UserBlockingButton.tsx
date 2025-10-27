'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/Button';

interface UserBlockingButtonProps {
  username: string;
  userId?: string;
  size?: 'sm' | 'md' | 'lg';
  variant?: 'text' | 'icon' | 'button';
  onBlockSuccess?: (username: string) => void;
  className?: string;
}

interface BlockUserRequest {
  block_type: 'content' | 'interaction' | 'complete';
  reason?: string;
}

export default function UserBlockingButton({
  username,
  userId,
  size = 'sm',
  variant = 'text',
  onBlockSuccess,
  className = ''
}: UserBlockingButtonProps) {
  const [showBlockModal, setShowBlockModal] = useState(false);
  const [isBlocked, setIsBlocked] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleBlock = async (blockType: 'content' | 'interaction' | 'complete', reason?: string) => {
    setLoading(true);
    try {
      const token = localStorage.getItem('auth_token');
      if (!token) {
        alert('You must be logged in to block users');
        return;
      }

      // Use userId if provided, otherwise try username-based endpoint
      const endpoint = userId 
        ? `/api/v1/users/${userId}/block`
        : `/api/v1/users/${encodeURIComponent(username)}/block`;

      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          block_type: blockType,
          reason: reason || 'User blocked via interface'
        } as BlockUserRequest)
      });

      if (response.ok) {
        setIsBlocked(true);
        setShowBlockModal(false);
        onBlockSuccess?.(username);
      } else {
        const errorData = await response.json().catch(() => ({}));
        alert(errorData.error || 'Failed to block user');
      }
    } catch (error) {
      console.error('Failed to block user:', error);
      alert('Failed to block user. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleUnblock = async () => {
    setLoading(true);
    try {
      const token = localStorage.getItem('auth_token');
      if (!token) {
        alert('You must be logged in to unblock users');
        return;
      }

      // Use userId if provided, otherwise try username-based endpoint
      const endpoint = userId 
        ? `/api/v1/users/${userId}/block`
        : `/api/v1/users/${encodeURIComponent(username)}/block`;

      const response = await fetch(endpoint, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });

      if (response.ok) {
        setIsBlocked(false);
        onBlockSuccess?.(username);
      } else {
        const errorData = await response.json().catch(() => ({}));
        alert(errorData.error || 'Failed to unblock user');
      }
    } catch (error) {
      console.error('Failed to unblock user:', error);
      alert('Failed to unblock user. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const renderTrigger = () => {
    if (isBlocked) {
      if (variant === 'icon') {
        return (
          <button
            onClick={handleUnblock}
            disabled={loading}
            className={`text-green-600 hover:text-green-700 ${className}`}
            title="Unblock user"
          >
            ✓
          </button>
        );
      }
      return (
        <button
          onClick={handleUnblock}
          disabled={loading}
          className={`text-green-600 hover:text-green-700 text-sm ${className}`}
        >
          {loading ? 'Unblocking...' : 'Unblock'}
        </button>
      );
    }

    if (variant === 'icon') {
      return (
        <button
          onClick={() => setShowBlockModal(true)}
          className={`text-gray-400 hover:text-red-600 ${className}`}
          title="Block user"
        >
          🚫
        </button>
      );
    }

    if (variant === 'text') {
      return (
        <button
          onClick={() => setShowBlockModal(true)}
          className={`text-gray-400 hover:text-red-600 text-sm ${className}`}
        >
          Block
        </button>
      );
    }

    return (
      <Button
        onClick={() => setShowBlockModal(true)}
        variant="outline"
        size={size}
        className={`text-red-600 border-red-300 hover:bg-red-50 ${className}`}
      >
        Block User
      </Button>
    );
  };

  const BlockModal = () => (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg max-w-md w-full">
        <div className="p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">
            Block {username}?
          </h3>
          
          <p className="text-sm text-gray-600 mb-6">
            Choose how you want to block this user:
          </p>

          <div className="space-y-4">
            <button
              onClick={() => handleBlock('content')}
              disabled={loading}
              className="w-full text-left p-4 border border-gray-200 rounded-lg hover:bg-gray-50 disabled:opacity-50"
            >
              <div className="font-medium text-gray-900">Hide Content</div>
              <div className="text-sm text-gray-500">
                Hide their works from your searches and recommendations
              </div>
            </button>

            <button
              onClick={() => handleBlock('interaction')}
              disabled={loading}
              className="w-full text-left p-4 border border-gray-200 rounded-lg hover:bg-gray-50 disabled:opacity-50"
            >
              <div className="font-medium text-gray-900">Block Interactions</div>
              <div className="text-sm text-gray-500">
                Prevent them from commenting on your works or contacting you
              </div>
            </button>

            <button
              onClick={() => handleBlock('complete')}
              disabled={loading}
              className="w-full text-left p-4 border border-red-200 rounded-lg hover:bg-red-50 disabled:opacity-50"
            >
              <div className="font-medium text-red-900">Complete Block</div>
              <div className="text-sm text-red-600">
                Hide all content and prevent all interactions
              </div>
            </button>
          </div>

          <div className="flex justify-end space-x-3 mt-6">
            <Button
              onClick={() => setShowBlockModal(false)}
              variant="outline"
              disabled={loading}
            >
              Cancel
            </Button>
          </div>
        </div>
      </div>
    </div>
  );

  return (
    <>
      {renderTrigger()}
      {showBlockModal && <BlockModal />}
    </>
  );
}