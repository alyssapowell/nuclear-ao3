// Nuclear AO3 - Ethical Offline Reading Manager Utilities
// Provides utilities for managing respectful offline caching with author consent

export interface OfflineWork {
  workId: string;
  title: string;
  authors: Array<{ pseud_name: string; username: string }>;
  consentLevel: 'files_and_pwa' | 'pwa_only' | 'none';
  cachedAt: number;
  expiresAt: number;
  isExpired: boolean;
  word_count?: number;
  chapter_count?: number;
  summary?: string;
  tags?: {
    fandoms: string[];
    rating: string;
  };
}

export interface CacheStats {
  totalWorks: number;
  expiredWorks: number;
  storageUsed: string;
  storageQuota: string;
  utilizationPercentage: number;
}

export class OfflineReadingManager {
  private static instance: OfflineReadingManager;
  private serviceWorkerReady = false;
  private messageHandlers = new Map<string, Function>();

  private constructor() {
    this.initializeServiceWorker();
  }

  static getInstance(): OfflineReadingManager {
    if (!OfflineReadingManager.instance) {
      OfflineReadingManager.instance = new OfflineReadingManager();
    }
    return OfflineReadingManager.instance;
  }

  private async initializeServiceWorker(): Promise<void> {
    if (!('serviceWorker' in navigator)) {
      console.warn('[OfflineManager] Service Worker not supported');
      return;
    }

    try {
      await navigator.serviceWorker.ready;
      this.serviceWorkerReady = true;

      // Listen for service worker messages
      navigator.serviceWorker.addEventListener('message', this.handleServiceWorkerMessage.bind(this));

      console.log('[OfflineManager] Service Worker ready for ethical caching');
    } catch (error) {
      console.error('[OfflineManager] Service Worker initialization failed:', error);
    }
  }

  private handleServiceWorkerMessage(event: MessageEvent): void {
    const { type, workId, consentLevel } = event.data;
    
    const handler = this.messageHandlers.get(type);
    if (handler) {
      handler(event.data);
    }

    // Global message handling
    switch (type) {
      case 'WORK_CACHED_WITH_CONSENT':
        console.log('[OfflineManager] Work cached with consent:', workId, consentLevel);
        break;
      case 'WORK_DELETED':
        console.log('[OfflineManager] Work removed (author deletion):', workId);
        break;
      default:
        break;
    }
  }

  // Register message handler for specific event types
  onMessage(type: string, handler: Function): void {
    this.messageHandlers.set(type, handler);
  }

  // Remove message handler
  offMessage(type: string): void {
    this.messageHandlers.delete(type);
  }

  // Check if service worker is ready
  isReady(): boolean {
    return this.serviceWorkerReady && !!navigator.serviceWorker.controller;
  }

  // Get all offline works
  async getOfflineWorks(): Promise<OfflineWork[]> {
    if (!this.isReady()) {
      console.warn('[OfflineManager] Service Worker not ready');
      return [];
    }

    try {
      const messageChannel = new MessageChannel();
      
      const response = await new Promise<OfflineWork[]>((resolve, reject) => {
        const timeout = setTimeout(() => {
          reject(new Error('Service Worker communication timeout'));
        }, 5000);

        messageChannel.port1.onmessage = (event) => {
          clearTimeout(timeout);
          resolve(event.data.works || []);
        };

        navigator.serviceWorker.controller?.postMessage(
          { type: 'GET_OFFLINE_WORKS' },
          [messageChannel.port2]
        );
      });

      return response;
    } catch (error) {
      console.error('[OfflineManager] Failed to get offline works:', error);
      return [];
    }
  }

  // Cache work with author consent
  async cacheWork(workId: string, workData: any, consentLevel: 'files_and_pwa' | 'pwa_only'): Promise<boolean> {
    if (!this.isReady()) {
      console.warn('[OfflineManager] Service Worker not ready');
      return false;
    }

    if (consentLevel === 'none') {
      console.warn('[OfflineManager] Cannot cache work - author does not allow offline reading');
      return false;
    }

    try {
      navigator.serviceWorker.controller?.postMessage({
        type: 'CACHE_WORK_WITH_CONSENT',
        data: {
          workId,
          workData,
          consentLevel
        }
      });

      console.log('[OfflineManager] Caching request sent for work:', workId);
      return true;
    } catch (error) {
      console.error('[OfflineManager] Failed to cache work:', error);
      return false;
    }
  }

  // Remove work from cache
  async removeWork(workId: string): Promise<boolean> {
    if (!this.isReady()) {
      console.warn('[OfflineManager] Service Worker not ready');
      return false;
    }

    try {
      navigator.serviceWorker.controller?.postMessage({
        type: 'CLEAR_CONSENT_CACHE',
        data: { workId }
      });

      console.log('[OfflineManager] Work removal request sent:', workId);
      return true;
    } catch (error) {
      console.error('[OfflineManager] Failed to remove work:', error);
      return false;
    }
  }

  // Update work consent level
  async updateWorkConsent(workId: string, newConsentLevel: 'files_and_pwa' | 'pwa_only' | 'none'): Promise<boolean> {
    if (!this.isReady()) {
      console.warn('[OfflineManager] Service Worker not ready');
      return false;
    }

    try {
      navigator.serviceWorker.controller?.postMessage({
        type: 'UPDATE_CONSENT_LEVEL',
        data: {
          workId,
          consentLevel: newConsentLevel
        }
      });

      console.log('[OfflineManager] Consent level update sent:', workId, newConsentLevel);
      return true;
    } catch (error) {
      console.error('[OfflineManager] Failed to update consent level:', error);
      return false;
    }
  }

  // Get cache statistics
  async getCacheStats(): Promise<CacheStats> {
    const defaultStats: CacheStats = {
      totalWorks: 0,
      expiredWorks: 0,
      storageUsed: '0 MB',
      storageQuota: 'Unknown',
      utilizationPercentage: 0
    };

    try {
      // Get offline works count
      const works = await this.getOfflineWorks();
      const expiredCount = works.filter(w => w.isExpired).length;

      // Get storage estimate
      let storageInfo = defaultStats;
      if ('storage' in navigator && 'estimate' in navigator.storage) {
        const estimate = await navigator.storage.estimate();
        if (estimate.usage && estimate.quota) {
          const usedMB = (estimate.usage / (1024 * 1024)).toFixed(1);
          const quotaMB = (estimate.quota / (1024 * 1024)).toFixed(0);
          const percentage = Math.round((estimate.usage / estimate.quota) * 100);

          storageInfo = {
            totalWorks: works.length,
            expiredWorks: expiredCount,
            storageUsed: `${usedMB} MB`,
            storageQuota: `${quotaMB} MB`,
            utilizationPercentage: percentage
          };
        }
      }

      return {
        ...storageInfo,
        totalWorks: works.length,
        expiredWorks: expiredCount
      };
    } catch (error) {
      console.error('[OfflineManager] Failed to get cache stats:', error);
      return defaultStats;
    }
  }

  // Clear expired works
  async clearExpiredWorks(): Promise<number> {
    try {
      const works = await this.getOfflineWorks();
      const expiredWorks = works.filter(w => w.isExpired);

      let clearedCount = 0;
      for (const work of expiredWorks) {
        const success = await this.removeWork(work.workId);
        if (success) clearedCount++;
      }

      console.log('[OfflineManager] Cleared expired works:', clearedCount);
      return clearedCount;
    } catch (error) {
      console.error('[OfflineManager] Failed to clear expired works:', error);
      return 0;
    }
  }

  // Clear all cached works
  async clearAllWorks(): Promise<boolean> {
    try {
      const works = await this.getOfflineWorks();

      for (const work of works) {
        await this.removeWork(work.workId);
      }

      console.log('[OfflineManager] Cleared all cached works');
      return true;
    } catch (error) {
      console.error('[OfflineManager] Failed to clear all works:', error);
      return false;
    }
  }

  // Check if specific work is cached
  async isWorkCached(workId: string): Promise<boolean> {
    try {
      const works = await this.getOfflineWorks();
      return works.some(w => w.workId === workId && !w.isExpired);
    } catch (error) {
      console.error('[OfflineManager] Failed to check work cache status:', error);
      return false;
    }
  }

  // Get work cache status
  async getWorkCacheStatus(workId: string): Promise<OfflineWork | null> {
    try {
      const works = await this.getOfflineWorks();
      return works.find(w => w.workId === workId) || null;
    } catch (error) {
      console.error('[OfflineManager] Failed to get work cache status:', error);
      return null;
    }
  }

  // Format time remaining for expiry
  formatTimeRemaining(expiresAt: number): string {
    const now = Date.now();
    const remaining = expiresAt - now;

    if (remaining <= 0) return 'Expired';

    const days = Math.floor(remaining / (1000 * 60 * 60 * 24));
    const hours = Math.floor((remaining % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));

    if (days > 7) return `${days} days`;
    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h ${minutes}m`;
    return `${minutes}m`;
  }

  // Get consent level description
  getConsentDescription(level: string): string {
    switch (level) {
      case 'files_and_pwa':
        return 'Full offline access - works can be cached for extended periods';
      case 'pwa_only':
        return 'PWA caching only - temporary offline access for convenience';
      case 'none':
        return 'Online only - author prefers no offline caching';
      default:
        return 'Unknown consent level';
    }
  }

  // Get connection status
  isOnline(): boolean {
    return navigator.onLine;
  }

  // Listen for online/offline events
  onConnectionChange(callback: (isOnline: boolean) => void): () => void {
    const handleOnline = () => callback(true);
    const handleOffline = () => callback(false);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    // Return cleanup function
    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }
}

// Export singleton instance
export const offlineReadingManager = OfflineReadingManager.getInstance();

// Convenience hooks for React components
export const useOfflineReading = () => {
  return offlineReadingManager;
};

// Local storage utilities for work metadata
export const workMetadataStorage = {
  set: (workId: string, metadata: Partial<OfflineWork>): void => {
    try {
      localStorage.setItem(`work-metadata-${workId}`, JSON.stringify(metadata));
    } catch (error) {
      console.error('[OfflineManager] Failed to save work metadata:', error);
    }
  },

  get: (workId: string): Partial<OfflineWork> | null => {
    try {
      const data = localStorage.getItem(`work-metadata-${workId}`);
      return data ? JSON.parse(data) : null;
    } catch (error) {
      console.error('[OfflineManager] Failed to get work metadata:', error);
      return null;
    }
  },

  remove: (workId: string): void => {
    try {
      localStorage.removeItem(`work-metadata-${workId}`);
    } catch (error) {
      console.error('[OfflineManager] Failed to remove work metadata:', error);
    }
  }
};

// Reading settings utilities
export const readingSettingsStorage = {
  get: () => {
    try {
      const settings = localStorage.getItem('nuclear-ao3-reading-settings');
      return settings ? JSON.parse(settings) : null;
    } catch (error) {
      console.error('[OfflineManager] Failed to get reading settings:', error);
      return null;
    }
  },

  set: (settings: any) => {
    try {
      localStorage.setItem('nuclear-ao3-reading-settings', JSON.stringify(settings));
    } catch (error) {
      console.error('[OfflineManager] Failed to save reading settings:', error);
    }
  }
};

export default offlineReadingManager;