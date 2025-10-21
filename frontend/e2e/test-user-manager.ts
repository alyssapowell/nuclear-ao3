import { expect } from '@playwright/test';

export class TestUserManager {
  private static userCounter = 0;
  private static workerUsers = new Map<string, number>();

  static async createUniqueUser(page: any, testInfo: any) {
    // Get worker index for isolation
    const workerId = testInfo.workerIndex || 0;
    
    // Initialize counter for this worker if not exists
    if (!this.workerUsers.has(workerId.toString())) {
      this.workerUsers.set(workerId.toString(), 0);
    }
    
    // Increment counter for this worker
    const workerCounter = this.workerUsers.get(workerId.toString())! + 1;
    this.workerUsers.set(workerId.toString(), workerCounter);
    
    // Create truly unique user ID
    const timestamp = Date.now();
    const random = Math.floor(Math.random() * 10000);
    const uniqueId = `w${workerId}_${workerCounter}_${timestamp}_${random}`;
    
    const testUser = {
      username: `testuser_${uniqueId}`,
      email: `test_${uniqueId}@example.com`,
      password: 'testpassword123'
    };

    // Register user with retry logic
    let authToken: string | undefined;
    let userId: string | undefined;
    
    for (let attempt = 0; attempt < 3; attempt++) {
      try {
        const registerResponse = await page.request.post('http://localhost:8080/api/v1/auth/register', {
          data: testUser
        });
        
        if (registerResponse.ok()) {
          console.log(`✅ Registered user: ${testUser.username}`);
          
          // Login immediately after successful registration
          const loginResponse = await page.request.post('http://localhost:8080/api/v1/auth/login', {
            data: {
              username: testUser.username,
              password: testUser.password
            }
          });
          
          if (loginResponse.ok()) {
            const loginData = await loginResponse.json();
            console.log(`✅ Login successful for: ${testUser.username}`);
            authToken = loginData.access_token || loginData.token;
            userId = loginData.user.id;
            break;
          } else {
            const loginError = await loginResponse.text();
            console.log(`❌ Login failed for ${testUser.username}: ${loginError}`);
          }
        } else {
          const registerError = await registerResponse.text();
          console.log(`❌ Registration failed for ${testUser.username}: ${registerError}`);
        }
        
        // If registration failed, try with different username
        if (attempt < 2) {
          const retryId = `${uniqueId}_retry${attempt}`;
          testUser.username = `testuser_${retryId}`;
          testUser.email = `test_${retryId}@example.com`;
        }
      } catch (error) {
        if (attempt === 2) throw error;
      }
    }
    
    if (!authToken || !userId) {
      throw new Error(`Failed to create test user after 3 attempts. Last attempted: ${testUser.username}`);
    }
    
    return {
      user: testUser,
      authToken,
      userId
    };
  }

  static async setAuthInBrowser(page: any, authToken: string) {
    await page.goto('/');
    await page.evaluate((token) => {
      localStorage.setItem('auth_token', token);
      localStorage.setItem('token', token);
    }, authToken);
  }
}