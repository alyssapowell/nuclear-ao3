import { test, expect } from '@playwright/test'

test.describe('Search Accessibility & E2E Workflows', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to search page
    await page.goto('/search')
  })

  test('complete search workflow with keyboard navigation', async ({ page }) => {
    // Test initial focus and accessibility
    await page.keyboard.press('Tab')
    const searchInput = page.getByRole('combobox', { name: /search for works/i })
    await expect(searchInput).toBeFocused()
    await expect(searchInput).toHaveAttribute('aria-expanded', 'false')
    await expect(searchInput).toHaveAttribute('aria-autocomplete', 'list')

    // Test search input and loading states
    await page.type('[role="combobox"]', 'harry potter')
    
    // Wait for loading state
    await expect(page.getByRole('img', { name: 'Loading suggestions' })).toBeVisible()
    await expect(page.getByRole('status')).toContainText('Searching for suggestions...')

    // Wait for suggestions to appear
    await expect(page.getByRole('listbox')).toBeVisible({ timeout: 5000 })
    await expect(searchInput).toHaveAttribute('aria-expanded', 'true')

    // Test suggestion navigation with keyboard
    await page.keyboard.press('ArrowDown')
    
    // First suggestion should be selected
    const firstOption = page.getByRole('option').first()
    await expect(firstOption).toHaveAttribute('aria-selected', 'true')
    await expect(searchInput).toHaveAttribute('aria-activedescendant', 'suggestion-0')

    // Navigate to second suggestion
    await page.keyboard.press('ArrowDown')
    const secondOption = page.getByRole('option').nth(1)
    await expect(secondOption).toHaveAttribute('aria-selected', 'true')
    await expect(searchInput).toHaveAttribute('aria-activedescendant', 'suggestion-1')

    // Select suggestion with Enter
    await page.keyboard.press('Enter')
    
    // Should navigate to search results
    await expect(page.getByRole('heading', { name: /search results/i })).toBeVisible({ timeout: 10000 })
    await expect(searchInput).toHaveAttribute('aria-expanded', 'false')
  })

  test('screen reader compatibility and announcements', async ({ page }) => {
    const searchInput = page.getByRole('combobox')
    
    // Test ARIA labels and descriptions
    await expect(searchInput).toHaveAttribute('aria-label', expect.stringContaining('Search'))
    await expect(searchInput).toHaveAttribute('aria-describedby')
    
    // Test help text is present
    await expect(page.getByText(/Type to search for works/)).toBeInTheDocument()
    
    // Test live region updates
    await searchInput.fill('agatha harkness')
    
    // Should announce suggestions available
    await expect(page.getByRole('status')).toContainText(/suggestions available|Searching/)
    
    // Wait for suggestions
    await expect(page.getByRole('listbox')).toBeVisible({ timeout: 5000 })
    
    // Test suggestion descriptions
    const suggestions = page.getByRole('option')
    const firstSuggestion = suggestions.first()
    await expect(firstSuggestion).toHaveAttribute('aria-describedby', expect.stringContaining('suggestion-desc-'))
  })

  test('keyboard-only navigation throughout search flow', async ({ page }) => {
    // Start with Tab navigation
    await page.keyboard.press('Tab')
    await expect(page.getByRole('combobox')).toBeFocused()
    
    // Type search query
    await page.keyboard.type('agatha harkness')
    
    // Wait for suggestions and navigate with keyboard
    await expect(page.getByRole('listbox')).toBeVisible({ timeout: 5000 })
    
    // Navigate through suggestions
    await page.keyboard.press('ArrowDown') // First suggestion
    await page.keyboard.press('ArrowDown') // Second suggestion
    await page.keyboard.press('Enter') // Select
    
    // Should be on search results page
    await expect(page.getByRole('heading', { name: /search results/i })).toBeVisible({ timeout: 10000 })
    
    // Continue keyboard navigation to first result
    await page.keyboard.press('Tab')
    const firstResult = page.locator('[role="link"]').first()
    await expect(firstResult).toBeFocused()
  })

  test('escape key behavior and focus management', async ({ page }) => {
    const searchInput = page.getByRole('combobox')
    
    // Open suggestions
    await searchInput.fill('test query')
    await expect(page.getByRole('listbox')).toBeVisible({ timeout: 5000 })
    
    // Navigate to a suggestion
    await page.keyboard.press('ArrowDown')
    await expect(searchInput).toHaveAttribute('aria-activedescendant', 'suggestion-0')
    
    // Press Escape to close
    await page.keyboard.press('Escape')
    
    // Suggestions should close and focus should return to input
    await expect(page.getByRole('listbox')).not.toBeVisible()
    await expect(searchInput).toHaveAttribute('aria-expanded', 'false')
    await expect(searchInput).toBeFocused()
  })

  test('mouse and keyboard interaction compatibility', async ({ page }) => {
    const searchInput = page.getByRole('combobox')
    
    // Type to open suggestions
    await searchInput.fill('marvel')
    await expect(page.getByRole('listbox')).toBeVisible({ timeout: 5000 })
    
    // Mix keyboard and mouse interaction
    await page.keyboard.press('ArrowDown') // Navigate with keyboard
    await page.keyboard.press('ArrowDown')
    
    // Click on a different suggestion with mouse
    const thirdSuggestion = page.getByRole('option').nth(2)
    await thirdSuggestion.click()
    
    // Should still work and navigate to results
    await expect(page.getByRole('heading', { name: /search results/i })).toBeVisible({ timeout: 10000 })
  })

  test('high contrast and reduced motion compatibility', async ({ page }) => {
    // Test high contrast mode
    await page.emulateMedia({ colorScheme: 'dark' })
    
    const searchInput = page.getByRole('combobox')
    await searchInput.fill('test')
    
    // Should still be visible and functional
    await expect(searchInput).toBeVisible()
    await expect(page.getByRole('listbox')).toBeVisible({ timeout: 5000 })
    
    // Test with reduced motion preference
    await page.emulateMedia({ reducedMotion: 'reduce' })
    
    // Animations should respect reduced motion
    const loadingSpinner = page.getByRole('img', { name: 'Loading suggestions' })
    if (await loadingSpinner.isVisible()) {
      // Should still show loading indicator but with minimal animation
      await expect(loadingSpinner).toBeVisible()
    }
  })

  test('mobile accessibility and touch interaction', async ({ page, isMobile }) => {
    if (!isMobile) {
      // Set mobile viewport
      await page.setViewportSize({ width: 375, height: 667 })
    }
    
    const searchInput = page.getByRole('combobox')
    
    // Test touch interaction
    await searchInput.tap()
    await expect(searchInput).toBeFocused()
    
    // Type on mobile
    await searchInput.fill('mobile test')
    await expect(page.getByRole('listbox')).toBeVisible({ timeout: 5000 })
    
    // Tap suggestion
    const firstSuggestion = page.getByRole('option').first()
    await firstSuggestion.tap()
    
    // Should navigate successfully
    await expect(page.getByRole('heading', { name: /search results/i })).toBeVisible({ timeout: 10000 })
  })

  test('error states and accessibility announcements', async ({ page }) => {
    // Mock network error
    await page.route('**/api/v1/search/**', route => {
      route.abort('failed')
    })
    
    const searchInput = page.getByRole('combobox')
    await searchInput.fill('error test')
    
    // Should announce error gracefully
    await expect(page.getByRole('status')).toContainText(/unavailable|try again/i)
    
    // Input should remain functional
    await expect(searchInput).toBeFocused()
    await expect(searchInput).toHaveAttribute('aria-expanded', 'false')
  })

  test('multilingual accessibility support', async ({ page }) => {
    // Test with different language input
    const searchInput = page.getByRole('combobox')
    
    // Test Unicode input
    await searchInput.fill('Ελληνικά')
    await expect(searchInput).toHaveValue('Ελληνικά')
    
    // Test right-to-left languages
    await searchInput.fill('العربية')
    await expect(searchInput).toHaveValue('العربية')
    
    // Test Asian languages
    await searchInput.fill('日本語')
    await expect(searchInput).toHaveValue('日本語')
    
    // Should maintain accessibility features
    await expect(searchInput).toHaveAttribute('role', 'combobox')
    await expect(searchInput).toHaveAttribute('aria-label')
  })

  test('performance with accessibility features enabled', async ({ page }) => {
    // Measure performance of accessibility-enhanced search
    const startTime = Date.now()
    
    const searchInput = page.getByRole('combobox')
    await searchInput.fill('performance test')
    
    // Wait for suggestions
    await expect(page.getByRole('listbox')).toBeVisible({ timeout: 5000 })
    
    const endTime = Date.now()
    const duration = endTime - startTime
    
    // Should complete within reasonable time even with accessibility features
    expect(duration).toBeLessThan(3000)
    
    // Verify all accessibility features are present
    await expect(searchInput).toHaveAttribute('aria-expanded', 'true')
    await expect(page.getByRole('listbox')).toHaveAttribute('aria-label')
    
    const suggestions = page.getByRole('option')
    const firstSuggestion = suggestions.first()
    await expect(firstSuggestion).toHaveAttribute('aria-selected', 'false')
  })
})

test.describe('Search Results Accessibility', () => {
  test('search results page accessibility', async ({ page }) => {
    // Navigate to search results
    await page.goto('/search?q=test')
    
    // Should have proper heading structure
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    await expect(page.getByRole('heading', { name: /search results/i })).toBeVisible()
    
    // Should have proper landmarks
    await expect(page.getByRole('main')).toBeVisible()
    await expect(page.getByRole('search')).toBeVisible()
    
    // Results should be properly marked up
    const results = page.getByRole('article')
    if (await results.count() > 0) {
      const firstResult = results.first()
      await expect(firstResult).toBeVisible()
      
      // Should have proper headings
      await expect(firstResult.getByRole('heading')).toBeVisible()
    }
  })

  test('pagination accessibility', async ({ page }) => {
    await page.goto('/search?q=popular')
    
    // Look for pagination controls
    const pagination = page.getByRole('navigation', { name: /pagination/i })
    if (await pagination.isVisible()) {
      // Should have proper aria labels
      await expect(pagination).toBeVisible()
      
      // Page links should be properly labeled
      const pageLinks = pagination.getByRole('link')
      const count = await pageLinks.count()
      
      if (count > 0) {
        const firstPageLink = pageLinks.first()
        await expect(firstPageLink).toHaveAttribute('aria-label')
      }
    }
  })
})