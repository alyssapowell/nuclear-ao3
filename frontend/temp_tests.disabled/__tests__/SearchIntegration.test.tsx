import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { MockedProvider } from '@apollo/client/testing';
import SearchPage from '@/app/search/page';

// Mock Next.js navigation
const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
    back: jest.fn(),
    forward: jest.fn(),
    refresh: jest.fn(),
    replace: jest.fn(),
  }),
  useParams: () => ({ id: 'test-id' }),
  usePathname: () => '/search',
  useSearchParams: () => new URLSearchParams(),
}));

// Mock the GraphQL API gateway
const mockGraphQLResponses = {
  enhancedSearch: {
    data: {
      works: {
        pagination: { total: 3 },
        works: [
          {
            id: '1',
            title: 'The Adventures of Agatha and Reader',
            author: 'test_author',
            summary: 'A magical adventure with Agatha Harkness and Reader.',
            word_count: 15000,
            chapter_count: 5,
            max_chapters: 10,
            rating: 'Teen And Up Audiences',
            status: 'published',
            language: 'en',
            published_date: '2023-01-15T10:30:00Z',
            updated_date: '2023-02-20T14:45:00Z',
            relationships: [{ name: 'Agatha Harkness/Reader', category: 'relationship' }],
            characters: [{ name: 'Agatha Harkness', category: 'character' }],
            freeform_tags: [{ name: 'Magic', category: 'freeform' }, { name: 'Adventure', category: 'freeform' }],
            fandoms: [{ name: 'Marvel Cinematic Universe', category: 'fandom' }],
            kudos_count: 25,
            bookmark_count: 8,
            hit_count: 150,
            comment_count: 12,
            tag_quality_score: 0.75,
            missing_tag_suggestions: ['Reader'],
          },
          {
            id: '2',
            title: 'Reader Insert Adventures',
            author: 'another_author',
            summary: 'Various reader insert stories.',
            word_count: 8500,
            chapter_count: 3,
            max_chapters: null,
            rating: 'General Audiences',
            status: 'published',
            language: 'en',
            published_date: '2023-03-10T14:20:00Z',
            updated_date: '2023-03-15T09:30:00Z',
            relationships: [{ name: 'Various/Reader', category: 'relationship' }],
            characters: [{ name: 'Reader', category: 'character' }],
            freeform_tags: [{ name: 'Fluff', category: 'freeform' }],
            fandoms: [{ name: 'Multifandom', category: 'fandom' }],
            kudos_count: 12,
            bookmark_count: 3,
            hit_count: 75,
            comment_count: 5,
            tag_quality_score: 0.65,
            missing_tag_suggestions: [],
          },
          {
            id: '3',
            title: 'Magical Encounters',
            author: 'magic_writer',
            summary: 'When magic meets reality.',
            word_count: 22000,
            chapter_count: 8,
            max_chapters: 12,
            rating: 'Mature',
            status: 'published',
            language: 'en',
            published_date: '2023-02-05T16:45:00Z',
            updated_date: '2023-04-01T11:15:00Z',
            relationships: [{ name: 'Agatha Harkness/Wanda Maximoff', category: 'relationship' }],
            characters: [
              { name: 'Agatha Harkness', category: 'character' },
              { name: 'Wanda Maximoff', category: 'character' }
            ],
            freeform_tags: [
              { name: 'Magic', category: 'freeform' },
              { name: 'Romance', category: 'freeform' },
              { name: 'Hurt/Comfort', category: 'freeform' }
            ],
            fandoms: [{ name: 'Marvel Cinematic Universe', category: 'fandom' }],
            kudos_count: 45,
            bookmark_count: 18,
            hit_count: 320,
            comment_count: 28,
            tag_quality_score: 0.92,
            missing_tag_suggestions: [],
          },
        ],
      },
    },
    recommendations: [
      {
        type: 'missing_character',
        title: 'Missing Character Tags',
        description: 'We detected potential missing character tags from your relationship search',
        suggestions: ['Reader', 'You'],
        confidence_score: 0.88,
        category: 'character',
      },
      {
        type: 'canonical_tags',
        title: 'Canonical Tag Suggestions',
        description: 'These canonical tags might improve your search results',
        suggestions: ['Fluff', 'Angst', 'Hurt/Comfort'],
        confidence_score: 0.72,
        category: 'freeform',
      },
    ],
  },
};

describe('Search Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    
    // Mock the GraphQL fetch calls
    global.fetch = jest.fn().mockImplementation((url, options) => {
      if (url.includes('/graphql')) {
        const body = JSON.parse(options.body);
        const query = body.query;
        
        if (query.includes('SearchWorks') || query.includes('works')) {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve(mockGraphQLResponses.enhancedSearch),
          });
        }
      }
      
      return Promise.resolve({
        ok: false,
        json: () => Promise.resolve({ error: 'Not found' }),
      });
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('Complete Search Flow', () => {
    it('performs end-to-end search with smart recommendations', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      // 1. Verify initial page load
      expect(screen.getAllByText('Enhanced Search')[0]).toBeInTheDocument();
      expect(screen.getByText('Search the archive with powerful filtering and get intelligent recommendations.')).toBeInTheDocument();

      // 2. Enter search query
      const searchInput = screen.getByPlaceholderText(/search for works/i);
      await user.type(searchInput, 'Agatha Harkness/Reader');

      // 3. Submit search
      const searchButton = screen.getByRole('button', { name: /search/i });
      await user.click(searchButton);

      // 4. Wait for loading to complete and results to appear
      await waitFor(() => {
        expect(screen.getByText('Search Results (3)')).toBeInTheDocument();
      });

      // 5. Verify search results are displayed
      expect(screen.getByText('The Adventures of Agatha and Reader')).toBeInTheDocument();
      expect(screen.getByText('Reader Insert Adventures')).toBeInTheDocument();
      expect(screen.getByText('Magical Encounters')).toBeInTheDocument();

      // 6. Verify smart recommendations are shown
      expect(screen.getByText('🤖 Smart Recommendations')).toBeInTheDocument();
      expect(screen.getByText('Missing Character Tags')).toBeInTheDocument();
      expect(screen.getByText('88% confidence')).toBeInTheDocument();

      // 7. Test recommendation interaction
      const readerSuggestion = screen.getByRole('button', { name: 'Reader' });
      await user.click(readerSuggestion);

      // The recommendation should be applied (this would typically update the search)
      expect(readerSuggestion).toBeInTheDocument();
    });

    it('handles search with advanced filters', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      // 1. Open advanced filters
      const advancedToggle = screen.getByText(/advanced filters/i);
      await user.click(advancedToggle);

      // 2. Set rating filter
      const teenRatingCheckbox = screen.getByLabelText(/teen and up/i);
      await user.click(teenRatingCheckbox);

      // 3. Set completion status filter
      const completeCheckbox = screen.getByLabelText(/complete works only/i);
      await user.click(completeCheckbox);

      // 4. Enter search query
      const searchInput = screen.getByPlaceholderText(/search for works/i);
      await user.type(searchInput, 'magic');

      // 5. Submit search
      const searchButton = screen.getByRole('button', { name: /search/i });
      await user.click(searchButton);

      // 6. Verify filters are applied in the API call
      await waitFor(() => {
        expect(global.fetch).toHaveBeenCalledWith(
          'http://localhost:8080/graphql',
          expect.objectContaining({
            method: 'POST',
            body: expect.stringContaining('Teen And Up Audiences'),
          })
        );
      });

      // 7. Verify search results appear
      await waitFor(() => {
        expect(screen.getByText('Search Results (3)')).toBeInTheDocument();
      });
    });

    it('provides real-time tag autocomplete functionality', async () => {
      const user = userEvent.setup();
      
      // Mock tag autocomplete response
      global.fetch = jest.fn().mockImplementation((url, options) => {
        if (url.includes('/graphql')) {
          const body = JSON.parse(options.body);
          if (body.query.includes('TagAutocomplete')) {
            return Promise.resolve({
              ok: true,
              json: () => Promise.resolve({
                data: {
                  tags: {
                    autocomplete: {
                      suggestions: [
                        {
                          id: '1',
                          name: 'Agatha Harkness',
                          type: 'CHARACTER',
                          canonical: true,
                          useCount: 150,
                          description: 'The witch from WandaVision',
                          relationships: [],
                        },
                        {
                          id: '2',
                          name: 'Agatha Christie',
                          type: 'CHARACTER',
                          canonical: true,
                          useCount: 25,
                          description: 'The mystery writer',
                          relationships: [],
                        },
                      ],
                    },
                  },
                },
              }),
            });
          }
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockGraphQLResponses.enhancedSearch),
        });
      });

      render(<SearchPage />);

      // 1. Open advanced filters to access tag inputs
      const advancedToggle = screen.getByText(/advanced filters/i);
      await user.click(advancedToggle);

      // 2. Type in character input to trigger autocomplete
      const characterInput = screen.getByPlaceholderText(/add characters/i);
      await user.type(characterInput, 'Agatha');

      // 3. Wait for autocomplete suggestions
      await waitFor(() => {
        expect(screen.getByText('Agatha Harkness')).toBeInTheDocument();
        expect(screen.getByText('Agatha Christie')).toBeInTheDocument();
      });

      // 4. Click on a suggestion
      const suggestion = screen.getByText('Agatha Harkness');
      await user.click(suggestion);

      // 5. Verify tag was added
      expect(screen.getByText('Agatha Harkness')).toBeInTheDocument();
      expect(characterInput).toHaveValue('');
    });

    it('shows loading states during search operations', async () => {
      const user = userEvent.setup();
      
      // Mock delayed response to test loading state
      global.fetch = jest.fn().mockImplementation(() => 
        new Promise(resolve => 
          setTimeout(() => resolve({
            ok: true,
            json: () => Promise.resolve(mockGraphQLResponses.enhancedSearch),
          }), 100)
        )
      );

      render(<SearchPage />);

      // 1. Enter search query
      const searchInput = screen.getByPlaceholderText(/search for works/i);
      await user.type(searchInput, 'test');

      // 2. Submit search
      const searchButton = screen.getByRole('button', { name: /search/i });
      await user.click(searchButton);

      // 3. Verify loading state appears
      expect(screen.getByText(/searching with enhanced intelligence/i)).toBeInTheDocument();
      expect(searchButton).toBeDisabled();

      // 4. Wait for search to complete
      await waitFor(() => {
        expect(screen.queryByText(/searching with enhanced intelligence/i)).not.toBeInTheDocument();
        expect(searchButton).not.toBeDisabled();
      });

      // 5. Verify results appear
      expect(screen.getByText('Search Results (3)')).toBeInTheDocument();
    });

    it('handles search errors gracefully', async () => {
      const user = userEvent.setup();
      
      // Mock error response
      global.fetch = jest.fn().mockRejectedValue(new Error('Network error'));

      render(<SearchPage />);

      // 1. Enter search query
      const searchInput = screen.getByPlaceholderText(/search for works/i);
      await user.type(searchInput, 'test');

      // 2. Submit search
      const searchButton = screen.getByRole('button', { name: /search/i });
      await user.click(searchButton);

      // 3. Wait for error message
      await waitFor(() => {
        expect(screen.getByText(/search failed/i)).toBeInTheDocument();
      });

      // 4. Verify error state is shown
      expect(screen.queryByText('Search Results')).not.toBeInTheDocument();
    });
  });

  describe('Work Display and Interaction', () => {
    it('displays work cards with enhanced features', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      // Perform search
      const searchInput = screen.getByPlaceholderText(/search for works/i);
      await user.type(searchInput, 'Agatha');
      
      const searchButton = screen.getByRole('button', { name: /search/i });
      await user.click(searchButton);

      // Wait for results
      await waitFor(() => {
        expect(screen.getByText('Search Results (3)')).toBeInTheDocument();
      });

      // Verify work details are displayed
      expect(screen.getByText('The Adventures of Agatha and Reader')).toBeInTheDocument();
      expect(screen.getByText('15,000 words')).toBeInTheDocument();
      expect(screen.getByText('5/10 chapters')).toBeInTheDocument();
      expect(screen.getByText('Teen')).toBeInTheDocument();

      // Verify tags are displayed
      expect(screen.getByText('Agatha Harkness/Reader')).toBeInTheDocument();
      expect(screen.getByText('Magic')).toBeInTheDocument();
      expect(screen.getByText('Marvel Cinematic Universe')).toBeInTheDocument();

      // Verify enhanced features
      expect(screen.getByText('75% tag quality')).toBeInTheDocument();
      expect(screen.getByText('Missing: Reader')).toBeInTheDocument();
    });

    it('allows filtering by clicking on tags', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      // Perform initial search
      const searchInput = screen.getByPlaceholderText(/search for works/i);
      await user.type(searchInput, 'magic');
      
      const searchButton = screen.getByRole('button', { name: /search/i });
      await user.click(searchButton);

      await waitFor(() => {
        expect(screen.getByText('Search Results (3)')).toBeInTheDocument();
      });

      // Click on a fandom tag to filter
      const fandomTag = screen.getByRole('link', { name: 'Marvel Cinematic Universe' });
      await user.click(fandomTag);

      // This would typically trigger a new search with that fandom filter
      // For this test, we're just verifying the link exists and is clickable
      expect(fandomTag).toHaveAttribute('href', '/works?fandom=Marvel%20Cinematic%20Universe');
    });
  });

  describe('Smart Recommendations Integration', () => {
    it('applies recommendations to enhance search', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      // Perform search that triggers recommendations
      const searchInput = screen.getByPlaceholderText(/search for works/i);
      await user.type(searchInput, 'Agatha/Reader');
      
      const searchButton = screen.getByRole('button', { name: /search/i });
      await user.click(searchButton);

      await waitFor(() => {
        expect(screen.getByText('🤖 Smart Recommendations')).toBeInTheDocument();
      });

      // Verify recommendations are displayed
      expect(screen.getByText('Missing Character Tags')).toBeInTheDocument();
      expect(screen.getByText('We detected potential missing character tags')).toBeInTheDocument();
      expect(screen.getByText('88% confidence')).toBeInTheDocument();

      // Apply a recommendation
      const readerSuggestion = screen.getByRole('button', { name: 'Reader' });
      await user.click(readerSuggestion);

      // Verify the recommendation application
      expect(readerSuggestion).toBeInTheDocument();
    });

    it('shows different types of recommendations', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      // Perform search
      const searchInput = screen.getByPlaceholderText(/search for works/i);
      await user.type(searchInput, 'magic');
      
      const searchButton = screen.getByRole('button', { name: /search/i });
      await user.click(searchButton);

      await waitFor(() => {
        expect(screen.getByText('🤖 Smart Recommendations')).toBeInTheDocument();
      });

      // Verify multiple recommendation types
      expect(screen.getByText('Missing Character Tags')).toBeInTheDocument();
      expect(screen.getByText('Canonical Tag Suggestions')).toBeInTheDocument();

      // Verify confidence scores are shown
      expect(screen.getByText('88% confidence')).toBeInTheDocument();
      expect(screen.getByText('72% confidence')).toBeInTheDocument();

      // Verify different recommendation categories
      const characterSuggestion = screen.getByRole('button', { name: 'Reader' });
      const freeformSuggestion = screen.getByRole('button', { name: 'Fluff' });
      
      expect(characterSuggestion).toHaveClass('bg-blue-50');
      expect(freeformSuggestion).toHaveClass('bg-gray-50');
    });
  });

  describe('Performance and Responsiveness', () => {
    it('handles rapid search interactions without issues', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      const searchInput = screen.getByPlaceholderText(/search for works/i);
      const searchButton = screen.getByRole('button', { name: /search/i });

      // Perform rapid searches
      await user.type(searchInput, 'test1');
      await user.click(searchButton);
      
      await user.clear(searchInput);
      await user.type(searchInput, 'test2');
      await user.click(searchButton);
      
      await user.clear(searchInput);
      await user.type(searchInput, 'test3');
      await user.click(searchButton);

      // Verify the final search completed
      await waitFor(() => {
        expect(screen.getByText('Search Results (3)')).toBeInTheDocument();
      });

      // Verify multiple API calls were made
      expect(global.fetch).toHaveBeenCalledTimes(3);
    });

    it('debounces tag autocomplete calls appropriately', async () => {
      const user = userEvent.setup();
      
      // Track autocomplete calls
      const autocompleteCallCount = { count: 0 };
      global.fetch = jest.fn().mockImplementation((url, options) => {
        const body = JSON.parse(options.body);
        if (body.query.includes('TagAutocomplete')) {
          autocompleteCallCount.count++;
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            data: { tags: { autocomplete: { suggestions: [] } } },
          }),
        });
      });

      render(<SearchPage />);

      // Open advanced filters
      const advancedToggle = screen.getByText(/advanced filters/i);
      await user.click(advancedToggle);

      // Type rapidly in character input
      const characterInput = screen.getByPlaceholderText(/add characters/i);
      await user.type(characterInput, 'Agatha Harkness', { delay: 10 });

      // Wait a bit for debouncing
      await waitFor(() => {
        // Should have fewer calls than characters typed due to debouncing
        expect(autocompleteCallCount.count).toBeLessThan(15);
      });
    });
  });

  describe('Accessibility and User Experience', () => {
    it('maintains proper focus management throughout search flow', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      // Test tab navigation
      await user.tab();
      const searchInput = screen.getByPlaceholderText(/search for works/i);
      expect(searchInput).toHaveFocus();

      await user.tab();
      const searchButton = screen.getByRole('button', { name: /search/i });
      expect(searchButton).toHaveFocus();

      // Perform search
      await user.type(searchInput, 'test');
      await user.click(searchButton);

      await waitFor(() => {
        expect(screen.getByText('Search Results (3)')).toBeInTheDocument();
      });

      // Focus should remain manageable after search
      expect(document.body).toHaveFocus();
    });

    it('provides appropriate ARIA labels and screen reader support', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      // Verify main search has proper labels
      const searchInput = screen.getByRole('textbox', { name: /search query/i });
      expect(searchInput).toBeInTheDocument();

      // Perform search
      await user.type(searchInput, 'test');
      const searchButton = screen.getByRole('button', { name: /search/i });
      await user.click(searchButton);

      await waitFor(() => {
        expect(screen.getByText('Search Results (3)')).toBeInTheDocument();
      });

      // Verify results have proper headings
      const resultsHeading = screen.getByRole('heading', { name: /search results/i });
      expect(resultsHeading).toBeInTheDocument();

      // Verify recommendation buttons have proper labels
      const recommendationButtons = screen.getAllByRole('button');
      recommendationButtons.forEach(button => {
        expect(button).toHaveAttribute('aria-label');
      });
    });
  });
});