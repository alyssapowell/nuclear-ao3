#!/usr/bin/env node

const { Client } = require('pg');
const axios = require('axios');

const config = {
  database: {
    user: 'ao3_user',
    host: 'localhost',
    database: 'ao3_nuclear',
    password: 'ao3_password',
    port: 5432,
  }
};

async function associateTagsWithWorks() {
  const db = new Client(config.database);
  await db.connect();
  
  console.log('🏷️ Quick Tag Association - Adding Harry Potter tags to existing works...\n');
  
  try {
    // Get some works to tag
    const works = await db.query('SELECT id, title FROM works WHERE status = \'published\' LIMIT 10');
    console.log(`📚 Found ${works.rows.length} published works`);
    
    // Get Harry Potter related tags
    const harryTags = await db.query(`
      SELECT id, name, type FROM tags 
      WHERE name ILIKE '%harry%' OR name ILIKE '%potter%' OR name ILIKE '%draco%' OR name ILIKE '%hermione%'
      LIMIT 10
    `);
    console.log(`🏷️ Found ${harryTags.rows.length} Harry Potter tags`);
    
    // Associate first 5 works with Harry Potter tags
    for (let i = 0; i < Math.min(5, works.rows.length); i++) {
      const work = works.rows[i];
      console.log(`\n📖 Tagging work: "${work.title}"`);
      
      // Add 3-5 random Harry Potter tags to each work
      for (let j = 0; j < Math.min(4, harryTags.rows.length); j++) {
        const tag = harryTags.rows[j];
        
        try {
          await db.query(`
            INSERT INTO work_tags (work_id, tag_id, created_at)
            VALUES ($1, $2, NOW())
            ON CONFLICT (work_id, tag_id) DO NOTHING
          `, [work.id, tag.id]);
          
          console.log(`   ✅ Added tag: ${tag.name} (${tag.type})`);
        } catch (error) {
          console.log(`   ❌ Failed to add tag ${tag.name}: ${error.message}`);
        }
      }
    }
    
    // Update work titles to include Harry Potter content
    const titleUpdates = [
      'Harry Potter and the Search Engine Test',
      'Harry Potter: A Nuclear AO3 Adventure', 
      'When Harry Met the Elasticsearch Index',
      'Harry Potter and the Case of Missing Search Results',
      'The Harry Potter Search Chronicles'
    ];
    
    for (let i = 0; i < Math.min(5, works.rows.length, titleUpdates.length); i++) {
      const work = works.rows[i];
      const newTitle = titleUpdates[i];
      
      await db.query(`
        UPDATE works 
        SET title = $1, summary = $2, updated_at = NOW()
        WHERE id = $3
      `, [
        newTitle,
        `A test work featuring Harry Potter characters and themes, created for Nuclear AO3 search testing. Original title: ${work.title}`,
        work.id
      ]);
      
      console.log(`📝 Updated work title to: "${newTitle}"`);
    }
    
    console.log('\n🔄 Triggering Elasticsearch re-sync...');
    
    // Trigger re-sync of updated works
    try {
      const response = await axios.post('http://localhost:8080/api/v1/search/reindex');
      console.log('✅ Elasticsearch re-sync triggered successfully');
    } catch (error) {
      console.log('⚠️ Re-sync endpoint not available, manually syncing...');
      
      // Manual sync using the existing sync script
      const { exec } = require('child_process');
      exec('cd /Users/alyssapowell/Documents/projects/otwarchive/nuclear-ao3/tools && node unified-elasticsearch-sync.js', (error, stdout, stderr) => {
        if (error) {
          console.error('❌ Manual sync failed:', error);
        } else {
          console.log('✅ Manual sync completed');
          console.log(stdout);
        }
      });
    }
    
    console.log('\n🎉 Quick tag association complete!');
    console.log('🔍 Harry Potter works should now be searchable');
    
  } catch (error) {
    console.error('❌ Error:', error);
  } finally {
    await db.end();
  }
}

associateTagsWithWorks().catch(console.error);