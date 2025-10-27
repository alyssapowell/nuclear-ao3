import { renderHook, act } from '@testing-library/react';
import { usePWAState } from '../../hooks/usePWAState';
import { 
  setupGlobalMocks, 
  resetMocks, 
  mockInstallPromptEvent,
  mockServiceWorkerRegistration,
  waitFor
} from '../test-utils';

// Mock the offline reading manager
jest.mock('../../utils/offlineReadingManager', () => ({
  offlineReadingManager: {
    onConnectionChange: jest.fn().mockReturnValue(() => {}),
    getInstance: jest.fn().mockReturnValue({
      onConnectionChange: jest.fn().mockReturnValue(() => {})
    })
  }
}));

describe('usePWAState', () => {
  let mockMatchMedia: jest.Mock;
  let originalAddEventListener: typeof window.addEventListener;
  let originalRemoveEventListener: typeof window.removeEventListener;

  beforeEach(() => {
    setupGlobalMocks();
    
    // Mock matchMedia
    mockMatchMedia = jest.fn().mockReturnValue({
      matches: false,
      addListener: jest.fn(),
      removeListener: jest.fn(),
      addEventListener: jest.fn(),
      removeEventListener: jest.fn()
    });
    Object.defineProperty(window, 'matchMedia', {
      value: mockMatchMedia,
      writable: true
    });

    // Store original event listeners
    originalAddEventListener = window.addEventListener;
    originalRemoveEventListener = window.removeEventListener;

    // Mock window event listeners
    window.addEventListener = jest.fn();
    window.removeEventListener = jest.fn();
  });

  afterEach(() => {
    resetMocks();
    // Restore original event listeners
    window.addEventListener = originalAddEventListener;
    window.removeEventListener = originalRemoveEventListener;
  });

  describe('Initial State', () => {
    it('should initialize with correct default state', () => {
      const { result } = renderHook(() => usePWAState());
      const [state] = result.current;

      expect(state.isOnline).toBe(true);
      expect(state.isInstalled).toBe(false);
      expect(state.isInstallable).toBe(false);
      expect(state.installPrompt).toBe(null);
      expect(state.serviceWorkerReady).toBe(false);
      expect(state.updateAvailable).toBe(false);
    });

    it('should detect service worker readiness', async () => {
      const { result } = renderHook(() => usePWAState());
      
      // Wait for service worker check
      await waitFor(100);
      
      const [state] = result.current;
      expect(state.serviceWorkerReady).toBe(true);
    });

    it('should detect if PWA is installed via standalone mode', () => {
      // Mock standalone mode
      mockMatchMedia.mockReturnValue({
        matches: true,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn()
      });

      const { result } = renderHook(() => usePWAState());
      const [state] = result.current;

      expect(state.isInstalled).toBe(true);
    });

    it('should detect if PWA is installed via navigator.standalone', () => {
      // Mock iOS standalone
      Object.defineProperty(window.navigator, 'standalone', {
        value: true,
        writable: true
      });

      const { result } = renderHook(() => usePWAState());
      const [state] = result.current;

      expect(state.isInstalled).toBe(true);
    });
  });

  describe('Install Prompt Handling', () => {
    it('should handle beforeinstallprompt event', async () => {
      const { result } = renderHook(() => usePWAState());
      
      // Simulate beforeinstallprompt event
      act(() => {
        const event = new Event('beforeinstallprompt');
        Object.defineProperty(event, 'preventDefault', {
          value: jest.fn(),
          writable: true
        });
        
        // Get the event listener that was registered
        const addEventListenerCalls = (window.addEventListener as jest.Mock).mock.calls;
        const beforeInstallPromptListener = addEventListenerCalls
          .find(call => call[0] === 'beforeinstallprompt')?.[1];
        
        if (beforeInstallPromptListener) {
          beforeInstallPromptListener(event);
        }
      });

      const [state] = result.current;
      expect(state.isInstallable).toBe(true);
      expect(state.installPrompt).toBeTruthy();
    });

    it('should handle app installed event', async () => {
      const { result } = renderHook(() => usePWAState());
      
      // First set installable state
      act(() => {
        const beforeInstallEvent = new Event('beforeinstallprompt');
        Object.defineProperty(beforeInstallEvent, 'preventDefault', {
          value: jest.fn(),
          writable: true
        });
        
        const addEventListenerCalls = (window.addEventListener as jest.Mock).mock.calls;
        const beforeInstallPromptListener = addEventListenerCalls
          .find(call => call[0] === 'beforeinstallprompt')?.[1];
        
        if (beforeInstallPromptListener) {
          beforeInstallPromptListener(beforeInstallEvent);
        }
      });

      // Then simulate app installed
      act(() => {
        const installedEvent = new Event('appinstalled');
        
        const addEventListenerCalls = (window.addEventListener as jest.Mock).mock.calls;
        const appInstalledListener = addEventListenerCalls
          .find(call => call[0] === 'appinstalled')?.[1];
        
        if (appInstalledListener) {
          appInstalledListener(installedEvent);
        }
      });

      const [state] = result.current;
      expect(state.isInstalled).toBe(true);
      expect(state.isInstallable).toBe(false);
      expect(state.installPrompt).toBe(null);
    });
  });

  describe('Service Worker Updates', () => {
    it('should detect service worker updates', async () => {
      // Mock registration with waiting worker
      const mockRegistrationWithWaiting = {
        ...mockServiceWorkerRegistration,
        waiting: {
          state: 'installed',
          addEventListener: jest.fn()
        }
      };

      // Replace the ready promise
      Object.defineProperty(global.navigator.serviceWorker, 'ready', {
        value: Promise.resolve(mockRegistrationWithWaiting),
        writable: true
      });

      const { result } = renderHook(() => usePWAState());
      
      await waitFor(100);
      
      const [state] = result.current;
      expect(state.updateAvailable).toBe(true);
    });

    it('should detect new service worker installing', async () => {
      const { result } = renderHook(() => usePWAState());
      
      await waitFor(100);
      
      // Simulate updatefound event
      act(() => {
        const mockNewWorker = {
          state: 'installing',
          addEventListener: jest.fn()
        };

        const updateFoundEvent = new Event('updatefound');
        Object.defineProperty(updateFoundEvent, 'target', {
          value: {
            installing: mockNewWorker
          },
          writable: true
        });

        // Get the updatefound listener
        const addEventListenerCalls = mockServiceWorkerRegistration.addEventListener.mock.calls;
        const updateFoundListener = addEventListenerCalls
          .find(call => call[0] === 'updatefound')?.[1];
        
        if (updateFoundListener) {
          updateFoundListener(updateFoundEvent);
        }

        // Simulate state change to installed
        const stateChangeEvent = new Event('statechange');
        Object.defineProperty(stateChangeEvent, 'target', {
          value: { state: 'installed' },
          writable: true
        });

        const stateChangeListener = mockNewWorker.addEventListener.mock.calls
          .find(call => call[0] === 'statechange')?.[1];
        
        if (stateChangeListener) {
          // Mock navigator.serviceWorker.controller to simulate existing worker
          Object.defineProperty(global.navigator.serviceWorker, 'controller', {
            value: mockServiceWorkerRegistration.active,
            writable: true
          });
          
          stateChangeListener(stateChangeEvent);
        }
      });

      const [state] = result.current;
      expect(state.updateAvailable).toBe(true);
    });
  });

  describe('PWA Actions', () => {
    it('should install PWA successfully', async () => {
      const mockPrompt = {
        ...mockInstallPromptEvent,
        prompt: jest.fn().mockResolvedValue(undefined),
        userChoice: Promise.resolve({ outcome: 'accepted' })
      };

      const { result } = renderHook(() => usePWAState());
      
      // Set install prompt
      act(() => {
        const event = new Event('beforeinstallprompt');
        Object.assign(event, mockPrompt);
        
        const addEventListenerCalls = (window.addEventListener as jest.Mock).mock.calls;
        const beforeInstallPromptListener = addEventListenerCalls
          .find(call => call[0] === 'beforeinstallprompt')?.[1];
        
        if (beforeInstallPromptListener) {
          beforeInstallPromptListener(event);
        }
      });

      const [, actions] = result.current;

      await act(async () => {
        const result = await actions.installPWA();
        expect(result).toBe(true);
      });

      expect(mockPrompt.prompt).toHaveBeenCalled();
    });

    it('should handle install PWA rejection', async () => {
      const mockPrompt = {
        ...mockInstallPromptEvent,
        prompt: jest.fn().mockResolvedValue(undefined),
        userChoice: Promise.resolve({ outcome: 'dismissed' })
      };

      const { result } = renderHook(() => usePWAState());
      
      // Set install prompt
      act(() => {
        const event = new Event('beforeinstallprompt');
        Object.assign(event, mockPrompt);
        
        const addEventListenerCalls = (window.addEventListener as jest.Mock).mock.calls;
        const beforeInstallPromptListener = addEventListenerCalls
          .find(call => call[0] === 'beforeinstallprompt')?.[1];
        
        if (beforeInstallPromptListener) {
          beforeInstallPromptListener(event);
        }
      });

      const [, actions] = result.current;

      await act(async () => {
        const result = await actions.installPWA();
        expect(result).toBe(false);
      });
    });

    it('should handle install PWA when no prompt available', async () => {
      const { result } = renderHook(() => usePWAState());
      const [, actions] = result.current;

      await act(async () => {
        const result = await actions.installPWA();
        expect(result).toBe(false);
      });
    });

    it('should check for updates', async () => {
      const mockUpdate = jest.fn().mockResolvedValue(undefined);
      const mockGetRegistration = jest.fn().mockResolvedValue({
        update: mockUpdate
      });

      Object.defineProperty(global.navigator.serviceWorker, 'getRegistration', {
        value: mockGetRegistration,
        writable: true
      });

      const { result } = renderHook(() => usePWAState());
      const [, actions] = result.current;

      await act(async () => {
        await actions.checkForUpdates();
      });

      expect(mockUpdate).toHaveBeenCalled();
    });

    it('should skip waiting for new service worker', () => {
      const mockPostMessage = jest.fn();
      
      Object.defineProperty(global.navigator.serviceWorker, 'controller', {
        value: { postMessage: mockPostMessage },
        writable: true
      });

      // Mock window.location.reload
      const mockReload = jest.fn();
      Object.defineProperty(window.location, 'reload', {
        value: mockReload,
        writable: true
      });

      const { result } = renderHook(() => usePWAState());
      const [, actions] = result.current;

      act(() => {
        actions.skipWaiting();
      });

      expect(mockPostMessage).toHaveBeenCalledWith({ type: 'SKIP_WAITING' });
      expect(mockReload).toHaveBeenCalled();
    });

    it('should dismiss install prompt', () => {
      const { result } = renderHook(() => usePWAState());
      
      // Set install prompt first
      act(() => {
        const event = new Event('beforeinstallprompt');
        Object.assign(event, mockInstallPromptEvent);
        
        const addEventListenerCalls = (window.addEventListener as jest.Mock).mock.calls;
        const beforeInstallPromptListener = addEventListenerCalls
          .find(call => call[0] === 'beforeinstallprompt')?.[1];
        
        if (beforeInstallPromptListener) {
          beforeInstallPromptListener(event);
        }
      });

      const [, actions] = result.current;

      act(() => {
        actions.dismissInstallPrompt();
      });

      const [state] = result.current;
      expect(state.isInstallable).toBe(false);
      expect(state.installPrompt).toBe(null);
    });
  });

  describe('Display Mode Changes', () => {
    it('should listen for display mode changes', () => {
      const mockMediaQuery = {
        matches: false,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn()
      };

      mockMatchMedia.mockReturnValue(mockMediaQuery);

      renderHook(() => usePWAState());

      expect(mockMediaQuery.addEventListener).toHaveBeenCalledWith('change', expect.any(Function));
    });

    it('should update install status on display mode change', () => {
      const mockMediaQuery = {
        matches: false,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn()
      };

      mockMatchMedia.mockReturnValue(mockMediaQuery);

      const { result } = renderHook(() => usePWAState());

      // Simulate display mode change to standalone
      act(() => {
        mockMediaQuery.matches = true;
        const changeHandler = mockMediaQuery.addEventListener.mock.calls
          .find(call => call[0] === 'change')?.[1];
        
        if (changeHandler) {
          changeHandler();
        }
      });

      const [state] = result.current;
      expect(state.isInstalled).toBe(true);
    });
  });

  describe('Error Handling', () => {
    it('should handle service worker check failure', async () => {
      // Mock service worker to reject
      Object.defineProperty(global.navigator.serviceWorker, 'ready', {
        value: Promise.reject(new Error('Service worker error')),
        writable: true
      });

      const { result } = renderHook(() => usePWAState());
      
      await waitFor(100);
      
      const [state] = result.current;
      expect(state.serviceWorkerReady).toBe(false);
    });

    it('should handle install PWA error', async () => {
      const mockPrompt = {
        ...mockInstallPromptEvent,
        prompt: jest.fn().mockRejectedValue(new Error('Install failed'))
      };

      const { result } = renderHook(() => usePWAState());
      
      // Set install prompt
      act(() => {
        const event = new Event('beforeinstallprompt');
        Object.assign(event, mockPrompt);
        
        const addEventListenerCalls = (window.addEventListener as jest.Mock).mock.calls;
        const beforeInstallPromptListener = addEventListenerCalls
          .find(call => call[0] === 'beforeinstallprompt')?.[1];
        
        if (beforeInstallPromptListener) {
          beforeInstallPromptListener(event);
        }
      });

      const [, actions] = result.current;

      await act(async () => {
        const result = await actions.installPWA();
        expect(result).toBe(false);
      });
    });

    it('should handle update check failure', async () => {
      const mockGetRegistration = jest.fn().mockRejectedValue(new Error('Update check failed'));

      Object.defineProperty(global.navigator.serviceWorker, 'getRegistration', {
        value: mockGetRegistration,
        writable: true
      });

      const { result } = renderHook(() => usePWAState());
      const [, actions] = result.current;

      // Should not throw
      await act(async () => {
        await expect(actions.checkForUpdates()).resolves.toBeUndefined();
      });
    });
  });

  describe('Cleanup', () => {
    it('should cleanup event listeners on unmount', () => {
      const { unmount } = renderHook(() => usePWAState());

      const removeEventListenerCalls = (window.removeEventListener as jest.Mock).mock.calls;
      const initialCallCount = removeEventListenerCalls.length;

      unmount();

      const finalCallCount = (window.removeEventListener as jest.Mock).mock.calls.length;
      expect(finalCallCount).toBeGreaterThan(initialCallCount);

      // Should remove beforeinstallprompt and appinstalled listeners
      expect(window.removeEventListener).toHaveBeenCalledWith('beforeinstallprompt', expect.any(Function));
      expect(window.removeEventListener).toHaveBeenCalledWith('appinstalled', expect.any(Function));
    });

    it('should cleanup media query listener on unmount', () => {
      const mockMediaQuery = {
        matches: false,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn()
      };

      mockMatchMedia.mockReturnValue(mockMediaQuery);

      const { unmount } = renderHook(() => usePWAState());

      unmount();

      expect(mockMediaQuery.removeEventListener).toHaveBeenCalledWith('change', expect.any(Function));
    });
  });

  describe('Service Worker Not Supported', () => {
    it('should handle missing service worker support', async () => {
      // Remove service worker support
      delete (global.navigator as any).serviceWorker;

      const { result } = renderHook(() => usePWAState());
      
      await waitFor(100);
      
      const [state] = result.current;
      expect(state.serviceWorkerReady).toBe(false);
    });
  });
});