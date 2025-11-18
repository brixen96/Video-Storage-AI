<template>
	<div class="video-card" :class="{ selected: isSelected }" @click="handleClick" @contextmenu.prevent="showContextMenu">
		<!-- Selection Checkbox -->
		<div class="selection-checkbox" @click.stop="toggleSelection">
			<input type="checkbox" :checked="isSelected" />
		</div>

		<!-- Thumbnail / Preview -->
		<div class="video-thumbnail" @mouseenter="startPreview" @mouseleave="stopPreview">
			<!-- Static Thumbnail -->
			<img v-if="!isPreviewPlaying && video.thumbnail_path" :src="getThumbnailURL(video)" :alt="video.title" loading="lazy" class="thumbnail-image" />
			<div v-else-if="!isPreviewPlaying" class="thumbnail-placeholder">
				<font-awesome-icon :icon="['fas', 'video']" size="3x" />
			</div>

			<!-- Video Preview (plays on hover) -->
			<video
				v-if="hasPreview"
				ref="previewVideo"
				:src="getPreviewURL(video)"
				class="preview-video"
				:class="{ active: isPreviewPlaying }"
				loop
				muted
				@loadeddata="onPreviewLoaded"
			></video>

			<!-- Hover Overlay -->
			<div class="hover-overlay">
				<button class="btn-play" @click.stop="playVideo">
					<font-awesome-icon :icon="['fas', 'play']" />
				</button>
				<div class="quick-actions">
					<button class="btn-quick-action" @click.stop="$emit('add-tag', video)" title="Add Tag">
						<font-awesome-icon :icon="['fas', 'tag']" />
					</button>
					<button class="btn-quick-action" @click.stop="$emit('edit-metadata', video)" title="Edit Metadata">
						<font-awesome-icon :icon="['fas', 'edit']" />
					</button>
					<button class="btn-quick-action" @click.stop="openInExplorer" title="Open in Explorer">
						<font-awesome-icon :icon="['fas', 'folder-open']" />
					</button>
				</div>
			</div>

			<!-- Status Badges -->
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
				<span v-if="video.file_size" class="badge bg-secondary">{{ formatFileSize(video.file_size) }}</span>
				<span v-if="video.play_count > 0" class="badge bg-info">
					<font-awesome-icon :icon="['fas', 'eye']" />
					{{ video.play_count }}
				</span>
			</div>

			<!-- Performers -->

			<!-- Studio -->
			<div v-if="video.studios && video.studios.length > 0" class="video-studio" @click.stop="$emit('open-studio', video.studios[0])">
				<font-awesome-icon :icon="['fas', 'building']" />
				{{ video.studios[0].name }}
			</div>

			<!-- Tags -->
			<div v-if="video.tags && video.tags.length > 0" class="video-tags">
				<span v-for="tag in video.tags.slice(0, 2)" :key="tag.id" class="tag-chip" :style="{ backgroundColor: tag.color || '#6c757d' }">
					<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" />
					{{ tag.name }}
				</span>
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
		}
	},
	computed: {
		hasPreview() {
			return this.video.preview_path || (this.video.previews && this.video.previews.length > 0)
		},
	},
	methods: {
		getAssetURL,
		getThumbnailURL(video) {
			if (video.thumbnail_path) {
				return getAssetURL(video.thumbnail_path)
			}
			return `http://localhost:8080/api/v1/videos/${video.id}/thumbnail`
		},
		getPreviewURL(video) {
			if (video.preview_path) {
				return getAssetURL(video.preview_path)
			}
			if (video.previews && video.previews.length > 0) {
				return getAssetURL(video.previews[0])
			}
			return null
		},
		startPreview() {
			// Delay preview start by 300ms to avoid loading on quick hovers
			this.hoverTimeout = setTimeout(() => {
				if (this.hasPreview && this.$refs.previewVideo) {
					this.isPreviewPlaying = true
					this.$refs.previewVideo.play().catch(() => {
						// Ignore play errors (e.g., if user navigates away quickly)
					})
				}
			}, 300)
		},
		stopPreview() {
			// Clear the timeout if user moves away before preview starts
			if (this.hoverTimeout) {
				clearTimeout(this.hoverTimeout)
				this.hoverTimeout = null
			}

			// Stop preview if playing
			if (this.isPreviewPlaying && this.$refs.previewVideo) {
				this.$refs.previewVideo.pause()
				this.$refs.previewVideo.currentTime = 0
				this.isPreviewPlaying = false
			}
		},
		onPreviewLoaded() {
			this.previewLoaded = true
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
		getInitials(name) {
			return name
				.split(' ')
				.map((n) => n[0])
				.join('')
				.toUpperCase()
				.slice(0, 2)
		},
		handleClick() {
			this.$emit('click', this.video)
		},
		toggleSelection() {
			this.$emit('toggle-select', this.video)
		},
		showContextMenu(event) {
			this.$emit('context-menu', { video: this.video, x: event.clientX, y: event.clientY })
		},
		playVideo() {
			this.$emit('play', this.video)
		},
		openInExplorer() {
			// This would need native integration or electron
			console.log('Open in explorer:', this.video.file_path)
		},
	},
	beforeUnmount() {
		// Clean up timeout on component destroy
		if (this.hoverTimeout) {
			clearTimeout(this.hoverTimeout)
		}
	},
}
</script>

<style scoped>
@import '@/styles/components/video_card.css';
</style>
