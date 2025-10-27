// Service Worker Communication Integration Test
// Tests the communication between UI components and the consent-aware service worker

import '@testing-library/jest-dom';

describe('Service Worker Communication Integration', () => {
  let mockServiceWorker: any;
  let mockRegistration: any;
  let mockMessageChannel: any;

  beforeEach(() => {
    // Reset all mocks
    jest.clearAllMocks();

    // Mock service worker registration
    mockRegistration = {
      installing: null,
      waiting: null,
      active: {
        postMessage: jest.fn()
      },
      addEventListener: jest.fn(),
      scope: '/',
      updatefound: false
    };

    // Mock service worker with our consent-aware methods
    mockServiceWorker = {
      register: jest.fn().mockResolvedValue(mockRegistration),
      ready: Promise.resolve(mockRegistration),
      controller: {
        postMessage: jest.fn()
      },
      addEventListener: jest.fn()
    };

    // Mock MessageChannel
    mockMessageChannel = {
      port1: {
        onmessage: null,
        postMessage: jest.fn()
      },
      port2: {
        onmessage: null,
        postMessage: jest.fn()
      }
    };

    // Setup global mocks
    Object.defineProperty(global, 'navigator', {
      value: {
        serviceWorker: mockServiceWorker,
        onLine: true
      },
      writable: true
    });

    Object.defineProperty(global, 'MessageChannel', {
      value: jest.fn(() => mockMessageChannel),
      writable: true
    });

    Object.defineProperty(global, 'window', {
      value: {
        dispatchEvent: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
        location: { reload: jest.fn() }
      },
      writable: true
    });
  });

  describe('OfflineReadingManager Integration', () => {
    it('should communicate with service worker for caching works', async () => {
      const { offlineReadingManager } = await import('@/utils/offlineReadingManager');
      
      // Wait for service worker to be ready (simulated)
      await new Promise(resolve => setTimeout(resolve, 100));

      const workData = {
        title: 'Test Work',
        authors: [{ pseud_name: 'TestAuthor', username: 'testuser' }],
        word_count: 5000,
        chapter_count: 1
      };

      // Test caching a work
      await offlineReadingManager.cacheWork('test-work-123', workData, 'files_and_pwa');

      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledWith({
        type: 'CACHE_WORK_WITH_CONSENT',
        data: {
          workId: 'test-work-123',
          workData,
          consentLevel: 'files_and_pwa'
        }
      });
    });

    it('should get offline works through service worker message channel', async () => {
      const { offlineReadingManager } = await import('@/utils/offlineReadingManager');
      
      const mockWorks = [
        {
          workId: 'work-1',
          title: 'Test Work 1',
          authors: [{ pseud_name: 'Author1', username: 'user1' }],
          consentLevel: 'files_and_pwa',
          cachedAt: Date.now() - 1000,
          expiresAt: Date.now() + 86400000,
          isExpired: false
        }
      ];

      // Mock the message response
      setTimeout(() => {
        if (mockMessageChannel.port1.onmessage) {
          mockMessageChannel.port1.onmessage({
            data: {
              type: 'OFFLINE_WORKS_RESPONSE',
              works: mockWorks
            }
          });
        }
      }, 10);

      const works = await offlineReadingManager.getOfflineWorks();

      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledWith(
        { type: 'GET_OFFLINE_WORKS' },
        [mockMessageChannel.port2]
      );

      expect(works).toEqual(mockWorks);
    });

    it('should remove works through service worker', async () => {
      const { offlineReadingManager } = await import('@/utils/offlineReadingManager');
      
      await offlineReadingManager.removeWork('test-work-456');

      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledWith({
        type: 'CLEAR_CONSENT_CACHE',
        data: { workId: 'test-work-456' }
      });
    });

    it('should handle service worker not ready gracefully', async () => {
      const { offlineReadingManager } = await import('@/utils/offlineReadingManager');
      
      // Mock no controller (service worker not ready)
      mockServiceWorker.controller = null;

      const result = await offlineReadingManager.cacheWork('test', {}, 'files_and_pwa');
      expect(result).toBe(false);

      const works = await offlineReadingManager.getOfflineWorks();
      expect(works).toEqual([]);
    });
  });

  describe('Service Worker Registration', () => {
    it('should register the consent-aware service worker', async () => {
      if ('serviceWorker' in navigator) {
        await navigator.serviceWorker.register('/sw-consent-aware.js', {
          scope: '/',
          updateViaCache: 'none'
        });
      }

      expect(mockServiceWorker.register).toHaveBeenCalledWith(
        '/sw-consent-aware.js',
        {
          scope: '/',
          updateViaCache: 'none'
        }
      );
    });
  });

  describe('Consent Level Validation', () => {
    it('should prevent caching when consent level is none', async () => {
      const { offlineReadingManager } = await import('@/utils/offlineReadingManager');
      
      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation();

      // This should fail because 'none' is not allowed
      const result = await offlineReadingManager.cacheWork('test', {}, 'none' as any);
      
      expect(result).toBe(false);
      expect(consoleSpy).toHaveBeenCalledWith(
        '[OfflineManager] Cannot cache work - author does not allow offline reading'
      );

      consoleSpy.mockRestore();
    });

    it('should allow valid consent levels', async () => {
      const { offlineReadingManager } = await import('@/utils/offlineReadingManager');
      
      // Test files_and_pwa consent
      await offlineReadingManager.cacheWork('test1', {}, 'files_and_pwa');
      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'CACHE_WORK_WITH_CONSENT',
          data: expect.objectContaining({
            consentLevel: 'files_and_pwa'
          })
        })
      );

      // Test pwa_only consent
      await offlineReadingManager.cacheWork('test2', {}, 'pwa_only');
      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'CACHE_WORK_WITH_CONSENT',
          data: expect.objectContaining({
            consentLevel: 'pwa_only'
          })
        })
      );
    });

    it('should provide correct consent descriptions', () => {
      const { offlineReadingManager } = require('@/utils/offlineReadingManager');
      
      expect(offlineReadingManager.getConsentDescription('files_and_pwa'))
        .toBe('Full offline access - works can be cached for extended periods');
      
      expect(offlineReadingManager.getConsentDescription('pwa_only'))
        .toBe('PWA caching only - temporary offline access for convenience');
      
      expect(offlineReadingManager.getConsentDescription('none'))
        .toBe('Online only - author prefers no offline caching');
    });
  });

  describe('Cache Management', () => {
    it('should get cache statistics', async () => {
      const { offlineReadingManager } = await import('@/utils/offlineReadingManager');
      
      // Mock storage estimate
      Object.defineProperty(global.navigator, 'storage', {
        value: {
          estimate: jest.fn().mockResolvedValue({
            usage: 1024 * 1024 * 5, // 5MB
            quota: 1024 * 1024 * 1024 // 1GB
          })
        },
        writable: true
      });

      // Mock offline works
      setTimeout(() => {
        if (mockMessageChannel.port1.onmessage) {
          mockMessageChannel.port1.onmessage({
            data: {
              type: 'OFFLINE_WORKS_RESPONSE',
              works: [
                { workId: '1', isExpired: false },
                { workId: '2', isExpired: true }
              ]
            }
          });
        }
      }, 10);

      const stats = await offlineReadingManager.getCacheStats();

      expect(stats).toEqual({
        totalWorks: 2,
        expiredWorks: 1,
        storageUsed: '5.0 MB',
        storageQuota: '1024 MB',
        utilizationPercentage: 0
      });
    });

    it('should format time remaining correctly', () => {
      const { offlineReadingManager } = require('@/utils/offlineReadingManager');
      
      const now = Date.now();
      
      // Test expired
      expect(offlineReadingManager.formatTimeRemaining(now - 1000)).toBe('Expired');
      
      // Test days (add a bit more time to account for timing differences)
      const tenDaysFromNow = now + (10 * 24 * 60 * 60 * 1000) + 1000;
      expect(offlineReadingManager.formatTimeRemaining(tenDaysFromNow))
        .toBe('10 days');
      
      // Test hours and minutes
      const timeWithHoursMinutes = now + (2 * 60 * 60 * 1000) + (30 * 60 * 1000);
      expect(offlineReadingManager.formatTimeRemaining(timeWithHoursMinutes))
        .toBe('2h 30m');
    });
  });

  describe('Network Status', () => {
    it('should detect online status', () => {
      const { offlineReadingManager } = require('@/utils/offlineReadingManager');
      
      expect(offlineReadingManager.isOnline()).toBe(true);
    });

    it('should handle connection change events', () => {
      const { offlineReadingManager } = require('@/utils/offlineReadingManager');
      
      const mockCallback = jest.fn();
      const cleanup = offlineReadingManager.onConnectionChange(mockCallback);

      expect(window.addEventListener).toHaveBeenCalledWith('online', expect.any(Function));
      expect(window.addEventListener).toHaveBeenCalledWith('offline', expect.any(Function));

      // Test cleanup
      cleanup();
      expect(window.removeEventListener).toHaveBeenCalledWith('online', expect.any(Function));
      expect(window.removeEventListener).toHaveBeenCalledWith('offline', expect.any(Function));
    });
  });
});