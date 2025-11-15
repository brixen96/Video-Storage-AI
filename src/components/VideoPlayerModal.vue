<template>
	<div v-if="show" class="video-player-modal" @click.self="close">
		<div class="modal-dialog modal-xl">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title">{{ video?.title || 'Video Player' }}</h5>
					<button type="button" class="btn-close" @click="close"></button>
				</div>
				<div class="modal-body p-0">
					<video
						ref="videoPlayer"
						class="video-player"
						controls
						autoplay
						@loadedmetadata="onVideoLoaded"
						@play="onPlay"
						@pause="onPause"
						@ended="onEnded"
						@timeupdate="onTimeUpdate"
					>
						<source :src="videoUrl" :type="mimeType" />
						Your browser does not support the video tag.
					</video>

					<!-- Video Controls Overlay -->
					<div class="video-info-overlay">
						<div class="playback-info">
							<span class="time-display">{{ formatTime(currentTime) }} / {{ formatTime(duration) }}</span>
							<span class="quality-badge">{{ video?.resolution || 'N/A' }}</span>
							<span class="codec-badge">{{ video?.codec || 'N/A' }}</span>
						</div>
					</div>
				</div>
				<div class="modal-footer">
					<div class="video-stats">
						<small class="text-muted">
							<font-awesome-icon :icon="['fas', 'eye']" />
							{{ video?.play_count || 0 }} views
						</small>
						<small class="text-muted ms-3">
							<font-awesome-icon :icon="['fas', 'file']" />
							{{ formatFileSize(video?.file_size) }}
						</small>
						<small class="text-muted ms-3">
							<font-awesome-icon :icon="['fas', 'signal']" />
							{{ formatBitrate(video?.bitrate) }}
						</small>
					</div>
					<div class="modal-actions">
						<button class="btn btn-sm btn-outline-secondary" @click="openInExplorer">
							<font-awesome-icon :icon="['fas', 'folder-open']" />
							Open Location
						</button>
						<button class="btn btn-sm btn-outline-primary" @click="editMetadata">
							<font-awesome-icon :icon="['fas', 'edit']" />
							Edit
						</button>
						<button class="btn btn-sm btn-secondary" @click="close">Close</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
export default {
	name: 'VideoPlayerModal',
	props: {
		show: {
			type: Boolean,
			default: false,
		},
		video: {
			type: Object,
			default: null,
		},
	},
	emits: ['close', 'edit-metadata', 'open-explorer'],
	data() {
		return {
			currentTime: 0,
			duration: 0,
			isPlaying: false,
		}
	},
	computed: {
		videoUrl() {
			if (!this.video) return ''
			// Use file_path to stream video from backend
			return `http://localhost:8080/api/v1/videos/${this.video.id}/stream`
		},
		mimeType() {
			if (!this.video?.file_path) return 'video/mp4'
			const ext = this.video.file_path.split('.').pop().toLowerCase()
			const mimeTypes = {
				mp4: 'video/mp4',
				webm: 'video/webm',
				ogg: 'video/ogg',
				avi: 'video/x-msvideo',
				mov: 'video/quicktime',
				mkv: 'video/x-matroska',
				flv: 'video/x-flv',
				wmv: 'video/x-ms-wmv',
			}
			return mimeTypes[ext] || 'video/mp4'
		},
	},
	watch: {
		show(newVal) {
			if (newVal) {
				// Add keyboard listeners when modal opens
				document.addEventListener('keydown', this.handleKeyPress)
				// Prevent body scroll
				document.body.style.overflow = 'hidden'
			} else {
				// Remove keyboard listeners when modal closes
				document.removeEventListener('keydown', this.handleKeyPress)
				// Restore body scroll
				document.body.style.overflow = ''
				// Pause video when closing
				if (this.$refs.videoPlayer) {
					this.$refs.videoPlayer.pause()
				}
			}
		},
	},
	beforeUnmount() {
		document.removeEventListener('keydown', this.handleKeyPress)
		document.body.style.overflow = ''
	},
	methods: {
		close() {
			this.$emit('close')
		},
		onVideoLoaded() {
			if (this.$refs.videoPlayer) {
				this.duration = this.$refs.videoPlayer.duration
			}
		},
		onPlay() {
			this.isPlaying = true
		},
		onPause() {
			this.isPlaying = false
		},
		onEnded() {
			this.isPlaying = false
			// Could implement auto-play next video here
		},
		onTimeUpdate() {
			if (this.$refs.videoPlayer) {
				this.currentTime = this.$refs.videoPlayer.currentTime
			}
		},
		formatTime(seconds) {
			if (!seconds || isNaN(seconds)) return '0:00'
			const mins = Math.floor(seconds / 60)
			const secs = Math.floor(seconds % 60)
			return `${mins}:${secs.toString().padStart(2, '0')}`
		},
		formatFileSize(bytes) {
			if (!bytes) return 'N/A'
			if (bytes < 1024) return bytes + ' B'
			if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
			if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
			return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
		},
		formatBitrate(bitrate) {
			if (!bitrate) return 'N/A'
			return (bitrate / 1000000).toFixed(1) + ' Mbps'
		},
		handleKeyPress(event) {
			if (!this.show) return

			switch (event.key.toLowerCase()) {
				case 'escape':
					this.close()
					break
				case ' ':
					event.preventDefault()
					this.togglePlayPause()
					break
				case 'arrowright':
					event.preventDefault()
					this.skip(10)
					break
				case 'arrowleft':
					event.preventDefault()
					this.skip(-10)
					break
				case 'f':
					event.preventDefault()
					this.toggleFullscreen()
					break
				case 'm':
					event.preventDefault()
					this.toggleMute()
					break
			}
		},
		togglePlayPause() {
			if (this.$refs.videoPlayer) {
				if (this.isPlaying) {
					this.$refs.videoPlayer.pause()
				} else {
					this.$refs.videoPlayer.play()
				}
			}
		},
		skip(seconds) {
			if (this.$refs.videoPlayer) {
				this.$refs.videoPlayer.currentTime += seconds
			}
		},
		toggleFullscreen() {
			if (this.$refs.videoPlayer) {
				if (document.fullscreenElement) {
					document.exitFullscreen()
				} else {
					this.$refs.videoPlayer.requestFullscreen()
				}
			}
		},
		toggleMute() {
			if (this.$refs.videoPlayer) {
				this.$refs.videoPlayer.muted = !this.$refs.videoPlayer.muted
			}
		},
		openInExplorer() {
			this.$emit('open-explorer', this.video)
		},
		editMetadata() {
			this.$emit('edit-metadata', this.video)
		},
	},
}
</script>

<style scoped>
.video-player-modal {
	position: fixed;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	background: rgba(0, 0, 0, 0.9);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 9999;
}

.modal-dialog {
	max-width: 90vw;
	width: 1400px;
	margin: 0;
}

.modal-content {
	background: #1a1a2e;
	border: 1px solid #2d2d44;
	border-radius: 8px;
	overflow: hidden;
}

.modal-header {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	color: white;
	border-bottom: none;
	padding: 1rem 1.5rem;
}

.modal-title {
	font-size: 1.25rem;
	font-weight: 600;
	margin: 0;
}

.btn-close {
	filter: brightness(0) invert(1);
	opacity: 0.8;
}

.btn-close:hover {
	opacity: 1;
}

.modal-body {
	position: relative;
	background: #000;
}

.video-player {
	width: 100%;
	height: auto;
	max-height: 75vh;
	display: block;
	background: #000;
}

.video-info-overlay {
	position: absolute;
	top: 10px;
	right: 10px;
	display: flex;
	gap: 10px;
	pointer-events: none;
}

.playback-info {
	display: flex;
	gap: 10px;
	flex-direction: column;
	align-items: flex-end;
}

.time-display,
.quality-badge,
.codec-badge {
	background: rgba(0, 0, 0, 0.7);
	color: white;
	padding: 4px 12px;
	border-radius: 4px;
	font-size: 0.875rem;
	font-weight: 500;
	backdrop-filter: blur(10px);
}

.quality-badge {
	background: rgba(102, 126, 234, 0.8);
}

.codec-badge {
	background: rgba(118, 75, 162, 0.8);
}

.modal-footer {
	background: #16213e;
	border-top: 1px solid #2d2d44;
	padding: 1rem 1.5rem;
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.video-stats {
	display: flex;
	align-items: center;
	gap: 1rem;
}

.video-stats small {
	color: #a0a0c0;
	font-size: 0.875rem;
}

.modal-actions {
	display: flex;
	gap: 0.5rem;
}

.btn-outline-secondary,
.btn-outline-primary {
	border-color: #2d2d44;
	color: #a0a0c0;
}

.btn-outline-secondary:hover,
.btn-outline-primary:hover {
	background: rgba(102, 126, 234, 0.2);
	border-color: #667eea;
	color: #667eea;
}
</style>
