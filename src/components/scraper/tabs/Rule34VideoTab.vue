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
			<input v-model="videoSearch" class="scraper-tab-search" type="text" placeholder="Search Rule34Video videos..." />
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
				@clear="onClearFilters"
			>
				<!-- Artist filter -->
				<div class="sfp-section">
					<div class="sfp-section-label">
						<font-awesome-icon :icon="['fas', 'user']" />
						Artist / Model
					</div>
					<input
						v-model="authorFilter"
						type="text"
						class="form-control form-control-sm"
						placeholder="Search by artist..."
					/>
				</div>

				<!-- Category filter -->
				<div class="sfp-section">
					<div class="sfp-section-label">
						<font-awesome-icon :icon="['fas', 'layer-group']" />
						Category
						<button class="sfp-refresh-btn" @click="loadLocalCategories" title="Refresh categories">
							<font-awesome-icon :icon="['fas', 'sync-alt']" />
						</button>
					</div>
					<select v-model="categoryFilter" class="form-select form-select-sm">
						<option value="">All Categories</option>
						<option v-for="cat in localCategories" :key="cat" :value="cat">{{ cat }}</option>
					</select>
				</div>

				<!-- Include Tags -->
				<div class="sfp-section">
					<div class="sfp-section-label">
						<font-awesome-icon :icon="['fas', 'tag']" class="text-success" />
						Include Tags
					</div>
					<div class="input-group input-group-sm mb-1">
						<input
							v-model="includeTagInput"
							type="text"
							class="form-control form-control-sm"
							placeholder="tag, press Enter"
							@keydown.enter.prevent="addIncludeTag"
						/>
						<button class="btn btn-outline-success" @click="addIncludeTag" title="Add">+</button>
					</div>
					<div v-if="includeTagFilters.length > 0" class="r34-tag-cloud">
						<span v-for="(t, i) in includeTagFilters" :key="i" class="r34-tag-chip r34-tag-include">
							{{ t }}<button @click="removeIncludeTag(i)" class="r34-tag-remove">×</button>
						</span>
					</div>
				</div>

				<!-- Exclude Tags -->
				<div class="sfp-section">
					<div class="sfp-section-label">
						<font-awesome-icon :icon="['fas', 'tag']" class="text-danger" />
						Exclude Tags
					</div>
					<div class="input-group input-group-sm mb-1">
						<input
							v-model="excludeTagInput"
							type="text"
							class="form-control form-control-sm"
							placeholder="tag, press Enter"
							@keydown.enter.prevent="addExcludeTag"
						/>
						<button class="btn btn-outline-danger" @click="addExcludeTag" title="Add">−</button>
					</div>
					<div v-if="excludeTagFilters.length > 0" class="r34-tag-cloud">
						<span v-for="(t, i) in excludeTagFilters" :key="i" class="r34-tag-chip r34-tag-exclude">
							−{{ t }}<button @click="removeExcludeTag(i)" class="r34-tag-remove">×</button>
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
				<ScraperJobProgress :job="activeJob" label="Rule34Video" />
				<LoadingState v-if="loading" show-text loading-text="Loading Rule34Video videos..." />
				<div v-else-if="totalCount > 0 || videos.length > 0">
					<div class="d-flex justify-content-between align-items-center mb-2">
						<h6 class="mb-0">
							<font-awesome-icon :icon="['fas', 'cube']" class="me-2" />
							Rule34Videos ({{ videos.length }} / {{ totalCount }})
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
							@filter-category="setCategoryFromCard"
							@filter-author="(a) => { authorFilter = a; filterCollapsed = false }"
						>
							<template #actions="{ video }">
								<button @click.stop="openInFirefox(video.url)" class="btn btn-sm btn-primary">
									<font-awesome-icon :icon="['fab', 'firefox-browser']" class="me-1" />
									Open
								</button>
								<button @click.stop="sendToJDownloader(video)" class="btn btn-sm btn-success" title="Send to JDownloader">
									<font-awesome-icon :icon="['fas', 'download']" class="me-1" />
									JD
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
				<EmptyState v-else-if="!scraping" :icon="['fas', 'cube']" icon-size="4x" title="No Rule34Videos Yet" message="Use the sidebar controls to start scraping" />
			</div>

			<!-- Right: Scraper Sidebar -->
			<div class="scraper-sidebar" :class="{ collapsed: sidebarCollapsed }">
				<button class="collapse-toggle-btn mb-3 w-100" @click="sidebarCollapsed = true">
					<font-awesome-icon :icon="['fas', 'times']" class="me-2" />
					Hide Panel
				</button>
				<AutoUpdatePanel scraper-type="rule34video" />
				<!-- Statistics -->
				<div class="card mb-3">
					<div class="card-header">
						<h6 class="mb-0"><font-awesome-icon :icon="['fas', 'chart-bar']" class="me-2" />Statistics</h6>
					</div>
					<div class="card-body p-2">
						<div class="sidebar-stats-grid">
							<StatCard :value="stats.total_videos || 0" label="Videos" :icon="['fas', 'video']" icon-class="videos" />
							<StatCard :value="stats.avg_rating || 0" label="Avg Rating" :icon="['fas', 'star']" icon-class="keywords" />
							<StatCard :value="activeJob?.videosFound || 0" label="Found" :icon="['fas', 'search']" icon-class="found" />
							<StatCard :value="activeJob?.videosSaved || 0" label="Saved" :icon="['fas', 'save']" icon-class="active" />
						</div>
					</div>
				</div>
				<!-- Scraper Controls -->
				<div class="card mb-3">
					<div class="card-header">
						<h6 class="mb-0"><font-awesome-icon :icon="['fas', 'cube']" class="me-2 text-info" />Scraper</h6>
					</div>
					<div class="card-body">
						<div class="mb-2">
							<label class="form-label small">Sort By</label>
							<select v-model="sortBy" class="form-select form-select-sm">
								<option v-for="opt in sortOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
							</select>
						</div>
						<div class="mb-2">
							<label class="form-label small">Max Pages</label>
							<input v-model.number="maxPages" type="number" class="form-control form-control-sm" min="1" max="9999" placeholder="0 = unlimited" />
						</div>
						<button class="btn btn-primary btn-sm w-100" @click="startScrape" :disabled="scraping">
							<font-awesome-icon :icon="['fas', scraping ? 'spinner' : 'search']" :spin="scraping" class="me-2" />
							{{ scraping ? 'Scraping...' : 'Start Scrape' }}
						</button>
						<div class="alert alert-info mt-2 mb-0 py-1 px-2 small">
							<font-awesome-icon :icon="['fas', 'info-circle']" class="me-1" />
							Fetches tags, categories &amp; download links per video
						</div>
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
import { scrapersAPI, jdownloaderAPI } from '@/services/api'
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
const { sendingToJD, sendToJD } = useSendToJD(toast, 'rule34video')
const sendSelectedToJD = () => sendToJD(videos.value, selectedVideos.value)
const { pollJob } = useScraperJobPolling()
const { selectionMode, selectedVideos, toggleSelectionMode, toggleVideoSelection, selectAll } = useVideoSelection()
const { openInFirefox } = useOpenURL()
const { sidebarCollapsed, activeTimeouts } = useScraperSidebar()

const videosPerPage = SCRAPER_PAGE_SIZE

// ── State ─────────────────────────────────────────────────────────────────────
const filterCollapsed  = ref(false)
const sortBy           = ref('post_date')
const maxPages         = ref(0)
const scraping         = ref(false)
const loading          = ref(false)
const videos           = shallowRef([])
const stats            = ref({ total_videos: 0, avg_rating: 0 })
const activeJob        = ref(null)
const localCategories  = ref([])

const sortOptions = ref([
	{ value: 'post_date',           label: 'Most Recent' },
	{ value: 'video_viewed',        label: 'Most Viewed' },
	{ value: 'rating',              label: 'Top Rated' },
	{ value: 'duration',            label: 'Longest' },
	{ value: 'pseudo_rand',         label: 'Random' },
	{ value: 'video_viewed_today',  label: 'Most Viewed Today' },
	{ value: 'video_viewed_week',   label: 'Most Viewed This Week' },
	{ value: 'video_viewed_month',  label: 'Most Viewed This Month' },
])

// Video list filter state
const videoSearch      = ref('')
const videoSort        = ref('newest')
const videoPage        = ref(1)
const minDuration      = ref(0)
const maxDuration      = ref(0)
const onlyDownloaded   = ref(false)
const authorFilter     = ref('')
const categoryFilter   = ref('')
const includeTagFilters = ref([])
const excludeTagFilters = ref([])
const includeTagInput  = ref('')
const excludeTagInput  = ref('')
const minRating        = ref(0)
const minViews         = ref(0)
const totalCount       = ref(0)
const totalPages       = ref(0)

const debouncedSearch  = useDebouncedRef(videoSearch)
const debouncedAuthor  = useDebouncedRef(authorFilter)
const selectAllVideos  = () => selectAll(videos.value)

const activeFilterCount = computed(() => {
	let n = 0
	if (videoSort.value !== 'newest')         n++
	if (minDuration.value > 0)              n++
	if (maxDuration.value > 0)              n++
	if (onlyDownloaded.value)                  n++
	if (authorFilter.value)                    n++
	if (categoryFilter.value)                  n++
	if (includeTagFilters.value.length > 0)    n++
	if (excludeTagFilters.value.length > 0)    n++
	if (parseInt(minRating.value) > 0)         n++
	if (parseInt(minViews.value) > 0)          n++
	return n
})

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

watch(debouncedSearch,   () => { videoPage.value = 1; loadVideos() })
watch(videoSort,         () => { videoPage.value = 1; loadVideos() })
watch(minDuration,      () => { videoPage.value = 1; loadVideos() })
watch(maxDuration,      () => { videoPage.value = 1; loadVideos() })
watch(onlyDownloaded,    () => { videoPage.value = 1; loadVideos() })
watch(debouncedAuthor,   () => { videoPage.value = 1; loadVideos() })
watch(categoryFilter,    () => { videoPage.value = 1; loadVideos() })
watch(includeTagFilters, () => { videoPage.value = 1; loadVideos() }, { deep: true })
watch(excludeTagFilters, () => { videoPage.value = 1; loadVideos() }, { deep: true })
watch(minRating,         () => { videoPage.value = 1; loadVideos() })
watch(minViews,          () => { videoPage.value = 1; loadVideos() })

// ── Tag helpers ───────────────────────────────────────────────────────────────
const addIncludeTag = () => {
	const t = includeTagInput.value.trim().toLowerCase()
	if (t && !includeTagFilters.value.includes(t)) includeTagFilters.value.push(t)
	includeTagInput.value = ''
}
const removeIncludeTag = (i) => includeTagFilters.value.splice(i, 1)

const addExcludeTag = () => {
	const t = excludeTagInput.value.trim().toLowerCase()
	if (t && !excludeTagFilters.value.includes(t)) excludeTagFilters.value.push(t)
	excludeTagInput.value = ''
}
const removeExcludeTag = (i) => excludeTagFilters.value.splice(i, 1)

// Card click-to-filter handlers
const addIncludeTagFromCard = (tag) => {
	const t = tag.trim().toLowerCase()
	if (t && !includeTagFilters.value.includes(t)) {
		includeTagFilters.value.push(t)
		filterCollapsed.value = false // open filter panel so user sees what was added
	}
}
const setCategoryFromCard = (cat) => {
	categoryFilter.value = cat
	filterCollapsed.value = false
}

// ── API ───────────────────────────────────────────────────────────────────────
const loadStats = async () => {
	try {
		const data = await scrapersAPI.rule34video.getStats()
		if (data.success) stats.value = data.data
	} catch (e) { console.error('Error loading Rule34Video stats:', e) }
}

const loadLocalCategories = async () => {
	try {
		const data = await scrapersAPI.rule34video.getLocalCategories()
		if (data.success && data.data) localCategories.value = data.data
	} catch (e) { console.error('Error loading local categories:', e) }
}

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
		if (parseInt(minDuration.value) > 0) params.min_duration = minDuration.value * 60
		if (parseInt(maxDuration.value) > 0) params.max_duration = maxDuration.value * 60
		if (onlyDownloaded.value)                params.downloaded_only = true
		if (debouncedAuthor.value)               params.author = debouncedAuthor.value
		if (categoryFilter.value)                params.category = categoryFilter.value
		if (includeTagFilters.value.length > 0)  params.include_tags = includeTagFilters.value.join(',')
		if (excludeTagFilters.value.length > 0)  params.exclude_tags = excludeTagFilters.value.join(',')
		if (parseInt(minRating.value) > 0)       params.min_rating   = minRating.value
		if (parseInt(minViews.value) > 0)        params.min_views    = minViews.value

		const data = await scrapersAPI.rule34video.getVideos(params)
		if (data.success && data.data) {
			videos.value     = data.data.videos     || []
			totalCount.value = data.data.total      || 0
			totalPages.value = data.data.totalPages || 0
		}
	} catch (e) { console.error('Error loading Rule34Video videos:', e) }
	finally { loading.value = false }
}

const onClearFilters = () => {
	authorFilter.value = ''
	categoryFilter.value = ''
	includeTagFilters.value = []
	excludeTagFilters.value = []
	minRating.value = 0
	minViews.value  = 0
	minDuration.value = 0
	maxDuration.value = 0
	videoPage.value = 1
	loadVideos()
}

const clearVideos = async () => {
	if (!confirm(`Are you sure you want to delete all ${totalCount.value} Rule34Video videos? This cannot be undone.`)) return
	try {
		const data = await scrapersAPI.rule34video.clearVideos()
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} Rule34Video videos`)
			videos.value = []; totalCount.value = 0; totalPages.value = 0
			localCategories.value = []
			await loadStats()
		}
	} catch (e) { console.error('Error deleting Rule34Video videos:', e); toast.error('Failed to delete videos') }
}

const deleteSelectedVideos = async () => {
	if (selectedVideos.value.size === 0) return
	if (!confirm(`Delete ${selectedVideos.value.size} selected videos?`)) return
	try {
		const data = await scrapersAPI.bulkDelete(Array.from(selectedVideos.value))
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} videos`)
			selectedVideos.value = new Set(); selectionMode.value = false
			await loadVideos(); await loadStats()
		}
	} catch (e) { console.error('Error deleting videos:', e); toast.error('Failed to delete videos') }
}

const sendToJDownloader = async (video) => {
	try {
		// Read download URLs from already-stored metadata (no extra HTTP request needed)
		let downloadURL = video.url
		try {
			const meta = typeof video.metadata === 'string' ? JSON.parse(video.metadata) : (video.metadata || {})
			if (meta.download_urls) {
				downloadURL = meta.download_urls['720p'] || meta.download_urls['480p'] || meta.download_urls['360p'] || video.url
			}
		} catch { /* use video page URL as fallback */ }

		await jdownloaderAPI.addLinks({ links: [downloadURL], packageName: video.title || 'Rule34Video' })
		toast.success('Sent to JDownloader')
	} catch (e) {
		console.error('JDownloader error:', e)
		toast.error('Failed to send to JDownloader')
	}
}

const startScrape = async () => {
	scraping.value = true
	try {
		const data = await scrapersAPI.rule34video.scrape({
			maxPages: maxPages.value || 0,
			sortBy:   sortBy.value,
		})
		if (data.success) {
			toast.success('Started Rule34Video scrape')
			activeJob.value = { id: data.data.job_id, status: 'running', pagesCompleted: 0, videosFound: 0, videosSaved: 0, errorsCount: 0 }
			pollJob(data.data.job_id, {
				activeJob, isScraping: scraping, activeTimeouts,
				onSuccess: async (job) => {
					toast.success(`Rule34Video scrape complete! Found ${job.videosFound} videos`)
					await loadVideos(); await loadStats(); await loadLocalCategories()
				},
				onFail: () => toast.error('Rule34Video scrape failed'),
			})
		} else {
			toast.error(data.message || 'Failed to start Rule34Video scrape')
			scraping.value = false
		}
	} catch (e) { console.error('Error starting Rule34Video scrape:', e); toast.error('Failed to start'); scraping.value = false }
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────
useScraperAutoRefresh('rule34video', () => { videoPage.value = 1; loadVideos() })
onMounted(() => {
	loadStats()
	loadVideos()
	loadLocalCategories()
})
</script>

<style scoped>
.sfp-refresh-btn {
	background: none;
	border: none;
	color: var(--text-muted, #888);
	cursor: pointer;
	padding: 0 0 0 6px;
	font-size: 0.75rem;
	line-height: 1;
}
.sfp-refresh-btn:hover { color: #fff; }

.r34-tag-cloud {
	display: flex;
	flex-wrap: wrap;
	gap: 4px;
	margin-top: 4px;
}

.r34-tag-chip {
	display: inline-flex;
	align-items: center;
	gap: 3px;
	border-radius: 12px;
	padding: 2px 8px;
	font-size: 0.72rem;
}

.r34-tag-include {
	background: rgba(25, 135, 84, 0.2);
	border: 1px solid rgba(25, 135, 84, 0.4);
	color: #75c98a;
}

.r34-tag-exclude {
	background: rgba(220, 53, 69, 0.2);
	border: 1px solid rgba(220, 53, 69, 0.4);
	color: #f08080;
}

.r34-tag-remove {
	background: none;
	border: none;
	color: inherit;
	cursor: pointer;
	padding: 0;
	line-height: 1;
	font-size: 0.85rem;
	opacity: 0.7;
}
.r34-tag-remove:hover { opacity: 1; }
</style>
