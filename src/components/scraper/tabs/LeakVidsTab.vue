<!-- eslint-disable no-undef -->
<template>
	<div class="tab-content">
		<!-- Top Bar -->
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
			<input v-model="videoSearch" class="scraper-tab-search" type="text" placeholder="Search LeakVids videos..." />
			<div class="scraper-tab-topbar-actions">
				<span v-if="totalCount > 0" class="scraper-video-count">{{ totalCount.toLocaleString() }} videos</span>
				<button class="btn btn-sm" :class="selectionMode ? 'btn-primary' : 'btn-outline-secondary'" @click="toggleSelectionMode" title="Select mode">
					<font-awesome-icon :icon="['fas', selectionMode ? 'times' : 'check-square']" />
				</button>
				<button class="btn btn-sm btn-outline-info" @click="loadVideos" :disabled="loading" title="Refresh">
					<font-awesome-icon :icon="['fas', loading ? 'spinner' : 'sync-alt']" :spin="loading" />
				</button>
			</div>
		</div>

		<div class="scraper-layout" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
			<!-- Left: Filter Panel -->
			<ScraperFilterPanel
				v-model:modelCollapsed="filterCollapsed"
				v-model:sort="videoSort"
				v-model:duration="durationFilter"
				v-model:onlyDownloaded="onlyDownloaded"
				v-model:minRating="minRating"
				v-model:minViews="minViews"
				:showMinRating="true"
				:showMinViews="true"
				@clear="onClearFilters"
			>
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
						<span v-for="tag in includeTagFilters" :key="tag" class="sfp-chip active" @click="includeTagFilters.splice(includeTagFilters.indexOf(tag), 1)">
							{{ tag }} <font-awesome-icon :icon="['fas', 'times']" class="ms-1" />
						</span>
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
						<span v-for="tag in excludeTagFilters" :key="tag" class="sfp-chip exclude-chip" @click="excludeTagFilters.splice(excludeTagFilters.indexOf(tag), 1)">
							{{ tag }} <font-awesome-icon :icon="['fas', 'times']" class="ms-1" />
						</span>
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
				<ScraperJobProgress :job="activeJob" label="LeakVids" />
				<LoadingState v-if="loading" show-text loading-text="Loading LeakVids videos..." />
				<div v-else-if="totalCount > 0 || videos.length > 0">
					<div class="d-flex justify-content-between align-items-center mb-2">
						<h6 class="mb-0">
							<font-awesome-icon :icon="['fas', 'film']" class="me-2" />
							LeakVids Videos ({{ videos.length }} / {{ totalCount }})
						</h6>
						<button class="btn btn-sm btn-outline-danger" @click="clearVideos" title="Clear all">
							<font-awesome-icon :icon="['fas', 'trash-alt']" />
						</button>
					</div>
					<div class="video-grid">
						<ScrapedVideoCard
							v-for="video in videos"
							:key="video.id"
							v-memo="[video.id, video.downloaded_at, selectedVideos.has(video.id), selectionMode]"
							:video="video"
							:selectionMode="selectionMode"
							:isSelected="selectedVideos.has(video.id)"
							@toggle-selection="toggleVideoSelection"
							@filter-tag="addIncludeTagFromCard"
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
						:currentPage="videoPage"
						:totalPages="totalPages"
						:totalItems="totalCount"
						:pageSize="videosPerPage"
						@update:currentPage="videoPage = $event; loadVideos()"
					/>
				</div>
				<EmptyState v-else-if="!scraping" :icon="['fas', 'film']" icon-size="4x" title="No LeakVids Videos Yet" message="Enter the LeakVids URL in the sidebar to start scraping" />
			</div>

			<!-- Right: Scraper Sidebar -->
			<div class="scraper-sidebar" :class="{ collapsed: sidebarCollapsed }">
				<button class="collapse-toggle-btn mb-3 w-100" @click="sidebarCollapsed = true">
					<font-awesome-icon :icon="['fas', 'times']" class="me-2" />
					Hide Panel
				</button>
				<AutoUpdatePanel scraper-type="leakvids" />
				<!-- Statistics -->
				<div class="card mb-3">
					<div class="card-header">
						<h6 class="mb-0"><font-awesome-icon :icon="['fas', 'chart-bar']" class="me-2" />Statistics</h6>
					</div>
					<div class="card-body p-2">
						<div class="sidebar-stats-grid">
							<StatCard :value="stats.total_videos || 0" label="Videos" :icon="['fas', 'video']" icon-class="videos" />
							<StatCard :value="activeJob?.videosFound || 0" label="Found" :icon="['fas', 'search']" icon-class="found" />
							<StatCard :value="activeJob?.videosSaved || 0" label="Saved" :icon="['fas', 'save']" icon-class="active" />
							<StatCard :value="activeJob?.errorsCount || 0" label="Errors" :icon="['fas', 'exclamation']" icon-class="keywords" />
						</div>
					</div>
				</div>
				<!-- Scraper Controls -->
				<div class="card mb-3">
					<div class="card-header">
						<h6 class="mb-0"><font-awesome-icon :icon="['fas', 'film']" class="me-2 text-danger" />URL Scraper</h6>
					</div>
					<div class="card-body">
						<div class="mb-2">
							<label class="form-label small">Base URL</label>
							<input v-model="baseURL" type="text" class="form-control form-control-sm" placeholder="https://leakvids.com/latest-updates/" @keyup.enter="startScrape" />
							<small class="text-muted">Paste LeakVids page URL</small>
						</div>
						<div class="mb-2">
							<label class="form-label small">Max Pages</label>
							<input v-model.number="maxPages" type="number" class="form-control form-control-sm" min="1" max="9999" placeholder="0 = unlimited" />
						</div>
						<button class="btn btn-danger btn-sm w-100" @click="startScrape" :disabled="!baseURL || scraping">
							<font-awesome-icon :icon="['fas', scraping ? 'spinner' : 'search']" :spin="scraping" class="me-2" />
							{{ scraping ? 'Scraping...' : 'Start Scrape' }}
						</button>
					</div>
				</div>
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
import AutoUpdatePanel           from '@/components/scraper/AutoUpdatePanel.vue'
import ScrapedVideoCard          from '@/components/scraper/ScrapedVideoCard.vue'
import ScraperJobProgress        from '@/components/scraper/ScraperJobProgress.vue'
import ScraperBulkActions        from '@/components/scraper/ScraperBulkActions.vue'
import ScraperPaginationControls from '@/components/scraper/ScraperPaginationControls.vue'
import ScraperFilterPanel        from '@/components/scraper/ScraperFilterPanel.vue'
import { useSendToJD } from '@/composables/useSendToJD'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast
const { sendingToJD, sendToJD } = useSendToJD(toast, 'leakvids')
const sendSelectedToJD = () => sendToJD(videos.value, selectedVideos.value)
const { pollJob } = useScraperJobPolling()
const { selectionMode, selectedVideos, toggleSelectionMode, toggleVideoSelection, selectAll } = useVideoSelection()
const { openInFirefox } = useOpenURL()
const { sidebarCollapsed, activeTimeouts } = useScraperSidebar()

const videosPerPage = SCRAPER_PAGE_SIZE

// ── State ─────────────────────────────────────────────────────────────────────
const filterCollapsed   = ref(false)
const scraping          = ref(false)
const loading           = ref(false)
const videos            = shallowRef([])
const stats             = ref({ total_videos: 0 })
const activeJob         = ref(null)
const baseURL           = ref('https://leakvids.com/latest-updates/')
const maxPages          = ref(0)
const videoSearch       = ref('')
const videoSort         = ref('newest')
const videoPage         = ref(1)
const totalCount        = ref(0)
const totalPages        = ref(0)
const durationFilter    = ref('all')
const onlyDownloaded    = ref(false)
const minRating         = ref(0)
const minViews          = ref(0)
const includeTagFilters = ref([])
const excludeTagFilters = ref([])
const newIncludeTag     = ref('')
const newExcludeTag     = ref('')

const debouncedSearch = useDebouncedRef(videoSearch)
const selectAllVideos = () => selectAll(videos.value)

// ── Computed ──────────────────────────────────────────────────────────────────
const activeFilterCount = computed(() => {
	let n = 0
	if (videoSort.value !== 'newest')       n++
	if (durationFilter.value !== 'all')     n++
	if (onlyDownloaded.value)               n++
	if (parseInt(minRating.value) > 0)      n++
	if (parseInt(minViews.value) > 0)       n++
	if (includeTagFilters.value.length > 0) n++
	if (excludeTagFilters.value.length > 0) n++
	return n
})

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

// ── Watchers ──────────────────────────────────────────────────────────────────
watch(debouncedSearch,   () => { videoPage.value = 1; loadVideos() })
watch(videoSort,         () => { videoPage.value = 1; loadVideos() })
watch(durationFilter,    () => { videoPage.value = 1; loadVideos() })
watch(onlyDownloaded,    () => { videoPage.value = 1; loadVideos() })
watch(minRating,         () => { videoPage.value = 1; loadVideos() })
watch(minViews,          () => { videoPage.value = 1; loadVideos() })
watch(includeTagFilters, () => { videoPage.value = 1; loadVideos() }, { deep: true })
watch(excludeTagFilters, () => { videoPage.value = 1; loadVideos() }, { deep: true })

// ── API ───────────────────────────────────────────────────────────────────────
const loadVideos = async () => {
	loading.value = true
	try {
		const { sort: sortField, order: sortOrder } = SORT_MAP[videoSort.value] || SORT_MAP.newest
		const params = {
			page:  videoPage.value,
			limit: videosPerPage,
			search: debouncedSearch.value,
			sort:  sortField,
			order: sortOrder,
		}
		if (durationFilter.value !== 'all')     params.duration_filter = durationFilter.value
		if (onlyDownloaded.value)               params.downloaded_only = true
		if (parseInt(minRating.value) > 0)      params.min_rating      = minRating.value
		if (parseInt(minViews.value) > 0)       params.min_views       = minViews.value
		if (includeTagFilters.value.length > 0) params.include_tags    = includeTagFilters.value.join(',')
		if (excludeTagFilters.value.length > 0) params.exclude_tags    = excludeTagFilters.value.join(',')

		const data = await scrapersAPI.leakvids.getVideos(params)
		if (data.success && data.data) {
			videos.value     = data.data.videos     || []
			totalCount.value = data.data.total      || 0
			totalPages.value = data.data.totalPages || 0
		}
	} catch (e) { console.error('Error loading LeakVids videos:', e) }
	finally { loading.value = false }
}

const loadStats = async () => {
	try {
		const data = await scrapersAPI.leakvids.getStats()
		if (data.success && data.data) stats.value = data.data
	} catch (e) { console.error('Error loading LeakVids stats:', e) }
}

const onClearFilters = () => {
	minRating.value         = 0
	minViews.value          = 0
	includeTagFilters.value = []
	excludeTagFilters.value = []
	videoPage.value         = 1
	loadVideos()
}

const addIncludeTagFromCard = (tag) => {
	if (!includeTagFilters.value.includes(tag)) {
		includeTagFilters.value = [...includeTagFilters.value, tag]
	}
	filterCollapsed.value = false
}

const startScrape = async () => {
	if (!baseURL.value) { toast.warning('Please enter a base URL'); return }
	scraping.value = true
	try {
		const data = await scrapersAPI.leakvids.scrape({ baseUrl: baseURL.value, maxPages: maxPages.value || 0 })
		if (data.success) {
			toast.success('Started LeakVids scrape')
			activeJob.value = { id: data.data.job_id, status: 'running', pagesCompleted: 0, videosFound: 0, videosSaved: 0, errorsCount: 0 }
			pollJob(data.data.job_id, {
				activeJob, isScraping: scraping, activeTimeouts,
				onSuccess: async (job) => {
					toast.success(`LeakVids scrape complete! Found ${job.videosFound} videos`)
					await loadVideos(); await loadStats()
				},
				onFail: () => toast.error('LeakVids scrape failed'),
			})
		} else {
			toast.error(data.message || 'Failed to start LeakVids scrape')
			scraping.value = false
		}
	} catch (e) { console.error('Error starting LeakVids scrape:', e); toast.error('Failed to start LeakVids scrape'); scraping.value = false }
}

const clearVideos = async () => {
	if (!confirm(`Are you sure you want to delete all ${totalCount.value} LeakVids videos? This cannot be undone.`)) return
	try {
		const data = await scrapersAPI.leakvids.clearVideos()
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} LeakVids videos`)
			videos.value = []; totalCount.value = 0
			await loadStats()
		} else {
			toast.error(data.message || 'Failed to delete videos')
		}
	} catch (e) { console.error('Error deleting LeakVids videos:', e); toast.error('Failed to delete videos') }
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

// ── Lifecycle ─────────────────────────────────────────────────────────────────
useScraperAutoRefresh('leakvids', () => { videoPage.value = 1; loadVideos() })
onMounted(() => { loadStats(); loadVideos() })
</script>
