
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
						<button v-if="tab.pathSegments.length > 0" class="btn btn-outline-primary btn-back" @click="goBack(tab)" title="Go back">
							<font-awesome-icon :icon="['fas', 'arrow-left']" />
						</button>
						<nav aria-label="breadcrumb" class="flex-grow-1">
							<ol class="breadcrumb mb-0">
								<li class="breadcrumb-item">
									<a href="#" @click.prevent="navigateToPath(tab, '')" class="breadcrumb-link">
										<font-awesome-icon :icon="['fas', 'home']" />
									</a>
								</li>
								<li v-for="(segment, index) in tab.pathSegments" :key="index" class="breadcrumb-item">
									<a v-if="index < tab.pathSegments.length - 1" href="#" @click.prevent="navigateToSegment(tab, index)" class="breadcrumb-link">
										{{ segment }}
									</a>
									<span v-else class="breadcrumb-current">{{ segment }}</span>
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
import { librariesAPI, getAssetURL } from '@/services/api'
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
			this.tabs[tabIndex].pathSegments = this.tabs[tabIndex].currentPath
				.split('/')
				.filter((s) => s && s.trim())

			console.log('Loading content for path:', this.tabs[tabIndex].currentPath)
			console.log('Path segments:', this.tabs[tabIndex].pathSegments)

			// Set loading state
			this.tabs[tabIndex].loading = true

			try {
				// Always request metadata extraction and thumbnail generation
				const response = await librariesAPI.browse(
					this.tabs[tabIndex].libraryId, 
					this.tabs[tabIndex].currentPath, 
					true
				)

				console.log('Browse response:', response.data)

				// Map API response to tab items
				if (response.data && response.data.items) {
					this.tabs[tabIndex].items = response.data.items
				} else {
					console.log('No items in response')
					this.tabs[tabIndex].items = []
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
			console.log('Item clicked:', item)
			if (item.type === 'folder') {
				this.navigateToFolder(tab, item.path)
			} else if (item.type === 'video') {
				this.playVideo(item)
			}
		},
		navigateToFolder(tab, folderPath) {
			console.log('navigateToFolder called with path:', folderPath)
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
			console.log('navigateToPath called with path:', path)
			const tabIndex = this.tabs.findIndex((t) => t.id === tab.id)
			if (tabIndex === -1) {
				console.log('Tab not found!')
				return
			}

			// Update the path
			this.tabs[tabIndex].currentPath = path
			
			// Reload content (which will update pathSegments)
			this.loadLibraryContent(this.tabs[tabIndex])
		},

		navigateToSegment(tab, index) {
			console.log('navigateToSegment called with index:', index)
			console.log('Current pathSegments:', tab.pathSegments)
			
			// Build the path from segments
			const path = tab.pathSegments.slice(0, index + 1).join('/')
			console.log('Navigating to path:', path)
			
			this.navigateToPath(tab, path)
		},

		goBack(tab) {
			console.log('goBack called')
			console.log('Current path:', tab.currentPath)
			console.log('Current pathSegments:', tab.pathSegments)
			
			if (tab.pathSegments.length > 0) {
				// Go back one level
				const path = tab.pathSegments.slice(0, -1).join('/')
				console.log('Going back to path:', path)
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
		toggleNotInterested(tab, item) {
			item.not_interested = !item.not_interested
			// TODO: Persist to backend
			console.log('Toggle not interested:', item.path, item.not_interested)
		},
		toggleEditList(tab, item) {
			item.in_edit_list = !item.in_edit_list
			// TODO: Persist to backend
			console.log('Toggle edit list:', item.path, item.in_edit_list)
		},
		filteredItems(tab) {
			if (!tab.items) return []

			let filtered = [...tab.items]

			// Apply search filter
			if (tab.searchQuery && tab.searchQuery.trim()) {
				const query = tab.searchQuery.toLowerCase().trim()
				filtered = filtered.filter((item) => item.name.toLowerCase().includes(query))
			}

			// Apply type filter
			if (tab.filterType === 'videos') {
				filtered = filtered.filter((item) => item.type === 'video')
			} else if (tab.filterType === 'folders') {
				filtered = filtered.filter((item) => item.type === 'folder')
			}

			// Apply marking filters
			if (tab.showNotInterested) {
				filtered = filtered.filter((item) => item.not_interested)
			}

			if (tab.showEditList) {
				filtered = filtered.filter((item) => item.in_edit_list)
			}

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
		},
		toggleShowEditList(tab) {
			tab.showEditList = !tab.showEditList
			// If enabling, disable not interested filter
			if (tab.showEditList) {
				tab.showNotInterested = false
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
		getAssetURL(path) {
			return getAssetURL(path)
		},
	},
}
</script>

<style scoped>
.browser-page {
	min-height: 100vh;
}

.page-header {
	text-align: left;
}

.page-title {
	font-size: 2rem;
	font-weight: 700;
	background: linear-gradient(90deg, #00d9ff, #00a8cc);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
	background-clip: text;
	margin-bottom: 0.5rem;
}

.page-subtitle {
	color: rgba(255, 255, 255, 0.6);
	font-size: 1rem;
	margin: 0;
}

.browser-tabs-container {
	background: rgba(255, 255, 255, 0.05);
	border-radius: 1rem;
	overflow: hidden;
	backdrop-filter: blur(10px);
}

.tabs-header {
	display: flex;
	align-items: center;
	padding: 0.2rem;
	background: rgba(0, 0, 0, 0.2);
	border-bottom: 2px solid rgba(0, 217, 255, 0.2);
}

.tabs-list {
	display: flex;
	gap: 0.5rem;
	flex: 1;
	overflow-x: auto;
}

.tab-item {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	padding: 0.75rem 1rem;
	background: rgba(255, 255, 255, 0.05);
	border-radius: 0.5rem;
	cursor: pointer;
	transition: all 0.3s ease;
	white-space: nowrap;
	min-width: 150px;
	border: 2px solid transparent;
}

.tab-item:hover {
	background: rgba(255, 255, 255, 0.1);
	transform: translateY(-2px);
}

.tab-item.active {
	background: linear-gradient(135deg, #0f3460, #00d9ff);
	border-color: #00d9ff;
	box-shadow: 0 4px 15px rgba(0, 217, 255, 0.3);
}

.tab-title {
	flex: 1;
	overflow: hidden;
	text-overflow: ellipsis;
}

.btn-close-tab {
	background: none;
	border: none;
	color: rgba(255, 255, 255, 0.6);
	cursor: pointer;
	padding: 0.25rem;
	display: flex;
	align-items: center;
	justify-content: center;
	width: 20px;
	height: 20px;
	border-radius: 50%;
	transition: all 0.2s ease;
}

.btn-close-tab:hover {
	background: rgba(220, 53, 69, 0.2);
	color: #dc3545;
}

.tab-content-container {
	padding: 1.5rem;
}

.library-selector {
	max-width: 400px;
}

.library-selector .form-label {
	font-weight: 600;
	color: #00d9ff;
	margin-bottom: 0.5rem;
}

.search-filter-bar {
	background: rgba(0, 0, 0, 0.2);
	padding: 1rem;
	border-radius: 0.5rem;
	border: 1px solid rgba(0, 217, 255, 0.2);
}

.search-filter-bar .input-group-text {
	color: rgba(255, 255, 255, 0.8);
}

.search-filter-bar .form-control:focus {
	border-color: #00d9ff;
	box-shadow: 0 0 0 0.25rem rgba(0, 217, 255, 0.25);
	background: rgba(0, 0, 0, 0.3);
}

.filter-buttons {
	justify-content: flex-end;
}

.filter-buttons .btn {
	transition: all 0.3s ease;
}

.btn-back {
	min-width: 40px;
	height: 40px;
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 0.5rem;
	transition: all 0.3s ease;
}

.btn-back:hover {
	transform: translateX(-3px);
}

.breadcrumb-nav .breadcrumb {
	background: rgba(0, 0, 0, 0.2);
	padding: 0.75rem 1rem;
	border-radius: 0.5rem;
	margin-bottom: 0;
}

.breadcrumb-item {
	display: inline-flex;
	align-items: center;
}

.breadcrumb-item + .breadcrumb-item::before {
	content: '/';
	padding: 0 0.5rem;
	color: rgba(255, 255, 255, 0.4);
}

.breadcrumb-link {
	color: #00d9ff;
	text-decoration: none;
	transition: all 0.2s ease;
	padding: 0.25rem 0.5rem;
	border-radius: 0.25rem;
}

.breadcrumb-link:hover {
	color: #00a8cc;
	background: rgba(0, 217, 255, 0.1);
}

.breadcrumb-current {
	color: rgba(255, 255, 255, 0.8);
	padding: 0.25rem 0.5rem;
}

.content-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
	gap: 2rem;
}

.content-item {
	transform: scale(1.1);
	background: rgba(255, 255, 255, 0.05);
	border-radius: 0.75rem;
	overflow: hidden;
	cursor: pointer;
	transition: all 0.3s ease;
	border: 2px solid transparent;
}

.content-item:hover {
	border-color: #00d9ff;
}

.item-thumbnail {
	aspect-ratio: 16/9;
	background: rgba(0, 0, 0, 0.3);
	display: flex;
	align-items: center;
	justify-content: center;
	overflow: hidden;
	position: relative;
}

.item-actions {
	position: absolute;
	top: 0.5rem;
	right: 0.5rem;
	display: flex;
	gap: 0.5rem;
	opacity: 0;
	transition: opacity 0.3s ease;
}

.content-item:hover .item-actions {
	opacity: 1;
}

.btn-action {
	background: rgba(0, 0, 0, 0.7);
	border: none;
	color: rgba(255, 255, 255, 0.8);
	width: 32px;
	height: 32px;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	transition: all 0.3s ease;
	backdrop-filter: blur(5px);
}

.btn-action:hover {
	transform: scale(1.1);
	color: #fff;
}

.btn-not-interested:hover {
	background: rgba(220, 53, 69, 0.8);
	color: #fff;
}

.btn-not-interested.active {
	background: #dc3545;
	color: #fff;
}

.btn-edit-list:hover {
	background: rgba(40, 167, 69, 0.8);
	color: #fff;
}

.btn-edit-list.active {
	background: #28a745;
	color: #fff;
}

.content-item.not-interested {
	opacity: 0.5;
	filter: grayscale(0.8);
}

.content-item.in-edit-list {
	border-color: #28a745 !important;
}

.badge-not-interested {
	background: #dc3545;
	color: #fff;
	font-size: 0.7rem;
	padding: 0.2rem 0.5rem;
	border-radius: 0.3rem;
	font-weight: 600;
}

.badge-edit-list {
	background: #28a745;
	color: #fff;
	font-size: 0.7rem;
	padding: 0.2rem 0.5rem;
	border-radius: 0.3rem;
	font-weight: 600;
}

.folder-icon {
	font-size: 4rem;
	color: #00d9ff;
}

.video-thumbnail {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.video-placeholder {
	width: 100%;
	height: 100%;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 3rem;
	color: rgba(255, 255, 255, 0.3);
}

.item-info {
	padding: 1rem;
}

.item-name {
	font-weight: 600;
	color: #fff;
	margin-bottom: 0.5rem;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.item-meta {
	font-size: 0.875rem;
	color: rgba(255, 255, 255, 0.6);
}

.video-meta-info {
	display: flex;
	flex-wrap: wrap;
	gap: 0.5rem;
}

.meta-badge {
	background: rgba(0, 217, 255, 0.15);
	padding: 0.2rem 0.5rem;
	border-radius: 0.3rem;
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.8);
	border: 1px solid rgba(0, 217, 255, 0.3);
}

.empty-state {
	text-align: center;
	padding: 4rem 2rem;
}

.empty-icon {
	font-size: 4rem;
	color: rgba(0, 217, 255, 0.3);
	margin-bottom: 1rem;
}

.empty-state h3 {
	color: rgba(255, 255, 255, 0.8);
	margin-bottom: 0.5rem;
}

/* Scrollbar styling */
.tabs-list::-webkit-scrollbar {
	height: 6px;
}

.tabs-list::-webkit-scrollbar-track {
	background: rgba(0, 0, 0, 0.2);
	border-radius: 3px;
}

.tabs-list::-webkit-scrollbar-thumb {
	background: rgba(0, 217, 255, 0.3);
	border-radius: 3px;
}

.tabs-list::-webkit-scrollbar-thumb:hover {
	background: rgba(0, 217, 255, 0.5);
}
</style>
