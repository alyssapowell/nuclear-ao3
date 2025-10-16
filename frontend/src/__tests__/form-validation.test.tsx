import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import LoginForm from '../components/auth/LoginForm';
import RegistrationForm from '../components/auth/RegistrationForm';
import WorkForm from '../components/works/WorkForm';
import ProfileForm from '../components/profile/ProfileForm';
import CommentForm from '../components/comments/CommentForm';

// Mock dependencies
jest.mock('../contexts/AuthContext', () => ({
  useAuth: jest.fn(() => ({
    user: null,
    isLoading: false,
    isAuthenticated: false,
    login: jest.fn(),
    logout: jest.fn(),
    register: jest.fn(),
  })),
}));

jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
    back: jest.fn(),
    forward: jest.fn(),
    refresh: jest.fn(),
    replace: jest.fn(),
    prefetch: jest.fn(),
  }),
}));

describe('Form Validation and Error Handling Tests', () => {
  describe('LoginForm Validation', () => {
    test('validates required username field', async () => {
      const user = userEvent.setup();
      render(<LoginForm />);

      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/username is required/i)).toBeInTheDocument();
      });
    });

    test('validates required password field', async () => {
      const user = userEvent.setup();
      render(<LoginForm />);

      const usernameField = screen.getByLabelText(/username/i);
      await user.type(usernameField, 'testuser');

      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/password is required/i)).toBeInTheDocument();
      });
    });

    test('validates minimum username length', async () => {
      const user = userEvent.setup();
      render(<LoginForm />);

      const usernameField = screen.getByLabelText(/username/i);
      await user.type(usernameField, 'ab'); // Too short

      fireEvent.blur(usernameField);

      await waitFor(() => {
        expect(screen.getByText(/username must be at least 3 characters/i)).toBeInTheDocument();
      });
    });

    test('validates maximum username length', async () => {
      const user = userEvent.setup();
      render(<LoginForm />);

      const usernameField = screen.getByLabelText(/username/i);
      await user.type(usernameField, 'a'.repeat(51)); // Too long

      fireEvent.blur(usernameField);

      await waitFor(() => {
        expect(screen.getByText(/username cannot exceed 50 characters/i)).toBeInTheDocument();
      });
    });

    test('validates username contains only allowed characters', async () => {
      const user = userEvent.setup();
      render(<LoginForm />);

      const usernameField = screen.getByLabelText(/username/i);
      await user.type(usernameField, 'user@name!'); // Invalid characters

      fireEvent.blur(usernameField);

      await waitFor(() => {
        expect(screen.getByText(/username can only contain letters, numbers, and underscores/i)).toBeInTheDocument();
      });
    });

    test('validates minimum password length', async () => {
      const user = userEvent.setup();
      render(<LoginForm />);

      const passwordField = screen.getByLabelText(/password/i);
      await user.type(passwordField, '123'); // Too short

      fireEvent.blur(passwordField);

      await waitFor(() => {
        expect(screen.getByText(/password must be at least 8 characters/i)).toBeInTheDocument();
      });
    });

    test('clears validation errors when fields are corrected', async () => {
      const user = userEvent.setup();
      render(<LoginForm />);

      // Trigger validation error
      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/username is required/i)).toBeInTheDocument();
      });

      // Fix the error
      const usernameField = screen.getByLabelText(/username/i);
      await user.type(usernameField, 'validuser');

      await waitFor(() => {
        expect(screen.queryByText(/username is required/i)).not.toBeInTheDocument();
      });
    });
  });

  describe('RegistrationForm Validation', () => {
    test('validates email format', async () => {
      const user = userEvent.setup();
      render(<RegistrationForm />);

      const emailField = screen.getByLabelText(/email/i);
      await user.type(emailField, 'invalid-email');

      fireEvent.blur(emailField);

      await waitFor(() => {
        expect(screen.getByText(/please enter a valid email address/i)).toBeInTheDocument();
      });
    });

    test('validates password confirmation match', async () => {
      const user = userEvent.setup();
      render(<RegistrationForm />);

      const passwordField = screen.getByLabelText(/^password$/i);
      const confirmField = screen.getByLabelText(/confirm password/i);

      await user.type(passwordField, 'password123');
      await user.type(confirmField, 'password456');

      fireEvent.blur(confirmField);

      await waitFor(() => {
        expect(screen.getByText(/passwords do not match/i)).toBeInTheDocument();
      });
    });

    test('validates password strength requirements', async () => {
      const user = userEvent.setup();
      render(<RegistrationForm />);

      const passwordField = screen.getByLabelText(/^password$/i);
      
      // Test weak password
      await user.type(passwordField, 'weak');
      fireEvent.blur(passwordField);

      await waitFor(() => {
        expect(screen.getByText(/password must contain at least one uppercase letter/i)).toBeInTheDocument();
      });

      // Clear and test another requirement
      await user.clear(passwordField);
      await user.type(passwordField, 'NoNumbers');
      fireEvent.blur(passwordField);

      await waitFor(() => {
        expect(screen.getByText(/password must contain at least one number/i)).toBeInTheDocument();
      });
    });

    test('validates terms of service acceptance', async () => {
      const user = userEvent.setup();
      render(<RegistrationForm />);

      // Fill in all fields except terms
      await user.type(screen.getByLabelText(/username/i), 'testuser');
      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/^password$/i), 'Password123');
      await user.type(screen.getByLabelText(/confirm password/i), 'Password123');

      const submitButton = screen.getByRole('button', { name: /register/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/you must accept the terms of service/i)).toBeInTheDocument();
      });
    });

    test('validates age confirmation for mature content', async () => {
      const user = userEvent.setup();
      render(<RegistrationForm />);

      // Fill in required fields
      await user.type(screen.getByLabelText(/username/i), 'testuser');
      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/^password$/i), 'Password123');
      await user.type(screen.getByLabelText(/confirm password/i), 'Password123');
      
      const termsCheckbox = screen.getByLabelText(/accept terms/i);
      await user.click(termsCheckbox);

      const submitButton = screen.getByRole('button', { name: /register/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/you must confirm you are 13 years or older/i)).toBeInTheDocument();
      });
    });
  });

  describe('WorkForm Validation', () => {
    test('validates required title field', async () => {
      const user = userEvent.setup();
      render(<WorkForm />);

      const submitButton = screen.getByRole('button', { name: /publish|save/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/title is required/i)).toBeInTheDocument();
      });
    });

    test('validates title length limits', async () => {
      const user = userEvent.setup();
      render(<WorkForm />);

      const titleField = screen.getByLabelText(/title/i);
      await user.type(titleField, 'a'.repeat(256)); // Too long

      fireEvent.blur(titleField);

      await waitFor(() => {
        expect(screen.getByText(/title cannot exceed 255 characters/i)).toBeInTheDocument();
      });
    });

    test('validates required summary field', async () => {
      const user = userEvent.setup();
      render(<WorkForm />);

      const titleField = screen.getByLabelText(/title/i);
      await user.type(titleField, 'Valid Title');

      const submitButton = screen.getByRole('button', { name: /publish|save/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/summary is required/i)).toBeInTheDocument();
      });
    });

    test('validates summary length limits', async () => {
      const user = userEvent.setup();
      render(<WorkForm />);

      const summaryField = screen.getByLabelText(/summary/i);
      await user.type(summaryField, 'a'.repeat(1251)); // Too long

      fireEvent.blur(summaryField);

      await waitFor(() => {
        expect(screen.getByText(/summary cannot exceed 1250 characters/i)).toBeInTheDocument();
      });
    });

    test('validates required fandom tags', async () => {
      const user = userEvent.setup();
      render(<WorkForm />);

      await user.type(screen.getByLabelText(/title/i), 'Valid Title');
      await user.type(screen.getByLabelText(/summary/i), 'Valid summary');

      const submitButton = screen.getByRole('button', { name: /publish|save/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/at least one fandom tag is required/i)).toBeInTheDocument();
      });
    });

    test('validates work content is not empty', async () => {
      const user = userEvent.setup();
      render(<WorkForm />);

      await user.type(screen.getByLabelText(/title/i), 'Valid Title');
      await user.type(screen.getByLabelText(/summary/i), 'Valid summary');

      // Add fandom tag
      const fandomInput = screen.getByLabelText(/fandom/i);
      await user.type(fandomInput, 'Test Fandom');
      await user.keyboard('{Enter}');

      const submitButton = screen.getByRole('button', { name: /publish|save/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/work content cannot be empty/i)).toBeInTheDocument();
      });
    });

    test('validates word count limits', async () => {
      const user = userEvent.setup();
      render(<WorkForm />);

      const contentField = screen.getByLabelText(/content|work text/i);
      const longContent = 'word '.repeat(100001); // Exceeds limit
      
      await user.type(contentField, longContent);
      fireEvent.blur(contentField);

      await waitFor(() => {
        expect(screen.getByText(/work exceeds maximum word count/i)).toBeInTheDocument();
      });
    });

    test('validates archive warnings selection', async () => {
      const user = userEvent.setup();
      render(<WorkForm />);

      // Fill required fields
      await user.type(screen.getByLabelText(/title/i), 'Valid Title');
      await user.type(screen.getByLabelText(/summary/i), 'Valid summary');
      await user.type(screen.getByLabelText(/content|work text/i), 'Valid content');

      const submitButton = screen.getByRole('button', { name: /publish|save/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/please select archive warnings/i)).toBeInTheDocument();
      });
    });
  });

  describe('ProfileForm Validation', () => {
    test('validates bio length limits', async () => {
      const user = userEvent.setup();
      render(<ProfileForm />);

      const bioField = screen.getByLabelText(/bio|about/i);
      await user.type(bioField, 'a'.repeat(5001)); // Too long

      fireEvent.blur(bioField);

      await waitFor(() => {
        expect(screen.getByText(/bio cannot exceed 5000 characters/i)).toBeInTheDocument();
      });
    });

    test('validates website URL format', async () => {
      const user = userEvent.setup();
      render(<ProfileForm />);

      const websiteField = screen.getByLabelText(/website/i);
      await user.type(websiteField, 'not-a-url');

      fireEvent.blur(websiteField);

      await waitFor(() => {
        expect(screen.getByText(/please enter a valid URL/i)).toBeInTheDocument();
      });
    });

    test('validates social media handles format', async () => {
      const user = userEvent.setup();
      render(<ProfileForm />);

      const twitterField = screen.getByLabelText(/twitter/i);
      await user.type(twitterField, 'invalid@handle!');

      fireEvent.blur(twitterField);

      await waitFor(() => {
        expect(screen.getByText(/invalid twitter handle format/i)).toBeInTheDocument();
      });
    });
  });

  describe('CommentForm Validation', () => {
    test('validates required comment content', async () => {
      const user = userEvent.setup();
      render(<CommentForm workId="test-work" />);

      const submitButton = screen.getByRole('button', { name: /post comment/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/comment cannot be empty/i)).toBeInTheDocument();
      });
    });

    test('validates comment length limits', async () => {
      const user = userEvent.setup();
      render(<CommentForm workId="test-work" />);

      const commentField = screen.getByLabelText(/comment/i);
      await user.type(commentField, 'a'.repeat(10001)); // Too long

      fireEvent.blur(commentField);

      await waitFor(() => {
        expect(screen.getByText(/comment cannot exceed 10000 characters/i)).toBeInTheDocument();
      });
    });

    test('validates appropriate content guidelines', async () => {
      const user = userEvent.setup();
      render(<CommentForm workId="test-work" />);

      const commentField = screen.getByLabelText(/comment/i);
      await user.type(commentField, 'This contains spam links: http://spam-site.com');

      fireEvent.blur(commentField);

      await waitFor(() => {
        expect(screen.getByText(/comments with external links require review/i)).toBeInTheDocument();
      });
    });
  });

  describe('Error Handling and Recovery', () => {
    test('handles network errors gracefully', async () => {
      const user = userEvent.setup();
      
      // Mock network failure
      global.fetch = jest.fn().mockRejectedValue(new Error('Network error'));

      render(<LoginForm />);

      await user.type(screen.getByLabelText(/username/i), 'testuser');
      await user.type(screen.getByLabelText(/password/i), 'password123');

      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/network error.*try again/i)).toBeInTheDocument();
      });
    });

    test('handles server validation errors', async () => {
      const user = userEvent.setup();
      
      // Mock server validation error
      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({
          errors: {
            username: 'Username is already taken',
            email: 'Invalid email address'
          }
        })
      });

      render(<RegistrationForm />);

      // Fill and submit form
      await user.type(screen.getByLabelText(/username/i), 'takenuser');
      await user.type(screen.getByLabelText(/email/i), 'invalid@email');
      await user.type(screen.getByLabelText(/^password$/i), 'Password123');
      await user.type(screen.getByLabelText(/confirm password/i), 'Password123');
      await user.click(screen.getByLabelText(/accept terms/i));
      await user.click(screen.getByLabelText(/13 years or older/i));

      const submitButton = screen.getByRole('button', { name: /register/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/username is already taken/i)).toBeInTheDocument();
        expect(screen.getByText(/invalid email address/i)).toBeInTheDocument();
      });
    });

    test('preserves form data during validation errors', async () => {
      const user = userEvent.setup();
      render(<WorkForm />);

      // Fill form with valid data
      const titleValue = 'My Test Work';
      const summaryValue = 'This is a test summary';
      
      await user.type(screen.getByLabelText(/title/i), titleValue);
      await user.type(screen.getByLabelText(/summary/i), summaryValue);

      // Trigger validation error by submitting without required fields
      const submitButton = screen.getByRole('button', { name: /publish|save/i });
      await user.click(submitButton);

      // Form data should be preserved
      expect(screen.getByDisplayValue(titleValue)).toBeInTheDocument();
      expect(screen.getByDisplayValue(summaryValue)).toBeInTheDocument();
    });

    test('provides helpful error messages for common mistakes', async () => {
      const user = userEvent.setup();
      render(<LoginForm />);

      // Test empty form submission
      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/please fill in all required fields/i)).toBeInTheDocument();
      });
    });

    test('handles timeout errors appropriately', async () => {
      const user = userEvent.setup();
      
      // Mock timeout
      global.fetch = jest.fn().mockImplementation(() => 
        new Promise((_, reject) => 
          setTimeout(() => reject(new Error('Request timeout')), 100)
        )
      );

      render(<LoginForm />);

      await user.type(screen.getByLabelText(/username/i), 'testuser');
      await user.type(screen.getByLabelText(/password/i), 'password123');

      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/request timed out.*please try again/i)).toBeInTheDocument();
      }, { timeout: 3000 });
    });

    test('allows retry after errors', async () => {
      const user = userEvent.setup();
      
      let callCount = 0;
      global.fetch = jest.fn().mockImplementation(() => {
        callCount++;
        if (callCount === 1) {
          return Promise.reject(new Error('Network error'));
        }
        return Promise.resolve({
          ok: true,
          json: async () => ({ success: true })
        });
      });

      render(<LoginForm />);

      await user.type(screen.getByLabelText(/username/i), 'testuser');
      await user.type(screen.getByLabelText(/password/i), 'password123');

      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      // First attempt fails
      await waitFor(() => {
        expect(screen.getByText(/network error/i)).toBeInTheDocument();
      });

      // Retry should succeed
      const retryButton = screen.getByRole('button', { name: /try again|retry/i });
      await user.click(retryButton);

      await waitFor(() => {
        expect(screen.queryByText(/network error/i)).not.toBeInTheDocument();
      });
    });
  });

  describe('Accessibility in Form Validation', () => {
    test('associates error messages with form fields using aria-describedby', async () => {
      const user = userEvent.setup();
      render(<LoginForm />);

      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      await waitFor(() => {
        const usernameField = screen.getByLabelText(/username/i);
        const errorId = usernameField.getAttribute('aria-describedby');
        expect(errorId).toBeTruthy();
        expect(document.getElementById(errorId!)).toHaveTextContent(/username is required/i);
      });
    });

    test('sets aria-invalid on fields with errors', async () => {
      const user = userEvent.setup();
      render(<LoginForm />);

      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      await waitFor(() => {
        const usernameField = screen.getByLabelText(/username/i);
        expect(usernameField).toHaveAttribute('aria-invalid', 'true');
      });
    });

    test('announces validation errors to screen readers', async () => {
      const user = userEvent.setup();
      render(<LoginForm />);

      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      await waitFor(() => {
        const alertRegion = screen.getByRole('alert');
        expect(alertRegion).toBeInTheDocument();
        expect(alertRegion).toHaveTextContent(/username is required/i);
      });
    });

    test('removes aria-invalid when errors are resolved', async () => {
      const user = userEvent.setup();
      render(<LoginForm />);

      // Trigger error
      const submitButton = screen.getByRole('button', { name: /login/i });
      await user.click(submitButton);

      await waitFor(() => {
        const usernameField = screen.getByLabelText(/username/i);
        expect(usernameField).toHaveAttribute('aria-invalid', 'true');
      });

      // Fix error
      const usernameField = screen.getByLabelText(/username/i);
      await user.type(usernameField, 'validuser');

      await waitFor(() => {
        expect(usernameField).toHaveAttribute('aria-invalid', 'false');
      });
    });
  });
});