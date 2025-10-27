# Contributing to Nuclear AO3

**Welcome to the future of fanfiction infrastructure!** 

Nuclear AO3 is built by the community, for the community. We welcome contributions from developers, designers, writers, and anyone passionate about creating a better fanfiction platform.

## 🎯 Project Vision

**Our mission:** Create a modern, scalable, and reliable fanfiction platform that serves millions of users with sub-second response times and zero-downtime deployments.

**Our values:**
- **Performance First:** Every feature must be fast and scalable
- **Community Driven:** Built by fans, for fans
- **Open Source:** Transparent development and decision-making
- **Modern Standards:** Use current best practices and technologies
- **User Experience:** Prioritize usability and accessibility

## 🚀 Quick Start

### Development Environment Setup

```bash
# Clone the repository
git clone https://github.com/your-org/nuclear-ao3.git
cd nuclear-ao3

# Start development environment (requires Docker)
make setup

# This will:
# - Start all services (PostgreSQL, Redis, Elasticsearch)  
# - Run database migrations
# - Seed with test data
# - Start all microservices
# - Launch frontend development server

# Visit http://localhost:3000 to see Nuclear AO3!
```

**Demo credentials:**
- Admin: `admin@nuclear-ao3.demo` / `password123`
- Author: `author@nuclear-ao3.demo` / `password123`  
- Reader: `reader@nuclear-ao3.demo` / `password123`

### Development Commands

```bash
make dev          # Start development environment
make test         # Run all tests
make benchmark    # Performance benchmarks  
make logs         # View service logs
make clean        # Clean up containers
```

## 🏗️ Architecture Overview

Nuclear AO3 uses a **modern microservices architecture**:

```
Frontend (Next.js + React)
    ↓
GraphQL Gateway (Node.js) 
    ↓
┌─────────────┬─────────────┬─────────────┬─────────────┐
│Auth Service │Work Service │Tag Service  │Search Service│
│   (Go)      │    (Go)     │    (Go)     │    (Go)     │
└─────────────┴─────────────┴─────────────┴─────────────┘
    ↓
PostgreSQL + Redis + Elasticsearch
    ↓
Kubernetes Auto-scaling
```

### Key Technologies

- **Backend:** Go 1.21+ with Gin framework
- **Frontend:** Next.js 14 with TypeScript and Tailwind CSS
- **Database:** PostgreSQL 15 with optimized schema
- **Cache:** Redis 7 for sessions and caching
- **Search:** Elasticsearch 8 for full-text search
- **Auth:** JWT tokens with RS256 signing
- **Deployment:** Docker + Kubernetes
- **Monitoring:** Prometheus + Grafana

## 📋 How to Contribute

### 1. Choose Your Contribution Type

**🔧 Backend Development (Go)**
- API endpoints and business logic
- Database optimizations
- Authentication and security
- Performance improvements

**🎨 Frontend Development (React/TypeScript)**
- User interface components  
- Mobile responsiveness
- User experience improvements
- Accessibility features

**📊 DevOps & Infrastructure**
- Kubernetes deployments
- Monitoring and alerting
- CI/CD pipeline improvements
- Performance optimization

**🧪 Quality Assurance**
- Test writing and automation
- Load testing and benchmarking  
- Security auditing
- Documentation

**📝 Documentation & Design**
- User documentation
- API documentation
- UI/UX design
- Content and copy

### 2. Find an Issue

**Good First Issues:** Look for issues labeled `good-first-issue`
**Help Wanted:** Issues labeled `help-wanted` need contributors
**Performance:** Issues labeled `performance` focus on speed improvements
**Security:** Issues labeled `security` address security concerns

**Priority Areas:**
- Mobile optimization
- Search improvements  
- Real-time notifications
- Content management features
- Moderation tools

### 3. Development Workflow

```bash
# 1. Fork the repository
# 2. Create a feature branch
git checkout -b feature/amazing-new-feature

# 3. Make your changes
# 4. Add tests for your changes
make test

# 5. Run performance benchmarks
make benchmark

# 6. Lint and format code
make lint
make format

# 7. Commit with descriptive messages
git commit -m "feat: add real-time notifications

- Implement WebSocket connection for live updates
- Add notification preferences to user profile
- Include performance optimizations for high concurrency
- Add comprehensive test coverage

Closes #123"

# 8. Push and create pull request
git push origin feature/amazing-new-feature
```

### 4. Pull Request Process

**Before submitting:**
- [ ] All tests pass (`make test`)
- [ ] **Accessibility tests pass** (`npm run test:accessibility` in frontend/)
- [ ] Performance benchmarks show no regression (`make benchmark`)
- [ ] Code is properly formatted (`make format`)
- [ ] Documentation is updated if needed
- [ ] Security considerations are addressed
- [ ] **WCAG 2.1 AA compliance verified** for any UI changes

**PR Requirements:**
- **Clear description** of what changes and why
- **Performance impact** assessment (faster/neutral/slower)
- **Test coverage** for new functionality
- **Breaking changes** clearly documented
- **Security implications** considered

**Review Process:**
1. **Automated checks** run (tests, linting, security scans)
2. **Performance review** (benchmark comparison)  
3. **Code review** by maintainers
4. **Security review** for sensitive changes
5. **Final approval** and merge

## 🎨 Design Guidelines

### Backend Principles

**Performance First:**
- Target <100ms API response times
- Use connection pooling and caching
- Optimize database queries with proper indexing
- Profile performance impact of all changes

**Security by Default:**
- Validate all inputs
- Use prepared statements (no SQL injection)
- Implement proper authentication/authorization
- Follow OWASP security guidelines

**Code Quality:**
- Write comprehensive tests (>90% coverage)
- Use meaningful variable and function names
- Include performance benchmarks
- Document complex algorithms

**Example Go service structure:**
```go
// handlers.go - HTTP request handlers
func (s *Service) CreateWork(c *gin.Context) {
    start := time.Now()
    
    // Input validation
    var req CreateWorkRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Business logic
    work, err := s.createWork(c.Request.Context(), &req)
    if err != nil {
        c.JSON(500, gin.H{"error": "Internal server error"})
        return  
    }
    
    // Performance monitoring
    c.Header("X-Response-Time", fmt.Sprintf("%.2fms", 
        float64(time.Since(start).Nanoseconds())/1e6))
    
    c.JSON(201, work)
}
```

### Frontend Principles

**Mobile First:**
- Design for mobile devices first
- Progressive enhancement for larger screens
- Touch-friendly interface elements
- Optimized for slow network connections

**Performance Focused:**
- Code splitting and lazy loading
- Image optimization and responsive images
- Minimal bundle sizes
- Service worker for offline functionality

**Accessibility (MANDATORY):**
- **WCAG 2.1 AA compliance** - All new components must pass accessibility testing
- **Semantic HTML structure** - Use proper landmarks, headings, and form labels  
- **Keyboard navigation support** - Every interactive element must be keyboard accessible
- **Screen reader compatibility** - Test with NVDA, JAWS, VoiceOver
- **ARIA patterns** - Implement proper ARIA attributes for complex components
- **Focus management** - Ensure clear focus indicators and logical tab order
- **Live regions** - Use for dynamic content updates and form validation
- **Color contrast** - Maintain minimum 4.5:1 ratio for normal text, 3:1 for large text

**Example React component:**
```tsx
// components/WorkCard.tsx
export const WorkCard: React.FC<WorkCardProps> = ({ work }) => {
  return (
    <article 
      className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border p-4 hover:shadow-md transition-shadow"
      aria-labelledby={`work-${work.id}-title`}
    >
      <h3 
        id={`work-${work.id}-title`}
        className="text-lg font-semibold mb-2"
      >
        <Link href={`/works/${work.id}`} className="hover:text-blue-600">
          {work.title}
        </Link>
      </h3>
      
      <p className="text-gray-600 dark:text-gray-300 mb-3">
        by {work.author.displayName}
      </p>
      
      {/* Mobile-optimized tag list */}
      <div className="flex flex-wrap gap-1 mb-3">
        {work.tags.slice(0, 3).map(tag => (
          <span 
            key={tag.id}
            className="px-2 py-1 text-xs bg-blue-100 text-blue-800 rounded"
          >
            {tag.name}
          </span>
        ))}
        {work.tags.length > 3 && (
          <span className="px-2 py-1 text-xs text-gray-500">
            +{work.tags.length - 3} more
          </span>
        )}
      </div>
    </article>
  );
};
```

## 🧪 Testing Guidelines

### Backend Testing

**Required test types:**
- **Unit tests:** Test individual functions and methods
- **Integration tests:** Test service interactions
- **Load tests:** Verify performance under load
- **Security tests:** Test for common vulnerabilities

**Example test structure:**
```go
func TestCreateWork_Success(t *testing.T) {
    // Setup
    service := setupTestService(t)
    defer service.cleanup()
    
    req := &CreateWorkRequest{
        Title:   "Test Work",
        Summary: "A test work for unit testing",
        Rating:  "General Audiences",
    }
    
    // Execute
    start := time.Now()
    work, err := service.CreateWork(context.Background(), req)
    elapsed := time.Since(start)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, req.Title, work.Title)
    assert.Less(t, elapsed, 50*time.Millisecond) // Performance assertion
}
```

### Frontend Testing

**Required test types:**
- **Unit tests:** Component behavior testing
- **Integration tests:** User interaction flows
- **E2E tests:** Full application workflows
- **Accessibility tests:** WCAG compliance testing (MANDATORY for all PRs)

**Example component test:**
```tsx
// __tests__/WorkCard.test.tsx
describe('WorkCard', () => {
  it('renders work information correctly', () => {
    const mockWork = {
      id: '123',
      title: 'Test Work',
      author: { displayName: 'Test Author' },
      tags: [{ id: '1', name: 'Test Tag' }]
    };
    
    render(<WorkCard work={mockWork} />);
    
    expect(screen.getByRole('heading', { name: 'Test Work' })).toBeInTheDocument();
    expect(screen.getByText('by Test Author')).toBeInTheDocument();
    expect(screen.getByText('Test Tag')).toBeInTheDocument();
  });
  
  it('meets accessibility standards', async () => {
    const mockWork = createMockWork();
    const { container } = render(<WorkCard work={mockWork} />);
    
    // Test WCAG 2.1 AA compliance
    const results = await axe(container);
    expect(results).toHaveNoViolations();
  });
  
  it('supports keyboard navigation', () => {
    const mockWork = createMockWork();
    render(<WorkCard work={mockWork} />);
    
    const titleLink = screen.getByRole('link', { name: 'Test Work' });
    
    // Test keyboard focus
    titleLink.focus();
    expect(titleLink).toHaveFocus();
    
    // Test Enter key activation
    fireEvent.keyDown(titleLink, { key: 'Enter', code: 'Enter' });
    // Assert navigation behavior
  });
  
  it('provides proper ARIA labels and descriptions', () => {
    const mockWork = createMockWork();
    render(<WorkCard work={mockWork} />);
    
    // Test semantic structure
    expect(screen.getByRole('article')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 3 })).toBeInTheDocument();
    
    // Test ARIA attributes
    const article = screen.getByRole('article');
    expect(article).toHaveAttribute('aria-labelledby');
  });
});
```

## ♿ Accessibility Guidelines

### 🎯 **Accessibility is Mandatory**

Nuclear AO3 maintains **WCAG 2.1 AA compliance** as a core requirement. Every contribution that affects the user interface must meet or exceed accessibility standards.

### 📋 **Accessibility Checklist**

**For Every Component/Page:**
- [ ] **Semantic HTML** - Use proper elements (`<button>`, `<nav>`, `<main>`, etc.)
- [ ] **Keyboard Navigation** - All interactive elements accessible via keyboard
- [ ] **Focus Management** - Clear focus indicators and logical tab order
- [ ] **Screen Reader Support** - Meaningful labels and descriptions
- [ ] **ARIA Patterns** - Proper implementation of ARIA attributes
- [ ] **Color Contrast** - Minimum 4.5:1 ratio for normal text
- [ ] **Live Regions** - Dynamic content changes announced to assistive technologies
- [ ] **Form Accessibility** - Clear labels, error handling, validation feedback

### 🛠️ **Implementation Standards**

**Required ARIA Patterns:**
```typescript
// For interactive components
<button
  aria-label="Close dialog"
  aria-describedby="close-description"
  onClick={handleClose}
>
  ×
</button>

// For complex widgets (autocomplete, etc.)
<input
  role="combobox"
  aria-expanded={isOpen}
  aria-owns="suggestions-list"
  aria-activedescendant={activeOption}
/>

// For dynamic content
<div aria-live="polite" aria-atomic="true">
  {statusMessage}
</div>
```

**Keyboard Support Requirements:**
- **Tab/Shift+Tab**: Navigate between elements
- **Enter/Space**: Activate buttons and controls
- **Arrow keys**: Navigate within components (menus, lists)
- **Escape**: Close modals, dropdowns, cancel actions
- **Home/End**: Jump to first/last items in lists

**Focus Management:**
```typescript
// Manage focus for dynamic content
useEffect(() => {
  if (isModalOpen) {
    modalRef.current?.focus();
  }
}, [isModalOpen]);

// Restore focus when closing modals
const handleClose = () => {
  setIsModalOpen(false);
  triggerRef.current?.focus();
};
```

### 🧪 **Accessibility Testing Requirements**

**All PRs must include:**

**1. Automated Testing:**
```bash
# Run accessibility test suite
npm run test:accessibility

# Test specific component
npm test TagAutocomplete.accessibility.test.tsx
```

**2. Manual Testing:**
- [ ] **Keyboard-only navigation** through entire feature
- [ ] **Screen reader testing** with NVDA/VoiceOver
- [ ] **High contrast mode** compatibility
- [ ] **Zoom testing** up to 200% without horizontal scrolling

**3. Code Review:**
- [ ] ARIA attributes reviewed for correctness
- [ ] Semantic HTML structure validated
- [ ] Focus management patterns approved
- [ ] Color contrast ratios verified

### 📖 **Accessibility Resources**

**Our Implementation Examples:**
- **SearchForm**: `frontend/src/components/SearchForm.tsx`
- **TagAutocomplete**: `frontend/src/components/TagAutocomplete.tsx`  
- **SearchResults**: `frontend/src/components/SearchResults.tsx`
- **SearchPagination**: `frontend/src/components/SearchPagination.tsx`

**Testing Documentation:**
- **Testing Guide**: `ACCESSIBILITY_TESTING_GUIDE.md`
- **Frontend Guide**: `frontend/ACCESSIBILITY.md`

**External Resources:**
- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/patterns/)
- [WebAIM Screen Reader Testing](https://webaim.org/articles/screenreader_testing/)

### 🚫 **Common Accessibility Violations to Avoid**

**Critical Issues:**
- Missing form labels or descriptions
- Keyboard traps or inaccessible focus
- Color-only information conveyance
- Missing ARIA attributes on complex widgets
- Insufficient color contrast ratios
- Inaccessible modal or dropdown implementations

**Code Examples of What NOT to Do:**
```typescript
// ❌ BAD - Missing label and keyboard support
<div onClick={handleSubmit}>Submit</div>

// ✅ GOOD - Proper button with label
<button 
  type="submit" 
  aria-label="Submit search form"
  onClick={handleSubmit}
>
  Submit
</button>

// ❌ BAD - No ARIA for custom dropdown
<div className="dropdown">
  <div onClick={toggle}>Options</div>
  {isOpen && <div>...</div>}
</div>

// ✅ GOOD - Proper ARIA combobox pattern
<div role="combobox" aria-expanded={isOpen}>
  <button aria-haspopup="listbox" onClick={toggle}>
    Options
  </button>
  {isOpen && (
    <ul role="listbox" aria-label="Available options">
      ...
    </ul>
  )}
</div>
```

### 🏆 **Accessibility Champions**

**Recognition for contributions:**
- Accessibility improvements are highlighted in release notes
- Contributors who improve accessibility get special recognition
- Accessibility-focused PRs are prioritized for review

**Become an Accessibility Champion:**
- Review PRs for accessibility compliance
- Contribute to accessibility testing documentation
- Help test with assistive technologies
- Mentor other contributors on accessibility best practices

## 🔒 Security Guidelines

### Security Requirements

**Authentication:**
- JWT tokens with RS256 signing
- Secure password hashing (bcrypt cost 12+)
- Rate limiting on sensitive endpoints
- Session monitoring and management

**Input Validation:**
- Validate all user inputs
- Sanitize HTML content  
- Use parameterized database queries
- Implement CSRF protection

**Data Protection:**
- Encrypt sensitive data at rest
- Use HTTPS for all communications
- Implement proper access controls
- Regular security audits

### Security Review Process

**All security-sensitive changes require:**
1. **Security impact assessment**
2. **Code review by security-focused maintainer**  
3. **Penetration testing** for major changes
4. **Documentation** of security considerations

## 📊 Performance Guidelines

### Performance Requirements

**API Response Times:**
- Simple queries: <50ms  
- Complex queries: <200ms
- Search requests: <100ms
- Authentication: <100ms

**Frontend Performance:**
- First Contentful Paint: <1s
- Time to Interactive: <2s  
- PageSpeed Insights: >90
- Bundle size: <250KB initial

### Performance Monitoring

**Required benchmarks:**
```bash
# Run performance benchmarks for all changes
make benchmark

# Expected output:
BenchmarkCreateWork-8           5000    234 ns/op    48 B/op    2 allocs/op
BenchmarkGetWork-8             20000     89 ns/op    32 B/op    1 allocs/op
BenchmarkSearchWorks-8          1000   1234 ns/op   256 B/op    8 allocs/op
```

**Performance regression policy:**
- >10% regression requires justification
- >25% regression requires optimization plan
- >50% regression blocks merge

## 📚 Documentation Standards

### Code Documentation

**Go code:**
- Public functions must have doc comments
- Complex algorithms need explanation comments  
- Performance characteristics documented
- Security considerations noted

**React components:**
- PropTypes or TypeScript interfaces
- Usage examples in Storybook
- Accessibility considerations  
- Performance notes for complex components

### API Documentation

- OpenAPI/Swagger specifications
- Request/response examples
- Error codes and messages
- Performance characteristics
- Authentication requirements

## 🌍 Community Guidelines

### Communication

**Be respectful:** We're building this together
**Be constructive:** Focus on solutions, not problems  
**Be inclusive:** Welcome all skill levels and backgrounds
**Be patient:** Everyone is learning

### Decision Making

**Technical decisions:** Discussed in GitHub issues
**Design decisions:** Community feedback via Discord
**Major changes:** RFC process with community input

### Code of Conduct

We follow the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). 

**In summary:**
- Be welcoming and inclusive
- Respect differing viewpoints
- Accept constructive criticism gracefully  
- Focus on what's best for the community

## 🏆 Recognition

**Contributors are recognized through:**
- GitHub contributions graph
- Contributors file in repository
- Community showcase posts
- Conference talk opportunities (if interested)

**Special recognition for:**
- Performance improvements
- Security enhancements  
- Accessibility improvements
- Mentoring new contributors

## 📞 Getting Help

**Technical questions:** GitHub Discussions
**Real-time chat:** Discord server (link in README)  
**Security issues:** security@nuclear-ao3.org
**General questions:** hello@nuclear-ao3.org

**Response times:**
- GitHub issues: 24-48 hours
- Discord: Real-time during community hours
- Security issues: 4 hours maximum

## 🎉 Thank You!

**Every contribution matters,** whether it's:
- Fixing a typo in documentation
- Optimizing a database query  
- Designing a better user interface
- Helping other contributors

**Together, we're building the fanfiction platform the community deserves.**

---

*Nuclear AO3 is built with ❤️ by the fanfiction community*