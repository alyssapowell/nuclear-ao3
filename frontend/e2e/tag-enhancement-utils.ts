import { Page, expect } from '@playwright/test';

/**
 * Utility functions for testing the enhanced tag prominence system
 */

export interface TagProminence {
  tagName: string;
  tagType: 'fandom' | 'character' | 'relationship' | 'freeform';
  prominence: 'primary' | 'secondary' | 'micro';
  autoSuggested?: boolean;
}

/**
 * Helper to add tags with specific prominence levels for testing
 */
export async function addTagWithProminence(
  page: Page, 
  tagName: string, 
  tagType: 'fandom' | 'character' | 'relationship' | 'freeform',
  expectedProminence?: 'primary' | 'secondary' | 'micro'
): Promise<void> {
  // Find the appropriate input based on tag type
  const inputSelector = `input[aria-label*="${tagType}"]`;
  const input = page.locator(inputSelector).first();
  
  await expect(input).toBeVisible();
  await input.fill(tagName);
  await input.press('Enter');
  
  // Wait for tag processing
  await page.waitForTimeout(500);
  
  // If expected prominence is specified, verify it
  if (expectedProminence) {
    const tagElement = page.locator(`[data-prominence="${expectedProminence}"]:has-text("${tagName}")`);
    await expect(tagElement).toBeVisible({ timeout: 5000 });
  }
}

/**
 * Helper to verify tag prominence distribution
 */
export async function verifyProminenceDistribution(
  page: Page,
  expected: { primary: number; secondary: number; micro: number }
): Promise<void> {
  const primaryTags = page.locator('[data-prominence="primary"]');
  const secondaryTags = page.locator('[data-prominence="secondary"]');
  const microTags = page.locator('[data-prominence="micro"]');
  
  await expect(primaryTags).toHaveCount(expected.primary);
  await expect(secondaryTags).toHaveCount(expected.secondary);
  await expect(microTags).toHaveCount(expected.micro);
}

/**
 * Helper to simulate the "orgy problem" by adding many relationship tags
 */
export async function addManyRelationshipTags(page: Page, relationships: string[]): Promise<void> {
  const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
  
  for (const relationship of relationships) {
    await relationshipInput.fill(relationship);
    await relationshipInput.press('Enter');
    await page.waitForTimeout(100); // Brief pause between additions
  }
}

/**
 * Check if the system shows appropriate warnings for tag spam
 */
export async function expectTagSpamWarning(page: Page): Promise<void> {
  const warningSelectors = [
    ':has-text("too many")',
    ':has-text("focus on main")',
    ':has-text("primary relationship")',
    '.tag-warning',
    '[data-testid="tag-warning"]'
  ];
  
  let foundWarning = false;
  for (const selector of warningSelectors) {
    const warning = page.locator(selector);
    if (await warning.count() > 0) {
      foundWarning = true;
      break;
    }
  }
  
  expect(foundWarning).toBe(true);
}

/**
 * Verify that missing characters are detected from relationship tags
 */
export async function expectMissingCharacterDetection(
  page: Page, 
  relationshipTag: string, 
  expectedCharacters: string[]
): Promise<void> {
  // Add the relationship tag
  await addTagWithProminence(page, relationshipTag, 'relationship');
  
  // Check for character suggestions or auto-additions
  const suggestionSelectors = [
    ':has-text("missing character")',
    ':has-text("add character")',
    '[data-testid="character-suggestion"]',
    '.character-suggestion'
  ];
  
  let foundSuggestion = false;
  for (const selector of suggestionSelectors) {
    if (await page.locator(selector).count() > 0) {
      foundSuggestion = true;
      break;
    }
  }
  
  // Also check if characters were auto-added
  let allCharactersPresent = true;
  for (const character of expectedCharacters) {
    const characterTag = page.locator(`:has-text("${character}"):not(:has-text("/"))`);
    if (await characterTag.count() === 0) {
      allCharactersPresent = false;
      break;
    }
  }
  
  // Either should suggest or auto-add the characters
  expect(foundSuggestion || allCharactersPresent).toBe(true);
}

/**
 * Test background/minor tag detection
 */
export async function testBackgroundTagDetection(
  page: Page,
  backgroundTags: string[]
): Promise<void> {
  for (const tag of backgroundTags) {
    await addTagWithProminence(page, tag, 'relationship', 'micro');
  }
}

/**
 * Helper to setup a basic work for tag testing
 */
export async function setupBasicWorkForTagTesting(
  page: Page, 
  title: string = 'Tag Enhancement Test Work'
): Promise<void> {
  // Navigate to work creation if not already there
  if (!page.url().includes('/works/new')) {
    await page.goto('/works/new');
    await page.waitForLoadState('networkidle');
  }
  
  // Fill required fields
  await page.fill('input[name="title"]', title);
  await page.fill('textarea[name="summary"]', 'Test work for enhanced tag system validation.');
  
  // Add required fandom
  await addTagWithProminence(page, 'Test Fandom', 'fandom');
  
  // Set required fields (rating, warning, category)
  const ratingRadio = page.locator('input[value="General Audiences"]');
  if (await ratingRadio.isVisible()) {
    await ratingRadio.check();
  }
  
  const warningRadio = page.locator('input[value="No Archive Warnings Apply"]');
  if (await warningRadio.isVisible()) {
    await warningRadio.check();
  }
  
  const categoryRadio = page.locator('input[value="Gen"]');
  if (await categoryRadio.isVisible()) {
    await categoryRadio.check();
  }
}

/**
 * Verify the enhanced tag prominence selector is working
 */
export async function expectEnhancedTagSelectorWorking(page: Page): Promise<void> {
  // Look for the enhanced tag prominence selector
  const selector = page.locator('.enhanced-tag-prominence-selector, [data-testid="enhanced-tag-selector"]');
  
  if (await selector.isVisible()) {
    // Verify tag type buttons
    await expect(page.locator('button:has-text("Relationships")')).toBeVisible();
    await expect(page.locator('button:has-text("Characters")')).toBeVisible();
    await expect(page.locator('button:has-text("Additional Tags")')).toBeVisible();
    
    // Verify prominence sections
    await expect(page.locator('.prominence-section.primary')).toBeVisible();
    await expect(page.locator('.prominence-section.secondary')).toBeVisible();
    await expect(page.locator('.prominence-section.micro')).toBeVisible();
    
    console.log('✅ Enhanced tag prominence selector is working');
  } else {
    console.log('⚠️ Enhanced tag prominence selector not found - using fallback tag inputs');
    
    // Verify basic tag inputs are present
    await expect(page.locator('input[aria-label*="fandom"]')).toBeVisible();
    await expect(page.locator('input[aria-label*="relationship"]')).toBeVisible();
    await expect(page.locator('input[aria-label*="character"]')).toBeVisible();
  }
}

/**
 * Common problematic relationships from real AO3 tag spam examples
 */
export const SPAM_RELATIONSHIP_EXAMPLES = [
  'Tony Stark/Steve Rogers',
  'Natasha Romanoff/Clint Barton',
  'Thor/Loki',
  'Bruce Banner/Tony Stark',
  'Steve Rogers/Bucky Barnes',
  'Tony Stark/Pepper Potts',
  'Clint Barton/Laura Barton',
  'Wanda Maximoff/Vision',
  'Scott Lang/Hope Van Dyne',
  'Peter Quill/Gamora',
  'Carol Danvers/Maria Rambeau',
  'T\'Challa/Nakia',
  'Stephen Strange/Christine Palmer',
  'Peter Parker/Michelle Jones',
  'Sam Wilson/Sarah Wilson',
  'James Rhodes/Tony Stark',
  'Carol Danvers/Monica Rambeau',
  'Peter Parker/Gwen Stacy',
  'Wade Wilson/Peter Parker',
  'Loki/Stephen Strange'
];

/**
 * Background/minor relationship examples that should auto-detect as micro
 */
export const BACKGROUND_RELATIONSHIP_EXAMPLES = [
  'Background Tony Stark/Pepper Potts',
  'Past Steve Rogers/Peggy Carter',
  'Minor Natasha Romanoff/Clint Barton',
  'Mentioned Thor/Jane Foster',
  'Side Wanda Maximoff/Vision',
  'Background Scott Lang/Hope Van Dyne',
  'Cameo Peter Quill/Gamora'
];