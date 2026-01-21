<template>
	<div class="tags-page">
		<div class="container-fluid py-4">
			<!-- Header -->
			<div class="page-header mb-4">
				<h1>
					<font-awesome-icon :icon="['fas', 'tags']" class="me-3" />
					Tag Management
					<span class="tag-count">({{ tags.length }})</span>
				</h1>
				<div class="header-actions">
					<button class="btn btn-success me-2" @click="openBatchCreateModal">
						<font-awesome-icon :icon="['fas', 'layer-group']" class="me-2" />
						Batch Create
					</button>
					<button class="btn btn-primary" @click="openCreateModal">
						<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
						Create Tag
					</button>
				</div>
			</div>

			<!-- Stats Cards -->
			<div class="stats-cards mb-4 d-flex justify-content-evenly">
				<div class="stat-card">
					<div class="stat-icon regular">
						<font-awesome-icon :icon="['fas', 'user']" />
					</div>
					<div class="stat-info">
						<div class="stat-value">{{ tagsByCategory.regular }}</div>
						<div class="stat-label">Regular Tags</div>
					</div>
				</div>
				<div class="stat-card">
					<div class="stat-icon zoo">
						<font-awesome-icon :icon="['fas', 'dog']" />
					</div>
					<div class="stat-info">
						<div class="stat-value">{{ tagsByCategory.zoo }}</div>
						<div class="stat-label">Zoo Tags</div>
					</div>
				</div>
				<div class="stat-card">
					<div class="stat-icon threed">
						<font-awesome-icon :icon="['fas', 'cube']" />
					</div>
					<div class="stat-info">
						<div class="stat-value">{{ tagsByCategory['3d'] }}</div>
						<div class="stat-label">3D Tags</div>
					</div>
				</div>
				<div class="stat-card">
					<div class="stat-icon total">
						<font-awesome-icon :icon="['fas', 'tags']" />
					</div>
					<div class="stat-info">
						<div class="stat-value">{{ totalVideoTags }}</div>
						<div class="stat-label">Total Video Tags</div>
					</div>
				</div>
			</div>

			<!-- Controls Bar -->
			<div class="controls-bar mb-4">
				<div class="row g-3">
					<!-- Search -->
					<div class="col-md-4">
						<SearchBox v-model="searchQuery" placeholder="Search tags..." />
					</div>

					<!-- Category Filter -->
					<div class="col-md-2">
						<select v-model="categoryFilter" class="form-select">
							<option value="">All Categories</option>
							<option value="regular">🎬 Regular</option>
							<option value="zoo">🐾 Zoo</option>
							<option value="3d">🎨 3D</option>
						</select>
					</div>

					<!-- Sort -->
					<div class="col-md-3">
						<select v-model="sortBy" class="form-select">
							<option value="name">Sort by Name</option>
							<option value="count">Sort by Usage Count</option>
							<option value="category">Sort by Category</option>
							<option value="created">Sort by Date Created</option>
						</select>
					</div>

					<!-- Bulk Actions -->
					<div class="col-md-3">
						<button class="btn btn-outline-danger w-100" :disabled="selectedCount === 0" @click="bulkDelete">
							<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
							Delete Selected ({{ selectedCount }})
						</button>
					</div>
				</div>
			</div>

			<!-- Tags Table -->
			<div class="tags-table-container">
				<table class="table table-hover">
					<thead>
						<tr>
							<th style="width: 40px">
								<input type="checkbox" class="form-check-input" :checked="allSelected" @change="toggleSelectAll" />
							</th>
							<th @click="sortBy = 'name'" class="sortable">
								Name
								<font-awesome-icon v-if="sortBy === 'name'" :icon="['fas', 'sort']" class="ms-1" />
							</th>
							<th @click="sortBy = 'category'" class="sortable" style="width: 150px">
								Category
								<font-awesome-icon v-if="sortBy === 'category'" :icon="['fas', 'sort']" class="ms-1" />
							</th>
							<th style="width: 150px">Color</th>
							<th style="width: 120px">Icon</th>
							<th @click="sortBy = 'count'" class="sortable" style="width: 100px">
								Videos
								<font-awesome-icon v-if="sortBy === 'count'" :icon="['fas', 'sort']" class="ms-1" />
							</th>
							<th style="width: 200px">Actions</th>
						</tr>
					</thead>
					<tbody>
						<tr v-if="filteredTags.length === 0">
							<td colspan="7" class="text-center py-4">
								<p>No tags found</p>
								<button class="btn btn-primary btn-sm" @click="openCreateModal">Create Your First Tag</button>
							</td>
						</tr>
						<tr v-for="tag in filteredTags" :key="tag.id" :class="{ selected: isSelected(tag.id) }">
							<td>
								<input type="checkbox" class="form-check-input" :checked="isSelected(tag.id)" @change="toggleSelection(tag.id)" />
							</td>
							<td class="text-light fw-bold">{{ tag.name }}</td>
							<td>
								<span :class="['category-badge', 'category-' + (tag.category || 'regular')]">
									<font-awesome-icon :icon="['fas', getCategoryIcon(tag.category)]" />
									{{ getCategoryName(tag.category) }}
								</span>
							</td>
							<td>
								<div class="d-flex align-items-center gap-2">
									<span class="color-preview" :style="{ backgroundColor: tag.color || '#6c757d' }"></span>
									<span class="color-code">{{ tag.color || '#6c757d' }}</span>
								</div>
							</td>
							<td>
								<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" class="me-2" size="lg" />
								<span class="text-muted">{{ tag.icon || 'None' }}</span>
							</td>
							<td>
								<span class="badge bg-secondary">{{ tag.video_count || 0 }}</span>
							</td>
							<td>
								<div class="btn-group btn-group-sm">
									<button class="btn btn-outline-primary" @click="startEdit(tag)" title="Edit">
										<font-awesome-icon :icon="['fas', 'edit']" />
									</button>
									<button class="btn btn-outline-info" @click="openMergeModal(tag)" title="Merge">
										<font-awesome-icon :icon="['fas', 'code-branch']" />
									</button>
									<button class="btn btn-outline-danger" @click="deleteTag(tag)" title="Delete">
										<font-awesome-icon :icon="['fas', 'trash']" />
									</button>
								</div>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<!-- Create/Edit Modal -->
		<div v-if="showCreateModal" class="modal show d-block" @click.self="closeCreateModal">
			<div class="modal-dialog modal-dialog-centered modal-lg">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							<font-awesome-icon :icon="['fas', editingTag ? 'edit' : 'plus']" class="me-2" />
							{{ editingTag ? 'Edit Tag' : 'Create New Tag' }}
						</h5>
						<button type="button" class="btn-close" @click="closeCreateModal"></button>
					</div>
					<div class="modal-body">
						<!-- Category Selection -->
						<div class="mb-4">
							<label class="form-label fw-bold">
								<font-awesome-icon :icon="['fas', 'layer-group']" class="me-2" />
								Category
							</label>
							<CategorySelector v-model="createForm.category" />
							<small class="form-text text-muted">Tags will only appear for content matching this category</small>
						</div>

						<!-- Tag Name -->
						<div class="mb-3">
							<label class="form-label">Tag Name</label>
							<input v-model="createForm.name" type="text" class="form-control" placeholder="e.g., Amateur, Blonde, POV" required />
						</div>

						<!-- Quick Templates (if not editing) -->
						<div v-if="!editingTag && createForm.category" class="mb-4">
							<label class="form-label">Quick Templates</label>
							<div class="template-chips">
								<span v-for="template in commonTagTemplates[createForm.category]" :key="template.name" class="template-chip" @click="applyTemplate(template)">
									{{ template.name }}
								</span>
							</div>
						</div>

						<div class="row">
							<!-- Color -->
							<div class="col-md-6 mb-3">
								<label class="form-label">Color</label>
								<div class="d-flex gap-2">
									<input v-model="createForm.color" type="color" class="form-control form-control-color" style="width: 60px" />
									<input v-model="createForm.color" type="text" class="form-control" placeholder="#6c757d" />
								</div>
							</div>

							<!-- Icon -->
							<div class="col-md-6 mb-3">
								<label class="form-label">Icon (optional)</label>
								<select v-model="createForm.icon" class="form-select">
									<option value="">No Icon</option>
									<option v-for="icon in commonIcons" :key="icon" :value="icon">{{ icon }}</option>
								</select>
								<div v-if="createForm.icon" class="mt-2">
									<font-awesome-icon :icon="['fas', createForm.icon]" size="2x" />
								</div>
							</div>
						</div>

						<!-- Preview -->
						<div class="tag-preview">
							<label class="form-label">Preview</label>
							<div class="preview-box">
								<span :class="['category-badge', 'category-' + (createForm.category || 'regular')]" :style="{ backgroundColor: createForm.color }">
									<font-awesome-icon v-if="createForm.icon" :icon="['fas', createForm.icon]" />
									{{ createForm.name || 'Tag Name' }}
								</span>
							</div>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="closeCreateModal">Cancel</button>
						<button type="button" class="btn btn-primary" :disabled="!createForm.category || !createForm.name" @click="editingTag ? updateTag() : createTag()">
							<font-awesome-icon :icon="['fas', editingTag ? 'save' : 'plus']" class="me-2" />
							{{ editingTag ? 'Update Tag' : 'Create Tag' }}
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Batch Create Modal -->
		<div v-if="showBatchCreateModal" class="modal show d-block" @click.self="closeBatchCreateModal">
			<div class="modal-dialog modal-dialog-centered modal-xl">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							<font-awesome-icon :icon="['fas', 'layer-group']" class="me-2" />
							Batch Create Tags
						</h5>
						<button type="button" class="btn-close" @click="closeBatchCreateModal"></button>
					</div>
					<div class="modal-body">
						<!-- Category Selection -->
						<div class="mb-4">
							<label class="form-label fw-bold">Select Category for All Tags</label>
							<CategorySelector v-model="batchForm.category" />
						</div>

						<!-- Quick Presets -->
						<div v-if="batchForm.category" class="mb-4">
							<label class="form-label fw-bold">Quick Presets</label>
							<div class="preset-buttons">
								<button class="btn btn-outline-primary btn-sm me-2 mb-2" @click="loadPreset('common')">
									<font-awesome-icon :icon="['fas', 'star']" class="me-1" />
									Common Tags
								</button>
								<button class="btn btn-outline-secondary btn-sm me-2 mb-2" @click="loadPreset('appearance')">
									<font-awesome-icon :icon="['fas', 'user-circle']" class="me-1" />
									Appearance
								</button>
								<button class="btn btn-outline-info btn-sm me-2 mb-2" @click="loadPreset('actions')">
									<font-awesome-icon :icon="['fas', 'film']" class="me-1" />
									Actions
								</button>
								<button class="btn btn-outline-warning btn-sm me-2 mb-2" @click="loadPreset('positions')">
									<font-awesome-icon :icon="['fas', 'arrows-alt']" class="me-1" />
									Positions
								</button>
							</div>
						</div>

						<!-- Tag Names Input -->
						<div class="mb-3">
							<label class="form-label fw-bold">Tag Names (one per line)</label>
							<textarea
								v-model="batchForm.tagNames"
								class="form-control"
								rows="10"
								placeholder="Enter tag names, one per line:
Amateur
Blonde
Brunette
POV
Anal"
							></textarea>
							<small class="form-text text-muted">
								Parsed tags: <strong>{{ parsedBatchTags.length }}</strong>
							</small>
						</div>

						<!-- Options -->
						<div class="row">
							<div class="col-md-6 mb-3">
								<label class="form-label">Default Color</label>
								<input v-model="batchForm.defaultColor" type="color" class="form-control form-control-color" />
							</div>
							<div class="col-md-6 mb-3">
								<div class="form-check mt-4">
									<input v-model="batchForm.autoColor" type="checkbox" class="form-check-input" id="autoColor" />
									<label class="form-check-label" for="autoColor">Auto-generate colors</label>
								</div>
							</div>
						</div>

						<!-- Preview -->
						<div v-if="parsedBatchTags.length > 0" class="batch-preview">
							<label class="form-label fw-bold">Preview ({{ parsedBatchTags.length }} tags)</label>
							<div class="preview-tags">
								<span
									v-for="(tagName, index) in parsedBatchTags.slice(0, 20)"
									:key="index"
									:class="['category-badge', 'category-' + batchForm.category]"
									:style="{ backgroundColor: batchForm.autoColor ? getAutoColor(index) : batchForm.defaultColor }"
								>
									{{ tagName }}
								</span>
								<span v-if="parsedBatchTags.length > 20" class="text-muted">...and {{ parsedBatchTags.length - 20 }} more</span>
							</div>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="closeBatchCreateModal">Cancel</button>
						<button type="button" class="btn btn-success" :disabled="!batchForm.category || parsedBatchTags.length === 0" @click="batchCreateTags">
							<font-awesome-icon :icon="['fas', 'layer-group']" class="me-2" />
							Create {{ parsedBatchTags.length }} Tags
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Merge Modal -->
		<div v-if="showMergeModal" class="modal show d-block" @click.self="closeMergeModal">
			<div class="modal-dialog modal-dialog-centered">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							<font-awesome-icon :icon="['fas', 'code-branch']" class="me-2" />
							Merge Tags
						</h5>
						<button type="button" class="btn-close" @click="closeMergeModal"></button>
					</div>
					<div class="modal-body">
						<p>
							Merge <strong>{{ mergeSourceTag?.name }}</strong> into:
						</p>
						<div class="mb-3">
							<label class="form-label">Target Tag</label>
							<select v-model="mergeTargetId" class="form-select">
								<option value="">Select target tag...</option>
								<option v-for="tag in availableTargetTags" :key="tag.id" :value="tag.id">{{ tag.name }} ({{ tag.category }})</option>
							</select>
						</div>
						<div class="alert alert-warning">
							<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="me-2" />
							All videos tagged with <strong>{{ mergeSourceTag?.name }}</strong> will be re-tagged with the target tag, and the source tag will be deleted.
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="closeMergeModal">Cancel</button>
						<button type="button" class="btn btn-warning" :disabled="!mergeTargetId" @click="performMerge">
							<font-awesome-icon :icon="['fas', 'code-branch']" class="me-2" />
							Merge Tags
						</button>
					</div>
				</div>
			</div>
		</div>

		<div v-if="showCreateModal || showMergeModal || showBatchCreateModal" class="modal-backdrop show"></div>

		<!-- Delete Confirmation Modal -->
		<DeleteConfirmationModal
			:visible="deleteModal.show"
			:title="deleteModal.isBulk ? 'Confirm Bulk Delete' : 'Confirm Delete'"
			message="Are you sure you want to delete"
			:itemName="deleteModal.isBulk ? `${selectedCount} selected tags` : deleteModal.tag?.name"
			warningMessage="This will remove the tag(s) from all videos."
			@confirm="deleteModal.isBulk ? confirmBulkDelete() : confirmDeleteTag()"
			@cancel="deleteModal.show = false"
		/>
	</div>
</template>

<script>
import { tagsAPI } from '@/services/api'
import { DeleteConfirmationModal, CategorySelector, SearchBox } from '@/components/shared'
import { useTableSelectionOptionsAPI } from '@/composables/useTableSelection'

export default {
	name: 'TagsPage',
	components: {
		DeleteConfirmationModal,
		CategorySelector,
		SearchBox,
	},
	data() {
		return {
			tags: [],
			searchQuery: '',
			categoryFilter: '',
			sortBy: 'name',
			...useTableSelectionOptionsAPI().data(),
			showCreateModal: false,
			showBatchCreateModal: false,
			showMergeModal: false,
			editingTag: null,
			mergeSourceTag: null,
			mergeTargetId: null,
			deleteModal: {
				show: false,
				tag: null,
				isBulk: false,
			},
			createForm: {
				name: '',
				color: '#6c757d',
				icon: '',
				category: '',
			},
			batchForm: {
				category: '',
				tagNames: '',
				defaultColor: '#6c757d',
				autoColor: true,
			},
			commonIcons: [
				'star',
				'heart',
				'fire',
				'bolt',
				'crown',
				'gem',
				'award',
				'trophy',
				'medal',
				'ribbon',
				'circle',
				'square',
				'triangle-exclamation',
				'flag',
				'bookmark',
				'thumbs-up',
				'eye',
				'camera',
				'video',
				'film',
				'user',
				'users',
				'user-group',
				'person',
				'child',
				'baby',
				'graduation-cap',
				'briefcase',
				'house',
				'building',
				'bed',
				'couch',
			],
			commonTagTemplates: {
				regular: [
					{ name: 'Amateur', color: '#3498db', icon: 'camera' },
					{ name: 'Professional', color: '#e74c3c', icon: 'award' },
					{ name: 'POV', color: '#9b59b6', icon: 'eye' },
					{ name: 'Blonde', color: '#f39c12', icon: 'user' },
					{ name: 'Brunette', color: '#795548', icon: 'user' },
					{ name: 'Redhead', color: '#e74c3c', icon: 'user' },
					{ name: 'Anal', color: '#8e44ad', icon: '' },
					{ name: 'Oral', color: '#16a085', icon: '' },
					{ name: 'Creampie', color: '#e67e22', icon: '' },
					{ name: 'Threesome', color: '#c0392b', icon: 'users' },
				],
				zoo: [
					{ name: 'Horse', color: '#795548', icon: '' },
					{ name: 'Dog', color: '#ff9800', icon: '' },
					{ name: 'Amateur', color: '#3498db', icon: 'camera' },
					{ name: 'Professional', color: '#e74c3c', icon: 'award' },
				],
				'3d': [
					{ name: 'Animation', color: '#9c27b0', icon: 'film' },
					{ name: 'SFM', color: '#3f51b5', icon: 'cube' },
					{ name: 'Blender', color: '#ff9800', icon: 'cube' },
					{ name: 'Loop', color: '#00bcd4', icon: '' },
					{ name: 'POV', color: '#9b59b6', icon: 'eye' },
				],
			},
			tagPresets: {
				common: `Amateur
Professional
HD
4K
POV
Homemade
Outdoor
Indoor`,
				appearance: `Blonde
Brunette
Redhead
Asian
Latina
Ebony
BBW
Petite
Teen
MILF
Mature`,
				actions: `Anal
Oral
Vaginal
Creampie
Facial
Cumshot
Squirting
Fingering
Masturbation`,
				positions: `Missionary
Doggy Style
Cowgirl
Reverse Cowgirl
Standing
Sitting
Spooning`,
			},
		}
	},
	computed: {
		...useTableSelectionOptionsAPI().computed(function () {
			return this.filteredTags
		}),
		filteredTags() {
			let result = this.tags

			// Category filter
			if (this.categoryFilter) {
				result = result.filter((tag) => tag.category === this.categoryFilter)
			}

			// Search filter
			if (this.searchQuery) {
				result = result.filter((tag) => tag.name.toLowerCase().includes(this.searchQuery.toLowerCase()))
			}

			// Sort
			return [...result].sort((a, b) => {
				if (this.sortBy === 'name') return a.name.localeCompare(b.name)
				if (this.sortBy === 'count') return (b.video_count || 0) - (a.video_count || 0)
				if (this.sortBy === 'category') return (a.category || 'regular').localeCompare(b.category || 'regular')
				return new Date(b.created_at) - new Date(a.created_at)
			})
		},
		availableTargetTags() {
			if (!this.mergeSourceTag) return this.tags
			// Only show tags from the same category
			return this.tags.filter((tag) => tag.id !== this.mergeSourceTag.id && tag.category === this.mergeSourceTag.category)
		},
		tagsByCategory() {
			return {
				regular: this.tags.filter((t) => !t.category || t.category === 'regular').length,
				zoo: this.tags.filter((t) => t.category === 'zoo').length,
				'3d': this.tags.filter((t) => t.category === '3d').length,
			}
		},
		totalVideoTags() {
			return this.tags.reduce((sum, tag) => sum + (tag.video_count || 0), 0)
		},
		parsedBatchTags() {
			if (!this.batchForm.tagNames) return []
			return this.batchForm.tagNames
				.split('\n')
				.map((line) => line.trim())
				.filter((line) => line.length > 0)
		},
	},
	mounted() {
		this.loadTags()
	},
	methods: {
		...useTableSelectionOptionsAPI().methods(function () {
			return this.filteredTags
		}),
		async loadTags() {
			try {
				const response = await tagsAPI.getAll()
				this.tags = response || []
			} catch (err) {
				console.error('Failed to load tags:', err)
			}
		},
		openCreateModal() {
			this.editingTag = null
			this.createForm = { name: '', color: '#6c757d', icon: '', category: '' }
			this.showCreateModal = true
		},
		closeCreateModal() {
			this.showCreateModal = false
			this.editingTag = null
		},
		openBatchCreateModal() {
			this.batchForm = {
				category: '',
				tagNames: '',
				defaultColor: '#6c757d',
				autoColor: true,
			}
			this.showBatchCreateModal = true
		},
		closeBatchCreateModal() {
			this.showBatchCreateModal = false
		},
		applyTemplate(template) {
			this.createForm.name = template.name
			this.createForm.color = template.color
			this.createForm.icon = template.icon
		},
		loadPreset(presetName) {
			this.batchForm.tagNames = this.tagPresets[presetName]
		},
		getAutoColor(index) {
			const colors = ['#e74c3c', '#3498db', '#2ecc71', '#f39c12', '#9b59b6', '#1abc9c', '#e67e22', '#95a5a6', '#34495e', '#16a085']
			return colors[index % colors.length]
		},
		async createTag() {
			if (!this.createForm.category || !this.createForm.name) return

			try {
				await tagsAPI.create(this.createForm)
				await this.loadTags()
				this.closeCreateModal()
			} catch (err) {
				console.error('Failed to create tag:', err)
				alert('Failed to create tag: ' + (err.response?.data?.error || err.message))
			}
		},
		async batchCreateTags() {
			if (!this.batchForm.category || this.parsedBatchTags.length === 0) return

			try {
				const promises = this.parsedBatchTags.map((tagName, index) => {
					const color = this.batchForm.autoColor ? this.getAutoColor(index) : this.batchForm.defaultColor
					return tagsAPI.create({
						name: tagName,
						category: this.batchForm.category,
						color: color,
						icon: '',
					})
				})

				await Promise.all(promises)
				await this.loadTags()
				this.closeBatchCreateModal()
				alert(`Successfully created ${this.parsedBatchTags.length} tags!`)
			} catch (err) {
				console.error('Failed to batch create tags:', err)
				alert('Some tags may have failed to create. Check console for details.')
			}
		},
		startEdit(tag) {
			this.editingTag = tag
			this.createForm = {
				name: tag.name,
				color: tag.color || '#6c757d',
				icon: tag.icon || '',
				category: tag.category || 'regular',
			}
			this.showCreateModal = true
		},
		async updateTag() {
			try {
				await tagsAPI.update(this.editingTag.id, this.createForm)
				await this.loadTags()
				this.closeCreateModal()
			} catch (err) {
				console.error('Failed to update tag:', err)
			}
		},
		deleteTag(tag) {
			this.deleteModal.tag = tag
			this.deleteModal.isBulk = false
			this.deleteModal.show = true
		},
		async confirmDeleteTag() {
			try {
				await tagsAPI.delete(this.deleteModal.tag.id)
				await this.loadTags()
			} catch (err) {
				console.error('Failed to delete tag:', err)
			}
			this.deleteModal.show = false
			this.deleteModal.tag = null
		},
		bulkDelete() {
			this.deleteModal.isBulk = true
			this.deleteModal.show = true
		},
		async confirmBulkDelete() {
			try {
				await Promise.all(this.selectedItems.map((id) => tagsAPI.delete(id)))
				this.clearSelection()
				await this.loadTags()
			} catch (err) {
				console.error('Failed to delete tags:', err)
			}
			this.deleteModal.show = false
			this.deleteModal.isBulk = false
		},
		openMergeModal(tag) {
			this.mergeSourceTag = tag
			this.mergeTargetId = null
			this.showMergeModal = true
		},
		closeMergeModal() {
			this.showMergeModal = false
			this.mergeSourceTag = null
		},
		async performMerge() {
			try {
				await tagsAPI.merge({
					source_tag_ids: [this.mergeSourceTag.id],
					target_tag_id: this.mergeTargetId,
				})
				await this.loadTags()
				this.closeMergeModal()
			} catch (err) {
				console.error('Merge Failed:', err)
			}
		},
		getCategoryIcon(category) {
			const icons = {
				regular: 'user',
				zoo: 'dog',
				'3d': 'cube',
			}
			return icons[category] || 'user'
		},
		getCategoryName(category) {
			const names = {
				regular: 'Regular',
				zoo: 'Zoo',
				'3d': '3D',
			}
			return names[category] || 'Regular'
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/tags_page.css';
</style>
