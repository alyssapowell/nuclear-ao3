import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { MockedProvider } from '@apollo/client/testing';
import SearchForm from '../SearchForm';
import { ENHANCED_SEARCH_WORKS } from '@/lib/graphql';

// Mock debounced search
jest.mock('lodash.debounce', () => jest.fn((fn) => fn));

const mockOnResults = jest.fn();
const mockOnRecommendations = jest.fn();

const mockSearchWorksMocks = [
  {
    request: {
      query: ENHANCED_SEARCH_WORKS,
      variables: {
        query: 'Harry Potter',
        filters: {
          title: 'Harry Potter',
          author: undefined,
          relationships: [],
          characters: [],
          freeform_tags: [],
          fandoms: [],
          rating: undefined,
          word_count_min: undefined,
          word_count_max: undefined,
          language: undefined,
          status: undefined,
        },
        options: {
          exclude_poorly_tagged: false,
          enable_smart_suggestions: true,
          limit: 20,
          offset: 0,
        },
      },
    },
    result: {
      data: {
        enhancedSearchWorks: [
          {
            id: '1',
            title: 'Harry Potter and the Test Story',
            author: 'Test Author',
            summary: 'A test story',
            word_count: 1000,
            chapter_count: 1,
            max_chapters: 1,
            rating: 'Teen And Up Audiences',
            status: 'complete',
            language: 'English',
            published_date: '2023-01-01',
            updated_date: '2023-01-01',
            relationships: [],
            characters: [],
            freeform_tags: [],
            fandoms: [],
            kudos_count: 10,
            bookmark_count: 5,
            hit_count: 100,
            comment_count: 3,
          },
        ],
        smart_recommendations: [
          {
            type: 'missing_character',
            title: 'Missing Character Tags',
            description: 'Consider adding character tags',
            suggestions: ['Harry Potter', 'Hermione Granger'],
            confidence_score: 0.85,
            category: 'character',
          },
        ],
      },
    },
  },
];

const defaultProps = {
  onResults: mockOnResults,
  onRecommendations: mockOnRecommendations,
};

const renderSearchForm = (props = {}, mocks = []) => {
  return render(
    <MockedProvider mocks={mocks} addTypename={false}>
      <SearchForm {...defaultProps} {...props} />
    </MockedProvider>
  );
};

describe('SearchForm - Accessibility-First Implementation', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Accessibility Structure', () => {
    it('renders with proper semantic structure and ARIA attributes', () => {
      renderSearchForm();

      // Check for search landmark
      const searchForm = screen.getByRole('search');
      expect(searchForm).toBeInTheDocument();
      expect(searchForm).toHaveAttribute('aria-labelledby');
      expect(searchForm).toHaveAttribute('aria-describedby');

      // Check for form heading
      expect(screen.getByRole('heading', { name: /enhanced search/i })).toBeInTheDocument();

      // Check for live region
      const liveRegion = screen.getByRole('status', { hidden: true });
      expect(liveRegion).toHaveAttribute('aria-live', 'polite');
      expect(liveRegion).toHaveAttribute('aria-atomic', 'true');
    });

    it('has proper labeling for all form controls', () => {
      renderSearchForm();

      // Main search input
      const titleInput = screen.getByLabelText(/title/i);
      expect(titleInput).toBeInTheDocument();
      expect(titleInput).toHaveAttribute('aria-describedby');

      const authorInput = screen.getByLabelText(/author/i);
      expect(authorInput).toBeInTheDocument();

      // Rating and status selects
      const ratingSelect = screen.getByLabelText(/rating/i);
      expect(ratingSelect).toBeInTheDocument();

      const statusSelect = screen.getByLabelText(/status/i);
      expect(statusSelect).toBeInTheDocument();

      // Checkboxes
      const excludePoorlyTaggedCheckbox = screen.getByLabelText(/exclude poorly tagged works/i);
      expect(excludePoorlyTaggedCheckbox).toBeInTheDocument();

      const smartSuggestionsCheckbox = screen.getByLabelText(/smart suggestions/i);
      expect(smartSuggestionsCheckbox).toBeInTheDocument();
    });

    it('provides proper help text for form fields', () => {
      renderSearchForm();

      // Check for help text
      expect(screen.getByText(/enter keywords from the work title/i)).toBeInTheDocument();
      expect(screen.getByText(/enter author username or display name/i)).toBeInTheDocument();
    });
  });

  describe('Keyboard Navigation', () => {
    it('maintains logical tab order', async () => {
      const user = userEvent.setup();
      renderSearchForm();

      const titleInput = screen.getByLabelText(/title/i);
      const authorInput = screen.getByLabelText(/author/i);
      const submitButton = screen.getByRole('button', { name: /search works/i });

      await user.tab();
      expect(titleInput).toHaveFocus();

      await user.tab();
      expect(authorInput).toHaveFocus();

      // Tab through all fields to submit button
      await user.tab(); // relationships
      await user.tab(); // characters  
      await user.tab(); // additional tags
      await user.tab(); // fandoms
      await user.tab(); // rating
      await user.tab(); // status
      await user.tab(); // exclude poorly tagged
      await user.tab(); // smart suggestions
      await user.tab(); // advanced search toggle
      await user.tab(); // clear all
      await user.tab(); // submit button
      expect(submitButton).toHaveFocus();
    });

    it('supports Enter key submission', async () => {
      const user = userEvent.setup();
      renderSearchForm({}, mockSearchWorksMocks);

      const titleInput = screen.getByLabelText(/title/i);
      await user.type(titleInput, 'Harry Potter');
      await user.keyboard('{Enter}');

      await waitFor(() => {
        expect(mockOnResults).toHaveBeenCalled();
      });
    });
  });

  describe('Advanced Search Toggle', () => {
    it('has proper ARIA attributes for collapsible section', () => {
      renderSearchForm();

      const advancedToggle = screen.getByRole('button', { name: /show advanced search/i });
      expect(advancedToggle).toHaveAttribute('aria-expanded', 'false');
      expect(advancedToggle).toHaveAttribute('aria-controls');
    });

    it('updates ARIA attributes when expanded', async () => {
      const user = userEvent.setup();
      renderSearchForm();

      const advancedToggle = screen.getByRole('button', { name: /show advanced search/i });
      await user.click(advancedToggle);

      expect(advancedToggle).toHaveAttribute('aria-expanded', 'true');
      expect(screen.getByRole('button', { name: /hide advanced search/i })).toBeInTheDocument();

      // Check advanced section is labeled
      const advancedSection = screen.getByRole('region');
      expect(advancedSection).toHaveAttribute('aria-labelledby');
    });

    it('manages focus when expanding advanced options', async () => {
      const user = userEvent.setup();
      renderSearchForm();

      const advancedToggle = screen.getByRole('button', { name: /show advanced search/i });
      await user.click(advancedToggle);

      // Should focus first advanced field
      await waitFor(() => {
        const wordCountMinInput = screen.getByLabelText(/minimum word count/i);
        expect(wordCountMinInput).toHaveFocus();
      });
    });
  });

  describe('Tag Management', () => {
    it('provides accessible tag addition and removal', async () => {
      const user = userEvent.setup();
      renderSearchForm();

      const relationshipsInput = screen.getByLabelText(/relationships/i);
      await user.type(relationshipsInput, 'Harry Potter/Draco Malfoy');
      await user.keyboard('{Enter}');

      // Check tag is added with proper accessibility
      const tagGroup = screen.getByRole('group', { name: /selected relationships/i });
      expect(tagGroup).toBeInTheDocument();

      const removeButton = screen.getByRole('button', { name: /remove harry potter\/draco malfoy from relationships/i });
      expect(removeButton).toBeInTheDocument();
      expect(removeButton).toHaveAttribute('title');

      // Remove tag
      await user.click(removeButton);
      expect(tagGroup).not.toBeInTheDocument();
    });

    it('announces tag changes to screen readers', async () => {
      const user = userEvent.setup();
      renderSearchForm();

      const relationshipsInput = screen.getByLabelText(/relationships/i);
      await user.type(relationshipsInput, 'Test Tag');
      await user.keyboard('{Enter}');

      // Live region should announce the addition
      const liveRegion = screen.getByRole('status', { hidden: true });
      await waitFor(() => {
        expect(liveRegion).toHaveTextContent(/added test tag to relationships/i);
      });
    });
  });

  describe('Error Handling', () => {
    it('displays validation errors with proper ARIA attributes', async () => {
      const user = userEvent.setup();
      renderSearchForm();

      // Try to submit without any search criteria
      const submitButton = screen.getByRole('button', { name: /search works/i });
      await user.click(submitButton);

      // Check for error alert
      const errorAlert = screen.getByRole('alert');
      expect(errorAlert).toBeInTheDocument();
      expect(errorAlert).toHaveTextContent(/please enter at least one search criterion/i);

      // Should focus the title input
      const titleInput = screen.getByLabelText(/title/i);
      expect(titleInput).toHaveFocus();
    });

    it('announces search errors to screen readers', async () => {
      const user = userEvent.setup();
      const errorMocks = [
        {
          request: {
            query: ENHANCED_SEARCH_WORKS,
            variables: expect.any(Object),
          },
          error: new Error('Search failed'),
        },
      ];

      renderSearchForm({}, errorMocks);

      const titleInput = screen.getByLabelText(/title/i);
      await user.type(titleInput, 'test');

      const submitButton = screen.getByRole('button', { name: /search works/i });
      await user.click(submitButton);

      await waitFor(() => {
        const errorAlert = screen.getByRole('alert');
        expect(errorAlert).toBeInTheDocument();
        expect(errorAlert).toHaveTextContent(/search error/i);
      });
    });
  });

  describe('Loading States', () => {
    it('provides accessible loading feedback', async () => {
      const user = userEvent.setup();
      const delayedMocks = [
        {
          request: {
            query: ENHANCED_SEARCH_WORKS,
            variables: expect.any(Object),
          },
          delay: 100,
          result: {
            data: {
              enhancedSearchWorks: [],
            },
          },
        },
      ];

      renderSearchForm({}, delayedMocks);

      const titleInput = screen.getByLabelText(/title/i);
      await user.type(titleInput, 'test');

      const submitButton = screen.getByRole('button', { name: /search works/i });
      await user.click(submitButton);

      // Check loading state
      expect(screen.getByText(/searching\.\.\./i)).toBeInTheDocument();
      expect(submitButton).toBeDisabled();

      const loadingStatus = screen.getByRole('status');
      expect(loadingStatus).toHaveAttribute('aria-live', 'polite');
      expect(loadingStatus).toHaveTextContent(/searching the archive/i);

      await waitFor(() => {
        expect(screen.queryByText(/searching\.\.\./i)).not.toBeInTheDocument();
        expect(submitButton).not.toBeDisabled();
      });
    });
  });

  describe('Search Functionality', () => {
    it('performs search with proper result handling', async () => {
      const user = userEvent.setup();
      renderSearchForm({}, mockSearchWorksMocks);

      const titleInput = screen.getByLabelText(/title/i);
      await user.type(titleInput, 'Harry Potter');

      const submitButton = screen.getByRole('button', { name: /search works/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockOnResults).toHaveBeenCalledWith([
          expect.objectContaining({
            id: '1',
            title: 'Harry Potter and the Test Story',
            author: 'Test Author',
          }),
        ]);
      });

      await waitFor(() => {
        expect(mockOnRecommendations).toHaveBeenCalledWith([
          expect.objectContaining({
            type: 'missing_character',
            title: 'Missing Character Tags',
          }),
        ]);
      });
    });

    it('applies advanced filters correctly', async () => {
      const user = userEvent.setup();
      renderSearchForm({}, mockSearchWorksMocks);

      // Expand advanced search
      const advancedToggle = screen.getByRole('button', { name: /show advanced search/i });
      await user.click(advancedToggle);

      // Set word count filter
      const wordCountMin = screen.getByLabelText(/minimum word count/i);
      await user.type(wordCountMin, '1000');

      // Set language filter
      const languageSelect = screen.getByLabelText(/language/i);
      await user.selectOptions(languageSelect, 'en');

      const titleInput = screen.getByLabelText(/title/i);
      await user.type(titleInput, 'test');

      const submitButton = screen.getByRole('button', { name: /search works/i });
      await user.click(submitButton);

      // Should call with advanced filters
      await waitFor(() => {
        expect(mockOnResults).toHaveBeenCalled();
      });
    });
  });

  describe('Clear Functionality', () => {
    it('clears all filters and announces the action', async () => {
      const user = userEvent.setup();
      renderSearchForm();

      // Add some data
      const titleInput = screen.getByLabelText(/title/i);
      await user.type(titleInput, 'test');

      const ratingSelect = screen.getByLabelText(/rating/i);
      await user.selectOptions(ratingSelect, 'Teen And Up Audiences');

      // Clear all
      const clearButton = screen.getByRole('button', { name: /clear all/i });
      await user.click(clearButton);

      // Check fields are cleared
      expect(titleInput).toHaveValue('');
      expect(ratingSelect).toHaveValue('');

      // Check announcement
      const liveRegion = screen.getByRole('status', { hidden: true });
      await waitFor(() => {
        expect(liveRegion).toHaveTextContent(/all search filters cleared/i);
      });

      // Should focus title input
      expect(titleInput).toHaveFocus();
    });
  });

  describe('Screen Reader Announcements', () => {
    it('announces search results count', async () => {
      const user = userEvent.setup();
      renderSearchForm({}, mockSearchWorksMocks);

      const titleInput = screen.getByLabelText(/title/i);
      await user.type(titleInput, 'Harry Potter');

      const submitButton = screen.getByRole('button', { name: /search works/i });
      await user.click(submitButton);

      const liveRegion = screen.getByRole('status', { hidden: true });
      await waitFor(() => {
        expect(liveRegion).toHaveTextContent(/search completed/i);
      });
    });

    it('announces advanced search state changes', async () => {
      const user = userEvent.setup();
      renderSearchForm();

      const advancedToggle = screen.getByRole('button', { name: /show advanced search/i });
      await user.click(advancedToggle);

      const liveRegion = screen.getByRole('status', { hidden: true });
      await waitFor(() => {
        expect(liveRegion).toHaveTextContent(/advanced search options expanded/i);
      });

      await user.click(screen.getByRole('button', { name: /hide advanced search/i }));
      await waitFor(() => {
        expect(liveRegion).toHaveTextContent(/advanced search options collapsed/i);
      });
    });
  });

  describe('Initial Props', () => {
    it('accepts initial filters and applies them', () => {
      const initialFilters = {
        title: 'Initial Title',
        rating: 'General Audiences',
        excludePoorlyTagged: true,
      };

      renderSearchForm({ initialFilters });

      const titleInput = screen.getByLabelText(/title/i);
      expect(titleInput).toHaveValue('Initial Title');

      const ratingSelect = screen.getByLabelText(/rating/i);
      expect(ratingSelect).toHaveValue('General Audiences');

      const excludeCheckbox = screen.getByLabelText(/exclude poorly tagged works/i);
      expect(excludeCheckbox).toBeChecked();
    });
  });
});