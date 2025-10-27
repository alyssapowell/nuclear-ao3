import { test, expect } from '@playwright/test';

// Helper function to authenticate user
async function authenticateUser(page) {
  // Set auth token in localStorage and cookies to simulate logged-in user
  await page.addInitScript(() => {
    localStorage.setItem('auth_token', 'test-token-12345');
    localStorage.setItem('user', JSON.stringify({
      id: 'test-user-123',
      username: 'testuser',
      email: 'test@example.com'
    }));
  });
  
  // Also set the cookie that middleware checks
  await page.context().addCookies([{
    name: 'auth_token',
    value: 'test-token-12345',
    domain: 'localhost',
    path: '/'
  }]);
}

test.describe('New Posts View - Unauthenticated', () => {
  test('should redirect to login when accessing /works/new without authentication', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works/new');
    
    // Should be redirected to login page
    await page.waitForSelector('main h1', { timeout: 15000 });
    await expect(page.locator('main h1')).toContainText('Log In');
    
    // Should have redirect parameter
    expect(page.url()).toContain('/auth/login');
    expect(page.url()).toContain('redirect=%2Fworks%2Fnew');
  });
});

test.describe('New Posts View - Authenticated', () => {
  test.beforeEach(async ({ page }) => {
    await authenticateUser(page);
  });

  test('should load new work creation page when authenticated', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works/new');
    
    // Wait for the page to load
    await page.waitForSelector('main h1', { timeout: 15000 });
    
    // Check if the page has the expected title
    await expect(page.locator('main h1')).toContainText('Post New Work');
    
    // Check for basic form elements
    await expect(page.locator('input[name="title"]')).toBeVisible();
    await expect(page.locator('textarea[name="summary"]')).toBeVisible();
    
    // Check for submit button
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('should show form fields in new work creation', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works/new');
    
    // Wait for the page to load
    await page.waitForSelector('main h1', { timeout: 15000 });
    
    // Check for required fields
    await expect(page.locator('input[name="title"]')).toBeVisible();
    await expect(page.locator('textarea[name="summary"]')).toBeVisible();
    
    // Check for RichTextEditor content area (replaced textarea)
    await expect(page.locator('.ProseMirror')).toBeVisible();
    
    // Check for all TagAutocomplete fields
    await expect(page.locator('input[id="fandoms"]')).toBeVisible();
    await expect(page.locator('input[id="characters"]')).toBeVisible();
    await expect(page.locator('input[id="relationships"]')).toBeVisible();
    await expect(page.locator('input[id="freeformTags"]')).toBeVisible();
    
    // Check for new AO3-style form fields
    // Archive Warnings - use heading selector
    await expect(page.locator('label:has-text("Archive Warnings *")').first()).toBeVisible();
    await expect(page.locator('label:has-text("No Archive Warnings Apply")')).toBeVisible();
    await expect(page.locator('label:has-text("Creator Chose Not To Use Archive Warnings")')).toBeVisible();
    await expect(page.locator('label:has-text("Graphic Depictions Of Violence")')).toBeVisible();
    await expect(page.locator('label:has-text("Major Character Death")')).toBeVisible();
    await expect(page.locator('label:has-text("Rape/Non-Con")')).toBeVisible();
    await expect(page.locator('label:has-text("Underage")')).toBeVisible();
    
    // Categories - use label selector
    await expect(page.locator('label:has-text("Categories")')).toBeVisible();
    await expect(page.locator('label:has-text("Gen")')).toBeVisible();
    await expect(page.locator('label:has-text("M/M")')).toBeVisible();
    await expect(page.locator('label:has-text("F/F")')).toBeVisible();
    await expect(page.locator('label:has-text("M/F")')).toBeVisible();
    await expect(page.locator('label:has-text("Multi")')).toBeVisible();
    await expect(page.locator('label:has-text("Other")')).toBeVisible();
    
    // Work metadata fields
    await expect(page.locator('select[name="rating"]')).toBeVisible();
    await expect(page.locator('select[name="language"]')).toBeVisible();
    await expect(page.locator('input[name="chapterTitle"]')).toBeVisible();
    await expect(page.locator('input[name="maxChapters"]')).toBeVisible();
    
    // The enhanced form should have these elements
    const pageContent = await page.textContent('body');
    expect(pageContent).toContain('Post New Work');
    expect(pageContent).toContain('Fandoms');
    expect(pageContent).toContain('Characters');
    expect(pageContent).toContain('Relationships');
    expect(pageContent).toContain('Additional Tags');
  });

  test('should allow typing in TagAutocomplete field', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works/new');
    
    // Wait for the page to load
    await page.waitForSelector('input[id="fandoms"]', { timeout: 15000 });
    
    // Type in the TagAutocomplete field
    await page.fill('input[id="fandoms"]', 'Harry Potter');
    
    // Verify the text was entered
    const inputValue = await page.inputValue('input[id="fandoms"]');
    expect(inputValue).toBe('Harry Potter');
  });

  test('should handle real API calls and show network activity', async ({ page }) => {
    test.setTimeout(30000);
    
    // Intercept API calls to see what's actually happening
    const apiCalls: string[] = [];
    await page.route('**/api/v1/tags/search*', async route => {
      const url = route.request().url();
      apiCalls.push(url);
      console.log(`API call intercepted: ${url}`);
      
      // Let the real API call go through
      await route.continue();
    });
    
    await page.goto('/works/new');
    
    // Wait for the page to load
    await page.waitForSelector('input[id="characters"]', { timeout: 15000 });
    
    // Type something that should trigger autocomplete in character field (we have character data)
    await page.type('input[id="characters"]', 'Harry');
    
    // Wait for API call and debounce (TagAutocomplete has 150ms debounce)
    await page.waitForTimeout(1000);
    
    // Check if API calls were made
    console.log(`Total API calls made: ${apiCalls.length}`);
    apiCalls.forEach(call => console.log(`  - ${call}`));
    
    // Look for the suggestions listbox
    const suggestionsListbox = page.locator('[role="listbox"]');
    const suggestionsVisible = await suggestionsListbox.isVisible().catch(() => false);
    
    if (suggestionsVisible) {
      console.log('✅ Autocomplete suggestions are working with real data!');
      const suggestionOptions = page.locator('[role="option"]');
      const optionCount = await suggestionOptions.count();
      console.log(`Found ${optionCount} real suggestions`);
    } else {
      console.log('ℹ️ No suggestions shown - either no backend data or API not responding');
    }
    
    // Verify the input functionality works regardless
    const inputValue = await page.inputValue('input[id="characters"]');
    expect(inputValue).toBe('Harry');
    expect(apiCalls.length).toBeGreaterThan(0); // Should have made at least one API call
  });

  test('should restrict fandom creation but allow other tag creation', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works/new');
    await page.waitForSelector('input[id="fandoms"]', { timeout: 15000 });
    
    // Test fandom field - should NOT allow creating new tags
    await page.fill('input[id="fandoms"]', 'NonExistentFandom123');
    await page.press('input[id="fandoms"]', 'Enter');
    
    // Should NOT create a tag chip for non-existent fandom
    const fandomChips = page.locator('span:has-text("NonExistentFandom123")');
    await expect(fandomChips).toHaveCount(0);
    
    // Test character field - SHOULD allow creating new tags
    await page.fill('input[id="characters"]', 'NewCharacter123');
    await page.press('input[id="characters"]', 'Enter');
    
    // Should create a tag chip for new character
    const characterChips = page.locator('span:has-text("NewCharacter123")');
    await expect(characterChips).toHaveCount(1);
    
    // Test relationship field - SHOULD allow creating new tags
    await page.fill('input[id="relationships"]', 'Character A/Character B');
    await page.press('input[id="relationships"]', 'Enter');
    
    // Should create a tag chip for new relationship
    const relationshipChips = page.locator('span:has-text("Character A/Character B")');
    await expect(relationshipChips).toHaveCount(1);
    
    // Test freeform tags - SHOULD allow creating new tags
    await page.fill('input[id="freeformTags"]', 'Custom Freeform Tag');
    await page.press('input[id="freeformTags"]', 'Enter');
    
    // Should create a tag chip for new freeform tag
    const freeformChips = page.locator('span:has-text("Custom Freeform Tag")');
    await expect(freeformChips).toHaveCount(1);
    
    console.log('✅ Tag creation restrictions working correctly');
  });

  test('should show helpful message for fandom restrictions', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works/new');
    await page.waitForSelector('input[id="fandoms"]', { timeout: 15000 });
    
    // Type in fandom field with non-existent fandom
    await page.fill('input[id="fandoms"]', 'NonExistentFandom');
    
    // Wait for suggestions to load (or not load)
    await page.waitForTimeout(500);
    
    // Check for helpful message about fandom restrictions
    const helpText = page.locator('text=choose from existing fandoms or contact admin');
    const isHelpVisible = await helpText.isVisible().catch(() => false);
    
    if (isHelpVisible) {
      console.log('✅ Helpful fandom restriction message shown');
    } else {
      console.log('ℹ️ No restriction message shown (may have found existing fandoms)');
    }
    
    // Verify fandom field description mentions restriction
    const fandomDescription = page.locator('text=Choose from existing fandoms only');
    await expect(fandomDescription).toBeVisible();
  });

  test('should work with all new AO3 form fields', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works/new');
    await page.waitForSelector('input[name="title"]', { timeout: 15000 });
    
    // Fill out basic info
    await page.fill('input[name="title"]', 'Test Enhanced Form Work');
    await page.fill('textarea[name="summary"]', 'This is a test summary for the enhanced form.');
    
    // Test RichTextEditor
    const richEditor = page.locator('.ProseMirror');
    await richEditor.click();
    await richEditor.fill('This is test chapter content with rich text formatting.');
    
    // Test bold button in RichTextEditor
    await page.locator('button[title="Bold"]').click();
    await richEditor.type(' Bold text here.');
    
    // Select rating
    await page.selectOption('select[name="rating"]', 'teen');
    
    // Select language
    await page.selectOption('select[name="language"]', 'en');
    
    // Check some archive warnings
    await page.check('input[type="checkbox"]:near(:text("No Archive Warnings Apply"))');
    
    // Check some categories
    await page.check('input[type="checkbox"]:near(:text("Gen"))');
    await page.check('input[type="checkbox"]:near(:text("M/M"))');
    
    // Fill chapter info
    await page.fill('input[name="chapterTitle"]', 'Chapter 1: The Beginning');
    await page.fill('input[name="maxChapters"]', '5');
    
    // Add some tags (using existing character that should work)
    await page.fill('input[id="characters"]', 'TestCharacter');
    await page.press('input[id="characters"]', 'Enter');
    
    await page.fill('input[id="freeformTags"]', 'Fluff');
    await page.press('input[id="freeformTags"]', 'Enter');
    
    // Verify form state
    expect(await page.inputValue('input[name="title"]')).toBe('Test Enhanced Form Work');
    expect(await page.inputValue('select[name="rating"]')).toBe('teen');
    expect(await page.inputValue('input[name="chapterTitle"]')).toBe('Chapter 1: The Beginning');
    
    // Verify tags were added
    await expect(page.locator('span:has-text("TestCharacter")')).toBeVisible();
    await expect(page.locator('span:has-text("Fluff")')).toBeVisible();
    
    console.log('✅ Enhanced form fields working correctly');
  });
});

test.describe('Works Browse Page', () => {
  test('should load works browse page', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works');
    
    // Wait for the page to load
    await page.waitForSelector('body', { timeout: 15000 });
    
    // Should have some content indicating it's the works page
    const pageContent = await page.textContent('body');
    expect(pageContent).toBeTruthy();
    
    // Should not redirect to login (works browse should be public)
    expect(page.url()).toContain('/works');
    expect(page.url()).not.toContain('/auth/login');
  });
});