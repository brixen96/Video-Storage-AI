<template>
	<div class="video-player-page">
		<div class="container-fluid">
			<div class="row g-4">
				<!-- Main Content Column -->
				<div class="col-lg-9">
					<!-- Video Player -->
					<div class="player-container">
						<video
							v-if="video"
							ref="videoElement"
							class="main-video-player"
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
								<font-awesome-icon :icon="['fas', 'exclamation-triangle']" size="3x" class="error-icon" />
								<h4>Unable to Play Video</h4>
								<p v-if="video?.extension === '.mkv'">MKV format is not natively supported by browsers. Consider converting to MP4.</p>
								<p v-else>This video format may not be supported by your browser.</p>
							</div>
						</div>
						<div v-if="loading" class="loading-overlay">
							<font-awesome-icon :icon="['fas', 'spinner']" spin size="3x" />
							<p>Loading video...</p>
						</div>
					</div>

					<!-- Video Info -->
					<div v-if="video" class="video-info-section">
						<h1 class="video-title">{{ video.title }}</h1>

						<!-- Metadata Badges -->
						<div class="video-metadata">
							<span v-if="video.resolution" class="badge bg-primary">
								<font-awesome-icon :icon="['fas', 'video']" class="me-1" />
								{{ video.resolution }}
							</span>
							<span v-if="video.fps" class="badge bg-success">
								<font-awesome-icon :icon="['fas', 'film']" class="me-1" />
								{{ formatFPS(video.fps) }} FPS
							</span>
							<span v-if="video.codec" class="badge bg-warning text-dark">
								<font-awesome-icon :icon="['fas', 'file-video']" class="me-1" />
								{{ video.codec }}
							</span>
							<span v-if="video.bitrate" class="badge bg-info">
								<font-awesome-icon :icon="['fas', 'signal']" class="me-1" />
								{{ formatBitrate(video.bitrate) }}
							</span>
							<span v-if="video.file_size" class="badge bg-secondary">
								<font-awesome-icon :icon="['fas', 'hdd']" class="me-1" />
								{{ formatFileSize(video.file_size) }}
							</span>
							<span v-if="video.duration" class="badge bg-dark">
								<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
								{{ formatDuration(video.duration) }}
							</span>
						</div>

						<!-- Action Buttons -->
						<div class="action-buttons mt-3">
							<button v-if="video.is_favorite" class="btn btn-danger" @click="toggleFavorite">
								<font-awesome-icon :icon="['fas', 'heart']" />
								Favorited
							</button>
							<button v-else class="btn btn-outline-danger" @click="toggleFavorite">
								<font-awesome-icon :icon="['far', 'heart']" />
								Favorite
							</button>
							<button class="btn btn-outline-primary" @click="editMetadata">
								<font-awesome-icon :icon="['fas', 'edit']" />
								Edit
							</button>
							<button class="btn btn-outline-info" @click="addToEditList">
								<font-awesome-icon :icon="['fas', video.in_edit_list ? 'check' : 'list-check']" />
								{{ video.in_edit_list ? 'In Edit List' : 'Add to Edit List' }}
							</button>
							<button v-if="video.rating" class="btn btn-outline-warning">
								<font-awesome-icon :icon="['fas', 'star']" />
								{{ video.rating }}/5
							</button>
						</div>

						<!-- Description / Details -->
						<div class="video-details mt-4">
							<div class="detail-section">
								<h3>Video Details</h3>
								<div class="details-grid">
									<div class="detail-item">
										<strong>Filename:</strong>
										<span>{{ getFilename(video.file_path) }}</span>
									</div>
									<div class="detail-item">
										<strong>File Path:</strong>
										<span class="text-muted small">{{ video.file_path }}</span>
									</div>
									<div class="detail-item">
										<strong>Views:</strong>
										<span>{{ video.play_count || 0 }}</span>
									</div>
									<div class="detail-item">
										<strong>Date Added:</strong>
										<span>{{ formatDate(video.created_at) }}</span>
									</div>
									<div v-if="video.updated_at" class="detail-item">
										<strong>Last Modified:</strong>
										<span>{{ formatDate(video.updated_at) }}</span>
									</div>
								</div>
							</div>

							<!-- Performers -->
							<div v-if="video.performers && video.performers.length > 0" class="detail-section mt-4">
								<h3>
									<font-awesome-icon :icon="['fas', 'users']" class="me-2" />
									Performers ({{ video.performers.length }})
								</h3>
								<div class="performers-list">
									<div v-for="performer in video.performers" :key="performer.id" class="performer-chip" @click="goToPerformer(performer.id)">
										<div class="performer-avatar">
											<img
												v-if="performer.metadata_obj?.image_url"
												:src="performer.metadata_obj.image_url"
												:alt="performer.name"
											/>
											<div v-else class="avatar-placeholder">
												<font-awesome-icon :icon="['fas', 'user']" />
											</div>
										</div>
										<div class="performer-info">
											<div class="performer-name">
												<font-awesome-icon v-if="performer.zoo" :icon="['fas', 'dog']" class="text-danger me-1" />
												{{ performer.name }}
											</div>
											<div class="performer-stats">
												{{ performer.video_count || 0 }} videos
											</div>
										</div>
									</div>
								</div>
							</div>

							<!-- Studio -->
							<div v-if="video.studios && video.studios.length > 0" class="detail-section mt-4">
								<h3>
									<font-awesome-icon :icon="['fas', 'building']" class="me-2" />
									Studio
								</h3>
								<div class="studio-info" @click="goToStudio(video.studios[0].id)">
									<font-awesome-icon :icon="['fas', 'building']" class="me-2" />
									{{ video.studios[0].name }}
								</div>
							</div>

							<!-- Tags -->
							<div class="detail-section mt-4">
								<h3>
									<font-awesome-icon :icon="['fas', 'tags']" class="me-2" />
									Tags
								</h3>
								<div class="tags-container">
									<span
										v-for="tag in video.tags"
										:key="tag.id"
										class="tag-chip"
										:style="{ backgroundColor: tag.color || '#6c757d' }"
									>
										<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" class="me-1" />
										{{ tag.name }}
									</span>
									<button class="btn btn-sm btn-outline-primary" @click="openTagModal">
										<font-awesome-icon :icon="['fas', 'plus']" />
										Add Tag
									</button>
								</div>
							</div>
						</div>
					</div>
				</div>

				<!-- Sidebar Column -->
				<div class="col-lg-3">
					<div class="sidebar">
						<!-- Related Videos -->
						<div class="related-section">
							<h3>Related Videos</h3>
							<div v-if="relatedVideos.length === 0" class="text-muted text-center py-4">
								<p>No related videos found</p>
							</div>
							<div v-else class="related-videos-list">
								<div
									v-for="relatedVideo in relatedVideos"
									:key="relatedVideo.id"
									class="related-video-item"
									@click="goToVideo(relatedVideo.id)"
								>
									<div class="related-thumbnail">
										<img
											v-if="relatedVideo.thumbnail_path"
											:src="getThumbnailURL(relatedVideo)"
											:alt="relatedVideo.title"
										/>
										<div v-else class="thumbnail-placeholder">
											<font-awesome-icon :icon="['fas', 'video']" />
										</div>
										<div v-if="relatedVideo.duration" class="duration-badge">
											{{ formatDuration(relatedVideo.duration) }}
										</div>
									</div>
									<div class="related-info">
										<div class="related-title">{{ relatedVideo.title }}</div>
										<div class="related-meta">
											<span v-if="relatedVideo.resolution">{{ relatedVideo.resolution }}</span>
											<span v-if="relatedVideo.play_count > 0">
												<font-awesome-icon :icon="['fas', 'eye']" />
												{{ relatedVideo.play_count }}
											</span>
										</div>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>

	</div>
</template>

<script>
import { videosAPI } from '@/services/api'
import { getAssetURL } from '@/services/api'
import { useFormatters } from '@/composables/useFormatters'

export default {
	name: 'VideoPlayerPage',
	data() {
		return {
			video: null,
			loading: true,
			videoError: false,
			duration: 0,
			currentTime: 0,
			relatedVideos: [],
			showMetadataModal: false,
			showTagModal: false,
		}
	},
	computed: {
		videoUrl() {
			if (!this.video) return ''
			return `http://localhost:8080/api/v1/videos/${this.video.id}/stream`
		},
		videoMimeType() {
			if (!this.video) return 'video/mp4'
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
			}
			return mimeTypes[ext] || 'video/mp4'
		},
	},
	created() {
		const formatters = useFormatters()
		this.formatDuration = formatters.formatDuration
		this.formatFileSize = formatters.formatFileSize
		this.formatDate = formatters.formatDate
	},
	async mounted() {
		const videoId = this.$route.params.id
		await this.loadVideo(videoId)
		await this.loadRelatedVideos()
	},
	methods: {
		async loadVideo(id) {
			this.loading = true
			try {
				const response = await videosAPI.getById(id)
				// Response is already unwrapped by axios interceptor
				this.video = response

				// Only increment play count if we have a valid video with an ID
				if (this.video && this.video.id) {
					await this.incrementPlayCount()
				}
			} catch (error) {
				console.error('Failed to load video:', error)
				this.$toast.error('Error', 'Failed to load video')
				this.$router.push('/videos')
			} finally {
				this.loading = false
			}
		},
		async loadRelatedVideos() {
			if (!this.video) return

			try {
				// Get directory path from current video
				const currentPath = this.video.file_path
				if (!currentPath) {
					this.relatedVideos = []
					return
				}

				// Extract directory (everything before the last slash/backslash)
				const separator = currentPath.includes('\\') ? '\\' : '/'
				const directoryPath = currentPath.substring(0, currentPath.lastIndexOf(separator))

				// Try to search by directory path using the search endpoint
				// Extract the last folder name from the path for searching
				const folderName = directoryPath.substring(directoryPath.lastIndexOf(separator) + 1)

				console.log('Directory path:', directoryPath)
				console.log('Folder name for search:', folderName)

				// Try search first if folder name is meaningful
				let allVideos = []
				if (folderName && folderName.length > 2) {
					try {
						const searchParams = {
							query: folderName,
							library_id: this.video.library_id,
							per_page: 100
						}
						const searchResponse = await videosAPI.search(searchParams)

						if (Array.isArray(searchResponse)) {
							allVideos = searchResponse
						} else if (searchResponse?.data && Array.isArray(searchResponse.data)) {
							allVideos = searchResponse.data
						}

						console.log('Search returned videos:', allVideos.length)
					} catch (searchError) {
						console.log('Search failed, falling back to pagination:', searchError)
					}
				}

				// If search didn't work or returned nothing, fall back to pagination
				if (allVideos.length === 0) {
					const maxPages = 5 // Reduced from 10 for better performance
					for (let page = 1; page <= maxPages; page++) {
						const params = {
							library_id: this.video.library_id,
							per_page: 100,
							page: page
						}
						const response = await videosAPI.getAll(params)

						let pageVideos = []
						if (Array.isArray(response)) {
							pageVideos = response
						} else if (response?.data && Array.isArray(response.data)) {
							pageVideos = response.data
						}

						allVideos.push(...pageVideos)

						if (pageVideos.length < 100) {
							break
						}

						// Stop if we already found enough related videos
						const relatedCount = allVideos.filter(v => {
							if (v.id === this.video.id) return false
							if (!v.file_path) return false
							const vSeparator = v.file_path.includes('\\') ? '\\' : '/'
							const videoDir = v.file_path.substring(0, v.file_path.lastIndexOf(vSeparator))
							return videoDir === directoryPath
						}).length

						if (relatedCount >= 10) {
							break
						}
					}
					console.log('Pagination returned videos:', allVideos.length)
				}

				// Filter videos from the same directory, exclude current video
				this.relatedVideos = allVideos
					.filter(v => {
						if (v.id === this.video.id) return false
						if (!v.file_path) return false

						const vSeparator = v.file_path.includes('\\') ? '\\' : '/'
						const videoDir = v.file_path.substring(0, v.file_path.lastIndexOf(vSeparator))
						return videoDir === directoryPath
					})
					.slice(0, 10)

				console.log('Related videos found:', this.relatedVideos.length)
			} catch (error) {
				console.error('Failed to load related videos:', error)
			}
		},
		async incrementPlayCount() {
			try {
				await videosAPI.update(this.video.id, { play_count: (this.video.play_count || 0) + 1 })
			} catch (error) {
				console.error('Failed to increment play count:', error)
			}
		},
		async toggleFavorite() {
			try {
				await videosAPI.update(this.video.id, { is_favorite: !this.video.is_favorite })
				this.video.is_favorite = !this.video.is_favorite
				this.$toast.success('Success', this.video.is_favorite ? 'Added to favorites' : 'Removed from favorites')
			} catch (error) {
				console.error('Failed to toggle favorite:', error)
				this.$toast.error('Error', 'Failed to update favorite status')
			}
		},
		async addToEditList() {
			try {
				await videosAPI.update(this.video.id, { in_edit_list: !this.video.in_edit_list })
				this.video.in_edit_list = !this.video.in_edit_list
				this.$toast.success('Success', this.video.in_edit_list ? 'Added to edit list' : 'Removed from edit list')
			} catch (error) {
				console.error('Failed to toggle edit list:', error)
				this.$toast.error('Error', 'Failed to update edit list')
			}
		},
		editMetadata() {
			this.$toast.info('Coming Soon', 'Metadata editing will be available soon')
		},
		openTagModal() {
			this.$toast.info('Coming Soon', 'Tag management will be available soon')
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
			console.log('Video ended')
		},
		onVideoError(event) {
			console.error('Video playback error:', event)
			this.videoError = true
		},
		goToVideo(videoId) {
			this.$router.push(`/watch/${videoId}`)
			this.loadVideo(videoId)
			this.loadRelatedVideos()
			window.scrollTo(0, 0)
		},
		goToPerformer(performerId) {
			this.$router.push(`/performers/${performerId}`)
		},
		goToStudio(studioId) {
			this.$router.push(`/studios?id=${studioId}`)
		},
		getThumbnailURL(video) {
			if (video.thumbnail_path) {
				return getAssetURL(video.thumbnail_path)
			}
			return `http://localhost:8080/api/v1/videos/${video.id}/thumbnail`
		},
		getFilename(path) {
			if (!path) return ''
			return path.split(/[\\/]/).pop()
		},
		// formatDuration, formatFileSize, and formatDate now provided by useFormatters composable
		formatBitrate(bitrate) {
			if (!bitrate) return ''
			const kbps = bitrate / 1000
			if (kbps < 1000) return kbps.toFixed(0) + ' Kbps'
			return (kbps / 1000).toFixed(1) + ' Mbps'
		},
		formatFPS(fps) {
			if (!fps) return ''
			return parseFloat(fps.toFixed(2))
		},
	},
	watch: {
		async '$route.params.id'(newId) {
			if (newId) {
				// Reset video state
				this.video = null
				this.videoError = false
				this.duration = 0
				this.currentTime = 0
				this.relatedVideos = []

				// Load new video first, then load related videos
				await this.loadVideo(newId)
				await this.loadRelatedVideos()
			}
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/video_player_page.css';
</style>
