import { test, expect } from '@playwright/test';

test.describe('Work Functionality Tests', () => {
  let authToken: string;
  let userId: string;
  
  test.beforeEach(async ({ page }) => {
    // Register and login a test user
    const timestamp = Date.now();
    const testUser = {
      username: `testuser_${timestamp}`,
      email: `test_${timestamp}@example.com`,
      password: 'testpassword123'
    };

    // Register user
    const registerResponse = await page.request.post('http://localhost:8080/api/v1/auth/register', {
      data: testUser
    });
    expect(registerResponse.ok()).toBeTruthy();

    // Login user
    const loginResponse = await page.request.post('http://localhost:8080/api/v1/auth/login', {
      data: {
        username: testUser.username,
        password: testUser.password
      }
    });
    expect(loginResponse.ok()).toBeTruthy();
    
    const loginData = await loginResponse.json();
    authToken = loginData.token;
    userId = loginData.user.id;
    
    // Set auth token in browser
    await page.goto('http://localhost:3001');
    await page.evaluate((token) => {
      localStorage.setItem('auth_token', token);
      localStorage.setItem('token', token);
    }, authToken);
  });

  test('should create and view a multi-chapter work', async ({ page }) => {
    await page.goto('http://localhost:3001/works/new');
    
    // Wait for the page to load
    await page.waitForSelector('form');
    
    // Fill out work creation form
    await page.fill('[name="title"]', 'Test Multi-Chapter Work');
    await page.fill('[name="summary"]', 'This is a test work with multiple chapters.');
    await page.fill('[name="notes"]', 'Author notes for the work.');
    
    // Set rating
    await page.selectOption('[name="rating"]', 'General Audiences');
    
    // Add fandoms (assuming there's a fandom input)
    await page.fill('[data-testid="fandom-input"]', 'Test Fandom');
    await page.press('[data-testid="fandom-input"]', 'Enter');
    
    // Add characters
    await page.fill('[data-testid="character-input"]', 'Test Character 1');
    await page.press('[data-testid="character-input"]', 'Enter');
    await page.fill('[data-testid="character-input"]', 'Test Character 2');
    await page.press('[data-testid="character-input"]', 'Enter');
    
    // Add relationships  
    await page.fill('[data-testid="relationship-input"]', 'Test Character 1/Test Character 2');
    await page.press('[data-testid="relationship-input"]', 'Enter');
    
    // Add additional tags
    await page.fill('[data-testid="freeform-input"]', 'Slow Burn');
    await page.press('[data-testid="freeform-input"]', 'Enter');
    await page.fill('[data-testid="freeform-input"]', 'Hurt/Comfort');
    await page.press('[data-testid="freeform-input"]', 'Enter');
    
    // Set max chapters
    await page.fill('[name="max_chapters"]', '3');
    
    // Fill first chapter
    await page.fill('[name="chapter_title"]', 'Chapter 1: The Beginning');
    await page.fill('[name="chapter_summary"]', 'The first chapter summary.');
    await page.fill('[name="chapter_notes"]', 'Chapter 1 author notes.');
    await page.fill('[name="chapter_content"]', 'This is the content of the first chapter. It should be interesting and engaging for readers.');
    await page.fill('[name="chapter_end_notes"]', 'End notes for chapter 1.');
    
    // Submit the form
    await page.click('button[type="submit"]');
    
    // Wait for redirect to work page
    await page.waitForURL(/\/works\/[a-f0-9-]{36}$/);
    
    // Verify work is displayed correctly
    await expect(page.locator('h1')).toContainText('Test Multi-Chapter Work');
    await expect(page.locator('text=This is a test work with multiple chapters.')).toBeVisible();
    await expect(page.locator('text=Author notes for the work.')).toBeVisible();
    
    // Verify author is shown
    await expect(page.locator('text=by')).toBeVisible();
    await expect(page.locator(`text=testuser_`)).toBeVisible();
    
    // Verify tags are displayed
    await expect(page.locator('text=Test Fandom')).toBeVisible();
    await expect(page.locator('text=Test Character 1')).toBeVisible();
    await expect(page.locator('text=Test Character 2')).toBeVisible();
    await expect(page.locator('text=Test Character 1/Test Character 2')).toBeVisible();
    await expect(page.locator('text=Slow Burn')).toBeVisible();
    await expect(page.locator('text=Hurt/Comfort')).toBeVisible();
    
    // Verify chapter content
    await expect(page.locator('text=Chapter 1: The Beginning')).toBeVisible();
    await expect(page.locator('text=The first chapter summary.')).toBeVisible();
    await expect(page.locator('text=Chapter 1 author notes.')).toBeVisible();
    await expect(page.locator('text=This is the content of the first chapter')).toBeVisible();
    await expect(page.locator('text=End notes for chapter 1.')).toBeVisible();
    
    // Verify chapter count
    await expect(page.locator('text=1/3')).toBeVisible();
    
    // Store work URL for adding more chapters
    const workUrl = page.url();
    const workId = workUrl.split('/').pop();
    
    // Add second chapter
    await page.goto(`http://localhost:3001/works/${workId}/chapters/new`);
    
    await page.fill('[name="title"]', 'Chapter 2: The Plot Thickens');
    await page.fill('[name="summary"]', 'The second chapter summary.');
    await page.fill('[name="notes"]', 'Chapter 2 author notes.');
    await page.fill('[name="content"]', 'This is the content of the second chapter. The plot continues to develop.');
    await page.fill('[name="end_notes"]', 'End notes for chapter 2.');
    
    await page.click('button[type="submit"]');
    
    // Return to work page
    await page.goto(workUrl);
    
    // Verify updated chapter count
    await expect(page.locator('text=2/3')).toBeVisible();
    
    // Verify chapter navigation is available
    await expect(page.locator('select')).toBeVisible();
    
    // Switch to chapter 2
    await page.selectOption('select', '2');
    await expect(page.locator('text=Chapter 2: The Plot Thickens')).toBeVisible();
    await expect(page.locator('text=This is the content of the second chapter')).toBeVisible();
    
    // Switch back to chapter 1
    await page.selectOption('select', '1');
    await expect(page.locator('text=Chapter 1: The Beginning')).toBeVisible();
  });

  test('should allow non-author to give kudos and comments', async ({ page }) => {
    // First, create a work as the first user
    const workResponse = await page.request.post('http://localhost:8080/api/v1/works', {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json',
      },
      data: {
        title: 'Test Work for Kudos',
        summary: 'A work to test kudos and comments',
        language: 'en',
        rating: 'general',
        fandoms: ['Test Fandom'],
        chapter_content: 'This is the content of the test work.'
      }
    });
    
    expect(workResponse.ok()).toBeTruthy();
    const workData = await workResponse.json();
    const workId = workData.work?.id || workData.id;
    
    // Logout current user and create a second user
    await page.evaluate(() => {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('token');
    });
    
    // Register and login second user
    const timestamp2 = Date.now() + 1000;
    const testUser2 = {
      username: `testuser2_${timestamp2}`,
      email: `test2_${timestamp2}@example.com`,
      password: 'testpassword123'
    };

    const registerResponse2 = await page.request.post('http://localhost:8080/api/v1/auth/register', {
      data: testUser2
    });
    expect(registerResponse2.ok()).toBeTruthy();

    const loginResponse2 = await page.request.post('http://localhost:8080/api/v1/auth/login', {
      data: {
        username: testUser2.username,
        password: testUser2.password
      }
    });
    expect(loginResponse2.ok()).toBeTruthy();
    
    const loginData2 = await loginResponse2.json();
    const authToken2 = loginData2.token;
    
    // Set second user's auth token
    await page.evaluate((token) => {
      localStorage.setItem('auth_token', token);
      localStorage.setItem('token', token);
    }, authToken2);
    
    // Visit the work page
    await page.goto(`http://localhost:3001/works/${workId}`);
    
    // Verify work loads
    await expect(page.locator('h1')).toContainText('Test Work for Kudos');
    
    // Give kudos
    const kudosButton = page.locator('button:has-text("Give Kudos")');
    await expect(kudosButton).toBeVisible();
    await kudosButton.click();
    
    // Verify kudos was given
    await expect(page.locator('button:has-text("Kudos Given")')).toBeVisible();
    await expect(page.locator('text=(1)')).toBeVisible();
    
    // Add a comment
    const commentTextarea = page.locator('textarea[placeholder*="comment"]');
    await expect(commentTextarea).toBeVisible();
    await commentTextarea.fill('This is a great work! Thanks for sharing.');
    
    const submitCommentButton = page.locator('button:has-text("Post Comment")');
    await submitCommentButton.click();
    
    // Verify comment appears
    await expect(page.locator('text=This is a great work! Thanks for sharing.')).toBeVisible();
    await expect(page.locator(`text=${testUser2.username}`)).toBeVisible();
  });

  test('should handle work viewing permissions correctly', async ({ page }) => {
    // Test that anonymous users can view public works
    await page.evaluate(() => {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('token');
    });
    
    // Visit a public work (using one we know exists)
    await page.goto('http://localhost:3001/works/251989f2-0a32-4d5d-96f0-0c2e2c1487ca');
    
    // Should be able to view the work
    await expect(page.locator('h1')).toContainText('Statistics Test');
    await expect(page.locator('text=Chapter content goes here.')).toBeVisible();
    
    // But kudos button should prompt for login or be disabled
    const kudosButton = page.locator('button:has-text("Give Kudos")');
    if (await kudosButton.isVisible()) {
      // Should either be disabled or clicking should show login prompt
      await kudosButton.click();
      // Check for either disabled state or error message
      const hasError = await page.locator('text=authorization').isVisible().catch(() => false);
      const isDisabled = await kudosButton.isDisabled().catch(() => false);
      expect(hasError || isDisabled).toBeTruthy();
    }
  });
});