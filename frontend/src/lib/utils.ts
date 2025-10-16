// Utility functions for Nuclear AO3 frontend
import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';
import React from 'react';

// Combine Tailwind classes properly
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// Format word count with commas
export function formatWordCount(count: number): string {
  return count.toLocaleString();
}

// Format dates relative to now
export function formatRelativeDate(date: string): string {
  const now = new Date();
  const past = new Date(date);
  const diffInSeconds = Math.floor((now.getTime() - past.getTime()) / 1000);

  if (diffInSeconds < 60) return 'just now';
  if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}m ago`;
  if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}h ago`;
  if (diffInSeconds < 2592000) return `${Math.floor(diffInSeconds / 86400)}d ago`;
  if (diffInSeconds < 31536000) return `${Math.floor(diffInSeconds / 2592000)}mo ago`;
  return `${Math.floor(diffInSeconds / 31536000)}y ago`;
}

// Format absolute dates
export function formatDate(date: string): string {
  return new Date(date).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

// Extract reading time estimate
export function getReadingTime(wordCount: number): string {
  const wordsPerMinute = 250; // Average reading speed
  const minutes = Math.ceil(wordCount / wordsPerMinute);
  
  if (minutes < 60) {
    return `${minutes} min read`;
  } else {
    const hours = Math.floor(minutes / 60);
    const remainingMinutes = minutes % 60;
    return remainingMinutes > 0 
      ? `${hours}h ${remainingMinutes}m read`
      : `${hours}h read`;
  }
}

// Truncate text with ellipsis
export function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength).trim() + '...';
}

// Rating display helpers
export const RATING_COLORS = {
  'General Audiences': 'bg-green-100 text-green-800',
  'Teen And Up Audiences': 'bg-blue-100 text-blue-800', 
  'Mature': 'bg-orange-100 text-orange-800',
  'Explicit': 'bg-red-100 text-red-800',
  'Not Rated': 'bg-gray-100 text-gray-800',
} as const;

export function getRatingColor(rating: string): string {
  return RATING_COLORS[rating as keyof typeof RATING_COLORS] || RATING_COLORS['Not Rated'];
}

// URL helpers for client-side navigation
export function getWorkUrl(workId: string): string {
  return `/works/${workId}`;
}

export function getUserUrl(userId: string): string {
  return `/users/${userId}`;
}

export function getChapterUrl(workId: string, chapterNumber: number): string {
  return `/works/${workId}/chapters/${chapterNumber}`;
}

// Local storage helpers with error handling
export function getStoredValue<T>(key: string, defaultValue: T): T {
  if (typeof window === 'undefined') return defaultValue;
  
  try {
    const item = localStorage.getItem(key);
    return item ? JSON.parse(item) : defaultValue;
  } catch {
    return defaultValue;
  }
}

export function setStoredValue<T>(key: string, value: T): void {
  if (typeof window === 'undefined') return;
  
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Silently fail if localStorage is not available
  }
}

// Hydration safety helper
export function useIsClient(): boolean {
  const [isClient, setIsClient] = React.useState(false);
  
  React.useEffect(() => {
    setIsClient(true);
  }, []);
  
  return isClient;
}

// Error boundary helper
export function isApiError(error: unknown): error is { message: string; status?: number } {
  return error !== null && typeof error === 'object' && 'message' in error && typeof (error as { message: unknown }).message === 'string';
}