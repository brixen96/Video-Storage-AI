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
						<button class="btn-icon btn-ai" @click="enhanceThread" :disabled="loading || enhancing" title="AI Enhance Thread">
							<font-awesome-icon :icon="['fas', enhancing ? 'spinner' : 'magic']" :spin="enhancing" />
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
					<button
						class="keyword-toggle-btn"
						:class="{ active: filterKeyword === 'bbc' }"
						@click="filterKeyword = filterKeyword === 'bbc' ? '' : 'bbc'"
						title="Filter posts: BBC, Interracial, Black, Ebony + BBC performers"
					>
						<font-awesome-icon :icon="['fas', 'filter']" class="me-1" />
						BBC
					</button>
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
					<button class="btn btn-jdownloader" @click="openJDModal" :disabled="totalLinks === 0" title="Send to JDownloader">
						<font-awesome-icon :icon="['fas', 'cloud-download-alt']" />
						Send to JD
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
				<p v-if="searchQuery || filterProvider || filterHasImages || filterKeyword">No posts match your filters</p>
				<p v-else>No posts found</p>
			</div>

			<!-- Posts Grid -->
			<div v-else class="posts-container">
				<ThreadPostCard
					v-for="post in filteredPosts"
					:key="post.id"
					:post="post"
					:links="getPostLinks(post.id)"
					@mark-downloaded="markAsDownloaded"
					@mark-failed="markAsFailed"
					@reset-status="resetDownloadStatus"
				/>
			</div>
		</div>

		<!-- AI Enhancement Modal -->
		<div v-if="showEnhanceModal" class="modal-overlay" @click.self="showEnhanceModal = false">
			<div class="modal-content enhance-modal">
				<div class="modal-header">
					<h3>
						<font-awesome-icon :icon="['fas', 'magic']" class="me-2" />
						AI Thread Enhancement
					</h3>
					<button class="btn-close" @click="showEnhanceModal = false">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
				<div class="modal-body">
					<p class="mb-4">Enhance this thread with AI-powered analysis and cleanup:</p>
					
					<div class="enhance-options">
						<label class="enhance-option">
							<input type="checkbox" v-model="enhanceOptions.clean_content" />
							<div class="option-content">
								<font-awesome-icon :icon="['fas', 'broom']" />
								<div>
									<strong>Clean Content</strong>
									<small>Remove forum boilerplate, signatures, and clutter</small>
								</div>
							</div>
						</label>
						
						<label class="enhance-option">
							<input type="checkbox" v-model="enhanceOptions.extract_performers" />
							<div class="option-content">
								<font-awesome-icon :icon="['fas', 'user']" />
								<div>
									<strong>Extract Performers</strong>
									<small>Find and extract performer names from posts</small>
								</div>
							</div>
						</label>
						
						<label class="enhance-option">
							<input type="checkbox" v-model="enhanceOptions.extract_tags" />
							<div class="option-content">
								<font-awesome-icon :icon="['fas', 'tags']" />
								<div>
									<strong>Extract Tags</strong>
									<small>Identify content categories and tags</small>
								</div>
							</div>
						</label>
						
						<label class="enhance-option">
							<input type="checkbox" v-model="enhanceOptions.generate_summary" />
							<div class="option-content">
								<font-awesome-icon :icon="['fas', 'file-alt']" />
								<div>
									<strong>Generate Summary</strong>
									<small>Create a summary of the thread content</small>
								</div>
							</div>
						</label>
						
						<label class="enhance-option">
							<input type="checkbox" v-model="enhanceOptions.validate_images" />
							<div class="option-content">
								<font-awesome-icon :icon="['fas', 'image']" />
								<div>
									<strong>Validate Images</strong>
									<small>Check image quality and detect broken images using AI vision</small>
								</div>
							</div>
						</label>
					</div>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="showEnhanceModal = false">Cancel</button>
					<button class="btn btn-ai" @click="startEnhancement" :disabled="!enhanceOptions.clean_content && !enhanceOptions.extract_performers && !enhanceOptions.extract_tags && !enhanceOptions.generate_summary && !enhanceOptions.validate_images">
						<font-awesome-icon :icon="['fas', 'magic']" class="me-2" />
						Start Enhancement
					</button>
				</div>
			</div>
		</div>

		<!-- Send to JDownloader Modal -->
		<ThreadJDModal
			v-if="showJDModal"
			:links="downloadLinks"
			:default-package-name="thread?.title || ''"
			@close="showJDModal = false"
			@sent="showJDModal = false"
		/>
	</div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch, getCurrentInstance } from 'vue'
import { useRoute } from 'vue-router'
import { SearchBox } from '@/components/shared'
import { API_ORIGIN } from '@/services/api'
import websocketService from '@/services/websocket'
import ThreadPostCard from '@/components/ThreadPostCard.vue'
import ThreadJDModal from '@/components/ThreadJDModal.vue'
import { useActionTracker } from '@/composables/ai/useActionTracker'

const route = useRoute()
const { proxy } = getCurrentInstance()
const toast = proxy.$toast
const { trackAction } = useActionTracker()

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
const showJDModal = ref(false)

// Performer creation state
const creatingPerformer = ref(false)

// AI Enhancement state
const enhancing = ref(false)
const showEnhanceModal = ref(false)
const enhanceOptions = ref({
	clean_content: true,
	extract_performers: true,
	extract_tags: true,
	generate_summary: true,
	validate_images: false
})

// Export state
const exportFormat = ref('txt')

// Filter state
const searchQuery = ref('')
const filterProvider = ref('')
const filterHasImages = ref('')
const filterKeyword = ref('')
const sortBy = ref('date')

// BBC-related keyword patterns (more specific to avoid false positives)
const bbcKeywords = [
	'bbc', 'bbc queen', 'bbc slut', 'bbc anal', 'bbc blowjob', 'bbc compilation',
	'bbc king', 'big black cock', 'big black dick', 'black cock', 'black dick',
	'interracial sex', 'interracial anal', 'interracial gangbang',
	'blackzilla', 'monster black',
	'mandingo'
]

// Common BBC performer names (well-known in the genre)
const bbcPerformerNames = [
	'mandingo', 'blackzilla', 'mr wood', 'bbc master'
]

// Combined pattern for BBC filter (keywords + performer names)
// Uses word boundaries to avoid partial matches like "blackboard"
const bbcPattern = computed(() => {
	const allTerms = [...bbcKeywords, ...bbcPerformerNames]
	const escaped = allTerms.map(t => t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
	return new RegExp(`\\b(${escaped.join('|')})\\b`, 'i')
})

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

	// Hide posts where ALL links are dead/expired by default
	filtered = filtered.filter((post) => {
		const postLinks = getPostLinks(post.id)
		if (postLinks.length === 0) return true // No links = keep (might have text/images)
		const hasActiveLink = postLinks.some((link) => link.status !== 'dead' && link.status !== 'expired')
		return hasActiveLink
	})

	// Search filter - use plain_text for accurate text matching
	if (searchQuery.value) {
		const query = searchQuery.value.toLowerCase()
		filtered = filtered.filter((post) => {
			const textMatch = post.plain_text?.toLowerCase().includes(query)
			const authorMatch = post.author?.toLowerCase().includes(query)
			const linksMatch = getPostLinks(post.id).some((link) => link.url.toLowerCase().includes(query) || link.provider.toLowerCase().includes(query))
			return textMatch || authorMatch || linksMatch
		})
	}

	// Keyword filter (BBC) - check both plain_text AND raw HTML content (links have text in HTML)
	if (filterKeyword.value === 'bbc') {
		filtered = filtered.filter((post) => {
			const text = post.plain_text || ''
			const html = post.content || ''
			return bbcPattern.value.test(text) || bbcPattern.value.test(html)
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
		const response = await fetch(`${API_ORIGIN}/api/v1/scraper/threads/${route.params.id}/link-stats`)
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
		const response = await fetch(`${API_ORIGIN}/api/v1/scraper/threads/${route.params.id}/verify-links`, {
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
		const url = `${API_ORIGIN}/api/v1/scraper/threads/${route.params.id}/export-links?${params.toString()}`
		window.open(url, '_blank')

		toast.success(`Exporting links as ${exportFormat.value.toUpperCase()}...`)
	} catch (error) {
		console.error('Error exporting links:', error)
		toast.error('Failed to export links: ' + error.message)
	}
}

// AI Enhancement functions
const enhanceThread = async () => {
	showEnhanceModal.value = true
}

const startEnhancement = async () => {
	enhancing.value = true
	showEnhanceModal.value = false
	
	try {
		const response = await fetch(`${API_ORIGIN}/api/v1/scraper/threads/${route.params.id}/enhance`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(enhanceOptions.value)
		})
		const data = await response.json()
		
		if (data.success) {
			toast.success('AI Enhancement started! Check the activity panel for progress.')
			trackAction('thread_enhanced', `AI-enhanced thread "${thread.value?.title || route.params.id}"`, { thread_id: route.params.id })
			// Poll for completion
			pollEnhancementStatus()
		} else {
			toast.error(data.message || 'Failed to start enhancement')
			enhancing.value = false
		}
	} catch (error) {
		console.error('Error starting enhancement:', error)
		toast.error('Failed to start enhancement: ' + error.message)
		enhancing.value = false
	}
}

const pollEnhancementStatus = async () => {
	const checkStatus = async () => {
		try {
			const response = await fetch(`${API_ORIGIN}/api/v1/scraper/threads/${route.params.id}/enhancement-status`)
			const data = await response.json()
			
			if (data.success && data.data?.jobs?.length > 0) {
				const latestJob = data.data.jobs[0]
				if (latestJob.status === 'completed' || latestJob.status === 'failed') {
					enhancing.value = false
					if (latestJob.status === 'completed') {
						toast.success('Thread enhancement completed!')
						loadThread() // Reload thread data
					} else {
						toast.error('Thread enhancement failed: ' + latestJob.message)
					}
					return
				}
			}
			// Continue polling
			setTimeout(checkStatus, 2000)
		} catch (error) {
			console.error('Error checking enhancement status:', error)
			enhancing.value = false
		}
	}
	checkStatus()
}

// Download tracking functions
const loadDownloadStats = async () => {
	try {
		const response = await fetch(`${API_ORIGIN}/api/v1/downloads/threads/${route.params.id}/stats`)
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
		const response = await fetch(`${API_ORIGIN}/api/v1/downloads/links/${linkId}/mark-downloaded`, {
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
		const response = await fetch(`${API_ORIGIN}/api/v1/downloads/links/${linkId}/mark-failed`, {
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
		const response = await fetch(`${API_ORIGIN}/api/v1/downloads/links/${linkId}/reset`, {
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
		const response = await fetch(`${API_ORIGIN}/api/v1/scraper/threads/${route.params.id}`)
		const data = await response.json()

		if (data.success) {
			thread.value = data.data.thread
			posts.value = data.data.posts || []
			downloadLinks.value = data.data.download_links || []
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
	loading.value = true
	try {
		const url = `${API_ORIGIN}/api/v1/scraper/threads/${route.params.id}/rescrape`
		const response = await fetch(url, {
			method: 'POST',
		})
		const data = await response.json()

		if (data.success) {
			toast.success('Thread rescraping started. Check Tasks page for progress.')
			trackAction('thread_rescraped', `Rescraped thread "${thread.value?.title || route.params.id}"`, { thread_id: route.params.id })
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
		const url = `${API_ORIGIN}/api/v1/scraper/threads/${route.params.id}`
		const response = await fetch(url, {
			method: 'DELETE',
		})
		const data = await response.json()

		if (data.success) {
			toast.success('Thread deleted successfully')
			trackAction('thread_deleted', `Deleted thread "${thread.value?.title || route.params.id}"`, { thread_id: route.params.id })
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
		const response = await fetch(`${API_ORIGIN}/api/v1/scraper/performers/create-from-thread`, {
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

const getPostImages = (post) => post.metadata?.attachments || []

// JDownloader modal - open

const openJDModal = async () => {
	try {
		const statusResponse = await fetch(`${API_ORIGIN}/api/v1/jdownloader/status`)
		const statusData = await statusResponse.json()
		if (!statusData.success && !statusData.available && !statusData.connected) {
			toast.error('JDownloader is not running! Please start JDownloader and enable Direct Connection API.')
			return
		}
	} catch (error) {
		toast.error('Cannot connect to JDownloader: ' + error.message)
		return
	}
	showJDModal.value = true
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
			filterKeyword.value = ''
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

/* Keyword Toggle Button */
.keyword-toggle-btn {
	padding: 0.75rem 1rem;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	color: rgba(255, 255, 255, 0.7);
	font-size: 0.9rem;
	font-weight: 600;
	cursor: pointer;
	transition: all 0.2s;
	text-transform: uppercase;
	letter-spacing: 0.5px;
}

.keyword-toggle-btn:hover {
	background: rgba(255, 255, 255, 0.08);
	border-color: rgba(255, 255, 255, 0.2);
	color: #fff;
}

.keyword-toggle-btn.active {
	background: rgba(34, 197, 94, 0.2);
	border-color: rgba(34, 197, 94, 0.5);
	color: #4ade80;
	box-shadow: 0 0 12px rgba(34, 197, 94, 0.2);
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

.stat-downloaded svg {
	color: #51cf66;
}

/* AI Enhancement Button */
.btn-ai {
	background: linear-gradient(135deg, #9f7aea 0%, #805ad5 100%) !important;
	border-color: rgba(159, 122, 234, 0.5) !important;
}

.btn-ai:hover:not(:disabled) {
	box-shadow: 0 4px 12px rgba(159, 122, 234, 0.4) !important;
	transform: translateY(-2px);
}

/* Enhancement Modal */
.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.7);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 1000;
	backdrop-filter: blur(4px);
}

.modal-content {
	background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 16px;
	width: 90%;
	max-width: 500px;
	box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
}

.modal-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.modal-header h3 {
	margin: 0;
	font-size: 1.25rem;
	font-weight: 600;
	color: #fff;
	display: flex;
	align-items: center;
}

.modal-header h3 svg {
	color: #9f7aea;
}

.btn-close {
	background: none;
	border: none;
	color: rgba(255, 255, 255, 0.5);
	cursor: pointer;
	padding: 0.5rem;
	border-radius: 8px;
	transition: all 0.2s;
}

.btn-close:hover {
	background: rgba(255, 255, 255, 0.1);
	color: #fff;
}

.modal-body {
	padding: 1.5rem;
}

.modal-body p {
	color: rgba(255, 255, 255, 0.7);
	margin: 0;
}

.enhance-options {
	display: flex;
	flex-direction: column;
	gap: 0.75rem;
}

.enhance-option {
	display: flex;
	align-items: center;
	padding: 1rem;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	cursor: pointer;
	transition: all 0.2s;
}

.enhance-option:hover {
	background: rgba(255, 255, 255, 0.08);
	border-color: rgba(159, 122, 234, 0.3);
}

.enhance-option:has(input:checked) {
	background: rgba(159, 122, 234, 0.1);
	border-color: rgba(159, 122, 234, 0.5);
}

.enhance-option input {
	margin-right: 1rem;
	width: 18px;
	height: 18px;
	accent-color: #9f7aea;
}

.option-content {
	display: flex;
	align-items: center;
	gap: 1rem;
	flex: 1;
}

.option-content svg {
	color: #9f7aea;
	font-size: 1.25rem;
	width: 24px;
	text-align: center;
}

.option-content strong {
	display: block;
	color: #fff;
	font-size: 0.95rem;
	margin-bottom: 0.25rem;
}

.option-content small {
	color: rgba(255, 255, 255, 0.5);
	font-size: 0.8rem;
}

.modal-footer {
	display: flex;
	justify-content: flex-end;
	gap: 1rem;
	padding: 1.5rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.btn {
	padding: 0.75rem 1.5rem;
	border-radius: 10px;
	font-weight: 600;
	cursor: pointer;
	transition: all 0.2s;
	border: none;
}

.btn-secondary {
	background: rgba(255, 255, 255, 0.1);
	color: rgba(255, 255, 255, 0.7);
}

.btn-secondary:hover {
	background: rgba(255, 255, 255, 0.15);
	color: #fff;
}

.btn-ai {
	background: linear-gradient(135deg, #9f7aea 0%, #805ad5 100%);
	color: #fff;
	display: flex;
	align-items: center;
}

.btn-ai:hover:not(:disabled) {
	box-shadow: 0 4px 12px rgba(159, 122, 234, 0.4);
	transform: translateY(-2px);
}

.btn-ai:disabled {
	opacity: 0.5;
	cursor: not-allowed;
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

</style>
