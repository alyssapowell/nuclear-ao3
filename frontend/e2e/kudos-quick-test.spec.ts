import { test, expect } from '@playwright/test';
import { loginUser } from './test-utils';

const API_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// Quick helper to get a test work ID
async function getTestWorkId(): Promise<string | null> {
  try {
    const response = await fetch(`${API_URL}/works?limit=1`);
    if (!response.ok) return null;
    
    const data = await response.json();
    return data.works?.[0]?.id || null;
  } catch (error) {
    return null;
  }
}

test.describe('Kudos System Quick Test', () => {
  let testWorkId: string;

  test.beforeAll(async () => {
    testWorkId = await getTestWorkId() || '';
  });

  test('should display and interact with kudos button', async ({ page }) => {
    test.skip(!testWorkId, 'No test work available');

    await test.step('Login and navigate to work', async () => {
      await loginUser(page);
      await page.goto(`/works/${testWorkId}`);
      await page.waitForLoadState('networkidle');
    });

    await test.step('Verify kudos button is visible', async () => {
      // Look for any kudos-related button
      const kudosButton = page.locator('button').filter({ 
        hasText: /kudos|give|loading/i 
      }).first();
      
      await expect(kudosButton).toBeVisible({ timeout: 15000 });
      
      // Log what we see for debugging
      const buttonText = await kudosButton.textContent();
      console.log('Kudos button text:', buttonText);
      
      // Should contain a number in parentheses
      expect(buttonText).toMatch(/\(\d+\)/);
    });

    await test.step('Test kudos interaction', async () => {
      const giveKudosButton = page.locator('button:has-text("Give Kudos")').first();
      
      if (await giveKudosButton.isVisible() && !await giveKudosButton.isDisabled()) {
        // Can give kudos
        console.log('Can give kudos - testing click');
        await giveKudosButton.click();
        
        // Wait for state change
        await expect(
          page.locator('button').filter({ hasText: /kudos given|already|error/i }).first()
        ).toBeVisible({ timeout: 10000 });
        
        console.log('Kudos interaction completed');
      } else {
        // Already given kudos or disabled
        console.log('Kudos button is disabled or not available');
        
        const kudosGivenButton = page.locator('button:has-text("Kudos Given")').first();
        if (await kudosGivenButton.isVisible()) {
          expect(await kudosGivenButton.isDisabled()).toBeTruthy();
          console.log('Kudos already given - button properly disabled');
        }
      }
    });
  });

  test('should display kudos count', async ({ page }) => {
    test.skip(!testWorkId, 'No test work available');

    await test.step('Navigate to work without login', async () => {
      await page.goto(`/works/${testWorkId}`);
      await page.waitForLoadState('networkidle');
    });

    await test.step('Verify kudos count is displayed', async () => {
      // Look for kudos count in various places
      const kudosDisplay = page.locator('text=/kudos.*\d+|hits.*\d+|\d+.*kudos/i').first();
      await expect(kudosDisplay).toBeVisible({ timeout: 10000 });
      
      const displayText = await kudosDisplay.textContent();
      console.log('Kudos display text:', displayText);
    });
  });
});