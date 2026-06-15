<template>
	<div class="dashboard-feed">
		<!-- Feed header + filter chips -->
		<div class="feed-header">
			<div class="feed-header-top">
				<h3>
					<font-awesome-icon :icon="['fas', 'rss']" class="me-2" />
					Activity Feed
				</h3>
				<button
					v-if="items.length > 0"
					class="clear-feed-btn"
					@click="$emit('clear-feed')"
					title="Clear all feed items"
				>
					<font-awesome-icon :icon="['fas', 'trash-alt']" class="me-1" />
					Clear
				</button>
			</div>
			<div class="feed-filters" v-if="scraperTypes.length > 0">
				<button class="filter-chip" :class="{ active: !activeFilter }" @click="$emit('filter-change', '')">All</button>
				<button
					v-for="st in mergedScraperTypes"
					:key="st.type || st"
					class="filter-chip"
					:class="{ active: activeFilter === (st.type || st) }"
					@click="$emit('filter-change', st.type || st)"
				>{{ st.display_name || formatScraperName(st) }}</button>
			</div>
		</div>

		<!-- SimpCity Watched Threads Tab Section — only shown when simpcity filter is active -->
		<div class="simpcity-tabs-section" v-if="activeFilter === 'simpcity' && watchedThreads.length > 0">
			<div class="tabs-header">
				<div class="tabs-title">
					<font-awesome-icon :icon="['fas', 'comments']" />
					SimpCity Watched Threads
				</div>
				<div class="thread-tabs">
					<button
						v-for="thread in watchedThreads"
						:key="thread.id"
						class="thread-tab"
						:class="{ active: activeThreadId === thread.id }"
						@click="selectThread(thread)"
					>
						<span class="tab-name">{{ thread.title }}</span>
						<span class="tab-open" @click.stop="openInFirefox(`/scraper/${thread.id}`)">
							<font-awesome-icon :icon="['fas', 'external-link-alt']" />
						</span>
					</button>
				</div>
			</div>

			<!-- Posts panel for selected thread -->
			<div class="thread-posts-panel" v-if="activeThreadId">
				<div class="posts-list" v-if="threadPosts.length > 0">
					<div
						v-for="post in threadPosts"
						:key="post.id"
						class="thread-post"
						:class="{ 'is-read': post.is_read }"
						@mouseenter="markPostRead(post)"
					>
						<div class="post-header">
							<span class="unread-dot" v-if="!post.is_read"></span>
							<span class="post-author">{{ post.author || 'Unknown' }}</span>
							<span class="post-time">{{ formatTimeAgo(post.posted_at) }}</span>
						</div>
						<div class="post-text" v-if="post.plain_text">{{ truncateText(post.plain_text, 250) }}</div>

						<!-- Images -->
						<div
							v-if="getPostImages(post).length > 0"
							class="post-images"
							:class="{ 'single-image': getPostImages(post).length === 1 }"
						>
							<div
								v-for="(img, i) in getPostImages(post).slice(0, 6)"
								:key="i"
								class="post-image"
								@click="openInFirefox(img.url)"
							>
								<img
									:src="img.thumbnail_url || img.url"
									:alt="img.filename || 'image'"
									@error="e => e.target.closest('.post-image').style.display='none'"
								/>
							</div>
						</div>

						<!-- Links -->
						<div class="post-links" v-if="getPostLinks(post).length > 0">
							<a
								v-for="(link, i) in getPostLinks(post).slice(0, 6)"
								:key="i"
								class="post-link"
								@click.stop="openInFirefox(link)"
								:title="link"
							>
								<font-awesome-icon :icon="['fas', 'link']" class="post-link-icon" />
								<span class="post-link-text">{{ formatLinkLabel(link) }}</span>
							</a>
						</div>
					</div>
				</div>
				<div class="posts-loading" v-else-if="loadingPosts">
					<font-awesome-icon :icon="['fas', 'spinner']" spin /> Loading posts...
				</div>
				<div class="posts-empty" v-else>
					<font-awesome-icon :icon="['fas', 'inbox']" /> No recent posts
				</div>
			</div>
		</div>

		<!-- YouTube-style video grid -->
		<div v-if="filteredFlatVideos.length > 0" class="yt-feed-grid">
			<div
				v-for="(video, idx) in filteredFlatVideos"
				:key="idx"
				class="yt-video-card"
				@click="openVideo(video)"
			>
				<div class="yt-thumb-wrap">
					<img
						v-if="video.thumbnail_url && !failedThumbs[video.thumbnail_url]"
						:src="video.thumbnail_url"
						:alt="video.title"
						loading="lazy"
						@error="handleThumbError(video)"
					/>
					<div class="yt-no-thumb" v-else>
						<font-awesome-icon :icon="['fas', 'video']" />
					</div>
					<span class="yt-dur" v-if="video.duration">{{ formatDuration(video.duration) }}</span>
				</div>
				<div class="yt-card-body">
					<div class="yt-card-title" :title="video.title">{{ video.title }}</div>
					<div class="yt-card-source">{{ video.scraper_name }}</div>
					<div class="yt-card-meta">
						<span v-if="video.views">{{ formatViews(video.views) }} views &bull; </span>{{ formatTimeAgo(video.found_at) }}
					</div>
				</div>
			</div>
		</div>

		<!-- Non-video activity items (errors, system messages, etc.) -->
		<div class="activity-log-items" v-if="nonVideoItems.length > 0">
			<div class="activity-log-label" v-if="filteredFlatVideos.length > 0">Activity Log</div>
			<TransitionGroup name="feed">
				<div v-for="item in nonVideoItems" :key="item.id" class="feed-item" :class="item.color">
					<div class="feed-icon">
						<font-awesome-icon :icon="['fas', item.icon || 'bell']" />
					</div>
					<div class="feed-content">
						<div class="feed-title">{{ item.title }}</div>
						<div class="feed-message">{{ item.message }}</div>
						<div class="feed-time">{{ formatTimeAgo(item.created_at) }}</div>
					</div>
				</div>
			</TransitionGroup>
		</div>

		<div class="empty-state" v-if="filteredFlatVideos.length === 0 && nonVideoItems.length === 0">
			<font-awesome-icon :icon="['fas', 'inbox']" class="empty-icon" />
			<p>No recent activity</p>
		</div>
		<div class="feed-footer" v-if="hasMore">
			<button class="btn btn-link" @click="$emit('load-more')">Load more</button>
		</div>
	</div>
</template>

<script>
import { scraperThreadsAPI, openInFirefox as openInFirefoxAPI } from '@/services/api'

export default {
	name: 'DashboardFeed',
	props: {
		items: { type: Array, default: () => [] },
		hasMore: { type: Boolean, default: false },
		scraperTypes: { type: Array, default: () => [] },
		activeFilter: { type: String, default: '' },
		allScraperTypes: { type: Array, default: () => [] },
	},
	emits: ['load-more', 'filter-change', 'clear-feed'],
	data() {
		return {
			watchedThreads: [],
			activeThreadId: null,
			threadPosts: [],
			loadingPosts: false,
			expandedItems: {},
			failedThumbs: {},
		}
	},
	computed: {
		mergedScraperTypes() {
			const seen = new Set()
			const result = []
			for (const st of this.scraperTypes) {
				const key = st.type || st
				if (!seen.has(key)) { seen.add(key); result.push(st) }
			}
			for (const s of this.allScraperTypes) {
				const key = s.scraper_type
				if (!seen.has(key)) {
					seen.add(key)
					result.push({ type: s.scraper_type, display_name: s.display_name || s.scraper_type })
				}
			}
			return result
		},
		flatVideos() {
			const videos = []
			for (const item of this.items) {
				if (item.item_type !== 'new_content') continue
				let metadata
				try {
					metadata = typeof item.metadata === 'string' ? JSON.parse(item.metadata) : (item.metadata || {})
				} catch { continue }
				for (const v of (metadata.videos || [])) {
					if (v.type === 'thread') continue
					videos.push({ ...v, scraper_type: item.scraper_type, scraper_name: item.title, found_at: item.created_at, feed_color: item.color })
				}
			}
			return videos
		},
		filteredFlatVideos() {
			if (!this.activeFilter) return this.flatVideos
			return this.flatVideos.filter((v) => v.scraper_type === this.activeFilter)
		},
		nonVideoItems() {
			return this.items.filter((item) => {
				if (item.item_type === 'new_content' && this.getVideos(item).length > 0) return false
				if (this.activeFilter && item.scraper_type && item.scraper_type !== this.activeFilter) return false
				return true
			})
		},
	},
	async mounted() {
		await this.loadWatchedThreads()
	},
	methods: {
		async loadWatchedThreads() {
			try {
				const response = await scraperThreadsAPI.getWatched()
				this.watchedThreads = response?.data || []
				if (this.watchedThreads.length > 0) {
					this.selectThread(this.watchedThreads[0])
				}
			} catch (e) {
				// No watched threads or endpoint not ready
			}
		},
		async selectThread(thread) {
			if (this.activeThreadId === thread.id) return
			this.activeThreadId = thread.id
			this.threadPosts = []
			this.loadingPosts = true
			try {
				const response = await scraperThreadsAPI.getRecentPosts(thread.id, 15)
				if (response?.data && this.activeThreadId === thread.id) {
					this.threadPosts = response.data
				}
			} catch (e) {
				console.error('Failed to load posts:', e)
			} finally {
				this.loadingPosts = false
			}
		},
		async markPostRead(post) {
			if (post.is_read) return
			post.is_read = true
			try { await scraperThreadsAPI.markPostAsRead(post.id) }
			catch (e) { console.error('Failed to mark post as read:', e) }
		},
		openInFirefox(url) {
			if (!url) return
			openInFirefoxAPI(url).catch(() => {
				// Fallback if backend unavailable
				window.open(url, '_blank')
			})
		},
		openVideo(video) {
			if (video.type === 'thread' && video.id) {
				this.openInFirefox(`/scraper/${video.id}`)
				return
			}
			if (video.url) this.openInFirefox(video.url)
		},
		getPostImages(post) {
			return (post.metadata?.attachments || []).filter((a) => a.type === 'image' && (a.url || a.thumbnail_url))
		},
		getPostLinks(post) {
			return post.metadata?.extracted_links || []
		},
		formatLinkLabel(url) {
			try {
				const u = new URL(url)
				return u.hostname + u.pathname.substring(0, 40)
			} catch {
				return url.substring(0, 60)
			}
		},
		truncateText(text, maxLength) {
			if (!text || text.length <= maxLength) return text
			return text.substring(0, maxLength) + '...'
		},
		getVideos(item) {
			if (!item.metadata) return []
			try {
				const metadata = typeof item.metadata === 'string' ? JSON.parse(item.metadata) : item.metadata
				return metadata.videos || []
			} catch { return [] }
		},
		getVisibleVideos(item) {
			const videos = this.getVideos(item)
			return this.expandedItems[item.id] ? videos : videos.slice(0, 6)
		},
		toggleExpand(itemId) {
			this.expandedItems = { ...this.expandedItems, [itemId]: !this.expandedItems[itemId] }
		},
		handleThumbnailError(e) {
			e.target.style.display = 'none'
			e.target.parentElement.querySelector('.thumbnail-placeholder')?.classList.add('show')
		},
		handleThumbError(video) {
			if (video.thumbnail_url) {
				this.failedThumbs = { ...this.failedThumbs, [video.thumbnail_url]: true }
			}
		},
		formatTimeAgo(dateString) {
			if (!dateString) return ''
			const date = new Date(dateString.replace(' ', 'T'))
			const now = new Date()
			const diffMs = now - date
			const diffMins = Math.floor(diffMs / 60000)
			const diffHours = Math.floor(diffMs / 3600000)
			const diffDays = Math.floor(diffMs / 86400000)
			if (diffMins < 1) return 'Just now'
			if (diffMins < 60) return `${diffMins}m ago`
			if (diffHours < 24) return `${diffHours}h ago`
			if (diffDays < 7) return `${diffDays}d ago`
			return date.toLocaleDateString()
		},
		formatDuration(seconds) {
			if (!seconds) return ''
			const mins = Math.floor(seconds / 60)
			const secs = seconds % 60
			if (mins >= 60) {
				const hours = Math.floor(mins / 60)
				return `${hours}:${(mins % 60).toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
			}
			return `${mins}:${secs.toString().padStart(2, '0')}`
		},
		formatViews(views) {
			if (!views) return '0'
			if (views >= 1000000) return (views / 1000000).toFixed(1) + 'M'
			if (views >= 1000) return (views / 1000).toFixed(1) + 'K'
			return views.toString()
		},
		formatScraperName(type) {
			const names = { simpcity: 'SimpCity', pornhub: 'PornHub', rule34video: 'Rule34Video', luxuretv: 'LuxureTV', sexuria: 'Sexuria', motherless: 'Motherless', leakvids: 'LeakVids', hqporner: 'HQPorner' }
			return names[type] || type
		},
	},
}
</script>

<style scoped>
.dashboard-feed {
	background: rgba(30, 30, 40, 0.6);
	border-radius: 12px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	overflow: hidden;
}

/* Feed header */
.feed-header {
	padding: 1rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}
.feed-header-top { display: flex; align-items: center; justify-content: space-between; }
.feed-header h3 { margin: 0; font-size: 1rem; font-weight: 600; color: rgba(255, 255, 255, 0.9); }
.feed-filters { display: flex; flex-wrap: wrap; gap: 0.375rem; margin-top: 0.75rem; }

.clear-feed-btn {
	display: flex;
	align-items: center;
	gap: 0.25rem;
	padding: 0.25rem 0.625rem;
	background: rgba(239, 68, 68, 0.08);
	border: 1px solid rgba(239, 68, 68, 0.2);
	border-radius: 6px;
	color: rgba(248, 113, 113, 0.7);
	font-size: 0.75rem;
	cursor: pointer;
	transition: all 0.2s;
}
.clear-feed-btn:hover {
	background: rgba(239, 68, 68, 0.18);
	border-color: rgba(239, 68, 68, 0.4);
	color: #f87171;
}

.filter-chip {
	background: rgba(255, 255, 255, 0.06);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 16px;
	color: rgba(255, 255, 255, 0.6);
	font-size: 0.75rem;
	padding: 0.25rem 0.75rem;
	cursor: pointer;
	transition: all 0.2s;
}
.filter-chip:hover { background: rgba(255, 255, 255, 0.1); color: rgba(255, 255, 255, 0.8); }
.filter-chip.active { background: rgba(59, 130, 246, 0.2); border-color: rgba(59, 130, 246, 0.4); color: #60a5fa; }

/* ─── SimpCity Thread Tabs ─────────────────────────────── */
.simpcity-tabs-section {
	border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	background: rgba(168, 85, 247, 0.04);
}

.tabs-header {
	padding: 0.625rem 1.25rem 0;
	border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.tabs-title {
	font-size: 0.6875rem;
	font-weight: 600;
	color: rgba(192, 132, 252, 0.7);
	text-transform: uppercase;
	letter-spacing: 0.05em;
	margin-bottom: 0.5rem;
	display: flex;
	align-items: center;
	gap: 0.375rem;
}

.thread-tabs {
	display: flex;
	gap: 2px;
	overflow-x: auto;
	padding-bottom: 0;
	scrollbar-width: none;
}
.thread-tabs::-webkit-scrollbar { display: none; }

.thread-tab {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	padding: 0.5rem 0.875rem;
	background: none;
	border: none;
	border-radius: 8px 8px 0 0;
	color: rgba(255, 255, 255, 0.45);
	font-size: 0.8125rem;
	cursor: pointer;
	white-space: nowrap;
	transition: all 0.15s;
	border-bottom: 2px solid transparent;
	position: relative;
}
.thread-tab:hover { color: rgba(255, 255, 255, 0.75); background: rgba(255, 255, 255, 0.04); }
.thread-tab.active {
	color: #c084fc;
	border-bottom-color: #c084fc;
	background: rgba(168, 85, 247, 0.08);
}

.tab-name {
	max-width: 180px;
	overflow: hidden;
	text-overflow: ellipsis;
}

.tab-open {
	opacity: 0;
	font-size: 0.625rem;
	color: rgba(255, 255, 255, 0.4);
	transition: opacity 0.15s;
}
.thread-tab:hover .tab-open { opacity: 1; }
.tab-open:hover { color: #c084fc !important; }

/* Posts panel */
.thread-posts-panel {
	max-height: 420px;
	overflow-y: auto;
	scrollbar-width: thin;
	scrollbar-color: rgba(168, 85, 247, 0.2) transparent;
}
.thread-posts-panel::-webkit-scrollbar { width: 4px; }
.thread-posts-panel::-webkit-scrollbar-thumb { background: rgba(168, 85, 247, 0.2); border-radius: 2px; }

.posts-loading, .posts-empty {
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 0.5rem;
	padding: 1.5rem;
	color: rgba(255, 255, 255, 0.3);
	font-size: 0.8125rem;
}

.thread-post {
	padding: 0.875rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.04);
	transition: background 0.15s;
}
.thread-post:hover { background: rgba(255, 255, 255, 0.02); }
.thread-post:last-child { border-bottom: none; }
.thread-post.is-read { opacity: 0.45; }

.post-header {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	margin-bottom: 0.375rem;
}

.unread-dot {
	width: 7px; height: 7px;
	border-radius: 50%;
	background: #818cf8;
	box-shadow: 0 0 6px rgba(129, 140, 248, 0.6);
	flex-shrink: 0;
}

.post-author { font-size: 0.8125rem; font-weight: 600; color: rgba(255, 255, 255, 0.85); }
.post-time { font-size: 0.6875rem; color: rgba(255, 255, 255, 0.3); margin-left: auto; }

.post-text {
	font-size: 0.8125rem;
	color: rgba(255, 255, 255, 0.6);
	line-height: 1.5;
	margin-bottom: 0.5rem;
	word-break: break-word;
}

.post-images {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
	gap: 4px;
	margin-bottom: 0.5rem;
	border-radius: 6px;
	overflow: hidden;
}
.post-images.single-image { grid-template-columns: 1fr; }
.post-images.single-image .post-image { max-height: 240px; }

.post-image {
	aspect-ratio: 16 / 9;
	overflow: hidden;
	background: rgba(0,0,0,0.3);
	cursor: pointer;
	border-radius: 4px;
}
.post-image img { width: 100%; height: 100%; object-fit: cover; display: block; transition: transform 0.2s; }
.post-image:hover img { transform: scale(1.04); }

.post-links { display: flex; flex-direction: column; gap: 3px; }
.post-link {
	display: flex;
	align-items: center;
	gap: 0.375rem;
	padding: 0.3rem 0.625rem;
	background: rgba(99, 102, 241, 0.08);
	border: 1px solid rgba(99, 102, 241, 0.18);
	border-radius: 5px;
	color: #a5b4fc;
	font-size: 0.75rem;
	text-decoration: none;
	cursor: pointer;
	transition: all 0.15s;
	overflow: hidden;
}
.post-link:hover { background: rgba(99, 102, 241, 0.18); color: #c7d2fe; }
.post-link-icon { flex-shrink: 0; opacity: 0.6; font-size: 0.6875rem; }
.post-link-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* ─── Feed items ──────────────────────────────────────── */
.feed-items {
	max-height: calc(100vh - 340px);
	overflow-y: auto;
}

.feed-item {
	display: flex;
	gap: 0.75rem;
	padding: 0.875rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.05);
	transition: background 0.2s;
}
.feed-item:hover { background: rgba(255, 255, 255, 0.03); }
.feed-item:last-child { border-bottom: none; }

.feed-icon {
	width: 32px; height: 32px;
	border-radius: 8px;
	display: flex; align-items: center; justify-content: center;
	flex-shrink: 0;
	font-size: 0.875rem;
}

.feed-item.primary .feed-icon, .feed-item-videos.primary .feed-icon { background: rgba(59,130,246,0.2); color: #60a5fa; }
.feed-item.success .feed-icon, .feed-item-videos.success .feed-icon { background: rgba(34,197,94,0.2); color: #4ade80; }
.feed-item.warning .feed-icon, .feed-item-videos.warning .feed-icon { background: rgba(234,179,8,0.2); color: #facc15; }
.feed-item.danger .feed-icon, .feed-item-videos.danger .feed-icon { background: rgba(239,68,68,0.2); color: #f87171; }
.feed-item.info .feed-icon, .feed-item-videos.info .feed-icon { background: rgba(6,182,212,0.2); color: #22d3ee; }
.feed-item.purple .feed-icon, .feed-item-videos.purple .feed-icon { background: rgba(168,85,247,0.2); color: #c084fc; }
.feed-item.orange .feed-icon, .feed-item-videos.orange .feed-icon { background: rgba(249,115,22,0.2); color: #fb923c; }
.feed-item.pink .feed-icon, .feed-item-videos.pink .feed-icon { background: rgba(236,72,153,0.2); color: #f472b6; }

.feed-content { flex: 1; min-width: 0; }
.feed-title { font-weight: 600; font-size: 0.875rem; color: rgba(255,255,255,0.9); margin-bottom: 0.125rem; }
.feed-message { font-size: 0.8125rem; color: rgba(255,255,255,0.6); margin-bottom: 0.25rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.feed-time { font-size: 0.75rem; color: rgba(255,255,255,0.4); }

.feed-item-videos { padding: 1rem 1.25rem; border-bottom: 1px solid rgba(255,255,255,0.05); }
.feed-item-header { display: flex; align-items: center; gap: 0.75rem; margin-bottom: 0.75rem; }
.feed-header-content { flex: 1; }
.feed-subtitle { font-size: 0.75rem; color: rgba(255,255,255,0.5); }

.video-cards-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.75rem; }
.video-mini-card { background: rgba(255,255,255,0.05); border-radius: 8px; overflow: hidden; cursor: pointer; transition: all 0.2s; }
.video-mini-card:hover { background: rgba(255,255,255,0.1); transform: translateY(-2px); }

.video-thumbnail { position: relative; width: 100%; aspect-ratio: 16/9; background: rgba(0,0,0,0.3); overflow: hidden; }
.video-thumbnail img { width: 100%; height: 100%; object-fit: cover; }
.thumbnail-placeholder { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; color: rgba(255,255,255,0.3); font-size: 1.5rem; }
.thumbnail-placeholder.show { display: flex; }
.video-duration { position: absolute; bottom: 4px; right: 4px; background: rgba(0,0,0,0.8); color: #fff; font-size: 0.6875rem; padding: 2px 4px; border-radius: 3px; font-weight: 500; }

.video-info { padding: 0.5rem; }
.video-title { font-size: 0.75rem; font-weight: 500; color: rgba(255,255,255,0.9); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; margin-bottom: 0.125rem; }
.video-meta { font-size: 0.6875rem; color: rgba(255,255,255,0.5); }

.more-videos { margin-top: 0.5rem; font-size: 0.75rem; color: rgba(255,255,255,0.5); text-align: center; }
.more-videos-btn { cursor: pointer; padding: 0.375rem 0.75rem; border-radius: 6px; transition: all 0.2s; user-select: none; }
.more-videos-btn:hover { background: rgba(255,255,255,0.07); color: rgba(255,255,255,0.8); }

.thread-card { display: flex; flex-direction: row; align-items: center; gap: 0.5rem; grid-column: span 3; padding: 0.5rem 0.75rem; }
.thread-icon { width: 32px; height: 32px; border-radius: 6px; background: rgba(168,85,247,0.2); color: #c084fc; display: flex; align-items: center; justify-content: center; flex-shrink: 0; font-size: 0.875rem; }
.thread-card .video-info { padding: 0; flex: 1; min-width: 0; }
.thread-badge { background: rgba(168,85,247,0.2); color: #c084fc; padding: 1px 6px; border-radius: 4px; font-size: 0.625rem; font-weight: 600; text-transform: uppercase; }

.empty-state { padding: 2rem; text-align: center; color: rgba(255,255,255,0.4); }
.empty-icon { font-size: 2rem; margin-bottom: 0.5rem; opacity: 0.5; }
.empty-state p { margin: 0; font-size: 0.875rem; }

.feed-footer { padding: 0.5rem 1rem; text-align: center; border-top: 1px solid rgba(255,255,255,0.05); }
.feed-footer .btn-link { color: rgba(255,255,255,0.5); font-size: 0.8125rem; background: none; border: none; cursor: pointer; }
.feed-footer .btn-link:hover { color: rgba(255,255,255,0.8); }

.feed-enter-active { transition: opacity 0.3s ease; }
.feed-leave-active { display: none; }
.feed-enter-from { opacity: 0; }
.feed-move { transition: transform 0.3s ease; }

.me-2 { margin-right: 0.5rem; }

/* ─── YouTube-style video grid ──────────────────────────── */
.yt-feed-grid {
	display: grid;
	grid-template-columns: repeat(4, 1fr);
	gap: 1rem;
	padding: 1rem 1.25rem 1.25rem;
}

.yt-video-card {
	cursor: pointer;
	border-radius: 8px;
	overflow: hidden;
	transition: transform 0.2s;
}
.yt-video-card:hover { transform: translateY(-3px); }

.yt-thumb-wrap {
	position: relative;
	width: 100%;
	aspect-ratio: 16 / 9;
	background: rgba(0, 0, 0, 0.4);
	border-radius: 6px;
	overflow: hidden;
}
.yt-thumb-wrap img { width: 100%; height: 100%; object-fit: cover; display: block; }

.yt-no-thumb {
	position: absolute;
	inset: 0;
	display: flex;
	align-items: center;
	justify-content: center;
	color: rgba(255, 255, 255, 0.18);
	font-size: 1.75rem;
	background: rgba(255, 255, 255, 0.03);
}

.yt-dur {
	position: absolute;
	bottom: 5px;
	right: 5px;
	background: rgba(0, 0, 0, 0.85);
	color: #fff;
	font-size: 0.6875rem;
	font-weight: 600;
	padding: 2px 5px;
	border-radius: 3px;
	pointer-events: none;
}

.yt-card-body { padding: 0.5rem 0.125rem; }
.yt-card-title {
	font-size: 0.8125rem;
	font-weight: 600;
	color: rgba(255, 255, 255, 0.9);
	overflow: hidden;
	text-overflow: ellipsis;
	display: -webkit-box;
	-webkit-line-clamp: 2;
	-webkit-box-orient: vertical;
	line-height: 1.35;
	margin-bottom: 0.25rem;
}
.yt-card-source { font-size: 0.75rem; color: rgba(255, 255, 255, 0.45); margin-bottom: 0.125rem; }
.yt-card-meta { font-size: 0.6875rem; color: rgba(255, 255, 255, 0.3); }

/* Activity log (non-video items below the grid) */
.activity-log-items { border-top: 1px solid rgba(255, 255, 255, 0.06); }
.activity-log-label {
	padding: 0.625rem 1.25rem 0.25rem;
	font-size: 0.6875rem;
	font-weight: 600;
	text-transform: uppercase;
	letter-spacing: 0.05em;
	color: rgba(255, 255, 255, 0.28);
}

@media (max-width: 1200px) { .yt-feed-grid { grid-template-columns: repeat(3, 1fr); } }
@media (max-width: 800px) { .yt-feed-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 600px) {
	.video-cards-grid { grid-template-columns: repeat(2, 1fr); }
	.yt-feed-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 480px) { .yt-feed-grid { grid-template-columns: 1fr; } }
</style>
