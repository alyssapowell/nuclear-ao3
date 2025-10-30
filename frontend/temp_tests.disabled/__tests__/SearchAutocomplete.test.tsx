import React from 'react'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import SearchAutocomplete from '../SearchAutocomplete'
import { getSearchSuggestions } from '@/lib/api'

// Mock the API
jest.mock('@/lib/api', () => ({
  getSearchSuggestions: jest.fn(),
}))

const mockGetSearchSuggestions = getSearchSuggestions as jest.MockedFunction<typeof getSearchSuggestions>

describe('SearchAutocomplete', () => {
  const defaultProps = {
    value: '',
    onChange: jest.fn(),
    onSearch: jest.fn(),
    placeholder: 'Search...',
  }

  beforeEach(() => {
    jest.clearAllMocks()
    mockGetSearchSuggestions.mockResolvedValue({
      authors: [],
      tags: [],
      works: []
    })
  })

  it('renders with proper accessibility features', () => {
    render(<SearchAutocomplete {...defaultProps} />)
    
    const searchInput = screen.getByRole('combobox')
    expect(searchInput).toBeInTheDocument()
    expect(searchInput).toHaveAttribute('aria-label', 'Search...')
    expect(searchInput).toHaveAttribute('aria-expanded', 'false')
    expect(searchInput).toHaveAttribute('aria-autocomplete', 'list')
    expect(searchInput).toHaveAttribute('aria-describedby')
    
    // Check that help text exists
    expect(screen.getByText(/Type to search for works/)).toBeInTheDocument()
  })

  it('calls onChange when user types', async () => {
    const onChange = jest.fn()
    
    render(<SearchAutocomplete {...defaultProps} onChange={onChange} />)
    
    const input = screen.getByPlaceholderText('Search...')
    fireEvent.change(input, { target: { value: 'test' } })
    
    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith('test')
  })

  it('shows accessible loading indicator when fetching suggestions', async () => {
    // Mock a delayed response
    mockGetSearchSuggestions.mockImplementation(() => 
      new Promise(resolve => setTimeout(() => resolve({ authors: [], tags: [], works: [] }), 100))
    )
    
    render(<SearchAutocomplete {...defaultProps} value="test" />)
    
    // Wait for the loading state to appear
    await waitFor(() => {
      const spinner = screen.getByTestId('loading-spinner')
      expect(spinner).toBeInTheDocument()
      expect(spinner).toHaveAttribute('aria-label', 'Loading suggestions')
      
      // Check for loading announcement
      expect(screen.getByText('Searching for suggestions...')).toBeInTheDocument()
    })
  })

  it('displays suggestions with proper accessibility markup', async () => {
    const mockSuggestions = {
      works: [
        { title: 'Test Work 1', id: '1', kudos: 10 },
        { title: 'Test Work 2', id: '2', kudos: 20 }
      ],
      tags: [
        { name: 'angst', count: 100 },
        { name: 'fluff', count: 50 }
      ],
      authors: [
        { name: 'TestAuthor', work_count: 5 }
      ]
    }
    
    mockGetSearchSuggestions.mockResolvedValue(mockSuggestions)
    
    render(<SearchAutocomplete {...defaultProps} value="test" />)
    
    await waitFor(() => {
      // Check that suggestions list has proper accessibility
      const suggestionsList = screen.getByRole('listbox')
      expect(suggestionsList).toBeInTheDocument()
      expect(suggestionsList).toHaveAttribute('aria-label', expect.stringContaining('Search suggestions'))
      
      // Check individual suggestions
      const suggestions = screen.getAllByRole('option')
      expect(suggestions).toHaveLength(5)
      
      // Check first suggestion accessibility
      expect(suggestions[0]).toHaveAttribute('aria-selected', 'false')
      expect(suggestions[0]).toHaveAttribute('id', 'suggestion-0')
      
      // Check content is present
      expect(screen.getByText('Test Work 1')).toBeInTheDocument()
      expect(screen.getByText('angst')).toBeInTheDocument()
      expect(screen.getByText('TestAuthor')).toBeInTheDocument()
    })
  })

  it('handles keyboard navigation with accessibility announcements', async () => {
    const user = userEvent.setup()
    const onSearch = jest.fn()
    
    const mockSuggestions = {
      works: [
        { title: 'Work 1', id: '1' },
        { title: 'Work 2', id: '2' }
      ],
      tags: [],
      authors: []
    }
    
    mockGetSearchSuggestions.mockResolvedValue(mockSuggestions)
    
    render(<SearchAutocomplete {...defaultProps} value="test" onSearch={onSearch} />)
    
    await waitFor(() => {
      expect(screen.getByText('Work 1')).toBeInTheDocument()
    })
    
    const input = screen.getByRole('combobox')
    input.focus()
    
    // Navigate down - should open suggestions and select first item
    await user.keyboard('{ArrowDown}')
    
    await waitFor(() => {
      // Check ARIA attributes
      expect(input).toHaveAttribute('aria-expanded', 'true')
      const suggestions = screen.getAllByRole('option')
      expect(suggestions[0]).toHaveAttribute('aria-selected', 'true')
      expect(input).toHaveAttribute('aria-activedescendant', 'suggestion-0')
    })
    
    // Navigate down again
    await user.keyboard('{ArrowDown}')
    
    await waitFor(() => {
      const suggestions = screen.getAllByRole('option')
      expect(suggestions[1]).toHaveAttribute('aria-selected', 'true')
      expect(input).toHaveAttribute('aria-activedescendant', 'suggestion-1')
    })
    
    // Press Enter to select
    await user.keyboard('{Enter}')
    
    expect(onSearch).toHaveBeenCalledWith('Work 2')
    expect(input).toHaveAttribute('aria-expanded', 'false')
  })

  it('handles Escape key to close suggestions', async () => {
    const user = userEvent.setup()
    
    const mockSuggestions = {
      works: [{ title: 'Work 1', id: '1' }],
      tags: [],
      authors: []
    }
    
    mockGetSearchSuggestions.mockResolvedValue(mockSuggestions)
    
    render(<SearchAutocomplete {...defaultProps} value="test" />)
    
    await waitFor(() => {
      expect(screen.getByText('Work 1')).toBeInTheDocument()
    })
    
    const input = screen.getByPlaceholderText('Search...')
    input.focus()
    await user.keyboard('{Escape}')
    
    await waitFor(() => {
      expect(screen.queryByText('Work 1')).not.toBeInTheDocument()
    })
  })

  it('calls onSearch when clicking a suggestion', async () => {
    const user = userEvent.setup()
    const onSearch = jest.fn()
    
    const mockSuggestions = {
      works: [{ title: 'Clicked Work', id: '1' }],
      tags: [],
      authors: []
    }
    
    mockGetSearchSuggestions.mockResolvedValue(mockSuggestions)
    
    render(<SearchAutocomplete {...defaultProps} value="test" onSearch={onSearch} />)
    
    await waitFor(() => {
      expect(screen.getByText('Clicked Work')).toBeInTheDocument()
    })
    
    await user.click(screen.getByText('Clicked Work'))
    
    expect(onSearch).toHaveBeenCalledWith('Clicked Work')
  })

  it('handles API errors gracefully', async () => {
    const user = userEvent.setup()
    
    mockGetSearchSuggestions.mockRejectedValue(new Error('API Error'))
    
    render(<SearchAutocomplete {...defaultProps} value="test" />)
    
    const input = screen.getByPlaceholderText('Search...')
    await user.type(input, 'error')
    
    // Should not crash and should not show suggestions
    await waitFor(() => {
      expect(screen.queryByText('📚')).not.toBeInTheDocument()
    })
  })

  it('does not fetch suggestions for queries shorter than 2 characters', async () => {
    const user = userEvent.setup()
    
    render(<SearchAutocomplete {...defaultProps} />)
    
    const input = screen.getByPlaceholderText('Search...')
    await user.type(input, 'a')
    
    expect(mockGetSearchSuggestions).not.toHaveBeenCalled()
  })

  it('debounces API calls', async () => {
    const { rerender } = render(<SearchAutocomplete {...defaultProps} />)
    
    // Quickly update value multiple times
    rerender(<SearchAutocomplete {...defaultProps} value="t" />)
    rerender(<SearchAutocomplete {...defaultProps} value="te" />)
    rerender(<SearchAutocomplete {...defaultProps} value="tes" />)
    rerender(<SearchAutocomplete {...defaultProps} value="test" />)
    
    // Should only make one API call after debounce delay
    await waitFor(() => {
      expect(mockGetSearchSuggestions).toHaveBeenCalledTimes(1)
    }, { timeout: 1000 })
  })

  it('displays correct suggestion counts and metadata', async () => {
    const mockSuggestions = {
      works: [{ title: 'Work with Kudos', id: '1', kudos: 42 }],
      tags: [{ name: 'popular-tag', count: 100 }],
      authors: [{ name: 'Prolific Author', work_count: 25 }]
    }
    
    mockGetSearchSuggestions.mockResolvedValue(mockSuggestions)
    
    render(<SearchAutocomplete {...defaultProps} value="test" />)
    
    await waitFor(() => {
      expect(screen.getByText(/42 kudos/)).toBeInTheDocument()
      expect(screen.getByText(/100 uses/)).toBeInTheDocument()
      expect(screen.getByText(/25 works/)).toBeInTheDocument()
    })
  })
})