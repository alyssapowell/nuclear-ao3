import { defineConfig, devices } from '@playwright/test';

/**
 * PWA-specific Playwright configuration
 * Tests service workers, offline functionality, and PWA installation
 */
export default defineConfig({
  testDir: './e2e',
  testMatch: '**/pwa-*.spec.ts',
  
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  
  reporter: [
    ['html', { outputFolder: 'playwright-report-pwa' }],
    ['json', { outputFile: 'test-results-pwa.json' }]
  ],
  
  use: {
    baseURL: 'http://localhost:3002',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    
    // Enable service workers for PWA testing
    serviceWorkers: 'allow',
    
    // Grant permissions that PWAs might need
    permissions: ['notifications'],
    
    // Set up for offline testing
    offline: false,
  },

  projects: [
    {
      name: 'PWA Chrome',
      use: { 
        ...devices['Desktop Chrome'],
        // Simulate PWA environment
        channel: 'chrome',
        launchOptions: {
          args: [
            '--enable-features=VaapiVideoDecoder',
            '--disable-background-timer-throttling',
            '--disable-renderer-backgrounding',
            '--disable-backgrounding-occluded-windows'
          ]
        }
      },
    },

    {
      name: 'PWA Firefox',
      use: { 
        ...devices['Desktop Firefox'],
        launchOptions: {
          firefoxUserPrefs: {
            'dom.serviceWorkers.enabled': true,
            'dom.push.enabled': true
          }
        }
      },
    },

    {
      name: 'PWA Safari',
      use: { 
        ...devices['Desktop Safari'],
      },
    },

    // Mobile PWA testing
    {
      name: 'PWA Mobile Chrome',
      use: { 
        ...devices['Pixel 5'],
        // Enable PWA manifest processing
        extraHTTPHeaders: {
          'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8'
        }
      },
    },

    {
      name: 'PWA Mobile Safari',
      use: { 
        ...devices['iPhone 12'],
      },
    },
  ],

  webServer: {
    command: 'npx next dev -p 3002',
    port: 3002,
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
    env: {
      NODE_ENV: 'development'
    }
  },
});