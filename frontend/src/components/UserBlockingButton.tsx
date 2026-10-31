import { useState } from 'react';
import { Button } from '@/components/ui/Button';

// Blocks the person behind a pseud. The block applies to every pseud they
// have and works both ways: works stay readable, comments are closed.
interface UserBlockingButtonProps {
  pseudId: string;
  pseudName: string;
  size?: 'sm' | 'md' | 'lg';
  variant?: 'text' | 'icon' | 'button';
  onBlockSuccess?: (pseudName: string) => void;
  className?: string;
}

export default function UserBlockingButton({
  pseudId,
  pseudName,
  size = 'sm',
  variant = 'text',
  onBlockSuccess,
  className = ''
}: UserBlockingButtonProps) {
  const [showBlockModal, setShowBlockModal] = useState(false);
  const [isBlocked, setIsBlocked] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleBlock = async () => {
    setLoading(true);
    try {
      const token = localStorage.getItem('auth_token');
      if (!token) {
        alert('You must be logged in to block');
        return;
      }

      const response = await fetch('/api/v1/blocks', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ pseud_id: pseudId })
      });

      if (response.ok) {
        setIsBlocked(true);
        setShowBlockModal(false);
        onBlockSuccess?.(pseudName);
      } else {
        const errorData = await response.json().catch(() => ({}));
        alert(errorData.message || errorData.error || 'Failed to block');
      }
    } catch (error) {
      console.error('Failed to block:', error);
      alert('Failed to block. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleUnblock = async () => {
    setLoading(true);
    try {
      const token = localStorage.getItem('auth_token');
      if (!token) {
        alert('You must be logged in to unblock');
        return;
      }

      const response = await fetch(`/api/v1/blocks/${pseudId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });

      if (response.ok) {
        setIsBlocked(false);
        onBlockSuccess?.(pseudName);
      } else {
        const errorData = await response.json().catch(() => ({}));
        alert(errorData.message || errorData.error || 'Failed to unblock');
      }
    } catch (error) {
      console.error('Failed to unblock:', error);
      alert('Failed to unblock. Please try again.');
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
            title="Unblock"
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
          title="Block"
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
        Block
      </Button>
    );
  };

  const BlockModal = () => (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg max-w-md w-full">
        <div className="p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">
            Block {pseudName}?
          </h3>

          <div className="text-sm text-gray-700 mb-6 space-y-2">
            <p>This blocks the person behind this pseud, under every pseud they have.</p>
            <p>You can still read each other's works, but neither of you can comment, reply or leave kudos on the other's works, and you won't see comments from them or be notified about what they do.</p>
          </div>

          <div className="flex justify-end space-x-3 mt-6">
            <Button
              onClick={() => setShowBlockModal(false)}
              variant="outline"
              disabled={loading}
            >
              Cancel
            </Button>
            <Button
              onClick={handleBlock}
              disabled={loading}
              className="bg-red-600 text-white hover:bg-red-700"
            >
              {loading ? 'Blocking...' : 'Block'}
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