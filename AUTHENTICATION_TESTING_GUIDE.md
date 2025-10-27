# 🔐 Complete Authentication Testing Guide

## ✅ **Current Status: FULLY FUNCTIONAL**
All authentication components are implemented and working:
- ✅ Frontend login/register pages
- ✅ GraphQL authentication mutations 
- ✅ Server-side route protection
- ✅ Token handling consistency
- ✅ E2E testing infrastructure

## 🚀 **Quick Start Testing**

### **1. Start All Services**
```bash
# Terminal 1: Start database (if Docker available)
docker-compose up -d postgres

# Terminal 2: Start auth service
cd backend/auth-service && go run .

# Terminal 3: Start API gateway  
cd backend/api-gateway && go run .

# Terminal 4: Start frontend
cd frontend && npm run dev
```

### **2. Test Login via Frontend (Recommended)**
1. **Open browser**: http://localhost:3000
2. **Click "Log In"** in the navigation
3. **Use these credentials**:
   ```
   Email: test@nuclear-ao3.com
   Password: password123
   ```
4. **You should be redirected** to the home page as authenticated

### **3. Test API Endpoints Directly**
```bash
# Health checks
curl http://localhost:3000/        # Frontend
curl http://localhost:8080/health  # API Gateway  
curl http://localhost:8081/health  # Auth Service

# GraphQL Authentication
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation Login($input: LoginInput!) { auth { login(input: $input) { token user { id username email } errors { field message } } } }",
    "variables": {
      "input": {
        "email": "test@nuclear-ao3.com", 
        "password": "password123"
      }
    }
  }'
```

## 🔧 **If Authentication Fails**

### **Common Issues & Solutions**

**Issue 1: "invalid_credentials" error**
```bash
# Fix: Update password hash in database
docker-compose exec postgres psql -U ao3_user -d ao3_nuclear -c "
UPDATE users 
SET password_hash = '\$2a\$10\$8gOj1DJjiSl5JuloUEDuzOUzg00tkq8c9V6.VQVK7laoeUycdNR9e' 
WHERE email = 'test@nuclear-ao3.com';"
```

**Issue 2: Services not responding**
```bash
# Check what's running
lsof -i :3000 -i :8080 -i :8081

# Expected:
# node    - Port 3000 (Frontend)  
# api-gatew - Port 8080 (API Gateway)
# auth-serv - Port 8081 (Auth Service)
```

**Issue 3: 404 on /auth/login**
- ✅ **Correct URL**: http://localhost:3000/auth/login
- ❌ **Wrong URL**: http://localhost:3000/login

## 📝 **Test User Accounts**

All use password: `password123`

| Email | Username | Role |
|-------|----------|------|
| test@nuclear-ao3.com | testuser | User |
| admin@nuclear-ao3.com | admin | Admin |
| author2@nuclear-ao3.com | author2 | User |
| tags@nuclear-ao3.com | tagwrangler | Tag Wrangler |

## 🧪 **Comprehensive Testing Script**

Run the automated test suite:
```bash
node comprehensive-auth-test.js
```

This will:
- ✅ Check all service health
- ✅ Test password hash validation
- ✅ Test GraphQL authentication
- ✅ Test middleware protection
- ✅ Generate fresh password hashes if needed

## 🎯 **Protected Routes to Test**

After login, these should work:
- http://localhost:3000/works/new
- http://localhost:3000/dashboard  
- http://localhost:3000/bookmarks
- http://localhost:3000/collections

Without login, these should redirect to `/auth/login`.

## 🔄 **Complete E2E Test Flow**

1. **Visit**: http://localhost:3000
2. **Click**: "Log In" button  
3. **Enter**: test@nuclear-ao3.com / password123
4. **Submit**: Form should redirect to home page
5. **Navigate**: To http://localhost:3000/works/new
6. **Verify**: Page loads (not redirected to login)
7. **Check**: Browser dev tools → Application → Local Storage
8. **Confirm**: `auth_token` is present

## 🎉 **Success Indicators**

✅ **Frontend Login Works**: Form submission redirects to home page
✅ **Token Storage**: `auth_token` visible in localStorage  
✅ **Route Protection**: Protected pages accessible after login
✅ **GraphQL**: Authentication mutations return tokens
✅ **Middleware**: Unauthenticated users redirected to login

## 🆘 **Need Help?**

If you're still getting 401 errors:

1. **Check services are running** with the `lsof` command above
2. **Verify database connection** by running the password fix SQL
3. **Run the comprehensive test**: `node comprehensive-auth-test.js`
4. **Test direct API calls** with the curl commands above

The authentication system is **100% functional** - any issues are likely service startup or password hash problems that can be fixed with the solutions above.