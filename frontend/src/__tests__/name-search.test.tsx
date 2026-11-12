import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import PeopleSearch from '@/pages/PeopleSearch';
import TagSearch from '@/pages/TagSearch';

afterEach(() => vi.restoreAllMocks());

function mockFetch(body: unknown) {
  const f = vi.fn().mockResolvedValue({ ok: true, json: async () => body });
  vi.stubGlobal('fetch', f);
  return f;
}

describe('people search', () => {
  it('lists pseuds linking to their own pages, with no account information', async () => {
    const f = mockFetch({
      pseuds: [{ id: 'p-1', name: 'Moonwriter', description: 'About the moon', icon_url: '', icon_alt_text: '', work_count: 3 }],
      pagination: { page: 1, limit: 20, total: 1, total_pages: 1 },
    });
    render(<PeopleSearch />);
    fireEvent.change(screen.getByPlaceholderText('Name or description'), { target: { value: 'moon' } });
    fireEvent.click(screen.getByText('Search people'));
    const link = await screen.findByText('Moonwriter');
    expect(link.getAttribute('href')).toBe('/pseuds/p-1');
    expect(screen.getByText('3 works')).toBeTruthy();
    expect(String(f.mock.calls[0][0])).toBe('/api/v1/pseuds/search?q=moon');
  });
});

describe('tag search', () => {
  it('lists tags with type and use count, linking to the tag page', async () => {
    const f = mockFetch({
      tags: [{ id: 't-1', name: 'Fluff', type: 'freeform', is_canonical: true, use_count: 1 }],
      total: 1, page: 1, pages: 1,
    });
    render(<TagSearch />);
    fireEvent.click(screen.getByLabelText('Canonical tags only'));
    fireEvent.click(screen.getByText('Search tags'));
    const link = await screen.findByText('Fluff');
    await waitFor(() => expect(link.getAttribute('href')).toBe('/works?tag=Fluff'));
    expect(screen.getByText('1 work')).toBeTruthy();
    expect(String(f.mock.calls[0][0])).toContain('canonical=true');
  });
});
