const { chromium } = require('playwright');

async function debugFrontend() {
  console.log('🚀 Starting frontend debugging...');
  
  const browser = await chromium.launch({ headless: false });
  const page = await browser.newPage();
  
  // Capture console logs
  page.on('console', msg => {
    const type = msg.type();
    const args = msg.args();
    if (type === 'log' || type === 'warn' || type === 'error') {
      console.log(`🖥️  Console ${type}: ${msg.text()}`);
    }
  });
  
  console.log('📱 Navigating to homepage...');
  await page.goto('http://localhost:3001');
  
  // Test 1: TagAutocomplete focus behavior
  console.log('\n=== TESTING TAGAUTOCOMPLETE FOCUS ===');
  
  const additionalTagsInput = page.locator('input[placeholder*="Add additional tag"]').first();
  
  try {
    await additionalTagsInput.waitFor({ timeout: 5000 });
    console.log('✅ Found additional tags input field');
    
    await additionalTagsInput.focus();
    console.log('✅ Focused on input field');
    
    let isFocused = await additionalTagsInput.evaluate(el => el === document.activeElement);
    console.log(`📍 Input focused after .focus(): ${isFocused}`);
    
    console.log('⌨️  Typing "h"...');
    await additionalTagsInput.type('h', { delay: 100 });
    
    await page.waitForTimeout(500); // Wait for render profiling logs
    
    isFocused = await additionalTagsInput.evaluate(el => el === document.activeElement);
    console.log(`📍 Input focused after typing "h": ${isFocused}`);
    
    const value = await additionalTagsInput.inputValue();
    console.log(`📝 Input value: "${value}"`);
    
    console.log('⌨️  Typing "e"...');
    await additionalTagsInput.type('e', { delay: 100 });
    
    await page.waitForTimeout(500); // Wait for render profiling logs
    
    isFocused = await additionalTagsInput.evaluate(el => el === document.activeElement);
    console.log(`📍 Input focused after typing "he": ${isFocused}`);
    
    const value2 = await additionalTagsInput.inputValue();
    console.log(`📝 Input value: "${value2}"`);
    
    console.log('⌨️  Typing "l"...');
    await additionalTagsInput.type('l', { delay: 100 });
    
    await page.waitForTimeout(500); // Wait for render profiling logs
    
    isFocused = await additionalTagsInput.evaluate(el => el === document.activeElement);
    console.log(`📍 Input focused after typing "hel": ${isFocused}`);
    
    const value3 = await additionalTagsInput.inputValue();
    console.log(`📝 Input value: "${value3}"`);
    
    const suggestions = page.locator('[role="listbox"]');
    const hasSuggestions = await suggestions.isVisible().catch(() => false);
    console.log(`🔍 Autocomplete suggestions visible: ${hasSuggestions}`);
    
    if (hasSuggestions) {
      const suggestionCount = await suggestions.locator('[role="option"]').count();
      console.log(`📋 Number of suggestions: ${suggestionCount}`);
    }
    
    await page.screenshot({ path: 'tagautocomplete-debug.png', fullPage: true });
    console.log('📷 Screenshot saved: tagautocomplete-debug.png');
    
  } catch (error) {
    console.log(`❌ Error testing TagAutocomplete: ${error.message}`);
    await page.screenshot({ path: 'tagautocomplete-error.png', fullPage: true });
  }
  
  // Test 2: Network requests during work creation
  console.log('\n=== TESTING WORK CREATION NETWORK REQUESTS ===');
  
  page.on('request', request => {
    if (request.url().includes('/api/v1/works')) {
      console.log(`🌐 REQUEST: ${request.method()} ${request.url()}`);
      const headers = request.headers();
      if (headers.authorization) {
        console.log(`🔑 Authorization: ${headers.authorization.substring(0, 20)}...`);
      }
      if (request.method() === 'POST') {
        try {
          console.log(`📤 POST data: ${request.postData()}`);
        } catch (e) {
          console.log('📤 POST data: (could not read)');
        }
      }
    }
  });
  
  page.on('response', response => {
    if (response.url().includes('/api/v1/works')) {
      console.log(`🌐 RESPONSE: ${response.status()} ${response.url()}`);
      response.text().then(body => {
        console.log(`📥 Response body: ${body.substring(0, 200)}...`);
      }).catch(() => {
        console.log('📥 Response body: (could not read)');
      });
    }
  });
  
  // Try to find work creation page or form
  try {
    // Look for navigation to works/new or create work button
    const createButton = page.locator('text=Create Work').or(page.locator('text=New Work')).first();
    const hasCreateButton = await createButton.isVisible({ timeout: 2000 }).catch(() => false);
    
    if (hasCreateButton) {
      console.log('✅ Found create work button');
      await createButton.click();
    } else {
      console.log('🔍 Trying to navigate to /works/new directly...');
      await page.goto('http://localhost:3001/works/new');
    }
    
    await page.waitForTimeout(1000);
    
    const titleInput = page.locator('input[name="title"]').or(page.locator('input[placeholder*="title" i]')).first();
    const hasTitleInput = await titleInput.isVisible({ timeout: 3000 }).catch(() => false);
    
    if (hasTitleInput) {
      console.log('✅ Found work creation form');
      
      await titleInput.fill('Test Work from Debug Script');
      console.log('✅ Filled title field');
      
      const summaryInput = page.locator('textarea[name="summary"]').or(page.locator('textarea[placeholder*="summary" i]')).first();
      const hasSummary = await summaryInput.isVisible().catch(() => false);
      
      if (hasSummary) {
        await summaryInput.fill('This is a test work created by the debug script.');
        console.log('✅ Filled summary field');
      }
      
      const submitButton = page.locator('button[type="submit"]').or(page.locator('text=Create Work')).first();
      const hasSubmitButton = await submitButton.isVisible().catch(() => false);
      
      if (hasSubmitButton) {
        console.log('🚀 Attempting to submit work...');
        await submitButton.click();
        
        // Wait for network activity
        await page.waitForTimeout(3000);
        
        const currentUrl = page.url();
        console.log(`📍 URL after submission: ${currentUrl}`);
        
      } else {
        console.log('❌ No submit button found');
      }
    } else {
      console.log('❌ No work creation form found');
      
      // Check if login is required
      const loginButton = page.locator('text=Login').or(page.locator('text=Sign In')).first();
      const needsLogin = await loginButton.isVisible().catch(() => false);
      
      if (needsLogin) {
        console.log('🔒 Login appears to be required');
        await page.screenshot({ path: 'login-required.png', fullPage: true });
      }
    }
    
    await page.screenshot({ path: 'work-creation-debug.png', fullPage: true });
    console.log('📷 Screenshot saved: work-creation-debug.png');
    
  } catch (error) {
    console.log(`❌ Error testing work creation: ${error.message}`);
    await page.screenshot({ path: 'work-creation-error.png', fullPage: true });
  }
  
  console.log('\n🏁 Frontend debugging complete!');
  console.log('Check the screenshots for visual inspection.');
  
  await browser.close();
}

debugFrontend().catch(console.error);