import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import TagProminenceSelector from '../TagProminenceSelector';

describe('TagProminenceSelector', () => {
  const mockOnTagsChange = jest.fn();
  
  const sampleTags = [
    {
      tagName: 'Harry Potter/Draco Malfoy',
      tagType: 'relationship',
      prominence: 'primary' as const,
    },
    {
      tagName: 'Background Ron Weasley/Hermione Granger',
      tagType: 'relationship', 
      prominence: 'micro' as const,
      autoSuggested: true,
    },
    {
      tagName: 'Angst',
      tagType: 'additional_tags',
      prominence: 'secondary' as const,
    },
  ];

  beforeEach(() => {
    mockOnTagsChange.mockClear();
  });

  it('renders primary relationships section', () => {
    render(
      <TagProminenceSelector 
        tags={sampleTags} 
        onTagsChange={mockOnTagsChange} 
      />
    );
    
    expect(screen.getByText('Primary Relationships')).toBeInTheDocument();
    expect(screen.getByText('Harry Potter/Draco Malfoy')).toBeInTheDocument();
    expect(screen.getByText('1/3 max')).toBeInTheDocument();
  });

  it('shows AI suggested tags with indicator', () => {
    render(
      <TagProminenceSelector 
        tags={sampleTags} 
        onTagsChange={mockOnTagsChange} 
      />
    );
    
    // Need to toggle advanced view to see micro tags
    fireEvent.click(screen.getByText('Show all tags & prominence levels'));
    
    expect(screen.getByText('AI suggested')).toBeInTheDocument();
  });

  it('allows moving tags between prominence levels', () => {
    render(
      <TagProminenceSelector 
        tags={sampleTags} 
        onTagsChange={mockOnTagsChange} 
      />
    );
    
    // Move primary to secondary
    fireEvent.click(screen.getByTitle('Move to secondary'));
    
    expect(mockOnTagsChange).toHaveBeenCalledWith([
      {
        tagName: 'Harry Potter/Draco Malfoy',
        tagType: 'relationship',
        prominence: 'secondary',
        autoSuggested: false,
      },
      sampleTags[1],
      sampleTags[2],
    ]);
  });

  it('allows removing tags', () => {
    render(
      <TagProminenceSelector 
        tags={sampleTags} 
        onTagsChange={mockOnTagsChange} 
      />
    );
    
    fireEvent.click(screen.getByTitle('Remove tag'));
    
    expect(mockOnTagsChange).toHaveBeenCalledWith([
      sampleTags[1],
      sampleTags[2],
    ]);
  });

  it('shows Gen fic message when no primary relationships', () => {
    const tagsWithoutPrimary = sampleTags.filter(tag => tag.prominence !== 'primary');
    
    render(
      <TagProminenceSelector 
        tags={tagsWithoutPrimary} 
        onTagsChange={mockOnTagsChange} 
      />
    );
    
    expect(screen.getByText(/No primary relationships selected/)).toBeInTheDocument();
    expect(screen.getByText(/Gen fic/)).toBeInTheDocument();
  });

  it('shows tag spam warning for excessive tags', () => {
    const manyTags = Array.from({ length: 25 }, (_, i) => ({
      tagName: `Tag ${i}`,
      tagType: 'additional_tags',
      prominence: 'secondary' as const,
    }));
    
    render(
      <TagProminenceSelector 
        tags={manyTags} 
        onTagsChange={mockOnTagsChange} 
      />
    );
    
    expect(screen.getByText(/Tag Spam Warning/)).toBeInTheDocument();
    expect(screen.getByText(/You have 25 tags/)).toBeInTheDocument();
  });

  it('prevents moving to primary when limit reached', () => {
    const tagsAtLimit = [
      ...Array.from({ length: 3 }, (_, i) => ({
        tagName: `Primary Ship ${i}`,
        tagType: 'relationship',
        prominence: 'primary' as const,
      })),
      {
        tagName: 'Secondary Ship',
        tagType: 'relationship',
        prominence: 'secondary' as const,
      },
    ];
    
    render(
      <TagProminenceSelector 
        tags={tagsAtLimit} 
        onTagsChange={mockOnTagsChange} 
      />
    );
    
    // Show advanced view
    fireEvent.click(screen.getByText('Show all tags & prominence levels'));
    
    // Try to move secondary to primary
    const moveButton = screen.getByTitle('Move to primary');
    expect(moveButton).toBeDisabled();
  });

  it('toggles advanced view', () => {
    render(
      <TagProminenceSelector 
        tags={sampleTags} 
        onTagsChange={mockOnTagsChange} 
      />
    );
    
    // Advanced view should be hidden initially
    expect(screen.queryByText('Secondary Tags')).not.toBeInTheDocument();
    
    // Toggle advanced view
    fireEvent.click(screen.getByText('Show all tags & prominence levels'));
    
    // Advanced view should now be visible
    expect(screen.getByText('Secondary Tags')).toBeInTheDocument();
    expect(screen.getByText('Background/Micro Tags')).toBeInTheDocument();
  });

  it('shows tagging guidelines', () => {
    render(
      <TagProminenceSelector 
        tags={sampleTags} 
        onTagsChange={mockOnTagsChange} 
      />
    );
    
    expect(screen.getByText('Tagging Guidelines')).toBeInTheDocument();
    expect(screen.getByText(/Primary:/)).toBeInTheDocument();
    expect(screen.getByText(/Secondary:/)).toBeInTheDocument();
    expect(screen.getByText(/Micro:/)).toBeInTheDocument();
  });

  it('handles empty tags array', () => {
    render(
      <TagProminenceSelector 
        tags={[]} 
        onTagsChange={mockOnTagsChange} 
      />
    );
    
    expect(screen.getByText(/No primary relationships selected/)).toBeInTheDocument();
    expect(screen.getByText('0/3 max')).toBeInTheDocument();
  });

  it('marks auto-suggested tags as non-auto when manually changed', () => {
    render(
      <TagProminenceSelector 
        tags={sampleTags} 
        onTagsChange={mockOnTagsChange} 
      />
    );
    
    // Show advanced view to access micro tags
    fireEvent.click(screen.getByText('Show all tags & prominence levels'));
    
    // Find the micro tag button (should have "↑ Secondary" text)
    const microMoveButton = screen.getByText('↑ Secondary');
    
    // Move the auto-suggested tag from micro to secondary
    fireEvent.click(microMoveButton);
    
    expect(mockOnTagsChange).toHaveBeenCalledTimes(1);
    const callArgs = mockOnTagsChange.mock.calls[0][0];
    const modifiedTag = callArgs.find((tag: { tagName: string }) => tag.tagName === 'Background Ron Weasley/Hermione Granger');
    
    expect(modifiedTag).toBeDefined();
    expect(modifiedTag.prominence).toBe('secondary');
    expect(modifiedTag.autoSuggested).toBe(false);
  });
});