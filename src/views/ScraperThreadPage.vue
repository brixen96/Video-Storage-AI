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
				<div class="search-box">
					<font-awesome-icon :icon="['fas', 'search']" class="search-icon" />
					<input v-model="searchQuery" type="text" placeholder="Search posts, links, or content..." class="search-input" />
					<button v-if="searchQuery" @click="searchQuery = ''" class="clear-btn">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
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
				<div v-for="post in filteredPosts" :key="post.id" class="post-card">
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
								<a
									v-for="link in getPostLinks(post.id)"
									:key="link.id"
									:href="link.url"
									target="_blank"
									class="download-link"
									:class="`provider-${link.provider.toLowerCase()}`"
								>
									<div class="link-icon">
										<font-awesome-icon :icon="getProviderIcon(link.provider)" />
									</div>
									<div class="link-info">
										<div class="link-provider">{{ link.provider }}</div>
										<div class="link-url">{{ truncateURL(link.url) }}</div>
									</div>
									<font-awesome-icon :icon="['fas', 'external-link-alt']" class="link-arrow" />
								</a>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, onMounted, watch, getCurrentInstance } from 'vue'
import { useRoute } from 'vue-router'
import DOMPurify from 'dompurify'

const route = useRoute()
const { proxy } = getCurrentInstance()
const toast = proxy.$toast

// State
const loading = ref(false)
const thread = ref(null)
const posts = ref([])
const downloadLinks = ref([])

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

// Methods
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

const getPostLinks = (postId) => {
	return downloadLinks.value.filter((link) => link.post_id === postId)
}

const getPostImages = (post) => {
	return post.metadata?.attachments || []
}

const openImage = (url) => {
	window.open(url, '_blank')
}

const getProviderIcon = (provider) => {
	const icons = {
		gofile: ['fas', 'cloud-download-alt'],
		pixeldrain: ['fas', 'cloud-download-alt'],
		bunkr: ['fas', 'cloud-download-alt'],
		bunkrr: ['fas', 'cloud-download-alt'],
		cyberdrop: ['fas', 'cloud-download-alt'],
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

onMounted(() => {
	loadThread()
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
