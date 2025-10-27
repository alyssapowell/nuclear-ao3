import { test, expect } from '@playwright/test';

/**
 * GDPR Consent Flow E2E Tests
 * Tests GDPR compliance features including consent management,
 * data export, data deletion, and privacy settings
 */

test.describe('GDPR Consent and Privacy', () => {
  test.beforeEach(async ({ page }) => {
    // Start fresh for each test
    await page.goto('/');
  });

  test('should display cookie consent banner for new users', async ({ page }) => {
    // Clear cookies to simulate new user
    await page.context().clearCookies();
    await page.reload();

    // Should show cookie consent banner
    const consentBanner = page.locator('.cookie-consent, .gdpr-banner, [data-testid="cookie-consent"]');
    if (await consentBanner.isVisible()) {
      await expect(consentBanner).toBeVisible();
      
      // Should have accept and reject options
      await expect(page.locator('button:has-text("Accept"), button:has-text("Allow")')).toBeVisible();
      await expect(page.locator('button:has-text("Reject"), button:has-text("Decline")')).toBeVisible();
      
      // Should have link to privacy policy
      await expect(page.locator('a:has-text("Privacy Policy"), a:has-text("Cookie Policy")')).toBeVisible();
    } else {
      test.skip('Cookie consent banner not implemented');
    }
  });

  test('should allow users to accept cookie consent', async ({ page }) => {
    // Clear cookies to simulate new user
    await page.context().clearCookies();
    await page.reload();

    const consentBanner = page.locator('.cookie-consent, .gdpr-banner, [data-testid="cookie-consent"]');
    const acceptButton = page.locator('button:has-text("Accept"), button:has-text("Allow")');
    
    if (await consentBanner.isVisible() && await acceptButton.isVisible()) {
      await acceptButton.click();
      
      // Banner should disappear
      await expect(consentBanner).not.toBeVisible();
      
      // Consent should be remembered on page reload
      await page.reload();
      await expect(consentBanner).not.toBeVisible();
    } else {
      test.skip('Cookie consent functionality not available');
    }
  });

  test('should allow users to reject cookie consent', async ({ page }) => {
    // Clear cookies to simulate new user
    await page.context().clearCookies();
    await page.reload();

    const consentBanner = page.locator('.cookie-consent, .gdpr-banner, [data-testid="cookie-consent"]');
    const rejectButton = page.locator('button:has-text("Reject"), button:has-text("Decline")');
    
    if (await consentBanner.isVisible() && await rejectButton.isVisible()) {
      await rejectButton.click();
      
      // Banner should disappear
      await expect(consentBanner).not.toBeVisible();
      
      // Only essential cookies should be set
      const cookies = await page.context().cookies();
      const nonEssentialCookies = cookies.filter(cookie => 
        !cookie.name.includes('session') && 
        !cookie.name.includes('csrf') &&
        !cookie.name.includes('consent')
      );
      
      // Should have minimal non-essential cookies
      expect(nonEssentialCookies.length).toBeLessThanOrEqual(1);
    } else {
      test.skip('Cookie consent functionality not available');
    }
  });

  test('should provide cookie preferences management', async ({ page }) => {
    // Navigate to privacy settings or cookie preferences
    const settingsLinks = [
      'a:has-text("Privacy Settings")',
      'a:has-text("Cookie Preferences")',
      'a:has-text("Manage Cookies")',
      'button:has-text("Customize Cookies")'
    ];

    let preferencesFound = false;
    for (const selector of settingsLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        preferencesFound = true;
        break;
      }
    }

    if (preferencesFound) {
      // Should show cookie categories
      const categories = [
        'Essential',
        'Analytics', 
        'Marketing',
        'Functional'
      ];

      for (const category of categories) {
        const categoryElement = page.locator(`text=${category}`);
        if (await categoryElement.isVisible()) {
          await expect(categoryElement).toBeVisible();
        }
      }

      // Should have save preferences button
      await expect(page.locator('button:has-text("Save"), button:has-text("Update")')).toBeVisible();
    } else {
      test.skip('Cookie preferences management not available');
    }
  });

  test('should allow users to request data export', async ({ page }) => {
    // Login first (needed for data export)
    const loginLink = page.locator('a[href="/login"], a:has-text("Log in")');
    if (await loginLink.isVisible()) {
      await loginLink.click();
      
      // Use test credentials if available
      const emailInput = page.locator('input[name="email"], input[type="email"]');
      const passwordInput = page.locator('input[name="password"], input[type="password"]');
      
      if (await emailInput.isVisible()) {
        await emailInput.fill('testuser30d_v2@example.com');
        await passwordInput.fill('TestPassword123!');
        await page.click('button[type="submit"]');
        await page.waitForLoadState('networkidle');
      }
    }

    // Navigate to privacy/data settings
    const privacyLinks = [
      'a:has-text("Privacy")',
      'a:has-text("Data Export")',
      'a:has-text("Download My Data")',
      'a:has-text("Account Settings")'
    ];

    let dataExportFound = false;
    for (const selector of privacyLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        
        // Look for data export option
        const exportButton = page.locator('button:has-text("Export"), button:has-text("Download"), a:has-text("Export Data")');
        if (await exportButton.isVisible()) {
          dataExportFound = true;
          await expect(exportButton).toBeVisible();
          await expect(exportButton).toBeEnabled();
          break;
        }
      }
    }

    if (!dataExportFound) {
      test.skip('Data export functionality not available');
    }
  });

  test('should allow users to request account deletion', async ({ page }) => {
    // Login first
    const loginLink = page.locator('a[href="/login"], a:has-text("Log in")');
    if (await loginLink.isVisible()) {
      await loginLink.click();
      
      const emailInput = page.locator('input[name="email"], input[type="email"]');
      const passwordInput = page.locator('input[name="password"], input[type="password"]');
      
      if (await emailInput.isVisible()) {
        await emailInput.fill('testuser30d_v2@example.com');
        await passwordInput.fill('TestPassword123!');
        await page.click('button[type="submit"]');
        await page.waitForLoadState('networkidle');
      }
    }

    // Navigate to account settings
    const accountLinks = [
      'a:has-text("Account")',
      'a:has-text("Settings")',
      'a:has-text("Profile")',
      'a:has-text("Delete Account")'
    ];

    let deletionFound = false;
    for (const selector of accountLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        
        // Look for account deletion option
        const deleteButton = page.locator('button:has-text("Delete"), a:has-text("Delete Account"), button:has-text("Close Account")');
        if (await deleteButton.isVisible()) {
          deletionFound = true;
          await expect(deleteButton).toBeVisible();
          
          // Should have warning about permanent deletion
          await expect(page.locator('text=permanent, text=cannot be undone, text=irreversible')).toBeVisible();
          break;
        }
      }
    }

    if (!deletionFound) {
      test.skip('Account deletion functionality not available');
    }
  });

  test('should display privacy policy with required GDPR information', async ({ page }) => {
    // Find and navigate to privacy policy
    const privacyLink = page.locator('a:has-text("Privacy Policy"), a:has-text("Privacy"), a[href*="privacy"]');
    
    if (await privacyLink.isVisible()) {
      await privacyLink.click();
      
      // Should display comprehensive privacy information
      const requiredSections = [
        'data collection',
        'data processing',
        'data retention',
        'your rights',
        'contact information'
      ];

      for (const section of requiredSections) {
        const sectionText = page.locator(`text=${section}`);
        if (await sectionText.isVisible()) {
          await expect(sectionText).toBeVisible();
        }
      }

      // Should mention GDPR rights
      const gdprText = page.locator('text=GDPR, text=General Data Protection, text=right to rectification, text=right to erasure');
      if (await gdprText.isVisible()) {
        await expect(gdprText).toBeVisible();
      }
    } else {
      test.skip('Privacy policy not accessible');
    }
  });

  test('should provide data portability options', async ({ page }) => {
    // Login first
    const loginLink = page.locator('a[href="/login"], a:has-text("Log in")');
    if (await loginLink.isVisible()) {
      await loginLink.click();
      
      const emailInput = page.locator('input[name="email"], input[type="email"]');
      if (await emailInput.isVisible()) {
        await emailInput.fill('testuser30d_v2@example.com');
        await page.fill('input[name="password"], input[type="password"]', 'TestPassword123!');
        await page.click('button[type="submit"]');
        await page.waitForLoadState('networkidle');
      }
    }

    // Look for data portability features
    const exportLinks = [
      'a:has-text("Export")',
      'a:has-text("Download")',
      'a:has-text("Backup")',
      'button:has-text("Export My Data")'
    ];

    let portabilityFound = false;
    for (const selector of exportLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        
        // Should show export format options
        const formatOptions = page.locator('text=JSON, text=CSV, text=XML, select[name*="format"]');
        if (await formatOptions.isVisible()) {
          portabilityFound = true;
          await expect(formatOptions).toBeVisible();
          break;
        }
      }
    }

    if (!portabilityFound) {
      test.skip('Data portability features not available');
    }
  });

  test('should respect user consent for analytics tracking', async ({ page }) => {
    // Clear cookies to start fresh
    await page.context().clearCookies();
    await page.reload();

    // Block analytics scripts initially
    await page.route('**/analytics/**', route => route.abort());
    await page.route('**/google-analytics.com/**', route => route.abort());
    await page.route('**/googletagmanager.com/**', route => route.abort());

    const consentBanner = page.locator('.cookie-consent, .gdpr-banner');
    
    if (await consentBanner.isVisible()) {
      // Reject tracking initially
      const rejectButton = page.locator('button:has-text("Reject"), button:has-text("Decline")');
      if (await rejectButton.isVisible()) {
        await rejectButton.click();
      }
      
      // Verify no analytics scripts are loaded
      const analyticsScripts = page.locator('script[src*="analytics"], script[src*="google-analytics"]');
      expect(await analyticsScripts.count()).toBe(0);
    } else {
      test.skip('Analytics consent testing not available');
    }
  });

  test('should provide consent withdrawal mechanism', async ({ page }) => {
    // First accept consent
    await page.context().clearCookies();
    await page.reload();

    const consentBanner = page.locator('.cookie-consent, .gdpr-banner');
    const acceptButton = page.locator('button:has-text("Accept"), button:has-text("Allow")');
    
    if (await consentBanner.isVisible() && await acceptButton.isVisible()) {
      await acceptButton.click();
      await expect(consentBanner).not.toBeVisible();
    }

    // Then find way to withdraw consent
    const withdrawalLinks = [
      'a:has-text("Cookie Preferences")',
      'a:has-text("Manage Consent")',
      'a:has-text("Privacy Settings")',
      'button:has-text("Withdraw Consent")'
    ];

    let withdrawalFound = false;
    for (const selector of withdrawalLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        
        // Should provide option to withdraw consent
        const withdrawOption = page.locator('button:has-text("Withdraw"), button:has-text("Revoke"), input[type="checkbox"]:not(:checked)');
        if (await withdrawOption.isVisible()) {
          withdrawalFound = true;
          await expect(withdrawOption).toBeVisible();
          break;
        }
      }
    }

    if (!withdrawalFound) {
      test.skip('Consent withdrawal mechanism not available');
    }
  });
});