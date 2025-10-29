# Nuclear AO3 Development Workflow

This document outlines the proper development workflow to maintain code quality and prevent the issues that occurred in previous sessions.

## 🚫 Never Again: What Went Wrong

Previously, we had TypeScript errors that should have been caught early but weren't because:

1. **Temporary workarounds were left in place** - Build error ignoring was enabled for "quick deployment" but never removed
2. **Local vs Production gap** - Development environment was too forgiving compared to production builds  
3. **No systematic validation** - No pre-commit hooks or CI/CD pipeline to catch issues early
4. **Technical debt accumulation** - Issues were deferred instead of fixed immediately

## ✅ Quality-First Development Process

### 1. Local Development Setup

Before starting any development work:

```bash
cd frontend
npm install
npm run validate  # Runs both TypeScript check and ESLint
```

### 2. Code Validation Commands

**Essential commands to run regularly:**

```bash
# Type checking only
npm run type-check

# Linting only  
npm run lint

# Full validation (both TypeScript + ESLint)
npm run validate

# Full build test
npm run build
```

### 3. Pre-Commit Validation

**Automatic validation is set up** via Git pre-commit hooks at `.git/hooks/pre-commit`.

Every commit will automatically:
- ✅ Run TypeScript checking (`npm run type-check`)
- ✅ Run ESLint validation (`npm run lint`)
- ❌ **Block the commit** if any errors are found

### 4. Commit Workflow

```bash
# 1. Make your changes
# 2. Stage your files
git add .

# 3. Attempt commit (pre-commit hook runs automatically)
git commit -m "Your commit message"

# 4. If pre-commit fails:
#    - Fix all TypeScript errors
#    - Fix all ESLint errors  
#    - Run `npm run validate` to verify
#    - Try commit again
```

### 5. CI/CD Pipeline

GitHub Actions automatically runs on every push/PR to `main` or `develop`:

- ✅ TypeScript validation
- ✅ ESLint validation  
- ✅ Full build test
- ✅ Unit tests

**All checks must pass before code can be merged.**

## 🔧 TypeScript Configuration

### Current Setup

- **Strict mode enabled** - No type errors allowed
- **Test files excluded** - Only main application code is type-checked during builds
- **Build-time validation** - TypeScript errors block production builds

### Excluded from Type Checking

- `e2e/**` - End-to-end tests
- `src/__tests__/**` - Unit tests  
- `**/__tests__/**` - Component tests
- `**/*.test.*` - Test files
- `**/*.spec.*` - Spec files

### Common Type Issues to Watch For

1. **`string | null` vs `string | undefined`** - Convert with `value || undefined`
2. **Missing type imports** - Import types from `@/lib/api`
3. **Untyped state** - Always provide type annotations for `useState<Type[]>([])`
4. **Component prop mismatches** - Ensure prop types match component definitions

## 🎯 Code Quality Standards

### TypeScript
- ❌ **No `any` types** unless absolutely necessary
- ❌ **No `@ts-ignore` comments** 
- ✅ **Explicit type annotations** for state and complex objects
- ✅ **Proper null/undefined handling**

### ESLint
- ✅ **Fix all linting errors** before committing
- ✅ **Follow existing code patterns**
- ✅ **Use consistent naming conventions**

### Component Development
- ✅ **Check existing components** for patterns before creating new ones
- ✅ **Ensure prop types match** between definition and usage
- ✅ **Import types properly** from shared libraries

## 🚨 Emergency Procedures

### If Build Fails in Production

**Never disable TypeScript checking.** Instead:

1. **Identify the root cause** - Run `npm run type-check` locally
2. **Fix the actual issue** - Don't ignore or suppress
3. **Test the fix** - Run `npm run build` locally
4. **Deploy the fix** - Only deploy after local validation passes

### If Pre-Commit Hook Fails

**Never bypass the pre-commit hook.** Instead:

1. **Read the error messages** carefully
2. **Fix each issue** one by one
3. **Verify the fix** with `npm run validate`
4. **Commit only after** all issues are resolved

## 📚 Commands Reference

| Command | Purpose | When to Use |
|---------|---------|-------------|
| `npm run type-check` | TypeScript validation only | During development |
| `npm run lint` | ESLint validation only | During development |
| `npm run validate` | Both TypeScript + ESLint | Before committing |
| `npm run build` | Full production build | Before deployment |
| `npm run dev` | Development server | During development |

## 🔄 Integration with Future Sessions

**For future Claude sessions:**

1. **Always run `npm run validate`** before making changes
2. **Fix any existing issues first** before adding new features  
3. **Test builds locally** before considering anything "complete"
4. **Never disable error checking** as a workaround
5. **Use this document** as a reference for proper workflow

## 🎉 Success Criteria

A feature is only considered complete when:

- ✅ `npm run validate` passes with zero errors
- ✅ `npm run build` completes successfully  
- ✅ Pre-commit hook allows commits
- ✅ CI/CD pipeline passes all checks
- ✅ Code follows established patterns and conventions

**Remember: Speed should never come at the expense of quality. Taking time to do it right prevents technical debt and future debugging sessions.**