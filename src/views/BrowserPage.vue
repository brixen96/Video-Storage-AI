<template>
	<div class="browser-page">
		<!-- Zoom and Panel Controls -->
		<div class="controls-bar mb-3 p-3 bg-dark rounded d-flex justify-content-between align-items-center">
			<div class="d-flex align-items-center gap-2">
				<label class="text-white-50 small me-2">Zoom:</label>
				<button class="btn btn-sm btn-outline-primary" @click="adjustZoom(-0.1)" :disabled="zoomLevel <= 0.5" title="Zoom Out">
					<font-awesome-icon :icon="['fas', 'search']" transform="shrink-4" />
				</button>
				<span class="zoom-level-text text-white px-2">{{ Math.round(zoomLevel * 100) }}%</span>
				<button class="btn btn-sm btn-outline-primary" @click="adjustZoom(0.1)" :disabled="zoomLevel >= 2.0" title="Zoom In">
					<font-awesome-icon :icon="['fas', 'search']" transform="grow-4" />
				</button>
				<button class="btn btn-sm btn-outline-secondary" @click="resetZoom" title="Reset Zoom">
					<font-awesome-icon :icon="['fas', 'sync']" />
				</button>
			</div>
			<button class="btn btn-sm btn-primary" @click="addNewPanel" title="Add Split Panel">
				<font-awesome-icon :icon="['fas', 'th']" class="me-1" />
				<span>Add Panel</span>
			</button>
		</div>

		<!-- Split Panels Container -->
		<div class="split-panels-wrapper d-flex flex-row gap-2">
			<div
				v-for="(panel, panelIndex) in splitPanels"
				:key="panel.id"
				class="split-panel"
				:style="{ flex: `0 0 ${panel.width}%` }"
			>
				<!-- Panel Container -->
				<div class="browser-tabs-container text-light">
					<!-- Tabs Header -->
					<div class="tabs-header">
						<div class="tabs-list" @drop="handleTabDrop($event, panel)" @dragover.prevent>
							<div
								v-for="tabId in panel.tabIds"
								:key="tabId"
								class="tab-item"
								:class="{ active: panel.activeTabId === tabId }"
								draggable="true"
								@dragstart="handleTabDragStart($event, tabId, panel)"
								@dragend="handleTabDragEnd"
								@click="setActiveTabForPanel(panel, tabId)"
							>
								<font-awesome-icon :icon="['fas', 'folder']" class="me-2" />
								<span class="tab-title">{{ getTabById(tabId)?.title || 'Tab' }}</span>
								<button v-if="panel.tabIds.length > 1 || splitPanels.length > 1" class="btn-close-tab" @click.stop="closeTab(tabId, panel.id)">
									<font-awesome-icon :icon="['fas', 'times']" />
								</button>
							</div>
						</div>
						<div class="panel-actions d-flex gap-2">
							<button class="btn btn-sm btn-outline-primary" @click="addTabToPanel(panel)" title="New Tab">
								<font-awesome-icon :icon="['fas', 'plus']" />
							</button>
							<button v-if="splitPanels.length > 1" class="btn btn-sm btn-outline-danger" @click="removePanel(panel.id)" title="Close Panel">
								<font-awesome-icon :icon="['fas', 'times']" />
							</button>
						</div>
					</div>

					<!-- Tab Content -->
					<div class="tab-content-container">
						<div v-for="tabId in panel.tabIds" :key="tabId" v-show="panel.activeTabId === tabId" class="tab-content">
							<template v-if="getTabById(tabId)">
								<div :ref="'tab-' + tabId">
									<!-- Library Selector -->
									<div class="library-selector mb-3">
										<select v-model="getTabById(tabId).libraryId" @change="loadLibraryContent(getTabById(tabId))" class="form-select">
											<option :value="null">Select a library...</option>
											<option v-for="library in libraries" :key="library.id" :value="library.id">
												{{ library.name }}
												<span v-if="library.primary">(Primary)</span>
											</option>
										</select>
									</div>

									<!-- Breadcrumb Navigation -->
									<BrowserBreadcrumbNav
										v-if="getTabById(tabId).libraryId"
										:pathSegments="getTabById(tabId).pathSegments"
										:showNotInterested="getTabById(tabId).showNotInterested"
										:showEditList="getTabById(tabId).showEditList"
										@navigate-to="navigateToSegment(getTabById(tabId), $event)"
										@back="goBack(getTabById(tabId))"
									/>

									<!-- Search and Filter Bar -->
									<BrowserSearchFilterBar
										v-if="getTabById(tabId).libraryId"
										v-model:searchQuery="getTabById(tabId).searchQuery"
										v-model:filterType="getTabById(tabId).filterType"
										v-model:sortBy="getTabById(tabId).sortBy"
										v-model:sortOrder="getTabById(tabId).sortOrder"
										v-model:showNotInterested="getTabById(tabId).showNotInterested"
										v-model:showEditList="getTabById(tabId).showEditList"
										:isLoading="getTabById(tabId).loading"
										@update:searchQuery="applyFilters(getTabById(tabId))"
										@update:showNotInterested="toggleShowNotInterested(getTabById(tabId))"
										@update:showEditList="toggleShowEditList(getTabById(tabId))"
										@refresh="refreshCurrentFolder(getTabById(tabId))"
									/>

									<!-- Loading State -->
									<div v-if="getTabById(tabId).loading" class="text-center py-5">
										<div class="spinner-border text-primary" role="status">
											<span class="visually-hidden">Loading...</span>
										</div>
										<p class="mt-3">Loading content...</p>
									</div>

									<!-- Content Grid with Zoom -->
									<div
										v-else-if="getTabById(tabId).libraryId && filteredItems(getTabById(tabId)).length > 0"
										class="content-grid"
										:style="{ '--grid-scale': zoomLevel }"
										@drop="handleItemDrop($event, getTabById(tabId))"
										@dragover.prevent
									>
										<div
											v-for="(item, index) in paginatedItems(getTabById(tabId))"
											:key="item.path"
											class="content-item"
											:class="{
												folder: item.type === 'folder',
												video: item.type === 'video',
												'not-interested': item.not_interested,
												'in-edit-list': item.in_edit_list,
												'drag-over': dropTargetTab === tabId,
												'selected': isItemSelected(tabId, item.path),
											}"
											draggable="true"
											@dragstart="handleItemDragStart($event, item, getTabById(tabId))"
											@dragend="handleItemDragEnd"
											@click="handleItemSelect($event, getTabById(tabId), item, index)"
											@dblclick="handleItemDoubleClick($event, getTabById(tabId), item)"
											@contextmenu.prevent="showContextMenu($event, getTabById(tabId), item)"
											@mouseenter="handleItemHover($event, item)"
											@mouseleave="handleItemHoverEnd"
										>
											<div class="item-thumbnail">
												<font-awesome-icon v-if="item.type === 'folder'" :icon="['fas', 'folder']" class="folder-icon" />
												<img v-else-if="item.thumbnail" :src="getAssetURL(item.thumbnail)" :alt="item.name" class="video-thumbnail" />
												<div v-else class="video-placeholder">
													<font-awesome-icon :icon="['fas', 'video']" />
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

									<!-- Pagination -->
									<BrowserPaginationControls
										v-if="getTabById(tabId).libraryId"
										:currentPage="getTabById(tabId).currentPage"
										:itemsPerPage="getTabById(tabId).itemsPerPage"
										:totalItems="getTabById(tabId).totalItems"
										@update:currentPage="getTabById(tabId).currentPage = $event"
										@update:itemsPerPage="getTabById(tabId).itemsPerPage = $event; getTabById(tabId).currentPage = 1"
									/>
								</div>
							</template>
						</div>
					</div>
				</div>

				<!-- Panel Resize Handle -->
				<div v-if="panelIndex < splitPanels.length - 1" class="panel-resize-handle" @mousedown="startPanelResize($event, panelIndex)"></div>
			</div>
		</div>

		<!-- Context Menu -->
		<BrowserContextMenu
			:visible="contextMenu.visible"
			:x="contextMenu.x"
			:y="contextMenu.y"
			:item="contextMenu.item"
			@play="playVideo(contextMenu.item); closeContextMenu()"
			@open="navigateToFolder(contextMenu.tab, contextMenu.item.path); closeContextMenu()"
			@toggle-not-interested="toggleNotInterested(contextMenu.tab, contextMenu.item); closeContextMenu()"
			@toggle-edit-list="toggleEditList(contextMenu.tab, contextMenu.item); closeContextMenu()"
			@copy-path="copyPathToClipboard(contextMenu.item); closeContextMenu()"
		/>

		<!-- Video Preview on Hover -->
		<div
			v-if="previewVideo && currentPreviewFrame"
			class="video-preview-popup"
			:style="{
				top: previewPosition.y + 'px',
				left: previewPosition.x + 'px',
			}"
		>
			<img
				:src="currentPreviewFrame"
				class="preview-video"
				:alt="`${previewVideo.name} - Preview`"
				loading="lazy"
			/>
			<div class="preview-info">
				<div class="preview-name">{{ previewVideo.name }}</div>
				<div class="preview-meta" v-if="previewVideo.duration">
					<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
					{{ formatDuration(previewVideo.duration) }}
				</div>
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
import { librariesAPI, videosAPI, filesAPI, getAssetURL } from '@/services/api'
import VideoPlayer from '@/components/VideoPlayer.vue'
import { BrowserPaginationControls, BrowserSearchFilterBar, BrowserBreadcrumbNav, BrowserContextMenu } from '@/components/browser'
import { useFormatters } from '@/composables/useFormatters'

export default {
	name: 'BrowserPage',
	components: {
		VideoPlayer,
		BrowserPaginationControls,
		BrowserSearchFilterBar,
		BrowserBreadcrumbNav,
		BrowserContextMenu,
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
			searchDebounceTimers: {},
			contextMenu: {
				visible: false,
				x: 0,
				y: 0,
				item: null,
				tab: null,
			},
			// Split view state
			splitPanels: [{ id: 1, tabIds: [], width: 100, activeTabId: null }],
			nextPanelId: 2,
			draggingTab: null,
			draggingFromPanel: null,
			// Drag and drop state
			draggedItem: null,
			draggedFromTab: null,
			dropTargetTab: null,
			// Zoom state
			zoomLevel: 1, // 0.5 to 2.0
			// Panel resize state
			resizingPanelIndex: null,
			resizeStartX: 0,
			resizeStartWidths: [],
			// Selection state (per tab)
			selectedItems: {}, // { tabId: [item.path, ...] }
			lastSelectedIndex: {}, // { tabId: index } for shift+click
			// Video preview state
			previewVideo: null,
			previewPosition: { x: 0, y: 0 },
			previewTimeout: null,
			previewFrameIndex: 0,
			previewInterval: null,
			previewFrames: [],
		}
	},
	computed: {
		currentPreviewFrame() {
			if (this.previewFrames.length === 0) return null
			return this.previewFrames[this.previewFrameIndex]
		},
	},
	created() {
		// Initialize formatters composable
		const formatters = useFormatters()
		this.formatFileSize = formatters.formatFileSize
		this.formatDuration = formatters.formatDuration
	},
	async mounted() {
		await this.loadLibraries()
		await this.initializeTabs()

		// Close context menu when clicking outside
		document.addEventListener('click', this.closeContextMenu)
		document.addEventListener('contextmenu', this.closeContextMenu)

		// Add keyboard shortcuts
		document.addEventListener('keydown', this.handleKeyPress)

		// Panel resize listeners
		document.addEventListener('mousemove', this.handlePanelResize)
		document.addEventListener('mouseup', this.stopPanelResize)
	},
	beforeUnmount() {
		document.removeEventListener('click', this.closeContextMenu)
		document.removeEventListener('contextmenu', this.closeContextMenu)
		document.removeEventListener('keydown', this.handleKeyPress)
		document.removeEventListener('mousemove', this.handlePanelResize)
		document.removeEventListener('mouseup', this.stopPanelResize)

		// Clean up preview timeout and interval
		if (this.previewTimeout) {
			clearTimeout(this.previewTimeout)
		}
		if (this.previewInterval) {
			clearInterval(this.previewInterval)
		}
	},
	methods: {
		// ==========================
		// Library and Tab Management
		// ==========================
		async loadLibraries() {
			try {
				const response = await librariesAPI.getAll()
				this.libraries = response.data || []
			} catch (error) {
				console.error('Failed to load libraries:', error)
			}
		},
		async initializeTabs() {
			try {
				const response = await librariesAPI.getPrimary()
				const primaryLibrary = response.data
				if (primaryLibrary) {
					await this.addTabToPanel(this.splitPanels[0], primaryLibrary.id, primaryLibrary.name)
					return
				}
			} catch (error) {
				console.log('No primary library found')
			}

			await this.addTabToPanel(this.splitPanels[0])
		},
		async addTabToPanel(panel, libraryId = null, libraryName = null) {
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
				sortBy: 'name',
				sortOrder: 'asc',
				currentPage: 1,
				itemsPerPage: 100,
				totalItems: 0,
			}

			this.tabs.push(tab)
			panel.tabIds.push(tab.id)
			panel.activeTabId = tab.id
			this.activeTabId = tab.id

			if (libraryId) {
				await this.loadLibraryContent(tab)
			}
		},
		closeTab(tabId, panelId) {
			const panel = this.splitPanels.find((p) => p.id === panelId)
			if (!panel) return

			const tabIndex = panel.tabIds.indexOf(tabId)
			if (tabIndex === -1) return

			panel.tabIds.splice(tabIndex, 1)
			this.tabs = this.tabs.filter((t) => t.id !== tabId)

			// Switch to another tab in this panel
			if (panel.activeTabId === tabId) {
				if (panel.tabIds.length > 0) {
					panel.activeTabId = panel.tabIds[Math.max(0, tabIndex - 1)]
					this.activeTabId = panel.activeTabId
				} else {
					// No tabs left in this panel, create a new one
					this.addTabToPanel(panel)
				}
			}
		},
		setActiveTab(tabId) {
			this.activeTabId = tabId
		},
		setActiveTabForPanel(panel, tabId) {
			panel.activeTabId = tabId
			this.activeTabId = tabId
		},
		getTabById(tabId) {
			return this.tabs.find((t) => t.id === tabId)
		},

		// ==========================
		// Split Panel Management
		// ==========================
		addNewPanel() {
			if (this.splitPanels.length >= 4) {
				this.$toast.warning('Maximum Panels', 'You can have up to 4 panels')
				return
			}

			const newWidth = 100 / (this.splitPanels.length + 1)
			this.splitPanels.forEach((p) => (p.width = newWidth))

			const newPanel = { id: this.nextPanelId++, tabIds: [], width: newWidth, activeTabId: null }
			this.splitPanels.push(newPanel)

			this.addTabToPanel(newPanel)
		},
		removePanel(panelId) {
			const panelIndex = this.splitPanels.findIndex((p) => p.id === panelId)
			if (panelIndex === -1) return

			const panel = this.splitPanels[panelIndex]

			// Remove all tabs in this panel
			panel.tabIds.forEach((tabId) => {
				this.tabs = this.tabs.filter((t) => t.id !== tabId)
			})

			// Remove the panel
			this.splitPanels.splice(panelIndex, 1)

			// Redistribute widths
			const newWidth = 100 / this.splitPanels.length
			this.splitPanels.forEach((p) => (p.width = newWidth))

			// Switch to first tab in first panel
			if (this.splitPanels.length > 0 && this.splitPanels[0].tabIds.length > 0) {
				this.activeTabId = this.splitPanels[0].tabIds[0]
			}
		},

		// ==========================
		// Panel Resizing
		// ==========================
		startPanelResize(event, panelIndex) {
			this.resizingPanelIndex = panelIndex
			this.resizeStartX = event.clientX
			this.resizeStartWidths = this.splitPanels.map((p) => p.width)
			event.preventDefault()
		},
		handlePanelResize(event) {
			if (this.resizingPanelIndex === null) return

			const containerWidth = document.querySelector('.split-panels-wrapper').offsetWidth
			const deltaX = event.clientX - this.resizeStartX
			const deltaPercent = (deltaX / containerWidth) * 100

			const leftPanel = this.splitPanels[this.resizingPanelIndex]
			const rightPanel = this.splitPanels[this.resizingPanelIndex + 1]

			const newLeftWidth = this.resizeStartWidths[this.resizingPanelIndex] + deltaPercent
			const newRightWidth = this.resizeStartWidths[this.resizingPanelIndex + 1] - deltaPercent

			// Enforce minimum width of 20%
			if (newLeftWidth >= 20 && newRightWidth >= 20) {
				leftPanel.width = newLeftWidth
				rightPanel.width = newRightWidth
			}
		},
		stopPanelResize() {
			this.resizingPanelIndex = null
		},

		// ==========================
		// Tab Drag and Drop
		// ==========================
		handleTabDragStart(event, tabId, panel) {
			this.draggingTab = tabId
			this.draggingFromPanel = panel
			event.dataTransfer.effectAllowed = 'move'
			event.dataTransfer.setData('text/plain', tabId)
		},
		handleTabDragEnd() {
			this.draggingTab = null
			this.draggingFromPanel = null
		},
		handleTabDrop(event, targetPanel) {
			if (!this.draggingTab || !this.draggingFromPanel) return

			// Remove tab from source panel
			const sourceIndex = this.draggingFromPanel.tabIds.indexOf(this.draggingTab)
			if (sourceIndex !== -1) {
				this.draggingFromPanel.tabIds.splice(sourceIndex, 1)
			}

			// Add tab to target panel if not already there
			if (!targetPanel.tabIds.includes(this.draggingTab)) {
				targetPanel.tabIds.push(this.draggingTab)
			}

			this.activeTabId = this.draggingTab
			this.draggingTab = null
			this.draggingFromPanel = null
		},

		// ==========================
		// Item Drag and Drop (Move between libraries)
		// ==========================
		handleItemDragStart(event, item, tab) {
			this.draggedItem = item
			this.draggedFromTab = tab
			event.dataTransfer.effectAllowed = 'move'
			event.dataTransfer.setData('text/plain', JSON.stringify({ path: item.full_path, type: item.type }))
		},
		handleItemDragEnd() {
			this.draggedItem = null
			this.draggedFromTab = null
			this.dropTargetTab = null
		},
		async handleItemDrop(event, targetTab) {
			if (!this.draggedItem || !this.draggedFromTab) {
				console.log('handleItemDrop: No dragged item or source tab')
				return
			}

			// Can't drop on same library
			if (this.draggedFromTab.libraryId === targetTab.libraryId) {
				this.$toast.info('Same Library', 'Cannot move to the same library')
				this.handleItemDragEnd()
				return
			}

			// Save references before they get cleared
			const sourceTab = this.draggedFromTab
			const draggedItem = this.draggedItem

			const itemType = draggedItem.type === 'folder' ? 'Folder' : 'File'
			const itemName = draggedItem.name

			console.log('Moving item:', {
				type: itemType,
				name: itemName,
				from: sourceTab.title,
				to: targetTab.title,
				sourcePath: draggedItem.path,
				targetPath: targetTab.currentPath
			})

			// Show loading toast
			const loadingToastId = this.$toast.loading(
				`Moving ${itemType}`,
				`Moving "${itemName}" to ${targetTab.title}...`
			)

			try {
				// Call API to move file/folder across libraries
				await filesAPI.moveAcrossLibraries({
					source_library_id: sourceTab.libraryId,
					source_path: draggedItem.path,
					target_library_id: targetTab.libraryId,
					target_path: targetTab.currentPath || '', // Move to current folder in target library
				})

				console.log('Move API call successful')

				// Remove loading toast
				try {
					this.$toast.removeToast(loadingToastId)
					console.log('Loading toast removed')
				} catch (e) {
					console.error('Error removing toast:', e)
				}

				// Show success
				this.$toast.success(
					`${itemType} Moved`,
					`"${itemName}" moved successfully to ${targetTab.title}`
				)
				console.log('Success toast shown')

				// Small delay to ensure filesystem has updated
				console.log('Waiting for filesystem to update...')
				await new Promise(resolve => setTimeout(resolve, 200))

				console.log('Refreshing source tab:', sourceTab.title)
				console.log('Refreshing target tab:', targetTab.title)

				// Clear items to force UI update
				const sourceTabIndex = this.tabs.findIndex(t => t.id === sourceTab.id)
				const targetTabIndex = this.tabs.findIndex(t => t.id === targetTab.id)
				console.log('Source tab index:', sourceTabIndex, 'Target tab index:', targetTabIndex)

				if (sourceTabIndex !== -1) {
					this.tabs[sourceTabIndex].items = []
					console.log('Cleared source tab items')
				}
				if (targetTabIndex !== -1) {
					this.tabs[targetTabIndex].items = []
					console.log('Cleared target tab items')
				}

				// Wait a moment for UI to clear
				await this.$nextTick()
				console.log('UI tick complete, loading content...')

				// Refresh both tabs to show updated content
				await this.loadLibraryContent(sourceTab)
				console.log('Source tab refreshed, items count:', this.tabs[sourceTabIndex]?.items?.length || 0)

				await this.loadLibraryContent(targetTab)
				console.log('Target tab refreshed, items count:', this.tabs[targetTabIndex]?.items?.length || 0)

			} catch (error) {
				console.error('Error in handleItemDrop:', error)
				// Remove loading toast
				try {
					this.$toast.removeToast(loadingToastId)
				} catch (e) {
					console.error('Error removing toast in catch:', e)
				}

				// Show error
				this.$toast.error(
					'Move Failed',
					error.response?.data?.error || error.message || 'Failed to move item'
				)
			} finally {
				this.handleItemDragEnd()
			}
		},

		// ==========================
		// Zoom Controls
		// ==========================
		adjustZoom(delta) {
			this.zoomLevel = Math.max(0.5, Math.min(2.0, this.zoomLevel + delta))
		},
		resetZoom() {
			this.zoomLevel = 1
		},

		// ==========================
		// Content Loading (Keep existing methods)
		// ==========================
		async loadLibraryContent(tab) {
			if (!tab.libraryId) return

			const tabIndex = this.tabs.findIndex((t) => t.id === tab.id)
			if (tabIndex === -1) return

			const library = this.libraries.find((l) => l.id === tab.libraryId)
			if (library) {
				this.tabs[tabIndex].title = library.name
			}

			const currentPath = this.tabs[tabIndex].currentPath || ''
			this.tabs[tabIndex].pathSegments = currentPath.split(/[/\\]/).filter((s) => s && s.trim())

			this.tabs[tabIndex].loading = true

			try {
				const hasSearchQuery = this.tabs[tabIndex].searchQuery && this.tabs[tabIndex].searchQuery.trim()
				if (hasSearchQuery || this.tabs[tabIndex].showNotInterested || this.tabs[tabIndex].showEditList) {
					const params = {
						library_id: this.tabs[tabIndex].libraryId,
						per_page: 1000,
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
					const videos = Array.isArray(response.data) ? response.data : response.data?.data || []

					if (videos.length > 0) {
						const library = this.libraries.find((l) => l.id === this.tabs[tabIndex].libraryId)
						const libraryPath = library?.path || ''

						this.tabs[tabIndex].items = videos.map((video) => {
							let relativePath = video.file_path
							if (libraryPath && video.file_path.startsWith(libraryPath)) {
								relativePath = video.file_path.substring(libraryPath.length)
								relativePath = relativePath.replace(/^[/\\]+/, '')
							}

							return {
								name: video.title || video.file_path.split(/[\\/]/).pop(),
								path: relativePath,
								full_path: video.file_path,
								type: 'video',
								is_dir: false,
								size: video.file_size || 0,
								modified: video.updated_at || video.created_at,
								duration: video.duration,
								thumbnail: video.thumbnail_path,
								preview_path: video.preview_path,
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
					const quickCheckResponse = await librariesAPI.browse(this.tabs[tabIndex].libraryId, this.tabs[tabIndex].currentPath, false)
					const videoCount = quickCheckResponse.data?.items?.filter((item) => item.type === 'video').length || 0
					const shouldExtractMetadata = videoCount <= 50

					if (shouldExtractMetadata && videoCount > 0) {
						const response = await librariesAPI.browse(this.tabs[tabIndex].libraryId, this.tabs[tabIndex].currentPath, true)
						if (response.data && response.data.items) {
							this.tabs[tabIndex].items = response.data.items
						} else {
							this.tabs[tabIndex].items = quickCheckResponse.data?.items || []
						}
					} else {
						if (quickCheckResponse.data && quickCheckResponse.data.items) {
							this.tabs[tabIndex].items = quickCheckResponse.data.items

							this.$nextTick(() => {
								this.startBackgroundThumbnailGeneration(this.tabs[tabIndex])
							})
						} else {
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

				if (response && response.message && response.message.includes('already exist')) {
					console.log('All thumbnails already exist')
					return
				}

				console.log('Background thumbnail generation initiated')
			} catch (error) {
				console.error('Failed to start background thumbnail generation:', error)
			}
		},
		async refreshCurrentFolder(tab) {
			console.log('Refreshing current folder...')
			await this.loadLibraryContent(tab)
			console.log('Folder refreshed')
		},
		// ==========================
		// Selection Management
		// ==========================
		isItemSelected(tabId, itemPath) {
			return this.selectedItems[tabId]?.includes(itemPath) || false
		},
		handleItemSelect(event, tab, item, index) {
			const tabId = tab.id

			// Initialize selection array for this tab if it doesn't exist
			if (!this.selectedItems[tabId]) {
				this.selectedItems[tabId] = []
			}

			if (event.ctrlKey || event.metaKey) {
				// Ctrl+Click: Toggle selection
				const selectedIndex = this.selectedItems[tabId].indexOf(item.path)
				if (selectedIndex > -1) {
					this.selectedItems[tabId].splice(selectedIndex, 1)
				} else {
					this.selectedItems[tabId].push(item.path)
				}
				this.lastSelectedIndex[tabId] = index
			} else if (event.shiftKey && this.lastSelectedIndex[tabId] !== undefined) {
				// Shift+Click: Range selection
				const items = this.paginatedItems(tab)
				const start = Math.min(this.lastSelectedIndex[tabId], index)
				const end = Math.max(this.lastSelectedIndex[tabId], index)

				this.selectedItems[tabId] = []
				for (let i = start; i <= end; i++) {
					if (items[i]) {
						this.selectedItems[tabId].push(items[i].path)
					}
				}
			} else {
				// Regular click: Single selection
				this.selectedItems[tabId] = [item.path]
				this.lastSelectedIndex[tabId] = index
			}
		},
		handleItemDoubleClick(event, tab, item) {
			if (item.type === 'folder') {
				this.navigateToFolder(tab, item.path)
			} else if (item.type === 'video') {
				this.playVideo(item)
			}
		},
		clearSelection(tabId) {
			if (this.selectedItems[tabId]) {
				this.selectedItems[tabId] = []
			}
			delete this.lastSelectedIndex[tabId]
		},
		navigateToFolder(tab, folderPath) {
			const tabIndex = this.tabs.findIndex((t) => t.id === tab.id)
			if (tabIndex === -1) return

			this.tabs[tabIndex].currentPath = folderPath
			this.loadLibraryContent(this.tabs[tabIndex])
		},
		navigateToPath(tab, path) {
			const tabIndex = this.tabs.findIndex((t) => t.id === tab.id)
			if (tabIndex === -1) return

			this.tabs[tabIndex].currentPath = path
			this.loadLibraryContent(this.tabs[tabIndex])
		},
		navigateToSegment(tab, index) {
			const path = tab.pathSegments.slice(0, index + 1).join('/')
			this.navigateToPath(tab, path)
		},
		goBack(tab) {
			if (tab.pathSegments.length > 0) {
				const path = tab.pathSegments.slice(0, -1).join('/')
				this.navigateToPath(tab, path)
			}
		},
		async playVideo(video) {
			const activeTab = this.tabs.find((t) => t.id === this.activeTabId)
			if (activeTab) {
				this.selectedVideo = video
				this.selectedLibraryId = activeTab.libraryId

				await this.ensureVideoRecord(video, activeTab.libraryId)

				this.playerVisible = true
			}
		},
		async ensureVideoRecord(browserItem, libraryId) {
			try {
				const searchResponse = await videosAPI.search({
					query: browserItem.name,
					library_id: libraryId,
				})

				const existingVideo = searchResponse.data?.find((v) => v.file_path === browserItem.full_path)

				if (existingVideo) {
					this.selectedVideoId = existingVideo.id
					return
				}

				const newVideo = {
					library_id: libraryId,
					title: browserItem.name.replace(/\.[^/.]+$/, ''),
					file_path: browserItem.full_path,
					file_size: browserItem.size || 0,
					duration: browserItem.duration || 0,
					resolution: browserItem.width && browserItem.height ? `${browserItem.width}x${browserItem.height}` : '',
					fps: browserItem.frame_rate || 0,
				}

				const createResponse = await videosAPI.create(newVideo)
				if (createResponse.data?.id) {
					this.selectedVideoId = createResponse.data.id
				}
			} catch (error) {
				console.error('Failed to ensure video record:', error)
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
			const activeTab = this.tabs.find((t) => t.id === this.activeTabId)
			if (activeTab && activeTab.libraryId) {
				await this.loadLibraryContent(activeTab)
			}
		},
		async toggleNotInterested(tab, item) {
			item.not_interested = !item.not_interested

			try {
				await videosAPI.updateVideoMarksByPath(item.full_path, {
					not_interested: item.not_interested,
				})
			} catch (error) {
				console.error('Failed to update not interested status:', error)
				item.not_interested = !item.not_interested
			}
		},
		async toggleEditList(tab, item) {
			item.in_edit_list = !item.in_edit_list

			try {
				await videosAPI.updateVideoMarksByPath(item.full_path, {
					in_edit_list: item.in_edit_list,
				})
			} catch (error) {
				console.error('Failed to update edit list status:', error)
				item.in_edit_list = !item.in_edit_list
			}
		},
		filteredItems(tab) {
			if (!tab.items) return []

			let filtered = [...tab.items]

			const hasSearchQuery = tab.searchQuery && tab.searchQuery.trim()
			if (!hasSearchQuery && !tab.showNotInterested && !tab.showEditList) {
				if (tab.filterType === 'videos') {
					filtered = filtered.filter((item) => item.type === 'video')
				} else if (tab.filterType === 'folders') {
					filtered = filtered.filter((item) => item.type === 'folder')
				}
			}

			filtered.sort((a, b) => {
				let comparison = 0

				if (a.type === 'folder' && b.type !== 'folder') return -1
				if (a.type !== 'folder' && b.type === 'folder') return 1

				switch (tab.sortBy) {
					case 'name':
						comparison = (a.name || '').localeCompare(b.name || '')
						break
					case 'date': {
						const dateA = new Date(a.modified || 0).getTime()
						const dateB = new Date(b.modified || 0).getTime()
						comparison = dateA - dateB
						break
				}
					case 'size':
						comparison = (a.size || 0) - (b.size || 0)
						break
					case 'duration':
						comparison = (a.duration || 0) - (b.duration || 0)
						break
				}

				return tab.sortOrder === 'asc' ? comparison : -comparison
			})

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
			window.scrollTo({ top: 0, behavior: 'smooth' })
		},
		applyFilters(tab) {
			if (tab && tab.id) {
				if (this.searchDebounceTimers[tab.id]) {
					clearTimeout(this.searchDebounceTimers[tab.id])
				}

				this.searchDebounceTimers[tab.id] = setTimeout(() => {
					this.loadLibraryContent(tab)
				}, 500)
			}
		},
		clearSearch(tab) {
			tab.searchQuery = ''
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
			if (tab.showNotInterested) {
				tab.showEditList = false
			}
			this.loadLibraryContent(tab)
		},
		toggleShowEditList(tab) {
			tab.showEditList = !tab.showEditList
			if (tab.showEditList) {
				tab.showNotInterested = false
			}
			this.loadLibraryContent(tab)
		},
		// formatDuration now provided by useFormatters composable
		getAssetURL(path) {
			return getAssetURL(path)
		},
		showContextMenu(event, tab, item) {
			console.log('showContextMenu called', { tab, item, x: event.clientX, y: event.clientY })
			event.stopPropagation()
			event.preventDefault()
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
		loadPreviewFrames(item) {
			if (!item.preview_path) {
				this.previewFrames = []
				console.log('No preview_path for item:', item.name)
				return
			}

			console.log('Loading preview frames for:', item.name, 'preview_path:', item.preview_path)

			// Generate URLs for all 10 preview frames
			// preview_path is like "1/someFolder/videoName" (directory containing frames)
			const frames = []
			for (let i = 1; i <= 10; i++) {
				const framePath = `previews/${item.preview_path}/frame_${i.toString().padStart(3, '0')}.jpg`
				frames.push(this.getAssetURL(framePath))
			}
			this.previewFrames = frames
			console.log('Loaded', frames.length, 'preview frames')
		},
		handleItemHover(event, item) {
			// Only show preview for videos
			if (item.type !== 'video') return

			// Clear any existing timeout
			if (this.previewTimeout) {
				clearTimeout(this.previewTimeout)
			}

			// Delay preview by 300ms to avoid showing on quick hovers
			this.previewTimeout = setTimeout(() => {
				// Only show preview if item has preview_path
				if (!item.preview_path) return

				// Load preview frames
				this.loadPreviewFrames(item)

				const rect = event.target.getBoundingClientRect()
				const viewportWidth = window.innerWidth
				const viewportHeight = window.innerHeight
				const previewWidth = 320
				const previewHeight = 240

				// Calculate position (show to the right or left of item)
				let x = rect.right + 10
				let y = rect.top

				// If preview would go off right edge, show on left
				if (x + previewWidth > viewportWidth) {
					x = rect.left - previewWidth - 10
				}

				// If preview would go off bottom, adjust upward
				if (y + previewHeight > viewportHeight) {
					y = viewportHeight - previewHeight - 10
				}

				// If preview would go off top, adjust downward
				if (y < 10) {
					y = 10
				}

				this.previewVideo = item
				this.previewPosition = { x, y }
				this.previewFrameIndex = 0

				// Cycle through frames every 200ms for smooth timelapse effect
				this.previewInterval = setInterval(() => {
					this.previewFrameIndex = (this.previewFrameIndex + 1) % this.previewFrames.length
				}, 200)
			}, 300)
		},
		handleItemHoverEnd() {
			// Clear timeout if user moves away before preview shows
			if (this.previewTimeout) {
				clearTimeout(this.previewTimeout)
				this.previewTimeout = null
			}

			// Stop preview cycling
			if (this.previewInterval) {
				clearInterval(this.previewInterval)
				this.previewInterval = null
			}

			// Hide preview and reset state
			this.previewVideo = null
			this.previewFrameIndex = 0
			this.previewFrames = []
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
			if (event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA') {
				return
			}

			const activeTab = this.tabs.find((t) => t.id === this.activeTabId)
			if (!activeTab) return

			if (event.key === 'Backspace' || (event.altKey && event.key === 'ArrowLeft')) {
				event.preventDefault()
				if (activeTab.pathSegments.length > 0) {
					const newSegments = activeTab.pathSegments.slice(0, -1)
					const newPath = newSegments.join('\\')
					this.navigateToPath(activeTab, newPath)
				}
			}

			if (event.ctrlKey && event.key === 't') {
				event.preventDefault()
				const panel = this.splitPanels.find((p) => p.tabIds.includes(this.activeTabId))
				if (panel) {
					this.addTabToPanel(panel)
				}
			}

			if (event.ctrlKey && event.key === 'w') {
				event.preventDefault()
				const panel = this.splitPanels.find((p) => p.tabIds.includes(this.activeTabId))
				if (panel) {
					this.closeTab(this.activeTabId, panel.id)
				}
			}

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

/* Additional styles for split view */
.browser-page {
	height: 100vh;
	display: flex;
	flex-direction: column;
	overflow: hidden;
}

.controls-bar {
	backdrop-filter: blur(10px);
	border: 1px solid rgba(0, 217, 255, 0.2);
	flex-shrink: 0;
}

/* Selection styling */
.content-item.selected {
	background: rgba(0, 217, 255, 0.2) !important;
	border-color: #00d9ff !important;
	box-shadow: 0 0 15px rgba(0, 217, 255, 0.4) !important;
}

.content-item.selected .item-name {
	color: #00d9ff;
	font-weight: 600;
}

/* Video Preview Popup */
.video-preview-popup {
	position: fixed;
	width: 320px;
	background: rgba(0, 0, 0, 0.95);
	border: 2px solid rgba(0, 217, 255, 0.5);
	border-radius: 0.75rem;
	overflow: hidden;
	z-index: 10000;
	box-shadow: 0 8px 32px rgba(0, 0, 0, 0.8);
	pointer-events: none;
}

.preview-video {
	width: 100%;
	height: auto;
	display: block;
	max-height: 240px;
	object-fit: contain;
	background: #000;
}

.preview-info {
	padding: 0.75rem;
	background: rgba(0, 0, 0, 0.9);
}

.preview-name {
	color: #fff;
	font-size: 0.9rem;
	font-weight: 600;
	margin-bottom: 0.25rem;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.preview-meta {
	color: rgba(255, 255, 255, 0.7);
	font-size: 0.8rem;
}

.zoom-level-text {
	min-width: 50px;
	text-align: center;
	font-weight: 600;
}

.split-panels-wrapper {
	display: flex !important;
	flex-direction: row !important;
	gap: 0.5rem;
	position: relative;
	flex: 1;
	overflow: hidden;
	min-height: 0;
}

.split-panel {
	position: relative;
	min-width: 300px;
	height: 100%;
	display: flex;
	flex-direction: column;
}

.browser-tabs-container {
	height: 100%;
	display: flex;
	flex-direction: column;
	overflow: hidden;
}

.tab-content-container {
	flex: 1;
	overflow-y: auto;
	overflow-x: hidden;
}

/* Custom scrollbar for panels */
.tab-content-container::-webkit-scrollbar {
	width: 10px;
}

.tab-content-container::-webkit-scrollbar-track {
	background: rgba(0, 0, 0, 0.3);
	border-radius: 5px;
}

.tab-content-container::-webkit-scrollbar-thumb {
	background: rgba(0, 217, 255, 0.4);
	border-radius: 5px;
}

.tab-content-container::-webkit-scrollbar-thumb:hover {
	background: rgba(0, 217, 255, 0.6);
}

.panel-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 0.2rem;
	background: rgba(0, 0, 0, 0.2);
	border-bottom: 2px solid rgba(0, 217, 255, 0.2);
}

.panel-actions {
	display: flex;
	gap: 0.5rem;
	margin-left: 0.5rem;
}

.panel-resize-handle {
	position: absolute;
	top: 0;
	right: -4px;
	width: 8px;
	height: 100%;
	cursor: col-resize;
	background: rgba(0, 217, 255, 0.1);
	transition: background 0.2s;
	z-index: 10;
}

.panel-resize-handle:hover {
	background: rgba(0, 217, 255, 0.3);
}

.content-grid {
	grid-template-columns: repeat(auto-fill, minmax(calc(280px * var(--grid-scale, 1)), 1fr));
	gap: calc(2rem * var(--grid-scale, 1));
}

.content-item {
	transform: scale(var(--grid-scale, 1));
	transform-origin: top left;
}

.content-item.drag-over {
	border: 2px dashed #00d9ff;
	background: rgba(0, 217, 255, 0.1);
}

.tabs-list {
	flex: 1;
	overflow-x: auto;
	display: flex;
	gap: 0.5rem;
}
</style>
