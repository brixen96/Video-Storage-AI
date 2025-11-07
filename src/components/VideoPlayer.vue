<template>
	<div class="video-player-modal" v-if="visible" @click.self="close">
		<div class="video-player-container">
			<!-- Header -->
			<div class="player-header">
				<div class="player-title">
					<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
					{{ video.name }}
				</div>
				<button class="btn-close-player" @click="close">
					<font-awesome-icon :icon="['fas', 'times']" />
				</button>
			</div>

			<!-- Video Player -->
			<div class="player-body">
				<video
					ref="videoElement"
					class="video-element"
					controls
					autoplay
					@loadedmetadata="onLoadedMetadata"
					@timeupdate="onTimeUpdate"
					@ended="onEnded"
					@error="onVideoError"
				>
					<source :src="videoUrl" :type="videoMimeType" />
					Your browser does not support the video tag.
				</video>
				<div v-if="videoError" class="video-error-overlay">
					<div class="error-content">
						<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="error-icon" />
						<h4>Unable to Play Video</h4>
						<p v-if="video.extension === '.mkv'">MKV format is not natively supported by browsers. Consider converting to MP4 for better compatibility.</p>
						<p v-else>This video format may not be supported by your browser.</p>
						<p class="file-info">File: {{ video.name }}</p>
					</div>
				</div>
			</div>

			<!-- Video Info -->
			<div class="player-footer">
				<div class="video-info">
					<div class="info-item">
						<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
						<span v-if="duration">{{ formatDuration(duration) }}</span>
						<span v-else>Loading...</span>
					</div>
					<div class="info-item" v-if="video.size">
						<font-awesome-icon :icon="['fas', 'file']" class="me-1" />
						{{ formatFileSize(video.size) }}
					</div>
					<div class="info-item" v-if="video.extension">
						<font-awesome-icon :icon="['fas', 'film']" class="me-1" />
						{{ video.extension.toUpperCase() }}
					</div>
				</div>
				<div class="player-actions">
					<button class="btn btn-sm btn-outline-primary me-2" @click="toggleFullscreen">
						<font-awesome-icon :icon="['fas', 'expand']" class="me-1" />
						Fullscreen
					</button>
					<button class="btn btn-sm btn-outline-secondary" @click="close">
						<font-awesome-icon :icon="['fas', 'times']" class="me-1" />
						Close
					</button>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
export default {
	name: 'VideoPlayer',
	props: {
		visible: {
			type: Boolean,
			default: false,
		},
		video: {
			type: Object,
			required: true,
		},
		libraryId: {
			type: Number,
			required: false,
			default: null,
		},
	},
	data() {
		return {
			duration: 0,
			currentTime: 0,
			videoError: false,
		}
	},
	computed: {
		videoUrl() {
			// Construct URL for streaming the video file
			const path = encodeURIComponent(this.video.path)
			return `http://localhost:8080/api/v1/libraries/${this.libraryId}/stream?path=${path}`
		},
		videoMimeType() {
			const ext = this.video.extension?.toLowerCase() || ''
			const mimeTypes = {
				'.mp4': 'video/mp4',
				'.webm': 'video/webm',
				'.ogg': 'video/ogg',
				'.mkv': 'video/mp4',
				'.avi': 'video/x-msvideo',
				'.mov': 'video/quicktime',
				'.wmv': 'video/x-ms-wmv',
				'.flv': 'video/x-flv',
				'.m4v': 'video/mp4',
				'.mpg': 'video/mpeg',
				'.mpeg': 'video/mpeg',
				'.3gp': 'video/3gpp',
			}
			return mimeTypes[ext] || 'video/mp4'
		},
	},
	methods: {
		close() {
			this.$emit('close')
			if (this.$refs.videoElement) {
				this.$refs.videoElement.pause()
			}
		},
		onLoadedMetadata() {
			if (this.$refs.videoElement) {
				this.duration = this.$refs.videoElement.duration
			}
		},
		onTimeUpdate() {
			if (this.$refs.videoElement) {
				this.currentTime = this.$refs.videoElement.currentTime
			}
		},
		onEnded() {
			this.$emit('ended')
		},
		onVideoError(event) {
			console.error('Video playback error:', event)
			this.videoError = true
		},
		toggleFullscreen() {
			if (this.$refs.videoElement) {
				if (this.$refs.videoElement.requestFullscreen) {
					this.$refs.videoElement.requestFullscreen()
				} else if (this.$refs.videoElement.webkitRequestFullscreen) {
					this.$refs.videoElement.webkitRequestFullscreen()
				} else if (this.$refs.videoElement.mozRequestFullScreen) {
					this.$refs.videoElement.mozRequestFullScreen()
				}
			}
		},
		formatDuration(seconds) {
			const hours = Math.floor(seconds / 3600)
			const minutes = Math.floor((seconds % 3600) / 60)
			const secs = Math.floor(seconds % 60)

			if (hours > 0) {
				return `${hours}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
			}
			return `${minutes}:${String(secs).padStart(2, '0')}`
		},
		formatFileSize(bytes) {
			if (bytes === 0) return '0 Bytes'
			const k = 1024
			const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
			const i = Math.floor(Math.log(bytes) / Math.log(k))
			return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
		},
	},
	watch: {
		visible(newVal) {
			if (newVal) {
				// Reset state when opening
				this.duration = 0
				this.currentTime = 0
				this.videoError = false
			}
		},
	},
}
</script>

<style scoped>
@import '@/styles/components/video_player.css';
</style>
