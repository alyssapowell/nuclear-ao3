import { 
  searchWorks, 
  searchUsers, 
  searchCollections, 
  searchSeries,
  getSearchSuggestions,
  getPopularSearches,
  getTrendingSearches,
  getSmartFilters 
} from '../api'

// Mock fetch
global.fetch = jest.fn()
const mockFetch = fetch as jest.MockedFunction<typeof fetch>

describe('Search API Functions', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  describe('searchWorks', () => {
    it('constructs correct URL with query parameters', async () => {
      const mockResponse = {
        works: [],
        results: [],
        total: 0,
        facets: {},
        pagination: { page: 1, limit: 20, pages: 0, total: 0 }
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockResponse,
      } as Response)

      await searchWorks('harry potter', {
        limit: 10,
        page: 2,
        rating: ['Teen And Up Audiences'],
        fandoms: ['Harry Potter'],
        sort: 'updated_at'
      })

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('q=harry+potter'),
        expect.objectContaining({
          method: 'GET',
          headers: {
            'Accept': 'application/json',
            'Content-Type': 'application/json',
          }
        })
      )

      const url = mockFetch.mock.calls[0][0] as string
      expect(url).toContain('limit=10')
      expect(url).toContain('page=2')
      expect(url).toContain('rating=Teen+And+Up+Audiences')
      expect(url).toContain('fandoms=Harry+Potter')
      expect(url).toContain('sort_by=updated_at')
    })

    it('handles search service success response', async () => {
      const mockResponse = {
        results: [
          { id: '1', title: 'Test Work', author: 'Test Author' }
        ],
        total: 1,
        facets: {
          fandoms: { buckets: [{ key: 'Harry Potter', doc_count: 100 }] }
        },
        page: 1,
        limit: 20,
        pages: 1
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockResponse,
      } as Response)

      const result = await searchWorks('test')

      expect(result).toEqual({
        works: mockResponse.results,
        results: mockResponse.results,
        total: 1,
        facets: mockResponse.facets,
        pagination: {
          page: 1,
          limit: 20,
          pages: 1,
          total: 1
        }
      })
    })

    it('falls back to Works API when search service returns no results', async () => {
      const searchServiceResponse = {
        results: [],
        total: 0,
        facets: {},
        page: 1,
        limit: 20,
        pages: 0
      }

      const worksApiResponse = {
        works: [
          { id: '1', title: 'Fallback Work', author: 'Fallback Author' }
        ],
        pagination: { page: 1, limit: 20, pages: 1, total: 1 }
      }

      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: async () => searchServiceResponse,
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => worksApiResponse,
        } as Response)

      const result = await searchWorks('test query')

      expect(mockFetch).toHaveBeenCalledTimes(2)
      expect(result.works).toEqual(worksApiResponse.works)
      expect(result.total).toBe(1)
    })

    it('falls back to Works API when search service fails', async () => {
      const worksApiResponse = {
        works: [{ id: '1', title: 'Fallback Work' }],
        pagination: { page: 1, limit: 20, pages: 1, total: 1 }
      }

      mockFetch
        .mockResolvedValueOnce({
          ok: false,
          status: 500,
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => worksApiResponse,
        } as Response)

      const result = await searchWorks('test')

      expect(result.works).toEqual(worksApiResponse.works)
    })

    it('throws error when both search service and fallback fail', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: false,
          status: 500,
        } as Response)
        .mockResolvedValueOnce({
          ok: false,
          status: 503,
        } as Response)

      await expect(searchWorks('test')).rejects.toThrow(
        'Search service and fallback both failed: 500'
      )
    })
  })

  describe('searchUsers', () => {
    it('calls correct endpoint with parameters', async () => {
      const mockResponse = {
        results: [{ id: '1', username: 'testuser' }],
        total: 1
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockResponse,
      } as Response)

      await searchUsers('author', { limit: 5, page: 1 })

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/search/users'),
        expect.objectContaining({ method: 'GET' })
      )

      const url = mockFetch.mock.calls[0][0] as string
      expect(url).toContain('q=author')
      expect(url).toContain('limit=5')
    })

    it('handles empty query parameter', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ results: [], total: 0 }),
      } as Response)

      await searchUsers('', { limit: 10 })

      const url = mockFetch.mock.calls[0][0] as string
      expect(url).not.toContain('q=')
    })
  })

  describe('searchCollections', () => {
    it('makes correct API call', async () => {
      const mockResponse = {
        results: [{ id: '1', title: 'Test Collection' }],
        total: 1
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockResponse,
      } as Response)

      const result = await searchCollections('collection', { limit: 15 })

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/search/collections'),
        expect.objectContaining({ method: 'GET' })
      )
      expect(result).toEqual(mockResponse)
    })
  })

  describe('searchSeries', () => {
    it('makes correct API call', async () => {
      const mockResponse = {
        results: [{ id: '1', title: 'Test Series' }],
        total: 1
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockResponse,
      } as Response)

      const result = await searchSeries('series', { limit: 8 })

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/search/series'),
        expect.objectContaining({ method: 'GET' })
      )
      expect(result).toEqual(mockResponse)
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

  describe('getPopularSearches', () => {
    it('fetches popular searches successfully', async () => {
      const mockSearches = ['harry potter', 'drarry', 'wolfstar']

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockSearches,
      } as Response)

      const result = await getPopularSearches()

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/search/popular'),
        expect.objectContaining({ method: 'GET' })
      )
      expect(result).toEqual(mockSearches)
    })

    it('returns empty array on failure', async () => {
      mockFetch.mockRejectedValue(new Error('Network error'))

      const result = await getPopularSearches()

      expect(result).toEqual([])
    })
  })

  describe('getTrendingSearches', () => {
    it('fetches trending searches successfully', async () => {
      const mockSearches = ['new fic', 'trending story']

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockSearches,
      } as Response)

      const result = await getTrendingSearches()

      expect(result).toEqual(mockSearches)
    })
  })

  describe('getSmartFilters', () => {
    it('sends correct POST request with query and filters', async () => {
      const mockFilters = {
        suggested_tags: [
          { name: 'angst', relevance: 95 },
          { name: 'hurt/comfort', relevance: 87 }
        ]
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockFilters,
      } as Response)

      const currentFilters = { rating: ['Mature'], fandoms: ['Harry Potter'] }
      const result = await getSmartFilters('emotional story', currentFilters)

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/search/facets/smart'),
        expect.objectContaining({
          method: 'POST',
          headers: {
            'Accept': 'application/json',
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            query: 'emotional story',
            current_filters: currentFilters,
            limit: 20
          })
        })
      )
      expect(result).toEqual(mockFilters)
    })

    it('returns empty object on failure', async () => {
      mockFetch.mockRejectedValue(new Error('API error'))

      const result = await getSmartFilters('test', {})

      expect(result).toEqual({})
    })
  })

  describe('Error Handling', () => {
    it('handles network errors gracefully', async () => {
      mockFetch.mockRejectedValue(new Error('Network error'))

      await expect(searchUsers('test')).rejects.toThrow('Network error')
      await expect(searchCollections('test')).rejects.toThrow('Network error')
      await expect(searchSeries('test')).rejects.toThrow('Network error')
    })

    it('handles non-200 responses', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
      } as Response)

      await expect(searchUsers('test')).rejects.toThrow('User search failed: 404')
      await expect(searchCollections('test')).rejects.toThrow('Collection search failed: 404')
      await expect(searchSeries('test')).rejects.toThrow('Series search failed: 404')
    })
  })

  describe('URL Encoding', () => {
    it('properly encodes special characters in queries', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ results: [], total: 0 }),
      } as Response)

      await searchWorks('test & special/characters?')

      const url = mockFetch.mock.calls[0][0] as string
      expect(url).toContain('test+%26+special%2Fcharacters%3F')
    })

    it('handles arrays in filter parameters', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ results: [], total: 0, facets: {} }),
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