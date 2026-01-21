# AIPage.vue Detailed Refactoring Plan

**Current State:** 1,757 lines - Too large for efficient AI-assisted development
**Target State:** ~250 lines main page + extracted components
**Strategy:** Extract each AI feature into its own component

---

## Current File Structure Analysis

### Template Section (lines 1-916)
- Dashboard header (lines 5-11)
- Summary cards (lines 14-61)
- 10 AI feature cards (lines 64-838):
  1. Auto-Link Performers (lines 65-209)
  2. Smart Tagging (lines 212-349)
  3. Scene Detection (lines 352-432)
  4. Content Classification (lines 435-467)
  5. Quality Analysis (lines 470-558)
  6. Missing Metadata (lines 561-647)
  7. Duplicate Detection (lines 650-738)
  8. Auto-Naming (lines 741-773)
  9. Library Analytics (lines 776-802)
  10. Thumbnail Quality (lines 805-837)
- Duplicate comparison modal (lines 842-914)

### Script Section (lines 918-1752)
- State management (~100 ref() declarations)
- Computed properties (~15 computed properties)
- Helper functions (export, confidence, selection)
- 10 feature functions (one per AI feature)
- Modal functions

### Style Section (line 1754-1757)
- External CSS import

---

## Extraction Strategy

### Phase 1: Create Shared Components (Foundation)

#### 1. AIDashboardSummary.vue (80 lines)
**Purpose:** Dashboard summary cards at top of page
**Props:**
```javascript
{
  libraryHealthScore: Number,
  totalVideosProcessed: Number,
  totalMatches: Number,
  totalIssues: Number
}
```
**Template:** Lines 14-61 from AIPage
**Logic:** Helper functions from lines 1093-1105

#### 2. AIFeatureCard.vue (100 lines)
**Purpose:** Reusable wrapper for all AI features
**Props:**
```javascript
{
  title: String,
  description: String,
  icon: String,
  stats: Object,  // { label: string, value: number }[]
  isAnalyzing: Boolean,
  buttonText: String
}
```
**Slots:**
```vue
<slot name="controls"></slot>  // Custom controls
<slot name="results"></slot>   // Results display
<slot name="empty-state"></slot> // Empty state
```

#### 3. AIResultsTable.vue (120 lines)
**Purpose:** Shared results display with search, selection, export
**Props:**
```javascript
{
  results: Array,
  searchPlaceholder: String,
  exportFilename: String,
  selectable: Boolean,
  showExport: Boolean
}
```
**Features:**
- Search filter
- Bulk selection
- CSV/JSON export
- Slot for custom result display

---

### Phase 2: Extract AI Feature Components

#### 1. AILinkPerformers.vue (220 lines)

**Location:** [src/components/ai/AILinkPerformers.vue](../src/components/ai/AILinkPerformers.vue)

**Extracted From:**
- Template: Lines 65-209
- State: Lines 926-930, 979, 986-990
- Computed: Lines 997-1011
- Functions: Lines 1165-1323

**Props:** None (self-contained)

**Structure:**
```vue
<template>
  <AIFeatureCard
    title="Auto-Link Performers"
    description="Automatically detect and link performers..."
    icon="user-plus"
    :stats="linkStats"
    :is-analyzing="isAnalyzing"
    @start-analysis="startAutoLink"
  >
    <template #controls>
      <!-- Auto-apply checkbox, confidence slider -->
    </template>
    <template #results>
      <AIResultsTable
        :results="filteredSuggestions"
        :selectable="true"
        @apply="applyLinks"
      >
        <template #item="{ item }">
          <!-- Custom match display -->
        </template>
      </AIResultsTable>
    </template>
  </AIFeatureCard>
</template>

<script setup>
// ~150 lines of focused logic
</script>
```

#### 2. AISuggestTags.vue (200 lines)

**Location:** [src/components/ai/AISuggestTags.vue](../src/components/ai/AISuggestTags.vue)

**Extracted From:**
- Template: Lines 212-349
- State: Lines 933-937, 980, 987-990
- Computed: Lines 1013-1027
- Functions: Lines 1325-1507

**Structure:** Similar to AILinkPerformers

#### 3. AIDetectScenes.vue (150 lines)

**Location:** [src/components/ai/AIDetectScenes.vue](../src/components/ai/AIDetectScenes.vue)

**Extracted From:**
- Template: Lines 352-432
- State: Lines 940-942, 981
- Computed: Lines 1029-1037
- Functions: Lines 1510-1532

#### 4. AIClassifyContent.vue (120 lines)

**Location:** [src/components/ai/AIClassifyContent.vue](../src/components/ai/AIClassifyContent.vue)

**Extracted From:**
- Template: Lines 435-467
- State: Lines 945-947
- Functions: Lines 1535-1557

#### 5. AIAnalyzeQuality.vue (180 lines)

**Location:** [src/components/ai/AIAnalyzeQuality.vue](../src/components/ai/AIAnalyzeQuality.vue)

**Extracted From:**
- Template: Lines 470-558
- State: Lines 950-952, 982
- Computed: Lines 1039-1047
- Functions: Lines 1560-1582

#### 6. AIDetectMissingMetadata.vue (170 lines)

**Location:** [src/components/ai/AIDetectMissingMetadata.vue](../src/components/ai/AIDetectMissingMetadata.vue)

**Extracted From:**
- Template: Lines 561-647
- State: Lines 955-957, 983
- Computed: Lines 1049-1057
- Functions: Lines 1585-1607

#### 7. AIDetectDuplicates.vue (220 lines)

**Location:** [src/components/ai/AIDetectDuplicates.vue](../src/components/ai/AIDetectDuplicates.vue)

**Extracted From:**
- Template: Lines 650-738, 842-914 (modal)
- State: Lines 960-962, 984, 993-994
- Computed: Lines 1059-1067
- Functions: Lines 1610-1680

**Special:** Includes comparison modal

#### 8. AISuggestNaming.vue (140 lines)

**Location:** [src/components/ai/AISuggestNaming.vue](../src/components/ai/AISuggestNaming.vue)

**Extracted From:**
- Template: Lines 741-773
- State: Lines 965-967
- Functions: Lines 1683-1705

#### 9. AILibraryAnalytics.vue (120 lines)

**Location:** [src/components/ai/AILibraryAnalytics.vue](../src/components/ai/AILibraryAnalytics.vue)

**Extracted From:**
- Template: Lines 776-802
- State: Lines 970-971
- Functions: Lines 1708-1724

#### 10. AIAnalyzeThumbnails.vue (150 lines)

**Location:** [src/components/ai/AIAnalyzeThumbnails.vue](../src/components/ai/AIAnalyzeThumbnails.vue)

**Extracted From:**
- Template: Lines 805-837
- State: Lines 974-976
- Functions: Lines 1727-1751

---

### Phase 3: Create Composables

#### 1. useAIFeature.js (120 lines)

**Location:** [src/composables/useAIFeature.js](../src/composables/useAIFeature.js)

**Purpose:** Shared logic for all AI features

```javascript
export function useAIFeature(featureName, apiCall) {
  const isAnalyzing = ref(false)
  const results = ref([])
  const stats = ref(null)
  const search = ref('')

  const startAnalysis = async (options = {}) => {
    isAnalyzing.value = true
    try {
      const response = await apiCall(options)
      results.value = response.data.results || []
      // Update stats
    } finally {
      isAnalyzing.value = false
    }
  }

  const filteredResults = computed(() => {
    // Search filtering logic
  })

  return {
    isAnalyzing,
    results,
    stats,
    search,
    filteredResults,
    startAnalysis
  }
}
```

#### 2. useAISelection.js (100 lines)

**Location:** [src/composables/useAISelection.js](../src/composables/useAISelection.js)

**Purpose:** Bulk selection logic

```javascript
export function useAISelection() {
  const selected = ref([])
  const selectAll = ref(false)

  const toggleSelection = (item) => { /* ... */ }
  const toggleSelectAll = (items) => { /* ... */ }
  const clearSelection = () => { /* ... */ }
  const isSelected = (itemId) => { /* ... */ }

  return {
    selected,
    selectAll,
    toggleSelection,
    toggleSelectAll,
    clearSelection,
    isSelected
  }
}
```

#### 3. useAIExport.js (80 lines)

**Location:** [src/composables/useAIExport.js](../src/composables/useAIExport.js)

**Purpose:** Export to CSV/JSON

```javascript
export function useAIExport() {
  const exportToCSV = (data, filename) => { /* ... */ }
  const exportToJSON = (data, filename) => { /* ... */ }
  const downloadFile = (content, filename, type) => { /* ... */ }

  return {
    exportToCSV,
    exportToJSON
  }
}
```

---

### Phase 4: Refactored AIPage.vue (250 lines)

**Location:** [src/views/AIPage.vue](../src/views/AIPage.vue)

```vue
<template>
  <div class="ai-page">
    <div class="container-fluid mt-3">
      <!-- Page Header -->
      <div class="page-header text-center mb-5">
        <h1>
          <font-awesome-icon :icon="['fas', 'robot']" class="me-3" />
          AI Assistant
        </h1>
        <p class="lead">Intelligent tools to organize, analyze, and optimize your video library</p>
      </div>

      <!-- Dashboard Summary -->
      <AIDashboardSummary
        :library-health-score="libraryHealthScore"
        :total-videos-processed="totalVideosProcessed"
        :total-matches="totalMatches"
        :total-issues="totalIssues"
      />

      <!-- AI Features Grid -->
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
        <div class="col-md-6">
          <AIAnalyzeQuality />
        </div>
        <div class="col-md-6">
          <AIDetectMissingMetadata />
        </div>
        <div class="col-md-6">
          <AIDetectDuplicates />
        </div>
        <div class="col-md-6">
          <AISuggestNaming />
        </div>
        <div class="col-md-6">
          <AILibraryAnalytics />
        </div>
        <div class="col-md-6">
          <AIAnalyzeThumbnails />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import AIDashboardSummary from '@/components/ai/AIDashboardSummary.vue'
import AILinkPerformers from '@/components/ai/AILinkPerformers.vue'
import AISuggestTags from '@/components/ai/AISuggestTags.vue'
import AIDetectScenes from '@/components/ai/AIDetectScenes.vue'
import AIClassifyContent from '@/components/ai/AIClassifyContent.vue'
import AIAnalyzeQuality from '@/components/ai/AIAnalyzeQuality.vue'
import AIDetectMissingMetadata from '@/components/ai/AIDetectMissingMetadata.vue'
import AIDetectDuplicates from '@/components/ai/AIDetectDuplicates.vue'
import AISuggestNaming from '@/components/ai/AISuggestNaming.vue'
import AILibraryAnalytics from '@/components/ai/AILibraryAnalytics.vue'
import AIAnalyzeThumbnails from '@/components/ai/AIAnalyzeThumbnails.vue'

// Dashboard summary will be calculated from child component events
// Or use provide/inject pattern for shared state if needed

const libraryHealthScore = computed(() => {
  // Calculate from aggregated stats
  return 85 // Placeholder
})

const totalVideosProcessed = computed(() => {
  // Aggregate from all features
  return 0
})

const totalMatches = computed(() => {
  // Aggregate from all features
  return 0
})

const totalIssues = computed(() => {
  // Aggregate from all features
  return 0
})
</script>

<style scoped>
@import '@/styles/pages/ai_page.css';
</style>
```

---

## Implementation Order

### Week 1: Shared Components
1. **Day 1:** Create AIFeatureCard wrapper
2. **Day 2:** Create AIResultsTable component
3. **Day 3:** Create AIDashboardSummary
4. **Day 4:** Create composables (useAIFeature, useAISelection, useAIExport)

### Week 2: Extract Features (Part 1)
5. **Day 5:** Extract AILinkPerformers (most complex)
6. **Day 6:** Extract AISuggestTags (similar to performers)
7. **Day 7:** Test both extracted components

### Week 3: Extract Features (Part 2)
8. **Day 8:** Extract AIDetectScenes, AIClassifyContent
9. **Day 9:** Extract AIAnalyzeQuality, AIDetectMissingMetadata
10. **Day 10:** Test all 6 components

### Week 4: Final Features & Integration
11. **Day 11:** Extract AIDetectDuplicates (with modal)
12. **Day 12:** Extract AISuggestNaming, AILibraryAnalytics, AIAnalyzeThumbnails
13. **Day 13:** Refactor main AIPage.vue
14. **Day 14:** Full integration testing

---

## Expected Results

### Before:
- 1 file: 1,757 lines
- Hard to modify
- 10 features mixed together
- Duplicate logic
- Difficult to test

### After:
- 1 main page: 250 lines
- 13 extracted components: ~1,700 lines total
- 3 composables: ~300 lines of shared logic
- **Total:** ~2,250 lines (distributed across 17 files)

### Benefits:
- **Easy to modify:** Change one feature without affecting others
- **Reusable logic:** Composables used across features
- **Clear organization:** One file = one responsibility
- **AI-friendly:** Each file small enough to read in full
- **Parallel development:** Multiple features can be worked on simultaneously

---

## Testing Strategy

### For Each Extracted Component:
1. Import and mount in isolation
2. Test with mock data
3. Verify all interactions work
4. Check props/events/slots
5. Test empty states
6. Test error handling

### For Main Page:
1. All features visible
2. Dashboard summary updates
3. No console errors
4. Performance (should be faster with code splitting)

---

## Next Steps

1. ✅ Create refactoring plan
2. ⏳ Create component directories (done)
3. ⏳ Create AIFeatureCard wrapper
4. ⏳ Create useAIFeature composable
5. ⏳ Extract AILinkPerformers
6. ⏳ Extract remaining 9 features
7. ⏳ Refactor main AIPage
8. ⏳ Test everything

---

**Status:** Plan complete, ready to begin implementation
**Estimated Time:** 2-3 weeks for complete refactoring
**Expected Outcome:** 10x improvement in development speed for AI page features
