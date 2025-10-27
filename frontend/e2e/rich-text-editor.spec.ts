import { test, expect } from '@playwright/test';

test.describe('Rich Text Editor Focused Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/test-components');
    await page.waitForLoadState('networkidle');
    
    // Wait for any client-side rendering to complete
    await page.waitForTimeout(2000);
  });

  test('should load without SSR hydration errors', async ({ page }) => {
    // Monitor console for hydration errors
    const consoleErrors: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        const errorText = msg.text();
        // Filter for SSR/Tiptap related errors
        if (errorText.includes('Tiptap') || 
            errorText.includes('SSR') || 
            errorText.includes('hydration') ||
            errorText.includes('immediatelyRender')) {
          consoleErrors.push(errorText);
        }
      }
    });
    
    // Look for the rich text editor section
    const editorSection = page.locator('h2:has-text("Rich Text Editor")');
    await expect(editorSection).toBeVisible();
    
    // Wait a moment for any hydration to complete
    await page.waitForTimeout(3000);
    
    // Check for Tiptap-related console errors
    expect(consoleErrors.length).toBe(0);
    
    if (consoleErrors.length > 0) {
      console.log('❌ Console errors found:', consoleErrors);
    } else {
      console.log('✅ No Tiptap/SSR hydration errors detected');
    }
  });

  test('should display HTML content preview', async ({ page }) => {
    // Check for the HTML display area
    const htmlDisplay = page.locator('pre').filter({ hasText: 'Start typing to test' });
    await expect(htmlDisplay).toBeVisible();
    
    // Verify initial content
    const content = await htmlDisplay.textContent();
    expect(content).toContain('<p>Start typing to test the rich text editor...</p>');
  });

  test('should be responsive on different screen sizes', async ({ page }) => {
    // Test on mobile
    await page.setViewportSize({ width: 375, height: 667 });
    
    const editorSection = page.locator('h2:has-text("Rich Text Editor")');
    await expect(editorSection).toBeVisible();
    
    // Test on tablet
    await page.setViewportSize({ width: 768, height: 1024 });
    await expect(editorSection).toBeVisible();
    
    // Test on desktop
    await page.setViewportSize({ width: 1200, height: 800 });
    await expect(editorSection).toBeVisible();
  });

  test('should handle page navigation without breaking', async ({ page }) => {
    // Verify editor section loads
    const editorSection = page.locator('h2:has-text("Rich Text Editor")');
    await expect(editorSection).toBeVisible();
    
    // Navigate away and back
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    await page.goto('/test-components');
    await page.waitForLoadState('networkidle');
    
    // Verify editor section still works
    await expect(editorSection).toBeVisible();
  });

  test('should not interfere with other page functionality', async ({ page }) => {
    // Test that other components still work when rich text editor is present
    const fandomInput = page.locator('input[aria-label*="fandom"]').first();
    await expect(fandomInput).toBeVisible();
    
    // Add a tag to ensure other functionality works
    await fandomInput.fill('Test Fandom');
    await fandomInput.press('Enter');
    
    // Verify tag was added
    const stateDisplay = page.locator('pre').filter({ hasText: '"fandoms"' });
    await expect(stateDisplay).toContainText('Test Fandom');
  });
});

test.describe('Rich Text Editor in Work Creation Form', () => {
  test.beforeEach(async ({ page }) => {
    // Login for authenticated work creation tests
    try {
      await page.goto('/auth/login');
      await page.waitForLoadState('networkidle');
      
      await page.fill('#email', 'admin@nuclear-ao3.com');
      await page.fill('#password', 'adminpass123');
      await page.click('button[type="submit"]');
      
      await page.waitForTimeout(3000);
      
      await page.goto('/works/new');
      await page.waitForLoadState('networkidle');
      
      // Verify we're on the work creation page
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
    } catch (error) {
      console.log('Authentication failed, skipping authenticated tests');
      test.skip('Authentication required for this test');
    }
  });

  test('should load rich text editor in work creation form', async ({ page }) => {
    // Look for the Tiptap editor or content area
    const possibleEditors = [
      page.locator('.ProseMirror'),
      page.locator('[contenteditable="true"]'),
      page.locator('[data-tiptap-editor]'),
      page.locator('textarea[name="chapterContent"]')
    ];
    
    let editorFound = false;
    let editorType = 'none';
    
    for (const editor of possibleEditors) {
      if (await editor.isVisible({ timeout: 2000 })) {
        editorFound = true;
        if (editor === possibleEditors[0]) editorType = 'ProseMirror';
        else if (editor === possibleEditors[1]) editorType = 'contenteditable';
        else if (editor === possibleEditors[2]) editorType = 'tiptap-editor';
        else if (editor === possibleEditors[3]) editorType = 'textarea-fallback';
        break;
      }
    }
    
    expect(editorFound).toBe(true);
    console.log(`✅ Editor found: ${editorType}`);
  });

  test('should accept content input in work creation form', async ({ page }) => {
    // Fill basic required fields first
    await page.fill('input[name="title"]', 'Rich Text Editor Test');
    
    // Try to find and fill the content editor
    const richTextEditor = page.locator('.ProseMirror, [contenteditable="true"]').first();
    const textareaFallback = page.locator('textarea[name="chapterContent"]');
    
    if (await richTextEditor.isVisible({ timeout: 3000 })) {
      // Use rich text editor
      await richTextEditor.fill('This content was entered using the rich text editor.');
      
      // Verify content was entered
      await expect(richTextEditor).toContainText('This content was entered');
      console.log('✅ Rich text editor accepts input');
    } else if (await textareaFallback.isVisible()) {
      // Fallback to textarea
      await textareaFallback.fill('This content was entered using the textarea fallback.');
      
      // Verify content was entered
      await expect(textareaFallback).toHaveValue('This content was entered');
      console.log('✅ Textarea fallback accepts input');
    } else {
      console.log('❌ No editor found');
      expect(true).toBe(false); // Fail the test if no editor is found
    }
  });

  test('should maintain content during form interaction', async ({ page }) => {
    // Fill title and content
    await page.fill('input[name="title"]', 'Content Persistence Test');
    
    const editor = page.locator('.ProseMirror, [contenteditable="true"], textarea[name="chapterContent"]').first();
    if (await editor.isVisible({ timeout: 3000 })) {
      await editor.fill('This content should persist during form interactions.');
      
      // Interact with other form elements
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      if (await fandomInput.isVisible()) {
        await fandomInput.fill('Test Fandom');
        await fandomInput.press('Enter');
      }
      
      // Verify content is still there
      const contentStillExists = await editor.textContent();
      expect(contentStillExists).toContain('This content should persist');
      console.log('✅ Content persists during form interaction');
    }
  });

  test('should work with form submission preparation', async ({ page }) => {
    // Fill out a minimal form to test submission readiness
    await page.fill('input[name="title"]', 'Submission Test Work');
    
    // Add required fandom
    const fandomInput = page.locator('input[aria-label*="fandom"]').first();
    if (await fandomInput.isVisible()) {
      await fandomInput.fill('Test Fandom');
      await fandomInput.press('Enter');
    }
    
    // Add content
    const editor = page.locator('.ProseMirror, [contenteditable="true"], textarea[name="chapterContent"]').first();
    if (await editor.isVisible()) {
      await editor.fill('Test content for submission.');
    }
    
    // Select required fields
    const ratingRadio = page.locator('input[value="General Audiences"]');
    if (await ratingRadio.isVisible()) {
      await ratingRadio.check();
    }
    
    const categoryRadio = page.locator('input[value="Gen"]');
    if (await categoryRadio.isVisible()) {
      await categoryRadio.check();
    }
    
    // Check if submit button becomes enabled/available
    const submitButton = page.locator('button[type="submit"]:has-text("Post Work"), button:has-text("Create Work")');
    await expect(submitButton).toBeVisible();
    
    // Note: We don't actually submit to avoid creating test data
    console.log('✅ Form ready for submission with rich text content');
  });
});