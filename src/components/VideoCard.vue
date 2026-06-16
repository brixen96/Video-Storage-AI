<template>
	<div class="video-card" :class="{ selected: isSelected }" @click="handleClick" @contextmenu.prevent="showContextMenu">
		<!-- Selection Checkbox -->
		<div class="selection-checkbox" @click.stop="toggleSelection">
			<input type="checkbox" :checked="isSelected" />
		</div>

		<!-- Thumbnail / Preview -->
		<div class="video-thumbnail" @mouseenter="startPreview" @mouseleave="stopPreview">
			<!-- Static Thumbnail -->
			<img
				v-if="!isPreviewPlaying && video.thumbnail_path && !thumbnailError"
				:src="getThumbnailURL(video)"
				:alt="video.title"
				loading="lazy"
				class="thumbnail-image"
				@error="handleThumbnailError"
			/>
			<div v-else-if="!isPreviewPlaying" class="thumbnail-placeholder">
				<font-awesome-icon :icon="['fas', 'video']" size="3x" />
				<span v-if="thumbnailError" class="error-text">Thumbnail unavailable</span>
			</div>

			<!-- Storyboard Preview (cycles through frames on hover) -->
			<img
				v-if="hasPreview && isPreviewPlaying && currentPreviewFrame"
				:src="currentPreviewFrame"
				class="preview-video active"
				:alt="`${video.title} - Preview`"
				loading="lazy"
			/>
			<!-- Video Preview (scraped videos with external video_preview_url) -->
			<video
				v-if="videoPreviewUrl && isPreviewPlaying && !previewVideoError"
				:src="videoPreviewUrl"
				class="preview-video active"
				autoplay
				muted
				loop
				playsinline
				@error="previewVideoError = true"
			/>

			<!-- Missing file overlay -->
			<div v-if="video.is_missing" class="missing-overlay" title="File not found on disk">
				<font-awesome-icon :icon="['fas', 'video-slash']" />
				<span>File Missing</span>
			</div>

			<!-- Status Badges (top right) -->
			<div class="status-badges">
				<div v-if="video.watched" class="badge-watched" title="Watched">
					<font-awesome-icon :icon="['fas', 'eye']" />
				</div>
				<div v-if="video.is_favorite" class="badge-favorite" title="Favorite">
					<font-awesome-icon :icon="['fas', 'heart']" />
				</div>
				<div v-if="video.is_pinned" class="badge-pinned" title="Pinned">
					<font-awesome-icon :icon="['fas', 'thumbtack']" />
				</div>
				<div v-if="video.rating" class="badge-rating" :title="`Rating: ${video.rating}/5`">
					<font-awesome-icon :icon="['fas', 'star']" />
					{{ video.rating }}
				</div>
				<div v-if="video.converted_from" class="badge-converted" title="Converted from original">
					<font-awesome-icon :icon="['fas', 'sync']" />
					MP4
				</div>
				<div v-if="video.converted_to" class="badge-has-conversion" title="Has MP4 conversion available">
					<font-awesome-icon :icon="['fas', 'check']" />
					Converted
				</div>
				<div v-if="video.linked_video_id" class="badge-linked" title="Has a linked version in another library">
					<font-awesome-icon :icon="['fas', 'link']" />
				</div>
			</div>

			<!-- Duration badge -->
			<div v-if="video.duration" class="badge-duration">{{ formatDuration(video.duration) }}</div>

			<!-- Quick favorite button (shown on hover) -->
			<button
				class="quick-favorite"
				:class="{ active: video.is_favorite }"
				:title="video.is_favorite ? 'Remove from favorites' : 'Add to favorites'"
				@click.stop="$emit('toggle-favorite', video)"
			>
				<font-awesome-icon :icon="[video.is_favorite ? 'fas' : 'far', 'heart']" />
			</button>
		</div>

		<!-- Video Info -->
		<div class="video-info">
			<h3 class="video-title" :title="video.title">{{ video.title }}</h3>

			<!-- Metadata Badges -->
			<div class="video-badges">
				<span v-if="video.resolution" class="badge bg-primary">{{ video.resolution }}</span>
				<span v-if="video.fps" class="badge bg-success">{{ formatFPS(video.fps) }} FPS</span>
				<span v-if="video.codec" class="badge bg-warning text-dark">{{ video.codec }}</span>
				<span v-if="video.bitrate" class="badge bg-info">{{ formatBitrate(video.bitrate) }}</span>
				<span v-if="video.file_size" class="badge bg-secondary">{{ formatFileSize(video.file_size) }}</span>
				<span v-if="video.play_count > 0" class="badge bg-dark">
					<font-awesome-icon :icon="['fas', 'eye']" />
					{{ video.play_count }}
				</span>
				<!-- Source badge for scraped videos -->
				<span v-if="video.is_scraped && video.source_tab_name" class="badge badge-source" :title="`Source: ${video.source_tab_name}`">
					<font-awesome-icon :icon="['fas', 'globe']" />
					{{ video.source_tab_name }}
				</span>
			</div>

			<!-- Performers -->
			<div v-if="video.performers && video.performers.length > 0" class="video-performers">
				<div
					v-for="performer in video.performers.slice(0, 4)"
					:key="performer.id"
					class="performer-avatar"
					:title="performer.name"
					@click.stop="$emit('open-performer', performer)"
				>
					<img
						v-if="performer.thumbnail_path"
						:src="getPerformerThumbnailURL(performer)"
						:alt="performer.name"
						loading="lazy"
					/>
					<div v-else class="performer-initials">{{ getInitials(performer.name) }}</div>
				</div>
				<span v-if="video.performers.length > 4" class="performer-more">+{{ video.performers.length - 4 }}</span>
			</div>

			<!-- Studio -->
			<div v-if="video.studios && video.studios.length > 0" class="video-studio" @click.stop="$emit('open-studio', video.studios[0])">
				<font-awesome-icon :icon="['fas', 'building']" />
				{{ video.studios[0].name }}
			</div>

			<!-- Tags (only show if tags exist and have content) -->
			<div v-if="video.tags && video.tags.length > 0 && hasValidTags" class="video-tags">
				<template v-for="(tag, index) in video.tags.slice(0, 2)" :key="index">
					<!-- Regular video tags (objects with id, name, color) -->
					<span v-if="typeof tag === 'object' && tag.name" class="tag-chip" :style="{ backgroundColor: tag.color || '#6c757d' }">
						<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" />
						{{ tag.name }}
					</span>
					<!-- Scraped video tags (simple strings) -->
					<span v-else-if="typeof tag === 'string' && tag.trim()" class="tag-chip" style="background-color: #6c757d">
						{{ tag }}
					</span>
				</template>
				<span v-if="video.tags.length > 2" class="tag-more">+{{ video.tags.length - 2 }}</span>
			</div>
		</div>
	</div>
</template>

<script>
import { getAssetURL, API_ORIGIN } from '@/services/api'

export default {
	name: 'VideoCard',
	props: {
		video: {
			type: Object,
			required: true,
		},
		isSelected: {
			type: Boolean,
			default: false,
		},
	},
	emits: ['click', 'toggle-select', 'context-menu', 'play', 'add-tag', 'edit-metadata', 'open-performer', 'open-studio', 'toggle-favorite'],
	data() {
		return {
			isPreviewPlaying: false,
			previewLoaded: false,
			hoverTimeout: null,
			previewFrameIndex: 0,
			previewInterval: null,
			previewFrames: [], // Array of frame URLs
			videoPreviewUrl: null, // External video URL (for scraped video_preview_url)
			previewVideoError: false,
			thumbnailError: false,
		}
	},
	computed: {
		hasPreview() {
			if (this.video.preview_path && this.video.preview_path !== '') return true
			if (this.video.thumbnail_frames && this.video.thumbnail_frames.length > 0) return true
			return false
		},
		currentPreviewFrame() {
			if (this.previewFrames.length === 0) return null
			return this.previewFrames[this.previewFrameIndex]
		},
		hasValidTags() {
			// Check if tags array has any valid tags (non-empty strings or objects with name)
			if (!this.video.tags || this.video.tags.length === 0) return false
			return this.video.tags.some(tag => {
				if (typeof tag === 'object' && tag.name) return true
				if (typeof tag === 'string' && tag.trim()) return true
				return false
			})
		},
	},
	methods: {
		getAssetURL,
		getPerformerThumbnailURL(performer) {
			if (!performer.thumbnail_path) return null
			if (performer.thumbnail_path.startsWith('http://') || performer.thumbnail_path.startsWith('https://')) {
				return performer.thumbnail_path
			}
			return getAssetURL(performer.thumbnail_path)
		},
		getThumbnailURL(video) {
			// For scraped videos, thumbnail_path is already a full URL
			if (video.is_scraped && video.thumbnail_path) {
				// Check if it's already an absolute URL
				if (video.thumbnail_path.startsWith('http://') || video.thumbnail_path.startsWith('https://')) {
					return video.thumbnail_path
				}
			}
			if (video.thumbnail_path) {
				return getAssetURL(video.thumbnail_path)
			}
			return `${API_ORIGIN}/api/v1/videos/${video.id}/thumbnail`
		},
		loadPreviewFrames() {
			// Scraped videos: thumbnail_frames array takes priority
			if (this.video.thumbnail_frames && this.video.thumbnail_frames.length > 0) {
				this.previewFrames = this.video.thumbnail_frames
				return
			}

			if (!this.video.preview_path) {
				this.previewFrames = []
				return
			}

			// Scraped videos: preview_path may be an external CDN video URL (e.g. video_preview_url)
			if (this.video.preview_path.startsWith('http://') || this.video.preview_path.startsWith('https://')) {
				this.videoPreviewUrl = this.video.preview_path
				this.previewFrames = []
				return
			}

			// Local videos: generate frame URLs from relative preview_path directory
			const frames = []
			for (let i = 1; i <= 10; i++) {
				const framePath = `previews/${this.video.preview_path}/frame_${i.toString().padStart(3, '0')}.jpg`
				frames.push(getAssetURL(framePath))
			}
			this.previewFrames = frames
		},
		startPreview() {
			// Delay preview start by 300ms to avoid loading on quick hovers
			this.hoverTimeout = setTimeout(() => {
				// Lazily build frame URLs on first hover (deferred from mounted)
				if (this.previewFrames.length === 0 && !this.videoPreviewUrl && this.hasPreview) {
					this.loadPreviewFrames()
				}
				// Video preview (scraped videos with external video_preview_url)
				if (this.videoPreviewUrl && !this.previewVideoError) {
					this.isPreviewPlaying = true
					return
				}
				// Frame cycling preview
				if (this.hasPreview && this.previewFrames.length > 0) {
					this.isPreviewPlaying = true
					this.previewFrameIndex = 0
					this.previewInterval = setInterval(() => {
						if (this.previewFrames.length > 0) {
							this.previewFrameIndex = (this.previewFrameIndex + 1) % this.previewFrames.length
						}
					}, 500)
				}
			}, 300)
		},
		stopPreview() {
			if (this.hoverTimeout) {
				clearTimeout(this.hoverTimeout)
				this.hoverTimeout = null
			}
			if (this.previewInterval) {
				clearInterval(this.previewInterval)
				this.previewInterval = null
			}
			this.isPreviewPlaying = false
			this.previewFrameIndex = 0
		},
		formatDuration(seconds) {
			const mins = Math.floor(seconds / 60)
			const secs = Math.floor(seconds % 60)
			return `${mins}:${secs.toString().padStart(2, '0')}`
		},
		formatFileSize(bytes) {
			if (bytes < 1024) return bytes + ' B'
			if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
			if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
			return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
		},
		formatBitrate(bitrate) {
			if (!bitrate) return ''
			const kbps = bitrate / 1000
			if (kbps < 1000) return kbps.toFixed(0) + ' Kbps'
			return (kbps / 1000).toFixed(1) + ' Mbps'
		},
		formatFPS(fps) {
			if (!fps) return ''
			// Round to 2 decimal places and remove trailing zeros
			return parseFloat(fps.toFixed(2))
		},
		getInitials(name) {
			if (!name) return '?'
			return name
				.split(' ')
				.map((n) => n[0])
				.join('')
				.toUpperCase()
				.slice(0, 2)
		},
		async handleClick() {
			// For scraped videos, emit play so the parent can decide (e.g. show modal)
			if (this.video.is_scraped) {
				this.$emit('play', this.video)
				return
			}
			// For regular videos, open video player page in new tab
			const route = this.$router.resolve(`/watch/${this.video.id}`)
			window.open(route.href, '_blank')
		},
		async openInFirefox(url) {
			try {
				const response = await fetch(`${API_ORIGIN}/api/v1/system/open-url`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ url })
				})
				if (!response.ok) {
					window.open(url, '_blank')
				}
			} catch (error) {
				window.open(url, '_blank')
			}
		},
		toggleSelection() {
			this.$emit('toggle-select', this.video)
		},
		showContextMenu(event) {
			this.$emit('context-menu', { video: this.video, x: event.clientX, y: event.clientY })
		},
		handleThumbnailError() {
			this.thumbnailError = true
			console.warn(`Thumbnail failed to load for video: ${this.video.title}`)
		},
	},
	mounted() {
		// Preview frames are loaded lazily on first hover (see startPreview)
	},
	watch: {
		'video.id'() {
			this.thumbnailError = false
			this.previewFrames = []
			this.videoPreviewUrl = null
			this.previewVideoError = false
		},
		'video.preview_path'() {
			// Reload frames if preview_path changes
			this.loadPreviewFrames()
		},
	},
	beforeUnmount() {
		// Clean up timeout and interval on component destroy
		if (this.hoverTimeout) {
			clearTimeout(this.hoverTimeout)
		}
		if (this.previewInterval) {
			clearInterval(this.previewInterval)
		}
	},
}
</script>

<style scoped>
@import '@/styles/components/video_card.css';
</style>
