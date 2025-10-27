import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright configuration specifically for accessibility and E2E search testing
 * 
 * This config enables accessibility tools and comprehensive device testing
 * for our accessibility-enhanced search functionality.
 */
export default defineConfig({
  testDir: './e2e',
  /* Run tests in files in parallel */
  fullyParallel: true,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  /* Opt out of parallel tests on CI. */
  workers: process.env.CI ? 1 : undefined,
  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: [
    ['html'],
    ['junit', { outputFile: 'test-results/accessibility-results.xml' }],
    ['json', { outputFile: 'test-results/accessibility-results.json' }]
  ],
  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: 'http://localhost:3000',

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: 'on-first-retry',

    /* Screenshot on failure */
    screenshot: 'only-on-failure',

    /* Video on failure */
    video: 'retain-on-failure',

    /* Accessibility testing options */
    // Enable accessibility tree snapshots
    actionTimeout: 10000,
    navigationTimeout: 30000,
  },

  /* Configure projects for major browsers with accessibility focus */
  projects: [
    // Desktop browsers
    {
      name: 'chromium-accessibility',
      use: { 
        ...devices['Desktop Chrome'],
        // Enable accessibility features
        launchOptions: {
          args: [
            '--enable-experimental-accessibility-features',
            '--force-prefers-reduced-motion',
            '--force-color-profile=srgb',
          ],
        },
      },
    },

    {
      name: 'firefox-accessibility',
      use: { 
        ...devices['Desktop Firefox'],
        // Firefox accessibility settings
        launchOptions: {
          firefoxUserPrefs: {
            'ui.prefersReducedMotion': 1,
            'browser.display.use_system_colors': true,
          },
        },
      },
    },

    {
      name: 'webkit-accessibility',
      use: { 
        ...devices['Desktop Safari'],
        // Safari accessibility settings
        launchOptions: {
          args: [
            '--enable-features=AccessibilityExposeARIAAnnotations',
          ],
        },
      },
    },

    // High contrast mode testing
    {
      name: 'high-contrast',
      use: {
        ...devices['Desktop Chrome'],
        colorScheme: 'dark',
        launchOptions: {
          args: [
            '--force-dark-mode',
            '--enable-features=WebContentsForceDarkMode',
          ],
        },
      },
    },

    // Reduced motion testing
    {
      name: 'reduced-motion',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          args: ['--force-prefers-reduced-motion'],
        },
      },
    },

    // Mobile devices with accessibility focus
    {
      name: 'Mobile Chrome',
      use: { 
        ...devices['Pixel 5'],
        // Mobile accessibility settings
        isMobile: true,
        hasTouch: true,
      },
    },

    {
      name: 'Mobile Safari',
      use: { 
        ...devices['iPhone 12'],
        // iOS accessibility settings
        isMobile: true,
        hasTouch: true,
      },
    },

    // Tablet testing
    {
      name: 'Tablet',
      use: { 
        ...devices['iPad Pro'],
        isMobile: true,
        hasTouch: true,
      },
    },

    // Large text/zoom testing
    {
      name: 'large-text',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1280, height: 720 },
        deviceScaleFactor: 1.5, // Simulate zoom
        launchOptions: {
          args: [
            '--force-device-scale-factor=1.5',
            '--enable-experimental-accessibility-features',
          ],
        },
      },
    },

    // Screen reader simulation
    {
      name: 'screen-reader-sim',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          args: [
            '--enable-experimental-accessibility-features',
            '--enable-automation', 
            '--enable-logging',
          ],
        },
        // Slow down interactions to simulate screen reader pace
        actionTimeout: 15000,
      },
    },
  ],

  /* Run your local dev server before starting the tests */
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
  },

  /* Test patterns for accessibility */
  testMatch: [
    '**/e2e/**/*accessibility*.spec.ts',
    '**/e2e/**/*search*.spec.ts',
    '**/e2e/**/*keyboard*.spec.ts',
  ],

  /* Global test timeout */
  timeout: 60000,

  /* Global setup for accessibility testing */
  globalSetup: require.resolve('./e2e/global-setup.ts'),
  globalTeardown: require.resolve('./e2e/global-teardown.ts'),

  /* Expect configuration */
  expect: {
    /* Maximum time expect() should wait for the condition to be met. */
    timeout: 10000,
  },
})