<template>
	<div class="tags-page">
		<div class="container-fluid py-4">
			<div class="page-header mb-4">
				<h1>
					<font-awesome-icon :icon="['fas', 'tags']" class="me-3" />
					Tag Management
					<span class="tag-count">({{ tags.length }})</span>
				</h1>
				<button class="btn btn-primary" @click="openCreateModal">
					<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
					Create Tag
				</button>
			</div>

			<div class="controls-bar mb-4">
				<div class="row g-3">
					<div class="col-md-6">
						<div class="search-box">
							<font-awesome-icon :icon="['fas', 'search']" class="search-icon" />
							<input v-model="searchQuery" type="text" class="form-control" placeholder="Search tags..." />
						</div>
					</div>
					<div class="col-md-3">
						<select v-model="sortBy" class="form-select">
							<option value="name">Sort by Name</option>
							<option value="count">Sort by Usage Count</option>
							<option value="created">Sort by Date Created</option>
						</select>
					</div>
					<div class="col-md-3">
						<button class="btn btn-outline-danger w-100" :disabled="selectedTags.length === 0" @click="bulkDelete">
							<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
							Delete Selected ({{ selectedTags.length }})
						</button>
					</div>
				</div>
			</div>

			<div class="tags-table-container">
				<table class="table table-hover">
					<thead>
						<tr>
							<th style="width: 40px">
								<input type="checkbox" class="form-check-input" :checked="allSelected" @change="toggleSelectAll" />
							</th>
							<th>Name</th>
							<th style="width: 150px">Color</th>
							<th style="width: 150px">Icon</th>
							<th style="width: 120px">Videos</th>
							<th style="width: 200px">Actions</th>
						</tr>
					</thead>
					<tbody>
						<tr v-if="filteredTags.length === 0">
							<td colspan="6" class="text-center py-4">
								<p>No tags found</p>
								<button class="btn btn-primary btn-sm" @click="openCreateModal">Create Your First Tag</button>
							</td>
						</tr>
						<tr v-for="tag in filteredTags" :key="tag.id">
							<td>
								<input type="checkbox" class="form-check-input" :checked="selectedTags.includes(tag.id)" @change="toggleSelect(tag.id)" />
							</td>
							<td>{{ tag.name }}</td>
							<td>
								<span class="color-badge" :style="{ backgroundColor: tag.color || '#6c757d' }">
									{{ tag.color || '#6c757d' }}
								</span>
							</td>
							<td>
								<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" class="me-2" />
								<span>{{ tag.icon || 'None' }}</span>
							</td>
							<td>
								<span class="badge bg-secondary">{{ tag.video_count || 0 }}</span>
							</td>
							<td>
								<button class="btn btn-sm btn-outline-primary me-1" @click="startEdit(tag)">
									<font-awesome-icon :icon="['fas', 'edit']" />
								</button>
								<button class="btn btn-sm btn-outline-info me-1" @click="openMergeModal(tag)">
									<font-awesome-icon :icon="['fas', 'code-branch']" />
								</button>
								<button class="btn btn-sm btn-outline-danger" @click="deleteTag(tag)">
									<font-awesome-icon :icon="['fas', 'trash']" />
								</button>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<!-- Create Modal -->
		<div v-if="showCreateModal" class="modal show d-block" @click.self="closeCreateModal">
			<div class="modal-dialog modal-dialog-centered">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">Create New Tag</h5>
						<button type="button" class="btn-close" @click="closeCreateModal"></button>
					</div>
					<div class="modal-body">
						<div class="mb-3">
							<label class="form-label">Tag Name</label>
							<input v-model="createForm.name" type="text" class="form-control" required />
						</div>
						<div class="mb-3">
							<label class="form-label">Color</label>
							<input v-model="createForm.color" type="color" class="form-control" />
						</div>
						<div class="mb-3">
							<label class="form-label">Icon</label>
							<select v-model="createForm.icon" class="form-select">
								<option value="">No Icon</option>
								<option v-for="icon in commonIcons" :key="icon" :value="icon">{{ icon }}</option>
							</select>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="closeCreateModal">Cancel</button>
						<button type="button" class="btn btn-primary" @click="createTag">Create</button>
					</div>
				</div>
			</div>
		</div>

		<div v-if="showCreateModal || showMergeModal" class="modal-backdrop show"></div>
	</div>
</template>

<script>
import { tagsAPI } from '@/services/api'

export default {
	name: 'TagsPage',
	data() {
		return {
			tags: [],
			searchQuery: '',
			sortBy: 'name',
			selectedTags: [],
			showCreateModal: false,
			showMergeModal: false,
			createForm: { name: '', color: '#6c757d', icon: '' },
			commonIcons: ['tag', 'star', 'heart', 'fire', 'bolt', 'crown', 'gem', 'award', 'bookmark', 'flag'],
		}
	},
	computed: {
		filteredTags() {
			let result = this.tags
			if (this.searchQuery) {
				result = result.filter((tag) => tag.name.toLowerCase().includes(this.searchQuery.toLowerCase()))
			}
			return [...result].sort((a, b) => {
				if (this.sortBy === 'name') return a.name.localeCompare(b.name)
				if (this.sortBy === 'count') return (b.video_count || 0) - (a.video_count || 0)
				return new Date(b.created_at) - new Date(a.created_at)
			})
		},
		allSelected() {
			return this.filteredTags.length > 0 && this.selectedTags.length === this.filteredTags.length
		},
	},
	methods: {
		async loadTags() {
			try {
				const response = await tagsAPI.getAll()
				this.tags = response.data || []
			} catch (err) {
				console.error('Failed to load tags:', err)
			}
		},
		toggleSelect(tagId) {
			const index = this.selectedTags.indexOf(tagId)
			if (index > -1) this.selectedTags.splice(index, 1)
			else this.selectedTags.push(tagId)
		},
		toggleSelectAll() {
			this.selectedTags = this.allSelected ? [] : this.filteredTags.map((tag) => tag.id)
		},
		openCreateModal() {
			this.showCreateModal = true
		},
		closeCreateModal() {
			this.showCreateModal = false
		},
		async createTag() {
			try {
				await tagsAPI.create(this.createForm)
				await this.loadTags()
				this.closeCreateModal()
				this.$toast.success('Tag Created', `Tag "${this.createForm.name}" created`)
			} catch (err) {
				this.$toast.error('Create Failed', 'Failed to create tag')
			}
		},
		async deleteTag(tag) {
			if (!confirm(`Delete tag "${tag.name}"?`)) return
			try {
				await tagsAPI.delete(tag.id)
				await this.loadTags()
				this.$toast.success('Tag Deleted', 'Tag deleted successfully')
			} catch (err) {
				this.$toast.error('Delete Failed', 'Failed to delete tag')
			}
		},
		async bulkDelete() {
			if (!confirm(`Delete ${this.selectedTags.length} tags?`)) return
			try {
				await Promise.all(this.selectedTags.map((id) => tagsAPI.delete(id)))
				await this.loadTags()
				this.selectedTags = []
				this.$toast.success('Tags Deleted', 'Tags deleted successfully')
			} catch (err) {
				this.$toast.error('Delete Failed', 'Failed to delete tags')
			}
		},
		startEdit(tag) {
			// Will implement inline editing
			console.log('Edit tag:', tag)
		},
		openMergeModal(tag) {
			// Will implement merge functionality
			console.log('Merge tag:', tag)
		},
	},
	mounted() {
		this.loadTags()
	},
}
</script>

<style scoped>
.tags-page {
	min-height: 100vh;
	background: #0f0c29;
	color: #e0e0e0;
}

.page-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.page-header h1 {
	font-size: 2rem;
	font-weight: 600;
	color: #00d9ff;
	margin: 0;
}

.tag-count {
	font-size: 1.2rem;
	color: #a0a0a0;
}

.controls-bar {
	background: #16213e;
	padding: 1.5rem;
	border-radius: 0.5rem;
	border: 1px solid #2a3f5f;
}

.search-box {
	position: relative;
}

.search-icon {
	position: absolute;
	left: 1rem;
	top: 50%;
	transform: translateY(-50%);
	color: #00d9ff;
}

.search-box input {
	padding-left: 2.5rem;
	background: #1a2942;
	border: 1px solid #2a3f5f;
	color: #e0e0e0;
}

.search-box input:focus {
	background: #1a2942;
	border-color: #00d9ff;
	color: #e0e0e0;
	box-shadow: 0 0 0 0.2rem rgba(0, 217, 255, 0.25);
}

.search-box input::placeholder {
	color: #6c757d;
}

.form-select {
	background: #1a2942;
	border: 1px solid #2a3f5f;
	color: #e0e0e0;
}

.form-select:focus {
	background: #1a2942;
	border-color: #00d9ff;
	color: #e0e0e0;
	box-shadow: 0 0 0 0.2rem rgba(0, 217, 255, 0.25);
}

.tags-table-container {
	background-color: rgba(255, 255, 255, 0.05);
	border-radius: 0.5rem;
	overflow: hidden;
	border: 1px solid #2a3f5f;
}

.table thead {
	background: #1a2942;
	border-bottom: 2px solid #2a3f5f;
}

.table thead th {
	border: none;
	color: #00d9ff;
	font-weight: 600;
	padding: 1rem;
	background-color: rgba(255, 255, 255, 0.05);
}

.table tbody tr {
	border-bottom: 1px solid #2a3f5f;
}

.table tbody tr:hover {
	background: #1a2942;
}

.table tbody td {
	background-color: rgba(255, 255, 255, 0.05);

	border: none;
	padding: 1rem;
	vertical-align: middle;
}

.form-check-input {
	background-color: #1a2942;
	border-color: #2a3f5f;
}

.form-check-input:checked {
	background-color: #00d9ff;
	border-color: #00d9ff;
}

.form-check-input:focus {
	border-color: #00d9ff;
	box-shadow: 0 0 0 0.2rem rgba(0, 217, 255, 0.25);
}

.color-badge {
	display: inline-block;
	padding: 0.25rem 0.75rem;
	border-radius: 0.375rem;
	color: #fff;
	font-family: monospace;
	font-size: 0.875rem;
}

.badge {
	background: #1a2942;
	color: #e0e0e0;
}

/* Modals */
.modal-content {
	background: #16213e;
	border: 1px solid #2a3f5f;
	color: #e0e0e0;
}

.modal-header {
	border-bottom-color: #2a3f5f;
}

.modal-title {
	color: #00d9ff;
}

.modal-footer {
	border-top-color: #2a3f5f;
}

.modal-body .form-label {
	color: #b0b0b0;
	font-weight: 500;
}

.form-control,
.form-select {
	background: #1a2942;
	border: 1px solid #2a3f5f;
	color: #e0e0e0;
}

.form-control:focus,
.form-select:focus {
	background: #1a2942;
	border-color: #00d9ff;
	color: #e0e0e0;
	box-shadow: 0 0 0 0.2rem rgba(0, 217, 255, 0.25);
}

.btn-close {
	filter: invert(1);
	opacity: 0.8;
}

.btn-close:hover {
	opacity: 1;
}

/* Empty state */
.text-center.py-4 p {
	color: #b0b0b0;
	margin-bottom: 1rem;
}
</style>
