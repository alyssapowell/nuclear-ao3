'use client';

import React, { useState, useEffect } from 'react';
import { Comment, CommentCreateRequest } from '@/types/comment';
import { useConversationNavigation } from '@/hooks/useConversationNavigation';

interface ConversationViewProps {
  threadId: string;
  workId: string;
  onClose?: () => void;
  onBackToComments?: () => void;
  className?: string;
}

interface FlattenedComment extends Comment {
  replyToUsername?: string;
  replyToCommentId?: string;
  level: number;
}

export default function ConversationView({
  threadId,
  workId,
  onClose,
  onBackToComments,
  className = ''
}: ConversationViewProps) {
  const [conversation, setConversation] = useState<FlattenedComment[]>([]);
  const [rootComment, setRootComment] = useState<Comment | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [replyContent, setReplyContent] = useState('');
  const [replyingTo, setReplyingTo] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Navigation hook for URL management
  const { navigateToComments, shareConversation } = useConversationNavigation({
    workId,
    threadId,
    onNavigateAway: onClose
  });

  // Fetch conversation thread
  useEffect(() => {
    fetchConversation();
  }, [threadId, workId]);

  const fetchConversation = async () => {
    try {
      setLoading(true);
      setError(null);

      // Fetch all comments for the work first
      const response = await fetch(`/api/works/${workId}/comments?threaded=true`);
      if (!response.ok) throw new Error('Failed to fetch comments');
      
      const data = await response.json();
      const allComments = data.comments || [];

      // Find the thread starting with threadId
      const thread = findCommentThread(allComments, threadId);
      if (!thread) {
        throw new Error('Thread not found');
      }

      // Flatten the thread for conversation view
      const flattened = flattenThread(thread);
      setConversation(flattened);
      setRootComment(thread);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load conversation');
    } finally {
      setLoading(false);
    }
  };

  // Find a specific comment thread by ID
  const findCommentThread = (comments: Comment[], targetId: string): Comment | null => {
    for (const comment of comments) {
      if (comment.id === targetId) {
        return comment;
      }
      if (comment.replies && comment.replies.length > 0) {
        const found = findCommentThread(comment.replies, targetId);
        if (found) return comment; // Return the root of the thread
      }
    }
    return null;
  };

  // Flatten a threaded comment structure for conversation view
  const flattenThread = (thread: Comment): FlattenedComment[] => {
    const flattened: FlattenedComment[] = [];
    
    const flattenRecursive = (comment: Comment, level: number = 0, parentUsername?: string, parentId?: string) => {
      flattened.push({
        ...comment,
        level,
        replyToUsername: parentUsername,
        replyToCommentId: parentId
      });

      if (comment.replies) {
        comment.replies.forEach(reply => {
          flattenRecursive(reply, level + 1, comment.author_name, comment.id);
        });
      }
    };

    flattenRecursive(thread);
    return flattened;
  };

  // Handle reply submission
  const handleReplySubmit = async (targetCommentId?: string) => {
    if (!replyContent.trim() || submitting) return;

    try {
      setSubmitting(true);
      
      const replyData: CommentCreateRequest = {
        work_id: workId,
        content: replyContent.trim(),
        parent_comment_id: targetCommentId || rootComment?.id
      };

      const response = await fetch(`/api/works/${workId}/comments`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(replyData),
      });

      if (!response.ok) throw new Error('Failed to post reply');

      // Refresh the conversation
      await fetchConversation();
      setReplyContent('');
      setReplyingTo(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to post reply');
    } finally {
      setSubmitting(false);
    }
  };

  // Format comment with @mentions for context
  const formatCommentContent = (comment: FlattenedComment) => {
    if (comment.level === 0 || !comment.replyToUsername) {
      return comment.content;
    }
    
    // Add @mention if it's not already in the content
    const mention = `@${comment.replyToUsername}`;
    if (!comment.content.startsWith(mention)) {
      return `${mention} ${comment.content}`;
    }
    return comment.content;
  };

  if (loading) {
    return (
      <div className={`flex items-center justify-center p-8 ${className}`}>
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={`p-4 ${className}`}>
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <p className="text-red-800">Error: {error}</p>
          <button
            onClick={fetchConversation}
            className="mt-2 text-red-600 hover:text-red-800 underline"
          >
            Try again
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className={`conversation-view ${className}`}>
      {/* Header */}
      <div className="conversation-header border-b border-gray-200 dark:border-gray-700 pb-4 mb-6">
        {(onBackToComments || onClose) && (
          <nav className="mb-3" aria-label="Conversation breadcrumb">
            <button
              onClick={onBackToComments || navigateToComments}
              className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 inline-flex items-center text-sm"
            >
                <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                </svg>
              Back to all comments
            </button>
            {onClose && (
              <button
                onClick={onClose}
                className="ml-4 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
                aria-label="Close conversation"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            )}
          </nav>
        )}
        
        <div className="flex items-start justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              Conversation Thread
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
              {conversation.length} {conversation.length === 1 ? 'comment' : 'comments'} in this thread
            </p>
          </div>
          
          <button
            onClick={() => shareConversation(rootComment?.content ? `Thread starting with "${rootComment.content.substring(0, 50)}..."` : undefined)}
            className="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 inline-flex items-center px-2 py-1 border border-blue-200 dark:border-blue-600 rounded hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors"
            title="Share this conversation"
          >
            <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.367 2.684 3 3 0 00-5.367-2.684z" />
            </svg>
            Share
          </button>
        </div>
      </div>

      {/* Flattened Comments */}
      <div className="conversation-comments space-y-4 mb-6" role="log" aria-label="Comment thread">
        {conversation.map((comment, index) => (
          <article
            key={comment.id}
            className={`
              comment-item p-4 rounded-lg border
              ${comment.level === 0 
                ? 'bg-blue-50 border-blue-200 dark:bg-blue-900/20 dark:border-blue-700' 
                : 'bg-gray-50 border-gray-200 dark:bg-gray-800 dark:border-gray-700'
              }
            `}
            style={{ marginLeft: `${Math.min(comment.level * 12, 48)}px` }}
          >
            {/* Comment Header */}
            <div className="flex items-start justify-between mb-3">
              <div className="flex items-center space-x-2 text-sm">
                <span className="font-medium text-gray-900 dark:text-white">
                  {comment.author_name}
                </span>
                {comment.author_type === 'guest' && (
                  <span className="text-xs bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-300 px-2 py-1 rounded">
                    Guest
                  </span>
                )}
                <span className="text-gray-500 dark:text-gray-400">
                  {new Date(comment.created_at).toLocaleDateString()}
                </span>
                {comment.level > 0 && (
                  <span className="text-xs text-gray-400 dark:text-gray-500">
                    ↳ Reply {comment.level === 1 ? 'to thread' : `(${comment.level} deep)`}
                  </span>
                )}
              </div>
              
              <div className="flex items-center space-x-2">
                {comment.kudos_count > 0 && (
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    ❤️ {comment.kudos_count}
                  </span>
                )}
              </div>
            </div>

            {/* Comment Content */}
            <div className="prose prose-sm max-w-none dark:prose-invert mb-3">
              <p className="text-gray-900 dark:text-white whitespace-pre-wrap">
                {formatCommentContent(comment)}
              </p>
            </div>

            {/* Comment Actions */}
            <div className="flex items-center space-x-4 text-sm">
              <button
                onClick={() => setReplyingTo(replyingTo === comment.id ? null : comment.id)}
                className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
              >
                Reply
              </button>
              {/* Add kudos button, etc. */}
            </div>

            {/* Reply Form */}
            {replyingTo === comment.id && (
              <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-600">
                <div className="space-y-3">
                  <textarea
                    value={replyContent}
                    onChange={(e) => setReplyContent(e.target.value)}
                    placeholder={`Reply to ${comment.author_name}...`}
                    className="w-full p-3 border border-gray-300 dark:border-gray-600 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                    rows={3}
                  />
                  <div className="flex items-center space-x-2">
                    <button
                      onClick={() => handleReplySubmit(comment.id)}
                      disabled={!replyContent.trim() || submitting}
                      className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {submitting ? 'Posting...' : 'Reply'}
                    </button>
                    <button
                      onClick={() => {
                        setReplyingTo(null);
                        setReplyContent('');
                      }}
                      className="px-4 py-2 text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </div>
            )}
          </article>
        ))}
      </div>

      {/* Reply to Conversation Form */}
      <div className="conversation-reply border-t border-gray-200 dark:border-gray-700 pt-6">
        <h3 className="text-md font-medium text-gray-900 dark:text-white mb-3">
          Reply to conversation
        </h3>
        <div className="space-y-3">
          <textarea
            value={replyingTo ? '' : replyContent}
            onChange={(e) => !replyingTo && setReplyContent(e.target.value)}
            placeholder={`Add to the conversation...`}
            className="w-full p-3 border border-gray-300 dark:border-gray-600 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
            rows={3}
            disabled={replyingTo !== null}
          />
          <div className="flex items-center space-x-2">
            <button
              onClick={() => handleReplySubmit()}
              disabled={(!replyContent.trim() && !replyingTo) || submitting || replyingTo !== null}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitting ? 'Posting...' : 'Reply to conversation'}
            </button>
            {replyingTo && (
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Close the reply above to reply to the conversation
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}