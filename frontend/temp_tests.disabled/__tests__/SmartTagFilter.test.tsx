import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import SmartTagFilter, { TagFilters } from '../SmartTagFilter';

describe('SmartTagFilter', () => {
  const mockOnFilterChange = jest.fn();

  beforeEach(() => {
    mockOnFilterChange.mockClear();
  });

  it('renders filter options correctly', () => {
    render(
      <SmartTagFilter 
        onFilterChange={mockOnFilterChange} 
      />
    );
    
    expect(screen.getByText('Smart Tag Filtering')).toBeInTheDocument();
    expect(screen.getByText('Primary relationships only')).toBeInTheDocument();
    expect(screen.getByText('Hide tag spam')).toBeInTheDocument();
    expect(screen.getByText('Show tag prominence')).toBeInTheDocument();
  });

  it('toggles primary relationships only filter', () => {
    render(
      <SmartTagFilter 
        onFilterChange={mockOnFilterChange} 
      />
    );
    
    const switchElement = screen.getByRole('switch', { name: /primary relationships only/i });
    fireEvent.click(switchElement);
    
    expect(mockOnFilterChange).toHaveBeenCalledWith(
      expect.objectContaining({
        primaryOnly: true,
      })
    );
  });

  it('toggles hide tag spam filter', () => {
    render(
      <SmartTagFilter 
        onFilterChange={mockOnFilterChange} 
      />
    );
    
    const switchElement = screen.getByRole('switch', { name: /hide tag spam/i });
    fireEvent.click(switchElement);
    
    expect(mockOnFilterChange).toHaveBeenCalledWith(
      expect.objectContaining({
        hideTagSpam: true,
      })
    );
  });

  it('toggles show tag prominence filter', () => {
    render(
      <SmartTagFilter 
        onFilterChange={mockOnFilterChange} 
      />
    );
    
    const switchElement = screen.getByRole('switch', { name: /show tag prominence/i });
    fireEvent.click(switchElement);
    
    expect(mockOnFilterChange).toHaveBeenCalledWith(
      expect.objectContaining({
        showProminence: false, // Default is true, so clicking makes it false
      })
    );
  });

  it('shows advanced filters when toggled', () => {
    render(
      <SmartTagFilter 
        onFilterChange={mockOnFilterChange} 
      />
    );
    
    // Advanced filters should be hidden initially
    expect(screen.queryByText('Max relationship tags')).not.toBeInTheDocument();
    
    // Toggle advanced filters
    fireEvent.click(screen.getByText('Advanced Filters'));
    
    // Advanced filters should now be visible
    expect(screen.getByText(/Maximum relationship tags/)).toBeInTheDocument();
    expect(screen.getByText(/Minimum words per major tag/)).toBeInTheDocument();
    expect(screen.getByText('Hide drabble collections')).toBeInTheDocument();
  });

  it('updates max relationship tags filter', () => {
    render(
      <SmartTagFilter 
        onFilterChange={mockOnFilterChange} 
      />
    );
    
    // Show advanced filters first
    fireEvent.click(screen.getByText('Advanced Filters'));
    
    const inputs = screen.getAllByRole('slider');
    const input = inputs[0]; // First slider is max relationship tags
    fireEvent.change(input, { target: { value: '25' } });
    
    expect(mockOnFilterChange).toHaveBeenCalledWith(
      expect.objectContaining({
        maxRelationshipTags: 25,
      })
    );
  });

  it('calls onFilterChange when any filter changes', () => {
    render(
      <SmartTagFilter 
        onFilterChange={mockOnFilterChange} 
      />
    );
    
    // Test multiple filter changes
    const primarySwitch = screen.getByRole('switch', { name: /primary relationships only/i });
    const tagSpamSwitch = screen.getByRole('switch', { name: /hide tag spam/i });
    
    fireEvent.click(primarySwitch);
    expect(mockOnFilterChange).toHaveBeenCalledTimes(1);
    
    fireEvent.click(tagSpamSwitch);
    expect(mockOnFilterChange).toHaveBeenCalledTimes(2);
  });

  it('has correct default filter values', () => {
    render(
      <SmartTagFilter 
        onFilterChange={mockOnFilterChange} 
      />
    );
    
    // Check default states
    const primarySwitch = screen.getByRole('switch', { name: /primary relationships only/i });
    const tagSpamSwitch = screen.getByRole('switch', { name: /hide tag spam/i });
    const prominenceSwitch = screen.getByRole('switch', { name: /show tag prominence/i });
    
    expect(primarySwitch).toHaveAttribute('aria-checked', 'false');
    expect(tagSpamSwitch).toHaveAttribute('aria-checked', 'false');
    expect(prominenceSwitch).toHaveAttribute('aria-checked', 'true'); // Default is true
  });
});