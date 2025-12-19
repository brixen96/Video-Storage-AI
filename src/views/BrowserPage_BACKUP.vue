<template>
	<div class="browser-page">
		<!-- Tab Navigation -->
		<div class="browser-tabs-container text-light">
			<div class="tabs-header">
				<div class="tabs-list">
					<div v-for="tab in tabs" :key="tab.id" class="tab-item" :class="{ active: activeTabId === tab.id }" @click="setActiveTab(tab.id)">
						<font-awesome-icon :icon="['fas', 'folder']" class="me-2" />
						<span class="tab-title">{{ tab.title }}</span>
						<button v-if="tabs.length > 1" class="btn-close-tab" @click.stop="closeTab(tab.id)" :title="'Close tab'">
							<font-awesome-icon :icon="['fas', 'times']" />
						</button>
					</div>
				</div>
				<button class="btn btn-sm btn-outline-primary ms-3" @click="addNewTab">
					<font-awesome-icon :icon="['fas', 'plus']" class="me-1" />
					New Tab
				</button>
			</div>

			<!-- Tab Content -->
			<div class="tab-content-container">
				<div v-for="tab in tabs" :key="tab.id" v-show="activeTabId === tab.id" class="tab-content">
					<!-- Library Selector -->
					<div class="library-selector mb-3">
						<select v-model="tab.libraryId" @change="loadLibraryContent(tab)" class="form-select">
							<option :value="null">Select a library...</option>
							<option v-for="library in libraries" :key="library.id" :value="library.id">
								{{ library.name }}
								<span v-if="library.primary">(Primary)</span>
							</option>
						</select>
					</div>

					<!-- Breadcrumb Navigation with Back Button -->
					<div v-if="tab.libraryId" class="breadcrumb-nav mb-3 d-flex align-items-center gap-2">
						<button
							v-if="tab.pathSegments.length > 0 && !tab.showNotInterested && !tab.showEditList"
							class="btn btn-outline-primary btn-back"
							@click="goBack(tab)"
							title="Go back"
						>
							<font-awesome-icon :icon="['fas', 'arrow-left']" />
						</button>
						<nav aria-label="breadcrumb" class="flex-grow-1">
							<!-- Show library-wide indicator when filtering by marks -->
							<div v-if="tab.showNotInterested || tab.showEditList" class="alert alert-info mb-0 py-2 d-flex align-items-center gap-2">
								<font-awesome-icon :icon="['fas', 'info-circle']" />
								<span>
									Showing all
									<strong v-if="tab.showNotInterested">Not Interested</strong>
									<strong v-if="tab.showEditList">Edit List</strong>
									videos from the entire library
								</span>
							</div>
							<!-- Normal breadcrumb navigation -->
							<ol v-else class="breadcrumb mb-0">
								<li class="breadcrumb-item">
									<a href="#" @click.prevent="navigateToPath(tab, '')" class="breadcrumb-link">
										<font-awesome-icon :icon="['fas', 'home']" />
									</a>
								</li>
								<li v-for="(segment, index) in tab.pathSegments" :key="index" class="breadcrumb-item">
									<a href="#" @click.prevent="navigateToSegment(tab, index)" class="breadcrumb-link">
										{{ segment }}
									</a>
								</li>
							</ol>
						</nav>
					</div>

					<!-- Search and Filter Bar -->
					<div v-if="tab.libraryId && !tab.loading" class="search-filter-bar mb-3">
						<div class="row g-3">
							<div class="col-md-6">
								<div class="input-group">
									<span class="input-group-text bg-dark border-secondary">
										<font-awesome-icon :icon="['fas', 'search']" />
									</span>
									<input
										v-model="tab.searchQuery"
										type="text"
										class="form-control bg-dark text-white border-secondary"
										placeholder="Search files and folders..."
										@input="applyFilters(tab)"
									/>
									<button v-if="tab.searchQuery" class="btn btn-outline-secondary" @click="clearSearch(tab)">
										<font-awesome-icon :icon="['fas', 'times']" />
									</button>
								</div>
							</div>
							<div class="col-md-6">
								<div class="d-flex gap-2 flex-wrap align-items-center">
									<div class="filter-buttons d-flex gap-2 flex-wrap">
										<button class="btn btn-sm" :class="tab.filterType === 'all' ? 'btn-primary' : 'btn-outline-primary'" @click="setFilterType(tab, 'all')">
											<font-awesome-icon :icon="['fas', 'list']" class="me-1" />
											All
										</button>
										<button
											class="btn btn-sm"
											:class="tab.filterType === 'videos' ? 'btn-primary' : 'btn-outline-primary'"
											@click="setFilterType(tab, 'videos')"
										>
											<font-awesome-icon :icon="['fas', 'video']" class="me-1" />
											Videos
										</button>
										<button
											class="btn btn-sm"
											:class="tab.filterType === 'folders' ? 'btn-primary' : 'btn-outline-primary'"
											@click="setFilterType(tab, 'folders')"
										>
											<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
											Folders
										</button>
										<button
											class="btn btn-sm"
											:class="tab.showNotInterested ? 'btn-danger' : 'btn-outline-danger'"
											@click="toggleShowNotInterested(tab)"
											:title="tab.showNotInterested ? 'Hide Not Interested' : 'Show Not Interested Only'"
										>
											<font-awesome-icon :icon="['fas', 'times-circle']" class="me-1" />
											Not Interested
										</button>
										<button
											class="btn btn-sm"
											:class="tab.showEditList ? 'btn-success' : 'btn-outline-success'"
											@click="toggleShowEditList(tab)"
											:title="tab.showEditList ? 'Hide Edit List' : 'Show Edit List Only'"
										>
											<font-awesome-icon :icon="['fas', 'list']" class="me-1" />
											Edit List
										</button>
									</div>
									<div class="vr d-none d-md-block"></div>
									<div class="d-flex gap-2 align-items-center">
										<label class="text-white small mb-0">Sort:</label>
										<select v-model="tab.sortBy" class="form-select form-select-sm bg-dark text-white border-secondary" style="width: auto">
											<option value="name">Name</option>
											<option value="date">Date</option>
											<option value="size">Size</option>
											<option value="duration">Duration</option>
										</select>
										<button
											class="btn btn-sm btn-outline-secondary"
											@click="toggleSortOrder(tab)"
											:title="tab.sortOrder === 'asc' ? 'Ascending' : 'Descending'"
										>
											<font-awesome-icon :icon="['fas', tab.sortOrder === 'asc' ? 'arrow-up' : 'arrow-down']" />
										</button>
									</div>
									<div class="vr d-none d-md-block"></div>
									<button
										class="btn btn-sm btn-outline-primary"
										@click="refreshCurrentFolder(tab)"
										title="Refresh folder to see updated thumbnails"
									>
										<font-awesome-icon :icon="['fas', 'sync']" :class="{ 'fa-spin': tab.loading }" />
										<span class="ms-1 d-none d-lg-inline">Refresh</span>
									</button>
								</div>
							</div>
						</div>
					</div>

					<!-- Loading State -->
					<div v-if="tab.loading" class="text-center py-5">
						<div class="spinner-border text-primary" role="status">
							<span class="visually-hidden">Loading...</span>
						</div>
						<p class="mt-3">Loading content...</p>
					</div>

					<!-- Content Grid -->
					<div v-else-if="tab.libraryId && filteredItems(tab).length > 0" class="content-grid">
						<div
							v-for="item in paginatedItems(tab)"
							:key="item.path"
							class="content-item"
							:class="{
								folder: item.type === 'folder',
								video: item.type === 'video',
								'not-interested': item.not_interested,
								'in-edit-list': item.in_edit_list,
							}"
							@click="handleItemClick(tab, item)"
							@contextmenu.prevent="showContextMenu($event, tab, item)"
						>
							<div class="item-thumbnail">
								<font-awesome-icon v-if="item.type === 'folder'" :icon="['fas', 'folder']" class="folder-icon" />
								<img v-else-if="item.thumbnail" :src="getAssetURL(item.thumbnail)" :alt="item.name" class="video-thumbnail" />
								<div v-else class="video-placeholder">
									<font-awesome-icon :icon="['fas', 'video']" />
								</div>

								<!-- Video Action Buttons -->
								<div v-if="item.type === 'video'" class="item-actions" @click.stop>
									<button
										class="btn-action btn-not-interested"
										:class="{ active: item.not_interested }"
										@click="toggleNotInterested(tab, item)"
										:title="item.not_interested ? 'Remove from Not Interested' : 'Mark as Not Interested'"
									>
										<font-awesome-icon :icon="['fas', 'times-circle']" />
									</button>
									<button
										class="btn-action btn-edit-list"
										:class="{ active: item.in_edit_list }"
										@click="toggleEditList(tab, item)"
										:title="item.in_edit_list ? 'Remove from Edit List' : 'Add to Edit List'"
									>
										<font-awesome-icon :icon="['fas', 'list']" />
									</button>
								</div>
							</div>
							<div class="item-info">
								<div class="item-name" :title="item.name">
									{{ item.name }}
									<span v-if="item.not_interested" class="badge-not-interested ms-1">Not Interested</span>
									<span v-if="item.in_edit_list" class="badge-edit-list ms-1">Edit List</span>
								</div>
								<div class="item-meta">
									<span v-if="item.type === 'folder'">
										<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
										Folder
									</span>
									<div v-else-if="item.type === 'video'" class="video-meta-info">
										<span v-if="item.duration" class="meta-badge">
											<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
											{{ formatDuration(item.duration) }}
										</span>
										<span v-if="item.width && item.height" class="meta-badge"> {{ item.width }}x{{ item.height }} </span>
										<span v-if="item.frame_rate" class="meta-badge"> {{ Math.round(item.frame_rate) }} fps </span>
									</div>
								</div>
							</div>
						</div>
					</div>

					<!-- Pagination Controls -->
					<div v-if="tab.libraryId && totalPages(tab) > 1" class="pagination-controls">
						<div class="pagination-info">
							Showing {{ (tab.currentPage - 1) * tab.itemsPerPage + 1 }} - {{ Math.min(tab.currentPage * tab.itemsPerPage, tab.totalItems) }} of
							{{ tab.totalItems }} items
						</div>
						<div class="pagination-buttons">
							<button class="btn btn-sm btn-outline-primary" :disabled="tab.currentPage === 1" @click="changePage(tab, 1)">
								<font-awesome-icon :icon="['fas', 'chevron-left']" />
								<font-awesome-icon :icon="['fas', 'chevron-left']" />
							</button>
							<button class="btn btn-sm btn-outline-primary" :disabled="tab.currentPage === 1" @click="changePage(tab, tab.currentPage - 1)">
								<font-awesome-icon :icon="['fas', 'chevron-left']" />
							</button>
							<span class="pagination-current"> Page {{ tab.currentPage }} of {{ totalPages(tab) }} </span>
							<button class="btn btn-sm btn-outline-primary" :disabled="tab.currentPage === totalPages(tab)" @click="changePage(tab, tab.currentPage + 1)">
								<font-awesome-icon :icon="['fas', 'chevron-right']" />
							</button>
							<button class="btn btn-sm btn-outline-primary" :disabled="tab.currentPage === totalPages(tab)" @click="changePage(tab, totalPages(tab))">
								<font-awesome-icon :icon="['fas', 'chevron-right']" />
								<font-awesome-icon :icon="['fas', 'chevron-right']" />
							</button>
						</div>
						<div class="pagination-size">
							<select v-model.number="tab.itemsPerPage" class="form-select form-select-sm bg-dark text-white border-secondary" @change="tab.currentPage = 1">
								<option :value="50">50 per page</option>
								<option :value="100">100 per page</option>
								<option :value="200">200 per page</option>
								<option :value="500">500 per page</option>
							</select>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Context Menu -->
		<div v-if="contextMenu.visible" class="context-menu" :style="{ top: contextMenu.y + 'px', left: contextMenu.x + 'px' }" @click="closeContextMenu">
			<div v-if="contextMenu.item?.type === 'video'" class="context-menu-item" @click="playVideo(contextMenu.item)">
				<font-awesome-icon :icon="['fas', 'play']" class="me-2" />
				Play Video
			</div>
			<div v-if="contextMenu.item?.type === 'folder'" class="context-menu-item" @click="navigateToFolder(contextMenu.tab, contextMenu.item.path)">
				<font-awesome-icon :icon="['fas', 'folder-open']" class="me-2" />
				Open Folder
			</div>
			<div class="context-menu-divider"></div>
			<div v-if="contextMenu.item?.type === 'video'" class="context-menu-item" @click="toggleNotInterested(contextMenu.tab, contextMenu.item)">
				<font-awesome-icon :icon="['fas', 'times-circle']" class="me-2" />
				{{ contextMenu.item.not_interested ? 'Remove from Not Interested' : 'Mark as Not Interested' }}
			</div>
			<div v-if="contextMenu.item?.type === 'video'" class="context-menu-item" @click="toggleEditList(contextMenu.tab, contextMenu.item)">
				<font-awesome-icon :icon="['fas', 'list']" class="me-2" />
				{{ contextMenu.item.in_edit_list ? 'Remove from Edit List' : 'Add to Edit List' }}
			</div>
			<div class="context-menu-divider"></div>
			<div class="context-menu-item" @click="copyPathToClipboard(contextMenu.item)">
				<font-awesome-icon :icon="['fas', 'check']" class="me-2" />
				Copy Path
			</div>
		</div>

		<!-- Video Player Modal -->
		<VideoPlayer
			:visible="playerVisible"
			:video="selectedVideo"
			:libraryId="selectedLibraryId"
			:videoId="selectedVideoId"
			@close="closePlayer"
			@video-converted="handleVideoConverted"
		/>
	</div>
</template>

<script>
import { librariesAPI, videosAPI, getAssetURL } from '@/services/api'
import VideoPlayer from '@/components/VideoPlayer.vue'

export default {
	name: 'BrowserPage',
	components: {
		VideoPlayer,
	},
	data() {
		return {
			libraries: [],
			tabs: [],
			activeTabId: null,
			nextTabId: 1,
			playerVisible: false,
			selectedVideo: {},
			selectedLibraryId: null,
			selectedVideoId: null,
			searchDebounceTimers: {}, // Store debounce timers per tab
			contextMenu: {
				visible: false,
				x: 0,
				y: 0,
				item: null,
				tab: null,
			},
			// Split view state
			splitPanels: [{ id: 1, tabIds: [], width: 100 }], // Start with single panel
			nextPanelId: 2,
			draggingPanel: null,
			resizingPanel: null,
			// Drag and drop state
			draggedItem: null,
			draggedFromTab: null,
			dropTargetTab: null,
			// Zoom state
			zoomLevel: 1, // 0.5 to 2.0
		}
	},
	async mounted() {
		await this.loadLibraries()
		await this.initializeTabs()

		// Close context menu when clicking outside
		document.addEventListener('click', this.closeContextMenu)
		document.addEventListener('contextmenu', this.closeContextMenu)

		// Add keyboard shortcuts
		document.addEventListener('keydown', this.handleKeyPress)
	},
	beforeUnmount() {
		// Clean up event listeners
		document.removeEventListener('click', this.closeContextMenu)
		document.removeEventListener('contextmenu', this.closeContextMenu)
		document.removeEventListener('keydown', this.handleKeyPress)
	},
	methods: {
		async loadLibraries() {
			try {
				const response = await librariesAPI.getAll()
				this.libraries = response.data || []
			} catch (error) {
				console.error('Failed to load libraries:', error)
			}
		},
		async initializeTabs() {
			// Try to load primary library
			try {
				const response = await librariesAPI.getPrimary()
				const primaryLibrary = response.data
				if (primaryLibrary) {
					await this.addNewTab(primaryLibrary.id, primaryLibrary.name)
					return
				}
			} catch (error) {
				console.log('No primary library found')
			}

			// If no primary library, just open an empty tab
			await this.addNewTab()
		},
		async addNewTab(libraryId = null, libraryName = null) {
			const tab = {
				id: this.nextTabId++,
				title: libraryName || 'New Tab',
				libraryId: libraryId,
				currentPath: '',
				pathSegments: [],
				items: [],
				loading: false,
				searchQuery: '',
				filterType: 'all',
				showNotInterested: false,
				showEditList: false,
				sortBy: 'name', // name, date, size, duration
				sortOrder: 'asc', // asc, desc
				currentPage: 1,
				itemsPerPage: 100,
				totalItems: 0,
			}

			this.tabs.push(tab)
			this.activeTabId = tab.id

			// Load content if library is set - AWAIT this!
			if (libraryId) {
				await this.loadLibraryContent(tab)
			}
		},
		closeTab(tabId) {
			const index = this.tabs.findIndex((t) => t.id === tabId)
			if (index === -1) return

			this.tabs.splice(index, 1)

			// If we closed the active tab, switch to another
			if (this.activeTabId === tabId) {
				if (this.tabs.length > 0) {
					this.activeTabId = this.tabs[Math.max(0, index - 1)].id
				} else {
					// If no tabs left, create a new one
					this.addNewTab()
				}
			}
		},
		setActiveTab(tabId) {
			this.activeTabId = tabId
		},
		async loadLibraryContent(tab) {
			if (!tab.libraryId) return

			// Find the tab index in the array
			const tabIndex = this.tabs.findIndex((t) => t.id === tab.id)
			if (tabIndex === -1) return

			// Update tab title
			const library = this.libraries.find((l) => l.id === tab.libraryId)
			if (library) {
				this.tabs[tabIndex].title = library.name
			}

			// IMPORTANT: Update pathSegments based on currentPath
			// Handle both forward slashes and backslashes
			const currentPath = this.tabs[tabIndex].currentPath || ''
			this.tabs[tabIndex].pathSegments = currentPath
				.split(/[/\\]/) // Split on both / and \
				.filter((s) => s && s.trim())

			// Set loading state
			this.tabs[tabIndex].loading = true

			try {
				// Check if we should search the entire library (when search query is active or marking filters are active)
				const hasSearchQuery = this.tabs[tabIndex].searchQuery && this.tabs[tabIndex].searchQuery.trim()
				if (hasSearchQuery || this.tabs[tabIndex].showNotInterested || this.tabs[tabIndex].showEditList) {
					// Load videos from the entire library based on search/filters
					const params = {
						library_id: this.tabs[tabIndex].libraryId,
						per_page: 1000, // Load a large number
					}

					if (hasSearchQuery) {
						params.query = this.tabs[tabIndex].searchQuery.trim()
					}

					if (this.tabs[tabIndex].showNotInterested) {
						params.not_interested = true
					}

					if (this.tabs[tabIndex].showEditList) {
						params.in_edit_list = true
					}

					const response = await videosAPI.getAll(params)

					// Convert video objects to browse items format
					// Check if response.data is an array directly or wrapped in a data property
					const videos = Array.isArray(response.data) ? response.data : response.data?.data || []

					if (videos.length > 0) {
						// Get library path to calculate relative paths
						const library = this.libraries.find((l) => l.id === this.tabs[tabIndex].libraryId)
						const libraryPath = library?.path || ''

						this.tabs[tabIndex].items = videos.map((video) => {
							// Calculate relative path from library root
							let relativePath = video.file_path
							if (libraryPath && video.file_path.startsWith(libraryPath)) {
								relativePath = video.file_path.substring(libraryPath.length)
								// Remove leading slash or backslash
								relativePath = relativePath.replace(/^[/\\]+/, '')
							}

							return {
								name: video.title || video.file_path.split(/[\\/]/).pop(),
								path: relativePath, // Use relative path for streaming
								full_path: video.file_path,
								type: 'video',
								is_dir: false,
								size: video.file_size || 0,
								modified: video.updated_at || video.created_at,
								duration: video.duration,
								thumbnail: video.thumbnail_path,
								not_interested: video.not_interested,
								in_edit_list: video.in_edit_list,
								video_id: video.id,
								in_database: true,
							}
						})
					} else {
						this.tabs[tabIndex].items = []
					}
				} else {
					// Normal browsing mode - load current folder
					// Use smart metadata extraction: quick for small folders, skip for large folders
					const quickCheckResponse = await librariesAPI.browse(this.tabs[tabIndex].libraryId, this.tabs[tabIndex].currentPath, false)

					// Count video files
					const videoCount = quickCheckResponse.data?.items?.filter((item) => item.type === 'video').length || 0

					// Only extract metadata if folder has <= 50 videos
					const shouldExtractMetadata = videoCount <= 50

					if (shouldExtractMetadata && videoCount > 0) {
						// Re-fetch with metadata for small folders
						const response = await librariesAPI.browse(this.tabs[tabIndex].libraryId, this.tabs[tabIndex].currentPath, true)
						if (response.data && response.data.items) {
							this.tabs[tabIndex].items = response.data.items
						} else {
							this.tabs[tabIndex].items = quickCheckResponse.data?.items || []
						}
					} else {
						// Use quick response for large folders (no metadata)
						if (quickCheckResponse.data && quickCheckResponse.data.items) {
							this.tabs[tabIndex].items = quickCheckResponse.data.items

							// Start background thumbnail generation with progress tracking
							this.$nextTick(() => {
								this.startBackgroundThumbnailGeneration(this.tabs[tabIndex])
							})
						} else {
							console.log('No items in response')
							this.tabs[tabIndex].items = []
						}
					}
				}
			} catch (error) {
				console.error('Failed to load library content:', error)
				this.tabs[tabIndex].items = []
			} finally {
				this.tabs[tabIndex].loading = false
				this.$forceUpdate()
			}
		},
		async startBackgroundThumbnailGeneration(tab) {
			try {
				console.log(`Starting background thumbnail generation for folder: ${tab.currentPath}`)
				const response = await librariesAPI.generateThumbnails(tab.libraryId, tab.currentPath)

				// If response indicates no thumbnails needed, exit early
				if (response && response.message && response.message.includes('already exist')) {
					console.log('All thumbnails already exist')
					return
				}

				console.log('Background thumbnail generation initiated')
				console.log('Refresh the folder or revisit it to see generated thumbnails')
			} catch (error) {
				console.error('Failed to start background thumbnail generation:', error)
			}
		},
		async refreshCurrentFolder(tab) {
			console.log('Refreshing current folder...')
			await this.loadLibraryContent(tab)
			console.log('Folder refreshed')
		},
		handleItemClick(tab, item) {
			if (item.type === 'folder') {
				this.navigateToFolder(tab, item.path)
			} else if (item.type === 'video') {
				this.playVideo(item)
			}
		},
		navigateToFolder(tab, folderPath) {
			const tabIndex = this.tabs.findIndex((t) => t.id === tab.id)
			if (tabIndex === -1) {
				console.log('Tab not found!')
				return
			}

			// Update the path
			this.tabs[tabIndex].currentPath = folderPath

			// Reload content (which will update pathSegments)
			this.loadLibraryContent(this.tabs[tabIndex])
		},

		navigateToPath(tab, path) {
			const tabIndex = this.tabs.findIndex((t) => t.id === tab.id)
			if (tabIndex === -1) {
				return
			}

			// Update the path
			this.tabs[tabIndex].currentPath = path

			// Reload content (which will update pathSegments)
			this.loadLibraryContent(this.tabs[tabIndex])
		},

		navigateToSegment(tab, index) {
			// Build the path from segments
			const path = tab.pathSegments.slice(0, index + 1).join('/')

			this.navigateToPath(tab, path)
		},

		goBack(tab) {
			if (tab.pathSegments.length > 0) {
				// Go back one level
				const path = tab.pathSegments.slice(0, -1).join('/')
				this.navigateToPath(tab, path)
			} else {
				console.log('Already at root')
			}
		},
		async playVideo(video) {
			const activeTab = this.tabs.find((t) => t.id === this.activeTabId)
			if (activeTab) {
				this.selectedVideo = video
				this.selectedLibraryId = activeTab.libraryId

				// Try to find or create a video record for conversion support
				await this.ensureVideoRecord(video, activeTab.libraryId)

				this.playerVisible = true
			}
		},
		async ensureVideoRecord(browserItem, libraryId) {
			try {
				// Check if video already exists in database by file path
				const searchResponse = await videosAPI.search({
					query: browserItem.name,
					library_id: libraryId,
				})

				// Look for exact file path match
				// searchResponse.data is the videos array (from backend's gin.H{"data": videos})
				const existingVideo = searchResponse.data?.find((v) => v.file_path === browserItem.full_path)

				if (existingVideo) {
					this.selectedVideoId = existingVideo.id
					return
				}

				// Video doesn't exist, create it
				const newVideo = {
					library_id: libraryId,
					title: browserItem.name.replace(/\.[^/.]+$/, ''), // Remove extension
					file_path: browserItem.full_path,
					file_size: browserItem.size || 0,
					duration: browserItem.duration || 0,
					resolution: browserItem.width && browserItem.height ? `${browserItem.width}x${browserItem.height}` : '',
					fps: browserItem.frame_rate || 0,
				}

				console.log('Creating video record with data:', newVideo)
				console.log('Browser item data:', browserItem)

				// Verify library exists
				try {
					const library = await librariesAPI.getById(libraryId)
					console.log('Library exists:', library)
				} catch (libErr) {
					console.error('Library does not exist:', libErr)
					throw new Error(`Library ${libraryId} does not exist`)
				}

				const createResponse = await videosAPI.create(newVideo)
				if (createResponse.data?.id) {
					this.selectedVideoId = createResponse.data.id
					console.log('Created video record for conversion:', createResponse.data.id)
				}
			} catch (error) {
				console.error('Failed to ensure video record:', error)
				// Continue without videoId - conversion won't be available
				this.selectedVideoId = null
			}
		},
		closePlayer() {
			this.playerVisible = false
			this.selectedVideo = {}
			this.selectedVideoId = null
			this.selectedLibraryId = null
		},
		async handleVideoConverted(convertedVideo) {
			console.log('Video converted:', convertedVideo)
			// Optionally reload the current tab to show the new converted file
			const activeTab = this.tabs.find((t) => t.id === this.activeTabId)
			if (activeTab && activeTab.libraryId) {
				await this.loadLibraryContent(activeTab)
			}
		},
		async toggleNotInterested(tab, item) {
			item.not_interested = !item.not_interested

			// Persist to backend
			try {
				await videosAPI.updateVideoMarksByPath(item.full_path, {
					not_interested: item.not_interested,
				})
			} catch (error) {
				console.error('Failed to update not interested status:', error)
				// Revert on error
				item.not_interested = !item.not_interested
			}
		},
		async toggleEditList(tab, item) {
			item.in_edit_list = !item.in_edit_list

			// Persist to backend
			try {
				await videosAPI.updateVideoMarksByPath(item.full_path, {
					in_edit_list: item.in_edit_list,
				})
			} catch (error) {
				console.error('Failed to update edit list status:', error)
				// Revert on error
				item.in_edit_list = !item.in_edit_list
			}
		},
		filteredItems(tab) {
			let dateA = new Date()
			let dateB = new Date()

			if (!tab.items) return []

			let filtered = [...tab.items]

			// Note: Search filter is now handled server-side in loadLibraryContent
			// when tab.searchQuery is active, so no need to filter here

			// Only apply type filter if not in library-wide mode
			// (in library-wide mode like search/marking, we only show videos anyway)
			const hasSearchQuery = tab.searchQuery && tab.searchQuery.trim()
			if (!hasSearchQuery && !tab.showNotInterested && !tab.showEditList) {
				if (tab.filterType === 'videos') {
					filtered = filtered.filter((item) => item.type === 'video')
				} else if (tab.filterType === 'folders') {
					filtered = filtered.filter((item) => item.type === 'folder')
				}
			}

			// Apply sorting
			filtered.sort((a, b) => {
				let comparison = 0

				// Always sort folders first
				if (a.type === 'folder' && b.type !== 'folder') return -1
				if (a.type !== 'folder' && b.type === 'folder') return 1

				// Then apply the selected sort
				switch (tab.sortBy) {
					case 'name':
						comparison = (a.name || '').localeCompare(b.name || '')
						break
					case 'date':
						dateA = new Date(a.modified || 0).getTime()
						dateB = new Date(b.modified || 0).getTime()
						comparison = dateA - dateB
						break
					case 'size':
						comparison = (a.size || 0) - (b.size || 0)
						break
					case 'duration':
						comparison = (a.duration || 0) - (b.duration || 0)
						break
					default:
						comparison = 0
				}

				// Apply sort order
				return tab.sortOrder === 'asc' ? comparison : -comparison
			})

			// Update total items for pagination
			tab.totalItems = filtered.length

			return filtered
		},
		paginatedItems(tab) {
			const filtered = this.filteredItems(tab)
			const start = (tab.currentPage - 1) * tab.itemsPerPage
			const end = start + tab.itemsPerPage
			return filtered.slice(start, end)
		},
		totalPages(tab) {
			return Math.ceil(tab.totalItems / tab.itemsPerPage)
		},
		changePage(tab, page) {
			if (page < 1 || page > this.totalPages(tab)) return
			tab.currentPage = page
			// Scroll to top when changing pages
			window.scrollTo({ top: 0, behavior: 'smooth' })
		},
		applyFilters(tab) {
			// When search query changes, reload library content after a debounce delay
			if (tab && tab.id) {
				// Clear existing timer for this tab
				if (this.searchDebounceTimers[tab.id]) {
					clearTimeout(this.searchDebounceTimers[tab.id])
				}

				// Set new timer (500ms debounce)
				this.searchDebounceTimers[tab.id] = setTimeout(() => {
					this.loadLibraryContent(tab)
				}, 500)
			}
		},
		clearSearch(tab) {
			tab.searchQuery = ''
			// Reload content to show folder browsing again
			this.loadLibraryContent(tab)
		},
		setFilterType(tab, type) {
			tab.filterType = type
		},
		toggleSortOrder(tab) {
			tab.sortOrder = tab.sortOrder === 'asc' ? 'desc' : 'asc'
		},
		toggleShowNotInterested(tab) {
			tab.showNotInterested = !tab.showNotInterested
			// If enabling, disable edit list filter
			if (tab.showNotInterested) {
				tab.showEditList = false
			}
			// Reload content to show all marked videos from the library
			this.loadLibraryContent(tab)
		},
		toggleShowEditList(tab) {
			tab.showEditList = !tab.showEditList
			// If enabling, disable not interested filter
			if (tab.showEditList) {
				tab.showNotInterested = false
			}
			// Reload content to show all marked videos from the library
			this.loadLibraryContent(tab)
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
		getAssetURL(path) {
			return getAssetURL(path)
		},
		showContextMenu(event, tab, item) {
			this.contextMenu.visible = true
			this.contextMenu.x = event.clientX
			this.contextMenu.y = event.clientY
			this.contextMenu.item = item
			this.contextMenu.tab = tab
		},
		closeContextMenu() {
			this.contextMenu.visible = false
			this.contextMenu.item = null
			this.contextMenu.tab = null
		},
		async copyPathToClipboard(item) {
			if (!item) return
			const path = item.full_path || item.path
			try {
				await navigator.clipboard.writeText(path)
				console.log('Path copied to clipboard:', path)
			} catch (error) {
				console.error('Failed to copy path:', error)
			}
		},
		handleKeyPress(event) {
			// Don't handle if typing in input field
			if (event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA') {
				return
			}

			const activeTab = this.tabs.find((t) => t.id === this.activeTabId)
			if (!activeTab) return

			// Backspace or Alt+Left: Go back to parent folder
			if (event.key === 'Backspace' || (event.altKey && event.key === 'ArrowLeft')) {
				event.preventDefault()
				if (activeTab.pathSegments.length > 0) {
					const newSegments = activeTab.pathSegments.slice(0, -1)
					const newPath = newSegments.join('\\')
					this.navigateToPath(activeTab, newPath)
				}
			}

			// Ctrl+T: New tab
			if (event.ctrlKey && event.key === 't') {
				event.preventDefault()
				this.addNewTab()
			}

			// Ctrl+W: Close tab
			if (event.ctrlKey && event.key === 'w') {
				event.preventDefault()
				this.closeTab(this.activeTabId)
			}

			// Ctrl+Tab or Ctrl+PageDown: Next tab
			if (event.ctrlKey && (event.key === 'Tab' || event.key === 'PageDown')) {
				event.preventDefault()
				const currentIndex = this.tabs.findIndex((t) => t.id === this.activeTabId)
				const nextIndex = (currentIndex + 1) % this.tabs.length
				this.activeTabId = this.tabs[nextIndex].id
			}

			// Ctrl+Shift+Tab or Ctrl+PageUp: Previous tab
			if (event.ctrlKey && ((event.shiftKey && event.key === 'Tab') || event.key === 'PageUp')) {
				event.preventDefault()
				const currentIndex = this.tabs.findIndex((t) => t.id === this.activeTabId)
				const prevIndex = currentIndex === 0 ? this.tabs.length - 1 : currentIndex - 1
				this.activeTabId = this.tabs[prevIndex].id
			}

			// F5 or Ctrl+R: Refresh current tab
			if (event.key === 'F5' || (event.ctrlKey && event.key === 'r')) {
				event.preventDefault()
				this.loadLibraryContent(activeTab)
			}
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/browser_page.css';
</style>
