# Nuclear AO3 Test Data Generator

This tool generates realistic test data for the Nuclear AO3 platform, simulating the scale and complexity of AO3's actual data structure.

## Features

- **Realistic Fandoms**: Pre-seeded with popular fandoms across different media types
- **Character Networks**: Generates character relationships within fandoms
- **Tag Hierarchies**: Creates canonical tags with synonyms and relationships
- **Diverse Works**: Generates works with realistic metadata distributions
- **User Simulation**: Creates diverse user profiles with different activity patterns
- **Performance Testing**: Configurable data scales for stress testing

## Data Generation Strategy

### Scale Options

1. **Small** (Development): ~1K works, ~100 users, ~500 tags
2. **Medium** (Testing): ~10K works, ~1K users, ~2K tags  
3. **Large** (Performance): ~100K works, ~10K users, ~10K tags
4. **Stress** (Load Testing): ~1M works, ~100K users, ~50K tags

### Data Composition

#### Fandoms (Based on AO3 popularity patterns)
- **Top Tier**: Marvel, Harry Potter, Sherlock Holmes, BTS, etc.
- **Popular**: Anime/Manga, TV shows, video games
- **Niche**: Books, podcasts, obscure media
- **Distribution**: 80/20 rule (80% works in top 20% fandoms)

#### Tags
- **Canonical Structure**: Main tags with synonym variants
- **Categories**: Relationships (40%), Characters (25%), Freeform (20%), Fandoms (10%), Warnings/Ratings (5%)
- **Wrangling**: Mix of wrangled and unwrangled tags for realistic workflow

#### Works
- **Length Distribution**: Realistic word count distribution (most 1-5K, some longfics 50K+)
- **Completion Status**: Mix of complete, in-progress, and abandoned works
- **Publication Patterns**: Realistic upload schedules and update frequencies

#### Users  
- **Activity Levels**: Power users, casual readers, occasional writers
- **Preferences**: Realistic reading/writing patterns based on fandom popularity
- **Geographic Distribution**: Timezone-aware activity patterns

### Quality Assurance

- **Data Integrity**: All foreign key relationships maintained
- **Realistic Ratios**: Kudos, comments, bookmarks follow real-world patterns  
- **Performance Indexes**: Generated data designed to stress-test indexes
- **Search Optimization**: Content structured for Elasticsearch indexing

## Usage

```bash
# Generate small development dataset
go run generator.go --scale small

# Generate with specific parameters
go run generator.go --users 1000 --works 5000 --fandoms 50

# Performance testing dataset
go run generator.go --scale stress --output performance_data.sql

# AO3-based realistic data (uses AO3 metadata patterns)
go run generator.go --realistic --source ao3_patterns.json
```

This approach ensures we have high-quality test data that reflects real-world usage patterns while being suitable for both development and performance testing.