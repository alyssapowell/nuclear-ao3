# Tag Autocomplete & Canonical System

Nuclear AO3 includes a comprehensive tag suggestion and canonical mapping system to improve tagging consistency and user experience.

## Overview

The system provides:
- **Real-time autocomplete** for tag input fields
- **Canonical tag mapping** to reduce duplicates
- **Tag synonym support** for alternative spellings
- **Popularity indicators** based on usage count
- **Fandom-specific filtering** for relevant suggestions
- **AI-powered prominence inference**

## System Architecture

### Backend Components

#### 1. Tag Models (`backend/shared/models/tag.go`)

```go
// Core tag with canonical information
type Tag struct {
    ID            uuid.UUID
    Name          string
    Type          string  // fandom, character, relationship, freeform, etc.
    Canonical     bool    // Whether this is a canonical tag
    CanonicalName *string // Points to canonical if this is a synonym
    UseCount      int     // Popularity metric
}

// Synonym mapping
type TagSynonym struct {
    ID          uuid.UUID
    CanonicalID uuid.UUID
    SynonymName string
}

// Autocomplete request/response
type TagAutoCompleteRequest struct {
    Query      string
    Type       []string    // Filter by tag types
    FandomID   *uuid.UUID  // Fandom-specific suggestions
    Limit      int
    ExcludeIDs []uuid.UUID // Don't suggest these tags
}
```

#### 2. Tag Service API (`backend/tag-service/`)

**Endpoints:**
- `GET /api/v1/tags/autocomplete` - Real-time tag suggestions
- `POST /api/v1/tags` - Create new tags
- `POST /api/v1/tags/:id/synonym` - Add synonym mapping
- `GET /api/v1/wrangling/tags` - Tag management for moderators

**Features:**
- **Cached suggestions** using Redis for performance
- **Fandom filtering** for character/relationship tags
- **Popularity ranking** by use count
- **Type filtering** (relationships, characters, freeform)
- **Exclude functionality** to avoid duplicate suggestions

#### 3. Tag Wrangling System

```go
type TagWrangling struct {
    TagID      uuid.UUID
    Status     string     // unwrangled, wrangled, disputed, banned
    WrangledBy *uuid.UUID
    Notes      string
}

type TagMerge struct {
    FromTagID uuid.UUID  // Merge this tag
    ToTagID   uuid.UUID  // Into this canonical tag
    Status    string     // pending, approved, rejected
}
```

### Frontend Components

#### 1. TagInput Component (`frontend/src/components/TagInput.tsx`)

**Features:**
- Debounced autocomplete search (300ms)
- Keyboard navigation (arrow keys, enter, escape)
- Visual indicators for canonical/popular tags
- Type-specific color coding
- Click and keyboard selection

**Usage:**
```tsx
<TagInput
  value={inputValue}
  onChange={setInputValue}
  onTagAdd={handleTagAdd}
  tagType={['relationship']}
  fandomId="harry-potter"
  placeholder="Type to search relationships..."
/>
```

#### 2. EnhancedTagProminenceSelector

**Enhanced Features:**
- Integrated tag autocomplete for each tag type
- AI-powered prominence inference
- Visual canonical/popularity indicators
- Real-time tag statistics
- Drag-and-drop organization

**AI Inference Logic:**
```typescript
const inferProminence = (tagName: string, tagType: string) => {
  const lowerTag = tagName.toLowerCase();
  
  // Background mentions -> micro
  if (lowerTag.includes('background') || lowerTag.includes('past')) {
    return 'micro';
  }
  
  // Major themes -> primary
  if (tagType === 'freeform' && lowerTag.includes('slow burn')) {
    return 'primary';
  }
  
  // Default logic based on type and existing tags
  return 'secondary';
};
```

## Tag Suggestion Features

### 1. Canonical Tag Recognition

```
User types: "harry potter/draco malfoy"
System suggests:
✓ Harry Potter/Draco Malfoy (CANONICAL) - 15,234 uses
  Harry/Draco - 8,891 uses  
  Drarry - 12,456 uses
```

### 2. Fandom-Specific Filtering

When a fandom is selected, character and relationship suggestions are filtered to that fandom:

```
Fandom: Harry Potter
User types: "hermione"
Suggests:
✓ Hermione Granger (CANONICAL) - 25,678 uses
  Hermione Jean Granger - 1,234 uses
  
Excludes:
❌ Hermione Lodge (Riverdale fandom)
```

### 3. Popularity Indicators

- **🔥 Popular** - High usage count (>1000 uses)
- **✓ Canonical** - Official canonical form
- **🤖 AI Suggested** - Prominence auto-suggested by AI

### 4. Smart Prominence Inference

The system automatically suggests prominence based on tag patterns:

| Pattern | Example | Suggested Prominence |
|---------|---------|---------------------|
| "Background X/Y" | Background Ron/Hermione | Micro |
| "Past X/Y" | Past Harry/Ginny | Micro |
| "Major themes" | Slow Burn, Angst | Primary |
| "Setting tags" | Coffee Shop AU | Secondary |
| "Multiple relationships" | 3rd+ relationship | Secondary |

## API Integration

### Frontend API Client (`frontend/src/lib/api.ts`)

```typescript
export async function getTagSuggestions(request: TagAutoCompleteRequest) {
  const response = await fetch(`/api/v1/tags/autocomplete?${params}`, {
    headers: { 'Accept': 'application/json' }
  });
  return response.json();
}
```

### Request Examples

```bash
# Basic autocomplete
GET /api/v1/tags/autocomplete?q=harry&limit=10

# Relationship suggestions for specific fandom
GET /api/v1/tags/autocomplete?q=harry&type=relationship&fandom_id=hp-uuid

# Exclude already selected tags
GET /api/v1/tags/autocomplete?q=hermione&exclude_id=tag1&exclude_id=tag2
```

### Response Format

```json
{
  "suggestions": [
    {
      "id": "uuid",
      "name": "Harry Potter/Draco Malfoy",
      "type": "relationship",
      "use_count": 15234,
      "canonical": true
    }
  ]
}
```

## Tag Wrangling Workflow

### 1. User Reports/Moderator Review
- Users can report mistagged content
- Moderators review unwrangled tags
- Status tracking: unwrangled → wrangled → reviewed

### 2. Synonym Creation
```bash
POST /api/v1/wrangling/tags/{canonical_id}/synonym
{
  "synonym_name": "Drarry",
  "notes": "Common ship name abbreviation"
}
```

### 3. Tag Merging
```bash
POST /api/v1/wrangling/tags/merge
{
  "from_tag_id": "duplicate-tag-uuid",
  "to_tag_id": "canonical-tag-uuid", 
  "reason": "Duplicate with different capitalization"
}
```

## Performance Optimizations

### 1. Redis Caching
```go
cacheKey := fmt.Sprintf("autocomplete:%s:%v:%v:%d", 
  query, tagTypes, fandomID, limit)
cached, err := redis.Get(ctx, cacheKey).Result()
```

### 2. Database Indexing
```sql
CREATE INDEX idx_tags_name_trgm ON tags USING gin (name gin_trgm_ops);
CREATE INDEX idx_tags_type_usecount ON tags (type, use_count DESC);
```

### 3. Debounced Requests
Frontend waits 300ms after user stops typing before making API request.

## Usage Examples

### For Authors
1. **Writing a Harry/Draco fic:**
   - Type "harry" → suggests "Harry Potter/Draco Malfoy" (canonical)
   - Type "background ron" → auto-suggests micro prominence
   - See popularity indicators to choose well-known tags

### For Readers  
1. **Filtering for primary relationships:**
   - Only shows works where Harry/Draco is marked as primary focus
   - Filters out works where it's just a background mention

### For Moderators
1. **Tag wrangling:**
   - Review new/disputed tags
   - Create synonym mappings
   - Merge duplicate tags
   - Monitor tag quality

## Benefits

1. **Consistency** - Canonical tags reduce variations
2. **Discoverability** - Popular tags help readers find content
3. **Accuracy** - AI prominence helps correct classification
4. **User Experience** - Fast, relevant suggestions
5. **Moderation** - Tools for maintaining tag quality

This system ensures that Nuclear AO3's tagging is both user-friendly and maintains the high quality needed for effective content discovery.