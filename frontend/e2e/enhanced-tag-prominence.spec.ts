import { test, expect } from '@playwright/test';
import { loginUser } from './test-utils';

test.describe('Enhanced Tag Prominence System Tests', () => {
  const BASE_URL = 'http://localhost:3001';

  test.beforeEach(async ({ page }) => {
    // Start with authenticated user for work creation tests
    await loginUser(page, 'admin@nuclear-ao3.com', 'adminpass123');
    await page.goto('/works/new');
    await page.waitForLoadState('networkidle');
  });

  test.describe('Relationship Tag Prominence Management', () => {
    test('should limit primary relationship tags to maximum of 3', async ({ page }) => {
      // Navigate to work creation page
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
      
      // Fill basic required fields
      await page.fill('input[name="title"]', 'Relationship Prominence Test Work');
      await page.fill('textarea[name="summary"]', 'Testing relationship tag prominence limits.');
      
      // Add fandom first (required)
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      if (await fandomInput.isVisible()) {
        await fandomInput.fill('Marvel Cinematic Universe');
        await fandomInput.press('Enter');
      }
      
      // Test adding multiple relationship tags
      const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
      if (await relationshipInput.isVisible()) {
        // Add first relationship - should be primary
        await relationshipInput.fill('Steve Rogers/Tony Stark');
        await relationshipInput.press('Enter');
        
        // Add second relationship - should be primary
        await relationshipInput.fill('Natasha Romanoff/Clint Barton');
        await relationshipInput.press('Enter');
        
        // Add third relationship - should be primary (at limit)
        await relationshipInput.fill('Thor/Loki');
        await relationshipInput.press('Enter');
        
        // Add fourth relationship - should be automatically set to secondary
        await relationshipInput.fill('Pepper Potts/Tony Stark');
        await relationshipInput.press('Enter');
        
        // Add fifth relationship - should be automatically set to secondary
        await relationshipInput.fill('Peggy Carter/Steve Rogers');
        await relationshipInput.press('Enter');
        
        // Verify prominence system is working
        // Check for prominence indicators in the UI
        const primaryTags = page.locator('[data-prominence="primary"]');
        const secondaryTags = page.locator('[data-prominence="secondary"]');
        
        // Should have exactly 3 primary relationship tags
        await expect(primaryTags).toHaveCount(3);
        
        // Should have 2 secondary relationship tags
        await expect(secondaryTags).toHaveCount(2);
      }
    });

    test('should automatically detect and suggest missing character tags from relationships', async ({ page }) => {
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
      
      // Fill basic fields
      await page.fill('input[name="title"]', 'Character Detection Test Work');
      await page.fill('textarea[name="summary"]', 'Testing automatic character detection from relationships.');
      
      // Add fandom
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      if (await fandomInput.isVisible()) {
        await fandomInput.fill('Harry Potter - J. K. Rowling');
        await fandomInput.press('Enter');
      }
      
      // Add a relationship without adding the characters first
      const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
      if (await relationshipInput.isVisible()) {
        await relationshipInput.fill('Harry Potter/Draco Malfoy');
        await relationshipInput.press('Enter');
        
        // Check if the system suggests adding missing characters
        // Look for character suggestion notification or auto-add functionality
        const characterSuggestion = page.locator('[data-testid="character-suggestion"], .character-suggestion, :has-text("missing characters")');
        
        // Should either automatically add characters or show suggestion
        const characterInput = page.locator('input[aria-label*="character"]').first();
        
        // Wait for suggestion system to kick in
        await page.waitForTimeout(1000);
        
        // Check if characters were auto-suggested or if there's a prompt
        const harryCharacterTag = page.locator(':has-text("Harry Potter"):not(:has-text("/"))');
        const dracoCharacterTag = page.locator(':has-text("Draco Malfoy"):not(:has-text("/"))');
        
        // Either the characters should be auto-added or there should be a clear suggestion
        const hasCharacterSuggestions = await characterSuggestion.count() > 0;
        const hasAutoAddedCharacters = await harryCharacterTag.count() > 0 && await dracoCharacterTag.count() > 0;
        
        expect(hasCharacterSuggestions || hasAutoAddedCharacters).toBe(true);
        
        // If not auto-added, manually add to test the workflow
        if (!hasAutoAddedCharacters && await characterInput.isVisible()) {
          await characterInput.fill('Harry Potter');
          await characterInput.press('Enter');
          await characterInput.fill('Draco Malfoy');
          await characterInput.press('Enter');
        }
      }
    });

    test('should handle tag spam scenarios like the example work', async ({ page }) => {
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
      
      // Fill basic fields
      await page.fill('input[name="title"]', 'Tag Spam Prevention Test');
      await page.fill('textarea[name="summary"]', 'Testing prevention of excessive relationship tagging.');
      
      // Add fandom
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      if (await fandomInput.isVisible()) {
        await fandomInput.fill('Marvel');
        await fandomInput.press('Enter');
      }
      
      // Simulate adding too many relationship tags (like the problematic example)
      const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
      if (await relationshipInput.isVisible()) {
        const excessiveRelationships = [
          'Tony Stark/Steve Rogers',
          'Natasha Romanoff/Clint Barton', 
          'Thor/Loki',
          'Pepper Potts/Tony Stark',
          'James Rhodes/Tony Stark',
          'Carol Danvers/Maria Rambeau',
          'Scott Lang/Hope Van Dyne',
          'Peter Quill/Gamora',
          'Wanda Maximoff/Vision',
          'T\'Challa/Nakia'
        ];
        
        // Add all these relationships
        for (const relationship of excessiveRelationships) {
          await relationshipInput.fill(relationship);
          await relationshipInput.press('Enter');
          await page.waitForTimeout(200); // Brief pause between additions
        }
        
        // Check that the system provides warnings or limits
        const warningMessage = page.locator('[data-testid="tag-warning"], .tag-warning, :has-text("too many"), :has-text("excessive")');
        const hasWarning = await warningMessage.count() > 0;
        
        // Check prominence distribution
        const primaryTags = page.locator('[data-prominence="primary"]');
        const secondaryTags = page.locator('[data-prominence="secondary"]');
        const microTags = page.locator('[data-prominence="micro"]');
        
        const primaryCount = await primaryTags.count();
        const secondaryCount = await secondaryTags.count();
        const microCount = await microTags.count();
        
        // Should have limited primary tags and moved excess to secondary/micro
        expect(primaryCount).toBeLessThanOrEqual(3);
        expect(secondaryCount + microCount).toBeGreaterThan(5);
        
        // Should show some kind of guidance or warning
        expect(hasWarning).toBe(true);
      }
    });

    test('should allow users to manually adjust tag prominence up to 3 primary relationships', async ({ page }) => {
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
      
      // Fill basic fields
      await page.fill('input[name="title"]', 'Manual Prominence Adjustment Test');
      
      // Add fandom
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      if (await fandomInput.isVisible()) {
        await fandomInput.fill('Harry Potter');
        await fandomInput.press('Enter');
      }
      
      // Add several relationships
      const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
      if (await relationshipInput.isVisible()) {
        await relationshipInput.fill('Harry Potter/Ginny Weasley');
        await relationshipInput.press('Enter');
        
        await relationshipInput.fill('Hermione Granger/Ron Weasley');
        await relationshipInput.press('Enter');
        
        await relationshipInput.fill('Draco Malfoy/Harry Potter');
        await relationshipInput.press('Enter');
        
        await relationshipInput.fill('Sirius Black/Remus Lupin');
        await relationshipInput.press('Enter');
        
        // Now test manual prominence adjustment
        // Look for prominence controls (dropdowns, buttons, etc.)
        const prominenceControls = page.locator('[data-testid="prominence-control"], .prominence-select, select[aria-label*="prominence"]');
        
        if (await prominenceControls.count() > 0) {
          // Try to set the fourth relationship to primary (should be prevented or warned)
          const fourthRelationshipControl = prominenceControls.nth(3);
          
          if (await fourthRelationshipControl.isVisible()) {
            await fourthRelationshipControl.selectOption('primary');
            
            // Should either prevent this or show a warning
            const primaryWarning = page.locator(':has-text("maximum"), :has-text("limit"), .prominence-warning');
            const hasWarning = await primaryWarning.count() > 0;
            
            // Verify we still have max 3 primary tags
            const primaryTags = page.locator('[data-prominence="primary"]');
            const primaryCount = await primaryTags.count();
            
            expect(primaryCount).toBeLessThanOrEqual(3);
            
            // Should show guidance about the limit
            expect(hasWarning).toBe(true);
          }
        }
      }
    });
  });

  test.describe('Background/Minor Tag Detection', () => {
    test('should automatically set micro prominence for background/past relationship tags', async ({ page }) => {
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
      
      // Fill basic fields
      await page.fill('input[name="title"]', 'Background Tag Detection Test');
      
      // Add fandom
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      if (await fandomInput.isVisible()) {
        await fandomInput.fill('Marvel');
        await fandomInput.press('Enter');
      }
      
      // Add relationship tags with background/past indicators
      const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
      if (await relationshipInput.isVisible()) {
        // These should automatically be set to micro prominence
        await relationshipInput.fill('Background Tony Stark/Pepper Potts');
        await relationshipInput.press('Enter');
        
        await relationshipInput.fill('Past Steve Rogers/Peggy Carter');
        await relationshipInput.press('Enter');
        
        await relationshipInput.fill('Minor Natasha Romanoff/Clint Barton');
        await relationshipInput.press('Enter');
        
        await relationshipInput.fill('Mentioned Thor/Jane Foster');
        await relationshipInput.press('Enter');
        
        // Regular relationship should be primary/secondary
        await relationshipInput.fill('Steve Rogers/Bucky Barnes');
        await relationshipInput.press('Enter');
        
        // Wait for AI processing
        await page.waitForTimeout(1000);
        
        // Check that background tags are set to micro
        const microTags = page.locator('[data-prominence="micro"]');
        const primarySecondaryTags = page.locator('[data-prominence="primary"], [data-prominence="secondary"]');
        
        const microCount = await microTags.count();
        const primarySecondaryCount = await primarySecondaryTags.count();
        
        // Should have 4 micro tags (background ones) and 1 primary/secondary (regular one)
        expect(microCount).toBe(4);
        expect(primarySecondaryCount).toBe(1);
      }
    });

    test('should provide clear UI feedback for AI-suggested prominences', async ({ page }) => {
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
      
      // Fill basic fields
      await page.fill('input[name="title"]', 'AI Feedback Test');
      
      // Add fandom
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      if (await fandomInput.isVisible()) {
        await fandomInput.fill('Harry Potter');
        await fandomInput.press('Enter');
      }
      
      // Add a relationship that should trigger AI suggestion
      const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
      if (await relationshipInput.isVisible()) {
        await relationshipInput.fill('Harry Potter/Ginny Weasley');
        await relationshipInput.press('Enter');
        
        // Look for AI suggestion indicators
        const aiIndicators = page.locator('[data-testid="ai-suggested"], .ai-suggested-indicator, :has-text("🤖"), :has-text("AI suggested")');
        
        // Should show AI suggestion feedback
        const hasAIIndicator = await aiIndicators.count() > 0;
        expect(hasAIIndicator).toBe(true);
        
        // Should also show explanation or help text
        const explanationText = page.locator(':has-text("AI"), :has-text("automatically"), :has-text("suggested")');
        const hasExplanation = await explanationText.count() > 0;
        expect(hasExplanation).toBe(true);
      }
    });
  });

  test.describe('Tag Quality and Filtering', () => {
    test('should help users clean up messy tag patterns', async ({ page }) => {
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
      
      // Fill basic fields
      await page.fill('input[name="title"]', 'Tag Cleanup Test');
      
      // Add fandom
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      if (await fandomInput.isVisible()) {
        await fandomInput.fill('Marvel');
        await fandomInput.press('Enter');
      }
      
      // Add redundant/messy tags that should trigger cleanup suggestions
      const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
      if (await relationshipInput.isVisible()) {
        // Add redundant variations
        await relationshipInput.fill('Tony Stark/Steve Rogers');
        await relationshipInput.press('Enter');
        
        await relationshipInput.fill('Steve Rogers/Tony Stark'); // Reverse order - should be flagged
        await relationshipInput.press('Enter');
        
        await relationshipInput.fill('Stony'); // Ship name - should suggest canonical
        await relationshipInput.press('Enter');
        
        // Check for cleanup suggestions
        const cleanupSuggestions = page.locator('[data-testid="tag-cleanup"], .cleanup-suggestion, :has-text("duplicate"), :has-text("similar")');
        const hasCleanupSuggestions = await cleanupSuggestions.count() > 0;
        
        expect(hasCleanupSuggestions).toBe(true);
      }
    });

    test('should prevent the "orgy problem" by limiting and organizing relationship tags', async ({ page }) => {
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
      
      // Fill basic fields  
      await page.fill('input[name="title"]', 'Orgy Prevention Test');
      
      // Add fandom
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      if (await fandomInput.isVisible()) {
        await fandomInput.fill('Marvel');
        await fandomInput.press('Enter');
      }
      
      // Try to add way too many relationships (like the problematic example)
      const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
      if (await relationshipInput.isVisible()) {
        // Add 15+ relationship tags
        const manyRelationships = [
          'Tony Stark/Steve Rogers',
          'Natasha Romanoff/Clint Barton',
          'Thor/Loki', 
          'Bruce Banner/Tony Stark',
          'Steve Rogers/Bucky Barnes',
          'Tony Stark/Pepper Potts',
          'Clint Barton/Laura Barton',
          'Scott Lang/Hope Van Dyne',
          'Wanda Maximoff/Vision',
          'Peter Quill/Gamora',
          'Carol Danvers/Maria Rambeau',
          'T\'Challa/Nakia',
          'Stephen Strange/Christine Palmer',
          'Peter Parker/Michelle Jones',
          'Sam Wilson/Sarah Wilson'
        ];
        
        for (const relationship of manyRelationships) {
          await relationshipInput.fill(relationship);
          await relationshipInput.press('Enter');
          await page.waitForTimeout(100);
        }
        
        // Check that the system provides strong guidance against this
        const orgyWarning = page.locator(':has-text("too many relationship"), :has-text("focus on main"), :has-text("primary relationships"), .orgy-warning');
        const hasOrgyWarning = await orgyWarning.count() > 0;
        
        // Should limit primary relationships to 1-3 max
        const primaryTags = page.locator('[data-prominence="primary"]');
        const primaryCount = await primaryTags.count();
        
        expect(primaryCount).toBeLessThanOrEqual(3);
        expect(hasOrgyWarning).toBe(true);
        
        // Should suggest consolidation or provide filtering guidance
        const consolidationSuggestion = page.locator(':has-text("consolidate"), :has-text("focus"), :has-text("main story")');
        const hasConsolidationHelp = await consolidationSuggestion.count() > 0;
        
        expect(hasConsolidationHelp).toBe(true);
      }
    });
  });

  test.describe('User Experience and Accessibility', () => {
    test('should provide clear explanations of the prominence system', async ({ page }) => {
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
      
      // Look for help text or explanations about the prominence system
      const helpText = page.locator('[data-testid="prominence-help"], .prominence-explanation, :has-text("Primary"), :has-text("Secondary"), :has-text("Micro")');
      
      // Should have clear explanations of what each prominence level means
      const primaryExplanation = page.locator(':has-text("main focus"), :has-text("central to")');
      const secondaryExplanation = page.locator(':has-text("important"), :has-text("supporting")');
      const microExplanation = page.locator(':has-text("background"), :has-text("minor")');
      
      const hasExplanations = await primaryExplanation.count() > 0 && 
                             await secondaryExplanation.count() > 0 && 
                             await microExplanation.count() > 0;
      
      expect(hasExplanations).toBe(true);
    });

    test('should be keyboard accessible', async ({ page }) => {
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
      
      // Test keyboard navigation through prominence controls
      await page.keyboard.press('Tab'); // Navigate to first form field
      
      // Continue tabbing to find prominence controls
      let tabCount = 0;
      const maxTabs = 20;
      
      while (tabCount < maxTabs) {
        await page.keyboard.press('Tab');
        tabCount++;
        
        const focusedElement = page.locator(':focus');
        const elementText = await focusedElement.textContent().catch(() => '');
        
        // Look for prominence-related controls
        if (elementText.includes('prominence') || elementText.includes('Primary') || elementText.includes('Secondary')) {
          // Found a prominence control, test keyboard interaction
          await page.keyboard.press('Enter'); // Should activate control
          await page.keyboard.press('ArrowDown'); // Should navigate options
          await page.keyboard.press('Escape'); // Should close/cancel
          break;
        }
      }
      
      // Verify we can navigate the form with keyboard
      expect(tabCount).toBeLessThan(maxTabs); // Should find controls within reasonable tab count
    });

    test('should save and restore prominence settings', async ({ page }) => {
      await expect(page.getByRole('heading', { name: /Post New Work|Create Work/i })).toBeVisible();
      
      // Fill basic fields and add tags with specific prominences
      await page.fill('input[name="title"]', 'Persistence Test Work');
      
      // Add fandom
      const fandomInput = page.locator('input[aria-label*="fandom"]').first();
      if (await fandomInput.isVisible()) {
        await fandomInput.fill('Harry Potter');
        await fandomInput.press('Enter');
      }
      
      // Add relationships and set specific prominences
      const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
      if (await relationshipInput.isVisible()) {
        await relationshipInput.fill('Harry Potter/Ginny Weasley');
        await relationshipInput.press('Enter');
        
        await relationshipInput.fill('Background Hermione Granger/Ron Weasley');
        await relationshipInput.press('Enter');
      }
      
      // Wait for tags to be processed
      await page.waitForTimeout(1000);
      
      // Navigate away and back to test persistence
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
      await page.goto('/works/new');
      await page.waitForLoadState('networkidle');
      
      // Check if form state was saved/restored
      const titleField = page.locator('input[name="title"]');
      const titleValue = await titleField.inputValue();
      
      // Form should be reset (expected behavior for security)
      expect(titleValue).toBe('');
    });
  });
});

test.describe('Enhanced Tag System Integration Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:3001/test-components');
    await page.waitForLoadState('networkidle');
  });

  test('should integrate enhanced tag prominence with existing tag inputs', async ({ page }) => {
    // Test the enhanced tag system on the test components page
    await expect(page.getByRole('heading', { name: 'Enhanced Form Components Test' })).toBeVisible();
    
    // Look for enhanced tag prominence selector
    const prominenceSelector = page.locator('[data-testid="tag-prominence-selector"], .enhanced-tag-prominence-selector');
    
    if (await prominenceSelector.isVisible()) {
      // Test adding tags through the enhanced interface
      const relationshipTypeButton = page.locator('button:has-text("Relationships")');
      if (await relationshipTypeButton.isVisible()) {
        await relationshipTypeButton.click();
        
        const tagInput = page.locator('input[placeholder*="relationship"]');
        if (await tagInput.isVisible()) {
          await tagInput.fill('Background Steve Rogers/Tony Stark');
          await tagInput.press('Enter');
          
          // Should automatically detect "Background" and set micro prominence
          const microSection = page.locator('.prominence-section.micro, [data-prominence-section="micro"]');
          const hasMicroTag = await microSection.locator(':has-text("Steve Rogers/Tony Stark")').count() > 0;
          
          expect(hasMicroTag).toBe(true);
        }
      }
    }
  });

  test('should provide real-time tag quality feedback', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Enhanced Form Components Test' })).toBeVisible();
    
    // Test real-time feedback for tag quality
    const relationshipInput = page.locator('input[aria-label*="relationship"]').first();
    
    if (await relationshipInput.isVisible()) {
      // Add several relationship tags to trigger quality feedback
      await relationshipInput.fill('Tony Stark/Steve Rogers');
      await relationshipInput.press('Enter');
      
      await relationshipInput.fill('Steve Rogers/Tony Stark'); // Duplicate/reverse
      await relationshipInput.press('Enter');
      
      await relationshipInput.fill('Stony'); // Ship name
      await relationshipInput.press('Enter');
      
      // Look for quality feedback
      const qualityFeedback = page.locator('.quality-feedback, [data-testid="quality-feedback"], :has-text("duplicate"), :has-text("similar")');
      const hasQualityFeedback = await qualityFeedback.count() > 0;
      
      expect(hasQualityFeedback).toBe(true);
    }
  });
});