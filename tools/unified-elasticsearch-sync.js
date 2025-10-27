#!/usr/bin/env node

/**
 * Unified Elasticsearch Sync Script
 * Syncs works from PostgreSQL to Elasticsearch using the unified schema
 */

const { Client } = require('pg');
const { Client: ElasticsearchClient } = require('@elastic/elasticsearch');

const ES_CLIENT = new ElasticsearchClient({ node: 'http://localhost:9200' });
const PG_CLIENT = new Client({
  host: 'localhost',
  port: 5432,
  user: 'ao3_user',
  password: 'ao3_password',
  database: 'ao3_nuclear'
});

class UnifiedElasticsearchSyncer {
  constructor() {
    this.batchSize = 100;
    this.totalSynced = 0;
  }

  async run() {
    console.log('🔄 Starting Unified Elasticsearch sync...\n');
    
    try {
      await PG_CLIENT.connect();
      console.log('✅ Connected to PostgreSQL');

      // Delete and recreate index with unified schema
      await this.recreateIndex();
      
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

  async recreateIndex() {
    try {
      // Delete existing index
      const exists = await ES_CLIENT.indices.exists({ index: 'works' });
      if (exists) {
        console.log('🗑️  Deleting existing index...');
        await ES_CLIENT.indices.delete({ index: 'works' });
      }
      
      console.log('🔧 Creating Elasticsearch index with unified schema...');
      
      await ES_CLIENT.indices.create({
        index: 'works',
        body: {
          mappings: {
            properties: {
              // Core fields - unified schema
              id: { type: 'keyword' },
              title: { 
                type: 'text', 
                analyzer: 'standard',
                fields: { keyword: { type: 'keyword' } }
              },
              summary: { type: 'text', analyzer: 'standard' },
              notes: { type: 'text', analyzer: 'standard' },
              user_id: { type: 'keyword' },
              
              // Tag fields - unified schema (arrays)
              fandoms: { 
                type: 'keyword',
                fields: { text: { type: 'text', analyzer: 'standard' } }
              },
              characters: { 
                type: 'keyword',
                fields: { text: { type: 'text', analyzer: 'standard' } }
              },
              relationships: { 
                type: 'keyword',
                fields: { text: { type: 'text', analyzer: 'standard' } }
              },
              freeform_tags: { 
                type: 'keyword',
                fields: { text: { type: 'text', analyzer: 'standard' } }
              },
              warnings: { type: 'keyword' },
              categories: { type: 'keyword' },
              
              // Metadata fields - unified schema
              rating: { type: 'keyword' },
              language: { type: 'keyword' },
              status: { type: 'keyword' },
              word_count: { type: 'integer' },
              chapter_count: { type: 'integer' },
              is_complete: { type: 'boolean' },
              
              // Statistics fields - unified schema
              hits_count: { type: 'integer' },
              kudos_count: { type: 'integer' },
              comments_count: { type: 'integer' },
              bookmarks_count: { type: 'integer' },
              
              // Timestamp fields - unified schema
              published_at: { type: 'date' },
              updated_at: { type: 'date' },
              created_at: { type: 'date' },
              
              // Search helper field
              content_text: { type: 'text', analyzer: 'standard' }
            }
          },
          settings: {
            number_of_shards: 1,
            number_of_replicas: 0,
            analysis: {
              analyzer: {
                standard: {
                  type: 'standard'
                }
              }
            }
          }
        }
      });
      
      console.log('✅ Index created with unified schema');
    } catch (error) {
      console.error('❌ Failed to create index:', error);
      throw error;
    }
  }

  async syncBatch(offset, limit) {
    try {
      const query = `
        SELECT 
          id,
          title,
          summary,
          notes,
          user_id,
          language,
          rating,
          word_count,
          chapter_count,
          is_complete,
          status,
          created_at,
          updated_at,
          published_at,
          hit_count,
          kudos_count,
          comment_count,
          bookmark_count,
          fandoms,
          characters,
          relationships,
          freeform_tags,
          warnings,
          category
        FROM works
        ORDER BY created_at DESC
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
        const cleanArray = (arr) => {
          if (!arr) return [];
          if (typeof arr === 'string') {
            // Handle stringified JSON arrays
            try {
              const parsed = JSON.parse(arr);
              if (Array.isArray(parsed)) return parsed.filter(item => item && String(item).trim());
              return [arr];
            } catch {
              return [arr];
            }
          }
          if (Array.isArray(arr)) return arr.filter(item => item && String(item).trim());
          return [];
        };

        // Map to unified schema
        const doc = {
          // Core fields
          id: work.id,
          title: work.title || '',
          summary: work.summary || '',
          notes: work.notes || '',
          user_id: work.user_id,
          
          // Tag fields (arrays) - unified schema
          fandoms: cleanArray(work.fandoms),
          characters: cleanArray(work.characters),
          relationships: cleanArray(work.relationships),
          freeform_tags: cleanArray(work.freeform_tags),
          warnings: cleanArray(work.warnings),
          categories: work.category ? [work.category] : [],
          
          // Metadata fields - unified schema
          rating: work.rating || 'Not Rated',
          language: work.language || 'English',
          status: this.mapStatus(work.is_complete, work.is_draft),
          word_count: parseInt(work.word_count) || 0,
          chapter_count: parseInt(work.chapter_count) || 1,
          is_complete: work.is_complete || false,
          
          // Statistics fields - unified schema
          hits_count: parseInt(work.hit_count) || 0,
          kudos_count: parseInt(work.kudos_count) || 0,
          comments_count: parseInt(work.comment_count) || 0,
          bookmarks_count: parseInt(work.bookmark_count) || 0,
          
          // Timestamp fields - unified schema
          published_at: work.published_at,
          updated_at: work.updated_at,
          created_at: work.created_at,
          
          // Search helper field
          content_text: [
            work.title, 
            work.summary, 
            work.notes,
            ...cleanArray(work.fandoms),
            ...cleanArray(work.characters),
            ...cleanArray(work.relationships),
            ...cleanArray(work.freeform_tags)
          ].filter(Boolean).join(' ')
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
          response.items.forEach((item, index) => {
            if (item.index?.error) {
              console.warn(`   Document ${index}: ${item.index.error.reason}`);
            }
          });
        }

        this.totalSynced += result.rows.length;
      }

    } catch (error) {
      console.error(`❌ Failed to sync batch ${offset}-${offset + limit}:`, error);
      throw error;
    }
  }

  mapStatus(isComplete, isDraft) {
    if (isDraft) return 'draft';
    if (isComplete) return 'complete';
    return 'posted';
  }

  async verifySync() {
    try {
      // Refresh index to make documents searchable
      await ES_CLIENT.indices.refresh({ index: 'works' });
      
      // Get document count
      const response = await ES_CLIENT.count({ index: 'works' });
      console.log(`\n✅ Verification: ${response.count} documents in Elasticsearch index`);
      
      // Test searches with unified schema
      const testSearches = [
        { query: 'Harry', field: 'title', description: 'Title search' },
        { query: 'Marvel Cinematic Universe', field: 'fandoms', description: 'Fandom search' },
        { query: 'Hermione Granger', field: 'characters', description: 'Character search' }
      ];

      for (const test of testSearches) {
        const searchResponse = await ES_CLIENT.search({
          index: 'works',
          body: {
            query: {
              match: {
                [test.field]: test.query
              }
            },
            size: 1
          }
        });

        console.log(`✅ ${test.description}: ${searchResponse.hits.total.value} results`);
        if (searchResponse.hits.hits.length > 0) {
          const sample = searchResponse.hits.hits[0]._source;
          console.log(`   Sample: "${sample.title}" (${sample.word_count} words)`);
        }
      }

    } catch (error) {
      console.error('❌ Verification failed:', error);
    }
  }
}

// Run the sync
const syncer = new UnifiedElasticsearchSyncer();
syncer.run().catch(error => {
  console.error('Sync process crashed:', error);
  process.exit(1);
});