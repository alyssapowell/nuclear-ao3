import type { Mock } from 'vitest';
import React from 'react';
import { render, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { jest } from '@jest/globals';

// Mock Next.js router

import LoginPage from '../pages/Login';

describe('Login Page Hooks Error', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    
    // Mock fetch for login API
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ 
          access_token: 'test-token',
          user: { id: '1', username: 'testuser' }
        }),
      })
    ) as Mock;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should render login form without hooks error', () => {
    const { getByText, getByLabelText } = render(<LoginPage />);
    
    expect(getByText('Sign in to your account')).toBeInTheDocument();
    expect(getByLabelText(/email/i)).toBeInTheDocument();
    expect(getByLabelText(/password/i)).toBeInTheDocument();
  });

  it('should handle form submission without hooks error', async () => {
    const { getByLabelText, getByRole } = render(<LoginPage />);
    
    const emailInput = getByLabelText(/email/i);
    const passwordInput = getByLabelText(/password/i);
    const submitButton = getByRole('button', { name: /sign in/i });

    // Fill out form
    fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
    fireEvent.change(passwordInput, { target: { value: 'password123' } });

    // Submit form - this should trigger the hooks error if it exists
    fireEvent.click(submitButton);

    // Wait for any async operations
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalled();
    });
  });
});