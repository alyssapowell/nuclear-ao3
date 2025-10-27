import { test, expect } from '@playwright/test'

test.describe('Gift System', () => {
  let authToken: string
  let workId: string
  let authorUserId: string

  test.beforeAll(async ({ request }) => {
    // Create a test user for authoring works
    const registerResponse = await request.post('http://localhost:8080/api/v1/auth/register', {
      data: {
        username: `giftauthor_${Date.now()}`,
        email: `giftauthor_${Date.now()}@example.com`,
        password: 'testpass123'
      }
    })
    
    expect(registerResponse.ok()).toBeTruthy()
    const authData = await registerResponse.json()
    authToken = authData.access_token
    authorUserId = authData.user.id

    // Create a test work
    const workResponse = await request.post('http://localhost:8080/api/v1/works', {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json'
      },
      data: {
        title: 'Gift Test Work',
        summary: 'A work for testing the gift system',
        rating: 'General Audiences',
        language: 'en',
        fandoms: ['Test Fandom'],
        chapters: [{
          title: 'Chapter 1',
          content: 'This is test content for gift testing.'
        }]
      }
    })
    
    expect(workResponse.ok()).toBeTruthy()
    const workData = await workResponse.json()
    workId = workData.work.id
  })

  test('should show gift button for work author', async ({ page }) => {
    // Set authentication
    await page.addInitScript((token) => {
      localStorage.setItem('auth_token', token)
    }, authToken)

    // Navigate to work page
    await page.goto(`http://localhost:3001/works/${workId}`)
    
    // Wait for page to load
    await expect(page.locator('h1')).toContainText('Gift Test Work')
    
    // Check that gift button is visible for author
    await expect(page.locator('button:has-text("Gift")')).toBeVisible()
  })

  test('should create a gift successfully', async ({ page }) => {
    // Set authentication
    await page.addInitScript((token) => {
      localStorage.setItem('auth_token', token)
    }, authToken)

    // Navigate to work page
    await page.goto(`http://localhost:3001/works/${workId}`)
    
    // Click gift button
    await page.click('button:has-text("Gift")')
    
    // Fill in recipient name
    await page.fill('input[placeholder*="recipient"]', 'Test Recipient')
    
    // Submit gift form
    await page.click('button:has-text("Create Gift")')
    
    // Check that gift appears in the work
    await expect(page.locator('text=Gifted to:')).toBeVisible()
    await expect(page.locator('text=Test Recipient')).toBeVisible()
  })

  test('should not show gift button for non-authors', async ({ page, request }) => {
    // Create a different user (not the author)
    const readerResponse = await request.post('http://localhost:8080/api/v1/auth/register', {
      data: {
        username: `reader_${Date.now()}`,
        email: `reader_${Date.now()}@example.com`,
        password: 'testpass123'
      }
    })
    
    expect(readerResponse.ok()).toBeTruthy()
    const readerData = await readerResponse.json()
    const readerToken = readerData.access_token

    // Set authentication as reader
    await page.addInitScript((token) => {
      localStorage.setItem('auth_token', token)
    }, readerToken)

    // Navigate to work page
    await page.goto(`http://localhost:3001/works/${workId}`)
    
    // Wait for page to load
    await expect(page.locator('h1')).toContainText('Gift Test Work')
    
    // Check that gift button is NOT visible for non-author
    await expect(page.locator('button:has-text("Gift")')).not.toBeVisible()
  })

  test('should handle multiple gifts', async ({ page }) => {
    // Set authentication
    await page.addInitScript((token) => {
      localStorage.setItem('auth_token', token)
    }, authToken)

    // Navigate to work page
    await page.goto(`http://localhost:3001/works/${workId}`)
    
    // Create first gift
    await page.click('button:has-text("Gift")')
    await page.fill('input[placeholder*="recipient"]', 'First Recipient')
    await page.click('button:has-text("Create Gift")')
    
    // Wait for first gift to appear
    await expect(page.locator('text=First Recipient')).toBeVisible()
    
    // Create second gift
    await page.click('button:has-text("Gift")')
    await page.fill('input[placeholder*="recipient"]', 'Second Recipient')
    await page.click('button:has-text("Create Gift")')
    
    // Check that both gifts appear
    await expect(page.locator('text=First Recipient')).toBeVisible()
    await expect(page.locator('text=Second Recipient')).toBeVisible()
    await expect(page.locator('text=Gifted to:')).toBeVisible()
  })

  test('should validate gift form input', async ({ page }) => {
    // Set authentication
    await page.addInitScript((token) => {
      localStorage.setItem('auth_token', token)
    }, authToken)

    // Navigate to work page
    await page.goto(`http://localhost:3001/works/${workId}`)
    
    // Click gift button
    await page.click('button:has-text("Gift")')
    
    // Try to submit without recipient name
    await page.click('button:has-text("Create Gift")')
    
    // Check for validation error
    await expect(page.locator('text=Recipient name is required')).toBeVisible()
  })

  test('should allow gift form cancellation', async ({ page }) => {
    // Set authentication
    await page.addInitScript((token) => {
      localStorage.setItem('auth_token', token)
    }, authToken)

    // Navigate to work page
    await page.goto(`http://localhost:3001/works/${workId}`)
    
    // Click gift button
    await page.click('button:has-text("Gift")')
    
    // Check that dialog is open
    await expect(page.locator('text=Gift this Work')).toBeVisible()
    
    // Click cancel
    await page.click('button:has-text("Cancel")')
    
    // Check that dialog is closed
    await expect(page.locator('text=Gift this Work')).not.toBeVisible()
  })
})