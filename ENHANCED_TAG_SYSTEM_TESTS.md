# Enhanced Tag System - Playwright Test Suite

## Overview

This test suite validates the enhanced tag prominence system designed to solve real-world AO3 tagging problems, particularly the "orgy problem" where works have excessive relationship tags that make filtering difficult.

## Problem Statement

Based on the example work: https://archiveofourown.org/works/72365631

**Issues we're solving:**
1. **Excessive relationship tagging** - Works with 20+ relationship tags
2. **Missing character tags** - Relationships listed without corresponding character tags
3. **Poor tag prioritization** - No distinction between main and background relationships
4. **Reader filtering difficulties** - Cannot effectively filter by primary relationships

## Test Files

### 1. `enhanced-tag-prominence.spec.ts`
**Comprehensive test suite for the enhanced tag prominence system**

**Key Test Scenarios:**
- ✅ **Relationship limit enforcement** - Max 3 primary relationships
- ✅ **Character auto-detection** - Missing characters detected from relationships
- ✅ **Background tag detection** - "Background", "Past", "Minor" keywords auto-set to micro
- ✅ **User prominence control** - Manual adjustment within limits
- ✅ **AI suggestion feedback** - Clear indicators for auto-suggested prominences

### 2. `tag-spam-prevention.spec.ts`
**Focused tests for preventing tag spam scenarios**

**Key Test Scenarios:**
- ✅ **Orgy problem prevention** - Limits primary relationships to 2 max
- ✅ **Character detection validation** - Ensures characters are added for relationships
- ✅ **Background relationship handling** - Auto-micro prominence for background tags
- ✅ **User reprioritization limits** - Enforces 2-relationship limit with warnings
- ✅ **Tagging guidance** - Clear best practices messaging

### 3. `tag-enhancement-utils.ts`
**Utility functions for consistent testing**

**Helper Functions:**
- `addTagWithProminence()` - Add tags with expected prominence verification
- `verifyProminenceDistribution()` - Check tag distribution across prominence levels  
- `expectTagSpamWarning()` - Validate anti-spam warnings appear
- `expectMissingCharacterDetection()` - Test character auto-detection
- `SPAM_RELATIONSHIP_EXAMPLES` - Real-world problematic tag patterns

## Configuration Files

### `playwright.tag-enhancement.config.ts`
**Specialized configuration for tag system tests**

**Features:**
- Sequential test execution (prevents database conflicts)
- Extended timeouts for form interactions
- Detailed failure reporting with screenshots/videos
- Cross-browser testing (Chrome, Firefox, Mobile)

## Running the Tests

### Quick Commands
```bash
# Run all enhanced tag tests
npm run test:tag-enhancement

# Run specific anti-spam tests  
npm run test:tag-spam

# Run with UI for debugging
npm run test:tag-enhancement:ui

# Run in headed mode (visible browser)
npm run test:tag-enhancement:headed
```

### Comprehensive Test Script
```bash
# Run full validation suite
./run-tag-enhancement-tests.sh
```

**This script:**
- ✅ Checks all required services are running
- ✅ Runs complete test suite with real-world scenarios
- ✅ Generates detailed test reports
- ✅ Validates specific problem scenarios from the example work
- ✅ Provides clear pass/fail summary

## Test Scenarios Covered

### 1. **The "Orgy Problem" Prevention**
**Problem:** Works like the example with 20+ relationship tags
**Solution:** Automatic demotion of excess relationships to secondary/micro prominence
**Test:** `should prevent orgy-style tag spam by limiting primary relationships to 2`

### 2. **Missing Character Detection**  
**Problem:** "Harry Potter/Draco Malfoy" relationship without character tags
**Solution:** AI detection prompts user to add missing characters
**Test:** `should auto-detect missing characters from relationship tags`

### 3. **Background Relationship Handling**
**Problem:** No distinction between main and background relationships
**Solution:** Keywords like "Background", "Past", "Minor" auto-set to micro prominence
**Test:** `should handle background/minor relationship detection`

### 4. **User Control with Limits**
**Problem:** Users need control but within reasonable limits  
**Solution:** Allow reprioritization but enforce 2-3 primary relationship maximum
**Test:** `should allow user to reprioritize but enforce 2-relationship limit`

### 5. **Clear User Guidance**
**Problem:** No guidance about effective tagging
**Solution:** Clear explanations of prominence levels and best practices
**Test:** `should provide clear guidance about relationship tagging best practices`

## Expected Outcomes

### ✅ **Successful Test Results Mean:**
1. **Primary relationships limited to 2-3 maximum**
2. **Missing characters auto-detected and suggested**  
3. **Background relationships automatically demoted**
4. **Clear warnings for excessive relationship tagging**
5. **User-friendly prominence adjustment controls**
6. **Comprehensive guidance about tagging best practices**

### ❌ **Test Failures Indicate:**
- EnhancedTagProminenceSelector component not integrated
- Prominence limit enforcement not working
- Character parsing/detection logic missing
- Warning systems not implemented
- Frontend-backend integration issues

## Integration Requirements

### Frontend Components
- `EnhancedTagProminenceSelector.tsx` - Main prominence management UI
- AI inference logic for tag prominence assignment
- Character parsing from relationship tag names
- Warning displays for tag spam scenarios

### Backend Support
- Tag prominence database schema (✅ **COMPLETE**)
- Prominence calculation APIs 
- Tag relationship analysis endpoints
- Character suggestion APIs

### Database Schema
- `work_tags.prominence` - Primary/secondary/micro levels (✅ **ACTIVE**)
- `tag_prominence_rules` - Auto-assignment rules (✅ **ACTIVE**)
- `work_tag_summaries` - Cached prominence statistics (✅ **ACTIVE**)

## Success Metrics

### Quantitative Goals
- ✅ **Primary relationships ≤ 3 per work**
- ✅ **Character coverage ≥ 90% for relationship tags**
- ✅ **Background relationship detection ≥ 95% accuracy**
- ✅ **User satisfaction with prominence controls**

### Qualitative Goals  
- ✅ **Improved work discoverability through better filtering**
- ✅ **Reduced tag spam in search results**
- ✅ **Clear guidance helps users tag more effectively**
- ✅ **Maintains author creative freedom within reasonable limits**

## Running Tests Against Live System

```bash
# Test against the problematic example work pattern
npx playwright test e2e/tag-spam-prevention.spec.ts -g "orgy.*prevention"

# Verify character detection works
npx playwright test e2e/enhanced-tag-prominence.spec.ts -g "missing.*character"

# Check user prominence controls
npx playwright test e2e/enhanced-tag-prominence.spec.ts -g "reprioritize"
```

## Production Readiness Checklist

- [ ] All test scenarios pass consistently
- [ ] EnhancedTagProminenceSelector integrated in work creation form
- [ ] Character parsing API endpoints active
- [ ] Tag spam warning systems functional
- [ ] User prominence adjustment controls working
- [ ] Cross-browser compatibility verified
- [ ] Mobile responsiveness tested
- [ ] Accessibility standards met
- [ ] Performance impact assessed

---

**🎯 Goal:** Transform Nuclear AO3 from having the same tagging problems as the example work to being the gold standard for intelligent, user-friendly fanfiction tagging that actually helps readers find what they want.