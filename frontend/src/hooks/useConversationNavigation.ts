'use client';

import { useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { routes, getConversationUrl, getWorkCommentsUrl } from '@/lib/urls';

interface UseConversationNavigationProps {
  workId: string;
  threadId: string;
  onNavigateAway?: () => void;
}

/**
 * Hook for managing conversation URL navigation and browser history
 */
export function useConversationNavigation({
  workId,
  threadId,
  onNavigateAway
}: UseConversationNavigationProps) {
  const router = useRouter();

  // Handle browser back/forward navigation
  useEffect(() => {
    const handlePopState = (event: PopStateEvent) => {
      const context = routes.getCurrentContext();
      
      // If we're no longer viewing this conversation, notify parent
      if (context.type !== 'conversation' || 
          context.threadId !== threadId || 
          context.workId !== workId) {
        onNavigateAway?.();
      }
    };

    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, [workId, threadId, onNavigateAway]);

  // Navigation methods
  const navigateToComments = useCallback(() => {
    router.push(getWorkCommentsUrl(workId));
  }, [router, workId]);

  const navigateToWork = useCallback(() => {
    router.push(`/works/${workId}`);
  }, [router, workId]);

  const navigateToConversation = useCallback((newThreadId: string, replace = false) => {
    const url = getConversationUrl(workId, newThreadId);
    if (replace) {
      router.replace(url);
    } else {
      router.push(url);
    }
  }, [router, workId]);

  // Share conversation
  const shareConversation = useCallback(async (workTitle?: string) => {
    const shareData = {
      title: workTitle ? `Conversation in "${workTitle}"` : 'Comment Conversation',
      url: window.location.href,
      text: `Join the discussion on this work's comment thread.`
    };

    if (navigator.share) {
      try {
        await navigator.share(shareData);
        return true;
      } catch (error) {
        // User cancelled or sharing failed, fall back to clipboard
      }
    }

    // Fallback to clipboard
    try {
      await navigator.clipboard.writeText(shareData.url);
      return true;
    } catch (error) {
      console.warn('Failed to copy URL to clipboard:', error);
      return false;
    }
  }, []);

  // Check if we're currently viewing this conversation
  const isCurrentConversation = useCallback(() => {
    const context = routes.getCurrentContext();
    return context.type === 'conversation' && 
           context.threadId === threadId && 
           context.workId === workId;
  }, [workId, threadId]);

  return {
    navigateToComments,
    navigateToWork,
    navigateToConversation,
    shareConversation,
    isCurrentConversation
  };
}