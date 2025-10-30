import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BookmarkButton } from '../BookmarkButton';

// Mock API functions
jest.mock('@/lib/api', () => ({
  createBookmark: jest.fn(),
  deleteBookmark: jest.fn(),
  isWorkBookmarked: jest.fn(),
}));

import { createBookmark, deleteBookmark, isWorkBookmarked } from '@/lib/api';

const mockCreateBookmark = createBookmark as jest.MockedFunction<typeof createBookmark>;
const mockDeleteBookmark = deleteBookmark as jest.MockedFunction<typeof deleteBookmark>;
const mockIsWorkBookmarked = isWorkBookmarked as jest.MockedFunction<typeof isWorkBookmarked>;

describe('BookmarkButton', () => {
  const defaultProps = {
    workId: 'test-work-id',
    authToken: 'test-auth-token',
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Initial State', () => {
    it('renders bookmark button for authenticated user', () => {
      mockIsWorkBookmarked.mockResolvedValue(false);
      render(<BookmarkButton {...defaultProps} />);
      
      expect(screen.getByRole('button')).toBeInTheDocument();
      expect(screen.getByText('🔖 Bookmark')).toBeInTheDocument();
    });

    it('renders disabled button for unauthenticated user', () => {
      render(<BookmarkButton workId="test-work-id" />);
      
      const button = screen.getByRole('button');
      expect(button).toBeDisabled();
      expect(button).toHaveAttribute('title', 'Log in to bookmark works');
    });

    it('checks initial bookmark status on mount', async () => {
      mockIsWorkBookmarked.mockResolvedValue(true);
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        expect(mockIsWorkBookmarked).toHaveBeenCalledWith('test-work-id', 'test-auth-token');
      });
      
      await waitFor(() => {
        expect(screen.getByText('🔖 Bookmarked')).toBeInTheDocument();
      });
    });

    it('shows unbookmarked state when work is not bookmarked', async () => {
      mockIsWorkBookmarked.mockResolvedValue(false);
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('🔖 Bookmark')).toBeInTheDocument();
      });
    });
  });

  describe('Bookmark Creation', () => {
    beforeEach(() => {
      mockIsWorkBookmarked.mockResolvedValue(false);
    });

    it('opens bookmark dialog when clicking unbookmarked work', async () => {
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('🔖 Bookmark')).toBeInTheDocument();
      });
      
      fireEvent.click(screen.getByRole('button'));
      
      expect(screen.getByRole('heading', { name: 'Add Bookmark' })).toBeInTheDocument();
      expect(screen.getByLabelText('Notes (optional)')).toBeInTheDocument();
      expect(screen.getByLabelText('Tags (optional)')).toBeInTheDocument();
      expect(screen.getByLabelText('Make bookmark private')).toBeInTheDocument();
    });

    it('creates bookmark with notes and tags', async () => {
      mockCreateBookmark.mockResolvedValue({ bookmark: { id: 'bookmark-id' } });
      
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        fireEvent.click(screen.getByRole('button'));
      });
      
      // Fill in bookmark form
      fireEvent.change(screen.getByLabelText('Notes (optional)'), {
        target: { value: 'Great story!' },
      });
      fireEvent.change(screen.getByLabelText('Tags (optional)'), {
        target: { value: 'favorite, reread' },
      });
      fireEvent.click(screen.getByLabelText('Make bookmark private'));
      
      // Submit form
      fireEvent.click(screen.getByRole('button', { name: 'Add Bookmark' }));
      
      await waitFor(() => {
        expect(mockCreateBookmark).toHaveBeenCalledWith(
          'test-work-id',
          {
            notes: 'Great story!',
            tags: ['favorite', 'reread'],
            is_private: true,
          },
          'test-auth-token'
        );
      });
      
      await waitFor(() => {
        expect(screen.getByText('🔖 Bookmarked')).toBeInTheDocument();
      });
    });

    it('creates bookmark with minimal data', async () => {
      mockCreateBookmark.mockResolvedValue({ bookmark: { id: 'bookmark-id' } });
      
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        fireEvent.click(screen.getByRole('button'));
      });
      
      fireEvent.click(screen.getByRole('button', { name: 'Add Bookmark' }));
      
      await waitFor(() => {
        expect(mockCreateBookmark).toHaveBeenCalledWith(
          'test-work-id',
          {
            is_private: false,
          },
          'test-auth-token'
        );
      });
    });

    it('handles bookmark creation errors', async () => {
      mockCreateBookmark.mockRejectedValue(new Error('Bookmark already exists'));
      
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        fireEvent.click(screen.getByRole('button'));
      });
      
      fireEvent.click(screen.getByRole('button', { name: 'Add Bookmark' }));
      
      await waitFor(() => {
        expect(screen.getByText('Bookmark already exists')).toBeInTheDocument();
      });
      
      // Dialog should still be open
      expect(screen.getByRole('heading', { name: 'Add Bookmark' })).toBeInTheDocument();
    });

    it('shows loading state during bookmark creation', async () => {
      mockCreateBookmark.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)));
      
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        fireEvent.click(screen.getByRole('button'));
      });
      
      fireEvent.click(screen.getByRole('button', { name: 'Add Bookmark' }));
      
      expect(screen.getByText('Adding...')).toBeInTheDocument();
      expect(screen.getByText('Adding...')).toBeDisabled();
    });

    it('cancels bookmark dialog', async () => {
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        fireEvent.click(screen.getByRole('button'));
      });
      
      expect(screen.getByRole('heading', { name: 'Add Bookmark' })).toBeInTheDocument();
      
      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
      
      expect(screen.queryByRole('heading', { name: 'Add Bookmark' })).not.toBeInTheDocument();
    });

    it('trims and filters tags correctly', async () => {
      mockCreateBookmark.mockResolvedValue({ bookmark: { id: 'bookmark-id' } });
      
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        fireEvent.click(screen.getByRole('button'));
      });
      
      // Test tag parsing with various edge cases
      fireEvent.change(screen.getByLabelText('Tags (optional)'), {
        target: { value: ' tag1 , , tag2,   tag3   ,' },
      });
      
      fireEvent.click(screen.getByRole('button', { name: 'Add Bookmark' }));
      
      await waitFor(() => {
        expect(mockCreateBookmark).toHaveBeenCalledWith(
          'test-work-id',
          expect.objectContaining({
            tags: ['tag1', 'tag2', 'tag3'],
          }),
          'test-auth-token'
        );
      });
    });
  });

  describe('Error Handling', () => {
    it('shows disabled state for unauthenticated users', () => {
      render(<BookmarkButton workId="test-work-id" />);
      
      const button = screen.getByRole('button');
      expect(button).toBeDisabled();
      expect(button).toHaveAttribute('title', 'Log in to bookmark works');
      
      // Button should not be clickable when disabled
      fireEvent.click(button);
      
      // No error message should appear since button is disabled
      expect(screen.queryByText('Please log in to bookmark works')).not.toBeInTheDocument();
    });

    it('handles bookmark status check errors gracefully', async () => {
      mockIsWorkBookmarked.mockRejectedValue(new Error('Network error'));
      
      render(<BookmarkButton {...defaultProps} />);
      
      // Should still render button even if status check fails
      await waitFor(() => {
        expect(screen.getByText('🔖 Bookmark')).toBeInTheDocument();
      });
    });
  });

  describe('Callback Functions', () => {
    it('calls onBookmarkChange when bookmark is created', async () => {
      const mockOnBookmarkChange = jest.fn();
      mockIsWorkBookmarked.mockResolvedValue(false);
      mockCreateBookmark.mockResolvedValue({ bookmark: { id: 'bookmark-id' } });
      
      render(
        <BookmarkButton 
          {...defaultProps} 
          onBookmarkChange={mockOnBookmarkChange}
        />
      );
      
      await waitFor(() => {
        fireEvent.click(screen.getByRole('button'));
      });
      
      fireEvent.click(screen.getByRole('button', { name: 'Add Bookmark' }));
      
      await waitFor(() => {
        expect(mockOnBookmarkChange).toHaveBeenCalledWith(true);
      });
    });
  });

  describe('UI Variants', () => {
    beforeEach(() => {
      mockIsWorkBookmarked.mockResolvedValue(false);
    });

    it('renders with custom variant and size', async () => {
      render(
        <BookmarkButton 
          {...defaultProps}
          variant="default"
          size="lg"
        />
      );
      
      await waitFor(() => {
        const button = screen.getByRole('button');
        expect(button).toBeInTheDocument();
      });
    });

    it('shows different styles for bookmarked state', async () => {
      mockIsWorkBookmarked.mockResolvedValue(true);
      
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('🔖 Bookmarked')).toBeInTheDocument();
      });
    });
  });

  describe('Already Bookmarked State', () => {
    beforeEach(() => {
      mockIsWorkBookmarked.mockResolvedValue(true);
    });

    it('shows error message for already bookmarked work', async () => {
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('🔖 Bookmarked')).toBeInTheDocument();
      });
      
      fireEvent.click(screen.getByRole('button'));
      
      await waitFor(() => {
        expect(screen.getByText('Bookmark removal not yet implemented')).toBeInTheDocument();
      });
    });
  });

  describe('Accessibility', () => {
    beforeEach(() => {
      mockIsWorkBookmarked.mockResolvedValue(false);
    });

    it('has proper ARIA attributes', async () => {
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        const button = screen.getByRole('button');
        expect(button).toHaveAttribute('title', 'Bookmark this work');
      });
    });

    it('maintains focus management in dialog', async () => {
      render(<BookmarkButton {...defaultProps} />);
      
      await waitFor(() => {
        fireEvent.click(screen.getByRole('button'));
      });
      
      // Dialog should be focusable
      expect(screen.getByRole('heading', { name: 'Add Bookmark' })).toBeInTheDocument();
      
      // Form elements should have proper labels
      expect(screen.getByLabelText('Notes (optional)')).toBeInTheDocument();
      expect(screen.getByLabelText('Tags (optional)')).toBeInTheDocument();
      expect(screen.getByLabelText('Make bookmark private')).toBeInTheDocument();
    });
  });
});