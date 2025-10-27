#!/usr/bin/env node
/**
 * E2E Test Data Seeder
 * Creates consistent test data for end-to-end integration tests
 * This ensures our Playwright tests have predictable data to work with
 */

const { Client } = require('pg');

const pgConfig = {
  host: 'localhost',
  port: 5432,
  database: 'ao3_nuclear',
  user: 'ao3_user',
  password: 'ao3_password'
};

// Test user that matches our integration tests
const TEST_USER = {
  username: 'testuser30d_v2',
  email: 'testuser30d_v2@example.com',
  password_hash: '$2b$10$abcdefghijklmnopqrstuvwxyz1234567890ABCDEF', // Dummy hash
  is_verified: true
};

// Test data for consistent e2e testing
const TEST_DATA = {
  works: [
    {
      title: 'E2E Test Fixture Work 1',
      summary: 'This is a test work created for end-to-end testing. It has predictable content.',
      content: 'This is the content of our test work. It has multiple paragraphs.\n\nThis is the second paragraph to test formatting.\n\nAnd a third paragraph for good measure.',
      word_count: 42,
      chapter_count: 1,
      expected_chapters: 1,
      rating: 'General Audiences',
      status: 'published',
      language: 'English',
      fandom: 'Test Fandom',
      categories: ['Gen'],
      warnings: ['No Archive Warnings Apply'],
      characters: ['Test Character A', 'Test Character B'],
      relationships: [],
      freeform_tags: ['test-tag', 'fixture-data', 'e2e-testing']
    },
    {
      title: 'E2E Test Fixture Work 2',
      summary: 'Another test work for searching and filtering tests.',
      content: 'Different content for the second test work. This one has different tags and metadata.',
      word_count: 156,
      chapter_count: 2,
      expected_chapters: 3,
      rating: 'Teen And Up Audiences',
      status: 'draft',
      language: 'English',
      fandom: 'Test Fandom',
      categories: ['F/M'],
      warnings: ['No Archive Warnings Apply'],
      characters: ['Test Character C', 'Test Character D'],
      relationships: ['Test Character C/Test Character D'],
      freeform_tags: ['romance', 'slow-burn', 'work-in-progress']
    }
  ],
  series: [
    {
      title: 'E2E Test Series',
      summary: 'This is a test series for end-to-end testing.',
      notes: 'Test series notes for e2e validation.'
    }
  ],
  bookmarks: [
    {
      notes: 'Great test work! Bookmarked for e2e validation.',
      tags: ['test-bookmark', 'e2e-fixture'],
      private: false
    }
  ]
};

async function seedE2ETestData() {
  const client = new Client(pgConfig);
  await client.connect();
  
  console.log('🌱 Seeding E2E test data...');
  
  try {
    // Clean up existing test data first
    console.log('🧹 Cleaning up existing test data...');
    await client.query('DELETE FROM bookmarks WHERE user_id IN (SELECT id FROM users WHERE email = $1)', [TEST_USER.email]);
    await client.query('DELETE FROM series_works WHERE series_id IN (SELECT id FROM series WHERE user_id IN (SELECT id FROM users WHERE email = $1))', [TEST_USER.email]);
    await client.query('DELETE FROM series WHERE user_id IN (SELECT id FROM users WHERE email = $1)', [TEST_USER.email]);
    await client.query('DELETE FROM work_tags WHERE work_id IN (SELECT id FROM works WHERE user_id IN (SELECT id FROM users WHERE email = $1))', [TEST_USER.email]);
    await client.query('DELETE FROM works WHERE user_id IN (SELECT id FROM users WHERE email = $1)', [TEST_USER.email]);
    await client.query('DELETE FROM users WHERE email = $1', [TEST_USER.email]);
    
    // Create test user
    console.log('👤 Creating test user...');
    const userResult = await client.query(
      `INSERT INTO users (username, email, password_hash, is_verified, created_at, updated_at) 
       VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id`,
      [TEST_USER.username, TEST_USER.email, TEST_USER.password_hash, TEST_USER.is_verified]
    );
    
    const userId = userResult.rows[0].id;
    console.log(`✅ Created test user with ID: ${userId}`);
    
    // Create test works
    console.log('📚 Creating test works...');
    const workIds = [];
    
    for (let i = 0; i < TEST_DATA.works.length; i++) {
      const work = TEST_DATA.works[i];
      
      const workResult = await client.query(
        `INSERT INTO works (
          user_id, title, summary, word_count, chapter_count, expected_chapters,
          rating, status, language, is_draft, published_at, updated_at, created_at,
          fandoms, characters, relationships, freeform_tags
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), $13, $14, $15, $16) RETURNING id`,
        [
          userId,
          work.title,
          work.summary,
          work.word_count,
          work.chapter_count,
          work.expected_chapters,
          work.rating,
          work.status,
          work.language,
          work.status !== 'published',
          work.status === 'published' ? new Date() : null,
          new Date(),
          [work.fandom],
          work.characters,
          work.relationships,
          work.freeform_tags
        ]
      );
      
      const workId = workResult.rows[0].id;
      workIds.push(workId);
      console.log(`✅ Created work "${work.title}" with ID: ${workId}`);
      
      // Create chapter for this work
      await client.query(
        `INSERT INTO chapters (work_id, title, content, chapter_number, word_count, is_draft, published_at, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
        [
          workId,
          `Chapter 1`,
          work.content,
          1,
          work.word_count,
          work.status !== 'published',
          work.status === 'published' ? new Date() : null
        ]
      );
      console.log(`✅ Created chapter for work "${work.title}"`);
      
      // Tags are now stored directly in the works table arrays, so no separate tag linking needed
    }
    
    // Create test series
    console.log('📖 Creating test series...');
    const seriesResult = await client.query(
      `INSERT INTO series (user_id, title, description, notes, created_at, updated_at) 
       VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id`,
      [userId, TEST_DATA.series[0].title, TEST_DATA.series[0].summary, TEST_DATA.series[0].notes]
    );
    
    const seriesId = seriesResult.rows[0].id;
    console.log(`✅ Created test series with ID: ${seriesId}`);
    
    // Add first work to series
    if (workIds.length > 0) {
      await client.query(
        `INSERT INTO series_works (series_id, work_id, position, created_at) 
         VALUES ($1, $2, 1, NOW())`,
        [seriesId, workIds[0]]
      );
      console.log(`✅ Added work to series`);
    }
    
    // Create test bookmark
    if (workIds.length > 1) {
      console.log('🔖 Creating test bookmark...');
      const bookmark = TEST_DATA.bookmarks[0];
      
      const bookmarkResult = await client.query(
        `INSERT INTO bookmarks (user_id, work_id, notes, is_private, tags, created_at, updated_at) 
         VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id`,
        [userId, workIds[1], bookmark.notes, bookmark.private, bookmark.tags]
      );
      
      const bookmarkId = bookmarkResult.rows[0].id;
      console.log(`✅ Created test bookmark with ID: ${bookmarkId}`);
    }
    
    console.log('🎉 E2E test data seeding completed successfully!');
    console.log('');
    console.log('Test user credentials:');
    console.log(`  Email: ${TEST_USER.email}`);
    console.log(`  Username: ${TEST_USER.username}`);
    console.log(`  Password: TestPassword123! (for testing)`);
    console.log('');
    console.log('Test data includes:');
    console.log(`  - ${TEST_DATA.works.length} test works`);
    console.log(`  - ${TEST_DATA.series.length} test series`);
    console.log(`  - ${TEST_DATA.bookmarks.length} test bookmarks`);
    console.log('  - Various tags and metadata');
    
  } catch (error) {
    console.error('❌ E2E test data seeding failed:', error.message);
    console.error('Full error:', error);
    process.exit(1);
  } finally {
    await client.end();
  }
}

function getTagType(tagName, work) {
  if (tagName === work.fandom) return 'Fandom';
  if (work.categories.includes(tagName)) return 'Category';
  if (work.warnings.includes(tagName)) return 'Warning';
  if (work.characters.includes(tagName)) return 'Character';
  if (work.relationships.includes(tagName)) return 'Relationship';
  return 'Freeform';
}

// Run if called directly
if (require.main === module) {
  seedE2ETestData();
}

module.exports = { seedE2ETestData, TEST_USER };