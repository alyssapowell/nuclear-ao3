// Runs before every unit test file (see "test" in vite.config.ts).
import '@testing-library/jest-dom/vitest'

vi.mock('react-router-dom', async () => (await import('./router-mock')).routerMock())

// A stand-in localStorage whose methods tests can inspect.
Object.defineProperty(window, 'localStorage', {
  value: {
    getItem: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn(),
  },
})

// Every fetch succeeds with an empty list unless a test says otherwise.
globalThis.fetch = vi.fn(() =>
  Promise.resolve({
    ok: true,
    status: 200,
    json: () => Promise.resolve({ success: true, data: [] }),
    text: () => Promise.resolve('{"success": true, "data": []}'),
    headers: new Headers(),
    statusText: 'OK',
  } as Response)
)

// Quiet the errors that tests trigger on purpose.
const expectedErrors = [
  'Search API error:',
  'Search error:',
  'Browse collections error:',
  'Create collection error:',
  'API error:',
]
const originalError = console.error
const originalWarn = console.warn
beforeAll(() => {
  console.error = (...args: unknown[]) => {
    if (typeof args[0] === 'string' && expectedErrors.some((e) => (args[0] as string).includes(e))) return
    originalError(...args)
  }
  console.warn = (...args: unknown[]) => {
    if (typeof args[0] === 'string' && args[0].includes('[OfflineManager] Service Worker not supported')) return
    originalWarn(...args)
  }
})
afterAll(() => {
  console.error = originalError
  console.warn = originalWarn
})
