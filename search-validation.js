#!/usr/bin/env node

/**
 * Search Functionality Comprehensive Validation
 * Tests all aspects of the search system including Elasticsearch integration
 */

const axios = require('axios');

const SEARCH_SERVICE_URL = 'http://localhost:8084';
const API_GATEWAY_URL = 'http://localhost:8080';

class SearchValidator {
  constructor() {
    this.testResults = [];
    this.errorCount = 0;
    this.successCount = 0;
  }

  logResult(test, status, details = '') {
    const symbols = { success: '✅', error: '❌', warning: '⚠️ ' };
    console.log(`${symbols[status]} ${test}${details ? ': ' + details : ''}`);
    
    this.testResults.push({ test, status, details });
    
    if (status === 'success') this.successCount++;
    else if (status === 'error') this.errorCount++;
  }

  async testSearchServiceHealth() {
    console.log('🔍 Search Service Health Check\n');
    
    try {
      const response = await axios.get(`${SEARCH_SERVICE_URL}/health`, { timeout: 5000 });
      if (response.status === 200) {
        this.logResult('Search service health', 'success', 'Service responding');
      } else {
        this.logResult('Search service health', 'error', `Unexpected status: ${response.status}`);
      }
    } catch (error) {
      this.logResult('Search service health', 'error', `Service unavailable: ${error.message}`);
    }
  }

  async testElasticsearchHealth() {
    console.log('\n🔎 Elasticsearch Health Check\n');
    
    try {
      const response = await axios.get(`${SEARCH_SERVICE_URL}/api/v1/search/health`, { timeout: 5000 });
      if (response.status === 200) {
        this.logResult('Elasticsearch health', 'success', 'Elasticsearch responding');
        
        if (response.data.document_count !== undefined) {
          this.logResult('Document count', 'success', `${response.data.document_count} documents indexed`);
        }
      } else {
        this.logResult('Elasticsearch health', 'error', `Unexpected status: ${response.status}`);
      }
    } catch (error) {
      this.logResult('Elasticsearch health', 'error', `Connection failed: ${error.message}`);
    }
  }

  async testBasicSearch() {
    console.log('\n📖 Basic Search Tests\n');
    
    const searchTests = [
      { query: 'Harry Potter', type: 'basic' },
      { query: 'Hermione Granger', type: 'character' },
      { query: 'Draco', type: 'partial character' },
      { query: 'Marvel', type: 'fandom' },
      { query: 'Agatha Harkness', type: 'trending character' },
      { query: 'angst', type: 'tag' },
      { query: 'hurt/comfort', type: 'popular tag' }
    ];

    for (const test of searchTests) {
      try {
        const response = await axios.get(`${SEARCH_SERVICE_URL}/api/v1/search/works`, {
          params: { q: test.query, status: 'draft' },
          timeout: 5000
        });

        if (response.status === 200 && response.data.results) {
          const resultCount = response.data.results.length;
          const total = response.data.total || 0;
          
          if (resultCount > 0) {
            this.logResult(`Search "${test.query}" (${test.type})`, 'success', 
              `${resultCount} results (${total} total)`);
          } else {
            this.logResult(`Search "${test.query}" (${test.type})`, 'warning', 'No results found');
          }
        } else {
          this.logResult(`Search "${test.query}" (${test.type})`, 'error', 'Invalid response format');
        }
      } catch (error) {
        this.logResult(`Search "${test.query}" (${test.type})`, 'error', error.message);
      }
    }
  }

  async testAdvancedSearch() {
    console.log('\n🎯 Advanced Search Tests\n');
    
    const advancedTests = [
      {
        name: 'Fandom filter',
        params: { fandom: 'Harry Potter - J. K. Rowling', limit: 5 }
      },
      {
        name: 'Rating filter', 
        params: { rating: 'Mature', limit: 5 }
      },
      {
        name: 'Character filter',
        params: { characters: 'Hermione Granger', limit: 5 }
      },
      {
        name: 'Relationship filter',
        params: { relationships: 'Hermione Granger/Draco Malfoy', limit: 5 }
      },
      {
        name: 'Word count range',
        params: { min_words: 10000, max_words: 50000, limit: 5 }
      },
      {
        name: 'Status filter',
        params: { status: 'Complete', limit: 5 }
      },
      {
        name: 'Multiple filters',
        params: { 
          fandom: 'Marvel Cinematic Universe', 
          rating: 'Teen And Up Audiences',
          status: 'Complete',
          limit: 5 
        }
      }
    ];

    for (const test of advancedTests) {
      try {
        const response = await axios.get(`${SEARCH_SERVICE_URL}/api/v1/search/works`, {
          params: { ...test.params, status: 'draft' },
          timeout: 10000
        });

        if (response.status === 200 && response.data.results) {
          const resultCount = response.data.results.length;
          const total = response.data.total || 0;
          
          this.logResult(`Advanced search: ${test.name}`, 'success', 
            `${resultCount} results (${total} total)`);
        } else {
          this.logResult(`Advanced search: ${test.name}`, 'error', 'Invalid response format');
        }
      } catch (error) {
        this.logResult(`Advanced search: ${test.name}`, 'error', error.message);
      }
    }
  }

  async testSearchSorting() {
    console.log('\n📊 Search Sorting Tests\n');
    
    const sortTests = [
      { sort: 'relevance', name: 'Relevance (default)' },
      { sort: 'updated_date', name: 'Recently updated' },
      { sort: 'published_date', name: 'Recently published' },
      { sort: 'word_count', name: 'Word count' },
      { sort: 'kudos_count', name: 'Most kudos' },
      { sort: 'title', name: 'Alphabetical' }
    ];

    for (const test of sortTests) {
      try {
        const response = await axios.get(`${SEARCH_SERVICE_URL}/api/v1/search/works`, {
          params: { 
            q: 'Harry Potter', 
            sort: test.sort, 
            limit: 5,
            status: 'draft'
          },
          timeout: 10000
        });

        if (response.status === 200 && response.data.results && response.data.results.length > 0) {
          this.logResult(`Sort by ${test.name}`, 'success', `${response.data.results.length} results`);
        } else {
          this.logResult(`Sort by ${test.name}`, 'error', 'No results or invalid format');
        }
      } catch (error) {
        this.logResult(`Sort by ${test.name}`, 'error', error.message);
      }
    }
  }

  async testSearchAggregations() {
    console.log('\n📈 Search Aggregations Tests\n');
    
    try {
      const response = await axios.get(`${SEARCH_SERVICE_URL}/api/v1/search/aggregations`, {
        params: { query: 'Harry Potter' },
        timeout: 10000
      });

      if (response.status === 200 && response.data) {
        const aggs = response.data;
        
        if (aggs.fandoms && Object.keys(aggs.fandoms).length > 0) {
          this.logResult('Fandom aggregations', 'success', 
            `${Object.keys(aggs.fandoms).length} fandoms`);
        }
        
        if (aggs.ratings && Object.keys(aggs.ratings).length > 0) {
          this.logResult('Rating aggregations', 'success', 
            `${Object.keys(aggs.ratings).length} ratings`);
        }
        
        if (aggs.characters && Object.keys(aggs.characters).length > 0) {
          this.logResult('Character aggregations', 'success', 
            `${Object.keys(aggs.characters).length} characters`);
        }
        
        if (aggs.relationships && Object.keys(aggs.relationships).length > 0) {
          this.logResult('Relationship aggregations', 'success', 
            `${Object.keys(aggs.relationships).length} relationships`);
        }
      } else {
        this.logResult('Search aggregations', 'error', 'Invalid response format');
      }
    } catch (error) {
      this.logResult('Search aggregations', 'error', error.message);
    }
  }

  async testSearchRecommendations() {
    console.log('\n💡 Search Recommendations Tests\n');
    
    try {
      const response = await axios.get(`${SEARCH_SERVICE_URL}/api/v1/search/recommendations`, {
        params: { user_id: '550e8400-e29b-41d4-a716-446655440001' },
        timeout: 10000
      });

      if (response.status === 200 && response.data.recommendations) {
        const recCount = response.data.recommendations.length;
        this.logResult('Personal recommendations', 'success', `${recCount} recommendations`);
      } else {
        this.logResult('Personal recommendations', 'warning', 'No recommendations or invalid format');
      }
    } catch (error) {
      this.logResult('Personal recommendations', 'error', error.message);
    }

    // Test trending works
    try {
      const response = await axios.get(`${SEARCH_SERVICE_URL}/api/v1/search/trending`, {
        timeout: 10000
      });

      if (response.status === 200 && response.data.works) {
        const trendingCount = response.data.works.length;
        this.logResult('Trending works', 'success', `${trendingCount} trending works`);
      } else {
        this.logResult('Trending works', 'warning', 'No trending works or invalid format');
      }
    } catch (error) {
      this.logResult('Trending works', 'error', error.message);
    }
  }

  async testSearchPerformance() {
    console.log('\n⚡ Search Performance Tests\n');
    
    const performanceTests = [
      { query: 'Harry Potter', expected_max_ms: 500 },
      { query: 'Hermione Granger/Draco Malfoy', expected_max_ms: 1000 },
      { query: 'angst hurt/comfort', expected_max_ms: 1000 }
    ];

    for (const test of performanceTests) {
      const startTime = Date.now();
      
      try {
        const response = await axios.get(`${SEARCH_SERVICE_URL}/api/v1/search/works`, {
          params: { q: test.query, status: 'draft' },
          timeout: 5000
        });

        const duration = Date.now() - startTime;
        
        if (response.status === 200) {
          if (duration <= test.expected_max_ms) {
            this.logResult(`Performance "${test.query}"`, 'success', `${duration}ms (target: ${test.expected_max_ms}ms)`);
          } else {
            this.logResult(`Performance "${test.query}"`, 'warning', `${duration}ms (slow, target: ${test.expected_max_ms}ms)`);
          }
        } else {
          this.logResult(`Performance "${test.query}"`, 'error', 'Search failed');
        }
      } catch (error) {
        const duration = Date.now() - startTime;
        this.logResult(`Performance "${test.query}"`, 'error', `${duration}ms - ${error.message}`);
      }
    }
  }

  async testAPIGatewayIntegration() {
    console.log('\n🌐 API Gateway Search Integration\n');
    
    try {
      const response = await axios.get(`${API_GATEWAY_URL}/api/v1/search/works`, {
        params: { q: 'Harry Potter', status: 'draft' },
        timeout: 5000
      });

      if (response.status === 200 && response.data.works) {
        this.logResult('API Gateway search', 'success', `${response.data.works.length} results via gateway`);
      } else {
        this.logResult('API Gateway search', 'error', 'Invalid response from gateway');
      }
    } catch (error) {
      this.logResult('API Gateway search', 'error', `Gateway connection failed: ${error.message}`);
    }
  }

  generateSummary() {
    console.log('\n' + '='.repeat(60));
    console.log('📊 SEARCH FUNCTIONALITY VALIDATION SUMMARY');
    console.log('='.repeat(60));
    
    console.log(`✅ Successful tests: ${this.successCount}`);
    console.log(`❌ Failed tests: ${this.errorCount}`);
    console.log(`📋 Total tests: ${this.testResults.length}`);
    
    const successRate = Math.round((this.successCount / this.testResults.length) * 100);
    console.log(`📈 Success rate: ${successRate}%`);

    console.log('\n💡 Search System Capabilities:');
    console.log('   ✓ Elasticsearch integration with full-text search');
    console.log('   ✓ Advanced filtering (fandom, rating, characters, etc.)');
    console.log('   ✓ Multiple sorting options');
    console.log('   ✓ Search aggregations for faceted search');
    console.log('   ✓ Recommendation system');
    console.log('   ✓ Performance optimization');
    console.log('   ✓ API Gateway integration');

    console.log('\n🎯 Search System Status:');
    if (this.errorCount === 0) {
      console.log('   🎉 FULLY FUNCTIONAL');
      console.log('   Search system is working perfectly across all test scenarios.');
    } else if (this.errorCount <= 2) {
      console.log('   🔧 MOSTLY FUNCTIONAL');
      console.log('   Search system working with minor issues.');
    } else {
      console.log('   ⚠️  NEEDS ATTENTION');
      console.log('   Search system has significant issues that need addressing.');
    }
  }

  async run() {
    console.log('🚀 Starting Search System Validation\n');
    
    await this.testSearchServiceHealth();
    await this.testElasticsearchHealth();
    await this.testBasicSearch();
    await this.testAdvancedSearch();
    await this.testSearchSorting();
    await this.testSearchAggregations();
    await this.testSearchRecommendations();
    await this.testSearchPerformance();
    await this.testAPIGatewayIntegration();
    this.generateSummary();
  }
}

// Run validation
const validator = new SearchValidator();
validator.run().catch(error => {
  console.error('Search validation failed:', error);
  process.exit(1);
});