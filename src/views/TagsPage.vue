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
@import '@/styles/pages/tags_page.css';
</style>
