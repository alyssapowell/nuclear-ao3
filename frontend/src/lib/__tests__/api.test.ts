import type { MockedFunction } from 'vitest';
import { searchWorks, getSearchSuggestions } from '../api'

// Mock fetch
global.fetch = vi.fn()
const mockFetch = fetch as MockedFunction<typeof fetch>

describe('Search API Functions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('searchWorks', () => {
    const empty = { works: [], total: 0, facets: {}, pagination: { page: 1, limit: 20, pages: 0, total: 0 } }

    it('sends the query and filters to the search endpoint', async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => empty } as Response)

      await searchWorks('harry potter', {
        limit: 10,
        page: 2,
        rating: ['Teen And Up Audiences'],
        fandoms: ['Harry Potter'],
        blockedTags: ['Major Character Death'],
        hideCrossovers: true,
        sort: 'updated_at',
      })

      const url = mockFetch.mock.calls[0][0] as string
      expect(url.startsWith('/api/v1/search/works?')).toBe(true)
      expect(url).toContain('q=harry+potter')
      expect(url).toContain('limit=10')
      expect(url).toContain('page=2')
      expect(url).toContain('rating=Teen+And+Up+Audiences')
      expect(url).toContain('fandoms=Harry+Potter')
      expect(url).toContain('exclude_tags=Major+Character+Death')
      expect(url).toContain('crossovers=exclude')
      expect(url).toContain('sort=updated_at')
    })

    it('returns the results as the API sends them', async () => {
      const response = { ...empty, works: [{ id: '1', title: 'Test Work', authors: [{ id: 'p1', name: 'Quill' }] }], total: 1 }
      mockFetch.mockResolvedValue({ ok: true, json: async () => response } as Response)

      expect(await searchWorks('test')).toEqual(response)
    })

    it('shows no results when nothing matches, instead of other works', async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => empty } as Response)

      const result = await searchWorks('no such thing')

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(result.works).toEqual([])
    })

    it("reports the API's error message", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({ error: 'word_count_min must be a whole number' }),
      } as Response)

      await expect(searchWorks('test')).rejects.toThrow('word_count_min must be a whole number')
    })
  })

  describe('getSearchSuggestions', () => {
    it('returns empty results for short queries', async () => {
      const result = await getSearchSuggestions('a')
      
      expect(result).toEqual({ authors: [], tags: [], works: [] })
      expect(mockFetch).not.toHaveBeenCalled()
    })

    it('makes API call for valid queries', async () => {
      const mockResponse = {
        works: [{ title: 'Test Work', id: '1' }],
        tags: [{ name: 'test-tag', count: 50 }],
        authors: [{ name: 'TestUser', work_count: 10 }]
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockResponse,
      } as Response)

      const result = await getSearchSuggestions('test')

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/search/suggestions?q=test'),
        expect.objectContaining({ method: 'GET' })
      )
      expect(result).toEqual(mockResponse)
    })

    it('returns empty results on API failure', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
      } as Response)

      const result = await getSearchSuggestions('test')

      expect(result).toEqual({ authors: [], tags: [], works: [] })
    })
  })

  describe('URL Encoding', () => {
    it('properly encodes special characters in queries', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ works: [], total: 0 }),
      } as Response)

      await searchWorks('test & special/characters?')

      const url = mockFetch.mock.calls[0][0] as string
      expect(url).toContain('test+%26+special%2Fcharacters%3F')
    })

    it('handles arrays in filter parameters', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ works: [], total: 0, facets: {} }),
      } as Response)

      await searchWorks('test', {
        rating: ['General Audiences', 'Teen And Up Audiences'],
        fandoms: ['Harry Potter', 'Marvel']
      })

      const url = mockFetch.mock.calls[0][0] as string
      expect(url).toContain('rating=General+Audiences')
      expect(url).toContain('rating=Teen+And+Up+Audiences')
      expect(url).toContain('fandoms=Harry+Potter')
      expect(url).toContain('fandoms=Marvel')
    })
  })
})