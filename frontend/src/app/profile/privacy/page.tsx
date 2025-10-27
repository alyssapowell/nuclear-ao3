'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/Button';
import PrivacyWizard from '@/components/privacy/PrivacyWizard';

interface BlockedUser {
  id: string;
  username: string;
  blocked_at: string;
  block_type: 'content' | 'interaction' | 'complete';
  reason?: string;
}

interface Comment {
  id: string;
  content: string;
  author_username: string;
  work_title: string;
  status: 'approved' | 'pending_moderation' | 'flagged' | 'hidden';
  created_at: string;
  flagged_reason?: string;
}

export default function PrivacySettingsPage() {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('settings');
  const [blockedUsers, setBlockedUsers] = useState<BlockedUser[]>([]);
  const [comments, setComments] = useState<Comment[]>([]);
  const [loading, setLoading] = useState(false);
  const [showWizard, setShowWizard] = useState(false);

  useEffect(() => {
    loadData();
  }, [activeTab]);

  const loadData = async () => {
    setLoading(true);
    try {
      const token = localStorage.getItem('auth_token');
      if (!token) return;

      if (activeTab === 'blocked-users') {
        await loadBlockedUsers(token);
      } else if (activeTab === 'moderation') {
        await loadCommentsForModeration(token);
      }
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadBlockedUsers = async (token: string) => {
    try {
      const response = await fetch('/api/v1/profile/blocked-users', {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });
      
      if (response.ok) {
        const data = await response.json();
        setBlockedUsers(data.blocked_users || []);
      }
    } catch (error) {
      console.warn('Blocked users API not available, using demo data');
      // Demo data for development
      setBlockedUsers([
        {
          id: '1',
          username: 'problematic_user',
          blocked_at: '2024-10-15T10:30:00Z',
          block_type: 'complete',
          reason: 'Harassment in comments'
        },
        {
          id: '2',
          username: 'spam_account',
          blocked_at: '2024-10-10T15:20:00Z',
          block_type: 'interaction',
          reason: 'Spamming comments'
        }
      ]);
    }
  };

  const loadCommentsForModeration = async (token: string) => {
    try {
      const response = await fetch('/api/v1/comments/moderation', {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });
      
      if (response.ok) {
        const data = await response.json();
        setComments(data.comments || []);
      }
    } catch (error) {
      console.warn('Comment moderation API not available, using demo data');
      // Demo data for development
      setComments([
        {
          id: '1',
          content: 'This comment has been flagged for review by community members.',
          author_username: 'reviewer123',
          work_title: 'My Amazing Story',
          status: 'flagged',
          created_at: '2024-10-20T14:30:00Z',
          flagged_reason: 'Inappropriate content'
        },
        {
          id: '2',
          content: 'Great work! I really enjoyed this chapter.',
          author_username: 'fan_reader',
          work_title: 'Another Story',
          status: 'pending_moderation',
          created_at: '2024-10-21T09:15:00Z'
        }
      ]);
    }
  };

  const unblockUser = async (userId: string) => {
    try {
      const token = localStorage.getItem('auth_token');
      if (!token) return;

      const response = await fetch(`/api/v1/users/${userId}/block`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });

      if (response.ok) {
        setBlockedUsers(prev => prev.filter(user => user.id !== userId));
      }
    } catch (error) {
      console.error('Failed to unblock user:', error);
      // For demo, just remove from list
      setBlockedUsers(prev => prev.filter(user => user.id !== userId));
    }
  };

  const moderateComment = async (commentId: string, action: 'approve' | 'hide' | 'delete', reason?: string) => {
    try {
      const token = localStorage.getItem('auth_token');
      if (!token) return;

      const response = await fetch(`/api/v1/comments/${commentId}/moderate`, {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          action,
          reason
        })
      });

      if (response.ok) {
        // Update comment status in local state
        setComments(prev => prev.map(comment => 
          comment.id === commentId 
            ? { ...comment, status: action === 'approve' ? 'approved' : 'hidden' }
            : comment
        ));
      }
    } catch (error) {
      console.error('Failed to moderate comment:', error);
      // For demo, just update local state
      setComments(prev => prev.map(comment => 
        comment.id === commentId 
          ? { ...comment, status: action === 'approve' ? 'approved' : 'hidden' }
          : comment
      ));
    }
  };

  const handleWizardComplete = () => {
    setShowWizard(false);
    setActiveTab('settings');
  };

  const renderTabButton = (tabId: string, label: string, icon: string) => (
    <button
      onClick={() => setActiveTab(tabId)}
      className={`flex items-center px-4 py-3 text-sm font-medium rounded-lg transition-colors ${
        activeTab === tabId
          ? 'bg-orange-100 text-orange-700 border border-orange-200'
          : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'
      }`}
    >
      <span className="mr-2">{icon}</span>
      {label}
    </button>
  );

  const renderSettings = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900">Privacy Settings</h2>
        <Button onClick={() => setShowWizard(true)} variant="outline">
          Open Privacy Wizard
        </Button>
      </div>
      
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
        <h3 className="font-semibold text-blue-900 mb-2">Privacy Setup</h3>
        <p className="text-blue-800 text-sm mb-4">
          Use the Privacy Wizard to configure your content filtering, profile visibility, 
          and notification preferences all in one place.
        </p>
        <Button onClick={() => setShowWizard(true)} className="bg-blue-600 hover:bg-blue-700">
          Launch Privacy Wizard
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <h3 className="font-semibold text-gray-900 mb-4">Content Filtering</h3>
          <ul className="space-y-2 text-sm text-gray-600">
            <li>• Hide explicit content</li>
            <li>• Filter by archive warnings</li>
            <li>• Block specific tags</li>
            <li>• Customize rating preferences</li>
          </ul>
        </div>

        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <h3 className="font-semibold text-gray-900 mb-4">Profile Privacy</h3>
          <ul className="space-y-2 text-sm text-gray-600">
            <li>• Control profile visibility</li>
            <li>• Manage contact permissions</li>
            <li>• Hide reading history</li>
            <li>• Configure public statistics</li>
          </ul>
        </div>

        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <h3 className="font-semibold text-gray-900 mb-4">Work Defaults</h3>
          <ul className="space-y-2 text-sm text-gray-600">
            <li>• Default work privacy level</li>
            <li>• Comment policy preferences</li>
            <li>• Constructive criticism settings</li>
            <li>• Anonymous posting options</li>
          </ul>
        </div>

        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <h3 className="font-semibold text-gray-900 mb-4">Notifications</h3>
          <ul className="space-y-2 text-sm text-gray-600">
            <li>• Email notification preferences</li>
            <li>• Comment and kudos alerts</li>
            <li>• Subscription updates</li>
            <li>• Digest frequency</li>
          </ul>
        </div>
      </div>
    </div>
  );

  const renderBlockedUsers = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900">Blocked Users</h2>
        <span className="text-sm text-gray-500">{blockedUsers.length} blocked</span>
      </div>

      {loading ? (
        <div className="flex justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-orange-600"></div>
        </div>
      ) : blockedUsers.length === 0 ? (
        <div className="text-center py-8">
          <div className="text-gray-400 mb-4">
            <svg className="mx-auto h-12 w-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728L5.636 5.636m12.728 12.728L5.636 5.636" />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-2">No blocked users</h3>
          <p className="text-gray-500">You haven't blocked any users yet.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {blockedUsers.map((user) => (
            <div key={user.id} className="bg-white border border-gray-200 rounded-lg p-6">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center space-x-3">
                    <h3 className="font-medium text-gray-900">{user.username}</h3>
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                      user.block_type === 'complete' 
                        ? 'bg-red-100 text-red-800'
                        : user.block_type === 'interaction'
                        ? 'bg-yellow-100 text-yellow-800'
                        : 'bg-gray-100 text-gray-800'
                    }`}>
                      {user.block_type === 'complete' ? 'Complete Block' : 
                       user.block_type === 'interaction' ? 'Interaction Block' : 'Content Block'}
                    </span>
                  </div>
                  <p className="text-sm text-gray-500 mt-1">
                    Blocked {new Date(user.blocked_at).toLocaleDateString()}
                  </p>
                  {user.reason && (
                    <p className="text-sm text-gray-600 mt-2">
                      <span className="font-medium">Reason:</span> {user.reason}
                    </p>
                  )}
                </div>
                <Button
                  onClick={() => unblockUser(user.id)}
                  variant="outline"
                  size="sm"
                  className="ml-4"
                >
                  Unblock
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6">
        <h3 className="font-semibold text-yellow-900 mb-2">About User Blocking</h3>
        <div className="text-sm text-yellow-800 space-y-2">
          <p><strong>Complete Block:</strong> Hides all content from this user and prevents all interactions.</p>
          <p><strong>Interaction Block:</strong> Prevents the user from commenting on your works or contacting you.</p>
          <p><strong>Content Block:</strong> Hides the user's works from your searches and recommendations.</p>
        </div>
      </div>
    </div>
  );

  const renderModeration = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900">Comment Moderation</h2>
        <span className="text-sm text-gray-500">
          {comments.filter(c => c.status === 'pending_moderation' || c.status === 'flagged').length} requiring review
        </span>
      </div>

      {loading ? (
        <div className="flex justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-orange-600"></div>
        </div>
      ) : comments.length === 0 ? (
        <div className="text-center py-8">
          <div className="text-gray-400 mb-4">
            <svg className="mx-auto h-12 w-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-2">No comments to moderate</h3>
          <p className="text-gray-500">All comments on your works are up to date.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {comments.map((comment) => (
            <div key={comment.id} className="bg-white border border-gray-200 rounded-lg p-6">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center space-x-3 mb-2">
                    <h4 className="font-medium text-gray-900">{comment.author_username}</h4>
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                      comment.status === 'flagged' 
                        ? 'bg-red-100 text-red-800'
                        : comment.status === 'pending_moderation'
                        ? 'bg-yellow-100 text-yellow-800'
                        : comment.status === 'approved'
                        ? 'bg-green-100 text-green-800'
                        : 'bg-gray-100 text-gray-800'
                    }`}>
                      {comment.status === 'flagged' ? 'Flagged' : 
                       comment.status === 'pending_moderation' ? 'Pending' : 
                       comment.status === 'approved' ? 'Approved' : 'Hidden'}
                    </span>
                  </div>
                  <p className="text-sm text-gray-500 mb-3">
                    On "{comment.work_title}" • {new Date(comment.created_at).toLocaleDateString()}
                  </p>
                  <div className="bg-gray-50 rounded-lg p-3 mb-3">
                    <p className="text-sm text-gray-700">{comment.content}</p>
                  </div>
                  {comment.flagged_reason && (
                    <p className="text-sm text-red-600">
                      <span className="font-medium">Flagged for:</span> {comment.flagged_reason}
                    </p>
                  )}
                </div>
                
                {(comment.status === 'pending_moderation' || comment.status === 'flagged') && (
                  <div className="flex space-x-2 ml-4">
                    <Button
                      onClick={() => moderateComment(comment.id, 'approve')}
                      size="sm"
                      className="bg-green-600 hover:bg-green-700"
                    >
                      Approve
                    </Button>
                    <Button
                      onClick={() => moderateComment(comment.id, 'hide', 'Inappropriate content')}
                      size="sm"
                      variant="outline"
                      className="text-red-600 border-red-300 hover:bg-red-50"
                    >
                      Hide
                    </Button>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
        <h3 className="font-semibold text-blue-900 mb-2">Comment Moderation</h3>
        <div className="text-sm text-blue-800 space-y-2">
          <p><strong>Approve:</strong> Allow the comment to be visible to all readers.</p>
          <p><strong>Hide:</strong> Hide the comment from public view (author can still see it).</p>
          <p><strong>Flagged:</strong> Comments reported by community members for review.</p>
          <p><strong>Pending:</strong> Comments waiting for your approval (if moderation is enabled).</p>
        </div>
      </div>
    </div>
  );

  if (showWizard) {
    return (
      <PrivacyWizard
        onComplete={handleWizardComplete}
        onSkip={handleWizardComplete}
        title="Update Privacy Settings"
        subtitle="Review and update your privacy and safety preferences"
      />
    );
  }

  return (
    <div className="max-w-6xl mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">Privacy & Safety</h1>
        <p className="text-gray-600">
          Manage your privacy settings, blocked users, and content moderation preferences.
        </p>
      </div>

      <div className="flex flex-col lg:flex-row gap-8">
        {/* Sidebar Navigation */}
        <div className="lg:w-64 space-y-2">
          {renderTabButton('settings', 'Privacy Settings', '⚙️')}
          {renderTabButton('blocked-users', 'Blocked Users', '🚫')}
          {renderTabButton('moderation', 'Comment Moderation', '🛡️')}
        </div>

        {/* Main Content */}
        <div className="flex-1">
          {activeTab === 'settings' && renderSettings()}
          {activeTab === 'blocked-users' && renderBlockedUsers()}
          {activeTab === 'moderation' && renderModeration()}
        </div>
      </div>
    </div>
  );
}