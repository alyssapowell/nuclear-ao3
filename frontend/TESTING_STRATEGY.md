# Nuclear AO3 Testing Strategy

## 🎯 **Production vs Demo Testing Philosophy**

### **Production Tests (ALWAYS Priority)**
- **What**: Test actual user-facing features that ship to production
- **Where**: Real routes like `/works/[id]`, `/search`, `/profile`, etc.
- **Why**: These tests validate the actual user experience
- **When**: Run in CI/CD, before deployment, during development

```bash
# Run only production feature tests
npm run test:production

# Test specific production features
npm run test:offline-consent
npm run test:tag-enhancement  
npm run test:accessibility
```

### **Demo Tests (Secondary/Optional)**
- **What**: Test demonstration and development tools
- **Where**: `/demos/*` routes (removed in production)
- **Why**: Validate demo functionality for stakeholders and development
- **When**: During development only, not in CI/CD

```bash
# Demos are automatically removed before production build
npm run build  # Runs clean:demos automatically
```

## 🧪 **Test Categories**

### **1. Production Feature Tests**
```bash
# Core functionality that ships to users
npm run test:e2e                    # All production tests
npm run test:offline-consent        # Offline reading consent system
npm run test:tag-enhancement        # Tag management features  
npm run test:subscription           # Notification subscriptions
npm run test:accessibility          # WCAG compliance
```

### **2. Integration Tests**
```bash
# Multi-service integration
npm run test:e2e:integration        # Backend integration
npm run test:collections            # Collections system
```

### **3. Unit Tests**
```bash
npm run test                        # Jest unit tests
npm run test:coverage               # Coverage reports
npm run test:accessibility          # A11y unit tests
```

## 📋 **Offline Reading Consent System Tests**

Our offline reading consent system is tested on **real work pages**, not demo pages:

### **Test Scenarios**
1. **Files + PWA Allowed**: Work ID `123e4567-e89b-12d3-a456-426614174000`
2. **PWA Only**: Work ID `223e4567-e89b-12d3-a456-426614174001`  
3. **Online Only**: Work ID `323e4567-e89b-12d3-a456-426614174002`

### **What We Test**
- ✅ RespectfulExportButton behavior based on author preferences
- ✅ Educational modals for restricted content
- ✅ Author override capabilities
- ✅ UI state changes (button text, icons, availability)
- ✅ Modal interactions (open, close, escape key)
- ✅ Export service integration
- ✅ Error handling and network issues

### **What We DON'T Test**
- ❌ Demo pages (`/demos/offline-consent`)
- ❌ Internal development tools
- ❌ Non-production routes

## 🚀 **Production Deployment**

### **Automatic Demo Removal**
```bash
npm run build  # Automatically runs clean:demos
```

The `prebuild` script ensures demos are removed before production:
1. `npm run clean:demos` - Removes `/src/app/demos` directory
2. `npm run update-attributions` - Updates dependency attributions
3. `npx next build` - Builds production bundle

### **Production Test Validation**
```bash
# Validate production-ready tests
npm run test:production

# Ensure no demo dependencies remain
npm run lint
```

## 📊 **CI/CD Integration**

```yaml
# Example GitHub Actions
- name: Run Production Tests
  run: npm run test:production

- name: Build Production Bundle  
  run: npm run build  # Demos automatically removed

- name: Validate No Demo Code
  run: |
    if [ -d "src/app/demos" ]; then
      echo "Demo code found in production build!"
      exit 1
    fi
```

## 🎭 **Demo Development**

### **When to Use Demos**
- Stakeholder presentations
- Feature development and iteration
- Manual testing of complex scenarios
- Documentation and training

### **Demo Structure**
```
src/app/demos/
├── page.tsx                    # Demo center index
├── offline-consent/            # Offline reading consent demo
├── export-system/              # Export features demo  
└── search/                     # Search functionality demo
```

### **Demo Features**
- Real work ID input and search
- Interactive preference testing
- Scenario simulation
- Development-time validation

---

**Remember**: Demos are for demonstration. Tests are for validation. Always prioritize testing the actual user experience over demo functionality.