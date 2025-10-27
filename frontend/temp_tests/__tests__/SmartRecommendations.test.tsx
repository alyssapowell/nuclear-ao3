import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import SmartRecommendations from '../SmartRecommendations';

const mockRecommendations = [
  {
    type: 'missing_character',
    title: 'Missing Character Tags',
    description: 'We detected potential missing character tags based on your search',
    suggestions: ['Reader', 'You', 'Original Character'],
    confidence_score: 0.85,
    category: 'character',
  },
  {
    type: 'canonical_tags',
    title: 'Canonical Tag Suggestions',
    description: 'These canonical tags might improve your search results',
    suggestions: ['Angst', 'Hurt/Comfort', 'Fluff'],
    confidence_score: 0.72,
    category: 'freeform',
  },
  {
    type: 'related_tags',
    title: 'Related Tags',
    description: 'Tags commonly used with your search terms',
    suggestions: ['Slow Burn', 'Enemies to Lovers', 'Mutual Pining'],
    confidence_score: 0.68,
    category: 'freeform',
  },
  {
    type: 'tag_quality',
    title: 'Tag Quality Improvement',
    description: 'Consider adding these tags for better discoverability',
    suggestions: ['Marvel Cinematic Universe', 'Alternate Universe - Modern Setting'],
    confidence_score: 0.91,
    category: 'fandom',
  },
  {
    type: 'relationship_expansion',
    title: 'Relationship Expansion',
    description: 'Detected relationships that could be expanded',
    suggestions: ['Steve Rogers/Tony Stark', 'Steve Rogers & Tony Stark'],
    confidence_score: 0.78,
    category: 'relationship',
  },
];

const defaultProps = {
  recommendations: mockRecommendations,
  onApplyRecommendation: jest.fn(),
};

describe('SmartRecommendations', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders the component with title', () => {
      render(<SmartRecommendations {...defaultProps} />);

      expect(screen.getByText('Smart Recommendations')).toBeInTheDocument();
      expect(screen.getByText('🤖')).toBeInTheDocument();
    });

    it('renders all recommendation cards', () => {
      render(<SmartRecommendations {...defaultProps} />);

      expect(screen.getByText('Missing Character Tags')).toBeInTheDocument();
      expect(screen.getByText('Canonical Tag Suggestions')).toBeInTheDocument();
      expect(screen.getByText('Related Tags')).toBeInTheDocument();
      expect(screen.getByText('Tag Quality Improvement')).toBeInTheDocument();
      expect(screen.getByText('Relationship Expansion')).toBeInTheDocument();
    });

    it('displays confidence scores correctly', () => {
      render(<SmartRecommendations {...defaultProps} />);

      expect(screen.getByText('85% confidence')).toBeInTheDocument();
      expect(screen.getByText('72% confidence')).toBeInTheDocument();
      expect(screen.getByText('91% confidence')).toBeInTheDocument();
    });

    it('shows descriptions for each recommendation', () => {
      render(<SmartRecommendations {...defaultProps} />);

      expect(screen.getByText(/We detected potential missing character tags/)).toBeInTheDocument();
      expect(screen.getByText(/These canonical tags might improve/)).toBeInTheDocument();
      expect(screen.getByText(/Tags commonly used with your search/)).toBeInTheDocument();
    });
  });

  describe('Confidence Score Styling', () => {
    it('applies correct styling for high confidence scores (80%+)', () => {
      render(<SmartRecommendations {...defaultProps} />);

      const highConfidenceText = screen.getByText('85% confidence');
      const confidenceBadge = highConfidenceText.closest('span');
      expect(confidenceBadge).toHaveClass('bg-green-100', 'text-green-800');
    });

    it('applies correct styling for medium confidence scores (60-79%)', () => {
      render(<SmartRecommendations {...defaultProps} />);

      const mediumConfidenceText = screen.getByText('72% confidence');
      const confidenceBadge = mediumConfidenceText.closest('span');
      expect(confidenceBadge).toHaveClass('bg-yellow-100', 'text-yellow-800');
    });

    it('applies correct styling for low confidence scores (<60%)', () => {
      const lowConfidenceRecs = [
        {
          type: 'low_confidence',
          title: 'Low Confidence Suggestion',
          description: 'This is a low confidence suggestion',
          suggestions: ['Test Tag'],
          confidence_score: 0.45,
          category: 'freeform',
        },
      ];

      render(<SmartRecommendations recommendations={lowConfidenceRecs} onApplyRecommendation={jest.fn()} />);

      const lowConfidenceText = screen.getByText('45% confidence');
      const confidenceBadge = lowConfidenceText.closest('span');
      expect(confidenceBadge).toHaveClass('bg-red-100', 'text-red-800');
    });
  });

  describe('Suggestion Interaction', () => {
    it('renders suggestion pills for each recommendation', () => {
      render(<SmartRecommendations {...defaultProps} />);

      // Check that suggestions are rendered as clickable elements
      expect(screen.getByRole('button', { name: 'Reader' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Angst' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Slow Burn' })).toBeInTheDocument();
    });

    it('calls onApplyRecommendation when a suggestion is clicked', async () => {
      const user = userEvent.setup();
      const onApplyRecommendation = jest.fn();

      render(<SmartRecommendations {...defaultProps} onApplyRecommendation={onApplyRecommendation} />);

      const readerSuggestion = screen.getByRole('button', { name: 'Reader' });
      await user.click(readerSuggestion);

      expect(onApplyRecommendation).toHaveBeenCalledWith(
        mockRecommendations[0],
        'Reader'
      );
    });

    it('shows hover effects on suggestion pills', async () => {
      const user = userEvent.setup();
      render(<SmartRecommendations {...defaultProps} />);

      const suggestion = screen.getByRole('button', { name: 'Reader' });
      
      await user.hover(suggestion);
      
      expect(suggestion).toHaveClass('hover:bg-blue-100');
    });

    it('displays suggestions with appropriate category styling', () => {
      render(<SmartRecommendations {...defaultProps} />);

      // Character suggestions (blue)
      const characterSuggestion = screen.getByRole('button', { name: 'Reader' });
      expect(characterSuggestion).toHaveClass('bg-blue-50', 'text-blue-700');

      // Freeform suggestions (gray)
      const freeformSuggestion = screen.getByRole('button', { name: 'Angst' });
      expect(freeformSuggestion).toHaveClass('bg-gray-50', 'text-gray-700');

      // Relationship suggestions (pink)
      const relationshipSuggestion = screen.getByRole('button', { name: 'Steve Rogers/Tony Stark' });
      expect(relationshipSuggestion).toHaveClass('bg-pink-50', 'text-pink-700');
    });
  });

  describe('Card Styling and Layout', () => {
    it('applies correct styling to recommendation cards', () => {
      render(<SmartRecommendations {...defaultProps} />);

      const cards = screen.getAllByText(/confidence$/).map(el => 
        el.closest('.bg-white.rounded-lg.p-4.shadow-sm.border')
      );
      
      expect(cards.length).toBeGreaterThan(0);
      cards.forEach(card => {
        expect(card).toHaveClass('bg-white', 'rounded-lg', 'p-4', 'shadow-sm', 'border');
      });
    });

    it('displays recommendation types with proper icons', () => {
      render(<SmartRecommendations {...defaultProps} />);

      // Check that titles are displayed properly
      expect(screen.getByText('Missing Character Tags')).toBeInTheDocument();
      expect(screen.getByText('Canonical Tag Suggestions')).toBeInTheDocument();
      expect(screen.getByText('Tag Quality Improvement')).toBeInTheDocument();
    });

    it('arranges cards in a responsive grid layout', () => {
      render(<SmartRecommendations {...defaultProps} />);

      const container = screen.getByText('Smart Recommendations').closest('div');
      const gridContainer = container?.querySelector('.grid');
      
      expect(gridContainer).toHaveClass('grid-cols-1', 'md:grid-cols-2', 'lg:grid-cols-3');
    });
  });

  describe('Empty State', () => {
    it('handles empty recommendations gracefully', () => {
      render(<SmartRecommendations recommendations={[]} onApplyRecommendation={jest.fn()} />);

      expect(screen.queryByText('Smart Recommendations')).not.toBeInTheDocument();
    });

    it('handles recommendations with empty suggestions', () => {
      const emptyRecommendations = [
        {
          type: 'empty_test',
          title: 'Empty Recommendation',
          description: 'This has no suggestions',
          suggestions: [],
          confidence_score: 0.5,
          category: 'freeform',
        },
      ];

      render(<SmartRecommendations recommendations={emptyRecommendations} onApplyRecommendation={jest.fn()} />);

      expect(screen.getByText('Empty Recommendation')).toBeInTheDocument();
      expect(screen.queryByRole('button')).not.toBeInTheDocument();
    });
  });

  describe('Accessibility Features', () => {
    it('has proper heading hierarchy', () => {
      render(<SmartRecommendations {...defaultProps} />);

      const mainHeading = screen.getByRole('heading', { name: /smart recommendations/i });
      expect(mainHeading).toBeInTheDocument();

      const cardHeadings = screen.getAllByRole('heading', { level: 3 });
      expect(cardHeadings.length).toBe(5); // One for each recommendation
    });

    it('provides accessible labels for suggestion buttons', () => {
      render(<SmartRecommendations {...defaultProps} />);

      const readerButton = screen.getByRole('button', { name: 'Reader' });
      expect(readerButton).toHaveAttribute('aria-label', 'Apply suggestion: Reader');
    });

    it('has proper ARIA attributes for recommendation cards', () => {
      render(<SmartRecommendations {...defaultProps} />);

      // Cards should have appropriate roles and labels
      const cards = screen.getAllByText(/confidence$/).map(el => 
        el.closest('[role]')
      );
      
      cards.forEach(card => {
        expect(card).toHaveAttribute('role');
      });
    });

    it('supports keyboard navigation for suggestions', async () => {
      const user = userEvent.setup();
      render(<SmartRecommendations {...defaultProps} />);

      const firstSuggestion = screen.getByRole('button', { name: 'Reader' });
      
      firstSuggestion.focus();
      expect(firstSuggestion).toHaveFocus();

      await user.keyboard('{Tab}');
      const nextSuggestion = screen.getByRole('button', { name: 'You' });
      expect(nextSuggestion).toHaveFocus();
    });
  });

  describe('Performance and Edge Cases', () => {
    it('handles large numbers of suggestions efficiently', () => {
      const largeSuggestionRec = {
        type: 'large_test',
        title: 'Large Suggestion Set',
        description: 'Many suggestions to test performance',
        suggestions: Array.from({ length: 50 }, (_, i) => `Suggestion ${i + 1}`),
        confidence_score: 0.8,
        category: 'freeform',
      };

      render(<SmartRecommendations recommendations={[largeSuggestionRec]} onApplyRecommendation={jest.fn()} />);

      expect(screen.getByText('Large Suggestion Set')).toBeInTheDocument();
      expect(screen.getAllByRole('button')).toHaveLength(50);
    });

    it('handles very long suggestion names gracefully', () => {
      const longNameRec = {
        type: 'long_name_test',
        title: 'Long Name Test',
        description: 'Testing very long suggestion names',
        suggestions: ['This is a very long suggestion name that might cause layout issues if not handled properly'],
        confidence_score: 0.7,
        category: 'freeform',
      };

      render(<SmartRecommendations recommendations={[longNameRec]} onApplyRecommendation={jest.fn()} />);

      const longButton = screen.getByRole('button');
      expect(longButton).toBeInTheDocument();
      expect(longButton).toHaveClass('text-sm'); // Should have appropriate text sizing
    });

    it('handles special characters in suggestions', () => {
      const specialCharRec = {
        type: 'special_char_test',
        title: 'Special Characters',
        description: 'Testing special characters in suggestions',
        suggestions: ['Reader/OC', 'Hurt & Comfort', 'Pre-Canon', 'Post-War'],
        confidence_score: 0.8,
        category: 'freeform',
      };

      render(<SmartRecommendations recommendations={[specialCharRec]} onApplyRecommendation={jest.fn()} />);

      expect(screen.getByRole('button', { name: 'Reader/OC' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Hurt & Comfort' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Pre-Canon' })).toBeInTheDocument();
    });
  });

  describe('Integration Features', () => {
    it('preserves recommendation data when suggestion is applied', async () => {
      const user = userEvent.setup();
      const onApplyRecommendation = jest.fn();

      render(<SmartRecommendations {...defaultProps} onApplyRecommendation={onApplyRecommendation} />);

      const suggestion = screen.getByRole('button', { name: 'Reader' });
      await user.click(suggestion);

      expect(onApplyRecommendation).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'missing_character',
          confidence_score: 0.85,
          category: 'character',
        }),
        'Reader'
      );
    });

    it('handles rapid clicking without duplicate calls', async () => {
      const user = userEvent.setup();
      const onApplyRecommendation = jest.fn();

      render(<SmartRecommendations {...defaultProps} onApplyRecommendation={onApplyRecommendation} />);

      const suggestion = screen.getByRole('button', { name: 'Reader' });
      
      // Rapid clicks
      await user.click(suggestion);
      await user.click(suggestion);
      await user.click(suggestion);

      // Should still only be called once per click
      expect(onApplyRecommendation).toHaveBeenCalledTimes(3);
    });
  });

  describe('Visual Feedback', () => {
    it('provides visual feedback on hover', async () => {
      const user = userEvent.setup();
      render(<SmartRecommendations {...defaultProps} />);

      const suggestion = screen.getByRole('button', { name: 'Reader' });
      
      // Initial state
      expect(suggestion).toHaveClass('bg-blue-50');
      
      // Hover state
      await user.hover(suggestion);
      expect(suggestion).toHaveClass('hover:bg-blue-100');
    });

    it('maintains consistent styling across different categories', () => {
      render(<SmartRecommendations {...defaultProps} />);

      // Check that different categories have distinct but consistent styling
      const characterSuggestion = screen.getByRole('button', { name: 'Reader' });
      const freeformSuggestion = screen.getByRole('button', { name: 'Angst' });
      const relationshipSuggestion = screen.getByRole('button', { name: 'Steve Rogers/Tony Stark' });

      // All should have consistent base classes
      [characterSuggestion, freeformSuggestion, relationshipSuggestion].forEach(suggestion => {
        expect(suggestion).toHaveClass('inline-block', 'px-3', 'py-1', 'rounded-full', 'text-sm');
      });

      // But different color schemes
      expect(characterSuggestion).toHaveClass('bg-blue-50', 'text-blue-700');
      expect(freeformSuggestion).toHaveClass('bg-gray-50', 'text-gray-700');
      expect(relationshipSuggestion).toHaveClass('bg-pink-50', 'text-pink-700');
    });
  });
});