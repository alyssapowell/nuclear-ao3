#!/usr/bin/env node

/**
 * Nuclear AO3 Rate Limit Manager for Testing
 *
 * Rate limit counters live in the app's memory, so there is nothing outside
 * the app to clear. Restarting the app gives every client a fresh window.
 * These methods remain so the load tests that call them keep working.
 */

class RateLimitManager {
  async connect() {}

  async disconnect() {}

  async clearAllRateLimits() {
    console.log('ℹ️  Rate limits are kept in the app\'s memory; restart the app to reset them');
  }
}

if (require.main === module) {
  new RateLimitManager().clearAllRateLimits();
}

module.exports = RateLimitManager;
