import { test, expect, type Page } from '@playwright/test';

// Test utilities for common work management actions
class WorkManagementUtils {
  constructor(private page: Page) {}

  async login(username = 'testuser', password = 'testpass123') {
    await this.page.goto('/auth/login');
    await this.page.fill('[data-testid="username"]', username);
    await this.page.fill('[data-testid="password"]', password);
    await this.page.click('[data-testid="login-button"]');
    await this.page.waitForURL('/dashboard');
  }

  async createBasicWork(workData: {
    title: string;
    summary?: string;
    content: string;
    fandoms: string[];
    rating?: string;
    warnings?: string[];
    categories?: string[];
    status?: string;
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
      await this.page.waitForTimeout(500); // Wait for tag to be added
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
    
    // Set status
    if (workData.status) {
      await this.page.selectOption('[name="status"]', workData.status);
    }
    
    // Fill content
    await this.page.fill('[name="chapterContent"]', workData.content);
    
    return {
      title: workData.title,
      content: workData.content,
      fandoms: workData.fandoms,
      rating: workData.rating || 'Not Rated',
      status: workData.status || 'draft'
    };
  }

  async publishWork() {
    await this.page.click('button[type="submit"]');
    await this.page.waitForURL(/\/works\/[a-f0-9-]+$/);
  }

  async saveDraft() {
    await this.page.click('button:has-text("Save as Draft")');
    await this.page.waitForURL(/\/works\/[a-f0-9-]+$/);
  }

  async goToDashboard() {
    await this.page.goto('/dashboard');
    await this.page.waitForLoadState('networkidle');
  }

  async openWorkEditor(workTitle: string) {
    await this.goToDashboard();
    const workRow = this.page.locator('.bg-white', { hasText: workTitle });
    await workRow.locator('text=Edit').click();
    await this.page.waitForURL(/\/works\/[a-f0-9-]+\/edit$/);
  }
}

test.describe('Work Creation and Management', () => {
  let utils: WorkManagementUtils;

  test.beforeEach(async ({ page }) => {
    utils = new WorkManagementUtils(page);
    await utils.login();
  });

  test.describe('Work Creation', () => {
    test('should create and publish a new work successfully', async ({ page }) => {
      const workData = {
        title: 'Test Work for Publishing',
        summary: 'A test work to verify publishing functionality',
        content: 'This is the content of my test work. It has multiple sentences to make it realistic.',
        fandoms: ['Original Work'],
        rating: 'General Audiences',
        warnings: ['no_warnings'],
        categories: ['gen'],
        status: 'posted'
      };

      await utils.createBasicWork(workData);
      await utils.publishWork();

      // Verify work was created and is published
      await expect(page.locator('h1')).toContainText(workData.title);
      await expect(page.locator('.bg-green-100')).toContainText('Published');
      await expect(page.locator('.prose')).toContainText(workData.content);
    });

    test('should create and save a draft work', async ({ page }) => {
      const workData = {
        title: 'Test Draft Work',
        summary: 'A work saved as draft',
        content: 'This is draft content that should not be publicly visible.',
        fandoms: ['Harry Potter'],
        rating: 'Teen And Up Audiences',
        status: 'draft'
      };

      await utils.createBasicWork(workData);
      await utils.saveDraft();

      // Verify draft was created
      await expect(page.locator('h1')).toContainText(workData.title);
      await expect(page.locator('.bg-yellow-100')).toContainText('Draft');
    });

    test('should require title and content', async ({ page }) => {
      await page.goto('/works/new');
      
      // Try to submit without title and content
      await expect(page.locator('button[type="submit"]')).toBeDisabled();
      
      // Add title but no content
      await page.fill('[name="title"]', 'Test Title');
      await expect(page.locator('button[type="submit"]')).toBeDisabled();
      
      // Add content but remove title
      await page.fill('[name="title"]', '');
      await page.fill('[name="chapterContent"]', 'Test content');
      await expect(page.locator('button[type="submit"]')).toBeDisabled();
      
      // Add both title and content
      await page.fill('[name="title"]', 'Test Title');
      await expect(page.locator('button[type="submit"]')).toBeEnabled();
    });

    test('should require at least one fandom', async ({ page }) => {
      await page.goto('/works/new');
      
      await page.fill('[name="title"]', 'Test Work');
      await page.fill('[name="chapterContent"]', 'Test content');
      
      // Should show warning about required fandom
      await expect(page.locator('text=Fandom Selection Required')).toBeVisible();
      
      // Add a fandom
      await page.fill('#fandoms', 'Test Fandom');
      await page.press('#fandoms', 'Enter');
      await page.waitForTimeout(500);
      
      // Should now allow submission
      await expect(page.locator('button[type="submit"]')).toBeEnabled();
    });

    test('should handle tag autocomplete suggestions', async ({ page }) => {
      await page.goto('/works/new');
      
      // Type in fandom field and check for suggestions
      await page.fill('#fandoms', 'Harry');
      await page.waitForTimeout(1000);
      
      // Should show autocomplete suggestions
      await expect(page.locator('[role="listbox"]')).toBeVisible();
      
      // Select a suggestion
      await page.click('[role="option"]:has-text("Harry Potter")');
      
      // Should add the tag
      await expect(page.locator('.bg-orange-100')).toContainText('Harry Potter');
    });
  });

  test.describe('Work Editing', () => {
    test('should edit work metadata successfully', async ({ page }) => {
      // Create a work first
      const workData = {
        title: 'Original Title',
        summary: 'Original summary',
        content: 'Original content',
        fandoms: ['Original Work'],
        rating: 'General Audiences'
      };

      await utils.createBasicWork(workData);
      await utils.publishWork();
      
      // Get work ID from URL
      const workId = page.url().split('/').pop();
      
      // Navigate to edit page
      await page.goto(`/works/${workId}/edit`);
      
      // Edit work metadata
      await page.fill('[name="title"]', 'Updated Title');
      await page.fill('[name="summary"]', 'Updated summary with more details');
      await page.selectOption('[name="rating"]', 'Teen And Up Audiences');
      
      // Save changes
      await page.click('text=Save Changes');
      await page.waitForSelector('text=Work updated successfully!');
      
      // Navigate back to work page and verify changes
      await page.goto(`/works/${workId}`);
      await expect(page.locator('h1')).toContainText('Updated Title');
      await expect(page.locator('text=Updated summary')).toBeVisible();
      await expect(page.locator('.bg-blue-100')).toContainText('Teen And Up Audiences');
    });

    test('should edit work privacy settings', async ({ page }) => {
      // Create a work first
      const workData = {
        title: 'Privacy Test Work',
        content: 'Content for privacy testing',
        fandoms: ['Original Work']
      };

      await utils.createBasicWork(workData);
      await utils.publishWork();
      
      const workId = page.url().split('/').pop();
      
      // Navigate to edit page
      await page.goto(`/works/${workId}/edit`);
      
      // Change privacy settings
      await page.check('[name="restricted_to_users"]');
      await page.check('[name="is_anonymous"]');
      await page.selectOption('[name="comment_policy"]', 'users_only');
      
      // Save changes
      await page.click('text=Save Changes');
      await page.waitForSelector('text=Work updated successfully!');
      
      // Verify privacy settings were saved
      await expect(page.locator('[name="restricted_to_users"]')).toBeChecked();
      await expect(page.locator('[name="is_anonymous"]')).toBeChecked();
      await expect(page.locator('[name="comment_policy"]')).toHaveValue('users_only');
    });

    test('should add and edit tags', async ({ page }) => {
      // Create a work first
      const workData = {
        title: 'Tag Test Work',
        content: 'Content for tag testing',
        fandoms: ['Original Work']
      };

      await utils.createBasicWork(workData);
      await utils.publishWork();
      
      const workId = page.url().split('/').pop();
      
      // Navigate to edit page
      await page.goto(`/works/${workId}/edit`);
      
      // Add tags using comma-separated input
      await page.fill('[id="characters"]', 'Character One, Character Two');
      await page.fill('[id="relationships"]', 'Character One/Character Two');
      await page.fill('[id="freeform_tags"]', 'Fluff, Hurt/Comfort, First Kiss');
      
      // Save changes
      await page.click('text=Save Changes');
      await page.waitForSelector('text=Work updated successfully!');
      
      // Navigate back to work page and verify tags
      await page.goto(`/works/${workId}`);
      await expect(page.locator('text=Character One')).toBeVisible();
      await expect(page.locator('text=Character One/Character Two')).toBeVisible();
      await expect(page.locator('text=Fluff')).toBeVisible();
    });
  });

  test.describe('Multi-Chapter Management', () => {
    test('should create and manage multiple chapters', async ({ page }) => {
      // Create initial work
      const workData = {
        title: 'Multi-Chapter Test Work',
        content: 'This is chapter 1 content.',
        fandoms: ['Original Work'],
        status: 'posted'
      };

      await utils.createBasicWork(workData);
      await utils.publishWork();
      
      const workId = page.url().split('/').pop();
      
      // Navigate to edit page and chapters tab
      await page.goto(`/works/${workId}/edit`);
      await page.click('text=Chapters');
      
      // Create second chapter
      await page.click('text=Add Chapter');
      await page.fill('[name="title"]', 'Chapter 2: The Adventure Continues');
      await page.fill('[name="content"]', 'This is the content for chapter 2. More exciting things happen.');
      await page.click('text=Create Chapter');
      await page.waitForSelector('text=Chapter created successfully!');
      
      // Verify chapter was created
      await expect(page.locator('text=Chapter 2: The Adventure Continues')).toBeVisible();
      
      // Create third chapter
      await page.click('text=Add Chapter');
      await page.fill('[name="title"]', 'Chapter 3: The Conclusion');
      await page.fill('[name="content"]', 'This is the final chapter where everything is resolved.');
      await page.click('text=Create Chapter');
      await page.waitForSelector('text=Chapter created successfully!');
      
      // Navigate back to work page and verify chapter navigation
      await page.goto(`/works/${workId}`);
      await expect(page.locator('text=Chapters: 3')).toBeVisible();
      
      // Test chapter navigation
      const chapterSelect = page.locator('select').first();
      await expect(chapterSelect).toBeVisible();
      
      // Navigate to chapter 2
      await chapterSelect.selectOption('1'); // 0-indexed
      await expect(page.locator('h2')).toContainText('Chapter 2: The Adventure Continues');
      await expect(page.locator('.prose')).toContainText('More exciting things happen');
      
      // Navigate to chapter 3
      await chapterSelect.selectOption('2');
      await expect(page.locator('h2')).toContainText('Chapter 3: The Conclusion');
      await expect(page.locator('.prose')).toContainText('everything is resolved');
    });

    test('should edit existing chapters', async ({ page }) => {
      // Create work with initial chapter
      const workData = {
        title: 'Chapter Edit Test',
        content: 'Original chapter content.',
        fandoms: ['Original Work']
      };

      await utils.createBasicWork(workData);
      await utils.publishWork();
      
      const workId = page.url().split('/').pop();
      
      // Navigate to edit page and chapters tab
      await page.goto(`/works/${workId}/edit`);
      await page.click('text=Chapters');
      
      // Select and edit the first chapter
      await page.click('.bg-orange-50'); // First chapter should be selected by default
      
      // Update chapter content
      await page.fill('[name="title"]', 'Updated Chapter Title');
      await page.fill('[name="content"]', 'This is the updated content for chapter 1.');
      await page.selectOption('[name="status"]', 'posted');
      
      // Save changes
      await page.click('text=Save Chapter');
      await page.waitForSelector('text=Chapter updated successfully!');
      
      // Navigate back to work page and verify changes
      await page.goto(`/works/${workId}`);
      await expect(page.locator('h2')).toContainText('Updated Chapter Title');
      await expect(page.locator('.prose')).toContainText('updated content for chapter 1');
    });

    test('should delete chapters', async ({ page }) => {
      // Create work with multiple chapters
      const workData = {
        title: 'Chapter Delete Test',
        content: 'Chapter 1 content.',
        fandoms: ['Original Work']
      };

      await utils.createBasicWork(workData);
      await utils.publishWork();
      
      const workId = page.url().split('/').pop();
      
      // Add a second chapter
      await page.goto(`/works/${workId}/edit`);
      await page.click('text=Chapters');
      await page.click('text=Add Chapter');
      await page.fill('[name="title"]', 'Chapter to Delete');
      await page.fill('[name="content"]', 'This chapter will be deleted.');
      await page.click('text=Create Chapter');
      await page.waitForSelector('text=Chapter created successfully!');
      
      // Delete the second chapter
      await page.click('text=Chapter to Delete');
      
      // Handle delete confirmation
      page.on('dialog', dialog => dialog.accept());
      await page.click('button[title="Delete chapter"]');
      await page.waitForSelector('text=Chapter deleted successfully!');
      
      // Verify chapter was deleted
      await expect(page.locator('text=Chapter to Delete')).not.toBeVisible();
    });
  });

  test.describe('Draft to Published Workflow', () => {
    test('should promote draft to published', async ({ page }) => {
      // Create a draft work
      const workData = {
        title: 'Draft to Published Test',
        summary: 'Testing draft promotion',
        content: 'This work starts as a draft and will be published.',
        fandoms: ['Original Work'],
        status: 'draft'
      };

      await utils.createBasicWork(workData);
      await utils.saveDraft();
      
      const workId = page.url().split('/').pop();
      
      // Verify it's a draft
      await expect(page.locator('.bg-yellow-100')).toContainText('Draft');
      
      // Navigate to edit page
      await page.goto(`/works/${workId}/edit`);
      
      // Change status to published
      await page.selectOption('[name="status"]', 'posted');
      await page.click('text=Save Changes');
      await page.waitForSelector('text=Work updated successfully!');
      
      // Navigate back to work page and verify it's published
      await page.goto(`/works/${workId}`);
      await expect(page.locator('.bg-green-100')).toContainText('Published');
      
      // Verify it appears in dashboard as published
      await utils.goToDashboard();
      const workRow = page.locator('.bg-white', { hasText: workData.title });
      await expect(workRow.locator('.bg-green-100')).toContainText('Published');
    });

    test('should revert published work to draft', async ({ page }) => {
      // Create a published work
      const workData = {
        title: 'Published to Draft Test',
        content: 'This work will be reverted to draft.',
        fandoms: ['Original Work'],
        status: 'posted'
      };

      await utils.createBasicWork(workData);
      await utils.publishWork();
      
      const workId = page.url().split('/').pop();
      
      // Verify it's published
      await expect(page.locator('.bg-green-100')).toContainText('Published');
      
      // Navigate to edit page
      await page.goto(`/works/${workId}/edit`);
      
      // Change status back to draft
      await page.selectOption('[name="status"]', 'draft');
      await page.click('text=Save Changes');
      await page.waitForSelector('text=Work updated successfully!');
      
      // Navigate back to work page and verify it's a draft
      await page.goto(`/works/${workId}`);
      await expect(page.locator('.bg-yellow-100')).toContainText('Draft');
    });

    test('should handle chapter-level draft/published status', async ({ page }) => {
      // Create work with published first chapter
      const workData = {
        title: 'Chapter Status Test',
        content: 'Published chapter content.',
        fandoms: ['Original Work'],
        status: 'posted'
      };

      await utils.createBasicWork(workData);
      await utils.publishWork();
      
      const workId = page.url().split('/').pop();
      
      // Add a draft chapter
      await page.goto(`/works/${workId}/edit`);
      await page.click('text=Chapters');
      await page.click('text=Add Chapter');
      await page.fill('[name="title"]', 'Draft Chapter');
      await page.fill('[name="content"]', 'This chapter is still a draft.');
      // Status should default to draft
      await page.click('text=Create Chapter');
      await page.waitForSelector('text=Chapter created successfully!');
      
      // Verify chapter statuses in the chapter list
      await expect(page.locator('text=Published').first()).toBeVisible(); // First chapter
      await expect(page.locator('text=Draft').last()).toBeVisible(); // Second chapter
      
      // Publish the draft chapter
      await page.click('text=Draft Chapter');
      await page.selectOption('[name="status"]', 'posted');
      await page.click('text=Save Chapter');
      await page.waitForSelector('text=Chapter updated successfully!');
      
      // Verify both chapters are now published
      const publishedBadges = page.locator('text=Published');
      await expect(publishedBadges).toHaveCount(2);
    });
  });

  test.describe('Dashboard Integration', () => {
    test('should show works in dashboard with correct metadata', async ({ page }) => {
      // Create multiple works with different statuses
      const works = [
        {
          title: 'Published Work',
          content: 'Content for published work.',
          fandoms: ['Fandom A'],
          status: 'posted'
        },
        {
          title: 'Draft Work',
          content: 'Content for draft work.',
          fandoms: ['Fandom B'],
          status: 'draft'
        }
      ];

      for (const workData of works) {
        await utils.createBasicWork(workData);
        if (workData.status === 'posted') {
          await utils.publishWork();
        } else {
          await utils.saveDraft();
        }
      }
      
      // Navigate to dashboard
      await utils.goToDashboard();
      
      // Verify both works appear with correct status
      await expect(page.locator('text=Published Work')).toBeVisible();
      await expect(page.locator('text=Draft Work')).toBeVisible();
      
      const publishedRow = page.locator('.bg-white', { hasText: 'Published Work' });
      await expect(publishedRow.locator('.bg-green-100')).toContainText('Published');
      
      const draftRow = page.locator('.bg-white', { hasText: 'Draft Work' });
      await expect(draftRow.locator('.bg-yellow-100')).toContainText('Draft');
    });

    test('should allow deleting works from dashboard', async ({ page }) => {
      // Create a work to delete
      const workData = {
        title: 'Work to Delete',
        content: 'This work will be deleted.',
        fandoms: ['Original Work']
      };

      await utils.createBasicWork(workData);
      await utils.saveDraft();
      
      // Navigate to dashboard
      await utils.goToDashboard();
      
      // Delete the work
      const workRow = page.locator('.bg-white', { hasText: workData.title });
      
      // Handle delete confirmation
      page.on('dialog', dialog => dialog.accept());
      await workRow.locator('text=Delete').click();
      
      // Verify work was deleted
      await expect(page.locator(`text=${workData.title}`)).not.toBeVisible();
    });

    test('should navigate to edit from dashboard', async ({ page }) => {
      // Create a work
      const workData = {
        title: 'Edit from Dashboard Test',
        content: 'Content to edit.',
        fandoms: ['Original Work']
      };

      await utils.createBasicWork(workData);
      await utils.saveDraft();
      
      // Navigate to dashboard and click edit
      await utils.goToDashboard();
      const workRow = page.locator('.bg-white', { hasText: workData.title });
      await workRow.locator('text=Edit').click();
      
      // Verify we're on the edit page
      await expect(page).toHaveURL(/\/works\/[a-f0-9-]+\/edit$/);
      await expect(page.locator('h1')).toContainText('Edit Work');
      await expect(page.locator('[name="title"]')).toHaveValue(workData.title);
    });
  });
});