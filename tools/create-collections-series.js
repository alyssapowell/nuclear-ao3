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

class CollectionsSeriesCreator {
  constructor() {
    this.db = new Client(config.database);
    this.stats = {
      collections: 0,
      series: 0,
      errors: []
    };
  }

  async initialize() {
    console.log('🗂️ Nuclear AO3 - Creating Collections & Series...\n');
    await this.db.connect();
    console.log('✅ Connected to database');
  }

  async createCollections() {
    console.log('\n📁 Creating Collections...');
    
    // Get a user to assign as owner
    const userResult = await this.db.query('SELECT id FROM users LIMIT 1');
    if (userResult.rows.length === 0) {
      console.log('   ❌ No users found - cannot create collections');
      return;
    }
    const userId = userResult.rows[0].id;
    
    const collections = [
      {
        title: 'Harry Potter Holiday Collection',
        description: 'A collection of Harry Potter fanfiction themed around holidays and special occasions.',
        type: 'collection',
        is_open: true,
        is_anonymous: false
      },
      {
        title: 'Next Generation Hogwarts',
        description: 'Stories featuring the children of Harry Potter characters attending Hogwarts.',
        type: 'collection',
        is_open: true,
        is_anonymous: false
      },
      {
        title: 'Marauders Era Collection',
        description: 'Fanfiction focusing on James Potter, Sirius Black, Remus Lupin, and Peter Pettigrew during their Hogwarts years.',
        type: 'collection',
        is_open: false,
        is_anonymous: false
      },
      {
        title: 'Alternative Universe Harry Potter',
        description: 'Harry Potter stories set in modern day, different time periods, or completely alternate universes.',
        type: 'collection',
        is_open: true,
        is_anonymous: false
      },
      {
        title: 'Short & Sweet HP Ficlets',
        description: 'A collection for Harry Potter stories under 1000 words. Perfect for quick reads!',
        type: 'collection',
        is_open: true,
        is_anonymous: false
      },
      {
        title: 'Rare Pairs & Unusual Ships',
        description: 'Featuring unusual or rare romantic pairings in the Harry Potter universe.',
        type: 'collection',
        is_open: true,
        is_anonymous: true
      }
    ];

    for (const collection of collections) {
      try {
        const result = await this.db.query(`
          INSERT INTO collections (
            title, description, user_id, type, is_open, is_anonymous, 
            created_at, updated_at
          )
          VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
          RETURNING id, title
        `, [
          collection.title,
          collection.description,
          userId,
          collection.type,
          collection.is_open,
          collection.is_anonymous
        ]);
        
        console.log(`   ✅ Created collection: "${result.rows[0].title}"`);
        this.stats.collections++;
        
      } catch (error) {
        console.log(`   ❌ Failed to create collection "${collection.title}": ${error.message}`);
        this.stats.errors.push(`Collection "${collection.title}": ${error.message}`);
      }
    }
  }

  async createSeries() {
    console.log('\n📚 Creating Series...');
    
    // Get a user to assign as owner
    const userResult = await this.db.query('SELECT id FROM users LIMIT 1');
    if (userResult.rows.length === 0) {
      console.log('   ❌ No users found - cannot create series');
      return;
    }
    const userId = userResult.rows[0].id;
    
    const series = [
      {
        title: 'Harry Potter: The Search Engine Chronicles',
        description: 'A multi-part series following Harry Potter as he discovers the magical world of search engines and databases. A humorous take on modern magic.',
        is_complete: false
      },
      {
        title: 'Nuclear Hogwarts Academy',
        description: 'An alternate universe where Hogwarts has been rebuilt as a modern academy with advanced magical technology.',
        is_complete: false
      },
      {
        title: 'The Elasticsearch Mysteries',
        description: 'Hermione Granger applies her logical mind to solving mysteries using magical search algorithms.',
        is_complete: true
      },
      {
        title: 'DevOps at the Ministry',
        description: 'Arthur Weasley starts a DevOps department at the Ministry of Magic, bringing Muggle technology into the wizarding world.',
        is_complete: false
      },
      {
        title: 'Database Administration: A Wizard\'s Guide',
        description: 'Percy Weasley finds his calling in magical database management. A surprisingly engaging bureaucratic adventure.',
        is_complete: true
      }
    ];

    for (const serie of series) {
      try {
        const result = await this.db.query(`
          INSERT INTO series (
            title, description, user_id, is_complete, created_at, updated_at
          )
          VALUES ($1, $2, $3, $4, NOW(), NOW())
          RETURNING id, title
        `, [
          serie.title,
          serie.description,
          userId,
          serie.is_complete
        ]);
        
        console.log(`   ✅ Created series: "${result.rows[0].title}"`);
        this.stats.series++;
        
      } catch (error) {
        console.log(`   ❌ Failed to create series "${serie.title}": ${error.message}`);
        this.stats.errors.push(`Series "${serie.title}": ${error.message}`);
      }
    }
  }

  async associateWorksWithCollections() {
    console.log('\n🔗 Associating works with collections...');
    
    try {
      // Get Harry Potter works
      const harryWorks = await this.db.query(`
        SELECT w.id, w.title 
        FROM works w 
        WHERE w.title ILIKE '%harry%' OR w.title ILIKE '%potter%'
        LIMIT 5
      `);
      
      // Get collections
      const collections = await this.db.query(`
        SELECT id, title FROM collections ORDER BY created_at DESC LIMIT 3
      `);
      
      if (harryWorks.rows.length > 0 && collections.rows.length > 0) {
        // Associate first few Harry Potter works with first collection
        const collection = collections.rows[0];
        
        for (const work of harryWorks.rows.slice(0, 3)) {
          try {
            await this.db.query(`
              INSERT INTO collection_works (collection_id, work_id, created_at)
              VALUES ($1, $2, NOW())
              ON CONFLICT (collection_id, work_id) DO NOTHING
            `, [collection.id, work.id]);
            
            console.log(`   ✅ Added "${work.title}" to "${collection.title}"`);
          } catch (error) {
            console.log(`   ❌ Failed to add work to collection: ${error.message}`);
          }
        }
      }
    } catch (error) {
      console.log(`   ❌ Error associating works with collections: ${error.message}`);
    }
  }

  async associateWorksWithSeries() {
    console.log('\n📖 Associating works with series...');
    
    try {
      // Get Harry Potter works
      const harryWorks = await this.db.query(`
        SELECT w.id, w.title 
        FROM works w 
        WHERE w.title ILIKE '%harry%' AND w.title ILIKE '%search%'
        ORDER BY w.created_at
        LIMIT 3
      `);
      
      // Get the search chronicles series
      const series = await this.db.query(`
        SELECT id, title 
        FROM series 
        WHERE title ILIKE '%search engine chronicles%'
        LIMIT 1
      `);
      
      if (harryWorks.rows.length > 0 && series.rows.length > 0) {
        const targetSeries = series.rows[0];
        
        for (let i = 0; i < harryWorks.rows.length; i++) {
          const work = harryWorks.rows[i];
          try {
            await this.db.query(`
              INSERT INTO series_works (series_id, work_id, position, created_at)
              VALUES ($1, $2, $3, NOW())
              ON CONFLICT (series_id, work_id) DO NOTHING
            `, [targetSeries.id, work.id, i + 1]);
            
            console.log(`   ✅ Added "${work.title}" to series at position ${i + 1}`);
          } catch (error) {
            console.log(`   ❌ Failed to add work to series: ${error.message}`);
          }
        }
      }
    } catch (error) {
      console.log(`   ❌ Error associating works with series: ${error.message}`);
    }
  }

  async generateReport() {
    console.log('\n📊 Final Collections & Series Report');
    console.log('=====================================');
    
    try {
      const collectionsCount = await this.db.query('SELECT COUNT(*) FROM collections');
      const seriesCount = await this.db.query('SELECT COUNT(*) FROM series');
      const collectionWorksCount = await this.db.query('SELECT COUNT(*) FROM collection_works');
      const seriesWorksCount = await this.db.query('SELECT COUNT(*) FROM series_works');
      
      console.log(`📁 Total Collections: ${collectionsCount.rows[0].count}`);
      console.log(`📚 Total Series: ${seriesCount.rows[0].count}`);
      console.log(`🔗 Works in Collections: ${collectionWorksCount.rows[0].count}`);
      console.log(`📖 Works in Series: ${seriesWorksCount.rows[0].count}`);
      
      if (this.stats.errors.length > 0) {
        console.log(`\n⚠️ Errors encountered: ${this.stats.errors.length}`);
        this.stats.errors.slice(0, 5).forEach(error => {
          console.log(`   - ${error}`);
        });
      }
      
    } catch (error) {
      console.error('❌ Error generating report:', error.message);
    }
  }

  async cleanup() {
    await this.db.end();
    console.log('\n✅ Collections & Series creation complete!');
  }
}

async function main() {
  const creator = new CollectionsSeriesCreator();
  
  try {
    await creator.initialize();
    await creator.createCollections();
    await creator.createSeries();
    await creator.associateWorksWithCollections();
    await creator.associateWorksWithSeries();
    await creator.generateReport();
  } catch (error) {
    console.error('❌ Script failed:', error);
  } finally {
    await creator.cleanup();
  }
}

main().catch(console.error);