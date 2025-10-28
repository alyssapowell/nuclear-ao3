#!/usr/bin/env node

/**
 * Nuclear AO3 Performance Load Testing Suite
 * Tests system performance under various load conditions:
 * - Concurrent API requests
 * - Frontend page load times
 * - Database query performance
 * - Search performance under load
 * - Authentication performance
 */

const RateLimitManager = require('./test-rate-limit-manager');

const API_BASE = 'http://localhost:8080/api/v1';
const FRONTEND_BASE = 'http://localhost:3001';

// Test configuration
const LOAD_TEST_CONFIG = {
  concurrent_users: 20,
  test_duration_seconds: 20,
  ramp_up_time_seconds: 5,
  endpoints_to_test: [
    { path: '/health', method: 'GET', weight: 5 },
    { path: '/works/', method: 'GET', weight: 20 },
    { path: '/search/works/?query=test', method: 'GET', weight: 15 },
    { path: '/tags/search?q=test', method: 'GET', weight: 10 },
    { path: '/series?limit=10', method: 'GET', weight: 5 }
  ],
  frontend_pages: [
    { path: '/', weight: 25 },
    { path: '/works', weight: 20 },
    { path: '/search', weight: 15 },
    { path: '/series', weight: 10 }
  ]
};

class PerformanceMetrics {
  constructor() {
    this.requests = [];
    this.errors = [];
    this.startTime = Date.now();
  }

  recordRequest(endpoint, responseTime, status, error = null) {
    const record = {
      endpoint,
      responseTime,
      status,
      timestamp: Date.now(),
      error
    };

    this.requests.push(record);
    if (error || status >= 400) {
      this.errors.push(record);
    }
  }

  getStats() {
    const now = Date.now();
    const duration = (now - this.startTime) / 1000;
    const responseTimes = this.requests.map(r => r.responseTime);
    
    return {
      duration: duration,
      total_requests: this.requests.length,
      successful_requests: this.requests.filter(r => r.status < 400).length,
      failed_requests: this.errors.length,
      requests_per_second: this.requests.length / duration,
      avg_response_time: responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length || 0,
      min_response_time: Math.min(...responseTimes) || 0,
      max_response_time: Math.max(...responseTimes) || 0,
      p95_response_time: this.percentile(responseTimes, 95),
      p99_response_time: this.percentile(responseTimes, 99),
      error_rate: (this.errors.length / this.requests.length) * 100 || 0
    };
  }

  percentile(arr, p) {
    if (arr.length === 0) return 0;
    const sorted = arr.slice().sort((a, b) => a - b);
    const index = Math.ceil((p / 100) * sorted.length) - 1;
    return sorted[Math.max(0, index)];
  }
}

async function makeRequest(url, options = {}) {
  const startTime = Date.now();
  let status = 0;
  let error = null;

  try {
    const response = await fetch(url, {
      method: 'GET',
      timeout: 10000,
      ...options
    });
    status = response.status;
    await response.text(); // Consume response body
  } catch (e) {
    error = e.message;
    status = 0;
  }

  const responseTime = Date.now() - startTime;
  return { responseTime, status, error };
}

async function testAPIEndpoint(endpoint, metrics) {
  const url = `${API_BASE}${endpoint.path}`;
  const result = await makeRequest(url, { method: endpoint.method });
  metrics.recordRequest(endpoint.path, result.responseTime, result.status, result.error);
  return result;
}

async function testFrontendPage(page, metrics) {
  const url = `${FRONTEND_BASE}${page.path}`;
  const result = await makeRequest(url);
  metrics.recordRequest(`frontend:${page.path}`, result.responseTime, result.status, result.error);
  return result;
}

function selectRandomEndpoint(endpoints) {
  const totalWeight = endpoints.reduce((sum, ep) => sum + ep.weight, 0);
  let random = Math.random() * totalWeight;
  
  for (const endpoint of endpoints) {
    random -= endpoint.weight;
    if (random <= 0) {
      return endpoint;
    }
  }
  return endpoints[0];
}

async function runLoadTestWorker(workerId, config, metrics) {
  const endTime = Date.now() + (config.test_duration_seconds * 1000);
  let requestCount = 0;

  console.log(`Worker ${workerId} started`);

  while (Date.now() < endTime) {
    try {
      // Test API endpoint
      const apiEndpoint = selectRandomEndpoint(config.endpoints_to_test);
      await testAPIEndpoint(apiEndpoint, metrics);
      requestCount++;

      // Test frontend page occasionally
      if (requestCount % 3 === 0) {
        const frontendPage = selectRandomEndpoint(config.frontend_pages);
        await testFrontendPage(frontendPage, metrics);
        requestCount++;
      }

      // Delay to respect rate limits
      await new Promise(resolve => setTimeout(resolve, 250));
    } catch (e) {
      metrics.recordRequest('error', 0, 500, e.message);
    }
  }

  console.log(`Worker ${workerId} completed ${requestCount} requests`);
}

async function runConcurrentSearch() {
  console.log('\n🔍 Testing Search Performance Under Load...');
  const searchMetrics = new PerformanceMetrics();
  const concurrentSearches = 20;
  const searchQueries = [
    'love', 'adventure', 'fantasy', 'romance', 'drama',
    'angst', 'fluff', 'hurt/comfort', 'alternate universe'
  ];

  const searchPromises = [];
  for (let i = 0; i < concurrentSearches; i++) {
    const query = searchQueries[i % searchQueries.length];
    const promise = testAPIEndpoint(
      { path: `/search/works/?query=${encodeURIComponent(query)}`, method: 'GET' },
      searchMetrics
    );
    searchPromises.push(promise);
  }

  await Promise.all(searchPromises);
  
  const stats = searchMetrics.getStats();
  console.log(`✅ Search Performance Results:`);
  console.log(`   Concurrent searches: ${concurrentSearches}`);
  console.log(`   Average response time: ${stats.avg_response_time.toFixed(2)}ms`);
  console.log(`   95th percentile: ${stats.p95_response_time.toFixed(2)}ms`);
  console.log(`   Success rate: ${(100 - stats.error_rate).toFixed(1)}%`);
  
  return stats;
}

async function runAuthenticationPerformance() {
  console.log('\n🔐 Testing Authentication Performance...');
  const authMetrics = new PerformanceMetrics();
  const concurrentLogins = 10;

  const loginPromises = [];
  for (let i = 0; i < concurrentLogins; i++) {
    const testUser = {
      email: `perftest_${Date.now()}_${i}@example.com`,
      password: 'PerfTest123!',
      username: `perftest_${Date.now()}_${i}`
    };

    // Register then login
    const promise = (async () => {
      // Registration
      const regResult = await makeRequest(`${API_BASE}/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(testUser)
      });
      authMetrics.recordRequest('/auth/register', regResult.responseTime, regResult.status, regResult.error);

      // Login
      const loginResult = await makeRequest(`${API_BASE}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: testUser.email,
          password: testUser.password
        })
      });
      authMetrics.recordRequest('/auth/login', loginResult.responseTime, loginResult.status, loginResult.error);
    })();

    loginPromises.push(promise);
  }

  await Promise.all(loginPromises);
  
  const stats = authMetrics.getStats();
  console.log(`✅ Authentication Performance Results:`);
  console.log(`   Concurrent auth operations: ${concurrentLogins * 2}`);
  console.log(`   Average response time: ${stats.avg_response_time.toFixed(2)}ms`);
  console.log(`   95th percentile: ${stats.p95_response_time.toFixed(2)}ms`);
  console.log(`   Success rate: ${(100 - stats.error_rate).toFixed(1)}%`);
  
  return stats;
}

async function runMainLoadTest() {
  console.log('🚀 Nuclear AO3 Performance Load Testing');
  console.log('========================================\n');
  
  console.log(`Configuration:`);
  console.log(`  Concurrent users: ${LOAD_TEST_CONFIG.concurrent_users}`);
  console.log(`  Test duration: ${LOAD_TEST_CONFIG.test_duration_seconds}s`);
  console.log(`  Ramp-up time: ${LOAD_TEST_CONFIG.ramp_up_time_seconds}s`);
  
  const metrics = new PerformanceMetrics();
  
  console.log('\n📊 Starting main load test...');
  
  // Create workers with ramp-up
  const workers = [];
  const rampUpDelay = (LOAD_TEST_CONFIG.ramp_up_time_seconds * 1000) / LOAD_TEST_CONFIG.concurrent_users;
  
  for (let i = 0; i < LOAD_TEST_CONFIG.concurrent_users; i++) {
    const delay = i * rampUpDelay;
    const worker = new Promise(resolve => {
      setTimeout(async () => {
        await runLoadTestWorker(i + 1, LOAD_TEST_CONFIG, metrics);
        resolve();
      }, delay);
    });
    workers.push(worker);
  }
  
  // Wait for all workers to complete
  await Promise.all(workers);
  
  // Get final statistics
  const stats = metrics.getStats();
  
  console.log('\n📈 Load Test Results');
  console.log('====================');
  console.log(`Duration: ${stats.duration.toFixed(2)}s`);
  console.log(`Total requests: ${stats.total_requests}`);
  console.log(`Successful requests: ${stats.successful_requests}`);
  console.log(`Failed requests: ${stats.failed_requests}`);
  console.log(`Requests/second: ${stats.requests_per_second.toFixed(2)}`);
  console.log(`Average response time: ${stats.avg_response_time.toFixed(2)}ms`);
  console.log(`Min response time: ${stats.min_response_time.toFixed(2)}ms`);
  console.log(`Max response time: ${stats.max_response_time.toFixed(2)}ms`);
  console.log(`95th percentile: ${stats.p95_response_time.toFixed(2)}ms`);
  console.log(`99th percentile: ${stats.p99_response_time.toFixed(2)}ms`);
  console.log(`Error rate: ${stats.error_rate.toFixed(2)}%`);
  
  return stats;
}

async function runPerformanceTestSuite() {
  console.log('🎯 Running Complete Performance Test Suite\n');
  
  // Initialize rate limit manager
  const rateLimitManager = new RateLimitManager();
  await rateLimitManager.connect();
  
  try {
    // Clear any existing rate limits before testing
    console.log('🧹 Clearing existing rate limits...');
    await rateLimitManager.clearAllRateLimits();
    
    // Test 1: Main load test
    const loadTestStats = await runMainLoadTest();
  
  // Test 2: Search performance
  const searchStats = await runConcurrentSearch();
  
  // Test 3: Authentication performance
  const authStats = await runAuthenticationPerformance();
  
  // Overall assessment
  console.log('\n🏆 Performance Assessment');
  console.log('==========================');
  
  const overallScore = calculatePerformanceScore(loadTestStats, searchStats, authStats);
  
  console.log(`Overall Performance Score: ${overallScore.score}/100`);
  console.log(`Assessment: ${overallScore.assessment}`);
  
  if (overallScore.recommendations.length > 0) {
    console.log('\n💡 Recommendations:');
    overallScore.recommendations.forEach(rec => console.log(`  - ${rec}`));
  }
  
    console.log('\n✅ Performance testing complete!');
    return overallScore.score >= 70; // Consider 70+ as passing
  } finally {
    // Clean up and disconnect
    await rateLimitManager.disconnect();
  }
}

function calculatePerformanceScore(loadStats, searchStats, authStats) {
  let score = 100;
  const recommendations = [];
  
  // Deduct points for high response times
  if (loadStats.avg_response_time > 200) {
    score -= 20;
    recommendations.push('Average response time is high (>200ms)');
  } else if (loadStats.avg_response_time > 100) {
    score -= 10;
    recommendations.push('Average response time could be improved (<100ms ideal)');
  }
  
  // Deduct points for high error rate
  if (loadStats.error_rate > 5) {
    score -= 30;
    recommendations.push('Error rate is too high (>5%)');
  } else if (loadStats.error_rate > 1) {
    score -= 15;
    recommendations.push('Error rate should be minimized (<1% ideal)');
  }
  
  // Deduct points for low throughput
  if (loadStats.requests_per_second < 10) {
    score -= 20;
    recommendations.push('Request throughput is low (<10 req/s)');
  }
  
  // Deduct points for high P95
  if (loadStats.p95_response_time > 500) {
    score -= 15;
    recommendations.push('95th percentile response time is high (>500ms)');
  }
  
  // Search performance
  if (searchStats.avg_response_time > 300) {
    score -= 10;
    recommendations.push('Search performance could be improved');
  }
  
  // Authentication performance
  if (authStats.avg_response_time > 1000) {
    score -= 10;
    recommendations.push('Authentication is slow (>1s)');
  }
  
  let assessment;
  if (score >= 90) assessment = 'Excellent';
  else if (score >= 80) assessment = 'Good';
  else if (score >= 70) assessment = 'Acceptable';
  else if (score >= 60) assessment = 'Needs Improvement';
  else assessment = 'Poor';
  
  return { score: Math.max(0, score), assessment, recommendations };
}

// Run the performance test suite
runPerformanceTestSuite()
  .then(success => {
    console.log(`\n${success ? '🎉' : '⚠️'} Performance testing ${success ? 'passed' : 'needs attention'}`);
    process.exit(success ? 0 : 1);
  })
  .catch(err => {
    console.error('💥 Performance testing crashed:', err);
    process.exit(1);
  });