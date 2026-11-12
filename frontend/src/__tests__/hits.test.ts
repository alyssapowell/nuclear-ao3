import { describe, it, expect, vi, afterEach } from 'vitest';
import { recordHit } from '@/lib/api';

describe('recordHit', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('posts one small request that outlives the page', () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);
    recordHit('0b8f7b3e-1111-4222-8333-444455556666', 'tok');
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/v1/works/0b8f7b3e-1111-4222-8333-444455556666/hit');
    expect(init).toMatchObject({ method: 'POST', keepalive: true, headers: { Authorization: 'Bearer tok' } });
  });

  it('never throws when the request fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')));
    expect(() => recordHit('w')).not.toThrow();
    await Promise.resolve();
  });
});
