import { defineConfig, devices } from '@playwright/test';

/**
 * Configuration for integration tests against existing running services
 * Enhanced to support collections tests specifically
 */
export default defineConfig({
  testDir: './e2e',
  
  /* Run collections tests with more careful coordination */
  fullyParallel: false, 
  
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  
  /* More retries for integration tests with backend dependencies */
  retries: process.env.CI ? 3 : 1,
  
  /* Fewer workers for integration tests to avoid conflicts */
  workers: process.env.CI ? 1 : 2,
  
  /* Enhanced reporting for collections tests */
  reporter: [
    ['line'],
    ['html', { outputFolder: 'playwright-report-integration' }],
    ['junit', { outputFile: 'test-results/integration-results.xml' }]
  ],
  
  /* Shared settings optimized for collections tests */
  use: {
    /* Base URL - can be overridden with environment variables */
    baseURL: process.env.BASE_URL || 'http://localhost:3001',

    /* Enhanced tracing for debugging collections issues */
    trace: 'on-first-retry',
    
    /* Screenshots for failed collections tests */
    screenshot: 'only-on-failure',
    
    /* Video recording for failed tests */
    video: 'retain-on-failure',
    
    /* Longer timeouts for collections operations */
    actionTimeout: 10000,
    navigationTimeout: 15000,
  },

  /* Test output directory */
  outputDir: 'test-results/integration',

  /* Configure projects for comprehensive collections testing */
  projects: [
    {
      name: 'chromium-collections',
      use: { ...devices['Desktop Chrome'] },
      testMatch: ['**/collections*.spec.ts'],
    },
    
    {
      name: 'firefox-collections',
      use: { ...devices['Desktop Firefox'] },
      testMatch: ['**/collections-flow.spec.ts'], // Core collections flow only
    },
    
    {
      name: 'mobile-collections',
      use: { ...devices['Pixel 5'] },
      testMatch: ['**/collections-flow.spec.ts'], // Mobile compatibility
    },
    
    {
      name: 'all-other-tests',
      use: { ...devices['Desktop Chrome'] },
      testIgnore: ['**/collections*.spec.ts'], // All non-collections tests
    },
  ],

  /* Global timeout for long-running integration tests */
  globalTimeout: 60 * 60 * 1000, // 1 hour

  /* Test expectations optimized for integration testing */
  expect: {
    timeout: 10000,
  },

  // No webServer - assume services are already running
  // If services need to be started, use the main playwright.config.ts
});