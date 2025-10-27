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

class RobustDataPopulator {
  constructor() {
    this.db = new Client(config.database);
    this.stats = {
      users: 0,
      works: 0,
      chapters: 0,
      collections: 0,
      series: 0,
      comments: 0,
      kudos: 0,
      bookmarks: 0,
      workTags: 0,
      seriesWorks: 0,
      collectionWorks: 0,
      errors: []
    };
    
    // Cache for created entities
    this.cache = {
      users: [],
      tags: {},
      collections: [],
      series: [],
      works: []
    };
    
    // Realistic distributions and data
    this.distributions = {
      ratings: [
        { value: 'General Audiences', weight: 35 },
        { value: 'Teen And Up Audiences', weight: 40 },
        { value: 'Mature', weight: 20 },
        { value: 'Explicit', weight: 5 }
      ],
      categories: [
        { value: 'Gen', weight: 30 },
        { value: 'M/M', weight: 35 },
        { value: 'F/M', weight: 20 },
        { value: 'F/F', weight: 10 },
        { value: 'Multi', weight: 3 },
        { value: 'Other', weight: 2 }
      ],
      warnings: [
        { value: 'No Archive Warnings Apply', weight: 70 },
        { value: 'Creator Chose Not To Use Archive Warnings', weight: 20 },
        { value: 'Graphic Depictions Of Violence', weight: 5 },
        { value: 'Major Character Death', weight: 3 },
        { value: 'Underage', weight: 1 },
        { value: 'Rape/Non-Con', weight: 1 }
      ],
      languages: [
        { value: 'English', weight: 85 },
        { value: 'Spanish', weight: 5 },
        { value: 'French', weight: 3 },
        { value: 'German', weight: 2 },
        { value: 'Italian', weight: 2 },
        { value: 'Portuguese', weight: 1 },
        { value: 'Russian', weight: 1 },
        { value: 'Chinese', weight: 1 }
      ],
      wordCounts: [
        { min: 100, max: 1000, weight: 25 },      // Drabbles/ficlets
        { min: 1000, max: 5000, weight: 40 },     // Short stories
        { min: 5000, max: 20000, weight: 25 },    // Medium stories
        { min: 20000, max: 100000, weight: 8 },   // Long stories
        { min: 100000, max: 500000, weight: 2 }   // Epic stories
      ],
      chapterCounts: [
        { count: 1, weight: 60 },       // One-shots
        { count: [2, 5], weight: 25 },  // Short multi-chapter
        { count: [6, 20], weight: 12 }, // Medium multi-chapter
        { count: [21, 50], weight: 2 }, // Long multi-chapter
        { count: [51, 100], weight: 1 } // Epic multi-chapter
      ]
    };

    // Title templates for realistic fanfiction titles
    this.titleTemplates = [
      "{adjective} {noun}",
      "The {adjective} {noun}",
      "A {adjective} {noun}",
      "{character}'s {noun}",
      "When {character} {verb}",
      "{noun} and {noun}",
      "The {noun} of {character}",
      "{adjective} {noun}s",
      "{character} and the {adjective} {noun}",
      "How {character} {verb}",
      "{noun} in {place}",
      "Five Times {character} {verb}",
      "The {noun} Chronicles",
      "Beyond the {noun}",
      "{adjective} Hearts",
      "Midnight {noun}",
      "Dancing with {noun}s",
      "The Secret of {noun}",
      "{adjective} Promises"
    ];

    this.adjectives = [
      "Dark", "Hidden", "Secret", "Lost", "Forgotten", "Ancient", "Mysterious", "Broken",
      "Golden", "Silver", "Crimson", "Emerald", "Midnight", "Silent", "Whispered",
      "Forbidden", "Stolen", "Endless", "Eternal", "Sacred", "Wild", "Fierce",
      "Gentle", "Tender", "Brave", "Bold", "Quiet", "Soft", "Strong", "Powerful"
    ];

    this.nouns = [
      "Heart", "Soul", "Mind", "Dream", "Memory", "Shadow", "Light", "Magic",
      "Power", "Truth", "Secret", "Mystery", "Promise", "Wish", "Hope", "Fear",
      "Love", "Kiss", "Touch", "Dance", "Song", "Story", "Tale", "Journey",
      "Path", "Way", "Road", "Bridge", "Door", "Key", "Lock", "Chain"
    ];

    this.verbs = [
      "falls", "rises", "dances", "sings", "whispers", "dreams", "remembers",
      "forgets", "discovers", "finds", "loses", "saves", "protects", "fights",
      "loves", "hates", "fears", "hopes", "wishes", "believes", "trusts"
    ];

    this.places = [
      "Darkness", "Light", "the Shadows", "the City", "the Forest", "the Mountains",
      "the Sea", "the Sky", "Time", "Space", "Dreams", "Memory", "the Past",
      "the Future", "Another World", "the Library", "the Garden", "the Tower"
    ];

    this.usernamePrefixes = [
      "Dark", "Shadow", "Moon", "Star", "Sun", "Fire", "Ice", "Storm", "Wind",
      "Magic", "Mystic", "Wild", "Free", "Lost", "Found", "Silver", "Golden",
      "Crimson", "Azure", "Raven", "Phoenix", "Dragon", "Wolf", "Rose", "Lily"
    ];

    this.usernameSuffixes = [
      "Writer", "Dreamer", "Weaver", "Keeper", "Hunter", "Walker", "Rider",
      "Singer", "Dancer", "Mage", "Witch", "Wizard", "Author", "Scribe",
      "Poet", "Bard", "Storyteller", "Heart", "Soul", "Spirit", "Wings"
    ];
  }

  async initialize() {
    console.log('🏗️ Nuclear AO3 - Robust Data Populator Starting...\n');
    console.log('🎯 Target: 15,000 works with comprehensive metadata\n');
    await this.db.connect();
    console.log('✅ Connected to database');
    
    // Load existing data into cache
    await this.loadExistingData();
  }

  async loadExistingData() {
    console.log('📋 Loading existing data into cache...');
    
    // Load users
    const users = await this.db.query('SELECT id, username FROM users ORDER BY created_at DESC');
    this.cache.users = users.rows;
    console.log(`   👥 Loaded ${users.rows.length} existing users`);
    
    // Load tags by type
    const tags = await this.db.query('SELECT id, name, type FROM tags');
    tags.rows.forEach(tag => {
      if (!this.cache.tags[tag.type]) this.cache.tags[tag.type] = [];
      this.cache.tags[tag.type].push(tag);
    });
    console.log(`   🏷️ Loaded ${tags.rows.length} existing tags`);
    
    // Load collections and series
    const collections = await this.db.query('SELECT id, title, user_id FROM collections');
    this.cache.collections = collections.rows;
    
    const series = await this.db.query('SELECT id, title, user_id FROM series');
    this.cache.series = series.rows;
    
    console.log(`   📁 Loaded ${collections.rows.length} collections, ${series.rows.length} series`);
  }

  // Utility functions
  randomChoice(arr) {
    return arr[Math.floor(Math.random() * arr.length)];
  }

  randomChoices(arr, min = 1, max = 5) {
    const count = Math.floor(Math.random() * (max - min + 1)) + min;
    const shuffled = [...arr].sort(() => 0.5 - Math.random());
    return shuffled.slice(0, Math.min(count, arr.length));
  }

  selectWeighted(items) {
    const totalWeight = items.reduce((sum, item) => sum + item.weight, 0);
    let random = Math.random() * totalWeight;
    
    for (const item of items) {
      random -= item.weight;
      if (random <= 0) return item.value ? item.value : item;
    }
    return items[0].value ? items[0].value : items[0];
  }

  generateUsername() {
    const patterns = [
      () => this.randomChoice(this.usernamePrefixes) + this.randomChoice(this.usernameSuffixes),
      () => this.randomChoice(this.usernamePrefixes) + "_" + this.randomChoice(this.usernameSuffixes),
      () => this.randomChoice(this.adjectives) + this.randomChoice(this.nouns),
      () => this.randomChoice(this.nouns) + Math.floor(Math.random() * 9999),
      () => "x" + this.randomChoice(this.adjectives) + this.randomChoice(this.nouns) + "x",
      () => this.randomChoice(this.nouns).toLowerCase() + "_" + this.randomChoice(this.verbs).toLowerCase()
    ];
    return this.randomChoice(patterns)().toLowerCase().replace(/[^a-z0-9_]/g, '');
  }

  generateWordCount() {
    const range = this.selectWeighted(this.distributions.wordCounts);
    return Math.floor(Math.random() * (range.max - range.min + 1)) + range.min;
  }

  generateChapterCount() {
    const item = this.selectWeighted(this.distributions.chapterCounts);
    const config = item.count || item;
    if (typeof config === 'number') return config;
    return Math.floor(Math.random() * (config[1] - config[0] + 1)) + config[0];
  }

  generateTitle() {
    const template = this.randomChoice(this.titleTemplates);
    return template
      .replace(/{adjective}/g, this.randomChoice(this.adjectives))
      .replace(/{noun}/g, this.randomChoice(this.nouns))
      .replace(/{verb}/g, this.randomChoice(this.verbs))
      .replace(/{character}/g, "Harry") // Simple placeholder
      .replace(/{place}/g, this.randomChoice(this.places));
  }

  generateSummary() {
    const templates = [
      "A story about love, loss, and finding yourself again.",
      "When everything changes, only one thing remains constant.",
      "Sometimes the greatest adventures happen in the most ordinary places.",
      "A tale of friendship, betrayal, and redemption.",
      "What happens when your past catches up with your future?",
      "In a world where anything is possible, everything is at stake.",
      "A journey of self-discovery and unexpected romance.",
      "When fate intervenes, lives are forever changed.",
      "Love finds a way, even in the darkest of times.",
      "A story of courage, sacrifice, and hope."
    ];
    return this.randomChoice(templates);
  }

  generateRandomDate(startYear = 2020, endYear = 2024) {
    const start = new Date(startYear, 0, 1);
    const end = new Date(endYear, 11, 31);
    return new Date(start.getTime() + Math.random() * (end.getTime() - start.getTime()));
  }

  async createUsers(count = 3000) {
    if (this.cache.users.length >= count) {
      console.log(`👥 Using ${this.cache.users.length} existing users`);
      return;
    }

    const needed = count - this.cache.users.length;
    console.log(`\n👥 Creating ${needed} additional users...`);
    
    const batchSize = 100;
    for (let i = 0; i < needed; i += batchSize) {
      const batch = Math.min(batchSize, needed - i);
      
      for (let j = 0; j < batch; j++) {
        try {
          const username = this.generateUsername();
          const email = `${username}@example.com`;
          
          const result = await this.db.query(`
            INSERT INTO users (username, email, password_hash, is_verified, created_at)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (username) DO NOTHING
            RETURNING id, username
          `, [
            username,
            email,
            '$2b$10$dummy.hash.for.testing',
            Math.random() > 0.3, // 70% verified
            this.generateRandomDate()
          ]);
          
          if (result.rows.length > 0) {
            this.cache.users.push(result.rows[0]);
            this.stats.users++;
          }
        } catch (error) {
          this.stats.errors.push(`User creation: ${error.message}`);
        }
      }
      
      if ((i + batch) % 500 === 0) {
        console.log(`   👥 Created ${i + batch}/${needed} users`);
      }
    }
    
    console.log(`   ✅ Created ${this.stats.users} new users (${this.cache.users.length} total)`);
  }

  selectRandomTags(type, count = { min: 1, max: 5 }) {
    const tags = this.cache.tags[type] || [];
    if (tags.length === 0) return [];
    
    const numTags = Math.floor(Math.random() * (count.max - count.min + 1)) + count.min;
    return this.randomChoices(tags, 1, Math.min(numTags, tags.length));
  }

  selectRandomUser() {
    return this.randomChoice(this.cache.users);
  }

  async createWork(index) {
    try {
      const user = this.selectRandomUser();
      const wordCount = this.generateWordCount();
      const chapterCount = this.generateChapterCount();
      const isComplete = Math.random() > (chapterCount === 1 ? 0.05 : 0.7); // One-shots almost always complete
      
      const title = this.generateTitle();
      const summary = Math.random() > 0.2 ? this.generateSummary() : null;
      const notes = Math.random() > 0.7 ? "Thank you for reading!" : null;
      
      // Generate dates
      const publishedAt = this.generateRandomDate();
      const updatedAt = new Date(publishedAt.getTime() + Math.random() * (new Date() - publishedAt));
      
      // Create work
      const workResult = await this.db.query(`
        INSERT INTO works (
          title, summary, notes, user_id, word_count, chapter_count, 
          is_complete, status, rating, category, language,
          published_at, updated_at, created_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
        RETURNING id
      `, [
        title,
        summary,
        notes,
        user.id,
        wordCount,
        chapterCount,
        isComplete,
        'published',
        this.selectWeighted(this.distributions.ratings),
        this.selectWeighted(this.distributions.categories),
        this.selectWeighted(this.distributions.languages),
        publishedAt,
        updatedAt,
        publishedAt
      ]);
      
      const workId = workResult.rows[0].id;
      this.cache.works.push({ id: workId, title, user_id: user.id });
      
      // Add work tags
      const fandoms = this.selectRandomTags('fandom', { min: 1, max: 2 });
      const characters = this.selectRandomTags('character', { min: 0, max: 8 });
      const relationships = this.selectRandomTags('relationship', { min: 0, max: 4 });
      const freeformTags = this.selectRandomTags('freeform', { min: 0, max: 10 });
      
      const allTags = [...fandoms, ...characters, ...relationships, ...freeformTags];
      for (const tag of allTags) {
        await this.db.query(`
          INSERT INTO work_tags (work_id, tag_id, created_at)
          VALUES ($1, $2, $3)
          ON CONFLICT (work_id, tag_id) DO NOTHING
        `, [workId, tag.id, publishedAt]);
        this.stats.workTags++;
      }
      
      // Add warnings occasionally
      if (Math.random() < 0.3) {
        const warning = this.selectWeighted(this.distributions.warnings);
        const warningTags = this.cache.tags.warning || [];
        const warningTag = warningTags.find(t => t.name === warning);
        if (warningTag) {
          await this.db.query(`
            INSERT INTO work_tags (work_id, tag_id, created_at)
            VALUES ($1, $2, $3)
            ON CONFLICT (work_id, tag_id) DO NOTHING
          `, [workId, warningTag.id, publishedAt]);
        }
      }
      
      // Create chapters
      for (let i = 1; i <= chapterCount; i++) {
        const chapterWordCount = Math.floor(wordCount / chapterCount);
        const chapterContent = `This is chapter ${i} of the story. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.`;
        
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
      
      // Maybe add to collection (20% chance)
      if (Math.random() < 0.2 && this.cache.collections.length > 0) {
        const collection = this.randomChoice(this.cache.collections);
        await this.db.query(`
          INSERT INTO collection_works (collection_id, work_id, created_at)
          VALUES ($1, $2, $3)
          ON CONFLICT (collection_id, work_id) DO NOTHING
        `, [collection.id, workId, publishedAt]);
        this.stats.collectionWorks++;
      }
      
      // Maybe add to series (15% chance)
      if (Math.random() < 0.15 && this.cache.series.length > 0) {
        const userSeries = this.cache.series.filter(s => s.user_id === user.id);
        if (userSeries.length > 0 || Math.random() < 0.3) {
          let targetSeries;
          
          if (userSeries.length > 0 && Math.random() < 0.7) {
            // Use existing series
            targetSeries = this.randomChoice(userSeries);
          } else {
            // Create new series
            const seriesTitle = `${this.randomChoice(this.adjectives)} ${this.randomChoice(this.nouns)} Series`;
            const seriesResult = await this.db.query(`
              INSERT INTO series (title, description, user_id, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5)
              RETURNING id
            `, [
              seriesTitle,
              "A series of related stories.",
              user.id,
              publishedAt,
              updatedAt
            ]);
            targetSeries = { id: seriesResult.rows[0].id, user_id: user.id };
            this.cache.series.push(targetSeries);
            this.stats.series++;
          }
          
          // Get next position in series
          const positionResult = await this.db.query(`
            SELECT COALESCE(MAX(position), 0) + 1 as next_position
            FROM series_works WHERE series_id = $1
          `, [targetSeries.id]);
          
          await this.db.query(`
            INSERT INTO series_works (series_id, work_id, position, created_at)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (series_id, work_id) DO NOTHING
          `, [targetSeries.id, workId, positionResult.rows[0].next_position, publishedAt]);
          this.stats.seriesWorks++;
        }
      }
      
      // Generate engagement (comments, kudos, bookmarks)
      await this.generateWorkEngagement(workId, publishedAt, wordCount);
      
      this.stats.works++;
      
    } catch (error) {
      this.stats.errors.push(`Work ${index}: ${error.message}`);
    }
  }

  async generateWorkEngagement(workId, publishedAt, wordCount) {
    const daysOld = Math.floor((new Date() - publishedAt) / (1000 * 60 * 60 * 24));
    const popularityFactor = Math.random() * Math.random(); // Exponential distribution
    
    // Kudos (exponential distribution favoring fewer kudos)
    const kudosCount = Math.floor(popularityFactor * Math.min(200, wordCount / 50 + daysOld / 10));
    const kudosUsers = this.randomChoices(this.cache.users, 0, Math.min(kudosCount, this.cache.users.length));
    
    for (const user of kudosUsers) {
      try {
        const kudosDate = new Date(publishedAt.getTime() + Math.random() * (new Date() - publishedAt));
        await this.db.query(`
          INSERT INTO kudos (work_id, user_id, created_at)
          VALUES ($1, $2, $3)
          ON CONFLICT (work_id, user_id) DO NOTHING
        `, [workId, user.id, kudosDate]);
        this.stats.kudos++;
      } catch (error) {
        // Ignore duplicate kudos
      }
    }
    
    // Comments (fewer than kudos)
    const commentCount = Math.floor(popularityFactor * popularityFactor * 20);
    for (let i = 0; i < commentCount; i++) {
      try {
        const user = this.randomChoice(this.cache.users);
        const commentDate = new Date(publishedAt.getTime() + Math.random() * (new Date() - publishedAt));
        
        const comments = [
          "Great story! I loved it!",
          "This was amazing, thank you for writing it.",
          "Please update soon!",
          "I'm crying, this is so good.",
          "Beautiful writing, can't wait for more.",
          "This made my day, thank you!",
          "Absolutely loved this chapter!",
          "Your characterization is perfect.",
          "I'm obsessed with this story!",
          "This deserves more kudos!"
        ];
        
        await this.db.query(`
          INSERT INTO comments (
            work_id, user_id, content, created_at, updated_at
          )
          VALUES ($1, $2, $3, $4, $5)
        `, [
          workId,
          user.id,
          this.randomChoice(comments),
          commentDate,
          commentDate
        ]);
        this.stats.comments++;
      } catch (error) {
        // Ignore comment errors
      }
    }
    
    // Bookmarks (even fewer)
    if (Math.random() < popularityFactor * 0.3) {
      try {
        const user = this.randomChoice(this.cache.users);
        const bookmarkDate = new Date(publishedAt.getTime() + Math.random() * (new Date() - publishedAt));
        
        await this.db.query(`
          INSERT INTO bookmarks (
            work_id, user_id, notes, private, created_at
          )
          VALUES ($1, $2, $3, $4, $5)
        `, [
          workId,
          user.id,
          Math.random() < 0.3 ? "Need to read this again!" : null,
          Math.random() < 0.2,
          bookmarkDate
        ]);
        this.stats.bookmarks++;
      } catch (error) {
        // Ignore bookmark errors
      }
    }
  }

  async createWorks(count = 15000) {
    console.log(`\n📚 Creating ${count} works with chapters, tags, and engagement...`);
    
    const batchSize = 25; // Smaller batches for more frequent progress updates
    for (let i = 0; i < count; i += batchSize) {
      const batch = Math.min(batchSize, count - i);
      
      // Process batch sequentially to avoid overwhelming the database
      for (let j = 0; j < batch; j++) {
        await this.createWork(i + j + 1);
      }
      
      if ((i + batch) % 100 === 0) {
        console.log(`   📚 Created ${i + batch}/${count} works (${this.stats.chapters} chapters, ${this.stats.kudos} kudos, ${this.stats.comments} comments)`);
      }
    }
    
    console.log(`   ✅ Created ${this.stats.works} works total`);
  }

  async updateElasticsearchIndex() {
    console.log('\n🔍 Updating Elasticsearch index...');
    
    try {
      // Run the existing sync script
      const { spawn } = require('child_process');
      
      return new Promise((resolve, reject) => {
        const syncProcess = spawn('node', ['unified-elasticsearch-sync.js'], {
          cwd: __dirname,
          stdio: 'inherit'
        });
        
        syncProcess.on('close', (code) => {
          if (code === 0) {
            console.log('   ✅ Elasticsearch index updated successfully');
            resolve();
          } else {
            console.log('   ⚠️ Elasticsearch sync completed with warnings');
            resolve(); // Don't fail the whole process
          }
        });
        
        syncProcess.on('error', (error) => {
          console.log(`   ⚠️ Elasticsearch sync error: ${error.message}`);
          resolve(); // Don't fail the whole process
        });
      });
    } catch (error) {
      console.log(`   ⚠️ Could not update Elasticsearch: ${error.message}`);
    }
  }

  async generateFinalReport() {
    console.log('\n📊 Final Data Population Report');
    console.log('====================================');
    
    try {
      const counts = await this.db.query(`
        SELECT 
          (SELECT COUNT(*) FROM users) as users,
          (SELECT COUNT(*) FROM works) as works,
          (SELECT COUNT(*) FROM chapters) as chapters,
          (SELECT COUNT(*) FROM collections) as collections,
          (SELECT COUNT(*) FROM series) as series,
          (SELECT COUNT(*) FROM work_tags) as work_tags,
          (SELECT COUNT(*) FROM kudos) as kudos,
          (SELECT COUNT(*) FROM comments) as comments,
          (SELECT COUNT(*) FROM bookmarks) as bookmarks,
          (SELECT COUNT(*) FROM collection_works) as collection_works,
          (SELECT COUNT(*) FROM series_works) as series_works
      `);
      
      const stats = counts.rows[0];
      
      console.log(`👥 Users: ${stats.users}`);
      console.log(`📚 Works: ${stats.works}`);
      console.log(`📄 Chapters: ${stats.chapters}`);
      console.log(`📁 Collections: ${stats.collections}`);
      console.log(`📖 Series: ${stats.series}`);
      console.log(`🏷️ Work-Tag Associations: ${stats.work_tags}`);
      console.log(`💖 Kudos: ${stats.kudos}`);
      console.log(`💬 Comments: ${stats.comments}`);
      console.log(`🔖 Bookmarks: ${stats.bookmarks}`);
      console.log(`🗂️ Works in Collections: ${stats.collection_works}`);
      console.log(`📚 Works in Series: ${stats.series_works}`);
      
      // Performance stats
      const perfStats = await this.db.query(`
        SELECT 
          AVG(word_count)::int as avg_words,
          MAX(word_count) as max_words,
          AVG(chapter_count)::decimal(5,2) as avg_chapters,
          MAX(chapter_count) as max_chapters,
          COUNT(CASE WHEN is_complete THEN 1 END) as complete_works,
          COUNT(CASE WHEN status = 'published' THEN 1 END) as published_works
        FROM works
      `);
      
      const perf = perfStats.rows[0];
      console.log(`\n📈 Work Statistics:`);
      console.log(`   Average word count: ${perf.avg_words}`);
      console.log(`   Maximum word count: ${perf.max_words}`);
      console.log(`   Average chapters: ${perf.avg_chapters}`);
      console.log(`   Maximum chapters: ${perf.max_chapters}`);
      console.log(`   Complete works: ${perf.complete_works}`);
      console.log(`   Published works: ${perf.published_works}`);
      
      // Rating distribution
      const ratings = await this.db.query(`
        SELECT rating, COUNT(*) as count 
        FROM works 
        GROUP BY rating 
        ORDER BY count DESC
      `);
      
      console.log(`\n⭐ Rating Distribution:`);
      ratings.rows.forEach(row => {
        const pct = ((row.count / stats.works) * 100).toFixed(1);
        console.log(`   ${row.rating}: ${row.count} (${pct}%)`);
      });
      
      if (this.stats.errors.length > 0) {
        console.log(`\n⚠️ Errors encountered: ${this.stats.errors.length}`);
        this.stats.errors.slice(0, 5).forEach(error => {
          console.log(`   - ${error}`);
        });
        if (this.stats.errors.length > 5) {
          console.log(`   ... and ${this.stats.errors.length - 5} more`);
        }
      }
      
    } catch (error) {
      console.error('❌ Error generating report:', error.message);
    }
  }

  async cleanup() {
    await this.db.end();
    console.log('\n✅ Robust data population complete!');
    console.log('🎉 Nuclear AO3 now ready for scale testing with 15k works!');
  }
}

async function main() {
  const populator = new RobustDataPopulator();
  
  try {
    await populator.initialize();
    
    // Create users first
    await populator.createUsers(3000);
    
    // Create 15k works with full metadata
    await populator.createWorks(15000);
    
    // Update Elasticsearch index
    await populator.updateElasticsearchIndex();
    
    // Generate final report
    await populator.generateFinalReport();
    
  } catch (error) {
    console.error('❌ Data population failed:', error);
  } finally {
    await populator.cleanup();
  }
}

main().catch(console.error);