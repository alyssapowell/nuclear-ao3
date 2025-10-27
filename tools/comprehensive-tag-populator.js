#!/usr/bin/env node

/**
 * Nuclear AO3 - Comprehensive Tag Population System
 * Ensures rich tag ecosystem for testing and development
 */

const { Client } = require('pg');

// Configuration
const config = {
  database: {
    user: 'ao3_user',
    password: 'ao3_password',
    host: 'localhost',
    port: 5432,
    database: 'ao3_nuclear'
  }
};

// Comprehensive tag data for a rich testing environment
const COMPREHENSIVE_TAGS = {
  fandom: [
    // Anime/Manga
    'Naruto', 'One Piece', 'Attack on Titan', 'My Hero Academia', 'Demon Slayer', 
    'Death Note', 'Fullmetal Alchemist', 'Dragon Ball', 'Bleach', 'Hunter x Hunter',
    'Jujutsu Kaisen', 'Tokyo Ghoul', 'Haikyuu!!', 'Sailor Moon', 'Evangelion',
    
    // Marvel/DC
    'Marvel Cinematic Universe', 'Spider-Man - All Media Types', 'The Avengers (Marvel)',
    'X-Men - All Media Types', 'Iron Man (Movies)', 'Captain America (Movies)',
    'Thor (Movies)', 'Batman - All Media Types', 'Superman - All Media Types',
    'Wonder Woman - All Media Types', 'Justice League - All Media Types',
    
    // Harry Potter Universe
    'Harry Potter - J. K. Rowling', 'Fantastic Beasts and Where to Find Them (Movies)',
    'Harry Potter - Fandom', 'Marauders Era - Fandom',
    
    // TV Shows
    'Supernatural (TV 2005)', 'Sherlock (TV)', 'Doctor Who', 'The Witcher (TV)',
    'Game of Thrones (TV)', 'Stranger Things (TV 2016)', 'The Mandalorian (TV)',
    'Critical Role (Web Series)', 'Good Omens (TV)', 'Our Flag Means Death (TV)',
    'Ted Lasso (TV)', 'The Office (US)', 'Friends (TV)', 'Brooklyn Nine-Nine (TV)',
    
    // Books/Literature
    'The Lord of the Rings - J. R. R. Tolkien', 'The Hobbit - J. R. R. Tolkien',
    'Percy Jackson and the Olympians - Rick Riordan', 'The Hunger Games - Suzanne Collins',
    'Pride and Prejudice - Jane Austen', 'Jane Austen - Works',
    'A Song of Ice and Fire - George R. R. Martin',
    
    // Video Games
    'The Legend of Zelda & Related Fandoms', 'Final Fantasy VII',
    'Genshin Impact (Video Game)', 'Minecraft (Video Game)', 'Among Us (Video Game)',
    'Hades (Video Game 2018)', 'The Witcher (Video Game)', 'Overwatch (Video Game)',
    'Fire Emblem Series', 'Persona Series',
    
    // K-Pop/Music
    'K-pop', 'BTS', 'BLACKPINK', 'Stray Kids (Band)', 'TWICE (Band)',
    'Red Velvet (Music Band)', 'ITZY (Band)', 'NewJeans (Band)',
    
    // Other
    'Original Work', 'Real Person Fiction', 'Historical RPF'
  ],
  
  character: [
    // Harry Potter
    'Harry Potter', 'Hermione Granger', 'Ron Weasley', 'Draco Malfoy', 'Severus Snape',
    'Sirius Black', 'Remus Lupin', 'James Potter', 'Lily Evans Potter', 'Ginny Weasley',
    'Luna Lovegood', 'Neville Longbottom', 'Albus Dumbledore', 'Minerva McGonagall',
    
    // Marvel
    'Tony Stark', 'Steve Rogers', 'Natasha Romanoff', 'Clint Barton', 'Bruce Banner',
    'Thor (Marvel)', 'Loki (Marvel)', 'Peter Parker', 'Bucky Barnes', 'Sam Wilson',
    'Wanda Maximoff', 'Pietro Maximoff', 'Scott Lang', 'Hope van Dyne',
    
    // Naruto
    'Uzumaki Naruto', 'Uchiha Sasuke', 'Haruno Sakura', 'Hatake Kakashi',
    'Uchiha Itachi', 'Namikaze Minato', 'Uzumaki Kushina', 'Senju Tsunade',
    'Jiraiya (Naruto)', 'Orochimaru (Naruto)', 'Gaara (Naruto)',
    
    // Supernatural
    'Dean Winchester', 'Sam Winchester', 'Castiel (Supernatural)', 'Bobby Singer',
    'John Winchester', 'Mary Winchester', 'Crowley (Supernatural)', 'Gabriel (Supernatural)',
    
    // Sherlock
    'Sherlock Holmes', 'John Watson', 'Mycroft Holmes', 'Greg Lestrade', 'Molly Hooper',
    'Jim Moriarty', 'Mrs. Hudson', 'Irene Adler',
    
    // My Hero Academia
    'Midoriya Izuku', 'Bakugou Katsuki', 'Todoroki Shouto', 'Iida Tenya',
    'Uraraka Ochako', 'Aizawa Shouta | Eraserhead', 'All Might | Toshinori Yagi',
    
    // Original Characters
    'Original Male Character(s)', 'Original Female Character(s)', 'Original Character(s)',
    'Original Non-Binary Character(s)'
  ],
  
  relationship: [
    // Harry Potter
    'Harry Potter/Draco Malfoy', 'Harry Potter/Ginny Weasley', 'Hermione Granger/Ron Weasley',
    'Hermione Granger/Draco Malfoy', 'James Potter/Lily Evans Potter', 'Sirius Black/Remus Lupin',
    'Harry Potter/Hermione Granger', 'Harry Potter/Tom Riddle | Voldemort',
    
    // Marvel
    'Steve Rogers/Tony Stark', 'Steve Rogers/Bucky Barnes', 'Clint Barton/Natasha Romanoff',
    'Pepper Potts/Tony Stark', 'Jane Foster/Thor', 'Loki/Tony Stark', 'Peter Parker/Wade Wilson',
    'Wanda Maximoff/Vision', 'Carol Danvers/Maria Rambeau',
    
    // Supernatural
    'Dean Winchester/Castiel', 'Dean Winchester/Sam Winchester', 'Sam Winchester/Gabriel',
    'Dean Winchester/Benny Lafitte', 'Castiel/Sam Winchester',
    
    // Sherlock
    'Sherlock Holmes/John Watson', 'Sherlock Holmes/Jim Moriarty', 'Greg Lestrade/Mycroft Holmes',
    'Irene Adler/Kate | The Countess', 'John Watson/Mary Morstan',
    
    // My Hero Academia
    'Bakugou Katsuki/Midoriya Izuku', 'Midoriya Izuku/Todoroki Shouto',
    'Bakugou Katsuki/Kirishima Eijirou', 'Aizawa Shouta | Eraserhead/Yamada Hizashi | Present Mic',
    
    // Naruto
    'Uzumaki Naruto/Uchiha Sasuke', 'Hatake Kakashi/Umino Iruka', 'Uzumaki Naruto/Hyuuga Hinata',
    'Haruno Sakura/Uchiha Sasuke', 'Namikaze Minato/Uzumaki Kushina',
    
    // General/Other
    'Original Male Character/Original Male Character',
    'Original Female Character/Original Female Character',
    'Original Character/Original Character'
  ],
  
  freeform: [
    // Tropes
    'Fluff', 'Angst', 'Hurt/Comfort', 'Enemies to Lovers', 'Friends to Lovers',
    'Slow Burn', 'Mutual Pining', 'Unrequited Love', 'Pining', 'Idiots in Love',
    'Love Confessions', 'First Kiss', 'Getting Together', 'Established Relationship',
    'Break Up & Make Up', 'Jealousy', 'Misunderstandings', 'Communication Failure',
    
    // AU Types
    'Alternate Universe', 'Alternate Universe - Modern Setting', 'Alternate Universe - College/University',
    'Alternate Universe - High School', 'Alternate Universe - Coffee Shops & Cafés',
    'Alternate Universe - Soulmates', 'Alternate Universe - Roommates/Housemates',
    'Alternate Universe - Childhood Friends', 'Alternate Universe - No Powers',
    'Alternate Universe - Canon Divergence', 'Alternate Universe - Everyone Lives/Nobody Dies',
    'Alternate Universe - Human', 'Alternate Universe - Historical',
    
    // Fix-its & Time Travel
    'Fix-It', 'Time Travel', 'Time Loop', 'Canon Divergence', 'Post-Canon',
    'Pre-Canon', 'Missing Scene', 'Character Study', 'Introspection',
    
    // Family & Friendship
    'Found Family', 'Family Bonding', 'Friendship', 'Platonic Relationships',
    'Brotherly Love', 'Sibling Bonding', 'Parent-Child Relationship',
    'Adoption', 'Family Dynamics', 'Protective Siblings',
    
    // Emotional
    'Emotional Hurt/Comfort', 'Emotional Support', 'Mental Health Issues',
    'Depression', 'Anxiety', 'PTSD', 'Therapy', 'Healing', 'Recovery',
    'Self-Discovery', 'Identity Issues', 'Coming of Age',
    
    // Physical
    'Whump', 'Hurt/Comfort', 'Injury', 'Sickfic', 'Caretaking',
    'Medical Procedures', 'Hospitalization', 'Physical Therapy',
    
    // Genres
    'Romance', 'Drama', 'Comedy', 'Humor', 'Crack', 'Parody',
    'Horror', 'Supernatural Elements', 'Magic', 'Fantasy',
    'Science Fiction', 'Space Opera', 'Dystopia', 'Apocalypse',
    'Mystery', 'Detective', 'Crime', 'Thriller',
    
    // Writing Style
    'POV First Person', 'POV Second Person', 'POV Third Person',
    'POV Multiple', 'POV Alternating', 'Stream of Consciousness',
    'Epistolary', 'Chat Fic', 'Social Media', 'Texting',
    
    // Length & Structure
    'One Shot', 'Drabble', 'Ficlet', 'Short & Sweet', 'Long',
    'Epic Length', 'Series', 'Connected', 'Standalone',
    
    // Content Warnings & Tags
    'No Archive Warnings Apply', 'Choose Not To Use Archive Warnings',
    'Creator Chose Not To Use Archive Warnings', 'Not Beta Read',
    'Beta Read', 'Work In Progress', 'Complete Work',
    
    // Seasonal/Holiday
    'Christmas', 'Halloween', 'Valentine\'s Day', 'New Year',
    'Birthday', 'Anniversary', 'Summer', 'Winter', 'Spring', 'Autumn',
    
    // Common Scenarios
    'Road Trips', 'Vacation', 'Moving In Together', 'Domestic',
    'Cooking', 'Baking', 'Shopping', 'Date Night', 'Movie Night',
    'Game Night', 'Cuddling & Snuggling', 'Sharing a Bed',
    'Only One Bed', 'Fake Dating', 'Pretend Relationship',
    'Marriage of Convenience', 'Arranged Marriage'
  ]
};

class ComprehensiveTagPopulator {
  constructor() {
    this.db = new Client(config.database);
    this.stats = {
      existing: 0,
      created: 0,
      updated: 0,
      errors: []
    };
  }

  async initialize() {
    console.log('🏷️ Nuclear AO3 - Comprehensive Tag Population Starting...\n');
    
    try {
      await this.db.connect();
      console.log('✅ Database connected');
      return true;
    } catch (error) {
      console.error('❌ Connection failed:', error.message);
      return false;
    }
  }

  async checkCurrentTags() {
    console.log('📊 Checking current tag state...\n');
    
    try {
      const result = await this.db.query('SELECT COUNT(*) FROM tags');
      this.stats.existing = parseInt(result.rows[0].count);
      
      console.log(`🏷️ Current tags: ${this.stats.existing}`);
      
      // Check tag distribution
      const distributionResult = await this.db.query(`
        SELECT type, COUNT(*) as count 
        FROM tags 
        GROUP BY type 
        ORDER BY count DESC
      `);
      
      console.log('📈 Tag distribution:');
      distributionResult.rows.forEach(row => {
        console.log(`   ${row.type}: ${row.count}`);
      });
      
      return this.stats.existing < 500; // Need more tags if less than 500
    } catch (error) {
      console.error('❌ Error checking tags:', error.message);
      return true;
    }
  }

  async populateTagsByType(type, tags) {
    console.log(`🔖 Populating ${type} tags...`);
    
    let created = 0;
    let skipped = 0;
    
    for (const tagName of tags) {
      try {
        // Calculate usage count based on tag popularity (simulate real usage)
        const usageCount = this.calculateUsageCount(tagName, type);
        const isCanonical = true;  // This tag is canonical
        const canonicalName = null; // Since is_canonical=true, canonical_name must be null
        const isCommon = usageCount > 50;
        
        const result = await this.db.query(`
          INSERT INTO tags (name, type, is_canonical, canonical_name, canonical, common, use_count, created_at, updated_at)
          VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
          ON CONFLICT (name) 
          DO UPDATE SET 
            use_count = EXCLUDED.use_count,
            common = EXCLUDED.common,
            updated_at = NOW()
          RETURNING id, (xmax = 0) AS was_inserted
        `, [tagName, type, isCanonical, canonicalName, isCanonical, isCommon, usageCount]);
        
        if (result.rows[0].was_inserted) {
          created++;
        } else {
          this.stats.updated++;
        }
        
      } catch (error) {
        this.stats.errors.push(`${type} tag "${tagName}": ${error.message}`);
        skipped++;
      }
    }
    
    console.log(`   ✅ Created: ${created}, Updated: ${this.stats.updated}, Skipped: ${skipped}`);
    return created;
  }

  calculateUsageCount(tagName, type) {
    // Simulate realistic usage patterns
    const baseUsage = {
      'fandom': { min: 10, max: 5000 },
      'character': { min: 5, max: 2000 },
      'relationship': { min: 3, max: 1500 },
      'freeform': { min: 1, max: 8000 }
    };
    
    const range = baseUsage[type] || { min: 1, max: 100 };
    
    // Popular tags get higher usage
    const popularTags = [
      'Fluff', 'Angst', 'Harry Potter', 'Tony Stark', 'Draco Malfoy',
      'Steve Rogers/Tony Stark', 'Harry Potter/Draco Malfoy', 'Hurt/Comfort',
      'Enemies to Lovers', 'Alternate Universe', 'Dean Winchester/Castiel'
    ];
    
    if (popularTags.includes(tagName)) {
      return Math.floor(Math.random() * (range.max - range.max * 0.3)) + range.max * 0.3;
    }
    
    // Regular distribution with some randomness
    const factor = Math.random();
    if (factor < 0.1) {
      // 10% are very popular
      return Math.floor(Math.random() * (range.max - range.max * 0.5)) + range.max * 0.5;
    } else if (factor < 0.3) {
      // 20% are moderately popular
      return Math.floor(Math.random() * (range.max * 0.5 - range.max * 0.1)) + range.max * 0.1;
    } else {
      // 70% are less popular
      return Math.floor(Math.random() * (range.max * 0.1 - range.min)) + range.min;
    }
  }

  async createTagRelationships() {
    console.log('🔗 Creating tag relationships and synonyms...');
    
    const relationships = [
      // Common synonyms and relationships
      { parent: 'Harry Potter - J. K. Rowling', child: 'Harry Potter - Fandom' },
      { parent: 'Marvel Cinematic Universe', child: 'Marvel' },
      { parent: 'Tony Stark', child: 'Iron Man' },
      { parent: 'Steve Rogers', child: 'Captain America' },
      { parent: 'Fluff', child: 'Tooth-Rotting Fluff' },
      { parent: 'Angst', child: 'Heavy Angst' },
      { parent: 'Hurt/Comfort', child: 'H/C' },
      { parent: 'Alternate Universe', child: 'AU' },
      { parent: 'Original Character', child: 'OC' }
    ];
    
    let created = 0;
    for (const rel of relationships) {
      try {
        // Get parent and child tag IDs
        const parentResult = await this.db.query('SELECT id FROM tags WHERE name = $1', [rel.parent]);
        const childResult = await this.db.query('SELECT id FROM tags WHERE name = $1', [rel.child]);
        
        if (parentResult.rows.length > 0 && childResult.rows.length > 0) {
          await this.db.query(`
            INSERT INTO tag_relationships (parent_tag_id, child_tag_id, relationship_type, created_at)
            VALUES ($1, $2, 'synonym', NOW())
            ON CONFLICT (parent_tag_id, child_tag_id) DO NOTHING
          `, [parentResult.rows[0].id, childResult.rows[0].id]);
          
          created++;
        }
      } catch (error) {
        this.stats.errors.push(`Tag relationship ${rel.parent} -> ${rel.child}: ${error.message}`);
      }
    }
    
    console.log(`   ✅ Created ${created} tag relationships`);
  }

  async updateTagProminence() {
    console.log('⭐ Updating tag prominence based on usage...');
    
    try {
      // Update common flag for highly used tags (using existing columns)
      const result = await this.db.query(`
        UPDATE tags 
        SET common = true, updated_at = NOW()
        WHERE use_count > 50 AND common = false
      `);
      
      console.log(`   ✅ Updated prominence for ${result.rowCount} tags`);
      
    } catch (error) {
      console.error('❌ Error updating prominence:', error.message);
      this.stats.errors.push(`Prominence update: ${error.message}`);
    }
  }

  async populateAllTags() {
    console.log('🏭 Populating comprehensive tag database...\n');
    
    for (const [type, tags] of Object.entries(COMPREHENSIVE_TAGS)) {
      const created = await this.populateTagsByType(type, tags);
      this.stats.created += created;
    }
    
    await this.createTagRelationships();
    await this.updateTagProminence();
  }

  async validateTagSystem() {
    console.log('✅ Validating tag system...');
    
    try {
      // Check final counts
      const finalCount = await this.db.query('SELECT COUNT(*) FROM tags');
      const prominentCount = await this.db.query('SELECT COUNT(*) FROM tags WHERE common = true');
      const relationshipCount = await this.db.query('SELECT COUNT(*) FROM tag_relationships');
      
      console.log(`📊 Final tag statistics:`);
      console.log(`   Total tags: ${finalCount.rows[0].count}`);
      console.log(`   Prominent tags: ${prominentCount.rows[0].count}`);
      console.log(`   Tag relationships: ${relationshipCount.rows[0].count}`);
      
      // Test tag autocomplete
      const autocompleteTest = await this.db.query(`
        SELECT name, type, use_count 
        FROM tags 
        WHERE name ILIKE '%harry%' 
        ORDER BY use_count DESC 
        LIMIT 5
      `);
      
      console.log(`🔍 Autocomplete test for "harry":`);
      autocompleteTest.rows.forEach(tag => {
        console.log(`   ${tag.name} (${tag.type}) - ${tag.use_count} uses`);
      });
      
      return true;
    } catch (error) {
      console.error('❌ Validation failed:', error.message);
      return false;
    }
  }

  async cleanup() {
    try {
      await this.db.end();
    } catch (error) {
      // Ignore cleanup errors
    }
  }

  printFinalReport() {
    console.log('\n🎉 Comprehensive Tag Population Complete!');
    console.log('=============================================');
    console.log(`🏷️ Total existing tags: ${this.stats.existing}`);
    console.log(`✨ New tags created: ${this.stats.created}`);
    console.log(`🔄 Tags updated: ${this.stats.updated}`);
    
    if (this.stats.errors.length > 0) {
      console.log(`\n⚠️ Errors encountered: ${this.stats.errors.length}`);
      this.stats.errors.slice(0, 5).forEach(error => console.log(`   - ${error}`));
      if (this.stats.errors.length > 5) {
        console.log(`   ... and ${this.stats.errors.length - 5} more`);
      }
    }
    
    console.log('\n✅ Nuclear AO3 tag system ready!');
    console.log('🚀 Rich tagging ecosystem available for testing');
  }
}

// Main execution
async function main() {
  const populator = new ComprehensiveTagPopulator();
  
  try {
    if (!(await populator.initialize())) {
      process.exit(1);
    }
    
    const needsPopulation = await populator.checkCurrentTags();
    
    if (needsPopulation) {
      console.log('📦 Creating comprehensive tag ecosystem...\n');
      await populator.populateAllTags();
    } else {
      console.log('✅ Tag system already comprehensive - updating prominence only\n');
      await populator.updateTagProminence();
    }
    
    const isValid = await populator.validateTagSystem();
    
    populator.printFinalReport();
    
    if (!isValid) {
      console.log('\n⚠️ Consider running this script again to fix validation issues');
    }
    
  } catch (error) {
    console.error('💥 Fatal error:', error.message);
    process.exit(1);
  } finally {
    await populator.cleanup();
  }
}

// Run if called directly
if (require.main === module) {
  main().catch(console.error);
}

module.exports = ComprehensiveTagPopulator;