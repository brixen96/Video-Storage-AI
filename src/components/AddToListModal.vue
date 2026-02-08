<template>
	<div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
		<div class="add-to-list-modal">
			<div class="modal-header">
				<h4>Add to List</h4>
				<button class="btn-close-modal" @click="$emit('close')">
					<font-awesome-icon :icon="['fas', 'times']" />
				</button>
			</div>
			<div class="modal-body">
				<!-- Bulk Selection Preview -->
				<div v-if="videos && videos.length > 1" class="bulk-preview">
					<div class="bulk-icon">
						<font-awesome-icon :icon="['fas', 'layer-group']" />
					</div>
					<div class="bulk-info">
						<span class="bulk-count">{{ videos.length }} videos selected</span>
						<span class="bulk-hint">Will be added to selected lists</span>
					</div>
				</div>

				<!-- Single Video Preview -->
				<div v-else-if="video" class="video-preview">
					<img v-if="thumbnailUrl" :src="thumbnailUrl" :alt="video.title" class="preview-thumbnail" />
					<div v-else class="preview-thumbnail-placeholder">
						<font-awesome-icon :icon="['fas', 'video']" />
					</div>
					<div class="preview-info">
						<span class="preview-title">{{ video.title }}</span>
						<span v-if="video.duration" class="preview-duration">{{ formatDuration(video.duration) }}</span>
					</div>
				</div>

				<!-- Loading State -->
				<div v-if="loading" class="lists-loading">
					<font-awesome-icon :icon="['fas', 'spinner']" spin />
					Loading lists...
				</div>

				<!-- Lists Selection -->
				<div v-else-if="lists.length > 0" class="lists-selection">
					<div
						v-for="list in lists"
						:key="list.id"
						class="list-option"
						:class="{ selected: selectedListIds.includes(list.id), 'already-in': isInList(list.id) }"
						@click="toggleListSelection(list.id)"
					>
						<div class="list-checkbox">
							<font-awesome-icon v-if="isInList(list.id)" :icon="['fas', 'check-circle']" class="text-success" />
							<font-awesome-icon v-else-if="selectedListIds.includes(list.id)" :icon="['fas', 'check-square']" />
							<font-awesome-icon v-else :icon="['far', 'square']" />
						</div>
						<div class="list-icon" :style="{ backgroundColor: list.color }">
							<font-awesome-icon :icon="['fas', list.icon || 'list']" />
						</div>
						<div class="list-info">
							<span class="list-name">{{ list.name }}</span>
							<span class="list-count">{{ list.video_count }} videos</span>
						</div>
						<span v-if="isInList(list.id)" class="already-badge">Already added</span>
					</div>
				</div>

				<!-- Empty State -->
				<div v-else class="no-lists">
					<font-awesome-icon :icon="['fas', 'inbox']" size="2x" class="mb-2" />
					<p>No lists available</p>
					<button class="btn btn-sm btn-primary" @click="createNewList">
						<font-awesome-icon :icon="['fas', 'plus']" class="me-1" />
						Create New List
					</button>
				</div>

				<!-- Create New List Section -->
				<div v-if="lists.length > 0 && !showNewListForm" class="create-new-section">
					<button class="btn btn-outline-primary btn-sm" @click="showNewListForm = true">
						<font-awesome-icon :icon="['fas', 'plus']" class="me-1" />
						Create New List
					</button>
				</div>

				<!-- New List Form -->
				<div v-if="showNewListForm" class="new-list-form">
					<div class="form-group">
						<input
							v-model="newListName"
							type="text"
							class="form-control"
							placeholder="New list name"
							@keyup.enter="createAndAddToList"
						/>
					</div>
					<div class="form-actions">
						<button class="btn btn-sm btn-secondary" @click="showNewListForm = false">Cancel</button>
						<button class="btn btn-sm btn-primary" :disabled="!newListName" @click="createAndAddToList">
							Create & Add
						</button>
					</div>
				</div>
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
				<button class="btn btn-primary" :disabled="selectedListIds.length === 0 || saving" @click="addToSelectedLists">
					<font-awesome-icon v-if="saving" :icon="['fas', 'spinner']" spin class="me-1" />
					{{ saving ? 'Adding...' : `Add to ${selectedListIds.length} List${selectedListIds.length > 1 ? 's' : ''}` }}
				</button>
			</div>
		</div>
	</div>
</template>

<script>
import { useFormatters } from '@/composables/useFormatters'
import { getAssetURL } from '@/services/api'

const API_BASE = 'http://localhost:8080/api/v1'

export default {
	name: 'AddToListModal',
	props: {
		visible: {
			type: Boolean,
			default: false,
		},
		video: {
			type: Object,
			default: null,
		},
		videos: {
			type: Array,
			default: null,
		},
		isScraped: {
			type: Boolean,
			default: false,
		},
	},
	emits: ['close', 'added'],
	data() {
		return {
			lists: [],
			loading: false,
			saving: false,
			selectedListIds: [],
			videoInLists: [],
			showNewListForm: false,
			newListName: '',
		}
	},
	computed: {
		thumbnailUrl() {
			if (!this.video) return null
			if (this.video.thumbnail_path) {
				return getAssetURL(this.video.thumbnail_path)
			}
			if (this.video.thumbnail_url) {
				return this.video.thumbnail_url
			}
			return null
		},
	},
	watch: {
		visible(val) {
			if (val) {
				this.loadLists()
				this.loadVideoInLists()
				this.selectedListIds = []
				this.showNewListForm = false
				this.newListName = ''
			}
		},
	},
	created() {
		const formatters = useFormatters()
		this.formatDuration = formatters.formatDuration
	},
	methods: {
		async loadLists() {
			this.loading = true
			try {
				const response = await fetch(`${API_BASE}/lists`)
				const data = await response.json()
				if (data.success) {
					this.lists = data.data || []
				}
			} catch (error) {
				console.error('Failed to load lists:', error)
			} finally {
				this.loading = false
			}
		},

		async loadVideoInLists() {
			// Skip checking "already in list" for bulk mode - too many API calls
			if (this.videos && this.videos.length > 1) {
				this.videoInLists = []
				return
			}
			if (!this.video) return

			try {
				const videoId = this.isScraped ? this.video.id : this.video.id
				const param = this.isScraped ? 'scraped_video_id' : 'video_id'
				const response = await fetch(`${API_BASE}/lists/for-video?${param}=${videoId}`)
				const data = await response.json()
				if (data.success) {
					this.videoInLists = (data.data || []).map(l => l.id)
				}
			} catch (error) {
				console.error('Failed to check video lists:', error)
				this.videoInLists = []
			}
		},

		isInList(listId) {
			return this.videoInLists.includes(listId)
		},

		toggleListSelection(listId) {
			// Don't allow toggling if already in list
			if (this.isInList(listId)) return

			const index = this.selectedListIds.indexOf(listId)
			if (index > -1) {
				this.selectedListIds.splice(index, 1)
			} else {
				this.selectedListIds.push(listId)
			}
		},

		async addToSelectedLists() {
			const videosToAdd = this.videos || (this.video ? [this.video] : [])
			if (this.selectedListIds.length === 0 || videosToAdd.length === 0) return

			this.saving = true
			try {
				// Build the body based on whether videos are scraped or local
				let body
				if (this.isScraped) {
					body = { scraped_video_ids: videosToAdd.map(v => v.id) }
				} else {
					body = { video_ids: videosToAdd.map(v => v.id) }
				}

				for (const listId of this.selectedListIds) {
					const response = await fetch(`${API_BASE}/lists/${listId}/videos`, {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify(body),
					})
					const data = await response.json()
					if (!data.success) {
						throw new Error(data.message || 'Failed to add to list')
					}
				}

				const videoCount = videosToAdd.length
				const listCount = this.selectedListIds.length
				this.$emit('added', this.selectedListIds, videoCount)
				this.$toast?.success?.(`Added ${videoCount} video${videoCount > 1 ? 's' : ''} to ${listCount} list${listCount > 1 ? 's' : ''}`)
				this.$emit('close')
			} catch (error) {
				console.error('Failed to add to lists:', error)
				this.$toast?.error?.(error.message || 'Failed to add to lists')
			} finally {
				this.saving = false
			}
		},

		createNewList() {
			// Navigate to lists page
			this.$router?.push?.('/lists')
			this.$emit('close')
		},

		async createAndAddToList() {
			if (!this.newListName || !this.video) return

			this.saving = true
			try {
				// Create the list
				const createResponse = await fetch(`${API_BASE}/lists`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ name: this.newListName }),
				})
				const createData = await createResponse.json()
				if (!createData.success) {
					throw new Error(createData.message || 'Failed to create list')
				}

				const newList = createData.data

				// Add video to the new list
				const body = this.isScraped
					? { scraped_video_ids: [this.video.id] }
					: { video_ids: [this.video.id] }

				const addResponse = await fetch(`${API_BASE}/lists/${newList.id}/videos`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(body),
				})
				const addData = await addResponse.json()
				if (!addData.success) {
					throw new Error(addData.message || 'Failed to add to list')
				}

				this.$emit('added', [newList.id])
				this.$toast?.success?.(`Created "${this.newListName}" and added video`)
				this.$emit('close')
			} catch (error) {
				console.error('Failed to create and add:', error)
				this.$toast?.error?.(error.message || 'Failed to create list')
			} finally {
				this.saving = false
			}
		},
	},
}
</script>

<style scoped>
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
	z-index: 1100;
}

.add-to-list-modal {
	background: #1a1a2e;
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 16px;
	width: 100%;
	max-width: 450px;
	max-height: 80vh;
	display: flex;
	flex-direction: column;
	overflow: hidden;
}

.modal-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1.25rem 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.modal-header h4 {
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

.modal-body {
	padding: 1.5rem;
	overflow-y: auto;
	flex: 1;
}

/* Bulk Preview */
.bulk-preview {
	display: flex;
	align-items: center;
	gap: 1rem;
	padding: 1rem;
	background: linear-gradient(135deg, rgba(102, 126, 234, 0.2) 0%, rgba(118, 75, 162, 0.2) 100%);
	border-radius: 8px;
	margin-bottom: 1rem;
	border: 1px solid rgba(102, 126, 234, 0.3);
}

.bulk-icon {
	width: 48px;
	height: 48px;
	border-radius: 8px;
	background: rgba(102, 126, 234, 0.3);
	display: flex;
	align-items: center;
	justify-content: center;
	color: #667eea;
	font-size: 1.25rem;
}

.bulk-info {
	flex: 1;
}

.bulk-count {
	display: block;
	color: white;
	font-weight: 600;
	font-size: 1rem;
}

.bulk-hint {
	display: block;
	color: rgba(255, 255, 255, 0.6);
	font-size: 0.8rem;
}

/* Video Preview */
.video-preview {
	display: flex;
	align-items: center;
	gap: 1rem;
	padding: 0.75rem;
	background: rgba(255, 255, 255, 0.05);
	border-radius: 8px;
	margin-bottom: 1rem;
}

.preview-thumbnail {
	width: 80px;
	height: 45px;
	border-radius: 6px;
	object-fit: cover;
}

.preview-thumbnail-placeholder {
	width: 80px;
	height: 45px;
	border-radius: 6px;
	background: rgba(0, 0, 0, 0.3);
	display: flex;
	align-items: center;
	justify-content: center;
	color: rgba(255, 255, 255, 0.3);
}

.preview-info {
	flex: 1;
	overflow: hidden;
}

.preview-title {
	display: block;
	color: white;
	font-weight: 500;
	font-size: 0.875rem;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.preview-duration {
	display: block;
	color: rgba(255, 255, 255, 0.5);
	font-size: 0.75rem;
}

/* Loading */
.lists-loading {
	text-align: center;
	padding: 2rem;
	color: rgba(255, 255, 255, 0.6);
}

/* Lists Selection */
.lists-selection {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.list-option {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	padding: 0.75rem;
	border-radius: 8px;
	cursor: pointer;
	transition: all 0.2s;
	border: 1px solid transparent;
	background: rgba(255, 255, 255, 0.03);
}

.list-option:hover:not(.already-in) {
	background: rgba(255, 255, 255, 0.08);
}

.list-option.selected {
	background: rgba(0, 217, 255, 0.1);
	border-color: #00d9ff;
}

.list-option.already-in {
	opacity: 0.6;
	cursor: default;
}

.list-checkbox {
	color: rgba(255, 255, 255, 0.5);
	font-size: 1rem;
}

.list-option.selected .list-checkbox {
	color: #00d9ff;
}

.list-icon {
	width: 32px;
	height: 32px;
	border-radius: 6px;
	display: flex;
	align-items: center;
	justify-content: center;
	color: white;
	font-size: 0.75rem;
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
}

.list-count {
	display: block;
	color: rgba(255, 255, 255, 0.5);
	font-size: 0.75rem;
}

.already-badge {
	background: rgba(40, 167, 69, 0.2);
	color: #28a745;
	padding: 0.2rem 0.5rem;
	border-radius: 4px;
	font-size: 0.7rem;
	font-weight: 500;
}

/* No Lists */
.no-lists {
	text-align: center;
	padding: 2rem;
	color: rgba(255, 255, 255, 0.6);
}

.no-lists p {
	margin: 0.5rem 0 1rem;
}

/* Create New Section */
.create-new-section {
	margin-top: 1rem;
	padding-top: 1rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}

/* New List Form */
.new-list-form {
	margin-top: 1rem;
	padding: 1rem;
	background: rgba(255, 255, 255, 0.05);
	border-radius: 8px;
}

.new-list-form .form-control {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	color: white;
	margin-bottom: 0.75rem;
}

.new-list-form .form-control:focus {
	background: rgba(255, 255, 255, 0.08);
	border-color: #00d9ff;
	box-shadow: 0 0 0 0.25rem rgba(0, 217, 255, 0.25);
}

.new-list-form .form-control::placeholder {
	color: rgba(255, 255, 255, 0.4);
}

.form-actions {
	display: flex;
	justify-content: flex-end;
	gap: 0.5rem;
}

/* Modal Footer */
.modal-footer {
	display: flex;
	justify-content: flex-end;
	gap: 0.75rem;
	padding: 1.25rem 1.5rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.btn-primary {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	border: none;
}

.btn-primary:disabled {
	opacity: 0.5;
}

.btn-secondary {
	background: rgba(255, 255, 255, 0.1);
	border: 1px solid rgba(255, 255, 255, 0.2);
	color: white;
}

.btn-secondary:hover {
	background: rgba(255, 255, 255, 0.15);
}
</style>
