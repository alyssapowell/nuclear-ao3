import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import DownloadMenu from '../DownloadMenu';

vi.mock('@/lib/api', () => ({
  DOWNLOAD_FORMATS: [
    { ext: 'epub', label: 'EPUB' },
    { ext: 'html', label: 'HTML' },
    { ext: 'txt', label: 'Text' },
  ],
  workDownloadUrl: (id: string, ext: string) => `/api/v1/works/${id}/download.${ext}`,
  downloadWork: vi.fn(),
}));

import { downloadWork } from '@/lib/api';

describe('DownloadMenu', () => {
  // jsdom cannot follow a link; the test only needs to know the click happened.
  const stopNavigation = (e: Event) => e.preventDefault();
  beforeEach(() => {
    vi.clearAllMocks();
    document.addEventListener('click', stopNavigation);
  });
  afterEach(() => document.removeEventListener('click', stopNavigation));

  it('is closed until opened, then lists the formats the server offers and no others', () => {
    render(<DownloadMenu workId="w1" />);
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Download' }));
    const items = screen.getAllByRole('menuitem').map((a) => a.textContent);
    expect(items).toEqual(['EPUB', 'HTML', 'Text']);
    expect(screen.queryByText(/PDF|MOBI|AZW3/i)).not.toBeInTheDocument();
  });

  it('links each format to its download address', () => {
    render(<DownloadMenu workId="w1" />);
    fireEvent.click(screen.getByRole('button', { name: 'Download' }));
    expect(screen.getByRole('menuitem', { name: 'EPUB' })).toHaveAttribute('href', '/api/v1/works/w1/download.epub');
    expect(screen.getByRole('menuitem', { name: 'Text' })).toHaveAttribute('href', '/api/v1/works/w1/download.txt');
  });

  it('leaves the link alone when logged out', () => {
    render(<DownloadMenu workId="w1" />);
    fireEvent.click(screen.getByRole('button', { name: 'Download' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'HTML' }));
    expect(downloadWork).not.toHaveBeenCalled();
  });

  it('fetches with the login when logged in, and shows a failure', async () => {
    (downloadWork as ReturnType<typeof vi.fn>).mockRejectedValueOnce(new Error('Download failed (404)'));
    render(<DownloadMenu workId="w1" authToken="tok" />);
    fireEvent.click(screen.getByRole('button', { name: 'Download' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'EPUB' }));
    expect(downloadWork).toHaveBeenCalledWith('w1', 'epub', 'tok');
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('Download failed (404)'));
  });
});
