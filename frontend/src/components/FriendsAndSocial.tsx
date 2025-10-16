'use client';

import { useState, useEffect, useCallback, useMemo, useRef, useId } from 'react';
import { 
  sendFriendRequest, 
  acceptFriendRequest,
  blockUser,
  getUserDashboard,
  UserRelationship,
  FriendRequestRequest,
  BlockUserRequest 
} from '@/lib/api';

interface FriendsAndSocialProps {
  authToken?: string;
}

export default function FriendsAndSocial({ authToken }: FriendsAndSocialProps) {
  const [relationships, setRelationships] = useState<UserRelationship[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<Set<string>>(new Set());
  const [activeTab, setActiveTab] = useState<'friends' | 'requests' | 'blocked'>('friends');
  const [announcements, setAnnouncements] = useState<string>('');
  
  // Add friend form state
  const [showAddFriend, setShowAddFriend] = useState(false);
  const [friendUsername, setFriendUsername] = useState('');
  const [submitting, setSubmitting] = useState(false);

  // Accessibility
  const componentId = useId();
  const mainHeadingId = `${componentId}-heading`;
  const liveRegionId = `${componentId}-live`;
  const friendFormRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    if (authToken) {
      loadDashboard();
    }
  }, [authToken]);

  const loadDashboard = useCallback(async () => {
    try {
      setLoading(true);
      const response = await getUserDashboard(authToken);
      // Extract relationships from dashboard response
      setRelationships(response.relationships || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load social data');
    } finally {
      setLoading(false);
    }
  }, [authToken]);

  const handleSendFriendRequest = useCallback(async () => {
    if (!friendUsername.trim()) {
      setError('Username is required');
      return;
    }

    try {
      setSubmitting(true);
      setError(null);
      
      // Note: We'll need to resolve username to user ID in the backend
      const requestData: FriendRequestRequest = {
        user_id: friendUsername.trim() // Backend should handle username lookup
      };
      
      await sendFriendRequest(requestData, authToken);
      setSuccessMessage('Friend request sent successfully!');
      setAnnouncements('Friend request sent successfully!');
      setTimeout(() => setSuccessMessage(null), 3000);
      setFriendUsername('');
      setShowAddFriend(false);
      await loadDashboard(); // Reload to show updated relationships
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send friend request');
    } finally {
      setSubmitting(false);
    }
  }, [friendUsername, authToken, loadDashboard]);

  const handleAcceptRequest = async (requestId: string) => {
    try {
      setActionLoading(prev => new Set(prev).add(requestId));
      setError(null);
      
      await acceptFriendRequest(requestId, authToken);
      setSuccessMessage('Friend request accepted!');
      setTimeout(() => setSuccessMessage(null), 3000);
      await loadDashboard();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to accept friend request');
    } finally {
      setActionLoading(prev => {
        const newSet = new Set(prev);
        newSet.delete(requestId);
        return newSet;
      });
    }
  };

  const handleBlockUser = async (userId: string, blockType: 'block_user' | 'block_comments' | 'block_works') => {
    try {
      setActionLoading(prev => new Set(prev).add(userId));
      setError(null);
      
      const blockData: BlockUserRequest = {
        user_id: userId,
        type: blockType
      };
      
      await blockUser(blockData, authToken);
      setSuccessMessage(`User ${blockType.replace('block_', '')} blocked successfully!`);
      setTimeout(() => setSuccessMessage(null), 3000);
      await loadDashboard();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to block user');
    } finally {
      setActionLoading(prev => {
        const newSet = new Set(prev);
        newSet.delete(userId);
        return newSet;
      });
    }
  };

  const getRelationshipsByType = (type: string, status?: string) => {
    return relationships.filter(rel => {
      if (type === 'friends') {
        return rel.type === 'friend' && rel.status === 'accepted';
      }
      if (type === 'requests') {
        return rel.type === 'friend' && rel.status === 'pending';
      }
      if (type === 'blocked') {
        return rel.type.startsWith('block_');
      }
      return false;
    });
  };

  const getBlockTypeLabel = (type: string) => {
    switch (type) {
      case 'block_user': return 'Fully Blocked';
      case 'block_comments': return 'Comments Blocked';
      case 'block_works': return 'Works Blocked';
      default: return 'Blocked';
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="loading-spinner mr-3"></div>
        <span>Loading social data...</span>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900 mb-2">Friends & Social</h1>
        <p className="text-slate-600">Manage your friends, requests, and blocked users</p>
      </div>

      {/* Tab Navigation */}
      <div className="border-b border-slate-200 mb-6">
        <nav className="-mb-px flex space-x-8">
          {[
            { id: 'friends', label: 'Friends', icon: '👥', count: getRelationshipsByType('friends').length },
            { id: 'requests', label: 'Requests', icon: '📩', count: getRelationshipsByType('requests').length },
            { id: 'blocked', label: 'Blocked', icon: '🚫', count: getRelationshipsByType('blocked').length },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`py-2 px-1 border-b-2 font-medium text-sm ${
                activeTab === tab.id
                  ? 'border-orange-500 text-orange-600'
                  : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
              }`}
            >
              <span className="mr-2">{tab.icon}</span>
              {tab.label}
              {tab.count > 0 && (
                <span className="ml-2 bg-slate-100 text-slate-600 px-2 py-1 rounded-full text-xs">
                  {tab.count}
                </span>
              )}
            </button>
          ))}
        </nav>
      </div>

      {error && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
          <p className="text-red-800">{error}</p>
          <button
            onClick={() => setError(null)}
            className="text-red-600 text-sm underline mt-1"
          >
            Dismiss
          </button>
        </div>
      )}

      {successMessage && (
        <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg">
          <p className="text-green-800">{successMessage}</p>
        </div>
      )}

      {/* Friends Tab */}
      {activeTab === 'friends' && (
        <div className="space-y-6">
          <div className="card">
            <div className="card-header flex items-center justify-between">
              <h2 className="text-lg font-semibold">My Friends</h2>
              <button
                onClick={() => setShowAddFriend(!showAddFriend)}
                className="btn btn-primary btn-sm"
              >
                {showAddFriend ? 'Cancel' : 'Add Friend'}
              </button>
            </div>
            <div className="card-body">
              {showAddFriend && (
                <div className="mb-6 p-4 bg-slate-50 rounded-lg">
                  <h3 className="font-medium mb-4">Send Friend Request</h3>
                  <div className="flex gap-3">
                    <input
                      type="text"
                      value={friendUsername}
                      onChange={(e) => setFriendUsername(e.target.value)}
                      placeholder="Enter username"
                      className="form-input flex-1"
                    />
                    <button
                      onClick={handleSendFriendRequest}
                      disabled={submitting || !friendUsername.trim()}
                      className="btn btn-primary"
                    >
                      {submitting ? 'Sending...' : 'Send Request'}
                    </button>
                  </div>
                </div>
              )}

              <div className="space-y-4">
                {getRelationshipsByType('friends').length === 0 ? (
                  <div className="text-center py-8">
                    <div className="text-slate-400 mb-4">
                      <svg className="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                      </svg>
                    </div>
                    <p className="text-slate-600 mb-4">No friends yet.</p>
                    <p className="text-sm text-slate-500">Start building your network by sending friend requests!</p>
                  </div>
                ) : (
                  getRelationshipsByType('friends').map((relationship) => {
                    const friend = relationship.requester_id === relationship.requested_id 
                      ? relationship.requested 
                      : relationship.requester;
                    
                    return (
                      <div key={relationship.id} className="border border-slate-200 rounded-lg p-4">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <div className="w-10 h-10 bg-orange-100 rounded-full flex items-center justify-center">
                              <span className="text-orange-600 font-medium">
                                {friend?.username?.charAt(0).toUpperCase() || '?'}
                              </span>
                            </div>
                            <div>
                              <h3 className="font-medium">{friend?.username || 'Unknown User'}</h3>
                              <p className="text-sm text-slate-500">
                                Friends since {new Date(relationship.created_at).toLocaleDateString()}
                              </p>
                            </div>
                          </div>
                          <div className="flex gap-2">
                            <button
                              onClick={() => handleBlockUser(friend?.user_id || '', 'block_user')}
                              disabled={actionLoading.has(friend?.user_id || '')}
                              className="btn btn-outline btn-sm text-red-600 border-red-200 hover:bg-red-50"
                            >
                              Block
                            </button>
                          </div>
                        </div>
                      </div>
                    );
                  })
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Requests Tab */}
      {activeTab === 'requests' && (
        <div className="card">
          <div className="card-header">
            <h2 className="text-lg font-semibold">Friend Requests</h2>
          </div>
          <div className="card-body">
            <div className="space-y-4">
              {getRelationshipsByType('requests').length === 0 ? (
                <div className="text-center py-8">
                  <div className="text-slate-400 mb-4">
                    <svg className="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>
                  </div>
                  <p className="text-slate-600">No pending friend requests.</p>
                </div>
              ) : (
                getRelationshipsByType('requests').map((relationship) => {
                  const requester = relationship.requester;
                  
                  return (
                    <div key={relationship.id} className="border border-slate-200 rounded-lg p-4">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 bg-orange-100 rounded-full flex items-center justify-center">
                            <span className="text-orange-600 font-medium">
                              {requester?.username?.charAt(0).toUpperCase() || '?'}
                            </span>
                          </div>
                          <div>
                            <h3 className="font-medium">{requester?.username || 'Unknown User'}</h3>
                            <p className="text-sm text-slate-500">
                              Sent {new Date(relationship.created_at).toLocaleDateString()}
                            </p>
                          </div>
                        </div>
                        <div className="flex gap-2">
                          <button
                            onClick={() => handleAcceptRequest(relationship.id)}
                            disabled={actionLoading.has(relationship.id)}
                            className="btn btn-primary btn-sm"
                          >
                            {actionLoading.has(relationship.id) ? 'Accepting...' : 'Accept'}
                          </button>
                          <button
                            onClick={() => handleBlockUser(requester?.user_id || '', 'block_user')}
                            disabled={actionLoading.has(requester?.user_id || '')}
                            className="btn btn-outline btn-sm text-red-600 border-red-200 hover:bg-red-50"
                          >
                            Block
                          </button>
                        </div>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </div>
        </div>
      )}

      {/* Blocked Tab */}
      {activeTab === 'blocked' && (
        <div className="card">
          <div className="card-header">
            <h2 className="text-lg font-semibold">Blocked Users</h2>
          </div>
          <div className="card-body">
            <div className="space-y-4">
              {getRelationshipsByType('blocked').length === 0 ? (
                <div className="text-center py-8">
                  <div className="text-slate-400 mb-4">
                    <svg className="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728L5.636 5.636m12.728 12.728L5.636 5.636" />
                    </svg>
                  </div>
                  <p className="text-slate-600">No blocked users.</p>
                </div>
              ) : (
                getRelationshipsByType('blocked').map((relationship) => {
                  const blockedUser = relationship.requested;
                  
                  return (
                    <div key={relationship.id} className="border border-slate-200 rounded-lg p-4">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 bg-red-100 rounded-full flex items-center justify-center">
                            <span className="text-red-600 font-medium">
                              {blockedUser?.username?.charAt(0).toUpperCase() || '?'}
                            </span>
                          </div>
                          <div>
                            <h3 className="font-medium">{blockedUser?.username || 'Unknown User'}</h3>
                            <p className="text-sm text-slate-500">
                              {getBlockTypeLabel(relationship.type)} • 
                              Blocked {new Date(relationship.created_at).toLocaleDateString()}
                            </p>
                          </div>
                        </div>
                        <div className="flex gap-2">
                          <button
                            className="btn btn-outline btn-sm"
                            onClick={() => {
                              // TODO: Implement unblock functionality
                              setError('Unblock functionality coming soon');
                            }}
                          >
                            Unblock
                          </button>
                        </div>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}