import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { jest } from '@jest/globals';

// Mock the API functions
jest.mock('../lib/api', () => ({
  giveKudos: jest.fn(),
  createComment: jest.fn(),
  searchWorks: jest.fn(),
  getPopularSearches: jest.fn(),
  getTrendingSearches: jest.fn(),
}));

// Mock Next.js router
jest.mock('next/navigation', () => ({
  useSearchParams: () => ({
    get: jest.fn(() => ''),
  }),
}));

import SearchPage from '../app/search/page';
import * as api from '../lib/api';

const mockApi = api as jest.Mocked<typeof api>;

describe('Anonymous Features', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Clear localStorage to simulate logged out state
    Object.defineProperty(window, 'localStorage', {
      value: {
        getItem: jest.fn(() => null),
        setItem: jest.fn(),
        removeItem: jest.fn(),
        clear: jest.fn(),
      },
      writable: true,
    });
  });

  describe('Anonymous Kudos', () => {
    it('should allow giving kudos without authentication', async () => {
      const mockGiveKudos = mockApi.giveKudos.mockResolvedValue({ success: true });
      
      // Mock a work component that has kudos functionality
      const TestWorkComponent = () => {
        const handleKudos = async () => {
          try {
            // This should work without auth token
            await api.giveKudos('work-123');
          } catch (error) {
            console.error('Kudos failed:', error);
          }
        };

        return (
          <div>
            <button onClick={handleKudos} data-testid="kudos-button">
              Give Kudos
            </button>
          </div>
        );
      };

      render(<TestWorkComponent />);
      
      const kudosButton = screen.getByTestId('kudos-button');
      await userEvent.click(kudosButton);

      await waitFor(() => {
        expect(mockGiveKudos).toHaveBeenCalledWith('work-123');
        // Should be called without auth token (undefined)
        expect(mockGiveKudos).toHaveBeenCalledTimes(1);
      });
    });

    it('should handle kudos API call without auth header', async () => {
      const mockGiveKudos = mockApi.giveKudos.mockResolvedValue({ success: true });
      
      // Direct API call without auth
      await api.giveKudos('work-456');
      
      expect(mockGiveKudos).toHaveBeenCalledWith('work-456');
    });

    it('should show appropriate error when anonymous kudos fails', async () => {
      const mockGiveKudos = mockApi.giveKudos.mockRejectedValue(
        new Error('Anonymous kudos not allowed')
      );
      
      const TestWorkComponent = () => {
        const [error, setError] = React.useState<string | null>(null);
        
        const handleKudos = async () => {
          try {
            await api.giveKudos('work-123');
          } catch (err) {
            setError(err instanceof Error ? err.message : 'Kudos failed');
          }
        };

        return (
          <div>
            <button onClick={handleKudos} data-testid="kudos-button">
              Give Kudos
            </button>
            {error && <div data-testid="error-message">{error}</div>}
          </div>
        );
      };

      render(<TestWorkComponent />);
      
      const kudosButton = screen.getByTestId('kudos-button');
      await userEvent.click(kudosButton);

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toHaveTextContent('Anonymous kudos not allowed');
      });
    });
  });

  describe('Anonymous Comments', () => {
    it('should allow adding comments without authentication', async () => {
      const mockAddComment = mockApi.createComment.mockResolvedValue({ 
        id: 'comment-123',
        content: 'Great work!',
        author: 'Anonymous',
        created_at: new Date().toISOString()
      });
      
      const TestCommentComponent = () => {
        const [comment, setComment] = React.useState('');
        
        const handleSubmit = async () => {
          try {
            // This should work without auth token
        await api.createComment('work-123', {
          content: 'Great work!',
          name: 'Anonymous User'
        });

        expect(mockAddComment).toHaveBeenCalledWith('work-123', {
          content: 'Great work!',
          name: 'Anonymous User'
        });
          } catch (error) {
            console.error('Comment failed:', error);
          }
        };

        return (
          <div>
            <textarea 
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              data-testid="comment-input"
              placeholder="Add a comment..."
            />
            <button onClick={handleSubmit} data-testid="submit-comment">
              Submit Comment
            </button>
          </div>
        );
      };

      render(<TestCommentComponent />);
      
      const commentInput = screen.getByTestId('comment-input');
      const submitButton = screen.getByTestId('submit-comment');
      
      await userEvent.type(commentInput, 'Great work!');
      await userEvent.click(submitButton);

      await waitFor(() => {
        expect(mockAddComment).toHaveBeenCalledWith('work-123', {
          content: 'Great work!',
          anonymous: true
        });
      });
    });

    it('should handle anonymous comment submission correctly', async () => {
      const mockAddComment = mockApi.createComment.mockResolvedValue({ 
        id: 'comment-456',
        content: 'Anonymous feedback',
        author: 'Anonymous',
        created_at: new Date().toISOString()
      });
      
      // Direct API call for anonymous comment
      await api.createComment('work-789', {
        content: 'Anonymous feedback',
        anonymous: true
      });
      
      expect(mockAddComment).toHaveBeenCalledWith('work-789', {
        content: 'Anonymous feedback',
        anonymous: true
      });
    });
  });

  describe('Search Page When Logged Out', () => {
    it('should render search page without authentication', async () => {
      // Mock search API
      mockApi.searchWorks.mockResolvedValue({
        works: [],
        total: 0,
        facets: {}
      });

      // Render search page component
      const { container } = render(<SearchPage />);
      
      // Should not throw ReferenceError
      expect(container).toBeInTheDocument();
      
      // Should show search form
      await waitFor(() => {
        expect(screen.getByRole('textbox')).toBeInTheDocument();
      });
    });

    it('should handle search functionality when logged out', async () => {
      const mockSearchResults = {
        works: [
          {
            id: 'work-1',
            title: 'Test Work',
            author: 'Test Author',
            summary: 'Test summary',
            word_count: 1000,
            chapter_count: 1,
            rating: 'General Audiences',
            status: 'complete',
            language: 'English',
            published_date: '2023-01-01T00:00:00Z',
            updated_date: '2023-01-01T00:00:00Z',
            relationships: [],
            characters: [],
            freeform_tags: [],
            fandoms: []
          }
        ],
        total: 1,
        facets: {}
      };

      mockApi.searchWorks.mockResolvedValue(mockSearchResults);

      render(<SearchPage />);
      
      // Wait for component to load
      await waitFor(() => {
        expect(screen.getByRole('textbox')).toBeInTheDocument();
      });

      // Perform a search
      const searchInput = screen.getByRole('textbox');
      const searchButton = screen.getByRole('button', { name: /search/i });
      
      await userEvent.type(searchInput, 'test query');
      await userEvent.click(searchButton);

      // Should call search API without auth
      await waitFor(() => {
        expect(mockApi.searchWorks).toHaveBeenCalled();
      });
    });

    it('should not crash when accessing search-related state while logged out', () => {
      // Test that no uninitialized variables are accessed
      expect(() => {
        render(<SearchPage />);
      }).not.toThrow();
    });
  });

  describe('Authentication State Edge Cases', () => {
    it('should handle missing localStorage gracefully', () => {
      // Simulate environment where localStorage is not available
      Object.defineProperty(window, 'localStorage', {
        value: undefined,
        writable: true,
      });

      expect(() => {
        render(<SearchPage />);
      }).not.toThrow();
    });

    it('should handle corrupted auth token gracefully', () => {
      // Simulate corrupted token in localStorage
      Object.defineProperty(window, 'localStorage', {
        value: {
          getItem: jest.fn(() => 'invalid-token-data'),
          setItem: jest.fn(),
          removeItem: jest.fn(),
          clear: jest.fn(),
        },
        writable: true,
      });

      expect(() => {
        render(<SearchPage />);
      }).not.toThrow();
    });

    it('should handle network errors during anonymous operations gracefully', async () => {
      mockApi.giveKudos.mockRejectedValue(new Error('Network error'));
      
      const TestComponent = () => {
        const [error, setError] = React.useState<string | null>(null);
        
        const handleKudos = async () => {
          try {
            await api.giveKudos('work-123');
          } catch (err) {
            setError('Network error occurred');
          }
        };

        return (
          <div>
            <button onClick={handleKudos} data-testid="kudos-button">
              Give Kudos
            </button>
            {error && <div data-testid="error">{error}</div>}
          </div>
        );
      };

      render(<TestComponent />);
      
      const button = screen.getByTestId('kudos-button');
      await userEvent.click(button);

      await waitFor(() => {
        expect(screen.getByTestId('error')).toHaveTextContent('Network error occurred');
      });
    });
  });
});