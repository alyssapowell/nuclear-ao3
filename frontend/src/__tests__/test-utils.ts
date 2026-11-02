import type { Mock } from 'vitest';
// Test utilities for Nuclear AO3 PWA tests
import { render, RenderOptions } from '@testing-library/react';
import { ReactElement } from 'react';

// Mock service worker registration
export const mockServiceWorkerRegistration = {
  active: {
    postMessage: vi.fn(),
    addEventListener: vi.fn(),
    state: 'activated'
  },
  installing: null,
  waiting: null,
  addEventListener: vi.fn(),
  update: vi.fn().mockResolvedValue(undefined),
  unregister: vi.fn().mockResolvedValue(true)
};

// Mock navigator.serviceWorker
export const mockServiceWorker = {
  ready: Promise.resolve(mockServiceWorkerRegistration),
  controller: mockServiceWorkerRegistration.active,
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
  getRegistration: vi.fn().mockResolvedValue(mockServiceWorkerRegistration),
  register: vi.fn().mockResolvedValue(mockServiceWorkerRegistration)
};

// Mock storage estimate
export const mockStorageEstimate = {
  quota: 1000000000, // 1GB
  usage: 50000000,   // 50MB
  usageDetails: {}
};

// Mock navigator.storage
export const mockStorage = {
  estimate: vi.fn().mockResolvedValue(mockStorageEstimate),
  persist: vi.fn().mockResolvedValue(true),
  persisted: vi.fn().mockResolvedValue(false)
};

// Mock offline work data
export const mockOfflineWork = {
  workId: 'test-work-123',
  title: 'Test Fanfiction',
  authors: [{ pseud_name: 'TestAuthor' }],
  consentLevel: 'files_and_pwa' as const,
  cachedAt: Date.now() - 3600000, // 1 hour ago
  expiresAt: Date.now() + 86400000, // 24 hours from now
  isExpired: false,
  word_count: 15000,
  chapter_count: 5
};

// Mock work data for testing
export const mockWorkData = {
  id: 'test-work-123',
  title: 'Test Fanfiction',
  authors: [{ pseud_name: 'TestAuthor' }],
  summary: 'A test work for our tests',
  word_count: 15000,
  chapter_count: 5,
  is_complete: true,
  tags: {
    fandoms: ['Test Fandom'],
    relationships: ['A/B'],
    characters: ['Character A', 'Character B'],
    freeform_tags: ['Fluff'],
    warnings: ['No Archive Warnings Apply'],
    categories: ['M/M'],
    rating: 'Teen And Up Audiences'
  },
  offline_reading_preference: 'files_and_pwa' as const,
  chapters: [
    {
      id: 'chapter-1',
      number: 1,
      title: 'Chapter 1',
      content: '<p>Test content</p>',
      word_count: 3000
    }
  ]
};

// Mock install prompt event
export const mockInstallPromptEvent = {
  prompt: vi.fn().mockResolvedValue(undefined),
  userChoice: Promise.resolve({ outcome: 'accepted' }),
  preventDefault: vi.fn()
};

// Mock message channel
export class MockMessageChannel {
  port1: {
    onmessage: ((event: MessageEvent) => void) | null;
    postMessage: Mock;
    close: Mock;
  };
  port2: {
    onmessage: ((event: MessageEvent) => void) | null;
    postMessage: Mock;
    close: Mock;
  };

  constructor() {
    this.port1 = {
      onmessage: null,
      postMessage: vi.fn(),
      close: vi.fn()
    };
    // What the service worker posts back on port2 arrives on port1, as in a browser.
    this.port2 = {
      onmessage: null,
      postMessage: vi.fn((data: unknown) => {
        this.port1.onmessage?.({ data } as MessageEvent);
      }),
      close: vi.fn()
    };
  }
}

// Setup global mocks
export const setupGlobalMocks = () => {
  // Mock navigator
  Object.defineProperty(global.navigator, 'serviceWorker', {
    value: mockServiceWorker,
    writable: true,
    configurable: true
  });

  Object.defineProperty(global.navigator, 'storage', {
    value: mockStorage,
    writable: true,
    configurable: true
  });

  Object.defineProperty(global.navigator, 'onLine', {
    value: true,
    writable: true,
    configurable: true
  });

  // Mock window
  Object.defineProperty(global.window, 'addEventListener', {
    value: vi.fn(),
    writable: true
  });

  Object.defineProperty(global.window, 'removeEventListener', {
    value: vi.fn(),
    writable: true
  });

  Object.defineProperty(global.window, 'matchMedia', {
    value: vi.fn().mockReturnValue({
      matches: false,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn()
    }),
    writable: true
  });

  // Mock MessageChannel
  global.MessageChannel = MockMessageChannel as any;

  // Mock localStorage
  const localStorageMock = {
    getItem: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn()
  };
  Object.defineProperty(global, 'localStorage', {
    value: localStorageMock,
    writable: true
  });

  // Mock fetch
  global.fetch = vi.fn();

  return {
    mockServiceWorker,
    mockStorage,
    localStorageMock,
    fetchMock: global.fetch as Mock
  };
};

// Reset all mocks
export const resetMocks = () => {
  vi.clearAllMocks();
  mockServiceWorkerRegistration.active.postMessage.mockReset();
  mockServiceWorkerRegistration.addEventListener.mockClear();
  mockServiceWorker.addEventListener.mockClear();
  mockServiceWorker.removeEventListener.mockClear();
  mockStorage.estimate.mockReset().mockResolvedValue(mockStorageEstimate);
};

// Makes the service worker answer GET_OFFLINE_WORKS with these works.
// Pass nothing to make it stay silent, so the request times out.
export const respondWithOfflineWorks = (works?: unknown[]) => {
  mockServiceWorkerRegistration.active.postMessage.mockImplementation(
    (message: { type: string }, ports?: Array<{ postMessage: (data: unknown) => void }>) => {
      if (works && message.type === 'GET_OFFLINE_WORKS') ports?.[0].postMessage({ works });
    }
  );
};

// Helper to simulate service worker messages
export const simulateServiceWorkerMessage = (type: string, data: any = {}) => {
  const event = new MessageEvent('message', {
    data: { type, ...data }
  });
  
  // Find the message listener and call it
  const calls = mockServiceWorker.addEventListener.mock.calls;
  const messageListener = calls.find(call => call[0] === 'message')?.[1];
  
  if (messageListener) {
    messageListener(event);
  }
  
  return event;
};

// Helper to simulate storage quota
export const simulateStorageQuota = (quota: number, usage: number) => {
  mockStorage.estimate.mockResolvedValueOnce({
    quota,
    usage,
    usageDetails: {}
  });
};

// Helper to simulate network status
export const simulateNetworkStatus = (isOnline: boolean) => {
  Object.defineProperty(global.navigator, 'onLine', {
    value: isOnline,
    writable: true,
    configurable: true
  });
  
  // Trigger online/offline events
  const eventType = isOnline ? 'online' : 'offline';
  const event = new Event(eventType);
  
  // Find and call event listeners
  const calls = (global.window.addEventListener as Mock).mock.calls;
  const removed = (global.window.removeEventListener as Mock).mock.calls;
  // A listener that has been removed no longer hears the event.
  const listeners = calls.filter(call =>
    call[0] === eventType && !removed.some(r => r[0] === eventType && r[1] === call[1])
  );
  
  listeners.forEach(([, listener]) => {
    listener(event);
  });
};

// Custom render function with providers
export const renderWithProviders = (
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) => {
  return render(ui, {
    ...options
  });
};

// Utility to wait for async operations
export const waitFor = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

export * from '@testing-library/react';
export { renderWithProviders as render };