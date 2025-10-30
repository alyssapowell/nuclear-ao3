import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import WorkCard from '../WorkCard';

// Mock Next.js Link component
jest.mock('next/link', () => {
  return function MockLink({ children, href }: { children: React.ReactNode; href: string }) {
    return <a href={href}>{children}</a>;
  };
});

describe('WorkCard', () => {
  const mockWork = {
    id: 'work-123',
    title: 'Test Work Title',
    summary: 'This is a test summary for the work that should be displayed.',
    language: 'en',
    rating: 'teen',
    status: 'published',
    word_count: 25000,
    chapter_count: 5,
    max_chapters: 10,
    published_date: '2023-01-15',
    updated_date: '2023-02-20',
    fandoms: [
      { name: 'Harry Potter', category: 'fandom' },
      { name: 'Sherlock Holmes', category: 'fandom' }
    ],
    characters: [
      { name: 'Harry Potter', category: 'character' },
      { name: 'Sherlock Holmes', category: 'character' }
    ],
    relationships: [
      { name: 'Harry Potter/Draco Malfoy', category: 'relationship' },
      { name: 'John Watson/Sherlock Holmes', category: 'relationship' }
    ],
    freeform_tags: [
      { name: 'Fluff', category: 'freeform' },
      { name: 'Hurt/Comfort', category: 'freeform' },
      { name: 'Enemies to Lovers', category: 'freeform' }
    ]
  };

  it('renders work title with correct link', () => {
    render(<WorkCard work={mockWork} />);
    
    const titleLink = screen.getByRole('link', { name: mockWork.title });
    expect(titleLink).toBeInTheDocument();
    expect(titleLink).toHaveAttribute('href', '/works/work-123');
  });

  it('displays work summary', () => {
    render(<WorkCard work={mockWork} />);
    
    expect(screen.getByText(mockWork.summary)).toBeInTheDocument();
  });

  it('displays rating with correct styling', () => {
    render(<WorkCard work={mockWork} />);
    
    const ratingBadge = screen.getByText('teen');
    expect(ratingBadge).toBeInTheDocument();
    expect(ratingBadge).toHaveClass('bg-blue-100', 'text-blue-800');
  });

  it('displays status badge', () => {
    render(<WorkCard work={mockWork} />);
    
    const statusBadge = screen.getByText('published');
    expect(statusBadge).toBeInTheDocument();
  });

  it('displays work statistics correctly', () => {
    render(<WorkCard work={mockWork} />);
    
    expect(screen.getByText(/25\.0K.*words/)).toBeInTheDocument();
    expect(screen.getByText(/5.*\/.*10.*chapters/)).toBeInTheDocument();
  });

  it('displays fandoms with links', () => {
    render(<WorkCard work={mockWork} />);
    
    expect(screen.getAllByText('Harry Potter')).toHaveLength(2); // Appears in both fandoms and characters
    expect(screen.getAllByText('Sherlock Holmes')).toHaveLength(2); // Appears in both fandoms and characters
  });

  it('displays relationships with links', () => {
    render(<WorkCard work={mockWork} />);
    
    expect(screen.getByText('Harry Potter/Draco Malfoy')).toBeInTheDocument();
    expect(screen.getByText('John Watson/Sherlock Holmes')).toBeInTheDocument();
  });

  it('displays freeform tags with links', () => {
    render(<WorkCard work={mockWork} />);
    
    expect(screen.getByText('Fluff')).toBeInTheDocument();
    expect(screen.getByText('Hurt/Comfort')).toBeInTheDocument();
    expect(screen.getByText('Enemies to Lovers')).toBeInTheDocument();
  });

  it('handles works with no tags gracefully', () => {
    const workWithoutTags = {
      ...mockWork,
      fandoms: [],
      characters: [],
      relationships: [],
      freeform_tags: []
    };
    
    render(<WorkCard work={workWithoutTags} />);
    
    expect(screen.getByText(mockWork.title)).toBeInTheDocument();
    expect(screen.getByText(mockWork.summary)).toBeInTheDocument();
  });
});