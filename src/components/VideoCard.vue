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
				v-if="!isPreviewPlaying && video.thumbnail_path"
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

			<!-- Status Badges (top right) -->
			<div class="status-badges">
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
			</div>

			<!-- Duration badge -->
			<div v-if="video.duration" class="badge-duration">{{ formatDuration(video.duration) }}</div>
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
import { getAssetURL } from '@/services/api'

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
	emits: ['click', 'toggle-select', 'context-menu', 'play', 'add-tag', 'edit-metadata', 'open-performer', 'open-studio'],
	data() {
		return {
			isPreviewPlaying: false,
			previewLoaded: false,
			hoverTimeout: null,
			previewFrameIndex: 0,
			previewInterval: null,
			previewFrames: [], // Array of frame URLs
			thumbnailError: false,
		}
	},
	computed: {
		hasPreview() {
			// Check for regular video preview path OR scraped video thumbnail frames
			if (this.video.preview_path && this.video.preview_path !== '') {
				return true
			}
			// For scraped videos, check thumbnail_frames array
			if (this.video.is_scraped && this.video.thumbnail_frames && this.video.thumbnail_frames.length > 0) {
				return true
			}
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
			return `http://localhost:8080/api/v1/videos/${video.id}/thumbnail`
		},
		loadPreviewFrames() {
			// For scraped videos, check if we have thumbnail_frames array
			if (this.video.is_scraped && this.video.thumbnail_frames && this.video.thumbnail_frames.length > 0) {
				this.previewFrames = this.video.thumbnail_frames
				return
			}

			if (!this.video.preview_path) {
				this.previewFrames = []
				return
			}

			// Generate URLs for all 10 preview frames
			// preview_path is like "1/someFolder/videoName" (directory containing frames)
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
				if (this.hasPreview && this.previewFrames.length > 0) {
					this.isPreviewPlaying = true
					this.previewFrameIndex = 0

					// Cycle through frames every 500ms for better viewing
					this.previewInterval = setInterval(() => {
						if (this.previewFrames.length > 0) {
							this.previewFrameIndex = (this.previewFrameIndex + 1) % this.previewFrames.length
						}
					}, 500)
				}
			}, 300)
		},
		stopPreview() {
			// Clear the timeout if user moves away before preview starts
			if (this.hoverTimeout) {
				clearTimeout(this.hoverTimeout)
				this.hoverTimeout = null
			}

			// Stop preview cycling
			if (this.previewInterval) {
				clearInterval(this.previewInterval)
				this.previewInterval = null
			}

			// Reset preview state
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
			return name
				.split(' ')
				.map((n) => n[0])
				.join('')
				.toUpperCase()
				.slice(0, 2)
		},
		async handleClick() {
			// For scraped videos, open the source URL in Firefox
			if (this.video.is_scraped && this.video.source_url) {
				await this.openInFirefox(this.video.source_url)
				return
			}
			// For regular videos, open video player page in new tab
			const route = this.$router.resolve(`/watch/${this.video.id}`)
			window.open(route.href, '_blank')
		},
		async openInFirefox(url) {
			try {
				// Call backend API to open URL in Firefox
				const response = await fetch('http://localhost:8080/api/v1/system/open-incognito', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ url })
				})
				if (!response.ok) {
					// Fallback to regular new tab if API fails
					console.warn('Firefox API failed, opening in regular tab')
					window.open(url, '_blank')
				}
			} catch (error) {
				// Fallback to regular new tab if API is unavailable
				console.warn('Firefox API unavailable, opening in regular tab:', error)
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
		// Load preview frames on mount
		this.loadPreviewFrames()
	},
	watch: {
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
