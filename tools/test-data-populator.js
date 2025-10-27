#!/usr/bin/env node

const { Client } = require('pg');

const config = {
  database: {
    user: 'ao3_user',
    host: 'localhost',
    database: 'ao3_nuclear',
    password: 'ao3_password',
    port: 5432,
  }
};

class TestDataPopulator {
  constructor() {
    this.db = new Client(config.database);
    this.stats = {
      users: 0,
      works: 0,
      chapters: 0,
      comments: 0,
      kudos: 0,
      bookmarks: 0,
      errors: []
    };
    
    this.cache = {
      users: [],
      tags: {}
    };
  }

  async initialize() {
    console.log('🧪 Nuclear AO3 - Test Data Populator (100 works)...\n');
    await this.db.connect();
    console.log('✅ Connected to database');
    
    // Load existing data
    const users = await this.db.query('SELECT id, username FROM users ORDER BY created_at DESC LIMIT 500');
    this.cache.users = users.rows;
    console.log(`📋 Loaded ${users.rows.length} existing users`);
    
    const tags = await this.db.query('SELECT id, name, type FROM tags');
    tags.rows.forEach(tag => {
      if (!this.cache.tags[tag.type]) this.cache.tags[tag.type] = [];
      this.cache.tags[tag.type].push(tag);
    });
    console.log(`🏷️ Loaded ${tags.rows.length} existing tags`);
  }

  randomChoice(arr) {
    return arr[Math.floor(Math.random() * arr.length)];
  }

  randomChoices(arr, min = 1, max = 5) {
    const count = Math.floor(Math.random() * (max - min + 1)) + min;
    const shuffled = [...arr].sort(() => 0.5 - Math.random());
    return shuffled.slice(0, Math.min(count, arr.length));
  }

  generateTitle() {
    const templates = [
      "The {adjective} {noun}",
      "{character}'s {noun}",
      "When {character} Falls",
      "{noun} and {noun}",
      "A {adjective} {noun}",
      "The {noun} Chronicles"
    ];
    
    const adjectives = ["Dark", "Hidden", "Secret", "Lost", "Golden", "Silver", "Broken", "Ancient"];
    const nouns = ["Heart", "Soul", "Dream", "Memory", "Promise", "Journey", "Story", "Love"];
    
    const template = this.randomChoice(templates);
    return template
      .replace(/{adjective}/g, this.randomChoice(adjectives))
      .replace(/{noun}/g, this.randomChoice(nouns))
      .replace(/{character}/g, "Alex");
  }

  generateWordCount() {
    const ranges = [
      { min: 500, max: 2000, weight: 40 },   // Short
      { min: 2000, max: 10000, weight: 35 }, // Medium
      { min: 10000, max: 50000, weight: 20 }, // Long
      { min: 50000, max: 200000, weight: 5 }  // Epic
    ];
    
    const totalWeight = ranges.reduce((sum, r) => sum + r.weight, 0);
    let random = Math.random() * totalWeight;
    
    for (const range of ranges) {
      random -= range.weight;
      if (random <= 0) {
        return Math.floor(Math.random() * (range.max - range.min + 1)) + range.min;
      }
    }
    return 1000; // fallback
  }

  generateChapterCount(wordCount) {
    if (wordCount < 2000) return 1;
    const avgWordsPerChapter = 2000 + Math.random() * 3000;
    return Math.max(1, Math.ceil(wordCount / avgWordsPerChapter));
  }

  selectRandomTags(type, count = { min: 0, max: 3 }) {
    const tags = this.cache.tags[type] || [];
    if (tags.length === 0) return [];
    
    const numTags = Math.floor(Math.random() * (count.max - count.min + 1)) + count.min;
    return this.randomChoices(tags, 0, Math.min(numTags, tags.length));
  }

  async createWork() {
    try {
      const user = this.randomChoice(this.cache.users);
      const wordCount = this.generateWordCount();
      const chapterCount = this.generateChapterCount(wordCount);
      const isComplete = Math.random() > (chapterCount === 1 ? 0.05 : 0.6);
      
      const title = this.generateTitle();
      const summary = Math.random() > 0.3 ? "A compelling story about love, adventure, and self-discovery." : null;
      
      const ratings = ['General Audiences', 'Teen And Up Audiences', 'Mature', 'Explicit'];
      const categories = ['Gen', 'M/M', 'F/M', 'F/F', 'Multi'];
      const languages = ['English', 'Spanish', 'French'];
      
      const publishedAt = new Date(2020 + Math.random() * 5, Math.floor(Math.random() * 12), Math.floor(Math.random() * 28) + 1);
      const updatedAt = new Date(publishedAt.getTime() + Math.random() * (new Date() - publishedAt));
      
      // Create work
      const workResult = await this.db.query(`
        INSERT INTO works (
          title, summary, user_id, word_count, chapter_count, 
          is_complete, status, rating, category, language,
          published_at, updated_at, created_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
        RETURNING id
      `, [
        title,
        summary,
        user.id,
        wordCount,
        chapterCount,
        isComplete,
        'published',
        this.randomChoice(ratings),
        this.randomChoice(categories),
        this.randomChoice(languages),
        publishedAt,
        updatedAt,
        publishedAt
      ]);
      
      const workId = workResult.rows[0].id;
      
      // Add tags
      const fandoms = this.selectRandomTags('fandom', { min: 1, max: 2 });
      const characters = this.selectRandomTags('character', { min: 0, max: 4 });
      const relationships = this.selectRandomTags('relationship', { min: 0, max: 2 });
      const freeformTags = this.selectRandomTags('freeform', { min: 0, max: 6 });
      
      const allTags = [...fandoms, ...characters, ...relationships, ...freeformTags];
      for (const tag of allTags) {
        await this.db.query(`
          INSERT INTO work_tags (work_id, tag_id, created_at)
          VALUES ($1, $2, $3)
          ON CONFLICT (work_id, tag_id) DO NOTHING
        `, [workId, tag.id, publishedAt]);
      }
      
      // Create chapters
      for (let i = 1; i <= chapterCount; i++) {
        const chapterWordCount = Math.floor(wordCount / chapterCount);
        const chapterContent = `This is chapter ${i}. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.`;
        
        await this.db.query(`
          INSERT INTO chapters (
            work_id, chapter_number, title, content, word_count, 
            created_at, updated_at
          )
          VALUES ($1, $2, $3, $4, $5, $6, $7)
        `, [
          workId,
          i,
          `Chapter ${i}`,
          chapterContent,
          chapterWordCount,
          publishedAt,
          i === chapterCount ? updatedAt : publishedAt
        ]);
        
        this.stats.chapters++;
      }
      
      // Generate some engagement
      const popularityFactor = Math.random() * Math.random(); // Exponential distribution
      
      // Kudos
      const kudosCount = Math.floor(popularityFactor * 50);
      const kudosUsers = this.randomChoices(this.cache.users, 0, Math.min(kudosCount, this.cache.users.length));
      
      for (const kudosUser of kudosUsers) {
        try {
          await this.db.query(`
            INSERT INTO kudos (work_id, user_id, created_at)
            VALUES ($1, $2, $3)
            ON CONFLICT (work_id, user_id) DO NOTHING
          `, [workId, kudosUser.id, new Date(publishedAt.getTime() + Math.random() * (new Date() - publishedAt))]);
          this.stats.kudos++;
        } catch (error) {
          // Ignore duplicates
        }
      }
      
      // Comments
      const commentCount = Math.floor(popularityFactor * popularityFactor * 10);
      for (let i = 0; i < commentCount; i++) {
        try {
          const commentUser = this.randomChoice(this.cache.users);
          const comments = ["Great story!", "Love this!", "Please update!", "Amazing work!", "Can't wait for more!"];
          
          await this.db.query(`
            INSERT INTO comments (work_id, user_id, content, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5)
          `, [
            workId,
            commentUser.id,
            this.randomChoice(comments),
            new Date(publishedAt.getTime() + Math.random() * (new Date() - publishedAt)),
            new Date(publishedAt.getTime() + Math.random() * (new Date() - publishedAt))
          ]);
          this.stats.comments++;
        } catch (error) {
          // Ignore errors
        }
      }
      
      // Bookmarks
      if (Math.random() < popularityFactor * 0.3) {
        try {
          const bookmarkUser = this.randomChoice(this.cache.users);
          await this.db.query(`
            INSERT INTO bookmarks (work_id, user_id, private, created_at)
            VALUES ($1, $2, $3, $4)
          `, [
            workId,
            bookmarkUser.id,
            Math.random() < 0.2,
            new Date(publishedAt.getTime() + Math.random() * (new Date() - publishedAt))
          ]);
          this.stats.bookmarks++;
        } catch (error) {
          // Ignore errors
        }
      }
      
      this.stats.works++;
      
    } catch (error) {
      this.stats.errors.push(`Work creation: ${error.message}`);
    }
  }

  async createWorks(count = 100) {
    console.log(`\n📚 Creating ${count} test works...`);
    
    for (let i = 0; i < count; i++) {
      await this.createWork();
      
      if ((i + 1) % 25 === 0) {
        console.log(`   📚 Created ${i + 1}/${count} works (${this.stats.chapters} chapters, ${this.stats.kudos} kudos, ${this.stats.comments} comments)`);
      }
    }
    
    console.log(`   ✅ Created ${this.stats.works} works total`);
  }

  async generateReport() {
    console.log('\n📊 Test Data Generation Report');
    console.log('===============================');
    
    const counts = await this.db.query(`
      SELECT 
        (SELECT COUNT(*) FROM works) as works,
        (SELECT COUNT(*) FROM chapters) as chapters,
        (SELECT COUNT(*) FROM kudos) as kudos,
        (SELECT COUNT(*) FROM comments) as comments,
        (SELECT COUNT(*) FROM bookmarks) as bookmarks
    `);
    
    const stats = counts.rows[0];
    console.log(`📚 Total Works: ${stats.works}`);
    console.log(`📄 Total Chapters: ${stats.chapters}`);
    console.log(`💖 Total Kudos: ${stats.kudos}`);
    console.log(`💬 Total Comments: ${stats.comments}`);
    console.log(`🔖 Total Bookmarks: ${stats.bookmarks}`);
    
    if (this.stats.errors.length > 0) {
      console.log(`\n⚠️ Errors: ${this.stats.errors.length}`);
      this.stats.errors.slice(0, 3).forEach(error => console.log(`   - ${error}`));
    }
  }

  async cleanup() {
    await this.db.end();
    console.log('\n✅ Test data generation complete!');
  }
}

async function main() {
  const populator = new TestDataPopulator();
  
  try {
    await populator.initialize();
    await populator.createWorks(100);
    await populator.generateReport();
  } catch (error) {
    console.error('❌ Test failed:', error);
  } finally {
    await populator.cleanup();
  }
}

main().catch(console.error);