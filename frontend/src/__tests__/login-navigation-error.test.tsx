import type { Mock } from 'vitest';
import React from 'react';
import { render, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { jest } from '@jest/globals';

// Mock the router with actual push behavior
const mockPush = vi.fn();

vi.mock('react-router-dom', async () =>
  (await import('@/test/router-mock')).routerMock({
    useNavigate: () => mockPush,
    useSearchParams: () => [new URLSearchParams(), vi.fn()],
  }));

import LoginPage from '../pages/Login';

describe('Login Navigation Error', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    
    // Mock localStorage
    Object.defineProperty(window, 'localStorage', {
      value: {
        getItem: vi.fn((key) => {
          if (key === 'auth_token') return null;
          return null;
        }),
        setItem: vi.fn(),
        removeItem: vi.fn(),
        clear: vi.fn(),
      },
      writable: true,
    });

    // Mock document.cookie
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: '',
    });

    // Mock successful login response
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ 
          access_token: 'test-token-12345',
          user: { 
            id: '1', 
            username: 'testuser',
            email: 'test@example.com'
          }
        }),
      })
    ) as Mock;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should handle successful login and navigation without hooks error', async () => {
    const { getByLabelText, getByRole } = render(<LoginPage />);
    
    const emailInput = getByLabelText(/email/i);
    const passwordInput = getByLabelText(/password/i);
    const submitButton = getByRole('button', { name: /sign in/i });

    // Fill out form with test credentials
    fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
    fireEvent.change(passwordInput, { target: { value: 'password123' } });

    // Submit form - this should trigger the hooks error if it exists
    fireEvent.click(submitButton);

    // Wait for the login API call
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8081/api/v1/auth/login',
        expect.objectContaining({
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            email: 'test@example.com',
            password: 'password123'
          }),
        })
      );
    });

    // Wait for localStorage to be updated
    await waitFor(() => {
      expect(window.localStorage.setItem).toHaveBeenCalledWith('auth_token', 'test-token-12345');
    });

    // Check that router.push was called (this is where the error might occur)
    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith('/dashboard');
    });

    // If we get here without a hooks error, the test passes
    expect(mockPush).toHaveBeenCalledTimes(1);
  });
});