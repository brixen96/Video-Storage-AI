# VIDEO STORAGE AI — COMPLETE TECHNICAL SPECIFICATION

> Generated: 2026-02-23 | Purpose: AI planning reference

---

## TABLE OF CONTENTS

1. [Project Overview](#1-project-overview)
2. [Project Structure](#2-project-structure)
3. [Frontend Architecture](#3-frontend-architecture)
   - Routes
   - Views
   - Components
   - Services (api.js)
   - WebSocket
   - Composables
4. [Backend Architecture](#4-backend-architecture)
   - Server Bootstrap
   - Database Schema (30+ tables)
   - API Routes (200+ endpoints)
   - Services (50+ files)
   - Models
   - WebSocket Hub
5. [AI Companion System](#5-ai-companion-system)
6. [JDownloader 2 Integration](#6-jdownloader-2-integration)
7. [Scraper System](#7-scraper-system)
8. [Key Cross-Cutting Features](#8-key-cross-cutting-features)
9. [API Response Formats](#9-api-response-formats)
10. [Configuration & Deployment](#10-configuration--deployment)
11. [Performance & Extension Points](#11-performance--extension-points)

---

## 1. PROJECT OVERVIEW

**Project Name:** Video Storage AI
**Tech Stack:** Vue 3 (Frontend) + Go/Gin (Backend) + SQLite (Database)
**Purpose:** A comprehensive self-hosted video management system with advanced multi-source scraping, AI-powered organization, performer/studio/tag management, and JDownloader 2 integration.

**Key Characteristics:**
- Single-page application (SPA) served by the Go backend
- RESTful JSON API + WebSocket for real-time updates
- 8+ site-specific scrapers + extensible custom tab system
- Always-running AI Companion with LM Studio (local LLM) integration
- JDownloader 2 remote control via deprecated HTTP API
- Activity logging, link verification, and auto-update scheduling
- Fully local — no external telemetry, all data stored in SQLite

**Code Footprint:**
- Frontend: ~3,000 lines in main SPA pages after tab extraction refactor
- Backend: ~1MB+ of Go service code across 50+ files
- Database: 30+ SQLite tables

---

## 2. PROJECT STRUCTURE

```
c:\Repos\Video Storage AI/
├── api/                              # Go backend
│   ├── cmd/server/main.go            # Entry point & server bootstrap
│   └── internal/
│       ├── api/                      # HTTP handlers (one file per feature)
│       │   ├── router.go             # Route registration
│       │   ├── video_handlers.go
│       │   ├── performer_handlers.go
│       │   ├── scraper_handlers.go
│       │   ├── scraper_tabs_handlers.go
│       │   ├── jdownloader_handlers.go
│       │   ├── ai_handlers.go
│       │   ├── auto_update_handlers.go
│       │   ├── scheduler_handlers.go
│       │   ├── dashboard_handlers.go
│       │   ├── system_handlers.go
│       │   ├── link_verification_handlers.go
│       │   ├── sexuria_handlers.go
│       │   ├── hqporner_handlers.go
│       │   ├── leakvids_handlers.go
│       │   ├── motherless_handlers.go
│       │   └── video_scraper_handlers.go
│       ├── services/                 # Business logic (50+ files)
│       ├── models/                   # Data models (20 files)
│       ├── database/                 # DB init, schema, migrations
│       │   └── database.go
│       ├── websocket/
│       │   └── hub.go               # WebSocket hub
│       ├── config/                   # App configuration
│       ├── middleware/               # HTTP middleware
│       └── logger/                   # Logging utilities
├── src/                              # Vue 3 frontend
│   ├── views/                        # 21 page components
│   ├── components/
│   │   ├── scraper/tabs/             # 7 extracted scraper tab components
│   │   ├── shared/                   # LoadingState, EmptyState, StatCard, etc.
│   │   ├── browser/                  # File browser
│   │   └── performers/               # Performer-specific display
│   ├── services/
│   │   ├── api.js                    # All API calls (20 namespaces)
│   │   └── websocket.js              # WebSocket client (singleton)
│   ├── composables/
│   │   ├── useFormatters.js
│   │   ├── useScraperJobPolling.js
│   │   ├── useVideoSelection.js
│   │   └── ...
│   ├── router/index.js               # Vue Router config
│   ├── plugins/fontawesome.js        # FontAwesome icon registry
│   ├── styles/                       # CSS organized by component & page
│   ├── App.vue                       # Root component
│   └── main.js                       # Entry point
├── public/                           # Static files
├── dist/                             # Production build (served by backend)
├── package.json
└── Dockerfile
```

**Runtime Data:**
- `api/data/database.db` — SQLite database (+ WAL journal files)
- `api/data/assets/` — Performer previews, thumbnails, video previews
- `api/data/backups/` — Database backup archives

---

## 3. FRONTEND ARCHITECTURE

### 3.1 Routes (`src/router/index.js`)

All routes are lazy-loaded via webpack code splitting. Scroll behavior resets to top on navigation.

| Path | Component | Purpose |
|------|-----------|---------|
| `/` | `HomePage.vue` | Dashboard — stats, running tasks, scraper feed |
| `/videos` | `VideosPage.vue` | Browse & manage local video library |
| `/watch/:id` | `VideoPlayerPage.vue` | Local video player |
| `/watch/scraped/:id` | `VideoPlayerPage.vue` | Scraped video player |
| `/performers` | `PerformersPage.vue` | Browse performers with filtering |
| `/performers/:id` | `PerformerDetailsPage.vue` | Performer profile & linked videos |
| `/studios` | `StudiosPage.vue` | Studio management |
| `/tags` | `TagsPage.vue` | Tag management & merging |
| `/libraries` | `LibrariesPage.vue` | Library path management |
| `/browser` | `BrowserPage.vue` | Filesystem browser for libraries |
| `/activity` | `ActivityDashboard.vue` | Background task monitoring |
| `/settings` | `SettingsPage.vue` | Application configuration |
| `/tasks` | `TasksPage.vue` | Scheduled task management |
| `/edit-list` | `EditListPage.vue` | Video list editing & bulk ops |
| `/scraper` | `ScraperPage.vue` | Multi-source scraper interface |
| `/scraper/:id` | `ScraperThreadPage.vue` | Scraped thread detail view |
| `/system-health` | `SystemHealthPage.vue` | System health monitoring |
| `/backups` | `BackupsPage.vue` | Database backup management |
| `/jdownloader` | `JDownloaderPage.vue` | JDownloader 2 control panel |
| `/ai` | `AIPage.vue` | AI Companion & analysis tools |

### 3.2 Views (Pages) in Detail

#### `HomePage.vue` — Dashboard
- **Data:** `stats` (totals: videos, scraped_videos, threads, performers, running_tasks), `feedItems`, `runningTasks`, `scraperConfigs`
- **API calls:** `GET /dashboard/stats`, `GET /dashboard/feed`, `GET /dashboard/running-tasks`, `GET /dashboard/scrapers`, `POST /auto-update/configs/:type/check-now`
- **Components:** `DashboardStatCard`, `DashboardFeed`, running task widgets
- **Features:** Auto-refreshes every 30s, WebSocket feed updates, scraper status chips

#### `VideosPage.vue` — Video Library
- **Data:** video list with full metadata, library selector, filter state, view mode
- **Filter options:** performers, tags, studios, content type, resolution, duration, rating, codec, library, favorite/pinned/edit-list flags
- **View modes:** grid, list, compact
- **Bulk ops:** assign tags, delete, convert to MP4
- **API:** `GET /videos`, `POST /videos/scan`, `POST /videos/scan-all-parallel`, `PATCH /videos/marks-by-path`

#### `ScraperPage.vue` — Multi-Tab Scraper (2,951 lines after extraction)
- **Built-in tabs (each now a standalone component):**
  1. `Simpcity Forum` — forum thread scraping with auth
  2. `PornHub` → `PornHubTab.vue`
  3. `LuxureTV` → `LuxureTVTab.vue`
  4. `Sexuria` → `SexuriaTab.vue`
  5. `Rule34Video` → `Rule34VideoTab.vue`
  6. `Motherless` → `MotherlessTab.vue`
  7. `LeakVids` → `LeakVidsTab.vue`
  8. `HQPorner` → `HQPornerTab.vue`
- **Dynamic tabs:** scraper_tabs from DB render as additional nav tabs
- **Simpcity still lives inside ScraperPage.vue** (complex: auth modal, scrape modal, thread detail modal, forum scrape modal, WebSocket link verification events)
- **Per-tab sidebar:** collapsible, contains AutoUpdatePanel + statistics + quick actions
- **Remaining JS in ScraperPage.vue:** Simpcity auth, thread management, video tab, custom tab engine, JDownloader check, link verification, `openInFirefox`, `getVideoPerformers`, `getVideoMatchedTags`

#### `JDownloaderPage.vue`
- **Sections:** Connection config, System info, Download speed, LinkGrabber, Download queue, History, Bulk ops
- **Key state:** `jdConfig`, `systemInfo`, `speed`, `linkGrabberPackages`, `downloadPackages`, `history`
- **Keyboard shortcuts:** Ctrl+S start, Ctrl+X stop, Space pause, Ctrl+E export
- **API:** Full JDownloader namespace (see Section 3.4)

#### `AIPage.vue`
- **Tabs:** Companion (chat UI) + Tools (batch AI ops)
- **Shows:** LM Studio connection status, AI config, stats
- **API:** `/ai/chat`, `/ai/config`, `/ai/test-connection`, `/ai/status`

#### `PerformersPage.vue`
- **Categories:** regular, zoo, 3d, all
- **Sort:** name, age, breast size, height, video count
- **Bulk:** cleanup empty performers, generate thumbnails
- **API:** `/performers`, `/performers/fetch-metadata`, `/performers/generate-thumbnails`

#### `ActivityDashboard.vue`
- **Filters:** status (running, completed, failed, paused), task_type
- **Actions per task:** pause, resume, cancel, retry
- **API:** `/activity`, `/activity/:id/pause`, `/activity/:id/resume`

### 3.3 Components

#### Navigation
- **`NavBar.vue`** — Sticky nav with WebSocket status indicator, active task count badge, notification bell, settings link
- Collapses to hamburger on mobile

#### Video Display
- **`VideoCard.vue`** — Grid card for local videos
  - Props: `video`, `isSelected`
  - Emits: `click`, `toggle-select`, `play`, `add-tag`, `edit-metadata`, `open-performer`
  - Features: hover thumbnail cycling (10 frames), status badges (favorite, pinned, rating, converted), metadata badges (resolution, fps, bitrate, file size, play count, codec, source)
- **`ScrapedVideoCard.vue`** — Variant for scraped videos, shows source URL and external link button

#### Dashboard Components
- **`DashboardStatCard.vue`** — Icon + value + label stat card
- **`DashboardFeed.vue`** — Paginated activity feed, filterable by scraper type
- **`DashboardStatCard.vue`** — Used on homepage for 6 top-level counts

#### Scraper Components
- **`AutoUpdatePanel.vue`** — Configures automated scraper scheduling; Props: `scraper-type`, `endpoint-type`; Shows interval, last run, next run, toggle enabled
- **`ScraperBulkActions.vue`** — Selection toolbar; Props: `selectionMode`, `selectedCount`; Emits: `select-all`, `delete-selected`
- **`ScraperJobProgress.vue`** — Active job progress bar & stats; Props: `job`, `label`
- **`ScraperPaginationControls.vue`** — Pagination buttons with page info
- **`ScrapedVideoCard.vue`** — Card for scraped videos

#### Scraper Tab Components (`src/components/scraper/tabs/`)

All follow the same pattern:
- Self-contained `<script setup>` with local state
- `sidebarCollapsed = ref(true)` (local, not shared)
- `activeTimeouts = ref([])` with `onBeforeUnmount` cleanup
- `useVideoSelection()` for bulk selection
- `useScraperJobPolling()` for job polling
- `openInFirefox(url)` local helper
- `onMounted` loads initial data

| File | ~Lines | Unique Features |
|------|--------|----------------|
| `HQPornerTab.vue` | 264 | performer/tag matching display |
| `PornHubTab.vue` | ~350 | JDownloader integration, `useRouter` for `/watch/scraped/:id`, client-side filtering of 10K+ videos |
| `LuxureTVTab.vue` | ~280 | `fetchEmbeds`, `startScrapeFromFirstCategory` helper |
| `Rule34VideoTab.vue` | ~290 | sortBy ref + sortOptions array, "Straight 3D" filter note |
| `MotherlessTab.vue` | ~280 | `searchTermFilter` dropdown from stats.search_term_counts |
| `LeakVidsTab.vue` | ~300 | Most filter options: duration, minRating, minViews, search, sort |
| `SexuriaTab.vue` | ~460 | Hover card with teleport, screenshot modal, links modal, link extraction/verification, advanced search |

#### AI & Chat
- **`AICompanionChat.vue`** (~23KB) — Floating chat widget
  - Toggle open/close
  - Chat history with role indicators (user/assistant/tool)
  - Markdown rendering via `MarkdownMessage.vue`
  - Search through history
  - Export conversations
  - Suggested prompts for cold start
  - Proactive alert display from WebSocket `companion_event`
  - Message copy & timestamps
  - Connection status indicator

#### Shared Components (`src/components/shared/index.js`)
- `LoadingState` — Spinner with optional text
- `EmptyState` — Icon + title + message for empty lists
- `StatCard` — Mini stat display used in sidebar panels
- `SearchBox` — Debounced search input
- `DeleteConfirmationModal` — Confirm before delete dialog

#### Other Components
- `VideoPlayerModal.vue` — Inline video player modal
- `AddPerformerModal.vue`, `AddTagModal.vue`, `AddToListModal.vue`, `EditMetadataModal.vue`
- `ToastNotification.vue` — Toast messages (success/error/warning/info)
- `MarkdownMessage.vue` — Render markdown with syntax highlighting
- `LazyImage.vue` — Lazy-load with fallback placeholder
- `SkeletonCard.vue` — Loading skeleton
- `TagBadge.vue` — Colored tag chip
- `VirtualScroller.vue` — Virtual scrolling for 1000+ item lists

### 3.4 Frontend Services (`src/services/api.js`)

**Base URL:** `http://localhost:8080/api/v1`
**Timeout:** 600 seconds
**Response handling:** Axios interceptor returns `response.data` directly — all callers receive `{ success, data, message }` directly.

#### API Namespaces & Methods

**`librariesAPI`**
```
getAll()  getPrimary()  getById(id)  create(data)  update(id, data)  delete(id)
browse(id, path, metadata)  generateThumbnails(id, path)
```

**`performersAPI`**
```
getAll(params)  search(query)  getById(id)  getPreviews(id)  create(data)  scan()
update(id, data)  delete(id)  cleanup()
fetchMetadata(id)  resetMetadata(id)  resetPreviews(id)
generateThumbnails()  generateThumbnail(id)
getTags(id)  addTag(id, tagId)  removeTag(id, tagId)  syncTags(id)
getVideos(id, params)  getScrapedThreads(id)
```

**`videosAPI`**
```
getAll(params)  getById(id)  search(params)  create(data)  update(id, data)  delete(id)
scan(libraryId, path)  scanAllParallel()
generatePreviews(videoIds)  generateThumbnails(videoIds)
openInExplorer(id)  updateVideoMarksByPath(data)
convert(id, options)
```

**`studiosAPI`**
```
getAll()  getById(id)  create(data)  update(id, data)  delete(id)  resetMetadata(id)
```

**`groupsAPI`**
```
getAll()  getById(id)  create(data)  update(id, data)  delete(id)  resetMetadata(id)
```

**`tagsAPI`**
```
getAll()  getById(id)  create(data)  update(id, data)  delete(id)  merge(sourceId, targetId)
```

**`filesAPI`**
```
scan(path)  rename(id, name)  move(id, targetPath)
moveAcrossLibraries(id, targetLibraryId, targetPath)  delete(id)
```

**`videoListsAPI`**
```
getAll()  getById(id)  create(data)  update(id, data)  delete(id)
getItems(id, params)  addVideos(id, videoIds)  removeVideos(id, videoIds)
reorderItems(id, items)  getListsForVideo(videoId)  exportList(id, format)
```

**`activityAPI`**
```
getAll(params)  getRecent()  getStatus()  getStats()  getPaused()
getById(id)  create(data)  update(id, data)  delete(id)
clean(olderThanDays)  clearAll()
pause(id, checkpoint)  resume(id)  cancel(id)  retry(id)
```

**`databaseAPI`**
```
getStats()  optimize()  backup()  listBackups()  restore(filename)
```

**`aiAPI`**
```
// Analysis
linkPerformers(params)  applyLinks(links)
suggestTags(videoIds)  applyTagSuggestions(suggestions)
detectScenes(videoId)  classifyContent(videoIds)  analyzeQuality(videoIds)
detectMissingMetadata()  detectDuplicates()  suggestNaming(videoIds)
getLibraryAnalytics()  analyzeThumbnailQuality()

// Companion
chat(message, history)  getStatus()
saveMemory(content)  getMemories()  searchMemories(query)
getConfig()  updateConfig(data)  testConnection()
```

**`scrapersAPI`** (largest namespace)
```
// Session (Simpcity auth)
getSession()  setSession({ cookie })

// Threads
getScrapedThreads(params)  searchScrapedThreads(q)  getScrapedThread(id)
scrapeThread({ url })  rescrapeThread(id)
deleteThread(id)  deleteThreads(ids)  deleteAllThreads()
toggleThreadWatch(id)  getRecentPosts(id)  markPostRead(id)

// Forum
scrapeForumCategory({ url, maxPages })  scrapeForumAndSaveAll(data)

// Performer linking
autoLinkThreadsToPerformers()  linkThreadToPerformer(threadId, performerId)

// Links
checkLinkStatuses(ids)  verifyThreadLinks(id)  getThreadLinkStats(id)
exportThreadLinks(id, format)  exportAllThreadsLinks(format)

// Video scrapers (generic)
scrapeVideo({ url, siteType })  testScraperHandler({ url })
getScrapedVideos(params)  deleteAllScrapedVideos()
deleteScrapedVideosBySiteType(type)  bulkDelete(ids)

// Tabs
getScraperTabs()  getScraperTab(id)  createScraperTab(data)
updateScraperTab(id, data)  deleteScraperTab(id)
detectPagination(url1, url2)

// Jobs
getScraperJobs(tabId?)  getScraperJob(id)
startScraperJob(tabId, params)  cancelScraperJob(id)

// Open URL
openURL(url)

// PornHub
pornhub.scrape(params)  pornhub.getVideos(params)  pornhub.getStats()

// LuxureTV
luxuretv.scrape(params)  luxuretv.getCategories()  luxuretv.getVideos(params)
luxuretv.getStats()  luxuretv.fetchEmbeds(videoId)

// Sexuria
sexuria.scrape(params)  sexuria.getCategories()  sexuria.getItems(params)
sexuria.getStats()  sexuria.getItemDetails({ url })  sexuria.getSearchOptions()
sexuria.advancedSearch(searchData)
sexuria.extractItemLinks(itemId, url)  sexuria.getItemLinks(itemId)
sexuria.verifyItemLinks(itemId)

// Rule34Video
rule34video.scrape(params)  rule34video.getVideos(params)
rule34video.getStats()  rule34video.getSortOptions()

// Motherless
motherless.scrape(params)  motherless.getVideos(params)
motherless.getStats()  motherless.getSearchTerms()

// LeakVids
leakvids.scrape(params)  leakvids.getVideos(params)  leakvids.getStats()

// HQPorner
hqporner.scrape(params)  hqporner.getVideos(params)  hqporner.getStats()
```

**`downloadTrackingAPI`**
```
markLinkAsDownloaded(id)  markLinkAsFailed(id)  resetLinkDownloadStatus(id)
bulkMarkDownloaded(ids)
getThreadDownloadStats(id)  getPerformerDownloadStats(id)  getGlobalDownloadStats()
```

**`jdownloaderAPI`**
```
// Config
getConfig()  updateConfig(data)  testConnection()  checkAvailability()

// System
getSystemInfo()  getSpeed()  setSpeedLimit(limit)

// Send
sendToJDownloader(data)  sendLinksToJDownloader(links, packageName)
sendThreadToJDownloader(threadId)

// History
getHistory()  clearHistory()  getPackageTimestamps()

// Control
getConnectionStatus()  start()  stop()  pause()

// LinkGrabber
getLinkGrabberLinks()  getLinkGrabberPackages()
moveToDownloads(linkIds, packageIds)  removeLinkGrabberLinks(linkIds, packageIds)
clearLinkGrabber()

// Downloads
getDownloadLinks()  getDownloadPackages()
removeDownloadLinks(linkIds, packageIds)  forceDownload(linkIds, packageIds)
cleanupFinished()  resetLinks(linkIds, packageIds)

// Common
setLinkEnabled(linkIds, packageIds, enabled)
setLinkPriority(linkIds, packageIds, priority)
renamePackage(packageId, name)
```

**`schedulerAPI`**
```
getStatus()  getAllJobs()  getJob(id)
createJob(data)  updateJob(id, data)  deleteJob(id)  toggleJob(id)
getJobExecutionHistory(id)
```

**`autoUpdateAPI`**
```
getConfigs()  getConfig(type)  updateConfig(type, data)
toggle(type, enabled)  checkNow(type)  getStatus()
```

**`dashboardAPI`**
```
getStats()  getFeed()  getScrapers()  getRunningTasks()
```

**`systemAPI`**
```
openInIncognito(url)  openLoginHelper(url)  openUrl(url)
```

**`notificationsAPI`**
```
getNotifications(params)  getStats()  getById(id)  create(data)
markAsRead(id)  markAllAsRead()  archiveNotification(id)
delete(id)  cleanOld(olderThanDays)
```

**`backupsAPI`**
```
createDatabaseBackup()  listDatabaseBackups()  getBackup(filename)
downloadBackup(filename)  restoreBackup(filename)
deleteBackup(filename)  cleanupOldBackups()
```

**`aiAuditAPI`**
```
getAuditLogs(params)  getStats()  searchAuditLogs(query)
getLogsByPerformer(id)  getLogsByThread(id)  deleteOldLogs(days)
```

### 3.5 WebSocket Client (`src/services/websocket.js`)

**Singleton** initialized in `App.vue`, shared via import.

**Connection:** `ws://localhost:8080/api/v1/ws`
**Reconnection:** Up to 5 attempts, 3-second delay between retries

**Event subscriptions (`.on(eventType, handler)`):**

| Event Type | Payload | Used By |
|-----------|---------|---------|
| `activity_update` | `{ id, task_type, status, progress, message }` | ActivityDashboard, NavBar badge |
| `status_update` | `{ ... }` | General status indicator |
| `console_log` | `{ source, level, message, details }` | Debug panel |
| `companion_event` | `{ type, severity, message, data }` | AICompanionChat proactive alerts |
| `link_verification_complete` | `{ thread_id, active, dead, expired, total }` | ScraperPage (Simpcity tab) |
| `feed_update` | feed item object | DashboardFeed |
| `notification` | notification object | NavBar bell |

### 3.6 Composables (`src/composables/`)

**`useFormatters.js`** — Returns pure formatting functions:
- `formatDuration(seconds)` → `"MM:SS"` or `"HH:MM:SS"`
- `formatFileSize(bytes)` → `"2.5 GB"`
- `formatDate(date)` → localized date
- `formatDateTime(date)` → localized date+time
- `formatTotalDuration(seconds)` → `"2h 15m"`
- `formatNumber(num)` → `"1,234,567"`
- `formatCompactNumber(num)` → `"1.2K"`, `"1.5M"`
- `formatPercentage(value, total, decimals)`
- `timeAgo(dateStr)` → `"3h ago"`, `"just now"`

**`useScraperJobPolling.js`** — Returns `{ pollJob }`:
- `pollJob(jobId, { activeJob, isScraping, activeTimeouts, onSuccess, onFail })`
- Polls job status every 2s, stops on completion/failure
- Pushes timeout IDs to `activeTimeouts` for cleanup

**`useVideoSelection.js`** — Returns selection state & actions:
- `selectionMode: ref(false)`
- `selectedVideos: ref(new Set())`
- `toggleSelectionMode()`
- `toggleVideoSelection(id)`
- `selectAll(items)`

**`useFormatters.js`** — `formatDate`, `formatDateTime`, `formatDuration`, `formatFileSize`, `formatCompactNumber`, `formatNumber`, `formatPercentage`, `timeAgo`, `formatTotalDuration`

**`useRequestCancellation.js`** — Provides abort signal for concurrent request cancellation

**`useRequestCache.js`** — Caches API responses with TTL expiration

**`useTableSelection.js`** — Extended table row selection management

**`useAIExport.js`** — Export AI chat/analysis results to file

---

## 4. BACKEND ARCHITECTURE

### 4.1 Server Bootstrap (`api/cmd/server/main.go`)

**Initialization Sequence:**
1. Load `.env` configuration (DATABASE_PATH, SERVER_HOST, SERVER_PORT, SERVER_MODE)
2. Initialize SQLite with WAL mode, connection pooling (25 max, 5 idle, 1h lifetime)
3. Run schema migrations (`runMigrations()`)
4. Setup console logging service
5. Check for paused tasks, create notification if any found
6. Scan performer folders, auto-generate missing thumbnails
7. Setup Gin router with middleware: Logger, Recovery, CORS, gzip, cache-control
8. Initialize WebSocket hub (centralized broadcast hub)
9. Initialize AI Companion service (always-running background goroutine)
10. Bridge AI Companion `CompanionEvent` channel → WebSocket hub broadcast
11. Initialize + start Scheduler service (background goroutine)
12. Serve static files from `/dist` (SPA fallback)
13. Start HTTP server (graceful shutdown on SIGINT/SIGTERM)

### 4.2 Database Schema (`api/internal/database/database.go`)

**SQLite Settings:**
- WAL journal mode (concurrent readers, serialized writer)
- Foreign keys: `ON`
- Max open connections: 25, max idle: 5, lifetime: 1h, idle timeout: 10min

---

#### Core Library Tables

**`libraries`**
```sql
id INTEGER PRIMARY KEY
name TEXT UNIQUE NOT NULL
path TEXT NOT NULL
primary_lib BOOLEAN DEFAULT FALSE
created_at, updated_at DATETIME
```

**`videos`**
```sql
id INTEGER PRIMARY KEY
library_id INTEGER REFERENCES libraries
title TEXT
file_path TEXT UNIQUE
file_size INTEGER, duration INTEGER
codec TEXT, resolution TEXT, bitrate INTEGER, fps REAL
thumbnail_path TEXT, preview_path TEXT
date DATETIME, rating INTEGER (0-5)
description TEXT, content_type TEXT (regular|zoo|3d)
is_favorite BOOLEAN, is_pinned BOOLEAN
not_interested BOOLEAN, in_edit_list BOOLEAN
converted_from INTEGER REFERENCES videos  -- source of conversion
converted_to INTEGER REFERENCES videos    -- converted copy
play_count INTEGER DEFAULT 0
created_at, updated_at, last_played_at DATETIME
```

**`performers`**
```sql
id INTEGER PRIMARY KEY
name TEXT UNIQUE NOT NULL
preview_path TEXT, thumbnail_path TEXT, folder_path TEXT
category TEXT (regular|zoo|3d)
metadata TEXT (JSON: birthdate, birthplace, height, weight,
               measurements, career_start, career_end, aliases[],
               nationality, eye_color, hair_color, etc.)
created_at, updated_at DATETIME
```

**`studios`**
```sql
id INTEGER PRIMARY KEY
name TEXT UNIQUE NOT NULL
logo_path TEXT, description TEXT
founded_date DATE, country TEXT
metadata TEXT (JSON)
created_at, updated_at DATETIME
```

**`groups`** (performer groups/companies)
```sql
id INTEGER PRIMARY KEY
name TEXT UNIQUE NOT NULL
studio_id INTEGER REFERENCES studios
metadata TEXT (JSON)
created_at, updated_at DATETIME
```

**`tags`**
```sql
id INTEGER PRIMARY KEY
name TEXT UNIQUE NOT NULL
description TEXT, color TEXT, icon TEXT
created_at, updated_at DATETIME
```

---

#### Association (M2M) Tables

| Table | Columns | Purpose |
|-------|---------|---------|
| `video_performers` | `video_id, performer_id, character_name` | Video ↔ Performer |
| `video_tags` | `video_id, tag_id` | Video ↔ Tag |
| `video_studios` | `video_id, studio_id` | Video ↔ Studio |
| `video_groups` | `video_id, group_id` | Video ↔ Group |
| `performer_tags` | `performer_id, tag_id` | Performer master tags |
| `performer_scraped_threads` | `performer_id, thread_id, confidence_score, extraction_method` | Performer ↔ Thread |

---

#### Scraper Tables

**`scraped_threads`** (Simpcity forum threads)
```sql
id INTEGER PRIMARY KEY
external_id TEXT, source TEXT (simpcity, etc.)
title TEXT, url TEXT UNIQUE, author TEXT, category TEXT
view_count INTEGER, reply_count INTEGER, post_count INTEGER, download_count INTEGER
metadata TEXT (JSON: tags[], thumbnails[], extracted_performer_names[])
is_watched BOOLEAN DEFAULT FALSE
first_scraped_at, last_scraped_at, last_updated_at, created_at DATETIME
```

**`scraped_posts`** (Posts within threads)
```sql
id INTEGER PRIMARY KEY
thread_id INTEGER REFERENCES scraped_threads
external_id TEXT, source TEXT, author TEXT
content TEXT (HTML), plain_text TEXT
post_number INTEGER, like_count INTEGER
metadata TEXT (JSON: attachments[], quotes[])
is_read BOOLEAN DEFAULT FALSE
posted_at, scraped_at, created_at DATETIME
```

**`scraped_download_links`** (Links extracted from posts)
```sql
id INTEGER PRIMARY KEY
thread_id INTEGER REFERENCES scraped_threads
post_id INTEGER REFERENCES scraped_posts
source TEXT, provider TEXT (gofile|pixeldrain|bunkr|...)
url TEXT, original_url TEXT, filename TEXT
file_size INTEGER, file_type TEXT
status TEXT (active|dead|expired|unknown)
download_status TEXT, download_error TEXT
verified_at, expires_at, created_at, updated_at DATETIME
```

**`scraped_videos`** (Videos from all scrapers)
```sql
id INTEGER PRIMARY KEY
source TEXT (pornhub|luxuretv|sexuria|rule34video|motherless|leakvids|hqporner|...)
source_url TEXT, site_type TEXT
title TEXT, description TEXT
thumbnail_path TEXT, thumbnail_url TEXT
thumbnail_frames TEXT (JSON array of frame URLs for preview cycling)
duration INTEGER, view_count INTEGER, rating REAL
video_source TEXT (embed URL or direct URL)
performer_names TEXT (JSON array)
tags TEXT (JSON array)
metadata TEXT (JSON: file hosts, resolutions, uploader, etc.)
links_extracted BOOLEAN DEFAULT FALSE
category TEXT, search_term TEXT, uploader TEXT
created_at, updated_at DATETIME
```

**`scraper_tabs`** (Custom scraper configurations)
```sql
id INTEGER PRIMARY KEY
name TEXT, site_url TEXT
selector_config TEXT (JSON: container, title, url, thumbnail, pagination)
pagination_config TEXT (JSON: type, pattern, pageParam)
color TEXT, icon TEXT, enabled BOOLEAN DEFAULT TRUE
created_at, updated_at DATETIME
```

**`scraper_jobs`** (Job tracking)
```sql
id INTEGER PRIMARY KEY
scraper_type TEXT, tab_id INTEGER REFERENCES scraper_tabs
status TEXT (pending|running|completed|failed|cancelled|paused)
progress INTEGER (0-100), total_items INTEGER
items_found INTEGER, items_saved INTEGER, pages_completed INTEGER
error_message TEXT
started_at, completed_at, created_at, updated_at DATETIME
```

---

#### System Tables

**`activity_logs`**
```sql
id INTEGER PRIMARY KEY
task_type TEXT (scan|scrape|convert|verify|generate|import|export|cleanup|...)
status TEXT (running|completed|failed|paused|cancelled)
message TEXT, progress INTEGER (0-100)
checkpoint TEXT (JSON for pauseable/resumable tasks)
is_paused BOOLEAN DEFAULT FALSE
error_details TEXT
created_at, started_at, completed_at, updated_at DATETIME
```

**`scheduled_jobs`**
```sql
id INTEGER PRIMARY KEY
name TEXT, scraper_type TEXT
interval_minutes INTEGER, max_pages_to_check INTEGER
enabled BOOLEAN DEFAULT TRUE
last_run_at, next_run_at, created_at, updated_at DATETIME
```

**`job_execution_history`**
```sql
id INTEGER PRIMARY KEY
job_id INTEGER REFERENCES scheduled_jobs
status TEXT, started_at, completed_at DATETIME, error_message TEXT
```

**`app_settings`**
```sql
key TEXT PRIMARY KEY, value TEXT (JSON), updated_at DATETIME
```

**`ai_config`**
```sql
id INTEGER PRIMARY KEY
lm_studio_url TEXT DEFAULT 'http://localhost:1234'
model_name TEXT, temperature REAL DEFAULT 0.7
max_tokens INTEGER DEFAULT 2000
enabled BOOLEAN DEFAULT TRUE
created_at, updated_at DATETIME
```

**`memories`** / **`ai_companion_memory`**
```sql
id INTEGER PRIMARY KEY
content TEXT (JSON) / key TEXT, value TEXT (JSON)
created_at, updated_at DATETIME
```

**`notifications`**
```sql
id INTEGER PRIMARY KEY
type TEXT, title TEXT, message TEXT
severity TEXT (info|warning|error|critical)
is_read BOOLEAN, is_archived BOOLEAN
action_url TEXT, created_at, updated_at DATETIME
```

**`console_logs`**
```sql
id INTEGER PRIMARY KEY
source TEXT, level TEXT (info|warn|error)
message TEXT, details TEXT (JSON)
created_at DATETIME
```

**`ai_audit_logs`**
```sql
id INTEGER PRIMARY KEY
operation TEXT, model TEXT
input_tokens INTEGER, output_tokens INTEGER, cost REAL
request_summary TEXT, response_summary TEXT, error TEXT
performer_id INTEGER, thread_id INTEGER
created_at DATETIME
```

**`jdownloader_config`**
```sql
id INTEGER PRIMARY KEY
connection_type TEXT (local|cloud)
local_host TEXT DEFAULT 'localhost', local_port INTEGER DEFAULT 3128
local_password TEXT, cloud_email TEXT, cloud_password TEXT
cloud_device_id TEXT, cloud_device_name TEXT
default_package_name TEXT, auto_start_downloads BOOLEAN
enabled BOOLEAN, connection_status TEXT
last_connected_at, created_at, updated_at DATETIME
```

**`video_lists`**
```sql
id INTEGER PRIMARY KEY
name TEXT, description TEXT
created_at, updated_at DATETIME
```

**`video_list_items`**
```sql
list_id INTEGER REFERENCES video_lists
video_id INTEGER REFERENCES videos
position INTEGER, notes TEXT
PRIMARY KEY (list_id, video_id)
```

### 4.3 API Routes

**Base path:** `/api/v1`
Unmatched routes serve the SPA from `/dist`.

---

#### Libraries — `/libraries`
```
GET    /                   List all libraries
GET    /primary            Get primary library
GET    /:id                Get library by ID
GET    /:id/browse         Browse library filesystem (path param)
GET    /:id/stream         Stream video file
POST   /                   Create library
PUT    /:id                Update library
DELETE /:id                Delete library
POST   /:id/generate-thumbnails  Generate thumbnails for folder
```

#### Videos — `/videos`
```
GET    /                   List videos (filters: performer_id, tag_id, studio_id, library_id,
                           content_type, resolution, duration_min/max, rating_min, rating_max,
                           codec, search, sort, page, limit, favorites, pinned, edit_list,
                           not_interested, converted)
GET    /search             Text search
GET    /:id                Get video
POST   /                   Create video entry
PUT    /:id                Update video metadata
DELETE /:id                Delete video + file
POST   /scan               Scan library path
POST   /scan-all-parallel  Scan all libraries concurrently
POST   /generate-previews  Generate preview storyboards
POST   /generate-thumbnails Generate thumbnails
PATCH  /marks-by-path      Bulk update flags by file path prefix
PATCH  /content-type/folder  Set content_type for all videos in folder
POST   /:id/convert        Convert to MP4
```

#### Performers — `/performers`
```
GET    /                   List (params: search, category, sort, order, page, limit)
GET    /:id                Get performer + linked videos
GET    /:id/previews       Get preview video refs
GET    /:id/tags           Get master tags
GET    /:id/videos         Get linked videos (paginated)
GET    /:id/scraped-threads Get linked forum threads
POST   /                   Create performer
POST   /scan               Scan performer folders
POST   /cleanup            Remove empty performers
POST   /generate-thumbnails  Generate for all
PUT    /:id                Update performer
DELETE /:id                Delete performer
POST   /:id/fetch-metadata  Fetch from AdultDataLink API
POST   /:id/reset-metadata  Clear metadata
POST   /:id/reset-previews  Clear preview assignments
POST   /:id/generate-thumbnail  Generate single thumbnail
POST   /:id/tags           Add master tag
DELETE /:id/tags/:tagId    Remove master tag
POST   /:id/sync-tags      Push performer tags to all linked videos
```

#### Studios — `/studios`
```
GET    /    GET    /:id    POST   /    PUT    /:id    DELETE /:id
POST   /:id/reset-metadata
```

#### Groups — `/groups`
```
GET    /    GET    /:id    POST   /    PUT    /:id    DELETE /:id
POST   /:id/reset-metadata
```

#### Tags — `/tags`
```
GET    /    GET    /:id    POST   /    PUT    /:id    DELETE /:id
POST   /merge              Merge tag (sourceId → targetId)
```

#### Activity — `/activity`
```
GET    /                   List (params: status, task_type, limit, page)
GET    /recent             Recent activities
GET    /status             Currently running tasks
GET    /stats              Stats by task_type
GET    /paused             All paused tasks
GET    /:id                Get single activity
POST   /                   Create activity
PUT    /:id                Update activity
DELETE /:id                Delete activity
POST   /clean              Clean old activities
POST   /clear-all          Clear all activities
POST   /:id/pause          Pause task (body: checkpoint JSON)
POST   /:id/resume         Resume paused task
POST   /:id/cancel         Cancel task
POST   /:id/retry          Retry failed task
```

#### Files — `/files`
```
POST   /scan               Scan directory (return file list)
POST   /rename             Rename file
POST   /move               Move within library
POST   /move-across-libraries  Move between libraries
DELETE /delete             Delete file
```

#### Database — `/database`
```
GET    /stats              DB size, table row counts
POST   /optimize           VACUUM database
POST   /backup             Create .db backup copy
GET    /backups            List backup files
POST   /restore            Restore from backup file
```

#### AI — `/ai`
```
// Analysis
POST   /link-performers    Analyze & suggest performer links
POST   /apply-links        Apply performer link suggestions
POST   /suggest-tags       Smart tag suggestions for videos
POST   /apply-tag-suggestions  Apply tag suggestions
POST   /detect-scenes      Detect scene boundaries in video
POST   /classify-content   Classify content type
POST   /analyze-quality    Quality analysis
POST   /detect-missing-metadata  Find videos with incomplete metadata
POST   /detect-duplicates  Find duplicate videos
POST   /suggest-naming     Filename normalization suggestions
GET    /library-analytics  Comprehensive library statistics
POST   /analyze-thumbnail-quality  Analyze thumbnail quality scores

// Companion
POST   /chat               Chat with AI Companion
GET    /status             Companion status (connected, model, etc.)
POST   /memories           Save a memory
GET    /memories           List memories
GET    /memories/search    Search memories (q param)
PUT    /memories/:id       Update memory
DELETE /memories/:id       Delete memory

// Config
GET    /config             Get AI config (LM Studio URL, model, etc.)
PUT    /config             Update AI config
POST   /test-connection    Test LM Studio connection
```

#### Scraper — `/scraper` (largest group)
```
GET    /stats              Overall scraper statistics

// Session (Simpcity auth)
POST   /session            Set session cookie
GET    /session            Get session status

// Threads
GET    /threads            List threads (filters: source, search, watched, page, limit)
GET    /threads/watched    Watched threads only
GET    /threads/search     Text search
GET    /threads/:id        Thread with posts & links
POST   /threads/scrape     Scrape thread from URL
POST   /threads/:id/rescrape  Re-scrape existing thread
POST   /threads/:id/watch  Toggle watch status
GET    /threads/:id/recent-posts  Recent posts
POST   /posts/:id/mark-read  Mark post as read
DELETE /threads/:id        Delete thread
DELETE /threads            Delete multiple threads (body: ids[])
DELETE /threads/all        Delete all threads

// Forum category listing
POST   /forum/scrape-listing  Scrape forum category page
POST   /forum/scrape-all      Scrape entire forum (all categories)

// Performer linking
POST   /performers/auto-link       Auto-link threads to performers
POST   /performers/link            Manual link thread to performer
POST   /performers/create-from-thread  Create performer from thread title
POST   /links/check-status         Check statuses for given link IDs

// Link verification
POST   /threads/:id/verify-links   Verify all links in thread
GET    /threads/:id/link-stats     Link health stats for thread
POST   /links/verify-all           Batch verify all unverified links
GET    /links/verification-stats   Overall verification statistics

// Provider health
GET    /providers/health           All provider health summaries
GET    /providers/:provider/health Specific provider health

// Link export
GET    /threads/:id/export-links   Export links (format: txt|json|csv)
GET    /links/export-all           Export all links

// Generic video scraper
POST   /video              Scrape video from URL (body: { url, siteType })
GET    /video/test         Test scraper without saving
GET    /videos             Get all scraped videos (params: source, page, limit)
DELETE /videos             Delete all scraped videos
DELETE /videos/site/:siteType  Delete by site type
DELETE /videos/source/:source  Delete by source
POST   /videos/bulk-delete     Delete by IDs array
GET    /videos/:id         Get single scraped video
GET    /videos/duplicates  Find duplicate videos
DELETE /videos/duplicates  Delete duplicate videos

// Custom scraper tabs
GET    /tabs               Get all custom tabs
GET    /tabs/:id           Get tab
POST   /tabs               Create tab
PUT    /tabs/:id           Update tab
DELETE /tabs/:id           Delete tab
POST   /detect-pagination  Detect pagination pattern from two URLs

// Scraper jobs
GET    /jobs               List jobs (param: tabId optional)
GET    /jobs/:id           Get job
POST   /jobs/start         Start job (body: { tabId, ... })
POST   /jobs/:id/cancel    Cancel job

// PornHub
POST   /pornhub/scrape     Start scrape (params: keyword, maxPages)
GET    /pornhub/keywords   Suggested keywords
GET    /pornhub/videos     Scraped videos (paginated)
GET    /pornhub/stats      Stats (total, by keyword)

// LuxureTV
POST   /luxuretv/scrape    Start scrape (params: categoryUrl, maxPages)
GET    /luxuretv/categories All categories
GET    /luxuretv/videos    Scraped videos
GET    /luxuretv/stats     Stats
POST   /luxuretv/fetch-embeds  Fetch embed IDs for video

// Sexuria
POST   /sexuria/scrape     Start category scrape
GET    /sexuria/categories All categories
GET    /sexuria/items      Items (params: limit, offset, category)
GET    /sexuria/stats      Stats (total_items, host_counts)
GET    /sexuria/item-categories  Distinct categories
POST   /sexuria/item-details    Item details (body: { url })
GET    /sexuria/search-options  Advanced search options
POST   /sexuria/advanced-search  Advanced search
GET    /sexuria/links/stats     Link health stats
POST   /sexuria/items/:id/extract-links  Extract links from item page
GET    /sexuria/items/:id/links  Get extracted links
POST   /sexuria/items/:id/verify-links  Verify links for item

// Rule34Video
POST   /rule34video/scrape    Start scrape (tags, sort, maxPages)
GET    /rule34video/sort-options  Available sort options
GET    /rule34video/videos    Scraped videos (paginated)
GET    /rule34video/stats     Stats

// Motherless
POST   /motherless/scrape     Start scrape (searchUrl, maxPages)
GET    /motherless/videos     Scraped videos (paginated)
GET    /motherless/stats      Stats (total, by_search_term)
GET    /motherless/search-terms  Distinct search terms

// LeakVids
POST   /leakvids/scrape    Start scrape (baseUrl, minDuration, maxDuration,
                           minRating, minViews, sortBy, maxPages)
GET    /leakvids/videos    Scraped videos (paginated)
GET    /leakvids/stats     Stats

// HQPorner
POST   /hqporner/scrape    Start scrape
GET    /hqporner/videos    Scraped videos (paginated)
GET    /hqporner/stats     Stats
GET    /hqporner/performers  Matched performers in scraped data
GET    /hqporner/tags        Matched tags in scraped data

// Open URL
POST   /open-url           Open URL in Firefox (body: { url })
```

#### Download Tracking — `/downloads`
```
POST   /links/:id/mark-downloaded   Mark link as downloaded
POST   /links/:id/mark-failed       Mark link as failed
POST   /links/:id/reset             Reset download status
POST   /links/bulk-mark-downloaded  Bulk mark downloaded
GET    /threads/:id/stats           Thread download stats
GET    /performers/:id/stats        Performer download stats
GET    /stats                       Global download stats
GET    /threads                     Filter by download status
GET    /recent                      Recently downloaded
```

#### JDownloader — `/jdownloader`
```
// Config
GET    /config             Get connection config
PUT    /config             Update config
POST   /test               Test connection
GET    /available          Quick availability check (returns bool)

// System
GET    /system             System info (version, uptime, state, RAM)
GET    /speed              Current download speed
POST   /speed-limit        Set global speed limit

// Send
POST   /send               Send links (body: { links[], packageName })
POST   /send-links         Legacy send endpoint
POST   /threads/:id/send   Send entire thread's links to JD

// History
GET    /history            Send history
DELETE /history            Clear history
GET    /package-timestamps Locally tracked package add times

// Control
GET    /status             Connection status
POST   /start              Start all downloads
POST   /stop               Stop all downloads
POST   /pause              Pause/unpause downloads

// LinkGrabber
GET    /linkgrabber/links       Get links
GET    /linkgrabber/packages    Get packages
POST   /linkgrabber/move        Move to downloads
POST   /linkgrabber/remove      Remove links
POST   /linkgrabber/clear       Clear all

// Downloads
GET    /downloads/links         Get active download links
GET    /downloads/packages      Get download packages
POST   /downloads/remove        Remove links
POST   /downloads/force         Force start download
POST   /downloads/cleanup       Remove finished downloads
POST   /downloads/reset         Reset failed links

// Common operations
POST   /enable             Enable/disable links (body: { linkIds[], packageIds[], enabled })
POST   /priority           Set link priority
POST   /rename             Rename package
```

#### Scheduler — `/scheduler`
```
GET    /status             Scheduler status
GET    /jobs               All scheduled jobs
GET    /jobs/:id           Single job
POST   /jobs               Create job
PUT    /jobs/:id           Update job
DELETE /jobs/:id           Delete job
POST   /jobs/:id/toggle    Enable/disable job
GET    /jobs/:id/history   Execution history
```

#### Auto-Update — `/auto-update`
```
GET    /configs            All auto-update configs
GET    /configs/:type      Config for specific scraper type
PUT    /configs/:type      Update config
POST   /configs/:type/toggle   Toggle enabled
POST   /configs/:type/check-now  Trigger immediate check
GET    /status             Overall auto-update status
```

#### Dashboard — `/dashboard`
```
GET    /stats              Aggregated counts + trends
GET    /feed               Activity feed items (paginated)
GET    /scrapers           Scraper status cards
GET    /running-tasks      Currently executing tasks
```

#### System — `/system`
```
GET    /health             Comprehensive system health check
GET    /metrics            Time-series performance metrics
POST   /open-incognito     Open URL in incognito browser
POST   /open-login-helper  Open login page in browser
POST   /open-url           Open any URL in Firefox
```

#### Notifications — `/notifications`
```
GET    /                   List (params: unread_only, type, page, limit)
GET    /stats              Notification statistics
GET    /:id                Single notification
POST   /                   Create notification
POST   /:id/read           Mark as read
POST   /read-all           Mark all as read
POST   /:id/archive        Archive notification
DELETE /:id                Delete notification
POST   /clean              Delete old notifications
```

#### Backups — `/backups`
```
POST   /                   Create database backup
GET    /                   List backup files
GET    /stats              Backup statistics
GET    /:filename          Backup info
GET    /:filename/download Download backup file
POST   /:filename/restore  Restore from backup
DELETE /:filename          Delete backup
POST   /cleanup            Auto-cleanup old backups
```

#### Console Logs — `/console-logs`
```
GET    /          GET    /stats     GET    /:id
POST   /          DELETE /:id
POST   /clear     POST   /clean
```

#### Video Lists — `/lists`
```
GET    /                   All lists
GET    /for-video          Lists containing a specific video (q: videoId)
GET    /:id                Single list
GET    /:id/items          Items with full video details
GET    /:id/stats          List statistics
POST   /                   Create list
PUT    /:id                Update list
DELETE /:id                Delete list
POST   /:id/videos         Add videos (body: { videoIds[] })
DELETE /:id/videos         Remove videos (body: { videoIds[] })
POST   /:id/reorder        Reorder items
PUT    /:id/items/:itemId/notes  Update item notes
```

#### AI Audit — `/ai-audit`
```
GET    /             GET    /stats      GET    /search
GET    /:id          GET    /performer/:id   GET    /thread/:id
POST   /clean
```

#### WebSocket — `/ws`
```
GET    /ws          WebSocket upgrade endpoint
```

### 4.4 Backend Services

All live in `api/internal/services/`. Key services:

| Service File | ~Size | Purpose |
|-------------|-------|---------|
| `ai_companion_service.go` | 178KB | Always-running AI background; file watchers; proactive events; LM Studio tool calling |
| `video_scraper_service.go` | 114KB | Generic video scraping; multi-source dispatch; deduplication |
| `scraper_service.go` | 107KB | Forum scraping (Simpcity); thread management; link verification; provider health |
| `video_service.go` | 65KB | Video CRUD; scan; preview/thumbnail generation; conversion; search |
| `ai_service.go` | 39KB | AI analysis tools (linking, tagging, classification, deduplication) |
| `jdownloader_service.go` | 38KB | JD HTTP API wrapper; `doJDAction` helper; all JD operations |
| `link_verification_service.go` | 26KB | Link health checks per provider; stats tracking |
| `sexuria_scraper_service.go` | 39KB | Sexuria scraping; link extraction; verification; advanced search |
| `activity_service.go` | 35KB | Activity logging; pause/resume/cancel; progress tracking |
| `scheduler_service.go` | 23KB | Cron-like job scheduling; execution history |
| `performer_service.go` | 19KB | Performer CRUD; folder scan; metadata; tag sync |
| `luxuretv_scraper_service.go` | 18KB | LuxureTV; embed ID fetch; category scraping |
| `hqporner_scraper_service.go` | 23KB | HQPorner; performer/tag matching |
| `rule34video_scraper_service.go` | 20KB | Rule34Video; tag search; sort options |
| `media_service.go` | 16KB | FFmpeg: extract video info, generate thumbnails/previews |
| `pornhub_scraper_service.go` | 15KB | PornHub keyword scraping |
| `motherless_scraper_service.go` | 13KB | Motherless search scraping |
| `leakvids_scraper_service.go` | 12KB | LeakVids multi-filter scraping |
| `auto_update_service.go` | 13KB | Per-scraper scheduled auto-updates |
| `notification_service.go` | 12KB | Notification CRUD; read/archive management |
| `download_tracking_service.go` | 12KB | Track download status per link |
| `scraper_tabs_service.go` | 12KB | Custom tab CRUD; pagination detection |
| `performer_thumbnail_service.go` | 10KB | Batch thumbnail generation |
| `ai_audit_service.go` | 10KB | AI operation audit trail |
| `smart_tagging.go` | 10KB | Intelligent tag suggestion logic |
| `backup_service.go` | 11KB | DB backup creation/restore/cleanup |
| `browser_service.go` | 9KB | Filesystem browser |
| `file_service.go` | 9KB | File rename/move/delete |
| `performer_scan_service.go` | 8KB | Performer folder discovery |
| `database_service.go` | 8KB | DB stats, optimize, backup |
| `performer_service.go` | 19KB | Performer CRUD; scan; metadata; tag sync |
| `library_service.go` | 7KB | Library CRUD |
| `tag_service.go` | 4KB | Tag CRUD; merge |
| `studio_service.go` | 7KB | Studio CRUD |
| `console_log_service.go` | 7KB | Console log CRUD |
| `scanner_service.go` | 5KB | Directory file scan |

### 4.5 WebSocket Hub (`api/internal/websocket/hub.go`)

**Architecture:**
- Single `Hub` struct maintained in server
- `clients map[*Client]bool` — registered clients
- `broadcast chan []byte` — message queue
- Per-client `send chan []byte` (buffered 256) to prevent blocking
- Client disconnection handled gracefully (slow clients kicked)

**Broadcast Methods (called by services):**
```go
hub.BroadcastActivityUpdate(data interface{})
hub.BroadcastStatusUpdate(data interface{})
hub.BroadcastConsoleLog(data interface{})
hub.BroadcastCompanionEvent(data interface{})
hub.BroadcastLinkVerificationComplete(data interface{})
hub.BroadcastFeedUpdate(data interface{})
hub.BroadcastNotification(data interface{})
```

**Wire-up:** `ai_companion_service.go` emits to a `chan CompanionEvent`, and `main.go` bridges those events to `hub.BroadcastCompanionEvent()`. Link verification sends directly from `scraper_service.go` after verification completes.

---

## 5. AI COMPANION SYSTEM

### 5.1 Architecture (ai_companion_service.go — Always-Running Goroutine)

The AI Companion is initialized at server start and runs permanently in the background.

**Initialization:**
1. Load `ai_config` from DB (LM Studio URL, model, temperature, max_tokens)
2. Subscribe to library file system changes via file watcher
3. Start periodic analysis ticker (every N minutes)
4. Start console log monitor goroutine
5. Open `CompanionEvent` channel for WebSocket bridge

**Components:**

**Context Builder (`GetFullContext()`):**
Returns a rich system prompt containing:
- Library stats: total videos, performers, tags, studios, libraries
- Scraper stats: videos/threads/posts count per source
- Currently running tasks (from activity_logs)
- Recent errors and warnings
- Database health metrics

**File Watchers:**
- Monitors configured library paths
- Emits `CompanionEvent{ Type: "file_added"|"file_removed"|"file_modified" }` on change
- Frontend AICompanionChat shows proactive notifications for these events

**Periodic Analysis:**
- Checks library health metrics
- Looks for missing metadata, untagged videos, unlinked performers
- Generates insight events for significant findings

**Event Types (CompanionEvent):**
```go
type CompanionEvent struct {
    Type     string      // file_added, file_removed, insight, notification
    Severity string      // info, warning, error, critical
    Message  string
    Data     interface{}
    Title    string
}
```

### 5.2 Chat Flow (POST /ai/chat)

```
Frontend AICompanionChat
    → POST /ai/chat { message, history[] }
    → backend: ChatCompanion(message, history)

ChatCompanion():
    1. Build system prompt via GetFullContext()
    2. Try ChatLLM() with LM Studio:
       a. POST to {lm_studio_url}/v1/chat/completions
       b. Messages array: [system, ...history, user]
       c. tools: 15 function definitions
       d. Loop up to 5 iterations:
          - If response has tool_calls: execute tool, append result, re-call LLM
          - If response has content: return to user
    3. On LM Studio failure: fallback to rule-based Chat() function
    4. Broadcast companion_event via WebSocket
    5. Return { response, model, tokens_used }
```

**LM Studio Tools Available (15):**
1. `get_video_library_stats` — Total counts
2. `search_videos` — Query videos by filters
3. `get_performer_info` — Performer details
4. `search_performers` — Find performers
5. `get_tag_info` — Tag details
6. `get_activity_status` — Running/recent tasks
7. `get_scraper_stats` — Scraper statistics
8. `get_scraped_threads` — Forum thread data
9. `check_jdownloader_status` — JD availability
10. `get_link_health` — Provider health stats
11. `get_library_analytics` — Detailed analytics
12. `get_settings` — App settings
13. `get_ai_config` — Current AI config
14. `get_notifications` — Recent notifications
15. `run_database_query` — (restricted) Safe DB reads

### 5.3 Frontend Chat UI (AICompanionChat.vue)

- **Floating widget** — collapsed by default, toggle button in bottom-right corner
- **Chat history** — messages with timestamps, role indicators (user/AI/tool call)
- **Markdown rendering** — code blocks, bold, links via `MarkdownMessage.vue`
- **Search** — filter chat history by keyword
- **Export** — download conversation as text/JSON
- **Suggested prompts** — shown on empty chat for quick start
- **Proactive alerts** — WebSocket `companion_event` triggers toast-like notification
- **Connection status** — shows if LM Studio is connected
- **Memory** — saves user preferences across sessions

---

## 6. JDOWNLOADER 2 INTEGRATION

### 6.1 Protocol Details

**JDownloader Deprecated HTTP API** (not the new REST API):
- Default port: `3128` (some installs: `8297`)
- Base URL: `http://localhost:3128`
- Authentication: Optional `Authorization: Basic` header or token

**Request Patterns:**

| Method Type | Body Format | Examples |
|-------------|-------------|---------|
| Single-object-param | Named JSON object `{ "key": value }` | `addLinks`, `queryPackages`, `queryLinks` |
| Multi-param | Positional JSON array `[param1, param2]` | `removeLinks`, `setEnabled`, `setPriority` |

**Required header for all requests:**
```
Accept-Encoding: identity
```
(Prevents gzip encoding which Go's HTTP client may not handle correctly)

**`doJDAction(endpoint, params)` helper** — Unified request handler with:
- Automatic retry on connection failure
- Correct header injection
- Config-aware base URL

### 6.2 JD Endpoints Used

```
POST /linkgrabber/cleanup
POST /linkgrabber/add            addLinks({ links, packageName })
POST /linkgrabber/query/links    queryLinks({ ... })
POST /linkgrabber/query/packages queryPackages({ ... })
POST /linkgrabber/move/downloads moveLinksToDownloadlist([linkIds, packageIds])
POST /linkgrabber/remove         removeLinks([linkIds, packageIds])

POST /downloads/query/links      queryDownloads({ ... })
POST /downloads/query/packages   queryPackages({ ... })
POST /downloads/remove           removeLinks([linkIds, packageIds])
POST /downloads/force            forceDownload([linkIds, packageIds])
POST /downloads/cleanup          cleanupPackages(...)
POST /downloads/reset            resetLinks([linkIds, packageIds])

POST /links/setenabled           setEnabled([linkIds, packageIds, enabled])
POST /links/setpriority          setPriority([linkIds, packageIds, priority])
POST /links/rename               renamePackage([packageId, name])

POST /jd/state                   getAppState()
GET  /update/version             getVersion()
GET  /jd/speed                   getSpeed()
POST /jd/setSpeedLimit           setSpeedLimit([limit])
GET  /system/informaticons       getSystemInformation()
```

### 6.3 Frontend JDownloader Page Features

- **Connection Status** — color-coded indicator (connected/disconnected/error)
- **System Info** — JD version, uptime, RAM usage, state
- **Speed Monitor** — Real-time download speed + limit setter
- **LinkGrabber Panel** — Packages list with link counts, move/remove operations
- **Downloads Panel** — Active downloads with progress, force/remove/reset operations
- **History** — Recently sent packages with timestamps
- **Bulk Select** — Multi-select packages/links for batch operations
- **Keyboard Shortcuts** — Ctrl+S start, Ctrl+X stop, Space pause, Ctrl+E export links

---

## 7. SCRAPER SYSTEM

### 7.1 Common Scraper Infrastructure

**Job System:**
- Each scraper run creates a `scraper_jobs` record
- One active job per scraper type at a time (checked before starting)
- Progress tracked: `items_found`, `items_saved`, `pages_completed`
- WebSocket broadcasts job updates to frontend
- Frontend `useScraperJobPolling` polls `GET /scraper/jobs/:id` every 2s

**Activity Logging:**
- Each scraper job also creates an `activity_logs` record
- Pauseable/resumable via checkpoint JSON
- Visible in ActivityDashboard and Dashboard running-tasks

**Auto-Update:**
- Per-scraper `auto_update_configs` in DB
- Scheduler checks intervals and triggers scraper runs automatically
- `POST /auto-update/configs/:type/check-now` for manual trigger

### 7.2 Simpcity Forum Scraper (lives in ScraperPage.vue)

**Authentication:**
- Session cookie stored in app_settings
- Cookie parser accepts both raw header format and DevTools table format (tab-separated)
- Cookie verified against Simpcity API after save
- `GET /scraper/session` / `POST /scraper/session`
- Login helper opens Simpcity in Firefox

**Thread Scraping:**
- `POST /scraper/threads/scrape { url }` — Scrapes all posts in thread
- Extracts: title, author, post content (HTML + plain text), all links
- Link detection: gofile.io, pixeldrain.com, bunkr.si, and other providers
- Provider status: checks link alive/dead/expired
- Watch list: `POST /scraper/threads/:id/watch`

**Forum Category Listing:**
- `POST /scraper/forum/scrape-listing { url, maxPages }` — Gets thread list from forum page
- `POST /scraper/forum/scrape-all` — Scrapes all categories

**Performer Linking:**
- `POST /scraper/performers/auto-link` — ML-based performer name extraction from thread titles
- Creates links in `performer_scraped_threads` with confidence scores

**Link Verification:**
- `POST /scraper/threads/:id/verify-links` — Checks all links in thread
- WebSocket broadcasts `link_verification_complete` when done
- Frontend clears verifying state and refreshes link stats
- `GET /scraper/threads/:id/link-stats` — Returns health breakdown

**JDownloader Integration:**
- `POST /jdownloader/threads/:id/send` — Sends all active links from thread to JD
- Button in thread detail view + bulk action in thread list

### 7.3 PornHub Scraper (PornHubTab.vue)

- Keyword-based search
- Loads up to 10,000 videos from DB, filters client-side via `filteredVideos` computed
- Debounced search (300ms)
- Play routing: `router.push('/watch/scraped/${video.id}')` for most videos; `openInFirefox` for xvideos/xhamster URLs
- JDownloader available check via `jdownloaderAPI.checkAvailability()`
- `jdSending` ref tracks per-video sending state
- Auto-Update Panel with `endpoint-type="keywords"`

### 7.4 LuxureTV Scraper (LuxureTVTab.vue)

- Category-based scraping with category URL input
- `fetchEmbeds(videoId)` — fetches embed IDs needed to resolve stream URLs
- `startScrapeFromFirstCategory()` — convenience: picks `categories[0].url` and starts
- Auto-Update Panel with `endpoint-type="categories"`

### 7.5 Sexuria Scraper (SexuriaTab.vue — Most Complex)

**Scraping Modes:**
1. **Category Mode** — URL input + max pages
2. **Advanced Search** — Categories (multi-select), title, description, pornstars, production, tags, resolutions (multi-select), file hosts (multi-select), max pages

**Item Display:**
- Grid with thumbnail, title, uploader, file host tags, link health badge
- Hover card (Teleport to body): appears after 300ms, fetches item details from server
  - Shows screenshots grid (click to open screenshot modal)
  - Shows tags, pornstar list
  - Cache: `sexuriaDetailsCache` prevents re-fetching
- Screenshot Modal (Teleport to body): full-size image viewer with prev/next navigation

**Link Management:**
- "Extract Links" button → `POST /sexuria/items/:id/extract-links`
- "View Links" button → opens Links Modal
- Links Modal: table with provider badge, status badge, URL, copy button
- "Verify All Links" in modal → `POST /sexuria/items/:id/verify-links`
- Link health badge: color-coded based on working/dead/unknown ratio

**Filtering:**
- Category dropdown (built from loaded items)
- Host filter dropdown (from stats.host_counts)
- Text search (debounced 300ms)
- Tag filter (multi-select dropdown, populated from cached item details)
- Pornstar filter (multi-select dropdown, populated from cached item details)
- Sort: newest, oldest, title A-Z, title Z-A

### 7.6 Rule34Video Scraper (Rule34VideoTab.vue)

- Tag-based scraping (comma-separated tags)
- Sort options fetched from `GET /rule34video/sort-options`
- UI note: "Straight 3D content" filter info alert
- Auto-Update Panel (no endpoint-type needed)

### 7.7 Motherless Scraper (MotherlessTab.vue)

- Search term based (paste search URL)
- `searchTermFilter` dropdown: populated from `stats.search_term_counts`
  - Shows each term + count, sorted by count descending
- Stats show total per search term
- Auto-Update Panel

### 7.8 LeakVids Scraper (LeakVidsTab.vue)

- Most filter options of all scrapers:
  - `baseURL` (default: `https://leakvids.com/latest-updates/`)
  - Min/max duration (minutes)
  - Min rating (1-10)
  - Min view count
  - Sort order
  - Max pages
- Auto-Update Panel

### 7.9 HQPorner Scraper (HQPornerTab.vue)

- Multi-filter scraping
- Shows performer matching from video metadata (`getVideoPerformers`)
- Shows matched tags (`getVideoMatchedTags`)
- Metadata stored as JSON in `scraped_videos.metadata`
- Auto-Update Panel

### 7.10 Custom Scraper Tabs

**Creation:**
1. User provides site URL, name, color, icon
2. Optionally provides two paginated URLs → `POST /scraper/detect-pagination` detects pattern
3. User configures CSS selectors (container, title, url, thumbnail)
4. Tab saved to `scraper_tabs` table

**Execution:**
- Same job system as built-in scrapers
- `scraper_tabs_service.go` handles the generic selector-based scraping
- Jobs tracked in `scraper_jobs` with `tab_id` reference

**Frontend:**
- Dynamic tab buttons rendered from `scraperTabs` array
- Tab content: same layout as built-in tabs with generic video grid
- `collapsedSections[tab-${tab.id}]` for sidebar state

---

## 8. KEY CROSS-CUTTING FEATURES

### 8.1 Activity Monitoring

**All background operations create activity records:**
- Video scan → `task_type: "scan"`
- Scraper run → `task_type: "scrape"`
- Video conversion → `task_type: "convert"`
- Link verification → `task_type: "verify"`
- Thumbnail generation → `task_type: "generate"`
- AI analysis → `task_type: "ai_analysis"`

**Pauseable Tasks:**
- On pause: `checkpoint` JSON saved to activity record (captures current state)
- On resume: checkpoint loaded, task continues from saved position
- `activity_logs.is_paused` flag controls pause state

**WebSocket Activity Updates:**
- Every progress tick broadcasts `activity_update` event
- Frontend ActivityDashboard and NavBar badge update in real-time

### 8.2 Link Verification System

**Process:**
1. Fetch link URL with HEAD request
2. Check response status (200=active, 404/410=dead, redirect=check target)
3. Provider-specific checks (gofile API, pixeldrain API, etc.)
4. Update `scraped_download_links.status` and `verified_at`
5. Calculate per-provider health scores
6. Broadcast `link_verification_complete` via WebSocket

**Health Classes:**
- `bg-success` — >75% working
- `bg-warning` — 25-50% dead
- `bg-danger` — >50% dead
- `bg-info` — mostly unknown
- `bg-secondary` — no data

### 8.3 Performer Thumbnail System

**Sources:**
1. Performer folder preview videos (cycling frames extracted via FFmpeg)
2. Auto-generated from video library associations
3. Manual assignment

**Thumbnail Path:** `api/data/assets/performers/{performer_id}/thumbnail.jpg`
**Preview Path:** `api/data/assets/performers/{performer_id}/preview.mp4`

### 8.4 Video Preview System

**Storyboard Generation:**
- FFmpeg extracts N frames evenly spaced throughout video
- Frames stored as `api/data/assets/videos/{video_id}/frame_*.jpg`
- Frontend VideoCard cycles through frames on hover

**Preview URL:** Served via `/api/v1/libraries/:id/stream?path=...`

### 8.5 Video Conversion

- Convert any video to MP4 (H.264/AAC)
- Original video preserved, conversion creates new video record
- `videos.converted_from` and `videos.converted_to` link them
- Progress tracked in activity_logs
- UI shows conversion status badge

### 8.6 Auto-Update System

**Configuration per scraper type** (stored in `app_settings` as JSON):
```json
{
  "enabled": true,
  "interval_minutes": 60,
  "max_pages_to_check": 5,
  "last_run_at": "2026-02-23T12:00:00Z",
  "next_run_at": "2026-02-23T13:00:00Z"
}
```

**Scheduler** checks intervals every minute and triggers appropriate scraper.
**Dashboard** shows each scraper's auto-update status with "Check Now" button.
**AutoUpdatePanel** component embedded in every scraper tab's sidebar.

### 8.7 Download Tracking

**Per-link download tracking** separate from JDownloader state:
- `scraped_download_links.download_status` (pending|downloading|downloaded|failed)
- Bulk mark operations
- Stats per thread, per performer, globally
- Used to avoid re-downloading already-completed links

---

## 9. API RESPONSE FORMATS

### Standard Success
```json
{ "data": { ... }, "message": "Success" }
```

### Paginated List
```json
{
  "data": [ ... ],
  "total": 1000,
  "page": 1,
  "limit": 50,
  "pages": 20
}
```

### Error
```json
{ "error": "Error message", "details": "..." }
```

### WebSocket Message
```json
{
  "type": "activity_update|status_update|console_log|companion_event|link_verification_complete|feed_update|notification",
  "data": { ... }
}
```

### Activity Update (WebSocket)
```json
{
  "type": "activity_update",
  "data": {
    "id": 42,
    "task_type": "scrape",
    "status": "running",
    "progress": 45,
    "message": "Processing page 3 of 10..."
  }
}
```

### Companion Event (WebSocket)
```json
{
  "type": "companion_event",
  "data": {
    "type": "file_added",
    "severity": "info",
    "title": "New Files Detected",
    "message": "3 new files found in library",
    "data": { "files": [...] }
  }
}
```

---

## 10. CONFIGURATION & DEPLOYMENT

### 10.1 Environment Variables

```env
DATABASE_PATH=api/data/database.db
SERVER_HOST=localhost
SERVER_PORT=8080
SERVER_MODE=debug   # or release
LOG_LEVEL=info
```

### 10.2 In-DB Configuration (app_settings table)

All user-facing settings stored as JSON values keyed by string:
- `theme` — dark/light
- `default_view_mode` — grid/list
- `items_per_page` — pagination default
- `jdownloader_*` — connection settings (now in `jdownloader_config` table)
- `ai_*` — AI settings (now in `ai_config` table)
- `auto_update_{type}` — per-scraper auto-update config

### 10.3 Build Commands

```bash
# Frontend
cd "c:\Repos\Video Storage AI"
npm run build
# Output: dist/ served by backend's SPA handler

# Backend
cd "c:\Repos\Video Storage AI\api"
go build ./...
# Run: ./server (or the built binary)
```

### 10.4 Docker Deployment

```dockerfile
# Dockerfile in project root
# /output volume for persistent data (database, assets, backups)
# Port 8080 exposed
```

**Volume mount:** `/output` → contains `database.db`, `assets/`, `backups/`

### 10.5 Runtime Directory Structure

```
api/data/
├── database.db           # Main SQLite database
├── database.db-wal       # WAL journal
├── database.db-shm       # WAL shared memory
├── assets/
│   ├── performers/
│   │   └── {id}/
│   │       ├── thumbnail.jpg
│   │       └── preview.mp4
│   └── videos/
│       └── {id}/
│           ├── thumbnail.jpg
│           └── frame_*.jpg
└── backups/
    └── database_2026-02-23.db
```

---

## 11. PERFORMANCE & EXTENSION POINTS

### 11.1 Performance Optimizations

**Frontend:**
- Lazy-loaded routes (webpack code splitting)
- `shallowRef` for large arrays (10,000+ scraped videos)
- Client-side filtering with debounced search (300ms)
- Virtual scrolling (`VirtualScroller.vue`) for 1000+ item lists
- Image lazy loading (`LazyImage.vue`)
- Request caching composables (`useRequestCache`)
- WebSocket for real-time updates (no polling for activity/status)

**Backend:**
- WAL mode for SQLite concurrent reads
- Connection pool (25 max, 5 idle)
- Response gzip compression (Gin middleware)
- Static file caching (1-year cache headers)
- Query result caching in services where appropriate
- Parallel library scanning (`scan-all-parallel`)
- Pagination defaults to prevent large result sets

**Database:**
- Foreign key indices auto-created
- Text search uses LIKE with indexed columns
- Pagination enforced on all list endpoints

### 11.2 Extension Points

**Adding a New Scraper:**
1. Create `api/internal/services/{site}_scraper_service.go`
2. Create `api/internal/api/{site}_handlers.go` with handler functions
3. Register routes in `api/internal/api/router.go`
4. Add API methods to `scrapersAPI.{site}` in `src/services/api.js`
5. Create `src/components/scraper/tabs/{Site}Tab.vue` component
6. Import and add tab to `ScraperPage.vue`

**Adding a New AI Tool:**
1. Define tool in `ai_companion_service.go` tools array
2. Implement handler function
3. Register in the tool dispatch switch statement
4. Tool automatically available to LLM via tool calling

**Adding a New Page:**
1. Create `src/views/NewPage.vue`
2. Add route to `src/router/index.js`
3. Add nav link to `NavBar.vue`
4. Create handler file in `api/internal/api/`
5. Register routes in `router.go`

**Adding Database Tables:**
1. Add `CREATE TABLE IF NOT EXISTS` to `database.go`'s `runMigrations()`
2. Create model in `api/internal/models/`
3. Implement service in `api/internal/services/`
4. Add handlers and routes

### 11.3 Known Architectural Decisions

- **SQLite chosen over PostgreSQL** for simplicity and self-hosted deployment
- **JDownloader deprecated HTTP API** used instead of newer REST API due to availability
- **LM Studio** as local LLM provider (privacy-first, no cloud dependency)
- **Scraper tabs extracted to standalone Vue components** to reduce ScraperPage.vue from 7,732 → 2,951 lines
- **`shallowRef` for scraped video arrays** — Vue's deep reactivity is too slow for 10,000+ items
- **`useVideoSelection` composable** — Avoids undefined toggle functions from bare `ref(new Set())`
- **`doJDAction` helper** — Centralizes JD HTTP API request logic with correct headers
- **WebSocket singleton** — Single connection shared across all components via import

---

*End of Specification*
*App version: as of 2026-02-23 | ScraperPage refactor complete (7 tab components extracted)*
