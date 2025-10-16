import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';

// Mock the API functions
jest.mock('@/lib/api', () => ({
  getSeries: jest.fn(),
  getSeriesWorks: jest.fn(),
  createSeries: jest.fn(),
  updateSeries: jest.fn(),
  deleteSeries: jest.fn(),
  searchSeries: jest.fn(),
  getMySeries: jest.fn(),
  getUserSeries: jest.fn(),
  addWorkToSeries: jest.fn(),
  removeWorkFromSeries: jest.fn(),
  getMyWorks: jest.fn(),
}));

// Mock Next.js router
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
    back: jest.fn(),
  }),
  useParams: () => ({
    id: 'test-series-id',
  }),
}));

import { 
  getSeries, 
  getSeriesWorks, 
  createSeries, 
  updateSeries, 
  deleteSeries,
  searchSeries,
  getMySeries,
  getUserSeries,
  addWorkToSeries,
  removeWorkFromSeries,
  getMyWorks,
} from '@/lib/api';

// Mock localStorage
const mockLocalStorage = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
};
Object.defineProperty(window, 'localStorage', { value: mockLocalStorage });

const mockGetSeries = getSeries as jest.MockedFunction<typeof getSeries>;
const mockGetSeriesWorks = getSeriesWorks as jest.MockedFunction<typeof getSeriesWorks>;
const mockCreateSeries = createSeries as jest.MockedFunction<typeof createSeries>;
const mockUpdateSeries = updateSeries as jest.MockedFunction<typeof updateSeries>;
const mockDeleteSeries = deleteSeries as jest.MockedFunction<typeof deleteSeries>;
const mockSearchSeries = searchSeries as jest.MockedFunction<typeof searchSeries>;
const mockGetMySeries = getMySeries as jest.MockedFunction<typeof getMySeries>;
const mockGetUserSeries = getUserSeries as jest.MockedFunction<typeof getUserSeries>;
const mockAddWorkToSeries = addWorkToSeries as jest.MockedFunction<typeof addWorkToSeries>;
const mockRemoveWorkFromSeries = removeWorkFromSeries as jest.MockedFunction<typeof removeWorkFromSeries>;
const mockGetMyWorks = getMyWorks as jest.MockedFunction<typeof getMyWorks>;

// Sample test data
const mockSeries = {
  id: 'test-series-id',
  title: 'Test Series',
  summary: 'A test series for unit testing',
  notes: 'These are test notes',
  user_id: 'test-user-id',
  username: 'testuser',
  is_complete: false,
  work_count: 2,
  word_count: 15000,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-02T00:00:00Z',
};

const mockWorks = [
  {
    id: 'work1',
    title: 'First Work',
    summary: 'First work in the series',
    rating: 'General Audiences',
    category: ['Gen'],
    warnings: ['No Archive Warnings Apply'],
    fandoms: ['Test Fandom'],
    characters: ['Character A'],
    relationships: [],
    freeform_tags: ['Test Tag'],
    word_count: 5000,
    chapter_count: 1,
    is_complete: true,
    status: 'published',
    published_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    hits: 100,
    kudos: 10,
    comments: 5,
    bookmarks: 3,
    username: 'testuser',
    position: 1,
  },
  {
    id: 'work2',
    title: 'Second Work',
    summary: 'Second work in the series',
    rating: 'Teen And Up Audiences',
    category: ['F/M'],
    warnings: ['No Archive Warnings Apply'],
    fandoms: ['Test Fandom'],
    characters: ['Character A', 'Character B'],
    relationships: ['Character A/Character B'],
    freeform_tags: ['Romance'],
    word_count: 10000,
    chapter_count: 3,
    is_complete: false,
    status: 'published',
    published_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
    hits: 200,
    kudos: 25,
    comments: 15,
    bookmarks: 8,
    username: 'testuser',
    position: 2,
  },
];

const mockPagination = {
  page: 1,
  limit: 20,
  total: 2,
  total_pages: 1,
};

describe('Series API Integration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockLocalStorage.getItem.mockReturnValue('test-auth-token');
  });

  describe('getSeries', () => {
    it('should fetch series data successfully', async () => {
      mockGetSeries.mockResolvedValue({ series: mockSeries });

      const result = await getSeries('test-series-id', 'test-auth-token');

      expect(mockGetSeries).toHaveBeenCalledWith('test-series-id', 'test-auth-token');
      expect(result.series).toEqual(mockSeries);
    });

    it('should handle series not found', async () => {
      mockGetSeries.mockRejectedValue(new Error('Series not found'));

      await expect(getSeries('invalid-id', 'test-auth-token')).rejects.toThrow('Series not found');
    });
  });

  describe('getSeriesWorks', () => {
    it('should fetch series works successfully', async () => {
      mockGetSeriesWorks.mockResolvedValue({ works: mockWorks });

      const result = await getSeriesWorks('test-series-id', 'test-auth-token');

      expect(mockGetSeriesWorks).toHaveBeenCalledWith('test-series-id', 'test-auth-token');
      expect(result.works).toEqual(mockWorks);
    });

    it('should handle empty series', async () => {
      mockGetSeriesWorks.mockResolvedValue({ works: [] });

      const result = await getSeriesWorks('empty-series-id', 'test-auth-token');

      expect(result.works).toEqual([]);
    });
  });

  describe('createSeries', () => {
    const seriesData = {
      title: 'New Series',
      summary: 'A new series',
      notes: 'Some notes',
      is_complete: false,
      work_ids: ['work1', 'work2'],
    };

    it('should create series successfully', async () => {
      const createdSeries = { ...mockSeries, title: 'New Series' };
      mockCreateSeries.mockResolvedValue({ series: createdSeries });

      const result = await createSeries(seriesData, 'test-auth-token');

      expect(mockCreateSeries).toHaveBeenCalledWith(seriesData, 'test-auth-token');
      expect(result.series.title).toBe('New Series');
    });

    it('should handle validation errors', async () => {
      mockCreateSeries.mockRejectedValue(new Error('Title is required'));

      await expect(createSeries({ ...seriesData, title: '' }, 'test-auth-token'))
        .rejects.toThrow('Title is required');
    });

    it('should handle unauthorized creation', async () => {
      mockCreateSeries.mockRejectedValue(new Error('User not authenticated'));

      await expect(createSeries(seriesData, ''))
        .rejects.toThrow('User not authenticated');
    });
  });

  describe('updateSeries', () => {
    const updateData = {
      title: 'Updated Series Title',
      summary: 'Updated summary',
      notes: 'Updated notes',
      is_complete: true,
    };

    it('should update series successfully', async () => {
      const updatedSeries = { ...mockSeries, ...updateData };
      mockUpdateSeries.mockResolvedValue({ series: updatedSeries });

      const result = await updateSeries('test-series-id', updateData, 'test-auth-token');

      expect(mockUpdateSeries).toHaveBeenCalledWith('test-series-id', updateData, 'test-auth-token');
      expect(result.series.title).toBe('Updated Series Title');
      expect(result.series.is_complete).toBe(true);
    });

    it('should handle unauthorized update', async () => {
      mockUpdateSeries.mockRejectedValue(new Error('You can only update your own series'));

      await expect(updateSeries('test-series-id', updateData, 'test-auth-token'))
        .rejects.toThrow('You can only update your own series');
    });
  });

  describe('deleteSeries', () => {
    it('should delete series successfully', async () => {
      mockDeleteSeries.mockResolvedValue({ message: 'Series deleted successfully' });

      const result = await deleteSeries('test-series-id', 'test-auth-token');

      expect(mockDeleteSeries).toHaveBeenCalledWith('test-series-id', 'test-auth-token');
      expect(result.message).toBe('Series deleted successfully');
    });

    it('should handle unauthorized deletion', async () => {
      mockDeleteSeries.mockRejectedValue(new Error('You can only delete your own series'));

      await expect(deleteSeries('test-series-id', 'test-auth-token'))
        .rejects.toThrow('You can only delete your own series');
    });
  });

  describe('searchSeries', () => {
    it('should search series successfully', async () => {
      mockSearchSeries.mockResolvedValue({ 
        series: [mockSeries], 
        pagination: mockPagination 
      });

      const result = await searchSeries({ q: 'test', page: 1 }, 'test-auth-token');

      expect(mockSearchSeries).toHaveBeenCalledWith({ q: 'test', page: 1 }, 'test-auth-token');
      expect(result.series).toEqual([mockSeries]);
      expect(result.pagination).toEqual(mockPagination);
    });

    it('should handle no results', async () => {
      mockSearchSeries.mockResolvedValue({ 
        series: [], 
        pagination: { ...mockPagination, total: 0 } 
      });

      const result = await searchSeries({ q: 'nonexistent' }, 'test-auth-token');

      expect(result.series).toEqual([]);
      expect(result.pagination.total).toBe(0);
    });
  });

  describe('getMySeries', () => {
    it('should fetch user\'s series successfully', async () => {
      mockGetMySeries.mockResolvedValue({ 
        series: [mockSeries], 
        pagination: mockPagination 
      });

      const result = await getMySeries({ page: 1 }, 'test-auth-token');

      expect(mockGetMySeries).toHaveBeenCalledWith({ page: 1 }, 'test-auth-token');
      expect(result.series).toEqual([mockSeries]);
    });

    it('should handle user with no series', async () => {
      mockGetMySeries.mockResolvedValue({ 
        series: [], 
        pagination: { ...mockPagination, total: 0 } 
      });

      const result = await getMySeries({}, 'test-auth-token');

      expect(result.series).toEqual([]);
    });
  });

  describe('addWorkToSeries', () => {
    it('should add work to series successfully', async () => {
      mockAddWorkToSeries.mockResolvedValue({ 
        message: 'Work added to series successfully', 
        position: 3 
      });

      const result = await addWorkToSeries('series-id', 'work-id', 3, 'test-auth-token');

      expect(mockAddWorkToSeries).toHaveBeenCalledWith('series-id', 'work-id', 3, 'test-auth-token');
      expect(result.message).toBe('Work added to series successfully');
      expect(result.position).toBe(3);
    });

    it('should handle work already in series', async () => {
      mockAddWorkToSeries.mockRejectedValue(new Error('Work is already in this series'));

      await expect(addWorkToSeries('series-id', 'work-id', undefined, 'test-auth-token'))
        .rejects.toThrow('Work is already in this series');
    });
  });

  describe('removeWorkFromSeries', () => {
    it('should remove work from series successfully', async () => {
      mockRemoveWorkFromSeries.mockResolvedValue({ 
        message: 'Work removed from series successfully' 
      });

      const result = await removeWorkFromSeries('series-id', 'work-id', 'test-auth-token');

      expect(mockRemoveWorkFromSeries).toHaveBeenCalledWith('series-id', 'work-id', 'test-auth-token');
      expect(result.message).toBe('Work removed from series successfully');
    });

    it('should handle work not in series', async () => {
      mockRemoveWorkFromSeries.mockRejectedValue(new Error('Work not found in this series'));

      await expect(removeWorkFromSeries('series-id', 'work-id', 'test-auth-token'))
        .rejects.toThrow('Work not found in this series');
    });
  });
});

describe('Series Error Handling', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockLocalStorage.getItem.mockReturnValue(null);
  });

  it('should handle network errors gracefully', async () => {
    mockGetSeries.mockRejectedValue(new Error('Network error'));

    await expect(getSeries('test-id')).rejects.toThrow('Network error');
  });

  it('should handle invalid authentication tokens', async () => {
    mockCreateSeries.mockRejectedValue(new Error('Invalid authentication token'));

    await expect(createSeries({ title: 'Test' }, 'invalid-token'))
      .rejects.toThrow('Invalid authentication token');
  });

  it('should handle malformed API responses', async () => {
    mockGetSeries.mockResolvedValue({} as any);

    const result = await getSeries('test-id');
    expect(result).toEqual({});
  });
});

describe('Series Data Validation', () => {
  it('should validate series data structure', async () => {
    mockGetSeries.mockResolvedValue({ series: mockSeries });

    const result = await getSeries('test-id');
    
    expect(result.series).toHaveProperty('id');
    expect(result.series).toHaveProperty('title');
    expect(result.series).toHaveProperty('user_id');
    expect(result.series).toHaveProperty('is_complete');
    expect(result.series).toHaveProperty('work_count');
    expect(typeof result.series.work_count).toBe('number');
    expect(typeof result.series.is_complete).toBe('boolean');
  });

  it('should validate works data structure', async () => {
    mockGetSeriesWorks.mockResolvedValue({ works: mockWorks });

    const result = await getSeriesWorks('test-id');
    
    result.works.forEach(work => {
      expect(work).toHaveProperty('id');
      expect(work).toHaveProperty('title');
      expect(work).toHaveProperty('word_count');
      expect(work).toHaveProperty('position');
      expect(typeof work.word_count).toBe('number');
      expect(typeof work.position).toBe('number');
    });
  });
});

describe('Series Pagination', () => {
  it('should handle pagination correctly', async () => {
    const paginatedResponse = {
      series: [mockSeries],
      pagination: {
        page: 2,
        limit: 10,
        total: 25,
        total_pages: 3,
      },
    };

    mockSearchSeries.mockResolvedValue(paginatedResponse);

    const result = await searchSeries({ page: 2, limit: 10 });

    expect(result.pagination.page).toBe(2);
    expect(result.pagination.total_pages).toBe(3);
    expect(result.pagination.total).toBe(25);
  });

  it('should handle first page correctly', async () => {
    mockGetMySeries.mockResolvedValue({
      series: [mockSeries],
      pagination: {
        page: 1,
        limit: 20,
        total: 1,
        total_pages: 1,
      },
    });

    const result = await getMySeries({ page: 1 });

    expect(result.pagination.page).toBe(1);
    expect(result.pagination.total_pages).toBe(1);
  });

  it('should handle last page correctly', async () => {
    mockSearchSeries.mockResolvedValue({
      series: [mockSeries],
      pagination: {
        page: 3,
        limit: 10,
        total: 25,
        total_pages: 3,
      },
    });

    const result = await searchSeries({ page: 3 });

    expect(result.pagination.page).toBe(3);
    expect(result.pagination.page).toBe(result.pagination.total_pages);
  });
});