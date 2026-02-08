<template>
	<div class="edit-list-page">
		<div class="container-fluid py-4">
			<div class="row">
				<!-- Lists Sidebar -->
				<div class="col-md-3 col-lg-2">
					<div class="lists-sidebar">
						<div class="sidebar-header">
							<h5>
								<font-awesome-icon :icon="['fas', 'folder-open']" class="me-2" />
								My Lists
							</h5>
							<button class="btn btn-sm btn-primary" @click="showCreateListModal = true" title="Create New List">
								<font-awesome-icon :icon="['fas', 'plus']" />
							</button>
						</div>

						<div v-if="listsLoading" class="sidebar-loading">
							<font-awesome-icon :icon="['fas', 'spinner']" spin />
							Loading lists...
						</div>

						<div v-else-if="displayLists.length === 0" class="sidebar-empty">
							<font-awesome-icon :icon="['fas', 'list']" size="2x" class="mb-2" />
							<p>No lists yet</p>
							<button class="btn btn-sm btn-outline-primary" @click="showCreateListModal = true">
								Create Your First List
							</button>
						</div>

						<div v-else class="lists-container">
							<div
								v-for="list in displayLists"
								:key="list.id"
								class="list-item-sidebar"
								:class="{ active: selectedListId === list.id }"
								@click="selectList(list.id)"
							>
								<div class="list-icon" :style="{ backgroundColor: list.color }">
									<font-awesome-icon :icon="['fas', list.icon || 'list']" />
								</div>
								<div class="list-info">
									<span class="list-name">{{ list.name }}</span>
									<span class="list-count">{{ list.video_count }} videos</span>
								</div>
								<div class="list-actions-sidebar">
									<button
										v-if="!list.is_default"
										class="btn-icon"
										@click.stop="editList(list)"
										title="Edit List"
									>
										<font-awesome-icon :icon="['fas', 'pen']" />
									</button>
								</div>
							</div>
						</div>
					</div>
				</div>

				<!-- Main Content -->
				<div class="col-md-9 col-lg-10">
					<!-- No List Selected -->
					<div v-if="!selectedListId" class="no-list-selected">
						<font-awesome-icon :icon="['fas', 'hand-pointer']" size="4x" class="mb-3" />
						<h3>Select a List</h3>
						<p>Choose a list from the sidebar to view its contents, or create a new one.</p>
						<button class="btn btn-primary" @click="showCreateListModal = true">
							<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
							Create New List
						</button>
					</div>

					<!-- List Content -->
					<div v-else>
						<!-- List Header -->
						<div class="list-header-main">
							<div class="header-left">
								<div class="list-icon-large" :style="{ backgroundColor: selectedList?.color }">
									<font-awesome-icon :icon="['fas', selectedList?.icon || 'list']" />
								</div>
								<div class="header-info">
									<h1>{{ selectedList?.name }}</h1>
									<p v-if="selectedList?.description">{{ selectedList.description }}</p>
								</div>
							</div>
							<div class="header-actions">
								<button v-if="selectedItems.length > 0" class="btn btn-danger me-2" @click="removeSelectedFromList">
									<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
									Remove Selected ({{ selectedItems.length }})
								</button>
								<button class="btn btn-outline-secondary me-2" @click="exportModal.visible = true" title="Export List">
									<font-awesome-icon :icon="['fas', 'download']" class="me-2" />
									Export
								</button>
								<button v-if="!selectedList?.is_default" class="btn btn-outline-primary me-2" @click="editList(selectedList)">
									<font-awesome-icon :icon="['fas', 'pen']" class="me-2" />
									Edit List
								</button>
								<button v-if="!selectedList?.is_default" class="btn btn-outline-danger me-2" @click="confirmDeleteList">
									<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
									Delete List
								</button>
								<button class="btn btn-primary" @click="refreshListItems">
									<font-awesome-icon :icon="['fas', 'sync']" :spin="itemsLoading" class="me-2" />
									Refresh
								</button>
							</div>
						</div>

						<!-- Stats Summary -->
						<div v-if="listStats" class="stats-summary mb-4">
							<div class="row g-3">
								<div class="col-md-3">
									<StatCard :value="listStats.total_videos" label="Total Videos" :icon="['fas', 'video']" icon-class="primary" />
								</div>
								<div class="col-md-3">
									<StatCard :value="listStats.local_videos" label="Local Videos" :icon="['fas', 'hdd']" icon-class="success" />
								</div>
								<div class="col-md-3">
									<StatCard :value="listStats.scraped_videos" label="Scraped Videos" :icon="['fas', 'cloud']" icon-class="info" />
								</div>
								<div class="col-md-3">
									<StatCard :value="formatTotalDuration(listStats.total_duration)" label="Total Duration" :icon="['fas', 'clock']" icon-class="warning" />
								</div>
							</div>
						</div>

						<!-- Filters and Controls -->
						<div class="filters-bar mb-4">
							<div class="row g-3">
								<div class="col-md-4">
									<div class="input-group">
										<span class="input-group-text">
											<font-awesome-icon :icon="['fas', 'search']" />
										</span>
										<input v-model="searchQuery" type="text" class="form-control" placeholder="Search videos..." @input="filterItems" />
										<button v-if="searchQuery" class="btn btn-outline-secondary" @click="searchQuery = ''; filterItems()">
											<font-awesome-icon :icon="['fas', 'times']" />
										</button>
									</div>
								</div>
								<div class="col-md-2">
									<select v-model="sortBy" class="form-select" @change="sortItems">
										<option value="added_at">Date Added</option>
										<option value="title">Title</option>
										<option value="duration">Duration</option>
									</select>
								</div>
								<div class="col-md-2">
									<select v-model="sortOrder" class="form-select" @change="sortItems">
										<option value="desc">Descending</option>
										<option value="asc">Ascending</option>
									</select>
								</div>
								<div class="col-md-4 text-end">
									<div class="btn-group" role="group">
										<input id="viewGrid" v-model="viewMode" type="radio" class="btn-check" value="grid" />
										<label class="btn btn-outline-primary" for="viewGrid">
											<font-awesome-icon :icon="['fas', 'th']" />
										</label>
										<input id="viewList" v-model="viewMode" type="radio" class="btn-check" value="list" />
										<label class="btn btn-outline-primary" for="viewList">
											<font-awesome-icon :icon="['fas', 'list']" />
										</label>
									</div>
								</div>
							</div>
						</div>

						<!-- Bulk Selection -->
						<div v-if="filteredItems.length > 0" class="bulk-actions mb-3">
							<div class="form-check">
								<input id="selectAll" :checked="allSelected" type="checkbox" class="form-check-input" @change="toggleSelectAll" />
								<label class="form-check-label" for="selectAll"> Select All ({{ filteredItems.length }}) </label>
							</div>
						</div>

						<!-- Loading State -->
						<LoadingState v-if="itemsLoading" spinner="fontawesome" show-text loading-text="Loading list items..." />

						<!-- Empty State -->
						<EmptyState
							v-else-if="filteredItems.length === 0"
							:icon="['fas', 'inbox']"
							icon-size="5x"
							:title="searchQuery ? 'No videos found' : 'This list is empty'"
							:message="searchQuery ? 'Try adjusting your search query' : 'Add videos to this list from the Videos or Browser page using the context menu'"
						/>

						<!-- Grid View -->
						<div v-else-if="viewMode === 'grid'" class="videos-grid">
							<div
								v-for="(item, index) in filteredItems"
								:key="item.id"
								class="video-card"
								:class="{
									selected: isSelected(item.id),
									dragging: dragState.dragging && dragState.draggedItem?.id === item.id,
									'drop-target': dragState.dropIndex === index && dragState.draggedIndex !== index,
								}"
								draggable="true"
								@dragstart="handleDragStart($event, item, index)"
								@dragend="handleDragEnd"
								@dragover.prevent="handleDragOver($event, index)"
								@drop="handleDrop($event, index)"
								@click="toggleSelection(item.id)"
								@contextmenu.prevent="showContextMenu($event, item)"
							>
								<div class="drag-handle" @mousedown.stop title="Drag to reorder">
									<font-awesome-icon :icon="['fas', 'grip-vertical']" />
								</div>
								<div class="selection-checkbox">
									<input type="checkbox" :checked="isSelected(item.id)" @click.stop="toggleSelection(item.id)" />
								</div>

								<div class="video-thumbnail">
									<img v-if="getItemThumbnail(item)" :src="getItemThumbnail(item)" :alt="item.title" loading="lazy" />
									<div v-else class="thumbnail-placeholder">
										<font-awesome-icon :icon="['fas', 'video']" size="3x" />
									</div>
									<div v-if="item.duration" class="badge-duration">{{ formatDuration(item.duration) }}</div>
									<div v-if="item.is_scraped" class="badge-source">
										<font-awesome-icon :icon="['fas', 'cloud']" />
									</div>
								</div>

								<div class="video-info">
									<h3 class="video-title" :title="item.title">{{ item.title }}</h3>

									<div class="video-badges">
										<span v-if="item.resolution" class="badge bg-primary">{{ item.resolution }}</span>
										<span v-if="item.source && item.is_scraped" class="badge bg-info">{{ formatSource(item.source) }}</span>
										<span v-if="item.library_name" class="badge bg-secondary">{{ item.library_name }}</span>
									</div>

									<div v-if="item.file_path" class="video-path" :title="item.file_path">
										<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
										{{ truncatePath(item.file_path) }}
									</div>

									<!-- Notes preview -->
									<div v-if="item.notes" class="video-notes" @click.stop="openNotesModal(item)">
										<font-awesome-icon :icon="['fas', 'sticky-note']" class="me-1" />
										{{ item.notes.length > 30 ? item.notes.substring(0, 30) + '...' : item.notes }}
									</div>

									<div class="video-actions mt-2">
										<button class="btn btn-sm btn-primary me-1" @click.stop="playVideo(item)" title="Play">
											<font-awesome-icon :icon="['fas', 'play']" />
										</button>
										<button class="btn btn-sm btn-outline-secondary me-1" @click.stop="openNotesModal(item)" title="Edit Notes">
											<font-awesome-icon :icon="['fas', 'sticky-note']" />
										</button>
										<button v-if="!item.is_scraped" class="btn btn-sm btn-secondary me-1" @click.stop="openInExplorer(item)" title="Open in Explorer">
											<font-awesome-icon :icon="['fas', 'folder-open']" />
										</button>
										<button class="btn btn-sm btn-danger" @click.stop="removeItemFromList(item)" title="Remove from List">
											<font-awesome-icon :icon="['fas', 'trash']" />
										</button>
									</div>
								</div>
							</div>
						</div>

						<!-- List View -->
						<div v-else class="videos-list">
							<div class="list-header">
								<div class="col-drag"></div>
								<div class="col-select"></div>
								<div class="col-title">Title</div>
								<div class="col-library">Library</div>
								<div class="col-duration">Duration</div>
								<div class="col-source">Source</div>
								<div class="col-added">Added</div>
								<div class="col-actions">Actions</div>
							</div>
							<div
								v-for="(item, index) in filteredItems"
								:key="item.id"
								class="list-item"
								:class="{
									selected: isSelected(item.id),
									dragging: dragState.dragging && dragState.draggedItem?.id === item.id,
									'drop-target': dragState.dropIndex === index && dragState.draggedIndex !== index,
								}"
								draggable="true"
								@dragstart="handleDragStart($event, item, index)"
								@dragend="handleDragEnd"
								@dragover.prevent="handleDragOver($event, index)"
								@drop="handleDrop($event, index)"
								@click="toggleSelection(item.id)"
								@contextmenu.prevent="showContextMenu($event, item)"
							>
								<div class="col-drag">
									<font-awesome-icon :icon="['fas', 'grip-vertical']" class="drag-handle-icon" />
								</div>
								<div class="col-select">
									<input type="checkbox" :checked="isSelected(item.id)" @click.stop="toggleSelection(item.id)" />
								</div>
								<div class="col-title">
									<img v-if="getItemThumbnail(item)" :src="getItemThumbnail(item)" :alt="item.title" class="list-thumbnail" />
									<div v-else class="list-thumbnail-placeholder">
										<font-awesome-icon :icon="['fas', 'video']" />
									</div>
									<div class="title-wrapper">
										<span class="title-text" :title="item.title">{{ item.title }}</span>
										<span v-if="item.file_path" class="path-text" :title="item.file_path">{{ truncatePath(item.file_path) }}</span>
										<span v-if="item.notes" class="notes-text" :title="item.notes">
											<font-awesome-icon :icon="['fas', 'sticky-note']" class="me-1" />
											{{ item.notes.length > 40 ? item.notes.substring(0, 40) + '...' : item.notes }}
										</span>
									</div>
								</div>
								<div class="col-library">
									<span v-if="item.library_name" class="badge bg-secondary">{{ item.library_name }}</span>
									<span v-else>-</span>
								</div>
								<div class="col-duration">{{ item.duration ? formatDuration(item.duration) : '-' }}</div>
								<div class="col-source">
									<span v-if="item.is_scraped" class="badge bg-info">{{ formatSource(item.source) }}</span>
									<span v-else class="badge bg-success">Local</span>
								</div>
								<div class="col-added">{{ formatDate(item.added_at) }}</div>
								<div class="col-actions">
									<button class="btn btn-sm btn-primary me-1" @click.stop="playVideo(item)" title="Play">
										<font-awesome-icon :icon="['fas', 'play']" />
									</button>
									<button class="btn btn-sm btn-outline-secondary me-1" @click.stop="openNotesModal(item)" title="Edit Notes">
										<font-awesome-icon :icon="['fas', 'sticky-note']" />
									</button>
									<button v-if="!item.is_scraped" class="btn btn-sm btn-secondary me-1" @click.stop="openInExplorer(item)" title="Open in Explorer">
										<font-awesome-icon :icon="['fas', 'folder-open']" />
									</button>
									<button class="btn btn-sm btn-danger" @click.stop="removeItemFromList(item)" title="Remove from List">
										<font-awesome-icon :icon="['fas', 'trash']" />
									</button>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Create/Edit List Modal -->
		<div v-if="showCreateListModal || showEditListModal" class="modal-overlay" @click.self="closeListModal">
			<div class="list-modal">
				<div class="modal-header">
					<h4>{{ showEditListModal ? 'Edit List' : 'Create New List' }}</h4>
					<button class="btn-close-modal" @click="closeListModal">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
				<div class="modal-body">
					<div class="mb-3">
						<label class="form-label">List Name *</label>
						<input v-model="listForm.name" type="text" class="form-control" placeholder="Enter list name" />
					</div>
					<div class="mb-3">
						<label class="form-label">Description</label>
						<textarea v-model="listForm.description" class="form-control" rows="2" placeholder="Optional description"></textarea>
					</div>
					<div class="mb-3">
						<label class="form-label">Color</label>
						<div class="color-picker">
							<div
								v-for="color in availableColors"
								:key="color"
								class="color-option"
								:class="{ selected: listForm.color === color }"
								:style="{ backgroundColor: color }"
								@click="listForm.color = color"
							></div>
						</div>
					</div>
					<div class="mb-3">
						<label class="form-label">Icon</label>
						<div class="icon-picker">
							<div
								v-for="icon in availableIcons"
								:key="icon"
								class="icon-option"
								:class="{ selected: listForm.icon === icon }"
								@click="listForm.icon = icon"
							>
								<font-awesome-icon :icon="['fas', icon]" />
							</div>
						</div>
					</div>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="closeListModal">Cancel</button>
					<button class="btn btn-primary" :disabled="!listForm.name" @click="saveList">
						{{ showEditListModal ? 'Save Changes' : 'Create List' }}
					</button>
				</div>
			</div>
		</div>

		<!-- Delete Confirmation Modal -->
		<div v-if="showDeleteConfirm" class="modal-overlay" @click.self="showDeleteConfirm = false">
			<div class="list-modal confirm-modal">
				<div class="modal-header">
					<h4>Delete List</h4>
					<button class="btn-close-modal" @click="showDeleteConfirm = false">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
				<div class="modal-body">
					<p>Are you sure you want to delete the list <strong>"{{ selectedList?.name }}"</strong>?</p>
					<p class="text-muted">This will remove all {{ selectedList?.video_count }} videos from this list. The videos themselves will not be deleted.</p>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="showDeleteConfirm = false">Cancel</button>
					<button class="btn btn-danger" @click="deleteList">Delete List</button>
				</div>
			</div>
		</div>

		<!-- Video Player Modal -->
		<VideoPlayer :visible="playerVisible" :video="selectedVideo" @close="playerVisible = false" />

		<!-- Context Menu -->
		<div
			v-if="contextMenu.visible"
			class="context-menu"
			:style="{ top: `${contextMenu.y}px`, left: `${contextMenu.x}px` }"
			@click.stop
		>
			<button class="context-menu-item" @click="playVideo(contextMenu.item); closeContextMenu()">
				<font-awesome-icon :icon="['fas', 'play']" class="me-2" />
				Play Video
			</button>
			<button class="context-menu-item" @click="openNotesModal(contextMenu.item); closeContextMenu()">
				<font-awesome-icon :icon="['fas', 'sticky-note']" class="me-2" />
				Edit Notes
			</button>
			<button v-if="!contextMenu.item?.is_scraped" class="context-menu-item" @click="openInExplorer(contextMenu.item); closeContextMenu()">
				<font-awesome-icon :icon="['fas', 'folder-open']" class="me-2" />
				Open in Explorer
			</button>
			<button v-if="contextMenu.item?.file_path" class="context-menu-item" @click="copyPath(contextMenu.item); closeContextMenu()">
				<font-awesome-icon :icon="['fas', 'copy']" class="me-2" />
				Copy Path
			</button>
			<div class="context-menu-divider"></div>
			<button class="context-menu-item text-danger" @click="removeItemFromList(contextMenu.item); closeContextMenu()">
				<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
				Remove from List
			</button>
		</div>

		<!-- Notes Modal -->
		<div v-if="notesModal.visible" class="modal-overlay" @click.self="closeNotesModal">
			<div class="list-modal">
				<div class="modal-header">
					<h4>Edit Notes</h4>
					<button class="btn-close-modal" @click="closeNotesModal">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
				<div class="modal-body">
					<p class="text-muted mb-2">{{ notesModal.item?.title }}</p>
					<textarea
						v-model="notesModal.notes"
						class="form-control"
						rows="4"
						placeholder="Add notes about this video..."
					></textarea>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="closeNotesModal">Cancel</button>
					<button class="btn btn-primary" :disabled="notesModal.saving" @click="saveNotes">
						<font-awesome-icon v-if="notesModal.saving" :icon="['fas', 'spinner']" spin class="me-2" />
						Save Notes
					</button>
				</div>
			</div>
		</div>

		<!-- Export Modal -->
		<div v-if="exportModal.visible" class="modal-overlay" @click.self="exportModal.visible = false">
			<div class="list-modal">
				<div class="modal-header">
					<h4>Export List</h4>
					<button class="btn-close-modal" @click="exportModal.visible = false">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
				<div class="modal-body">
					<p>Export "{{ selectedList?.name }}" to a file.</p>
					<div class="mb-3">
						<label class="form-label">Format</label>
						<select v-model="exportModal.format" class="form-select">
							<option value="json">JSON (Full data)</option>
							<option value="csv">CSV (Spreadsheet)</option>
							<option value="txt">Text (Simple list)</option>
						</select>
					</div>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="exportModal.visible = false">Cancel</button>
					<button class="btn btn-primary" :disabled="exportModal.exporting" @click="exportList">
						<font-awesome-icon v-if="exportModal.exporting" :icon="['fas', 'spinner']" spin class="me-2" />
						<font-awesome-icon v-else :icon="['fas', 'download']" class="me-2" />
						Export
					</button>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import VideoPlayer from '@/components/VideoPlayer.vue'
import { useFormatters } from '@/composables/useFormatters'
import { LoadingState, EmptyState, StatCard } from '@/components/shared'
import { getAssetURL, videoListsAPI } from '@/services/api'

const API_BASE = 'http://localhost:8080/api/v1'

export default {
	name: 'EditListPage',
	components: {
		VideoPlayer,
		LoadingState,
		EmptyState,
		StatCard,
	},
	data() {
		return {
			// Lists
			lists: [],
			listsLoading: false,
			selectedListId: null,

			// List Items
			listItems: [],
			filteredItems: [],
			listStats: null,
			itemsLoading: false,

			// Selection
			selectedItems: [],

			// Filters
			searchQuery: '',
			sortBy: 'added_at',
			sortOrder: 'desc',
			viewMode: 'grid',

			// Modals
			showCreateListModal: false,
			showEditListModal: false,
			showDeleteConfirm: false,
			editingListId: null,

			// Form
			listForm: {
				name: '',
				description: '',
				color: '#007bff',
				icon: 'list',
			},

			// Video Player
			playerVisible: false,
			selectedVideo: null,

			// Context Menu
			contextMenu: {
				visible: false,
				x: 0,
				y: 0,
				item: null,
			},

			// Drag and Drop
			dragState: {
				dragging: false,
				draggedItem: null,
				draggedIndex: -1,
				dropIndex: -1,
			},

			// Notes Modal
			notesModal: {
				visible: false,
				item: null,
				notes: '',
				saving: false,
			},

			// Export Modal
			exportModal: {
				visible: false,
				format: 'json',
				exporting: false,
			},

			// Options
			availableColors: [
				'#007bff', '#6c757d', '#28a745', '#dc3545', '#ffc107',
				'#17a2b8', '#6f42c1', '#e83e8c', '#fd7e14', '#20c997',
			],
			availableIcons: [
				'list', 'star', 'heart', 'bookmark', 'folder',
				'film', 'video', 'play-circle', 'fire', 'clock',
				'eye', 'thumbs-up', 'trophy', 'gem', 'crown',
			],
		}
	},
	computed: {
		selectedList() {
			return this.lists.find(l => l.id === this.selectedListId)
		},
		allSelected() {
			return this.filteredItems.length > 0 && this.selectedItems.length === this.filteredItems.length
		},
		// Filter out the default "Edit List" that was for old functionality
		displayLists() {
			return this.lists.filter(l => !l.is_default)
		},
	},
	created() {
		const formatters = useFormatters()
		this.formatDuration = formatters.formatDuration
		this.formatDate = formatters.formatDate
		this.formatTotalDuration = formatters.formatTotalDuration
	},
	async mounted() {
		await this.loadLists()
		// Auto-select first non-default list if available
		if (this.displayLists.length > 0) {
			this.selectList(this.displayLists[0].id)
		}
		// Close context menu when clicking outside
		document.addEventListener('click', this.closeContextMenu)
		// Keyboard shortcuts
		document.addEventListener('keydown', this.handleKeydown)
	},
	beforeUnmount() {
		document.removeEventListener('click', this.closeContextMenu)
		document.removeEventListener('keydown', this.handleKeydown)
	},
	methods: {
		// List Management
		async loadLists() {
			this.listsLoading = true
			try {
				const response = await fetch(`${API_BASE}/lists`)
				const data = await response.json()
				if (data.success) {
					this.lists = data.data || []
				}
			} catch (error) {
				console.error('Failed to load lists:', error)
				this.$toast.error('Failed to load lists')
			} finally {
				this.listsLoading = false
			}
		},

		async selectList(listId) {
			this.selectedListId = listId
			this.selectedItems = []
			await this.loadListItems()
		},

		async loadListItems() {
			if (!this.selectedListId) return

			this.itemsLoading = true
			try {
				const response = await fetch(`${API_BASE}/lists/${this.selectedListId}/items`)
				const data = await response.json()
				if (data.success) {
					this.listItems = data.data.items || []
					this.listStats = data.data.stats
					this.filterItems()
				}
			} catch (error) {
				console.error('Failed to load list items:', error)
				this.$toast.error('Failed to load list items')
			} finally {
				this.itemsLoading = false
			}
		},

		async refreshListItems() {
			await this.loadListItems()
			this.$toast.success('List refreshed')
		},

		// Create/Edit List
		editList(list) {
			this.editingListId = list.id
			this.listForm = {
				name: list.name,
				description: list.description || '',
				color: list.color || '#007bff',
				icon: list.icon || 'list',
			}
			this.showEditListModal = true
		},

		closeListModal() {
			this.showCreateListModal = false
			this.showEditListModal = false
			this.editingListId = null
			this.listForm = {
				name: '',
				description: '',
				color: '#007bff',
				icon: 'list',
			}
		},

		async saveList() {
			try {
				if (this.showEditListModal && this.editingListId) {
					// Update existing list
					const response = await fetch(`${API_BASE}/lists/${this.editingListId}`, {
						method: 'PUT',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify(this.listForm),
					})
					const data = await response.json()
					if (data.success) {
						this.$toast.success('List updated')
						await this.loadLists()
					} else {
						throw new Error(data.message)
					}
				} else {
					// Create new list
					const response = await fetch(`${API_BASE}/lists`, {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify(this.listForm),
					})
					const data = await response.json()
					if (data.success) {
						this.$toast.success('List created')
						await this.loadLists()
						// Select the new list
						if (data.data && data.data.id) {
							this.selectList(data.data.id)
						}
					} else {
						throw new Error(data.message)
					}
				}
				this.closeListModal()
			} catch (error) {
				console.error('Failed to save list:', error)
				this.$toast.error(error.message || 'Failed to save list')
			}
		},

		// Delete List
		confirmDeleteList() {
			this.showDeleteConfirm = true
		},

		async deleteList() {
			try {
				const response = await fetch(`${API_BASE}/lists/${this.selectedListId}`, {
					method: 'DELETE',
				})
				const data = await response.json()
				if (data.success) {
					this.$toast.success('List deleted')
					this.showDeleteConfirm = false
					this.selectedListId = null
					await this.loadLists()
					// Select first non-default list if available
					if (this.displayLists.length > 0) {
						this.selectList(this.displayLists[0].id)
					}
				} else {
					throw new Error(data.message)
				}
			} catch (error) {
				console.error('Failed to delete list:', error)
				this.$toast.error(error.message || 'Failed to delete list')
			}
		},

		// Item Management
		async removeItemFromList(item) {
			if (!confirm(`Remove "${item.title}" from this list?`)) return

			try {
				const body = {}
				if (item.video_id) {
					body.video_ids = [item.video_id]
				} else if (item.scraped_video_id) {
					body.scraped_video_ids = [item.scraped_video_id]
				} else {
					body.item_ids = [item.id]
				}

				const response = await fetch(`${API_BASE}/lists/${this.selectedListId}/videos`, {
					method: 'DELETE',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(body),
				})
				const data = await response.json()
				if (data.success) {
					this.$toast.success('Video removed from list')
					await this.loadListItems()
					await this.loadLists() // Update video count
				} else {
					throw new Error(data.message)
				}
			} catch (error) {
				console.error('Failed to remove item:', error)
				this.$toast.error(error.message || 'Failed to remove video')
			}
		},

		async removeSelectedFromList() {
			if (!confirm(`Remove ${this.selectedItems.length} videos from this list?`)) return

			try {
				const itemIDs = this.selectedItems
				const response = await fetch(`${API_BASE}/lists/${this.selectedListId}/videos`, {
					method: 'DELETE',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ item_ids: itemIDs }),
				})
				const data = await response.json()
				if (data.success) {
					this.$toast.success(`${itemIDs.length} videos removed from list`)
					this.selectedItems = []
					await this.loadListItems()
					await this.loadLists()
				} else {
					throw new Error(data.message)
				}
			} catch (error) {
				console.error('Failed to remove items:', error)
				this.$toast.error(error.message || 'Failed to remove videos')
			}
		},

		// Selection
		isSelected(itemId) {
			return this.selectedItems.includes(itemId)
		},

		toggleSelection(itemId) {
			const index = this.selectedItems.indexOf(itemId)
			if (index > -1) {
				this.selectedItems.splice(index, 1)
			} else {
				this.selectedItems.push(itemId)
			}
		},

		toggleSelectAll() {
			if (this.allSelected) {
				this.selectedItems = []
			} else {
				this.selectedItems = this.filteredItems.map(item => item.id)
			}
		},

		// Filtering & Sorting
		filterItems() {
			if (!this.searchQuery.trim()) {
				this.filteredItems = [...this.listItems]
			} else {
				const query = this.searchQuery.toLowerCase()
				this.filteredItems = this.listItems.filter(item =>
					item.title?.toLowerCase().includes(query)
				)
			}
			this.sortItems()
		},

		sortItems() {
			const order = this.sortOrder === 'asc' ? 1 : -1
			this.filteredItems.sort((a, b) => {
				let aVal, bVal
				switch (this.sortBy) {
					case 'title':
						aVal = (a.title || '').toLowerCase()
						bVal = (b.title || '').toLowerCase()
						return aVal.localeCompare(bVal) * order
					case 'duration':
						aVal = a.duration || 0
						bVal = b.duration || 0
						return (aVal - bVal) * order
					case 'added_at':
					default:
						aVal = new Date(a.added_at || 0)
						bVal = new Date(b.added_at || 0)
						return (aVal - bVal) * order
				}
			})
		},

		// Utility Methods
		getItemThumbnail(item) {
			if (item.thumbnail_path) {
				return getAssetURL(item.thumbnail_path)
			}
			if (item.thumbnail_url) {
				return item.thumbnail_url
			}
			return null
		},

		formatSource(source) {
			if (!source) return 'Unknown'
			// Extract domain name
			const match = source.match(/([^.]+)\.com/)
			if (match) return match[1].charAt(0).toUpperCase() + match[1].slice(1)
			return source
		},

		playVideo(item) {
			if (item.is_scraped && item.scraped_video_id) {
				this.$router.push(`/watch/scraped/${item.scraped_video_id}`)
			} else if (item.video_id) {
				// For local videos, use the video player modal
				this.selectedVideo = {
					id: item.video_id,
					title: item.title,
					thumbnail_path: item.thumbnail_path,
					duration: item.duration,
				}
				this.playerVisible = true
			}
		},

		// Context Menu
		showContextMenu(event, item) {
			this.contextMenu.visible = true
			this.contextMenu.x = event.clientX
			this.contextMenu.y = event.clientY
			this.contextMenu.item = item
		},

		closeContextMenu() {
			this.contextMenu.visible = false
			this.contextMenu.item = null
		},

		// Utility methods
		truncatePath(path) {
			if (!path) return ''
			const maxLength = 50
			if (path.length <= maxLength) return path
			// Show the last part of the path
			const parts = path.split(/[/\\]/)
			let result = parts[parts.length - 1]
			for (let i = parts.length - 2; i >= 0; i--) {
				const newResult = parts[i] + '/' + result
				if (newResult.length > maxLength) {
					return '...' + '/' + result
				}
				result = newResult
			}
			return result
		},

		async openInExplorer(item) {
			if (!item.video_id) {
				this.$toast.error('Cannot open: video not in database')
				return
			}
			try {
				const response = await fetch(`${API_BASE}/videos/${item.video_id}/open-in-explorer`, {
					method: 'POST',
				})
				const data = await response.json()
				if (data.success) {
					this.$toast.success('Opened in Explorer')
				} else {
					throw new Error(data.message)
				}
			} catch (error) {
				console.error('Failed to open in explorer:', error)
				this.$toast.error(error.message || 'Failed to open in Explorer')
			}
		},

		async copyPath(item) {
			if (!item.file_path) return
			try {
				await navigator.clipboard.writeText(item.file_path)
				this.$toast.success('Path copied to clipboard')
			} catch (error) {
				console.error('Failed to copy path:', error)
				this.$toast.error('Failed to copy path')
			}
		},

		// Drag and Drop
		handleDragStart(event, item, index) {
			this.dragState.dragging = true
			this.dragState.draggedItem = item
			this.dragState.draggedIndex = index
			event.dataTransfer.effectAllowed = 'move'
			event.dataTransfer.setData('text/plain', item.id.toString())
		},

		handleDragEnd() {
			this.dragState.dragging = false
			this.dragState.draggedItem = null
			this.dragState.draggedIndex = -1
			this.dragState.dropIndex = -1
		},

		handleDragOver(event, index) {
			event.preventDefault()
			this.dragState.dropIndex = index
		},

		async handleDrop(event, dropIndex) {
			event.preventDefault()
			const dragIndex = this.dragState.draggedIndex

			if (dragIndex === dropIndex || dragIndex === -1) {
				this.handleDragEnd()
				return
			}

			// Reorder locally first for immediate feedback
			const items = [...this.filteredItems]
			const [movedItem] = items.splice(dragIndex, 1)
			items.splice(dropIndex, 0, movedItem)
			this.filteredItems = items
			this.listItems = items

			// Update on server
			try {
				const itemIds = items.map(i => i.id)
				await videoListsAPI.reorderItems(this.selectedListId, itemIds)
				this.$toast.success('Order saved')
			} catch (error) {
				console.error('Failed to reorder:', error)
				this.$toast.error('Failed to save order')
				// Reload to restore original order
				await this.loadListItems()
			}

			this.handleDragEnd()
		},

		// Notes Modal
		openNotesModal(item) {
			this.notesModal.item = item
			this.notesModal.notes = item.notes || ''
			this.notesModal.visible = true
		},

		closeNotesModal() {
			this.notesModal.visible = false
			this.notesModal.item = null
			this.notesModal.notes = ''
		},

		async saveNotes() {
			this.notesModal.saving = true
			try {
				// Update via API - we need to update the item's notes
				const response = await fetch(`${API_BASE}/lists/${this.selectedListId}/items/${this.notesModal.item.id}/notes`, {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ notes: this.notesModal.notes }),
				})
				const data = await response.json()
				if (data.success) {
					// Update locally
					const item = this.listItems.find(i => i.id === this.notesModal.item.id)
					if (item) item.notes = this.notesModal.notes
					const filteredItem = this.filteredItems.find(i => i.id === this.notesModal.item.id)
					if (filteredItem) filteredItem.notes = this.notesModal.notes
					this.$toast.success('Notes saved')
					this.closeNotesModal()
				} else {
					throw new Error(data.message)
				}
			} catch (error) {
				console.error('Failed to save notes:', error)
				this.$toast.error(error.message || 'Failed to save notes')
			} finally {
				this.notesModal.saving = false
			}
		},

		// Export
		async exportList() {
			this.exportModal.exporting = true
			try {
				const list = this.selectedList
				const items = this.listItems
				let content = ''
				let filename = ''
				let mimeType = ''

				if (this.exportModal.format === 'json') {
					const exportData = {
						list: {
							name: list.name,
							description: list.description,
							color: list.color,
							icon: list.icon,
						},
						items: items.map(i => ({
							title: i.title,
							file_path: i.file_path,
							duration: i.duration,
							resolution: i.resolution,
							is_scraped: i.is_scraped,
							source: i.source,
							notes: i.notes,
							added_at: i.added_at,
						})),
						exported_at: new Date().toISOString(),
					}
					content = JSON.stringify(exportData, null, 2)
					filename = `${list.name.replace(/[^a-z0-9]/gi, '_')}.json`
					mimeType = 'application/json'
				} else if (this.exportModal.format === 'csv') {
					const headers = ['Title', 'File Path', 'Duration', 'Resolution', 'Source', 'Notes', 'Added At']
					const rows = items.map(i => [
						`"${(i.title || '').replace(/"/g, '""')}"`,
						`"${(i.file_path || '').replace(/"/g, '""')}"`,
						i.duration || '',
						i.resolution || '',
						i.is_scraped ? i.source : 'Local',
						`"${(i.notes || '').replace(/"/g, '""')}"`,
						i.added_at || '',
					])
					content = [headers.join(','), ...rows.map(r => r.join(','))].join('\n')
					filename = `${list.name.replace(/[^a-z0-9]/gi, '_')}.csv`
					mimeType = 'text/csv'
				} else {
					// Plain text
					content = `${list.name}\n${'='.repeat(list.name.length)}\n\n`
					content += items.map((i, idx) => `${idx + 1}. ${i.title}${i.notes ? ` - ${i.notes}` : ''}`).join('\n')
					filename = `${list.name.replace(/[^a-z0-9]/gi, '_')}.txt`
					mimeType = 'text/plain'
				}

				// Download file
				const blob = new Blob([content], { type: mimeType })
				const url = URL.createObjectURL(blob)
				const a = document.createElement('a')
				a.href = url
				a.download = filename
				document.body.appendChild(a)
				a.click()
				document.body.removeChild(a)
				URL.revokeObjectURL(url)

				this.$toast.success(`List exported as ${filename}`)
				this.exportModal.visible = false
			} catch (error) {
				console.error('Failed to export:', error)
				this.$toast.error('Failed to export list')
			} finally {
				this.exportModal.exporting = false
			}
		},

		// Keyboard Shortcuts
		handleKeydown(event) {
			// Don't handle if typing in input/textarea
			if (event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA') return

			// Delete key - remove selected items
			if (event.key === 'Delete' && this.selectedItems.length > 0) {
				event.preventDefault()
				this.removeSelectedFromList()
			}

			// Escape - deselect all
			if (event.key === 'Escape') {
				this.selectedItems = []
				this.closeContextMenu()
				this.closeNotesModal()
				this.exportModal.visible = false
			}

			// Ctrl+A - select all
			if ((event.ctrlKey || event.metaKey) && event.key === 'a' && this.filteredItems.length > 0) {
				event.preventDefault()
				this.selectedItems = this.filteredItems.map(item => item.id)
			}
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/edit_list_page.css';

/* Lists Sidebar */
.lists-sidebar {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	padding: 1rem;
	position: sticky;
	top: 1rem;
}

.sidebar-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 1rem;
	padding-bottom: 0.75rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.sidebar-header h5 {
	margin: 0;
	color: white;
	font-size: 1rem;
	font-weight: 600;
}

.sidebar-loading,
.sidebar-empty {
	text-align: center;
	padding: 2rem 1rem;
	color: rgba(255, 255, 255, 0.6);
}

.sidebar-empty p {
	margin: 0.5rem 0;
	font-size: 0.875rem;
}

.lists-container {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.list-item-sidebar {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	padding: 0.75rem;
	border-radius: 8px;
	cursor: pointer;
	transition: all 0.2s;
	border: 1px solid transparent;
}

.list-item-sidebar:hover {
	background: rgba(255, 255, 255, 0.05);
}

.list-item-sidebar.active {
	background: rgba(0, 217, 255, 0.1);
	border-color: #00d9ff;
}

.list-icon {
	width: 36px;
	height: 36px;
	border-radius: 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	color: white;
	font-size: 0.875rem;
	flex-shrink: 0;
}

.list-info {
	flex: 1;
	overflow: hidden;
}

.list-name {
	display: block;
	color: white;
	font-weight: 500;
	font-size: 0.875rem;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.list-count {
	display: block;
	color: rgba(255, 255, 255, 0.5);
	font-size: 0.75rem;
}

.list-actions-sidebar {
	opacity: 0;
	transition: opacity 0.2s;
}

.list-item-sidebar:hover .list-actions-sidebar {
	opacity: 1;
}

.btn-icon {
	background: rgba(255, 255, 255, 0.1);
	border: none;
	color: rgba(255, 255, 255, 0.7);
	width: 28px;
	height: 28px;
	border-radius: 6px;
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	transition: all 0.2s;
}

.btn-icon:hover {
	background: rgba(255, 255, 255, 0.2);
	color: white;
}

/* No List Selected */
.no-list-selected {
	text-align: center;
	padding: 6rem 2rem;
	color: rgba(255, 255, 255, 0.7);
}

.no-list-selected h3 {
	color: white;
	margin-bottom: 1rem;
}

.no-list-selected p {
	margin-bottom: 2rem;
}

/* List Header Main */
.list-header-main {
	display: flex;
	justify-content: space-between;
	align-items: flex-start;
	flex-wrap: wrap;
	gap: 1rem;
	margin-bottom: 2rem;
}

.header-left {
	display: flex;
	align-items: center;
	gap: 1rem;
}

.list-icon-large {
	width: 60px;
	height: 60px;
	border-radius: 12px;
	display: flex;
	align-items: center;
	justify-content: center;
	color: white;
	font-size: 1.5rem;
	flex-shrink: 0;
}

.header-info h1 {
	color: white;
	font-weight: 600;
	font-size: 1.75rem;
	margin-bottom: 0.25rem;
}

.header-info p {
	color: rgba(255, 255, 255, 0.6);
	margin: 0;
	font-size: 0.875rem;
}

/* Video Grid */
.badge-source {
	position: absolute;
	top: 8px;
	left: 8px;
	background: rgba(23, 162, 184, 0.9);
	color: white;
	padding: 4px 8px;
	border-radius: 4px;
	font-size: 0.7rem;
}

/* Videos List */
.videos-list .col-source {
	width: 100px;
}

/* Modal Styles */
.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.8);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 1000;
}

.list-modal {
	background: #1a1a2e;
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 16px;
	width: 100%;
	max-width: 500px;
	overflow: hidden;
}

.list-modal.confirm-modal {
	max-width: 400px;
}

.list-modal .modal-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1.25rem 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.list-modal .modal-header h4 {
	margin: 0;
	color: white;
	font-weight: 600;
}

.btn-close-modal {
	background: rgba(255, 255, 255, 0.1);
	border: none;
	color: rgba(255, 255, 255, 0.7);
	width: 32px;
	height: 32px;
	border-radius: 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	transition: all 0.2s;
}

.btn-close-modal:hover {
	background: rgba(255, 255, 255, 0.2);
	color: white;
}

.list-modal .modal-body {
	padding: 1.5rem;
}

.list-modal .modal-body .form-label {
	color: rgba(255, 255, 255, 0.9);
	font-weight: 500;
	margin-bottom: 0.5rem;
}

.list-modal .modal-body .form-control,
.list-modal .modal-body .form-select {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	color: white;
}

.list-modal .modal-body .form-control:focus,
.list-modal .modal-body .form-select:focus {
	background: rgba(255, 255, 255, 0.08);
	border-color: #00d9ff;
	box-shadow: 0 0 0 0.25rem rgba(0, 217, 255, 0.25);
}

.list-modal .modal-body .form-control::placeholder {
	color: rgba(255, 255, 255, 0.4);
}

.list-modal .modal-body p {
	color: rgba(255, 255, 255, 0.9);
	margin-bottom: 0.5rem;
}

.list-modal .modal-body .text-muted {
	color: rgba(255, 255, 255, 0.5) !important;
	font-size: 0.875rem;
}

.list-modal .modal-footer {
	display: flex;
	justify-content: flex-end;
	gap: 0.75rem;
	padding: 1.25rem 1.5rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}

/* Color Picker */
.color-picker {
	display: flex;
	flex-wrap: wrap;
	gap: 0.5rem;
}

.color-option {
	width: 32px;
	height: 32px;
	border-radius: 8px;
	cursor: pointer;
	border: 2px solid transparent;
	transition: all 0.2s;
}

.color-option:hover {
	transform: scale(1.1);
}

.color-option.selected {
	border-color: white;
	box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.3);
}

/* Icon Picker */
.icon-picker {
	display: flex;
	flex-wrap: wrap;
	gap: 0.5rem;
}

.icon-option {
	width: 40px;
	height: 40px;
	border-radius: 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	background: rgba(255, 255, 255, 0.05);
	color: rgba(255, 255, 255, 0.7);
	cursor: pointer;
	border: 2px solid transparent;
	transition: all 0.2s;
}

.icon-option:hover {
	background: rgba(255, 255, 255, 0.1);
	color: white;
}

.icon-option.selected {
	border-color: #00d9ff;
	color: #00d9ff;
	background: rgba(0, 217, 255, 0.1);
}

/* Video Path Display */
.video-path {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.5);
	margin-top: 0.25rem;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

/* List View Improvements */
.videos-list .col-library {
	width: 120px;
}

.title-wrapper {
	display: flex;
	flex-direction: column;
	overflow: hidden;
}

.title-text {
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.path-text {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.4);
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

/* Context Menu */
.context-menu {
	position: fixed;
	background: #1a1a2e;
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 8px;
	padding: 0.5rem 0;
	min-width: 180px;
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
	z-index: 1100;
}

.context-menu-item {
	display: flex;
	align-items: center;
	width: 100%;
	padding: 0.5rem 1rem;
	background: none;
	border: none;
	color: rgba(255, 255, 255, 0.9);
	font-size: 0.875rem;
	cursor: pointer;
	transition: all 0.2s;
	text-align: left;
}

.context-menu-item:hover {
	background: rgba(255, 255, 255, 0.1);
}

.context-menu-item.text-danger {
	color: #dc3545;
}

.context-menu-item.text-danger:hover {
	background: rgba(220, 53, 69, 0.2);
}

.context-menu-divider {
	height: 1px;
	background: rgba(255, 255, 255, 0.1);
	margin: 0.25rem 0;
}

/* Drag and Drop */
.drag-handle {
	position: absolute;
	top: 8px;
	left: 8px;
	width: 24px;
	height: 24px;
	display: flex;
	align-items: center;
	justify-content: center;
	background: rgba(0, 0, 0, 0.5);
	border-radius: 4px;
	color: rgba(255, 255, 255, 0.5);
	cursor: grab;
	opacity: 0;
	transition: opacity 0.2s;
	z-index: 5;
}

.video-card:hover .drag-handle {
	opacity: 1;
}

.drag-handle:hover {
	color: rgba(255, 255, 255, 0.9);
	background: rgba(0, 0, 0, 0.7);
}

.video-card.dragging,
.list-item.dragging {
	opacity: 0.5;
	transform: scale(0.98);
}

.video-card.drop-target,
.list-item.drop-target {
	border: 2px dashed #00d9ff !important;
	background: rgba(0, 217, 255, 0.1);
}

.videos-list .col-drag {
	width: 30px;
	display: flex;
	align-items: center;
	justify-content: center;
}

.drag-handle-icon {
	color: rgba(255, 255, 255, 0.3);
	cursor: grab;
	transition: color 0.2s;
}

.list-item:hover .drag-handle-icon {
	color: rgba(255, 255, 255, 0.7);
}

/* Video Notes */
.video-notes {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.6);
	margin-top: 0.25rem;
	padding: 0.25rem 0.5rem;
	background: rgba(255, 255, 255, 0.05);
	border-radius: 4px;
	cursor: pointer;
	transition: background 0.2s;
}

.video-notes:hover {
	background: rgba(255, 255, 255, 0.1);
}

.notes-text {
	font-size: 0.7rem;
	color: rgba(255, 255, 255, 0.5);
	font-style: italic;
}

/* Responsive */
@media (max-width: 768px) {
	.lists-sidebar {
		position: static;
		margin-bottom: 1rem;
	}

	.list-header-main {
		flex-direction: column;
	}

	.header-actions {
		width: 100%;
		display: flex;
		flex-wrap: wrap;
	}

	.header-actions .btn {
		flex: 1;
		min-width: 120px;
	}
}
</style>
