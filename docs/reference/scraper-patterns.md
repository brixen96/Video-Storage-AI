# Scraper Patterns — Reference

## Generic Tabs — Filter Reload Pattern
- `ScraperFilterPanel` emits `update:sort`, `update:duration`, etc. via v-model
- `clearAll()` fires ALL `update:*` events + `clear` event
- Solution: debounce `onTabFilterChange` with 50ms, cancel in `resetVideoFilters`
- `resetVideoFilters` must cancel pending `tabFilterTimeouts[tabId]` and `tabSearchTimeouts[tabId]` before calling `loadVideosForTab`

## TabSortMap
- `tabSortMap` in ScraperPage.vue maps UI sort values to backend `sort_by`/`sort_order`
- Must include: newest, oldest, views-desc/asc, likes-desc/asc, rating-desc/asc, duration-desc/asc, title-asc/desc, filesize-desc/asc
- Built-in tab SORT_MAPs in PornHub/LuxureTV/Rule34/HQPorner/Motherless/LeakVids must match

## Backend Sort Validation
- `validSortFields` in `video_scraper_service_query.go` must include all sort_by values
- `like_count` was missing once → silent fallback to `discovered_at` (symptom: sort appears ignored)

## Backend Filter Params (added 2026-06-15)
- `date_from`, `date_to` — discovered_at range
- `min_duration`, `max_duration` — precise duration in seconds
- `min_file_size`, `max_file_size` — file size range in bytes
- `min_likes` — minimum like count
- `has_thumbnail` — filter by thumbnail existence
- `only_with_author`, `only_with_rating` — quick boolean filters
