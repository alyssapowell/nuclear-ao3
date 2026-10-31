import { OfflineReadingManager, workMetadataStorage, readingSettingsStorage, useOfflineReading } from '../../utils/offlineReadingManager';
import { 
  setupGlobalMocks, 
  resetMocks, 
  mockOfflineWork, 
  mockWorkData,
  simulateServiceWorkerMessage,
  simulateStorageQuota,
  simulateNetworkStatus,
  respondWithOfflineWorks,
  mockServiceWorkerRegistration,
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
    
    // By default the service worker has no offline works to report.
    respondWithOfflineWorks([]);

    // Reset singleton instance
    (OfflineReadingManager as any).instance = undefined;
    manager = OfflineReadingManager.getInstance();
  });

  afterEach(() => {
    vi.useRealTimers();
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
      // Remove service worker support, then start a fresh manager
      delete (global.navigator as any).serviceWorker;
      (OfflineReadingManager as any).instance = undefined;

      const manager = OfflineReadingManager.getInstance();
      expect(manager.isReady()).toBe(false);
    });

    it('should register message handlers', () => {
      const handler = vi.fn();
      manager.onMessage('TEST_MESSAGE', handler);
      
      simulateServiceWorkerMessage('TEST_MESSAGE', { data: 'test' });
      
      expect(handler).toHaveBeenCalledWith({ type: 'TEST_MESSAGE', data: 'test' });
    });

    it('should remove message handlers', () => {
      const handler = vi.fn();
      manager.onMessage('TEST_MESSAGE', handler);
      manager.offMessage('TEST_MESSAGE');
      
      simulateServiceWorkerMessage('TEST_MESSAGE', { data: 'test' });
      
      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe('Offline Works Management', () => {
    beforeEach(async () => {
      await waitFor(0); // let the manager finish connecting to the service worker
    });

    it('should get offline works', async () => {
      respondWithOfflineWorks([mockOfflineWork]);

      const works = await manager.getOfflineWorks();
      expect(works).toEqual([mockOfflineWork]);
    });

    it('should handle empty offline works', async () => {
      const works = await manager.getOfflineWorks();
      expect(works).toEqual([]);
    });

    it('should handle service worker communication timeout', async () => {
      // The service worker never answers; the manager gives up after 5 seconds.
      respondWithOfflineWorks();
      vi.useFakeTimers();

      const pending = manager.getOfflineWorks();
      await vi.advanceTimersByTimeAsync(5000);

      expect(await pending).toEqual([]);
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
      expect(mockServiceWorkerRegistration.active.postMessage).not.toHaveBeenCalled();
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
      await waitFor(0);
      simulateStorageQuota(1000000000, 50000000); // 1GB quota, 50MB used
      respondWithOfflineWorks([mockOfflineWork]);

      const stats = await manager.getCacheStats();
      
      expect(stats.totalWorks).toBe(1);
      expect(stats.expiredWorks).toBe(0);
      expect(stats.storageUsed).toBe('47.7 MB');
      expect(stats.storageQuota).toBe('954 MB');
      expect(stats.utilizationPercentage).toBe(5);
    });

    it('should handle storage estimation unavailable', async () => {
      await waitFor(0);
      // Mock storage not supported
      delete (global.navigator as any).storage;
      
      const stats = await manager.getCacheStats();
      
      expect(stats.storageUsed).toBe('0 MB');
      expect(stats.storageQuota).toBe('Unknown');
      expect(stats.utilizationPercentage).toBe(0);
    });
  });

  describe('Cache Management Operations', () => {
    beforeEach(async () => {
      await waitFor(0);
      // Two cached works, one of them expired
      const expiredWork = {
        ...mockOfflineWork,
        workId: 'expired-work',
        isExpired: true
      };
      respondWithOfflineWorks([mockOfflineWork, expiredWork]);
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
      expect(await manager.isWorkCached(mockWorkData.id)).toBe(true);
      expect(await manager.isWorkCached('expired-work')).toBe(false);
      expect(await manager.isWorkCached('never-cached')).toBe(false);
    });

    it('should get work cache status', async () => {
      const status = await manager.getWorkCacheStatus(mockWorkData.id);
      expect(status).toEqual(mockOfflineWork);
    });
  });

  describe('Utility Functions', () => {
    it('should format time remaining correctly', () => {
      // Freeze the clock so the remaining time is exact.
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2024-06-01T12:00:00Z'));
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
      const callback = vi.fn();
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
      
      const metadata = { title: 'Test Work', authors: [] };
      workMetadataStorage.set('test-id', metadata);
      
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'work-metadata-test-id',
        JSON.stringify(metadata)
      );
    });

    it('should get work metadata', () => {
      
      const metadata = { title: 'Test Work', authors: [] };
      localStorageMock.getItem.mockReturnValue(JSON.stringify(metadata));
      
      const result = workMetadataStorage.get('test-id');
      expect(result).toEqual(metadata);
    });

    it('should handle missing work metadata', () => {
      
      localStorageMock.getItem.mockReturnValue(null);
      
      const result = workMetadataStorage.get('test-id');
      expect(result).toBe(null);
    });

    it('should remove work metadata', () => {
      
      workMetadataStorage.remove('test-id');
      
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('work-metadata-test-id');
    });

    it('should handle localStorage errors gracefully', () => {
      
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
      
      const settings = { fontSize: 'large', theme: 'dark' };
      readingSettingsStorage.set(settings);
      
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'nuclear-ao3-reading-settings',
        JSON.stringify(settings)
      );
    });

    it('should get reading settings', () => {
      
      const settings = { fontSize: 'large', theme: 'dark' };
      localStorageMock.getItem.mockReturnValue(JSON.stringify(settings));
      
      const result = readingSettingsStorage.get();
      expect(result).toEqual(settings);
    });

    it('should handle missing reading settings', () => {
      
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
      await waitFor(0);
      mockStorage.estimate.mockRejectedValue(new Error('Storage error'));
      
      const stats = await manager.getCacheStats();
      expect(stats.storageUsed).toBe('0 MB');
    });

    it('should handle JSON parsing errors in localStorage', () => {
      
      localStorageMock.getItem.mockReturnValue('invalid json');
      
      const result = workMetadataStorage.get('test-id');
      expect(result).toBe(null);
    });
  });
});

describe('useOfflineReading hook', () => {
  it('should return the singleton instance', () => {
    setupGlobalMocks();
    
    const manager = useOfflineReading();
    
    expect(manager).toBeInstanceOf(OfflineReadingManager);
  });
});