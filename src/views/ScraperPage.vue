<template>
	<div class="scraper-page">
		<div class="container-fluid py-4">
			<!-- Page Header -->
			<div class="page-header mb-4">
				<div class="d-flex justify-content-between align-items-center">
					<div>
						<h1>
							<font-awesome-icon :icon="['fas', 'spider']" class="me-3" />
							Web Scraper
						</h1>
						<p class="text-muted mb-0">Scrape content from simpcity.is and other sources</p>
					</div>
					<div class="d-flex gap-2">
						<button class="btn btn-outline-light" @click="showAuthModal = true">
							<font-awesome-icon :icon="['fas', 'key']" class="me-2" />
							{{ sessionCookieSet ? 'Auth: Active' : 'Set Auth' }}
						</button>
						<button class="btn btn-success" @click="showForumScrapeModal = true">
							<font-awesome-icon :icon="['fas', 'globe']" class="me-2" />
							Scrape Entire Forum
						</button>
						<button class="btn btn-warning" @click="autoLinkPerformers">
							<font-awesome-icon :icon="['fas', 'link']" class="me-2" />
							Auto-Link Performers
						</button>
						<button class="btn btn-info" @click="checkLinkStatuses">
							<font-awesome-icon :icon="['fas', 'check-circle']" class="me-2" />
							Check Link Status
						</button>
						<button class="btn btn-primary" @click="showScrapeModal = true">
							<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
							Single Thread
						</button>
					</div>
				</div>
			</div>

			<!-- Statistics Cards -->
			<div class="row g-3 mb-4">
				<div class="col-md-3">
					<StatCard :value="stats.total_threads || 0" label="Threads Scraped" :icon="['fas', 'comments']" icon-class="threads" />
				</div>
				<div class="col-md-3">
					<StatCard :value="stats.total_posts || 0" label="Posts Collected" :icon="['fas', 'comment-dots']" icon-class="posts" />
				</div>
				<div class="col-md-3">
					<StatCard :value="stats.total_download_links || 0" label="Download Links" :icon="['fas', 'link']" icon-class="links" />
				</div>
				<div class="col-md-3">
					<StatCard :value="stats.active_links || 0" label="Active Links" :icon="['fas', 'check-circle']" icon-class="active" />
				</div>
			</div>

			<!-- Provider Breakdown -->
			<div v-if="stats.provider_breakdown" class="card mb-4">
				<div class="card-header">
					<h5 class="mb-0">
						<font-awesome-icon :icon="['fas', 'chart-pie']" class="me-2" />
						Provider Breakdown
					</h5>
				</div>
				<div class="card-body">
					<div class="provider-tags">
						<span v-for="(count, provider) in stats.provider_breakdown" :key="provider" class="provider-tag">
							<font-awesome-icon :icon="['fas', 'server']" class="me-1" />
							{{ provider }}: <strong>{{ count }}</strong>
						</span>
					</div>
				</div>
			</div>

			<!-- Search and Filters -->
			<div class="filters-section mb-4">
				<div class="row g-3">
					<div class="col-md-12">
						<SearchBox v-model="searchQuery" placeholder="Search threads by title or author..." @input="searchThreads" />
					</div>
				</div>
				<div class="row g-3 mt-2">
					<div class="col-md-3">
						<label class="filter-label">Sort By</label>
						<select v-model="sortBy" @change="handleSortChange" class="form-select">
							<option value="date_desc">Date (Newest First)</option>
							<option value="date_asc">Date (Oldest First)</option>
							<option value="title_asc">Title (A-Z)</option>
							<option value="title_desc">Title (Z-A)</option>
							<option value="views_desc">Views (High to Low)</option>
							<option value="views_asc">Views (Low to High)</option>
							<option value="replies_desc">Replies (Most First)</option>
							<option value="downloads_desc">Downloads (Most First)</option>
						</select>
					</div>
					<div class="col-md-3">
						<label class="filter-label">Provider</label>
						<select v-model="filterProvider" @change="handleFilterChange" class="form-select">
							<option value="">All Providers</option>
							<option v-for="(count, provider) in stats.provider_breakdown" :key="provider" :value="provider">
								{{ provider }} ({{ count }})
							</option>
						</select>
					</div>
					<div class="col-md-3">
						<label class="filter-label">Content</label>
						<select v-model="filterContent" @change="handleFilterChange" class="form-select">
							<option value="">All Threads</option>
							<option value="has_downloads">Has Downloads</option>
							<option value="no_downloads">No Downloads</option>
						</select>
					</div>
					<div class="col-md-3">
						<label class="filter-label">View Mode</label>
						<select v-model="viewMode" class="form-select">
							<option value="grid">Grid View</option>
							<option value="list">List View</option>
							<option value="compact">Compact View</option>
						</select>
					</div>
				</div>
			</div>

			<!-- Bulk Action Toolbar -->
			<div v-if="threads.length > 0" class="bulk-actions-toolbar mb-4">
				<div class="d-flex justify-content-between align-items-center">
					<div class="d-flex gap-2 align-items-center">
						<button class="btn btn-sm" :class="selectionMode ? 'btn-primary' : 'btn-outline-primary'" @click="toggleSelectionMode">
							<font-awesome-icon :icon="['fas', selectionMode ? 'times' : 'check-square']" class="me-2" />
							{{ selectionMode ? 'Cancel Selection' : 'Select Threads' }}
						</button>
						<div v-if="selectionMode" class="d-flex gap-2">
							<button class="btn btn-sm btn-outline-secondary" @click="allSelectedOnPage ? deselectAllThreads() : selectAllThreads()">
								<font-awesome-icon :icon="['fas', allSelectedOnPage ? 'square' : 'check-square']" class="me-2" />
								{{ allSelectedOnPage ? 'Deselect All' : 'Select All' }}
							</button>
							<span v-if="selectedCount > 0" class="badge bg-primary align-self-center">{{ selectedCount }} selected</span>
						</div>
					</div>
					<div v-if="selectionMode || threads.length > 0" class="d-flex gap-2">
						<button v-if="selectionMode && selectedCount > 0" class="btn btn-sm btn-info" @click="bulkVerifyLinks" :disabled="bulkOperationInProgress">
							<font-awesome-icon :icon="['fas', bulkOperationInProgress ? 'spinner' : 'check-double']" :spin="bulkOperationInProgress" class="me-2" />
							Verify Selected
						</button>
						<button v-if="selectionMode && selectedCount > 0" class="btn btn-sm btn-warning" @click="bulkSendToJDownloader" :disabled="bulkOperationInProgress">
							<font-awesome-icon :icon="['fas', bulkOperationInProgress ? 'spinner' : 'cloud-download-alt']" :spin="bulkOperationInProgress" class="me-2" />
							Send to JD
						</button>
						<button v-if="selectionMode && selectedCount > 0" class="btn btn-sm btn-danger" @click="deleteSelectedThreads">
							<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
							Delete Selected
						</button>
						<button class="btn btn-sm btn-outline-danger" @click="deleteAllThreads">
							<font-awesome-icon :icon="['fas', 'trash-alt']" class="me-2" />
							Delete All
						</button>
					</div>
				</div>
			</div>

			<!-- Loading State -->
			<LoadingState v-if="loading" show-text loading-text="Loading scraped threads..." />

			<!-- Empty State -->
			<EmptyState v-else-if="threads.length === 0 && !loading" :icon="['fas', 'inbox']" icon-size="4x" title="No Threads Found" message="Start scraping to collect thread data">
				<template #actions>
					<button class="btn btn-primary mt-3" @click="showScrapeModal = true">
						<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
						Start Scraping
					</button>
				</template>
			</EmptyState>

			<!-- Threads Grid View -->
			<div v-else-if="viewMode === 'grid'" class="threads-grid">
				<div v-for="thread in threads" :key="thread.id" class="thread-card" :class="{ 'selected': isThreadSelected(thread.id) }" @click="openThread(thread)">
					<div v-if="selectionMode" class="thread-checkbox" @click="toggleThreadSelection(thread.id, $event)">
						<input type="checkbox" :checked="isThreadSelected(thread.id)" @click.stop>
					</div>
					<div v-if="thread.metadata?.thumbnail_url || thread.metadata?.thumbnail_urls?.length" class="thread-thumbnail">
						<img
							:src="thread.metadata.thumbnail_url"
							:data-fallback-urls="JSON.stringify(thread.metadata.thumbnail_urls || [])"
							alt="Thread thumbnail"
							@error="handleThumbnailError" />
					</div>
					<div v-else class="thread-thumbnail placeholder">
						<font-awesome-icon :icon="['fas', 'image']" size="3x" />
					</div>
					<div class="thread-content">
						<h5 class="thread-title">{{ thread.title }}</h5>
						<div class="thread-meta">
							<span class="meta-item">
								<font-awesome-icon :icon="['fas', 'user']" class="me-1" />
								{{ thread.author }}
							</span>
							<span class="meta-item">
								<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
								{{ thread.category || 'General' }}
							</span>
						</div>
						<div class="thread-stats">
							<span class="stat-badge">
								<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
								{{ thread.view_count }}
							</span>
							<span class="stat-badge">
								<font-awesome-icon :icon="['fas', 'comment']" class="me-1" />
								{{ thread.post_count }}
							</span>
							<span class="stat-badge">
								<font-awesome-icon :icon="['fas', 'download']" class="me-1" />
								{{ thread.download_count }}
							</span>
						</div>
						<div v-if="thread.metadata?.tags?.length" class="thread-tags">
							<span v-for="tag in thread.metadata.tags.slice(0, 3)" :key="tag" class="thread-tag">
								{{ tag }}
							</span>
						</div>
					</div>
				</div>
			</div>

			<!-- Threads List View -->
			<div v-else-if="viewMode === 'list'" class="threads-list">
				<div v-for="thread in threads" :key="thread.id" class="thread-list-item" @click="openThread(thread)">
					<div class="thread-list-content">
						<h5 class="thread-title">{{ thread.title }}</h5>
						<div class="thread-info">
							<span class="info-item">
								<font-awesome-icon :icon="['fas', 'user']" class="me-1" />
								{{ thread.author }}
							</span>
							<span class="info-item">
								<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
								{{ thread.category }}
							</span>
							<span class="info-item">
								<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
								{{ thread.view_count }} views
							</span>
							<span class="info-item">
								<font-awesome-icon :icon="['fas', 'comment']" class="me-1" />
								{{ thread.post_count }} posts
							</span>
							<span class="info-item">
								<font-awesome-icon :icon="['fas', 'download']" class="me-1" />
								{{ thread.download_count }} links
							</span>
						</div>
					</div>
					<div class="thread-list-actions">
						<button class="btn btn-sm btn-outline-primary" @click.stop="rescrapeThread(thread)">
							<font-awesome-icon :icon="['fas', 'sync']" />
						</button>
						<button class="btn btn-sm btn-outline-secondary" @click.stop="openThreadURL(thread.url)">
							<font-awesome-icon :icon="['fas', 'external-link-alt']" />
						</button>
					</div>
				</div>
			</div>

			<!-- Pagination -->
			<div v-if="totalPages > 1" class="pagination-container">
				<nav>
					<ul class="pagination justify-content-center">
						<li class="page-item" :class="{ disabled: currentPage === 1 }">
							<button class="page-link" @click="goToPage(currentPage - 1)">Previous</button>
						</li>
						<li v-for="page in visiblePages" :key="page" class="page-item" :class="{ active: page === currentPage }">
							<button class="page-link" @click="goToPage(page)">{{ page }}</button>
						</li>
						<li class="page-item" :class="{ disabled: currentPage === totalPages }">
							<button class="page-link" @click="goToPage(currentPage + 1)">Next</button>
						</li>
					</ul>
				</nav>
			</div>
		</div>

		<!-- Authentication Modal -->
		<div v-if="showAuthModal" class="modal-overlay" @click="showAuthModal = false">
			<div class="modal-dialog" @click.stop>
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							<font-awesome-icon :icon="['fas', 'key']" class="me-2" />
							Set Authentication Cookie
						</h5>
						<button type="button" class="btn-close" @click="showAuthModal = false"></button>
					</div>
					<div class="modal-body">
						<div class="mb-3">
							<label class="form-label">Session Cookie</label>
							<textarea
								v-model="authCookie"
								class="form-control"
								rows="4"
								placeholder="Paste your session cookie here..."
							></textarea>
							<small class="form-text text-muted">
								Get your cookie from browser Dev Tools (F12 → Application → Cookies → simpcity.is)
							</small>
						</div>
						<div class="alert alert-info">
							<font-awesome-icon :icon="['fas', 'info-circle']" class="me-2" />
							<strong>Required cookies:</strong> ogaddgmetaprof_csrf, ogaddgmetaprof_session, ogaddgmetaprof_user,
							cucksed, cucksid
						</div>
						<div v-if="sessionCookieSet" class="alert alert-success">
							<font-awesome-icon :icon="['fas', 'check-circle']" class="me-2" />
							Authentication is currently <strong>active</strong>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="showAuthModal = false">Cancel</button>
						<button type="button" class="btn btn-primary" @click="setAuthCookie" :disabled="!authCookie">
							<font-awesome-icon :icon="['fas', 'save']" class="me-2" />
							Save Cookie
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Scrape Modal -->
		<div v-if="showScrapeModal" class="modal-overlay" @click="showScrapeModal = false">
			<div class="modal-dialog" @click.stop>
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							<font-awesome-icon :icon="['fas', 'spider']" class="me-2" />
							New Scrape Job
						</h5>
						<button type="button" class="btn-close" @click="showScrapeModal = false"></button>
					</div>
					<div class="modal-body">
						<div class="mb-3">
							<label class="form-label">Thread URL</label>
							<input
								v-model="scrapeURL"
								type="text"
								class="form-control"
								placeholder="https://simpcity.is/threads/thread-name.123456/"
							/>
							<small class="form-text text-muted">Enter the full URL of the thread you want to scrape</small>
						</div>
						<div class="alert alert-info">
							<font-awesome-icon :icon="['fas', 'info-circle']" class="me-2" />
							Scraping will run in the background. Check the Tasks page to monitor progress.
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="showScrapeModal = false">Cancel</button>
						<button type="button" class="btn btn-primary" @click="startScrape" :disabled="!scrapeURL">
							<font-awesome-icon :icon="['fas', 'play']" class="me-2" />
							Start Scraping
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Thread Detail Modal -->
		<div v-if="selectedThread" class="modal-overlay" @click="selectedThread = null">
			<div class="modal-dialog modal-lg" @click.stop>
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">{{ selectedThread.title }}</h5>
						<button type="button" class="btn-close" @click="selectedThread = null"></button>
					</div>
					<div class="modal-body">
						<div class="thread-detail">
							<div class="detail-row">
								<strong>Author:</strong> {{ selectedThread.author }}
							</div>
							<div class="detail-row">
								<strong>Category:</strong> {{ selectedThread.category }}
							</div>
							<div class="detail-row">
								<strong>URL:</strong>
								<a :href="selectedThread.url" target="_blank">{{ selectedThread.url }}</a>
							</div>
							<div class="detail-row">
								<strong>Statistics:</strong>
								<div class="stats-inline">
									<span>{{ selectedThread.view_count }} views</span>
									<span>{{ selectedThread.post_count }} posts</span>
									<span>{{ selectedThread.download_count }} download links</span>
								</div>
							</div>
							<div v-if="selectedThread.metadata?.performer_names?.length" class="detail-row">
								<strong>Detected Performers:</strong>
								<div class="tags-container">
									<span v-for="performer in selectedThread.metadata.performer_names" :key="performer" class="tag-badge">
										{{ performer }}
									</span>
								</div>
							</div>
							<div v-if="selectedThread.metadata?.studio_names?.length" class="detail-row">
								<strong>Detected Studios:</strong>
								<div class="tags-container">
									<span v-for="studio in selectedThread.metadata.studio_names" :key="studio" class="tag-badge">
										{{ studio }}
									</span>
								</div>
							</div>
							<div v-if="selectedThread.metadata?.tags?.length" class="detail-row">
								<strong>Tags:</strong>
								<div class="tags-container">
									<span v-for="tag in selectedThread.metadata.tags" :key="tag" class="tag-badge">
										{{ tag }}
									</span>
								</div>
							</div>
							<div class="detail-row">
								<strong>Last Scraped:</strong> {{ formatDate(selectedThread.last_scraped_at) }}
							</div>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="openThreadURL(selectedThread.url)">
							<font-awesome-icon :icon="['fas', 'external-link-alt']" class="me-2" />
							Open in Browser
						</button>
						<button type="button" class="btn btn-primary" @click="rescrapeThread(selectedThread)">
							<font-awesome-icon :icon="['fas', 'sync']" class="me-2" />
							Rescrape
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Forum Scrape Modal -->
		<div v-if="showForumScrapeModal" class="modal-overlay" @click="showForumScrapeModal = false">
			<div class="modal-dialog" @click.stop>
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							<font-awesome-icon :icon="['fas', 'globe']" class="me-2" />
							Scrape Entire Forum
						</h5>
						<button type="button" class="btn-close" @click="showForumScrapeModal = false"></button>
					</div>
					<div class="modal-body">
						<div class="mb-3">
							<label class="form-label">Forum Category URL</label>
							<input
								v-model="forumURL"
								type="text"
								class="form-control"
								placeholder="https://simpcity.is/forums/xxx-porn.50/"
							/>
							<small class="form-text text-muted">
								Enter the forum category URL (e.g., https://simpcity.is/forums/xxx-porn.50/)
							</small>
						</div>
						<div class="alert alert-warning">
							<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="me-2" />
							<strong>Warning:</strong> This will scrape ALL threads in the forum category. This may take several hours and consume significant bandwidth. The scraper will run in the background.
						</div>
						<div class="alert alert-info">
							<font-awesome-icon :icon="['fas', 'info-circle']" class="me-2" />
							<ul class="mb-0">
								<li>The scraper will automatically handle pagination</li>
								<li>Each thread will be fully scraped with all posts and download links</li>
								<li>Performer names will be extracted from thread titles</li>
								<li>You can monitor progress in the server logs</li>
							</ul>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="showForumScrapeModal = false">Cancel</button>
						<button type="button" class="btn btn-success" @click="startForumScrape" :disabled="!forumURL">
							<font-awesome-icon :icon="['fas', 'play']" class="me-2" />
							Start Forum Scraping
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Delete Selected Confirmation Modal -->
		<DeleteConfirmationModal
			:visible="showDeleteConfirmModal"
			title="Confirm Deletion"
			message="Are you sure you want to delete"
			:itemName="`${selectedCount} selected thread(s)`"
			warningMessage="This action cannot be undone. All posts and download links associated with these threads will also be deleted."
			:confirmText="`Delete ${selectedCount} Thread(s)`"
			:icon="['fas', 'trash']"
			@confirm="confirmDeleteSelected"
			@cancel="showDeleteConfirmModal = false"
		/>

		<!-- Delete All Confirmation Modal -->
		<DeleteConfirmationModal
			:visible="showDeleteAllConfirmModal"
			title="Confirm Delete All"
			message="Are you sure you want to delete"
			:itemName="`ALL ${totalThreads} scraped threads`"
			warningMessage="DANGER: This will permanently delete ALL scraped threads, posts, and download links from the database. This action cannot be undone!"
			confirmText="Yes, Delete Everything"
			:icon="['fas', 'trash-alt']"
			@confirm="confirmDeleteAll"
			@cancel="showDeleteAllConfirmModal = false"
		/>
	</div>
</template>

<script setup>
import { ref, onMounted, computed, getCurrentInstance } from 'vue'
import { useRouter } from 'vue-router'
import { DeleteConfirmationModal, LoadingState, EmptyState, StatCard, SearchBox } from '@/components/shared'
import { useFormatters } from '@/composables/useFormatters'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast
const router = useRouter()
const { formatDateTime } = useFormatters()

// State
const loading = ref(false)
const stats = ref({})
const threads = ref([])
const searchQuery = ref('')
const viewMode = ref('grid')
const currentPage = ref(1)
const totalPages = ref(1)
const totalThreads = ref(0)
const limit = 20

// Filters and sorting
const sortBy = ref('date_desc')
const filterProvider = ref('')
const filterContent = ref('')

const showScrapeModal = ref(false)
const showAuthModal = ref(false)
const showForumScrapeModal = ref(false)
const showDeleteConfirmModal = ref(false)
const showDeleteAllConfirmModal = ref(false)
const scrapeURL = ref('')
const forumURL = ref('https://simpcity.is/forums/xxx-porn.50/')
const authCookie = ref('')
const sessionCookieSet = ref(false)
const selectedThread = ref(null)
const selectedThreadIds = ref(new Set())
const selectionMode = ref(false)

// Computed
const visiblePages = computed(() => {
	const pages = []
	const start = Math.max(1, currentPage.value - 2)
	const end = Math.min(totalPages.value, currentPage.value + 2)

	for (let i = start; i <= end; i++) {
		pages.push(i)
	}
	return pages
})

// Methods
const checkAuthStatus = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/session')
		const data = await response.json()
		if (data.success && data.data) {
			sessionCookieSet.value = data.data.is_set
		}
	} catch (error) {
		console.error('Error checking auth status:', error)
	}
}

const setAuthCookie = async () => {
	if (!authCookie.value) {
		toast.warning('Please enter a cookie')
		return
	}

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/session', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				cookie: authCookie.value
			})
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Authentication cookie set successfully!')
			sessionCookieSet.value = true
			showAuthModal.value = false
			authCookie.value = ''
		} else {
			toast.error(data.message || 'Failed to set cookie')
		}
	} catch (error) {
		console.error('Error setting cookie:', error)
		toast.error('Failed to set authentication cookie')
	}
}

const loadStats = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/stats')
		const data = await response.json()
		if (data.success) {
			stats.value = data.data
		}
	} catch (error) {
		console.error('Error loading stats:', error)
	}
}

const loadThreads = async () => {
	loading.value = true
	try {
		let url = searchQuery.value
			? `http://localhost:8080/api/v1/scraper/threads/search?q=${encodeURIComponent(searchQuery.value)}`
			: `http://localhost:8080/api/v1/scraper/threads?`

		// Add pagination
		url += `page=${currentPage.value}&limit=${limit}`

		// Add sorting
		if (sortBy.value) {
			url += `&sort=${sortBy.value}`
		}

		// Add filters
		if (filterProvider.value) {
			url += `&provider=${encodeURIComponent(filterProvider.value)}`
		}
		if (filterContent.value) {
			url += `&filter=${filterContent.value}`
		}

		const response = await fetch(url)
		const data = await response.json()

		if (data.success) {
			threads.value = data.data || []
			totalPages.value = data.pagination.total_pages
			totalThreads.value = data.pagination.total
		}
	} catch (error) {
		console.error('Error loading threads:', error)
		toast.error('Failed to load threads')
	} finally {
		loading.value = false
	}
}

const searchThreads = () => {
	currentPage.value = 1
	loadThreads()
}

const handleSortChange = () => {
	currentPage.value = 1
	loadThreads()
}

const handleFilterChange = () => {
	currentPage.value = 1
	loadThreads()
}

const goToPage = (page) => {
	if (page >= 1 && page <= totalPages.value) {
		currentPage.value = page
		loadThreads()
	}
}

const startScrape = async () => {
	if (!scrapeURL.value) {
		toast.warning('Please enter a URL')
		return
	}

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/threads/scrape', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				url: scrapeURL.value
			})
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Scraping started! Check the Tasks page for progress.')
			showScrapeModal.value = false
			scrapeURL.value = ''

			// Reload threads after a short delay
			setTimeout(() => {
				loadThreads()
				loadStats()
			}, 2000)
		} else {
			toast.error(data.message || 'Failed to start scraping')
		}
	} catch (error) {
		console.error('Error starting scrape:', error)
		toast.error('Failed to start scraping')
	}
}

const rescrapeThread = async (thread) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/threads/${thread.id}/rescrape`, {
			method: 'POST'
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Rescaping started! Check the Tasks page for progress.')
			selectedThread.value = null

			setTimeout(() => {
				loadThreads()
				loadStats()
			}, 2000)
		} else {
			toast.error(data.message || 'Failed to start rescraping')
		}
	} catch (error) {
		console.error('Error rescraping thread:', error)
		toast.error('Failed to start rescraping')
	}
}

const openThread = (thread) => {
	router.push(`/scraper/${thread.id}`)
}

const openThreadURL = (url) => {
	window.open(url, '_blank')
}

// formatDate replaced with formatDateTime from useFormatters composable
const formatDate = formatDateTime

const startForumScrape = async () => {
	if (!forumURL.value) {
		toast.warning('Please enter a forum URL')
		return
	}

	if (!sessionCookieSet.value) {
		toast.warning('Please set authentication cookie first')
		showForumScrapeModal.value = false
		showAuthModal.value = true
		return
	}

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/forum/scrape-all', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				url: forumURL.value
			})
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Forum scraping started! This will take a while. Check server logs for progress.')
			showForumScrapeModal.value = false

			// Reload threads periodically
			setTimeout(() => {
				loadThreads()
				loadStats()
			}, 5000)
		} else {
			toast.error(data.message || 'Failed to start forum scraping')
		}
	} catch (error) {
		console.error('Error starting forum scrape:', error)
		toast.error('Failed to start forum scraping')
	}
}

const autoLinkPerformers = async () => {
	try {
		toast.info('Auto-linking threads to performers...')

		const response = await fetch('http://localhost:8080/api/v1/scraper/performers/auto-link', {
			method: 'POST'
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Auto-linking started! Check server logs for progress. Performers will be created/linked automatically.')

			// Reload threads after a delay
			setTimeout(() => {
				loadThreads()
				loadStats()
			}, 3000)
		} else {
			toast.error(data.message || 'Failed to start auto-linking')
		}
	} catch (error) {
		console.error('Error auto-linking performers:', error)
		toast.error('Failed to start auto-linking')
	}
}

const checkLinkStatuses = async () => {
	try {
		toast.info('Checking link statuses...')

		const response = await fetch('http://localhost:8080/api/v1/scraper/links/check-status', {
			method: 'POST'
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Link status check started! This will take a while. Check server logs for progress.')
		} else {
			toast.error(data.message || 'Failed to start link check')
		}
	} catch (error) {
		console.error('Error checking link statuses:', error)
		toast.error('Failed to start link status check')
	}
}

// Selection methods
const toggleSelectionMode = () => {
	selectionMode.value = !selectionMode.value
	if (!selectionMode.value) {
		selectedThreadIds.value.clear()
	}
}

const toggleThreadSelection = (threadId, event) => {
	event.stopPropagation()
	if (selectedThreadIds.value.has(threadId)) {
		selectedThreadIds.value.delete(threadId)
	} else {
		selectedThreadIds.value.add(threadId)
	}
	selectedThreadIds.value = new Set(selectedThreadIds.value)
}

const selectAllThreads = () => {
	threads.value.forEach(thread => {
		selectedThreadIds.value.add(thread.id)
	})
	selectedThreadIds.value = new Set(selectedThreadIds.value)
}

const deselectAllThreads = () => {
	selectedThreadIds.value.clear()
	selectedThreadIds.value = new Set(selectedThreadIds.value)
}

const isThreadSelected = (threadId) => {
	return selectedThreadIds.value.has(threadId)
}

const selectedCount = computed(() => selectedThreadIds.value.size)

const allSelectedOnPage = computed(() => {
	if (threads.value.length === 0) return false
	return threads.value.every(thread => selectedThreadIds.value.has(thread.id))
})

// Deletion methods
// Bulk Operations
const bulkOperationInProgress = ref(false)

const bulkVerifyLinks = async () => {
	if (selectedThreadIds.value.size === 0) {
		toast.warning('No threads selected')
		return
	}

	bulkOperationInProgress.value = true
	const threadIds = Array.from(selectedThreadIds.value)
	let completed = 0

	try {
		toast.info(`Starting verification for ${threadIds.length} threads...`)

		// Verify each thread sequentially with progress updates
		for (const threadId of threadIds) {
			try {
				const response = await fetch(`http://localhost:8080/api/v1/scraper/threads/${threadId}/verify-links`, {
					method: 'POST'
				})
				const data = await response.json()

				if (data.success) {
					completed++
					toast.success(`Verified thread ${completed}/${threadIds.length}`, {
						duration: 2000
					})
				}
			} catch (error) {
				console.error(`Failed to verify thread ${threadId}:`, error)
			}
		}

		toast.success(`✅ Completed! Verified ${completed}/${threadIds.length} threads`)
		selectedThreadIds.value.clear()
		selectionMode.value = false
	} catch (error) {
		console.error('Bulk verify error:', error)
		toast.error('Bulk verification failed: ' + error.message)
	} finally {
		bulkOperationInProgress.value = false
	}
}

const bulkSendToJDownloader = async () => {
	if (selectedThreadIds.value.size === 0) {
		toast.warning('No threads selected')
		return
	}

	bulkOperationInProgress.value = true
	const threadIds = Array.from(selectedThreadIds.value)
	let completed = 0
	let totalLinksSent = 0

	try {
		// First check if JDownloader is available
		const jdStatusResponse = await fetch('http://localhost:8080/api/v1/jdownloader/status')
		const jdStatusData = await jdStatusResponse.json()

		if (!jdStatusData.success || !jdStatusData.data.available) {
			toast.error('JDownloader is not running! Please start JDownloader first.')
			return
		}

		toast.info(`Sending ${threadIds.length} threads to JDownloader...`)

		// Send each thread to JDownloader
		for (const threadId of threadIds) {
			try {
				const response = await fetch(`http://localhost:8080/api/v1/jdownloader/threads/${threadId}/send`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						only_active: true,
						only_pending: true,
						auto_start: false
					})
				})
				const data = await response.json()

				if (data.success) {
					completed++
					totalLinksSent += data.data.links_sent || 0
					toast.success(`Sent thread ${completed}/${threadIds.length} (${data.data.links_sent} links)`, {
						duration: 2000
					})
				}
			} catch (error) {
				console.error(`Failed to send thread ${threadId}:`, error)
			}
		}

		toast.success(`🎉 Sent ${totalLinksSent} links from ${completed}/${threadIds.length} threads to JDownloader!`)
		selectedThreadIds.value.clear()
		selectionMode.value = false
	} catch (error) {
		console.error('Bulk JDownloader send error:', error)
		toast.error('Bulk send failed: ' + error.message)
	} finally {
		bulkOperationInProgress.value = false
	}
}

const deleteSelectedThreads = async () => {
	if (selectedThreadIds.value.size === 0) {
		toast.warning('No threads selected')
		return
	}
	showDeleteConfirmModal.value = true
}

const confirmDeleteSelected = async () => {
	const threadIds = Array.from(selectedThreadIds.value)

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/threads', {
			method: 'DELETE',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				thread_ids: threadIds
			})
		})

		const data = await response.json()

		if (data.success) {
			toast.success(`Successfully deleted ${threadIds.length} thread(s)`)
			selectedThreadIds.value.clear()
			selectionMode.value = false
			showDeleteConfirmModal.value = false
			loadThreads()
			loadStats()
		} else {
			toast.error(data.message || 'Failed to delete threads')
		}
	} catch (error) {
		console.error('Error deleting threads:', error)
		toast.error('Failed to delete selected threads')
	}
}

const deleteAllThreads = async () => {
	showDeleteAllConfirmModal.value = true
}

const confirmDeleteAll = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/threads/all', {
			method: 'DELETE'
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Successfully deleted all threads')
			selectedThreadIds.value.clear()
			selectionMode.value = false
			showDeleteAllConfirmModal.value = false
			loadThreads()
			loadStats()
		} else {
			toast.error(data.message || 'Failed to delete all threads')
		}
	} catch (error) {
		console.error('Error deleting all threads:', error)
		toast.error('Failed to delete all threads')
	}
}

// Thumbnail fallback handler
const handleThumbnailError = (event) => {
	const img = event.target
	const fallbackURLs = JSON.parse(img.dataset.fallbackUrls || '[]')

	// Get current src index in fallback array
	const currentIndex = fallbackURLs.indexOf(img.src)

	// Try next fallback URL
	if (currentIndex < fallbackURLs.length - 1) {
		img.src = fallbackURLs[currentIndex + 1]
	} else {
		// All fallbacks failed, hide image and show placeholder
		img.style.display = 'none'
		const placeholder = img.parentElement.nextElementSibling
		if (placeholder && placeholder.classList.contains('placeholder')) {
			placeholder.style.display = 'flex'
		}
	}
}

// Lifecycle
onMounted(() => {
	checkAuthStatus()
	loadStats()
	loadThreads()
})
</script>

<style scoped src="@/styles/pages/scraper_page.css"></style>
