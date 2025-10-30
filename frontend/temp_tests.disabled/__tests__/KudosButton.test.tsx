import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import KudosButton from '../KudosButton';
import { giveKudos } from '@/lib/api';

// Mock the API function
jest.mock('@/lib/api', () => ({
  giveKudos: jest.fn(),
}));

const mockGiveKudos = giveKudos as jest.MockedFunction<typeof giveKudos>;

describe('KudosButton', () => {
  const defaultProps = {
    workId: 'work-123',
    initialKudos: 42,
    hasGivenKudos: false,
    authToken: 'test-token',
  };

  beforeEach(() => {
    mockGiveKudos.mockClear();
  });

  it('renders initial state correctly', () => {
    render(<KudosButton {...defaultProps} />);
    
    expect(screen.getByRole('button')).toBeInTheDocument();
    expect(screen.getByText('Give Kudos')).toBeInTheDocument();
    expect(screen.getByText('(42)')).toBeInTheDocument();
    expect(screen.getByTitle('Leave kudos')).toBeInTheDocument();
  });

  it('renders when user has already given kudos', () => {
    render(<KudosButton {...defaultProps} hasGivenKudos={true} />);
    
    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
    expect(screen.getByText('Kudos Given')).toBeInTheDocument();
    expect(screen.getByText('(42)')).toBeInTheDocument();
    expect(screen.getByTitle('You have already left kudos')).toBeInTheDocument();
  });

  it('shows correct styling when kudos already given', () => {
    render(<KudosButton {...defaultProps} hasGivenKudos={true} />);
    
    const button = screen.getByRole('button');
    expect(button).toHaveClass('bg-red-100', 'text-red-700', 'cursor-not-allowed');
  });

  it('shows correct styling when kudos not given', () => {
    render(<KudosButton {...defaultProps} />);
    
    const button = screen.getByRole('button');
    expect(button).toHaveClass('bg-red-50', 'text-red-600');
  });

  it('successfully gives kudos when clicked', async () => {
    mockGiveKudos.mockResolvedValueOnce(undefined);
    
    render(<KudosButton {...defaultProps} />);
    
    const button = screen.getByRole('button');
    fireEvent.click(button);

    // Should show loading state
    expect(screen.getByText('Giving...')).toBeInTheDocument();
    expect(button).toBeDisabled();

    // Wait for API call to complete
    await waitFor(() => {
      expect(mockGiveKudos).toHaveBeenCalledWith('work-123', 'test-token');
    });

    // Should update to kudos given state
    await waitFor(() => {
      expect(screen.getByText('Kudos Given')).toBeInTheDocument();
      expect(screen.getByText('(43)')).toBeInTheDocument(); // Should increment
      expect(button).toBeDisabled();
    });
  });

  it('handles API error gracefully', async () => {
    const errorMessage = 'Failed to give kudos';
    mockGiveKudos.mockRejectedValueOnce(new Error(errorMessage));
    
    render(<KudosButton {...defaultProps} />);
    
    const button = screen.getByRole('button');
    fireEvent.click(button);

    // Should show loading state
    expect(screen.getByText('Giving...')).toBeInTheDocument();

    // Wait for error to appear
    await waitFor(() => {
      expect(screen.getByText(errorMessage)).toBeInTheDocument();
    });

    // Should return to normal state
    await waitFor(() => {
      expect(screen.getByText('Give Kudos')).toBeInTheDocument();
      expect(screen.getByText('(42)')).toBeInTheDocument(); // Should not increment
      expect(button).not.toBeDisabled();
    });
  });

  it('handles non-Error API failures', async () => {
    mockGiveKudos.mockRejectedValueOnce('String error');
    
    render(<KudosButton {...defaultProps} />);
    
    const button = screen.getByRole('button');
    fireEvent.click(button);

    // Wait for generic error message
    await waitFor(() => {
      expect(screen.getByText('Failed to give kudos')).toBeInTheDocument();
    });
  });

  it('works without auth token', async () => {
    mockGiveKudos.mockResolvedValueOnce(undefined);
    
    render(<KudosButton {...defaultProps} authToken={undefined} />);
    
    const button = screen.getByRole('button');
    fireEvent.click(button);

    await waitFor(() => {
      expect(mockGiveKudos).toHaveBeenCalledWith('work-123', undefined);
    });
  });

  it('prevents multiple clicks while loading', async () => {
    mockGiveKudos.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)));
    
    render(<KudosButton {...defaultProps} />);
    
    const button = screen.getByRole('button');
    
    // Click multiple times rapidly
    fireEvent.click(button);
    fireEvent.click(button);
    fireEvent.click(button);

    // Should only call API once
    expect(mockGiveKudos).toHaveBeenCalledTimes(1);
  });

  it('prevents clicks when kudos already given', () => {
    render(<KudosButton {...defaultProps} hasGivenKudos={true} />);
    
    const button = screen.getByRole('button');
    fireEvent.click(button);

    // Should not call API
    expect(mockGiveKudos).not.toHaveBeenCalled();
  });

  it('shows heart icon with correct fill state', () => {
    // Test unfilled state
    render(<KudosButton {...defaultProps} hasGivenKudos={false} />);
    
    let heartIcon = screen.getByRole('button').querySelector('svg');
    expect(heartIcon).toHaveAttribute('fill', 'none');
    expect(heartIcon).toHaveAttribute('stroke', 'currentColor');
    
    // Test filled state in a separate test component
    const { unmount } = render(<KudosButton {...defaultProps} hasGivenKudos={false} />);
    unmount();
  });

  it('shows filled heart icon when kudos given', () => {
    render(<KudosButton {...defaultProps} hasGivenKudos={true} />);
    
    const heartIcon = screen.getByRole('button').querySelector('svg');
    expect(heartIcon).toHaveAttribute('fill', 'currentColor');
    
    // Test that button text shows kudos given state
    expect(screen.getByText('Kudos Given')).toBeInTheDocument();
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('shows pulse animation while loading', async () => {
    mockGiveKudos.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)));
    
    render(<KudosButton {...defaultProps} />);
    
    const button = screen.getByRole('button');
    fireEvent.click(button);

    // Should show pulse animation
    const heartIcon = button.querySelector('svg');
    expect(heartIcon).toHaveClass('animate-pulse');
  });

  it('handles zero initial kudos', () => {
    render(<KudosButton {...defaultProps} initialKudos={0} />);
    
    expect(screen.getByText('(0)')).toBeInTheDocument();
  });

  it('handles large kudos numbers', () => {
    render(<KudosButton {...defaultProps} initialKudos={9999} />);
    
    expect(screen.getByText('(9999)')).toBeInTheDocument();
  });
});