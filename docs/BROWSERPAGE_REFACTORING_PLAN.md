# BrowserPage.vue Refactoring Plan

**Date:** December 30, 2025
**Current Size:** 1,561 lines
**Target Size:** ~800 lines (49% reduction)
**Complexity:** HIGH - Multi-panel file browser with drag-drop
**Priority:** HIGH (2nd largest file)

---

## 📊 Current Structure Analysis

### Overview
BrowserPage.vue is a sophisticated multi-panel file browser featuring:
- **Split Panel System** - Up to 4 resizable panels
- **Tabbed Navigation** - Multiple tabs within each panel
- **Drag & Drop** - Files between libraries, tabs between panels
- **Advanced Filtering** - Search, type filters, sorting
- **Video Management** - Mark not interested, edit list
- **Zoom Controls** - Grid scaling (0.5x - 2.0x)
- **Keyboard Shortcuts** - Back, Ctrl+T, Ctrl+W, F5
- **Video Preview** - Hover preview with frame animation

### Template Structure (Lines 1-412)

1. **Controls Bar** (Lines 3-22)
   - Zoom controls (+, -, reset)
   - Add panel button

2. **Split Panels Container** (Lines 25-352)
   - 1-4 resizable panels
   - Each panel contains:
     - Tabs header (35-62)
     - Library selector (70-78)
     - Breadcrumb navigation (81-128)
     - Search & filter bar (131-220)
     - Content grid (231-288)
     - Pagination controls (291-342)

3. **Context Menu** (Lines 355-378)
   - Right-click menu for items

4. **Video Preview Popup** (Lines 381-402)
   - Hover preview with positioning

5. **Video Player Modal** (Lines 405-412)
   - Full video playback

### Script Structure (Lines 414-1559)

**11 Major State Objects:**
- Libraries & Tabs management
- Split panels configuration
- Drag & Drop state
- UI state (zoom, modals, selections)
- Preview state

**12 Functional Groups:**
1. Library & Tab Management (~100 lines)
2. Split Panel System (~70 lines)
3. Tab Drag-Drop (~30 lines)
4. Item Drag-Drop (~120 lines)
5. Content Loading & Filtering (~150 lines)
6. Sorting & Pagination (~60 lines)
7. Video Playback (~40 lines)
8. Video Marking (~20 lines)
9. Zoom Controls (~5 lines)
10. Context Menu (~30 lines)
11. Video Preview (~80 lines)
12. Keyboard Shortcuts (~35 lines)

---

## 🎯 Extraction Strategy

### Phase 1: High-Impact Components (Lines Saved: ~320)

#### 1. BrowserContentGridItem.vue (200 lines saved)
**Extracted From:** Template lines 238-287 (repeated in v-for)
**Purpose:** Individual grid item (folder or video)

**Props:**
```javascript
{
  item: Object,           // File/folder data
  zoomLevel: Number,      // Grid scale
  isSelected: Boolean,    // Selection state
  tab: Object,            // Parent tab reference
  libraryId: Number       // Library context
}
```

**Features:**
- Thumbnail display (video or folder icon)
- Video metadata (duration, resolution, FPS)
- Badges (not interested, edit list, has performers)
- Drag-drop handlers
- Double-click navigation
- Context menu trigger
- Selection handling
- Preview on hover

**Emits:**
```javascript
emit('navigate', path)
emit('play-video', item)
emit('show-context-menu', { item, x, y })
emit('toggle-selection', item)
emit('start-preview', item)
emit('stop-preview')
emit('drag-start', item)
emit('drag-end')
```

#### 2. BrowserSearchFilterBar.vue (80 lines saved)
**Extracted From:** Template lines 131-220
**Purpose:** Search, filter, and sort controls

**Props:**
```javascript
{
  search: String,
  filterType: String,     // 'all', 'videos', 'folders'
  sortBy: String,         // 'name', 'date', 'size', 'duration'
  sortOrder: String,      // 'asc', 'desc'
  showNotInterested: Boolean,
  showEditList: Boolean,
  isLoading: Boolean
}
```

**Emits:**
```javascript
emit('update:search', value)
emit('update:filterType', type)
emit('update:sortBy', field)
emit('update:sortOrder', order)
emit('update:showNotInterested', value)
emit('update:showEditList', value)
emit('refresh')
```

#### 3. BrowserPaginationControls.vue (40 lines saved)
**Extracted From:** Template lines 291-342
**Purpose:** Pagination UI

**Props:**
```javascript
{
  currentPage: Number,
  totalPages: Number,
  itemsPerPage: Number,
  totalItems: Number
}
```

**Emits:**
```javascript
emit('update:currentPage', page)
emit('update:itemsPerPage', size)
emit('previous-page')
emit('next-page')
```

---

### Phase 2: Medium-Impact Components (Lines Saved: ~73)

#### 4. BrowserBreadcrumbNav.vue (40 lines saved)
**Extracted From:** Template lines 81-128
**Purpose:** Path navigation breadcrumbs

**Props:**
```javascript
{
  pathSegments: Array,    // ['folder1', 'folder2']
  showFilterAlert: Boolean,
  filterType: String
}
```

**Emits:**
```javascript
emit('navigate-to', index)
emit('back')
```

#### 5. BrowserVideoPreview.vue (18 lines saved)
**Extracted From:** Template lines 381-402
**Purpose:** Hover preview popup

**Props:**
```javascript
{
  video: Object,
  position: Object,       // { x, y }
  frames: Array,
  currentFrame: Number
}
```

**Features:**
- Smart positioning (avoid screen edges)
- Frame animation
- Video duration display

#### 6. BrowserContextMenu.vue (15 lines saved)
**Extracted From:** Template lines 355-378
**Purpose:** Right-click context menu

**Props:**
```javascript
{
  visible: Boolean,
  x: Number,
  y: Number,
  item: Object,
  canPlay: Boolean,       // Computed based on item type
  canOpen: Boolean
}
```

**Emits:**
```javascript
emit('play')
emit('open')
emit('toggle-not-interested')
emit('toggle-edit-list')
emit('copy-path')
emit('close')
```

---

### Phase 3: Composables for Shared Logic

#### 1. useBrowserDragDrop.js (150 lines)
**Purpose:** Centralize all drag-drop logic

```javascript
export function useBrowserDragDrop(tabs, splitPanels, libraries, toast) {
  // Item drag-drop state
  const draggedItem = ref(null)
  const draggedFromTab = ref(null)
  const dropTargetTab = ref(null)

  // Tab drag-drop state
  const draggingTab = ref(null)
  const draggingFromPanel = ref(null)

  // Item drag handlers
  const onItemDragStart = (item, tabId) => { ... }
  const onItemDragOver = (event, tabId) => { ... }
  const onItemDrop = async (tabId) => { ... }
  const onItemDragEnd = () => { ... }

  // Tab drag handlers
  const onTabDragStart = (tab, panelId) => { ... }
  const onTabDragOver = (event, panelId) => { ... }
  const onTabDrop = (panelId) => { ... }
  const onTabDragEnd = () => { ... }

  return {
    // State
    draggedItem,
    draggedFromTab,
    dropTargetTab,
    draggingTab,
    draggingFromPanel,
    // Methods
    onItemDragStart,
    onItemDragOver,
    onItemDrop,
    onItemDragEnd,
    onTabDragStart,
    onTabDragOver,
    onTabDrop,
    onTabDragEnd
  }
}
```

#### 2. useBrowserKeyboard.js (50 lines)
**Purpose:** Keyboard shortcuts logic

```javascript
export function useBrowserKeyboard(tabs, splitPanels, loadContent) {
  const handleKeyDown = (event) => {
    // Backspace/Alt+Left: Navigate back
    if (event.key === 'Backspace' || (event.altKey && event.key === 'ArrowLeft')) {
      navigateBack()
    }

    // Ctrl+T: New tab
    if (event.ctrlKey && event.key === 't') {
      event.preventDefault()
      addNewTab()
    }

    // Ctrl+W: Close tab
    if (event.ctrlKey && event.key === 'w') {
      event.preventDefault()
      closeCurrentTab()
    }

    // F5/Ctrl+R: Refresh
    if (event.key === 'F5' || (event.ctrlKey && event.key === 'r')) {
      event.preventDefault()
      refreshCurrentTab()
    }
  }

  return { handleKeyDown }
}
```

#### 3. useBrowserSelection.js (80 lines)
**Purpose:** Multi-item selection logic

```javascript
export function useBrowserSelection() {
  const selectedItems = ref({})      // { tabId: Set<itemId> }
  const lastSelectedIndex = ref({})  // { tabId: number }

  const toggleSelection = (tabId, item, index, items, ctrlKey, shiftKey) => {
    // Ctrl+Click: Toggle single item
    // Shift+Click: Range selection
    // Regular click: Clear and select
  }

  const clearSelection = (tabId) => { ... }
  const selectAll = (tabId, items) => { ... }
  const isSelected = (tabId, itemId) => { ... }

  return {
    selectedItems,
    lastSelectedIndex,
    toggleSelection,
    clearSelection,
    selectAll,
    isSelected
  }
}
```

#### 4. useBrowserPreview.js (100 lines)
**Purpose:** Video preview functionality

```javascript
export function useBrowserPreview(browserAPI) {
  const previewVideo = ref(null)
  const previewPosition = ref({ x: 0, y: 0 })
  const previewFrames = ref([])
  const previewFrameIndex = ref(0)
  const previewTimeout = ref(null)
  const previewInterval = ref(null)

  const startPreview = async (item, event) => {
    // Delay 300ms before showing
    // Load preview frames
    // Position popup intelligently
    // Start frame animation
  }

  const stopPreview = () => {
    // Clear timeouts/intervals
    // Reset state
  }

  const updatePreviewPosition = (event) => {
    // Smart positioning to avoid screen edges
  }

  return {
    previewVideo,
    previewPosition,
    previewFrames,
    previewFrameIndex,
    startPreview,
    stopPreview
  }
}
```

---

## 📁 Target File Structure

```
src/
├── components/
│   └── browser/                    # NEW - Browser Components
│       ├── BrowserContentGridItem.vue
│       ├── BrowserSearchFilterBar.vue
│       ├── BrowserPaginationControls.vue
│       ├── BrowserBreadcrumbNav.vue
│       ├── BrowserVideoPreview.vue
│       └── BrowserContextMenu.vue
├── composables/
│   ├── useBrowserDragDrop.js      # NEW
│   ├── useBrowserKeyboard.js      # NEW
│   ├── useBrowserSelection.js     # NEW
│   └── useBrowserPreview.js       # NEW
└── views/
    └── BrowserPage.vue            # REFACTORED - ~800 lines
```

---

## 🎯 Expected Results

### Before
```
BrowserPage.vue (1,561 lines)
├── All grid item logic inline
├── All filter controls inline
├── All drag-drop logic mixed in
├── All keyboard shortcuts inline
├── Difficult to test or modify
└── Hard to understand flow
```

### After
```
BrowserPage.vue (~800 lines) - Orchestrator
├── BrowserContentGridItem.vue (250 lines)
├── BrowserSearchFilterBar.vue (180 lines)
├── BrowserPaginationControls.vue (120 lines)
├── BrowserBreadcrumbNav.vue (100 lines)
├── BrowserVideoPreview.vue (80 lines)
├── BrowserContextMenu.vue (70 lines)
├── useBrowserDragDrop.js (150 lines)
├── useBrowserKeyboard.js (50 lines)
├── useBrowserSelection.js (80 lines)
└── useBrowserPreview.js (100 lines)
```

### Benefits
1. **49% smaller main file** (1,561 → 800 lines)
2. **Reusable components** (grid item, filters, pagination)
3. **Testable in isolation** (each component/composable)
4. **Clear separation** (UI vs. logic)
5. **Better organization** (easy to find specific functionality)

---

## 📋 Implementation Checklist

### Phase 1: Foundation
- [ ] Create `src/components/browser/` directory
- [ ] Create `BrowserContentGridItem.vue`
- [ ] Create `BrowserSearchFilterBar.vue`
- [ ] Create `BrowserPaginationControls.vue`
- [ ] Test each component in isolation

### Phase 2: Supporting Components
- [ ] Create `BrowserBreadcrumbNav.vue`
- [ ] Create `BrowserVideoPreview.vue`
- [ ] Create `BrowserContextMenu.vue`

### Phase 3: Logic Extraction
- [ ] Create `useBrowserDragDrop.js` composable
- [ ] Create `useBrowserKeyboard.js` composable
- [ ] Create `useBrowserSelection.js` composable
- [ ] Create `useBrowserPreview.js` composable

### Phase 4: Main File Refactoring
- [ ] Update BrowserPage.vue to import all components
- [ ] Replace inline templates with components
- [ ] Replace inline logic with composables
- [ ] Test drag-drop functionality
- [ ] Test keyboard shortcuts
- [ ] Test multi-panel system
- [ ] Verify all features work

### Phase 5: Testing & Verification
- [ ] Frontend build successful
- [ ] No compilation errors
- [ ] Multi-panel layout works
- [ ] Tab drag-drop works
- [ ] Item drag-drop works
- [ ] Video playback works
- [ ] Context menu works
- [ ] Keyboard shortcuts work
- [ ] Video preview works
- [ ] Zoom controls work

---

## ⚠️ Challenges & Considerations

### Complex State Management
- Split panels, tabs, and items all have interdependent state
- Need careful prop drilling or consider provide/inject
- May need event bus for cross-component communication

### Drag-Drop Complexity
- Item drag-drop between libraries requires API calls
- Tab drag-drop between panels requires state manipulation
- Must preserve all edge case handling

### Performance
- Grid with 500+ items must remain performant
- Virtual scrolling might be needed later
- Zoom feature requires efficient re-rendering

### Keyboard Shortcuts
- Must work regardless of focused element
- Need global event listeners
- Should be easy to disable/enable

---

## 🚀 Next Steps

1. **Start with BrowserContentGridItem** - Highest impact
2. **Then BrowserSearchFilterBar** - Self-contained
3. **Then composables** - Extract complex logic
4. **Finally refactor main file** - Orchestration

**Estimated Time:** 8-12 hours total
**Expected Outcome:** 49% reduction, better organization, easier maintenance

---

**Status:** Planning Complete - Ready to Begin Implementation
