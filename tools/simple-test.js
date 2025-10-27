#!/usr/bin/env node
const { Client } = require('pg');
const fs = require('fs');
const path = require('path');

const pgConfig = {
  host: 'localhost',
  port: 5432,
  database: 'ao3_nuclear',
  user: 'ao3_user',
  password: 'ao3_password'
};

async function simpleTest() {
  const client = new Client(pgConfig);
  await client.connect();
  
  // Load and prepare data
  const filePath = path.join(__dirname, '..', 'frontend', 'src', 'data', 'massive-ao3-dataset.json');
  const data = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  const firstWork = data.works[0];
  
  function mapStatus(status) {
    const statusMap = {
      'Complete': 'complete',
      'Work in Progress': 'published', 
      'Hiatus': 'hiatus',
      'Abandoned': 'abandoned'
    };
    return statusMap[status] || 'published';
  }
  
  console.log('📖 First work:', firstWork.title, 'by', firstWork.author);
  console.log('📝 Status mapping:', firstWork.status, '->', mapStatus(firstWork.status));
  
  let userId;
  
  try {
    // Try the exact same logic as the main script
    console.log('👤 Checking if user exists...');
    const existingUser = await client.query(
      'SELECT id FROM users WHERE username = $1',
      [firstWork.author]
    );
    
    if (existingUser.rows.length > 0) {
      userId = existingUser.rows[0].id;
      console.log(`✅ User found with ID: ${userId}`);
    } else {
      console.log('➕ Creating new user...');
      const newUser = await client.query(
        `INSERT INTO users (username, email, password_hash, created_at, updated_at) 
         VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id`,
        [firstWork.author, `${firstWork.author}@example.com`, 'hashedpassword']
      );
      userId = newUser.rows[0].id;
      console.log(`✅ Created user with ID: ${userId}`);
    }
    
    console.log('📖 Creating work...');
    const workResult = await client.query(
      `INSERT INTO works (
        user_id, title, summary, word_count, chapter_count, expected_chapters,
        rating, status, language, published_at, updated_at, created_at
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()) RETURNING id`,
      [
        userId,
        firstWork.title,
        firstWork.summary || '',
        firstWork.word_count,
        firstWork.chapter_count,
        firstWork.max_chapters,
        firstWork.rating,
        mapStatus(firstWork.status),
        firstWork.language,
        new Date(firstWork.published_date),
        new Date(firstWork.updated_date)
      ]
    );
    
    const workId = workResult.rows[0].id;
    console.log(`✅ Created work with ID: ${workId}`);
    console.log('🎉 Success! The flow works correctly.');
    
  } catch (error) {
    console.error('❌ Error:', error.message);
    console.error('❌ Code:', error.code);
    if (error.detail) console.error('❌ Detail:', error.detail);
    
    // Show which step failed
    if (error.message.includes('duplicate key')) {
      console.log('🔍 This is a duplicate key error - might be expected');
    } else {
      console.log('🔍 This is a different error - investigating...');
    }
  } finally {
    await client.end();
  }
}

simpleTest();