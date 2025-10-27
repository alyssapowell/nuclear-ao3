import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { offlineReadingManager } from '../../utils/offlineReadingManager';

// Mock service worker registration
const mockServiceWorker = {
  ready: Promise.resolve({
    active: {
      postMessage: vi.fn(),
    },
  }),
  controller: {
    postMessage: vi.fn(),
  },
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
};

Object.defineProperty(navigator, 'serviceWorker', {
  value: mockServiceWorker,
  configurable: true,
});

// Mock MessageChannel
global.MessageChannel = class MessageChannel {
  port1 = {
    onmessage: null as any,
    postMessage: vi.fn(),
    start: vi.fn(),
    close: vi.fn(),
  };
  port2 = {
    onmessage: null as any,
    postMessage: vi.fn(),
    start: vi.fn(),
    close: vi.fn(),
  };
};

describe('Service Worker Communication Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    
    // Reset service worker mocks
    mockServiceWorker.ready = Promise.resolve({
      active: {
        postMessage: vi.fn(),
      },
    });
    mockServiceWorker.controller = {
      postMessage: vi.fn(),
    };
  });

  afterEach(() => {
    // Clean up any listeners
    offlineReadingManager.clearAllListeners();
  });

  describe('Work Caching Operations', () => {
    it('sends cache work message with correct data', async () => {
      const mockWork = {
        id: 'work-123',
        title: 'Test Work',
        authors: [{ pseud_name: 'Test Author', username: 'testauthor' }],
        chapters: [
          {
            id: 'ch-1',
            number: 1,
            title: 'Chapter 1',
            content: 'Test content',
            word_count: 500,
          },
        ],
        offline_reading_preference: 'files_and_pwa' as const,
      };

      await offlineReadingManager.cacheWorkForOffline(mockWork);

      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledWith({
        type: 'CACHE_WORK_WITH_CONSENT',
        data: {
          workId: 'work-123',
          workData: mockWork,
          consentLevel: 'files_and_pwa',
        },
      });
    });

    it('handles cache work success response', async () => {
      const mockWork = {
        id: 'work-123',
        title: 'Test Work',
        authors: [],
        chapters: [],
        offline_reading_preference: 'files_and_pwa' as const,
      };

      let messageCallback: Function;
      mockServiceWorker.addEventListener.mockImplementation((event, callback) => {
        if (event === 'message') {
          messageCallback = callback;
        }
      });

      const cachePromise = offlineReadingManager.cacheWorkForOffline(mockWork);

      // Simulate success response from service worker
      const mockEvent = {
        data: {
          type: 'WORK_CACHED_WITH_CONSENT',
          workId: 'work-123',
          consentLevel: 'files_and_pwa',
          success: true,
        },
      };

      // Wait a bit then trigger the response
      setTimeout(() => {
        messageCallback?.(mockEvent);
      }, 10);

      const result = await cachePromise;
      expect(result).toBe(true);
    });

    it('handles cache work failure response', async () => {
      const mockWork = {
        id: 'work-123',
        title: 'Test Work',
        authors: [],
        chapters: [],
        offline_reading_preference: 'files_and_pwa' as const,
      };

      let messageCallback: Function;
      mockServiceWorker.addEventListener.mockImplementation((event, callback) => {
        if (event === 'message') {
          messageCallback = callback;
        }
      });

      const cachePromise = offlineReadingManager.cacheWorkForOffline(mockWork);

      // Simulate failure response from service worker
      const mockEvent = {
        data: {
          type: 'WORK_CACHE_FAILED',
          workId: 'work-123',
          error: 'Storage quota exceeded',
        },
      };

      setTimeout(() => {
        messageCallback?.(mockEvent);
      }, 10);

      const result = await cachePromise;
      expect(result).toBe(false);
    });

    it('rejects work caching for none consent level', async () => {
      const mockWork = {
        id: 'work-123',
        title: 'Test Work',
        authors: [],
        chapters: [],
        offline_reading_preference: 'none' as const,
      };

      const result = await offlineReadingManager.cacheWorkForOffline(mockWork);
      expect(result).toBe(false);
      expect(mockServiceWorker.controller.postMessage).not.toHaveBeenCalled();
    });
  });

  describe('Work Retrieval Operations', () => {
    it('requests offline works from service worker', async () => {
      const mockWorks = [
        {
          workId: 'work-1',
          consentLevel: 'files_and_pwa' as const,
          cachedAt: Date.now(),
          expiresAt: Date.now() + 1000000,
          isExpired: false,
        },
        {
          workId: 'work-2',
          consentLevel: 'pwa_only' as const,
          cachedAt: Date.now(),
          expiresAt: Date.now() + 1000000,
          isExpired: false,
        },
      ];

      // Mock MessageChannel communication
      const originalMessageChannel = global.MessageChannel;
      global.MessageChannel = class MockMessageChannel {
        port1 = {
          onmessage: null as any,
          postMessage: vi.fn(),
          start: vi.fn(),
          close: vi.fn(),
        };
        port2 = {
          onmessage: null as any,
          postMessage: vi.fn(),
          start: vi.fn(),
          close: vi.fn(),
        };

        constructor() {
          // Simulate service worker response
          setTimeout(() => {
            if (this.port1.onmessage) {
              this.port1.onmessage({
                data: { works: mockWorks },
              });
            }
          }, 10);
        }
      };

      const works = await offlineReadingManager.getOfflineWorks();

      expect(works).toEqual(mockWorks);
      expect(mockServiceWorker.ready).toHaveBeenCalled();

      // Restore original MessageChannel
      global.MessageChannel = originalMessageChannel;
    });

    it('handles empty offline works response', async () => {
      const originalMessageChannel = global.MessageChannel;
      global.MessageChannel = class MockMessageChannel {
        port1 = {
          onmessage: null as any,
          postMessage: vi.fn(),
          start: vi.fn(),
          close: vi.fn(),
        };
        port2 = {
          onmessage: null as any,
          postMessage: vi.fn(),
          start: vi.fn(),
          close: vi.fn(),
        };

        constructor() {
          setTimeout(() => {
            if (this.port1.onmessage) {
              this.port1.onmessage({
                data: { works: [] },
              });
            }
          }, 10);
        }
      };

      const works = await offlineReadingManager.getOfflineWorks();

      expect(works).toEqual([]);

      global.MessageChannel = originalMessageChannel;
    });
  });

  describe('Work Deletion Operations', () => {
    it('sends delete work message', async () => {
      await offlineReadingManager.removeWorkFromCache('work-123');

      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledWith({
        type: 'CLEAR_CONSENT_CACHE',
        data: { workId: 'work-123' },
      });
    });

    it('handles bulk work deletion', async () => {
      const workIds = ['work-1', 'work-2', 'work-3'];

      await offlineReadingManager.clearMultipleWorks(workIds);

      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledWith({
        type: 'CLEAR_MULTIPLE_WORKS',
        data: { workIds },
      });
    });

    it('handles clear all cache operation', async () => {
      await offlineReadingManager.clearAllCache();

      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledWith({
        type: 'CLEAR_ALL_CONSENT_CACHE',
      });
    });
  });

  describe('Message Event Handling', () => {
    it('sets up message listeners on initialization', async () => {
      await offlineReadingManager.initialize();

      expect(mockServiceWorker.addEventListener).toHaveBeenCalledWith(
        'message',
        expect.any(Function)
      );
    });

    it('handles work cached messages', async () => {
      const mockCallback = vi.fn();
      offlineReadingManager.onMessage('WORK_CACHED_WITH_CONSENT', mockCallback);

      await offlineReadingManager.initialize();

      // Get the registered message handler
      const messageHandler = mockServiceWorker.addEventListener.mock.calls.find(
        call => call[0] === 'message'
      )?.[1];

      expect(messageHandler).toBeDefined();

      // Simulate receiving a work cached message
      const mockEvent = {
        data: {
          type: 'WORK_CACHED_WITH_CONSENT',
          workId: 'work-123',
          consentLevel: 'files_and_pwa',
        },
      };

      messageHandler?.(mockEvent);

      expect(mockCallback).toHaveBeenCalledWith({
        type: 'WORK_CACHED_WITH_CONSENT',
        workId: 'work-123',
        consentLevel: 'files_and_pwa',
      });
    });

    it('handles work deleted messages', async () => {
      const mockCallback = vi.fn();
      offlineReadingManager.onMessage('WORK_DELETED', mockCallback);

      await offlineReadingManager.initialize();

      const messageHandler = mockServiceWorker.addEventListener.mock.calls.find(
        call => call[0] === 'message'
      )?.[1];

      const mockEvent = {
        data: {
          type: 'WORK_DELETED',
          workId: 'work-123',
          reason: 'author_deleted',
        },
      };

      messageHandler?.(mockEvent);

      expect(mockCallback).toHaveBeenCalledWith({
        type: 'WORK_DELETED',
        workId: 'work-123',
        reason: 'author_deleted',
      });
    });

    it('handles work expired messages', async () => {
      const mockCallback = vi.fn();
      offlineReadingManager.onMessage('WORK_EXPIRED', mockCallback);

      await offlineReadingManager.initialize();

      const messageHandler = mockServiceWorker.addEventListener.mock.calls.find(
        call => call[0] === 'message'
      )?.[1];

      const mockEvent = {
        data: {
          type: 'WORK_EXPIRED',
          workId: 'work-123',
          consentLevel: 'pwa_only',
        },
      };

      messageHandler?.(mockEvent);

      expect(mockCallback).toHaveBeenCalledWith({
        type: 'WORK_EXPIRED',
        workId: 'work-123',
        consentLevel: 'pwa_only',
      });
    });

    it('filters messages for specific work IDs', async () => {
      const mockCallback = vi.fn();
      offlineReadingManager.onMessage('WORK_CACHED_WITH_CONSENT', mockCallback, 'work-123');

      await offlineReadingManager.initialize();

      const messageHandler = mockServiceWorker.addEventListener.mock.calls.find(
        call => call[0] === 'message'
      )?.[1];

      // Send message for different work - should not trigger callback
      const mockEvent1 = {
        data: {
          type: 'WORK_CACHED_WITH_CONSENT',
          workId: 'work-456',
          consentLevel: 'files_and_pwa',
        },
      };

      messageHandler?.(mockEvent1);
      expect(mockCallback).not.toHaveBeenCalled();

      // Send message for matching work - should trigger callback
      const mockEvent2 = {
        data: {
          type: 'WORK_CACHED_WITH_CONSENT',
          workId: 'work-123',
          consentLevel: 'files_and_pwa',
        },
      };

      messageHandler?.(mockEvent2);
      expect(mockCallback).toHaveBeenCalledWith({
        type: 'WORK_CACHED_WITH_CONSENT',
        workId: 'work-123',
        consentLevel: 'files_and_pwa',
      });
    });
  });

  describe('Error Handling', () => {
    it('handles service worker registration failures', async () => {
      mockServiceWorker.ready = Promise.reject(new Error('SW not available'));

      const result = await offlineReadingManager.initialize();
      expect(result).toBe(false);
    });

    it('handles missing service worker controller', async () => {
      mockServiceWorker.controller = null;

      const mockWork = {
        id: 'work-123',
        title: 'Test Work',
        authors: [],
        chapters: [],
        offline_reading_preference: 'files_and_pwa' as const,
      };

      const result = await offlineReadingManager.cacheWorkForOffline(mockWork);
      expect(result).toBe(false);
    });

    it('handles message channel communication failures', async () => {
      const originalMessageChannel = global.MessageChannel;
      global.MessageChannel = class MockMessageChannel {
        port1 = {
          onmessage: null as any,
          postMessage: vi.fn(),
          start: vi.fn(),
          close: vi.fn(),
        };
        port2 = {
          onmessage: null as any,
          postMessage: vi.fn(),
          start: vi.fn(),
          close: vi.fn(),
        };

        constructor() {
          // Simulate timeout - no response from service worker
        }
      };

      const worksPromise = offlineReadingManager.getOfflineWorks();
      
      // Should timeout and return empty array
      const works = await worksPromise;
      expect(works).toEqual([]);

      global.MessageChannel = originalMessageChannel;
    });

    it('handles malformed service worker responses', async () => {
      const originalMessageChannel = global.MessageChannel;
      global.MessageChannel = class MockMessageChannel {
        port1 = {
          onmessage: null as any,
          postMessage: vi.fn(),
          start: vi.fn(),
          close: vi.fn(),
        };
        port2 = {
          onmessage: null as any,
          postMessage: vi.fn(),
          start: vi.fn(),
          close: vi.fn(),
        };

        constructor() {
          setTimeout(() => {
            if (this.port1.onmessage) {
              // Send malformed response
              this.port1.onmessage({
                data: { invalid: 'response' },
              });
            }
          }, 10);
        }
      };

      const works = await offlineReadingManager.getOfflineWorks();
      expect(works).toEqual([]);

      global.MessageChannel = originalMessageChannel;
    });
  });

  describe('Connection Status Integration', () => {
    beforeEach(() => {
      // Mock navigator.onLine
      Object.defineProperty(navigator, 'onLine', {
        value: true,
        configurable: true,
      });
    });

    it('tracks online status correctly', () => {
      expect(offlineReadingManager.isOnline()).toBe(true);

      // Simulate going offline
      Object.defineProperty(navigator, 'onLine', { value: false, configurable: true });
      expect(offlineReadingManager.isOnline()).toBe(false);
    });

    it('updates connection status on network events', () => {
      const statusCallback = vi.fn();
      offlineReadingManager.onConnectionChange(statusCallback);

      // Simulate offline event
      const offlineEvent = new Event('offline');
      window.dispatchEvent(offlineEvent);

      expect(statusCallback).toHaveBeenCalledWith(false);

      // Simulate online event
      const onlineEvent = new Event('online');
      window.dispatchEvent(onlineEvent);

      expect(statusCallback).toHaveBeenCalledWith(true);
    });

    it('allows offline operations when cached data is available', async () => {
      // Simulate being offline
      Object.defineProperty(navigator, 'onLine', { value: false, configurable: true });

      // Should still allow cache removal operations
      await offlineReadingManager.removeWorkFromCache('work-123');
      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalled();
    });
  });

  describe('Storage Operations', () => {
    it('gets cache storage estimates', async () => {
      const mockEstimate = {
        usage: 1024 * 1024 * 50, // 50 MB
        quota: 1024 * 1024 * 1024, // 1 GB
      };

      Object.defineProperty(navigator, 'storage', {
        value: {
          estimate: () => Promise.resolve(mockEstimate),
        },
        configurable: true,
      });

      const estimate = await offlineReadingManager.getStorageEstimate();
      expect(estimate).toEqual(mockEstimate);
    });

    it('handles storage API unavailability', async () => {
      Object.defineProperty(navigator, 'storage', {
        value: undefined,
        configurable: true,
      });

      const estimate = await offlineReadingManager.getStorageEstimate();
      expect(estimate).toEqual({ usage: 0, quota: 0 });
    });
  });

  describe('Lifecycle Management', () => {
    it('properly initializes and cleans up', async () => {
      await offlineReadingManager.initialize();
      expect(mockServiceWorker.addEventListener).toHaveBeenCalled();

      offlineReadingManager.destroy();
      expect(mockServiceWorker.removeEventListener).toHaveBeenCalled();
    });

    it('prevents multiple initialization', async () => {
      await offlineReadingManager.initialize();
      await offlineReadingManager.initialize(); // Second call

      // Should only add listener once
      expect(mockServiceWorker.addEventListener).toHaveBeenCalledTimes(1);
    });

    it('cleans up all message listeners', () => {
      const callback1 = vi.fn();
      const callback2 = vi.fn();

      offlineReadingManager.onMessage('WORK_CACHED_WITH_CONSENT', callback1);
      offlineReadingManager.onMessage('WORK_DELETED', callback2);

      offlineReadingManager.clearAllListeners();

      // Verify listeners are cleared (this would be implementation-specific)
      expect(offlineReadingManager.getListenerCount()).toBe(0);
    });
  });

  describe('Singleton Behavior', () => {
    it('maintains singleton instance', () => {
      const instance1 = offlineReadingManager;
      const instance2 = offlineReadingManager;

      expect(instance1).toBe(instance2);
    });

    it('preserves state across multiple access', async () => {
      await offlineReadingManager.initialize();
      
      const isInitialized1 = offlineReadingManager.isInitialized();
      const isInitialized2 = offlineReadingManager.isInitialized();

      expect(isInitialized1).toBe(true);
      expect(isInitialized2).toBe(true);
    });
  });
});