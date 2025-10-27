import { test, expect, type Page, type BrowserContext } from '@playwright/test';

// Multi-user test utilities for Nuclear AO3
class MultiUserTestUtils {
  constructor(private page: Page) {}

  // User management utilities
  async createUser(username: string, email: string, password: string = 'testpass123') {
    await this.page.goto('/auth/register');
    await this.page.fill('[data-testid="username"]', username);
    await this.page.fill('[data-testid="email"]', email);
    await this.page.fill('[data-testid="password"]', password);
    await this.page.fill('[data-testid="confirm-password"]', password);
    
    // Click register and wait for either success (redirect to /) or error
    await this.page.click('[data-testid="register-button"]');
    
    try {
      // Try to wait for redirect to home (successful registration)
      await this.page.waitForURL('/', { timeout: 10000 });
    } catch {
      // If redirect fails, check if we're still on registration page with success
      // Sometimes Apollo errors don't prevent the registration from working
      await this.page.waitForTimeout(2000);
      
      // If we're still on register page, check for errors or try login
      if (this.page.url().includes('/auth/register')) {
        // Try logging in with the credentials to see if account was created
        await this.page.goto('/auth/login');
        await this.page.fill('[data-testid="username"]', email);
        await this.page.fill('[data-testid="password"]', password);
        await this.page.click('[data-testid="login-button"]');
        await this.page.waitForURL('/');
      }
    }
    
    return { username, email, password };
  }

  async loginAs(email: string, password: string = 'testpass123') {
    await this.page.goto('/auth/login');
    await this.page.fill('[data-testid="username"]', email); // login uses email field
    await this.page.fill('[data-testid="password"]', password);
    await this.page.click('[data-testid="login-button"]');
    await this.page.waitForURL('/');
  }

  async logout() {
    await this.page.click('[data-testid="user-menu"]');
    await this.page.click('[data-testid="logout-button"]');
    await this.page.waitForURL('/');
  }

  async createWork(workData: {
    title: string;
    summary?: string;
    content: string;
    fandoms: string[];
    rating?: string;
    warnings?: string[];
    categories?: string[];
    status?: string;
    restrictedToUsers?: boolean;
    isAnonymous?: boolean;
    commentPolicy?: string;
  }) {
    await this.page.goto('/works/new');
    
    // Fill basic work information
    await this.page.fill('[name="title"]', workData.title);
    if (workData.summary) {
      await this.page.fill('[name="summary"]', workData.summary);
    }
    
    // Add fandoms (required)
    for (const fandom of workData.fandoms) {
      await this.page.fill('#fandoms', fandom);
      await this.page.press('#fandoms', 'Enter');
      await this.page.waitForTimeout(500);
    }
    
    // Set rating
    if (workData.rating) {
      await this.page.selectOption('[name="rating"]', workData.rating);
    }
    
    // Set warnings
    if (workData.warnings) {
      for (const warning of workData.warnings) {
        await this.page.check(`input[type="checkbox"][value="${warning}"]`);
      }
    }
    
    // Set categories
    if (workData.categories) {
      for (const category of workData.categories) {
        await this.page.check(`input[type="checkbox"][value="${category}"]`);
      }
    }
    
    // Set privacy settings
    if (workData.restrictedToUsers) {
      await this.page.check('[name="restricted_to_users"]');
    }
    
    if (workData.isAnonymous) {
      await this.page.check('[name="is_anonymous"]');
    }
    
    if (workData.commentPolicy) {
      await this.page.selectOption('[name="comment_policy"]', workData.commentPolicy);
    }
    
    // Set status
    if (workData.status) {
      await this.page.selectOption('[name="status"]', workData.status);
    }
    
    // Fill content
    await this.page.fill('[name="chapterContent"]', workData.content);
    
    // Submit
    await this.page.click('button[type="submit"]');
    await this.page.waitForURL(/\/works\/[a-f0-9-]+$/);
    
    // Extract work ID from URL
    const workId = this.page.url().split('/').pop();
    return { ...workData, id: workId };
  }

  async goToWork(workId: string) {
    await this.page.goto(`/works/${workId}`);
  }

  async bookmarkWork(workId: string, notes?: string, tags?: string[], isPrivate?: boolean) {
    await this.goToWork(workId);
    await this.page.click('[data-testid="bookmark-button"]');
    
    if (notes) {
      await this.page.fill('[name="notes"]', notes);
    }
    
    if (tags) {
      await this.page.fill('[name="tags"]', tags.join(', '));
    }
    
    if (isPrivate) {
      await this.page.check('[name="is_private"]');
    }
    
    await this.page.click('[data-testid="save-bookmark"]');
    await this.page.waitForSelector('text=Bookmark saved successfully');
  }

  async subscribeToWork(workId: string, events: string[] = ['new_chapter'], frequency = 'immediate') {
    await this.goToWork(workId);
    await this.page.click('[data-testid="subscription-button"]');
    
    // Select events
    for (const event of events) {
      await this.page.check(`input[value="${event}"]`);
    }
    
    // Select frequency
    await this.page.selectOption('[name="frequency"]', frequency);
    
    await this.page.click('[data-testid="create-subscription"]');
    await this.page.waitForSelector('text=Successfully subscribed');
  }

  async commentOnWork(workId: string, comment: string) {
    await this.goToWork(workId);
    await this.page.fill('[data-testid="comment-textarea"]', comment);
    await this.page.click('[data-testid="submit-comment"]');
    await this.page.waitForSelector(`text=${comment}`);
  }

  async giveKudos(workId: string) {
    await this.goToWork(workId);
    await this.page.click('[data-testid="kudos-button"]');
    await this.page.waitForSelector('text=Kudos given');
  }

  async checkWorkAccess(workId: string): Promise<'accessible' | 'restricted' | 'not-found'> {
    try {
      await this.goToWork(workId);
      
      if (await this.page.isVisible('text=This work is restricted')) {
        return 'restricted';
      }
      
      if (await this.page.isVisible('h1')) {
        return 'accessible';
      }
      
      return 'not-found';
    } catch {
      return 'not-found';
    }
  }

  async searchForWork(title: string) {
    await this.page.goto('/search');
    await this.page.fill('[data-testid="search-input"]', title);
    await this.page.click('[data-testid="search-button"]');
    await this.page.waitForLoadState('networkidle');
  }
}

// Test scenarios for multiple users
test.describe('Multi-User Work Management', () => {
  let authorContext: BrowserContext;
  let readerContext: BrowserContext;
  let anonymousContext: BrowserContext;
  
  let authorPage: Page;
  let readerPage: Page;
  let anonymousPage: Page;
  
  let authorUtils: MultiUserTestUtils;
  let readerUtils: MultiUserTestUtils;
  let anonymousUtils: MultiUserTestUtils;

  test.beforeAll(async ({ browser }) => {
    // Create separate browser contexts for different user types
    authorContext = await browser.newContext();
    readerContext = await browser.newContext();
    anonymousContext = await browser.newContext();
    
    authorPage = await authorContext.newPage();
    readerPage = await readerContext.newPage();
    anonymousPage = await anonymousContext.newPage();
    
    authorUtils = new MultiUserTestUtils(authorPage);
    readerUtils = new MultiUserTestUtils(readerPage);
    anonymousUtils = new MultiUserTestUtils(anonymousPage);
  });

  test.afterAll(async () => {
    await authorContext.close();
    await readerContext.close();
    await anonymousContext.close();
  });

  test.describe('User Creation and Authentication', () => {
    test('should create multiple test users successfully', async () => {
      // Create author user
      await authorUtils.createUser('testauthor', 'author@example.com');
      await authorPage.goto('/dashboard');
      await expect(authorPage.locator('h1')).toContainText('Dashboard');
      
      // Create reader user
      await readerUtils.createUser('testreader', 'reader@example.com');
      await readerPage.goto('/dashboard');
      await expect(readerPage.locator('h1')).toContainText('Dashboard');
      
      // Verify both users can navigate independently
      await authorPage.goto('/works/new');
      await expect(authorPage.locator('h1')).toContainText('Post New Work');
      
      await readerPage.goto('/search');
      await expect(readerPage.locator('[data-testid="search-input"]')).toBeVisible();
    });

    test('should maintain separate user sessions', async () => {
      // Author should see their dashboard
      await authorPage.goto('/dashboard');
      await expect(authorPage.locator('h1')).toContainText('Dashboard');
      
      // Reader should see their own dashboard
      await readerPage.goto('/dashboard');
      await expect(readerPage.locator('h1')).toContainText('Dashboard');
      
      // Anonymous user should be redirected to login
      await anonymousPage.goto('/dashboard');
      await expect(anonymousPage).toHaveURL(/\/auth\/login/);
    });
  });

  test.describe('Work Access Control and Privacy', () => {
    let publicWork: any;
    let restrictedWork: any;
    let anonymousWork: any;

    test('should create works with different privacy settings', async () => {
      // Create public work
      publicWork = await authorUtils.createWork({
        title: 'Public Work for All',
        content: 'This work should be visible to everyone.',
        fandoms: ['Original Work'],
        status: 'posted'
      });

      // Create restricted work
      restrictedWork = await authorUtils.createWork({
        title: 'Restricted Work',
        content: 'This work should only be visible to registered users.',
        fandoms: ['Original Work'],
        status: 'posted',
        restrictedToUsers: true
      });

      // Create anonymous work
      anonymousWork = await authorUtils.createWork({
        title: 'Anonymous Work',
        content: 'This work is posted anonymously.',
        fandoms: ['Original Work'],
        status: 'posted',
        isAnonymous: true
      });

      // Verify works were created
      await expect(authorPage.locator('h1')).toContainText('Anonymous Work');
    });

    test('should enforce access control for public works', async () => {
      // Registered user should access public work
      const readerAccess = await readerUtils.checkWorkAccess(publicWork.id);
      expect(readerAccess).toBe('accessible');
      
      // Anonymous user should also access public work
      const anonymousAccess = await anonymousUtils.checkWorkAccess(publicWork.id);
      expect(anonymousAccess).toBe('accessible');
      
      // Verify work content is visible
      await readerPage.goto(`/works/${publicWork.id}`);
      await expect(readerPage.locator('.prose')).toContainText('This work should be visible to everyone');
    });

    test('should enforce access control for restricted works', async () => {
      // Registered user should access restricted work
      const readerAccess = await readerUtils.checkWorkAccess(restrictedWork.id);
      expect(readerAccess).toBe('accessible');
      
      // Anonymous user should be restricted
      const anonymousAccess = await anonymousUtils.checkWorkAccess(restrictedWork.id);
      expect(anonymousAccess).toBe('restricted');
      
      // Verify restriction message for anonymous users
      await anonymousPage.goto(`/works/${restrictedWork.id}`);
      await expect(anonymousPage.locator('text=This work is restricted')).toBeVisible();
    });

    test('should handle anonymous works correctly', async () => {
      // Work should be accessible but show as anonymous
      await readerPage.goto(`/works/${anonymousWork.id}`);
      await expect(readerPage.locator('text=Anonymous')).toBeVisible();
      await expect(readerPage.locator('text=testauthor')).not.toBeVisible();
      
      // Content should be visible
      await expect(readerPage.locator('.prose')).toContainText('This work is posted anonymously');
    });

    test('should show works in search results based on access', async () => {
      // Search as registered user - should see all works
      await readerUtils.searchForWork('Work');
      await expect(readerPage.locator('text=Public Work for All')).toBeVisible();
      await expect(readerPage.locator('text=Restricted Work')).toBeVisible();
      await expect(readerPage.locator('text=Anonymous Work')).toBeVisible();
      
      // Search as anonymous user - should only see public and anonymous works
      await anonymousUtils.searchForWork('Work');
      await expect(anonymousPage.locator('text=Public Work for All')).toBeVisible();
      await expect(anonymousPage.locator('text=Anonymous Work')).toBeVisible();
      await expect(anonymousPage.locator('text=Restricted Work')).not.toBeVisible();
    });
  });

  test.describe('Author vs Reader Permissions', () => {
    let authorWork: any;

    test('should allow only authors to edit their works', async () => {
      // Author creates a work
      authorWork = await authorUtils.createWork({
        title: 'Author Only Editable',
        content: 'Only the author should be able to edit this.',
        fandoms: ['Original Work'],
        status: 'posted'
      });

      // Author should be able to access edit page
      await authorPage.goto(`/works/${authorWork.id}/edit`);
      await expect(authorPage.locator('h1')).toContainText('Edit Work');
      await expect(authorPage.locator('[name="title"]')).toHaveValue('Author Only Editable');

      // Reader should not be able to access edit page
      await readerPage.goto(`/works/${authorWork.id}/edit`);
      await expect(readerPage.locator('text=Forbidden')).toBeVisible();
    });

    test('should show edit controls only to authors', async () => {
      // Author viewing their own work should see edit controls
      await authorPage.goto(`/works/${authorWork.id}`);
      await expect(authorPage.locator('text=Edit')).toBeVisible();
      
      // Reader viewing someone else's work should not see edit controls
      await readerPage.goto(`/works/${authorWork.id}`);
      await expect(readerPage.locator('text=Edit')).not.toBeVisible();
    });

    test('should allow only authors to delete their works', async () => {
      // Create work to delete
      const deleteWork = await authorUtils.createWork({
        title: 'Work to Delete',
        content: 'This will be deleted.',
        fandoms: ['Original Work'],
        status: 'draft'
      });

      // Author should see delete option in dashboard
      await authorPage.goto('/dashboard');
      const workRow = authorPage.locator('.bg-white', { hasText: 'Work to Delete' });
      await expect(workRow.locator('text=Delete')).toBeVisible();

      // Reader should not see author's works in their dashboard
      await readerPage.goto('/dashboard');
      await expect(readerPage.locator('text=Work to Delete')).not.toBeVisible();
    });
  });

  test.describe('Cross-User Interactions', () => {
    let interactionWork: any;

    test('should allow readers to bookmark author works', async () => {
      // Author creates a work
      interactionWork = await authorUtils.createWork({
        title: 'Work for Interactions',
        content: 'This work will be bookmarked and subscribed to.',
        fandoms: ['Original Work'],
        status: 'posted'
      });

      // Reader bookmarks the work
      await readerUtils.bookmarkWork(
        interactionWork.id,
        'This is a great work!',
        ['favorite', 'must-read'],
        false
      );

      // Verify bookmark appears in reader's dashboard
      await readerPage.goto('/dashboard');
      await readerPage.click('text=Bookmarks');
      await expect(readerPage.locator('text=Work for Interactions')).toBeVisible();
      await expect(readerPage.locator('text=This is a great work!')).toBeVisible();

      // Author should not see reader's bookmark in their dashboard
      await authorPage.goto('/dashboard');
      await authorPage.click('text=Bookmarks');
      await expect(authorPage.locator('text=Work for Interactions')).not.toBeVisible();
    });

    test('should allow readers to subscribe to author works', async () => {
      // Reader subscribes to the work
      await readerUtils.subscribeToWork(
        interactionWork.id,
        ['new_chapter', 'work_updated'],
        'immediate'
      );

      // Verify subscription appears in reader's dashboard
      await readerPage.goto('/dashboard');
      await readerPage.click('text=Subscriptions');
      await expect(readerPage.locator('text=Work for Interactions')).toBeVisible();
      await expect(readerPage.locator('text=work')).toBeVisible(); // subscription type

      // Author should not see reader's subscription
      await authorPage.goto('/dashboard');
      await authorPage.click('text=Subscriptions');
      await expect(authorPage.locator('text=Work for Interactions')).not.toBeVisible();
    });

    test('should allow readers to comment on works', async () => {
      // Reader comments on work
      await readerUtils.commentOnWork(
        interactionWork.id,
        'This is an excellent work! I really enjoyed the character development.'
      );

      // Author should see the comment on their work
      await authorPage.goto(`/works/${interactionWork.id}`);
      await expect(authorPage.locator('text=This is an excellent work!')).toBeVisible();
      await expect(authorPage.locator('text=testreader')).toBeVisible();

      // Reader should also see their own comment
      await readerPage.goto(`/works/${interactionWork.id}`);
      await expect(readerPage.locator('text=This is an excellent work!')).toBeVisible();
    });

    test('should allow readers to give kudos', async () => {
      // Get initial kudos count
      await readerPage.goto(`/works/${interactionWork.id}`);
      const initialKudos = await readerPage.locator('[data-testid="kudos-count"]').textContent();

      // Reader gives kudos
      await readerUtils.giveKudos(interactionWork.id);

      // Verify kudos count increased
      await readerPage.reload();
      const newKudos = await readerPage.locator('[data-testid="kudos-count"]').textContent();
      expect(parseInt(newKudos || '0')).toBeGreaterThan(parseInt(initialKudos || '0'));

      // Author should see the kudos on their work
      await authorPage.goto(`/works/${interactionWork.id}`);
      const authorKudos = await authorPage.locator('[data-testid="kudos-count"]').textContent();
      expect(authorKudos).toBe(newKudos);
    });

    test('should prevent duplicate kudos from same user', async () => {
      // Try to give kudos again
      await readerPage.goto(`/works/${interactionWork.id}`);
      const kudosButton = readerPage.locator('[data-testid="kudos-button"]');
      
      // Button should be disabled or show "already given"
      await expect(kudosButton).toBeDisabled();
      // OR await expect(kudosButton).toContainText('Kudos Given');
    });
  });

  test.describe('Anonymous User Limitations', () => {
    let publicWork: any;

    test('should allow anonymous users to read public works', async () => {
      // Create a public work as author
      publicWork = await authorUtils.createWork({
        title: 'Public Work for Anonymous',
        content: 'Anonymous users should be able to read this.',
        fandoms: ['Original Work'],
        status: 'posted'
      });

      // Anonymous user should be able to read
      await anonymousPage.goto(`/works/${publicWork.id}`);
      await expect(anonymousPage.locator('h1')).toContainText('Public Work for Anonymous');
      await expect(anonymousPage.locator('.prose')).toContainText('Anonymous users should be able to read this');
    });

    test('should prevent anonymous users from creating works', async () => {
      // Anonymous user tries to access work creation
      await anonymousPage.goto('/works/new');
      await expect(anonymousPage).toHaveURL(/\/auth\/login/);
      await expect(anonymousPage.locator('text=Please log in')).toBeVisible();
    });

    test('should prevent anonymous users from accessing dashboard', async () => {
      // Anonymous user tries to access dashboard
      await anonymousPage.goto('/dashboard');
      await expect(anonymousPage).toHaveURL(/\/auth\/login/);
    });

    test('should prevent anonymous users from bookmarking', async () => {
      // Anonymous user views work but cannot bookmark
      await anonymousPage.goto(`/works/${publicWork.id}`);
      
      // Bookmark button should either be hidden or require login
      const bookmarkButton = anonymousPage.locator('[data-testid="bookmark-button"]');
      if (await bookmarkButton.isVisible()) {
        await bookmarkButton.click();
        await expect(anonymousPage).toHaveURL(/\/auth\/login/);
      } else {
        await expect(bookmarkButton).not.toBeVisible();
      }
    });

    test('should prevent anonymous users from commenting', async () => {
      // Anonymous user views work but cannot comment
      await anonymousPage.goto(`/works/${publicWork.id}`);
      
      // Comment form should either be hidden or require login
      const commentForm = anonymousPage.locator('[data-testid="comment-form"]');
      if (await commentForm.isVisible()) {
        await anonymousPage.fill('[data-testid="comment-textarea"]', 'Anonymous comment');
        await anonymousPage.click('[data-testid="submit-comment"]');
        await expect(anonymousPage).toHaveURL(/\/auth\/login/);
      } else {
        await expect(commentForm).not.toBeVisible();
        await expect(anonymousPage.locator('text=Log in to comment')).toBeVisible();
      }
    });

    test('should prevent anonymous users from giving kudos', async () => {
      // Anonymous user views work but cannot give kudos
      await anonymousPage.goto(`/works/${publicWork.id}`);
      
      const kudosButton = anonymousPage.locator('[data-testid="kudos-button"]');
      if (await kudosButton.isVisible()) {
        await kudosButton.click();
        await expect(anonymousPage).toHaveURL(/\/auth\/login/);
      } else {
        await expect(kudosButton).not.toBeVisible();
      }
    });
  });

  test.describe('Comment Policy Enforcement', () => {
    test('should enforce different comment policies', async () => {
      // Create work with comments disabled
      const noCommentsWork = await authorUtils.createWork({
        title: 'No Comments Allowed',
        content: 'This work has comments disabled.',
        fandoms: ['Original Work'],
        status: 'posted',
        commentPolicy: 'disabled'
      });

      // Reader should not be able to comment
      await readerPage.goto(`/works/${noCommentsWork.id}`);
      await expect(readerPage.locator('[data-testid="comment-form"]')).not.toBeVisible();
      await expect(readerPage.locator('text=Comments are disabled')).toBeVisible();

      // Create work with users-only comments
      const usersOnlyWork = await authorUtils.createWork({
        title: 'Users Only Comments',
        content: 'Only registered users can comment.',
        fandoms: ['Original Work'],
        status: 'posted',
        commentPolicy: 'users_only'
      });

      // Registered user should be able to comment
      await readerPage.goto(`/works/${usersOnlyWork.id}`);
      await expect(readerPage.locator('[data-testid="comment-form"]')).toBeVisible();

      // Anonymous user should not be able to comment
      await anonymousPage.goto(`/works/${usersOnlyWork.id}`);
      await expect(anonymousPage.locator('[data-testid="comment-form"]')).not.toBeVisible();
      await expect(anonymousPage.locator('text=Only registered users can comment')).toBeVisible();
    });
  });

  test.describe('Work Statistics and Visibility', () => {
    test('should track statistics across different users', async () => {
      // Author creates work
      const statsWork = await authorUtils.createWork({
        title: 'Statistics Test Work',
        content: 'Testing statistics across users.',
        fandoms: ['Original Work'],
        status: 'posted'
      });

      // Multiple users interact with the work
      await readerUtils.giveKudos(statsWork.id);
      await readerUtils.bookmarkWork(statsWork.id, 'Great work!');
      await readerUtils.commentOnWork(statsWork.id, 'Amazing story!');

      // Check statistics from author's perspective
      await authorPage.goto(`/works/${statsWork.id}`);
      await expect(authorPage.locator('[data-testid="kudos-count"]')).toContainText('1');
      await expect(authorPage.locator('[data-testid="bookmark-count"]')).toContainText('1');
      await expect(authorPage.locator('[data-testid="comment-count"]')).toContainText('1');

      // Check statistics from reader's perspective (should be same)
      await readerPage.goto(`/works/${statsWork.id}`);
      await expect(readerPage.locator('[data-testid="kudos-count"]')).toContainText('1');
      await expect(readerPage.locator('[data-testid="bookmark-count"]')).toContainText('1');
      await expect(readerPage.locator('[data-testid="comment-count"]')).toContainText('1');
    });
  });
});