<template>
	<div class="libraries-page">
		<div class="container-fluid">
			<div class="row">
				<div class="col d-flex justify-content-between my-3">
					<span class="lead">Manage your video collections and directories</span>
					<button class="btn btn-primary" @click="showCreateModal = true">
						<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
						Add Library
					</button>
				</div>
			</div>

			<!-- Loading State -->
			<div v-if="loading" class="text-center py-5">
				<font-awesome-icon :icon="['fas', 'spinner']" spin size="3x" class="text-primary" />
				<p class="mt-3">Loading libraries...</p>
			</div>

			<!-- Libraries List -->
			<div v-else-if="libraries.length > 0" class="row g-4">
				<div v-for="library in libraries" :key="library.id" class="col-md-6 col-lg-4">
					<div class="library-card card h-100 text-primary">
						<div class="card-body">
							<div class="d-flex justify-content-between align-items-start mb-3">
								<h5 class="card-title mb-0">
									<font-awesome-icon :icon="['fas', 'folder']" class="me-2" />
									{{ library.name }}
									<span v-if="library.primary" class="badge bg-primary ms-2">Primary</span>
								</h5>
								<div class="btn-group btn-group-sm">
									<button class="btn btn-outline-secondary me-2 text-warning" @click="editLibrary(library)" title="Edit">
										<font-awesome-icon :icon="['fas', 'edit']" />
									</button>
									<button class="btn btn-outline-danger" @click="confirmDelete(library)" title="Delete">
										<font-awesome-icon :icon="['fas', 'trash']" />
									</button>
								</div>
							</div>

							<div class="library-path mb-3">
								<small class="">
									<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
									{{ library.path }}
								</small>
							</div>

							<div class="library-stats">
								<div class="stat-item">
									<font-awesome-icon :icon="['fas', 'video']" class="me-1" />
									<span>{{ library.video_count || 0 }} videos</span>
								</div>
							</div>
						</div>
						<div class="card-footer">
							<small>Created {{ formatDate(library.created_at) }}</small>
						</div>
					</div>
				</div>
			</div>

			<!-- Empty State -->
			<div v-else class="empty-state text-center py-5">
				<font-awesome-icon :icon="['fas', 'folder']" size="5x" class="mb-3" />
				<h3>No Libraries Yet</h3>
				<p class="">Get started by adding your first library</p>
				<button class="btn btn-primary btn-lg" @click="showCreateModal = true">
					<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
					Add Your First Library
				</button>
			</div>
		</div>

		<!-- Create/Edit Modal -->
		<div class="modal fade" :class="{ show: showCreateModal || showEditModal }" :style="{ display: showCreateModal || showEditModal ? 'block' : 'none' }" tabindex="-1">
			<div class="modal-dialog">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							{{ showEditModal ? 'Edit Library' : 'Add New Library' }}
						</h5>
						<button type="button" class="btn-close" @click="closeModal"></button>
					</div>
					<div class="modal-body">
						<form @submit.prevent="saveLibrary">
							<div class="mb-3">
								<label for="libraryName" class="form-label">Library Name *</label>
								<input id="libraryName" v-model="form.name" type="text" class="form-control" placeholder="e.g., Movies, Personal Collection" required />
							</div>
							<div class="mb-3">
								<label for="libraryPath" class="form-label">Path *</label>
								<input id="libraryPath" v-model="form.path" type="text" class="form-control" placeholder="e.g., D:\Videos\Movies" required />
								<small class="">Full path to the directory containing your videos</small>
							</div>
							<div class="mb-3 form-check">
								<input id="libraryPrimary" v-model="form.primary" type="checkbox" class="form-check-input" />
								<label for="libraryPrimary" class="form-check-label">
									Set as primary library
									<small class="d-block text-muted">The primary library will be loaded by default in the browser</small>
								</label>
							</div>
						</form>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="closeModal">Cancel</button>
						<button type="button" class="btn btn-primary" @click="saveLibrary" :disabled="saving">
							<font-awesome-icon v-if="saving" :icon="['fas', 'spinner']" spin class="me-2" />
							{{ saving ? 'Saving...' : 'Save Library' }}
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Modal Backdrop -->
		<div v-if="showCreateModal || showEditModal" class="modal-backdrop fade show" @click="closeModal"></div>
	</div>
</template>

<script>
import { librariesAPI } from '@/services/api'

export default {
	name: 'LibrariesPage',
	data() {
		return {
			libraries: [],
			loading: false,
			saving: false,
			showCreateModal: false,
			showEditModal: false,
			editingLibrary: null,
			form: {
				name: '',
				path: '',
				primary: false,
			},
		}
	},
	async mounted() {
		await this.loadLibraries()
	},
	methods: {
		async loadLibraries() {
			this.loading = true
			try {
				const response = await librariesAPI.getAll()
				this.libraries = response.data || []
			} catch (error) {
				console.error('Failed to load libraries:', error)
				alert('Failed to load libraries. Please try again.')
			} finally {
				this.loading = false
			}
		},
		editLibrary(library) {
			this.editingLibrary = library
			this.form.name = library.name
			this.form.path = library.path
			this.form.primary = library.primary || false
			this.showEditModal = true
		},
		async saveLibrary() {
			if (!this.form.name || !this.form.path) {
				alert('Please fill in all required fields')
				return
			}

			this.saving = true
			try {
				if (this.showEditModal && this.editingLibrary) {
					// Update existing library
					await librariesAPI.update(this.editingLibrary.id, this.form)
				} else {
					// Create new library
					await librariesAPI.create(this.form)
				}

				await this.loadLibraries()
				this.closeModal()
			} catch (error) {
				console.error('Failed to save library:', error)
				alert(error.response?.data?.error || 'Failed to save library. Please check the path and try again.')
			} finally {
				this.saving = false
			}
		},
		confirmDelete(library) {
			if (confirm(`Are you sure you want to delete "${library.name}"?\n\nThis will not delete any files.`)) {
				this.deleteLibrary(library.id)
			}
		},
		async deleteLibrary(id) {
			try {
				await librariesAPI.delete(id)
				await this.loadLibraries()
			} catch (error) {
				console.error('Failed to delete library:', error)
				alert('Failed to delete library. Please try again.')
			}
		},
		closeModal() {
			this.showCreateModal = false
			this.showEditModal = false
			this.editingLibrary = null
			this.form = { name: '', path: '', primary: false }
		},
		formatDate(dateString) {
			const date = new Date(dateString)
			return date.toLocaleDateString()
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/libraries_page.css';
</style>
