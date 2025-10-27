import { chromium, FullConfig } from '@playwright/test'

async function globalSetup(config: FullConfig) {
  // Accessibility testing setup
  console.log('🎯 Setting up accessibility testing environment...')
  
  // Start browser for pre-test validation
  const browser = await chromium.launch()
  const page = await browser.newPage()
  
  // Validate that the dev server is accessible
  try {
    await page.goto(config.webServer?.url || 'http://localhost:3001', { 
      waitUntil: 'networkidle',
      timeout: 30000 
    })
    console.log('✅ Dev server is accessible')
  } catch (error) {
    console.error('❌ Dev server is not accessible:', error)
    await browser.close()
    throw error
  }
  
  // Check that search functionality is available
  try {
    const searchInput = page.getByRole('combobox', { name: /search/i })
    await searchInput.waitFor({ timeout: 10000 })
    console.log('✅ Search component is available')
  } catch (error) {
    console.log('⚠️  Search component not found, tests may be limited')
  }
  
  await browser.close()
  console.log('🚀 Accessibility testing environment ready!')
}

export default globalSetup