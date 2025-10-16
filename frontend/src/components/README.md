# Nuclear AO3 Components Documentation

This directory contains reusable React components for the Nuclear AO3 frontend application.

## Table of Contents

- [Core Components](#core-components)
- [Comment System Components](#comment-system-components)
- [UI Components](#ui-components)
- [User Interface Components](#user-interface-components)
- [Search Components](#search-components)
- [Component Guidelines](#component-guidelines)

## Core Components

### Navigation.tsx
Main navigation component for the application.

### ApolloProvider.tsx
GraphQL Apollo Client provider for the application.

## Comment System Components

### Comments.tsx
**Main comment display and interaction component.**

- Displays threaded comments for works/chapters
- Handles comment creation, replies, and kudos
- Integrates with conversation view via SlideoutPanel
- Supports both authenticated and guest commenting
- Responsive design with mobile-first approach

**Props:**
- `workId: string` - ID of the work to show comments for
- `chapterId?: string` - Optional chapter ID for chapter-specific comments
- `allowComments?: boolean` - Whether to show comment form (default: true)
- `authToken?: string` - Authentication token for API calls

**Features:**
- Threaded comment display with configurable depth
- Conversation view links for multi-reply threads
- Real-time kudos toggle functionality
- Guest and authenticated user support
- Responsive slideout panel integration

### ConversationView.tsx
**Flattened conversation view component for comment threads.**

- Displays comment threads in a flattened, chronological format
- Provides better readability for deep conversation threads
- Handles reply targeting and @mentions automatically
- Optimized for both desktop slideout and mobile full-screen

**Props:**
- `threadId: string` - ID of the root comment to display thread for
- `workId: string` - ID of the work containing the comments
- `onClose?: () => void` - Callback when conversation is closed
- `onBackToComments?: () => void` - Callback to return to main comments view
- `className?: string` - Additional CSS classes

**Features:**
- Automatic @mention formatting for context
- Visual threading indicators
- Reply-to-specific-comment functionality
- Reply-to-conversation functionality
- Accessible navigation and keyboard support

### SlideoutPanel.tsx
**Reusable slideout panel component for desktop/tablet interfaces.**

- Provides modal-like experience for secondary content
- Handles focus management and accessibility
- Supports left/right positioning and custom widths
- Includes backdrop click and escape key handling

**Props:**
- `isOpen: boolean` - Controls panel visibility
- `onClose: () => void` - Callback when panel should close
- `title: string` - Panel title for accessibility
- `children: React.ReactNode` - Panel content
- `side?: 'left' | 'right'` - Panel slide direction (default: 'right')
- `width?: string` - Panel width CSS value (default: '600px')
- `className?: string` - Additional CSS classes

**Features:**
- Portal rendering for proper z-index stacking
- Focus trapping within panel
- Body scroll prevention
- Smooth slide animations
- Full accessibility support (ARIA attributes, keyboard navigation)

## UI Components

Located in `ui/` subdirectory:

### Badge.tsx
Reusable badge component for status indicators, tags, etc.

### Button.tsx
Standardized button component with variants and states.

### Card.tsx
Card container component for content grouping.

### Switch.tsx
Toggle switch component for boolean settings.

## User Interface Components

### UserProfileSettings.tsx
Comprehensive user profile management interface.

### FriendsAndSocial.tsx
Social features and friend management interface.

### BookmarkButton.tsx
Work bookmarking functionality component.

### KudosButton.tsx
Kudos giving/removal functionality component.

## Search Components

### SearchForm.tsx
Main search interface with GraphQL integration.

### SearchFormREST.tsx
Alternative search interface using REST API.

### SmartRecommendations.tsx
AI-powered work recommendation component.

### SmartTagFilter.tsx
Intelligent tag filtering with prominence scoring.

### TagAutocomplete.tsx
Tag input with autocomplete functionality.

### TagInput.tsx
Basic tag input component.

### TagProminenceSelector.tsx
Tag prominence scoring interface.

### EnhancedTagProminenceSelector.tsx
Advanced tag prominence with analytics.

## Component Guidelines

### File Organization
- One component per file
- Co-locate tests in `__tests__/` directory
- Use descriptive, PascalCase filenames
- Export default from component files

### Component Structure
```typescript
'use client'; // If component uses hooks or client-side features

import React from 'react';
import { /* types */ } from '@/types/...';

interface ComponentProps {
  // Define all props with JSDoc comments
}

export default function ComponentName({ 
  prop1, 
  prop2 = 'default' 
}: ComponentProps) {
  // Component logic

  return (
    <div className="component-wrapper">
      {/* Component JSX */}
    </div>
  );
}
```

### Styling Guidelines
- Use Tailwind CSS classes for styling
- Follow responsive-first design (mobile → desktop)
- Use semantic HTML elements
- Include proper ARIA attributes for accessibility

### Testing Requirements
- Write unit tests for all components
- Test user interactions (clicks, form submissions)
- Test accessibility features
- Mock external dependencies (APIs, portals)
- Aim for >80% coverage

### Accessibility Standards
- Include proper ARIA labels and roles
- Ensure keyboard navigation support
- Maintain focus management
- Support screen readers
- Test with accessibility tools

### Performance Considerations
- Use React.memo for expensive render operations
- Implement proper loading states
- Optimize images and assets
- Minimize bundle size impact

## Recent Updates

### Conversation View System (Latest)
Added comprehensive conversation threading system:

1. **SlideoutPanel** - New reusable component for modal-like experiences
2. **ConversationView** - Flattened comment thread display
3. **Enhanced Comments** - Integration with conversation view
4. **Mobile Support** - Full-screen conversation view on mobile
5. **Accessibility** - Full keyboard and screen reader support

### Testing Infrastructure
- Comprehensive test coverage for new components
- Portal mocking for SlideoutPanel tests
- Accessibility testing patterns
- User interaction testing

## Usage Examples

### Basic Comment Display
```typescript
<Comments 
  workId="work-123" 
  allowComments={true}
  authToken={userToken}
/>
```

### Slideout Panel with Custom Content
```typescript
<SlideoutPanel
  isOpen={showPanel}
  onClose={() => setShowPanel(false)}
  title="Custom Panel"
  width="700px"
>
  <CustomContent />
</SlideoutPanel>
```

### Conversation View Integration
```typescript
<ConversationView
  threadId="comment-456"
  workId="work-123"
  onClose={handleClose}
  onBackToComments={handleBack}
/>
```