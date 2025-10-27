import { test, expect } from '@playwright/test';

test.describe('Direct API Test', () => {
  test('should create work via direct API call', async ({ page }) => {
    // Get a fresh token first
    const loginResponse = await page.request.post('http://localhost:8080/api/v1/auth/login', {
      data: {
        email: 'admin@nuclear-ao3.com',
        password: 'adminpass123'
      }
    });
    
    expect(loginResponse.ok()).toBeTruthy();
    const loginData = await loginResponse.json();
    const token = loginData.access_token;
    
    console.log('Got token:', token.substring(0, 50) + '...');
    
    // Create work via API
    const workResponse = await page.request.post('http://localhost:8080/api/v1/works', {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      data: {
        title: 'Playwright Direct API Test',
        summary: 'Testing direct API call from Playwright',
        fandoms: ['Playwright Testing'],
        rating: 'General Audiences',
        category: ['Gen'],
        warnings: ['No Archive Warnings Apply'],
        characters: ['Test Character'],
        relationships: [],
        freeform_tags: ['playwright', 'api-test', 'direct'],
        language: 'en',
        max_chapters: 1
      }
    });
    
    console.log('Work creation response status:', workResponse.status());
    const responseText = await workResponse.text();
    console.log('Response:', responseText.substring(0, 200));
    
    expect(workResponse.status()).toBe(201);
    
    const workData = await workResponse.json();
    expect(workData.work).toBeDefined();
    expect(workData.work.title).toBe('Playwright Direct API Test');
    
    console.log('SUCCESS: Work created with ID:', workData.work.id);
  });
});