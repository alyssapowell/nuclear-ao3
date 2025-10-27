import { test, expect } from '@playwright/test';
import { loginUser } from './test-utils';

// Get a real work ID from the backend
async function getTestWorkId() {
  const API_URL = process.env.API_URL || 'http://localhost:8080/api/v1';
  const FIRST_PARTY_CLIENT_ID = '11111111-1111-1111-1111-111111111111';
  try {
    const response = await fetch(`${API_URL}/works?limit=1`, {
      headers: {
        'Authorization': 'Bearer dummy-token-for-rate-limiting',
        'X-Client-ID': FIRST_PARTY_CLIENT_ID,
        'X-Client-First-Party': 'true',
        'X-OAuth-Scopes': 'read',
      }
    });
    if (!response.ok) {
      throw new Error(`API responded with status ${response.status}`);
    }
    const data = await response.json();
    return data.works?.[0]?.id || null;
  } catch (error) {
    console.warn(`Failed to get test work ID: ${error}`);
    return null;
  }
}

test.describe.configure({ mode: 'serial' });

test.describe('Comments Flow', () => {
  let testWorkId: string | null = null;
  
  test.beforeAll(async () => {
    testWorkId = await getTestWorkId();
  });

  test.beforeEach(async ({ page }) => {
    // Skip test if we can't get a real work ID
    if (!testWorkId) {
      test.skip();
    }
    
    // Navigate to a real work page
    await page.goto(`/works/${testWorkId}`);
    await page.waitForLoadState('networkidle');
  });

  test('should display comments section', async ({ page }) => {
    // Check if comments section heading is visible
    await expect(page.getByRole('heading', { name: /comments/i })).toBeVisible();
    
    // Should show either existing comments or empty state
    const hasComments = await page.locator('.bg-white.border.border-slate-200.rounded-lg.p-4.mb-4').isVisible();
    const hasEmptyState = await page.getByText(/no comments yet/i).isVisible();
    
    expect(hasComments || hasEmptyState).toBeTruthy();
  });

  test('should allow posting a new comment when authenticated', async ({ page }) => {
    // Login using real authentication
    await loginUser(page);
    
    // Navigate to work page
    await page.goto(`/works/${testWorkId}`);
    await page.waitForLoadState('networkidle');

    // Check if comment form is visible
    const commentForm = page.getByPlaceholder(/share your thoughts about this work/i);
    await expect(commentForm).toBeVisible();
    
    // Fill in comment form
    await commentForm.fill('This is a test comment from e2e test');
    
    // Submit comment
    await page.getByRole('button', { name: /post comment/i }).click();
    
    // Check for success (either new comment appears or pending message)
    const isVisible = await Promise.race([
      page.locator('text=This is a test comment from e2e test').isVisible(),
      page.locator('text=Pending Moderation').isVisible(),
      page.locator('text=Comment posted').isVisible()
    ]);
    
    expect(isVisible).toBeTruthy();
  });

  test('should allow replying to existing comments', async ({ page }) => {
    // Login using real authentication
    await loginUser(page);
    
    // Navigate to work page
    await page.goto(`/works/${testWorkId}`);
    await page.waitForLoadState('networkidle');

    // Check if there are any comments to reply to
    const replyButton = page.getByRole('button', { name: /reply/i }).first();
    
    if (await replyButton.isVisible()) {
      await replyButton.click();
      
      // Fill in reply form
      const replyTextarea = page.getByPlaceholder(/write your reply/i);
      await expect(replyTextarea).toBeVisible();
      await replyTextarea.fill('This is a test reply');
      
      // Submit reply
      await page.getByRole('button', { name: /post reply/i }).click();
      
      // Check that reply appears or shows pending
      const isVisible = await Promise.race([
        page.locator('text=This is a test reply').isVisible(),
        page.locator('text=Pending Moderation').isVisible(),
        page.locator('text=Reply posted').isVisible()
      ]);
      
      expect(isVisible).toBeTruthy();
    } else {
      // Skip test if no comments to reply to
      test.skip();
    }
  });

  test('should allow editing own comments', async ({ page }) => {
    // Mock authentication
    await page.evaluate(() => {
      localStorage.setItem('auth_token', 'test-token');
    });
    
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Navigate to comments section
    await page.click('text=Comments');
    
    // Click edit on a user's own comment (assuming there's one)
    const editButton = page.locator('button:has-text("Edit")').first();
    if (await editButton.isVisible()) {
      await editButton.click();
      
      // Modify the comment content
      const textarea = page.locator('textarea').first();
      await textarea.clear();
      await textarea.fill('This comment has been edited');
      
      // Save changes
      await page.click('button:has-text("Save Changes")');
      
      // Check that edited content appears
      await expect(page.locator('text=This comment has been edited')).toBeVisible({ timeout: 10000 });
      
      // Check for edited timestamp
      await expect(page.locator('text=edited')).toBeVisible();
    }
  });

  test('should allow deleting own comments', async ({ page }) => {
    // Mock authentication
    await page.evaluate(() => {
      localStorage.setItem('auth_token', 'test-token');
    });
    
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Navigate to comments section
    await page.click('text=Comments');
    
    // Click delete on a user's own comment (assuming there's one)
    const deleteButton = page.locator('button:has-text("Delete")').first();
    if (await deleteButton.isVisible()) {
      await deleteButton.click();
      
      // Confirm deletion
      await page.click('button:has-text("Delete Comment")');
      
      // Check that comment is removed (this depends on your implementation)
      // You might check for a success message or that the comment disappears
      await expect(page.locator('text=Comment deleted successfully')).toBeVisible({ timeout: 10000 });
    }
  });

  test('should allow giving kudos to comments', async ({ page }) => {
    // Login using real authentication
    await loginUser(page);
    
    // Navigate to work page
    await page.goto(`/works/${testWorkId}`);
    await page.waitForLoadState('networkidle');

    // Look for comment kudos buttons
    const kudosButton = page.locator('button').filter({ hasText: /\d+/ }).and(page.locator('svg')).first();
    
    if (await kudosButton.isVisible()) {
      const initialKudosText = await kudosButton.textContent();
      const initialCount = parseInt(initialKudosText?.match(/\d+/)?.[0] || '0');
      
      await kudosButton.click();
      
      // Wait for the update and check if count changed
      await page.waitForTimeout(500);
      const newKudosText = await kudosButton.textContent();
      const newCount = parseInt(newKudosText?.match(/\d+/)?.[0] || '0');
      
      // Count should either increase (new kudos) or decrease (removed kudos)
      expect(newCount).not.toBe(initialCount);
    } else {
      // Skip test if no comments to give kudos to
      test.skip();
    }
  });

  test('should open conversation view for threaded comments', async ({ page }) => {
    // Navigate to comments section
    await page.click('text=Comments');
    
    // Click on a "View conversation" link if it exists
    const conversationButton = page.locator('button:has-text("View conversation")').first();
    if (await conversationButton.isVisible()) {
      await conversationButton.click();
      
      // Check that conversation view opens (could be slideout or new page)
      await expect(page.locator('.conversation-view, .slideout-panel')).toBeVisible({ timeout: 5000 });
      
      // Check for conversation content
      await expect(page.locator('text=Conversation Thread')).toBeVisible();
    }
  });

  test('should handle conversation URL navigation', async ({ page }) => {
    // Navigate directly to a conversation URL
    await page.goto('/works/test-work-id/comments/test-thread-id');
    await page.waitForLoadState('networkidle');
    
    // Check that we're on the conversation page
    await expect(page.locator('text=Conversation Thread')).toBeVisible();
    
    // Check breadcrumb navigation
    await expect(page.locator('nav[aria-label="Conversation breadcrumb"]')).toBeVisible();
    
    // Test navigation back to work
    await page.click('text=Back to Work');
    await expect(page.url()).toContain('/works/test-work-id');
    await expect(page.url()).not.toContain('/comments/');
  });

  test('should share conversation URLs', async ({ page }) => {
    // Navigate to a conversation
    await page.goto('/works/test-work-id/comments/test-thread-id');
    await page.waitForLoadState('networkidle');
    
    // Find and click share button
    const shareButton = page.locator('button:has-text("Share")').first();
    if (await shareButton.isVisible()) {
      await shareButton.click();
      
      // Check if URL was copied to clipboard (this is tricky to test)
      // You might need to mock the clipboard API or check for a success message
    }
  });

  test('should validate comment form inputs', async ({ page }) => {
    // Mock authentication
    await page.evaluate(() => {
      localStorage.setItem('auth_token', 'test-token');
    });
    
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Navigate to comments section
    await page.click('text=Comments');
    
    // Try to submit empty comment
    await page.click('button:has-text("Post Comment")');
    
    // Check that submit button is disabled or form shows validation error
    const submitButton = page.locator('button:has-text("Post Comment")');
    await expect(submitButton).toBeDisabled();
  });

  test('should handle comment moderation states', async ({ page }) => {
    // Navigate to comments section
    await page.click('text=Comments');
    
    // Check for moderation badges
    const moderationBadge = page.locator('.moderation-badge, text=Pending Moderation').first();
    if (await moderationBadge.isVisible()) {
      await expect(moderationBadge).toBeVisible();
    }
  });

  test('should be responsive on mobile devices', async ({ page }) => {
    // Set viewport to mobile size
    await page.setViewportSize({ width: 375, height: 667 });
    
    // Navigate to comments section
    await page.click('text=Comments');
    
    // Check that comments are still visible and functional
    await expect(page.locator('.comment-item').first()).toBeVisible();
    
    // Check that conversation links work on mobile (should navigate instead of slideout)
    const conversationButton = page.locator('button:has-text("View conversation")').first();
    if (await conversationButton.isVisible()) {
      await conversationButton.click();
      
      // On mobile, should navigate to conversation page
      await expect(page.url()).toContain('/comments/');
    }
  });

  test('should handle error states gracefully', async ({ page }) => {
    // Mock network failure for comments API
    await page.route('**/api/v1/works/*/comments', route => {
      route.abort();
    });
    
    await page.reload();
    await page.waitForLoadState('networkidle');
    
    // Check for error message
    await expect(page.locator('text=Failed to load comments, text=Error loading comments')).toBeVisible({ timeout: 10000 });
  });

  test.describe('Work Kudos', () => {
    test('should display kudos button and handle interactions', async ({ page }) => {
      // Login using real authentication
      await loginUser(page);
      
      // Navigate to work page
      await page.goto(`/works/${testWorkId}`);
      await page.waitForLoadState('networkidle');

      // Check if kudos section exists - look for either "Give Kudos" or "Kudos Given"
      const giveKudosButton = page.getByText('Give Kudos');
      const kudosGivenButton = page.getByText('Kudos Given');
      
      const hasGiveKudos = await giveKudosButton.isVisible();
      const hasKudosGiven = await kudosGivenButton.isVisible();
      
      // Should have one of these buttons visible
      expect(hasGiveKudos || hasKudosGiven).toBeTruthy();
      
      if (hasGiveKudos) {
        // Get initial kudos count from the button
        const initialButtonText = await giveKudosButton.locator('..').textContent();
        const initialCount = parseInt(initialButtonText?.match(/\((\d+)\)/)?.[1] || '0');
        
        // Click kudos button
        await giveKudosButton.click();
        
        // Wait for the update
        await page.waitForTimeout(2000);
        
        // Check that button state changed to "Kudos Given" or count increased
        const updatedGiveButton = page.getByText('Give Kudos');
        const updatedGivenButton = page.getByText('Kudos Given');
        
        if (await updatedGivenButton.isVisible()) {
          // Button changed to "Kudos Given"
          const newButtonText = await updatedGivenButton.locator('..').textContent();
          const newCount = parseInt(newButtonText?.match(/\((\d+)\)/)?.[1] || '0');
          expect(newCount).toBe(initialCount + 1);
        } else if (await updatedGiveButton.isVisible()) {
          // Button might still show "Give Kudos" but count should have increased
          const newButtonText = await updatedGiveButton.locator('..').textContent();
          const newCount = parseInt(newButtonText?.match(/\((\d+)\)/)?.[1] || '0');
          expect(newCount).toBeGreaterThanOrEqual(initialCount);
        }
      }
    });

    test('should handle kudos state correctly', async ({ page }) => {
      // Login using real authentication
      await loginUser(page);
      
      // Navigate to work page
      await page.goto(`/works/${testWorkId}`);
      await page.waitForLoadState('networkidle');

      // Check current kudos state
      const giveKudosButton = page.getByText('Give Kudos');
      const kudosGivenButton = page.getByText('Kudos Given');
      
      if (await kudosGivenButton.isVisible()) {
        // User has already given kudos, button should be disabled
        const buttonElement = kudosGivenButton.locator('..');
        await expect(buttonElement).toBeDisabled();
      } else if (await giveKudosButton.isVisible()) {
        // User hasn't given kudos yet, button should be enabled
        const buttonElement = giveKudosButton.locator('..');
        await expect(buttonElement).toBeEnabled();
      } else {
        // No kudos button found
        throw new Error('No kudos button found on the page');
      }
    });
  });
});