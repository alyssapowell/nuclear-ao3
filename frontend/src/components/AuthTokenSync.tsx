'use client';

import { useEffect } from 'react';

/**
 * AuthTokenSync - Client component that syncs localStorage auth tokens to cookies
 * This ensures the middleware can access authentication tokens for protected routes
 */
export default function AuthTokenSync() {
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const authToken = localStorage.getItem('auth_token');
      if (authToken && authToken !== 'test-token') {
        // Sync existing token to cookie for middleware
        document.cookie = `auth_token=${authToken}; path=/; max-age=${7 * 24 * 60 * 60}; SameSite=Lax`;
      }
    }
  }, []);

  return null; // This component renders nothing
}