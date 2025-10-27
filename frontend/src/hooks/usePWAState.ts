'use client';

import { useState, useEffect, useCallback } from 'react';
import { offlineReadingManager } from '../utils/offlineReadingManager';

export interface PWAState {
  isOnline: boolean;
  isInstalled: boolean;
  isInstallable: boolean;
  installPrompt: any | null;
  serviceWorkerReady: boolean;
  updateAvailable: boolean;
}

export interface PWAActions {
  installPWA: () => Promise<boolean>;
  checkForUpdates: () => Promise<void>;
  skipWaiting: () => void;
  dismissInstallPrompt: () => void;
}

export function usePWAState(): [PWAState, PWAActions] {
  const [state, setState] = useState<PWAState>({
    isOnline: typeof navigator !== 'undefined' ? navigator.onLine : true,
    isInstalled: false,
    isInstallable: false,
    installPrompt: null,
    serviceWorkerReady: false,
    updateAvailable: false
  });

  // Check if PWA is installed
  const checkInstallStatus = useCallback(() => {
    if (typeof window === 'undefined') return;

    // Check if running in standalone mode (iOS)
    const isIOSInstalled = window.matchMedia('(display-mode: standalone)').matches;
    
    // Check if running as PWA (Android/Desktop)
    const isPWAInstalled = window.matchMedia('(display-mode: standalone)').matches ||
                           (window.navigator as any).standalone === true ||
                           document.referrer.includes('android-app://');

    setState(prev => ({
      ...prev,
      isInstalled: isIOSInstalled || isPWAInstalled
    }));
  }, []);

  // Check service worker status
  const checkServiceWorkerStatus = useCallback(async () => {
    if (!('serviceWorker' in navigator)) {
      setState(prev => ({ ...prev, serviceWorkerReady: false }));
      return;
    }

    try {
      const registration = await navigator.serviceWorker.ready;
      setState(prev => ({ ...prev, serviceWorkerReady: !!registration.active }));

      // Check for updates
      if (registration.waiting) {
        setState(prev => ({ ...prev, updateAvailable: true }));
      }

      // Listen for new service worker installing
      registration.addEventListener('updatefound', () => {
        const newWorker = registration.installing;
        if (newWorker) {
          newWorker.addEventListener('statechange', () => {
            if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
              setState(prev => ({ ...prev, updateAvailable: true }));
            }
          });
        }
      });

    } catch (error) {
      console.error('[PWA] Service worker check failed:', error);
      setState(prev => ({ ...prev, serviceWorkerReady: false }));
    }
  }, []);

  // Initialize PWA state
  useEffect(() => {
    checkInstallStatus();
    checkServiceWorkerStatus();

    // Listen for online/offline events
    const cleanup = offlineReadingManager.onConnectionChange((isOnline) => {
      setState(prev => ({ ...prev, isOnline }));
    });

    // Listen for display mode changes
    const mediaQuery = window.matchMedia('(display-mode: standalone)');
    const handleDisplayModeChange = () => checkInstallStatus();
    mediaQuery.addEventListener('change', handleDisplayModeChange);

    // Listen for beforeinstallprompt (install prompt)
    const handleBeforeInstallPrompt = (e: Event) => {
      e.preventDefault();
      setState(prev => ({
        ...prev,
        isInstallable: true,
        installPrompt: e
      }));
    };

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt);

    // Listen for app installed
    const handleAppInstalled = () => {
      setState(prev => ({
        ...prev,
        isInstalled: true,
        isInstallable: false,
        installPrompt: null
      }));
    };

    window.addEventListener('appinstalled', handleAppInstalled);

    // Cleanup
    return () => {
      cleanup();
      mediaQuery.removeEventListener('change', handleDisplayModeChange);
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
      window.removeEventListener('appinstalled', handleAppInstalled);
    };
  }, [checkInstallStatus, checkServiceWorkerStatus]);

  // Actions
  const installPWA = useCallback(async (): Promise<boolean> => {
    if (!state.installPrompt) {
      console.warn('[PWA] No install prompt available');
      return false;
    }

    try {
      const result = await state.installPrompt.prompt();
      const accepted = result.outcome === 'accepted';

      setState(prev => ({
        ...prev,
        isInstallable: false,
        installPrompt: null,
        isInstalled: accepted
      }));

      return accepted;
    } catch (error) {
      console.error('[PWA] Install failed:', error);
      return false;
    }
  }, [state.installPrompt]);

  const checkForUpdates = useCallback(async (): Promise<void> => {
    if (!('serviceWorker' in navigator)) return;

    try {
      const registration = await navigator.serviceWorker.getRegistration();
      if (registration) {
        await registration.update();
      }
    } catch (error) {
      console.error('[PWA] Update check failed:', error);
    }
  }, []);

  const skipWaiting = useCallback(() => {
    if (!('serviceWorker' in navigator)) return;

    navigator.serviceWorker.controller?.postMessage({ type: 'SKIP_WAITING' });
    setState(prev => ({ ...prev, updateAvailable: false }));
    
    // Reload to use new service worker
    window.location.reload();
  }, []);

  const dismissInstallPrompt = useCallback(() => {
    setState(prev => ({
      ...prev,
      isInstallable: false,
      installPrompt: null
    }));
  }, []);

  const actions: PWAActions = {
    installPWA,
    checkForUpdates,
    skipWaiting,
    dismissInstallPrompt
  };

  return [state, actions];
}

export default usePWAState;