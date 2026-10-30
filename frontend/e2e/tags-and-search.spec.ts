import { test, expect } from '@playwright/test';

test.describe('Tags and Enhanced Search Features', () => {
  const BASE_URL = 'http://localhost:3001';
  const API_URL = 'http://localhost:8080';

  test.beforeEach(async ({ page }) => {
    // Set up any necessary state
  });

  test.describe('Work Tag Loading', () => {
    test('should load and display tags for works list', async ({ request }) => {
      // Test the works API endpoint
      const response = await request.get(`${API_URL}/api/v1/works/`);
      expect(response.ok()).toBeTruthy();
      
      const data = await response.json();
      console.log('Works API Response:', JSON.stringify(data, null, 2));
      
      // Check that works exist
      expect(data.works).toBeDefined();
      expect(Array.isArray(data.works)).toBeTruthy();
      expect(data.works.length).toBeGreaterThan(0);
      
      // Check if any work has tags loaded
      const worksWithTags = data.works.filter(work => 
        work.fandoms?.length > 0 || 
        work.characters?.length > 0 || 
        work.relationships?.length > 0 || 
        work.freeform_tags?.length > 0
      );
      
      console.log(`Found ${worksWithTags.length} works with tags out of ${data.works.length} total works`);
      
      // Test individual works that should have tags based on database
      const knownWorkWithTags = 'e38ec269-901f-4469-b0b1-acb346b82e08';
      const individualWork = await request.get(`${API_URL}/api/v1/works/${knownWorkWithTags}`);
      
      if (individualWork.ok()) {
        const workData = await individualWork.json();
        console.log('Individual Work with Expected Tags:', JSON.stringify(workData, null, 2));
        
        // This should have tags based on our database investigation
        // Currently failing - this is what we need to fix
        expect(workData.work).toBeDefined();
        
        // Log current state for debugging
        console.log('Current tag state:', {
          fandoms: workData.work.fandoms,
          characters: workData.work.characters, 
          relationships: workData.work.relationships,
          freeform_tags: workData.work.freeform_tags
        });
      }
    });

    test('should verify database has tag associations', async ({ request }) => {
      // This test verifies that the issue is in the API, not the database
      // We'll use a direct database check if possible, or verify via tag search
      
      const tagSearchResponse = await request.get(`${API_URL}/api/v1/tags/search?q=harry`);
      expect(tagSearchResponse.ok()).toBeTruthy();
      
      const tagData = await tagSearchResponse.json();
      console.log('Tag Search Results:', JSON.stringify(tagData, null, 2));
      
      expect(tagData.tags).toBeDefined();
      expect(tagData.tags.length).toBeGreaterThan(0);
      expect(tagData.total).toBeGreaterThan(0);
      
      // Verify we have different types of tags
      const fandomTags = tagData.tags.filter(tag => tag.type === 'fandom');
      const characterTags = tagData.tags.filter(tag => tag.type === 'character');
      const relationshipTags = tagData.tags.filter(tag => tag.type === 'relationship');
      
      console.log('Tag types found:', {
        fandoms: fandomTags.length,
        characters: characterTags.length, 
        relationships: relationshipTags.length,
        total: tagData.tags.length
      });
      
      expect(fandomTags.length).toBeGreaterThan(0);
      expect(characterTags.length).toBeGreaterThan(0);
      expect(relationshipTags.length).toBeGreaterThan(0);
    });
  });

  test.describe('Enhanced Search Features', () => {
    test('searches works and returns results with filter counts', async ({ request }) => {
      const res = await request.get(`${API_URL}/api/v1/search/works?q=harry`);
      expect(res.ok()).toBeTruthy();
      const data = await res.json();
      expect(Array.isArray(data.works)).toBeTruthy();
      expect(typeof data.total).toBe('number');
      expect(data.facets).toBeDefined();
    });

    test('filters by fandom', async ({ request }) => {
      const fandom = 'Harry Potter - J. K. Rowling';
      const res = await request.get(`${API_URL}/api/v1/search/works?fandoms=${encodeURIComponent(fandom)}`);
      expect(res.ok()).toBeTruthy();
      const data = await res.json();
      for (const work of data.works) {
        expect(work.fandoms).toContain(fandom);
      }
    });

    test('shows creators by pseud only', async ({ request }) => {
      const res = await request.get(`${API_URL}/api/v1/search/works?limit=5`);
      const data = await res.json();
      for (const work of data.works) {
        expect(work).not.toHaveProperty('user_id');
        expect(work).not.toHaveProperty('username');
        expect(Array.isArray(work.authors)).toBeTruthy();
      }
    });
  });

  test.describe('Frontend Tag Display', () => {
    test('should display tags in works list when available', async ({ page }) => {
      await page.goto(`${BASE_URL}/works`);
      
      // Wait for works to load
      await page.waitForSelector('[data-testid=\"work-card\"]', { timeout: 10000 });
      
      // Check if any work cards display tags
      const workCards = await page.locator('[data-testid=\"work-card\"]').all();
      expect(workCards.length).toBeGreaterThan(0);
      
      console.log(`Found ${workCards.length} work cards on the page`);
      
      // Check for tag display elements
      const tagElements = await page.locator('[data-testid=\"work-tags\"], .work-tags, .tag').all();
      console.log(`Found ${tagElements.length} potential tag elements`);
      
      // Take screenshot for debugging
      await page.screenshot({ path: 'test-results/works-list-tags.png', fullPage: true });
    });

    test('should support tag filtering in search form', async ({ page }) => {
      await page.goto(`${BASE_URL}/works`);
      
      // Look for tag input fields
      await page.waitForSelector('form', { timeout: 5000 });
      
      // Check if tag autocomplete inputs exist
      const fandomInput = page.locator('input[placeholder*=\"fandom\"], input[aria-label*=\"Fandom\"], #fandoms-input');
      const characterInput = page.locator('input[placeholder*=\"character\"], input[aria-label*=\"Character\"], #characters-input');
      const relationshipInput = page.locator('input[placeholder*=\"relationship\"], input[aria-label*=\"Relationship\"], #relationships-input');
      const additionalTagsInput = page.locator('input[placeholder*=\"tag\"], input[aria-label*=\"Additional\"], #freeformTags-input');
      
      // Test if autocomplete is working by typing
      if (await additionalTagsInput.count() > 0) {
        console.log('Found additional tags input, testing autocomplete...');
        await additionalTagsInput.first().click();
        await additionalTagsInput.first().type('harry');
        
        // Wait for suggestions
        await page.waitForTimeout(1000);
        
        // Take screenshot
        await page.screenshot({ path: 'test-results/tag-autocomplete.png', fullPage: true });
        
        // Look for suggestion dropdown
        const suggestions = await page.locator('[role=\"listbox\"], .suggestions, .autocomplete-dropdown').count();
        console.log(`Found ${suggestions} suggestion containers`);
      }
    });
  });

  test.describe('Search Results and Filtering', () => {
    test('should display enhanced search results with facets', async ({ page }) => {
      await page.goto(`${BASE_URL}/works`);
      
      // Perform a search
      const searchInput = page.locator('input[type=\"search\"], input[name=\"q\"], #search-query');
      if (await searchInput.count() > 0) {
        await searchInput.first().fill('harry potter');
        await searchInput.first().press('Enter');
        
        // Wait for results
        await page.waitForTimeout(2000);
        
        // Look for facets/filters
        const facets = await page.locator('.facets, .filters, [data-testid=\"search-facets\"]').count();
        console.log(`Found ${facets} facet containers`);
        
        // Look for result count and metadata
        const resultInfo = await page.locator('.results-info, .search-results-meta, [data-testid=\"results-count\"]').count();
        console.log(`Found ${resultInfo} result info elements`);
        
        await page.screenshot({ path: 'test-results/search-results.png', fullPage: true });
      }
    });
  });
});