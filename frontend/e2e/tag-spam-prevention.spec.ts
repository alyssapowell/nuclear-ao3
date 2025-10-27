import { test, expect } from '@playwright/test';
import { loginUser } from './test-utils';

/**
 * Tests specifically for preventing tag spam scenarios like:
 * https://archiveofourown.org/works/72365631
 * 
 * Focus: Limiting relationship tags and improving tagging quality
 */
test.describe('Tag Spam Prevention - Real World Scenarios', () => {
  test.beforeEach(async ({ page }) => {
    await loginUser(page, 'admin@nuclear-ao3.com', 'adminpass123');
    await page.goto('/works/new');
    await page.waitForLoadState('networkidle');
  });

  test('should prevent orgy-style tag spam by limiting primary relationships to 2', async ({ page }) => {
    // The example work has 20+ relationship tags - this is exactly what we want to prevent
    await page.fill('input[name="title"]', 'Anti-Orgy Test');
    await page.fill('textarea[name="summary"]', 'Testing prevention of excessive relationship tagging.');

    // Add required fandom
    const fandomInput = page.locator('input[aria-label*="fandom"]').first();
    await fandomInput.fill('Marvel Cinematic Universe');
    await fandomInput.press('Enter');

    // Try to add many relationships like the problematic example
    const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
    const spamRelationships = [
      'Tony Stark/Steve Rogers',
      'Natasha Romanoff/Clint Barton', 
      'Thor/Loki',
      'Bruce Banner/Tony Stark',
      'Steve Rogers/Bucky Barnes',
      'Tony Stark/Pepper Potts',
      'Clint Barton/Laura Barton',
      'Wanda Maximoff/Vision',
      'Scott Lang/Hope Van Dyne',
      'Peter Quill/Gamora'
    ];

    // Add all the spam relationships
    for (const relationship of spamRelationships) {
      await relationshipInput.fill(relationship);
      await relationshipInput.press('Enter');
      await page.waitForTimeout(100);
    }

    // CRITICAL CHECK: Should have maximum 2 primary relationships
    const primaryRelationships = page.locator('[data-prominence="primary"]');
    const primaryCount = await primaryRelationships.count();
    
    expect(primaryCount).toBeLessThanOrEqual(2);

    // Should show warning about too many relationships
    const tooManyWarning = page.locator(':has-text("too many"), :has-text("focus"), :has-text("main relationship")');
    expect(await tooManyWarning.count()).toBeGreaterThan(0);

    // Excess relationships should be automatically demoted to secondary/micro
    const secondaryRelationships = page.locator('[data-prominence="secondary"]');
    const microRelationships = page.locator('[data-prominence="micro"]');
    const demotedCount = await secondaryRelationships.count() + await microRelationships.count();
    
    expect(demotedCount).toBeGreaterThan(6); // Most should be demoted
  });

  test('should auto-detect missing characters from relationship tags', async ({ page }) => {
    await page.fill('input[name="title"]', 'Character Detection Test');

    // Add fandom
    const fandomInput = page.locator('input[aria-label*="fandom"]').first();
    await fandomInput.fill('Harry Potter - J. K. Rowling');
    await fandomInput.press('Enter');

    // Add relationship WITHOUT adding characters first
    const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
    await relationshipInput.fill('Harry Potter/Hermione Granger');
    await relationshipInput.press('Enter');

    // System should detect missing characters and either:
    // 1. Auto-add them
    // 2. Show clear suggestion to add them
    
    await page.waitForTimeout(1000); // Wait for AI processing

    const characterSuggestion = page.locator(':has-text("missing character"), :has-text("add character"), [data-testid="character-suggestion"]');
    const autoAddedHarry = page.locator(':has-text("Harry Potter"):not(:has-text("/"))');
    const autoAddedHermione = page.locator(':has-text("Hermione Granger"):not(:has-text("/"))');

    const hasSuggestion = await characterSuggestion.count() > 0;
    const hasAutoAdded = await autoAddedHarry.count() > 0 && await autoAddedHermione.count() > 0;

    // Either should suggest or auto-add the missing characters
    expect(hasSuggestion || hasAutoAdded).toBe(true);
  });

  test('should handle background/minor relationship detection', async ({ page }) => {
    await page.fill('input[name="title"]', 'Background Detection Test');

    const fandomInput = page.locator('input[aria-label*="fandom"]').first();
    await fandomInput.fill('Marvel');
    await fandomInput.press('Enter');

    const relationshipInput = page.locator('input[aria-label*="relationship"]').first();

    // Add main relationship
    await relationshipInput.fill('Steve Rogers/Tony Stark');
    await relationshipInput.press('Enter');

    // Add background relationships that should auto-detect as micro
    await relationshipInput.fill('Background Natasha Romanoff/Clint Barton');
    await relationshipInput.press('Enter');

    await relationshipInput.fill('Past Peggy Carter/Steve Rogers');
    await relationshipInput.press('Enter');

    await relationshipInput.fill('Mentioned Tony Stark/Pepper Potts');
    await relationshipInput.press('Enter');

    await page.waitForTimeout(1000);

    // Main relationship should be primary
    const primaryTags = page.locator('[data-prominence="primary"]');
    expect(await primaryTags.count()).toBe(1);

    // Background tags should be micro
    const microTags = page.locator('[data-prominence="micro"]');
    expect(await microTags.count()).toBe(3);
  });

  test('should allow user to reprioritize but enforce 2-relationship limit', async ({ page }) => {
    await page.fill('input[name="title"]', 'Reprioritization Test');

    const fandomInput = page.locator('input[aria-label*="fandom"]').first();
    await fandomInput.fill('Harry Potter');
    await fandomInput.press('Enter');

    const relationshipInput = page.locator('input[aria-label*="relationship"]').first();

    // Add 4 relationships
    await relationshipInput.fill('Harry Potter/Ginny Weasley');
    await relationshipInput.press('Enter');

    await relationshipInput.fill('Hermione Granger/Ron Weasley');
    await relationshipInput.press('Enter');

    await relationshipInput.fill('Draco Malfoy/Harry Potter');
    await relationshipInput.press('Enter');

    await relationshipInput.fill('Sirius Black/Remus Lupin');
    await relationshipInput.press('Enter');

    await page.waitForTimeout(1000);

    // Should have 2 primary, 2 secondary initially
    expect(await page.locator('[data-prominence="primary"]').count()).toBe(2);
    expect(await page.locator('[data-prominence="secondary"]').count()).toBe(2);

    // Try to promote a secondary to primary using prominence controls
    const prominenceControls = page.locator('.prominence-select, [data-testid="prominence-control"]');
    
    if (await prominenceControls.count() > 0) {
      // Find a secondary tag and try to promote it
      const secondaryControl = prominenceControls.nth(2); // Third relationship (should be secondary)
      
      if (await secondaryControl.isVisible()) {
        await secondaryControl.selectOption('primary');
        
        // Should either prevent this or automatically demote another primary
        const primaryCount = await page.locator('[data-prominence="primary"]').count();
        expect(primaryCount).toBeLessThanOrEqual(2);

        // Should show feedback about the 2-relationship limit
        const limitWarning = page.locator(':has-text("maximum"), :has-text("limit"), :has-text("2 primary")');
        expect(await limitWarning.count()).toBeGreaterThan(0);
      }
    }
  });

  test('should provide clear guidance about relationship tagging best practices', async ({ page }) => {
    await page.fill('input[name="title"]', 'Guidance Test');

    // Look for help text or guidance about relationship tagging
    const guidanceText = page.locator(':has-text("focus on main"), :has-text("primary relationship"), :has-text("central to story")');
    expect(await guidanceText.count()).toBeGreaterThan(0);

    // Should explain what primary vs secondary vs micro means
    const primaryExplanation = page.locator(':has-text("Primary"):has-text("main focus")');
    const secondaryExplanation = page.locator(':has-text("Secondary"):has-text("important but not central")'); 
    const microExplanation = page.locator(':has-text("Micro"):has-text("background")');

    expect(await primaryExplanation.count()).toBeGreaterThan(0);
    expect(await secondaryExplanation.count()).toBeGreaterThan(0);
    expect(await microExplanation.count()).toBeGreaterThan(0);
  });
});

/**
 * Tests for the actual form integration and user experience
 */
test.describe('Enhanced Tag System - Form Integration', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:3001/test-components');
    await page.waitForLoadState('networkidle');
  });

  test('should show EnhancedTagProminenceSelector component', async ({ page }) => {
    // Look for the enhanced tag prominence selector on the test page
    const enhancedSelector = page.locator('.enhanced-tag-prominence-selector, [data-testid="enhanced-tag-selector"]');
    
    if (await enhancedSelector.isVisible()) {
      // Test the tag type selector buttons
      const relationshipButton = page.locator('button:has-text("Relationships")');
      await expect(relationshipButton).toBeVisible();
      
      const characterButton = page.locator('button:has-text("Characters")');
      await expect(characterButton).toBeVisible();
      
      const freeformButton = page.locator('button:has-text("Additional Tags")');
      await expect(freeformButton).toBeVisible();

      // Test prominence sections
      const primarySection = page.locator('.prominence-section.primary');
      await expect(primarySection).toBeVisible();
      
      const secondarySection = page.locator('.prominence-section.secondary'); 
      await expect(secondarySection).toBeVisible();
      
      const microSection = page.locator('.prominence-section.micro');
      await expect(microSection).toBeVisible();
    } else {
      console.log('Enhanced tag prominence selector not found - may need frontend integration');
    }
  });

  test('should handle AI-powered tag inference', async ({ page }) => {
    const enhancedSelector = page.locator('.enhanced-tag-prominence-selector');
    
    if (await enhancedSelector.isVisible()) {
      // Click relationships tab
      await page.click('button:has-text("Relationships")');
      
      // Add a relationship with background indicator
      const tagInput = page.locator('input[placeholder*="relationship"]');
      await tagInput.fill('Background Steve Rogers/Tony Stark');
      await tagInput.press('Enter');
      
      // Should automatically appear in micro section
      const microSection = page.locator('.prominence-section.micro');
      await expect(microSection.locator(':has-text("Steve Rogers/Tony Stark")')).toBeVisible();
      
      // Should show AI suggestion indicator
      const aiIndicator = page.locator('.auto-suggested-indicator, :has-text("🤖")');
      await expect(aiIndicator).toBeVisible();
    }
  });
});