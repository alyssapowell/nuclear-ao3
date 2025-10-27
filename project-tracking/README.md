# Nuclear AO3 - Project Tracking

This directory contains project management and issue tracking documentation separate from code documentation.

## Directory Structure

### 📋 `/issues/`
**Purpose**: Bug reports, feature requests, and issue resolution tracking
- `COMMENTS_SYSTEM_ISSUES.md` - Complete comments system debugging history
- `TAGS_COMMENTS_STATUS_REPORT.md` - Tags and comments status tracking
- Future issue tracking files

### 🧪 `/testing/`
**Purpose**: Test results, validation reports, and quality assurance
- `FINAL_TESTING_SUMMARY.md` - Comprehensive testing results across all services
- Performance test results
- Integration test reports

### 📝 `/sessions/`
**Purpose**: Development session summaries and progress tracking
- Session notes from debugging/development work
- Progress summaries
- Decision logs

## Guidelines

### What Goes Here vs Code Repo
- ✅ **Project Tracking**: Bug reports, feature requests, testing results, session notes
- ✅ **Temporary Documentation**: Development logs, debugging notes, status reports
- ❌ **Permanent Documentation**: API docs, setup guides, architectural docs (belongs in `/docs/`)
- ❌ **Code Documentation**: README files, inline docs, technical specifications

### File Naming Convention
- Issues: `COMPONENT_SYSTEM_ISSUES.md` (e.g., `AUTH_SYSTEM_ISSUES.md`)
- Testing: `COMPONENT_TEST_RESULTS.md` (e.g., `FRONTEND_TEST_RESULTS.md`)
- Sessions: `YYYY-MM-DD_SESSION_SUMMARY.md` (e.g., `2025-10-06_COMMENTS_DEBUG_SESSION.md`)

### Issue Lifecycle
1. **Open**: New issue identified → Create in `/issues/`
2. **In Progress**: Being worked on → Update status in issue file
3. **Resolved**: Fixed and tested → Mark as resolved with date
4. **Closed**: Verified working → Move to archive or mark as historical

This separation keeps the main codebase clean while maintaining comprehensive project tracking.