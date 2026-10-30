import { useEffect, useState } from 'react';

/**
 * Authentication utilities
 * This provides authentication functions that work with localStorage-based auth
 */

export interface User {
  id: string;
  username: string;
  email: string;
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
 * Logout user (clear local storage)
 */
export function logout(): void {
  if (typeof window === 'undefined') return;
  
  localStorage.removeItem('auth_token');
  localStorage.removeItem('user');
  
  // Clear auth cookie
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
export function useAuth(): AuthState & { logout: () => void } {
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
