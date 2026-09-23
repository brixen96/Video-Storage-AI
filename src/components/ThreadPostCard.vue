<template>
	<div class="post-card" :class="postCardHealthClass">
		<!-- Post Header -->
		<div class="post-header">
			<div class="post-author">
				<div class="author-avatar">{{ post.author.charAt(0).toUpperCase() }}</div>
				<div class="author-info">
					<div class="author-name">{{ post.author }}</div>
					<div class="post-date">{{ formatDate(post.posted_at) }}</div>
				</div>
			</div>
			<div class="post-meta">
				<span v-if="visibleImages.length > 0" class="meta-badge">
					<font-awesome-icon :icon="['fas', 'image']" />
					{{ visibleImages.length }}
				</span>
				<span v-if="links.length > 0" class="meta-badge">
					<font-awesome-icon :icon="['fas', 'download']" />
					{{ links.length }}
				</span>
				<span v-if="links.length > 0 && hasVerifiedLinks" class="meta-badge" :class="postHealthClass">
					<font-awesome-icon :icon="postHealthIcon" />
					{{ postHealthSummary }}
				</span>
			</div>
		</div>

		<!-- Post Content -->
		<div v-if="post.content" class="post-content" v-html="sanitizeHTML(post.content)"></div>

		<!-- Images and Links Combined -->
		<div v-if="visibleImages.length > 0 || links.length > 0" class="post-media">
			<!-- Image Gallery -->
			<div v-if="visibleImages.length > 0" class="media-grid">
				<div v-for="(image, idx) in visibleImages" :key="idx" class="media-item">
					<div class="image-wrapper" @click="openImage(image.url)">
						<img :src="image.thumbnail_url || image.url" :alt="`Image ${idx + 1}`" @error="onImageError(idx)" />
						<div class="image-overlay">
							<font-awesome-icon :icon="['fas', 'search-plus']" />
						</div>
					</div>
				</div>
			</div>

			<!-- Download Links -->
			<div v-if="links.length > 0" class="download-section">
				<div class="download-header">
					<font-awesome-icon :icon="['fas', 'download']" />
					<span>Download Links ({{ links.length }})</span>
				</div>
				<div class="download-links">
					<div
						v-for="link in links"
						:key="link.id"
						class="download-link-wrapper"
						:class="[`status-${link.download_status || 'pending'}`, `health-${link.status || 'unknown'}`]"
					>
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
							<button v-if="link.download_status !== 'downloaded'" @click.stop="$emit('mark-downloaded', link.id)" class="action-btn success-btn" title="Mark as downloaded">
								<font-awesome-icon :icon="['fas', 'check']" />
							</button>
							<button v-if="link.download_status !== 'failed'" @click.stop="$emit('mark-failed', link.id)" class="action-btn danger-btn" title="Mark as failed">
								<font-awesome-icon :icon="['fas', 'times']" />
							</button>
							<button v-if="link.download_status && link.download_status !== 'pending'" @click.stop="$emit('reset-status', link.id)" class="action-btn reset-btn" title="Reset status">
								<font-awesome-icon :icon="['fas', 'undo']" />
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed } from 'vue'
import DOMPurify from 'dompurify'
import { openInFirefox } from '@/services/api'

const props = defineProps({
	post: { type: Object, required: true },
	links: { type: Array, default: () => [] },
})
defineEmits(['mark-downloaded', 'mark-failed', 'reset-status'])

const postImages = computed(() => props.post.metadata?.attachments || [])

// Track broken images by index - hidden by default once error is detected
const brokenImages = ref(new Set())

const visibleImages = computed(() =>
	postImages.value.filter((_, idx) => !brokenImages.value.has(idx))
)

const onImageError = (idx) => {
	brokenImages.value.add(idx)
}

const hasVerifiedLinks = computed(() =>
	props.links.some((l) => l.status === 'active' || l.status === 'dead' || l.status === 'expired')
)

const postHealthSummary = computed(() => {
	if (props.links.length === 0) return ''
	const active = props.links.filter((l) => l.status === 'active').length
	return `${active}/${props.links.length}`
})

const postHealthClass = computed(() => {
	if (props.links.length === 0) return ''
	const active = props.links.filter((l) => l.status === 'active').length
	const dead = props.links.filter((l) => l.status === 'dead').length
	if (active === props.links.length) return 'health-all-active'
	if (dead === props.links.length) return 'health-all-dead'
	return 'health-mixed'
})

const postHealthIcon = computed(() => {
	if (props.links.length === 0) return ['fas', 'link']
	const active = props.links.filter((l) => l.status === 'active').length
	if (active === props.links.length) return ['fas', 'check-circle']
	if (active === 0) return ['fas', 'times-circle']
	return ['fas', 'exclamation-triangle']
})

const postCardHealthClass = computed(() => {
	if (props.links.length === 0) return ''
	if (!hasVerifiedLinks.value) return ''
	const active = props.links.filter((l) => l.status === 'active').length
	if (active === props.links.length) return 'post-health-active'
	if (active === 0) return 'post-health-dead'
	return 'post-health-mixed'
})

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
	if (url.length > 50) return url.substring(0, 47) + '...'
	return url
}

const getLinkHealthIcon = (status) => {
	switch (status) {
		case 'active': return ['fas', 'check-circle']
		case 'dead': return ['fas', 'skull-crossbones']
		case 'expired': return ['fas', 'hourglass-end']
		default: return ['fas', 'question-circle']
	}
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

const sanitizeHTML = (html) => {
	return DOMPurify.sanitize(html, {
		ALLOWED_TAGS: ['p', 'br', 'b', 'i', 'u', 'strong', 'em', 'a', 'ul', 'ol', 'li', 'blockquote', 'code', 'pre'],
		ALLOWED_ATTR: ['href', 'target', 'rel'],
	})
}

const openImage = (url) => openInFirefox(url)
</script>

<style scoped>
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

/* Hide broken images in HTML content - they take up blank space otherwise */
.post-content :deep(img) {
	max-width: 100%;
	border-radius: 8px;
}

.post-content :deep(img[src=""]),
.post-content :deep(img:not([src])) {
	display: none;
}

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

.provider-gofile .link-icon { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.provider-pixeldrain .link-icon { background: rgba(139, 92, 246, 0.15); color: #8b5cf6; }
.provider-bunkr .link-icon,
.provider-bunkrr .link-icon { background: rgba(236, 72, 153, 0.15); color: #ec4899; }
.provider-cyberdrop .link-icon { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
.provider-mediafire .link-icon { background: rgba(251, 146, 60, 0.15); color: #fb923c; }
.provider-mega .link-icon { background: rgba(239, 68, 68, 0.15); color: #ef4444; }

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

.provider-gofile .link-provider { color: #3b82f6; }
.provider-pixeldrain .link-provider { color: #8b5cf6; }
.provider-bunkr .link-provider,
.provider-bunkrr .link-provider { color: #ec4899; }
.provider-cyberdrop .link-provider { color: #22c55e; }
.provider-mediafire .link-provider { color: #fb923c; }
.provider-mega .link-provider { color: #ef4444; }

.link-url {
	font-size: 0.85rem;
	color: rgba(255, 255, 255, 0.5);
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

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

.download-link-wrapper.health-active { border-color: rgba(81, 207, 102, 0.5); }
.download-link-wrapper.health-dead { border-color: rgba(255, 107, 107, 0.5); }
.download-link-wrapper.health-expired { border-color: rgba(255, 169, 77, 0.5); }
.download-link-wrapper.status-downloaded { opacity: 0.7; }
.download-link-wrapper .download-link { flex: 1; min-width: 0; }

.link-actions {
	display: flex;
	gap: 0.25rem;
	flex-shrink: 0;
}

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

.link-health-badge.health-active { background: rgba(81, 207, 102, 0.15); color: #51cf66; }
.link-health-badge.health-dead { background: rgba(255, 107, 107, 0.15); color: #ff6b6b; }
.link-health-badge.health-expired { background: rgba(255, 169, 77, 0.15); color: #ffa94d; }

.download-link-wrapper.health-dead .download-link { opacity: 0.45; }
.download-link-wrapper.health-dead .link-url { text-decoration: line-through; color: rgba(255, 107, 107, 0.5); }
.download-link-wrapper.health-expired .link-url { color: rgba(255, 169, 77, 0.6); }

.meta-badge.health-all-active { background: rgba(81, 207, 102, 0.15); color: #51cf66; border: 1px solid rgba(81, 207, 102, 0.3); }
.meta-badge.health-all-dead { background: rgba(255, 107, 107, 0.15); color: #ff6b6b; border: 1px solid rgba(255, 107, 107, 0.3); }
.meta-badge.health-mixed { background: rgba(255, 169, 77, 0.15); color: #ffa94d; border: 1px solid rgba(255, 169, 77, 0.3); }

.post-card.post-health-active { border-left: 3px solid #51cf66; }
.post-card.post-health-dead { border-left: 3px solid #ff6b6b; }
.post-card.post-health-mixed { border-left: 3px solid #ffa94d; }

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

.action-btn:hover { transform: scale(1.1); }
.success-btn { background: rgba(81, 207, 102, 0.15); color: #51cf66; }
.success-btn:hover { background: rgba(81, 207, 102, 0.25); }
.danger-btn { background: rgba(255, 107, 107, 0.15); color: #ff6b6b; }
.danger-btn:hover { background: rgba(255, 107, 107, 0.25); }
.reset-btn { background: rgba(255, 255, 255, 0.05); color: rgba(255, 255, 255, 0.6); }
.reset-btn:hover { background: rgba(255, 255, 255, 0.1); color: #fff; }

@media (max-width: 768px) {
	.download-links { grid-template-columns: 1fr; }
	.media-grid { grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); }
}
</style>
