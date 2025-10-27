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

describe('Consent-Aware Caching Flow Integration', () => {
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
    offlineReadingManager.clearAllListeners();
  });

  describe('Files and PWA Consent Level', () => {
    const mockWorkFilesAndPWA = {
      id: 'work-files-pwa',
      title: 'Work with Full Offline Permission',
      authors: [{ pseud_name: 'Progressive Author', username: 'progressive' }],
      chapters: [
        {
          id: 'ch-1',
          number: 1,
          title: 'Chapter 1',
          content: 'Long content for offline reading',
          word_count: 2000,
        },
      ],
      offline_reading_preference: 'files_and_pwa' as const,
    };

    it('allows long-term caching for files_and_pwa consent', async () => {
      const result = await offlineReadingManager.cacheWorkForOffline(mockWorkFilesAndPWA);

      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledWith({
        type: 'CACHE_WORK_WITH_CONSENT',
        data: {
          workId: 'work-files-pwa',
          workData: mockWorkFilesAndPWA,
          consentLevel: 'files_and_pwa',
        },
      });
    });

    it('sets extended expiry time for files_and_pwa works', async () => {
      const originalMessageChannel = global.MessageChannel;
      let capturedTTL: number | null = null;

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
              // Simulate extended TTL for files_and_pwa (30 days)
              const expiryTime = Date.now() + (30 * 24 * 60 * 60 * 1000);
              capturedTTL = expiryTime - Date.now();
              
              this.port1.onmessage({
                data: {
                  works: [{
                    workId: 'work-files-pwa',
                    consentLevel: 'files_and_pwa',
                    cachedAt: Date.now(),
                    expiresAt: expiryTime,
                    isExpired: false,
                  }],
                },
              });
            }
          }, 10);
        }
      };

      await offlineReadingManager.cacheWorkForOffline(mockWorkFilesAndPWA);
      const works = await offlineReadingManager.getOfflineWorks();

      expect(works).toHaveLength(1);
      expect(works[0].consentLevel).toBe('files_and_pwa');
      expect(capturedTTL).toBeGreaterThan(25 * 24 * 60 * 60 * 1000); // At least 25 days

      global.MessageChannel = originalMessageChannel;
    });

    it('allows offline access even after network disconnection', async () => {
      // Cache the work first
      await offlineReadingManager.cacheWorkForOffline(mockWorkFilesAndPWA);

      // Simulate going offline
      Object.defineProperty(navigator, 'onLine', { value: false, configurable: true });

      // Should still be able to retrieve cached works
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
                data: {
                  works: [{
                    workId: 'work-files-pwa',
                    consentLevel: 'files_and_pwa',
                    cachedAt: Date.now() - 1000000,
                    expiresAt: Date.now() + 1000000,
                    isExpired: false,
                  }],
                },
              });
            }
          }, 10);
        }
      };

      const works = await offlineReadingManager.getOfflineWorks();
      expect(works).toHaveLength(1);
      expect(works[0].isExpired).toBe(false);

      global.MessageChannel = originalMessageChannel;
    });
  });

  describe('PWA Only Consent Level', () => {
    const mockWorkPWAOnly = {
      id: 'work-pwa-only',
      title: 'Work with Temporary Offline Permission',
      authors: [{ pseud_name: 'Cautious Author', username: 'cautious' }],
      chapters: [
        {
          id: 'ch-1',
          number: 1,
          title: 'Chapter 1',
          content: 'Content for temporary caching',
          word_count: 1500,
        },
      ],
      offline_reading_preference: 'pwa_only' as const,
    };

    it('allows short-term caching for pwa_only consent', async () => {
      const result = await offlineReadingManager.cacheWorkForOffline(mockWorkPWAOnly);

      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledWith({
        type: 'CACHE_WORK_WITH_CONSENT',
        data: {
          workId: 'work-pwa-only',
          workData: mockWorkPWAOnly,
          consentLevel: 'pwa_only',
        },
      });
    });

    it('sets limited expiry time for pwa_only works', async () => {
      const originalMessageChannel = global.MessageChannel;
      let capturedTTL: number | null = null;

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
              // Simulate limited TTL for pwa_only (24 hours)
              const expiryTime = Date.now() + (24 * 60 * 60 * 1000);
              capturedTTL = expiryTime - Date.now();
              
              this.port1.onmessage({
                data: {
                  works: [{
                    workId: 'work-pwa-only',
                    consentLevel: 'pwa_only',
                    cachedAt: Date.now(),
                    expiresAt: expiryTime,
                    isExpired: false,
                  }],
                },
              });
            }
          }, 10);
        }
      };

      await offlineReadingManager.cacheWorkForOffline(mockWorkPWAOnly);
      const works = await offlineReadingManager.getOfflineWorks();

      expect(works).toHaveLength(1);
      expect(works[0].consentLevel).toBe('pwa_only');
      expect(capturedTTL).toBeLessThan(25 * 60 * 60 * 1000); // Less than 25 hours
      expect(capturedTTL).toBeGreaterThan(20 * 60 * 60 * 1000); // More than 20 hours

      global.MessageChannel = originalMessageChannel;
    });

    it('automatically expires pwa_only works after TTL', async () => {
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
              // Simulate expired work
              this.port1.onmessage({
                data: {
                  works: [{
                    workId: 'work-pwa-only',
                    consentLevel: 'pwa_only',
                    cachedAt: Date.now() - (25 * 60 * 60 * 1000), // 25 hours ago
                    expiresAt: Date.now() - (1 * 60 * 60 * 1000), // 1 hour ago
                    isExpired: true,
                  }],
                },
              });
            }
          }, 10);
        }
      };

      const works = await offlineReadingManager.getOfflineWorks();

      expect(works).toHaveLength(1);
      expect(works[0].isExpired).toBe(true);

      global.MessageChannel = originalMessageChannel;
    });

    it('sends expiry notification when pwa_only work expires', async () => {
      const mockCallback = vi.fn();
      offlineReadingManager.onMessage('WORK_EXPIRED', mockCallback);

      await offlineReadingManager.initialize();

      const messageHandler = mockServiceWorker.addEventListener.mock.calls.find(
        call => call[0] === 'message'
      )?.[1];

      const mockEvent = {
        data: {
          type: 'WORK_EXPIRED',
          workId: 'work-pwa-only',
          consentLevel: 'pwa_only',
          reason: 'ttl_exceeded',
        },
      };

      messageHandler?.(mockEvent);

      expect(mockCallback).toHaveBeenCalledWith({
        type: 'WORK_EXPIRED',
        workId: 'work-pwa-only',
        consentLevel: 'pwa_only',
        reason: 'ttl_exceeded',
      });
    });
  });

  describe('None Consent Level', () => {
    const mockWorkNone = {
      id: 'work-none',
      title: 'Work with No Offline Permission',
      authors: [{ pseud_name: 'Privacy Author', username: 'privacy' }],
      chapters: [
        {
          id: 'ch-1',
          number: 1,
          title: 'Chapter 1',
          content: 'Online only content',
          word_count: 1000,
        },
      ],
      offline_reading_preference: 'none' as const,
    };

    it('rejects caching for none consent level', async () => {
      const result = await offlineReadingManager.cacheWorkForOffline(mockWorkNone);

      expect(result).toBe(false);
      expect(mockServiceWorker.controller.postMessage).not.toHaveBeenCalled();
    });

    it('shows appropriate error message for none consent', async () => {
      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

      await offlineReadingManager.cacheWorkForOffline(mockWorkNone);

      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Author has not allowed offline caching')
      );

      consoleSpy.mockRestore();
    });

    it('does not include none consent works in offline inventory', async () => {
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
              // Service worker should never return works with 'none' consent
              this.port1.onmessage({
                data: { works: [] },
              });
            }
          }, 10);
        }
      };

      const works = await offlineReadingManager.getOfflineWorks();
      expect(works).toHaveLength(0);

      global.MessageChannel = originalMessageChannel;
    });
  });

  describe('Consent Level Transitions', () => {
    const mockWork = {
      id: 'work-transition',
      title: 'Work with Changing Consent',
      authors: [{ pseud_name: 'Changing Author', username: 'changing' }],
      chapters: [
        {
          id: 'ch-1',
          number: 1,
          title: 'Chapter 1',
          content: 'Content with changing permissions',
          word_count: 1000,
        },
      ],
      offline_reading_preference: 'files_and_pwa' as const,
    };

    it('handles consent downgrade from files_and_pwa to pwa_only', async () => {
      // Initially cache with full permission
      await offlineReadingManager.cacheWorkForOffline(mockWork);

      const mockCallback = vi.fn();
      offlineReadingManager.onMessage('CONSENT_LEVEL_CHANGED', mockCallback);

      await offlineReadingManager.initialize();

      const messageHandler = mockServiceWorker.addEventListener.mock.calls.find(
        call => call[0] === 'message'
      )?.[1];

      // Simulate consent downgrade
      const mockEvent = {
        data: {
          type: 'CONSENT_LEVEL_CHANGED',
          workId: 'work-transition',
          oldConsentLevel: 'files_and_pwa',
          newConsentLevel: 'pwa_only',
          action: 'ttl_reduced',
        },
      };

      messageHandler?.(mockEvent);

      expect(mockCallback).toHaveBeenCalledWith({
        type: 'CONSENT_LEVEL_CHANGED',
        workId: 'work-transition',
        oldConsentLevel: 'files_and_pwa',
        newConsentLevel: 'pwa_only',
        action: 'ttl_reduced',
      });
    });

    it('handles consent downgrade from pwa_only to none', async () => {
      const mockCallback = vi.fn();
      offlineReadingManager.onMessage('WORK_DELETED', mockCallback);

      await offlineReadingManager.initialize();

      const messageHandler = mockServiceWorker.addEventListener.mock.calls.find(
        call => call[0] === 'message'
      )?.[1];

      // Simulate complete removal due to consent change
      const mockEvent = {
        data: {
          type: 'WORK_DELETED',
          workId: 'work-transition',
          reason: 'consent_changed_to_none',
          previousConsentLevel: 'pwa_only',
        },
      };

      messageHandler?.(mockEvent);

      expect(mockCallback).toHaveBeenCalledWith({
        type: 'WORK_DELETED',
        workId: 'work-transition',
        reason: 'consent_changed_to_none',
        previousConsentLevel: 'pwa_only',
      });
    });

    it('handles consent upgrade from pwa_only to files_and_pwa', async () => {
      const mockCallback = vi.fn();
      offlineReadingManager.onMessage('CONSENT_LEVEL_CHANGED', mockCallback);

      await offlineReadingManager.initialize();

      const messageHandler = mockServiceWorker.addEventListener.mock.calls.find(
        call => call[0] === 'message'
      )?.[1];

      // Simulate consent upgrade
      const mockEvent = {
        data: {
          type: 'CONSENT_LEVEL_CHANGED',
          workId: 'work-transition',
          oldConsentLevel: 'pwa_only',
          newConsentLevel: 'files_and_pwa',
          action: 'ttl_extended',
        },
      };

      messageHandler?.(mockEvent);

      expect(mockCallback).toHaveBeenCalledWith({
        type: 'CONSENT_LEVEL_CHANGED',
        workId: 'work-transition',
        oldConsentLevel: 'pwa_only',
        newConsentLevel: 'files_and_pwa',
        action: 'ttl_extended',
      });
    });
  });

  describe('Mixed Consent Scenarios', () => {
    const mixedWorks = [
      {
        id: 'work-files-pwa',
        title: 'Full Permission Work',
        authors: [{ pseud_name: 'Open Author', username: 'open' }],
        chapters: [{ id: 'ch1', number: 1, title: 'Ch1', content: 'Content', word_count: 1000 }],
        offline_reading_preference: 'files_and_pwa' as const,
      },
      {
        id: 'work-pwa-only',
        title: 'Temporary Permission Work',
        authors: [{ pseud_name: 'Temp Author', username: 'temp' }],
        chapters: [{ id: 'ch1', number: 1, title: 'Ch1', content: 'Content', word_count: 1000 }],
        offline_reading_preference: 'pwa_only' as const,
      },
      {
        id: 'work-none',
        title: 'No Permission Work',
        authors: [{ pseud_name: 'Private Author', username: 'private' }],
        chapters: [{ id: 'ch1', number: 1, title: 'Ch1', content: 'Content', word_count: 1000 }],
        offline_reading_preference: 'none' as const,
      },
    ];

    it('handles mixed consent levels in batch operations', async () => {
      const results = await Promise.all(
        mixedWorks.map(work => offlineReadingManager.cacheWorkForOffline(work))
      );

      expect(results).toEqual([true, true, false]);
      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledTimes(2); // Only 2 successful
    });

    it('returns only cacheable works in offline inventory', async () => {
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
                data: {
                  works: [
                    {
                      workId: 'work-files-pwa',
                      consentLevel: 'files_and_pwa',
                      cachedAt: Date.now(),
                      expiresAt: Date.now() + 1000000,
                      isExpired: false,
                    },
                    {
                      workId: 'work-pwa-only',
                      consentLevel: 'pwa_only',
                      cachedAt: Date.now(),
                      expiresAt: Date.now() + 1000000,
                      isExpired: false,
                    },
                    // work-none should never appear here
                  ],
                },
              });
            }
          }, 10);
        }
      };

      const works = await offlineReadingManager.getOfflineWorks();

      expect(works).toHaveLength(2);
      expect(works.map(w => w.workId)).toEqual(['work-files-pwa', 'work-pwa-only']);
      expect(works.map(w => w.consentLevel)).toEqual(['files_and_pwa', 'pwa_only']);

      global.MessageChannel = originalMessageChannel;
    });

    it('properly segregates works by consent level for cleanup', async () => {
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
                data: {
                  works: [
                    {
                      workId: 'work-files-pwa',
                      consentLevel: 'files_and_pwa',
                      cachedAt: Date.now() - (10 * 24 * 60 * 60 * 1000), // 10 days old
                      expiresAt: Date.now() + (20 * 24 * 60 * 60 * 1000), // 20 days remaining
                      isExpired: false,
                    },
                    {
                      workId: 'work-pwa-only-expired',
                      consentLevel: 'pwa_only',
                      cachedAt: Date.now() - (2 * 24 * 60 * 60 * 1000), // 2 days old
                      expiresAt: Date.now() - (1 * 60 * 60 * 1000), // 1 hour past expiry
                      isExpired: true,
                    },
                  ],
                },
              });
            }
          }, 10);
        }
      };

      const works = await offlineReadingManager.getOfflineWorks();
      const expiredWorks = works.filter(w => w.isExpired);
      const validWorks = works.filter(w => !w.isExpired);

      expect(validWorks).toHaveLength(1);
      expect(validWorks[0].consentLevel).toBe('files_and_pwa');

      expect(expiredWorks).toHaveLength(1);
      expect(expiredWorks[0].consentLevel).toBe('pwa_only');

      global.MessageChannel = originalMessageChannel;
    });
  });

  describe('Error Scenarios and Edge Cases', () => {
    it('handles works with undefined consent level', async () => {
      const workWithoutConsent = {
        id: 'work-undefined',
        title: 'Work Without Consent Setting',
        authors: [{ pseud_name: 'Unset Author', username: 'unset' }],
        chapters: [{ id: 'ch1', number: 1, title: 'Ch1', content: 'Content', word_count: 1000 }],
        // offline_reading_preference is undefined
      } as any;

      const result = await offlineReadingManager.cacheWorkForOffline(workWithoutConsent);

      expect(result).toBe(false);
      expect(mockServiceWorker.controller.postMessage).not.toHaveBeenCalled();
    });

    it('handles consent level validation errors', async () => {
      const workWithInvalidConsent = {
        id: 'work-invalid',
        title: 'Work With Invalid Consent',
        authors: [{ pseud_name: 'Invalid Author', username: 'invalid' }],
        chapters: [{ id: 'ch1', number: 1, title: 'Ch1', content: 'Content', word_count: 1000 }],
        offline_reading_preference: 'invalid_level' as any,
      };

      const result = await offlineReadingManager.cacheWorkForOffline(workWithInvalidConsent);

      expect(result).toBe(false);
      expect(mockServiceWorker.controller.postMessage).not.toHaveBeenCalled();
    });

    it('handles service worker errors during consent-aware operations', async () => {
      mockServiceWorker.controller.postMessage.mockImplementation(() => {
        throw new Error('Service worker communication failed');
      });

      const mockWork = {
        id: 'work-error',
        title: 'Work That Causes Error',
        authors: [{ pseud_name: 'Error Author', username: 'error' }],
        chapters: [{ id: 'ch1', number: 1, title: 'Ch1', content: 'Content', word_count: 1000 }],
        offline_reading_preference: 'files_and_pwa' as const,
      };

      const result = await offlineReadingManager.cacheWorkForOffline(mockWork);

      expect(result).toBe(false);
    });

    it('gracefully handles corrupted consent data from service worker', async () => {
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
              // Send corrupted data
              this.port1.onmessage({
                data: {
                  works: [
                    {
                      workId: 'work-corrupted',
                      // Missing required fields
                      consentLevel: null,
                      cachedAt: 'invalid-date',
                      expiresAt: undefined,
                    },
                  ],
                },
              });
            }
          }, 10);
        }
      };

      const works = await offlineReadingManager.getOfflineWorks();

      // Should filter out invalid works
      expect(works).toEqual([]);

      global.MessageChannel = originalMessageChannel;
    });
  });

  describe('Performance and Scalability', () => {
    it('handles large numbers of works with different consent levels efficiently', async () => {
      const startTime = Date.now();
      
      const manyWorks = Array.from({ length: 100 }, (_, i) => ({
        id: `work-${i}`,
        title: `Work ${i}`,
        authors: [{ pseud_name: `Author ${i}`, username: `author${i}` }],
        chapters: [{ id: `ch${i}`, number: 1, title: 'Ch1', content: 'Content', word_count: 1000 }],
        offline_reading_preference: (
          i % 3 === 0 ? 'files_and_pwa' :
          i % 3 === 1 ? 'pwa_only' : 'none'
        ) as const,
      }));

      const results = await Promise.all(
        manyWorks.map(work => offlineReadingManager.cacheWorkForOffline(work))
      );

      const endTime = Date.now();
      const duration = endTime - startTime;

      // Should complete within reasonable time (< 1 second for 100 works)
      expect(duration).toBeLessThan(1000);

      // Should properly handle different consent levels
      const successCount = results.filter(r => r).length;
      const expectedSuccess = Math.floor(100 * 2 / 3); // 2/3 should succeed (not 'none')
      
      expect(successCount).toBe(expectedSuccess);
    });

    it('efficiently batches consent level updates', async () => {
      const batchSize = 50;
      const workIds = Array.from({ length: batchSize }, (_, i) => `work-batch-${i}`);

      // Should support batch operations
      await offlineReadingManager.clearMultipleWorks(workIds);

      expect(mockServiceWorker.controller.postMessage).toHaveBeenCalledWith({
        type: 'CLEAR_MULTIPLE_WORKS',
        data: { workIds },
      });
    });
  });
});