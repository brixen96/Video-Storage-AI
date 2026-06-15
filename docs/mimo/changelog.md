# Assistant Changelog

> Track all changes made by the AI assistant

## 2026-06-16 — Filter Fix + Version Badge + Build Auto-Increment

### Root Cause
- All filtering was broken because the **Windows services were running old backend binaries** without the new filter code
- Frontend params were correct (verified via console logs) but old backend didn't parse `min_duration`/`max_duration`

### Fix
- Stopped services, rebuilt Go binaries with new code, restarted services
- Verified API test: unfiltered = 38,154 videos, filtered (50-90min) = 579 videos

### Frontend — NavBar.vue
- Added version badge next to "Video Storage AI" title (fetches from `/health` endpoint)
- Fixed health URL to use `window.VSAI_API_URL` (production proxy issue)

### build.bat
- Added auto-increment patch version on each build (e.g. 1.0.0 -> 1.0.1 -> 1.0.2)
- Writes incremented version back to `VERSION` file before building

### Debug cleanup
- Removed debug `console.log` statements from ScraperFilterPanel and PornHubTab

## 2026-06-08

-   **Created** `docs/mimo/` folder structure and documentation files
-   **Set up** project reference notes
-   **Modernized** generic scraper tab filtering in `ScraperPage.vue`
    -   Replaced inline Bootstrap filter bar with shared `ScraperFilterPanel` component
    -   Replaced custom inline pagination with shared `ScraperPaginationControls`
    -   Added tag include/exclude filtering (chip input, Enter to add, ✕ to remove)
    -   Added min rating filter (50%, 70%, 80%, 90%)
    -   Added `ScraperJobProgress` component for active scrape job display
    -   Reordered layout: FilterPanel (left) → Main (center) → Sidebar (right)
    -   Moved scraper controls to right sidebar (matching fixed tabs)
    -   Added new state: `videoMinRating`, `videoIncludeTags`, `videoExcludeTags`
    -   Updated `loadVideosForTab` to send `min_rating`, `include_tags`, `exclude_tags` params
    -   Added tag helper functions: `addIncludeTag`, `removeIncludeTag`, `addExcludeTag`, `removeExcludeTag`    - **Fix**: Added `@update:*` event handlers on `ScraperFilterPanel` to trigger reloads on filter changes
    - **Fix**: Debounced `onTabFilterChange` (50ms) to batch `clearAll()` events into single reload
    - **Fix**: `resetVideoFilters` cancels pending debounce timeouts before direct reload
    - **Fix**: Added `rating-desc`/`rating-asc` to `tabSortMap` (missing sort mapping)

## 2026-06-15

### Backend (Go)
- **Fixed** `like_count` was missing from `validSortFields` in `video_scraper_service_query.go` — caused silent fallback to `discovered_at`
- **Added** `file_size` to `validSortFields`
- **Added** `MinLikes` to `ScrapedVideosQuery` struct
- **Added** `DateFrom`, `DateTo`, `MinDuration`, `MaxDuration`, `MinFileSize`, `MaxFileSize`, `HasThumbnail`, `OnlyWithAuthor`, `OnlyWithRating` to `ScrapedVideosQuery` struct
- **Added** corresponding WHERE clauses in `GetScrapedVideosPaginated` for all new filter fields
- **Updated** generic `/scraper/videos` handler to parse new query params: `date_from`, `date_to`, `min_duration`, `max_duration`, `min_file_size`, `max_file_size`, `has_thumbnail`, `only_with_author`, `only_with_rating`, `min_likes`

### Frontend — ScraperFilterPanel.vue
- **Expanded** sort dropdown with `<optgroup>` sections: Date, Popularity, Quality, Length, Title, Size
- **Added** sort options: `likes-asc`, `title-desc`, `filesize-desc`, `filesize-asc`
- **Added** Precise Duration Range filter (min/max seconds inputs)
- **Added** Date Range filter (date pickers)
- **Added** File Size Range filter (preset dropdown: Tiny/Small/Medium/Large/Huge)
- **Added** Min Likes filter (100+, 1K+, 10K+, 100K+)
- **Added** "Has Rating" toggle (onlyWithRating)
- **Added** "Has Author" toggle (onlyWithAuthor)
- **Added** new props: `minLikes`, `showMinLikes`, `showFileSizeSort`, `showFileSizeFilter`, `fileSizeRange`, `showDateRange`, `dateFrom`, `dateTo`, `showDurationRange`, `minDuration`, `maxDuration`, `showOnlyWithRating`, `onlyWithRating`, `showOnlyWithAuthor`, `onlyWithAuthor`
- **Added** corresponding emits and clearAll support for all new fields
- **Added** CSS for `.sfp-range-row` and `.sfp-range-sep` in `scraper_page.css`

### Frontend — ScraperPage.vue (Generic Tabs)
- **Added** new state variables: `videoMinLikes`, `videoFileSizeRange`, `videoDateFrom`, `videoDateTo`, `videoMinDuration`, `videoMaxDuration`, `videoOnlyWithRating`, `videoOnlyWithAuthor`
- **Updated** `ScraperFilterPanel` bindings with all new props and `@update:*` handlers
- **Updated** `loadVideosForTab` to send: `min_likes`, `date_from`, `date_to`, `min_duration`, `max_duration`, `min_file_size`, `max_file_size`, `only_with_rating`, `only_with_author`
- **Updated** `tabSortMap` with `likes-asc`, `filesize-desc`, `filesize-asc`
- **Updated** `resetVideoFilters` to reset all new filter state
- **Updated** `activeTabFilterCount` to count all new filters

### Frontend — All 6 Fixed Tabs
- **Updated** `SORT_MAP` in PornHubTab, LuxureTVTab, Rule34VideoTab, HQPornerTab, MotherlessTab, LeakVidsTab
- **Added** sort options: `likes-desc`, `likes-asc`, `title-desc` to all SORT_MAPs
## 2026-06-15 — Duration Slider Redesign

### Frontend — ScraperFilterPanel.vue
-   **Replaced** old duration preset buttons (All / <5m / 5-20m / >20m) with dual range sliders (min/max in minutes)
-   **Removed** `duration` prop and `DURATIONS` constant (no longer needed)
-   **Removed** `showDurationRange` prop (sliders always visible now)
-   **Added** `onMinDurationInput` / `onMaxDurationInput` handlers with auto-clamp (min ≤ max)
-   **Added** `resetDuration` helper function
-   **Added** `formatMinutes` display helper (e.g. 65m → "1h 5m")
-   **Added** `durationLabel` computed property for active filter summary
-   **Slider range**: 0–120 minutes, step of 5 minutes
-   **Added** "Clear" button when duration filters are active

### Frontend — All 7 Tab Components
-   **Replaced** `v-model:duration` with `v-model:minDuration` + `v-model:maxDuration` in all tabs
-   **Removed** unused `durationFilter` / `hqPornerDurationFilter` refs from all tabs
-   **Updated** watchers to watch `minDuration` and `maxDuration` instead of `durationFilter`
-   **Updated** `loadVideos` to send `min_duration` / `max_duration` (converted to seconds × 60)
-   **Updated** `activeFilterCount` to count min/maxDuration
-   **Updated** `onClearFilters` to reset minDuration/maxDuration to 0

### Frontend — CSS (scraper_page.css)
-   **Added** `.sfp-slider-group`, `.sfp-slider-row`, `.sfp-slider-label`, `.sfp-slider-val` layout styles
-   **Added** `.sfp-slider` range input styling (dark theme, purple gradient thumb)
-   **Added** `.sfp-duration-value` label and `.sfp-slider-reset` clear button

### Backend (Go) — All 7 Handlers
-   **Added** `min_duration` / `max_duration` parsing (ParseFloat) to: Pornhub, LuxureTV, Rule34Video, HQPorner, Motherless, LeakVids, and generic `/scraper/videos` handlers
-   **Added** `MinDuration` / `MaxDuration` fields to `HQPornerVideoQuery` struct
-   **Added** WHERE clauses for `min_duration` / `max_duration` in HQPorner `GetVideosPaginated`