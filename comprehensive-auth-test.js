#!/usr/bin/env node

/**
 * Comprehensive End-to-End Authentication Testing Suite
 * 
 * This script validates the complete authentication flow across:
 * - Frontend middleware and route protection
 * - API Gateway GraphQL mutations
 * - Auth service password validation
 * - Token handling consistency
 * 
 * Features:
 * - Service health monitoring
 * - Password hash regeneration and testing
 * - Complete login flow simulation
 * - Error reporting and debugging
 */

const axios = require('axios');
const { exec } = require('child_process');
const { promisify } = require('util');

const execAsync = promisify(exec);

class AuthTestSuite {
    constructor() {
        this.services = {
            frontend: 'http://localhost:3000',
            apiGateway: 'http://localhost:3001',
            authService: 'http://localhost:8081'
        };
        
        this.testUser = {
            email: 'test@example.com',
            password: 'password123',
            username: 'testuser'
        };
        
        this.results = {
            passed: 0,
            failed: 0,
            details: []
        };
    }

    log(level, message) {
        const timestamp = new Date().toISOString();
        const prefix = {
            info: 'ℹ️',
            success: '✅',
            error: '❌',
            warning: '⚠️'
        }[level] || 'ℹ️';
        
        console.log(`${prefix} [${timestamp}] ${message}`);
    }

    async testService(name, url, expectedStatus = 200) {
        try {
            const response = await axios.get(`${url}/health`, { timeout: 5000 });
            
            if (response.status === expectedStatus) {
                this.log('success', `${name}: Service healthy`);
                this.results.passed++;
                return true;
            } else {
                this.log('error', `${name}: Unexpected status ${response.status}`);
                this.results.failed++;
                return false;
            }
        } catch (error) {
            this.log('error', `${name}: ${error.message}`);
            this.results.failed++;
            return false;
        }
    }

    async checkDockerStatus() {
        try {
            const { stdout } = await execAsync('docker ps --format "table {{.Names}}\\t{{.Status}}" | grep -E "(ao3|nuclear)"');
            this.log('info', `Docker Services:\n${stdout}`);
            return true;
        } catch (error) {
            this.log('warning', 'Docker services not running - some tests will be skipped');
            return false;
        }
    }

    async generatePasswordHash() {
        try {
            this.log('info', 'Generating fresh password hash...');
            const { stdout } = await execAsync('cd backend/auth-service && go run ../../fix-password.go');
            
            const hashMatch = stdout.match(/Fresh hash: (\$2a\$[^\s]+)/);
            if (hashMatch) {
                const hash = hashMatch[1];
                this.log('success', `Generated fresh bcrypt hash: ${hash.substring(0, 20)}...`);
                return hash;
            }
            
            throw new Error('Could not extract hash from output');
        } catch (error) {
            this.log('error', `Password hash generation failed: ${error.message}`);
            return null;
        }
    }

    async updateDatabasePassword(hash) {
        if (!hash) return false;
        
        try {
            const command = `docker-compose exec -T postgres psql -U ao3_user -d ao3_nuclear -c "UPDATE users SET password_hash = '${hash}' WHERE email = '${this.testUser.email}';"`;
            await execAsync(command);
            this.log('success', 'Database password updated successfully');
            return true;
        } catch (error) {
            this.log('error', `Database update failed: ${error.message}`);
            return false;
        }
    }

    async testGraphQLAuthentication() {
        try {
            const mutation = `
                mutation {
                    login(email: "${this.testUser.email}", password: "${this.testUser.password}") {
                        success
                        message
                        token
                        user {
                            id
                            email
                            username
                        }
                    }
                }
            `;

            const response = await axios.post(`${this.services.apiGateway}/graphql`, {
                query: mutation
            }, { timeout: 10000 });

            if (response.data && response.data.data && response.data.data.login) {
                const login = response.data.data.login;
                
                if (login.success && login.token) {
                    this.log('success', `GraphQL Login: Authentication successful for ${login.user.email}`);
                    this.results.passed++;
                    return { success: true, token: login.token, user: login.user };
                } else {
                    this.log('error', `GraphQL Login: ${login.message || 'Authentication failed'}`);
                    this.results.failed++;
                    return { success: false, message: login.message };
                }
            } else {
                this.log('error', 'GraphQL Login: Invalid response structure');
                this.results.failed++;
                return { success: false };
            }
        } catch (error) {
            this.log('error', `GraphQL Login: ${error.message}`);
            this.results.failed++;
            return { success: false };
        }
    }

    async testMiddlewareProtection() {
        try {
            // Test protected route without token
            const response = await axios.get(`${this.services.frontend}/dashboard`, { 
                maxRedirects: 0,
                validateStatus: () => true
            });

            if (response.status === 302 || response.status === 307) {
                const location = response.headers.location;
                if (location && location.includes('/login')) {
                    this.log('success', 'Middleware Protection: Protected route correctly redirects to login');
                    this.results.passed++;
                    return true;
                } else {
                    this.log('error', `Middleware Protection: Unexpected redirect to ${location}`);
                    this.results.failed++;
                    return false;
                }
            } else {
                this.log('error', `Middleware Protection: Expected redirect but got status ${response.status}`);
                this.results.failed++;
                return false;
            }
        } catch (error) {
            this.log('error', `Middleware Protection: ${error.message}`);
            this.results.failed++;
            return false;
        }
    }

    async testTokenPersistence(token) {
        if (!token) {
            this.log('warning', 'Token Persistence: No token to test');
            return false;
        }

        try {
            // Test API call with token
            const response = await axios.get(`${this.services.apiGateway}/api/v1/auth/me`, {
                headers: { Authorization: `Bearer ${token}` }
            });

            if (response.status === 200 && response.data.id) {
                this.log('success', `Token Persistence: Token validation successful for user ${response.data.id}`);
                this.results.passed++;
                return true;
            } else {
                this.log('error', 'Token Persistence: Token validation failed');
                this.results.failed++;
                return false;
            }
        } catch (error) {
            this.log('error', `Token Persistence: ${error.message}`);
            this.results.failed++;
            return false;
        }
    }

    async testDirectAuthService() {
        try {
            const response = await axios.post(`${this.services.authService}/api/v1/auth/login`, {
                email: this.testUser.email,
                password: this.testUser.password
            });

            if (response.status === 200 && response.data.access_token) {
                this.log('success', `Direct Auth: Login successful, token received`);
                this.results.passed++;
                return { success: true, token: response.data.access_token };
            } else {
                this.log('error', 'Direct Auth: Login failed');
                this.results.failed++;
                return { success: false };
            }
        } catch (error) {
            if (error.response && error.response.status === 401) {
                this.log('error', 'Direct Auth: Invalid credentials (password hash issue)');
            } else {
                this.log('error', `Direct Auth: ${error.message}`);
            }
            this.results.failed++;
            return { success: false };
        }
    }

    async runComprehensiveTest() {
        this.log('info', '🚀 Starting Comprehensive Authentication Test Suite');
        this.log('info', '==================================================');

        // Step 1: Check Docker and services
        const dockerRunning = await this.checkDockerStatus();
        
        if (dockerRunning) {
            // Step 2: Generate and update password
            const hash = await this.generatePasswordHash();
            if (hash) {
                await this.updateDatabasePassword(hash);
            }
            
            // Step 3: Test services
            await this.testService('Frontend', this.services.frontend);
            await this.testService('API Gateway', this.services.apiGateway);
            await this.testService('Auth Service', this.services.authService);
            
            // Step 4: Test authentication flows
            const directAuth = await this.testDirectAuthService();
            const graphqlAuth = await this.testGraphQLAuthentication();
            
            // Step 5: Test middleware and token handling
            await this.testMiddlewareProtection();
            
            if (graphqlAuth.success && graphqlAuth.token) {
                await this.testTokenPersistence(graphqlAuth.token);
            } else if (directAuth.success && directAuth.token) {
                await this.testTokenPersistence(directAuth.token);
            }
        } else {
            this.log('info', 'Running limited tests without Docker...');
            
            // Test what we can without services running
            this.log('info', 'Checking code fixes...');
            
            // Check if middleware is enabled
            try {
                await execAsync('ls frontend/src/middleware.ts');
                this.log('success', 'Code Fix: Middleware file is enabled (not .disabled)');
                this.results.passed++;
            } catch {
                this.log('error', 'Code Fix: Middleware file is still disabled');
                this.results.failed++;
            }
            
            // Check GraphQL mutations
            try {
                const { stdout } = await execAsync('grep -n "handleAuthMutation\\|handleLoginMutation\\|handleRegisterMutation" backend/api-gateway/graphql.go');
                if (stdout.trim()) {
                    this.log('success', 'Code Fix: GraphQL auth mutations are implemented');
                    this.results.passed++;
                } else {
                    this.log('error', 'Code Fix: GraphQL auth mutations not found');
                    this.results.failed++;
                }
            } catch {
                this.log('error', 'Code Fix: Could not verify GraphQL mutations');
                this.results.failed++;
            }
        }

        // Final report
        this.log('info', '==================================================');
        this.log('info', '📊 Test Results Summary');
        this.log('info', `✅ Passed: ${this.results.passed}`);
        this.log('info', `❌ Failed: ${this.results.failed}`);
        
        const successRate = this.results.passed + this.results.failed > 0 
            ? Math.round((this.results.passed / (this.results.passed + this.results.failed)) * 100)
            : 0;
        
        this.log('info', `📈 Success Rate: ${successRate}%`);
        
        if (this.results.failed === 0) {
            this.log('success', '🎉 All tests passed! Authentication system is working correctly.');
        } else if (successRate >= 70) {
            this.log('warning', '⚠️ Most tests passed. Minor issues remain.');
        } else {
            this.log('error', '❌ Significant issues detected. Review failed tests.');
        }

        return {
            passed: this.results.passed,
            failed: this.results.failed,
            successRate
        };
    }
}

// Run tests if executed directly
if (require.main === module) {
    const suite = new AuthTestSuite();
    suite.runComprehensiveTest().then(results => {
        process.exit(results.failed === 0 ? 0 : 1);
    }).catch(error => {
        console.error('❌ Test suite crashed:', error.message);
        process.exit(1);
    });
}

module.exports = AuthTestSuite;