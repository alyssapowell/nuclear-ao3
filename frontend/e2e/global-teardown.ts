import { FullConfig } from '@playwright/test'

async function globalTeardown(config: FullConfig) {
  // Accessibility testing teardown
  console.log('🧹 Cleaning up accessibility testing environment...')
  
  // Nothing to clean up - the setup only validated server availability
  // Add cleanup tasks here if needed in the future
  
  console.log('✅ Accessibility testing environment cleaned up!')
}

export default globalTeardown
