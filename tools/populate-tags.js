#!/usr/bin/env node

/**
 * Populate Tags Table from Existing Works
 * Extracts unique tags from works and populates the tags table
 */

const { Client } = require('pg');
const { v4: uuidv4 } = require('uuid');

const client = new Client({
  connectionString: process.env.DATABASE_URL || 'postgres://ao3_user:ao3_password@localhost/ao3_nuclear?sslmode=disable'
});

async function main() {
  try {
    await client.connect();
    console.log('🔗 Connected to PostgreSQL');

    // Get all unique tags from works with their types
    const query = `
      SELECT DISTINCT 
        t.name,
        t.tag_type,
        COUNT(wt.work_id) as use_count
      FROM (
        SELECT name, 
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
      ) t
      LEFT JOIN work_tags wt ON EXISTS (
        SELECT 1 FROM tags tag_inner WHERE tag_inner.name = t.name AND tag_inner.id = wt.tag_id
      )
      WHERE t.name IS NOT NULL AND t.name != ''
      GROUP BY t.name, t.tag_type
      ORDER BY use_count DESC, t.name ASC
    `;

    console.log('📊 Analyzing existing tags...');
    const result = await client.query(query);

    console.log(`📋 Found ${result.rows.length} unique tags to process`);

    if (result.rows.length === 0) {
      console.log('⚠️  No tags found. Creating sample tags...');
      
      // Create some sample tags
      const sampleTags = [
        { name: 'Harry Potter', type: 'character' },
        { name: 'Hermione Granger', type: 'character' },
        { name: 'Draco Malfoy', type: 'character' },
        { name: 'Harry Potter - J. K. Rowling', type: 'fandom' },
        { name: 'Marvel Cinematic Universe', type: 'fandom' },
        { name: 'Supernatural', type: 'fandom' },
        { name: 'Harry Potter/Hermione Granger', type: 'relationship' },
        { name: 'Harry Potter/Draco Malfoy', type: 'relationship' },
        { name: 'Angst', type: 'freeform' },
        { name: 'Hurt/Comfort', type: 'freeform' },
        { name: 'Fluff', type: 'freeform' },
        { name: 'Slow Burn', type: 'freeform' },
        { name: 'Mature', type: 'rating' },
        { name: 'Teen And Up Audiences', type: 'rating' },
        { name: 'General Audiences', type: 'rating' },
        { name: 'M/M', type: 'category' },
        { name: 'F/M', type: 'category' },
        { name: 'Gen', type: 'category' },
        { name: 'No Archive Warnings Apply', type: 'warning' }
      ];

      console.log('📝 Inserting sample tags...');
      
      for (const tag of sampleTags) {
        const tagId = uuidv4();
        await client.query(`
          INSERT INTO tags (id, name, canonical_name, type, description, is_canonical, is_filterable, use_count, created_at, updated_at)
          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
          ON CONFLICT (name) DO NOTHING
        `, [
          tagId,
          tag.name,
          null, // canonical_name must be NULL when is_canonical is true
          tag.type,
          null, // No description for now
          true, // Make canonical
          true, // Make filterable
          Math.floor(Math.random() * 100) + 10 // Random use count
        ]);
      }

      console.log(`✅ Inserted ${sampleTags.length} sample tags`);
    } else {
      // Clear existing tags first
      await client.query('DELETE FROM tags');
      console.log('🗑️  Cleared existing tags table');

      // Insert all found tags
      console.log('📝 Inserting tags...');
      
      let inserted = 0;
      for (const row of result.rows) {
        const tagId = uuidv4();
        await client.query(`
          INSERT INTO tags (id, name, canonical_name, type, description, is_canonical, is_filterable, use_count, created_at, updated_at)
          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
        `, [
          tagId,
          row.name,
          null, // canonical_name must be NULL when is_canonical is true
          row.tag_type,
          null, // No description for now
          true, // Make canonical
          true, // Make filterable
          parseInt(row.use_count) || 0
        ]);
        inserted++;

        if (inserted % 100 === 0) {
          console.log(`   Inserted ${inserted}/${result.rows.length} tags...`);
        }
      }

      console.log(`✅ Successfully inserted ${inserted} tags`);
    }

    // Verify results
    const countResult = await client.query('SELECT COUNT(*) as total FROM tags');
    const total = countResult.rows[0].total;
    
    console.log(`\n📊 Tag Table Summary:`);
    console.log(`   Total tags: ${total}`);

    // Show breakdown by type
    const breakdown = await client.query(`
      SELECT type, COUNT(*) as count 
      FROM tags 
      GROUP BY type 
      ORDER BY count DESC
    `);
    
    console.log(`\n📋 Tags by Type:`);
    for (const row of breakdown.rows) {
      console.log(`   ${row.type}: ${row.count} tags`);
    }

    console.log('\n🎉 Tag population completed successfully!');

  } catch (error) {
    console.error('❌ Error:', error);
    process.exit(1);
  } finally {
    await client.end();
  }
}

main();