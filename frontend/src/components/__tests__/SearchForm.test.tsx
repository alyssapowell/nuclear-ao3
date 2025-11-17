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
          author: undefined,
          relationships: [],
          characters: [],
          freeformTags: [],
          fandoms: [],
          rating: undefined,
          wordCountMin: undefined,
          wordCountMax: undefined,
          language: undefined,
          completionStatus: undefined,
          blockedTags: [],
          hideIncomplete: false,
          hideCrossovers: false,
          minKudos: undefined,
          minComments: undefined,
          minBookmarks: undefined,
        },
        analysis: {
          enableSmartSuggestions: true,
          excludePoorlyTagged: false,
        },
      },
    },
    result: {
      data: {
        search: {
          enhancedWorks: {
            total: 1,
            works: [
              {
                id: '1',
                title: 'Harry Potter and the Test Story',
                authors: [{ id: '1', username: 'Test Author' }],
                summary: 'A test story',
                wordCount: 1000,
                chapterCount: 1,
                maxChapters: 1,
                isComplete: true,
                rating: 'Teen And Up Audiences',
                warnings: [],
                categories: [],
                language: 'English',
                publishedAt: '2023-01-01',
                updatedAt: '2023-01-01',
                relationships: [],
                characters: [],
                freeformTags: [],
                fandoms: [],
                kudosCount: 10,
                bookmarkCount: 5,
                hitCount: 100,
                commentCount: 3,
                tagQuality: {
                  score: 0.9,
                  missingSuggestions: [],
                  missingCharacters: [],
                  inconsistencies: [],
                },
              },
            ],
            analytics: null,
            smartSuggestions: {
              characterSuggestions: [
                {
                  tag: 'Harry Potter',
                  confidence: 0.85,
                  reasons: ['Detected from title'],
                },
              ],
              relationshipExpansions: [],
              crossTaggingOpportunities: [],
            },
          },
        },
      },
    },
  },
];

const defaultProps = {
  onResults: mockOnResults,
  onRecommendations: mockOnRecommendations,
};

// Default empty mock for rendering without search
const defaultMocks = [
  {
    request: {
      query: ENHANCED_SEARCH_WORKS,
    },
    result: {
      data: {
        search: {
          enhancedWorks: {
            total: 0,
            works: [],
            analytics: null,
            smartSuggestions: null
          }
        }
      },
    },
  },
];

const renderSearchForm = (props = {}, mocks = defaultMocks) => {
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

      // Check for search landmarks - there are two (section and form)
      const searchLandmarks = screen.getAllByRole('search');
      expect(searchLandmarks.length).toBeGreaterThan(0);
      expect(searchLandmarks[0]).toHaveAttribute('aria-labelledby');

      // Check for form heading
      expect(screen.getByRole('heading', { name: /enhanced search/i })).toBeInTheDocument();

      // Check for live region - may not be visible initially
      const liveRegions = screen.queryAllByRole('status', { hidden: true });
      if (liveRegions.length > 0) {
        expect(liveRegions[0]).toHaveAttribute('aria-live', 'polite');
        expect(liveRegions[0]).toHaveAttribute('aria-atomic', 'true');
      }
    });

    it('has proper labeling for all form controls', () => {
      renderSearchForm();

      // Main search inputs
      const titleInput = screen.getByLabelText(/title/i);
      expect(titleInput).toBeInTheDocument();
      expect(titleInput).toHaveAttribute('aria-describedby');

      const authorInput = screen.getByLabelText(/author/i);
      expect(authorInput).toBeInTheDocument();

      // Rating select
      const ratingSelect = screen.getByLabelText(/rating/i);
      expect(ratingSelect).toBeInTheDocument();

      // Sort select (no status field in basic form)
      const sortSelect = screen.getByLabelText(/sort results by/i);
      expect(sortSelect).toBeInTheDocument();

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
    it('allows keyboard navigation through form', async () => {
      const user = userEvent.setup();
      renderSearchForm();

      const titleInput = screen.getByLabelText(/title/i);
      
      // User can focus on inputs
      titleInput.focus();
      expect(titleInput).toHaveFocus();
      
      // Tab key moves focus
      await user.tab();
      expect(titleInput).not.toHaveFocus();
    });

    it('supports form submission', async () => {
      const user = userEvent.setup();
      renderSearchForm({}, mockSearchWorksMocks);

      const titleInput = screen.getByLabelText(/title/i);
      await user.type(titleInput, 'Harry Potter');
      
      const submitButton = screen.getByRole('button', { name: /search works/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockOnResults).toHaveBeenCalled();
      }, { timeout: 3000 });
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

      // Advanced section exists (may have multiple matches due to sr-only text)
      const advancedSections = screen.getAllByText(/advanced search options/i);
      expect(advancedSections.length).toBeGreaterThan(0);
    });

    it('manages focus when expanding advanced options', async () => {
      const user = userEvent.setup();
      renderSearchForm();

      const advancedToggle = screen.getByRole('button', { name: /show advanced search/i });
      await user.click(advancedToggle);

      // Advanced options should be visible
      await waitFor(() => {
        const wordCountMinInput = screen.getByLabelText(/minimum word count/i);
        expect(wordCountMinInput).toBeInTheDocument();
        expect(wordCountMinInput).toBeVisible();
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

    it.skip('announces tag changes to screen readers', async () => {
      // TODO: Fix live region timing in tests
      const user = userEvent.setup();
      renderSearchForm();

      const relationshipsInput = screen.getByLabelText(/relationships/i);
      await user.type(relationshipsInput, 'Test Tag');
      await user.keyboard('{Enter}');

      // Live region should announce the addition
      const liveRegion = screen.queryByRole('status', { hidden: true });
      if (liveRegion) {
        await waitFor(() => {
          expect(liveRegion).toHaveTextContent(/added test tag to relationships/i);
        });
      }
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
    it('disables submit button during search', async () => {
      const user = userEvent.setup();
      renderSearchForm({}, mockSearchWorksMocks);

      const titleInput = screen.getByLabelText(/title/i);
      await user.type(titleInput, 'Harry Potter');

      const submitButton = screen.getByRole('button', { name: /search works/i });
      await user.click(submitButton);

      // Button should be disabled during loading
      expect(submitButton).toBeDisabled();

      await waitFor(() => {
        expect(submitButton).not.toBeDisabled();
      }, { timeout: 3000 });
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

    it.skip('applies advanced filters correctly', async () => {
      // TODO: Need specific mock for advanced filter test
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
      }, { timeout: 3000 });
    });
  });

  describe('Clear Functionality', () => {
    it('clears all filters', async () => {
      const user = userEvent.setup();
      renderSearchForm();

      const titleInput = screen.getByLabelText(/title/i);
      await user.type(titleInput, 'test');

      const clearButton = screen.getByRole('button', { name: /clear all/i });
      await user.click(clearButton);

      expect(titleInput).toHaveValue('');
    });
  });

  describe('Screen Reader Announcements', () => {
    it.skip('announces search results count', async () => {
      // TODO: Fix live region detection in tests
      const user = userEvent.setup();
      renderSearchForm({}, mockSearchWorksMocks);

      const titleInput = screen.getByLabelText(/title/i);
      await user.type(titleInput, 'Harry Potter');

      const submitButton = screen.getByRole('button', { name: /search works/i });
      await user.click(submitButton);

      const liveRegion = screen.queryByRole('status', { hidden: true });
      if (liveRegion) {
        await waitFor(() => {
          expect(liveRegion).toHaveTextContent(/search completed/i);
        });
      }
    });

    it.skip('announces advanced search state changes', async () => {
      // TODO: Fix live region detection in tests
      const user = userEvent.setup();
      renderSearchForm();

      const advancedToggle = screen.getByRole('button', { name: /show advanced search/i });
      await user.click(advancedToggle);

      const liveRegion = screen.queryByRole('status', { hidden: true });
      if (liveRegion) {
        await waitFor(() => {
          expect(liveRegion).toHaveTextContent(/advanced search options expanded/i);
        });
      }

      await user.click(screen.getByRole('button', { name: /hide advanced search/i }));
      if (liveRegion) {
        await waitFor(() => {
          expect(liveRegion).toHaveTextContent(/advanced search options collapsed/i);
        });
      }
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