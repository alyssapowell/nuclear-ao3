import { test, expect } from '@playwright/test';

test.describe('Tag Suggestions Debugging', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the page with tag input
    await page.goto('http://localhost:3001');
    
    // Wait for the page to load
    await page.waitForLoadState('networkidle');
  });

  test('Debug tag suggestion functionality step by step', async ({ page }) => {
    console.log('=== STARTING TAG SUGGESTION DEBUG TEST ===');

    // Step 1: Check for JavaScript errors
    const jsErrors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        jsErrors.push(msg.text());
        console.log('JS ERROR:', msg.text());
      }
    });

    page.on('pageerror', (error) => {
      jsErrors.push(error.message);
      console.log('PAGE ERROR:', error.message);
    });

    // Step 2: Find the tag input field (try multiple possible selectors)
    const possibleSelectors = [
      'input[placeholder*="tag"]',
      'input[placeholder*="Tag"]', 
      'input[name*="tag"]',
      'input[id*="tag"]',
      '.tag-input',
      '#tag-input',
      '[data-testid*="tag"]',
      'input[type="text"]'
    ];

    let tagInput = null;
    let usedSelector = '';

    for (const selector of possibleSelectors) {
      try {
        const element = page.locator(selector).first();
        if (await element.isVisible({ timeout: 1000 })) {
          tagInput = element;
          usedSelector = selector;
          console.log(`Found tag input with selector: ${selector}`);
          break;
        }
      } catch (e) {
        // Continue to next selector
      }
    }

    if (!tagInput) {
      console.log('Available input fields:');
      const allInputs = await page.locator('input').all();
      for (let i = 0; i < allInputs.length; i++) {
        const input = allInputs[i];
        const placeholder = await input.getAttribute('placeholder') || '';
        const name = await input.getAttribute('name') || '';
        const id = await input.getAttribute('id') || '';
        const type = await input.getAttribute('type') || '';
        console.log(`Input ${i}: placeholder="${placeholder}", name="${name}", id="${id}", type="${type}"`);
      }
      
      // Try the first visible text input as fallback
      tagInput = page.locator('input[type="text"]').first();
      usedSelector = 'input[type="text"] (fallback)';
    }

    expect(tagInput).toBeTruthy();
    console.log(`Using tag input selector: ${usedSelector}`);

    // Step 3: Set up network monitoring
    const requests: any[] = [];
    page.on('request', (request) => {
      if (request.url().includes('/tags/search')) {
        requests.push({
          url: request.url(),
          method: request.method(),
          headers: request.headers()
        });
        console.log('TAG SEARCH REQUEST:', request.url());
      }
    });

    const responses: any[] = [];
    page.on('response', async (response) => {
      if (response.url().includes('/tags/search')) {
        const responseData = {
          url: response.url(),
          status: response.status(),
          statusText: response.statusText(),
        };
        
        try {
          const json = await response.json();
          responseData.body = json;
          console.log('TAG SEARCH RESPONSE:', JSON.stringify(responseData, null, 2));
        } catch (e) {
          const text = await response.text();
          responseData.body = text;
          console.log('TAG SEARCH RESPONSE (text):', responseData);
        }
        
        responses.push(responseData);
      }
    });

    // Step 4: Type in the tag input and trigger search
    console.log('Typing "flu" in tag input...');
    await tagInput.fill('');
    await tagInput.type('flu', { delay: 100 });

    // Wait a bit for potential debouncing/async requests
    await page.waitForTimeout(1000);

    // Step 5: Check if network request was made
    console.log(`Network requests made: ${requests.length}`);
    if (requests.length === 0) {
      console.log('No tag search requests were made. Trying additional triggers...');
      
      // Try different events that might trigger search
      await tagInput.press('ArrowDown');
      await page.waitForTimeout(500);
      
      await tagInput.press('Enter');
      await page.waitForTimeout(500);
      
      await tagInput.blur();
      await tagInput.focus();
      await page.waitForTimeout(500);
    }

    // Step 6: Look for suggestion containers
    const suggestionSelectors = [
      '.suggestions',
      '.tag-suggestions', 
      '.autocomplete',
      '.dropdown',
      '.results',
      '[role="listbox"]',
      '[role="menu"]',
      '.suggestion-list',
      '.typeahead'
    ];

    let suggestionContainer = null;
    let suggestionSelector = '';

    for (const selector of suggestionSelectors) {
      try {
        const element = page.locator(selector);
        if (await element.count() > 0) {
          suggestionContainer = element;
          suggestionSelector = selector;
          console.log(`Found suggestion container: ${selector}`);
          
          const isVisible = await element.isVisible();
          console.log(`Suggestion container visible: ${isVisible}`);
          
          if (isVisible) {
            const text = await element.textContent();
            console.log(`Suggestion container content: "${text}"`);
          }
          break;
        }
      } catch (e) {
        // Continue
      }
    }

    // Step 7: Check DOM for any elements that might contain "Fluff"
    console.log('Checking DOM for "Fluff" text...');
    const fluffElements = page.locator('text=Fluff');
    const fluffCount = await fluffElements.count();
    console.log(`Elements containing "Fluff": ${fluffCount}`);
    
    if (fluffCount > 0) {
      for (let i = 0; i < fluffCount; i++) {
        const element = fluffElements.nth(i);
        const isVisible = await element.isVisible();
        const tag = await element.evaluate(el => el.tagName);
        const classes = await element.getAttribute('class') || '';
        console.log(`Fluff element ${i}: tag=${tag}, visible=${isVisible}, classes="${classes}"`);
      }
    }

    // Step 8: Check computed styles of potential suggestion elements
    if (suggestionContainer) {
      const styles = await suggestionContainer.evaluate(el => {
        const computed = window.getComputedStyle(el);
        return {
          display: computed.display,
          visibility: computed.visibility,
          opacity: computed.opacity,
          position: computed.position,
          zIndex: computed.zIndex,
          top: computed.top,
          left: computed.left
        };
      });
      console.log('Suggestion container styles:', styles);
    }

    // Step 9: Take a screenshot for visual debugging
    await page.screenshot({ 
      path: '/tmp/tag-suggestions-debug.png', 
      fullPage: true 
    });
    console.log('Screenshot saved to /tmp/tag-suggestions-debug.png');

    // Step 10: Final assertions and summary
    console.log('=== DEBUG SUMMARY ===');
    console.log(`JavaScript errors: ${jsErrors.length}`);
    console.log(`Network requests made: ${requests.length}`);
    console.log(`Network responses received: ${responses.length}`);
    console.log(`Suggestion container found: ${!!suggestionContainer}`);
    console.log(`Elements with "Fluff" text: ${fluffCount}`);

    if (jsErrors.length > 0) {
      console.log('JavaScript errors found:', jsErrors);
    }

    if (requests.length > 0) {
      console.log('Last request URL:', requests[requests.length - 1].url);
    }

    if (responses.length > 0) {
      const lastResponse = responses[responses.length - 1];
      console.log('Last response status:', lastResponse.status);
      if (lastResponse.body && lastResponse.body.tags) {
        console.log('Response contains tags:', lastResponse.body.tags.length);
      }
    }

    // Basic assertion - at least the input should work
    expect(await tagInput.inputValue()).toBe('flu');
  });

  test('Test API endpoint directly from browser', async ({ page }) => {
    // Test the API endpoint directly to ensure it works from browser context
    const response = await page.request.get('http://localhost:8080/api/v1/tags/search?q=flu&limit=10', {
      headers: {
        'Origin': 'http://localhost:3001',
        'Accept': 'application/json'
      }
    });

    expect(response.status()).toBe(200);
    
    const data = await response.json();
    console.log('Direct API test response:', JSON.stringify(data, null, 2));
    
    expect(data.tags).toBeTruthy();
    expect(data.tags.length).toBeGreaterThan(0);
    expect(data.tags[0].name).toBe('Fluff');
  });
});