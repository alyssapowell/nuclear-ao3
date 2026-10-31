/**
 * What a reader who is not logged in can do. Giving kudos and leaving a guest
 * comment are tested with the real components in KudosButton.test.tsx and
 * Comments.test.tsx; this file covers search, which has no login at all.
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import SearchPage from '../pages/Search';
import { searchWorks } from '../lib/api';

const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;

const apiReply = (body: unknown) =>
  Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(body) } as Response);

// setup.ts gives every test a fake localStorage; this logs the reader in or out.
const logIn = (token: string | null) => {
  vi.mocked(localStorage.getItem).mockImplementation((key: string) => (key === 'auth_token' ? token : null));
};

const oneWork = {
  total: 1,
  works: [
    {
      id: 'work-1',
      title: 'Test Work',
      authors: [{ id: 'pseud-1', name: 'Test Pseud' }],
      anonymous: false,
      summary: 'Test summary',
      word_count: 1000,
      chapter_count: 1,
      rating: 'General Audiences',
      language: 'English',
      relationships: [],
      characters: [],
      freeform_tags: [],
      fandoms: [],
    },
  ],
};

describe('Anonymous Features', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    logIn(null);
    fetchMock.mockImplementation(() => apiReply(oneWork));
  });

  describe('Search Page When Logged Out', () => {
    it('should render the search form', () => {
      render(<SearchPage />);

      expect(screen.getByPlaceholderText(/search by title/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /search works/i })).toBeInTheDocument();
    });

    it('should search and show results without logging in', async () => {
      const user = userEvent.setup();
      render(<SearchPage />);

      await user.type(screen.getByPlaceholderText(/search by title/i), 'test query');
      await user.click(screen.getByRole('button', { name: /search works/i }));

      expect(await screen.findByRole('button', { name: /test work/i })).toBeInTheDocument();
      const [url] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(url).toContain('/api/v1/search/works?');
      expect(url).toContain('q=test+query');
    });

    it('should send no Authorization header', async () => {
      await searchWorks('test');

      const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(init.headers).not.toHaveProperty('Authorization');
    });

    it('should send the token once logged in, so restricted works are included', async () => {
      logIn('real-token');

      await searchWorks('test');

      const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(init.headers).toHaveProperty('Authorization', 'Bearer real-token');
    });
  });

  describe('Authentication State Edge Cases', () => {
    it('should still search when the stored token is garbage', async () => {
      logIn('invalid-token-data');
      const user = userEvent.setup();
      render(<SearchPage />);

      await user.type(screen.getByPlaceholderText(/search by title/i), 'test');
      await user.click(screen.getByRole('button', { name: /search works/i }));

      // The server decides whether the token is any good; the page just searches.
      await waitFor(() => expect(fetchMock).toHaveBeenCalled());
      expect(await screen.findByRole('button', { name: /test work/i })).toBeInTheDocument();
    });

    it('should show the error when the server turns the search down', async () => {
      fetchMock.mockImplementation(() =>
        Promise.resolve({ ok: false, status: 401, json: () => Promise.resolve({ error: 'Invalid token' }) } as Response)
      );
      const user = userEvent.setup();
      render(<SearchPage />);

      await user.type(screen.getByPlaceholderText(/search by title/i), 'test');
      await user.click(screen.getByRole('button', { name: /search works/i }));

      expect(await screen.findByRole('alert')).toHaveTextContent('Invalid token');
    });
  });
});
