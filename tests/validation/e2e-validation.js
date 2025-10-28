#!/usr/bin/env node

/**
 * Simple E2E validation script to verify Nuclear AO3 is working end-to-end
 * Tests: Backend APIs + Frontend Pages + Integration
 */

const http = require('http');
const https = require('https');

// Helper function to make HTTP requests
function makeRequest(url, options = {}) {
  return new Promise((resolve, reject) => {
    const lib = url.startsWith('https') ? https : http;
    const req = lib.request(url, options, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        resolve({
          status: res.statusCode,
          headers: res.headers,
          body: data
        });
      });
    });
    req.on('error', reject);
    if (options.body) {
      req.write(options.body);
    }
    req.end();
  });
}

async function validateSystem() {
  console.log('🚀 Nuclear AO3 E2E System Validation');
  console.log('=====================================\n');

  let passed = 0;
  let total = 0;

  // Test 1: Backend Health
  total++;
  console.log('1️⃣  Testing Backend Health...');
  try {
    const health = await makeRequest('http://localhost:8080/health');
    if (health.status === 200) {
      console.log('✅ Backend is healthy');
      passed++;
    } else {
      console.log('❌ Backend health check failed');
    }
  } catch (e) {
    console.log('❌ Backend unreachable:', e.message);
  }

  // Test 2: Frontend Homepage
  total++;
  console.log('\n2️⃣  Testing Frontend Homepage...');
  try {
    const homepage = await makeRequest('http://localhost:3001/');
    if (homepage.status === 200 && homepage.body.includes('Nuclear AO3')) {
      console.log('✅ Frontend homepage loads correctly');
      passed++;
    } else {
      console.log('❌ Frontend homepage failed');
    }
  } catch (e) {
    console.log('❌ Frontend unreachable:', e.message);
  }

  // Test 3: Works API
  total++;
  console.log('\n3️⃣  Testing Works API...');
  try {
    const works = await makeRequest('http://localhost:8080/api/v1/works/');
    if (works.status === 200) {
      const data = JSON.parse(works.body);
      console.log(`✅ Works API responds with ${data.length} works`);
      passed++;
    } else {
      console.log('❌ Works API failed');
    }
  } catch (e) {
    console.log('❌ Works API error:', e.message);
  }

  // Test 4: Frontend Works Page
  total++;
  console.log('\n4️⃣  Testing Frontend Works Page...');
  try {
    const worksPage = await makeRequest('http://localhost:3001/works');
    if (worksPage.status === 200 && worksPage.body.includes('Browse Works')) {
      console.log('✅ Frontend works page loads');
      passed++;
    } else {
      console.log('❌ Frontend works page failed');
    }
  } catch (e) {
    console.log('❌ Frontend works page error:', e.message);
  }

  // Test 5: Search API
  total++;
  console.log('\n5️⃣  Testing Search API...');
  try {
    const search = await makeRequest('http://localhost:8080/api/v1/search/works/?query=test');
    if (search.status === 200) {
      console.log('✅ Search API responds');
      passed++;
    } else {
      console.log('❌ Search API failed');
    }
  } catch (e) {
    console.log('❌ Search API error:', e.message);
  }

  // Test 6: Frontend Search Page
  total++;
  console.log('\n6️⃣  Testing Frontend Search Page...');
  try {
    const searchPage = await makeRequest('http://localhost:3001/search');
    if (searchPage.status === 200 && searchPage.body.includes('Enhanced Search')) {
      console.log('✅ Frontend search page loads');
      passed++;
    } else {
      console.log('❌ Frontend search page failed');
    }
  } catch (e) {
    console.log('❌ Frontend search page error:', e.message);
  }

  // Results
  console.log('\n📊 E2E Validation Results');
  console.log('==========================');
  console.log(`✅ Passed: ${passed}/${total}`);
  console.log(`📈 Success Rate: ${Math.round((passed/total) * 100)}%`);
  
  if (passed === total) {
    console.log('\n🎉 All systems operational! Nuclear AO3 is working end-to-end.');
    console.log('🚀 Ready for production use and further testing.');
  } else {
    console.log('\n⚠️  Some systems need attention. See failures above.');
  }

  return passed === total;
}

// Run validation
validateSystem()
  .then(success => process.exit(success ? 0 : 1))
  .catch(err => {
    console.error('💥 Validation crashed:', err);
    process.exit(1);
  });