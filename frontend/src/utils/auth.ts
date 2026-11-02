import { useEffect, useState } from 'react';

/**
 * Authentication utilities
 * This provides authentication functions that work with localStorage-based auth
 */

/**
 * An account has an email address and pseuds; it has no username. The
 * default pseud is how the account is shown to its owner.
 */
export interface User {
  id: string;
  email: string;
  default_pseud?: { id: string; name: string };
}

/** The name to show for the signed-in person: the name of their default pseud. */
export function penNameOf(user?: Pick<User, 'default_pseud'> | null): string {
  return user?.default_pseud?.name ?? '';
}

export interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  token: string | null;
}

/**
 * Get current authentication state
 */
export function getAuthState(): AuthState {
  if (typeof window === 'undefined') {
    return {
      user: null,
      isAuthenticated: false,
      token: null
    };
  }

  const token = localStorage.getItem('auth_token');
  const userData = localStorage.getItem('user');
  
  let user: User | null = null;
  if (userData) {
    try {
      user = JSON.parse(userData);
    } catch (error) {
      console.warn('Failed to parse user data from localStorage:', error);
    }
  }

  return {
    user,
    isAuthenticated: !!token,
    token
  };
}

/**
 * Check if user is authenticated
 */
export function isAuthenticated(): boolean {
  if (typeof window === 'undefined') return false;
  const token = localStorage.getItem('auth_token');
  return !!token;
}

/**
 * Get current user
 */
export function getCurrentUser(): User | null {
  if (typeof window === 'undefined') return null;
  
  const userData = localStorage.getItem('user');
  if (!userData) return null;
  
  try {
    return JSON.parse(userData);
  } catch (error) {
    console.warn('Failed to parse user data from localStorage:', error);
    return null;
  }
}

/**
 * Get auth token
 */
export function getAuthToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('auth_token');
}

/**
 * Log out: tell the server to end this login (so the token stops working
 * everywhere), then clear it from this browser. A network failure doesn't
 * stop the local logout.
 */
export async function logout(): Promise<void> {
  if (typeof window === 'undefined') return;

  const token = localStorage.getItem('auth_token') || localStorage.getItem('token');
  if (token) {
    try {
      await fetch('/api/v1/auth/logout', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
    } catch {
      // Offline or server down: still log out here.
    }
  }

  localStorage.removeItem('auth_token');
  localStorage.removeItem('token');
  localStorage.removeItem('user');

  // Older versions also kept the token in a cookie the server never read; clear any leftover one
  document.cookie = 'auth_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT';

  // Trigger storage event for other tabs
  window.dispatchEvent(new Event('authChange'));
}

/**
 * The logged-in user and their token, for use in components.
 *
 * It returns the same objects until someone logs in or out (in this tab or
 * another), so it's safe to list `user` or `token` in a useEffect's
 * dependencies without the effect running on every render.
 */
export function useAuth(): AuthState & { logout: () => Promise<void> } {
  const [authState, setAuthState] = useState(getAuthState);

  useEffect(() => {
    const refresh = () => setAuthState(getAuthState());
    window.addEventListener('authChange', refresh);
    window.addEventListener('storage', refresh);
    return () => {
      window.removeEventListener('authChange', refresh);
      window.removeEventListener('storage', refresh);
    };
  }, []);

  return { ...authState, logout };
}

/**
 * Where to send someone after they log in or sign up, from a `?redirect=`
 * value. Only addresses on this site are allowed: anything else (another
 * site, `//other.site`, `javascript:`) becomes the home page, so a link can't
 * use the login page to send people elsewhere or run code.
 */
export function safeRedirect(raw: string | null): string {
  if (!raw) return '/';
  try {
    const url = new URL(raw, window.location.origin);
    if (url.origin !== window.location.origin) return '/';
    const path = url.pathname + url.search + url.hash;
    // Tidying "/.//other.site" gives "//other.site", which browsers read as
    // another site, so the result must start with exactly one slash.
    if (!path.startsWith('/') || path.startsWith('//') || path.startsWith('/\\')) return '/';
    return path;
  } catch {
    return '/';
  }
}
