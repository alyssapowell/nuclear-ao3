# Unified Data Schema for Nuclear AO3

## Work Model - Standardized Field Names

This document defines the unified field names that ALL services must use consistently across:
- PostgreSQL database schema
- Go models/structs  
- Elasticsearch mapping
- API responses
- Frontend components

### Core Work Fields

| Field Name | Type | Description | Validation |
|------------|------|-------------|------------|
| `id` | UUID | Unique identifier | Required |
| `title` | string | Work title | Required, 1-500 chars |
| `summary` | string | Work summary | Optional |
| `notes` | string | Author notes | Optional |
| `user_id` | UUID | Author ID | Required |
| `language` | string | Language code | Required, ISO 639-1 |

### Tag Fields (Arrays)
| Field Name | Type | Description |
|------------|------|-------------|
| `fandoms` | []string | Fandom tags |
| `characters` | []string | Character tags |
| `relationships` | []string | Relationship tags |
| `freeform_tags` | []string | Additional/freeform tags |
| `warnings` | []string | Archive warnings |
| `categories` | []string | Relationship categories |

### Metadata Fields
| Field Name | Type | Description | Valid Values |
|------------|------|-------------|--------------|
| `rating` | string | Content rating | `General Audiences`, `Teen And Up Audiences`, `Mature`, `Explicit`, `Not Rated` |
| `status` | string | Publication status | `draft`, `posted`, `hidden` |
| `word_count` | int | Total words | >= 0 |
| `chapter_count` | int | Chapter count | >= 1 |
| `is_complete` | bool | Completion status | true/false |

### Statistics Fields
| Field Name | Type | Description |
|------------|------|-------------|
| `hits_count` | int | View count |
| `kudos_count` | int | Kudos/likes count |
| `comments_count` | int | Comments count |
| `bookmarks_count` | int | Bookmarks count |

### Timestamp Fields
| Field Name | Type | Description |
|------------|------|-------------|
| `published_at` | timestamp | Publication date |
| `updated_at` | timestamp | Last update |
| `created_at` | timestamp | Creation date |

## Status Value Mapping

### Database/API Status Values:
- `draft` - Work is in draft state
- `posted` - Work is published and visible
- `hidden` - Work is hidden from public view

### Display Status Values:
- `Complete` - Work is finished (`is_complete: true`)
- `Work in Progress` - Work is ongoing (`is_complete: false`)
- `On Hiatus` - Work is paused
- `Abandoned` - Work is discontinued

## Implementation Requirements

1. **All database migrations** must use these exact field names
2. **All Go models** must use these field names in json/db tags
3. **All Elasticsearch mappings** must use these field names
4. **All API endpoints** must return these field names
5. **All frontend components** must expect these field names
6. **All search queries** must reference these field names

## Migration Strategy

1. Update Elasticsearch mapping to match unified schema
2. Update search service field references
3. Update tag service field references  
4. Re-sync all data with correct field names
5. Update frontend components if needed
6. Run comprehensive tests to validate alignment