# Optimization Integration Guide

This guide shows you how to integrate the new optimization utilities into your existing components for immediate performance gains.

## 🚀 Quick Start - Highest Impact Changes

### 1. Virtual Scrolling for Videos Page (Recommended First Step)

The videos page currently renders all videos at once. With virtual scrolling, you'll only render what's visible.

**File:** [src/views/VideosPage.vue](../src/views/VideosPage.vue)

**Current Implementation:**
```vue
<div class="videos-grid" v-if="viewMode === 'grid'">
  <VideoCard
    v-for="video in videos"
    :key="video.id"
    v-memo="[video.id, video.title, video.rating, selectedVideos.includes(video.id)]"
    :video="video"
    ...
  />
</div>
```

**Optimized Implementation:**
```vue
<script setup>
import VirtualScroller from '@/components/VirtualScroller.vue'
// ... existing imports
</script>

<template>
  <!-- Grid View with Virtual Scrolling -->
  <div v-if="viewMode === 'grid'" class="videos-container">
    <VirtualScroller
      :items="videos"
      :item-height="280"
      height="calc(100vh - 250px)"
      :buffer="3"
      key-field="id"
    >
      <template #item="{ item: video }">
        <VideoCard
          :video="video"
          :is-selected="selectedVideos.includes(video.id)"
          @toggle-select="toggleVideoSelection"
          @context-menu="showContextMenu"
          @play="playVideo"
          @add-tag="openTagModal"
          @edit-metadata="editMetadata"
          @open-performer="openPerformer"
          @open-studio="openStudio"
        />
      </template>
    </VirtualScroller>
  </div>
</template>

<style scoped>
.videos-container {
  padding: 20px;
}

/* Update grid layout to work within virtual scroller */
.videos-container :deep(.scroll-container) {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 20px;
  padding: 10px;
}
</style>
```

**Expected Impact:**
- Initial render: ~5000 videos → ~20 videos
- Faster scrolling and interactions
- Lower memory usage

---

### 2. Request Caching for API Calls

Add caching to frequently accessed API endpoints to reduce redundant network requests.

**File:** [src/views/VideosPage.vue](../src/views/VideosPage.vue)

**Add to script:**
```javascript
import { useCachedRequest } from '@/composables/useRequestCache'
import requestCache from '@/utils/requestCache'

// Replace direct API calls with cached versions
const {
  data: cachedVideos,
  loading: videosLoading,
  execute: fetchVideos,
  invalidate: invalidateVideosCache
} = useCachedRequest(
  videosAPI.getAll,
  {
    cacheKey: 'all-videos',
    ttl: 5 * 60 * 1000, // 5 minutes
    enableCache: true
  }
)

// On component mount
onMounted(async () => {
  await fetchVideos()
  videos.value = cachedVideos.value || []
})

// Invalidate cache when videos change
const handleVideoUpdate = async (videoId, updates) => {
  await videosAPI.update(videoId, updates)
  invalidateVideosCache() // Clear cache
  await fetchVideos() // Refresh
}

// Invalidate on operations that modify videos
const handleVideoDelete = async (videoId) => {
  await videosAPI.delete(videoId)
  requestCache.invalidatePattern('all-videos')
  requestCache.invalidatePattern('video-')
  await fetchVideos()
}
```

**Expected Impact:**
- 40-60% fewer API calls for repeated page visits
- Faster page loads on navigation back
- Reduced server load

---

### 3. Optimized Filtering for Large Lists

Replace expensive filtering operations with optimized versions.

**File:** [src/views/VideosPage.vue](../src/views/VideosPage.vue)

**Current:**
```javascript
const filteredVideos = computed(() => {
  let result = videos.value

  if (searchQuery.value) {
    result = result.filter(video =>
      video.title.toLowerCase().includes(searchQuery.value.toLowerCase())
    )
  }

  if (selectedLibrary.value) {
    result = result.filter(video => video.library_id === selectedLibrary.value)
  }

  // More filters...

  return result
})
```

**Optimized:**
```javascript
import { optimizedFilter, memoizedComputed } from '@/utils/computedOptimizer'

const filteredVideos = memoizedComputed(
  () => {
    let result = videos.value

    // Use optimizedFilter for large arrays
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase()
      result = optimizedFilter(result, video =>
        video.title.toLowerCase().includes(query)
      )
    }

    if (selectedLibrary.value) {
      result = optimizedFilter(result, video =>
        video.library_id === selectedLibrary.value
      )
    }

    // Apply other filters...

    return result
  },
  [videos, searchQuery, selectedLibrary], // Dependencies
  { maxCacheSize: 20 } // Cache last 20 filter combinations
)
```

**Expected Impact:**
- 2-3x faster filtering on 1000+ items
- Cached results for common filter combinations
- Smoother real-time search

---

### 4. Debounced Search

Add debouncing to search inputs to reduce unnecessary computations.

**File:** [src/views/VideosPage.vue](../src/views/VideosPage.vue)

**Add to script:**
```javascript
import { debounce } from '@/utils/debounce'

// Create debounced search handler
const searchQuery = ref('')
const debouncedSearchQuery = ref('')

const updateSearch = debounce((value) => {
  debouncedSearchQuery.value = value
}, 300)

// Watch for search input changes
watch(searchQuery, (newValue) => {
  updateSearch(newValue)
})

// Use debouncedSearchQuery in your filters
const filteredVideos = computed(() => {
  let result = videos.value

  if (debouncedSearchQuery.value) {
    const query = debouncedSearchQuery.value.toLowerCase()
    result = optimizedFilter(result, video =>
      video.title.toLowerCase().includes(query)
    )
  }

  return result
})
```

**In template:**
```vue
<input
  v-model="searchQuery"
  type="text"
  placeholder="Search videos..."
  class="form-control"
/>
```

**Expected Impact:**
- No lag during typing
- Reduced CPU usage
- Better UX for search

---

## 📊 Performance Monitoring in Development

### Track Component Performance

**File:** [src/views/VideosPage.vue](../src/views/VideosPage.vue)

```javascript
import performanceMonitor from '@/utils/performanceMonitor'

export default {
  name: 'VideosPage',

  setup() {
    // Track initial load
    onMounted(() => {
      performanceMonitor.start('VideosPage:mount')
      // ... your mount logic
      performanceMonitor.end('VideosPage:mount')
    })

    // Track filtering performance
    const applyFilters = async () => {
      await performanceMonitor.measure('VideosPage:filter', async () => {
        // Your filtering logic
      })
    }

    // Track API calls
    const fetchVideos = async () => {
      return await performanceMonitor.measure('API:getVideos', async () => {
        return await videosAPI.getAll()
      })
    }

    return { /* ... */ }
  }
}
```

**View performance report in console:**
```javascript
// Open browser console and type:
performanceMonitor.report()

// Or check specific metrics:
performanceMonitor.getStats('VideosPage:mount')
```

---

## 🎯 Component-Specific Optimizations

### PerformersPage.vue

**Add Virtual Scrolling:**
```vue
<VirtualScroller
  :items="filteredPerformers"
  :item-height="300"
  height="calc(100vh - 200px)"
  key-field="id"
>
  <template #item="{ item: performer }">
    <div
      class="performer-card"
      v-memo="[performer.id, performer.name, performer.video_count]"
      @click="openDetails(performer)"
    >
      <!-- Performer card content -->
    </div>
  </template>
</VirtualScroller>
```

---

### AICompanionChat.vue

**Add Request Caching for Chat History:**
```javascript
import { useCachedRequest } from '@/composables/useRequestCache'

const {
  data: memories,
  execute: fetchMemories,
  invalidate: invalidateMemories
} = useCachedRequest(
  aiAPI.getMemories,
  {
    cacheKey: 'ai-memories',
    ttl: 2 * 60 * 1000, // 2 minutes
  }
)

// After sending message, invalidate cache
const sendMessage = async (message) => {
  await aiAPI.chat(message)
  invalidateMemories()
  await fetchMemories()
}
```

---

### TagsPage.vue

**Optimize Tag Filtering:**
```javascript
import { optimizedFilter, debouncedComputed } from '@/utils/computedOptimizer'

const searchTerm = ref('')

const { value: filteredTags } = debouncedComputed(
  () => {
    if (!searchTerm.value) return tags.value

    const query = searchTerm.value.toLowerCase()
    return optimizedFilter(tags.value, tag =>
      tag.name.toLowerCase().includes(query)
    )
  },
  300 // 300ms debounce
)
```

---

## 🔧 Advanced Optimizations

### Lazy Computed for Expensive Operations

Use lazy computed for data that's not always needed:

```javascript
import { lazyComputed } from '@/utils/computedOptimizer'

const { value: videoStatistics, refresh: refreshStats } = lazyComputed(() => {
  // Expensive calculation
  return videos.value.reduce((stats, video) => {
    stats.totalDuration += video.duration
    stats.totalSize += video.file_size
    stats.avgRating += video.rating
    return stats
  }, { totalDuration: 0, totalSize: 0, avgRating: 0 })
})

// Only computed when accessed
console.log(videoStatistics.value)

// Manually refresh when needed
refreshStats()
```

---

### Profiled Computed for Debugging

Track performance of specific computed properties:

```javascript
import { profiledComputed } from '@/utils/computedOptimizer'

const expensiveComputation = profiledComputed('expensiveFilter', () => {
  // Your expensive logic here
  return videos.value.filter(/* complex logic */)
})

// In console, check stats:
expensiveComputation.getStats()
// { name: 'expensiveFilter', callCount: 45, totalTime: '1234.56', avgTime: '27.43' }
```

---

## 📋 Migration Checklist

Use this checklist to track your optimization progress:

- [ ] **VideosPage.vue**
  - [ ] Implement virtual scrolling
  - [ ] Add request caching
  - [ ] Optimize filtering with memoizedComputed
  - [ ] Add debounced search

- [ ] **PerformersPage.vue**
  - [ ] Implement virtual scrolling
  - [ ] Optimize grid rendering
  - [ ] Cache performer data

- [ ] **TagsPage.vue**
  - [ ] Add debounced search
  - [ ] Cache tag list

- [ ] **AICompanionChat.vue**
  - [ ] Cache chat history
  - [ ] Cache memories

- [ ] **API Services** ([src/services/api.js](../src/services/api.js))
  - [ ] Add caching wrapper for frequently called endpoints
  - [ ] Implement request deduplication

---

## 🎨 CSS Optimization Notes

The PurgeCSS configuration in [vue.config.js](../vue.config.js) automatically removes unused CSS in production builds. The safelist includes:

- All Bootstrap utility classes
- FontAwesome icons
- Animation classes
- Dynamic classes

If you add new dynamic classes, update the safelist:

```javascript
safelist: {
  standard: [
    /^btn-/,
    /^bg-/,
    /^your-new-pattern-/,
    // ...
  ]
}
```

---

## 📈 Expected Performance Gains

| Component | Optimization | Expected Improvement |
|-----------|-------------|---------------------|
| VideosPage | Virtual Scrolling | 90% faster initial render |
| VideosPage | Request Caching | 50% fewer API calls |
| VideosPage | Optimized Filter | 3x faster filtering |
| PerformersPage | Virtual Scrolling | 85% faster render |
| Search Inputs | Debouncing | No lag during typing |
| API Calls | Gzip Compression | 70% smaller payloads |
| Database | WAL + Pooling | 5x better concurrency |

---

## 🐛 Debugging Tips

### Check Cache Statistics
```javascript
// In browser console
import requestCache from '@/utils/requestCache'
requestCache.getStats()
```

### Monitor Performance
```javascript
// View all performance metrics
performanceMonitor.report()

// Clear metrics
performanceMonitor.clear()
```

### Verify Gzip Compression
```bash
# In browser DevTools Network tab, check Response Headers:
# Should see: Content-Encoding: gzip
```

---

## 💡 Best Practices

1. **Use virtual scrolling for lists > 100 items**
2. **Cache API calls that don't change frequently**
3. **Debounce search inputs (300ms is a good default)**
4. **Use memoizedComputed for expensive filters/sorts**
5. **Monitor performance in development, disable in production**
6. **Invalidate caches after mutations**
7. **Keep v-memo dependencies minimal and specific**

---

## 🔗 Related Documentation

- [Optimization Report](./OPTIMIZATION_REPORT.md) - Complete overview of all optimizations
- [Vue Performance Guide](https://vuejs.org/guide/best-practices/performance.html)
- [SQLite WAL Mode](https://www.sqlite.org/wal.html)

---

## 🆘 Need Help?

If you encounter issues or have questions about integrating these optimizations, check:

1. The [Optimization Report](./OPTIMIZATION_REPORT.md) for detailed implementation examples
2. The source files for the utilities (they have extensive JSDoc comments)
3. The browser console for performance metrics and warnings

Happy optimizing! 🚀
