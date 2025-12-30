<template>
	<div class="videos-page">
		<!-- Top Bar -->
		<div class="vp-top-bar">
			<div class="container-fluid">
				<div class="row align-items-center g-3">
					<div class="col-md-3">
						<h1>
							<font-awesome-icon :icon="['fas', 'video']" />
							Videos
							<span class="vp-video-count">({{ totalVideos }})</span>
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
							<button class="btn btn-outline-primary" @click="refreshVideos" title="Refresh video list">
								<font-awesome-icon :icon="['fas', 'sync']" :class="{ 'fa-spin': loading }" />
								Refresh
							</button>
							<button class="btn btn-success" @click="scanVideos">
								<font-awesome-icon :icon="['fas', 'database']" />
								Scan
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>

		<div class="vp-page-content">
			<!-- Left Sidebar (Filters) -->
			<div v-if="showFilters" class="vp-filter-sidebar" :class="{ collapsed: !showFilters }">
				<div class="vp-filter-panel">
					<div class="vp-filter-header">
						<div class="vp-filter-title-group">
							<h3>
								<font-awesome-icon :icon="['fas', 'sliders-h']" />
								Filters & Sort
							</h3>
							<span v-if="activeFiltersCount > 0" class="vp-active-filters-badge">{{ activeFiltersCount }}</span>
						</div>
						<button class="vp-clear-filters-btn" @click="clearFilters" title="Clear all filters">
							<font-awesome-icon :icon="['fas', 'times-circle']" />
						</button>
					</div>

					<!-- Sort Section -->
					<div class="vp-filter-section">
						<div class="vp-section-header">
							<font-awesome-icon :icon="['fas', 'sort']" />
							<span>Sort</span>
						</div>
						<div class="vp-sort-controls">
							<select v-model="sortBy" class="form-select form-select-sm" @change="loadVideos">
								<option value="created_at">📅 Date Added</option>
								<option value="title">🔤 Title</option>
								<option value="duration">⏱️ Duration</option>
								<option value="play_count">👁️ Views</option>
							</select>
							<div class="vp-sort-order-toggle">
								<button :class="{ active: sortOrder === 'desc' }" @click=";(sortOrder = 'desc'), loadVideos()" title="Descending">
									<font-awesome-icon :icon="['fas', 'sort-amount-down']" />
								</button>
								<button :class="{ active: sortOrder === 'asc' }" @click=";(sortOrder = 'asc'), loadVideos()" title="Ascending">
									<font-awesome-icon :icon="['fas', 'sort-amount-up']" />
								</button>
							</div>
						</div>
					</div>

					<!-- Content Filters -->
					<div class="vp-filter-section">
						<div class="vp-section-header">
							<font-awesome-icon :icon="['fas', 'filter']" />
							<span>Content</span>
						</div>

						<div v-if="performers.length > 0" class="vp-filter-group">
							<label>
								<font-awesome-icon :icon="['fas', 'user']" />
								Performers
								<span v-if="filteredPerformers.length !== performers.length" class="vp-filter-note">({{ filteredPerformers.length }})</span>
							</label>
							<select v-model="filters.performerId" class="form-select form-select-sm" @change="loadVideos">
								<option :value="null">All Performers</option>
								<option v-for="performer in filteredPerformers" :key="performer.id" :value="performer.id">{{ performer.name }}</option>
							</select>
						</div>

						<div v-if="studios.length > 0" class="vp-filter-group">
							<label>
								<font-awesome-icon :icon="['fas', 'building']" />
								Studios
							</label>
							<select v-model="filters.studioId" class="form-select form-select-sm" @change="onStudioChange">
								<option :value="null">All Studios</option>
								<option v-for="studio in studios" :key="studio.id" :value="studio.id">{{ studio.name }}</option>
							</select>
						</div>

						<div v-if="filteredGroups.length > 0" class="vp-filter-group">
							<label>
								<font-awesome-icon :icon="['fas', 'layer-group']" />
								Groups
							</label>
							<select v-model="filters.groupId" class="form-select form-select-sm" @change="loadVideos">
								<option :value="null">All Groups</option>
								<option v-for="group in filteredGroups" :key="group.id" :value="group.id">{{ group.name }}</option>
							</select>
						</div>

						<div v-if="filteredTags.length > 0" class="vp-filter-group">
							<label>
								<font-awesome-icon :icon="['fas', 'tags']" />
								Tags
								<span v-if="filters.selectedTags.length > 0" class="vp-filter-value">{{ filters.selectedTags.length }} selected</span>
							</label>
							<div class="vp-tags-multiselect">
								<div
									v-for="tag in filteredTags"
									:key="tag.id"
									class="vp-tag-item"
									:class="{ selected: filters.selectedTags.includes(tag.id) }"
									@click="toggleTag(tag.id)"
								>
									<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" />
									<span>{{ tag.name }}</span>
								</div>
							</div>
						</div>

						<div class="vp-filter-group">
							<label>
								<font-awesome-icon :icon="['fas', 'list']" />
								Content Type
							</label>
							<select v-model="filters.contentType" class="form-select form-select-sm" @change="loadVideos">
								<option :value="null">All Types</option>
								<option value="regular">Regular</option>
								<option value="zoo">Zoo</option>
								<option value="3d">3D</option>
							</select>
						</div>

						<div class="vp-filter-group">
							<label>
								<font-awesome-icon :icon="['fas', 'calendar']" />
								Date Range
							</label>
							<input v-model="filters.dateFrom" type="date" class="form-control form-control-sm mb-2" placeholder="From" @change="loadVideos" />
							<input v-model="filters.dateTo" type="date" class="form-control form-control-sm" placeholder="To" @change="loadVideos" />
						</div>
					</div>

					<!-- Media Properties -->
					<div class="vp-filter-section">
						<div class="vp-section-header">
							<font-awesome-icon :icon="['fas', 'cog']" />
							<span>Media Properties</span>
						</div>

						<div class="vp-filter-group">
							<label>
								<font-awesome-icon :icon="['fas', 'clock']" />
								Duration
								<span v-if="filters.minDuration || filters.maxDuration" class="vp-filter-value">
									{{ filters.minDuration || 0 }}m - {{ filters.maxDuration || 180 }}m
								</span>
							</label>
							<div class="vp-range-slider">
								<input v-model.number="filters.minDuration" type="range" min="0" max="180" step="5" class="vp-slider vp-slider-min" @input="loadVideos" />
								<input v-model.number="filters.maxDuration" type="range" min="0" max="180" step="5" class="vp-slider vp-slider-max" @input="loadVideos" />
								<div class="vp-slider-track"></div>
							</div>
						</div>

						<div class="vp-filter-group">
							<label>
								<font-awesome-icon :icon="['fas', 'expand']" />
								Resolution
							</label>
							<select v-model="filters.resolution" class="form-select form-select-sm" @change="loadVideos">
								<option value="">All Resolutions</option>
								<option value="3840x2160">🎬 4K (2160p)</option>
								<option value="1920x1080">📺 1080p (Full HD)</option>
								<option value="1280x720">📱 720p (HD)</option>
							</select>
						</div>

						<div class="vp-filter-group">
							<label>
								<font-awesome-icon :icon="['fas', 'hdd']" />
								File Size
								<span v-if="filters.minSize || filters.maxSize" class="vp-filter-value"> {{ filters.minSize || 0 }}MB - {{ filters.maxSize || 10000 }}MB </span>
							</label>
							<div class="vp-range-slider">
								<input v-model.number="filters.minSize" type="range" min="0" max="10000" step="100" class="vp-slider vp-slider-min" @input="loadVideos" />
								<input v-model.number="filters.maxSize" type="range" min="0" max="10000" step="100" class="vp-slider vp-slider-max" @input="loadVideos" />
								<div class="vp-slider-track"></div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Main Content Area -->
			<div class="vp-main-content p-0 m-0" :class="{ 'full-width': !showFilters }">
				<div class="container-fluid px-0">
					<!-- Loading State -->
					<div v-if="loading" class="vp-loading-state">
						<div class="vp-loading-spinner">
							<div class="spinner"></div>
							<p>Loading your videos...</p>
						</div>
					</div>

					<!-- Empty State -->
					<div v-else-if="videos.length === 0" class="vp-empty-state">
						<div class="vp-empty-content">
							<div class="vp-empty-icon">
								<font-awesome-icon :icon="['fas', 'video']" />
							</div>
							<h3>No Videos Found</h3>
							<p>Start by scanning your libraries to discover videos</p>
							<button class="btn btn-success btn-lg" @click="scanVideos">
								<font-awesome-icon :icon="['fas', 'database']" />
								Scan for Videos
							</button>
						</div>
					</div>

					<!-- Grid View -->
					<div v-else-if="viewMode === 'grid'" class="vp-video-grid p-3">
						<VideoCard
							v-for="video in videos"
							:key="video.id"
							v-memo="[video.id, video.title, video.rating, selectedVideos.includes(video.id)]"
							:video="video"
							:is-selected="selectedVideos.includes(video.id)"
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
					<div v-else class="vp-video-list">
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
								<tr
									v-for="video in videos"
									:key="video.id"
									v-memo="[video.id, video.title, selectedVideos.includes(video.id)]"
									:class="{ selected: selectedVideos.includes(video.id) }"
								>
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
					<div v-if="totalPages > 1" class="vp-pagination-controls mt-4">
						<nav>
							<ul class="pagination justify-content-center">
								<li class="vp-page-item" :class="{ disabled: currentPage === 1 }">
									<a class="vp-page-link" @click="goToPage(currentPage - 1)">Previous</a>
								</li>
								<li v-for="page in visiblePages" :key="page" class="vp-page-item" :class="{ active: page === currentPage }">
									<a class="vp-page-link" @click="goToPage(page)">{{ page }}</a>
								</li>
								<li class="vp-page-item" :class="{ disabled: currentPage === totalPages }">
									<a class="vp-page-link" @click="goToPage(currentPage + 1)">Next</a>
								</li>
							</ul>
						</nav>
					</div>
				</div>
			</div>
		</div>

		<!-- Context Menu -->
		<div v-if="contextMenu.show" class="vp-context-menu" :style="{ top: contextMenu.y + 'px', left: contextMenu.x + 'px' }">
			<div class="vp-context-menu-item" @click="playVideo(contextMenu.video)">
				<font-awesome-icon :icon="['fas', 'play']" />
				Play
			</div>
			<div class="vp-context-menu-item" @click="editMetadata(contextMenu.video)">
				<font-awesome-icon :icon="['fas', 'edit']" />
				Edit Metadata
			</div>
			<div class="vp-context-menu-item" @click="fetchMetadata(contextMenu.video)">
				<font-awesome-icon :icon="['fas', 'download']" />
				Fetch Metadata
			</div>
			<div class="vp-context-menu-item" @click="openTagModal(contextMenu.video)">
				<font-awesome-icon :icon="['fas', 'tag']" />
				Add Tags
			</div>
			<div class="vp-context-menu-item" @click="openInExplorer(contextMenu.video)">
				<font-awesome-icon :icon="['fas', 'folder-open']" />
				Open in Explorer
			</div>
			<div class="vp-context-menu-item danger" @click="deleteVideo(contextMenu.video)">
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

		<!-- Video Player Modal -->
		<VideoPlayerModal :show="showVideoPlayer" :video="videoForModal" @close="onVideoPlayerClose" @edit-metadata="editMetadata" @open-explorer="openInExplorer" />

		<!-- Edit Metadata Modal -->
		<EditMetadataModal :show="showEditMetadata" :video="videoForModal" @close="onEditMetadataClose" @saved="onEditMetadataSaved" />

		<!-- Add Tags Modal -->
		<AddTagModal :show="showTagModal" :video="videoForModal" @close="onTagModalClose" @saved="onTagModalSaved" />
	</div>
</template>

<script>
import { defineAsyncComponent } from 'vue'
import VideoCard from '@/components/VideoCard.vue'
import { videosAPI, librariesAPI, getAssetURL } from '@/services/api'
import settingsService from '@/services/settingsService'

// Lazy load heavy modal components
const VideoPlayerModal = defineAsyncComponent(() => import('@/components/VideoPlayerModal.vue'))
const EditMetadataModal = defineAsyncComponent(() => import('@/components/EditMetadataModal.vue'))
const AddTagModal = defineAsyncComponent(() => import('@/components/AddTagModal.vue'))

export default {
	name: 'VideosPage',
	components: {
		VideoCard,
		VideoPlayerModal,
		EditMetadataModal,
		AddTagModal,
	},
	data() {
		const settings = settingsService.getSettings()
		return {
			videos: [],
			loading: false,
			viewMode: settings.defaultViewMode || 'grid', // 'grid' or 'list'
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
				groupId: null,
				tagId: null,
				selectedTags: [],
				contentType: null,
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
			showBulkActions: false,
			contextMenu: {
				show: false,
				x: 0,
				y: 0,
				video: null,
			},
			performers: [],
			studios: [],
			groups: [],
			tags: [],
			libraries: [],
			selectedLibrary: '',
			showVideoPlayer: false,
			showEditMetadata: false,
			showTagModal: false,
			videoForModal: null,
			previewingPerformer: null,
			performerPreviewTimeouts: {},
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
		activeFiltersCount() {
			let count = 0
			if (this.filters.minDuration) count++
			if (this.filters.maxDuration) count++
			if (this.filters.resolution) count++
			if (this.filters.performerId) count++
			if (this.filters.studioId) count++
			if (this.filters.groupId) count++
			if (this.filters.tagId) count++
			if (this.filters.selectedTags.length > 0) count += this.filters.selectedTags.length
			if (this.filters.contentType !== null) count++
			if (this.filters.hasPreview) count++
			if (this.filters.missingMetadata) count++
			if (this.filters.dateFrom) count++
			if (this.filters.dateTo) count++
			if (this.filters.minSize) count++
			if (this.filters.maxSize) count++
			if (this.searchQuery) count++
			if (this.selectedLibrary) count++
			return count
		},
		filteredPerformers() {
			if (this.filters.contentType === null) {
				return this.performers
			}
			return this.performers.filter(p => p.category === this.filters.contentType)
		},
		filteredStudios() {
			if (this.filters.contentType === null) {
				return this.studios
			}
			return this.studios.filter(s => s.category === this.filters.contentType)
		},
		filteredGroups() {
			let groups = this.groups
			if (this.filters.studioId) {
				groups = groups.filter(g => g.studio_id === this.filters.studioId)
			}
			if (this.filters.contentType) {
				groups = groups.filter(g => g.category === this.filters.contentType)
			}
			return groups
		},
		filteredTags() {
			if (this.filters.contentType === null) {
				return this.tags
			}
			return this.tags.filter(t => t.category === this.filters.contentType)
		},
	},
	mounted() {
		this.loadVideos()
		this.loadPerformers()
		this.loadStudios()
		this.loadGroups()
		this.loadTags()
		this.loadLibraries()
		document.addEventListener('click', this.hideContextMenu)
		document.addEventListener('keydown', this.handleKeyPress)
	},
	beforeUnmount() {
		document.removeEventListener('click', this.hideContextMenu)
		document.removeEventListener('keydown', this.handleKeyPress)
	},
	watch: {
		'filters.contentType'() {
			// Reset performer, studio, group, and tag filters when content type changes
			this.filters.performerId = null
			this.filters.studioId = null
			this.filters.groupId = null
			this.filters.selectedTags = []
		},
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
					group_id: this.filters.groupId || undefined,
					tag_id: this.filters.tagId || undefined,
					tag_ids: this.filters.selectedTags.length > 0 ? this.filters.selectedTags : undefined,
					category: this.filters.contentType || undefined,
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
		async refreshVideos() {
			console.log('Refreshing videos...')
			await this.loadVideos()
			this.$toast.success('Videos refreshed')
		},
		async loadPerformers() {
			try {
				this.performers = await this.$store.dispatch('fetchPerformers')
			} catch (error) {
				console.error('Failed to load performers:', error)
			}
		},
		async loadStudios() {
			try {
				this.studios = await this.$store.dispatch('fetchStudios')
			} catch (error) {
				console.error('Failed to load studios:', error)
			}
		},
		async loadGroups() {
			try {
				this.groups = await this.$store.dispatch('fetchGroups')
			} catch (error) {
				console.error('Failed to load groups:', error)
			}
		},
		async loadTags() {
			try {
				this.tags = await this.$store.dispatch('fetchTags', true)
			} catch (error) {
				console.error('Failed to load tags:', error)
			}
		},
		onStudioChange() {
			// Reset group filter when studio changes
			this.filters.groupId = null
			this.loadVideos()
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
		toggleTag(tagId) {
			const index = this.filters.selectedTags.indexOf(tagId)
			if (index > -1) {
				this.filters.selectedTags.splice(index, 1)
			} else {
				this.filters.selectedTags.push(tagId)
			}
			this.loadVideos()
		},
		clearFilters() {
			this.filters = {
				minDuration: null,
				maxDuration: null,
				resolution: '',
				performerId: null,
				studioId: null,
				groupId: null,
				tagId: null,
				selectedTags: [],
				contentType: null,
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
			this.videoForModal = video
			this.showVideoPlayer = true
		},
		editMetadata(video) {
			this.videoForModal = video
			this.showEditMetadata = true
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
			this.videoForModal = video
			this.showTagModal = true
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
		async openInExplorer(video) {
			if (!video.file_path) {
				this.$toast.error('File path not available')
				return
			}

			try {
				await videosAPI.openInExplorer(video.id)
				this.$toast.success('Opening file location...')
			} catch (error) {
				console.error('Failed to open in explorer:', error)
				this.$toast.error('Failed to open file location')
			}
		},
		onVideoPlayerClose() {
			this.showVideoPlayer = false
			this.videoForModal = null
		},
		onEditMetadataClose() {
			this.showEditMetadata = false
			this.videoForModal = null
		},
		onEditMetadataSaved() {
			this.loadVideos()
			this.showEditMetadata = false
			this.videoForModal = null
		},
		onTagModalClose() {
			this.showTagModal = false
			this.videoForModal = null
		},
		onTagModalSaved() {
			this.loadVideos()
			this.showTagModal = false
			this.videoForModal = null
		},
		startPerformerPreview(performer) {
			// Delay preview start by 300ms to avoid loading on quick hovers
			this.performerPreviewTimeouts[performer.id] = setTimeout(() => {
				if (performer.preview_path) {
					const videoRef = this.$refs[`performer-preview-${performer.id}`]
					if (videoRef && videoRef[0]) {
						this.previewingPerformer = performer.id
						videoRef[0].play().catch(() => {
							// Ignore play errors
						})
					}
				}
			}, 300)
		},
		stopPerformerPreview(performer) {
			// Clear the timeout if user moves away before preview starts
			if (this.performerPreviewTimeouts[performer.id]) {
				clearTimeout(this.performerPreviewTimeouts[performer.id])
				delete this.performerPreviewTimeouts[performer.id]
			}

			// Stop preview if playing
			if (this.previewingPerformer === performer.id) {
				const videoRef = this.$refs[`performer-preview-${performer.id}`]
				if (videoRef && videoRef[0]) {
					videoRef[0].pause()
					videoRef[0].currentTime = 0
				}
				this.previewingPerformer = null
			}
		},
		onPerformerPreviewLoaded(performer) {
			// Preview loaded successfully
			console.log(`Preview loaded for ${performer.name}`)
		},
		calculateAge(birthdate) {
			if (!birthdate) return ''
			const birth = new Date(birthdate)
			const today = new Date()
			let age = today.getFullYear() - birth.getFullYear()
			const monthDiff = today.getMonth() - birth.getMonth()
			if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < birth.getDate())) {
				age--
			}
			return `${age} years`
		},
		getPerformerPreviewUrl(performer) {
			if (!performer.preview_path) {
				return ''
			}
			// If already a full URL, return as-is
			if (performer.preview_path.startsWith('http://') || performer.preview_path.startsWith('https://')) {
				return performer.preview_path
			}
			// If path already starts with /assets/, just prepend the base URL
			if (performer.preview_path.startsWith('/assets/') || performer.preview_path.startsWith('assets/')) {
				const cleanPath = performer.preview_path.startsWith('/') ? performer.preview_path.slice(1) : performer.preview_path
				console.log(cleanPath)

				return `http://localhost:8080/${cleanPath}`
			}
			// Otherwise use getAssetURL for full path conversion
			return getAssetURL(performer.preview_path)
		},
		handlePreviewError(event) {
			const video = event.target
			console.error('Performer preview failed to load:', {
				src: video.src,
				error: video.error,
				networkState: video.networkState,
				readyState: video.readyState,
			})
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/videos_page.css';
</style>
