import { OfflineReadingManager } from '../../utils/offlineReadingManager';
import { 
  setupGlobalMocks, 
  resetMocks, 
  mockOfflineWork, 
  mockWorkData,
  simulateServiceWorkerMessage,
  simulateStorageQuota,
  simulateNetworkStatus,
  waitFor
} from '../test-utils';

describe('OfflineReadingManager', () => {
  let manager: OfflineReadingManager;
  let mockStorage: any;
  let localStorageMock: any;

  beforeEach(() => {
    const mocks = setupGlobalMocks();
    mockStorage = mocks.mockStorage;
    localStorageMock = mocks.localStorageMock;
    
    // Reset singleton instance
    (OfflineReadingManager as any).instance = undefined;
    manager = OfflineReadingManager.getInstance();
  });

  afterEach(() => {
    resetMocks();
  });

  describe('Singleton Pattern', () => {
    it('should return the same instance', () => {
      const manager1 = OfflineReadingManager.getInstance();
      const manager2 = OfflineReadingManager.getInstance();
      
      expect(manager1).toBe(manager2);
    });

    it('should initialize only once', () => {
      const manager1 = OfflineReadingManager.getInstance();
      const manager2 = OfflineReadingManager.getInstance();
      
      // Both should be the same instance
      expect(manager1).toBe(manager2);
    });
  });

  describe('Service Worker Communication', () => {
    it('should detect when service worker is ready', async () => {
      await waitFor(100); // Allow initialization
      expect(manager.isReady()).toBe(true);
    });

    it('should handle service worker not available', () => {
      // Remove service worker support
      delete (global.navigator as any).serviceWorker;
      
      const manager = OfflineReadingManager.getInstance();
      expect(manager.isReady()).toBe(false);
    });

    it('should register message handlers', () => {
      const handler = jest.fn();
      manager.onMessage('TEST_MESSAGE', handler);
      
      simulateServiceWorkerMessage('TEST_MESSAGE', { data: 'test' });
      
      expect(handler).toHaveBeenCalledWith({ type: 'TEST_MESSAGE', data: 'test' });
    });

    it('should remove message handlers', () => {
      const handler = jest.fn();
      manager.onMessage('TEST_MESSAGE', handler);
      manager.offMessage('TEST_MESSAGE');
      
      simulateServiceWorkerMessage('TEST_MESSAGE', { data: 'test' });
      
      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe('Offline Works Management', () => {
    beforeEach(() => {
      // Mock MessageChannel communication
      const originalMessageChannel = global.MessageChannel;
      global.MessageChannel = jest.fn().mockImplementation(() => ({
        port1: {
          onmessage: null,
          postMessage: jest.fn(),
          close: jest.fn()
        },
        port2: {
          onmessage: null,
          postMessage: jest.fn(),
          close: jest.fn()
        }
      }));
    });

    it('should get offline works', async () => {
      const mockWorks = [mockOfflineWork];
      
      // Mock the message channel response
      const messageChannel = new (global.MessageChannel as any)();
      setTimeout(() => {
        if (messageChannel.port1.onmessage) {
          messageChannel.port1.onmessage(new MessageEvent('message', {
            data: { works: mockWorks }
          }));
        }
      }, 10);

      const works = await manager.getOfflineWorks();
      expect(works).toEqual(mockWorks);
    });

    it('should handle empty offline works', async () => {
      const messageChannel = new (global.MessageChannel as any)();
      setTimeout(() => {
        if (messageChannel.port1.onmessage) {
          messageChannel.port1.onmessage(new MessageEvent('message', {
            data: { works: [] }
          }));
        }
      }, 10);

      const works = await manager.getOfflineWorks();
      expect(works).toEqual([]);
    });

    it('should handle service worker communication timeout', async () => {
      // Don't respond to the message channel
      const works = await manager.getOfflineWorks();
      expect(works).toEqual([]);
    });

    it('should cache work with consent', async () => {
      const result = await manager.cacheWork(
        mockWorkData.id, 
        mockWorkData, 
        'files_and_pwa'
      );
      
      expect(result).toBe(true);
    });

    it('should not cache work with no consent', async () => {
      const result = await manager.cacheWork(
        mockWorkData.id, 
        mockWorkData, 
        'none' as any
      );
      
      expect(result).toBe(false);
    });

    it('should remove work from cache', async () => {
      const result = await manager.removeWork(mockWorkData.id);
      expect(result).toBe(true);
    });

    it('should update work consent level', async () => {
      const result = await manager.updateWorkConsent(
        mockWorkData.id, 
        'pwa_only'
      );
      
      expect(result).toBe(true);
    });
  });

  describe('Cache Statistics', () => {
    it('should get cache statistics', async () => {
      simulateStorageQuota(1000000000, 50000000); // 1GB quota, 50MB used
      
      const messageChannel = new (global.MessageChannel as any)();
      setTimeout(() => {
        if (messageChannel.port1.onmessage) {
          messageChannel.port1.onmessage(new MessageEvent('message', {
            data: { works: [mockOfflineWork] }
          }));
        }
      }, 10);

      const stats = await manager.getCacheStats();
      
      expect(stats.totalWorks).toBe(1);
      expect(stats.expiredWorks).toBe(0);
      expect(stats.storageUsed).toBe('48.8 MB');
      expect(stats.storageQuota).toBe('954 MB');
      expect(stats.utilizationPercentage).toBe(5);
    });

    it('should handle storage estimation unavailable', async () => {
      // Mock storage not supported
      delete (global.navigator as any).storage;
      
      const stats = await manager.getCacheStats();
      
      expect(stats.storageUsed).toBe('0 MB');
      expect(stats.storageQuota).toBe('Unknown');
      expect(stats.utilizationPercentage).toBe(0);
    });
  });

  describe('Cache Management Operations', () => {
    beforeEach(() => {
      // Mock works with some expired
      const expiredWork = {
        ...mockOfflineWork,
        workId: 'expired-work',
        isExpired: true
      };
      
      const messageChannel = new (global.MessageChannel as any)();
      setTimeout(() => {
        if (messageChannel.port1.onmessage) {
          messageChannel.port1.onmessage(new MessageEvent('message', {
            data: { works: [mockOfflineWork, expiredWork] }
          }));
        }
      }, 10);
    });

    it('should clear expired works', async () => {
      const clearedCount = await manager.clearExpiredWorks();
      expect(clearedCount).toBe(1);
    });

    it('should clear all works', async () => {
      const result = await manager.clearAllWorks();
      expect(result).toBe(true);
    });

    it('should check if specific work is cached', async () => {
      const isCached = await manager.isWorkCached(mockWorkData.id);
      expect(typeof isCached).toBe('boolean');
    });

    it('should get work cache status', async () => {
      const status = await manager.getWorkCacheStatus(mockWorkData.id);
      expect(status).toEqual(mockOfflineWork);
    });
  });

  describe('Utility Functions', () => {
    it('should format time remaining correctly', () => {
      const now = Date.now();
      
      // Test expired
      expect(manager.formatTimeRemaining(now - 1000)).toBe('Expired');
      
      // Test minutes
      expect(manager.formatTimeRemaining(now + 30 * 60 * 1000)).toBe('30m');
      
      // Test hours and minutes
      expect(manager.formatTimeRemaining(now + 2 * 60 * 60 * 1000 + 30 * 60 * 1000)).toBe('2h 30m');
      
      // Test days and hours
      expect(manager.formatTimeRemaining(now + 3 * 24 * 60 * 60 * 1000 + 2 * 60 * 60 * 1000)).toBe('3d 2h');
      
      // Test many days
      expect(manager.formatTimeRemaining(now + 10 * 24 * 60 * 60 * 1000)).toBe('10 days');
    });

    it('should get consent level descriptions', () => {
      expect(manager.getConsentDescription('files_and_pwa'))
        .toBe('Full offline access - works can be cached for extended periods');
      
      expect(manager.getConsentDescription('pwa_only'))
        .toBe('PWA caching only - temporary offline access for convenience');
      
      expect(manager.getConsentDescription('none'))
        .toBe('Online only - author prefers no offline caching');
      
      expect(manager.getConsentDescription('unknown'))
        .toBe('Unknown consent level');
    });

    it('should detect online status', () => {
      simulateNetworkStatus(true);
      expect(manager.isOnline()).toBe(true);
      
      simulateNetworkStatus(false);
      expect(manager.isOnline()).toBe(false);
    });

    it('should handle connection change events', () => {
      const callback = jest.fn();
      const cleanup = manager.onConnectionChange(callback);
      
      simulateNetworkStatus(false);
      expect(callback).toHaveBeenCalledWith(false);
      
      simulateNetworkStatus(true);
      expect(callback).toHaveBeenCalledWith(true);
      
      // Test cleanup
      cleanup();
      simulateNetworkStatus(false);
      expect(callback).toHaveBeenCalledTimes(2); // Should not be called again
    });
  });

  describe('Local Storage Integration', () => {
    it('should save work metadata', () => {
      const { workMetadataStorage } = require('../../utils/offlineReadingManager');
      
      const metadata = { title: 'Test Work', authors: [] };
      workMetadataStorage.set('test-id', metadata);
      
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'work-metadata-test-id',
        JSON.stringify(metadata)
      );
    });

    it('should get work metadata', () => {
      const { workMetadataStorage } = require('../../utils/offlineReadingManager');
      
      const metadata = { title: 'Test Work', authors: [] };
      localStorageMock.getItem.mockReturnValue(JSON.stringify(metadata));
      
      const result = workMetadataStorage.get('test-id');
      expect(result).toEqual(metadata);
    });

    it('should handle missing work metadata', () => {
      const { workMetadataStorage } = require('../../utils/offlineReadingManager');
      
      localStorageMock.getItem.mockReturnValue(null);
      
      const result = workMetadataStorage.get('test-id');
      expect(result).toBe(null);
    });

    it('should remove work metadata', () => {
      const { workMetadataStorage } = require('../../utils/offlineReadingManager');
      
      workMetadataStorage.remove('test-id');
      
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('work-metadata-test-id');
    });

    it('should handle localStorage errors gracefully', () => {
      const { workMetadataStorage } = require('../../utils/offlineReadingManager');
      
      localStorageMock.setItem.mockImplementation(() => {
        throw new Error('Storage quota exceeded');
      });
      
      // Should not throw
      expect(() => {
        workMetadataStorage.set('test-id', { title: 'Test' });
      }).not.toThrow();
    });
  });

  describe('Reading Settings Storage', () => {
    it('should save reading settings', () => {
      const { readingSettingsStorage } = require('../../utils/offlineReadingManager');
      
      const settings = { fontSize: 'large', theme: 'dark' };
      readingSettingsStorage.set(settings);
      
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'nuclear-ao3-reading-settings',
        JSON.stringify(settings)
      );
    });

    it('should get reading settings', () => {
      const { readingSettingsStorage } = require('../../utils/offlineReadingManager');
      
      const settings = { fontSize: 'large', theme: 'dark' };
      localStorageMock.getItem.mockReturnValue(JSON.stringify(settings));
      
      const result = readingSettingsStorage.get();
      expect(result).toEqual(settings);
    });

    it('should handle missing reading settings', () => {
      const { readingSettingsStorage } = require('../../utils/offlineReadingManager');
      
      localStorageMock.getItem.mockReturnValue(null);
      
      const result = readingSettingsStorage.get();
      expect(result).toBe(null);
    });
  });

  describe('Error Handling', () => {
    it('should handle service worker communication failures', async () => {
      // Mock service worker controller as null
      const originalController = global.navigator.serviceWorker.controller;
      global.navigator.serviceWorker.controller = null;
      
      const result = await manager.cacheWork('test-id', mockWorkData, 'files_and_pwa');
      expect(result).toBe(false);
      
      // Restore
      global.navigator.serviceWorker.controller = originalController;
    });

    it('should handle storage estimation failures', async () => {
      mockStorage.estimate.mockRejectedValue(new Error('Storage error'));
      
      const stats = await manager.getCacheStats();
      expect(stats.storageUsed).toBe('0 MB');
    });

    it('should handle JSON parsing errors in localStorage', () => {
      const { workMetadataStorage } = require('../../utils/offlineReadingManager');
      
      localStorageMock.getItem.mockReturnValue('invalid json');
      
      const result = workMetadataStorage.get('test-id');
      expect(result).toBe(null);
    });
  });
});

describe('useOfflineReading hook', () => {
  it('should return the singleton instance', () => {
    setupGlobalMocks();
    
    const { useOfflineReading } = require('../../utils/offlineReadingManager');
    const manager = useOfflineReading();
    
    expect(manager).toBeInstanceOf(OfflineReadingManager);
  });
});