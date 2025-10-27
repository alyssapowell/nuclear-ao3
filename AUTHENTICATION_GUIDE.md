# Authentication Guide for Nuclear AO3

## ✅ Authentication Issue Resolution

**Problem**: Users getting redirected to login when trying to access `/collections/new`  
**Status**: ✅ **RESOLVED** - This is correct behavior for protected routes

## 🔐 How Authentication Works

### Protected Routes
The following routes require authentication and will redirect to login:
- `/works/new` - Create new works
- `/works/edit` - Edit existing works  
- `/dashboard` - User dashboard
- `/bookmarks` - User bookmarks
- `/series/new` - Create new series
- `/collections/new` - Create new collections
- `/profile/*` - Profile management
- `/onboarding` - User onboarding

### Authentication Flow
1. **Middleware Protection**: `src/middleware.ts` checks for valid `auth_token` cookie
2. **Automatic Redirect**: Unauthenticated users are redirected to `/auth/login?redirect={original_url}`
3. **Post-Login Redirect**: After successful login, users are automatically redirected to their intended destination

## 👤 Test User Credentials

### Admin User
- **Email**: `admin@nuclear-ao3.com`
- **Password**: `password123`
- **Roles**: Admin, User

### How to Log In
1. Go to http://localhost:3001/auth/login
2. Enter credentials above
3. You'll be automatically redirected to your intended page

## 🔧 Technical Details

### Token Storage
- **Frontend**: Stores `auth_token` in both localStorage and cookies
- **Middleware**: Reads `auth_token` cookie for route protection
- **API**: Uses `Authorization: Bearer {token}` header for GraphQL requests

### Authentication States
- **Authenticated**: Has valid token, can access protected routes
- **Unauthenticated**: No token or invalid token, redirected to login
- **Expired Token**: Automatically handled, user prompted to re-login

## 🚨 Common Issues & Solutions

### Issue: "Still getting redirected to login after entering credentials"
**Solution**: 
1. Check that you're using the correct credentials: `admin@nuclear-ao3.com` / `password123`
2. Ensure cookies are enabled in your browser
3. Clear localStorage and cookies, then try logging in again

### Issue: "Login form shows but doesn't work"
**Solution**:
1. Verify backend services are running: `docker-compose ps`
2. Check API gateway health: `curl http://localhost:8080/health`
3. Verify database connection is working

### Issue: "Can't access collections after login"
**Solution**:
1. This usually means the login didn't complete successfully
2. Check browser developer tools for JavaScript errors
3. Try logging out and logging back in

## 🔄 Testing Authentication

### Quick Test Commands
```bash
# Test login API directly
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation Login($input: LoginInput!) { auth { login(input: $input) { token user { id username email } errors { message code } } } }",
    "variables": {
      "input": {
        "email": "admin@nuclear-ao3.com",
        "password": "password123"
      }
    }
  }'

# Test collections redirect behavior
curl -s http://localhost:3001/collections/new | head -10
# Should show: /auth/login?redirect=%2Fcollections%2Fnew
```

### Browser Testing
1. Open http://localhost:3001/collections/new
2. Should redirect to login page with redirect parameter
3. Login with admin credentials
4. Should be redirected back to collections/new

## 📋 System Status

### ✅ Working Components
- Authentication middleware (`src/middleware.ts`)
- Login endpoint (GraphQL mutation)
- Token generation and validation
- Protected route redirects
- Frontend login form
- Cookie-based authentication

### ⚠️ Notes
- The `me` query endpoint may have routing issues but doesn't affect core authentication
- E2E test users exist but may need password hash updates
- Some unit tests need AuthContext import fixes

## 🎯 Next Steps for Users

1. **Test the authentication flow**:
   - Visit http://localhost:3001/collections/new
   - Login with admin@nuclear-ao3.com / password123
   - Verify you can access the collections creation page

2. **Create your own account**:
   - Visit http://localhost:3001/auth/register
   - Use a valid email and strong password
   - Login with your new credentials

3. **Report issues**:
   - If authentication still doesn't work, check browser console for errors
   - Verify all Docker services are running healthy
   - Try clearing browser data and logging in again

## 🛠️ For Developers

### Password Hash Update
If you need to create new test users with `password123`:
```sql
UPDATE users 
SET password_hash = '$2a$10$8gOj1DJjiSl5JuloUEDuzOUzg00tkq8c9V6.VQVK7laoeUycdNR9e'
WHERE email = 'your-test-user@example.com';
```

### Debugging Authentication
1. Check middleware logs in browser network tab
2. Verify JWT token format and expiration
3. Test GraphQL authentication endpoints directly
4. Ensure cookies are being set correctly