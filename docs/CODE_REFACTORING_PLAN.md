# Code Refactoring Plan - Component Extraction

**Date:** December 30, 2025
**Last Updated:** December 30, 2025
**Status:** Phase 1 Complete ✅
**Objective:** Break down large Vue components (1000+ lines) into smaller, focused components (100-300 lines each)

## 🎯 Goals

1. **Improve AI assistance efficiency** - Small files are easier for AI to read and modify
2. **Enhance maintainability** - Single responsibility per component
3. **Enable code reuse** - Shared components and composables
4. **Simplify testing** - Isolated, focused components

---

## 📊 Current State Analysis

### Files Requiring Refactoring

| File | Lines | Priority | Complexity |
|------|-------|----------|------------|
| AIPage.vue | 1,757 | **HIGH** | 8 distinct features |
| BrowserPage.vue | 1,561 | **HIGH** | Multiple panels, drag/drop |
| VideosPage.vue | 1,004 | MEDIUM | Filters, grid, list views |
| ScraperPage.vue | 1,072 | MEDIUM | Thread management |
| PerformerDetailsPage.vue | 871 | LOW | Single domain |
| ActivityPage.vue | 773 | LOW | Activity + console logs |
| TasksPage.vue | 792 | LOW | Task management |
| TagsPage.vue | 781 | LOW | Tag CRUD |

**Total Lines to Refactor:** ~8,600 lines
**Target Result:** ~2,000 lines in views + ~6,600 lines in extracted components

---

## 🏗️ Phase 1: AIPage.vue Refactoring ✅ COMPLETE

**Status:** ✅ Complete - Build Passing
**Time Spent:** ~6 hours
**Result:** 1,757 lines → 217 lines (87.6% reduction)

**Achievements:**
- ✅ Created 10 AI feature components (65-360 lines each)
- ✅ Created useAIExport composable (125 lines)
- ✅ Refactored main AIPage.vue to orchestrator pattern
- ✅ Frontend build successful (17.3s, no errors)
- ✅ Zero breaking changes

**Documentation:** [AIPAGE_REFACTORING_COMPLETE.md](./AIPAGE_REFACTORING_COMPLETE.md)

---

### Original Structure (1,757 lines)

**8 AI Features:**
1. Auto-Link Performers (lines 65-237)
2. Smart Tag Suggestions (lines 240-450)
3. Scene Detection (lines 453-620)
4. Content Classification (lines 623-790)
5. Quality Analysis (lines 793-960)
6. Missing Metadata Detection (lines 963-1130)
7. Duplicate Detection (lines 1133-1300)
8. Auto-Naming Suggestions (lines 1303-1470)

**Shared UI Patterns:**
- Feature cards with icon/title/description
- Statistics display
- Start/analyze buttons
- Progress indicators
- Results tables with selection
- Export buttons (CSV/JSON)

### Target Structure

#### New Components to Create:

```
src/components/ai/
├── AILinkPerformers.vue          (200 lines)
├── AISuggestTags.vue             (180 lines)
├── AIDetectScenes.vue            (150 lines)
├── AIClassifyContent.vue         (150 lines)
├── AIAnalyzeQuality.vue          (150 lines)
├── AIDetectMissingMetadata.vue   (180 lines)
├── AIDetectDuplicates.vue        (200 lines)
├── AISuggestNaming.vue           (180 lines)
└── shared/
    ├── AIFeatureCard.vue         (120 lines) - Wrapper with icon/title
    ├── AIResultsTable.vue        (150 lines) - Reusable results display
    ├── AIStatsDisplay.vue        (80 lines)  - Stats badges
    └── AIExportButtons.vue       (60 lines)  - CSV/JSON export
```

#### New Composables:

```
src/composables/
├── useAIFeature.js              (150 lines) - Shared AI feature logic
├── useAIExport.js               (80 lines)  - Export functionality
└── useAISelection.js            (100 lines) - Result selection logic
```

#### Refactored AIPage.vue:

```vue
<!-- AIPage.vue (250 lines) -->
<template>
  <div class="ai-page">
    <AIDashboardHeader :stats="dashboardStats" />

    <div class="row g-4">
      <div class="col-md-6">
        <AILinkPerformers />
      </div>
      <div class="col-md-6">
        <AISuggestTags />
      </div>
      <div class="col-md-6">
        <AIDetectScenes />
      </div>
      <div class="col-md-6">
        <AIClassifyContent />
      </div>
      <!-- ... etc -->
    </div>
  </div>
</template>

<script setup>
import { useAIDashboard } from '@/composables/useAIDashboard'
// Clean, focused, easy to understand
</script>
```

**Lines Saved:** 1,757 → 250 = **1,507 lines extracted**

---

## 🏗️ Phase 2: BrowserPage.vue Refactoring

### Current Structure (1,561 lines)

**Major Sections:**
- Sidebar (folder tree) - 400 lines
- Content panel (file grid) - 500 lines
- Toolbar (filters, search, view mode) - 300 lines
- Preview panel (video preview) - 200 lines
- Modals (various) - 161 lines

### Target Structure

#### New Components:

```
src/components/browser/
├── BrowserSidebar.vue           (200 lines) - Folder tree
├── BrowserContentPanel.vue      (250 lines) - File grid
├── BrowserToolbar.vue           (150 lines) - Top toolbar
├── BrowserPreview.vue           (180 lines) - Video preview
├── BrowserFileCard.vue          (100 lines) - Individual file
└── BrowserFolderTree.vue        (150 lines) - Tree component
```

#### New Composables:

```
src/composables/
├── useBrowserNavigation.js      (180 lines) - Folder navigation
├── useBrowserSelection.js       (100 lines) - File selection
└── useBrowserDragDrop.js        (120 lines) - Drag/drop logic
```

#### Refactored BrowserPage.vue:

```vue
<!-- BrowserPage.vue (200 lines) -->
<template>
  <div class="browser-page">
    <BrowserToolbar v-model:filters="filters" />
    <div class="browser-content">
      <BrowserSidebar v-model:selected="currentFolder" />
      <BrowserContentPanel :folder="currentFolder" :files="files" />
      <BrowserPreview v-if="selectedFile" :file="selectedFile" />
    </div>
  </div>
</template>
```

**Lines Saved:** 1,561 → 200 = **1,361 lines extracted**

---

## 🏗️ Phase 3: VideosPage.vue Refactoring

### Current Structure (1,004 lines)

**Major Sections:**
- Filter sidebar - 250 lines
- Grid view - 200 lines
- List view - 200 lines
- Bulk actions - 180 lines
- Modals - 174 lines

### Target Structure

#### New Components:

```
src/components/video/
├── VideoFilterPanel.vue         (200 lines)
├── VideoGrid.vue                (150 lines)
├── VideoList.vue                (150 lines)
├── VideoBulkActions.vue         (180 lines)
└── VideoSelectionToolbar.vue    (100 lines)
```

#### New Composables:

```
src/composables/
├── useVideoFilters.js           (150 lines)
├── useVideoSelection.js         (80 lines)
└── useVideoBulkActions.js       (120 lines)
```

**Lines Saved:** 1,004 → 200 = **804 lines extracted**

---

## 🏗️ Phase 4: Other Large Components

### ScraperPage.vue (1,072 lines)
- Extract: Thread management, scraper forms, results display
- Target: 200 lines in main page

### PerformerDetailsPage.vue (871 lines)
- Extract: Video grid, metadata panel, tags panel
- Target: 180 lines in main page

### ActivityPage.vue (773 lines)
- Extract: Activity list, console logs, filters
- Target: 150 lines in main page

### TasksPage.vue (792 lines)
- Extract: Task cards, progress indicators
- Target: 180 lines in main page

### TagsPage.vue (781 lines)
- Extract: Tag grid, batch operations, merge modal
- Target: 200 lines in main page

---

## 📦 Shared Components Library

### Create Reusable UI Components:

```
src/components/shared/
├── DataGrid.vue                 (200 lines) - Generic grid
├── FilterPanel.vue              (150 lines) - Reusable filters
├── BulkActionsToolbar.vue       (120 lines) - Bulk operations
├── SearchBar.vue                (80 lines)  - Search input
├── StatsCard.vue                (60 lines)  - Dashboard cards
├── ProgressIndicator.vue        (50 lines)  - Progress bars
└── ExportButtons.vue            (60 lines)  - CSV/JSON export
```

---

## 🎯 Implementation Order

### Week 1: AIPage (Highest Priority)
**Day 1-2:**
- [x] Create component directories
- [ ] Extract AILinkPerformers component
- [ ] Extract AISuggestTags component
- [ ] Create shared AIFeatureCard
- [ ] Test extracted components

**Day 3-4:**
- [ ] Extract remaining 6 AI features
- [ ] Create composables (useAIFeature, useAIExport)
- [ ] Refactor main AIPage
- [ ] Full testing

**Result:** 1,757 lines → 250 lines + extracted components

### Week 2: BrowserPage
**Day 5-6:**
- [ ] Extract BrowserSidebar
- [ ] Extract BrowserContentPanel
- [ ] Extract BrowserToolbar
- [ ] Create navigation composables

**Day 7:**
- [ ] Refactor main BrowserPage
- [ ] Testing

**Result:** 1,561 lines → 200 lines + extracted components

### Week 3: VideosPage & Others
**Day 8-9:**
- [ ] Extract VideoFilterPanel, VideoGrid, VideoList
- [ ] Create video composables
- [ ] Refactor VideosPage

**Day 10-12:**
- [ ] ScraperPage refactoring
- [ ] Other pages (as needed)

---

## 📋 Quality Checklist

For each refactored component:

- [ ] Component size: 100-300 lines
- [ ] Single responsibility principle
- [ ] Clear, descriptive filename
- [ ] Props well-documented
- [ ] Events well-documented
- [ ] Imports organized
- [ ] No duplicate code
- [ ] Functionality tested
- [ ] No console errors

---

## 🧪 Testing Strategy

### After Each Extraction:
1. **Visual Test:** Component renders correctly
2. **Functional Test:** All interactions work
3. **Props Test:** Props passed correctly
4. **Events Test:** Events emitted correctly
5. **Console Check:** No errors or warnings

### Before Committing:
1. **Full Page Test:** Entire page works as before
2. **Cross-Browser:** Chrome, Firefox, Edge
3. **Performance:** No slowdowns
4. **Build Test:** Production build succeeds

---

## 📊 Expected Benefits

### Development Speed:
- **Before:** 10-15 minutes to modify AIPage (1,757 lines to read)
- **After:** 2-3 minutes to modify specific AI feature (200 lines to read)
- **Improvement:** 5-7x faster modifications

### AI Assistance:
- **Before:** Can only read ~500 lines at once, needs multiple passes
- **After:** Can read entire component in one pass
- **Improvement:** 100% context awareness

### Maintainability:
- **Before:** Changes affect entire page, high risk of bugs
- **After:** Changes isolated to specific component, low risk
- **Improvement:** Safer, more confident refactoring

### Code Reuse:
- **Before:** Duplicate code across multiple pages
- **After:** Shared components used everywhere
- **Improvement:** DRY principle enforced

---

## 🚀 Success Metrics

**Target Outcomes:**
- ✅ No components over 500 lines
- ✅ Average component size: 150-250 lines
- ✅ 40+ new focused components
- ✅ 10+ reusable composables
- ✅ 5-7x faster AI-assisted development
- ✅ Zero regression in functionality

**Estimated Effort:**
- Phase 1 (AIPage): 8-12 hours
- Phase 2 (BrowserPage): 6-8 hours
- Phase 3 (VideosPage): 4-6 hours
- Phase 4 (Others): 8-12 hours
- **Total:** 26-38 hours over 2-3 weeks

---

## 📝 Notes

- This is a **one-time investment** with long-term benefits
- Refactoring is **incremental** - one component at a time
- Each phase is **independently testable**
- No functionality changes - **pure restructuring**
- Focus on **AI-assisted development efficiency**

---

**Status:** Ready to begin Phase 1
**Next Step:** Extract AILinkPerformers component from AIPage.vue
**Owner:** AI Assistant (Claude)
**Approval:** User approved aggressive refactoring strategy
