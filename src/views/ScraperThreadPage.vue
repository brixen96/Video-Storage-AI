<template>
	<div class="scraper-thread-page">
		<div>
			<!-- Compact Header -->
			<div class="page-header">
				<router-link to="/scraper" class="back-link">
					<font-awesome-icon :icon="['fas', 'arrow-left']" />
					<span>Back</span>
				</router-link>
				<div class="header-content">
					<div class="header-left">
						<h1>{{ thread?.title || 'Loading...' }}</h1>
						<div class="thread-badges">
							<span class="badge badge-user">
								<font-awesome-icon :icon="['fas', 'user']" />
								{{ thread?.author }}
							</span>
							<span class="badge badge-category">{{ thread?.category }}</span>
							<span class="badge badge-stats">
								<font-awesome-icon :icon="['fas', 'comment-dots']" />
								{{ thread?.post_count || 0 }}
							</span>
							<span class="badge badge-stats">
								<font-awesome-icon :icon="['fas', 'download']" />
								{{ filteredDownloadLinks.length }}
							</span>
						</div>
					</div>
					<div class="header-actions">
						<button class="btn-icon btn-success" @click="createPerformer" :disabled="loading || creatingPerformer" title="Create Performer">
							<font-awesome-icon :icon="['fas', creatingPerformer ? 'spinner' : 'user-plus']" :spin="creatingPerformer" />
						</button>
						<button class="btn-icon" @click="rescrapeThread" :disabled="loading" title="Rescrape">
							<font-awesome-icon :icon="['fas', 'sync']" :spin="loading" />
						</button>
						<button class="btn-icon btn-danger" @click="confirmDelete" :disabled="loading" title="Delete">
							<font-awesome-icon :icon="['fas', 'trash']" />
						</button>
						<a :href="thread?.url" target="_blank" class="btn-icon btn-primary" title="View Original">
							<font-awesome-icon :icon="['fas', 'external-link-alt']" />
						</a>
					</div>
				</div>
			</div>

			<!-- Search and Filters Bar -->
			<div class="filters-bar">
				<SearchBox v-model="searchQuery" placeholder="Search posts, links, or content..." />
				<div class="filter-group">
					<select v-model="filterProvider" class="filter-select">
						<option value="">All Providers</option>
						<option value="gofile">Gofile</option>
						<option value="pixeldrain">Pixeldrain</option>
						<option value="bunkr">Bunkr</option>
						<option value="cyberdrop">Cyberdrop</option>
						<option value="mediafire">Mediafire</option>
						<option value="mega">Mega</option>
					</select>
					<select v-model="filterHasImages" class="filter-select">
						<option value="">All Posts</option>
						<option value="true">With Images</option>
						<option value="false">Without Images</option>
					</select>
					<select v-model="sortBy" class="filter-select">
						<option value="date">Sort by Date</option>
						<option value="links">Sort by Links</option>
						<option value="images">Sort by Images</option>
					</select>
				</div>
			</div>

			<!-- Link Health Stats -->
			<div v-if="linkStats" class="link-stats-bar">
				<div class="stats-group">
					<div class="stat-item stat-total">
						<font-awesome-icon :icon="['fas', 'link']" />
						<span class="stat-value">{{ totalLinks }}</span>
						<span class="stat-label">Total Links</span>
					</div>
					<div class="stat-item stat-active">
						<font-awesome-icon :icon="['fas', 'check-circle']" />
						<span class="stat-value">{{ linkStats.active || 0 }}</span>
						<span class="stat-label">Active</span>
					</div>
					<div class="stat-item stat-dead">
						<font-awesome-icon :icon="['fas', 'times-circle']" />
						<span class="stat-value">{{ linkStats.dead || 0 }}</span>
						<span class="stat-label">Dead</span>
					</div>
					<div class="stat-item stat-expired">
						<font-awesome-icon :icon="['fas', 'exclamation-circle']" />
						<span class="stat-value">{{ linkStats.expired || 0 }}</span>
						<span class="stat-label">Expired</span>
					</div>
					<div v-if="downloadStats" class="stat-item stat-downloaded">
						<font-awesome-icon :icon="['fas', 'cloud-download-alt']" />
						<span class="stat-value">{{ downloadStats.downloaded || 0 }}</span>
						<span class="stat-label">Downloaded ({{ downloadStats.completion_percent || 0 }}%)</span>
					</div>
				</div>
				<div class="link-actions">
					<button class="btn btn-primary" @click="verifyLinks" :disabled="verifying">
						<font-awesome-icon :icon="['fas', verifying ? 'spinner' : 'sync']" :spin="verifying" />
						{{ verifying ? 'Verifying...' : 'Verify Links' }}
					</button>
					<button class="btn btn-jdownloader" @click="openJDModal" :disabled="totalLinks === 0 || sendingToJD" title="Send to JDownloader">
						<font-awesome-icon :icon="['fas', sendingToJD ? 'spinner' : 'cloud-download-alt']" :spin="sendingToJD" />
						{{ sendingToJD ? 'Sending...' : 'Send to JD' }}
					</button>
					<div class="export-controls">
						<select v-model="exportFormat" class="export-format-select">
							<option value="txt">TXT</option>
							<option value="json">JSON</option>
							<option value="csv">CSV</option>
						</select>
						<button class="btn btn-success" @click="exportLinks" :disabled="totalLinks === 0">
							<font-awesome-icon :icon="['fas', 'download']" />
							Export Links
						</button>
					</div>
				</div>
			</div>

			<!-- Loading State -->
			<div v-if="loading" class="loading-state">
				<font-awesome-icon :icon="['fas', 'spinner']" spin size="3x" />
				<p>Loading posts...</p>
			</div>

			<!-- Empty State -->
			<div v-else-if="filteredPosts.length === 0" class="empty-state">
				<font-awesome-icon :icon="['fas', 'inbox']" size="3x" />
				<p v-if="searchQuery || filterProvider || filterHasImages">No posts match your filters</p>
				<p v-else>No posts found</p>
			</div>

			<!-- Posts Grid -->
			<div v-else class="posts-container">
				<div v-for="post in filteredPosts" :key="post.id" class="post-card" :class="getPostCardHealthClass(post.id)">
					<!-- Post Header -->
					<div class="post-header">
						<div class="post-author">
							<div class="author-avatar">
								{{ post.author.charAt(0).toUpperCase() }}
							</div>
							<div class="author-info">
								<div class="author-name">{{ post.author }}</div>
								<div class="post-date">{{ formatDate(post.posted_at) }}</div>
							</div>
						</div>
						<div class="post-meta">
							<span v-if="getPostImages(post).length > 0" class="meta-badge">
								<font-awesome-icon :icon="['fas', 'image']" />
								{{ getPostImages(post).length }}
							</span>
							<span v-if="getPostLinks(post.id).length > 0" class="meta-badge">
								<font-awesome-icon :icon="['fas', 'download']" />
								{{ getPostLinks(post.id).length }}
							</span>
							<span v-if="getPostLinks(post.id).length > 0 && hasVerifiedLinks(post.id)" class="meta-badge" :class="getPostHealthClass(post.id)">
								<font-awesome-icon :icon="getPostHealthIcon(post.id)" />
								{{ getPostHealthSummary(post.id) }}
							</span>
						</div>
					</div>

					<!-- Post Content -->
					<div v-if="post.content" class="post-content" v-html="sanitizeHTML(post.content)"></div>

					<!-- Images and Links Combined -->
					<div v-if="getPostImages(post).length > 0 || getPostLinks(post.id).length > 0" class="post-media">
						<!-- Image Gallery with Download Links -->
						<div v-if="getPostImages(post).length > 0" class="media-grid">
							<div v-for="(image, idx) in getPostImages(post)" :key="idx" class="media-item">
								<div class="image-wrapper" @click="openImage(image.url)">
									<img :src="image.thumbnail_url || image.url" :alt="`Image ${idx + 1}`" />
									<div class="image-overlay">
										<font-awesome-icon :icon="['fas', 'search-plus']" />
									</div>
								</div>
							</div>
						</div>

						<!-- Download Links -->
						<div v-if="getPostLinks(post.id).length > 0" class="download-section">
							<div class="download-header">
								<font-awesome-icon :icon="['fas', 'download']" />
								<span>Download Links ({{ getPostLinks(post.id).length }})</span>
							</div>
							<div class="download-links">
								<div v-for="link in getPostLinks(post.id)" :key="link.id" class="download-link-wrapper" :class="[`status-${link.download_status || 'pending'}`, `health-${link.status || 'unknown'}`]">
									<a :href="link.url" target="_blank" class="download-link" :class="`provider-${link.provider.toLowerCase()}`">
										<div class="link-icon">
											<font-awesome-icon :icon="getProviderIcon(link.provider)" />
										</div>
										<div class="link-info">
											<div class="link-provider">{{ link.provider }}</div>
											<div class="link-url">{{ truncateURL(link.url) }}</div>
										</div>
										<div v-if="link.status && link.status !== 'unknown'" class="link-health-badge" :class="`health-${link.status}`" :title="`Link ${link.status}`">
											<font-awesome-icon :icon="getLinkHealthIcon(link.status)" />
										</div>
									</a>
									<div class="link-actions">
										<button
											v-if="link.download_status !== 'downloaded'"
											@click.stop="markAsDownloaded(link.id)"
											class="action-btn success-btn"
											title="Mark as downloaded"
										>
											<font-awesome-icon :icon="['fas', 'check']" />
										</button>
										<button v-if="link.download_status !== 'failed'" @click.stop="markAsFailed(link.id)" class="action-btn danger-btn" title="Mark as failed">
											<font-awesome-icon :icon="['fas', 'times']" />
										</button>
										<button
											v-if="link.download_status && link.download_status !== 'pending'"
											@click.stop="resetDownloadStatus(link.id)"
											class="action-btn reset-btn"
											title="Reset status"
										>
											<font-awesome-icon :icon="['fas', 'undo']" />
										</button>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Send to JDownloader Modal -->
		<div v-if="showJDModal" class="jd-modal-backdrop" @click.self="showJDModal = false">
			<div class="jd-modal-content">
				<div class="jd-modal-header">
					<h5 class="jd-modal-title">
						<font-awesome-icon :icon="['fas', 'cloud-download-alt']" class="me-2" />
						Send to JDownloader
					</h5>
					<button class="jd-btn-close" @click="showJDModal = false">&times;</button>
				</div>
				<div class="jd-modal-body">
					<div v-if="jdDedupInfo" class="jd-dedup-info">
						<font-awesome-icon :icon="['fas', 'info-circle']" />
						<span>{{ jdDedupInfo }}</span>
					</div>

					<!-- Provider Priority Config -->
					<div class="priority-config-section">
						<button class="priority-toggle-btn" @click="showPriorityConfig = !showPriorityConfig">
							<font-awesome-icon :icon="['fas', 'sort-amount-down']" class="me-1" />
							Provider Priority
							<font-awesome-icon :icon="['fas', showPriorityConfig ? 'chevron-up' : 'chevron-down']" class="ms-auto" />
						</button>
						<div v-if="showPriorityConfig" class="priority-list">
							<div v-for="(provider, idx) in priorityList" :key="provider" class="priority-item">
								<span class="priority-rank">{{ idx + 1 }}</span>
								<span class="priority-name">{{ provider }}</span>
								<div class="priority-actions">
									<button @click="movePriorityUp(provider)" :disabled="idx === 0" class="priority-btn" title="Move up">
										<font-awesome-icon :icon="['fas', 'chevron-up']" />
									</button>
									<button @click="movePriorityDown(provider)" :disabled="idx === priorityList.length - 1" class="priority-btn" title="Move down">
										<font-awesome-icon :icon="['fas', 'chevron-down']" />
									</button>
								</div>
							</div>
							<button class="priority-reset-btn" @click="resetPriority">
								<font-awesome-icon :icon="['fas', 'undo']" class="me-1" />Reset to defaults
							</button>
						</div>
					</div>

					<div class="jd-form-group">
						<label class="jd-form-label">Links ({{ jdLinksCount }} link{{ jdLinksCount !== 1 ? 's' : '' }})</label>
						<textarea v-model="jdLinksText" class="jd-form-control jd-textarea" rows="10" placeholder="No active links found..."></textarea>
					</div>

					<div class="jd-form-group">
						<label class="jd-form-label">Package Name</label>
						<input v-model="jdPackageName" type="text" class="jd-form-control" placeholder="Package name for JDownloader" />
					</div>

					<div class="jd-form-group">
						<label class="jd-form-label">Download Path</label>
						<div class="jd-input-group">
							<span class="jd-input-prefix">/output/</span>
							<input v-model="jdDownloadPath" type="text" class="jd-form-control jd-input-with-prefix" placeholder="subfolder (leave empty for root)" />
						</div>
					</div>

					<div class="jd-form-check">
						<input v-model="jdAutoStart" type="checkbox" class="jd-check-input" id="jdAutoStart" />
						<label class="jd-check-label" for="jdAutoStart">Auto-start downloads</label>
					</div>
				</div>
				<div class="jd-modal-footer">
					<button class="btn btn-secondary" @click="showJDModal = false">Cancel</button>
					<button class="btn btn-jdownloader" @click="confirmSendToJD" :disabled="!jdLinksText.trim() || sendingToJD">
						<font-awesome-icon v-if="sendingToJD" :icon="['fas', 'spinner']" spin class="me-1" />
						<font-awesome-icon v-else :icon="['fas', 'cloud-download-alt']" class="me-1" />
						Send {{ jdLinksCount }} Link{{ jdLinksCount !== 1 ? 's' : '' }}
					</button>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch, getCurrentInstance } from 'vue'
import { useRoute } from 'vue-router'
import DOMPurify from 'dompurify'
import { SearchBox } from '@/components/shared'
import { openInFirefox } from '@/services/api'
import websocketService from '@/services/websocket'

const route = useRoute()
const { proxy } = getCurrentInstance()
const toast = proxy.$toast

// State
const loading = ref(false)
const thread = ref(null)
const posts = ref([])
const downloadLinks = ref([])

// Link verification state
const linkStats = ref(null)
const verifying = ref(false)

// Download tracking state
const downloadStats = ref(null)

// JDownloader state
const sendingToJD = ref(false)
const showJDModal = ref(false)
const jdLinksText = ref('')
const jdPackageName = ref('')
const jdDownloadPath = ref('')
const jdAutoStart = ref(false)
const jdDedupInfo = ref('')

// Performer creation state
const creatingPerformer = ref(false)

// Export state
const exportFormat = ref('txt')

// Filter state
const searchQuery = ref('')
const filterProvider = ref('')
const filterHasImages = ref('')
const sortBy = ref('date')

// Computed
const filteredDownloadLinks = computed(() => {
	let links = downloadLinks.value

	if (filterProvider.value) {
		links = links.filter((link) => link.provider.toLowerCase() === filterProvider.value.toLowerCase())
	}

	return links
})

const filteredPosts = computed(() => {
	let filtered = posts.value

	// Search filter
	if (searchQuery.value) {
		const query = searchQuery.value.toLowerCase()
		filtered = filtered.filter((post) => {
			const contentMatch = post.content?.toLowerCase().includes(query)
			const authorMatch = post.author?.toLowerCase().includes(query)
			const linksMatch = getPostLinks(post.id).some((link) => link.url.toLowerCase().includes(query) || link.provider.toLowerCase().includes(query))
			return contentMatch || authorMatch || linksMatch
		})
	}

	// Provider filter
	if (filterProvider.value) {
		filtered = filtered.filter((post) => {
			const postLinks = getPostLinks(post.id)
			return postLinks.some((link) => link.provider.toLowerCase() === filterProvider.value.toLowerCase())
		})
	}

	// Image filter
	if (filterHasImages.value) {
		const hasImages = filterHasImages.value === 'true'
		filtered = filtered.filter((post) => {
			const images = getPostImages(post)
			return hasImages ? images.length > 0 : images.length === 0
		})
	}

	// Sort
	if (sortBy.value === 'links') {
		filtered = [...filtered].sort((a, b) => {
			return getPostLinks(b.id).length - getPostLinks(a.id).length
		})
	} else if (sortBy.value === 'images') {
		filtered = [...filtered].sort((a, b) => {
			return getPostImages(b).length - getPostImages(a).length
		})
	} else {
		// Default: sort by date (newest first)
		filtered = [...filtered].sort((a, b) => {
			return new Date(b.posted_at) - new Date(a.posted_at)
		})
	}

	return filtered
})

const totalLinks = computed(() => {
	if (!linkStats.value) return downloadLinks.value.length
	return (linkStats.value.active || 0) + (linkStats.value.dead || 0) + (linkStats.value.expired || 0)
})

// Methods
const loadLinkStats = async () => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/threads/${route.params.id}/link-stats`)
		const data = await response.json()
		if (data.success) {
			linkStats.value = data.data
		}
	} catch (error) {
		console.error('Error loading link stats:', error)
	}
}

const verifyLinks = async () => {
	verifying.value = true
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/threads/${route.params.id}/verify-links`, {
			method: 'POST',
		})
		const data = await response.json()
		if (data.success) {
			toast.success('Link verification started! Results will update automatically when done.')
		} else {
			toast.error(data.message || 'Failed to start verification')
			verifying.value = false
		}
	} catch (error) {
		console.error('Error verifying links:', error)
		toast.error('Failed to verify links: ' + error.message)
		verifying.value = false
	}
}

const exportLinks = async () => {
	try {
		// Build query params
		const params = new URLSearchParams({
			format: exportFormat.value,
			status: 'active', // Only export active links by default
		})

		if (filterProvider.value) {
			params.append('provider', filterProvider.value)
		}

		// Trigger download
		const url = `http://localhost:8080/api/v1/scraper/threads/${route.params.id}/export-links?${params.toString()}`
		window.open(url, '_blank')

		toast.success(`Exporting links as ${exportFormat.value.toUpperCase()}...`)
	} catch (error) {
		console.error('Error exporting links:', error)
		toast.error('Failed to export links: ' + error.message)
	}
}

// Download tracking functions
const loadDownloadStats = async () => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/downloads/threads/${route.params.id}/stats`)
		const data = await response.json()
		if (data.success) {
			downloadStats.value = data.data
		}
	} catch (error) {
		console.error('Error loading download stats:', error)
	}
}

const markAsDownloaded = async (linkId) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/downloads/links/${linkId}/mark-downloaded`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				download_path: '',
				notes: '',
			}),
		})
		const data = await response.json()
		if (data.success) {
			toast.success('Link marked as downloaded!')
			// Update the link in the list
			const link = downloadLinks.value.find((l) => l.id === linkId)
			if (link) link.download_status = 'downloaded'
			// Reload stats
			await loadDownloadStats()
		} else {
			toast.error(data.message || 'Failed to mark as downloaded')
		}
	} catch (error) {
		console.error('Error marking link as downloaded:', error)
		toast.error('Failed to mark as downloaded: ' + error.message)
	}
}

const markAsFailed = async (linkId) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/downloads/links/${linkId}/mark-failed`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ notes: 'User marked as failed' }),
		})
		const data = await response.json()
		if (data.success) {
			toast.success('Link marked as failed')
			// Update the link in the list
			const link = downloadLinks.value.find((l) => l.id === linkId)
			if (link) link.download_status = 'failed'
			// Reload stats
			await loadDownloadStats()
		} else {
			toast.error(data.message || 'Failed to mark as failed')
		}
	} catch (error) {
		console.error('Error marking link as failed:', error)
		toast.error('Failed to mark as failed: ' + error.message)
	}
}

const resetDownloadStatus = async (linkId) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/downloads/links/${linkId}/reset`, {
			method: 'POST',
		})
		const data = await response.json()
		if (data.success) {
			toast.success('Download status reset')
			// Update the link in the list
			const link = downloadLinks.value.find((l) => l.id === linkId)
			if (link) link.download_status = 'pending'
			// Reload stats
			await loadDownloadStats()
		} else {
			toast.error(data.message || 'Failed to reset status')
		}
	} catch (error) {
		console.error('Error resetting download status:', error)
		toast.error('Failed to reset status: ' + error.message)
	}
}


const loadThread = async () => {
	loading.value = true
	try {
		console.log('Loading thread:', route.params.id)
		const response = await fetch(`http://localhost:8080/api/v1/scraper/threads/${route.params.id}`)
		const data = await response.json()
		console.log('Thread data received:', data)

		if (data.success) {
			thread.value = data.data.thread
			posts.value = data.data.posts || []
			downloadLinks.value = data.data.download_links || []
			console.log('Loaded:', posts.value.length, 'posts and', downloadLinks.value.length, 'links')
		} else {
			toast.error(data.message || 'Failed to load thread')
			console.error('Server error:', data.message)
		}
	} catch (error) {
		console.error('Error loading thread:', error)
		toast.error('Failed to load thread: ' + error.message)
	} finally {
		loading.value = false
	}
}

const rescrapeThread = async () => {
	console.log('Rescrape button clicked for thread:', route.params.id)
	loading.value = true
	try {
		const url = `http://localhost:8080/api/v1/scraper/threads/${route.params.id}/rescrape`
		console.log('Sending rescrape request to:', url)
		const response = await fetch(url, {
			method: 'POST',
		})
		const data = await response.json()
		console.log('Rescrape response:', data)

		if (data.success) {
			toast.success('Thread rescraping started. Check Tasks page for progress.')
			setTimeout(loadThread, 3000)
		} else {
			toast.error(data.message || 'Failed to start rescraping')
		}
	} catch (error) {
		console.error('Error rescraping thread:', error)
		toast.error('Failed to rescrape thread: ' + error.message)
	} finally {
		loading.value = false
	}
}

const confirmDelete = async () => {
	if (
		!confirm(
			`Are you sure you want to delete this thread?\n\nThis will permanently delete:\n- Thread: ${thread.value?.title}\n- ${posts.value.length} posts\n- ${downloadLinks.value.length} download links\n\nThis action cannot be undone.`
		)
	) {
		return
	}

	await deleteThread()
}

const deleteThread = async () => {
	loading.value = true
	try {
		const url = `http://localhost:8080/api/v1/scraper/threads/${route.params.id}`
		console.log('Sending delete request to:', url)
		const response = await fetch(url, {
			method: 'DELETE',
		})
		const data = await response.json()
		console.log('Delete response:', data)

		if (data.success) {
			toast.success('Thread deleted successfully')
			setTimeout(() => {
				window.location.href = '/scraper'
			}, 1000)
		} else {
			toast.error(data.message || 'Failed to delete thread')
		}
	} catch (error) {
		console.error('Error deleting thread:', error)
		toast.error('Failed to delete thread: ' + error.message)
	} finally {
		loading.value = false
	}
}

const createPerformer = async () => {
	if (!thread.value?.title) {
		toast.error('Thread title not available')
		return
	}

	// Prompt user for performer name with thread title as default
	const performerName = prompt('Enter performer name:', thread.value.title)

	if (!performerName || performerName.trim() === '') {
		return
	}

	creatingPerformer.value = true
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/performers/create-from-thread', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				thread_id: parseInt(route.params.id),
				name: performerName.trim(),
			}),
		})
		const data = await response.json()

		if (data.success) {
			toast.success(`Performer "${performerName}" created and linked to this thread!`)
		} else {
			toast.error(data.message || 'Failed to create performer')
		}
	} catch (error) {
		console.error('Error creating performer:', error)
		toast.error('Failed to create performer: ' + error.message)
	} finally {
		creatingPerformer.value = false
	}
}

const getPostLinks = (postId) => {
	return downloadLinks.value.filter((link) => link.post_id === postId)
}

const getPostImages = (post) => {
	return post.metadata?.attachments || []
}

const openImage = (url) => {
	openInFirefox(url)
}

const getProviderIcon = (provider) => {
	const icons = {
		gofile: ['fas', 'cloud-download-alt'],
		pixeldrain: ['fas', 'cloud-download-alt'],
		bunkr: ['fas', 'cloud-download-alt'],
		bunkrr: ['fas', 'cloud-download-alt'],
		cyberdrop: ['fas', 'cloud-download-alt'],
		cyberfile: ['fas', 'cloud-download-alt'],
		mediafire: ['fas', 'cloud-download-alt'],
		mega: ['fas', 'cloud-download-alt'],
	}
	return icons[provider.toLowerCase()] || ['fas', 'link']
}

const truncateURL = (url) => {
	if (url.length > 50) {
		return url.substring(0, 47) + '...'
	}
	return url
}

// Link health helpers
const getLinkHealthIcon = (status) => {
	switch (status) {
		case 'active':
			return ['fas', 'check-circle']
		case 'dead':
			return ['fas', 'skull-crossbones']
		case 'expired':
			return ['fas', 'hourglass-end']
		default:
			return ['fas', 'question-circle']
	}
}

const hasVerifiedLinks = (postId) => {
	const links = getPostLinks(postId)
	return links.some((l) => l.status === 'active' || l.status === 'dead' || l.status === 'expired')
}

const getPostHealthSummary = (postId) => {
	const links = getPostLinks(postId)
	if (links.length === 0) return ''
	const active = links.filter((l) => l.status === 'active').length
	return `${active}/${links.length}`
}

const getPostHealthClass = (postId) => {
	const links = getPostLinks(postId)
	if (links.length === 0) return ''
	const active = links.filter((l) => l.status === 'active').length
	const dead = links.filter((l) => l.status === 'dead').length
	if (active === links.length) return 'health-all-active'
	if (dead === links.length) return 'health-all-dead'
	return 'health-mixed'
}

const getPostHealthIcon = (postId) => {
	const links = getPostLinks(postId)
	if (links.length === 0) return ['fas', 'link']
	const active = links.filter((l) => l.status === 'active').length
	if (active === links.length) return ['fas', 'check-circle']
	if (active === 0) return ['fas', 'times-circle']
	return ['fas', 'exclamation-triangle']
}

const getPostCardHealthClass = (postId) => {
	const links = getPostLinks(postId)
	if (links.length === 0) return ''
	if (!hasVerifiedLinks(postId)) return ''
	const active = links.filter((l) => l.status === 'active').length
	if (active === links.length) return 'post-health-active'
	if (active === 0) return 'post-health-dead'
	return 'post-health-mixed'
}

// JDownloader modal - deduplication and sending
// Default priority order (lower number = higher priority, picked first)
const DEFAULT_PROVIDER_PRIORITY = {
	pixeldrain: 1,
	gofile: 2,
	cyberfile: 3,
	bunkr: 4,
	bunkrr: 4,
	cyberdrop: 5,
	mediafire: 6,
	mega: 7,
}

// Load user-configured priority from localStorage, fall back to defaults
const loadProviderPriority = () => {
	try {
		const saved = localStorage.getItem('providerPriority')
		if (saved) {
			const parsed = JSON.parse(saved)
			for (const [provider, priority] of Object.entries(DEFAULT_PROVIDER_PRIORITY)) {
				if (!(provider in parsed)) parsed[provider] = priority
			}
			const sorted = Object.entries(parsed).sort(([, a], [, b]) => a - b)
			const values = sorted.map(([, v]) => v)
			if (new Set(values).size !== values.length) {
				const fixed = {}
				sorted.forEach(([name], idx) => { fixed[name] = idx + 1 })
				localStorage.setItem('providerPriority', JSON.stringify(fixed))
				return fixed
			}
			return parsed
		}
	} catch (e) { /* ignore */ }
	return { ...DEFAULT_PROVIDER_PRIORITY }
}

const providerPriority = ref(loadProviderPriority())
const showPriorityConfig = ref(false)

const savePriority = () => {
	localStorage.setItem('providerPriority', JSON.stringify(providerPriority.value))
}

// Ordered list for the priority config UI (ascending: lower number = higher priority)
// Filter out bunkrr alias - it stays synced with bunkr
const priorityList = computed(() => {
	return Object.entries(providerPriority.value)
		.filter(([name]) => name !== 'bunkrr')
		.sort(([, a], [, b]) => a - b)
		.map(([name]) => name)
})

const movePriorityUp = (provider) => {
	const list = priorityList.value
	const idx = list.indexOf(provider)
	if (idx <= 0) return
	const above = list[idx - 1]
	const temp = providerPriority.value[provider]
	providerPriority.value[provider] = providerPriority.value[above]
	providerPriority.value[above] = temp
	// Keep bunkrr synced with bunkr
	if (provider === 'bunkr' || above === 'bunkr') {
		providerPriority.value['bunkrr'] = providerPriority.value['bunkr']
	}
	savePriority()
}

const movePriorityDown = (provider) => {
	const list = priorityList.value
	const idx = list.indexOf(provider)
	if (idx < 0 || idx >= list.length - 1) return
	const below = list[idx + 1]
	const temp = providerPriority.value[provider]
	providerPriority.value[provider] = providerPriority.value[below]
	providerPriority.value[below] = temp
	if (provider === 'bunkr' || below === 'bunkr') {
		providerPriority.value['bunkrr'] = providerPriority.value['bunkr']
	}
	savePriority()
}

const resetPriority = () => {
	providerPriority.value = { ...DEFAULT_PROVIDER_PRIORITY }
	savePriority()
}

const getProviderPriority = (provider) => {
	return providerPriority.value[provider.toLowerCase()] || 0
}

const jdLinksCount = computed(() => {
	if (!jdLinksText.value.trim()) return 0
	return jdLinksText.value
		.trim()
		.split('\n')
		.filter((l) => l.trim()).length
})

const getDeduplicatedActiveLinks = () => {
	const activeLinks = downloadLinks.value.filter((l) => l.status === 'active')
	const totalActive = activeLinks.length

	// Group by post_id
	const byPost = {}
	for (const link of activeLinks) {
		if (!byPost[link.post_id]) {
			byPost[link.post_id] = []
		}
		byPost[link.post_id].push(link)
	}

	// For each post, pick the highest-priority provider link
	const deduplicated = []
	for (const postId of Object.keys(byPost)) {
		const postLinks = byPost[postId]
		if (postLinks.length === 1) {
			deduplicated.push(postLinks[0])
			continue
		}
		postLinks.sort((a, b) => getProviderPriority(a.provider) - getProviderPriority(b.provider))
		deduplicated.push(postLinks[0])
	}

	return {
		links: deduplicated.map((l) => l.url),
		removedCount: totalActive - deduplicated.length,
		totalActive,
	}
}

const openJDModal = async () => {
	// Check JDownloader availability
	try {
		const statusResponse = await fetch('http://localhost:8080/api/v1/jdownloader/status')
		const statusData = await statusResponse.json()

		if (!statusData.success && !statusData.available && !statusData.connected) {
			toast.error('JDownloader is not running! Please start JDownloader and enable Direct Connection API.')
			return
		}
	} catch (error) {
		toast.error('Cannot connect to JDownloader: ' + error.message)
		return
	}

	// Run deduplication
	const result = getDeduplicatedActiveLinks()

	// Pre-fill modal fields
	jdLinksText.value = result.links.join('\n')
	jdPackageName.value = thread.value?.title || ''
	jdDownloadPath.value = ''
	jdAutoStart.value = false

	// Info about deduplication
	if (result.removedCount > 0) {
		jdDedupInfo.value = `${result.removedCount} duplicate link${result.removedCount !== 1 ? 's' : ''} removed (${result.totalActive} active → ${result.links.length}). Priority: ${priorityList.value.join(' > ')}`
	} else if (result.totalActive > 0) {
		jdDedupInfo.value = `${result.links.length} verified active link${result.links.length !== 1 ? 's' : ''} ready to send.`
	} else {
		jdDedupInfo.value = 'No active (verified) links found. Try verifying links first.'
	}

	showJDModal.value = true
}

const confirmSendToJD = async () => {
	if (!jdLinksText.value.trim()) return

	sendingToJD.value = true
	try {
		const urls = jdLinksText.value
			.trim()
			.split('\n')
			.filter((l) => l.trim())
		const downloadPath = jdDownloadPath.value.trim() ? `/output/${jdDownloadPath.value.trim().replace(/^\/+/, '')}` : ''

		const links = urls.map((url) => ({
			url: url.trim(),
			packageName: jdPackageName.value || undefined,
			downloadPath: downloadPath || undefined,
			autoStart: jdAutoStart.value,
		}))

		const response = await fetch('http://localhost:8080/api/v1/jdownloader/send', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ links }),
		})

		const data = await response.json()

		if (data.successCount > 0) {
			toast.success(`Sent ${data.successCount} link${data.successCount !== 1 ? 's' : ''} to JDownloader!`)
		}
		if (data.failCount > 0) {
			toast.error(`${data.failCount} link${data.failCount !== 1 ? 's' : ''} failed to send`)
		}
		if (!data.successCount && !data.failCount) {
			toast.info('No links were sent')
		}

		showJDModal.value = false
	} catch (error) {
		console.error('Error sending to JDownloader:', error)
		toast.error('Failed to send links: ' + error.message)
	} finally {
		sendingToJD.value = false
	}
}

const sanitizeHTML = (html) => {
	return DOMPurify.sanitize(html, {
		ALLOWED_TAGS: ['p', 'br', 'b', 'i', 'u', 'strong', 'em', 'a', 'ul', 'ol', 'li', 'blockquote', 'code', 'pre'],
		ALLOWED_ATTR: ['href', 'target', 'rel'],
	})
}

const formatDate = (dateString) => {
	if (!dateString) return 'N/A'
	const date = new Date(dateString)
	const now = new Date()
	const diff = now - date
	const days = Math.floor(diff / (1000 * 60 * 60 * 24))

	if (days === 0) {
		const hours = Math.floor(diff / (1000 * 60 * 60))
		if (hours === 0) {
			const minutes = Math.floor(diff / (1000 * 60))
			return `${minutes}m ago`
		}
		return `${hours}h ago`
	} else if (days === 1) {
		return 'Yesterday'
	} else if (days < 7) {
		return `${days}d ago`
	}

	return date.toLocaleDateString('en-US', {
		month: 'short',
		day: 'numeric',
		year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined,
	})
}

let unsubVerification = null
let unsubActivity = null

onMounted(() => {
	loadThread()
	loadLinkStats()
	loadDownloadStats()

	const threadId = parseInt(route.params.id)

	// Auto-update when verification completes
	unsubVerification = websocketService.on('link_verification_complete', (data) => {
		if (data.thread_id !== threadId) return
		verifying.value = false
		toast.success(`Verification complete: ${data.active} active, ${data.dead} dead, ${data.expired} expired (${data.total} total)`)
		loadLinkStats()
		loadThread() // reload link statuses on each card
	})

	// Keep verifying spinner alive while task is in progress
	unsubActivity = websocketService.on('activity_update', (activity) => {
		if (activity.task_type !== 'link_verification') return
		if (activity.details?.thread_id !== threadId) return
		if (activity.status === 'completed' || activity.status === 'failed') {
			verifying.value = false
			if (activity.status === 'completed') {
				loadLinkStats()
				loadThread()
			}
		}
	})
})

onBeforeUnmount(() => {
	if (unsubVerification) unsubVerification()
	if (unsubActivity) unsubActivity()
})

// Watch for route parameter changes (when navigating from one thread to another)
watch(
	() => route.params.id,
	(newId, oldId) => {
		if (newId && newId !== oldId) {
			// Reset state
			thread.value = null
			posts.value = []
			downloadLinks.value = []
			searchQuery.value = ''
			filterProvider.value = ''
			filterHasImages.value = ''
			sortBy.value = 'date'

			// Load new thread
			loadThread()
		}
	}
)
</script>

<style scoped>
.scraper-thread-page {
	min-height: 100vh;
	background: linear-gradient(135deg, #0f0f1e 0%, #1a1a2e 50%, #16213e 100%);
	padding: 1rem;
}

.container-fluid {
	max-width: 1400px;
	margin: 0 auto;
}

/* Header */
.page-header {
	margin-bottom: 1.5rem;
}

.back-link {
	display: inline-flex;
	align-items: center;
	gap: 0.5rem;
	color: rgba(255, 255, 255, 0.6);
	text-decoration: none;
	font-size: 0.9rem;
	margin-bottom: 1rem;
	transition: color 0.2s;
}

.back-link:hover {
	color: #fff;
}

.header-content {
	display: flex;
	justify-content: space-between;
	align-items: flex-start;
	gap: 1rem;
	background: rgba(255, 255, 255, 0.05);
	backdrop-filter: blur(20px);
	border-radius: 16px;
	padding: 1.5rem;
	border: 1px solid rgba(255, 255, 255, 0.1);
}

.header-left {
	flex: 1;
}

.header-left h1 {
	font-size: 1.5rem;
	font-weight: 600;
	margin: 0 0 0.75rem 0;
	color: #fff;
	line-height: 1.3;
}

.thread-badges {
	display: flex;
	flex-wrap: wrap;
	gap: 0.5rem;
}

.badge {
	display: inline-flex;
	align-items: center;
	gap: 0.4rem;
	padding: 0.35rem 0.75rem;
	border-radius: 8px;
	font-size: 0.85rem;
	font-weight: 500;
}

.badge-user {
	background: rgba(74, 171, 247, 0.15);
	color: #4dabf7;
	border: 1px solid rgba(74, 171, 247, 0.3);
}

.badge-category {
	background: rgba(139, 92, 246, 0.15);
	color: #a78bfa;
	border: 1px solid rgba(139, 92, 246, 0.3);
}

.badge-stats {
	background: rgba(255, 255, 255, 0.05);
	color: rgba(255, 255, 255, 0.8);
	border: 1px solid rgba(255, 255, 255, 0.1);
}

.header-actions {
	display: flex;
	gap: 0.5rem;
}

.btn-icon {
	width: 40px;
	height: 40px;
	border-radius: 10px;
	border: 1px solid rgba(255, 255, 255, 0.15);
	background: rgba(255, 255, 255, 0.05);
	color: #fff;
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	transition: all 0.2s;
	text-decoration: none;
}

.btn-icon:hover {
	background: rgba(255, 255, 255, 0.1);
	border-color: rgba(255, 255, 255, 0.3);
	transform: translateY(-2px);
}

.btn-icon.btn-danger {
	border-color: rgba(220, 53, 69, 0.3);
	color: #dc3545;
}

.btn-icon.btn-danger:hover {
	background: rgba(220, 53, 69, 0.1);
	border-color: #dc3545;
}

.btn-icon.btn-primary {
	background: rgba(74, 171, 247, 0.15);
	border-color: rgba(74, 171, 247, 0.3);
	color: #4dabf7;
}

.btn-icon.btn-primary:hover {
	background: rgba(74, 171, 247, 0.25);
}

.btn-icon.btn-success {
	border-color: rgba(81, 207, 102, 0.3);
	background: rgba(81, 207, 102, 0.15);
	color: #51cf66;
}

.btn-icon.btn-success:hover {
	background: rgba(81, 207, 102, 0.25);
	border-color: #51cf66;
}

/* Filters Bar */
.filters-bar {
	display: flex;
	gap: 1rem;
	margin-bottom: 1.5rem;
	flex-wrap: wrap;
}

.search-box {
	flex: 1;
	min-width: 300px;
	position: relative;
}

.search-icon {
	position: absolute;
	left: 1rem;
	top: 50%;
	transform: translateY(-50%);
	color: rgba(255, 255, 255, 0.4);
	pointer-events: none;
}

.search-input {
	width: 100%;
	padding: 0.75rem 2.5rem 0.75rem 2.75rem;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	color: #fff;
	font-size: 0.95rem;
	transition: all 0.2s;
}

.search-input:focus {
	outline: none;
	background: rgba(255, 255, 255, 0.08);
	border-color: rgba(74, 171, 247, 0.5);
}

.search-input::placeholder {
	color: rgba(255, 255, 255, 0.3);
}

.clear-btn {
	position: absolute;
	right: 0.5rem;
	top: 50%;
	transform: translateY(-50%);
	width: 28px;
	height: 28px;
	border-radius: 8px;
	border: none;
	background: rgba(255, 255, 255, 0.1);
	color: rgba(255, 255, 255, 0.6);
	cursor: pointer;
	display: flex;
	align-items: center;
	justify-content: center;
	transition: all 0.2s;
}

.clear-btn:hover {
	background: rgba(255, 255, 255, 0.15);
	color: #fff;
}

.filter-group {
	display: flex;
	gap: 0.5rem;
	flex-wrap: wrap;
}

.filter-select {
	padding: 0.75rem 1rem;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	color: #fff;
	font-size: 0.9rem;
	cursor: pointer;
	transition: all 0.2s;
}

.filter-select:focus {
	outline: none;
	background: rgba(255, 255, 255, 0.08);
	border-color: rgba(74, 171, 247, 0.5);
}

.filter-select option {
	background: #1a1a2e;
	color: #fff;
}

/* Link Stats Bar */
.link-stats-bar {
	display: flex;
	justify-content: space-between;
	align-items: center;
	background: rgba(255, 255, 255, 0.05);
	backdrop-filter: blur(20px);
	border-radius: 12px;
	padding: 1rem 1.5rem;
	margin-bottom: 1.5rem;
	border: 1px solid rgba(255, 255, 255, 0.1);
}

.stats-group {
	display: flex;
	gap: 2rem;
}

.stat-item {
	display: flex;
	align-items: center;
	gap: 0.75rem;
}

.stat-item svg {
	font-size: 1.5rem;
}

.stat-total svg {
	color: #4dabf7;
}

.stat-active svg {
	color: #51cf66;
}

.stat-dead svg {
	color: #ff6b6b;
}

.stat-expired svg {
	color: #ffa94d;
}

.stat-value {
	font-size: 1.5rem;
	font-weight: 700;
	color: #fff;
	line-height: 1;
}

.stat-label {
	font-size: 0.85rem;
	color: rgba(255, 255, 255, 0.6);
	text-transform: uppercase;
	letter-spacing: 0.5px;
}

.link-stats-bar .btn {
	padding: 0.65rem 1.25rem;
	border-radius: 10px;
	border: none;
	background: linear-gradient(135deg, #4dabf7 0%, #339af0 100%);
	color: #fff;
	font-weight: 500;
	cursor: pointer;
	transition: all 0.2s;
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.link-stats-bar .btn:hover:not(:disabled) {
	transform: translateY(-2px);
	box-shadow: 0 4px 12px rgba(74, 171, 247, 0.4);
}

.link-stats-bar .btn:disabled {
	opacity: 0.5;
	cursor: not-allowed;
}

.link-actions {
	display: flex;
	gap: 1rem;
	align-items: center;
}

.export-controls {
	display: flex;
	gap: 0.5rem;
	align-items: center;
}

.export-format-select {
	padding: 0.65rem 1rem;
	border-radius: 10px;
	border: 1px solid rgba(255, 255, 255, 0.2);
	background: rgba(255, 255, 255, 0.05);
	color: #fff;
	font-weight: 500;
	cursor: pointer;
	transition: all 0.2s;
	backdrop-filter: blur(10px);
}

.export-format-select:hover {
	background: rgba(255, 255, 255, 0.1);
	border-color: rgba(255, 255, 255, 0.3);
}

.export-format-select option {
	background: #1a1a2e;
	color: #fff;
}

.btn-success {
	background: linear-gradient(135deg, #51cf66 0%, #37b24d 100%) !important;
}

.btn-success:hover:not(:disabled) {
	box-shadow: 0 4px 12px rgba(81, 207, 102, 0.4) !important;
}

.btn-jdownloader {
	background: linear-gradient(135deg, #ff6b6b 0%, #ee5a52 100%) !important;
}

.btn-jdownloader:hover:not(:disabled) {
	box-shadow: 0 4px 12px rgba(255, 107, 107, 0.4) !important;
}

/* Loading & Empty States */
.loading-state,
.empty-state {
	text-align: center;
	padding: 4rem 2rem;
	color: rgba(255, 255, 255, 0.5);
}

.loading-state svg,
.empty-state svg {
	color: rgba(255, 255, 255, 0.3);
	margin-bottom: 1rem;
}

/* Posts Container */
.posts-container {
	display: flex;
	flex-direction: column;
	gap: 1.5rem;
}

.post-card {
	background: rgba(255, 255, 255, 0.03);
	backdrop-filter: blur(20px);
	border-radius: 16px;
	border: 1px solid rgba(255, 255, 255, 0.08);
	overflow: hidden;
	transition: all 0.3s;
}

.post-card:hover {
	border-color: rgba(255, 255, 255, 0.15);
	box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

.post-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.post-author {
	display: flex;
	align-items: center;
	gap: 0.75rem;
}

.author-avatar {
	width: 40px;
	height: 40px;
	border-radius: 10px;
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	display: flex;
	align-items: center;
	justify-content: center;
	font-weight: 600;
	font-size: 1.1rem;
	color: #fff;
}

.author-info {
	display: flex;
	flex-direction: column;
	gap: 0.15rem;
}

.author-name {
	font-weight: 600;
	color: #fff;
	font-size: 0.95rem;
}

.post-date {
	font-size: 0.85rem;
	color: rgba(255, 255, 255, 0.5);
}

.post-meta {
	display: flex;
	gap: 0.5rem;
}

.meta-badge {
	display: flex;
	align-items: center;
	gap: 0.3rem;
	padding: 0.3rem 0.6rem;
	border-radius: 8px;
	background: rgba(255, 255, 255, 0.05);
	color: rgba(255, 255, 255, 0.7);
	font-size: 0.85rem;
}

.post-content {
	padding: 1rem 1.25rem;
	color: rgba(255, 255, 255, 0.85);
	line-height: 1.6;
	font-size: 1.3rem;
}

.post-content :deep(p) {
	margin-bottom: 0.75rem;
}

.post-content :deep(a) {
	color: #4dabf7;
	text-decoration: none;
}

.post-content :deep(a:hover) {
	text-decoration: underline;
}

/* Media Section */
.post-media {
	padding: 1.25rem;
	background: rgba(0, 0, 0, 0.2);
}

.media-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(750px, 5fr));
	gap: 1rem;
	margin-bottom: 1rem;
}

.media-item {
	position: relative;
	border-radius: 12px;
	overflow: hidden;
	aspect-ratio: 16 / 9;
}

.image-wrapper {
	width: 100%;
	height: 100%;
	position: relative;
	cursor: pointer;
	overflow: hidden;
}

.image-wrapper img {
	width: 100%;
	height: 100%;
	object-fit: contain;
	transition: transform 0.3s;
}

.download-section {
	margin-top: 1rem;
}

.download-header {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	color: #51cf66;
	font-weight: 600;
	font-size: 0.95rem;
	margin-bottom: 0.75rem;
}

.download-links {
	display: grid;
	gap: 0.75rem;
	grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
}

.download-link {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	padding: 0.85rem 1rem;
	background: rgba(255, 255, 255, 0.05);
	border-radius: 12px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	text-decoration: none;
	transition: all 0.2s;
}

.download-link:hover {
	background: rgba(255, 255, 255, 0.08);
	border-color: rgba(255, 255, 255, 0.2);
	transform: translateX(4px);
}

.link-icon {
	width: 36px;
	height: 36px;
	border-radius: 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.1rem;
}

.provider-gofile .link-icon {
	background: rgba(59, 130, 246, 0.15);
	color: #3b82f6;
}

.provider-pixeldrain .link-icon {
	background: rgba(139, 92, 246, 0.15);
	color: #8b5cf6;
}

.provider-bunkr .link-icon,
.provider-bunkrr .link-icon {
	background: rgba(236, 72, 153, 0.15);
	color: #ec4899;
}

.provider-cyberdrop .link-icon {
	background: rgba(34, 197, 94, 0.15);
	color: #22c55e;
}

.provider-mediafire .link-icon {
	background: rgba(251, 146, 60, 0.15);
	color: #fb923c;
}

.provider-mega .link-icon {
	background: rgba(239, 68, 68, 0.15);
	color: #ef4444;
}

.link-info {
	flex: 1;
	min-width: 0;
}

.link-provider {
	font-weight: 600;
	font-size: 0.9rem;
	text-transform: capitalize;
	margin-bottom: 0.15rem;
}

.provider-gofile .link-provider {
	color: #3b82f6;
}
.provider-pixeldrain .link-provider {
	color: #8b5cf6;
}
.provider-bunkr .link-provider,
.provider-bunkrr .link-provider {
	color: #ec4899;
}
.provider-cyberdrop .link-provider {
	color: #22c55e;
}
.provider-mediafire .link-provider {
	color: #fb923c;
}
.provider-mega .link-provider {
	color: #ef4444;
}

.link-url {
	font-size: 0.85rem;
	color: rgba(255, 255, 255, 0.5);
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.link-arrow {
	color: rgba(255, 255, 255, 0.3);
	font-size: 0.9rem;
}

/* Download Tracking Styles */
.download-link-wrapper {
	display: flex;
	gap: 0.5rem;
	align-items: center;
	transition: border-color 0.2s, opacity 0.2s;
	border: 2px solid rgba(255, 255, 255, 0.08);
	border-radius: 8px;
	padding: 0.4rem;
	margin-bottom: 0.35rem;
}

/* Health-based border colors */
.download-link-wrapper.health-active {
	border-color: rgba(81, 207, 102, 0.5);
}

.download-link-wrapper.health-dead {
	border-color: rgba(255, 107, 107, 0.5);
}

.download-link-wrapper.health-expired {
	border-color: rgba(255, 169, 77, 0.5);
}

.download-link-wrapper.status-downloaded {
	opacity: 0.7;
}

.download-link-wrapper .download-link {
	flex: 1;
	min-width: 0;
}

.link-actions {
	display: flex;
	gap: 0.25rem;
	flex-shrink: 0;
}

/* Link Health Indicators */
.link-health-badge {
	width: 24px;
	height: 24px;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 0.7rem;
	flex-shrink: 0;
}

.link-health-badge.health-active {
	background: rgba(81, 207, 102, 0.15);
	color: #51cf66;
}

.link-health-badge.health-dead {
	background: rgba(255, 107, 107, 0.15);
	color: #ff6b6b;
}

.link-health-badge.health-expired {
	background: rgba(255, 169, 77, 0.15);
	color: #ffa94d;
}

/* Dead link styling */
.download-link-wrapper.health-dead .download-link {
	opacity: 0.45;
}

.download-link-wrapper.health-dead .link-url {
	text-decoration: line-through;
	color: rgba(255, 107, 107, 0.5);
}

/* Expired link styling */
.download-link-wrapper.health-expired .link-url {
	color: rgba(255, 169, 77, 0.6);
}

/* Post health summary badges */
.meta-badge.health-all-active {
	background: rgba(81, 207, 102, 0.15);
	color: #51cf66;
	border: 1px solid rgba(81, 207, 102, 0.3);
}

.meta-badge.health-all-dead {
	background: rgba(255, 107, 107, 0.15);
	color: #ff6b6b;
	border: 1px solid rgba(255, 107, 107, 0.3);
}

.meta-badge.health-mixed {
	background: rgba(255, 169, 77, 0.15);
	color: #ffa94d;
	border: 1px solid rgba(255, 169, 77, 0.3);
}

/* Post card left border based on health */
.post-card.post-health-active {
	border-left: 3px solid #51cf66;
}

.post-card.post-health-dead {
	border-left: 3px solid #ff6b6b;
}

.post-card.post-health-mixed {
	border-left: 3px solid #ffa94d;
}

.action-btn {
	width: 32px;
	height: 32px;
	border-radius: 8px;
	border: none;
	cursor: pointer;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 0.85rem;
	transition: all 0.2s;
	background: rgba(255, 255, 255, 0.05);
	color: #fff;
}

.action-btn:hover {
	transform: scale(1.1);
}

.success-btn {
	background: rgba(81, 207, 102, 0.15);
	color: #51cf66;
}

.success-btn:hover {
	background: rgba(81, 207, 102, 0.25);
}

.danger-btn {
	background: rgba(255, 107, 107, 0.15);
	color: #ff6b6b;
}

.danger-btn:hover {
	background: rgba(255, 107, 107, 0.25);
}

.reset-btn {
	background: rgba(255, 255, 255, 0.05);
	color: rgba(255, 255, 255, 0.6);
}

.reset-btn:hover {
	background: rgba(255, 255, 255, 0.1);
	color: #fff;
}

.stat-downloaded svg {
	color: #51cf66;
}

@media (max-width: 768px) {
	.header-content {
		flex-direction: column;
	}

	.header-actions {
		width: 100%;
		justify-content: flex-end;
	}

	.filters-bar {
		flex-direction: column;
	}

	.search-box {
		min-width: auto;
	}

	.download-links {
		grid-template-columns: 1fr;
	}

	.media-grid {
		grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
	}
}

/* Send to JDownloader Modal */
.jd-modal-backdrop {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.7);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 1050;
}

.jd-modal-content {
	background: #1a1a2e;
	border-radius: 16px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	width: 90%;
	max-width: 650px;
	max-height: 90vh;
	overflow: auto;
	box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
}

.jd-modal-header {
	padding: 1.25rem 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.jd-modal-title {
	margin: 0;
	color: #fff;
	font-size: 1.2rem;
	font-weight: 600;
	display: flex;
	align-items: center;
}

.jd-btn-close {
	background: transparent;
	border: none;
	color: rgba(255, 255, 255, 0.6);
	font-size: 1.5rem;
	cursor: pointer;
	padding: 0;
	line-height: 1;
}

.jd-btn-close:hover {
	color: #fff;
}

.jd-modal-body {
	padding: 1.5rem;
}

.jd-modal-footer {
	padding: 1rem 1.5rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
	display: flex;
	justify-content: flex-end;
	gap: 0.75rem;
}

.jd-form-group {
	margin-bottom: 1.25rem;
}

.jd-form-label {
	display: block;
	color: rgba(255, 255, 255, 0.8);
	font-size: 0.9rem;
	font-weight: 500;
	margin-bottom: 0.5rem;
}

.jd-form-control {
	width: 100%;
	padding: 0.75rem 1rem;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.15);
	border-radius: 10px;
	color: #fff;
	font-size: 0.9rem;
	transition: border-color 0.2s;
}

.jd-form-control:focus {
	outline: none;
	border-color: rgba(74, 171, 247, 0.5);
	background: rgba(255, 255, 255, 0.08);
}

.jd-form-control::placeholder {
	color: rgba(255, 255, 255, 0.3);
}

.jd-textarea {
	resize: vertical;
	min-height: 120px;
	font-family: 'Fira Code', 'Consolas', monospace;
	font-size: 0.8rem;
	line-height: 1.5;
}

.jd-input-group {
	display: flex;
}

.jd-input-prefix {
	padding: 0.75rem 0.75rem;
	background: rgba(255, 255, 255, 0.1);
	border: 1px solid rgba(255, 255, 255, 0.15);
	border-right: none;
	border-radius: 10px 0 0 10px;
	color: rgba(255, 255, 255, 0.6);
	font-size: 0.9rem;
	display: flex;
	align-items: center;
}

.jd-input-with-prefix {
	border-radius: 0 10px 10px 0;
}

.jd-form-check {
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.jd-check-input {
	width: 18px;
	height: 18px;
	accent-color: #4dabf7;
	cursor: pointer;
}

.jd-check-label {
	color: rgba(255, 255, 255, 0.8);
	font-size: 0.9rem;
	cursor: pointer;
}

.jd-dedup-info {
	display: flex;
	align-items: flex-start;
	gap: 0.75rem;
	padding: 0.75rem 1rem;
	border-radius: 10px;
	background: rgba(74, 171, 247, 0.1);
	border: 1px solid rgba(74, 171, 247, 0.2);
	color: rgba(255, 255, 255, 0.8);
	font-size: 0.85rem;
	margin-bottom: 1.25rem;
	line-height: 1.4;
}

.jd-dedup-info svg {
	color: #4dabf7;
	font-size: 1rem;
	margin-top: 0.1rem;
	flex-shrink: 0;
}

/* Priority Config */
.priority-config-section {
	margin-bottom: 1rem;
}

.priority-toggle-btn {
	display: flex;
	align-items: center;
	width: 100%;
	padding: 0.6rem 1rem;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 8px;
	color: rgba(255, 255, 255, 0.8);
	font-size: 0.85rem;
	cursor: pointer;
	transition: background 0.2s;
}
.priority-toggle-btn:hover {
	background: rgba(255, 255, 255, 0.1);
}

.priority-list {
	margin-top: 0.5rem;
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 8px;
	overflow: hidden;
}

.priority-item {
	display: flex;
	align-items: center;
	padding: 0.5rem 0.75rem;
	background: rgba(255, 255, 255, 0.03);
	border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}
.priority-item:last-of-type {
	border-bottom: none;
}

.priority-rank {
	width: 1.5rem;
	height: 1.5rem;
	display: flex;
	align-items: center;
	justify-content: center;
	background: rgba(139, 92, 246, 0.2);
	color: #a78bfa;
	border-radius: 50%;
	font-size: 0.75rem;
	font-weight: 600;
	margin-right: 0.75rem;
	flex-shrink: 0;
}

.priority-name {
	flex: 1;
	color: rgba(255, 255, 255, 0.85);
	font-size: 0.85rem;
	text-transform: capitalize;
}

.priority-actions {
	display: flex;
	gap: 0.25rem;
}

.priority-btn {
	width: 1.6rem;
	height: 1.6rem;
	display: flex;
	align-items: center;
	justify-content: center;
	background: rgba(255, 255, 255, 0.06);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 4px;
	color: rgba(255, 255, 255, 0.6);
	cursor: pointer;
	font-size: 0.7rem;
	transition: all 0.2s;
}
.priority-btn:hover:not(:disabled) {
	background: rgba(139, 92, 246, 0.2);
	color: #a78bfa;
	border-color: rgba(139, 92, 246, 0.3);
}
.priority-btn:disabled {
	opacity: 0.3;
	cursor: not-allowed;
}

.priority-reset-btn {
	display: flex;
	align-items: center;
	justify-content: center;
	width: 100%;
	padding: 0.45rem;
	background: rgba(255, 255, 255, 0.03);
	border: none;
	border-top: 1px solid rgba(255, 255, 255, 0.06);
	color: rgba(255, 255, 255, 0.5);
	font-size: 0.8rem;
	cursor: pointer;
	transition: all 0.2s;
}
.priority-reset-btn:hover {
	background: rgba(255, 255, 255, 0.06);
	color: rgba(255, 255, 255, 0.8);
}
</style>
