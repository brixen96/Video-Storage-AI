<template>
	<div class="dashboard-feed">
		<div class="feed-header">
			<h3>
				<font-awesome-icon :icon="['fas', 'rss']" class="me-2" />
				Activity Feed
			</h3>
		</div>
		<div class="feed-items" v-if="items.length > 0">
			<TransitionGroup name="feed">
				<div v-for="item in items" :key="item.id">
					<!-- Video content item -->
					<div v-if="item.item_type === 'new_content' && getVideos(item).length > 0" class="feed-item-videos" :class="item.color">
						<div class="feed-item-header">
							<div class="feed-icon">
								<font-awesome-icon :icon="['fas', item.icon || 'bell']" />
							</div>
							<div class="feed-header-content">
								<div class="feed-title">{{ item.title }}</div>
								<div class="feed-subtitle">{{ item.message }} &bull; {{ formatTimeAgo(item.created_at) }}</div>
							</div>
						</div>
						<div class="video-cards-grid">
							<div
								v-for="(video, idx) in getVideos(item).slice(0, 6)"
								:key="idx"
								class="video-mini-card"
								@click="openVideo(video)"
							>
								<div class="video-thumbnail">
									<img
										v-if="video.thumbnail_url"
										:src="video.thumbnail_url"
										:alt="video.title"
										@error="handleThumbnailError"
									/>
									<div v-else class="thumbnail-placeholder">
										<font-awesome-icon :icon="['fas', 'video']" />
									</div>
									<span class="video-duration" v-if="video.duration">{{ formatDuration(video.duration) }}</span>
								</div>
								<div class="video-info">
									<div class="video-title">{{ video.title }}</div>
									<div class="video-meta">
										<span v-if="video.views">{{ formatViews(video.views) }} views</span>
									</div>
								</div>
							</div>
						</div>
						<div v-if="getVideos(item).length > 6" class="more-videos">
							+{{ getVideos(item).length - 6 }} more videos
						</div>
					</div>

					<!-- Regular feed item -->
					<div v-else class="feed-item" :class="item.color">
						<div class="feed-icon">
							<font-awesome-icon :icon="['fas', item.icon || 'bell']" />
						</div>
						<div class="feed-content">
							<div class="feed-title">{{ item.title }}</div>
							<div class="feed-message">{{ item.message }}</div>
							<div class="feed-time">{{ formatTimeAgo(item.created_at) }}</div>
						</div>
					</div>
				</div>
			</TransitionGroup>
		</div>
		<div class="empty-state" v-else>
			<font-awesome-icon :icon="['fas', 'inbox']" class="empty-icon" />
			<p>No recent activity</p>
		</div>
		<div class="feed-footer" v-if="items.length > 0 && hasMore">
			<button class="btn btn-link" @click="$emit('load-more')">Load more</button>
		</div>
	</div>
</template>

<script>
export default {
	name: 'DashboardFeed',
	props: {
		items: {
			type: Array,
			default: () => [],
		},
		hasMore: {
			type: Boolean,
			default: false,
		},
	},
	emits: ['load-more'],
	methods: {
		formatTimeAgo(dateString) {
			if (!dateString) return ''
			const date = new Date(dateString)
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
				const remainingMins = mins % 60
				return `${hours}:${remainingMins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
			}
			return `${mins}:${secs.toString().padStart(2, '0')}`
		},
		formatViews(views) {
			if (!views) return '0'
			if (views >= 1000000) return (views / 1000000).toFixed(1) + 'M'
			if (views >= 1000) return (views / 1000).toFixed(1) + 'K'
			return views.toString()
		},
		getVideos(item) {
			if (!item.metadata) return []
			try {
				const metadata = typeof item.metadata === 'string' ? JSON.parse(item.metadata) : item.metadata
				return metadata.videos || []
			} catch {
				return []
			}
		},
		openVideo(video) {
			if (video.url) {
				window.open(video.url, '_blank')
			}
		},
		handleThumbnailError(e) {
			e.target.style.display = 'none'
			e.target.parentElement.querySelector('.thumbnail-placeholder')?.classList.add('show')
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

.feed-header {
	padding: 1rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.feed-header h3 {
	margin: 0;
	font-size: 1rem;
	font-weight: 600;
	color: rgba(255, 255, 255, 0.9);
}

.feed-items {
	max-height: 600px;
	overflow-y: auto;
}

/* Regular feed item */
.feed-item {
	display: flex;
	gap: 0.75rem;
	padding: 0.875rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.05);
	transition: background 0.2s;
}

.feed-item:hover {
	background: rgba(255, 255, 255, 0.03);
}

.feed-item:last-child {
	border-bottom: none;
}

.feed-icon {
	width: 32px;
	height: 32px;
	border-radius: 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0;
	font-size: 0.875rem;
}

.feed-item.primary .feed-icon,
.feed-item-videos.primary .feed-icon {
	background: rgba(59, 130, 246, 0.2);
	color: #60a5fa;
}

.feed-item.success .feed-icon,
.feed-item-videos.success .feed-icon {
	background: rgba(34, 197, 94, 0.2);
	color: #4ade80;
}

.feed-item.warning .feed-icon,
.feed-item-videos.warning .feed-icon {
	background: rgba(234, 179, 8, 0.2);
	color: #facc15;
}

.feed-item.danger .feed-icon,
.feed-item-videos.danger .feed-icon {
	background: rgba(239, 68, 68, 0.2);
	color: #f87171;
}

.feed-item.info .feed-icon,
.feed-item-videos.info .feed-icon {
	background: rgba(6, 182, 212, 0.2);
	color: #22d3ee;
}

.feed-item.purple .feed-icon,
.feed-item-videos.purple .feed-icon {
	background: rgba(168, 85, 247, 0.2);
	color: #c084fc;
}

.feed-item.orange .feed-icon,
.feed-item-videos.orange .feed-icon {
	background: rgba(249, 115, 22, 0.2);
	color: #fb923c;
}

.feed-item.pink .feed-icon,
.feed-item-videos.pink .feed-icon {
	background: rgba(236, 72, 153, 0.2);
	color: #f472b6;
}

.feed-content {
	flex: 1;
	min-width: 0;
}

.feed-title {
	font-weight: 600;
	font-size: 0.875rem;
	color: rgba(255, 255, 255, 0.9);
	margin-bottom: 0.125rem;
}

.feed-message {
	font-size: 0.8125rem;
	color: rgba(255, 255, 255, 0.6);
	margin-bottom: 0.25rem;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.feed-time {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.4);
}

/* Video feed item */
.feed-item-videos {
	padding: 1rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.feed-item-header {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	margin-bottom: 0.75rem;
}

.feed-header-content {
	flex: 1;
}

.feed-subtitle {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.5);
}

.video-cards-grid {
	display: grid;
	grid-template-columns: repeat(3, 1fr);
	gap: 0.75rem;
}

.video-mini-card {
	background: rgba(255, 255, 255, 0.05);
	border-radius: 8px;
	overflow: hidden;
	cursor: pointer;
	transition: all 0.2s;
}

.video-mini-card:hover {
	background: rgba(255, 255, 255, 0.1);
	transform: translateY(-2px);
}

.video-thumbnail {
	position: relative;
	width: 100%;
	aspect-ratio: 16 / 9;
	background: rgba(0, 0, 0, 0.3);
	overflow: hidden;
}

.video-thumbnail img {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.thumbnail-placeholder {
	position: absolute;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	display: flex;
	align-items: center;
	justify-content: center;
	color: rgba(255, 255, 255, 0.3);
	font-size: 1.5rem;
}

.thumbnail-placeholder.show {
	display: flex;
}

.video-duration {
	position: absolute;
	bottom: 4px;
	right: 4px;
	background: rgba(0, 0, 0, 0.8);
	color: #fff;
	font-size: 0.6875rem;
	padding: 2px 4px;
	border-radius: 3px;
	font-weight: 500;
}

.video-info {
	padding: 0.5rem;
}

.video-title {
	font-size: 0.75rem;
	font-weight: 500;
	color: rgba(255, 255, 255, 0.9);
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
	margin-bottom: 0.125rem;
}

.video-meta {
	font-size: 0.6875rem;
	color: rgba(255, 255, 255, 0.5);
}

.more-videos {
	margin-top: 0.5rem;
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.5);
	text-align: center;
}

/* Empty state */
.empty-state {
	padding: 2rem;
	text-align: center;
	color: rgba(255, 255, 255, 0.4);
}

.empty-icon {
	font-size: 2rem;
	margin-bottom: 0.5rem;
	opacity: 0.5;
}

.empty-state p {
	margin: 0;
	font-size: 0.875rem;
}

.feed-footer {
	padding: 0.5rem 1rem;
	text-align: center;
	border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.feed-footer .btn-link {
	color: rgba(255, 255, 255, 0.5);
	font-size: 0.8125rem;
	background: none;
	border: none;
	cursor: pointer;
}

.feed-footer .btn-link:hover {
	color: rgba(255, 255, 255, 0.8);
}

/* Transition animations */
.feed-enter-active,
.feed-leave-active {
	transition: all 0.3s ease;
}

.feed-enter-from {
	opacity: 0;
	transform: translateX(-20px);
}

.feed-leave-to {
	opacity: 0;
	transform: translateX(20px);
}

/* Responsive */
@media (max-width: 600px) {
	.video-cards-grid {
		grid-template-columns: repeat(2, 1fr);
	}
}
</style>
