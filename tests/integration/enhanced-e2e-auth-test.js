#!/usr/bin/env node

/**
 * Enhanced E2E Authentication Flow Tests
 * Tests complete user journeys including:
 * - User registration and login
 * - Authenticated API operations  
 * - Work creation and publishing
 * - Bookmark management
 * - Series management
 */

const API_BASE = 'http://localhost:8080/api/v1';

const TEST_USER = {
  email: `testuser_${Date.now()}@example.com`,
  password: 'TestPassword123!',
  username: `testuser_${Date.now()}`
};

let authToken = null;
let userId = null;
let createdWorkId = null;
let createdSeriesId = null;

async function apiRequest(endpoint, options = {}) {
  const url = `${API_BASE}${endpoint}`;
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers
  };
  
  if (authToken && !options.skipAuth) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }
  
  console.log(`📡 ${options.method || 'GET'} ${endpoint}`);
  
  const response = await fetch(url, {
    method: 'GET',
    ...options,
    headers
  });
  
  // Get response text first, then parse
  const responseText = await response.text();
  let data;
  
  try {
    data = JSON.parse(responseText);
  } catch (e) {
    data = responseText;
  }
  
  if (!response.ok) {
    throw new Error(`API Error ${response.status}: ${typeof data === 'object' ? data.error || response.statusText : data}`);
  }
  
  return data;
}

async function testUserRegistration() {
  console.log('\n🔐 Testing User Registration...');
  try {
    const result = await apiRequest('/auth/register', {
      method: 'POST',
      body: JSON.stringify(TEST_USER),
      skipAuth: true
    });
    
    console.log('✅ User registration successful');
    console.log(`   User ID: ${result.user?.id || 'N/A'}`);
    userId = result.user?.id;
    return true;
  } catch (e) {
    console.log(`❌ Registration failed: ${e.message}`);
    console.log('ℹ️  This may be expected if user already exists');
    return false;
  }
}

async function testUserLogin() {
  console.log('\n🔑 Testing User Login...');
  try {
    const result = await apiRequest('/auth/login', {
      method: 'POST',
      body: JSON.stringify({
        email: TEST_USER.email,
        password: TEST_USER.password
      }),
      skipAuth: true
    });
    
    authToken = result.token || result.access_token;
    userId = result.user?.id || userId;
    
    if (authToken) {
      console.log('✅ Login successful');
      console.log(`   Token: ${authToken.substring(0, 20)}...`);
      return true;
    } else {
      console.log('❌ Login failed: No token received');
      return false;
    }
  } catch (e) {
    console.log(`❌ Login failed: ${e.message}`);
    return false;
  }
}

async function testAuthenticatedProfile() {
  console.log('\n👤 Testing Authenticated Profile Access...');
  try {
    const profile = await apiRequest('/auth/profile');
    console.log('✅ Profile access successful');
    console.log(`   Username: ${profile.username || 'N/A'}`);
    console.log(`   Email: ${profile.email || 'N/A'}`);
    return true;
  } catch (e) {
    console.log(`❌ Profile access failed: ${e.message}`);
    return false;
  }
}

async function testWorkCreation() {
  console.log('\n📝 Testing Authenticated Work Creation...');
  try {
    const workData = {
      title: `Test Work - Auth Flow ${Date.now()}`,
      summary: 'This is a test work created during authenticated E2E testing.',
      content: 'Chapter 1\n\nThis is the content of the test work for authentication flow testing.',
      tags: ['test', 'e2e', 'authentication'],
      characters: ['Test Character'],
      relationships: ['Test Character/Reader'],
      fandoms: ['Test Fandom'],
      rating: 'General Audiences',
      warnings: ['No Archive Warnings Apply'],
      categories: ['Gen'],
      language: 'English',
      status: 'completed'
    };
    
    const result = await apiRequest('/works', {
      method: 'POST',
      body: JSON.stringify(workData)
    });
    
    createdWorkId = result.id || result.work?.id;
    console.log('✅ Work creation successful');
    console.log(`   Work ID: ${createdWorkId}`);
    console.log(`   Title: "${result.title || workData.title}"`);
    return true;
  } catch (e) {
    console.log(`❌ Work creation failed: ${e.message}`);
    return false;
  }
}

async function testWorkRetrieval() {
  console.log('\n📚 Testing Work Retrieval...');
  if (!createdWorkId) {
    console.log('⚠️  Skipping - no work ID available');
    return false;
  }
  
  try {
    const work = await apiRequest(`/works/${createdWorkId}`);
    console.log('✅ Work retrieval successful');
    console.log(`   Title: "${work.title}"`);
    console.log(`   Author: ${work.author_name || work.user?.username || 'N/A'}`);
    return true;
  } catch (e) {
    console.log(`❌ Work retrieval failed: ${e.message}`);
    return false;
  }
}

async function testBookmarkCreation() {
  console.log('\n🔖 Testing Bookmark Creation...');
  if (!createdWorkId) {
    console.log('⚠️  Skipping - no work ID available');
    return false;
  }
  
  try {
    const bookmarkData = {
      work_id: createdWorkId,
      notes: 'Great test work!',
      tags: ['favorite', 'test'],
      private: false
    };
    
    const result = await apiRequest('/bookmarks', {
      method: 'POST',
      body: JSON.stringify(bookmarkData)
    });
    
    console.log('✅ Bookmark creation successful');
    console.log(`   Bookmark ID: ${result.id || 'N/A'}`);
    return true;
  } catch (e) {
    console.log(`❌ Bookmark creation failed: ${e.message}`);
    return false;
  }
}

async function testSeriesCreation() {
  console.log('\n📖 Testing Series Creation...');
  try {
    const seriesData = {
      title: `Test Series - Auth Flow ${Date.now()}`,
      summary: 'This is a test series created during authenticated E2E testing.',
      notes: 'Series notes for testing'
    };
    
    const result = await apiRequest('/series', {
      method: 'POST',
      body: JSON.stringify(seriesData)
    });
    
    createdSeriesId = result.id || result.series?.id;
    console.log('✅ Series creation successful');
    console.log(`   Series ID: ${createdSeriesId}`);
    console.log(`   Title: "${result.title || seriesData.title}"`);
    return true;
  } catch (e) {
    console.log(`❌ Series creation failed: ${e.message}`);
    return false;
  }
}

async function testMyBookmarks() {
  console.log('\n📋 Testing My Bookmarks Retrieval...');
  try {
    const bookmarks = await apiRequest('/bookmarks');
    console.log('✅ Bookmarks retrieval successful');
    console.log(`   Total bookmarks: ${Array.isArray(bookmarks) ? bookmarks.length : (bookmarks.total || 'N/A')}`);
    return true;
  } catch (e) {
    console.log(`❌ Bookmarks retrieval failed: ${e.message}`);
    return false;
  }
}

async function testMyWorks() {
  console.log('\n📄 Testing My Works Retrieval...');
  try {
    const works = await apiRequest('/works?author=me');
    console.log('✅ My works retrieval successful');
    console.log(`   My works count: ${Array.isArray(works) ? works.length : (works.total || 'N/A')}`);
    return true;
  } catch (e) {
    console.log(`❌ My works retrieval failed: ${e.message}`);
    return false;
  }
}

async function runEnhancedE2ETests() {
  console.log('🚀 Enhanced E2E Authentication Flow Tests');
  console.log('==========================================\n');
  
  let passed = 0;
  let total = 0;
  
  // Test 1: User Registration
  total++;
  if (await testUserRegistration()) passed++;
  
  // Test 2: User Login
  total++;
  if (await testUserLogin()) passed++;
  
  // Skip remaining tests if login failed
  if (!authToken) {
    console.log('\n⚠️  Skipping authenticated tests - login failed');
    console.log('\n📊 Enhanced E2E Test Results');
    console.log('===============================');
    console.log(`✅ Passed: ${passed}/${total}`);
    console.log(`📈 Success Rate: ${Math.round((passed/total) * 100)}%`);
    return passed === total;
  }
  
  // Test 3: Profile Access
  total++;
  if (await testAuthenticatedProfile()) passed++;
  
  // Test 4: Work Creation
  total++;
  if (await testWorkCreation()) passed++;
  
  // Test 5: Work Retrieval
  total++;
  if (await testWorkRetrieval()) passed++;
  
  // Test 6: Series Creation
  total++;
  if (await testSeriesCreation()) passed++;
  
  // Test 7: Bookmark Creation
  total++;
  if (await testBookmarkCreation()) passed++;
  
  // Test 8: My Bookmarks
  total++;
  if (await testMyBookmarks()) passed++;
  
  // Test 9: My Works
  total++;
  if (await testMyWorks()) passed++;
  
  // Results
  console.log('\n📊 Enhanced E2E Test Results');
  console.log('===============================');
  console.log(`✅ Passed: ${passed}/${total}`);
  console.log(`📈 Success Rate: ${Math.round((passed/total) * 100)}%`);
  
  if (passed === total) {
    console.log('\n🎉 All authenticated user flows working perfectly!');
    console.log('🔐 Authentication system fully functional');
    console.log('📝 Content creation and management working');
    console.log('🔖 Bookmark system operational');
  } else {
    console.log('\n⚠️  Some authenticated flows need attention.');
    console.log('💡 Check API endpoints and authentication implementation');
  }
  
  return passed === total;
}

// Run the enhanced E2E tests
runEnhancedE2ETests()
  .then(success => process.exit(success ? 0 : 1))
  .catch(err => {
    console.error('💥 Enhanced E2E tests crashed:', err);
    process.exit(1);
  });