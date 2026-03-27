<!-- eslint-disable no-undef -->
<template>
	<div class="tab-content">
		<!-- Top Bar: search + actions -->
		<div class="scraper-tab-topbar">
			<button
				class="scraper-filter-toggle-btn"
				:class="{ 'is-active': !filterCollapsed }"
				@click="filterCollapsed = !filterCollapsed"
				title="Toggle filters"
			>
				<font-awesome-icon :icon="['fas', 'sliders-h']" />
				Filters
				<span v-if="activeFilterCount > 0" class="scraper-filter-toggle-badge">{{ activeFilterCount }}</span>
			</button>
			<input v-model="hqPornerSearch" class="scraper-tab-search" type="text" placeholder="Search HQPorner videos..." />
			<div class="scraper-tab-topbar-actions">
				<span v-if="hqPornerTotalCount > 0" class="scraper-video-count">{{ hqPornerTotalCount.toLocaleString() }} videos</span>
				<button class="btn btn-sm" :class="selectionMode ? 'btn-primary' : 'btn-outline-secondary'" @click="toggleSelectionMode" title="Select mode">
					<font-awesome-icon :icon="['fas', selectionMode ? 'times' : 'check-square']" />
				</button>
				<button class="btn btn-sm btn-outline-info" @click="loadVideos" :disabled="hqPornerLoading" title="Refresh">
					<font-awesome-icon :icon="['fas', hqPornerLoading ? 'spinner' : 'sync-alt']" :spin="hqPornerLoading" />
				</button>
			</div>
		</div>

		<div class="scraper-layout" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
			<!-- Left: Filter Panel -->
			<ScraperFilterPanel
				v-model:modelCollapsed="filterCollapsed"
				v-model:sort="hqPornerSort"
				v-model:duration="hqPornerDurationFilter"
				v-model:onlyDownloaded="hqPornerOnlyDownloaded"
				v-model:minRating="minRating"
				v-model:minViews="minViews"
				:showMinRating="true"
				:showMinViews="true"
				:performers="hqPornerMatchedPerformers"
				v-model:performerId="hqPornerPerformerFilter"
				:tags="hqPornerMatchedTags"
				v-model:tagId="hqPornerTagFilter"
				@clear="onClearFilters"
			>
				<!-- Performer / Author filter -->
				<div class="sfp-section">
					<div class="sfp-section-label">
						<font-awesome-icon :icon="['fas', 'user-circle']" />
						Performer / Author
					</div>
					<input
						v-model="hqPornerAuthorFilter"
						type="text"
						class="form-control form-control-sm sfp-input"
						placeholder="Filter by performer..."
					/>
				</div>

				<!-- Include Tags -->
				<div class="sfp-section">
					<div class="sfp-section-label">
						<font-awesome-icon :icon="['fas', 'plus-circle']" />
						Include Tags
					</div>
					<input
						v-model="newIncludeTag"
						type="text"
						class="form-control form-control-sm sfp-input"
						placeholder="Add tag, press Enter..."
						@keydown.enter.prevent="if (newIncludeTag.trim() && !includeTagFilters.includes(newIncludeTag.trim())) { includeTagFilters.push(newIncludeTag.trim()); newIncludeTag = '' }"
					/>
					<div v-if="includeTagFilters.length > 0" class="sfp-chips mt-1">
						<span
							v-for="tag in includeTagFilters"
							:key="tag"
							class="sfp-chip active"
							@click="includeTagFilters.splice(includeTagFilters.indexOf(tag), 1)"
						>{{ tag }} <font-awesome-icon :icon="['fas', 'times']" class="ms-1" /></span>
					</div>
				</div>

				<!-- Exclude Tags -->
				<div class="sfp-section">
					<div class="sfp-section-label">
						<font-awesome-icon :icon="['fas', 'minus-circle']" />
						Exclude Tags
					</div>
					<input
						v-model="newExcludeTag"
						type="text"
						class="form-control form-control-sm sfp-input"
						placeholder="Exclude tag, press Enter..."
						@keydown.enter.prevent="if (newExcludeTag.trim() && !excludeTagFilters.includes(newExcludeTag.trim())) { excludeTagFilters.push(newExcludeTag.trim()); newExcludeTag = '' }"
					/>
					<div v-if="excludeTagFilters.length > 0" class="sfp-chips mt-1">
						<span
							v-for="tag in excludeTagFilters"
							:key="tag"
							class="sfp-chip exclude-chip"
							@click="excludeTagFilters.splice(excludeTagFilters.indexOf(tag), 1)"
						>{{ tag }} <font-awesome-icon :icon="['fas', 'times']" class="ms-1" /></span>
					</div>
				</div>
			</ScraperFilterPanel>

			<!-- Center: Main Content -->
			<div class="scraper-main">
				<ScraperBulkActions
					:selectionMode="selectionMode"
					:selectedCount="selectedVideos.size"
					:sendingToJD="sendingToJD"
					@select-all="selectAllVideos"
					@delete-selected="deleteSelectedVideos"
					@send-to-jd="sendSelectedToJD"
				/>
				<ScraperJobProgress :job="hqPornerActiveJob" label="HQPorner" />
				<LoadingState v-if="hqPornerLoading" show-text loading-text="Loading HQPorner videos..." />
				<div v-else-if="hqPornerTotalCount > 0 || hqPornerVideos.length > 0">
					<div class="d-flex justify-content-between align-items-center mb-2">
						<h6 class="mb-0">
							<font-awesome-icon :icon="['fas', 'play-circle']" class="me-2" />
							HQPorner Videos ({{ hqPornerVideos.length }} / {{ hqPornerTotalCount }})
						</h6>
						<button class="btn btn-sm btn-outline-danger" @click="clearVideos" title="Clear all">
							<font-awesome-icon :icon="['fas', 'trash-alt']" />
						</button>
					</div>
					<div class="video-grid">
						<ScrapedVideoCard
							v-for="video in hqPornerVideos"
							:key="video.id"
							v-memo="[video.id, video.downloaded_at, selectedVideos.has(video.id), selectionMode]"
							:video="video"
							:selectionMode="selectionMode"
							:isSelected="selectedVideos.has(video.id)"
							@toggle-selection="toggleVideoSelection"
							@filter-tag="addIncludeTagFromCard"
							@filter-author="(a) => { hqPornerAuthorFilter = a; filterCollapsed = false }"
						>
							<template #actions="{ video }">
								<button @click.stop="openInFirefox(video.url)" class="btn btn-sm btn-primary">
									<font-awesome-icon :icon="['fab', 'firefox-browser']" class="me-1" />
									Open
								</button>
							</template>
						</ScrapedVideoCard>
					</div>
					<ScraperPaginationControls
						:currentPage="hqPornerVideoPage"
						:totalPages="hqPornerTotalPages"
						:totalItems="hqPornerTotalCount"
						:pageSize="videosPerPage"
						@update:currentPage="hqPornerVideoPage = $event; loadVideos()"
					/>
				</div>
				<EmptyState v-else-if="!hqPornerScraping" :icon="['fas', 'play-circle']" icon-size="4x" title="No HQPorner Videos Yet" message="Click Start Scrape to begin scraping hqporner.com/hdporn/" />
			</div>

			<!-- Right: Scraper Sidebar -->
			<div class="scraper-sidebar" :class="{ collapsed: sidebarCollapsed }">
				<button class="collapse-toggle-btn mb-3 w-100" @click="sidebarCollapsed = true">
					<font-awesome-icon :icon="['fas', 'times']" class="me-2" />
					Hide Panel
				</button>
				<div class="card mb-3">
					<div class="card-header">
						<h6 class="mb-0"><font-awesome-icon :icon="['fas', 'chart-bar']" class="me-2" />Statistics</h6>
					</div>
					<div class="card-body p-2">
						<div class="sidebar-stats-grid">
							<StatCard :value="hqPornerStats.total_videos || 0" label="Videos" :icon="['fas', 'video']" icon-class="videos" />
							<StatCard :value="hqPornerMatchedPerformers.length" label="Performers" :icon="['fas', 'user']" icon-class="active" />
							<StatCard :value="hqPornerActiveJob?.videosFound || 0" label="Found" :icon="['fas', 'search']" icon-class="found" />
							<StatCard :value="hqPornerActiveJob?.videosSaved || 0" label="Saved" :icon="['fas', 'save']" icon-class="keywords" />
						</div>
					</div>
				</div>
				<div class="card mb-3">
					<div class="card-header">
						<h6 class="mb-0"><font-awesome-icon :icon="['fas', 'play-circle']" class="me-2 text-danger" />URL Scraper</h6>
					</div>
					<div class="card-body">
						<div class="mb-2">
							<small class="text-muted">Scrapes <strong>hqporner.com/hdporn/</strong> starting from page 1</small>
						</div>
						<div class="mb-2">
							<label class="form-label small">Max Pages</label>
							<input v-model.number="hqPornerMaxPages" type="number" class="form-control form-control-sm" min="1" max="9999" placeholder="0 = unlimited" />
						</div>
						<button class="btn btn-danger btn-sm w-100" @click="startScrape" :disabled="hqPornerScraping">
							<font-awesome-icon :icon="['fas', hqPornerScraping ? 'spinner' : 'search']" :spin="hqPornerScraping" class="me-2" />
							{{ hqPornerScraping ? 'Scraping...' : 'Start Scrape' }}
						</button>
					</div>
				</div>
				<AutoUpdatePanel scraper-type="hqporner" />
			</div>
		</div>

		<button v-if="sidebarCollapsed" class="sidebar-toggle" @click="sidebarCollapsed = false" title="Show Controls Panel">
			<font-awesome-icon :icon="['fas', 'cog']" />
		</button>
		<button class="filter-panel-toggle" :class="{ show: filterCollapsed }" @click="filterCollapsed = false" title="Show Filters">
			<font-awesome-icon :icon="['fas', 'sliders-h']" />
		</button>
	</div>
</template>

<script setup>
import { ref, shallowRef, computed, onMounted, watch, getCurrentInstance } from 'vue'
import { scrapersAPI } from '@/services/api'
import { useScraperJobPolling } from '@/composables/useScraperJobPolling'
import { useVideoSelection }    from '@/composables/useVideoSelection'
import { useOpenURL }           from '@/composables/useOpenURL'
import { useScraperSidebar }    from '@/composables/useScraperSidebar'
import { useScraperAutoRefresh } from '@/composables/useScraperAutoRefresh'
import { useDebouncedRef }      from '@/composables/useDebouncedRef'
import { SCRAPER_PAGE_SIZE }    from '@/utils/scraperConstants'
import { LoadingState, EmptyState, StatCard } from '@/components/shared'
import ScrapedVideoCard          from '@/components/scraper/ScrapedVideoCard.vue'
import ScraperJobProgress        from '@/components/scraper/ScraperJobProgress.vue'
import ScraperBulkActions        from '@/components/scraper/ScraperBulkActions.vue'
import ScraperPaginationControls from '@/components/scraper/ScraperPaginationControls.vue'
import ScraperFilterPanel        from '@/components/scraper/ScraperFilterPanel.vue'
import AutoUpdatePanel           from '@/components/scraper/AutoUpdatePanel.vue'
import { useSendToJD } from '@/composables/useSendToJD'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast
const { sendingToJD, sendToJD } = useSendToJD(toast, 'hqporner')
const sendSelectedToJD = () => sendToJD(hqPornerVideos.value, selectedVideos.value)

const { pollJob } = useScraperJobPolling()
const { selectionMode, selectedVideos, toggleSelectionMode, toggleVideoSelection, selectAll } = useVideoSelection()
const { openInFirefox } = useOpenURL()
const { sidebarCollapsed, activeTimeouts } = useScraperSidebar()

const videosPerPage = SCRAPER_PAGE_SIZE

// ── State ─────────────────────────────────────────────────────────────────────
const filterCollapsed         = ref(false)
const hqPornerScraping        = ref(false)
const hqPornerLoading         = ref(false)
const hqPornerVideos          = shallowRef([])
const hqPornerStats           = ref({ total_videos: 0 })
const hqPornerActiveJob       = ref(null)
const hqPornerMaxPages        = ref(0)
const hqPornerSearch          = ref('')
const hqPornerSort            = ref('newest')
const hqPornerVideoPage       = ref(1)
const hqPornerDurationFilter  = ref('all')
const hqPornerOnlyDownloaded  = ref(false)
const hqPornerPerformerFilter = ref(0)
const hqPornerTagFilter       = ref(0)
const hqPornerTotalCount      = ref(0)
const hqPornerTotalPages      = ref(0)
const hqPornerMatchedPerformers = ref([])
const hqPornerMatchedTags       = ref([])
const hqPornerAuthorFilter    = ref('')
const includeTagFilters       = ref([])
const excludeTagFilters       = ref([])
const newIncludeTag           = ref('')
const newExcludeTag           = ref('')
const minRating               = ref(0)
const minViews                = ref(0)

const debouncedSearch = useDebouncedRef(hqPornerSearch)
const debouncedAuthor = useDebouncedRef(hqPornerAuthorFilter)
const selectAllVideos = () => selectAll(hqPornerVideos.value)

const activeFilterCount = computed(() => {
	let n = 0
	if (hqPornerSort.value !== 'newest')        n++
	if (hqPornerDurationFilter.value !== 'all') n++
	if (hqPornerOnlyDownloaded.value)           n++
	if (hqPornerPerformerFilter.value !== 0)    n++
	if (hqPornerTagFilter.value !== 0)          n++
	if (hqPornerAuthorFilter.value)             n++
	if (includeTagFilters.value.length > 0)     n++
	if (excludeTagFilters.value.length > 0)     n++
	if (parseInt(minRating.value) > 0)          n++
	if (parseInt(minViews.value) > 0)           n++
	return n
})

// ── Watchers ──────────────────────────────────────────────────────────────────
watch(debouncedSearch,           () => { hqPornerVideoPage.value = 1; loadVideos() })
watch(hqPornerSort,              () => { hqPornerVideoPage.value = 1; loadVideos() })
watch(hqPornerDurationFilter,    () => { hqPornerVideoPage.value = 1; loadVideos() })
watch(hqPornerOnlyDownloaded,    () => { hqPornerVideoPage.value = 1; loadVideos() })
watch(hqPornerPerformerFilter,   () => { hqPornerVideoPage.value = 1; loadVideos() })
watch(hqPornerTagFilter,         () => { hqPornerVideoPage.value = 1; loadVideos() })
watch(debouncedAuthor,           () => { hqPornerVideoPage.value = 1; loadVideos() })
watch(includeTagFilters,         () => { hqPornerVideoPage.value = 1; loadVideos() }, { deep: true })
watch(excludeTagFilters,         () => { hqPornerVideoPage.value = 1; loadVideos() }, { deep: true })
watch(minRating,                 () => { hqPornerVideoPage.value = 1; loadVideos() })
watch(minViews,                  () => { hqPornerVideoPage.value = 1; loadVideos() })

// ── Helpers ───────────────────────────────────────────────────────────────────
const SORT_MAP = {
	newest:          { sort: 'discovered_at', order: 'desc' },
	oldest:          { sort: 'discovered_at', order: 'asc'  },
	'views-desc':    { sort: 'view_count',    order: 'desc' },
	'views-asc':     { sort: 'view_count',    order: 'asc'  },
	'rating-desc':   { sort: 'rating',        order: 'desc' },
	'rating-asc':    { sort: 'rating',        order: 'asc'  },
	'duration-desc': { sort: 'duration',      order: 'desc' },
	'duration-asc':  { sort: 'duration',      order: 'asc'  },
	'title-asc':     { sort: 'title',         order: 'asc'  },
}

// ── API ───────────────────────────────────────────────────────────────────────
const loadVideos = async () => {
	hqPornerLoading.value = true
	try {
		const { sort: sortField, order: sortOrder } = SORT_MAP[hqPornerSort.value] || SORT_MAP.newest
		const params = {
			page:  hqPornerVideoPage.value,
			limit: videosPerPage,
			search: debouncedSearch.value,
			sort:  sortField,
			order: sortOrder,
		}
		if (hqPornerPerformerFilter.value > 0)      params.performer_id    = hqPornerPerformerFilter.value
		if (hqPornerTagFilter.value > 0)            params.tag_id          = hqPornerTagFilter.value
		if (hqPornerDurationFilter.value !== 'all') params.duration_filter = hqPornerDurationFilter.value
		if (hqPornerOnlyDownloaded.value)           params.downloaded_only = true
		if (hqPornerAuthorFilter.value)             params.author          = hqPornerAuthorFilter.value
		if (includeTagFilters.value.length > 0)     params.include_tags    = includeTagFilters.value.join(',')
		if (excludeTagFilters.value.length > 0)     params.exclude_tags    = excludeTagFilters.value.join(',')
		if (parseInt(minRating.value) > 0)          params.min_rating      = minRating.value
		if (parseInt(minViews.value) > 0)           params.min_views       = minViews.value

		const data = await scrapersAPI.hqporner.getVideos(params)
		if (data.success && data.data) {
			hqPornerVideos.value     = data.data.videos     || []
			hqPornerTotalCount.value = data.data.total      || 0
			hqPornerTotalPages.value = data.data.totalPages || 0
		}
	} catch (e) { console.error('Error loading HQPorner videos:', e) }
	finally { hqPornerLoading.value = false }
}

const loadPerformers = async () => {
	try {
		const data = await scrapersAPI.hqporner.getPerformers()
		if (data.success && data.data) hqPornerMatchedPerformers.value = data.data || []
	} catch (e) { console.error('Error loading HQPorner performers:', e) }
}

const loadTags = async () => {
	try {
		const data = await scrapersAPI.hqporner.getTags()
		if (data.success && data.data) hqPornerMatchedTags.value = data.data || []
	} catch (e) { console.error('Error loading HQPorner tags:', e) }
}

const loadStats = async () => {
	try {
		const data = await scrapersAPI.hqporner.getStats()
		if (data.success && data.data) hqPornerStats.value = data.data
	} catch (e) { console.error('Error loading HQPorner stats:', e) }
}

const onClearFilters = () => {
	hqPornerAuthorFilter.value = ''
	includeTagFilters.value    = []
	excludeTagFilters.value    = []
	minRating.value            = 0
	minViews.value             = 0
	hqPornerVideoPage.value    = 1
	loadVideos()
}

const addIncludeTagFromCard = (tag) => {
	if (!includeTagFilters.value.includes(tag)) {
		includeTagFilters.value = [...includeTagFilters.value, tag]
	}
	filterCollapsed.value = false
}

const startScrape = async () => {
	hqPornerScraping.value = true
	try {
		const data = await scrapersAPI.hqporner.scrape({ maxPages: hqPornerMaxPages.value || 0 })
		if (data.success) {
			toast.success('Started HQPorner scrape')
			hqPornerActiveJob.value = { id: data.data.job_id, status: 'running', pagesCompleted: 0, videosFound: 0, videosSaved: 0, errorsCount: 0 }
			pollJob(data.data.job_id, {
				activeJob: hqPornerActiveJob, isScraping: hqPornerScraping, activeTimeouts,
				onSuccess: async (job) => {
					toast.success(`HQPorner scrape complete! Found ${job.videosFound} videos`)
					await loadVideos(); await loadStats(); await loadPerformers(); await loadTags()
				},
				onFail: () => toast.error('HQPorner scrape failed'),
			})
		} else { toast.error(data.message || 'Failed to start HQPorner scrape'); hqPornerScraping.value = false }
	} catch (e) { console.error('Error starting HQPorner scrape:', e); toast.error('Failed to start HQPorner scrape'); hqPornerScraping.value = false }
}

const clearVideos = async () => {
	if (!confirm(`Are you sure you want to delete all ${hqPornerVideos.value.length} HQPorner videos? This cannot be undone.`)) return
	try {
		const data = await scrapersAPI.hqporner.clearVideos()
		if (data.success) { toast.success(`Deleted ${data.data.deleted} HQPorner videos`); hqPornerVideos.value = []; await loadStats() }
		else toast.error(data.message || 'Failed to delete videos')
	} catch (e) { console.error('Error deleting HQPorner videos:', e); toast.error('Failed to delete videos') }
}

const deleteSelectedVideos = async () => {
	const ids = Array.from(selectedVideos.value)
	if (ids.length === 0) return
	if (!confirm(`Delete ${ids.length} selected videos?`)) return
	try {
		const data = await scrapersAPI.bulkDelete(ids)
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} videos`)
			selectedVideos.value = new Set(); selectionMode.value = false
			await loadVideos(); await loadStats()
		}
	} catch (e) { console.error('Error deleting videos:', e); toast.error('Failed to delete videos') }
}

useScraperAutoRefresh('hqporner', () => { hqPornerVideoPage.value = 1; loadVideos() })
onMounted(() => { Promise.all([loadVideos(), loadStats(), loadPerformers(), loadTags()]) })
</script>
