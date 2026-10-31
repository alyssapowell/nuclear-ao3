import type { MockedFunction } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { browseCollections, getMyCollections, createCollection } from '@/lib/api';
import CollectionsPage from '@/pages/CollectionList';
import CreateCollectionPage from '@/pages/CollectionNew';

vi.mock('@/lib/api', () => ({
  browseCollections: vi.fn(),
  getMyCollections: vi.fn(),
  createCollection: vi.fn(),
}));
const mockBrowse = browseCollections as MockedFunction<typeof browseCollections>;
const mockMine = getMyCollections as MockedFunction<typeof getMyCollections>;
const mockCreate = createCollection as MockedFunction<typeof createCollection>;

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () =>
  (await import('@/test/router-mock')).routerMock({ useNavigate: () => mockNavigate }));

// setup.ts gives every test a fake localStorage; this logs the reader in or out.
const logIn = (token: string | null) => {
  vi.mocked(localStorage.getItem).mockImplementation((key: string) => (key === 'auth_token' ? token : null));
};

// The API wraps each collection with the pseuds that organise it.
const collectionsResponse = {
  collections: [
    {
      collection: {
        id: '1',
        name: 'test-collection',
        title: 'Test Collection',
        description: 'A test collection',
        work_count: 5,
        is_open: true,
        is_moderated: false,
        is_anonymous: false,
        is_unrevealed: false,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      },
      organisers: [{ id: 'pseud-1', name: 'TestPseud', role: 'owner' }],
    },
  ],
  pagination: { page: 1, limit: 20, total: 1, total_pages: 1 },
};

const emptyResponse = {
  collections: [],
  pagination: { page: 1, limit: 20, total: 0, total_pages: 0 },
};

describe('Collections Components', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    logIn('mock-token');
    mockBrowse.mockResolvedValue(collectionsResponse);
    mockMine.mockResolvedValue(collectionsResponse);
  });

  describe('Collections Browse Page', () => {
    it('should render the page with its controls and the collections', async () => {
      render(<CollectionsPage />);

      expect(screen.getByRole('heading', { name: 'Collections' })).toBeInTheDocument();
      expect(screen.getByPlaceholderText(/search collections/i)).toBeInTheDocument();
      expect(screen.getByText('Browse All Collections')).toBeInTheDocument();
      expect(screen.getByText('My Collections')).toBeInTheDocument();
      expect(screen.getByRole('link', { name: /create collection/i })).toHaveAttribute('href', '/collections/new');

      expect(await screen.findByText('Test Collection')).toBeInTheDocument();
      expect(mockBrowse).toHaveBeenCalledWith({ page: 1, limit: 20 });
    });

    it('should hide My Collections and Create Collection from logged-out readers', async () => {
      logIn(null);
      render(<CollectionsPage />);

      await screen.findByText('Test Collection');
      expect(screen.queryByText('My Collections')).not.toBeInTheDocument();
      expect(screen.queryByRole('link', { name: /create collection/i })).not.toBeInTheDocument();
    });

    it('should display collection cards correctly', async () => {
      render(<CollectionsPage />);

      const title = await screen.findByRole('link', { name: 'Test Collection' });
      expect(title).toHaveAttribute('href', '/collections/1');
      expect(screen.getByText('A test collection')).toBeInTheDocument();
      expect(screen.getByText(/5 works/)).toBeInTheDocument();
      expect(screen.getByText('Open')).toBeInTheDocument();
    });

    it('should name the organising pseud, as plain text', async () => {
      render(<CollectionsPage />);

      expect(await screen.findByText(/Collection by TestPseud/)).toBeInTheDocument();
      expect(screen.queryByRole('link', { name: /TestPseud/ })).not.toBeInTheDocument();
    });

    it('should search with the typed query', async () => {
      const user = userEvent.setup();
      render(<CollectionsPage />);
      await screen.findByText('Test Collection');

      await user.type(screen.getByPlaceholderText(/search collections/i), 'test search');
      await user.click(screen.getByRole('button', { name: /search collections/i }));

      await waitFor(() => {
        expect(mockBrowse).toHaveBeenLastCalledWith({ q: 'test search', page: 1, limit: 20 });
      });
    });

    it('should switch between tabs', async () => {
      const user = userEvent.setup();
      render(<CollectionsPage />);
      await screen.findByText('Test Collection');
      expect(mockBrowse).toHaveBeenCalledTimes(1);

      await user.click(screen.getByText('My Collections'));
      await waitFor(() => expect(mockMine).toHaveBeenCalledWith({ page: 1, limit: 20 }, 'mock-token'));

      await user.click(screen.getByText('Browse All Collections'));
      await waitFor(() => expect(mockBrowse).toHaveBeenCalledTimes(2));
    });

    it('should handle loading state', async () => {
      mockBrowse.mockReturnValue(new Promise(() => {}));
      render(<CollectionsPage />);

      // Skeleton cards pulse while loading.
      await waitFor(() => {
        expect(document.querySelectorAll('.animate-pulse').length).toBeGreaterThan(0);
      });
    });

    it('should show the error and let the reader try again', async () => {
      const user = userEvent.setup();
      mockBrowse.mockRejectedValueOnce(new Error('Failed to fetch'));
      render(<CollectionsPage />);

      expect(await screen.findByText('Failed to fetch')).toBeInTheDocument();

      await user.click(screen.getByRole('button', { name: /try again/i }));
      expect(await screen.findByText('Test Collection')).toBeInTheDocument();
    });
  });

  describe('Error Handling and Edge Cases', () => {
    it('should say so when there are no collections', async () => {
      mockBrowse.mockResolvedValue(emptyResponse);
      render(<CollectionsPage />);

      expect(await screen.findByText('No collections available')).toBeInTheDocument();
    });

    it('should tell a reader with none of their own to create one', async () => {
      const user = userEvent.setup();
      mockMine.mockResolvedValue(emptyResponse);
      render(<CollectionsPage />);
      await screen.findByText('Test Collection');

      await user.click(screen.getByText('My Collections'));

      expect(await screen.findByText(/haven't created any collections yet/i)).toBeInTheDocument();
    });

    it('should say so when a search matches nothing', async () => {
      const user = userEvent.setup();
      render(<CollectionsPage />);
      await screen.findByText('Test Collection');

      mockBrowse.mockResolvedValue(emptyResponse);
      await user.type(screen.getByPlaceholderText(/search collections/i), 'zzz');
      await user.click(screen.getByRole('button', { name: /search collections/i }));

      expect(await screen.findByText('No collections found')).toBeInTheDocument();
    });

    it('should cope with a collection that has only some fields', async () => {
      mockBrowse.mockResolvedValue({
        collections: [{ collection: { id: '1', name: 'partial-collection', title: '' } }],
        pagination: { page: 1, limit: 20, total: 1, total_pages: 1 },
      });
      render(<CollectionsPage />);

      // It falls back to the collection's name for the byline and does not crash.
      expect(await screen.findByText(/Collection by partial-collection/)).toBeInTheDocument();
      expect(screen.getByRole('heading', { name: 'Collections' })).toBeInTheDocument();
    });
  });

  describe('Collection Creation Page', () => {
    it('should render creation form with all fields', () => {
      render(<CreateCollectionPage />);

      expect(screen.getByRole('heading', { name: /create new collection/i })).toBeInTheDocument();
      expect(screen.getByLabelText(/collection name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/collection title/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/description/i)).toBeInTheDocument();

      expect(screen.getByLabelText(/open submissions/i)).toBeChecked();
      expect(screen.getByLabelText(/moderated submissions/i)).not.toBeChecked();
      expect(screen.getByLabelText(/anonymous collection/i)).not.toBeChecked();
      expect(screen.getByLabelText(/unrevealed collection/i)).not.toBeChecked();

      expect(screen.getByRole('button', { name: /create collection/i })).toBeInTheDocument();
    });

    it('should require a name and title', () => {
      render(<CreateCollectionPage />);

      // The inputs are `required`, so submit the form directly to reach the script's own check.
      fireEvent.submit(document.querySelector('form')!);

      expect(screen.getByText('Collection name and title are required')).toBeInTheDocument();
      expect(mockCreate).not.toHaveBeenCalled();
    });

    it('should need a logged in reader', async () => {
      logIn(null);
      const user = userEvent.setup();
      render(<CreateCollectionPage />);

      await user.type(screen.getByLabelText(/collection name/i), 'my-collection');
      await user.type(screen.getByLabelText(/collection title/i), 'My Collection');
      await user.click(screen.getByRole('button', { name: /create collection/i }));

      expect(await screen.findByText('You must be logged in to create a collection')).toBeInTheDocument();
      expect(mockCreate).not.toHaveBeenCalled();
    });

    it('should create the collection and open it', async () => {
      const user = userEvent.setup();
      mockCreate.mockResolvedValue({ collection: { id: 'new-collection-id' } });
      render(<CreateCollectionPage />);

      await user.type(screen.getByLabelText(/collection name/i), 'test-collection-name');
      await user.type(screen.getByLabelText(/collection title/i), 'Test Collection Title');
      await user.type(screen.getByLabelText(/description/i), 'Test description');
      await user.click(screen.getByLabelText(/anonymous collection/i));
      await user.click(screen.getByRole('button', { name: /create collection/i }));

      await waitFor(() => {
        expect(mockCreate).toHaveBeenCalledWith(
          {
            name: 'test-collection-name',
            title: 'Test Collection Title',
            description: 'Test description',
            is_open: true,
            is_moderated: false,
            is_anonymous: true,
            is_unrevealed: false,
          },
          'mock-token'
        );
      });
      expect(mockNavigate).toHaveBeenCalledWith('/collections/new-collection-id');
    });

    it('should show why creation failed', async () => {
      const user = userEvent.setup();
      mockCreate.mockRejectedValue(new Error('Creation failed'));
      render(<CreateCollectionPage />);

      await user.type(screen.getByLabelText(/collection name/i), 'test-collection');
      await user.type(screen.getByLabelText(/collection title/i), 'Test Collection');
      await user.click(screen.getByRole('button', { name: /create collection/i }));

      expect(await screen.findByText('Creation failed')).toBeInTheDocument();
      expect(mockNavigate).not.toHaveBeenCalled();
    });

    it('should toggle the collection settings', async () => {
      const user = userEvent.setup();
      render(<CreateCollectionPage />);

      const open = screen.getByLabelText(/open submissions/i);
      const moderated = screen.getByLabelText(/moderated submissions/i);
      await user.click(open);
      await user.click(moderated);

      expect(open).not.toBeChecked();
      expect(moderated).toBeChecked();
    });

    it('should explain each collection setting', () => {
      render(<CreateCollectionPage />);

      expect(screen.getByText(/allow anyone to submit works/i)).toBeInTheDocument();
      expect(screen.getByText(/require approval before works appear/i)).toBeInTheDocument();
      expect(screen.getByText(/hide author names for works/i)).toBeInTheDocument();
      expect(screen.getByText(/hide works until you choose to reveal/i)).toBeInTheDocument();
    });
  });
});
