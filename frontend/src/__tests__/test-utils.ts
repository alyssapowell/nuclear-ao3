// Test utilities for Nuclear AO3 PWA tests
import { render, RenderOptions } from '@testing-library/react';
import { ReactElement } from 'react';

// Mock service worker registration
export const mockServiceWorkerRegistration = {
  active: {
    postMessage: jest.fn(),
    addEventListener: jest.fn(),
    state: 'activated'
  },
  installing: null,
  waiting: null,
  addEventListener: jest.fn(),
  update: jest.fn().mockResolvedValue(undefined),
  unregister: jest.fn().mockResolvedValue(true)
};

// Mock navigator.serviceWorker
export const mockServiceWorker = {
  ready: Promise.resolve(mockServiceWorkerRegistration),
  controller: mockServiceWorkerRegistration.active,
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  getRegistration: jest.fn().mockResolvedValue(mockServiceWorkerRegistration),
  register: jest.fn().mockResolvedValue(mockServiceWorkerRegistration)
};

// Mock storage estimate
export const mockStorageEstimate = {
  quota: 1000000000, // 1GB
  usage: 50000000,   // 50MB
  usageDetails: {}
};

// Mock navigator.storage
export const mockStorage = {
  estimate: jest.fn().mockResolvedValue(mockStorageEstimate),
  persist: jest.fn().mockResolvedValue(true),
  persisted: jest.fn().mockResolvedValue(false)
};

// Mock offline work data
export const mockOfflineWork = {
  workId: 'test-work-123',
  title: 'Test Fanfiction',
  authors: [{ pseud_name: 'TestAuthor', username: 'test_author' }],
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
  authors: [{ pseud_name: 'TestAuthor', username: 'test_author' }],
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
  prompt: jest.fn().mockResolvedValue(undefined),
  userChoice: Promise.resolve({ outcome: 'accepted' }),
  preventDefault: jest.fn()
};

// Mock message channel
export class MockMessageChannel {
  port1: {
    onmessage: ((event: MessageEvent) => void) | null;
    postMessage: jest.Mock;
    close: jest.Mock;
  };
  port2: {
    onmessage: ((event: MessageEvent) => void) | null;
    postMessage: jest.Mock;
    close: jest.Mock;
  };

  constructor() {
    this.port1 = {
      onmessage: null,
      postMessage: jest.fn(),
      close: jest.fn()
    };
    this.port2 = {
      onmessage: null,
      postMessage: jest.fn(),
      close: jest.fn()
    };
  }
}

// Setup global mocks
export const setupGlobalMocks = () => {
  // Mock navigator
  Object.defineProperty(global.navigator, 'serviceWorker', {
    value: mockServiceWorker,
    writable: true
  });

  Object.defineProperty(global.navigator, 'storage', {
    value: mockStorage,
    writable: true
  });

  Object.defineProperty(global.navigator, 'onLine', {
    value: true,
    writable: true
  });

  // Mock window
  Object.defineProperty(global.window, 'addEventListener', {
    value: jest.fn(),
    writable: true
  });

  Object.defineProperty(global.window, 'removeEventListener', {
    value: jest.fn(),
    writable: true
  });

  Object.defineProperty(global.window, 'matchMedia', {
    value: jest.fn().mockReturnValue({
      matches: false,
      addListener: jest.fn(),
      removeListener: jest.fn(),
      addEventListener: jest.fn(),
      removeEventListener: jest.fn()
    }),
    writable: true
  });

  // Mock MessageChannel
  global.MessageChannel = MockMessageChannel as any;

  // Mock localStorage
  const localStorageMock = {
    getItem: jest.fn(),
    setItem: jest.fn(),
    removeItem: jest.fn(),
    clear: jest.fn()
  };
  Object.defineProperty(global, 'localStorage', {
    value: localStorageMock,
    writable: true
  });

  // Mock fetch
  global.fetch = jest.fn();

  return {
    mockServiceWorker,
    mockStorage,
    localStorageMock,
    fetchMock: global.fetch as jest.Mock
  };
};

// Reset all mocks
export const resetMocks = () => {
  jest.clearAllMocks();
  mockServiceWorkerRegistration.active.postMessage.mockClear();
  mockServiceWorkerRegistration.addEventListener.mockClear();
  mockServiceWorker.addEventListener.mockClear();
  mockServiceWorker.removeEventListener.mockClear();
  mockStorage.estimate.mockClear();
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
    writable: true
  });
  
  // Trigger online/offline events
  const eventType = isOnline ? 'online' : 'offline';
  const event = new Event(eventType);
  
  // Find and call event listeners
  const calls = (global.window.addEventListener as jest.Mock).mock.calls;
  const listeners = calls.filter(call => call[0] === eventType);
  
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