import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import SearchResults from '../SearchResults';

const mockWork = {
  id: '1',
  title: 'Test Work Title',
  author: 'Test Author',
  summary: 'This is a test work summary that is long enough to test the expand/collapse functionality and needs to be over 200 characters to trigger the show more button so here is some extra text to make it longer and longer and even more text.',
  word_count: 15000,
  chapter_count: 5,
  max_chapters: 10,
  rating: 'Teen And Up Audiences',
  status: 'in_progress',
  language: 'English',
  published_date: '2023-01-01T00:00:00Z',
  updated_date: '2023-02-01T00:00:00Z',
  relationships: [
    { name: 'Harry Potter/Draco Malfoy', category: 'M/M' },
  ],
  characters: [
    { name: 'Harry Potter', category: 'Main Character' },
    { name: 'Draco Malfoy', category: 'Main Character' },
  ],
  freeform_tags: [
    { name: 'Enemies to Lovers', category: 'Relationship Dynamic' },
    { name: 'Hogwarts Era', category: 'Setting' },
  ],
  fandoms: [
    { name: 'Harry Potter - J. K. Rowling', category: 'Literature' },
  ],
  kudos_count: 150,
  bookmark_count: 45,
  hit_count: 2300,
  comment_count: 23,
  tag_quality_score: 0.85,
  missing_tag_suggestions: ['Slow Burn', 'Hurt/Comfort'],
};

const mockRecommendations = [
  {
    type: 'missing_character',
    title: 'Missing Character Tags',
    description: 'Consider adding these character tags',
    suggestions: ['Hermione Granger', 'Ron Weasley'],
    confidence_score: 0.85,
    category: 'character',
  },
];

const defaultProps = {
  results: [mockWork],
  recommendations: mockRecommendations,
  loading: false,
  totalCount: 1,
  currentPage: 1,
  pageSize: 20,
};

describe('SearchResults - Accessibility Implementation', () => {
  describe('Accessibility Structure', () => {
    it('renders with proper semantic structure and landmarks', () => {
      render(<SearchResults {...defaultProps} />);

      // Check for main region
      const resultsRegion = screen.getByRole('region', { name: /search results/i });
      expect(resultsRegion).toHaveAttribute('aria-labelledby');

      // Check for results heading
      const resultsHeading = screen.getByRole('heading', { name: /search results/i });
      expect(resultsHeading).toBeInTheDocument();
      expect(resultsHeading).toHaveAttribute('tabIndex', '-1');

      // Check for status region
      const statusRegion = screen.getByRole('status');
      expect(statusRegion).toHaveAttribute('aria-live', 'polite');

      // Check for live region
      const liveRegion = screen.getAllByRole('status', { hidden: true })[0];
      expect(liveRegion).toHaveAttribute('aria-live', 'polite');
      expect(liveRegion).toHaveAttribute('aria-atomic', 'true');
    });

    it('structures each work as an article with proper labeling', () => {
      render(<SearchResults {...defaultProps} />);

      const workArticle = screen.getByRole('article');
      expect(workArticle).toHaveAttribute('aria-labelledby');
      expect(workArticle).toHaveAttribute('aria-describedby');

      // Check work title button (which is semantically an h4 with button role)
      const workTitle = screen.getByRole('button', { name: /test work title/i });
      expect(workTitle).toBeInTheDocument();
      expect(workTitle).toHaveAttribute('tabIndex', '0');
    });

    it('provides comprehensive metadata with proper labeling', () => {
      render(<SearchResults {...defaultProps} />);

      // Check rating with visual and screen reader info
      expect(screen.getByText('Teen And Up Audiences')).toBeInTheDocument();
      expect(screen.getByText('Content rating: Teen And Up Audiences', { selector: '.sr-only' })).toBeInTheDocument();

      // Check dates with proper time elements
      const publishedDate = screen.getByText('January 1, 2023');
      expect(publishedDate.closest('time')).toHaveAttribute('datetime', '2023-01-01');

      // Check statistics with screen reader labels
      expect(screen.getByText('150')).toBeInTheDocument();
      expect(screen.getByText('kudos', { selector: '.sr-only' })).toBeInTheDocument();
    });
  });

  describe('Interactive Elements', () => {
    it('makes work titles keyboard accessible', async () => {
      const mockOnWorkClick = jest.fn();
      const user = userEvent.setup();
      
      render(<SearchResults {...defaultProps} onWorkClick={mockOnWorkClick} />);

      const workTitle = screen.getByRole('button', { name: /test work title/i });
      
      // Test click
      await user.click(workTitle);
      expect(mockOnWorkClick).toHaveBeenCalledWith('1', 'Test Work Title');

      // Test Enter key
      mockOnWorkClick.mockClear();
      await user.keyboard('{Enter}');
      expect(mockOnWorkClick).toHaveBeenCalledWith('1', 'Test Work Title');

      // Test Space key
      mockOnWorkClick.mockClear();
      await user.keyboard(' ');
      expect(mockOnWorkClick).toHaveBeenCalledWith('1', 'Test Work Title');
    });

    it('provides accessible summary expansion', async () => {
      const user = userEvent.setup();
      const longSummaryWork = {
        ...mockWork,
        summary: 'This is a very long summary that should trigger the show more/less functionality and needs to be over 200 characters to trigger the show more button so here is some extra text to make it longer and longer and even more text.',
      };
      
      render(<SearchResults {...defaultProps} results={[longSummaryWork]} />);

      // Debug: check what buttons are available
      screen.debug();
      
      const expandButton = screen.getByRole('button', { name: /show more/i });
      expect(expandButton).toHaveAttribute('aria-expanded', 'false');
      expect(expandButton).toHaveAttribute('aria-controls');

      await user.click(expandButton);

      const collapseButton = screen.getByRole('button', { name: /show less/i });
      expect(collapseButton).toHaveAttribute('aria-expanded', 'true');

      // Check live region announcement
      const liveRegion = screen.getAllByRole('status', { hidden: true })[0];
      await waitFor(() => {
        expect(liveRegion).toHaveTextContent(/summary expanded/i);
      });
    });
    });
  });

  describe('Smart Recommendations', () => {
    it('renders recommendations with proper accessibility attributes', () => {
      render(<SearchResults {...defaultProps} />);

      const recommendationsSection = screen.getByRole('region', { name: /smart suggestions/i });
      expect(recommendationsSection).toBeInTheDocument();

      const toggleButton = screen.getByRole('button', { name: 'Show' });
      expect(toggleButton).toHaveAttribute('aria-expanded', 'false');
      expect(toggleButton).toHaveAttribute('aria-controls');
    });

    it('manages focus and announcements when expanding recommendations', async () => {
      const user = userEvent.setup();
      render(<SearchResults {...defaultProps} />);

      const toggleButton = screen.getByRole('button', { name: 'Show' });
      await user.click(toggleButton);

      expect(toggleButton).toHaveAttribute('aria-expanded', 'true');
      expect(screen.getByRole('button', { name: /hide/i })).toBeInTheDocument();

      // Check announcement
      await waitFor(() => {
        const liveRegion = screen.getByText(/suggestions expanded/i);
        expect(liveRegion).toBeInTheDocument();
      });
    });
  });

  describe('Loading States', () => {
    it('provides accessible loading feedback', () => {
      render(<SearchResults {...defaultProps} loading={true} />);

      const loadingStatus = screen.getByRole('status');
      expect(loadingStatus).toHaveAttribute('aria-live', 'polite');
      expect(loadingStatus).toHaveAttribute('aria-label', 'Loading search results');
      expect(screen.getByText(/searching the archive/i)).toBeInTheDocument();
    });

    it('announces when results finish loading', async () => {
      const { rerender } = render(<SearchResults {...defaultProps} loading={true} />);
      
      await act(async () => {
        rerender(<SearchResults {...defaultProps} loading={false} />);
      });

      const liveRegion = screen.getByText(/found 1 works.*results loaded/i);
      expect(liveRegion).toBeInTheDocument();
    });
  });

  describe('Error States', () => {
    it('displays errors with proper alert role', () => {
      const error = 'Search failed due to network error';
      render(<SearchResults {...defaultProps} error={error} />);

      const errorAlert = screen.getByRole('alert');
      expect(errorAlert).toBeInTheDocument();
      expect(errorAlert).toHaveTextContent('Search Error');
      expect(errorAlert).toHaveTextContent(error);
    });

    it('provides helpful error recovery information', () => {
      render(<SearchResults {...defaultProps} error="Network error" />);

      expect(screen.getByText(/please try adjusting your search criteria/i)).toBeInTheDocument();
    });
  });

  describe('Empty States', () => {
    it('provides helpful empty state messaging', () => {
      render(<SearchResults {...defaultProps} results={[]} />);

      expect(screen.getByRole('heading', { name: /no works found/i })).toBeInTheDocument();
      expect(screen.getByText(/try adjusting your search criteria/i)).toBeInTheDocument();
    });
  });

  describe('Tag Organization', () => {
    it('organizes tags with proper grouping and labeling', () => {
      render(<SearchResults {...defaultProps} />);

      // Check fandom tags
      expect(screen.getByText('Fandoms:')).toBeInTheDocument();
      expect(screen.getByText('Harry Potter - J. K. Rowling')).toBeInTheDocument();

      // Check relationship tags
      expect(screen.getByText('Relationships:')).toBeInTheDocument();
      expect(screen.getByText('Harry Potter/Draco Malfoy')).toBeInTheDocument();

      // Check character tags
      expect(screen.getByText('Characters:')).toBeInTheDocument();
      expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      expect(screen.getByText('Draco Malfoy')).toBeInTheDocument();

      // Check additional tags
      expect(screen.getByText('Additional Tags:')).toBeInTheDocument();
      expect(screen.getByText('Enemies to Lovers')).toBeInTheDocument();
      expect(screen.getByText('Hogwarts Era')).toBeInTheDocument();
    });

    it('displays missing tag suggestions accessibly', () => {
      render(<SearchResults {...defaultProps} />);

      expect(screen.getByRole('heading', { name: /suggested additional tags/i })).toBeInTheDocument();
      expect(screen.getByText('Slow Burn')).toBeInTheDocument();
      expect(screen.getByText('Hurt/Comfort')).toBeInTheDocument();
    });
  });

  describe('Statistics Display', () => {
    it('formats statistics with proper accessibility labels', () => {
      render(<SearchResults {...defaultProps} />);

      // All statistics should have screen reader labels
      expect(screen.getByText('150')).toBeInTheDocument();
      expect(screen.getByText('kudos', { selector: '.sr-only' })).toBeInTheDocument();

      expect(screen.getByText('45')).toBeInTheDocument();
      expect(screen.getByText('bookmarks', { selector: '.sr-only' })).toBeInTheDocument();

      expect(screen.getByText('23')).toBeInTheDocument();
      expect(screen.getByText('comments', { selector: '.sr-only' })).toBeInTheDocument();

      expect(screen.getByText('2,300')).toBeInTheDocument();
      expect(screen.getByText('hits', { selector: '.sr-only' })).toBeInTheDocument();
    });

    it('displays tag quality score', () => {
      render(<SearchResults {...defaultProps} />);

      expect(screen.getByText('Tag Quality: 85%')).toBeInTheDocument();
    });
  });

  describe('Date Formatting', () => {
    it('formats dates with proper time elements and ISO attributes', () => {
      render(<SearchResults {...defaultProps} />);

      const publishedTime = screen.getByText('January 1, 2023').closest('time');
      expect(publishedTime).toHaveAttribute('datetime', '2023-01-01');
      expect(publishedTime).toHaveAttribute('title', 'Published January 1, 2023');

      const updatedTime = screen.getByText('February 1, 2023').closest('time');
      expect(updatedTime).toHaveAttribute('datetime', '2023-02-01');
      expect(updatedTime).toHaveAttribute('title', 'Updated February 1, 2023');
    });
  });

  describe('Pagination Info', () => {
    it('displays result count information accessibly', () => {
      render(
        <SearchResults 
          {...defaultProps} 
          totalCount={100} 
          currentPage={2} 
          pageSize={20} 
        />
      );

      const statusRegion = screen.getByRole('status');
      expect(statusRegion).toHaveTextContent('Showing 21-40 of 100 works');
    });
  });

  describe('Screen Reader Announcements', () => {
    it('announces result loading completion', () => {
      render(<SearchResults {...defaultProps} />);

      const liveRegion = screen.getByText(/found 1 works.*results loaded/i);
      expect(liveRegion).toBeInTheDocument();
    });

    it('announces interactive state changes', async () => {
      const user = userEvent.setup();
      const longSummaryWork = {
        ...mockWork,
        summary: 'This is a very long summary that should trigger the show more/less functionality and needs to be over 200 characters to trigger the show more button so here is some extra text to make it longer and longer and even more text.',
      };
      
      render(<SearchResults {...defaultProps} results={[longSummaryWork]} />);

      const expandButton = screen.getByRole('button', { name: /show more/i });
      await user.click(expandButton);

      await waitFor(() => {
        const liveRegion = screen.getByText(/summary expanded/i);
        expect(liveRegion).toBeInTheDocument();
      });
    });
  });

  describe('Multiple Works', () => {
    it('handles multiple works with proper list structure', async () => {
      const multipleWorks = [
        mockWork,
        { ...mockWork, id: '2', title: 'Second Work' },
        { ...mockWork, id: '3', title: 'Third Work' },
      ];

      await act(async () => {
        render(<SearchResults {...defaultProps} results={multipleWorks} />);
      });

      const worksList = screen.getByRole('list');
      expect(worksList).toBeInTheDocument();

      const workItems = screen.getAllByRole('listitem');
      expect(workItems).toHaveLength(3);

      const articles = screen.getAllByRole('article');
      expect(articles).toHaveLength(3);
    });
  });

  describe('Focus Management', () => {
    it('focuses results heading when results load', async () => {
      const { rerender } = render(<SearchResults {...defaultProps} loading={true} />);
      
      await act(async () => {
        rerender(<SearchResults {...defaultProps} loading={false} />);
      });

      await waitFor(() => {
        const resultsHeading = screen.getByRole('heading', { name: /search results/i });
        expect(resultsHeading).toHaveFocus();
      });
    });
  });
});
