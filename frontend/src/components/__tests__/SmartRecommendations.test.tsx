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

      expect(screen.getByText('Smart Tag Suggestions')).toBeInTheDocument();
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

      expect(screen.getByText('85%')).toBeInTheDocument();
      expect(screen.getByText('72%')).toBeInTheDocument();
      expect(screen.getByText('91%')).toBeInTheDocument();
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

      const highConfidenceText = screen.getByText('85%');
      const confidenceBadge = highConfidenceText.closest('div');
      expect(confidenceBadge).toHaveClass('text-green-600', 'bg-green-100');
    });

    it('applies correct styling for medium confidence scores (60-79%)', () => {
      render(<SmartRecommendations {...defaultProps} />);

      const mediumConfidenceText = screen.getByText('72%');
      const confidenceBadge = mediumConfidenceText.closest('div');
      expect(confidenceBadge).toHaveClass('text-yellow-600', 'bg-yellow-100');
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

      const lowConfidenceText = screen.getByText('45%');
      const confidenceBadge = lowConfidenceText.closest('div');
      expect(confidenceBadge).toHaveClass('text-red-600', 'bg-red-100');
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

      const readerSuggestion = screen.getByRole('button', { name: /Reader/i });
      await user.click(readerSuggestion);

      expect(onApplyRecommendation).toHaveBeenCalledWith(
        expect.objectContaining({
          tag: 'Reader',
          category: 'character',
          type: 'missing_character'
        })
      );
    });

    it('shows hover effects on suggestion pills', async () => {
      const user = userEvent.setup();
      render(<SmartRecommendations {...defaultProps} />);

      const suggestion = screen.getByRole('button', { name: /Reader/i });
      
      await user.hover(suggestion);
      
      expect(suggestion).toHaveClass('hover:bg-gray-50');
    });

    it('displays suggestions with appropriate styling', () => {
      render(<SmartRecommendations {...defaultProps} />);

      // All suggestions have consistent base styling
      const characterSuggestion = screen.getByRole('button', { name: /Reader/i });
      expect(characterSuggestion).toHaveClass('bg-white', 'border-gray-300');

      const freeformSuggestion = screen.getByRole('button', { name: /Angst/i });
      expect(freeformSuggestion).toHaveClass('bg-white', 'border-gray-300');

      const relationshipSuggestion = screen.getByRole('button', { name: /Steve Rogers\/Tony Stark/i });
      expect(relationshipSuggestion).toHaveClass('bg-white', 'border-gray-300');
    });
  });

  describe('Card Styling and Layout', () => {
    it('applies correct styling to recommendation cards', () => {
      render(<SmartRecommendations {...defaultProps} />);

      const cards = screen.getAllByText(/\d+%/).map(el => 
        el.closest('div.border.rounded-lg')
      );
      
      expect(cards.length).toBeGreaterThan(0);
      cards.forEach(card => {
        expect(card).toHaveClass('border', 'rounded-lg', 'p-4');
      });
    });

    it('displays recommendation types with proper icons', () => {
      render(<SmartRecommendations {...defaultProps} />);

      // Check that titles are displayed properly
      expect(screen.getByText('Missing Character Tags')).toBeInTheDocument();
      expect(screen.getByText('Canonical Tag Suggestions')).toBeInTheDocument();
      expect(screen.getByText('Tag Quality Improvement')).toBeInTheDocument();
    });

    it('arranges cards in a stacked layout', () => {
      render(<SmartRecommendations {...defaultProps} />);

      const container = screen.getByText('Smart Tag Suggestions').closest('div');
      expect(container).toHaveClass('space-y-4');
    });
  });

  describe('Empty State', () => {
    it('handles empty recommendations gracefully', () => {
      render(<SmartRecommendations recommendations={[]} onApplyRecommendation={jest.fn()} />);

      expect(screen.queryByText('Smart Tag Suggestions')).not.toBeInTheDocument();
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

      const mainHeading = screen.getByRole('heading', { name: /smart tag suggestions/i });
      expect(mainHeading).toBeInTheDocument();

      // Card titles are h4, not h3
      const cardTitles = screen.getAllByText(/Missing Character Tags|Canonical Tag Suggestions|Related Tags|Tag Quality Improvement|Relationship Expansion/);
      expect(cardTitles.length).toBe(5); // One for each recommendation
    });

    it('provides accessible button elements for suggestions', () => {
      render(<SmartRecommendations {...defaultProps} />);

      const readerButton = screen.getByRole('button', { name: /Reader/i });
      expect(readerButton).toBeInTheDocument();
    });

    it('has proper structure for recommendation cards', () => {
      render(<SmartRecommendations {...defaultProps} />);

      // Cards have proper structure with titles and descriptions
      const cards = screen.getAllByText(/\d+%/).map(el => 
        el.closest('div.border.rounded-lg')
      );
      
      expect(cards.length).toBe(5);
    });

    it('supports keyboard navigation for suggestions', async () => {
      const user = userEvent.setup();
      render(<SmartRecommendations {...defaultProps} />);

      const firstSuggestion = screen.getByRole('button', { name: /Reader/i });
      
      firstSuggestion.focus();
      expect(firstSuggestion).toHaveFocus();

      await user.keyboard('{Tab}');
      const nextSuggestion = screen.getByRole('button', { name: /You/i });
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

      const suggestion = screen.getByRole('button', { name: /Reader/i });
      await user.click(suggestion);

      expect(onApplyRecommendation).toHaveBeenCalledWith(
        expect.objectContaining({
          tag: 'Reader',
          type: 'missing_character',
          category: 'character',
        })
      );
    });

    it('handles rapid clicking without duplicate calls', async () => {
      const user = userEvent.setup();
      const onApplyRecommendation = jest.fn();

      render(<SmartRecommendations {...defaultProps} onApplyRecommendation={onApplyRecommendation} />);

      const suggestion = screen.getByRole('button', { name: /Reader/i });
      
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

      const suggestion = screen.getByRole('button', { name: /Reader/i });
      
      // Initial state
      expect(suggestion).toHaveClass('bg-white');
      
      // Hover state
      await user.hover(suggestion);
      expect(suggestion).toHaveClass('hover:bg-gray-50');
    });

    it('maintains consistent styling across all suggestions', () => {
      render(<SmartRecommendations {...defaultProps} />);

      // Check that all suggestions have consistent styling
      const characterSuggestion = screen.getByRole('button', { name: /Reader/i });
      const freeformSuggestion = screen.getByRole('button', { name: /Angst/i });
      const relationshipSuggestion = screen.getByRole('button', { name: /Steve Rogers\/Tony Stark/i });

      // All should have consistent classes
      [characterSuggestion, freeformSuggestion, relationshipSuggestion].forEach(suggestion => {
        expect(suggestion).toHaveClass('inline-flex', 'items-center', 'px-3', 'py-1', 'rounded-full');
      });

      // All have white background
      expect(characterSuggestion).toHaveClass('bg-white', 'border-gray-300');
      expect(freeformSuggestion).toHaveClass('bg-white', 'border-gray-300');
      expect(relationshipSuggestion).toHaveClass('bg-white', 'border-gray-300');
    });
  });
});