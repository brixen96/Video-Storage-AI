<template>
	<div class="video-player-page">
		<div class="container-fluid">
			<div class="row g-4">
				<!-- Main Content Column -->
				<div class="col-lg-9">
					<!-- Video Player -->
					<div class="player-container">
						<!-- Scraped Video: native player when direct URL extracted (e.g. FapZoo, generic CDN sites) -->
						<video
							v-if="video && isScrapedVideo && directVideoUrl"
							class="main-video-player"
							controls
							autoplay
						>
							<source :src="directVideoUrl" type="video/mp4" />
							Your browser does not support the video tag.
						</video>

						<!-- Scraped Video: extracting video URL spinner -->
						<div v-else-if="video && isScrapedVideo && extractingVideoUrl" class="loading-overlay">
							<font-awesome-icon :icon="['fas', 'spinner']" spin size="3x" />
							<p>Extracting video URL...</p>
						</div>

						<!-- Scraped Video: embed iframe (PornHub, Rule34Video, or fallback) -->
						<iframe
							v-else-if="video && isScrapedVideo"
							:src="embedUrl"
							frameborder="0"
							border="0"
							scrolling="no"
							width="100%"
							height="100%"
							allowfullscreen
							class="scraped-video-embed"
						></iframe>

						<!-- Local Video Player -->
						<video
							v-else-if="video"
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
						<div v-if="videoError && !isScrapedVideo" class="video-error-overlay">
							<div class="error-content">
								<font-awesome-icon :icon="['fas', 'exclamation-triangle']" size="3x" class="error-icon" />
								<h4>Unable to Play Video</h4>
								<p v-if="!isBrowserPlayable && videoFileExt">
									<strong>{{ videoFileExt.toUpperCase().slice(1) }}</strong> files are not natively supported by browsers.
									<span v-if="video?.codec"> (Codec: {{ video.codec }})</span>
									Consider converting to MP4.
								</p>
								<p v-else-if="video?.codec && ['hevc', 'h265', 'av1', 'prores'].includes(video.codec.toLowerCase())">
									Codec <strong>{{ video.codec }}</strong> is not supported by your browser. Consider converting to MP4 (H.264).
								</p>
								<p v-else>
									This video could not be played.
									<span v-if="video?.codec"> (Codec: {{ video.codec }})</span>
									<span v-if="videoFileExt"> Format: {{ videoFileExt }}</span>
								</p>
							</div>
						</div>
						<div v-if="loading" class="loading-overlay">
							<font-awesome-icon :icon="['fas', 'spinner']" spin size="3x" />
							<p>Loading video...</p>
						</div>
					</div>

					<!-- Auto-play next countdown overlay -->
					<Transition name="autoplay-fade">
						<div v-if="autoPlayCountdown > 0 && relatedVideos.length > 0" class="autoplay-overlay">
							<div class="autoplay-card">
								<div class="autoplay-label">Playing next in</div>
								<div class="autoplay-countdown">{{ autoPlayCountdown }}</div>
								<div class="autoplay-next-title">{{ relatedVideos[0].title }}</div>
								<div class="autoplay-actions">
									<button class="btn btn-sm btn-primary" @click="goToVideo(relatedVideos[0].id)">
										<font-awesome-icon :icon="['fas', 'play']" class="me-1" />
										Play Now
									</button>
									<button class="btn btn-sm btn-outline-secondary" @click="cancelAutoPlay">Cancel</button>
								</div>
							</div>
						</div>
					</Transition>

					<!-- Keyboard shortcut hint -->
					<Transition name="hint-fade">
						<div v-if="showKeyboardHint && !isScrapedVideo" class="keyboard-hint">
							<span>Space/K pause · J/← -10s · L/→ +10s · M mute · F fullscreen</span>
						</div>
					</Transition>

					<!-- Video Info -->
					<div v-if="video" class="video-info-section">
						<h1 class="video-title">{{ video.title }}</h1>

						<!-- Metadata Badges -->
						<div class="video-metadata">
							<span v-if="video.resolution" class="badge bg-primary">
								<font-awesome-icon :icon="['fas', 'video']" class="me-1" />
								{{ getVideoResolution(video) }}
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
							<span v-if="isScrapedVideo && video.view_count > 0" class="badge bg-purple">
								<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
								{{ formatViewCount(video.view_count) }}
							</span>
							<span v-if="isScrapedVideo && video.rating > 0" class="badge bg-success">
								<font-awesome-icon :icon="['fas', 'thumbs-up']" class="me-1" />
								{{ video.rating }}%
							</span>
							<span v-if="isScrapedVideo && video.like_count > 0" class="badge bg-danger">
								<font-awesome-icon :icon="['fas', 'heart']" class="me-1" />
								{{ formatViewCount(video.like_count) }}
							</span>
						</div>

						<!-- Action Buttons (Only for local videos) -->
						<div v-if="!isScrapedVideo" class="action-buttons mt-3">
							<button v-if="video.is_favorite" class="btn btn-danger" @click="toggleFavorite">
								<font-awesome-icon :icon="['fas', 'heart']" />
								Favorited
							</button>
							<button v-else class="btn btn-outline-danger" @click="toggleFavorite">
								<font-awesome-icon :icon="['far', 'heart']" />
								Favorite
							</button>
							<button
								class="btn"
								:class="video.watched ? 'btn-success' : 'btn-outline-success'"
								@click="toggleWatched"
								title="Toggle watched status"
							>
								<font-awesome-icon :icon="['fas', 'eye']" />
								{{ video.watched ? 'Watched' : 'Mark Watched' }}
							</button>
							<button class="btn btn-outline-primary" @click="editMetadata">
								<font-awesome-icon :icon="['fas', 'edit']" />
								Edit
							</button>
							<button class="btn btn-outline-info" @click="addToEditList">
								<font-awesome-icon :icon="['fas', video.in_edit_list ? 'check' : 'list-check']" />
								{{ video.in_edit_list ? 'In Edit List' : 'Add to Edit List' }}
							</button>
							<!-- Star rating -->
							<div class="star-rating" title="Rate this video">
								<font-awesome-icon
									v-for="star in 5"
									:key="star"
									:icon="[star <= (hoverRating || video.rating || 0) ? 'fas' : 'far', 'star']"
									class="star-icon"
									:class="{ filled: star <= (hoverRating || video.rating || 0), hovered: hoverRating > 0 && star <= hoverRating }"
									@mouseenter="hoverRating = star"
									@mouseleave="hoverRating = 0"
									@click.stop="setRating(star)"
								/>
								<span v-if="video.rating" class="star-label">{{ video.rating }}/5</span>
							</div>
						</div>

						<!-- Scraped Video Actions -->
						<div v-if="isScrapedVideo" class="action-buttons mt-3">
							<a :href="video.url" target="_blank" class="btn btn-outline-primary">
								<font-awesome-icon :icon="['fas', 'external-link-alt']" class="me-1" />
								Open Source
							</a>
							<!-- Download button for Rule34Video -->
							<button
								v-if="isRule34Video"
								class="btn btn-outline-success"
								@click="fetchAndDownload"
								:disabled="downloadLoading"
							>
								<font-awesome-icon :icon="['fas', downloadLoading ? 'spinner' : 'download']" :spin="downloadLoading" class="me-1" />
								{{ downloadLoading ? 'Fetching...' : 'Download (Best Quality)' }}
							</button>
							<!-- Download button when a direct video URL was extracted -->
							<button
								v-if="directVideoUrl"
								class="btn btn-outline-success"
								@click="downloadDirectVideo"
							>
								<font-awesome-icon :icon="['fas', 'download']" class="me-1" />
								Download
							</button>
						</div>

						<!-- Description / Details -->
						<div class="video-details mt-4">
							<div v-if="!isScrapedVideo" class="detail-section">
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

							<div v-if="isScrapedVideo" class="detail-section">
								<h3>Video Details</h3>
								<div class="details-grid">
									<div v-if="video.uploader || video.author" class="detail-item">
										<strong>Uploader:</strong>
										<span>{{ video.uploader || video.author }}</span>
									</div>
									<div class="detail-item">
										<strong>Source:</strong>
										<span>{{ video.source }}</span>
									</div>
									<div class="detail-item">
										<strong>Discovered:</strong>
										<span>{{ formatDate(video.discovered_at) }}</span>
									</div>
									<div class="detail-item">
										<strong>Status:</strong>
										<span>{{ video.status }}</span>
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
												v-if="getPerformerThumbURL(performer)"
												:src="getPerformerThumbURL(performer)"
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
						<!-- Linked Version Panel -->
						<div v-if="linkedVideo" class="related-section linked-version-panel mb-4">
							<h3 class="text-info">
								<font-awesome-icon :icon="['fas', 'link']" class="me-2" />
								Linked Version
							</h3>
							<div class="linked-video-card" @click="goToVideo(linkedVideo.id)">
								<div class="related-thumbnail">
									<img
										v-if="linkedVideo.thumbnail_path"
										:src="getThumbnailURL(linkedVideo)"
										:alt="linkedVideo.title"
									/>
									<div v-else class="thumbnail-placeholder">
										<font-awesome-icon :icon="['fas', 'video']" />
									</div>
									<div v-if="linkedVideo.duration" class="duration-badge">
										{{ formatDuration(linkedVideo.duration) }}
									</div>
								</div>
								<div class="related-info">
									<div class="related-title">{{ linkedVideo.title }}</div>
									<div class="related-meta">
										<span v-if="linkedVideo.resolution" class="badge bg-primary me-1" style="font-size:0.65rem">{{ linkedVideo.resolution }}</span>
										<span v-if="linkedVideo.file_size" class="text-muted" style="font-size:0.7rem">{{ formatFileSize(linkedVideo.file_size) }}</span>
									</div>
								</div>
							</div>
						</div>

						<!-- Related Videos -->
						<div class="related-section">
							<h3>
								<template v-if="video && video.performers && video.performers.length > 0">
									More from {{ video.performers[0].name }}
								</template>
								<template v-else>Related Videos</template>
							</h3>
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
import { videosAPI, openInFirefox, scrapersAPI, jdownloaderAPI } from '@/services/api'
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
			downloadLoading: false,
			directVideoUrl: null,
			extractingVideoUrl: false,
			autoPlayCountdown: 0,
			autoPlayTimer: null,
			showKeyboardHint: false,
			keyboardHintTimer: null,
			hoverRating: 0,
			linkedVideo: null,
		}
	},
	computed: {
		isScrapedVideo() {
			// Check if this is a scraped video (has a source field)
			return this.video && this.video.source
		},
		isRule34Video() {
			// Check if this is a Rule34Video
			return this.video && this.video.url && this.video.url.includes('rule34video.com')
		},
		embedUrl() {
			if (!this.isScrapedVideo) return ''

			const url = this.video.url

			// Handle PornHub URLs - extract viewkey and use embed format
			// URL: https://www.pornhub.com/view_video.php?viewkey=6796ae6bec815
			// Embed: https://www.pornhub.com/embed/6796ae6bec815
			if (url && url.includes('pornhub.com')) {
				const viewkeyMatch = url.match(/viewkey=([a-zA-Z0-9]+)/)
				if (viewkeyMatch && viewkeyMatch[1]) {
					return `https://www.pornhub.com/embed/${viewkeyMatch[1]}`
				}
			}

			// Handle Rule34Video URLs - extract video ID and use embed format
			// URL: https://rule34video.com/video/4202425/video-title-slug/
			// Embed: https://rule34video.com/embed/4202425
			if (url && url.includes('rule34video.com')) {
				const videoIdMatch = url.match(/\/video\/(\d+)\//)
				if (videoIdMatch && videoIdMatch[1]) {
					return `https://rule34video.com/embed/${videoIdMatch[1]}`
				}
			}

			// Extract video ID from the URL for other sites
			// Pattern: https://www.WEBSITE.net/12345/title-slug/
			const videoId = this.extractVideoIdFromUrl(url)
			if (!videoId) return url

			// Extract the host from the source URL
			try {
				const sourceUrl = new URL(url)
				const host = sourceUrl.origin
				return `${host}/embed/${videoId}/`
			} catch (error) {
				console.error('Failed to construct embed URL:', error)
				return url
			}
		},
		videoUrl() {
			if (!this.video) return ''
			const origin = window.VSAI_API_URL || 'http://localhost:8080'
			return `${origin}/api/v1/videos/${this.video.id}/stream`
		},
		videoFileExt() {
			if (!this.video?.file_path) return ''
			const parts = this.video.file_path.replace(/\\/g, '/').split('/')
			const filename = parts[parts.length - 1] || ''
			const dot = filename.lastIndexOf('.')
			return dot >= 0 ? filename.slice(dot).toLowerCase() : ''
		},
		videoMimeType() {
			const ext = this.videoFileExt
			const mimeTypes = {
				'.mp4': 'video/mp4',
				'.webm': 'video/webm',
				'.ogg': 'video/ogg',
				'.mkv': 'video/x-matroska',
				'.avi': 'video/x-msvideo',
				'.mov': 'video/quicktime',
				'.wmv': 'video/x-ms-wmv',
				'.flv': 'video/x-flv',
				'.m4v': 'video/mp4',
				'.mpg': 'video/mpeg',
				'.mpeg': 'video/mpeg',
			}
			return mimeTypes[ext] || 'video/mp4'
		},
		isBrowserPlayable() {
			const playable = ['.mp4', '.webm', '.ogg', '.m4v', '.mpg', '.mpeg']
			return playable.includes(this.videoFileExt)
		},
	},
	created() {
		const formatters = useFormatters()
		this.formatDuration = formatters.formatDuration
		this.formatFileSize = formatters.formatFileSize
		this.formatDate = formatters.formatDate
	},
	async mounted() {
		window.addEventListener('keydown', this.handleKeyboard)
		this.keyboardHintTimer = setTimeout(() => {
			this.showKeyboardHint = true
			setTimeout(() => { this.showKeyboardHint = false }, 3000)
		}, 1500)
		const videoId = this.$route.params.id
		// Detect video type from route name
		const videoType = this.$route.name === 'ScrapedVideoPlayer' ? 'scraped' : 'local'
		await this.loadVideo(videoId, videoType)
		await this.loadRelatedVideos()
	},
	methods: {
		async loadVideo(id, type = 'local') {
			this.loading = true
			try {
				let response
				if (type === 'scraped') {
					// Load scraped video from scraper API
					response = await fetch(`http://localhost:8080/api/v1/scraper/videos/${id}`)
					const data = await response.json()
					if (data.success) {
						this.video = data.data
					} else {
						throw new Error(data.message || 'Failed to load scraped video')
					}
				} else {
					// Load local video
					response = await videosAPI.getById(id)
					// Response is already unwrapped by axios interceptor
					this.video = response
				}

				// Only increment play count if we have a valid video with an ID and it's a local video
				if (this.video && this.video.id && type === 'local') {
					await this.incrementPlayCount()
				}

				// Load confirmed linked video if one exists
				this.linkedVideo = null
				if (this.video?.linked_video_id) {
					this.loadLinkedVideo(this.video.linked_video_id)
				}

				// For scraped videos, try to extract direct video URL in background
				if (this.video && type === 'scraped') {
					this.fetchDirectVideoUrl()
				}
			} catch (error) {
				console.error('Failed to load video:', error)
				this.$toast.error('Error', 'Failed to load video')
				this.$router.push('/videos')
			} finally {
				this.loading = false
			}
		},
		async loadLinkedVideo(linkedId) {
			try {
				const res = await videosAPI.getById(linkedId)
				this.linkedVideo = res.data || res
			} catch (err) {
				console.error('Failed to load linked video:', err)
			}
		},
		async loadRelatedVideos() {
			if (!this.video) return

			try {
				// 1. Performer-based related videos (most semantically relevant)
				if (this.video.performers && this.video.performers.length > 0) {
					const seen = new Set([this.video.id])
					const results = []

					// Fetch from up to 2 performers in parallel
					const performerFetches = this.video.performers.slice(0, 2).map((p) =>
						videosAPI.getAll({ performer_id: p.id, per_page: 12 }).catch(() => null)
					)
					const responses = await Promise.all(performerFetches)

					for (const res of responses) {
						const videos = Array.isArray(res) ? res : res?.data
						if (!Array.isArray(videos)) continue
						for (const v of videos) {
							if (!seen.has(v.id)) {
								seen.add(v.id)
								results.push(v)
							}
						}
					}

					if (results.length > 0) {
						this.relatedVideos = results.slice(0, 12)
						return
					}
				}

				// 2. Folder-based fallback for local videos with no performer data
				const currentPath = this.video.file_path
				if (!currentPath) return

				const separator = currentPath.includes('\\') ? '\\' : '/'
				const directoryPath = currentPath.substring(0, currentPath.lastIndexOf(separator))
				const folderName = directoryPath.substring(directoryPath.lastIndexOf(separator) + 1)

				if (!folderName || folderName.length <= 2) return

				const searchResponse = await videosAPI.search({
					query: folderName,
					library_id: this.video.library_id,
					per_page: 50,
				}).catch(() => null)

				const candidates = Array.isArray(searchResponse)
					? searchResponse
					: searchResponse?.data ?? []

				this.relatedVideos = candidates
					.filter((v) => {
						if (v.id === this.video.id || !v.file_path) return false
						const sep = v.file_path.includes('\\') ? '\\' : '/'
						return v.file_path.substring(0, v.file_path.lastIndexOf(sep)) === directoryPath
					})
					.slice(0, 12)
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
		async setRating(star) {
			// Toggle off if same star clicked
			const newRating = star === this.video.rating ? 0 : star
			try {
				await videosAPI.update(this.video.id, { rating: newRating })
				this.video.rating = newRating
				this.hoverRating = 0
				this.$toast.success('Success', newRating ? `Rated ${newRating}/5` : 'Rating cleared')
			} catch (error) {
				console.error('Failed to set rating:', error)
				this.$toast.error('Error', 'Failed to update rating')
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
		async toggleWatched() {
			try {
				await videosAPI.markWatched(this.video.id, !this.video.watched)
				this.video.watched = !this.video.watched
				this.$toast.success('Success', this.video.watched ? 'Marked as watched' : 'Marked as unwatched')
			} catch (error) {
				console.error('Failed to toggle watched:', error)
				this.$toast.error('Error', 'Failed to update watched status')
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
		async fetchDirectVideoUrl() {
			if (!this.video?.url || !this.isScrapedVideo) return
			// PornHub and Rule34Video have working embed iframes — skip extraction
			const url = this.video.url
			if (url.includes('pornhub.com') || url.includes('rule34video.com')) return

			this.extractingVideoUrl = true
			try {
				const response = await scrapersAPI.extractVideoUrl(url)
				if (response?.direct_url) {
					this.directVideoUrl = response.direct_url
				}
			} catch (error) {
				console.error('Failed to extract video URL:', error)
			} finally {
				this.extractingVideoUrl = false
			}
		},
		async downloadDirectVideo() {
			if (!this.directVideoUrl) return
			const title = this.video?.title || 'Scraped Video'
			const scraperName = this.video?.source || 'Scraper'
			try {
				const result = await jdownloaderAPI.sendLinks([{
					url: this.directVideoUrl,
					packageName: title,
					downloadPath: `/output/${scraperName}`,
					autoStart: true,
				}])
				if (result?.successCount > 0) {
					this.$toast.success('Sent to JDownloader')
				} else {
					this.$toast.error('Failed to send to JDownloader')
				}
			} catch {
				this.$toast.error('JDownloader is not available')
			}
		},
		async fetchAndDownload() {
			if (!this.video || !this.video.url) return

			this.downloadLoading = true
			try {
				const response = await fetch(`http://localhost:8080/api/v1/scraper/rule34video/download-url?url=${encodeURIComponent(this.video.url)}`)
				const data = await response.json()

				if (data.success && data.data && data.data.download_url) {
					openInFirefox(data.data.download_url)
					this.$toast.success('Download started')
				} else {
					throw new Error(data.message || 'Failed to get download URL')
				}
			} catch (error) {
				console.error('Failed to fetch download URL:', error)
				this.$toast.error('Failed to get download URL. Try opening the source page.')
			} finally {
				this.downloadLoading = false
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
			if (!this.isScrapedVideo && this.relatedVideos.length > 0) {
				this.autoPlayCountdown = 5
				this.autoPlayTimer = setInterval(() => {
					this.autoPlayCountdown--
					if (this.autoPlayCountdown <= 0) {
						const nextId = this.relatedVideos[0].id
						this.cancelAutoPlay()
						this.goToVideo(nextId)
					}
				}, 1000)
			}
		},
		cancelAutoPlay() {
			if (this.autoPlayTimer) { clearInterval(this.autoPlayTimer); this.autoPlayTimer = null }
			this.autoPlayCountdown = 0
		},
		handleKeyboard(e) {
			if (this.isScrapedVideo) return
			const v = this.$refs.videoElement
			if (!v) return
			const tag = e.target?.tagName
			if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
			switch (e.key) {
				case ' ': case 'k': case 'K':
					e.preventDefault(); v.paused ? v.play() : v.pause(); break
				case 'ArrowLeft': case 'j': case 'J':
					e.preventDefault(); v.currentTime = Math.max(0, v.currentTime - 10); break
				case 'ArrowRight': case 'l': case 'L':
					e.preventDefault(); v.currentTime = Math.min(v.duration, v.currentTime + 10); break
				case 'ArrowUp':
					e.preventDefault(); v.volume = Math.min(1, v.volume + 0.1); break
				case 'ArrowDown':
					e.preventDefault(); v.volume = Math.max(0, v.volume - 0.1); break
				case 'm': case 'M':
					e.preventDefault(); v.muted = !v.muted; break
				case 'f': case 'F':
					e.preventDefault()
					document.fullscreenElement ? document.exitFullscreen() : v.requestFullscreen()
					break
			}
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
		getPerformerThumbURL(performer) {
			if (performer.thumbnail_path) {
				if (performer.thumbnail_path.startsWith('http')) return performer.thumbnail_path
				return getAssetURL(performer.thumbnail_path)
			}
			return performer.metadata_obj?.image_url || null
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
		extractVideoIdFromUrl(url) {
			if (!url) return null
			// Pattern: https://www.WEBSITE.net/12345/title-slug/
			// Or: https://www.WEBSITE.net/12345/
			const match = url.match(/\/(\d+)\//)
			return match ? match[1] : null
		},
		getVideoResolution(video) {
			if (!video) return ''

			// For scraped videos, check metadata
			if (video.metadata) {
				try {
					const metadata = typeof video.metadata === 'string'
						? JSON.parse(video.metadata)
						: video.metadata
					if (metadata.max_resolution) {
						return metadata.max_resolution
					}
				} catch (e) {
					console.error('Failed to parse video metadata:', e)
				}
			}

			// Fallback to video.resolution
			return video.resolution || ''
		},
		formatViewCount(count) {
			if (!count) return '0'
			if (count >= 1000000) {
				return (count / 1000000).toFixed(1) + 'M'
			}
			if (count >= 1000) {
				return (count / 1000).toFixed(1) + 'K'
			}
			return count.toString()
		},
	},
	beforeUnmount() {
		window.removeEventListener('keydown', this.handleKeyboard)
		this.cancelAutoPlay()
		if (this.keyboardHintTimer) clearTimeout(this.keyboardHintTimer)
	},
	watch: {
		async '$route.params.id'(newId) {
			if (newId) {
				// Reset video state
				this.cancelAutoPlay()
				this.video = null
				this.videoError = false
				this.duration = 0
				this.currentTime = 0
				this.relatedVideos = []
				this.directVideoUrl = null
				this.extractingVideoUrl = false
				this.linkedVideo = null

				// Load new video first, then load related videos
				const videoType = this.$route.name === 'ScrapedVideoPlayer' ? 'scraped' : 'local'
				await this.loadVideo(newId, videoType)
				await this.loadRelatedVideos()
			}
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/video_player_page.css';
</style>
