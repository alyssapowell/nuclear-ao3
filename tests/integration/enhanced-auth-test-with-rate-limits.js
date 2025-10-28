#!/usr/bin/env node

/**
 * Enhanced Authentication Test with Rate Limit Management
 * Tests authentication functionality while properly managing rate limits
 */

const RateLimitManager = require('./test-rate-limit-manager');

const API_BASE = 'http://localhost:8080/api/v1';

class AuthTestSuite {
  constructor() {
    this.rateLimitManager = new RateLimitManager();
    this.testResults = [];
  }

  async initialize() {
    await this.rateLimitManager.connect();
    console.log('🔧 Clearing rate limits for clean testing...');
    await this.rateLimitManager.clearAllRateLimits();
  }

  async cleanup() {
    await this.rateLimitManager.disconnect();
  }

  async makeRequest(url, options = {}) {
    const startTime = Date.now();
    let status = 0;
    let error = null;
    let responseData = null;

    try {
      const response = await fetch(url, {
        method: 'GET',
        timeout: 10000,
        ...options
      });
      status = response.status;
      const text = await response.text();
      
      try {
        responseData = JSON.parse(text);
      } catch {
        responseData = text;
      }
    } catch (e) {
      error = e.message;
      status = 0;
    }

    const responseTime = Date.now() - startTime;
    return { responseTime, status, error, data: responseData };
  }

  recordTest(testName, success, details = {}) {
    this.testResults.push({
      test: testName,
      success,
      timestamp: new Date().toISOString(),
      ...details
    });
    
    const emoji = success ? '✅' : '❌';
    console.log(`${emoji} ${testName}: ${success ? 'PASSED' : 'FAILED'}`);
    if (details.error) {
      console.log(`   Error: ${details.error}`);
    }
    if (details.responseTime) {
      console.log(`   Response time: ${details.responseTime}ms`);
    }
  }

  async testUserRegistration() {
    console.log('\n🔐 Testing User Registration...');
    
    const testUser = {
      email: `testuser_${Date.now()}@example.com`,
      password: 'Test123!',
      username: `testuser_${Date.now()}`
    };

    const result = await this.makeRequest(`${API_BASE}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(testUser)
    });

    const success = result.status === 200 || result.status === 201;
    this.recordTest('User Registration', success, {
      status: result.status,
      responseTime: result.responseTime,
      error: result.error || (result.data?.error)
    });

    return { success, user: testUser, token: result.data?.access_token };
  }

  async testUserLogin(user) {
    console.log('\n🔑 Testing User Login...');
    
    const result = await this.makeRequest(`${API_BASE}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: user.email,
        password: user.password
      })
    });

    const success = result.status === 200;
    this.recordTest('User Login', success, {
      status: result.status,
      responseTime: result.responseTime,
      error: result.error || (result.data?.error)
    });

    return { success, token: result.data?.access_token };
  }

  async testProfileAccess(token) {
    console.log('\n👤 Testing Profile Access...');
    
    const result = await this.makeRequest(`${API_BASE}/auth/me`, {
      method: 'GET',
      headers: { 'Authorization': `Bearer ${token}` }
    });

    const success = result.status === 200;
    this.recordTest('Profile Access', success, {
      status: result.status,
      responseTime: result.responseTime,
      error: result.error || (result.data?.error)
    });

    return { success, profile: result.data };
  }

  async testRateLimitBehavior() {
    console.log('\n🚫 Testing Rate Limit Behavior...');
    
    // Make rapid requests to trigger rate limit
    const requests = [];
    const testEndpoint = `${API_BASE}/auth/me`;
    
    for (let i = 0; i < 105; i++) { // Exceed the 100 req/min limit
      requests.push(this.makeRequest(testEndpoint, {
        method: 'GET',
        headers: { 'Authorization': 'Bearer invalid' }
      }));
    }

    const results = await Promise.all(requests);
    const rateLimitedCount = results.filter(r => r.status === 429).length;
    const success = rateLimitedCount > 0;

    this.recordTest('Rate Limit Enforcement', success, {
      totalRequests: results.length,
      rateLimitedRequests: rateLimitedCount,
      responseTime: results.reduce((avg, r) => avg + r.responseTime, 0) / results.length
    });

    // Clear rate limits after test
    await this.rateLimitManager.clearAllRateLimits();
    
    return { success, rateLimitedCount };
  }

  async testPasswordChange(token) {
    console.log('\n🔒 Testing Password Change...');
    
    const result = await this.makeRequest(`${API_BASE}/auth/change-password`, {
      method: 'POST',
      headers: { 
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        current_password: 'Test123!',
        new_password: 'NewTest123!'
      })
    });

    const success = result.status === 200;
    this.recordTest('Password Change', success, {
      status: result.status,
      responseTime: result.responseTime,
      error: result.error || (result.data?.error)
    });

    return { success };
  }

  async testLogout(token) {
    console.log('\n🚪 Testing Logout...');
    
    const result = await this.makeRequest(`${API_BASE}/auth/logout`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}` }
    });

    const success = result.status === 200;
    this.recordTest('User Logout', success, {
      status: result.status,
      responseTime: result.responseTime,
      error: result.error || (result.data?.error)
    });

    return { success };
  }

  async runFullTestSuite() {
    console.log('🧪 Enhanced Authentication Test Suite with Rate Limit Management');
    console.log('====================================================================\n');

    await this.initialize();

    try {
      // Test 1: User Registration
      const registrationResult = await this.testUserRegistration();
      if (!registrationResult.success) {
        console.log('⚠️  Registration failed, cannot continue with authenticated tests');
        return this.generateReport();
      }

      // Test 2: User Login
      const loginResult = await this.testUserLogin(registrationResult.user);
      if (!loginResult.success) {
        console.log('⚠️  Login failed, cannot continue with authenticated tests');
        return this.generateReport();
      }

      const token = loginResult.token;

      // Test 3: Profile Access
      await this.testProfileAccess(token);

      // Test 4: Password Change
      await this.testPasswordChange(token);

      // Test 5: Rate Limit Behavior
      await this.testRateLimitBehavior();

      // Test 6: Logout
      await this.testLogout(token);

      return this.generateReport();
      
    } finally {
      await this.cleanup();
    }
  }

  generateReport() {
    console.log('\n📊 Enhanced Authentication Test Results');
    console.log('======================================');
    
    const passedTests = this.testResults.filter(r => r.success);
    const failedTests = this.testResults.filter(r => !r.success);
    
    console.log(`✅ Passed: ${passedTests.length}/${this.testResults.length}`);
    console.log(`❌ Failed: ${failedTests.length}/${this.testResults.length}`);
    console.log(`📈 Success Rate: ${((passedTests.length / this.testResults.length) * 100).toFixed(1)}%`);
    
    if (failedTests.length > 0) {
      console.log('\n❌ Failed Tests:');
      failedTests.forEach(test => {
        console.log(`   - ${test.test}: ${test.error || 'Unknown error'}`);
      });
    }

    console.log('\n📋 Test Details:');
    this.testResults.forEach(test => {
      const status = test.success ? '✅' : '❌';
      const time = test.responseTime ? ` (${test.responseTime}ms)` : '';
      console.log(`   ${status} ${test.test}${time}`);
    });

    const successRate = (passedTests.length / this.testResults.length) * 100;
    const overallSuccess = successRate >= 80; // 80% success rate threshold

    console.log(`\n${overallSuccess ? '🎉' : '⚠️'} Overall: ${overallSuccess ? 'PASSED' : 'NEEDS ATTENTION'}`);
    
    return {
      success: overallSuccess,
      successRate,
      totalTests: this.testResults.length,
      passedTests: passedTests.length,
      failedTests: failedTests.length,
      results: this.testResults
    };
  }
}

// Run the test suite
async function main() {
  const testSuite = new AuthTestSuite();
  const results = await testSuite.runFullTestSuite();
  
  process.exit(results.success ? 0 : 1);
}

if (require.main === module) {
  main().catch(console.error);
}

module.exports = AuthTestSuite;