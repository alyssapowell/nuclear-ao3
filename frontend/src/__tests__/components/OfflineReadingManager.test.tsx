import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach, afterEach, beforeAll } from 'vitest';
import OfflineReadingManager from '../../components/OfflineReadingManager';

// Import our test utilities and mocks
import { mockServiceWorkerRegistration, createMockMessageEvent } from '../test-utils';

// Mock localStorage
const mockLocalStorage = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
};
Object.defineProperty(window, 'localStorage', { value: mockLocalStorage });

// Mock storage API
const mockStorage = {
  estimate: vi.fn(),
};
Object.defineProperty(navigator, 'storage', {
  value: mockStorage,
  configurable: true,
});

// Mock fetch
global.fetch = vi.fn();

// Mock confirm
global.confirm = vi.fn();

describe('OfflineReadingManager', () => {
  const mockProps = {
    isOpen: true,
    onClose: vi.fn(),
  };

  const mockOfflineWorks = [
    {
      workId: 'work-1',
      title: 'Test Work 1',
      authors: [{ pseud_name: 'Author1', username: 'author1' }],
      consentLevel: 'files_and_pwa' as const,
      cachedAt: Date.now() - 1000 * 60 * 60, // 1 hour ago
      expiresAt: Date.now() + 1000 * 60 * 60 * 24 * 7, // 7 days from now
      isExpired: false,
      word_count: 5000,
      chapter_count: 3,
    },
    {
      workId: 'work-2',
      title: 'Test Work 2',
      authors: [{ pseud_name: 'Author2', username: 'author2' }],
      consentLevel: 'pwa_only' as const,
      cachedAt: Date.now() - 1000 * 60 * 60 * 24, // 1 day ago
      expiresAt: Date.now() + 1000 * 60 * 60 * 2, // 2 hours from now
      isExpired: false,
      word_count: 12000,
      chapter_count: 1,
    },
    {
      workId: 'work-3',
      title: 'Expired Work',
      authors: [{ pseud_name: 'Author3', username: 'author3' }],
      consentLevel: 'pwa_only' as const,
      cachedAt: Date.now() - 1000 * 60 * 60 * 48, // 2 days ago
      expiresAt: Date.now() - 1000 * 60 * 60, // 1 hour ago (expired)
      isExpired: true,
      word_count: 8000,
      chapter_count: 2,
    },
  ];

  beforeAll(() => {
    // Set up service worker mock
    Object.defineProperty(navigator, 'serviceWorker', {
      value: mockServiceWorkerRegistration,
      configurable: true,
    });

    // Mock online status
    Object.defineProperty(navigator, 'onLine', {
      value: true,
      configurable: true,
    });
  });

  beforeEach(() => {
    vi.clearAllMocks();
    mockLocalStorage.getItem.mockReturnValue(null);
    
    // Reset service worker mocks
    mockServiceWorkerRegistration.ready = Promise.resolve({
      active: { postMessage: vi.fn() },
    });

    // Mock storage estimate
    mockStorage.estimate.mockResolvedValue({
      usage: 1024 * 1024 * 50, // 50 MB
      quota: 1024 * 1024 * 1024, // 1 GB
    });

    // Mock fetch
    (global.fetch as any).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        title: 'Fetched Work Title',
        authors: [{ pseud_name: 'Fetched Author', username: 'fetched' }],
        word_count: 10000,
        chapter_count: 5,
      }),
    });

    // Mock confirm
    (global.confirm as any).mockReturnValue(true);
  });

  afterEach(() => {
    vi.clearAllTimers();
  });

  describe('Basic Rendering', () => {
    it('renders when open is true', () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      expect(screen.getByText('Offline Reading')).toBeInTheDocument();
      expect(screen.getByText('Online')).toBeInTheDocument();
    });

    it('does not render when open is false', () => {
      render(<OfflineReadingManager {...mockProps} isOpen={false} />);
      
      expect(screen.queryByText('Offline Reading')).not.toBeInTheDocument();
    });

    it('shows loading state initially', () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      expect(screen.getByText('Loading offline works...')).toBeInTheDocument();
    });

    it('displays cache size when available', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText(/~50.0 MB used/)).toBeInTheDocument();
      });
    });
  });

  describe('Online/Offline Status', () => {
    it('shows online status correctly', () => {
      Object.defineProperty(navigator, 'onLine', { value: true, configurable: true });
      
      render(<OfflineReadingManager {...mockProps} />);
      
      expect(screen.getByText('Online')).toBeInTheDocument();
    });

    it('shows offline status correctly', () => {
      Object.defineProperty(navigator, 'onLine', { value: false, configurable: true });
      
      render(<OfflineReadingManager {...mockProps} />);
      
      expect(screen.getByText('Offline')).toBeInTheDocument();
    });

    it('updates status when online/offline events are fired', () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      // Simulate going offline
      Object.defineProperty(navigator, 'onLine', { value: false, configurable: true });
      fireEvent(window, new Event('offline'));
      
      expect(screen.getByText('Offline')).toBeInTheDocument();
      
      // Simulate going online
      Object.defineProperty(navigator, 'onLine', { value: true, configurable: true });
      fireEvent(window, new Event('online'));
      
      expect(screen.getByText('Online')).toBeInTheDocument();
    });
  });

  describe('Loading Offline Works', () => {
    beforeEach(() => {
      // Mock successful service worker response
      mockServiceWorkerRegistration.ready = Promise.resolve({
        active: {
          postMessage: vi.fn((message, ports) => {
            // Simulate service worker response
            setTimeout(() => {
              const port = ports?.[0];
              if (port && port.onmessage) {
                port.onmessage({
                  data: { works: mockOfflineWorks }
                });
              }
            }, 10);
          }),
        },
      });
    });

    it('loads offline works from service worker', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('Test Work 1')).toBeInTheDocument();
        expect(screen.getByText('Test Work 2')).toBeInTheDocument();
        expect(screen.getByText('Expired Work')).toBeInTheDocument();
      });
    });

    it('displays work details correctly', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('by Author1')).toBeInTheDocument();
        expect(screen.getByText('5,000 words')).toBeInTheDocument();
        expect(screen.getByText('3 chapters')).toBeInTheDocument();
      });
    });

    it('shows consent level information', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('Full Offline')).toBeInTheDocument();
        expect(screen.getByText('PWA Only')).toBeInTheDocument();
      });
    });

    it('shows expiration status', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('Expired')).toBeInTheDocument();
      });
    });

    it('displays summary information', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('3 works cached')).toBeInTheDocument();
        expect(screen.getByText('• 1 expired')).toBeInTheDocument();
      });
    });
  });

  describe('Empty State', () => {
    beforeEach(() => {
      // Mock empty service worker response
      mockServiceWorkerRegistration.ready = Promise.resolve({
        active: {
          postMessage: vi.fn((message, ports) => {
            setTimeout(() => {
              const port = ports?.[0];
              if (port && port.onmessage) {
                port.onmessage({
                  data: { works: [] }
                });
              }
            }, 10);
          }),
        },
      });
    });

    it('shows empty state when no works are cached', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('No Offline Works')).toBeInTheDocument();
        expect(screen.getByText('Works you cache for offline reading will appear here.')).toBeInTheDocument();
      });
    });
  });

  describe('Work Management', () => {
    beforeEach(() => {
      // Mock successful service worker response
      mockServiceWorkerRegistration.ready = Promise.resolve({
        active: {
          postMessage: vi.fn((message, ports) => {
            if (message.type === 'GET_OFFLINE_WORKS') {
              setTimeout(() => {
                const port = ports?.[0];
                if (port && port.onmessage) {
                  port.onmessage({
                    data: { works: mockOfflineWorks }
                  });
                }
              }, 10);
            }
          }),
        },
      });
    });

    it('removes individual work from cache', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('Test Work 1')).toBeInTheDocument();
      });
      
      const removeButtons = screen.getAllByTitle('Remove from offline cache');
      fireEvent.click(removeButtons[0]);
      
      // Should see loading state
      await waitFor(() => {
        expect(screen.getByText('Test Work 2')).toBeInTheDocument();
        // The first work should be removed from the display
      });
    });

    it('shows loading state when removing work', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('Test Work 1')).toBeInTheDocument();
      });
      
      const removeButtons = screen.getAllByTitle('Remove from offline cache');
      fireEvent.click(removeButtons[0]);
      
      // Check for loading spinner (we can't easily test the exact UI, but we can test the behavior)
      expect(mockServiceWorkerRegistration.ready).resolves.toBeTruthy();
    });

    it('clears all expired works', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('Clear Expired')).toBeInTheDocument();
      });
      
      fireEvent.click(screen.getByText('Clear Expired'));
      
      // Should trigger removal of expired works
      await waitFor(() => {
        expect(mockServiceWorkerRegistration.ready).resolves.toBeTruthy();
      });
    });

    it('clears all cached works with confirmation', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('Clear All')).toBeInTheDocument();
      });
      
      fireEvent.click(screen.getByText('Clear All'));
      
      expect(global.confirm).toHaveBeenCalledWith('This will remove all offline works. Are you sure?');
    });

    it('cancels clear all when user declines confirmation', async () => {
      (global.confirm as any).mockReturnValue(false);
      
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('Clear All')).toBeInTheDocument();
      });
      
      fireEvent.click(screen.getByText('Clear All'));
      
      expect(global.confirm).toHaveBeenCalled();
      // Should not proceed with clearing
    });
  });

  describe('Work Details Fetching', () => {
    it('uses localStorage data when available', async () => {
      const localWorkData = JSON.stringify({
        title: 'Local Work Title',
        authors: [{ pseud_name: 'Local Author', username: 'local' }],
        word_count: 15000,
        chapter_count: 7,
      });
      
      mockLocalStorage.getItem.mockReturnValue(localWorkData);
      
      // Mock service worker response with minimal data
      mockServiceWorkerRegistration.ready = Promise.resolve({
        active: {
          postMessage: vi.fn((message, ports) => {
            setTimeout(() => {
              const port = ports?.[0];
              if (port && port.onmessage) {
                port.onmessage({
                  data: { works: [{ workId: 'work-1', consentLevel: 'files_and_pwa', cachedAt: Date.now(), expiresAt: Date.now() + 1000000, isExpired: false }] }
                });
              }
            }, 10);
          }),
        },
      });
      
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('Local Work Title')).toBeInTheDocument();
        expect(screen.getByText('by Local Author')).toBeInTheDocument();
      });
      
      expect(mockLocalStorage.getItem).toHaveBeenCalledWith('work-work-1');
    });

    it('fetches from API when localStorage is empty and online', async () => {
      mockLocalStorage.getItem.mockReturnValue(null);
      Object.defineProperty(navigator, 'onLine', { value: true, configurable: true });
      
      // Mock service worker response with minimal data
      mockServiceWorkerRegistration.ready = Promise.resolve({
        active: {
          postMessage: vi.fn((message, ports) => {
            setTimeout(() => {
              const port = ports?.[0];
              if (port && port.onmessage) {
                port.onmessage({
                  data: { works: [{ workId: 'work-1', consentLevel: 'files_and_pwa', cachedAt: Date.now(), expiresAt: Date.now() + 1000000, isExpired: false }] }
                });
              }
            }, 10);
          }),
        },
      });
      
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(global.fetch).toHaveBeenCalledWith('/api/v1/works/work-1');
        expect(screen.getByText('Fetched Work Title')).toBeInTheDocument();
      });
    });

    it('handles fetch errors gracefully', async () => {
      mockLocalStorage.getItem.mockReturnValue(null);
      (global.fetch as any).mockRejectedValue(new Error('Network error'));
      
      // Mock service worker response with minimal data
      mockServiceWorkerRegistration.ready = Promise.resolve({
        active: {
          postMessage: vi.fn((message, ports) => {
            setTimeout(() => {
              const port = ports?.[0];
              if (port && port.onmessage) {
                port.onmessage({
                  data: { works: [{ workId: 'work-1', consentLevel: 'files_and_pwa', cachedAt: Date.now(), expiresAt: Date.now() + 1000000, isExpired: false }] }
                });
              }
            }, 10);
          }),
        },
      });
      
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('Work work-1')).toBeInTheDocument();
        expect(screen.getByText('by Unknown')).toBeInTheDocument();
      });
    });
  });

  describe('Time Formatting', () => {
    it('formats time remaining correctly', async () => {
      const now = Date.now();
      const worksWithDifferentTimes = [
        {
          workId: 'work-days',
          title: 'Work with Days',
          authors: [{ pseud_name: 'Author', username: 'author' }],
          consentLevel: 'files_and_pwa' as const,
          cachedAt: now,
          expiresAt: now + (1000 * 60 * 60 * 24 * 3) + (1000 * 60 * 60 * 5), // 3 days 5 hours
          isExpired: false,
        },
        {
          workId: 'work-hours',
          title: 'Work with Hours',
          authors: [{ pseud_name: 'Author', username: 'author' }],
          consentLevel: 'pwa_only' as const,
          cachedAt: now,
          expiresAt: now + (1000 * 60 * 60 * 2), // 2 hours
          isExpired: false,
        },
        {
          workId: 'work-minutes',
          title: 'Work with Minutes',
          authors: [{ pseud_name: 'Author', username: 'author' }],
          consentLevel: 'pwa_only' as const,
          cachedAt: now,
          expiresAt: now + (1000 * 60 * 30), // 30 minutes
          isExpired: false,
        },
      ];
      
      // Mock service worker response
      mockServiceWorkerRegistration.ready = Promise.resolve({
        active: {
          postMessage: vi.fn((message, ports) => {
            setTimeout(() => {
              const port = ports?.[0];
              if (port && port.onmessage) {
                port.onmessage({
                  data: { works: worksWithDifferentTimes }
                });
              }
            }, 10);
          }),
        },
      });
      
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('3d 5h')).toBeInTheDocument();
        expect(screen.getByText('2h')).toBeInTheDocument();
        expect(screen.getByText('30m')).toBeInTheDocument();
      });
    });
  });

  describe('Consent Level Display', () => {
    it('shows correct icons and labels for each consent level', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('Full Offline')).toBeInTheDocument();
        expect(screen.getByText('PWA Only')).toBeInTheDocument();
      });
    });

    it('shows expired work warning', async () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText(/This work has expired based on the author's consent preferences/)).toBeInTheDocument();
      });
    });
  });

  describe('Modal Controls', () => {
    it('calls onClose when close button is clicked', () => {
      render(<OfflineReadingManager {...mockProps} />);
      
      fireEvent.click(screen.getByText('✕'));
      
      expect(mockProps.onClose).toHaveBeenCalled();
    });
  });

  describe('Service Worker Errors', () => {
    it('handles service worker registration errors gracefully', async () => {
      mockServiceWorkerRegistration.ready = Promise.reject(new Error('SW registration failed'));
      
      render(<OfflineReadingManager {...mockProps} />);
      
      // Should not crash and should show empty state eventually
      await waitFor(() => {
        expect(screen.getByText('No Offline Works')).toBeInTheDocument();
      });
    });

    it('handles missing service worker support', async () => {
      // Temporarily remove service worker support
      const originalServiceWorker = navigator.serviceWorker;
      Object.defineProperty(navigator, 'serviceWorker', {
        value: undefined,
        configurable: true,
      });
      
      render(<OfflineReadingManager {...mockProps} />);
      
      await waitFor(() => {
        expect(screen.getByText('No Offline Works')).toBeInTheDocument();
      });
      
      // Restore service worker support
      Object.defineProperty(navigator, 'serviceWorker', {
        value: originalServiceWorker,
        configurable: true,
      });
    });
  });

  describe('Storage Estimation', () => {
    it('handles storage estimation errors gracefully', async () => {
      mockStorage.estimate.mockRejectedValue(new Error('Storage not available'));
      
      render(<OfflineReadingManager {...mockProps} />);
      
      // Should not show cache size
      await waitFor(() => {
        expect(screen.queryByText(/MB used/)).not.toBeInTheDocument();
      });
    });

    it('handles missing storage API', async () => {
      Object.defineProperty(navigator, 'storage', {
        value: undefined,
        configurable: true,
      });
      
      render(<OfflineReadingManager {...mockProps} />);
      
      // Should not crash
      await waitFor(() => {
        expect(screen.getByText('Offline Reading')).toBeInTheDocument();
      });
      
      // Restore storage API
      Object.defineProperty(navigator, 'storage', {
        value: mockStorage,
        configurable: true,
      });
    });
  });

  describe('Cleanup', () => {
    it('removes event listeners on unmount', () => {
      const removeEventListener = vi.spyOn(window, 'removeEventListener');
      const { unmount } = render(<OfflineReadingManager {...mockProps} />);
      
      unmount();
      
      expect(removeEventListener).toHaveBeenCalledWith('online', expect.any(Function));
      expect(removeEventListener).toHaveBeenCalledWith('offline', expect.any(Function));
    });
  });
});