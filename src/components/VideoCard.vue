<template>
	<div class="video-card" :class="{ selected: isSelected }" @click="handleClick" @contextmenu.prevent="showContextMenu">
		<!-- Selection Checkbox -->
		<div class="selection-checkbox" @click.stop="toggleSelection">
			<input type="checkbox" :checked="isSelected" />
		</div>

		<!-- Thumbnail / Preview -->
		<div class="video-thumbnail">
			<img v-if="video.thumbnail_path" :src="getThumbnailURL(video)" :alt="video.title" loading="lazy" />
			<div v-else class="thumbnail-placeholder">
				<font-awesome-icon :icon="['fas', 'video']" size="3x" />
			</div>

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
			<div v-if="video.performers && video.performers.length > 0" class="video-performers">
				<div v-for="performer in video.performers.slice(0, 3)" :key="performer.id" class="performer-avatar" :title="performer.name" @click.stop="$emit('open-performer', performer)">
					<img v-if="performer.image_path" :src="getAssetURL(performer.image_path)" :alt="performer.name" />
					<div v-else class="performer-initials">{{ getInitials(performer.name) }}</div>
				</div>
				<div v-if="video.performers.length > 3" class="performer-more">+{{ video.performers.length - 3 }}</div>
			</div>

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
	methods: {
		getAssetURL,
		getThumbnailURL(video) {
			if (video.thumbnail_path) {
				return getAssetURL(video.thumbnail_path)
			}
			return `http://localhost:8080/api/v1/videos/${video.id}/thumbnail`
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
}
</script>

<style scoped>
.video-card {
	background: var(--bs-body-bg);
	border: 1px solid var(--bs-border-color);
	border-radius: 0.5rem;
	overflow: hidden;
	cursor: pointer;
	transition: all 0.3s;
	position: relative;
	height: 100%;
	display: flex;
	flex-direction: column;
}

.video-card:hover {
	transform: translateY(-4px);
	box-shadow: 0 8px 16px rgba(0, 0, 0, 0.3);
	border-color: var(--bs-primary);
}

.video-card.selected {
	border-color: var(--bs-success);
	box-shadow: 0 0 0 2px var(--bs-success);
}

/* Selection Checkbox */
.selection-checkbox {
	position: absolute;
	top: 0.5rem;
	left: 0.5rem;
	z-index: 10;
	background: rgba(0, 0, 0, 0.7);
	border-radius: 0.25rem;
	padding: 0.25rem;
}

.selection-checkbox input[type='checkbox'] {
	cursor: pointer;
	width: 1.25rem;
	height: 1.25rem;
}

/* Thumbnail */
.video-thumbnail {
	position: relative;
	aspect-ratio: 16 / 9;
	background: var(--bs-secondary-bg);
	overflow: hidden;
}

.video-thumbnail img {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.thumbnail-placeholder {
	width: 100%;
	height: 100%;
	display: flex;
	align-items: center;
	justify-content: center;
	color: var(--bs-secondary);
}

/* Hover Overlay */
.hover-overlay {
	position: absolute;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.7);
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	opacity: 0;
	transition: opacity 0.3s;
}

.video-card:hover .hover-overlay {
	opacity: 1;
}

.btn-play {
	background: var(--bs-primary);
	border: none;
	border-radius: 50%;
	width: 4rem;
	height: 4rem;
	color: white;
	font-size: 1.5rem;
	cursor: pointer;
	transition: transform 0.2s;
	display: flex;
	align-items: center;
	justify-content: center;
	margin-bottom: 1rem;
}

.btn-play:hover {
	transform: scale(1.1);
}

.quick-actions {
	display: flex;
	gap: 0.5rem;
}

.btn-quick-action {
	background: rgba(255, 255, 255, 0.2);
	border: 1px solid rgba(255, 255, 255, 0.3);
	border-radius: 0.25rem;
	padding: 0.5rem;
	color: white;
	cursor: pointer;
	transition: all 0.2s;
}

.btn-quick-action:hover {
	background: rgba(255, 255, 255, 0.3);
}

/* Duration Badge */
.badge-duration {
	position: absolute;
	bottom: 0.5rem;
	right: 0.5rem;
	background: rgba(0, 0, 0, 0.8);
	color: white;
	padding: 0.25rem 0.5rem;
	border-radius: 0.25rem;
	font-size: 0.75rem;
	font-weight: 600;
}

/* Video Info */
.video-info {
	padding: 1rem;
	flex: 1;
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.video-title {
	font-size: 0.95rem;
	font-weight: 600;
	margin: 0;
	overflow: hidden;
	text-overflow: ellipsis;
	display: -webkit-box;
	-webkit-line-clamp: 2;
	-webkit-box-orient: vertical;
	line-height: 1.3;
}

.video-badges {
	display: flex;
	gap: 0.25rem;
	flex-wrap: wrap;
}

.video-badges .badge {
	font-size: 0.7rem;
}

/* Performers */
.video-performers {
	display: flex;
	gap: 0.25rem;
	align-items: center;
}

.performer-avatar {
	width: 2rem;
	height: 2rem;
	border-radius: 50%;
	overflow: hidden;
	background: var(--bs-secondary-bg);
	cursor: pointer;
	transition: transform 0.2s;
}

.performer-avatar:hover {
	transform: scale(1.1);
}

.performer-avatar img {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.performer-initials {
	width: 100%;
	height: 100%;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 0.7rem;
	font-weight: 600;
	color: var(--bs-secondary);
}

.performer-more {
	font-size: 0.75rem;
	color: var(--bs-secondary);
	font-weight: 600;
}

/* Studio */
.video-studio {
	font-size: 0.75rem;
	color: var(--bs-secondary);
	cursor: pointer;
	transition: color 0.2s;
}

.video-studio:hover {
	color: var(--bs-primary);
}

/* Tags */
.video-tags {
	display: flex;
	gap: 0.25rem;
	flex-wrap: wrap;
}

.tag-chip {
	display: inline-flex;
	align-items: center;
	gap: 0.25rem;
	padding: 0.25rem 0.5rem;
	border-radius: 0.25rem;
	font-size: 0.7rem;
	color: white;
	font-weight: 500;
}

.tag-more {
	font-size: 0.7rem;
	color: var(--bs-secondary);
	padding: 0.25rem 0.5rem;
}
</style>
