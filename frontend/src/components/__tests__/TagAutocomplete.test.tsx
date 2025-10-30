import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import TagAutocomplete from '../TagAutocomplete';

// Mock the API call
jest.mock('@/lib/api', () => ({
  searchTags: jest.fn().mockResolvedValue([
    {
      name: 'Harry Potter',
      type: 'character',
      use_count: 150,
      id: '1'
    },
    {
      name: 'Harry Dresden',
      type: 'character', 
      use_count: 45,
      id: '2'
    }
  ])
}));

const defaultProps = {
  value: '',
  onChange: jest.fn(),
  onTagSelect: jest.fn(),
  placeholder: 'Add characters...',
  tagType: 'character' as const,
};

const renderTagAutocomplete = (props = {}) => {
  return render(
    <TagAutocomplete {...defaultProps} {...props} />
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

    it('renders with initial value when provided', () => {
      renderTagAutocomplete({ value: 'Harry Potter, Hermione Granger' });

      const input = screen.getByDisplayValue('Harry Potter, Hermione Granger');
      expect(input).toBeInTheDocument();
    });

    it('renders with correct accessibility attributes', () => {
      renderTagAutocomplete();

      const input = screen.getByRole('combobox');
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

    it('calls onChange when typing', async () => {
      const onChange = jest.fn();
      const user = userEvent.setup();
      renderTagAutocomplete({ onChange });

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'H');

      expect(onChange).toHaveBeenCalledWith('H');
    });

    it('hides suggestions when input is empty', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete({ value: 'Harry' });

      const input = screen.getByPlaceholderText('Add characters...');
      
      // First trigger suggestions
      await user.type(input, ' Potter');
      
      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      });

      // Clear input
      await user.clear(input);

      await waitFor(() => {
        expect(screen.queryByText('Harry Potter')).not.toBeInTheDocument();
      });
    });
  });

  describe('Tag Selection', () => {
    it('calls onTagSelect when clicking on suggestion', async () => {
      const onTagSelect = jest.fn();
      const user = userEvent.setup();
      renderTagAutocomplete({ onTagSelect });

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      });

      const suggestion = screen.getByText('Harry Potter');
      await user.click(suggestion);

      expect(onTagSelect).toHaveBeenCalledWith({ name: 'Harry Potter' });
    });

    it('adds tag on Enter key press', async () => {
      const onChange = jest.fn();
      const user = userEvent.setup();
      renderTagAutocomplete({ onChange });

      const input = screen.getByPlaceholderText('Add characters...');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      });

      await user.keyboard('{ArrowDown}{Enter}');

      expect(onChange).toHaveBeenCalledWith('Harry Potter, ');
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

      // Check that aria-activedescendant is set
      await waitFor(() => {
        expect(input).toHaveAttribute('aria-activedescendant');
      });
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

      await waitFor(() => {
        expect(screen.queryByText('Harry Potter')).not.toBeInTheDocument();
      });
    });
  });

  describe('Accessibility Features', () => {
    it('has proper ARIA attributes', () => {
      renderTagAutocomplete();

      const input = screen.getByRole('combobox');
      expect(input).toHaveAttribute('aria-autocomplete', 'list');
      expect(input).toHaveAttribute('aria-expanded', 'false');
      expect(input).toHaveAttribute('role', 'combobox');
    });

    it('updates ARIA attributes when suggestions are shown', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByRole('combobox');
      await user.type(input, 'Harry');

      await waitFor(() => {
        expect(input).toHaveAttribute('aria-expanded', 'true');
      });
    });
  });

  describe('Error Handling', () => {
    it('handles disabled state correctly', () => {
      renderTagAutocomplete({ disabled: true });

      const input = screen.getByRole('combobox');
      expect(input).toBeDisabled();
    });

    it('handles required attribute correctly', () => {
      renderTagAutocomplete({ required: true });

      const input = screen.getByRole('combobox');
      expect(input).toBeRequired();
    });
  });
});