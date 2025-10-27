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

async function debugPopulation() {
  const client = new Client(pgConfig);
  await client.connect();
  
  console.log('🧪 Loading dataset...');
  const filePath = path.join(__dirname, '..', 'frontend', 'src', 'data', 'massive-ao3-dataset.json');
  const data = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  const works = data.works || data;
  console.log(`✅ Loaded ${works.length} works`);
  
  const firstWork = works[0];
  console.log('🔍 First work:', {
    title: firstWork.title,
    author: firstWork.author,
    status: firstWork.status,
    rating: firstWork.rating,
    language: firstWork.language
  });
  
  // Map status
  function mapStatus(status) {
    const statusMap = {
      'Complete': 'complete',
      'Work in Progress': 'published',
      'Hiatus': 'hiatus',
      'Abandoned': 'abandoned'
    };
    return statusMap[status] || 'published';
  }
  
  console.log(`📝 Status mapped: "${firstWork.status}" -> "${mapStatus(firstWork.status)}"`);
  
  try {
    console.log('👤 Creating user...');
    const userResult = await client.query(
      `INSERT INTO users (username, email, password_hash, created_at, updated_at) 
       VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id`,
      [firstWork.author, `${firstWork.author}@example.com`, 'hashedpassword']
    );
    
    const userId = userResult.rows[0].id;
    console.log(`✅ Created user with ID: ${userId}`);
    
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
    
    console.log('🎉 Success! The data structure is correct.');
    
  } catch (error) {
    console.error('❌ Error:', error.message);
    console.error('❌ Code:', error.code);
    console.error('❌ Detail:', error.detail);
  } finally {
    await client.end();
  }
}

debugPopulation();