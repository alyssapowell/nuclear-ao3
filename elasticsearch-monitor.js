#!/usr/bin/env node

/**
 * Elasticsearch Health Monitor & Auto-Recovery System
 * Ensures ES is always available and synced with PostgreSQL
 */

const { Client } = require('@elastic/elasticsearch');
const { Pool } = require('pg');
const cron = require('node-cron');

class ElasticsearchMonitor {
  constructor() {
    this.es = new Client({ node: 'http://localhost:9200' });
    this.pg = new Pool({
      host: 'localhost',
      port: 5432,
      database: 'ao3_nuclear',
      user: 'ao3_user',
      password: 'ao3_password'
    });
    
    this.isHealthy = false;
    this.lastSync = null;
    this.consecutiveFailures = 0;
    this.maxFailures = 3;
  }

  async checkHealth() {
    try {
      // Check ES cluster health
      const health = await this.es.cluster.health();
      const isESHealthy = health.status === 'green' || health.status === 'yellow';
      
      // Check if works index exists and has data
      const indexExists = await this.es.indices.exists({ index: 'works' });
      let docCount = 0;
      
      if (indexExists) {
        const count = await this.es.count({ index: 'works' });
        docCount = count.count;
      }
      
      // Check PostgreSQL work count
      const pgResult = await this.pg.query('SELECT COUNT(*) as count FROM works');
      const pgCount = parseInt(pgResult.rows[0].count);
      
      const isSynced = Math.abs(docCount - pgCount) <= 10; // Allow small diff
      
      console.log(`[${new Date().toISOString()}] Health Check:`);
      console.log(`  ES Status: ${health.status}`);
      console.log(`  ES Docs: ${docCount}`);
      console.log(`  PG Works: ${pgCount}`);
      console.log(`  Synced: ${isSynced ? '✅' : '❌'}`);
      
      const wasHealthy = this.isHealthy;
      this.isHealthy = isESHealthy && indexExists && isSynced;
      
      if (this.isHealthy) {
        this.consecutiveFailures = 0;
        if (!wasHealthy) {
          console.log('🎉 Elasticsearch is now healthy!');
        }
      } else {
        this.consecutiveFailures++;
        console.log(`⚠️  Health check failed (${this.consecutiveFailures}/${this.maxFailures})`);
        
        // Auto-recovery if consecutive failures
        if (this.consecutiveFailures >= this.maxFailures) {
          console.log('🔧 Starting auto-recovery...');
          await this.autoRecover();
        }
      }
      
      return this.isHealthy;
    } catch (error) {
      console.error('❌ Health check error:', error.message);
      this.isHealthy = false;
      this.consecutiveFailures++;
      return false;
    }
  }

  async autoRecover() {
    try {
      console.log('1. Checking if ES is responsive...');
      await this.es.ping();
      
      console.log('2. Ensuring works index exists...');
      const indexExists = await this.es.indices.exists({ index: 'works' });
      
      if (!indexExists) {
        console.log('3. Creating works index...');
        await this.createWorksIndex();
      }
      
      console.log('4. Triggering full resync...');
      await this.triggerResync();
      
      console.log('5. Resetting failure counter...');
      this.consecutiveFailures = 0;
      this.lastSync = new Date();
      
      console.log('✅ Auto-recovery completed!');
    } catch (error) {
      console.error('❌ Auto-recovery failed:', error.message);
    }
  }

  async createWorksIndex() {
    const indexConfig = {
      index: 'works',
      body: {
        mappings: {
          properties: {
            title: { type: 'text', analyzer: 'standard' },
            summary: { type: 'text' },
            content_text: { type: 'text' },
            rating: { type: 'keyword' },
            language: { type: 'keyword' },
            fandoms: { type: 'text', analyzer: 'standard' },
            characters: { type: 'text', analyzer: 'standard' },
            relationships: { type: 'text', analyzer: 'standard' },
            freeform_tags: { type: 'text', analyzer: 'standard' },
            warnings: { type: 'keyword' },
            categories: { type: 'keyword' },
            word_count: { type: 'integer' },
            chapter_count: { type: 'integer' },
            is_complete: { type: 'boolean' },
            status: { type: 'keyword' },
            published_at: { type: 'date' },
            updated_at: { type: 'date' },
            created_at: { type: 'date' },
            user_id: { type: 'keyword' },
            hits_count: { type: 'integer' },
            kudos_count: { type: 'integer' },
            comments_count: { type: 'integer' },
            bookmarks_count: { type: 'integer' }
          }
        },
        settings: {
          number_of_shards: 1,
          number_of_replicas: 0
        }
      }
    };
    
    await this.es.indices.create(indexConfig);
    console.log('✅ Works index created successfully');
  }

  async triggerResync() {
    // Use the existing sync script
    const { exec } = require('child_process');
    const util = require('util');
    const execAsync = util.promisify(exec);
    
    try {
      const { stdout, stderr } = await execAsync('cd /Users/alyssapowell/Documents/projects/otwarchive/nuclear-ao3/tools && node unified-elasticsearch-sync.js');
      console.log('Sync output:', stdout);
      if (stderr) console.error('Sync errors:', stderr);
      this.lastSync = new Date();
    } catch (error) {
      console.error('Sync failed:', error.message);
      throw error;
    }
  }

  async monitorPerformance() {
    try {
      const stats = await this.es.cluster.stats();
      const nodes = await this.es.nodes.stats();
      
      console.log(`[${new Date().toISOString()}] Performance Stats:`);
      console.log(`  Cluster Status: ${stats.status}`);
      console.log(`  Total Docs: ${stats.indices.count.toLocaleString()}`);
      console.log(`  Index Size: ${(stats.indices.store.size_in_bytes / 1024 / 1024).toFixed(2)} MB`);
      
      // Check for performance issues
      const nodeStats = Object.values(nodes.nodes)[0];
      if (nodeStats) {
        const heapPercent = nodeStats.jvm.mem.heap_used_percent;
        const cpuPercent = nodeStats.os.cpu?.percent || 0;
        
        console.log(`  Heap Usage: ${heapPercent}%`);
        console.log(`  CPU Usage: ${cpuPercent}%`);
        
        if (heapPercent > 80) {
          console.log('⚠️  High heap usage detected!');
        }
        if (cpuPercent > 80) {
          console.log('⚠️  High CPU usage detected!');
        }
      }
    } catch (error) {
      console.error('Performance monitoring failed:', error.message);
    }
  }

  async testSearchPerformance() {
    try {
      const start = Date.now();
      const response = await this.es.search({
        index: 'works',
        body: {
          query: { match_all: {} },
          size: 10
        }
      });
      const duration = Date.now() - start;
      
      console.log(`Search Test: ${duration}ms for ${response.hits.total.value} documents`);
      
      if (duration > 1000) {
        console.log('⚠️  Slow search performance detected!');
      }
    } catch (error) {
      console.error('Search test failed:', error.message);
    }
  }

  start() {
    console.log('🚀 Starting Elasticsearch Monitor...');
    
    // Initial health check
    this.checkHealth();
    
    // Health checks every 30 seconds
    cron.schedule('*/30 * * * * *', () => {
      this.checkHealth();
    });
    
    // Performance monitoring every 5 minutes
    cron.schedule('*/5 * * * *', () => {
      this.monitorPerformance();
      this.testSearchPerformance();
    });
    
    // Full sync verification every hour
    cron.schedule('0 * * * *', () => {
      console.log('⏰ Hourly sync verification...');
      this.checkHealth();
    });
    
    // Graceful shutdown
    process.on('SIGINT', async () => {
      console.log('🛑 Shutting down monitor...');
      await this.pg.end();
      process.exit(0);
    });
  }
}

// Start monitoring if run directly
if (require.main === module) {
  const monitor = new ElasticsearchMonitor();
  monitor.start();
}

module.exports = ElasticsearchMonitor;