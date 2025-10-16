import React from 'react';
import { render, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { jest } from '@jest/globals';

// Mock Next.js router
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
    replace: jest.fn(),
    back: jest.fn(),
  }),
}));

import LoginPage from '../app/auth/login/page';

describe('Login Page Hooks Error', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    
    // Mock fetch for login API
    global.fetch = jest.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ 
          access_token: 'test-token',
          user: { id: '1', username: 'testuser' }
        }),
      })
    ) as jest.Mock;
  });

  afterEach(() => {
    jest.restoreAllMocks();
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