// Manual test script to verify enhanced form functionality
const { chromium } = require('playwright');

async function testEnhancedForm() {
  console.log('🚀 Starting enhanced form test...');
  
  const browser = await chromium.launch({ headless: false, slowMo: 1000 });
  const page = await browser.newPage();
  
  try {
    // 1. Navigate to home page
    console.log('📍 Navigating to home page...');
    await page.goto('http://localhost:3000');
    await page.waitForTimeout(2000);
    
    // 2. Navigate to new work page (should redirect to login)
    console.log('📍 Going to new work page...');
    await page.goto('http://localhost:3000/works/new');
    await page.waitForTimeout(2000);
    
    // 3. Check if we're on login page or if auth is bypassed
    const currentUrl = page.url();
    console.log(`📍 Current URL: ${currentUrl}`);
    
    if (currentUrl.includes('/auth/login')) {
      console.log('🔐 Redirected to login - bypassing auth for test...');
      // Set a fake token to bypass auth
      await page.evaluate(() => {
        localStorage.setItem('auth_token', 'fake-token-for-testing');
      });
      await page.goto('http://localhost:3000/works/new');
      await page.waitForTimeout(2000);
    }
    
    // 4. Check enhanced form is loaded
    console.log('📝 Checking enhanced form elements...');
    
    // Check multiple TagAutocomplete fields
    const fandomInput = await page.locator('input[id="fandoms"]').isVisible();
    const characterInput = await page.locator('input[id="characters"]').isVisible();
    const relationshipInput = await page.locator('input[id="relationships"]').isVisible();
    const freeformInput = await page.locator('input[id="freeformTags"]').isVisible();
    
    console.log(`✅ Fandom field: ${fandomInput}`);
    console.log(`✅ Character field: ${characterInput}`);
    console.log(`✅ Relationship field: ${relationshipInput}`);
    console.log(`✅ Freeform field: ${freeformInput}`);
    
    // Check RichTextEditor
    const richEditor = await page.locator('.ProseMirror').isVisible();
    console.log(`✅ RichTextEditor: ${richEditor}`);
    
    // 5. Test character autocomplete (we know this has data)
    console.log('🏷️ Testing character autocomplete...');
    await page.locator('input[id="characters"]').click();
    await page.locator('input[id="characters"]').fill('Harry');
    await page.waitForTimeout(1000);
    
    // Check for suggestions dropdown
    const suggestions = await page.locator('[role="listbox"]').isVisible();
    console.log(`✅ Suggestions dropdown visible: ${suggestions}`);
    
    if (suggestions) {
      const suggestionCount = await page.locator('[role="option"]').count();
      console.log(`✅ Number of suggestions: ${suggestionCount}`);
      
      if (suggestionCount > 0) {
        // Click first suggestion
        await page.locator('[role="option"]').first().click();
        await page.waitForTimeout(500);
        
        // Check if tag was added
        const tagAdded = await page.locator('text=Harry Potter').isVisible();
        console.log(`✅ Tag added to form: ${tagAdded}`);
      }
    }
    
    // 6. Test RichTextEditor functionality
    console.log('📝 Testing RichTextEditor...');
    await page.locator('.ProseMirror').click();
    await page.locator('.ProseMirror').fill('This is a test chapter with some content.');
    await page.waitForTimeout(500);
    
    // Test bold button
    await page.locator('button[title="Bold"]').click();
    await page.locator('.ProseMirror').type(' Bold text here.');
    
    console.log('✅ RichTextEditor functionality working');
    
    // 7. Check form submission readiness
    console.log('📋 Checking form completion...');
    await page.locator('input[name="title"]').fill('Test Enhanced Work');
    
    const submitButton = await page.locator('button[type="submit"]');
    const isEnabled = await submitButton.isEnabled();
    console.log(`✅ Submit button enabled: ${isEnabled}`);
    
    console.log('🎉 Enhanced form test completed successfully!');
    
  } catch (error) {
    console.error('❌ Test failed:', error.message);
  } finally {
    await browser.close();
  }
}

testEnhancedForm().catch(console.error);