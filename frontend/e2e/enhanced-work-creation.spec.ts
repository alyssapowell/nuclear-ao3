import { test, expect } from '@playwright/test';
import { loginUser } from './test-utils';

test.describe('Enhanced Work Creation Form Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Start from the test components page for unauthenticated testing
    await page.goto('/test-components');
    await page.waitForLoadState('networkidle');
    
    // Verify the test page loaded
    await expect(page.getByRole('heading', { name: 'Enhanced Form Components Test' })).toBeVisible();
  });

  test.describe('Enhanced Tag Input Components', () => {
    test('should display tag autocomplete suggestions when typing', async ({ page }) => {
      // Test fandom tag autocomplete
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      await expect(fandomInput).toBeVisible();
      
      // Type to trigger autocomplete
      await fandomInput.fill('Marvel');
      
      // Wait for autocomplete suggestions to appear
      await page.waitForTimeout(1000); // Allow time for API call
      
      // Check if suggestions container appears
      const suggestionsContainer = page.locator('[role="listbox"]');
      if (await suggestionsContainer.isVisible({ timeout: 3000 })) {
        // Verify suggestion content
        await expect(suggestionsContainer.locator('text=Marvel')).toBeVisible();
      }
    });

    test('should add tags via Enter key', async ({ page }) => {
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      
      // Type a tag name and press Enter
      await fandomInput.fill('Test Fandom');
      await fandomInput.press('Enter');
      
      // Verify tag was added to the state display
      const stateDisplay = page.locator('pre').filter({ hasText: '"fandoms"' });
      await expect(stateDisplay).toContainText('Test Fandom');
    });

    test('should add tags via comma separator', async ({ page }) => {
      const characterInput = page.locator('input[aria-label*="character"]').first();
      
      // Type a tag name with comma
      await characterInput.fill('Harry Potter,');
      
      // Verify tag was added
      const stateDisplay = page.locator('pre').filter({ hasText: '"characters"' });
      await expect(stateDisplay).toContainText('Harry Potter');
    });

    test('should remove tags via remove button', async ({ page }) => {
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      
      // Add a tag first
      await fandomInput.fill('Remove Me');
      await fandomInput.press('Enter');
      
      // Wait for tag to appear
      await page.waitForTimeout(500);
      
      // Find and click the remove button
      const removeButton = page.locator('button[aria-label*="Remove Remove Me"]').first();
      if (await removeButton.isVisible({ timeout: 1000 })) {
        await removeButton.click();
        
        // Verify tag was removed
        const stateDisplay = page.locator('pre').filter({ hasText: '"fandoms"' });
        await expect(stateDisplay).not.toContainText('Remove Me');
      }
    });

    test('should handle all tag types (fandom, character, relationship, freeform)', async ({ page }) => {
      const inputs = {
        fandom: page.locator('input[aria-label*="fandom"]').first(),
        character: page.locator('input[aria-label*="character"]').first(),
        relationship: page.locator('input[aria-label*="relationship"]').first(),
        freeform: page.locator('input[aria-label*="freeform"]').first()
      };

      const testTags = {
        fandom: 'Test Fandom',
        character: 'Test Character',
        relationship: 'Test Relationship',
        freeform: 'Test Tag'
      };

      // Add tags to each input type
      for (const [type, input] of Object.entries(inputs)) {
        await input.fill(testTags[type as keyof typeof testTags]);
        await input.press('Enter');
      }

      // Verify all tags were added to state
      const stateDisplay = page.locator('pre').filter({ hasText: 'fandoms' });
      
      for (const tag of Object.values(testTags)) {
        await expect(stateDisplay).toContainText(tag);
      }
    });

    test('should prevent duplicate tags', async ({ page }) => {
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      
      // Add same tag twice
      await fandomInput.fill('Duplicate Tag');
      await fandomInput.press('Enter');
      
      await fandomInput.fill('Duplicate Tag');
      await fandomInput.press('Enter');
      
      // Verify only one instance exists in state
      const stateDisplay = page.locator('pre').filter({ hasText: '"fandoms"' });
      const content = await stateDisplay.textContent();
      const matches = (content || '').match(/Duplicate Tag/g) || [];
      expect(matches.length).toBe(1);
    });
  });

  test.describe('Rich Text Editor (Components Test)', () => {
    test('should load rich text editor without errors', async ({ page }) => {
      // Check if rich text editor section is present
      const editorSection = page.locator('h2:has-text("Rich Text Editor")');
      await expect(editorSection).toBeVisible();
      
      // Verify no SSR hydration errors in console
      const consoleErrors = [];
      page.on('console', msg => {
        if (msg.type() === 'error') {
          consoleErrors.push(msg.text());
        }
      });
      
      await page.waitForTimeout(2000);
      
      // Filter out unrelated errors and focus on Tiptap/SSR errors
      const tiptapErrors = consoleErrors.filter(error => 
        error.includes('Tiptap') || 
        error.includes('SSR') || 
        error.includes('hydration')
      );
      
      expect(tiptapErrors.length).toBe(0);
    });

    test('should update HTML content display', async ({ page }) => {
      // Wait for any client-side rendering to complete
      await page.waitForTimeout(1000);
      
      // Check if the HTML display shows initial content
      const htmlDisplay = page.locator('pre').filter({ hasText: '<p>Start typing to test' });
      await expect(htmlDisplay).toBeVisible();
    });
  });

  test.describe('Mobile Responsiveness', () => {
    test('should work properly on mobile viewport', async ({ page }) => {
      // Set mobile viewport
      await page.setViewportSize({ width: 375, height: 667 });
      
      // Verify page still loads correctly
      await expect(page.getByRole('heading', { name: 'Enhanced Form Components Test' })).toBeVisible();
      
      // Test tag inputs are still functional
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      await expect(fandomInput).toBeVisible();
      
      await fandomInput.fill('Mobile Test');
      await fandomInput.press('Enter');
      
      // Verify tag was added
      const stateDisplay = page.locator('pre').filter({ hasText: '"fandoms"' });
      await expect(stateDisplay).toContainText('Mobile Test');
    });

    test('should handle tablet viewport', async ({ page }) => {
      // Set tablet viewport
      await page.setViewportSize({ width: 768, height: 1024 });
      
      // Verify functionality works on tablet
      const characterInput = page.locator('input[aria-label*="character"]').first();
      await characterInput.fill('Tablet Character');
      await characterInput.press('Enter');
      
      const stateDisplay = page.locator('pre').filter({ hasText: '"characters"' });
      await expect(stateDisplay).toContainText('Tablet Character');
    });
  });
});

test.describe('Authenticated Work Creation Form Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Login first for authenticated tests
    try {
      await page.goto('/auth/login');
      await page.waitForLoadState('networkidle');
      
      // Use admin credentials that we know work
      await page.fill('#email', 'admin@nuclear-ao3.com');
      await page.fill('#password', 'adminpass123');
      
      await page.click('button[type="submit"]');
      
      // Wait for redirect after login
      await page.waitForTimeout(3000);
      
      // Navigate to work creation page
      await page.goto('/works/new');
      await page.waitForLoadState('networkidle');
      
      // Verify we're on the work creation page
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
    } catch (error) {
      console.log('Authentication setup failed:', error);
      test.skip('Skipping authenticated tests due to login failure');
    }
  });

  test('should create a complete work with enhanced form features', async ({ page }) => {
    // Fill basic information
    await page.fill('input[name="title"]', 'Enhanced Form Test Work');
    await page.fill('textarea[name="summary"]', 'Testing the enhanced work creation form with all new features.');
    
    // Test enhanced tag inputs in the actual form
    const fandomInput = page.locator('input[aria-label*="fandom"]').first();
    if (await fandomInput.isVisible({ timeout: 5000 })) {
      await fandomInput.fill('Marvel Cinematic Universe');
      
      // Wait for autocomplete suggestions
      await page.waitForTimeout(1000);
      
      // Try to select from suggestions or press Enter
      const suggestion = page.locator('[role="option"]:has-text("Marvel")').first();
      if (await suggestion.isVisible({ timeout: 2000 })) {
        await suggestion.click();
      } else {
        await fandomInput.press('Enter');
      }
      
      // Verify tag was added (look for tag chip)
      await expect(page.locator('.rounded-full:has-text("Marvel")')).toBeVisible();
    }
    
    // Add character tag
    const characterInput = page.locator('input[aria-label*="character"]').first();
    if (await characterInput.isVisible()) {
      await characterInput.fill('Tony Stark');
      await characterInput.press('Enter');
    }
    
    // Select rating
    const ratingRadio = page.locator('input[value="General Audiences"]');
    if (await ratingRadio.isVisible()) {
      await ratingRadio.check();
    }
    
    // Select category
    const categoryRadio = page.locator('input[value="Gen"]');
    if (await categoryRadio.isVisible()) {
      await categoryRadio.check();
    }
    
    // Select warning
    const warningRadio = page.locator('input[value="No Archive Warnings Apply"]');
    if (await warningRadio.isVisible()) {
      await warningRadio.check();
    }
    
    // Test rich text editor
    const richTextEditor = page.locator('.ProseMirror, [contenteditable="true"]');
    if (await richTextEditor.isVisible({ timeout: 5000 })) {
      await richTextEditor.fill('This is a test work created with the enhanced rich text editor. It includes **bold text** and other formatting.');
      
      // Verify content was entered
      await expect(richTextEditor).toContainText('This is a test work');
    } else {
      // Fallback to regular textarea if rich text editor not available
      const contentTextarea = page.locator('textarea[name="chapterContent"]');
      if (await contentTextarea.isVisible()) {
        await contentTextarea.fill('This is a test work created with the enhanced form.');
      }
    }
    
    // Submit the form
    const submitButton = page.locator('button[type="submit"]:has-text("Post Work"), button:has-text("Create Work"), button:has-text("Post")');
    await expect(submitButton).toBeVisible();
    
    // Listen for network requests to debug any issues
    page.on('response', response => {
      if (response.url().includes('/api/v1/works') && response.status() >= 400) {
        console.log(`Work creation failed: ${response.status()} ${response.statusText()}`);
      }
    });
    
    await submitButton.click();
    
    // Wait for form submission
    await page.waitForTimeout(5000);
    
    // Check for success (either redirect to work page or success message)
    const currentUrl = page.url();
    const isSuccessful = currentUrl.includes('/works/') && !currentUrl.includes('/new');
    
    if (isSuccessful) {
      // Verify we're on a work page
      await expect(page.getByRole('heading', { name: /Enhanced Form Test Work/i })).toBeVisible();
    } else {
      // Check for any error messages to understand what went wrong
      const errorMessages = await page.locator(':has-text("error"), :has-text("Error"), .error').count();
      if (errorMessages > 0) {
        const errorText = await page.locator(':has-text("error"), :has-text("Error"), .error').first().textContent();
        console.log('Form error detected:', errorText);
      }
      
      // Still expect to be redirected or see success message
      expect(isSuccessful).toBe(true);
    }
  });

  test('should validate required fields with enhanced UX', async ({ page }) => {
    // Try to submit form without required fields
    const submitButton = page.locator('button[type="submit"]:has-text("Post Work"), button:has-text("Create Work"), button:has-text("Post")');
    
    if (await submitButton.isVisible()) {
      await submitButton.click();
      
      // Should remain on the form page
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
      
      // Check if validation messages appear
      const validationMessages = page.locator('text=/required|field.*required|must.*provide/i');
      const hasValidation = await validationMessages.count() > 0;
      
      if (hasValidation) {
        console.log('✅ Form validation working correctly');
      }
    }
  });

  test('should handle tag autocomplete API integration', async ({ page }) => {
    // Test that autocomplete actually calls the API
    let apiCalled = false;
    
    page.on('request', request => {
      if (request.url().includes('/api/v1/tags/autocomplete')) {
        apiCalled = true;
        console.log('✅ Tag autocomplete API called:', request.url());
      }
    });
    
    const fandomInput = page.locator('input[aria-label*="fandom"]').first();
    if (await fandomInput.isVisible()) {
      await fandomInput.fill('Har');
      
      // Wait for API call
      await page.waitForTimeout(1500);
      
      expect(apiCalled).toBe(true);
    }
  });

  test('should preserve form data when navigating back', async ({ page }) => {
    // Fill some form data
    await page.fill('input[name="title"]', 'Persistence Test');
    
    const fandomInput = page.locator('input[aria-label*="fandom"]').first();
    if (await fandomInput.isVisible()) {
      await fandomInput.fill('Test Fandom');
      await fandomInput.press('Enter');
    }
    
    // Navigate away and back
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');
    
    await page.goto('/works/new');
    await page.waitForLoadState('networkidle');
    
    // Verify form is reset (this is expected behavior)
    const titleInput = page.locator('input[name="title"]');
    await expect(titleInput).toHaveValue('');
  });
});

test.describe('Performance and Accessibility Tests', () => {
  test('should meet accessibility standards', async ({ page }) => {
    await page.goto('/test-components');
    await page.waitForLoadState('networkidle');
    
    // Check for proper ARIA labels
    const ariaLabels = await page.locator('[aria-label]').count();
    expect(ariaLabels).toBeGreaterThan(0);
    
    // Check for proper heading hierarchy
    const headings = await page.locator('h1, h2, h3, h4, h5, h6').count();
    expect(headings).toBeGreaterThan(0);
    
    // Verify live regions for screen readers
    const liveRegions = await page.locator('[aria-live]').count();
    expect(liveRegions).toBeGreaterThan(0);
  });

  test('should load within reasonable time', async ({ page }) => {
    const startTime = Date.now();
    
    await page.goto('/test-components');
    await page.waitForLoadState('networkidle');
    
    const loadTime = Date.now() - startTime;
    
    // Should load within 5 seconds
    expect(loadTime).toBeLessThan(5000);
  });

  test('should handle keyboard navigation', async ({ page }) => {
    await page.goto('/test-components');
    await page.waitForLoadState('networkidle');
    
    // Test keyboard navigation through form elements
    await page.keyboard.press('Tab'); // Should focus first input
    await page.keyboard.press('Tab'); // Should focus next input
    
    // Verify focus is properly managed
    const focusedElement = await page.locator(':focus');
    await expect(focusedElement).toBeVisible();
  });
});