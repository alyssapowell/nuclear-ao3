#!/usr/bin/env node

/**
 * Elasticsearch Sync Script
 * Syncs works from PostgreSQL to Elasticsearch for search functionality
 */

const { Client } = require('pg');
const { Client: ElasticsearchClient } = require('@elastic/elasticsearch');

const ES_CLIENT = new ElasticsearchClient({ node: 'http://localhost:9200' });
const PG_CLIENT = new Client({
  host: 'localhost',
  port: 5432,
  user: 'ao3_user',
  password: 'ao3_nuclear_password',
  database: 'ao3_nuclear'
});

class ElasticsearchSyncer {
  constructor() {
    this.batchSize = 100;
    this.totalSynced = 0;
  }

  async run() {
    console.log('🔄 Starting Elasticsearch sync...\n');
    
    try {
      await PG_CLIENT.connect();
      console.log('✅ Connected to PostgreSQL');

      // Create index if it doesn't exist
      await this.createIndex();
      
      // Get total count
      const countResult = await PG_CLIENT.query('SELECT COUNT(*) FROM works');
      const totalWorks = parseInt(countResult.rows[0].count);
      console.log(`📚 Found ${totalWorks} works to sync\n`);

      if (totalWorks === 0) {
        console.log('❌ No works found in database');
        return;
      }

      // Sync in batches
      let offset = 0;
      while (offset < totalWorks) {
        await this.syncBatch(offset, this.batchSize);
        offset += this.batchSize;
        
        const progress = Math.min(offset, totalWorks);
        const percent = Math.round((progress / totalWorks) * 100);
        console.log(`📊 Progress: ${progress}/${totalWorks} (${percent}%)`);
      }

      console.log(`\n🎉 Sync complete! ${this.totalSynced} works indexed`);
      
      // Verify the sync
      await this.verifySync();

    } catch (error) {
      console.error('❌ Sync failed:', error);
    } finally {
      await PG_CLIENT.end();
    }
  }

  async createIndex() {
    try {
      const exists = await ES_CLIENT.indices.exists({ index: 'works' });
      
      if (!exists) {
        console.log('🔧 Creating Elasticsearch index...');
        
        await ES_CLIENT.indices.create({
          index: 'works',
          body: {
            mappings: {
              properties: {
                title: { type: 'text', analyzer: 'standard' },
                summary: { type: 'text', analyzer: 'standard' },
                notes: { type: 'text', analyzer: 'standard' },
                fandom: { type: 'keyword' },
                characters: { type: 'keyword' },
                relationships: { type: 'keyword' },
                additional_tags: { type: 'keyword' },
                rating: { type: 'keyword' },
                warnings: { type: 'keyword' },
                categories: { type: 'keyword' },
                language: { type: 'keyword' },
                status: { type: 'keyword' },
                word_count: { type: 'integer' },
                chapter_count: { type: 'integer' },
                kudos_count: { type: 'integer' },
                hits_count: { type: 'integer' },
                bookmarks_count: { type: 'integer' },
                comments_count: { type: 'integer' },
                published_date: { type: 'date' },
                updated_date: { type: 'date' },
                user_id: { type: 'keyword' },
                is_complete: { type: 'boolean' },
                content_text: { type: 'text', analyzer: 'standard' }
              }
            }
          }
        });
        
        console.log('✅ Index created');
      } else {
        console.log('✅ Index already exists');
      }
    } catch (error) {
      console.error('❌ Failed to create index:', error);
      throw error;
    }
  }

  async syncBatch(offset, limit) {
    try {
      const query = `
        SELECT 
          w.*,
          array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL AND tt.tag_type = 'fandom') as fandoms,
          array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL AND tt.tag_type = 'character') as characters,
          array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL AND tt.tag_type = 'relationship') as relationships,
          array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL AND tt.tag_type = 'freeform') as additional_tags,
          array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL AND tt.tag_type = 'warning') as warnings,
          array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL AND tt.tag_type = 'category') as categories
        FROM works w
        LEFT JOIN work_tags wt ON w.id = wt.work_id
        LEFT JOIN tags t ON wt.tag_id = t.id
        LEFT JOIN (
          SELECT DISTINCT name,
            CASE 
              WHEN name ILIKE '%/%' AND name NOT ILIKE 'Marvel Cinematic Universe%' THEN 'relationship'
              WHEN name IN ('Gen', 'F/F', 'F/M', 'M/M', 'Multi', 'Other') THEN 'category'
              WHEN name IN ('General Audiences', 'Teen And Up Audiences', 'Mature', 'Explicit', 'Not Rated') THEN 'rating'
              WHEN name IN ('Graphic Depictions Of Violence', 'Major Character Death', 'No Archive Warnings Apply', 'Rape/Non-Con', 'Underage') THEN 'warning'
              WHEN name ILIKE '%- J. K. Rowling%' OR name ILIKE '%Marvel%' OR name ILIKE '%Supernatural%' OR name ILIKE '%My Hero Academia%' THEN 'fandom'
              WHEN name ~ '^[A-Z][a-z]+ [A-Z][a-z]+$' OR name ~ '^[A-Z][a-z]+ [A-Z]\\. [A-Z][a-z]+$' THEN 'character'
              ELSE 'freeform'
            END as tag_type
          FROM tags
        ) tt ON t.name = tt.name
        GROUP BY w.id
        ORDER BY w.id
        OFFSET $1 LIMIT $2
      `;

      const result = await PG_CLIENT.query(query, [offset, limit]);
      
      if (result.rows.length === 0) {
        return;
      }

      // Prepare bulk index operations
      const bulkOps = [];
      
      for (const work of result.rows) {
        bulkOps.push({
          index: {
            _index: 'works',
            _id: work.id
          }
        });

        // Clean up arrays (remove nulls and empty values)
        const cleanArray = (arr) => arr ? arr.filter(item => item && item.trim()) : [];

        const doc = {
          title: work.title || '',
          summary: work.summary || '',
          notes: work.notes || '',
          fandom: cleanArray(work.fandoms),
          characters: cleanArray(work.characters),
          relationships: cleanArray(work.relationships),
          additional_tags: cleanArray(work.additional_tags),
          rating: work.rating || 'Not Rated',
          warnings: cleanArray(work.warnings),
          categories: cleanArray(work.categories),
          language: work.language || 'English',
          status: work.is_complete ? 'Complete' : 'Work in Progress',
          word_count: parseInt(work.word_count) || 0,
          chapter_count: parseInt(work.chapter_count) || 1,
          kudos_count: parseInt(work.kudos_count) || 0,
          hits_count: parseInt(work.hits_count) || 0,
          bookmarks_count: parseInt(work.bookmarks_count) || 0,
          comments_count: parseInt(work.comments_count) || 0,
          published_date: work.published_date,
          updated_date: work.updated_date,
          user_id: work.user_id,
          is_complete: work.is_complete || false,
          content_text: [work.title, work.summary, work.notes].filter(Boolean).join(' ')
        };

        bulkOps.push(doc);
      }

      // Bulk index to Elasticsearch
      if (bulkOps.length > 0) {
        const response = await ES_CLIENT.bulk({
          refresh: false,
          body: bulkOps
        });

        if (response.errors) {
          console.warn(`⚠️  Some documents failed to index in batch ${offset}-${offset + limit}`);
        }

        this.totalSynced += result.rows.length;
      }

    } catch (error) {
      console.error(`❌ Failed to sync batch ${offset}-${offset + limit}:`, error);
      throw error;
    }
  }

  async verifySync() {
    try {
      // Refresh index to make documents searchable
      await ES_CLIENT.indices.refresh({ index: 'works' });
      
      // Get document count
      const response = await ES_CLIENT.count({ index: 'works' });
      console.log(`\n✅ Verification: ${response.count} documents in Elasticsearch index`);
      
      // Test a simple search
      const searchResponse = await ES_CLIENT.search({
        index: 'works',
        body: {
          query: { match_all: {} },
          size: 1
        }
      });

      if (searchResponse.hits.total.value > 0) {
        console.log('✅ Search verification: Index is searchable');
        
        const sampleDoc = searchResponse.hits.hits[0]._source;
        console.log(`📝 Sample document: "${sampleDoc.title}" (${sampleDoc.word_count} words)`);
      } else {
        console.log('❌ Search verification: No documents found');
      }

    } catch (error) {
      console.error('❌ Verification failed:', error);
    }
  }
}

// Check if we have the required dependencies
try {
  require('pg');
  require('@elastic/elasticsearch');
} catch (error) {
  console.error('❌ Missing dependencies. Installing...');
  require('child_process').execSync('npm install pg @elastic/elasticsearch', { stdio: 'inherit' });
  console.log('✅ Dependencies installed');
}

// Run the sync
const syncer = new ElasticsearchSyncer();
syncer.run().catch(error => {
  console.error('Sync process crashed:', error);
  process.exit(1);
});