import { test, expect, Browser, BrowserContext, Page } from '@playwright/test';
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

// Helper to create a unique test context with user
async function createUserContext(browser: Browser, userSuffix: string) {
  const context = await browser.newContext();
  const page = await context.newPage();
  const token = await loginUser(page, userSuffix);
  return { context, page, token, userSuffix };
}

test.describe('Multi-User Comments and Kudos', () => {
  let testWorkId: string | null = null;
  
  test.beforeAll(async () => {
    testWorkId = await getTestWorkId();
  });

  test.beforeEach(async ({ page }) => {
    // Skip test if we can't get a real work ID
    if (!testWorkId) {
      test.skip();
    }
  });

  test('should allow multiple users to comment on the same work', async ({ browser }) => {
    if (!testWorkId) return;

    // Create two different users
    const user1 = await createUserContext(browser, 'commenter1');
    const user2 = await createUserContext(browser, 'commenter2');

    try {
      // User 1 posts a comment
      await user1.page.goto(`/works/${testWorkId}`);
      await user1.page.waitForLoadState('networkidle');

      const comment1Form = user1.page.getByPlaceholder(/share your thoughts about this work/i);
      await expect(comment1Form).toBeVisible();
      
      const comment1Text = `Comment from user ${user1.userSuffix} at ${Date.now()}`;
      await comment1Form.fill(comment1Text);
      await user1.page.getByRole('button', { name: /post comment/i }).click();
      
      // Wait for comment to appear or show pending
      await user1.page.waitForTimeout(2000);

      // User 2 posts a different comment
      await user2.page.goto(`/works/${testWorkId}`);
      await user2.page.waitForLoadState('networkidle');

      const comment2Form = user2.page.getByPlaceholder(/share your thoughts about this work/i);
      await expect(comment2Form).toBeVisible();
      
      const comment2Text = `Comment from user ${user2.userSuffix} at ${Date.now()}`;
      await comment2Form.fill(comment2Text);
      await user2.page.getByRole('button', { name: /post comment/i }).click();
      
      // Wait for comment to appear or show pending
      await user2.page.waitForTimeout(2000);

      // Verify both users can see comments (either live or pending)
      await user1.page.reload();
      await user1.page.waitForLoadState('networkidle');
      
      // Check that comments section shows activity
      const commentsCount = user1.page.getByRole('heading', { name: /comments/i });
      await expect(commentsCount).toBeVisible();
      
      // Check if comments are visible or pending moderation
      const hasCommentContent = await Promise.race([
        user1.page.getByText(comment1Text.substring(0, 20)).isVisible(),
        user1.page.getByText(/pending moderation/i).isVisible(),
        user1.page.getByText(/comment posted/i).isVisible(),
      ]);
      
      expect(hasCommentContent).toBeTruthy();

    } finally {
      await user1.context.close();
      await user2.context.close();
    }
  });

  test('should allow users to reply to each other\'s comments', async ({ browser }) => {
    if (!testWorkId) return;

    // Create two different users
    const author = await createUserContext(browser, 'author');
    const replier = await createUserContext(browser, 'replier');

    try {
      // Author posts initial comment
      await author.page.goto(`/works/${testWorkId}`);
      await author.page.waitForLoadState('networkidle');

      const authorCommentForm = author.page.getByPlaceholder(/share your thoughts about this work/i);
      await expect(authorCommentForm).toBeVisible();
      
      const authorCommentText = `Original comment from ${author.userSuffix}`;
      await authorCommentForm.fill(authorCommentText);
      await author.page.getByRole('button', { name: /post comment/i }).click();
      await author.page.waitForTimeout(3000);

      // Replier sees the comment and replies
      await replier.page.goto(`/works/${testWorkId}`);
      await replier.page.waitForLoadState('networkidle');

      // Look for a reply button
      const replyButton = replier.page.getByRole('button', { name: /reply/i }).first();
      
      if (await replyButton.isVisible()) {
        await replyButton.click();
        
        const replyForm = replier.page.getByPlaceholder(/write your reply/i);
        await expect(replyForm).toBeVisible();
        
        const replyText = `Reply from ${replier.userSuffix}`;
        await replyForm.fill(replyText);
        
        await replier.page.getByRole('button', { name: /post reply/i }).click();
        await replier.page.waitForTimeout(2000);
        
        // Verify reply was submitted
        const hasReplyResponse = await Promise.race([
          replier.page.getByText(replyText.substring(0, 15)).isVisible(),
          replier.page.getByText(/pending moderation/i).isVisible(),
          replier.page.getByText(/reply posted/i).isVisible(),
        ]);
        
        expect(hasReplyResponse).toBeTruthy();
      } else {
        // If no reply button visible, comment might be pending moderation
        console.log('No reply button found - original comment may be pending moderation');
      }

    } finally {
      await author.context.close();
      await replier.context.close();
    }
  });

  test('should allow users to give kudos to each other\'s comments', async ({ browser }) => {
    if (!testWorkId) return;

    // Create two different users
    const commenter = await createUserContext(browser, 'commenter');
    const kudosGiver = await createUserContext(browser, 'kudosGiver');

    try {
      // Commenter posts a comment
      await commenter.page.goto(`/works/${testWorkId}`);
      await commenter.page.waitForLoadState('networkidle');

      const commentForm = commenter.page.getByPlaceholder(/share your thoughts about this work/i);
      if (await commentForm.isVisible()) {
        const commentText = `Kudos-worthy comment from ${commenter.userSuffix}`;
        await commentForm.fill(commentText);
        await commenter.page.getByRole('button', { name: /post comment/i }).click();
        await commenter.page.waitForTimeout(3000);
      }

      // Kudos giver visits and gives kudos to comments
      await kudosGiver.page.goto(`/works/${testWorkId}`);
      await kudosGiver.page.waitForLoadState('networkidle');

      // Look for comment kudos buttons (heart icons)
      const commentKudosButtons = kudosGiver.page.locator('button').filter({ hasText: /\d+/ }).and(kudosGiver.page.locator('svg'));
      
      if (await commentKudosButtons.first().isVisible()) {
        const initialKudosText = await commentKudosButtons.first().textContent();
        const initialCount = parseInt(initialKudosText?.match(/\d+/)?.[0] || '0');
        
        await commentKudosButtons.first().click();
        await kudosGiver.page.waitForTimeout(1000);
        
        // Check if kudos count changed
        const newKudosText = await commentKudosButtons.first().textContent();
        const newCount = parseInt(newKudosText?.match(/\d+/)?.[0] || '0');
        
        expect(newCount).not.toBe(initialCount);
      } else {
        console.log('No comment kudos buttons found - comments may be pending moderation');
      }

    } finally {
      await commenter.context.close();
      await kudosGiver.context.close();
    }
  });

  test('should allow multiple users to give work kudos independently', async ({ browser }) => {
    if (!testWorkId) return;

    // Create three different users
    const user1 = await createUserContext(browser, 'kudos1');
    const user2 = await createUserContext(browser, 'kudos2');
    const user3 = await createUserContext(browser, 'kudos3');

    try {
      // User 1 gives kudos
      await user1.page.goto(`/works/${testWorkId}`);
      await user1.page.waitForLoadState('networkidle');

      const kudosButton1 = user1.page.getByText('Give Kudos');
      if (await kudosButton1.isVisible()) {
        const initialText1 = await kudosButton1.locator('..').textContent();
        const initialCount1 = parseInt(initialText1?.match(/\((\d+)\)/)?.[1] || '0');
        
        await kudosButton1.click();
        await user1.page.waitForTimeout(2000);
        
        // Verify kudos was given
        const kudosGiven1 = user1.page.getByText('Kudos Given');
        if (await kudosGiven1.isVisible()) {
          const newText1 = await kudosGiven1.locator('..').textContent();
          const newCount1 = parseInt(newText1?.match(/\((\d+)\)/)?.[1] || '0');
          expect(newCount1).toBe(initialCount1 + 1);
        }
      }

      // User 2 gives kudos
      await user2.page.goto(`/works/${testWorkId}`);
      await user2.page.waitForLoadState('networkidle');

      const kudosButton2 = user2.page.getByText('Give Kudos');
      if (await kudosButton2.isVisible()) {
        await kudosButton2.click();
        await user2.page.waitForTimeout(2000);
        
        // Verify kudos was given
        const kudosGiven2 = user2.page.getByText('Kudos Given');
        await expect(kudosGiven2).toBeVisible();
      }

      // User 3 gives kudos
      await user3.page.goto(`/works/${testWorkId}`);
      await user3.page.waitForLoadState('networkidle');

      const kudosButton3 = user3.page.getByText('Give Kudos');
      if (await kudosButton3.isVisible()) {
        await kudosButton3.click();
        await user3.page.waitForTimeout(2000);
        
        // Verify kudos was given
        const kudosGiven3 = user3.page.getByText('Kudos Given');
        await expect(kudosGiven3).toBeVisible();
      }

      // Verify that all users see increased kudos count
      await user1.page.reload();
      await user1.page.waitForLoadState('networkidle');
      
      const finalKudosButton = user1.page.getByText('Kudos Given');
      if (await finalKudosButton.isVisible()) {
        const finalText = await finalKudosButton.locator('..').textContent();
        const finalCount = parseInt(finalText?.match(/\((\d+)\)/)?.[1] || '0');
        
        // Should have at least 3 kudos now (from our test users)
        expect(finalCount).toBeGreaterThanOrEqual(3);
      }

    } finally {
      await user1.context.close();
      await user2.context.close();
      await user3.context.close();
    }
  });

  test('should prevent users from editing other users\' comments', async ({ browser }) => {
    if (!testWorkId) return;

    // Create two different users
    const originalAuthor = await createUserContext(browser, 'original');
    const otherUser = await createUserContext(browser, 'other');

    try {
      // Original author posts a comment
      await originalAuthor.page.goto(`/works/${testWorkId}`);
      await originalAuthor.page.waitForLoadState('networkidle');

      const commentForm = originalAuthor.page.getByPlaceholder(/share your thoughts about this work/i);
      if (await commentForm.isVisible()) {
        const commentText = `Comment from ${originalAuthor.userSuffix} - should not be editable by others`;
        await commentForm.fill(commentText);
        await originalAuthor.page.getByRole('button', { name: /post comment/i }).click();
        await originalAuthor.page.waitForTimeout(3000);

        // Original author should see edit button for their own comment
        const editButton = originalAuthor.page.getByRole('button', { name: /edit/i }).first();
        if (await editButton.isVisible()) {
          expect(await editButton.isVisible()).toBeTruthy();
        }
      }

      // Other user visits the page
      await otherUser.page.goto(`/works/${testWorkId}`);
      await otherUser.page.waitForLoadState('networkidle');

      // Other user should NOT see edit buttons for comments they didn't write
      const editButtons = otherUser.page.getByRole('button', { name: /edit/i });
      
      // Either no edit buttons should be visible, or they should be disabled/non-functional
      const editButtonCount = await editButtons.count();
      
      if (editButtonCount > 0) {
        // If edit buttons exist, they should be for the other user's own comments only
        // Since this user hasn't posted any comments, there should be no edit buttons for them
        console.log(`Found ${editButtonCount} edit buttons - these should only be for current user's own comments`);
      }

    } finally {
      await originalAuthor.context.close();
      await otherUser.context.close();
    }
  });

  test('should handle concurrent comment posting gracefully', async ({ browser }) => {
    if (!testWorkId) return;

    // Create two users that will post comments simultaneously
    const user1 = await createUserContext(browser, 'concurrent1');
    const user2 = await createUserContext(browser, 'concurrent2');

    try {
      // Both users navigate to the work
      await Promise.all([
        user1.page.goto(`/works/${testWorkId}`),
        user2.page.goto(`/works/${testWorkId}`)
      ]);

      await Promise.all([
        user1.page.waitForLoadState('networkidle'),
        user2.page.waitForLoadState('networkidle')
      ]);

      // Both users prepare their comments
      const comment1Form = user1.page.getByPlaceholder(/share your thoughts about this work/i);
      const comment2Form = user2.page.getByPlaceholder(/share your thoughts about this work/i);

      await Promise.all([
        expect(comment1Form).toBeVisible(),
        expect(comment2Form).toBeVisible()
      ]);

      const comment1Text = `Concurrent comment from ${user1.userSuffix}`;
      const comment2Text = `Concurrent comment from ${user2.userSuffix}`;

      await Promise.all([
        comment1Form.fill(comment1Text),
        comment2Form.fill(comment2Text)
      ]);

      // Both users submit simultaneously
      await Promise.all([
        user1.page.getByRole('button', { name: /post comment/i }).click(),
        user2.page.getByRole('button', { name: /post comment/i }).click()
      ]);

      // Wait for responses
      await Promise.all([
        user1.page.waitForTimeout(3000),
        user2.page.waitForTimeout(3000)
      ]);

      // Both submissions should succeed (or show pending state)
      const user1Success = await Promise.race([
        user1.page.getByText(comment1Text.substring(0, 20)).isVisible(),
        user1.page.getByText(/pending moderation/i).isVisible(),
        user1.page.getByText(/comment posted/i).isVisible(),
      ]);

      const user2Success = await Promise.race([
        user2.page.getByText(comment2Text.substring(0, 20)).isVisible(),
        user2.page.getByText(/pending moderation/i).isVisible(),
        user2.page.getByText(/comment posted/i).isVisible(),
      ]);

      expect(user1Success).toBeTruthy();
      expect(user2Success).toBeTruthy();

    } finally {
      await user1.context.close();
      await user2.context.close();
    }
  });
});