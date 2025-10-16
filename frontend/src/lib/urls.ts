// URL generation utilities for Nuclear AO3

/**
 * Generate a URL for a work page
 */
export function getWorkUrl(workId: string): string {
  return `/works/${workId}`;
}

/**
 * Generate a URL for a work's comments section
 */
export function getWorkCommentsUrl(workId: string): string {
  return `/works/${workId}#comments`;
}

/**
 * Generate a URL for a specific conversation thread
 */
export function getConversationUrl(workId: string, threadId: string): string {
  return `/works/${workId}/comments/${threadId}`;
}

/**
 * Generate a URL for a specific chapter of a work
 */
export function getChapterUrl(workId: string, chapterNumber: number): string {
  return `/works/${workId}?chapter=${chapterNumber}`;
}

/**
 * Generate a URL for a user's profile
 */
export function getUserUrl(username: string): string {
  return `/users/${username}`;
}

/**
 * Generate a URL for a series page
 */
export function getSeriesUrl(seriesId: string): string {
  return `/series/${seriesId}`;
}

/**
 * Generate a URL for a collection page
 */
export function getCollectionUrl(collectionId: string): string {
  return `/collections/${collectionId}`;
}

/**
 * Generate a search URL with parameters
 */
export function getSearchUrl(params: Record<string, string | string[]>): string {
  const searchParams = new URLSearchParams();
  
  Object.entries(params).forEach(([key, value]) => {
    if (Array.isArray(value)) {
      value.forEach(v => searchParams.append(key, v));
    } else {
      searchParams.append(key, value);
    }
  });
  
  return `/search?${searchParams.toString()}`;
}

/**
 * Parse a conversation URL to extract workId and threadId
 */
export function parseConversationUrl(url: string): { workId: string; threadId: string } | null {
  const match = url.match(/\/works\/([^\/]+)\/comments\/([^\/]+)/);
  if (!match) return null;
  
  return {
    workId: match[1],
    threadId: match[2]
  };
}

/**
 * Check if a URL is a conversation URL
 */
export function isConversationUrl(url: string): boolean {
  return /\/works\/[^\/]+\/comments\/[^\/]+/.test(url);
}

/**
 * Generate a shareable URL for a conversation thread
 * Includes metadata for social sharing
 */
export function getShareableConversationUrl(
  workId: string, 
  threadId: string, 
  workTitle?: string
): {
  url: string;
  title: string;
  description: string;
} {
  const url = getConversationUrl(workId, threadId);
  const title = workTitle 
    ? `Conversation in "${workTitle}"`
    : 'Comment Conversation';
  
  return {
    url,
    title,
    description: `Join the discussion on this work's comment thread.`
  };
}

/**
 * URL validation utilities
 */
export const urlValidation = {
  /**
   * Validate UUID format for IDs
   */
  isValidUUID(id: string): boolean {
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
    return uuidRegex.test(id);
  },

  /**
   * Validate work ID
   */
  isValidWorkId(workId: string): boolean {
    return this.isValidUUID(workId);
  },

  /**
   * Validate thread/comment ID
   */
  isValidThreadId(threadId: string): boolean {
    return this.isValidUUID(threadId);
  },

  /**
   * Validate username format
   */
  isValidUsername(username: string): boolean {
    // Basic username validation - adjust as needed
    return /^[a-zA-Z0-9_-]{3,50}$/.test(username);
  }
};

/**
 * Browser navigation utilities
 */
export const navigation = {
  /**
   * Navigate to a conversation with proper history handling
   */
  toConversation(workId: string, threadId: string, replace = false): void {
    const url = getConversationUrl(workId, threadId);
    if (replace) {
      window.history.replaceState(null, '', url);
    } else {
      window.history.pushState(null, '', url);
    }
  },

  /**
   * Navigate back to work comments
   */
  toWorkComments(workId: string, replace = false): void {
    const url = getWorkCommentsUrl(workId);
    if (replace) {
      window.history.replaceState(null, '', url);
    } else {
      window.history.pushState(null, '', url);
    }
  },

  /**
   * Navigate back to work
   */
  toWork(workId: string, replace = false): void {
    const url = getWorkUrl(workId);
    if (replace) {
      window.history.replaceState(null, '', url);
    } else {
      window.history.pushState(null, '', url);
    }
  },

  /**
   * Check if browser supports sharing API
   */
  canShare(): boolean {
    return typeof navigator !== 'undefined' && 'share' in navigator;
  },

  /**
   * Share a conversation URL using Web Share API or clipboard
   */
  async shareConversation(
    workId: string, 
    threadId: string, 
    workTitle?: string
  ): Promise<boolean> {
    const shareData = getShareableConversationUrl(workId, threadId, workTitle);
    
    if (this.canShare()) {
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
  }
};

/**
 * Route analysis utilities
 */
export const routes = {
  /**
   * Analyze current path to determine context
   */
  getCurrentContext(): {
    type: 'work' | 'conversation' | 'comments' | 'other';
    workId?: string;
    threadId?: string;
  } {
    if (typeof window === 'undefined') {
      return { type: 'other' };
    }
    
    const path = window.location.pathname;
    
    // Conversation URL
    const conversationMatch = path.match(/\/works\/([^\/]+)\/comments\/([^\/]+)/);
    if (conversationMatch) {
      return {
        type: 'conversation',
        workId: conversationMatch[1],
        threadId: conversationMatch[2]
      };
    }
    
    // Work URL with comments hash
    const workMatch = path.match(/\/works\/([^\/]+)/);
    if (workMatch) {
      return {
        type: window.location.hash === '#comments' ? 'comments' : 'work',
        workId: workMatch[1]
      };
    }
    
    return { type: 'other' };
  },

  /**
   * Check if user is currently viewing a conversation
   */
  isInConversation(): boolean {
    return this.getCurrentContext().type === 'conversation';
  },

  /**
   * Get the current work ID if viewing a work-related page
   */
  getCurrentWorkId(): string | null {
    const context = this.getCurrentContext();
    return context.workId || null;
  },

  /**
   * Get the current thread ID if viewing a conversation
   */
  getCurrentThreadId(): string | null {
    const context = this.getCurrentContext();
    return context.threadId || null;
  }
};

export default {
  getWorkUrl,
  getWorkCommentsUrl,
  getConversationUrl,
  getChapterUrl,
  getUserUrl,
  getSeriesUrl,
  getCollectionUrl,
  getSearchUrl,
  parseConversationUrl,
  isConversationUrl,
  getShareableConversationUrl,
  urlValidation,
  navigation,
  routes
};