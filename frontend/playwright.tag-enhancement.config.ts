import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration specifically for Enhanced Tag System tests
 * Run with: npx playwright test --config=playwright.tag-enhancement.config.ts
 */
export default defineConfig({
  testDir: './e2e',
  
  // Focus on tag enhancement tests only
  testMatch: [
    '**/enhanced-tag-prominence.spec.ts',
    '**/tag-spam-prevention.spec.ts'
  ],

  /* Run tests in files in parallel */
  fullyParallel: false, // Run sequentially for tag tests to avoid conflicts
  
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 1,
  
  /* Opt out of parallel tests for tag system tests */
  workers: 1, // Single worker to avoid database conflicts
  
  /* Reporter to use */
  reporter: [
    ['html', { outputFolder: 'test-results/tag-enhancement-report' }],
    ['list'],
    ['json', { outputFile: 'test-results/tag-enhancement-results.json' }]
  ],
  
  /* Shared settings for enhanced tag system tests */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: 'http://localhost:3001', // Frontend on 3001
    
    /* Run tests in headless mode by default, but allow override */
    headless: process.env.HEADLESS !== 'false',
    
    /* Collect trace when retrying failed tests */
    trace: 'retain-on-failure',
    
    /* Take screenshot on failure */
    screenshot: 'only-on-failure',
    
    /* Record video on failure */
    video: 'retain-on-failure',
    
    /* Slower actions for form interactions */
    actionTimeout: 10000,
    
    /* Wait for network requests to complete */
    navigationTimeout: 30000,
  },

  /* Global test timeout */
  timeout: 60000,

  /* Configure projects for testing enhanced tag system */
  projects: [
    // Primary testing environment
    {
      name: 'tag-enhancement-chrome',
      use: { 
        ...devices['Desktop Chrome'],
        // Extra time for tag processing
        actionTimeout: 15000,
      },
    },

    // Test on Firefox for cross-browser compatibility
    {
      name: 'tag-enhancement-firefox',
      use: { 
        ...devices['Desktop Firefox'],
        actionTimeout: 15000,
      },
    },

    // Mobile testing for responsive design
    {
      name: 'tag-enhancement-mobile',
      use: { 
        ...devices['Pixel 5'],
        actionTimeout: 20000, // Mobile might be slower
      },
    },
  ],

  /* Global setup for tag enhancement tests */
  globalSetup: require.resolve('./e2e/global-setup.ts'),

  /* Run your local services before starting the tests */
  webServer: [
    // Frontend
    {
      command: 'npm run dev',
      url: 'http://localhost:3001',
      reuseExistingServer: !process.env.CI,
      timeout: 120000,
    },
    // Backend services should already be running via docker-compose
  ],

  /* Expect clause configuration */
  expect: {
    /* Maximum time expect() should wait for the condition to be met */
    timeout: 10000,
    
    /* Take screenshot on assertion failure */
    toHaveScreenshot: { 
      threshold: 0.2, // Allow some visual differences 
    },
  },
});