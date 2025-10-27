import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { MockedProvider } from '@apollo/client/testing';
import TagAutocomplete from '../TagAutocomplete';
import { TAG_AUTOCOMPLETE } from '@/lib/graphql';

const mockTagAutocompleteMocks = [
  {
    request: {
      query: TAG_AUTOCOMPLETE,
      variables: {
        query: 'Harry',
        limit: 10,
        types: ['CHARACTER'],
      },
    },
    result: {
      data: {
        tags: {
          autocomplete: {
            suggestions: [
              {
                id: '1',
                name: 'Harry Potter',
                type: 'CHARACTER',
                canonical: true,
                useCount: 150,
                description: 'The Boy Who Lived',
                relationships: [],
              },
              {
                id: '2',
                name: 'Harry Dresden',
                type: 'CHARACTER',
                canonical: true,
                useCount: 45,
                description: 'Wizard Detective',
                relationships: [],
              },
            ],
          },
        },
      },
    },
  },
  {
    request: {
      query: TAG_AUTOCOMPLETE,
      variables: {
        query: 'Steve',
        limit: 10,
        types: ['CHARACTER'],
      },
    },
    result: {
      data: {
        tags: {
          autocomplete: {
            suggestions: [
              {
                id: '3',
                name: 'Steve Rogers',
                type: 'CHARACTER',
                canonical: true,
                useCount: 200,
                description: 'Captain America',
                relationships: [],
              },
            ],
          },
        },
      },
    },
  },
];

const defaultProps = {
  type: 'CHARACTER' as const,
  placeholder: 'Add characters...',
  onTagsChange: jest.fn(),
  initialTags: [],
};

const renderTagAutocomplete = (props = {}, mocks = mockTagAutocompleteMocks) => {
  return render(
    <MockedProvider mocks={mocks} addTypename={false}>
      <TagAutocomplete {...defaultProps} {...props} />
    </MockedProvider>
  );
};

describe('TagAutocomplete', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders the input with correct placeholder', () => {
      renderTagAutocomplete();

      expect(screen.getByPlaceholderText('Add characters...')).toBeInTheDocument();
    });

    it('renders initial tags when provided', () => {
      const initialTags = [
        { id: '1', name: 'Harry Potter', type: 'CHARACTER' },
        { id: '2', name: 'Hermione Granger', type: 'CHARACTER' },
      ];

      renderTagAutocomplete({ initialTags });

      expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      expect(screen.getByText('Hermione Granger')).toBeInTheDocument();
    });

    it('renders with correct accessibility attributes', () => {
      renderTagAutocomplete();

      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('aria-autocomplete', 'list');
      expect(input).toHaveAttribute('aria-expanded', 'false');
    });
  });

  describe('Tag Input and Suggestions', () => {
    it('shows suggestions when typing', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
        expect(screen.getByText('Harry Dresden')).toBeInTheDocument();
      });
    });

    it('shows tag type badges in suggestions', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getAllByText('CHARACTER')).toHaveLength(2);
      });
    });

    it('shows use count in suggestions', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('150 uses')).toBeInTheDocument();
        expect(screen.getByText('45 uses')).toBeInTheDocument();
      });
    });

    it('shows canonical badge for canonical tags', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getAllByText('Canonical')).toHaveLength(2);
      });
    });

    it('hides suggestions when input is empty', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      });

      await user.clear(input);

      await waitFor(() => {
        expect(screen.queryByText('Harry Potter')).not.toBeInTheDocument();
      });
    });

    it('filters suggestions based on input', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Steve');

      await waitFor(() => {
        expect(screen.getByText('Steve Rogers')).toBeInTheDocument();
        expect(screen.queryByText('Harry Potter')).not.toBeInTheDocument();
      });
    });
  });

  describe('Tag Selection and Management', () => {
    it('adds tag when clicking on suggestion', async () => {
      const user = userEvent.setup();
      const onTagsChange = jest.fn();
      renderTagAutocomplete({ onTagsChange });

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      });

      const suggestion = screen.getByText('Harry Potter');
      await user.click(suggestion);

      expect(onTagsChange).toHaveBeenCalledWith([
        {
          id: '1',
          name: 'Harry Potter',
          type: 'CHARACTER',
          canonical: true,
          useCount: 150,
          description: 'The Boy Who Lived',
          relationships: [],
        },
      ]);
    });

    it('clears input after adding tag', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      });

      const suggestion = screen.getByText('Harry Potter');
      await user.click(suggestion);

      expect(input).toHaveValue('');
    });

    it('removes tag when clicking X button', async () => {
      const user = userEvent.setup();
      const onTagsChange = jest.fn();
      const initialTags = [
        { id: '1', name: 'Harry Potter', type: 'CHARACTER' },
      ];

      renderTagAutocomplete({ initialTags, onTagsChange });

      const removeButton = screen.getByRole('button', { name: /remove harry potter/i });
      await user.click(removeButton);

      expect(onTagsChange).toHaveBeenCalledWith([]);
    });

    it('prevents duplicate tags', async () => {
      const user = userEvent.setup();
      const onTagsChange = jest.fn();
      const initialTags = [
        { id: '1', name: 'Harry Potter', type: 'CHARACTER' },
      ];

      renderTagAutocomplete({ initialTags, onTagsChange });

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      });

      const suggestion = screen.getByText('Harry Potter');
      await user.click(suggestion);

      // Should not add duplicate
      expect(onTagsChange).not.toHaveBeenCalled();
    });

    it('adds tag on Enter key press', async () => {
      const user = userEvent.setup();
      const onTagsChange = jest.fn();
      renderTagAutocomplete({ onTagsChange });

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      });

      await user.keyboard('{ArrowDown}{Enter}');

      expect(onTagsChange).toHaveBeenCalledWith([
        {
          id: '1',
          name: 'Harry Potter',
          type: 'CHARACTER',
          canonical: true,
          useCount: 150,
          description: 'The Boy Who Lived',
          relationships: [],
        },
      ]);
    });
  });

  describe('Keyboard Navigation', () => {
    it('navigates suggestions with arrow keys', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      });

      await user.keyboard('{ArrowDown}');

      // First suggestion should be highlighted
      const firstSuggestion = screen.getByText('Harry Potter').closest('li');
      expect(firstSuggestion).toHaveClass('bg-blue-50');

      await user.keyboard('{ArrowDown}');

      // Second suggestion should be highlighted
      const secondSuggestion = screen.getByText('Harry Dresden').closest('li');
      expect(secondSuggestion).toHaveClass('bg-blue-50');
    });

    it('wraps navigation at end of list', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      });

      // Navigate to last item and then down again
      await user.keyboard('{ArrowDown}{ArrowDown}{ArrowDown}');

      // Should wrap to first item
      const firstSuggestion = screen.getByText('Harry Potter').closest('li');
      expect(firstSuggestion).toHaveClass('bg-blue-50');
    });

    it('supports escape key to close suggestions', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      });

      await user.keyboard('{Escape}');

      expect(screen.queryByText('Harry Potter')).not.toBeInTheDocument();
    });
  });

  describe('Tag Display and Styling', () => {
    it('displays tags with correct styling', () => {
      const initialTags = [
        { id: '1', name: 'Harry Potter', type: 'CHARACTER' },
        { id: '2', name: 'Hermione Granger', type: 'CHARACTER' },
      ];

      renderTagAutocomplete({ initialTags });

      const tags = screen.getAllByText(/Harry Potter|Hermione Granger/);
      tags.forEach(tag => {
        const tagElement = tag.closest('span');
        expect(tagElement).toHaveClass('bg-blue-100', 'text-blue-800');
      });
    });

    it('shows different colors for different tag types', () => {
      const relationshipTags = [
        { id: '1', name: 'Harry Potter/Draco Malfoy', type: 'RELATIONSHIP' },
      ];

      renderTagAutocomplete({ 
        initialTags: relationshipTags,
        type: 'RELATIONSHIP'
      });

      const tagElement = screen.getByText('Harry Potter/Draco Malfoy').closest('span');
      expect(tagElement).toHaveClass('bg-pink-100', 'text-pink-800');
    });

    it('truncates long tag names', () => {
      const longTags = [
        { id: '1', name: 'This is a very long tag name that should be truncated', type: 'CHARACTER' },
      ];

      renderTagAutocomplete({ initialTags: longTags });

      expect(screen.getByText('This is a very long...')).toBeInTheDocument();
    });
  });

  describe('Loading and Error States', () => {
    it('shows loading state while fetching suggestions', async () => {
      const user = userEvent.setup();
      const delayedMocks = [
        {
          ...mockTagAutocompleteMocks[0],
          delay: 1000,
        },
      ];

      renderTagAutocomplete({}, delayedMocks);

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      expect(screen.getByText('Searching...')).toBeInTheDocument();
    });

    it('handles empty results gracefully', async () => {
      const user = userEvent.setup();
      const emptyMocks = [
        {
          request: {
            query: TAG_AUTOCOMPLETE,
            variables: {
              query: 'XYZ',
              limit: 10,
              types: ['CHARACTER'],
            },
          },
          result: {
            data: {
              tags: {
                autocomplete: {
                  suggestions: [],
                },
              },
            },
          },
        },
      ];

      renderTagAutocomplete({}, emptyMocks);

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'XYZ');

      await waitFor(() => {
        expect(screen.getByText('No tags found')).toBeInTheDocument();
      });
    });

    it('handles GraphQL errors gracefully', async () => {
      const user = userEvent.setup();
      const errorMocks = [
        {
          request: {
            query: TAG_AUTOCOMPLETE,
            variables: {
              query: 'Harry',
              limit: 10,
              types: ['CHARACTER'],
            },
          },
          error: new Error('Network error'),
        },
      ];

      renderTagAutocomplete({}, errorMocks);

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Error loading suggestions')).toBeInTheDocument();
      });
    });
  });

  describe('Accessibility Features', () => {
    it('has proper ARIA attributes', () => {
      renderTagAutocomplete();

      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('aria-autocomplete', 'list');
      expect(input).toHaveAttribute('aria-expanded', 'false');
      expect(input).toHaveAttribute('role', 'combobox');
    });

    it('updates ARIA attributes when suggestions are shown', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByRole('textbox');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(input).toHaveAttribute('aria-expanded', 'true');
        expect(input).toHaveAttribute('aria-activedescendant');
      });
    });

    it('has proper role and labels for suggestions list', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        const suggestionsList = screen.getByRole('listbox');
        expect(suggestionsList).toBeInTheDocument();

        const suggestions = screen.getAllByRole('option');
        expect(suggestions).toHaveLength(2);
      });
    });

    it('has proper labels for remove buttons', () => {
      const initialTags = [
        { id: '1', name: 'Harry Potter', type: 'CHARACTER' },
      ];

      renderTagAutocomplete({ initialTags });

      const removeButton = screen.getByRole('button', { name: /remove harry potter/i });
      expect(removeButton).toBeInTheDocument();
    });
  });

  describe('Performance and Debouncing', () => {
    it('debounces API calls when typing quickly', async () => {
      const user = userEvent.setup();
      const mockFetch = jest.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({
          data: {
            tags: {
              autocomplete: {
                suggestions: [],
              },
            },
          },
        }),
      });

      global.fetch = mockFetch;

      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      
      // Type quickly
      await user.type(input, 'Harry Potter', { delay: 10 });

      // Should only make one API call due to debouncing
      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledTimes(1);
      });
    });

    it('cancels previous requests when new input is provided', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      
      // Start typing one thing
      await user.type(input, 'Harry');
      
      // Quickly change to something else
      await user.clear(input);
      await user.type(input, 'Steve');

      await waitFor(() => {
        expect(screen.getByText('Steve Rogers')).toBeInTheDocument();
        expect(screen.queryByText('Harry Potter')).not.toBeInTheDocument();
      });
    });
  });

  describe('Integration with Forms', () => {
    it('integrates with form validation', () => {
      const initialTags = [
        { id: '1', name: 'Harry Potter', type: 'CHARACTER' },
      ];

      renderTagAutocomplete({ 
        initialTags,
        required: true,
        'aria-invalid': false,
      });

      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('required');
      expect(input).toHaveAttribute('aria-invalid', 'false');
    });

    it('updates form state when tags change', async () => {
      const user = userEvent.setup();
      const onTagsChange = jest.fn();
      
      renderTagAutocomplete({ onTagsChange });

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      });

      const suggestion = screen.getByText('Harry Potter');
      await user.click(suggestion);

      expect(onTagsChange).toHaveBeenCalledWith([
        expect.objectContaining({
          id: '1',
          name: 'Harry Potter',
          type: 'CHARACTER',
        }),
      ]);
    });
  });
});