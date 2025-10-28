#!/usr/bin/env node

/**
 * Nuclear AO3 Rate Limit Manager for Testing
 * Provides utilities to manage rate limits during testing
 */

const { createClient } = require('redis');

class RateLimitManager {
  constructor() {
    this.redis = createClient({
      host: process.env.REDIS_HOST || 'localhost',
      port: process.env.REDIS_PORT || 6379,
      db: process.env.REDIS_DB || 0
    });
  }

  async connect() {
    try {
      await this.redis.connect();
      console.log('✅ Connected to Redis for rate limit management');
    } catch (error) {
      console.error('❌ Failed to connect to Redis:', error.message);
      throw error;
    }
  }

  async disconnect() {
    try {
      await this.redis.disconnect();
      console.log('👋 Disconnected from Redis');
    } catch (error) {
      console.error('❌ Error disconnecting from Redis:', error.message);
    }
  }

  // Clear all rate limit keys
  async clearAllRateLimits() {
    try {
      const keys = await this.redis.keys('rate_limit:*');
      if (keys.length > 0) {
        await this.redis.del(keys);
        console.log(`🧹 Cleared ${keys.length} rate limit entries`);
      } else {
        console.log('ℹ️  No rate limit entries to clear');
      }
      return keys.length;
    } catch (error) {
      console.error('❌ Error clearing rate limits:', error.message);
      throw error;
    }
  }

  // Clear rate limit for specific IP
  async clearRateLimitForIP(ip) {
    try {
      const key = `rate_limit:${ip}`;
      const result = await this.redis.del(key);
      if (result) {
        console.log(`🧹 Cleared rate limit for IP: ${ip}`);
      } else {
        console.log(`ℹ️  No rate limit found for IP: ${ip}`);
      }
      return result;
    } catch (error) {
      console.error(`❌ Error clearing rate limit for IP ${ip}:`, error.message);
      throw error;
    }
  }

  // Check current rate limit status for IP
  async checkRateLimitStatus(ip) {
    try {
      const key = `rate_limit:${ip}`;
      const current = await this.redis.get(key);
      const ttl = await this.redis.ttl(key);
      
      return {
        ip,
        current: current ? parseInt(current) : 0,
        limit: 100, // Current limit in middleware
        remaining: Math.max(0, 100 - (current ? parseInt(current) : 0)),
        ttl: ttl > 0 ? ttl : 0,
        resetTime: ttl > 0 ? new Date(Date.now() + (ttl * 1000)) : null
      };
    } catch (error) {
      console.error(`❌ Error checking rate limit for IP ${ip}:`, error.message);
      throw error;
    }
  }

  // List all current rate limits
  async listAllRateLimits() {
    try {
      const keys = await this.redis.keys('rate_limit:*');
      const statuses = [];

      for (const key of keys) {
        const ip = key.replace('rate_limit:', '');
        const status = await this.checkRateLimitStatus(ip);
        statuses.push(status);
      }

      return statuses;
    } catch (error) {
      console.error('❌ Error listing rate limits:', error.message);
      throw error;
    }
  }

  // Set testing mode (bypass rate limits)
  async enableTestMode() {
    try {
      await this.redis.set('testing:rate_limit_bypass', 'true', { EX: 3600 }); // 1 hour
      console.log('🧪 Test mode enabled - rate limits bypassed for 1 hour');
    } catch (error) {
      console.error('❌ Error enabling test mode:', error.message);
      throw error;
    }
  }

  async disableTestMode() {
    try {
      await this.redis.del('testing:rate_limit_bypass');
      console.log('🔒 Test mode disabled - rate limits restored');
    } catch (error) {
      console.error('❌ Error disabling test mode:', error.message);
      throw error;
    }
  }

  async isTestModeEnabled() {
    try {
      const result = await this.redis.get('testing:rate_limit_bypass');
      return result === 'true';
    } catch (error) {
      console.error('❌ Error checking test mode:', error.message);
      return false;
    }
  }
}

// CLI Interface
async function main() {
  const manager = new RateLimitManager();
  await manager.connect();

  const command = process.argv[2];
  const arg = process.argv[3];

  try {
    switch (command) {
      case 'clear':
        if (arg) {
          await manager.clearRateLimitForIP(arg);
        } else {
          await manager.clearAllRateLimits();
        }
        break;

      case 'status':
        if (arg) {
          const status = await manager.checkRateLimitStatus(arg);
          console.log('📊 Rate Limit Status:');
          console.log(`   IP: ${status.ip}`);
          console.log(`   Current: ${status.current}/${status.limit}`);
          console.log(`   Remaining: ${status.remaining}`);
          console.log(`   TTL: ${status.ttl}s`);
          console.log(`   Reset Time: ${status.resetTime || 'N/A'}`);
        } else {
          const statuses = await manager.listAllRateLimits();
          if (statuses.length === 0) {
            console.log('ℹ️  No active rate limits');
          } else {
            console.log('📊 Active Rate Limits:');
            statuses.forEach(status => {
              console.log(`   ${status.ip}: ${status.current}/${status.limit} (${status.remaining} remaining, ${status.ttl}s TTL)`);
            });
          }
        }
        break;

      case 'test-mode':
        if (arg === 'on') {
          await manager.enableTestMode();
        } else if (arg === 'off') {
          await manager.disableTestMode();
        } else {
          const enabled = await manager.isTestModeEnabled();
          console.log(`🧪 Test mode: ${enabled ? 'ENABLED' : 'DISABLED'}`);
        }
        break;

      case 'help':
      default:
        console.log(`
🔧 Nuclear AO3 Rate Limit Manager

Usage:
  node test-rate-limit-manager.js <command> [args]

Commands:
  clear [ip]       Clear all rate limits or specific IP
  status [ip]      Show rate limit status for all IPs or specific IP
  test-mode [on|off] Enable/disable test mode or show current status
  help             Show this help message

Examples:
  node test-rate-limit-manager.js clear
  node test-rate-limit-manager.js clear 127.0.0.1
  node test-rate-limit-manager.js status
  node test-rate-limit-manager.js status 127.0.0.1
  node test-rate-limit-manager.js test-mode on
        `);
        break;
    }
  } finally {
    await manager.disconnect();
  }
}

// Export for use in other scripts
if (require.main === module) {
  main().catch(console.error);
} else {
  module.exports = RateLimitManager;
}