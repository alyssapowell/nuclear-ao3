# Demo System Segregation

This document explains how Nuclear AO3's demo functionality is segregated from production code for easy removal during deployment.

## Overview

The demo system showcases Nuclear AO3's Tag Prominence System with interactive examples for both readers and authors. All demo functionality is contained under the `/demo/*` route structure and can be easily excluded from production builds.

## Demo Pages Structure

```
/demo/                    - Demo hub with overview
/demo/tag-prominence      - Tag prominence system overview
/demo/browse              - Reader filtering experience
/demo/author-workflow     - Author tagging workflow
/demo/reader              - Mobile reading experience
/demo/components          - Design system showcase
```

## Environment Controls

### Development Environment
- Demos are **always enabled** in development (`NODE_ENV=development`)
- All demo routes are accessible
- Demo links appear in navigation

### Production Environment
- Demos are **disabled by default** in production
- All `/demo/*` routes redirect to 404
- Demo links are hidden from navigation
- Can be enabled with `ENABLE_DEMOS=true` environment variable

### Manual Override
Set `ENABLE_DEMOS=true` in production to enable demos for showcasing:

```bash
# Enable demos in production
ENABLE_DEMOS=true npm run build
ENABLE_DEMOS=true npm start
```

## Implementation Details

### Route Protection
- `next.config.ts` - Redirects `/demo/*` to 404 when demos disabled
- Uses Next.js rewrites for efficient blocking

### Environment Detection
- `src/lib/demo.ts` - Utilities for checking demo availability
- `isDemoEnabled()` - Returns true if demos should be shown
- `getDemoRoutes()` - Returns available demo routes

### Component Integration
- Demo links only render when `isDemoEnabled()` returns true
- Main page shows demo banner only in development/when enabled
- Navigation components use demo utilities

## Files Structure

### Demo Pages (can be excluded)
```
frontend/src/app/demo/               - All demo routes
├── page.tsx                         - Demo hub
├── tag-prominence/page.tsx          - Tag system overview
├── browse/page.tsx                  - Reader demo
├── author-workflow/page.tsx         - Author demo
├── reader/page.tsx                  - Mobile reading demo
└── components/page.tsx              - Component showcase
```

### Production Components (keep)
```
frontend/src/components/             - All production components
├── ui/                              - Design system
├── SmartTagFilter.tsx               - Reader filtering
├── TagProminenceSelector.tsx        - Author tagging
├── WorkCard.tsx                     - Work display
└── ...                              - Other components
```

### Infrastructure
```
frontend/src/lib/demo.ts             - Demo utilities
frontend/next.config.ts              - Route protection
```

## Deployment Strategy

### Option 1: Environment Variable (Recommended)
Deploy to production without setting `ENABLE_DEMOS`, demos will be automatically disabled.

### Option 2: Build-time Exclusion
Exclude demo directory during build:
```bash
# Example build script that excludes demos
rm -rf src/app/demo
npm run build
```

### Option 3: Build Configuration
Modify `next.config.ts` to exclude demo routes entirely in production builds.

## Benefits

1. **Clean Separation** - All demo code is isolated under `/demo/*`
2. **Easy Removal** - Delete one directory to remove all demos
3. **Environment Aware** - Automatically handles dev vs production
4. **Performance** - No demo code in production builds
5. **Flexibility** - Can enable demos for showcasing when needed

## Production Components Usage

The core Tag Prominence components can be used in production applications:

```tsx
import SmartTagFilter from '@/components/SmartTagFilter';
import TagProminenceSelector from '@/components/TagProminenceSelector';
import WorkCard from '@/components/WorkCard';
import { Badge } from '@/components/ui';

// Reader filtering
<SmartTagFilter filters={filters} onFilterChange={handleChange} />

// Author tagging interface
<TagProminenceSelector tags={tags} onTagsChange={handleTags} />

// Work display with prominence
<WorkCard work={work} showTagProminence={true} />

// Prominence badges
<Badge prominence="primary">Main Ship</Badge>
<Badge prominence="secondary">Important Theme</Badge>
<Badge prominence="micro">Background Element</Badge>
```

## Testing Demo Exclusion

### Test in Development
```bash
ENABLE_DEMOS=false npm run dev
# Visit /demo - should show 404
```

### Test Production Build
```bash
npm run build
npm start
# Visit /demo - should show 404
# Main page should not show demo links
```

### Test Manual Override
```bash
ENABLE_DEMOS=true npm run build
ENABLE_DEMOS=true npm start
# Visit /demo - should work normally
```

This system ensures that Nuclear AO3's powerful Tag Prominence System can be demonstrated effectively while keeping production deployments clean and focused.