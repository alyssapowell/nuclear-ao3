const { test, expect } = require('@playwright/test');

test.describe('Frontend Issues Investigation', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the homepage
    await page.goto('http://localhost:3001');
  });

  test('TagAutocomplete focus behavior investigation', async ({ page }) => {
    console.log('=== INVESTIGATING TAGAUTOCOMPLETE FOCUS ISSUE ===');
    
    // Find a tag input field
    const additionalTagsInput = page.locator('input[placeholder*="Add additional tag"]').first();
    await expect(additionalTagsInput).toBeVisible();
    
    console.log('Found additional tags input field');
    
    // Focus on the input
    await additionalTagsInput.focus();
    console.log('Focused on input field');
    
    // Check if it's focused
    let isFocused = await additionalTagsInput.evaluate(el => el === document.activeElement);
    console.log(`Input is focused after .focus(): ${isFocused}`);
    
    // Type a single character
    await additionalTagsInput.type('h');
    console.log('Typed "h"');
    
    // Wait a moment for any re-renders
    await page.waitForTimeout(200);
    
    // Check if still focused after typing one character
    isFocused = await additionalTagsInput.evaluate(el => el === document.activeElement);
    console.log(`Input is focused after typing "h": ${isFocused}`);
    
    // Check the input value
    const value = await additionalTagsInput.inputValue();
    console.log(`Input value after typing "h": "${value}"`);
    
    // Try typing another character
    await additionalTagsInput.type('e');
    console.log('Typed "e"');
    
    await page.waitForTimeout(200);
    
    // Check focus again
    isFocused = await additionalTagsInput.evaluate(el => el === document.activeElement);
    console.log(`Input is focused after typing "he": ${isFocused}`);
    
    // Check the input value
    const value2 = await additionalTagsInput.inputValue();
    console.log(`Input value after typing "he": "${value2}"`);
    
    // Try typing a third character
    await additionalTagsInput.type('l');
    console.log('Typed "l"');
    
    await page.waitForTimeout(200);
    
    // Final focus check
    isFocused = await additionalTagsInput.evaluate(el => el === document.activeElement);
    console.log(`Input is focused after typing "hel": ${isFocused}`);
    
    const value3 = await additionalTagsInput.inputValue();
    console.log(`Input value after typing "hel": "${value3}"`);
    
    // Check if autocomplete suggestions appear
    const suggestions = page.locator('[role="listbox"]');
    const hasSuggestions = await suggestions.isVisible().catch(() => false);
    console.log(`Autocomplete suggestions visible: ${hasSuggestions}`);
    
    if (hasSuggestions) {
      const suggestionCount = await suggestions.locator('[role="option"]').count();
      console.log(`Number of suggestions: ${suggestionCount}`);
    }
    
    // Take a screenshot for manual inspection
    await page.screenshot({ path: 'frontend/tagautocomplete-focus-test.png', fullPage: true });
    console.log('Screenshot saved: tagautocomplete-focus-test.png');
    
    console.log('=== TAGAUTOCOMPLETE TEST COMPLETE ===');
  });

  test('Work creation flow investigation', async ({ page }) => {
    console.log('=== INVESTIGATING WORK CREATION FLOW ===');
    
    // First, let's see if we can find a work creation form or button
    await page.goto('http://localhost:3001');
    
    // Look for navigation or work creation elements
    const workCreationButton = page.locator('text=Create Work').or(page.locator('text=New Work')).or(page.locator('[href*="/works/new"]')).first();
    const hasWorkCreationButton = await workCreationButton.isVisible().catch(() => false);
    console.log(`Work creation button found: ${hasWorkCreationButton}`);
    
    if (hasWorkCreationButton) {
      await workCreationButton.click();
      console.log('Clicked work creation button');
    } else {
      // Try to navigate directly to work creation page
      console.log('Trying to navigate directly to /works/new');
      await page.goto('http://localhost:3001/works/new');
    }
    
    // Wait for page load
    await page.waitForTimeout(1000);
    
    // Check if we're on a work creation page
    const hasWorkForm = await page.locator('form').isVisible().catch(() => false);
    const hasTitle = await page.locator('input[name="title"]').or(page.locator('input[placeholder*="title" i]')).isVisible().catch(() => false);
    
    console.log(`Work creation form found: ${hasWorkForm}`);
    console.log(`Title input found: ${hasTitle}`);
    
    if (!hasTitle) {
      console.log('No work creation form found. Let me check if there\'s a login requirement...');
      
      // Check if we need to login first
      const needsLogin = await page.locator('text=Login').or(page.locator('text=Sign In')).isVisible().catch(() => false);
      console.log(`Login required: ${needsLogin}`);
      
      if (needsLogin) {
        console.log('Attempting to login...');
        // Try to login with test credentials
        await page.locator('text=Login').or(page.locator('text=Sign In')).first().click();
        
        // Fill login form if present
        const emailInput = page.locator('input[type="email"]').or(page.locator('input[name="email"]')).first();
        const passwordInput = page.locator('input[type="password"]').or(page.locator('input[name="password"]')).first();
        
        const hasEmailInput = await emailInput.isVisible().catch(() => false);
        const hasPasswordInput = await passwordInput.isVisible().catch(() => false);
        
        console.log(`Email input found: ${hasEmailInput}`);
        console.log(`Password input found: ${hasPasswordInput}`);
        
        if (hasEmailInput && hasPasswordInput) {
          await emailInput.fill('test@example.com');
          await passwordInput.fill('testpassword');
          
          const submitButton = page.locator('button[type="submit"]').or(page.locator('text=Login')).first();
          await submitButton.click();
          
          console.log('Submitted login form');
          await page.waitForTimeout(2000);
          
          // Now try to go to work creation again
          await page.goto('http://localhost:3001/works/new');
          await page.waitForTimeout(1000);
        }
      }
    }
    
    // Check for work creation form again
    const titleInput = page.locator('input[name="title"]').or(page.locator('input[placeholder*="title" i]')).first();
    const hasTitleInput = await titleInput.isVisible().catch(() => false);
    
    console.log(`Title input found after potential login: ${hasTitleInput}`);
    
    if (hasTitleInput) {
      console.log('Found work creation form! Testing work creation...');
      
      // Fill out basic work information
      await titleInput.fill('Test Work from Playwright');
      console.log('Filled title');
      
      const summaryInput = page.locator('textarea[name="summary"]').or(page.locator('textarea[placeholder*="summary" i]')).first();
      const hasSummary = await summaryInput.isVisible().catch(() => false);
      
      if (hasSummary) {
        await summaryInput.fill('This is a test work created by Playwright to investigate the 500 error.');
        console.log('Filled summary');
      }
      
      // Look for submit button
      const submitButton = page.locator('button[type="submit"]').or(page.locator('text=Create Work')).or(page.locator('text=Save')).first();
      const hasSubmitButton = await submitButton.isVisible().catch(() => false);
      
      console.log(`Submit button found: ${hasSubmitButton}`);
      
      if (hasSubmitButton) {
        console.log('Attempting to submit work...');
        
        // Listen for network requests
        page.on('request', request => {
          if (request.url().includes('/api/v1/works')) {
            console.log(`REQUEST: ${request.method()} ${request.url()}`);
            console.log(`Headers:`, request.headers());
            if (request.method() === 'POST') {
              console.log('POST data:', request.postData());
            }
          }
        });
        
        page.on('response', response => {
          if (response.url().includes('/api/v1/works')) {
            console.log(`RESPONSE: ${response.status()} ${response.url()}`);
            response.text().then(body => {
              console.log('Response body:', body);
            }).catch(() => {
              console.log('Could not read response body');
            });
          }
        });
        
        await submitButton.click();
        console.log('Clicked submit button');
        
        // Wait for response
        await page.waitForTimeout(3000);
        
        // Check for error messages
        const errorMessage = page.locator('[role="alert"]').or(page.locator('.error')).or(page.locator('text=error')).first();
        const hasError = await errorMessage.isVisible().catch(() => false);
        
        if (hasError) {
          const errorText = await errorMessage.textContent();
          console.log(`Error message found: "${errorText}"`);
        }
        
        // Check if we stayed on the same page (indicating an error) or navigated away (success)
        const currentUrl = page.url();
        console.log(`Current URL after submission: ${currentUrl}`);
      }
    } else {
      console.log('Could not find work creation form. Taking screenshot for inspection...');
    }
    
    // Take a screenshot for manual inspection
    await page.screenshot({ path: 'frontend/work-creation-flow-test.png', fullPage: true });
    console.log('Screenshot saved: work-creation-flow-test.png');
    
    console.log('=== WORK CREATION TEST COMPLETE ===');
  });

  test('Search form and work click investigation', async ({ page }) => {
    console.log('=== INVESTIGATING SEARCH AND WORK CLICK ===');
    
    // Navigate to search page
    await page.goto('http://localhost:3001/search');
    
    // Try to perform a search
    const titleInput = page.locator('input[name="title"]').or(page.locator('input[placeholder*="title"]')).first();
    const hasTitleInput = await titleInput.isVisible().catch(() => false);
    
    if (hasTitleInput) {
      console.log('Found search form');
      await titleInput.fill('test');
      
      const searchButton = page.locator('button[type="submit"]').or(page.locator('text=Search')).first();
      const hasSearchButton = await searchButton.isVisible().catch(() => false);
      
      if (hasSearchButton) {
        console.log('Clicking search button...');
        await searchButton.click();
        await page.waitForTimeout(2000);
        
        // Look for search results
        const workResults = page.locator('[data-testid="work-item"]').or(page.locator('article').or(page.locator('.work-item')));
        const resultCount = await workResults.count();
        console.log(`Found ${resultCount} work results`);
        
        if (resultCount > 0) {
          console.log('Testing work click navigation...');
          
          // Listen for navigation
          page.on('request', request => {
            if (request.url().includes('/works/')) {
              console.log(`Work navigation request: ${request.url()}`);
            }
          });
          
          // Click on first work
          await workResults.first().click();
          await page.waitForTimeout(1000);
          
          const newUrl = page.url();
          console.log(`URL after work click: ${newUrl}`);
          
          if (newUrl.includes('/works/')) {
            console.log('✅ Work click navigation successful!');
          } else {
            console.log('❌ Work click navigation failed - URL did not change');
          }
        }
      }
    }
    
    await page.screenshot({ path: 'frontend/search-and-click-test.png', fullPage: true });
    console.log('Screenshot saved: search-and-click-test.png');
    
    console.log('=== SEARCH AND CLICK TEST COMPLETE ===');
  });
});