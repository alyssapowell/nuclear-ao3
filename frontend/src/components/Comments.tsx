'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { getWorkComments, createComment, updateComment, deleteComment, giveCommentKudos, removeCommentKudos, WorkComment, CreateCommentRequest, CommentUpdateRequest } from '@/lib/api';
import { getConversationUrl, navigation, routes } from '@/lib/urls';
import SlideoutPanel from './SlideoutPanel';
import ConversationView from './ConversationView';

interface CommentsProps {
  workId: string;
  chapterId?: string;
  allowComments?: boolean;
  authToken?: string;
}

interface CommentTreeNode extends WorkComment {
  children: CommentTreeNode[];
}

export default function Comments({ workId, chapterId, allowComments = true, authToken }: CommentsProps) {
  const router = useRouter();
  const [comments, setComments] = useState<WorkComment[]>([]);
  const [commentTree, setCommentTree] = useState<CommentTreeNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [newComment, setNewComment] = useState('');
  const [replyingTo, setReplyingTo] = useState<string | null>(null);
  const [replyContent, setReplyContent] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [isAnonymous, setIsAnonymous] = useState(false);
  const [kudosLoading, setKudosLoading] = useState<Set<string>>(new Set());
  const [conversationView, setConversationView] = useState<{
    threadId: string;
    open: boolean;
  } | null>(null);
  const [editingComment, setEditingComment] = useState<string | null>(null);
  const [editContent, setEditContent] = useState('');
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  // Fetch comments on mount
  useEffect(() => {
    fetchComments();
  }, [workId]);

  // Build comment tree whenever comments change
  useEffect(() => {
    buildCommentTree();
  }, [comments]);

  // Handle browser navigation for conversation URLs
  useEffect(() => {
    const handlePopState = () => {
      const context = routes.getCurrentContext();
      
      // If URL indicates we should show a conversation but we're not showing one
      if (context.type === 'conversation' && context.workId === workId && !conversationView?.open) {
        setConversationView({ threadId: context.threadId!, open: true });
      }
      // If URL doesn't indicate conversation but we're showing one
      else if (context.type !== 'conversation' && conversationView?.open) {
        setConversationView(null);
      }
    };

    // Check initial URL state
    handlePopState();

    // Listen for browser navigation
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, [workId, conversationView?.open]);

  const fetchComments = async () => {
    try {
      setLoading(true);
      const data = await getWorkComments(workId, authToken);
      setComments(data.comments || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load comments');
    } finally {
      setLoading(false);
    }
  };

  const buildCommentTree = () => {
    const commentMap = new Map<string, CommentTreeNode>();
    const rootComments: CommentTreeNode[] = [];

    // Create nodes
    comments.forEach(comment => {
      commentMap.set(comment.id, { ...comment, children: [] });
    });

    // Build tree structure
    comments.forEach(comment => {
      const node = commentMap.get(comment.id)!;
      if (comment.parent_comment_id) {
        const parent = commentMap.get(comment.parent_comment_id);
        if (parent) {
          parent.children.push(node);
        } else {
          // Parent not found, treat as root
          rootComments.push(node);
        }
      } else {
        rootComments.push(node);
      }
    });

    setCommentTree(rootComments);
  };

  const handleSubmitComment = async (content: string, parentId?: string) => {
    if (!content.trim()) return;

    try {
      setSubmitting(true);
      const commentData: CreateCommentRequest = {
        content: content.trim(),
        chapter_id: chapterId,
        parent_comment_id: parentId,
        is_anonymous: isAnonymous,
      };

      const response = await createComment(workId, commentData, authToken);
      
      // Add new comment to list
      if (response.comment) {
        setComments(prev => [...prev, response.comment]);
      }

      // Clear form
      if (parentId) {
        setReplyContent('');
        setReplyingTo(null);
      } else {
        setNewComment('');
      }

      if (response.message) {
        // Show moderation message if needed
        console.log(response.message);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to post comment');
    } finally {
      setSubmitting(false);
    }
  };

  const handleCommentKudos = async (commentId: string) => {
    if (!authToken) {
      setError('You must be logged in to give kudos');
      return;
    }

    try {
      setKudosLoading(prev => new Set(prev).add(commentId));
      
      const comment = comments.find(c => c.id === commentId);
      if (!comment) return;

      if (comment.has_kudos) {
        await removeCommentKudos(workId, commentId, authToken);
        // Update local state
        setComments(prev => prev.map(c => 
          c.id === commentId 
            ? { ...c, has_kudos: false, kudos_count: (c.kudos_count || 1) - 1 }
            : c
        ));
      } else {
        await giveCommentKudos(workId, commentId, authToken);
        // Update local state
        setComments(prev => prev.map(c => 
          c.id === commentId 
            ? { ...c, has_kudos: true, kudos_count: (c.kudos_count || 0) + 1 }
            : c
        ));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update kudos');
    } finally {
      setKudosLoading(prev => {
        const newSet = new Set(prev);
        newSet.delete(commentId);
        return newSet;
      });
    }
  };

  const handleEditComment = async (commentId: string, content: string) => {
    if (!content.trim() || !authToken) return;

    try {
      setSubmitting(true);
      const updateData: CommentUpdateRequest = {
        content: content.trim(),
      };

      const response = await updateComment(workId, commentId, updateData, authToken);
      
      // Update local state
      setComments(prev => prev.map(c => 
        c.id === commentId 
          ? { ...c, content: response.content, updated_at: response.updated_at, edited_at: response.edited_at }
          : c
      ));

      // Clear edit state
      setEditingComment(null);
      setEditContent('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to edit comment');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteComment = async (commentId: string) => {
    if (!authToken) return;

    try {
      setSubmitting(true);
      
      await deleteComment(workId, commentId, authToken);
      
      // Remove comment from local state
      setComments(prev => prev.filter(c => c.id !== commentId));
      setDeleteConfirm(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete comment');
    } finally {
      setSubmitting(false);
    }
  };

  const startEditComment = (comment: CommentTreeNode) => {
    setEditingComment(comment.id);
    setEditContent(comment.content);
  };

  const canEditComment = (comment: CommentTreeNode) => {
    // User can edit their own comments (guest comments can't be edited)
    // For now, we'll check if the user is authenticated and it's not a guest comment
    // TODO: Implement proper user ID checking when user context is available
    return authToken && comment.author_type === 'user' && !comment.is_deleted;
  };

  const canDeleteComment = (comment: CommentTreeNode) => {
    // User can delete their own comments, or moderators can delete any comment
    // For now, we'll check if the user is authenticated and it's not a guest comment
    // TODO: Implement proper user ID checking and moderator role checking
    return authToken && comment.author_type === 'user' && !comment.is_deleted;
  };

  const renderComment = (comment: CommentTreeNode, depth: number = 0) => {
    const maxDepth = 5;
    const canReply = depth < maxDepth && allowComments;

    return (
      <div key={comment.id} className={`${depth > 0 ? 'ml-6 border-l border-slate-200 pl-4' : ''}`}>
        <div className="bg-white border border-slate-200 rounded-lg p-4 mb-4">
          <div className="flex items-start justify-between mb-3">
            <div className="flex items-center gap-2">
              <span className="font-medium text-slate-900">
                {comment.username || 'Anonymous'}
              </span>
              {comment.status === 'pending' && (
                <span className="text-xs px-2 py-1 bg-yellow-100 text-yellow-800 rounded">
                  Pending Moderation
                </span>
              )}
            </div>
            <div className="text-sm text-slate-500">
              <time>
                {new Date(comment.created_at).toLocaleDateString()}
              </time>
              {comment.edited_at && (
                <span className="ml-2 text-xs">
                  (edited {new Date(comment.edited_at).toLocaleDateString()})
                </span>
              )}
            </div>
          </div>
          
          <div className="prose prose-sm max-w-none text-slate-700 mb-3">
            {comment.content.split('\n').map((line, idx) => (
              <p key={idx} className="mb-2 last:mb-0">
                {line}
              </p>
            ))}
          </div>

          <div className="flex items-center gap-4">
            {/* Kudos button */}
            <button
              onClick={() => handleCommentKudos(comment.id)}
              disabled={kudosLoading.has(comment.id)}
              className={`flex items-center gap-1 text-sm transition-colors ${
                comment.has_kudos 
                  ? 'text-red-600 hover:text-red-700' 
                  : 'text-slate-500 hover:text-red-600'
              }`}
            >
              {kudosLoading.has(comment.id) ? (
                <div className="w-4 h-4 loading-spinner"></div>
              ) : (
                <svg className="w-4 h-4" fill={comment.has_kudos ? 'currentColor' : 'none'} stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
                </svg>
              )}
              {comment.kudos_count || 0}
            </button>

            {canReply && (
              <button
                onClick={() => setReplyingTo(replyingTo === comment.id ? null : comment.id)}
                className="text-sm text-orange-600 hover:text-orange-700 transition-colors"
              >
                {replyingTo === comment.id ? 'Cancel Reply' : 'Reply'}
              </button>
            )}

            {/* Conversation link - show if comment has replies */}
            {comment.children.length > 0 && (
              <>
                {/* Desktop/Tablet: Use slideout panel */}
                <button
                  onClick={() => {
                    setConversationView({ threadId: comment.id, open: true });
                    // Update URL for sharing/navigation
                    navigation.toConversation(workId, comment.id, true);
                  }}
                  className="hidden md:inline-flex text-sm text-blue-600 hover:text-blue-700 transition-colors items-center gap-1"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                  </svg>
                  View conversation ({comment.children.length + 1})
                </button>
                
                {/* Mobile: Navigate to conversation page */}
                <button
                  onClick={() => router.push(getConversationUrl(workId, comment.id))}
                  className="md:hidden text-sm text-blue-600 hover:text-blue-700 transition-colors inline-flex items-center gap-1"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                  </svg>
                  View conversation ({comment.children.length + 1})
                </button>
                
                {/* Share conversation link */}
                <button
                  onClick={async () => {
                    const success = await navigation.shareConversation(workId, comment.id, 'Work Title');
                    if (!success) {
                      // Could add a toast notification here
                      console.log('Failed to share conversation');
                    }
                  }}
                  className="text-sm text-slate-500 hover:text-slate-700 transition-colors inline-flex items-center gap-1"
                  title="Share this conversation"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.367 2.684 3 3 0 00-5.367-2.684z" />
                  </svg>
                  Share
                </button>
              </>
            )}

            {/* Edit button */}
            {canEditComment(comment) && (
              <button
                onClick={() => startEditComment(comment)}
                className="text-sm text-gray-600 hover:text-gray-800 transition-colors"
              >
                Edit
              </button>
            )}

            {/* Delete button */}
            {canDeleteComment(comment) && (
              <button
                onClick={() => setDeleteConfirm(comment.id)}
                className="text-sm text-red-600 hover:text-red-800 transition-colors"
              >
                Delete
              </button>
            )}
          </div>

          {replyingTo === comment.id && (
            <div className="mt-4 p-3 bg-slate-50 rounded-lg">
              <textarea
                value={replyContent}
                onChange={(e) => setReplyContent(e.target.value)}
                placeholder="Write your reply..."
                className="form-textarea mb-3"
                rows={3}
              />
              <div className="flex items-center justify-between">
                <label className="flex items-center text-sm text-slate-600">
                  <input
                    type="checkbox"
                    checked={isAnonymous}
                    onChange={(e) => setIsAnonymous(e.target.checked)}
                    className="form-checkbox mr-2"
                  />
                  Post anonymously
                </label>
                <div className="flex gap-2">
                  <button
                    onClick={() => setReplyingTo(null)}
                    className="btn btn-outline btn-sm"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={() => handleSubmitComment(replyContent, comment.id)}
                    disabled={submitting || !replyContent.trim()}
                    className="btn btn-primary btn-sm"
                  >
                    {submitting ? 'Posting...' : 'Post Reply'}
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Edit form */}
          {editingComment === comment.id && (
            <div className="mt-4 p-3 bg-blue-50 rounded-lg">
              <textarea
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
                placeholder="Edit your comment..."
                className="form-textarea mb-3"
                rows={3}
              />
              <div className="flex gap-2">
                <button
                  onClick={() => {
                    setEditingComment(null);
                    setEditContent('');
                  }}
                  className="btn btn-outline btn-sm"
                >
                  Cancel
                </button>
                <button
                  onClick={() => handleEditComment(comment.id, editContent)}
                  disabled={submitting || !editContent.trim()}
                  className="btn btn-primary btn-sm"
                >
                  {submitting ? 'Saving...' : 'Save Changes'}
                </button>
              </div>
            </div>
          )}

          {/* Delete confirmation */}
          {deleteConfirm === comment.id && (
            <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg">
              <p className="text-sm text-red-800 mb-3">
                Are you sure you want to delete this comment? This action cannot be undone.
              </p>
              <div className="flex gap-2">
                <button
                  onClick={() => setDeleteConfirm(null)}
                  className="btn btn-outline btn-sm"
                >
                  Cancel
                </button>
                <button
                  onClick={() => handleDeleteComment(comment.id)}
                  disabled={submitting}
                  className="btn btn-danger btn-sm"
                >
                  {submitting ? 'Deleting...' : 'Delete Comment'}
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Render child comments */}
        {comment.children.map(child => renderComment(child, depth + 1))}
      </div>
    );
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="loading-spinner mr-3"></div>
        <span>Loading comments...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h3 className="text-xl font-semibold text-slate-900">
          Comments ({comments.length})
        </h3>
      </div>

      {error && (
        <div className="error-message">
          {error}
          <button
            onClick={() => setError(null)}
            className="ml-2 text-sm underline"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* New comment form */}
      {allowComments && (
        <div className="card">
          <div className="card-header">
            <h4 className="text-lg font-medium text-slate-900">Leave a Comment</h4>
          </div>
          <div className="card-body">
            <textarea
              value={newComment}
              onChange={(e) => setNewComment(e.target.value)}
              placeholder="Share your thoughts about this work..."
              className="form-textarea mb-4"
              rows={4}
            />
            <div className="flex items-center justify-between">
              <label className="flex items-center text-sm text-slate-600">
                <input
                  type="checkbox"
                  checked={isAnonymous}
                  onChange={(e) => setIsAnonymous(e.target.checked)}
                  className="form-checkbox mr-2"
                />
                Post anonymously
              </label>
              <button
                onClick={() => handleSubmitComment(newComment)}
                disabled={submitting || !newComment.trim()}
                className="btn btn-primary"
              >
                {submitting ? (
                  <>
                    <div className="loading-spinner mr-2"></div>
                    Posting...
                  </>
                ) : (
                  'Post Comment'
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Comments list */}
      {commentTree.length === 0 ? (
        <div className="text-center py-16">
          <div className="text-slate-400 mb-4">
            <svg className="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
          </div>
          <p className="text-slate-600 mb-4">No comments yet.</p>
          {allowComments && (
            <p className="text-sm text-slate-500">Be the first to share your thoughts!</p>
          )}
        </div>
      ) : (
        <div className="space-y-6">
          {commentTree.map(comment => renderComment(comment))}
        </div>
      )}

      {/* Conversation SlideoutPanel */}
      {conversationView && (
        <SlideoutPanel
          isOpen={conversationView.open}
          onClose={() => {
            setConversationView(null);
            // Update URL back to comments when closing
            navigation.toWorkComments(workId, true);
          }}
          title="Comment Conversation"
          side="right"
          width="700px"
          className="md:block hidden"
        >
          <ConversationView
            threadId={conversationView.threadId}
            workId={workId}
            onClose={() => {
              setConversationView(null);
              navigation.toWorkComments(workId, true);
            }}
            onBackToComments={() => {
              setConversationView(null);
              navigation.toWorkComments(workId, true);
            }}
          />
        </SlideoutPanel>
      )}

      {/* Mobile full-screen conversation view */}
      {conversationView && (
        <div className="md:hidden fixed inset-0 bg-white z-50 flex flex-col">
          <ConversationView
            threadId={conversationView.threadId}
            workId={workId}
            onClose={() => {
              setConversationView(null);
              navigation.toWorkComments(workId, true);
            }}
            onBackToComments={() => {
              setConversationView(null);
              navigation.toWorkComments(workId, true);
            }}
            className="flex-1 p-4 overflow-y-auto"
          />
        </div>
      )}
    </div>
  );
}