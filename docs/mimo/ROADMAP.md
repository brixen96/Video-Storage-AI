# Video Storage AI — Upgrade Roadmap to "11/10"

> Goal: Transform from impressive solo project to production-quality codebase
> Priority: Code quality, architecture, developer experience (NO security work needed)

## Phase 1: Foundation Cleanup (Quick Wins)
> Impact: High, Effort: Low — These are easy wins that immediately improve code quality

- [ ] Delete dead files (BrowserPage_BACKUP.vue, *.backup, nul)
- [ ] Fix hardcoded localhost:8080 in videosAPI.getThumbnail
- [ ] Add CSS custom properties for the color palette (replace 20+ hardcoded hex values)
- [ ] Extract Vuex deduplication pattern into a reusable helper
- [ ] Remove debug console.log statements from production code

## Phase 2: Component Architecture
> Impact: High, Effort: Medium — Break up oversized components into manageable pieces

- [ ] Break BrowserPage.vue (1830 lines) into ~5 sub-components
- [ ] Break VideosPage.vue (1262 lines) into filter, grid, bulk actions
- [ ] Break ScraperPage.vue (1053 lines) — extract custom tab rendering
- [ ] Extract JDownloaderPage.vue sidebar and main content
- [ ] Add onActivated/onDeactivated hooks to cached views

## Phase 3: State Management
> Impact: Medium, Effort: Medium — Clean up the state architecture

- [ ] Migrate Vuex → Pinia (Vue 3 native, simpler API)
- [ ] Standardize on <script setup> — migrate remaining 13 views
- [ ] Extract shared state patterns into composables
- [ ] Add provide/inject for global toast (instead of fragile $refs pattern)

## Phase 4: CSS Architecture
> Impact: Medium, Effort: Medium — Create a proper design system

- [ ] Create CSS custom properties for all colors, spacing, typography
- [ ] Reduce FontAwesome imports to only icons actually used
- [ ] Extract Bootstrap overrides into a proper theme file
- [ ] Add consistent spacing/sizing system

## Phase 5: Backend Architecture
> Impact: High, Effort: High — Improve Go code quality

- [ ] Replace global state with dependency injection
- [ ] Add HTTP client timeouts to all scrapers
- [ ] Add User-Agent headers to scraper requests
- [ ] Fix goroutine lifecycle (WaitGroup, context cancellation)
- [ ] Split SetupRouter into domain-specific registration functions
- [ ] Use database config values instead of hardcoded limits
- [ ] Add structured logging (replace log.Printf)

## Phase 6: Performance & Polish
> Impact: Medium, Effort: Medium — Optimize for large datasets

- [ ] Add virtual scrolling to VideosPage for 38K+ videos
- [ ] Lazy-load scraper tab components
- [ ] Reduce Bootstrap JS bundle (import only needed plugins)
- [ ] Add image lazy loading with IntersectionObserver
- [ ] Fix keep-alive data freshness

## Phase 7: Developer Experience
> Impact: Low, Effort: Low — Make development smoother

- [ ] Add hot-reload for Go backend
- [ ] Create proper dev setup documentation
- [ ] Add Makefile or task runner for common operations
- [ ] Clean up outdated documentation files

---

## Execution Order
1. Phase 1 first (quick wins build momentum)
2. Phase 2 (biggest visual impact — smaller files)
3. Phase 4 (CSS cleanup improves everything)
4. Phase 5 (backend hardening)
5. Phase 3 (state management)
6. Phase 6 (performance)
7. Phase 7 (DX)

Each phase will be committed separately with clear, descriptive commit messages.
