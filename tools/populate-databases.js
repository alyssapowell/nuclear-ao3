#!/usr/bin/env node
/**
 * Database Population Script
 * Loads the comprehensive fanfiction dataset into PostgreSQL. Search reads
 * the works table directly, so there's no separate index to fill.
 */

const fs = require('fs');
const path = require('path');
const { Client } = require('pg');

// Database configuration
const pgConfig = {
  host: 'localhost',
  port: 5432,
  database: 'ao3_nuclear',
  user: 'ao3_user',
  password: 'ao3_password'
};

class DatabasePopulator {
  constructor() {
    this.pgClient = null;
    this.works = [];
    this.userIds = new Map(); // Cache for user IDs
    this.tagIds = new Map();   // Cache for tag IDs
    this.batchSize = 100;
  }

  async connect() {
    console.log('🔌 Connecting to PostgreSQL...');
    this.pgClient = new Client(pgConfig);
    try {
      await this.pgClient.connect();
      console.log('✅ Connected to PostgreSQL');
    } catch (error) {
      console.error('❌ Failed to connect to PostgreSQL:', error.message);
      throw error;
    }
  }

  async disconnect() {
    if (this.pgClient) {
      await this.pgClient.end();
      console.log('📴 Disconnected from PostgreSQL');
    }
  }

  async loadDataset(filename = 'massive-ao3-dataset.json') {
    const filePath = path.join(__dirname, '..', 'frontend', 'src', 'data', filename);
    
    if (!fs.existsSync(filePath)) {
      throw new Error(`Dataset file not found: ${filePath}`);
    }

    console.log('📖 Loading dataset...');
    const data = JSON.parse(fs.readFileSync(filePath, 'utf8'));
    this.works = data.works || data;
    console.log(`✅ Loaded ${this.works.length.toLocaleString()} works from dataset`);
  }

  async clearExistingData() {
    console.log('🧹 Clearing existing data...');
    
    try {
      // Clear PostgreSQL tables in dependency order
      const tables = [
        'work_tags', 'work_statistics', 'works', 
        'tags', 'users'
      ];
      
      for (const table of tables) {
        try {
          await this.pgClient.query(`TRUNCATE TABLE ${table} RESTART IDENTITY CASCADE`);
          console.log(`  ✅ Cleared ${table}`);
        } catch (error) {
          // Table might not exist, continue
          console.log(`  ⚠️  Could not clear ${table}: ${error.message}`);
        }
      }

    } catch (error) {
      console.error('❌ Error clearing data:', error.message);
    }
  }

  async getOrCreateUser(authorName) {
    if (this.userIds.has(authorName)) {
      return this.userIds.get(authorName);
    }

    try {
      // Check if user exists
      const existingUser = await this.pgClient.query(
        'SELECT id FROM users WHERE username = $1',
        [authorName]
      );

      if (existingUser.rows.length > 0) {
        const userId = existingUser.rows[0].id;
        this.userIds.set(authorName, userId);
        return userId;
      }

      // Create new user
      const newUser = await this.pgClient.query(
        `INSERT INTO users (username, email, password_hash, created_at, updated_at) 
         VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id`,
        [authorName, `${authorName.toLowerCase()}@example.com`, 'hashed_password']
      );

      const userId = newUser.rows[0].id;
      this.userIds.set(authorName, userId);
      
      // Skip user profile creation - table doesn't exist in this schema

      return userId;
    } catch (error) {
      console.error(`Error creating user ${authorName}:`, error.message);
      throw error;
    }
  }

  async getOrCreateTag(tagName, category) {
    const cacheKey = `${tagName}:${category}`;
    if (this.tagIds.has(cacheKey)) {
      return this.tagIds.get(cacheKey);
    }

    try {
      // Check if tag exists
      const existingTag = await this.pgClient.query(
        'SELECT id FROM tags WHERE name = $1 AND type = $2',
        [tagName, category]
      );

      if (existingTag.rows.length > 0) {
        const tagId = existingTag.rows[0].id;
        this.tagIds.set(cacheKey, tagId);
        return tagId;
      }

      // Create new tag
      const newTag = await this.pgClient.query(
        `INSERT INTO tags (name, type, is_canonical) 
         VALUES ($1, $2, $3) RETURNING id`,
        [tagName, category, true]
      );

      const tagId = newTag.rows[0].id;
      this.tagIds.set(cacheKey, tagId);
      return tagId;
    } catch (error) {
      console.error(`Error creating tag ${tagName}:`, error.message);
      throw error;
    }
  }

  mapStatus(status) {
    const statusMap = {
      'Complete': 'complete',
      'Work in Progress': 'published',
      'Hiatus': 'hiatus',
      'Abandoned': 'abandoned'
    };
    return statusMap[status] || 'published';
  }

  async insertWork(work) {
    try {
      // Get or create user
      const userId = await this.getOrCreateUser(work.author);

      // Insert work
      const workResult = await this.pgClient.query(
        `INSERT INTO works (
          user_id, title, summary, word_count, chapter_count, expected_chapters,
          rating, status, language, published_at, updated_at, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()) RETURNING id`,
        [
          userId,
          work.title,
          work.summary || '',
          work.word_count,
          work.chapter_count,
          work.max_chapters,
          work.rating,
          this.mapStatus(work.status),
          work.language,
          work.published_date,
          work.updated_date
        ]
      );

      const workId = workResult.rows[0].id;

      // Insert work statistics
      await this.pgClient.query(
        `INSERT INTO work_statistics (
          work_id, kudos_count, comment_count, bookmark_count, hit_count
        ) VALUES ($1, $2, $3, $4, $5)`,
        [
          workId,
          work.kudos_count || 0,
          work.comment_count || 0,
          work.bookmark_count || 0,
          work.hit_count || 0
        ]
      );

      // Insert tags
      const allTags = [
        ...work.fandoms.map(tag => ({ ...tag, category: 'fandom' })),
        ...work.characters.map(tag => ({ ...tag, category: 'character' })),
        ...work.relationships.map(tag => ({ ...tag, category: 'relationship' })),
        ...work.freeform_tags.map(tag => ({ ...tag, category: 'freeform' }))
      ];

      for (const tag of allTags) {
        const tagId = await this.getOrCreateTag(tag.name, tag.category);
        
        await this.pgClient.query(
          `INSERT INTO work_tags (work_id, tag_id, created_at) VALUES ($1, $2, NOW())
           ON CONFLICT (work_id, tag_id) DO NOTHING`,
          [workId, tagId]
        );
      }

      return workId;
    } catch (error) {
      // Skip duplicate or constraint errors and continue
      if (error.message.includes('duplicate key')) {
        console.log(`⚠️  Skipping duplicate work: "${work.title}"`);
        return null;
      }
      console.error(`❌ Error inserting work "${work.title}":`, error.message);
      console.error(`❌ Full error:`, error);
      return null; // Don't throw, just skip
    }
  }

  async populateDatabases() {
    console.log('🚀 Starting database population...');
    const startTime = Date.now();

    let processed = 0;
    const total = this.works.length;

    for (const work of this.works) {
      // Insert to PostgreSQL
      const workId = await this.insertWork(work);
      
      if (workId) {
        processed++;
      }

      // Progress reporting
      if (processed % 500 === 0) {
        const elapsed = (Date.now() - startTime) / 1000;
        const rate = processed / elapsed;
        const eta = (total - processed) / rate;
        console.log(`📊 Processed ${processed.toLocaleString()}/${total.toLocaleString()} works (${(processed/total*100).toFixed(1)}%) - ${rate.toFixed(1)} works/sec - ETA: ${eta.toFixed(0)}s`);
      }
    }

    const elapsed = (Date.now() - startTime) / 1000;
    console.log(`✅ Population complete! Processed ${processed.toLocaleString()} works in ${elapsed.toFixed(1)} seconds`);
  }

  async run() {
    try {
      await this.loadDataset();
      await this.connect();
      await this.clearExistingData();
      await this.populateDatabases();
      
      console.log('\n📊 Final Statistics:');
      
      // PostgreSQL stats
      const pgStats = await this.pgClient.query(`
        SELECT 
          (SELECT COUNT(*) FROM users) as users,
          (SELECT COUNT(*) FROM works) as works,
          (SELECT COUNT(*) FROM tags) as tags,
          (SELECT COUNT(*) FROM work_tags) as work_tags
      `);
      
      console.log('PostgreSQL:');
      console.log(`  Users: ${pgStats.rows[0].users.toLocaleString()}`);
      console.log(`  Works: ${pgStats.rows[0].works.toLocaleString()}`);
      console.log(`  Tags: ${pgStats.rows[0].tags.toLocaleString()}`);
      console.log(`  Work-Tag relationships: ${pgStats.rows[0].work_tags.toLocaleString()}`);

    } catch (error) {
      console.error('❌ Population failed:', error.message);
      process.exit(1);
    } finally {
      await this.disconnect();
    }
  }
}

// Main execution
if (require.main === module) {
  const populator = new DatabasePopulator();
  populator.run().then(() => {
    console.log('🎉 Database population completed successfully!');
    process.exit(0);
  }).catch(error => {
    console.error('💥 Database population failed:', error);
    process.exit(1);
  });
}

module.exports = DatabasePopulator;