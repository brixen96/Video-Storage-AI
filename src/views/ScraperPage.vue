<template>
	<div class="scraper-page">
		<div class="container-fluid py-3">
			<!-- Tabs for different scrapers -->
			<div class="scraper-tabs mb-3">
				<!-- Simpcity Tab (Always present) -->
				<button class="scraper-tab" :class="{ active: activeTab === 'simpcity' }" @click="activeTab = 'simpcity'">
					<font-awesome-icon :icon="['fas', 'comments']" class="me-2" />
					Simpcity Forum
				</button>

				<!-- PornHub Tab (Always present) -->
				<button class="scraper-tab pornhub-tab" :class="{ active: activeTab === 'pornhub' }" @click="activeTab = 'pornhub'">
					<font-awesome-icon :icon="['fas', 'fire']" class="me-2" />
					PornHub
				</button>

				<!-- LuxureTV Tab (Always present) -->
				<button class="scraper-tab luxuretv-tab" :class="{ active: activeTab === 'luxuretv' }" @click="activeTab = 'luxuretv'">
					<font-awesome-icon :icon="['fas', 'paw']" class="me-2" />
					LuxureTV
				</button>

				<!-- Sexuria Tab (Always present) -->
				<button class="scraper-tab sexuria-tab" :class="{ active: activeTab === 'sexuria' }" @click="activeTab = 'sexuria'">
					<font-awesome-icon :icon="['fas', 'download']" class="me-2" />
					Sexuria
				</button>

				<!-- Rule34Video Tab (Always present) -->
				<button class="scraper-tab rule34video-tab" :class="{ active: activeTab === 'rule34video' }" @click="activeTab = 'rule34video'">
					<font-awesome-icon :icon="['fas', 'cube']" class="me-2" />
					Rule34Videos
				</button>

				<!-- Motherless Tab -->
				<button class="scraper-tab motherless-tab" :class="{ active: activeTab === 'motherless' }" @click="activeTab = 'motherless'">
					<font-awesome-icon :icon="['fas', 'film']" class="me-2" />
					Motherless
				</button>

				<!-- HQPorner Tab -->
				<button class="scraper-tab hqporner-tab" :class="{ active: activeTab === 'hqporner' }" @click="activeTab = 'hqporner'">
					<font-awesome-icon :icon="['fas', 'bolt']" class="me-2" />
					HQPorner
				</button>

				<!-- LeakVids Tab -->
				<button class="scraper-tab leakvids-tab" :class="{ active: activeTab === 'leakvids' }" @click="activeTab = 'leakvids'">
					<font-awesome-icon :icon="['fas', 'eye-slash']" class="me-2" />
					LeakVids
				</button>

				<!-- Dynamic Custom Tabs -->
				<button v-for="tab in scraperTabs" :key="tab.id" class="scraper-tab" :class="{ active: activeTab === `tab-${tab.id}` }" :style="{ borderColor: tab.color }" @click="selectTab(tab.id)">
					<font-awesome-icon :icon="['fas', tab.icon]" class="me-2" />
					{{ tab.name }}
					<button class="btn-close-tab" @click.stop="confirmDeleteTab(tab)" title="Delete tab">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</button>

				<!-- Add New Tab Button -->
				<button class="scraper-tab add-tab" @click="showAddTabModal = true">
					<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
					Add Tab
				</button>
			</div>

			<!-- Simpcity Tab Content -->
			<div v-if="activeTab === 'simpcity'">
				<SimpcityTab />
			</div>
			<!-- End Simpcity Tab Content (modals are shared between tabs) -->

			<!-- PornHub Tab Content -->
			<div v-if="activeTab === 'pornhub'" class="tab-content">
				<PornHubTab />
			</div>
			<!-- End PornHub Tab Content -->

			<!-- LuxureTV Tab Content -->
			<div v-if="activeTab === 'luxuretv'" class="tab-content">
				<LuxureTVTab />
			</div>
			<!-- End LuxureTV Tab Content -->

			<!-- Sexuria Tab Content -->
			<div v-if="activeTab === 'sexuria'" class="tab-content">
				<SexuriaTab />
			</div>
			<!-- End Sexuria Tab Content -->

			<!-- Rule34Video Tab Content -->
			<div v-if="activeTab === 'rule34video'" class="tab-content">
				<Rule34VideoTab />
			</div>
			<!-- End Rule34Video Tab Content -->
			<!-- Motherless Tab Content -->
			<div v-if="activeTab === 'motherless'" class="tab-content">
				<MotherlessTab />
			</div>

			<!-- HQPorner Tab Content -->
			<div v-if="activeTab === 'hqporner'" class="tab-content">
				<HQPornerTab />
			</div>

			<!-- LeakVids Tab Content -->
			<div v-if="activeTab === 'leakvids'" class="tab-content">
				<LeakVidsTab />
			</div>

			<!-- Dynamic Custom Tab Content -->
			<div v-for="tab in scraperTabs" :key="`content-${tab.id}`">
				<div v-if="activeTab === `tab-${tab.id}`" class="tab-content">
					<div class="scraper-layout" :class="{ 'sidebar-collapsed': collapsedSections[`tab-${tab.id}`] }">
						<!-- Main Content (Left) -->
						<div class="scraper-main">
							<!-- Search and Filters -->
							<div class="filter-bar mb-3">
								<div class="row g-2 align-items-end">
									<div class="col-md-4">
										<input v-model="videoSearch[tab.id]" type="text" class="form-control form-control-sm" placeholder="Search videos..." @input="onTabSearchInput(tab.id)" />
									</div>
									<div class="col-md-3">
										<select v-model="videoSort[tab.id]" class="form-select form-select-sm" @change="onTabFilterChange(tab.id)">
											<option value="newest">Newest First</option>
											<option value="oldest">Oldest First</option>
											<option value="views-desc">Most Views</option>
											<option value="duration-desc">Longest</option>
											<option value="duration-asc">Shortest</option>
											<option value="title-asc">Title (A-Z)</option>
											<option value="title-desc">Title (Z-A)</option>
											<option value="likes-desc">Most Likes</option>
										</select>
									</div>
									<div class="col-md-2">
										<select v-model="videoDurationFilter[tab.id]" class="form-select form-select-sm" @change="onTabFilterChange(tab.id)">
											<option value="">Any Duration</option>
											<option value="short">Short (&lt;5 min)</option>
											<option value="medium">Medium (5–20 min)</option>
											<option value="long">Long (&gt;20 min)</option>
										</select>
									</div>
									<div class="col-md-3 d-flex gap-1">
										<button class="btn btn-sm btn-outline-secondary flex-grow-1" @click="toggleVideoFilters(tab.id)">
											<font-awesome-icon :icon="['fas', 'filter']" class="me-1" />
											More
											<span v-if="activeTabFilterCount(tab.id) > 0" class="badge bg-primary ms-1">{{ activeTabFilterCount(tab.id) }}</span>
										</button>
										<button class="btn btn-sm btn-outline-warning" @click="resetVideoFilters(tab.id)" title="Reset all filters">
											<font-awesome-icon :icon="['fas', 'redo']" />
										</button>
									</div>
								</div>
								<!-- Advanced Filters (Collapsible) -->
								<div v-if="showVideoFilters[tab.id]" class="mt-2 pt-2 border-top border-secondary">
									<div class="row g-2 align-items-center">
										<div class="col-md-4">
											<input v-model="videoAuthorFilter[tab.id]" type="text" class="form-control form-control-sm" placeholder="Author / uploader..." @input="onTabSearchInput(tab.id)" />
										</div>
										<div class="col-md-3">
											<input v-model.number="videoMinViews[tab.id]" type="number" min="0" class="form-control form-control-sm" placeholder="Min views (0 = any)" @change="onTabFilterChange(tab.id)" />
										</div>
										<div class="col-md-3 d-flex align-items-center">
											<div class="form-check mb-0">
												<input :id="`only-dl-${tab.id}`" v-model="videoOnlyDownloaded[tab.id]" class="form-check-input" type="checkbox" @change="onTabFilterChange(tab.id)" />
												<label :for="`only-dl-${tab.id}`" class="form-check-label small">Downloaded only</label>
											</div>
										</div>
									</div>
								</div>
							</div>

							<!-- Active Job Progress -->
							<div v-if="customTabJobs[tab.id]" class="card bg-dark mb-3">
								<div class="card-body py-2">
									<div class="d-flex justify-content-between align-items-center">
										<span class="small">
											<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-2 text-primary" />
											{{ customTabJobs[tab.id].status }}
										</span>
										<span class="small">
											<strong>{{ customTabJobs[tab.id].pagesCompleted }}</strong> pages | <strong>{{ customTabJobs[tab.id].videosFound }}</strong> found | <strong>{{ customTabJobs[tab.id].videosSaved }}</strong> saved
										</span>
									</div>
								</div>
							</div>

							<!-- Loading State -->
							<LoadingState v-if="customTabLoading[tab.id]" show-text :loading-text="`Loading ${tab.name} videos...`" />

							<!-- Scraped Videos Grid -->
							<div v-else-if="customTabVideos[tab.id] && customTabVideos[tab.id].length > 0">
								<div class="d-flex justify-content-between align-items-center mb-2">
									<h6 class="mb-0">
										<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
										{{ tab.name }} Videos ({{ (tabVideoTotal[tab.id] || 0).toLocaleString() }})
									</h6>
									<div class="d-flex gap-1">
										<button
											class="btn btn-sm"
											:class="customTabSelectionMode[tab.id] ? 'btn-primary' : 'btn-outline-secondary'"
											@click="toggleTabSelectionMode(tab.id)"
											title="Toggle selection mode"
										>
											<font-awesome-icon :icon="['fas', 'check-square']" class="me-1" />
											{{ customTabSelectionMode[tab.id] ? `${(customTabSelectedVideos[tab.id]?.size || 0)} selected` : 'Select' }}
										</button>
										<button class="btn btn-sm btn-outline-secondary" @click="loadVideosForTab(tab.id)" title="Reload videos">
											<font-awesome-icon :icon="['fas', 'sync']" />
										</button>
										<button class="btn btn-sm btn-outline-danger" @click="clearCustomTabVideos(tab)" title="Delete all videos for this scraper">
											<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />
											Delete All
										</button>
									</div>
								</div>

								<!-- Bulk Actions Bar -->
								<ScraperBulkActions
									:selectionMode="customTabSelectionMode[tab.id] || false"
									:selectedCount="customTabSelectedVideos[tab.id]?.size || 0"
									:sendingToJD="customTabJDSending[tab.id] || false"
									@select-all="selectAllTabVideos(tab.id)"
									@send-to-jd="sendTabSelectedToJD(tab)"
									@delete-selected="deleteTabSelectedVideos(tab)"
								/>

								<div class="video-grid">
									<ScrapedVideoCard
										v-for="video in customTabVideos[tab.id]"
										:key="video.id"
										:video="video"
										:selectionMode="customTabSelectionMode[tab.id] || false"
										:isSelected="customTabSelectedVideos[tab.id]?.has(video.id) || false"
										@toggle-selection="toggleTabVideoSelection(tab.id, $event)"
									>
										<template #actions="{ video: v }">
											<button @click="playScrapedVideo(v, tab)" class="btn btn-sm btn-primary">
												<font-awesome-icon :icon="['fas', 'play']" class="me-1" />
												Play
											</button>
											<button @click="scrapersAPI.openURL(v.source_url || v.url).catch(() => {})" class="btn btn-sm btn-outline-primary" title="Open in Firefox">
												<font-awesome-icon :icon="['fab', 'firefox-browser']" />
											</button>
											<button v-if="jdAvailable && (v.source_url || v.embed_url)" @click="sendToJDownloader(v, tab)" class="btn btn-sm btn-warning" :disabled="jdSending[v.id]" title="Send to JDownloader">
												<font-awesome-icon :icon="['fas', jdSending[v.id] ? 'spinner' : 'cloud-download-alt']" :spin="jdSending[v.id]" />
											</button>
										</template>
									</ScrapedVideoCard>
								</div>

								<!-- Pagination -->
								<div class="d-flex align-items-center justify-content-between mt-3">
									<!-- Left: count + per-page selector -->
									<div class="d-flex align-items-center gap-2">
										<span class="text-muted small">
											Showing {{ ((videoPage[tab.id] || 1) - 1) * videosPerPage.value + 1 }}–{{ Math.min((videoPage[tab.id] || 1) * videosPerPage.value, tabVideoTotal[tab.id] || 0) }} of {{ (tabVideoTotal[tab.id] || 0).toLocaleString() }}
										</span>
										<select class="form-select form-select-sm" style="width: auto" @change="changeVideosPerPage($event.target.value, tab.id)">
											<option :value="12" :selected="videosPerPage.value === 12">12/page</option>
											<option :value="24" :selected="videosPerPage.value === 24">24/page</option>
											<option :value="48" :selected="videosPerPage.value === 48">48/page</option>
											<option :value="96" :selected="videosPerPage.value === 96">96/page</option>
										</select>
									</div>
									<!-- Center: nav buttons -->
									<div v-if="(tabVideoTotalPages[tab.id] || 1) > 1" class="btn-group">
										<button class="btn btn-sm btn-outline-secondary" :disabled="(videoPage[tab.id] || 1) === 1" @click="changeVideoPage(tab.id, -1)">
											<font-awesome-icon :icon="['fas', 'chevron-left']" />
										</button>
										<button class="btn btn-sm btn-outline-secondary disabled">Page {{ videoPage[tab.id] || 1 }} / {{ tabVideoTotalPages[tab.id] || 1 }}</button>
										<button class="btn btn-sm btn-outline-secondary" :disabled="(videoPage[tab.id] || 1) >= (tabVideoTotalPages[tab.id] || 1)" @click="changeVideoPage(tab.id, 1)">
											<font-awesome-icon :icon="['fas', 'chevron-right']" />
										</button>
									</div>
									<div v-else></div>
									<!-- Right spacer -->
									<div style="min-width: 120px"></div>
								</div>
							</div>

							<!-- Empty State -->
							<EmptyState v-else :icon="['fas', 'video']" icon-size="4x" :title="`No ${tab.name} Videos Yet`" message="Use the sidebar to configure and start scraping" />
						</div>
						<!-- End Main Content -->

						<!-- Sidebar (Right) -->
						<div class="scraper-sidebar" :class="{ collapsed: collapsedSections[`tab-${tab.id}`] }">
							<!-- Collapse Toggle -->
							<button class="btn btn-sm btn-outline-secondary w-100 mb-3" @click="toggleCollapse(`tab-${tab.id}`)">
								<font-awesome-icon :icon="['fas', 'chevron-right']" class="me-2" />
								Hide Panel
							</button>

							<!-- Auto-Update Panel -->
							<div class="mb-3">
								<AutoUpdatePanel :scraper-type="`custom_${tab.id}`" />
							</div>

							<!-- Statistics -->
							<div class="card mb-3">
								<div class="card-header">
									<h6 class="mb-0">
										<font-awesome-icon :icon="['fas', 'chart-bar']" class="me-2" />
										Statistics
									</h6>
								</div>
								<div class="card-body p-2">
									<div class="sidebar-stats-grid">
										<StatCard :value="customTabVideos[tab.id]?.length || 0" label="Videos" :icon="['fas', 'video']" icon-class="videos" />
										<StatCard :value="customTabJobs[tab.id] ? 1 : 0" label="Active" :icon="['fas', 'spinner']" icon-class="active" />
									</div>
								</div>
							</div>

							<!-- Scraper Controls -->
							<div class="card mb-3">
								<div class="card-header">
									<h6 class="mb-0">
										<font-awesome-icon :icon="['fas', tab.icon || 'globe']" class="me-2" :style="{ color: tab.color }" />
										Scraper
									</h6>
								</div>
								<div class="card-body">
									<div class="mb-2">
										<label class="form-label small">Base URL</label>
										<input :value="tab.baseUrl" type="text" class="form-control form-control-sm" disabled />
									</div>
									<div class="mb-2">
										<label class="form-label small">Site Type</label>
										<input :value="tab.siteType || 'Generic'" type="text" class="form-control form-control-sm" disabled />
									</div>
									<div class="d-flex flex-column gap-2">
										<button class="btn btn-primary btn-sm w-100" @click="startCustomTabScrape(tab)" :disabled="customTabJobs[tab.id]">
											<font-awesome-icon :icon="['fas', customTabJobs[tab.id] ? 'spinner' : 'play']" :spin="customTabJobs[tab.id]" class="me-2" />
											{{ customTabJobs[tab.id] ? 'Scraping...' : 'Start Scrape' }}
										</button>
										<button class="btn btn-outline-secondary btn-sm w-100" @click="editScraperTab(tab)">
											<font-awesome-icon :icon="['fas', 'edit']" class="me-2" />
											Edit Tab
										</button>
										<button class="btn btn-outline-danger btn-sm w-100" @click="clearCustomTabVideos(tab)">
											<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
											Delete All Videos
										</button>
									</div>
								</div>
							</div>
						</div>
						<!-- End Sidebar -->
					</div>
					<!-- End scraper-layout -->

					<!-- Sidebar Toggle Button (shown when collapsed) -->
					<button v-if="collapsedSections[`tab-${tab.id}`]" class="sidebar-toggle" @click="toggleCollapse(`tab-${tab.id}`)" title="Show Controls Panel">
						<font-awesome-icon :icon="['fas', 'cog']" />
					</button>
				</div>
			</div>
			<!-- End Dynamic Custom Tab Content -->

			<!-- Add/Edit Tab Modal -->
			<div v-if="showAddTabModal || showEditTabModal" class="modal show d-block" tabindex="-1" style="background-color: rgba(0, 0, 0, 0.5)">
				<div class="modal-dialog modal-lg">
					<div class="modal-content bg-dark text-light">
						<div class="modal-header">
							<h5 class="modal-title">
								<font-awesome-icon :icon="['fas', showEditTabModal ? 'edit' : 'plus']" class="me-2" />
								{{ showEditTabModal ? 'Edit' : 'Add New' }} Scraper Tab
							</h5>
							<button type="button" class="btn-close btn-close-white" @click="closeTabModal"></button>
						</div>
						<div class="modal-body">
							<div class="mb-3">
								<label class="form-label">Tab Name</label>
								<input v-model="newTab.name" type="text" class="form-control" placeholder="e.g., My Favorite Site" />
							</div>
							<div class="mb-3">
								<label class="form-label">Base URL</label>
								<input v-model="newTab.baseUrl" type="text" class="form-control" placeholder="https://example.com/videos" />
								<small class="text-muted">The base URL pattern for this site's video listings</small>
							</div>
							<div class="row g-3 mb-3">
								<div class="col-md-6">
									<label class="form-label">Site Type</label>
									<select v-model="newTab.siteType" class="form-select">
										<option value="generic">Generic</option>
										<option value="youtube">YouTube</option>
										<option value="vimeo">Vimeo</option>
										<option value="custom">Custom</option>
									</select>
								</div>
								<div class="col-md-6">
									<label class="form-label">Icon</label>
									<select v-model="newTab.icon" class="form-select">
										<option value="video">Video</option>
										<option value="film">Film</option>
										<option value="play-circle">Play Circle</option>
										<option value="youtube">YouTube</option>
										<option value="vimeo">Vimeo</option>
										<option value="camera">Camera</option>
										<option value="globe">Globe</option>
									</select>
								</div>
							</div>
							<div class="row g-3 mb-3">
								<div class="col-md-6">
									<label class="form-label">Tab Color</label>
									<input v-model="newTab.color" type="color" class="form-control" />
								</div>
								<div class="col-md-6">
									<label class="form-label">Content Type</label>
									<select v-model="newTab.contentType" class="form-select">
										<option value="regular">Regular</option>
										<option value="zoo">Zoo</option>
										<option value="3d">3D / Animation</option>
									</select>
									<small class="text-muted">Used for filtering in Videos page</small>
								</div>
							</div>
							<div class="mb-3">
								<label class="form-label">Scrape Options</label>
								<div class="form-check">
									<input v-model="newTab.scrapeOptions.extractMetadata" class="form-check-input" type="checkbox" id="newTabExtractMetadata" />
									<label class="form-check-label" for="newTabExtractMetadata"> Extract video metadata </label>
								</div>
								<div class="form-check">
									<input v-model="newTab.scrapeOptions.scrapeThumbnails" class="form-check-input" type="checkbox" id="newTabScrapeThumbnails" />
									<label class="form-check-label" for="newTabScrapeThumbnails"> Download thumbnails </label>
								</div>
							</div>
						</div>
						<div class="modal-footer">
							<button type="button" class="btn btn-secondary" @click="closeTabModal">Cancel</button>
							<button type="button" class="btn btn-primary" @click="saveTabConfig">
								<font-awesome-icon :icon="['fas', 'save']" class="me-2" />
								Save Tab
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>

	<ScrapedVideoModal :video="modalVideo" :tab="modalTab" @close=";(modalVideo = null), (modalTab = null)" @downloaded="onVideoDownloaded" @unmarked="onVideoUnmarked" />
</template>

<script setup>
/* eslint-disable no-unused-vars */
import { ref, onMounted, onBeforeUnmount, getCurrentInstance } from 'vue'
import { LoadingState, EmptyState, StatCard } from '@/components/shared'
import AutoUpdatePanel from '@/components/scraper/AutoUpdatePanel.vue'
import ScrapedVideoCard from '@/components/scraper/ScrapedVideoCard.vue'
import ScrapedVideoModal from '@/components/scraper/ScrapedVideoModal.vue'
import ScraperBulkActions from '@/components/scraper/ScraperBulkActions.vue'
import SimpcityTab from '@/components/scraper/tabs/SimpcityTab.vue'
import LuxureTVTab from '@/components/scraper/tabs/LuxureTVTab.vue'
import PornHubTab from '@/components/scraper/tabs/PornHubTab.vue'
import SexuriaTab from '@/components/scraper/tabs/SexuriaTab.vue'
import Rule34VideoTab from '@/components/scraper/tabs/Rule34VideoTab.vue'
import MotherlessTab from '@/components/scraper/tabs/MotherlessTab.vue'
import HQPornerTab from '@/components/scraper/tabs/HQPornerTab.vue'
import LeakVidsTab from '@/components/scraper/tabs/LeakVidsTab.vue'
import { jdownloaderAPI, scrapersAPI, deleteVideosBySource } from '@/services/api'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast

// JDownloader integration
const jdAvailable = ref(false)
const jdSending = ref({}) // Track sending state per video ID

const checkJDownloaderStatus = async () => {
	try {
		const result = await jdownloaderAPI.isAvailable()
		jdAvailable.value = result?.available || false
	} catch {
		jdAvailable.value = false
	}
}

const sendToJDownloader = async (video, tab = null) => {
	if (!video?.source_url && !video?.embed_url) {
		toast.warning('No URL available for this video')
		return
	}

	const key = video.id || video.source_url
	jdSending.value[key] = true

	try {
		let url = video.source_url || video.embed_url
		const videoTitle = video.title || ''
		const scraperName = tab?.name || video.source || 'Scraper'

		// Try to extract direct video URL (MP4/etc) from the page
		try {
			const extracted = await scrapersAPI.extractVideoUrl(url)
			if (extracted?.direct_url) {
				url = extracted.direct_url
			}
		} catch {
			// Fall back to source URL if extraction fails
		}

		const result = await jdownloaderAPI.sendLinks([
			{
				url,
				packageName: videoTitle,
				downloadPath: `/output/${scraperName}`,
				videoId: video?.id ?? null,
			},
		])

		if (result?.successCount > 0) {
			toast.success('Link sent to JDownloader')
		} else if (result?.results?.[0]?.message) {
			toast.error(result.results[0].message)
		} else {
			toast.error('Failed to send link')
		}
	} catch (error) {
		console.error('JDownloader send error:', error)
		toast.error('Failed to send to JDownloader')
	} finally {
		jdSending.value[key] = false
	}
}

// Scraped video modal
const modalVideo = ref(null)
const modalTab = ref(null)

// Tab state
const activeTab = ref('simpcity')

// Collapsible controls state (collapsed by default to focus on content)
const collapsedSections = ref({
	simpcity: true,
	pornhub: true,
	luxuretv: true,
	sexuria: true,
	rule34video: true,
})

const toggleCollapse = (tab) => {
	collapsedSections.value[tab] = !collapsedSections.value[tab]
}

// Scraper tabs state
const scraperTabs = ref([])
const showAddTabModal = ref(false)
const showEditTabModal = ref(false)
const editingTab = ref(null)
const newTab = ref({
	name: '',
	baseUrl: '',
	siteType: 'generic',
	icon: 'video',
	color: '#007bff',
	contentType: 'regular',
	scrapeOptions: {
		extractMetadata: true,
		scrapeThumbnails: true,
		downloadVideo: false,
	},
})
const customTabUrls = ref({}) // { tabId: url }
const customTabJobs = ref({}) // { tabId: job }
const customTabVideos = ref({}) // { tabId: [videos] }
const customTabLoading = ref({}) // { tabId: boolean } - Loading state for each custom tab
const videoSearch = ref({}) // { tabId: searchQuery }
const videoSort = ref({}) // { tabId: sortType }
const videoPage = ref({}) // { tabId: pageNumber }
const videoStatusFilter = ref({}) // { tabId: status }
const videoResolutionFilter = ref({}) // { tabId: resolution }
const videoDurationFilter = ref({}) // { tabId: durationRange }
const videoThumbnailFilter = ref({}) // { tabId: 'yes'|'no'|'' } — kept for compat, not used in server query
const videoOnlyDownloaded = ref({}) // { tabId: boolean }
const videoAuthorFilter = ref({}) // { tabId: string }
const videoMinViews = ref({}) // { tabId: number }
const tabVideoTotal = ref({}) // { tabId: number } — server-side total count
const tabVideoTotalPages = ref({}) // { tabId: number } — server-side total pages
const showVideoFilters = ref({}) // { tabId: boolean }
const tabSearchTimeouts = {} // debounce per tab
// Per-tab selection state (bulk actions)
const customTabSelectionMode = ref({}) // { tabId: boolean }
const customTabSelectedVideos = ref({}) // { tabId: Set<number> }
const customTabJDSending = ref({}) // { tabId: boolean } — loading state for bulk JD send
const videosPerPage = ref(parseInt(localStorage.getItem('scraper-page-size') || '24'))

// Timeout tracking for cleanup
const activeTimeouts = ref([])

// Scraper Tabs Functions
const loadScraperTabs = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/tabs')
		const data = await response.json()

		if (data.success && data.data) {
			scraperTabs.value = data.data
			// Initialize URLs, jobs, and video controls for each tab
			data.data.forEach((tab) => {
				if (!customTabUrls.value[tab.id]) {
					customTabUrls.value[tab.id] = tab.baseUrl
				}
				// Initialize video controls
				if (!videoSort.value[tab.id]) {
					videoSort.value[tab.id] = 'newest'
				}
				if (!videoPage.value[tab.id]) {
					videoPage.value[tab.id] = 1
				}
				if (!videoSearch.value[tab.id]) {
					videoSearch.value[tab.id] = ''
				}
				// Start collapsed by default (same as built-in scrapers)
				const key = `tab-${tab.id}`
				if (collapsedSections.value[key] === undefined) {
					collapsedSections.value[key] = true
				}
			})
		}
	} catch (error) {
		console.error('Error loading scraper tabs:', error)
	}
}

const selectTab = (tabId) => {
	activeTab.value = `tab-${tabId}`
	loadTabJobProgress(tabId)
	loadVideosForTab(tabId)
}

// Helper to get the endpoint type for auto-update based on site type
// eslint-disable-next-line no-unused-vars
const getEndpointTypeForSiteType = (siteType) => {
	switch (siteType) {
		case 'pornhub':
			return 'keywords'
		// All other tabs (LuxureTV, Sexuria, generic) use their baseUrl directly
		// No additional endpoint configuration needed
		default:
			return null
	}
}

const toggleVideoFilters = (tabId) => {
	showVideoFilters.value[tabId] = !showVideoFilters.value[tabId]
}

const resetVideoFilters = (tabId) => {
	videoSearch.value[tabId] = ''
	videoSort.value[tabId] = 'newest'
	videoStatusFilter.value[tabId] = ''
	videoResolutionFilter.value[tabId] = ''
	videoDurationFilter.value[tabId] = ''
	videoThumbnailFilter.value[tabId] = ''
	videoOnlyDownloaded.value[tabId] = false
	videoAuthorFilter.value[tabId] = ''
	videoMinViews.value[tabId] = 0
	videoPage.value[tabId] = 1
	loadVideosForTab(tabId)
}

const confirmDeleteTab = async (tab) => {
	if (!confirm(`Are you sure you want to delete the "${tab.name}" tab? This cannot be undone.`)) {
		return
	}

	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/tabs/${tab.id}`, {
			method: 'DELETE',
		})
		const data = await response.json()

		if (data.success) {
			toast.success(`Tab "${tab.name}" deleted successfully`)

			// Switch to simpcity tab if we're deleting the active tab
			if (activeTab.value === `tab-${tab.id}`) {
				activeTab.value = 'simpcity'
			}

			// Reload tabs
			await loadScraperTabs()
		} else {
			toast.error(data.message || 'Failed to delete tab')
		}
	} catch (error) {
		console.error('Error deleting tab:', error)
		toast.error('Failed to delete tab: ' + error.message)
	}
}

// eslint-disable-next-line no-unused-vars
const editScraperTab = (tab) => {
	editingTab.value = tab
	newTab.value = {
		name: tab.name,
		baseUrl: tab.baseUrl,
		siteType: tab.siteType,
		icon: tab.icon,
		color: tab.color,
		contentType: tab.contentType || 'regular',
		scrapeOptions: { ...tab.scrapeOptions },
		isActive: tab.isActive !== false, // Preserve active state, default to true
		position: tab.position || 0, // Preserve position
	}
	showEditTabModal.value = true
}

const closeTabModal = () => {
	showAddTabModal.value = false
	showEditTabModal.value = false
	editingTab.value = null
	newTab.value = {
		name: '',
		baseUrl: '',
		siteType: 'generic',
		icon: 'video',
		color: '#007bff',
		contentType: 'regular',
		scrapeOptions: {
			extractMetadata: true,
			scrapeThumbnails: true,
			downloadVideo: false,
		},
	}
}

const saveTabConfig = async () => {
	if (!newTab.value.name || !newTab.value.baseUrl) {
		toast.error('Please fill in all required fields')
		return
	}

	try {
		const url = showEditTabModal.value ? `http://localhost:8080/api/v1/scraper/tabs/${editingTab.value.id}` : 'http://localhost:8080/api/v1/scraper/tabs'

		const method = showEditTabModal.value ? 'PUT' : 'POST'

		const response = await fetch(url, {
			method,
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(newTab.value),
		})

		const data = await response.json()

		if (data.success) {
			toast.success(`Tab ${showEditTabModal.value ? 'updated' : 'created'} successfully`)
			closeTabModal()
			await loadScraperTabs()
		} else {
			toast.error(data.message || `Failed to ${showEditTabModal.value ? 'update' : 'create'} tab`)
		}
	} catch (error) {
		console.error('Error saving tab:', error)
		toast.error('Failed to save tab: ' + error.message)
	}
}

// ── Per-tab selection helpers ─────────────────────────────────────────────────

const toggleTabSelectionMode = (tabId) => {
	customTabSelectionMode.value[tabId] = !customTabSelectionMode.value[tabId]
	if (!customTabSelectionMode.value[tabId]) {
		customTabSelectedVideos.value[tabId] = new Set()
	}
}

const toggleTabVideoSelection = (tabId, videoId) => {
	const current = customTabSelectedVideos.value[tabId] || new Set()
	const next = new Set(current)
	if (next.has(videoId)) next.delete(videoId)
	else next.add(videoId)
	customTabSelectedVideos.value[tabId] = next
}

const selectAllTabVideos = (tabId) => {
	const videos = customTabVideos.value[tabId] || []
	customTabSelectedVideos.value[tabId] = new Set(videos.map((v) => v.id))
}

const sendTabSelectedToJD = async (tab) => {
	const tabId = tab.id
	const selectedIds = customTabSelectedVideos.value[tabId]
	if (!selectedIds || selectedIds.size === 0) {
		toast.warning('No videos selected')
		return
	}

	const allVideos = customTabVideos.value[tabId] || []
	const links = allVideos
		.filter((v) => selectedIds.has(v.id) && (v.source_url || v.url))
		.map((v) => ({
			url: v.source_url || v.url,
			packageName: v.title || '',
			downloadPath: `/output/${tab.name}`,
			videoId: v.id,
		}))

	if (links.length === 0) {
		toast.warning('No valid URLs in selection')
		return
	}

	customTabJDSending.value[tabId] = true
	try {
		const result = await jdownloaderAPI.sendLinks(links)
		if (result?.successCount > 0) {
			toast.success(`Sent ${result.successCount} link(s) to JDownloader`)
			customTabSelectedVideos.value[tabId] = new Set()
			customTabSelectionMode.value[tabId] = false
		} else {
			toast.error(result?.results?.[0]?.message || 'Failed to send links')
		}
	} catch (err) {
		console.error('Bulk JD send error:', err)
		toast.error('Failed to send to JDownloader')
	} finally {
		customTabJDSending.value[tabId] = false
	}
}

const deleteTabSelectedVideos = async (tab) => {
	const tabId = tab.id
	const selectedIds = customTabSelectedVideos.value[tabId]
	if (!selectedIds || selectedIds.size === 0) return

	const ids = Array.from(selectedIds)
	if (!confirm(`Delete ${ids.length} selected video(s)? This cannot be undone.`)) return

	try {
		await scrapersAPI.bulkDelete(ids)
		toast.success(`Deleted ${ids.length} video(s)`)
		customTabSelectedVideos.value[tabId] = new Set()
		customTabSelectionMode.value[tabId] = false
		await loadVideosForTab(tabId)
	} catch (err) {
		console.error('Error deleting selected videos:', err)
		toast.error('Failed to delete videos')
	}
}

const clearCustomTabVideos = async (tab) => {
	if (!confirm(`Delete ALL scraped videos for "${tab.name}"? This cannot be undone.`)) return
	try {
		const source = new URL(tab.baseUrl).host
		await deleteVideosBySource(source)
		customTabVideos.value[tab.id] = []
		toast.success(`Deleted all videos for ${tab.name}`)
	} catch (err) {
		console.error('Failed to delete videos:', err)
		toast.error('Failed to delete videos')
	}
}

const startCustomTabScrape = async (tab) => {
	const url = customTabUrls.value[tab.id]

	if (!url) {
		toast.error('Please enter a URL to scrape')
		return
	}

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/jobs/start', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				tabId: tab.id,
				url: url,
			}),
		})

		const data = await response.json()

		if (data.success) {
			customTabJobs.value[tab.id] = data.data
			toast.success('Scraping job started! It will run in the background.')

			// Poll for progress
			pollJobProgress(tab.id, data.data.id)
		} else {
			toast.error(data.message || 'Failed to start scraping job')
		}
	} catch (error) {
		console.error('Error starting scrape:', error)
		toast.error('Failed to start scraping: ' + error.message)
	}
}

const loadTabJobProgress = async (tabId) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/jobs?tabId=${tabId}`)
		const data = await response.json()

		if (data.success && data.data && data.data.length > 0) {
			// Get the most recent job
			const latestJob = data.data[0]

			// Only show job status if it's actually running or pending
			// Don't show completed/failed jobs to avoid confusion
			if (latestJob.status === 'running' || latestJob.status === 'pending') {
				customTabJobs.value[tabId] = latestJob
				// Continue polling
				const timeoutId = setTimeout(() => loadTabJobProgress(tabId), 3000)
				activeTimeouts.value.push(timeoutId)
			} else {
				// Clear the job status if it's completed/failed
				customTabJobs.value[tabId] = null
			}
		} else {
			// No jobs found, clear status
			customTabJobs.value[tabId] = null
		}
	} catch (error) {
		console.error('Error loading job progress:', error)
	}
}

const pollJobProgress = async (tabId, jobId) => {
	const checkProgress = async () => {
		try {
			const response = await fetch(`http://localhost:8080/api/v1/scraper/jobs/${jobId}`)
			const data = await response.json()

			if (data.success && data.data) {
				customTabJobs.value[tabId] = data.data

				// Continue polling if still running
				if (data.data.status === 'running' || data.data.status === 'pending') {
					const timeoutId = setTimeout(checkProgress, 3000)
					activeTimeouts.value.push(timeoutId)
				} else if (data.data.status === 'completed') {
					toast.success(`Scraping completed! Found ${data.data.videosFound} videos.`)
					loadVideosForTab(tabId)
				} else if (data.data.status === 'failed') {
					toast.error('Scraping job failed. Check the error messages for details.')
				}
			}
		} catch (error) {
			console.error('Error polling job progress:', error)
		}
	}

	checkProgress()
}

// eslint-disable-next-line no-unused-vars
const cancelJob = async (job) => {
	if (!confirm('Are you sure you want to cancel this scraping job?')) {
		return
	}

	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/jobs/${job.id}/cancel`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Job cancelled successfully')
			// Clear the job from display
			customTabJobs.value[job.tabId] = null
		} else {
			toast.error(data.message || 'Failed to cancel job')
		}
	} catch (error) {
		console.error('Error cancelling job:', error)
		toast.error('Failed to cancel job: ' + error.message)
	}
}

// Maps UI sort value → { sort_by, sort_order } for backend
const tabSortMap = {
	newest:       { sort_by: 'discovered_at', sort_order: 'desc' },
	oldest:       { sort_by: 'discovered_at', sort_order: 'asc' },
	'views-desc': { sort_by: 'view_count',    sort_order: 'desc' },
	'views-asc':  { sort_by: 'view_count',    sort_order: 'asc' },
	'duration-desc': { sort_by: 'duration',   sort_order: 'desc' },
	'duration-asc':  { sort_by: 'duration',   sort_order: 'asc' },
	'title-asc':  { sort_by: 'title',         sort_order: 'asc' },
	'title-desc': { sort_by: 'title',         sort_order: 'desc' },
	'likes-desc': { sort_by: 'like_count',    sort_order: 'desc' },
}

const loadVideosForTab = async (tabId) => {
	customTabLoading.value[tabId] = true
	try {
		const tab = scraperTabs.value.find((t) => t.id === tabId)
		if (!tab) return

		const source = new URL(tab.baseUrl).host
		const sort = videoSort.value[tabId] || 'newest'
		const { sort_by, sort_order } = tabSortMap[sort] || tabSortMap.newest

		const params = new URLSearchParams({
			source,
			page:       String(videoPage.value[tabId] || 1),
			limit:      String(videosPerPage.value),
			sort_by,
			sort_order,
		})

		const search = (videoSearch.value[tabId] || '').trim()
		if (search) params.set('search', search)

		const duration = videoDurationFilter.value[tabId] || ''
		if (duration) params.set('duration_filter', duration)

		if (videoOnlyDownloaded.value[tabId]) params.set('downloaded_only', 'true')

		const author = (videoAuthorFilter.value[tabId] || '').trim()
		if (author) params.set('author', author)

		const minViews = videoMinViews.value[tabId] || 0
		if (minViews > 0) params.set('min_views', String(minViews))

		const response = await fetch(`http://localhost:8080/api/v1/scraper/videos?${params}`)
		const data = await response.json()

		if (data.success && data.data) {
			const result = data.data
			customTabVideos.value[tabId] = result.videos || []
			tabVideoTotal.value[tabId] = result.total || 0
			tabVideoTotalPages.value[tabId] = result.totalPages || 1
		}
	} catch (error) {
		console.error('Error loading videos for tab:', error)
	} finally {
		customTabLoading.value[tabId] = false
	}
}

// Called by filter dropdowns / checkboxes — resets to page 1 then reloads
const onTabFilterChange = (tabId) => {
	videoPage.value[tabId] = 1
	customTabSelectedVideos.value[tabId] = new Set()
	loadVideosForTab(tabId)
}

// Called by search/author inputs — debounced
const onTabSearchInput = (tabId) => {
	clearTimeout(tabSearchTimeouts[tabId])
	tabSearchTimeouts[tabId] = setTimeout(() => {
		videoPage.value[tabId] = 1
		loadVideosForTab(tabId)
	}, 400)
}

// Returns number of active (non-default) advanced filters for a tab
const activeTabFilterCount = (tabId) => {
	let n = 0
	if (videoDurationFilter.value[tabId]) n++
	if (videoOnlyDownloaded.value[tabId]) n++
	if (videoAuthorFilter.value[tabId]) n++
	if (videoMinViews.value[tabId] > 0) n++
	return n
}

// Video filtering, sorting and pagination

const changeVideoPage = (tabId, direction) => {
	const currentPage = videoPage.value[tabId] || 1
	const totalPages = tabVideoTotalPages.value[tabId] || 1
	const newPage = currentPage + direction

	if (newPage >= 1 && newPage <= totalPages) {
		videoPage.value[tabId] = newPage
		loadVideosForTab(tabId)
		window.scrollTo({ top: 0, behavior: 'smooth' })
	}
}

const changeVideosPerPage = (newSize, tabId = null) => {
	videosPerPage.value = parseInt(newSize) || 24
	localStorage.setItem('scraper-page-size', String(videosPerPage.value))
	if (tabId != null) {
		videoPage.value[tabId] = 1
		loadVideosForTab(tabId)
	} else {
		videoPage.value = {}
	}
}

const playScrapedVideo = (video, tab = null) => {
	modalVideo.value = video
	modalTab.value = tab
}

const onVideoDownloaded = (videoId) => {
	// Mark the video as downloaded in all local lists so the badge appears immediately
	const now = new Date().toISOString()
	const markInList = (list) => {
		if (!list) return
		const v = list.find((v) => v.id === videoId)
		if (v) v.downloaded_at = now
	}
	markInList(customTabVideos.value)
}

const onVideoUnmarked = (videoId) => {
	// Clear downloaded_at from all local lists
	const clearInList = (list) => {
		if (!list) return
		const v = list.find((v) => v.id === videoId)
		if (v) v.downloaded_at = null
	}
	clearInList(customTabVideos.value)
}

// Lifecycle
onMounted(() => {
	loadScraperTabs()
	checkJDownloaderStatus() // Check JDownloader availability
})

// Cleanup all active polling timeouts when component unmounts
onBeforeUnmount(() => {
	// Clear all active timeouts to prevent memory leaks
	activeTimeouts.value.forEach((timeoutId) => {
		clearTimeout(timeoutId)
	})
	activeTimeouts.value = []
})
</script>

<style src="@/styles/pages/scraper_page.css"></style>
