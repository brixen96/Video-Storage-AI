<template>
	<div class="videos-page">
		<!-- Top Bar -->
		<div class="top-bar">
			<div class="container-fluid">
				<div class="row align-items-center g-3">
					<div class="col-md-3">
						<h1>
							<font-awesome-icon :icon="['fas', 'video']" />
							Videos
							<span class="video-count">({{ totalVideos }})</span>
						</h1>
					</div>
					<div class="col-md-2">
						<select v-model="selectedLibrary" class="form-select" @change="loadVideos">
							<option value="">All Libraries</option>
							<option v-for="library in libraries" :key="library.id" :value="library.id">{{ library.name }}</option>
						</select>
					</div>
					<div class="col-md-3">
						<div class="input-group">
							<span class="input-group-text">
								<font-awesome-icon :icon="['fas', 'search']" />
							</span>
							<input v-model="searchQuery" type="text" class="form-control" placeholder="Search videos, performers, studios, tags..." @input="debounceSearch" />
						</div>
					</div>
					<div class="col-md-4 text-end">
						<div class="d-flex gap-2 justify-content-end">
							<button class="btn btn-outline-secondary" @click="toggleView">
								<font-awesome-icon :icon="viewMode === 'grid' ? ['fas', 'list'] : ['fas', 'th']" />
								{{ viewMode === 'grid' ? 'List' : 'Grid' }}
							</button>
							<button class="btn btn-outline-secondary" @click="toggleFilters">
								<font-awesome-icon :icon="['fas', 'filter']" />
								Filters
							</button>
							<button v-if="selectedVideos.length > 0" class="btn btn-primary" @click="showBulkActions = true">
								<font-awesome-icon :icon="['fas', 'tasks']" />
								Bulk ({{ selectedVideos.length }})
							</button>
							<button class="btn btn-success" @click="scanVideos">
								<font-awesome-icon :icon="['fas', 'sync']" />
								Scan
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>

		<div class="page-content">
			<!-- Left Sidebar (Filters) -->
			<div v-if="showFilters" class="filter-sidebar" :class="{ collapsed: !showFilters }">
				<div class="filter-panel">
					<h3>Filters</h3>

					<!-- Sort -->
					<div class="filter-group">
						<label>Sort By</label>
						<select v-model="sortBy" class="form-select form-select-sm" @change="loadVideos">
							<option value="created_at">Date Added</option>
							<option value="title">Title</option>
							<option value="duration">Duration</option>
							<option value="play_count">Views</option>
						</select>
						<select v-model="sortOrder" class="form-select form-select-sm mt-2" @change="loadVideos">
							<option value="desc">Descending</option>
							<option value="asc">Ascending</option>
						</select>
					</div>

					<!-- Duration Range -->
					<div class="filter-group">
						<label>Duration (minutes)</label>
						<div class="d-flex gap-2">
							<input v-model.number="filters.minDuration" type="number" class="form-control form-control-sm" placeholder="Min" @change="loadVideos" />
							<input v-model.number="filters.maxDuration" type="number" class="form-control form-control-sm" placeholder="Max" @change="loadVideos" />
						</div>
					</div>

					<!-- Resolution -->
					<div class="filter-group">
						<label>Resolution</label>
						<select v-model="filters.resolution" class="form-select form-select-sm" @change="loadVideos">
							<option value="">All</option>
							<option value="1920x1080">1080p</option>
							<option value="1280x720">720p</option>
							<option value="3840x2160">4K</option>
						</select>
					</div>

					<!-- Performers -->
					<div v-if="performers.length > 0" class="filter-group">
						<label>Performers</label>
						<select v-model="filters.performerId" class="form-select form-select-sm" @change="loadVideos">
							<option :value="null">All</option>
							<option v-for="performer in performers" :key="performer.id" :value="performer.id">{{ performer.name }}</option>
						</select>
					</div>

					<!-- Studios -->
					<div v-if="studios.length > 0" class="filter-group">
						<label>Studios</label>
						<select v-model="filters.studioId" class="form-select form-select-sm" @change="loadVideos">
							<option :value="null">All</option>
							<option v-for="studio in studios" :key="studio.id" :value="studio.id">{{ studio.name }}</option>
						</select>
					</div>

					<!-- File Size Range -->
					<div class="filter-group">
						<label>File Size (MB)</label>
						<div class="d-flex gap-2">
							<input v-model.number="filters.minSize" type="number" class="form-control form-control-sm" placeholder="Min" @change="loadVideos" />
							<input v-model.number="filters.maxSize" type="number" class="form-control form-control-sm" placeholder="Max" @change="loadVideos" />
						</div>
					</div>

					<!-- Date Range -->
					<div class="filter-group">
						<label>Date Range</label>
						<input v-model="filters.dateFrom" type="date" class="form-control form-control-sm mb-2" @change="loadVideos" />
						<input v-model="filters.dateTo" type="date" class="form-control form-control-sm" @change="loadVideos" />
					</div>

					<!-- Toggle Filters -->
					<div class="filter-group">
						<label>Quick Filters</label>
						<div class="form-check">
							<input id="hasPreview" v-model="filters.hasPreview" type="checkbox" class="form-check-input" @change="loadVideos" />
							<label class="form-check-label" for="hasPreview">Has Preview</label>
						</div>
						<div class="form-check">
							<input id="missingMetadata" v-model="filters.missingMetadata" type="checkbox" class="form-check-input" @change="loadVideos" />
							<label class="form-check-label" for="missingMetadata">Missing Metadata</label>
						</div>
					</div>

					<!-- Clear Filters -->
					<button class="btn btn-sm btn-outline-secondary w-100 mt-3" @click="clearFilters">Clear All</button>
				</div>
			</div>

			<!-- Main Content Area -->
			<div class="main-content p-0 m-0" :class="{ 'full-width': !showFilters }">
				<div class="container-fluid px-0">
					<!-- Loading State -->
					<div v-if="loading" class="text-center py-5">
						<div class="spinner-border text-primary" role="status">
							<span class="visually-hidden">Loading...</span>
						</div>
					</div>

					<!-- Empty State -->
					<div v-else-if="videos.length === 0" class="text-center py-5">
						<font-awesome-icon :icon="['fas', 'video']" size="3x" class="mb-3" />
						<p>No videos found</p>
						<button class="btn btn-primary" @click="scanVideos">Scan for Videos</button>
					</div>

					<!-- Grid View -->
					<div v-else-if="viewMode === 'grid'" class="video-grid p-3">
						<VideoCard
							v-for="video in videos"
							:key="video.id"
							:video="video"
							:is-selected="selectedVideos.includes(video.id)"
							@click="openVideoDetails"
							@toggle-select="toggleVideoSelection"
							@context-menu="showContextMenu"
							@play="playVideo"
							@add-tag="openTagModal"
							@edit-metadata="editMetadata"
							@open-performer="openPerformer"
							@open-studio="openStudio"
						/>
					</div>

					<!-- List View -->
					<div v-else class="video-list">
						<table class="table-dark table-hover text-bg-dark w-100">
							<thead>
								<tr>
									<th style="width: 40px">
										<input type="checkbox" @change="toggleSelectAll" />
									</th>
									<th>Title</th>
									<th>Duration</th>
									<th>Resolution</th>
									<th>Size</th>
									<th>Performers</th>
									<th>Studio</th>
									<th>Views</th>
									<th>Actions</th>
								</tr>
							</thead>
							<tbody>
								<tr v-for="video in videos" :key="video.id" :class="{ selected: selectedVideos.includes(video.id) }" @click="openVideoDetails(video)">
									<td @click.stop>
										<input type="checkbox" :checked="selectedVideos.includes(video.id)" @change="toggleVideoSelection(video)" />
									</td>
									<td>{{ video.title }}</td>
									<td>{{ formatDuration(video.duration) }}</td>
									<td>{{ video.resolution }}</td>
									<td>{{ formatFileSize(video.file_size) }}</td>
									<td>{{ video.performers?.map((p) => p.name).join(', ') || '-' }}</td>
									<td>{{ video.studios?.[0]?.name || '-' }}</td>
									<td>{{ video.play_count || 0 }}</td>
									<td @click.stop>
										<button class="btn btn-sm btn-outline-primary" @click="editMetadata(video)">
											<font-awesome-icon :icon="['fas', 'edit']" />
										</button>
									</td>
								</tr>
							</tbody>
						</table>
					</div>

					<!-- Pagination -->
					<div v-if="totalPages > 1" class="pagination-controls mt-4">
						<nav>
							<ul class="pagination justify-content-center">
								<li class="page-item" :class="{ disabled: currentPage === 1 }">
									<a class="page-link" @click="goToPage(currentPage - 1)">Previous</a>
								</li>
								<li v-for="page in visiblePages" :key="page" class="page-item" :class="{ active: page === currentPage }">
									<a class="page-link" @click="goToPage(page)">{{ page }}</a>
								</li>
								<li class="page-item" :class="{ disabled: currentPage === totalPages }">
									<a class="page-link" @click="goToPage(currentPage + 1)">Next</a>
								</li>
							</ul>
						</nav>
					</div>
				</div>
			</div>
		</div>

		<!-- Video Details Panel (Right Drawer) -->
		<div v-if="selectedVideo" class="video-details-panel" :class="{ open: selectedVideo }">
			<div class="panel-header">
				<h2>{{ selectedVideo.title }}</h2>
				<button class="btn-close" @click="selectedVideo = null"></button>
			</div>
			<div class="panel-body">
				<!-- Metadata -->
				<div class="detail-section">
					<h3>Metadata</h3>
					<div class="info-grid">
						<div class="info-item"><strong>Filename:</strong> {{ getFilename(selectedVideo.file_path) }}</div>
						<div class="info-item"><strong>Path:</strong> {{ selectedVideo.file_path }}</div>
						<div class="info-item"><strong>Duration:</strong> {{ formatDuration(selectedVideo.duration) }}</div>
						<div class="info-item"><strong>Resolution:</strong> {{ selectedVideo.resolution }}</div>
						<div class="info-item"><strong>Codec:</strong> {{ selectedVideo.codec }}</div>
						<div class="info-item"><strong>Size:</strong> {{ formatFileSize(selectedVideo.file_size) }}</div>
						<div class="info-item"><strong>Bitrate:</strong> {{ formatBitrate(selectedVideo.bitrate) }}</div>
						<div class="info-item"><strong>FPS:</strong> {{ selectedVideo.fps }}</div>
						<div class="info-item"><strong>Views:</strong> {{ selectedVideo.play_count || 0 }}</div>
					</div>
				</div>

				<!-- Actions -->
				<div class="detail-section">
					<h3>Actions</h3>
					<div class="action-buttons">
						<button class="btn btn-sm btn-primary" @click="playVideo(selectedVideo)">
							<font-awesome-icon :icon="['fas', 'play']" />
							Play
						</button>
						<button class="btn btn-sm btn-outline-primary" @click="editMetadata(selectedVideo)">
							<font-awesome-icon :icon="['fas', 'edit']" />
							Edit
						</button>
						<button class="btn btn-sm btn-outline-info" @click="fetchMetadata(selectedVideo)">
							<font-awesome-icon :icon="['fas', 'download']" />
							Fetch Metadata
						</button>
						<button class="btn btn-sm btn-outline-danger" @click="deleteVideo(selectedVideo)">
							<font-awesome-icon :icon="['fas', 'trash']" />
							Delete
						</button>
					</div>
				</div>

				<!-- Performers -->
				<div v-if="selectedVideo.performers && selectedVideo.performers.length > 0" class="detail-section">
					<h3>Performers</h3>
					<div class="performers-list">
						<div v-for="performer in selectedVideo.performers" :key="performer.id" class="performer-item" @click="openPerformer(performer)">
							<img v-if="performer.image_path" :src="getAssetURL(performer.image_path)" :alt="performer.name" />
							<div v-else class="performer-placeholder">
								<font-awesome-icon :icon="['fas', 'user']" />
							</div>
							<span>{{ performer.name }}</span>
						</div>
					</div>
				</div>

				<!-- Studio -->
				<div v-if="selectedVideo.studios && selectedVideo.studios.length > 0" class="detail-section">
					<h3>Studio</h3>
					<div class="studio-item" @click="openStudio(selectedVideo.studios[0])">
						<font-awesome-icon :icon="['fas', 'building']" />
						{{ selectedVideo.studios[0].name }}
					</div>
				</div>

				<!-- Tags -->
				<div class="detail-section">
					<h3>Tags</h3>
					<div class="tags-container">
						<span v-for="tag in selectedVideo.tags" :key="tag.id" class="tag-chip" :style="{ backgroundColor: tag.color || '#6c757d' }">
							<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" />
							{{ tag.name }}
						</span>
						<button class="btn btn-sm btn-outline-primary" @click="openTagModal(selectedVideo)">
							<font-awesome-icon :icon="['fas', 'plus']" />
							Add Tag
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Context Menu -->
		<div v-if="contextMenu.show" class="context-menu" :style="{ top: contextMenu.y + 'px', left: contextMenu.x + 'px' }">
			<div class="context-menu-item" @click="playVideo(contextMenu.video)">
				<font-awesome-icon :icon="['fas', 'play']" />
				Play
			</div>
			<div class="context-menu-item" @click="editMetadata(contextMenu.video)">
				<font-awesome-icon :icon="['fas', 'edit']" />
				Edit Metadata
			</div>
			<div class="context-menu-item" @click="fetchMetadata(contextMenu.video)">
				<font-awesome-icon :icon="['fas', 'download']" />
				Fetch Metadata
			</div>
			<div class="context-menu-item" @click="openTagModal(contextMenu.video)">
				<font-awesome-icon :icon="['fas', 'tag']" />
				Add Tags
			</div>
			<div class="context-menu-item" @click="openInExplorer(contextMenu.video)">
				<font-awesome-icon :icon="['fas', 'folder-open']" />
				Open in Explorer
			</div>
			<div class="context-menu-item danger" @click="deleteVideo(contextMenu.video)">
				<font-awesome-icon :icon="['fas', 'trash']" />
				Delete
			</div>
		</div>

		<!-- Bulk Actions Modal -->
		<div v-if="showBulkActions" class="modal show d-block" tabindex="-1">
			<div class="modal-dialog modal-dialog-centered">
				<div class="modal-content text-bg-dark">
					<div class="modal-header">
						<h5 class="modal-title">Bulk Actions ({{ selectedVideos.length }} videos)</h5>
						<button type="button" class="btn-close" @click="showBulkActions = false"></button>
					</div>
					<div class="modal-body">
						<div class="d-grid gap-2">
							<button class="btn btn-outline-primary" @click="bulkAddTags">
								<font-awesome-icon :icon="['fas', 'tag']" />
								Add Tags
							</button>
							<button class="btn btn-outline-info" @click="bulkFetchMetadata">
								<font-awesome-icon :icon="['fas', 'download']" />
								Fetch Metadata
							</button>
							<button class="btn btn-outline-danger" @click="bulkDelete">
								<font-awesome-icon :icon="['fas', 'trash']" />
								Delete Selected
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>
		<div v-if="showBulkActions" class="modal-backdrop show"></div>
	</div>
</template>

<script>
import VideoCard from '@/components/VideoCard.vue'
import { videosAPI, performersAPI, studiosAPI, librariesAPI, getAssetURL } from '@/services/api'

export default {
	name: 'VideosPage',
	components: {
		VideoCard,
	},
	data() {
		return {
			videos: [],
			loading: false,
			viewMode: 'grid', // 'grid' or 'list'
			showFilters: true,
			searchQuery: '',
			searchTimeout: null,
			sortBy: 'created_at',
			sortOrder: 'desc',
			filters: {
				minDuration: null,
				maxDuration: null,
				resolution: '',
				performerId: null,
				studioId: null,
				hasPreview: null,
				missingMetadata: null,
				dateFrom: '',
				dateTo: '',
				minSize: null,
				maxSize: null,
			},
			currentPage: 1,
			pageSize: 60,
			totalVideos: 0,
			selectedVideos: [],
			selectedVideo: null,
			showBulkActions: false,
			contextMenu: {
				show: false,
				x: 0,
				y: 0,
				video: null,
			},
			performers: [],
			studios: [],
			libraries: [],
			selectedLibrary: '',
		}
	},
	computed: {
		totalPages() {
			return Math.ceil(this.totalVideos / this.pageSize)
		},
		visiblePages() {
			const pages = []
			const start = Math.max(1, this.currentPage - 2)
			const end = Math.min(this.totalPages, this.currentPage + 2)
			for (let i = start; i <= end; i++) {
				pages.push(i)
			}
			return pages
		},
	},
	mounted() {
		this.loadVideos()
		this.loadPerformers()
		this.loadStudios()
		this.loadLibraries()
		document.addEventListener('click', this.hideContextMenu)
		document.addEventListener('keydown', this.handleKeyPress)
	},
	beforeUnmount() {
		document.removeEventListener('click', this.hideContextMenu)
		document.removeEventListener('keydown', this.handleKeyPress)
	},
	methods: {
		getAssetURL,
		async loadVideos() {
			this.loading = true
			try {
				const params = {
					page: this.currentPage,
					limit: this.pageSize,
					sort_by: this.sortBy,
					sort_order: this.sortOrder,
					query: this.searchQuery || undefined,
					library_id: this.selectedLibrary || undefined,
					min_duration: this.filters.minDuration ? this.filters.minDuration * 60 : undefined,
					max_duration: this.filters.maxDuration ? this.filters.maxDuration * 60 : undefined,
					resolution: this.filters.resolution || undefined,
					performer_id: this.filters.performerId || undefined,
					studio_id: this.filters.studioId || undefined,
					has_preview: this.filters.hasPreview || undefined,
					missing_metadata: this.filters.missingMetadata || undefined,
					date_from: this.filters.dateFrom || undefined,
					date_to: this.filters.dateTo || undefined,
					min_size: this.filters.minSize ? this.filters.minSize * 1024 * 1024 : undefined,
					max_size: this.filters.maxSize ? this.filters.maxSize * 1024 * 1024 : undefined,
				}

				const response = await videosAPI.search(params)

				this.videos = response.data || []
				this.totalVideos = response.total || this.videos.length
			} catch (error) {
				console.error('Failed to load videos:', error)
				this.$toast.error('Failed to load videos')
			} finally {
				this.loading = false
			}
		},
		async loadPerformers() {
			try {
				const response = await performersAPI.getAll()
				this.performers = response.data || []
			} catch (error) {
				console.error('Failed to load performers:', error)
			}
		},
		async loadStudios() {
			try {
				const response = await studiosAPI.getAll()
				this.studios = response.data || []
			} catch (error) {
				console.error('Failed to load studios:', error)
			}
		},
		async loadLibraries() {
			try {
				const response = await librariesAPI.getAll()
				this.libraries = response.data || []
			} catch (error) {
				console.error('Failed to load libraries:', error)
			}
		},
		debounceSearch() {
			clearTimeout(this.searchTimeout)
			this.searchTimeout = setTimeout(() => {
				this.currentPage = 1
				this.loadVideos()
			}, 500)
		},
		toggleView() {
			this.viewMode = this.viewMode === 'grid' ? 'list' : 'grid'
		},
		toggleFilters() {
			this.showFilters = !this.showFilters
		},
		clearFilters() {
			this.filters = {
				minDuration: null,
				maxDuration: null,
				resolution: '',
				performerId: null,
				studioId: null,
				hasPreview: null,
				missingMetadata: null,
				dateFrom: '',
				dateTo: '',
				minSize: null,
				maxSize: null,
			}
			this.searchQuery = ''
			this.selectedLibrary = ''
			this.loadVideos()
		},
		goToPage(page) {
			if (page >= 1 && page <= this.totalPages) {
				this.currentPage = page
				this.loadVideos()
			}
		},
		async scanVideos() {
			try {
				// Use selected library or primary library
				let libraryId = this.selectedLibrary
				if (!libraryId) {
					// Get primary library
					const primaryLib = await librariesAPI.getPrimary()
					libraryId = primaryLib.data.id
				}

				await videosAPI.scan(libraryId)
				this.$toast.success('Video scan started. Check Activity Monitor for progress.')
				setTimeout(() => this.loadVideos(), 2000)
			} catch (error) {
				console.error('Failed to start scan:', error)
				this.$toast.error('Failed to start video scan: ' + (error.response?.data?.error || error.message))
			}
		},
		openVideoDetails(video) {
			this.selectedVideo = video
		},
		toggleVideoSelection(video) {
			const index = this.selectedVideos.indexOf(video.id)
			if (index > -1) {
				this.selectedVideos.splice(index, 1)
			} else {
				this.selectedVideos.push(video.id)
			}
		},
		toggleSelectAll(event) {
			if (event.target.checked) {
				this.selectedVideos = this.videos.map((v) => v.id)
			} else {
				this.selectedVideos = []
			}
		},
		showContextMenu({ video, x, y }) {
			this.contextMenu = {
				show: true,
				x,
				y,
				video,
			}
		},
		hideContextMenu() {
			this.contextMenu.show = false
		},
		playVideo(video) {
			console.log('Play video:', video)
			// Implement video player
		},
		editMetadata(video) {
			console.log('Edit metadata:', video)
			// Implement metadata editor
		},
		async fetchMetadata(video) {
			try {
				await videosAPI.fetchMetadata(video.id)
				this.$toast.success('Metadata fetch started')
				this.loadVideos()
			} catch (error) {
				console.error('Failed to fetch metadata:', error)
				this.$toast.error('Failed to fetch metadata')
			}
		},
		async deleteVideo(video) {
			if (!confirm(`Are you sure you want to delete "${video.title}"?`)) return

			try {
				await videosAPI.delete(video.id)
				this.$toast.success('Video deleted successfully')
				this.loadVideos()
				if (this.selectedVideo?.id === video.id) {
					this.selectedVideo = null
				}
			} catch (error) {
				console.error('Failed to delete video:', error)
				this.$toast.error('Failed to delete video')
			}
			this.hideContextMenu()
		},
		openTagModal(video) {
			console.log('Open tag modal:', video)
			// Implement tag selector
		},
		openPerformer(performer) {
			this.$router.push(`/performers/${performer.id}`)
		},
		openStudio(studio) {
			this.$router.push(`/studios/${studio.id}`)
		},
		bulkAddTags() {
			console.log('Bulk add tags to:', this.selectedVideos)
			// Implement bulk tag operation
		},
		async bulkFetchMetadata() {
			try {
				await videosAPI.bulk('fetch_metadata', this.selectedVideos)
				this.$toast.success('Bulk metadata fetch started')
				this.showBulkActions = false
				this.selectedVideos = []
			} catch (error) {
				console.error('Bulk fetch failed:', error)
				this.$toast.error('Bulk fetch failed')
			}
		},
		async bulkDelete() {
			if (!confirm(`Are you sure you want to delete ${this.selectedVideos.length} videos?`)) return

			try {
				await videosAPI.bulk('delete', this.selectedVideos)
				this.$toast.success('Videos deleted successfully')
				this.showBulkActions = false
				this.selectedVideos = []
				this.loadVideos()
			} catch (error) {
				console.error('Bulk delete failed:', error)
				this.$toast.error('Bulk delete failed')
			}
		},
		formatDuration(seconds) {
			if (!seconds) return 'N/A'
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
		getFilename(filepath) {
			if (!filepath) return ''
			return filepath.split(/[/\\]/).pop()
		},
		handleKeyPress(event) {
			// Skip if user is typing in an input field
			if (['INPUT', 'TEXTAREA', 'SELECT'].includes(event.target.tagName)) return

			switch (event.key.toLowerCase()) {
				case 'escape':
					if (this.selectedVideo) {
						this.selectedVideo = null
					} else if (this.showBulkActions) {
						this.showBulkActions = false
					}
					break
				case '/':
					event.preventDefault()
					document.querySelector('.top-bar input[type="text"]')?.focus()
					break
				case 'f':
					if (this.selectedVideo && !event.ctrlKey && !event.metaKey) {
						event.preventDefault()
						this.toggleFavorite(this.selectedVideo)
					}
					break
				case 't':
					if (this.selectedVideo && !event.ctrlKey && !event.metaKey) {
						event.preventDefault()
						this.openTagModal(this.selectedVideo)
					}
					break
				case 'm':
					if (this.selectedVideo && !event.ctrlKey && !event.metaKey) {
						event.preventDefault()
						this.editMetadata(this.selectedVideo)
					}
					break
				case 'delete':
					if (this.selectedVideo && !event.ctrlKey && !event.metaKey) {
						event.preventDefault()
						this.deleteVideo(this.selectedVideo)
					}
					break
			}
		},
		toggleFavorite(video) {
			// Placeholder for favorite toggle
			console.log('Toggle favorite:', video)
			this.$toast.success(`Favorite toggled for ${video.title}`)
		},
		openInExplorer(video) {
			if (!video.filepath) {
				this.$toast.error('File path not available')
				return
			}
			// Note: Opening file explorer requires backend support or Electron integration
			console.log('Open in explorer:', video.filepath)
			this.$toast.success('Opening file location...')
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/videos_page.css';
</style>
