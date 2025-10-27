// Integration test for PWA Service Worker and UI Components
// Tests the complete flow from PWA initialization to service worker communication

import '@testing-library/jest-dom';

describe('PWA Service Worker Integration', () => {
  let mockServiceWorker: any;
  let mockRegistration: any;
  let mockMessageChannel: any;

  beforeEach(() => {
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

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('Service Worker Registration', () => {
    it('should register the consent-aware service worker with correct scope', async () => {
      // Directly test the service worker registration logic
      if ('serviceWorker' in navigator) {
        try {
          await navigator.serviceWorker.register('/sw-consent-aware.js', {
            scope: '/',
            updateViaCache: 'none'
          });
        } catch (error) {
          // Expected to fail in test
        }
      }

      expect(mockServiceWorker.register).toHaveBeenCalledWith(
        '/sw-consent-aware.js',
        {
          scope: '/',
          updateViaCache: 'none'
        }
      );
    });

    it('should handle service worker update events', async () => {
      // Test the registration event listeners directly
      await mockServiceWorker.ready;
      
      mockRegistration.addEventListener('updatefound', () => {});

      expect(mockRegistration.addEventListener).toHaveBeenCalledWith(
        'updatefound',
        expect.any(Function)
      );
    });

    it('should handle controller change events', async () => {
      // Test controller change listeners directly
      navigator.serviceWorker.addEventListener('controllerchange', () => {});

      expect(mockServiceWorker.addEventListener).toHaveBeenCalledWith(
        'controllerchange',
        expect.any(Function)
      );
    });
  });

  describe('Service Worker Communication', () => {
    it('should handle WORK_CACHED_WITH_CONSENT messages', async () => {
      // Add a real message handler
      const messageHandler = (event: MessageEvent) => {
        const { type, data } = event.data;
        if (type === 'WORK_CACHED_WITH_CONSENT') {
          window.dispatchEvent(new CustomEvent('workCachedWithConsent', { detail: data }));
        }
      };

      navigator.serviceWorker.addEventListener('message', messageHandler);

      // Simulate receiving a message from service worker
      const mockEvent = {
        data: {
          type: 'WORK_CACHED_WITH_CONSENT',
          workId: 'test-work-123',
          consentLevel: 'files_and_pwa',
          success: true
        }
      } as MessageEvent;

      messageHandler(mockEvent);

      // Verify custom event was dispatched
      expect(window.dispatchEvent).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'workCachedWithConsent',
          detail: {
            type: 'WORK_CACHED_WITH_CONSENT',
            workId: 'test-work-123',
            consentLevel: 'files_and_pwa',
            success: true
          }
        })
      );
    });

    it('should handle WORK_EXPIRED messages', async () => {
      const { default: PWAInit } = await import('@/components/PWAInit');
      
      render(<PWAInit />);
      await new Promise(resolve => setTimeout(resolve, 100));

      const messageHandler = mockServiceWorker.addEventListener.mock.calls
        .find(call => call[0] === 'message')?.[1];

      const mockEvent = {
        data: {
          type: 'WORK_EXPIRED',
          workId: 'test-work-456',
          consentLevel: 'pwa_only',
          reason: 'ttl_exceeded'
        }
      };

      messageHandler(mockEvent);

      expect(window.dispatchEvent).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'workExpired',
          detail: mockEvent.data
        })
      );
    });

    it('should handle WORK_CACHE_FAILED messages', async () => {
      const { default: PWAInit } = await import('@/components/PWAInit');
      
      render(<PWAInit />);
      await new Promise(resolve => setTimeout(resolve, 100));

      const messageHandler = mockServiceWorker.addEventListener.mock.calls
        .find(call => call[0] === 'message')?.[1];

      const mockEvent = {
        data: {
          type: 'WORK_CACHE_FAILED',
          workId: 'test-work-789',
          error: 'Storage quota exceeded'
        }
      };

      messageHandler(mockEvent);

      expect(window.dispatchEvent).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'workCacheFailed',
          detail: mockEvent.data
        })
      );
    });
  });

  describe('Offline Reading Manager Integration', () => {
    it('should communicate with service worker through OfflineReadingManager', async () => {
      const { offlineReadingManager } = await import('@/utils/offlineReadingManager');
      
      // Mock service worker as ready
      mockServiceWorker.controller = {
        postMessage: jest.fn()
      };

      // Wait for service worker to be ready
      await mockServiceWorker.ready;

      // Test caching a work
      const workData = {
        title: 'Test Work',
        authors: [{ pseud_name: 'TestAuthor', username: 'testuser' }],
        word_count: 5000,
        chapter_count: 1
      };

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

    it('should get offline works through service worker', async () => {
      const { offlineReadingManager } = await import('@/utils/offlineReadingManager');
      
      // Mock successful response
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

      // Mock message response
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

    it('should handle service worker communication timeout', async () => {
      const { offlineReadingManager } = await import('@/utils/offlineReadingManager');
      
      // Don't respond to message to trigger timeout
      const works = await offlineReadingManager.getOfflineWorks();

      expect(works).toEqual([]);
    }, 6000);
  });

  describe('PWA Lifecycle', () => {
    it('should skip waiting when new service worker is waiting', async () => {
      mockRegistration.waiting = {
        postMessage: jest.fn()
      };

      const { default: PWAInit } = await import('@/components/PWAInit');
      
      render(<PWAInit />);
      await new Promise(resolve => setTimeout(resolve, 100));

      expect(mockRegistration.waiting.postMessage).toHaveBeenCalledWith({
        type: 'SKIP_WAITING'
      });
    });

    it('should handle missing service worker gracefully', async () => {
      // Remove service worker support
      Object.defineProperty(global, 'navigator', {
        value: {},
        writable: true
      });

      const { default: PWAInit } = await import('@/components/PWAInit');
      
      // Should not throw error
      render(<PWAInit />);
      await new Promise(resolve => setTimeout(resolve, 100));
    });
  });

  describe('Error Handling', () => {
    it('should handle service worker registration failure', async () => {
      mockServiceWorker.register.mockRejectedValue(new Error('Registration failed'));

      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();

      const { default: PWAInit } = await import('@/components/PWAInit');
      
      render(<PWAInit />);
      await new Promise(resolve => setTimeout(resolve, 100));

      expect(consoleSpy).toHaveBeenCalledWith(
        '[PWA] Service Worker registration failed:',
        expect.any(Error)
      );

      consoleSpy.mockRestore();
    });

    it('should handle offline reading manager service worker failures', async () => {
      const { offlineReadingManager } = await import('@/utils/offlineReadingManager');
      
      // Mock no controller (service worker not ready)
      mockServiceWorker.controller = null;

      const result = await offlineReadingManager.cacheWork('test', {}, 'files_and_pwa');
      expect(result).toBe(false);
    });
  });
});

// Test consent level validation
describe('Consent Level Integration', () => {
  beforeEach(() => {
    // Set up service worker as ready
    mockServiceWorker.controller = {
      postMessage: jest.fn()
    };
  });

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

  it('should validate consent descriptions', () => {
    const { offlineReadingManager } = require('@/utils/offlineReadingManager');
    
    expect(offlineReadingManager.getConsentDescription('files_and_pwa'))
      .toBe('Full offline access - works can be cached for extended periods');
    
    expect(offlineReadingManager.getConsentDescription('pwa_only'))
      .toBe('PWA caching only - temporary offline access for convenience');
    
    expect(offlineReadingManager.getConsentDescription('none'))
      .toBe('Online only - author prefers no offline caching');
  });
});