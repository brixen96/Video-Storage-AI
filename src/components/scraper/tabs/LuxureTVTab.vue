<!-- eslint-disable no-undef -->
<template>
	<div class="tab-content">
		<!-- Top Bar -->
		<div class="scraper-tab-topbar">
			<button class="scraper-filter-toggle-btn" :class="{ 'is-active': !filterCollapsed }" @click="filterCollapsed = !filterCollapsed" title="Toggle filters">
				<font-awesome-icon :icon="['fas', 'sliders-h']" />
				Filters
				<span v-if="activeFilterCount > 0" class="scraper-filter-toggle-badge">{{ activeFilterCount }}</span>
			</button>
			<input v-model="videoSearch" class="scraper-tab-search" type="text" placeholder="Search LuxureTV videos..." />
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
				v-model:minDuration="minDuration"
				v-model:maxDuration="maxDuration"
				v-model:onlyDownloaded="onlyDownloaded"
				v-model:minRating="minRating"
				v-model:minViews="minViews"
				:showMinRating="true"
				:showMinViews="true"
				:performers="matchedPerformers"
				v-model:performerId="performerFilter"
				:tags="matchedTags"
				v-model:tagId="tagFilter"
				@clear="onClearFilters"
			>
				<!-- Category custom filter -->
				<div v-if="categories.length > 0" class="sfp-section">
					<div class="sfp-section-label">
						<font-awesome-icon :icon="['fas', 'folder']" />
						Category
					</div>
					<select v-model="categoryFilter" class="form-select form-select-sm sfp-select">
						<option value="">All Categories</option>
						<option v-for="cat in categories" :key="cat.url" :value="cat.name">{{ cat.name }}</option>
					</select>
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
				<ScraperJobProgress :job="activeJob" label="LuxureTV" />
				<LoadingState v-if="loading" show-text loading-text="Loading LuxureTV videos..." />
				<div v-else-if="videos.length > 0">
					<div class="d-flex justify-content-between align-items-center mb-2">
						<h6 class="mb-0">
							<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
							LuxureTV Videos ({{ totalCount.toLocaleString() }})
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
				<EmptyState v-else-if="!scraping" :icon="['fas', 'paw']" icon-size="4x" title="No LuxureTV Videos Yet" message="Use the sidebar controls to start scraping" />
			</div>

			<!-- Right: Scraper Sidebar -->
			<div class="scraper-sidebar" :class="{ collapsed: sidebarCollapsed }">
				<button class="collapse-toggle-btn mb-3 w-100" @click="sidebarCollapsed = true">
					<font-awesome-icon :icon="['fas', 'times']" class="me-2" />Hide Panel
				</button>
				<AutoUpdatePanel scraper-type="luxuretv" endpoint-type="categories" />
				<div class="card mb-3">
					<div class="card-header">
						<h6 class="mb-0"><font-awesome-icon :icon="['fas', 'chart-bar']" class="me-2" />Statistics</h6>
					</div>
					<div class="card-body p-2">
						<div class="sidebar-stats-grid">
							<StatCard :value="totalCount" label="Videos" :icon="['fas', 'video']" icon-class="videos" />
							<StatCard :value="activeJob ? 1 : 0" label="Active" :icon="['fas', 'spinner']" icon-class="active" />
							<StatCard :value="activeJob?.videosFound || 0" label="Found" :icon="['fas', 'search']" icon-class="found" />
							<StatCard :value="activeJob?.videosSaved || 0" label="Saved" :icon="['fas', 'save']" icon-class="links" />
						</div>
					</div>
				</div>
				<div class="card mb-3">
					<div class="card-header">
						<h6 class="mb-0"><font-awesome-icon :icon="['fas', 'paw']" class="me-2 text-info" />Scraper</h6>
					</div>
					<div class="card-body">
						<button class="btn btn-success btn-sm w-100 mb-2" :disabled="scraping" @click="startScrapeFromFirstCategory">
							<font-awesome-icon :icon="['fas', scraping ? 'spinner' : 'play']" :spin="scraping" class="me-1" />
							{{ scraping ? 'Scraping...' : 'Start Scrape' }}
						</button>
						<button class="btn btn-outline-info btn-sm w-100" @click="fetchEmbeds" :disabled="fetchingEmbeds">
							<font-awesome-icon :icon="['fas', fetchingEmbeds ? 'spinner' : 'code']" :spin="fetchingEmbeds" class="me-1" />
							{{ fetchingEmbeds ? 'Fetching...' : 'Fetch Embeds' }}
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
const { sendingToJD, sendToJD } = useSendToJD(toast, 'luxuretv')
const sendSelectedToJD = () => sendToJD(videos.value, selectedVideos.value)
const { pollJob } = useScraperJobPolling()
const { selectionMode, selectedVideos, toggleSelectionMode, toggleVideoSelection, selectAll } = useVideoSelection()
const { openInFirefox } = useOpenURL()
const { sidebarCollapsed, activeTimeouts } = useScraperSidebar()

const videosPerPage = SCRAPER_PAGE_SIZE

// ── State ──────────────────────────────────────────────────────────────────────
const filterCollapsed  = ref(false)
const categoryURL      = ref('')
const maxPages         = ref(0)
const scraping         = ref(false)
const loading          = ref(false)
const videos           = shallowRef([])
const categories       = ref([])
const activeJob        = ref(null)
const videoSearch      = ref('')
const videoSort        = ref('newest')
const videoPage        = ref(1)
const fetchingEmbeds   = ref(false)
const performerFilter  = ref(0)
const tagFilter        = ref(0)
const minDuration      = ref(0)
const maxDuration      = ref(0)
const categoryFilter   = ref('')
const onlyDownloaded   = ref(false)
const matchedPerformers = ref([])
const matchedTags       = ref([])
const totalCount = ref(0)
const totalPages = ref(0)

const debouncedSearch   = useDebouncedRef(videoSearch)
const selectAllVideos   = () => selectAll(videos.value)
const includeTagFilters = ref([])
const excludeTagFilters = ref([])
const newIncludeTag     = ref('')
const newExcludeTag     = ref('')
const minRating         = ref(0)
const minViews          = ref(0)

const activeFilterCount = computed(() => {
	let n = 0
	if (videoSort.value !== 'newest')       n++
	if (minDuration.value > 0)             n++
	if (maxDuration.value > 0)             n++
	if (onlyDownloaded.value)               n++
	if (performerFilter.value !== 0)        n++
	if (tagFilter.value !== 0)              n++
	if (categoryFilter.value !== '')        n++
	if (includeTagFilters.value.length > 0) n++
	if (excludeTagFilters.value.length > 0) n++
	if (parseInt(minRating.value) > 0)      n++
	if (parseInt(minViews.value) > 0)       n++
	return n
})

// ── Watchers ───────────────────────────────────────────────────────────────────
watch(debouncedSearch,   () => { videoPage.value = 1; loadVideos() })
watch(videoSort,         () => { videoPage.value = 1; loadVideos() })
watch(performerFilter,   () => { videoPage.value = 1; loadVideos() })
watch(tagFilter,         () => { videoPage.value = 1; loadVideos() })
watch(minDuration,      () => { videoPage.value = 1; loadVideos() })
watch(maxDuration,      () => { videoPage.value = 1; loadVideos() })
watch(categoryFilter,    () => { videoPage.value = 1; loadVideos() })
watch(onlyDownloaded,    () => { videoPage.value = 1; loadVideos() })
watch(includeTagFilters, () => { videoPage.value = 1; loadVideos() }, { deep: true })
watch(excludeTagFilters, () => { videoPage.value = 1; loadVideos() }, { deep: true })
watch(minRating,         () => { videoPage.value = 1; loadVideos() })
watch(minViews,          () => { videoPage.value = 1; loadVideos() })

// ── Helpers ────────────────────────────────────────────────────────────────────
const SORT_MAP = {
	newest:          { sort: 'discovered_at', order: 'desc' },
	oldest:          { sort: 'discovered_at', order: 'asc'  },
	'views-desc':    { sort: 'view_count',    order: 'desc' },
	'views-asc':     { sort: 'view_count',    order: 'asc'  },
	'likes-desc':    { sort: 'like_count',    order: 'desc' },
	'likes-asc':     { sort: 'like_count',    order: 'asc'  },
	'rating-desc':   { sort: 'rating',        order: 'desc' },
	'rating-asc':    { sort: 'rating',        order: 'asc'  },
	'duration-desc': { sort: 'duration',      order: 'desc' },
	'duration-asc':  { sort: 'duration',      order: 'asc'  },
	'title-asc':     { sort: 'title',         order: 'asc'  },
	'title-desc':    { sort: 'title',         order: 'desc' },
}

// ── API ────────────────────────────────────────────────────────────────────────
const loadCategories = async () => {
	try {
		const data = await scrapersAPI.luxuretv.getCategories()
		if (data.success && data.data) categories.value = data.data
	} catch (e) { console.error('Error loading LuxureTV categories:', e) }
}

const loadPerformers = async () => {
	try {
		const data = await scrapersAPI.luxuretv.getPerformers()
		if (data.success && data.data) matchedPerformers.value = data.data || []
	} catch (e) { console.error('Error loading LuxureTV performers:', e) }
}

const loadTags = async () => {
	try {
		const data = await scrapersAPI.luxuretv.getTags()
		if (data.success && data.data) matchedTags.value = data.data || []
	} catch (e) { console.error('Error loading LuxureTV tags:', e) }
}

const loadStats = async () => {
	try { await scrapersAPI.luxuretv.getStats() } catch (e) { console.error('Error loading LuxureTV stats:', e) }
}

const loadVideos = async () => {
	loading.value = true
	try {
		const { sort: sortField, order: sortOrder } = SORT_MAP[videoSort.value] || SORT_MAP.newest
		const params = {
			page:   videoPage.value,
			limit:  videosPerPage,
			search: debouncedSearch.value,
			sort:   sortField,
			order:  sortOrder,
		}
		if (performerFilter.value > 0)                              params.performer_id    = performerFilter.value
		if (tagFilter.value > 0)                                    params.tag_id          = tagFilter.value
		if (parseInt(minDuration.value) > 0) params.min_duration = minDuration.value * 60
		if (parseInt(maxDuration.value) > 0) params.max_duration = maxDuration.value * 60
		if (categoryFilter.value)                                   params.category        = categoryFilter.value
		if (onlyDownloaded.value)                                   params.downloaded_only = true
		if (includeTagFilters.value.length > 0)                     params.include_tags    = includeTagFilters.value.join(',')
		if (excludeTagFilters.value.length > 0)                     params.exclude_tags    = excludeTagFilters.value.join(',')
		if (parseInt(minRating.value) > 0)                          params.min_rating      = minRating.value
		if (parseInt(minViews.value) > 0)                           params.min_views       = minViews.value
		const data = await scrapersAPI.luxuretv.getVideos(params)
		if (data.success && data.data) {
			videos.value     = data.data.videos     || []
			totalCount.value = data.data.total      || 0
			totalPages.value = data.data.totalPages || 0
		}
	} catch (e) { console.error('Error loading LuxureTV videos:', e) }
	finally { loading.value = false }
}

const clearVideos = async () => {
	if (!confirm(`Are you sure you want to delete all ${totalCount.value} LuxureTV videos? This cannot be undone.`)) return
	try {
		const data = await scrapersAPI.luxuretv.clearVideos()
		if (data.success) { toast.success(`Deleted ${data.data.deleted} LuxureTV videos`); videos.value = []; await loadStats() }
		else toast.error(data.message || 'Failed to delete videos')
	} catch (e) { console.error('Error deleting LuxureTV videos:', e); toast.error('Failed to delete videos') }
}

const deleteSelectedVideos = async () => {
	if (selectedVideos.value.size === 0) return
	if (!confirm(`Are you sure you want to delete ${selectedVideos.value.size} selected videos?`)) return
	try {
		const data = await scrapersAPI.bulkDelete(Array.from(selectedVideos.value))
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} videos`)
			selectedVideos.value = new Set(); selectionMode.value = false
			await loadVideos(); await loadStats()
		} else toast.error(data.message || 'Failed to delete videos')
	} catch (e) { console.error('Error deleting selected videos:', e); toast.error('Failed to delete selected videos') }
}

const onClearFilters = () => {
	categoryFilter.value    = ''
	includeTagFilters.value = []
	excludeTagFilters.value = []
	minRating.value         = 0
	minViews.value          = 0
	minDuration.value       = 0
	maxDuration.value       = 0
	videoPage.value         = 1
	loadVideos()
}

const addIncludeTagFromCard = (tag) => {
	if (!includeTagFilters.value.includes(tag)) {
		includeTagFilters.value = [...includeTagFilters.value, tag]
	}
	filterCollapsed.value = false
}

const startScrapeFromFirstCategory = async () => {
	if (categories.value.length > 0) categoryURL.value = categories.value[0].url
	await startScrape()
}

const startScrape = async () => {
	if (!categoryURL.value) { toast.warning('Please select a category'); return }
	scraping.value = true
	try {
		const data = await scrapersAPI.luxuretv.scrape({ categoryUrl: categoryURL.value, maxPages: maxPages.value || 0 })
		if (data.success) {
			toast.success(`Started LuxureTV scrape`)
			activeJob.value = { id: data.data.job_id, status: 'running', pagesCompleted: 0, videosFound: 0, videosSaved: 0, errorsCount: 0 }
			pollJob(data.data.job_id, {
				activeJob, isScraping: scraping, activeTimeouts,
				onSuccess: async (job) => { toast.success(`LuxureTV scrape complete! Found ${job.videosFound} videos`); await loadVideos(); await loadStats() },
				onFail: () => toast.error('LuxureTV scrape failed'),
			})
		} else { toast.error(data.message || 'Failed to start LuxureTV scrape'); scraping.value = false }
	} catch (e) { console.error('Error starting LuxureTV scrape:', e); toast.error('Failed to start LuxureTV scrape'); scraping.value = false }
}

const fetchEmbeds = async () => {
	fetchingEmbeds.value = true
	try {
		const data = await scrapersAPI.luxuretv.fetchEmbeds()
		if (data.success) {
			toast.success(`Started fetching embed IDs for ${data.data.videos_to_process} videos.`)
			setTimeout(() => { fetchingEmbeds.value = false; loadVideos() }, 5000)
		} else { toast.error(data.message || 'Failed to start fetching embed IDs'); fetchingEmbeds.value = false }
	} catch (e) { console.error('Error fetching embed IDs:', e); toast.error('Failed to fetch embed IDs'); fetchingEmbeds.value = false }
}

useScraperAutoRefresh('luxuretv', () => { videoPage.value = 1; loadVideos() })
onMounted(() => { loadCategories(); loadPerformers(); loadTags(); loadStats(); loadVideos() })
</script>
