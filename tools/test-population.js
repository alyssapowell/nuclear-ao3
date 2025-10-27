#!/usr/bin/env node
const { Client } = require('pg');

const pgConfig = {
  host: 'localhost',
  port: 5432,
  database: 'ao3_nuclear',
  user: 'ao3_user',
  password: 'ao3_password'
};

async function testPopulation() {
  const client = new Client(pgConfig);
  await client.connect();
  
  console.log('🧪 Testing user and work creation...');
  
  try {
    // Create a test user
    console.log('Creating test user...');
    const userResult = await client.query(
      `INSERT INTO users (username, email, password_hash, created_at, updated_at) 
       VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id`,
      ['testuser', 'test@example.com', 'hashedpassword']
    );
    
    const userId = userResult.rows[0].id;
    console.log(`✅ Created user with ID: ${userId}`);
    
    // Create a test work
    console.log('Creating test work...');
    const workResult = await client.query(
      `INSERT INTO works (
        user_id, title, summary, word_count, chapter_count, expected_chapters,
        rating, status, language, published_at, updated_at, created_at
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()) RETURNING id`,
      [
        userId,
        'Test Work',
        'This is a test summary',
        1000,
        1,
        1,
        'General Audiences',
        'published',
        'English',
        new Date(),
        new Date()
      ]
    );
    
    const workId = workResult.rows[0].id;
    console.log(`✅ Created work with ID: ${workId}`);
    
    console.log('🎉 Test successful!');
    
  } catch (error) {
    console.error('❌ Test failed:', error.message);
    console.error('Full error:', error);
  } finally {
    await client.end();
  }
}

testPopulation();