import type { MockedFunction } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import SearchPage from '@/pages/Search';
import { searchWorks, searchTags } from '@/lib/api';

// The search page, form and results working together. Only the REST calls in
// lib/api are replaced, so everything from typing to the rendered cards is real.
vi.mock('@/lib/api', () => ({ searchWorks: vi.fn(), searchTags: vi.fn() }));
const mockedSearchWorks = searchWorks as MockedFunction<typeof searchWorks>;
const mockedSearchTags = searchTags as MockedFunction<typeof searchTags>;

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () =>
  (await import('@/test/router-mock')).routerMock({ useNavigate: () => mockNavigate }))

// The API shape: people are pseuds {id, name}; tags are plain names.
const apiResponse = {
  total: 3,
  works: [
    {
      id: '1',
      title: 'The Adventures of Agatha and Reader',
      authors: [{ id: 'pseud-1', name: 'MagicWriter' }],
      anonymous: false,
      summary: 'A magical adventure with Agatha Harkness and Reader.',
      word_count: 15000,
      chapter_count: 5,
      max_chapters: 10,
      rating: 'Teen And Up Audiences',
      language: 'English',
      published_at: '2023-01-15T10:30:00Z',
      updated_at: '2023-02-20T14:45:00Z',
      relationships: ['Agatha Harkness/Reader'],
      characters: ['Agatha Harkness'],
      freeform_tags: ['Magic', 'Adventure'],
      fandoms: ['Marvel Cinematic Universe'],
      kudos_count: 25,
      bookmark_count: 8,
      hit_count: 150,
      comment_count: 12,
    },
    {
      id: '2',
      title: 'Reader Insert Adventures',
      authors: [{ id: 'pseud-2', name: 'SecretPseud' }],
      anonymous: true,
      summary: 'Various reader insert stories.',
      word_count: 8500,
      chapter_count: 3,
      rating: 'General Audiences',
      language: 'English',
      relationships: [],
      characters: ['Reader'],
      freeform_tags: ['Fluff'],
      fandoms: ['Multifandom'],
    },
    {
      id: '3',
      title: 'Magical Encounters',
      authors: [{ id: 'pseud-3', name: 'Second' }, { id: 'pseud-4', name: 'Third' }],
      anonymous: false,
      word_count: 22000,
      chapter_count: 8,
      max_chapters: 12,
      rating: 'Mature',
      language: 'English',
      relationships: [],
      characters: [],
      freeform_tags: [],
      fandoms: ['Marvel Cinematic Universe'],
    },
  ],
};

const search = async (user: ReturnType<typeof userEvent.setup>, title: string) => {
  await user.type(screen.getByPlaceholderText(/search by title/i), title);
  await user.click(screen.getByRole('button', { name: /search works/i }));
};

describe('Search Integration Tests', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedSearchWorks.mockResolvedValue(apiResponse as any);
    mockedSearchTags.mockResolvedValue([]);
  });

  describe('Complete Search Flow', () => {
    it('shows the search tips before the first search', () => {
      render(<SearchPage />);

      // The page and the form each have an "Enhanced Search" heading.
      expect(screen.getAllByRole('heading', { name: 'Enhanced Search' })).toHaveLength(2);
      expect(screen.getByText('Search Tips')).toBeInTheDocument();
      expect(screen.queryByText('Search Results')).not.toBeInTheDocument();
    });

    it('searches and shows the works, then hides the tips', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      await search(user, 'Agatha');

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Search Results' })).toBeInTheDocument();
      });
      expect(mockedSearchWorks).toHaveBeenCalledTimes(1);
      expect(mockedSearchWorks).toHaveBeenCalledWith('Agatha', expect.objectContaining({ sort: 'relevance' }));

      expect(screen.getByRole('button', { name: /the adventures of agatha and reader/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /reader insert adventures/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /magical encounters/i })).toBeInTheDocument();
      expect(screen.getByText('Found 3 works')).toBeInTheDocument();
      expect(screen.queryByText('Search Tips')).not.toBeInTheDocument();
    });

    it('shows each work with its stats and tags', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      await search(user, 'Agatha');
      await screen.findByRole('heading', { name: 'Search Results' });

      expect(screen.getByText('15,000')).toBeInTheDocument();
      expect(screen.getByText('Teen And Up Audiences', { selector: 'span.border' })).toBeInTheDocument();
      expect(screen.getByText('Agatha Harkness/Reader')).toBeInTheDocument();
      expect(screen.getByText('Magic')).toBeInTheDocument();
      // Two of the three works are in this fandom.
      expect(screen.getAllByText('Marvel Cinematic Universe')).toHaveLength(2);
      expect(screen.getByText('MagicWriter')).toBeInTheDocument();
      expect(screen.getByText('Second, Third')).toBeInTheDocument();
    });

    it('names no author on an anonymous work', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      await search(user, 'Reader');
      await screen.findByRole('heading', { name: 'Search Results' });

      expect(screen.getByText('Anonymous')).toBeInTheDocument();
      expect(screen.queryByText('SecretPseud')).not.toBeInTheDocument();
    });

    it('does not turn authors or tags into links', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      await search(user, 'Agatha');
      await screen.findByRole('heading', { name: 'Search Results' });

      // Nothing in the results links to a pseud or user page.
      expect(screen.queryAllByRole('link')).toHaveLength(0);
    });

    it('opens a work when its title is clicked', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      await search(user, 'Agatha');
      await user.click(await screen.findByRole('button', { name: /magical encounters/i }));

      expect(mockNavigate).toHaveBeenCalledWith('/works/3');
    });

    it('passes advanced filters to the search', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      await user.click(screen.getByRole('button', { name: /show advanced search/i }));
      await user.selectOptions(screen.getByLabelText('Rating'), 'Teen And Up Audiences');
      await user.click(screen.getByLabelText(/hide incomplete works/i));
      await search(user, 'magic');

      await waitFor(() => expect(mockedSearchWorks).toHaveBeenCalled());
      expect(mockedSearchWorks).toHaveBeenCalledWith(
        'magic',
        expect.objectContaining({ rating: ['Teen And Up Audiences'], status: 'complete' })
      );
    });

    it('adds a tag picked from the autocomplete and searches with it', async () => {
      const user = userEvent.setup();
      mockedSearchTags.mockResolvedValue([
        { id: '1', name: 'Agatha Harkness', type: 'character', use_count: 150 },
        { id: '2', name: 'Agatha Christie', type: 'character', use_count: 25 },
      ] as any);
      render(<SearchPage />);

      const characterInput = screen.getByPlaceholderText(/add character tag/i);
      await user.type(characterInput, 'Agatha');

      await user.click(await screen.findByText('Agatha Harkness'));

      expect(screen.getByRole('button', { name: /remove agatha harkness from characters/i })).toBeInTheDocument();
      expect(characterInput).toHaveValue('');

      await user.click(screen.getByRole('button', { name: /search works/i }));
      await waitFor(() => expect(mockedSearchWorks).toHaveBeenCalled());
      expect(mockedSearchWorks).toHaveBeenCalledWith(
        '',
        expect.objectContaining({ characters: ['Agatha Harkness'] })
      );
    });

    it('asks for a search term instead of searching with nothing', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      await user.click(screen.getByRole('button', { name: /search works/i }));

      expect(screen.getByRole('alert')).toHaveTextContent('Please enter at least one search criterion');
      expect(mockedSearchWorks).not.toHaveBeenCalled();
    });

    it('shows loading state while the search runs', async () => {
      const user = userEvent.setup();
      let finish!: (value: any) => void;
      mockedSearchWorks.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
      render(<SearchPage />);

      await search(user, 'test');

      expect(screen.getByText('Searching the archive...')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /searching/i })).toBeDisabled();

      finish(apiResponse);

      await screen.findByRole('heading', { name: 'Search Results' });
      expect(screen.queryByText('Searching the archive...')).not.toBeInTheDocument();
      expect(screen.getByRole('button', { name: /search works/i })).toBeEnabled();
    });

    it('handles search errors gracefully', async () => {
      const user = userEvent.setup();
      mockedSearchWorks.mockRejectedValue(new Error('Network error'));
      render(<SearchPage />);

      await search(user, 'test');

      expect(await screen.findByRole('alert')).toHaveTextContent('Search error: Network error');
      expect(screen.queryByRole('heading', { name: 'Search Results' })).not.toBeInTheDocument();
    });

    it('says so when nothing matches', async () => {
      const user = userEvent.setup();
      mockedSearchWorks.mockResolvedValue({ total: 0, works: [] } as any);
      render(<SearchPage />);

      await search(user, 'nothing');

      expect(await screen.findByText('No works found')).toBeInTheDocument();
    });

    it('can search again with new terms', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      await search(user, 'test1');
      await screen.findByRole('heading', { name: 'Search Results' });

      const title = screen.getByPlaceholderText(/search by title/i);
      await user.clear(title);
      await user.type(title, 'test2');
      await user.click(screen.getByRole('button', { name: /search works/i }));

      await waitFor(() => expect(mockedSearchWorks).toHaveBeenCalledTimes(2));
      expect(mockedSearchWorks).toHaveBeenLastCalledWith('test2', expect.anything());
    });
  });

  describe('Accessibility', () => {
    it('labels the search form and announces results', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      expect(screen.getAllByRole('search').length).toBeGreaterThan(0);
      expect(screen.getByLabelText('Title')).toBeInTheDocument();

      await search(user, 'test');
      await screen.findByRole('heading', { name: 'Search Results' });

      expect(screen.getByText('Search completed. Found 3 results.')).toBeInTheDocument();
    });
  });
});
