# Accessibility Guidelines and Implementation

This document outlines the accessibility standards, implementation patterns, and testing strategies used in the Nuclear AO3 project to ensure 100% WCAG 2.1 AA compliance.

## Overview

Our accessibility-first approach ensures that all users, regardless of ability, can effectively use the search and discovery features of the archive. This implementation goes beyond basic compliance to provide an excellent user experience for screen reader users, keyboard-only users, and users with cognitive disabilities.

## WCAG 2.1 AA Compliance

### Four Principles of Accessibility

#### 1. Perceivable
- **Text Alternatives**: All images and icons have appropriate alt text or are marked as decorative
- **Captions and Descriptions**: Audio and video content include captions where applicable
- **Adaptable Content**: Content structure is preserved across different presentations
- **Distinguishable**: Text has sufficient color contrast (4.5:1 for normal text, 3:1 for large text)

#### 2. Operable
- **Keyboard Accessible**: All functionality is available via keyboard navigation
- **No Seizures**: No content flashes more than 3 times per second
- **Navigable**: Clear navigation with skip links, headings, and focus management
- **Input Assistance**: Users are helped to avoid and correct mistakes

#### 3. Understandable
- **Readable**: Text is readable and understandable
- **Predictable**: Web pages appear and operate in predictable ways
- **Input Assistance**: Users are helped to avoid and correct mistakes

#### 4. Robust
- **Compatible**: Content works with current and future assistive technologies

## Implementation Patterns

### 1. Semantic HTML Structure

```tsx
// Use semantic landmarks
<main role="main">
  <section aria-labelledby="search-heading">
    <h2 id="search-heading">Search Works</h2>
    <form role="search" aria-labelledby="search-heading">
      {/* Form content */}
    </form>
  </section>
</main>
```

### 2. ARIA Labels and Descriptions

```tsx
// Proper labeling
<input
  id="search-query"
  type="text"
  aria-labelledby="search-label"
  aria-describedby="search-help"
/>
<label id="search-label" htmlFor="search-query">Search Query</label>
<div id="search-help">Enter keywords to search for works</div>
```

### 3. Live Regions for Dynamic Content

```tsx
// Announce search results
<div aria-live="polite" aria-atomic="true" className="sr-only">
  {announceResults}
</div>

// Status updates
<div role="status" aria-live="polite">
  {loading ? 'Searching...' : `Found ${results.length} results`}
</div>
```

### 4. Focus Management

```tsx
// Focus management after search
useEffect(() => {
  if (results.length > 0 && !loading) {
    setAnnounceResults(`Found ${results.length} works. Results loaded.`);
    setTimeout(() => {
      resultsHeaderRef.current?.focus();
    }, 100);
  }
}, [results, loading]);
```

### 5. Keyboard Navigation

```tsx
// Custom keyboard handlers
const handleKeyDown = (e: React.KeyboardEvent) => {
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault();
    handleAction();
  }
  if (e.key === 'Escape') {
    handleEscape();
  }
};
```

## Component-Specific Accessibility Features

### SearchForm Component

#### Accessibility Features:
- **Semantic Form Structure**: Uses `role="search"` with proper form labeling
- **Error Handling**: `role="alert"` for immediate error announcements
- **Live Regions**: Announces form state changes and validation results
- **Focus Management**: Logical tab order and focus restoration
- **Tag Management**: Accessible tag addition/removal with screen reader feedback
- **Advanced Search**: Collapsible sections with proper ARIA states

#### Key Patterns:
```tsx
// Error announcement
<div role="alert" className="error-message">
  {error}
</div>

// Tag removal
<button
  onClick={() => removeTag(field, index, tag)}
  aria-label={`Remove ${tag} from ${field}`}
  title={`Remove ${tag}`}
>
  <X className="h-3 w-3" aria-hidden="true" />
</button>

// Advanced search toggle
<button
  aria-expanded={showAdvanced}
  aria-controls={filtersId}
>
  {showAdvanced ? 'Hide' : 'Show'} Advanced Search
</button>
```

### SearchResults Component

#### Accessibility Features:
- **Structured Content**: Each result is an `article` with proper heading hierarchy
- **Rich Metadata**: All work information is available to screen readers
- **Interactive Elements**: Work titles are properly accessible buttons/links
- **Status Information**: Loading and result count announcements
- **Tag Organization**: Logical grouping and labeling of tags

#### Key Patterns:
```tsx
// Work article structure
<article
  role="article"
  aria-labelledby={`work-${work.id}-title`}
  aria-describedby={`work-${work.id}-meta work-${work.id}-summary`}
>
  <h4 id={`work-${work.id}-title`}>
    {work.title}
  </h4>
  {/* Metadata and content */}
</article>

// Interactive work title
<h4
  onClick={() => handleWorkClick(work.id, work.title)}
  onKeyDown={(e) => handleWorkKeyDown(e, work.id, work.title)}
  role="button"
  tabIndex={0}
  aria-describedby={`work-${work.id}-action-help`}
>
  {work.title}
</h4>
```

### SearchPagination Component

#### Accessibility Features:
- **Navigation Landmark**: Uses `role="navigation"` with descriptive label
- **Page Status**: Current page clearly indicated with `aria-current="page"`
- **Action Descriptions**: Clear labels for all pagination actions
- **Keyboard Support**: Full keyboard navigation with Enter/Space support
- **Screen Reader Context**: Page numbers include context ("Go to page X")

#### Key Patterns:
```tsx
// Navigation structure
<nav
  role="navigation"
  aria-labelledby="pagination-label"
>
  <h3 id="pagination-label" className="sr-only">
    Search Results Pagination
  </h3>
  {/* Pagination controls */}
</nav>

// Current page indicator
<button
  aria-label={`Current page, page ${page}`}
  aria-current="page"
  disabled
>
  {page}
</button>

// Navigation buttons
<button
  aria-label={`Go to page ${page}`}
  onClick={() => handlePageChange(page)}
>
  {page}
</button>
```

## Testing Strategy

### Automated Testing

We use comprehensive automated accessibility testing with axe-core:

```typescript
// Accessibility test example
test('SearchForm - Complete accessibility audit', async ({ page }) => {
  await page.goto('/search');
  
  const accessibilityScanResults = await new AxeBuilder({ page })
    .include('[role="search"]')
    .analyze();

  expect(accessibilityScanResults.violations).toEqual([]);
});
```

### Manual Testing Checklist

#### Keyboard Navigation
- [ ] All interactive elements are keyboard accessible
- [ ] Tab order is logical and follows visual flow
- [ ] Focus indicators are clearly visible
- [ ] No keyboard traps exist
- [ ] Escape key closes modals/dropdowns

#### Screen Reader Testing
- [ ] All content is announced properly
- [ ] Landmarks provide clear navigation
- [ ] Form labels and descriptions are associated correctly
- [ ] Live regions announce dynamic changes
- [ ] Error messages are announced immediately

#### Color and Contrast
- [ ] Text meets minimum contrast ratios
- [ ] Color is not the only way to convey information
- [ ] Focus indicators have sufficient contrast
- [ ] Error states are distinguishable without color

#### Cognitive Accessibility
- [ ] Interface is predictable and consistent
- [ ] Error prevention and correction is available
- [ ] Instructions are clear and helpful
- [ ] Time limits are avoidable or adjustable

## Browser and Assistive Technology Support

### Target Support
- **Screen Readers**: NVDA, JAWS, VoiceOver, TalkBack
- **Browsers**: Chrome, Firefox, Safari, Edge (latest 2 versions)
- **Keyboard Navigation**: All major browsers
- **Voice Control**: Dragon NaturallySpeaking, Voice Control

### Testing Matrix
- Windows: NVDA + Chrome/Firefox, JAWS + Chrome/Edge
- macOS: VoiceOver + Safari/Chrome
- iOS: VoiceOver + Safari
- Android: TalkBack + Chrome

## Best Practices

### 1. Use Semantic HTML First
Always start with semantic HTML before adding ARIA. Use `<button>` for actions, `<a>` for navigation, proper headings, etc.

### 2. Provide Multiple Ways to Access Content
- Keyboard navigation
- Mouse/touch interaction
- Voice commands
- Search functionality

### 3. Make Error Recovery Easy
- Clear error messages
- Suggestions for correction
- Preserve user input when possible
- Provide multiple ways to complete tasks

### 4. Test with Real Users
- Include users with disabilities in testing
- Use actual assistive technologies
- Test in realistic scenarios

### 5. Progressive Enhancement
- Ensure basic functionality works without JavaScript
- Add enhanced experiences with proper fallbacks
- Consider users with older assistive technologies

## Common Patterns and Solutions

### Autocomplete/Combobox
```tsx
<input
  role="combobox"
  aria-expanded={isOpen}
  aria-owns="listbox-id"
  aria-autocomplete="list"
  aria-activedescendant={activeOptionId}
/>
<ul role="listbox" id="listbox-id">
  <li role="option" aria-selected={isSelected}>
    Option text
  </li>
</ul>
```

### Loading States
```tsx
<div role="status" aria-live="polite">
  {loading ? 'Loading...' : 'Content loaded'}
</div>
```

### Error Messages
```tsx
<div role="alert" aria-live="assertive">
  {error}
</div>
```

### Skip Links
```tsx
<a href="#main-content" className="skip-link">
  Skip to main content
</a>
```

## Resources

### Documentation
- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [ARIA Authoring Practices Guide](https://www.w3.org/WAI/ARIA/apg/)
- [MDN Accessibility](https://developer.mozilla.org/en-US/docs/Web/Accessibility)

### Testing Tools
- [axe DevTools](https://www.deque.com/axe/devtools/)
- [WAVE Web Accessibility Evaluator](https://wave.webaim.org/)
- [Color Contrast Analyzers](https://www.tpgi.com/color-contrast-checker/)

### Screen Readers
- [NVDA (Free)](https://www.nvaccess.org/)
- [VoiceOver (Built into macOS/iOS)](https://support.apple.com/guide/voiceover/)
- [JAWS](https://www.freedomscientific.com/products/software/jaws/)

## Contributing

When contributing to this project:

1. **Test your changes** with keyboard navigation and screen readers
2. **Run accessibility tests** before submitting PRs
3. **Follow established patterns** documented here
4. **Update documentation** when adding new patterns
5. **Consider edge cases** and different user scenarios

Remember: Accessibility is not a feature to be added later—it's a fundamental requirement that should be considered from the beginning of development.