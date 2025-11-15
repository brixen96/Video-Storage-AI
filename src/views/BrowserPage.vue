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
								<div class="filter-buttons d-flex gap-2 flex-wrap">
									<button class="btn btn-sm" :class="tab.filterType === 'all' ? 'btn-primary' : 'btn-outline-primary'" @click="setFilterType(tab, 'all')">
										<font-awesome-icon :icon="['fas', 'list']" class="me-1" />
										All
									</button>
									<button class="btn btn-sm" :class="tab.filterType === 'videos' ? 'btn-primary' : 'btn-outline-primary'" @click="setFilterType(tab, 'videos')">
										<font-awesome-icon :icon="['fas', 'video']" class="me-1" />
										Videos
									</button>
									<button class="btn btn-sm" :class="tab.filterType === 'folders' ? 'btn-primary' : 'btn-outline-primary'" @click="setFilterType(tab, 'folders')">
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
							v-for="item in filteredItems(tab)"
							:key="item.path"
							class="content-item"
							:class="{
								folder: item.type === 'folder',
								video: item.type === 'video',
								'not-interested': item.not_interested,
								'in-edit-list': item.in_edit_list,
							}"
							@click="handleItemClick(tab, item)"
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

					<!-- Empty State -->
					<div v-else-if="tab.libraryId" class="empty-state">
						<font-awesome-icon :icon="['fas', 'folder-open']" class="empty-icon" />
						<h3>No items found</h3>
						<p>This folder is empty</p>
					</div>

					<!-- No Library Selected -->
					<div v-else class="empty-state">
						<font-awesome-icon :icon="['fas', 'book-open']" class="empty-icon" />
						<h3>Select a Library</h3>
						<p>Choose a library from the dropdown to start browsing</p>
					</div>
				</div>
			</div>
		</div>

		<!-- Video Player Modal -->
		<VideoPlayer :visible="playerVisible" :video="selectedVideo" :libraryId="selectedLibraryId" @close="closePlayer" />
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
		}
	},
	async mounted() {
		await this.loadLibraries()
		await this.initializeTabs()
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
				// Check if we should load from the entire library (when marking filters are active)
				if (this.tabs[tabIndex].showNotInterested || this.tabs[tabIndex].showEditList) {
					// Load all marked videos from the entire library
					const params = {
						library_id: this.tabs[tabIndex].libraryId,
						per_page: 1000, // Load a large number to show all marked videos
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
						this.tabs[tabIndex].items = videos.map((video) => ({
							name: video.title || video.file_path.split(/[\\/]/).pop(),
							path: video.file_path,
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
						}))
					} else {
						this.tabs[tabIndex].items = []
					}
				} else {
					// Normal browsing mode - load current folder
					// Always request metadata extraction and thumbnail generation
					const response = await librariesAPI.browse(this.tabs[tabIndex].libraryId, this.tabs[tabIndex].currentPath, true)

					// Map API response to tab items
					if (response.data && response.data.items) {
						this.tabs[tabIndex].items = response.data.items
					} else {
						console.log('No items in response')
						this.tabs[tabIndex].items = []
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
		playVideo(video) {
			const activeTab = this.tabs.find((t) => t.id === this.activeTabId)
			if (activeTab) {
				this.selectedVideo = video
				this.selectedLibraryId = activeTab.libraryId
				this.playerVisible = true
			}
		},
		closePlayer() {
			this.playerVisible = false
			this.selectedVideo = {}
			this.selectedLibraryId = null
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
			if (!tab.items) return []

			let filtered = [...tab.items]

			// Apply search filter
			if (tab.searchQuery && tab.searchQuery.trim()) {
				const query = tab.searchQuery.toLowerCase().trim()
				filtered = filtered.filter((item) => item.name.toLowerCase().includes(query))
			}

			// Only apply type filter if not in library-wide marking mode
			// (in library-wide mode, we only show videos anyway)
			if (!tab.showNotInterested && !tab.showEditList) {
				if (tab.filterType === 'videos') {
					filtered = filtered.filter((item) => item.type === 'video')
				} else if (tab.filterType === 'folders') {
					filtered = filtered.filter((item) => item.type === 'folder')
				}
			}

			// Note: No need to filter by marks here anymore since loadLibraryContent
			// already loads only marked videos when showNotInterested or showEditList is active

			return filtered
		},
		applyFilters() {
			// Filters are applied through the computed filteredItems method
			// This is just for triggering reactivity when search query changes
		},
		clearSearch(tab) {
			tab.searchQuery = ''
		},
		setFilterType(tab, type) {
			tab.filterType = type
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
	},
}
</script>

<style scoped>
@import '@/styles/pages/browser_page.css';
</style>
