// Stub auth library for testing PWA functionality
'use client';

import { createContext, useContext, ReactNode } from 'react';

export interface User {
  id: string;
  username: string;
  email: string;
}

export interface Work {
  id: string;
  title: string;
  summary: string;
  wordCount: number;
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface Bookmark {
  id: string;
  workId: string;
  work: Work;
  notes: string;
  isPrivate: boolean;
  createdAt: string;
}

interface AuthContext {
  user: User | null;
  isLoading: boolean;
  token: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContext | null>(null);

export function useAuth(): AuthContext {
  const context = useContext(AuthContext);
  if (!context) {
    // Return mock auth context for testing
    return {
      user: null,
      isLoading: false,
      token: null,
      login: async () => {},
      logout: async () => {}
    };
  }
  return context;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  return (
    <AuthContext.Provider value={{
      user: null,
      isLoading: false,
      token: null,
      login: async () => {},
      logout: async () => {}
    }}>
      {children}
    </AuthContext.Provider>
  );
}

// Stub API functions
export async function getMyWorks(): Promise<Work[]> {
  // Return mock works for testing
  return [
    {
      id: "1",
      title: "Sample Work 1",
      summary: "A test work for PWA testing",
      wordCount: 1500,
      status: "completed",
      createdAt: "2024-01-01T00:00:00Z",
      updatedAt: "2024-01-01T00:00:00Z"
    },
    {
      id: "2", 
      title: "Sample Work 2",
      summary: "Another test work",
      wordCount: 2500,
      status: "in_progress",
      createdAt: "2024-01-02T00:00:00Z",
      updatedAt: "2024-01-02T00:00:00Z"
    }
  ];
}

export async function getUserBookmarks(): Promise<Bookmark[]> {
  const works = await getMyWorks();
  return [
    {
      id: "1",
      workId: "1",
      work: works[0],
      notes: "Great read!",
      isPrivate: false,
      createdAt: "2024-01-01T00:00:00Z"
    }
  ];
}