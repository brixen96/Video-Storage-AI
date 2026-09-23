<template>
	<div class="video-card" :class="{ selected: isSelected }" @click="selectionMode && emit('toggle-selection', video.id)">
		<!-- Selection checkbox -->
		<div v-if="selectionMode" class="video-checkbox" @click.stop="emit('toggle-selection', video.id)">
			<input type="checkbox" :checked="isSelected" @change.stop />
		</div>

		<!-- Thumbnail with animated frame preview on hover -->
		<div class="video-thumbnail" @mouseenter="startPreview" @mouseleave="stopPreview">
			<img v-if="video.thumbnail_url && !imgError && !isHovering" :src="proxiedThumbnailUrl" :alt="video.title" @error="imgError = true" />
			<!-- Prefer frame animation when frames are available -->
			<img v-else-if="video.thumbnail_url && !imgError && isHovering && proxiedThumbFrames.length > 1" :src="thumbnailSrc" :alt="video.title" @error="imgError = true" />
			<!-- Fall back to video preview only when no frames available -->
			<video v-else-if="isHovering && previewUrl && !previewError" :src="previewUrl" class="video-preview-overlay" autoplay muted loop playsinline @error="onPreviewError" />
			<!-- Final fallback: static thumbnail while hovering with no preview -->
			<img v-else-if="video.thumbnail_url && !imgError && isHovering" :src="proxiedThumbnailUrl" :alt="video.title" @error="imgError = true" />
			<div v-if="!video.thumbnail_url || imgError" class="no-thumbnail">
				<font-awesome-icon :icon="['fas', 'video']" size="3x" />
			</div>
			<span v-if="video.duration" class="duration-badge">
				{{ formatDuration(video.duration) }}
			</span>
			<span v-if="video.downloaded_at" class="downloaded-badge"> <font-awesome-icon :icon="['fas', 'check']" class="me-1" />Downloaded </span>
		</div>

		<!-- Info -->
		<div class="video-info">
			<h6 class="video-title" :title="video.title">{{ video.title }}</h6>

			<!-- Uploader / author — clickable to filter -->
			<span
				v-if="video.uploader || video.author"
				class="author-chip"
				@click.stop="emit('filter-author', video.uploader || video.author)"
				:title="'Filter by: ' + (video.uploader || video.author)"
			>
				<font-awesome-icon :icon="['fas', 'user']" class="me-1" />
				{{ video.uploader || video.author }}
			</span>

			<!-- Added timestamp -->
			<span v-if="formatTimeAgo(video.created_at)" class="video-added-time">
				<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
				{{ formatTimeAgo(video.created_at) }}
			</span>

			<!-- View count + rating -->
			<div v-if="video.view_count > 0 || video.rating > 0" class="d-flex flex-wrap gap-2 mt-1">
				<span v-if="video.view_count > 0" class="text-muted small">
					<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
					{{ formatCompactNumber(video.view_count) }}
				</span>
				<span v-if="video.rating > 0" class="text-success small">
					<font-awesome-icon :icon="['fas', 'thumbs-up']" class="me-1" />
					{{ video.rating }}%
				</span>
			</div>

			<!-- Category badge -->
			<div v-if="video.category" class="video-category mt-1">
				<span class="category-chip" @click.stop="emit('filter-category', video.category)" :title="'Filter: ' + video.category">
					<font-awesome-icon :icon="['fas', 'layer-group']" class="me-1" />{{ video.category }}
				</span>
			</div>

			<!-- Matched performers (from metadata) -->
			<div v-if="performers.length" class="scraped-performers mt-1">
				<router-link v-for="p in performers" :key="p.name" :to="p.id > 0 ? '/performers/' + p.id : '#'" :class="['performer-chip', { matched: p.id > 0 }]" @click.prevent="p.id > 0 ? $router.push('/performers/' + p.id) : null">{{ p.name }}</router-link>
			</div>

			<!-- Matched tags (internal library) -->
			<div v-if="matchedTags.length" class="scraped-matched-tags mt-1">
				<span v-for="t in matchedTags" :key="t.id" class="tag-chip" :style="{ backgroundColor: t.color || '#6c757d' }">
					<font-awesome-icon v-if="t.icon" :icon="['fas', t.icon]" class="me-1" />
					{{ t.name }}
				</span>
			</div>

			<!-- Raw scraped tags -->
			<div v-if="visibleTags.length" class="scraped-raw-tags mt-1">
				<span
					v-for="t in visibleTags"
					:key="t"
					class="raw-tag-chip"
					@click.stop="emit('filter-tag', t)"
					:title="'Filter: ' + t"
				>{{ t }}</span>
				<span v-if="extraTagCount > 0" class="raw-tag-chip raw-tag-more" :title="hiddenTags.join(', ')">+{{ extraTagCount }}</span>
			</div>

			<!-- Action buttons (provided by parent via slot) -->
			<div class="video-actions mt-2">
				<slot name="actions" :video="video" />
			</div>
		</div>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
import { ref, computed, onBeforeUnmount } from 'vue'
import { useFormatters } from '@/composables/useFormatters'
import { getProxiedImageURL } from '@/services/api'

const props = defineProps({
	video: { type: Object, required: true },
	selectionMode: { type: Boolean, default: false },
	isSelected: { type: Boolean, default: false },
})

const emit = defineEmits(['toggle-selection', 'filter-tag', 'filter-category', 'filter-author'])

const { formatDuration, formatCompactNumber } = useFormatters()

const formatTimeAgo = (dateStr) => {
	if (!dateStr) return null
	const date = new Date(dateStr)
	if (isNaN(date)) return null
	const diffMs = Date.now() - date
	const diffMins = Math.floor(diffMs / 60000)
	const diffHours = Math.floor(diffMs / 3600000)
	const diffDays = Math.floor(diffMs / 86400000)
	if (diffMins < 1) return 'Just now'
	if (diffMins < 60) return `${diffMins}m ago`
	if (diffHours < 24) return `${diffHours}h ago`
	if (diffDays < 7) return `${diffDays}d ago`
	return date.toLocaleDateString()
}

// ── Metadata parsing ──────────────────────────────────────────────────────────

const parsedMeta = computed(() => {
	if (!props.video.metadata) return {}
	try {
		return typeof props.video.metadata === 'string' ? JSON.parse(props.video.metadata) : props.video.metadata
	} catch {
		return {}
	}
})

const performers = computed(() => parsedMeta.value.performers || [])
const matchedTags = computed(() => parsedMeta.value.matched_tags || [])
const thumbFrames = computed(() => parsedMeta.value.thumbnail_frames || [])
const previewUrl = computed(() => parsedMeta.value.preview_url || parsedMeta.value.video_preview_url || null)

// Raw scraped tags — show up to 6, collapse the rest into "+N"
const MAX_TAGS = 6
const rawTags = computed(() => {
	const t = props.video.tags
	if (!t || !Array.isArray(t)) return []
	return t
})
const visibleTags = computed(() => rawTags.value.slice(0, MAX_TAGS))
const hiddenTags = computed(() => rawTags.value.slice(MAX_TAGS))
const extraTagCount = computed(() => hiddenTags.value.length)

// Proxy all CDN URLs through the backend to satisfy hotlink-protection Referer checks
const proxiedThumbnailUrl = computed(() => getProxiedImageURL(props.video.thumbnail_url))
const proxiedThumbFrames = computed(() => thumbFrames.value.map((f) => getProxiedImageURL(f)))

// ── Thumbnail / video preview animation ──────────────────────────────────────

const isHovering = ref(false)
const frameIdx = ref(0)
const previewError = ref(false)
const imgError = ref(false)
let previewTimer = null

const thumbnailSrc = computed(() => {
	if (proxiedThumbFrames.value.length > 0 && isHovering.value) {
		return proxiedThumbFrames.value[frameIdx.value] || proxiedThumbnailUrl.value
	}
	return proxiedThumbnailUrl.value
})

const startFrameCycle = () => {
	if (proxiedThumbFrames.value.length <= 1 || previewTimer) return
	frameIdx.value = 0
	previewTimer = setInterval(() => {
		frameIdx.value = (frameIdx.value + 1) % proxiedThumbFrames.value.length
	}, 300)
}

const startPreview = () => {
	isHovering.value = true
	// Always attempt frame cycling — it's a no-op when no frames are available.
	// Frames take priority over video preview in the template.
	startFrameCycle()
}

const onPreviewError = () => {
	previewError.value = true
	startFrameCycle()
}

const stopPreview = () => {
	isHovering.value = false
	if (previewTimer) clearInterval(previewTimer)
	previewTimer = null
	frameIdx.value = 0
}

onBeforeUnmount(stopPreview)
</script>

<style>
/* Global layout for the grid that wraps ScrapedVideoCard instances.
   Must be non-scoped so it applies inside child tab components
   (ScraperPage.vue imports scraper_page.css as scoped, so the grid
   rule would otherwise be stripped from child component elements). */
.video-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
	gap: 1.25rem;
	padding: 1.5rem;
}

.video-card {
	position: relative;
	background: rgba(255, 255, 255, 0.05);
	backdrop-filter: blur(10px);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	overflow: hidden;
	transition: all 0.3s ease;
	cursor: pointer;
}

.video-card:hover {
	background: rgba(255, 255, 255, 0.1);
	border-color: rgba(77, 171, 247, 0.5);
	box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

.video-card.selected {
	border-color: rgba(77, 171, 247, 0.8);
	background: rgba(77, 171, 247, 0.15);
	box-shadow: 0 0 0 2px rgba(77, 171, 247, 0.3);
}

.video-checkbox {
	position: absolute;
	top: 8px;
	left: 8px;
	z-index: 10;
	background: rgba(0, 0, 0, 0.7);
	border-radius: 4px;
	padding: 4px 8px;
	cursor: pointer;
}

.video-checkbox input[type='checkbox'] {
	width: 18px;
	height: 18px;
	cursor: pointer;
	accent-color: #4dabf7;
}

.video-thumbnail {
	position: relative;
	width: 100%;
	height: 0;
	padding-bottom: 56.25%;
	background: rgba(0, 0, 0, 0.3);
	overflow: hidden;
}

.video-thumbnail img {
	position: absolute;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	object-fit: cover;
	transition: transform 0.3s ease;
}

.video-preview-overlay {
	position: absolute;
	inset: 0;
	width: 100%;
	height: 100%;
	object-fit: cover;
	z-index: 2;
}

.video-thumbnail .no-thumbnail {
	position: absolute;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	display: flex;
	align-items: center;
	justify-content: center;
	color: rgba(255, 255, 255, 0.3);
	background: rgba(0, 0, 0, 0.2);
}

.duration-badge {
	position: absolute;
	bottom: 8px;
	right: 8px;
	background: rgba(0, 0, 0, 0.85);
	backdrop-filter: blur(4px);
	color: #fff;
	padding: 0.25rem 0.5rem;
	border-radius: 4px;
	font-size: 0.75rem;
	font-weight: 600;
	letter-spacing: 0.5px;
}

.downloaded-badge {
	position: absolute;
	top: 8px;
	left: 8px;
	background: rgba(40, 167, 69, 0.9);
	backdrop-filter: blur(4px);
	color: #fff;
	padding: 0.2rem 0.5rem;
	border-radius: 4px;
	font-size: 0.7rem;
	font-weight: 600;
	letter-spacing: 0.3px;
	pointer-events: none;
}

.video-info {
	padding: 1rem;
}

.video-title {
	font-size: 0.95rem;
	font-weight: 600;
	margin-bottom: 0.5rem;
	color: #fff;
	overflow: hidden;
	text-overflow: ellipsis;
	display: -webkit-box;
	-webkit-line-clamp: 2;
	-webkit-box-orient: vertical;
	line-height: 1.4;
	min-height: 2.8rem;
}

.video-actions {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: 0.5rem;
	margin-top: 0.5rem;
}

.video-actions .btn {
	font-size: 0.85rem;
	padding: 0.4rem 0.75rem;
	white-space: nowrap;
}

.video-actions .btn-primary {
	flex: 1 1 auto;
}

.scraped-performers {
	display: flex;
	flex-wrap: wrap;
	gap: 4px;
	margin-top: 4px;
}

.performer-chip {
	font-size: 0.68rem;
	padding: 2px 7px;
	border-radius: 10px;
	background: #2c3e50;
	color: #ecf0f1;
	text-decoration: none;
	cursor: default;
}

.performer-chip.matched {
	background: #8e44ad;
	cursor: pointer;
}

.performer-chip.matched:hover {
	background: #9b59b6;
}

.scraped-matched-tags {
	display: flex;
	flex-wrap: wrap;
	gap: 4px;
	margin-top: 4px;
}

.tag-chip {
	display: inline-flex;
	align-items: center;
	font-size: 0.68rem;
	padding: 2px 8px;
	border-radius: 20px;
	color: #fff;
	white-space: nowrap;
}

.video-added-time {
	display: block;
	font-size: 0.72rem;
	color: rgba(255, 255, 255, 0.35);
	margin-bottom: 0.25rem;
}

.video-category {
	display: flex;
	flex-wrap: wrap;
	gap: 4px;
}

.author-chip {
	display: inline-flex;
	align-items: center;
	font-size: 0.68rem;
	padding: 1px 7px;
	border-radius: 10px;
	background: rgba(120, 200, 140, 0.12);
	border: 1px solid rgba(120, 200, 140, 0.25);
	color: rgba(150, 220, 165, 0.85);
	cursor: pointer;
	transition: background 0.15s;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	max-width: 100%;
	margin-bottom: 3px;
}

.author-chip:hover {
	background: rgba(120, 200, 140, 0.25);
	color: #aeefc0;
}

.category-chip {
	display: inline-flex;
	align-items: center;
	font-size: 0.68rem;
	padding: 2px 8px;
	border-radius: 10px;
	background: rgba(13, 110, 253, 0.2);
	border: 1px solid rgba(13, 110, 253, 0.35);
	color: #74b3ff;
	cursor: pointer;
	transition: background 0.15s;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	max-width: 100%;
}

.category-chip:hover {
	background: rgba(13, 110, 253, 0.35);
}

.scraped-raw-tags {
	display: flex;
	flex-wrap: wrap;
	gap: 3px;
	margin-top: 4px;
}

.raw-tag-chip {
	font-size: 0.65rem;
	padding: 1px 6px;
	border-radius: 8px;
	background: rgba(255, 255, 255, 0.07);
	border: 1px solid rgba(255, 255, 255, 0.12);
	color: rgba(255, 255, 255, 0.55);
	cursor: pointer;
	transition: all 0.15s;
	white-space: nowrap;
}

.raw-tag-chip:hover {
	background: rgba(255, 255, 255, 0.14);
	color: rgba(255, 255, 255, 0.85);
	border-color: rgba(255, 255, 255, 0.3);
}

.raw-tag-more {
	cursor: default;
	font-style: italic;
	color: rgba(255, 255, 255, 0.35);
}

.raw-tag-more:hover {
	background: rgba(255, 255, 255, 0.07);
	color: rgba(255, 255, 255, 0.55);
	border-color: rgba(255, 255, 255, 0.12);
}
</style>
