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
							<option value="scraped">🌐 Scraped Videos</option>
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
						<div class="d-flex gap-2 justify-content-end flex-wrap align-items-center">
							<button class="btn btn-outline-secondary" @click="toggleView">
								<font-awesome-icon :icon="viewMode === 'grid' ? ['fas', 'list'] : ['fas', 'th']" />
								{{ viewMode === 'grid' ? 'List' : 'Grid' }}
							</button>
							<!-- Card size controls (grid mode only) -->
							<div v-if="viewMode === 'grid'" class="vp-card-size-controls">
								<label>Size</label>
								<input
									type="range"
									class="vp-card-size-slider"
									min="200"
									max="600"
									step="20"
									:value="cardMinWidth"
									@input="setCardSize(+$event.target.value)"
								/>
								<div class="vp-card-size-presets">
									<button class="vp-size-preset" :class="{ active: cardMinWidth === 220 }" @click="setCardSize(220)" title="Small">S</button>
									<button class="vp-size-preset" :class="{ active: cardMinWidth === 300 }" @click="setCardSize(300)" title="Medium">M</button>
									<button class="vp-size-preset" :class="{ active: cardMinWidth === 400 }" @click="setCardSize(400)" title="Large">L</button>
									<button class="vp-size-preset" :class="{ active: cardMinWidth === 520 }" @click="setCardSize(520)" title="XL">XL</button>
								</div>
							</div>
														<button class="btn btn-outline-secondary" @click="toggleFilters">
								<font-awesome-icon :icon="['fas', 'filter']" />
								Filters
							</button>
							<button v-if="selectedCount > 0" class="btn btn-primary" @click="showBulkActions = true">
								<font-awesome-icon :icon="['fas', 'tasks']" />
								Bulk ({{ selectedCount }})
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
			<VideoFilterSidebar
				v-if="showFilters"
				:performers="performers"
				:filtered-performers="filteredPerformers"
				:studios="studios"
				:filtered-studios="filteredStudios"
				:groups="groups"
				:filtered-groups="filteredGroups"
				:tags="tags"
				:filtered-tags="filteredTags"
				:filters="filters"
				:active-filters-count="activeFiltersCount"
				v-model:sort-by="sortBy"
				v-model:sort-order="sortOrder"
				@change="loadVideos"
				@clear="clearFilters"
			/>

			<!-- Main Content Area -->
			<div ref="mainContent" class="vp-main-content p-0 m-0" :class="{ 'full-width': !showFilters }">
				<div class="container-fluid px-0">
					<!-- Missing files banner -->
					<div v-if="filters.showMissing && totalVideos > 0 && !loading" class="vp-missing-banner">
						<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="text-danger me-2" />
						<span>Showing <strong>{{ totalVideos }}</strong> missing file{{ totalVideos !== 1 ? 's' : '' }} — these videos no longer exist on disk.</span>
						<button class="btn btn-sm btn-outline-danger ms-3" :disabled="missingCleanupLoading" @click="missingCleanupConfirm = true">
							<font-awesome-icon :icon="['fas', 'broom']" />
							{{ missingCleanupLoading ? 'Cleaning...' : 'Cleanup All' }}
						</button>
					</div>

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
					<div v-else-if="viewMode === 'grid'" class="vp-video-grid p-3" :style="{ '--card-min-width': cardMinWidth + 'px' }">
						<VideoCard
							v-for="video in videos"
							:key="video.id"
							v-memo="[video.id, video.title, video.rating, video.is_favorite, isSelected(video.id)]"
							:video="video"
							:is-selected="isSelected(video.id)"
							@toggle-select="(video) => toggleSelection(video.id)"
							@context-menu="showContextMenu"
							@play="playVideo"
							@add-tag="openTagModal"
							@edit-metadata="editMetadata"
							@open-performer="openPerformer"
							@open-studio="openStudio"
							@toggle-favorite="quickToggleFavorite"
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
									v-memo="[video.id, video.title, isSelected(video.id)]"
									:class="{ selected: isSelected(video.id) }"
								>
									<td @click.stop>
										<input type="checkbox" :checked="isSelected(video.id)" @change="toggleSelection(video.id)" />
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
					<BrowserPaginationControls
						:currentPage="currentPage"
						:itemsPerPage="pageSize"
						:totalItems="totalVideos"
						@update:currentPage="currentPage = $event; loadVideos(); $refs.mainContent.scrollTo({ top: 0, behavior: 'smooth' })"
						@update:itemsPerPage="pageSize = $event; currentPage = 1; loadVideos(); $refs.mainContent.scrollTo({ top: 0, behavior: 'smooth' })"
					/>
				</div>
			</div>
		</div>

		<!-- Context Menu -->
		<VideoContextMenu
			:show="contextMenu.show"
			:x="contextMenu.x"
			:y="contextMenu.y"
			:video="contextMenu.video"
			@action="handleContextMenuAction"
			@hide="hideContextMenu"
		/>

		<!-- Bulk Actions Modal -->
		<div v-if="showBulkActions" class="modal show d-block" tabindex="-1">
			<div class="modal-dialog modal-dialog-centered">
				<div class="modal-content text-bg-dark">
					<div class="modal-header">
						<h5 class="modal-title">Bulk Actions ({{ selectedCount }} videos)</h5>
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
							<button class="btn btn-outline-warning" @click="bulkAnalyzePath">
								<font-awesome-icon :icon="['fas', 'link']" />
								Analyze Path
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
		<AddTagModal :show="showTagModal" :video="videoForModal" :videos="bulkTagVideoIds" @close="onTagModalClose" @saved="onTagModalSaved" />

		<!-- Delete Confirmation Modal -->
		<DeleteConfirmationModal
			:visible="deleteModal.show"
			:title="deleteModal.isBulk ? 'Confirm Bulk Delete' : 'Confirm Delete'"
			message="Are you sure you want to delete"
			:itemName="deleteModal.isBulk ? `${selectedCount} videos` : deleteModal.video?.title"
			@confirm="deleteModal.isBulk ? confirmBulkDelete() : confirmDeleteVideo()"
			@cancel="deleteModal.show = false"
		/>

		<!-- Missing file cleanup confirm -->
		<div v-if="missingCleanupConfirm" class="modal show d-block" tabindex="-1">
			<div class="modal-dialog modal-dialog-centered">
				<div class="modal-content text-bg-dark border-danger">
					<div class="modal-header border-danger">
						<h5 class="modal-title text-danger">
							<font-awesome-icon :icon="['fas', 'exclamation-triangle']" />
							Cleanup Missing Videos
						</h5>
						<button type="button" class="btn-close btn-close-white" @click="missingCleanupConfirm = false"></button>
					</div>
					<div class="modal-body">
						<p>This will permanently delete all <strong>{{ totalVideos }} missing video records</strong> from the database.</p>
						<p class="text-muted" style="font-size:0.85em;">Only the database entry is removed — no files on disk are touched. This affects the currently shown set (library filter applies).</p>
					</div>
					<div class="modal-footer border-danger">
						<button class="btn btn-secondary" @click="missingCleanupConfirm = false">Cancel</button>
						<button class="btn btn-danger" @click="confirmCleanupMissing">
							<font-awesome-icon :icon="['fas', 'broom']" />
							Delete All Records
						</button>
					</div>
				</div>
			</div>
		</div>
		<div v-if="missingCleanupConfirm" class="modal-backdrop show"></div>

		<!-- Locate / Remap missing file modal -->
		<div v-if="missingModal.show" class="modal show d-block" tabindex="-1">
			<div class="modal-dialog modal-dialog-centered">
				<div class="modal-content text-bg-dark border-warning">
					<div class="modal-header border-warning">
						<h5 class="modal-title text-warning">
							<font-awesome-icon :icon="['fas', missingModal.mode === 'locate' ? 'search-location' : 'route']" />
							{{ missingModal.mode === 'locate' ? 'Locate File' : 'Remap Path' }}
						</h5>
						<button type="button" class="btn-close btn-close-white" @click="missingModal.show = false"></button>
					</div>
					<div class="modal-body">
						<p class="text-muted" style="font-size:0.85em;">{{ missingModal.video?.title }}</p>
						<p class="text-muted" style="font-size:0.75em; word-break:break-all;">Was at: {{ missingModal.video?.file_path }}</p>

						<!-- Locate mode -->
						<template v-if="missingModal.mode === 'locate'">
							<div v-if="missingModal.loading" class="text-center py-3">
								<div class="spinner-border spinner-border-sm text-warning"></div>
								<span class="ms-2 text-muted">Searching library folder...</span>
							</div>
							<div v-else-if="missingModal.candidates.length === 0" class="text-muted text-center py-2">
								No matching files found in the library folder.
							</div>
							<div v-else>
								<p class="text-warning" style="font-size:0.85em;">Found {{ missingModal.candidates.length }} candidate(s) — click to remap:</p>
								<div v-for="path in missingModal.candidates" :key="path" class="missing-candidate" @click="confirmRemap(missingModal.video, path)">
									<font-awesome-icon :icon="['fas', 'file-video']" class="text-muted me-2" />
									<span>{{ path }}</span>
								</div>
							</div>
						</template>

						<!-- Manual remap mode -->
						<template v-else>
							<label class="form-label text-muted" style="font-size:0.85em;">New file path (must exist on disk):</label>
							<input
								v-model="missingModal.remapInput"
								type="text"
								class="form-control bg-dark text-light border-warning"
								placeholder="e.g. D:\Videos\file.mp4"
								@keydown.enter="confirmRemap(missingModal.video, missingModal.remapInput)"
							/>
						</template>
					</div>
					<div class="modal-footer border-warning">
						<button class="btn btn-secondary" @click="missingModal.show = false">Cancel</button>
						<button
							v-if="missingModal.mode === 'remap'"
							class="btn btn-warning"
							:disabled="!missingModal.remapInput.trim()"
							@click="confirmRemap(missingModal.video, missingModal.remapInput)"
						>
							<font-awesome-icon :icon="['fas', 'route']" />
							Apply Remap
						</button>
					</div>
				</div>
			</div>
		</div>
		<div v-if="missingModal.show" class="modal-backdrop show"></div>

		<!-- Add to List Modal -->
		<AddToListModal
			:visible="addToListModal.visible"
			:video="addToListModal.video"
			:is-scraped="addToListModal.isScraped"
			@close="addToListModal.visible = false"
			@added="onAddedToList"
		/>

		<!-- Scraped Video Modal -->
		<ScrapedVideoModal
			:video="scrapedModalVideo"
			@close="scrapedModalVideo = null"
			@downloaded="scrapedModalVideo = null"
		/>
	</div>
</template>

<script>
import { defineAsyncComponent } from 'vue'
import VideoCard from '@/components/VideoCard.vue'
import VideoFilterSidebar from '@/components/VideoFilterSidebar.vue'
import VideoContextMenu from '@/components/VideoContextMenu.vue'
import { videosAPI, librariesAPI, scrapersAPI, jdownloaderAPI, getAssetURL } from '@/services/api'
import settingsService from '@/services/settingsService'
import { BrowserPaginationControls } from '@/components/browser'
import { DeleteConfirmationModal } from '@/components/shared'
import { useFormatters } from '@/composables/useFormatters'
import { useTableSelectionOptionsAPI } from '@/composables/useTableSelection'

// Lazy load heavy modal components
const VideoPlayerModal = defineAsyncComponent(() => import('@/components/VideoPlayerModal.vue'))
const EditMetadataModal = defineAsyncComponent(() => import('@/components/EditMetadataModal.vue'))
const AddTagModal = defineAsyncComponent(() => import('@/components/AddTagModal.vue'))
const AddToListModal = defineAsyncComponent(() => import('@/components/AddToListModal.vue'))
const ScrapedVideoModal = defineAsyncComponent(() => import('@/components/scraper/ScrapedVideoModal.vue'))

export default {
	name: 'VideosPage',
	components: {
		VideoCard,
		VideoFilterSidebar,
		VideoContextMenu,
		VideoPlayerModal,
		EditMetadataModal,
		AddTagModal,
		AddToListModal,
		ScrapedVideoModal,
		BrowserPaginationControls,
		DeleteConfirmationModal,
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
				showMissing: false,
				noPerformer: false,
				watched: null,
				dateFrom: '',
				dateTo: '',
				minSize: null,
				maxSize: null,
			},
			currentPage: 1,
			pageSize: 60,
			cardMinWidth: parseInt(localStorage.getItem('vp-card-size') || '300'),
			totalVideos: 0,
			...useTableSelectionOptionsAPI().data(),
			showBulkActions: false,
			contextMenu: {
				show: false,
				x: 0,
				y: 0,
				video: null,
			},
			addToListModal: {
				visible: false,
				video: null,
				isScraped: false,
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
			scrapedModalVideo: null,
			bulkTagVideoIds: [],
			previewingPerformer: null,
			missingModal: {
				show: false,
				video: null,
				candidates: [],
				loading: false,
				remapInput: '',
				mode: 'locate', // 'locate' | 'remap'
			},
			missingCleanupConfirm: false,
			missingCleanupLoading: false,
			deleteModal: {
				show: false,
				video: null,
				isBulk: false,
			},
			performerPreviewTimeouts: {},
		}
	},
	computed: {
		...useTableSelectionOptionsAPI().computed(function () {
			return this.videos
		}),
		totalPages() {
			return Math.ceil(this.totalVideos / this.pageSize)
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
			if (this.filters.showMissing) count++
			if (this.filters.noPerformer) count++
			if (this.filters.watched !== null) count++
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
			if (!this.filters.contentType) {
				return this.tags
			}
			// Show tags whose category matches the selected content type, or tags with no
			// category (they apply globally and should always be visible).
			return this.tags.filter(tag => !tag.category || tag.category === this.filters.contentType)
		},
	},
	created() {
		// Initialize formatters composable
		const formatters = useFormatters()
		this.formatDuration = formatters.formatDuration
		this.formatFileSize = formatters.formatFileSize
		this.formatDate = formatters.formatDate
	},
	mounted() {
		if (this.$route.query.no_performer === 'true') {
			this.filters.noPerformer = true
		}
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

		// Clean up search debounce timeout
		if (this.searchTimeout) {
			clearTimeout(this.searchTimeout)
		}

		// Clean up all performer preview timeouts
		if (this.performerPreviewTimeouts) {
			Object.values(this.performerPreviewTimeouts).forEach((id) => clearTimeout(id))
			this.performerPreviewTimeouts = {}
		}
	},
	watch: {
		'filters.contentType'() {
			// Reset performer/studio/group when content type changes (they are filtered by category).
			this.filters.performerId = null
			this.filters.studioId = null
			this.filters.groupId = null
			// Drop any selected tags that don't belong to the new content type.
			// filteredTags already reflects the new contentType at this point.
			if (this.filters.contentType && this.filters.selectedTags.length > 0) {
				const validIds = new Set(this.filteredTags.map(t => t.id))
				this.filters.selectedTags = this.filters.selectedTags.filter(id => validIds.has(id))
			}
		},
	},
	methods: {
		...useTableSelectionOptionsAPI().methods(function () {
			return this.videos
		}),
		getAssetURL,
		setCardSize(width) {
			this.cardMinWidth = width
			localStorage.setItem('vp-card-size', width)
		},
		async loadVideos() {
			this.loading = true
			try {
				if (this.selectedLibrary === 'scraped') {
					await this.loadScrapedVideos()
					return
				}

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
					content_type: this.filters.contentType || undefined,
					has_preview: this.filters.hasPreview || undefined,
					missing_metadata: this.filters.missingMetadata || undefined,
					show_missing: this.filters.showMissing || undefined,
					no_performer: this.filters.noPerformer || undefined,
					watched: this.filters.watched !== null ? this.filters.watched : undefined,
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
		async loadScrapedVideos() {
			try {
				// Map frontend sort fields to backend column names
				const sortFieldMap = {
					created_at: 'discovered_at',
					play_count: 'view_count',
					title: 'title',
					duration: 'duration',
					rating: 'rating',
				}

				const params = {
					page: this.currentPage,
					limit: this.pageSize,
					sort_by: sortFieldMap[this.sortBy] || 'discovered_at',
					sort_order: this.sortOrder,
					includeTabInfo: 'true',
				}
				if (this.searchQuery) params.search = this.searchQuery
				if (this.filters.contentType) params.content_type = this.filters.contentType
				if (this.filters.minDuration) params.duration_filter = 'long' // fallback; server has short/medium/long

				const res = await scrapersAPI.getVideos(params)
				if (!res.success) throw new Error('API error')

				const result = res.data
				this.totalVideos = result.total
				this.videos = (result.videos || []).map(v => this.mapScrapedVideoToVideo(v))
			} catch (error) {
				console.error('Failed to load scraped videos:', error)
				this.$toast.error('Failed to load scraped videos')
				this.videos = []
				this.totalVideos = 0
			} finally {
				this.loading = false
			}
		},
		mapScrapedVideoToVideo(scraped) {
			// Parse metadata if it's a string
			let metadata = {}
			if (scraped.metadata) {
				try {
					metadata = typeof scraped.metadata === 'string' ? JSON.parse(scraped.metadata) : scraped.metadata
				} catch (e) {
					metadata = {}
				}
			}

			// Map scraped video fields to match the regular video format
			return {
				id: scraped.id,
				title: scraped.title || 'Untitled',
				thumbnail_path: scraped.thumbnail_url || scraped.thumbnail_path,
				// video_preview_url is stored inside the metadata JSON, not as a top-level field
				preview_path: metadata.video_preview_url || null,
				thumbnail_frames: metadata.thumbnail_frames || [],
				duration: scraped.duration || 0,
				file_size: scraped.file_size || 0,
				resolution: scraped.resolution || '',
				play_count: scraped.view_count || 0,
				rating: scraped.rating || 0,
				created_at: scraped.discovered_at || scraped.created_at,
				// Mark as scraped video for special handling
				is_scraped: true,
				source_url: scraped.source_url || scraped.url,
				source: scraped.source || 'Unknown',
				source_tab_name: scraped.source_tab_name || null,
				content_type: scraped.content_type || null,
				uploader: scraped.uploader || scraped.author || '',
				// Empty relationships for scraped videos
				performers: [],
				studios: [],
				tags: scraped.tags || [],
				groups: [],
				// Flags
				is_favorite: false,
				is_pinned: false,
				not_interested: false,
				in_edit_list: false,
			}
		},
		sortScrapedVideos(videos) {
			const sorted = [...videos]
			const order = this.sortOrder === 'asc' ? 1 : -1

			switch (this.sortBy) {
				case 'title':
					sorted.sort((a, b) => order * (a.title || '').localeCompare(b.title || ''))
					break
				case 'duration':
					sorted.sort((a, b) => order * ((a.duration || 0) - (b.duration || 0)))
					break
				case 'play_count':
					sorted.sort((a, b) => order * ((a.play_count || 0) - (b.play_count || 0)))
					break
				case 'created_at':
				default:
					sorted.sort((a, b) => order * (new Date(a.created_at || 0) - new Date(b.created_at || 0)))
					break
			}

			return sorted
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
				showMissing: false,
				noPerformer: false,
				watched: null,
				dateFrom: '',
				dateTo: '',
				minSize: null,
				maxSize: null,
			}
			this.searchQuery = ''
			this.selectedLibrary = ''
			this.loadVideos()
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
		async rateVideo(video, star) {
			if (!video) return
			const newRating = star === video.rating ? 0 : star
			this.hideContextMenu()
			try {
				await videosAPI.update(video.id, { rating: newRating })
				video.rating = newRating
				this.$toast.success(newRating ? `Rated ${newRating}/5` : 'Rating cleared')
			} catch (error) {
				console.error('Failed to rate video:', error)
				this.$toast.error('Failed to update rating')
			}
		},

		async quickToggleFavorite(video) {
			if (!video) return
			const newVal = !video.is_favorite
			try {
				await videosAPI.update(video.id, { is_favorite: newVal })
				video.is_favorite = newVal
			} catch (error) {
				console.error('Failed to toggle favorite:', error)
				this.$toast.error('Failed to update favorite')
			}
		},

		openAddToListModal(video) {
			this.addToListModal.video = video
			this.addToListModal.isScraped = video.is_scraped || false
			this.addToListModal.visible = true
			this.hideContextMenu()
		},
		onAddedToList(listIds) {
			this.$toast.success(`Added to ${listIds.length} list${listIds.length > 1 ? 's' : ''}`)
		},
		playVideo(video) {
			// For scraped videos, show the ScrapedVideoModal
			if (video.is_scraped) {
				this.scrapedModalVideo = video
				this.hideContextMenu()
				return
			}
			this.videoForModal = video
			this.showVideoPlayer = true
		},
		openLinkedVideo(video) {
			this.hideContextMenu()
			if (video.linked_video_id) {
				const route = this.$router.resolve(`/watch/${video.linked_video_id}`)
				window.open(route.href, '_blank')
			}
		},
		async openLocateModal(video) {
			this.hideContextMenu()
			this.missingModal = { show: true, video, candidates: [], loading: true, remapInput: '', mode: 'locate' }
			try {
				const res = await videosAPI.locateFile(video.id)
				this.missingModal.candidates = (res.data || res).candidates || []
			} catch {
				this.missingModal.candidates = []
			} finally {
				this.missingModal.loading = false
			}
		},
		openRemapModal(video) {
			this.hideContextMenu()
			this.missingModal = { show: true, video, candidates: [], loading: false, remapInput: '', mode: 'remap' }
		},
		async confirmRemap(video, newPath) {
			if (!newPath?.trim()) return
			try {
				await videosAPI.remapPath(video.id, newPath.trim())
				this.$toast.success('Video path updated')
				this.missingModal.show = false
				this.loadVideos()
			} catch (e) {
				this.$toast.error(e?.response?.data?.message || 'Remap failed')
			}
		},
		async confirmCleanupMissing() {
			this.missingCleanupConfirm = false
			this.missingCleanupLoading = true
			try {
				const libraryId = this.selectedLibrary && this.selectedLibrary !== 'scraped' ? this.selectedLibrary : 0
				const res = await videosAPI.cleanupMissing(libraryId)
				const deleted = (res.data || res).deleted ?? 0
				this.$toast.success(`Deleted ${deleted} missing video records`)
				this.loadVideos()
			} catch (e) {
				this.$toast.error('Cleanup failed')
			} finally {
				this.missingCleanupLoading = false
			}
		},
		openScrapedInFirefox(video) {
			const url = video.source_url || video.url
			if (url) scrapersAPI.openURL(url).catch(() => window.open(url, '_blank'))
			this.hideContextMenu()
		},
		async downloadScrapedVideo(video) {
			const url = video.source_url || video.url
			if (!url) return
			this.hideContextMenu()
			const scraperFolder = video.source_tab_name || video.source || 'Scraper'
			try {
				const result = await jdownloaderAPI.sendLinks([{
					url,
					packageName: video.title || '',
					downloadPath: `/output/${scraperFolder}`,
					videoId: video.id ?? null,
				}])
				if (result?.successCount > 0) {
					this.$toast.success('Sent to JDownloader')
				} else {
					this.$toast.warning('JDownloader: no links queued')
				}
			} catch {
				this.$toast.error('Failed to send to JDownloader')
			}
		},
		copyVideoUrl(video) {
			if (video.source_url) {
				navigator.clipboard.writeText(video.source_url)
				this.$toast.success('URL copied to clipboard')
			}
			this.hideContextMenu()
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
		deleteVideo(video) {
			this.deleteModal.video = video
			this.deleteModal.isBulk = false
			this.deleteModal.show = true
			this.hideContextMenu()
		},
		async confirmDeleteVideo() {
			try {
				await videosAPI.delete(this.deleteModal.video.id)
				this.$toast.success('Video deleted successfully')
				this.loadVideos()
				if (this.selectedVideo?.id === this.deleteModal.video.id) {
					this.selectedVideo = null
				}
			} catch (error) {
				console.error('Failed to delete video:', error)
				this.$toast.error('Failed to delete video')
			}
			this.deleteModal.show = false
			this.deleteModal.video = null
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
			this.videoForModal = null
			this.bulkTagVideoIds = [...this.selectedItems]
			this.showTagModal = true
			this.showBulkActions = false
		},
		async bulkFetchMetadata() {
			try {
				await videosAPI.bulk('fetch_metadata', this.selectedItems)
				this.$toast.success('Bulk metadata fetch started')
				this.showBulkActions = false
				this.clearSelection()
			} catch (error) {
				console.error('Bulk fetch failed:', error)
				this.$toast.error('Bulk fetch failed')
			}
		},
		async bulkAnalyzePath() {
			try {
				await videosAPI.bulk('analyze_path', this.selectedItems)
				this.$toast.success(`Path analysis started for ${this.selectedItems.length} videos`)
				this.showBulkActions = false
				this.clearSelection()
			} catch (error) {
				console.error('Bulk analyze path failed:', error)
				this.$toast.error('Bulk path analysis failed')
			}
		},
		bulkDelete() {
			this.deleteModal.isBulk = true
			this.deleteModal.show = true
		},
		async confirmBulkDelete() {
			try {
				await videosAPI.bulk('delete', this.selectedItems)
				this.$toast.success('Videos deleted successfully')
				this.showBulkActions = false
				this.clearSelection()
				this.loadVideos()
			} catch (error) {
				console.error('Bulk delete failed:', error)
				this.$toast.error('Bulk delete failed')
			}
			this.deleteModal.show = false
			this.deleteModal.isBulk = false
		},
		// formatDuration and formatFileSize now provided by useFormatters composable
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
		async analyzePath(video) {
			this.hideContextMenu()
			const loadingId = this.$toast.loading('Analyzing path...', video?.title || '')
			try {
				const res = await videosAPI.analyzePath(video.id)
				const r = res.data || res
				this.$toast.removeToast(loadingId)
				if (r.performers_linked === 0 && r.tags_applied === 0 && r.studios_linked === 0) {
					this.$toast.info('No new links found', 'Path did not match any known performers, tags or studios', 5000)
				} else {
					this.$toast.success(
						'Path analysis complete',
						`Linked ${r.performers_linked} performer(s), ${r.tags_applied} tag(s), ${r.studios_linked} studio(s)`,
						6000
					)
					this.loadVideos()
				}
			} catch (error) {
				this.$toast.removeToast(loadingId)
				console.error('Path analysis failed:', error)
				this.$toast.error('Path analysis failed', error?.response?.data?.message || '', 5000)
			}
		},
		openInExplorer(video) {
			if (!video?.file_path) {
				this.$toast.error('File path not available')
				return
			}

			const filePath = video.file_path
			const lastSep = Math.max(filePath.lastIndexOf('/'), filePath.lastIndexOf('\\'))
			const dir = lastSep > 0 ? filePath.substring(0, lastSep) : filePath

			// execCommand works on HTTP/SMB origins where navigator.clipboard is blocked
			const execCommandCopy = () => {
				const el = document.createElement('textarea')
				el.value = dir
				el.style.cssText = 'position:fixed;left:-9999px;top:-9999px;opacity:0'
				document.body.appendChild(el)
				el.select()
				try { document.execCommand('copy') } catch (e) { /* ignore */ }
				document.body.removeChild(el)
			}

			if (navigator.clipboard?.writeText) {
				navigator.clipboard.writeText(dir).catch(execCommandCopy)
			} else {
				execCommandCopy()
			}

			this.$toast.info('Folder path copied to clipboard', dir, 8000)

			// Best-effort: try backend (works when running as a regular process, not a service)
			videosAPI.openInExplorer(video.id).catch(() => {})
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
			this.bulkTagVideoIds = []
		},
		onTagModalSaved() {
			this.loadVideos()
			this.showTagModal = false
			this.videoForModal = null
			this.bulkTagVideoIds = []
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
