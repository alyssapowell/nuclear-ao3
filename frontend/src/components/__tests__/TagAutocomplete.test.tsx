import { useState } from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import TagAutocomplete from '../TagAutocomplete';

// Mock the API call
vi.mock('@/lib/api', () => ({
  searchTags: vi.fn().mockResolvedValue([
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
  onChange: vi.fn(),
  onTagSelect: vi.fn(),
  placeholder: 'Add characters...',
  tagType: 'character' as const,
};

// TagAutocomplete is a controlled input: it only searches once its parent
// feeds the typed text back as `value`. This wrapper does that, and still
// reports every change to the onChange spy.
function Controlled(props: Partial<React.ComponentProps<typeof TagAutocomplete>>) {
  const { value: initial = '', onChange, ...rest } = props;
  const [value, setValue] = useState(initial);
  return (
    <TagAutocomplete
      {...defaultProps}
      {...rest}
      value={value}
      onChange={(v) => {
        setValue(v);
        onChange?.(v);
      }}
    />
  );
}

const renderTagAutocomplete = (props = {}) => {
  return render(<Controlled {...props} />);
};

describe('TagAutocomplete', () => {
  beforeEach(() => {
    vi.clearAllMocks();
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

      const input = screen.getByPlaceholderText('Add characters...');
      expect(input).toHaveAttribute('aria-autocomplete', 'list');
      expect(input).toHaveAttribute('aria-expanded', 'false');
      expect(input).toHaveAttribute('role', 'combobox');
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
      const onChange = vi.fn();
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
      const onTagSelect = vi.fn();
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
      const onChange = vi.fn();
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

      const input = screen.getByPlaceholderText('Add characters...');
      expect(input).toHaveAttribute('aria-autocomplete', 'list');
      expect(input).toHaveAttribute('aria-expanded', 'false');
      expect(input).toHaveAttribute('role', 'combobox');
    });

    it('updates ARIA attributes when suggestions are shown', async () => {
      const user = userEvent.setup();
      renderTagAutocomplete();

      const input = screen.getByPlaceholderText('Add characters...');
      
      // Type and wait for debounced search to complete
      await user.type(input, 'Harry');
      
      // Wait for suggestions to appear (debounced 150ms + API call)
      await waitFor(() => {
        expect(screen.getByText('Harry Potter')).toBeInTheDocument();
      }, { timeout: 3000 });
      
      // Then check aria-expanded
      expect(input).toHaveAttribute('aria-expanded', 'true');
    });
  });

  describe('Error Handling', () => {
    it('handles disabled state correctly', () => {
      renderTagAutocomplete({ disabled: true });

      const input = screen.getByPlaceholderText('Add characters...');
      expect(input).toBeDisabled();
    });

    it('handles required attribute correctly', () => {
      renderTagAutocomplete({ required: true });

      const input = screen.getByPlaceholderText('Add characters...');
      expect(input).toBeRequired();
    });
  });
});