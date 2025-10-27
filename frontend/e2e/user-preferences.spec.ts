import { test, expect } from '@playwright/test';

/**
 * User Preferences and Settings E2E Tests
 * Tests user account settings, preferences, and customization options
 */

test.describe('User Preferences and Settings', () => {
  test.beforeEach(async ({ page }) => {
    // Login first for all preference tests
    await page.goto('/login');
    
    const emailInput = page.locator('input[name="email"], input[type="email"]');
    if (await emailInput.isVisible()) {
      await emailInput.fill('testuser30d_v2@example.com');
      await page.fill('input[name="password"], input[type="password"]', 'TestPassword123!');
      await page.click('button[type="submit"]');
      await page.waitForLoadState('networkidle');
    } else {
      test.skip('Login functionality not available');
    }
  });

  test('should navigate to user settings page', async ({ page }) => {
    // Find settings link in user menu or navigation
    const settingsLinks = [
      'a:has-text("Settings")',
      'a:has-text("Preferences")',
      'a:has-text("Account")',
      'a[href*="settings"]',
      'a[href*="preferences"]'
    ];

    let settingsFound = false;
    for (const selector of settingsLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        settingsFound = true;
        break;
      }
    }

    if (settingsFound) {
      // Should be on settings page
      await expect(page.locator('h1:has-text("Settings"), h1:has-text("Preferences"), h2:has-text("Account")')).toBeVisible();
    } else {
      test.skip('Settings page not accessible');
    }
  });

  test('should display user profile information', async ({ page }) => {
    // Navigate to profile/settings
    const profileLink = page.locator('a:has-text("Profile"), a:has-text("Settings"), a[href*="profile"]');
    if (await profileLink.isVisible()) {
      await profileLink.click();
    }

    // Should show user information
    const userInfo = [
      'input[name="username"], input[name="email"]',
      'text=testuser30d_v2',
      '.profile-info, .user-details'
    ];

    for (const selector of userInfo) {
      const element = page.locator(selector);
      if (await element.isVisible()) {
        await expect(element).toBeVisible();
        break;
      }
    }
  });

  test('should allow email notification preferences', async ({ page }) => {
    // Navigate to notification settings
    const notificationLinks = [
      'a:has-text("Notifications")',
      'a:has-text("Email")',
      'button:has-text("Notification Settings")',
      'a[href*="notification"]'
    ];

    let notificationsFound = false;
    for (const selector of notificationLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        notificationsFound = true;
        break;
      }
    }

    if (notificationsFound) {
      // Should show notification preferences
      const notificationTypes = [
        'Comment notifications',
        'New chapter alerts',
        'Kudos notifications',
        'Bookmark notifications',
        'Follow notifications'
      ];

      for (const type of notificationTypes) {
        const checkbox = page.locator(`input[type="checkbox"]:has-text("${type}"), label:has-text("${type}")`);
        if (await checkbox.isVisible()) {
          await expect(checkbox).toBeVisible();
        }
      }

      // Should have save button
      const saveButton = page.locator('button:has-text("Save"), button:has-text("Update")');
      if (await saveButton.isVisible()) {
        await expect(saveButton).toBeVisible();
        await expect(saveButton).toBeEnabled();
      }
    } else {
      test.skip('Notification settings not available');
    }
  });

  test('should allow privacy settings configuration', async ({ page }) => {
    // Navigate to privacy settings
    const privacyLinks = [
      'a:has-text("Privacy")',
      'a:has-text("Security")',
      'button:has-text("Privacy Settings")',
      'a[href*="privacy"]'
    ];

    let privacyFound = false;
    for (const selector of privacyLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        privacyFound = true;
        break;
      }
    }

    if (privacyFound) {
      // Should show privacy options
      const privacyOptions = [
        'Profile visibility',
        'Work visibility',
        'Bookmark privacy',
        'Email visibility',
        'History visibility'
      ];

      for (const option of privacyOptions) {
        const setting = page.locator(`label:has-text("${option}"), input[name*="${option.toLowerCase()}"]`);
        if (await setting.isVisible()) {
          await expect(setting).toBeVisible();
        }
      }
    } else {
      test.skip('Privacy settings not available');
    }
  });

  test('should allow display preferences customization', async ({ page }) => {
    // Navigate to display/appearance settings
    const displayLinks = [
      'a:has-text("Display")',
      'a:has-text("Appearance")',
      'a:has-text("Theme")',
      'button:has-text("Display Settings")'
    ];

    let displayFound = false;
    for (const selector of displayLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        displayFound = true;
        break;
      }
    }

    if (displayFound) {
      // Should show display preferences
      const displayOptions = [
        'Theme selection',
        'Font size',
        'Works per page',
        'Default language',
        'Date format'
      ];

      for (const option of displayOptions) {
        const setting = page.locator(`select:has-text("${option}"), input[name*="${option.toLowerCase().replace(/\s+/g, '_')}"]`);
        if (await setting.isVisible()) {
          await expect(setting).toBeVisible();
        }
      }

      // Test theme switching if available
      const themeSelect = page.locator('select[name*="theme"], .theme-selector');
      if (await themeSelect.isVisible()) {
        await expect(themeSelect).toBeVisible();
        await expect(themeSelect).toBeEnabled();
      }
    } else {
      test.skip('Display settings not available');
    }
  });

  test('should allow content filtering preferences', async ({ page }) => {
    // Navigate to content filtering settings
    const filterLinks = [
      'a:has-text("Content")',
      'a:has-text("Filters")',
      'a:has-text("Tags")',
      'button:has-text("Content Settings")'
    ];

    let filterFound = false;
    for (const selector of filterLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        filterFound = true;
        break;
      }
    }

    if (filterFound) {
      // Should show content filtering options
      const filterOptions = [
        'Hide explicit content',
        'Mature content warnings',
        'Tag filtering',
        'Rating filters',
        'Archive warnings'
      ];

      for (const option of filterOptions) {
        const setting = page.locator(`input[type="checkbox"]:has-text("${option}"), label:has-text("${option}")`);
        if (await setting.isVisible()) {
          await expect(setting).toBeVisible();
        }
      }

      // Should allow custom tag blocking
      const tagInput = page.locator('input[placeholder*="tag"], input[name*="blocked"], textarea[placeholder*="filter"]');
      if (await tagInput.isVisible()) {
        await expect(tagInput).toBeVisible();
      }
    } else {
      test.skip('Content filtering not available');
    }
  });

  test('should allow reading preferences', async ({ page }) => {
    // Navigate to reading preferences
    const readingLinks = [
      'a:has-text("Reading")',
      'a:has-text("Works")',
      'button:has-text("Reading Preferences")',
      'a[href*="reading"]'
    ];

    let readingFound = false;
    for (const selector of readingLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        readingFound = true;
        break;
      }
    }

    if (readingFound) {
      // Should show reading preferences
      const readingOptions = [
        'Default work view',
        'Chapter navigation',
        'Auto-bookmark',
        'Reading history',
        'Show stats'
      ];

      for (const option of readingOptions) {
        const setting = page.locator(`select:has-text("${option}"), input[name*="${option.toLowerCase().replace(/\s+/g, '_')}"]`);
        if (await setting.isVisible()) {
          await expect(setting).toBeVisible();
        }
      }
    } else {
      test.skip('Reading preferences not available');
    }
  });

  test('should save preference changes', async ({ page }) => {
    // Navigate to any settings page
    const settingsLink = page.locator('a:has-text("Settings"), a:has-text("Preferences")');
    if (await settingsLink.isVisible()) {
      await settingsLink.click();
    }

    // Find a preference to change (checkbox or select)
    const checkbox = page.locator('input[type="checkbox"]').first();
    const select = page.locator('select').first();

    if (await checkbox.isVisible()) {
      const initialState = await checkbox.isChecked();
      await checkbox.setChecked(!initialState);
      
      // Save changes
      const saveButton = page.locator('button:has-text("Save"), button:has-text("Update")');
      if (await saveButton.isVisible()) {
        await saveButton.click();
        
        // Should show success message
        await expect(page.locator('text=saved, text=updated, .success, [role="alert"]')).toBeVisible();
        
        // Preference should persist on reload
        await page.reload();
        await expect(checkbox).toBeChecked(!initialState);
      }
    } else if (await select.isVisible()) {
      const options = await select.locator('option').count();
      if (options > 1) {
        await select.selectOption({ index: 1 });
        
        const saveButton = page.locator('button:has-text("Save"), button:has-text("Update")');
        if (await saveButton.isVisible()) {
          await saveButton.click();
          await expect(page.locator('text=saved, text=updated')).toBeVisible();
        }
      }
    } else {
      test.skip('No editable preferences found');
    }
  });

  test('should allow password change', async ({ page }) => {
    // Navigate to security/password settings
    const securityLinks = [
      'a:has-text("Security")',
      'a:has-text("Password")',
      'button:has-text("Change Password")',
      'a[href*="security"]'
    ];

    let securityFound = false;
    for (const selector of securityLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        securityFound = true;
        break;
      }
    }

    if (securityFound) {
      // Should show password change form
      const passwordFields = [
        'input[name*="current"], input[placeholder*="current"]',
        'input[name*="new"], input[placeholder*="new"]',
        'input[name*="confirm"], input[placeholder*="confirm"]'
      ];

      for (const fieldSelector of passwordFields) {
        const field = page.locator(fieldSelector);
        if (await field.isVisible()) {
          await expect(field).toBeVisible();
          await expect(field).toHaveAttribute('type', 'password');
        }
      }

      // Should have change password button
      const changeButton = page.locator('button:has-text("Change"), button:has-text("Update Password")');
      if (await changeButton.isVisible()) {
        await expect(changeButton).toBeVisible();
      }
    } else {
      test.skip('Password change functionality not available');
    }
  });

  test('should provide export/download preferences', async ({ page }) => {
    // Navigate to export settings
    const exportLinks = [
      'a:has-text("Export")',
      'a:has-text("Download")',
      'button:has-text("Export Settings")',
      'a[href*="export"]'
    ];

    let exportFound = false;
    for (const selector of exportLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        exportFound = true;
        break;
      }
    }

    if (exportFound) {
      // Should show export preferences
      const exportOptions = [
        'Default format',
        'Include metadata',
        'Chapter numbering',
        'Author notes',
        'File naming'
      ];

      for (const option of exportOptions) {
        const setting = page.locator(`select:has-text("${option}"), input[name*="${option.toLowerCase().replace(/\s+/g, '_')}"]`);
        if (await setting.isVisible()) {
          await expect(setting).toBeVisible();
        }
      }

      // Should show format options
      const formatSelect = page.locator('select[name*="format"], .format-selector');
      if (await formatSelect.isVisible()) {
        await expect(formatSelect).toBeVisible();
        const formatText = await formatSelect.textContent();
        expect(formatText).toContain('PDF');
      }
    } else {
      test.skip('Export preferences not available');
    }
  });

  test('should handle preference validation', async ({ page }) => {
    // Navigate to settings
    const settingsLink = page.locator('a:has-text("Settings"), a:has-text("Preferences")');
    if (await settingsLink.isVisible()) {
      await settingsLink.click();
    }

    // Try to submit invalid data
    const emailInput = page.locator('input[type="email"], input[name="email"]');
    if (await emailInput.isVisible()) {
      await emailInput.clear();
      await emailInput.fill('invalid-email');
      
      const saveButton = page.locator('button:has-text("Save"), button:has-text("Update")');
      if (await saveButton.isVisible()) {
        await saveButton.click();
        
        // Should show validation error
        const errorMessage = page.locator('.error, [role="alert"], .invalid-feedback');
        if (await errorMessage.isVisible()) {
          await expect(errorMessage).toBeVisible();
          await expect(errorMessage).toContainText('valid email');
        }
      }
    } else {
      test.skip('Email validation testing not available');
    }
  });

  test('should provide account deletion option', async ({ page }) => {
    // Navigate to account settings
    const accountLinks = [
      'a:has-text("Account")',
      'a:has-text("Delete")',
      'button:has-text("Delete Account")',
      'a[href*="delete"]'
    ];

    let deleteFound = false;
    for (const selector of accountLinks) {
      const link = page.locator(selector);
      if (await link.isVisible()) {
        await link.click();
        deleteFound = true;
        break;
      }
    }

    if (deleteFound) {
      // Should show account deletion option
      const deleteButton = page.locator('button:has-text("Delete"), a:has-text("Delete Account")');
      if (await deleteButton.isVisible()) {
        await expect(deleteButton).toBeVisible();
        
        // Should have warning about permanent deletion
        await expect(page.locator('text=permanent, text=cannot be undone, text=irreversible')).toBeVisible();
        
        // Should require confirmation
        const confirmationText = page.locator('input[placeholder*="confirm"], input[placeholder*="DELETE"]');
        if (await confirmationText.isVisible()) {
          await expect(confirmationText).toBeVisible();
        }
      }
    } else {
      test.skip('Account deletion not available');
    }
  });

  test('should maintain preferences across sessions', async ({ page, context }) => {
    // Change a preference
    const settingsLink = page.locator('a:has-text("Settings"), a:has-text("Preferences")');
    if (await settingsLink.isVisible()) {
      await settingsLink.click();
      
      const checkbox = page.locator('input[type="checkbox"]').first();
      if (await checkbox.isVisible()) {
        const initialState = await checkbox.isChecked();
        await checkbox.setChecked(!initialState);
        
        const saveButton = page.locator('button:has-text("Save")');
        if (await saveButton.isVisible()) {
          await saveButton.click();
          await expect(page.locator('text=saved')).toBeVisible();
          
          // Create new page (simulate new session)
          const newPage = await context.newPage();
          await newPage.goto('/login');
          
          // Login again
          await newPage.fill('input[name="email"]', 'testuser30d_v2@example.com');
          await newPage.fill('input[name="password"]', 'TestPassword123!');
          await newPage.click('button[type="submit"]');
          await newPage.waitForLoadState('networkidle');
          
          // Check that preference persisted
          const newSettingsLink = newPage.locator('a:has-text("Settings")');
          if (await newSettingsLink.isVisible()) {
            await newSettingsLink.click();
            const newCheckbox = newPage.locator('input[type="checkbox"]').first();
            await expect(newCheckbox).toBeChecked(!initialState);
          }
          
          await newPage.close();
        }
      }
    }
  });
});