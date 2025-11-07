<template>
	<div class="studios-page">
		<div class="container-fluid py-4">
			<!-- Header -->
			<div class="page-header mb-4">
				<div class="d-flex justify-content-between align-items-center">
					<div>
						<h1>
							<font-awesome-icon :icon="['fas', 'building']" />
							Studios
							<span class="studio-count">({{ studios.length }})</span>
						</h1>
					</div>
					<div class="d-flex gap-2">
						<button class="btn btn-primary" @click="openCreateModal">
							<font-awesome-icon :icon="['fas', 'plus']" />
							Add Studio
						</button>
					</div>
				</div>
			</div>

			<!-- Search and filters -->
			<div class="filters-section mb-4">
				<div class="row g-3">
					<div class="col-md-6">
						<div class="input-group">
							<span class="input-group-text">
								<font-awesome-icon :icon="['fas', 'search']" />
							</span>
							<input v-model="searchQuery" type="text" class="form-control" placeholder="Search studios..." />
						</div>
					</div>
					<div class="col-md-3">
						<select v-model="sortBy" class="form-select">
							<option value="name">Sort by Name</option>
							<option value="created">Sort by Date Added</option>
							<option value="videos">Sort by Video Count</option>
						</select>
					</div>
					<div class="col-md-3">
						<select v-model="filterCountry" class="form-select">
							<option value="">All Countries</option>
							<option v-for="country in availableCountries" :key="country" :value="country">
								{{ country }}
							</option>
						</select>
					</div>
				</div>
			</div>

			<!-- Studio Wall Grid -->
			<div class="studio-wall">
				<div v-if="loading" class="text-center py-5">
					<div class="spinner-border text-primary" role="status">
						<span class="visually-hidden">Loading...</span>
					</div>
				</div>
				<div v-else-if="filteredStudios.length === 0" class="text-center py-5 text-muted">
					<font-awesome-icon :icon="['fas', 'building']" size="3x" class="mb-3" />
					<p>No studios found</p>
				</div>
				<div v-else class="row g-4">
					<div v-for="studio in filteredStudios" :key="studio.id" class="col-xl-2 col-lg-3 col-md-4 col-sm-6">
						<div class="studio-card" @click="selectStudio(studio)" @contextmenu.prevent="showContextMenu($event, studio)">
							<div class="studio-logo">
								<img v-if="studio.logo_path" :src="getAssetURL(studio.logo_path)" :alt="studio.name" />
								<div v-else class="logo-placeholder">
									<font-awesome-icon :icon="['fas', 'building']" size="3x" />
								</div>
							</div>
							<div class="studio-info">
								<h3 class="studio-name">{{ studio.name }}</h3>
								<div class="studio-meta">
									<span v-if="studio.country" class="badge bg-secondary">
										<font-awesome-icon :icon="['fas', 'globe']" />
										{{ studio.country }}
									</span>
									<span class="badge bg-primary">
										<font-awesome-icon :icon="['fas', 'video']" />
										{{ studio.video_count || 0 }}
									</span>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Studio Details Panel (Sidebar) -->
			<div v-if="selectedStudio" class="studio-details-panel" :class="{ open: selectedStudio }">
				<div class="panel-header">
					<h2>
						<font-awesome-icon :icon="['fas', 'building']" />
						{{ selectedStudio.name }}
					</h2>
					<button class="btn-close" @click="selectedStudio = null"></button>
				</div>
				<div class="panel-body">
					<!-- Studio Overview -->
					<div class="detail-section">
						<h3>Overview</h3>
						<div class="studio-overview">
							<div v-if="selectedStudio.logo_path" class="overview-logo">
								<img :src="getAssetURL(selectedStudio.logo_path)" :alt="selectedStudio.name" />
							</div>
							<div class="overview-info">
								<p v-if="selectedStudio.description">{{ selectedStudio.description }}</p>
								<div class="info-grid">
									<div v-if="selectedStudio.founded_date" class="info-item"><strong>Founded:</strong> {{ selectedStudio.founded_date }}</div>
									<div v-if="selectedStudio.country" class="info-item"><strong>Country:</strong> {{ selectedStudio.country }}</div>
									<div v-if="selectedStudio.metadata?.website" class="info-item">
										<strong>Website:</strong>
										<a :href="selectedStudio.metadata.website" target="_blank">
											{{ selectedStudio.metadata.website }}
											<font-awesome-icon :icon="['fas', 'external-link-alt']" />
										</a>
									</div>
								</div>
							</div>
						</div>
					</div>

					<!-- Quick Actions -->
					<div class="detail-section">
						<h3>Actions</h3>
						<div class="action-buttons">
							<button class="btn btn-sm btn-outline-primary" @click="editStudio(selectedStudio)">
								<font-awesome-icon :icon="['fas', 'edit']" />
								Edit
							</button>
							<button class="btn btn-sm btn-outline-danger" @click="deleteStudio(selectedStudio)">
								<font-awesome-icon :icon="['fas', 'trash']" />
								Delete
							</button>
						</div>
					</div>

					<!-- Groups -->
					<div class="detail-section">
						<div class="d-flex justify-content-between align-items-center mb-3">
							<h3>Groups ({{ studioGroups.length }})</h3>
							<button class="btn btn-sm btn-primary" @click="openCreateGroupModal">
								<font-awesome-icon :icon="['fas', 'plus']" />
								Add Group
							</button>
						</div>
						<div v-if="studioGroups.length === 0" class="text-muted">
							<p>No groups in this studio</p>
						</div>
						<div v-else class="groups-list">
							<div v-for="group in studioGroups" :key="group.id" class="group-item" @click="selectGroup(group)">
								<div class="group-info">
									<strong>{{ group.name }}</strong>
									<p v-if="group.description" class="text-muted small mb-0">{{ group.description }}</p>
								</div>
								<div class="group-actions">
									<button class="btn btn-sm btn-outline-secondary" @click.stop="editGroup(group)">
										<font-awesome-icon :icon="['fas', 'edit']" />
									</button>
									<button class="btn btn-sm btn-outline-danger" @click.stop="deleteGroup(group)">
										<font-awesome-icon :icon="['fas', 'trash']" />
									</button>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Create/Edit Studio Modal -->
			<div v-if="showStudioModal" class="modal show d-block" tabindex="-1">
				<div class="modal-dialog modal-dialog-centered">
					<div class="modal-content">
						<div class="modal-header">
							<h5 class="modal-title">{{ editingStudio ? 'Edit Studio' : 'Create Studio' }}</h5>
							<button type="button" class="btn-close" @click="closeStudioModal"></button>
						</div>
						<div class="modal-body">
							<form @submit.prevent="saveStudio">
								<div class="mb-3">
									<label class="form-label">Name *</label>
									<input v-model="studioForm.name" type="text" class="form-control" required />
								</div>
								<div class="mb-3">
									<label class="form-label">Logo Path</label>
									<input v-model="studioForm.logo_path" type="text" class="form-control" />
								</div>
								<div class="mb-3">
									<label class="form-label">Description</label>
									<textarea v-model="studioForm.description" class="form-control" rows="3"></textarea>
								</div>
								<div class="row">
									<div class="col-md-6 mb-3">
										<label class="form-label">Founded Date</label>
										<input v-model="studioForm.founded_date" type="text" class="form-control" placeholder="YYYY-MM-DD" />
									</div>
									<div class="col-md-6 mb-3">
										<label class="form-label">Country</label>
										<input v-model="studioForm.country" type="text" class="form-control" />
									</div>
								</div>
								<div class="mb-3">
									<label class="form-label">Website</label>
									<input v-model="studioForm.website" type="url" class="form-control" />
								</div>
							</form>
						</div>
						<div class="modal-footer">
							<button type="button" class="btn btn-secondary" @click="closeStudioModal">Cancel</button>
							<button type="button" class="btn btn-primary" @click="saveStudio">Save</button>
						</div>
					</div>
				</div>
			</div>

			<!-- Create/Edit Group Modal -->
			<div v-if="showGroupModal" class="modal show d-block" tabindex="-1">
				<div class="modal-dialog modal-dialog-centered">
					<div class="modal-content">
						<div class="modal-header">
							<h5 class="modal-title">{{ editingGroup ? 'Edit Group' : 'Create Group' }}</h5>
							<button type="button" class="btn-close" @click="closeGroupModal"></button>
						</div>
						<div class="modal-body">
							<form @submit.prevent="saveGroup">
								<div class="mb-3">
									<label class="form-label">Name *</label>
									<input v-model="groupForm.name" type="text" class="form-control" required />
								</div>
								<div class="mb-3">
									<label class="form-label">Logo Path</label>
									<input v-model="groupForm.logo_path" type="text" class="form-control" />
								</div>
								<div class="mb-3">
									<label class="form-label">Description</label>
									<textarea v-model="groupForm.description" class="form-control" rows="3"></textarea>
								</div>
							</form>
						</div>
						<div class="modal-footer">
							<button type="button" class="btn btn-secondary" @click="closeGroupModal">Cancel</button>
							<button type="button" class="btn btn-primary" @click="saveGroup">Save</button>
						</div>
					</div>
				</div>
			</div>

			<!-- Modal backdrop -->
			<div v-if="showStudioModal || showGroupModal" class="modal-backdrop show"></div>

			<!-- Context Menu -->
			<div v-if="contextMenu.show" class="context-menu" :style="{ top: contextMenu.y + 'px', left: contextMenu.x + 'px' }">
				<div class="context-menu-item" @click="editStudio(contextMenu.studio)">
					<font-awesome-icon :icon="['fas', 'edit']" />
					Edit
				</div>
				<div class="context-menu-item danger" @click="deleteStudio(contextMenu.studio)">
					<font-awesome-icon :icon="['fas', 'trash']" />
					Delete
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import { studiosAPI, groupsAPI } from '@/services/api'
import { getAssetURL } from '@/services/api'

export default {
	name: 'StudiosPage',
	data() {
		return {
			studios: [],
			groups: [],
			loading: false,
			searchQuery: '',
			sortBy: 'name',
			filterCountry: '',
			selectedStudio: null,
			showStudioModal: false,
			showGroupModal: false,
			editingStudio: null,
			editingGroup: null,
			studioForm: {
				name: '',
				logo_path: '',
				description: '',
				founded_date: '',
				country: '',
				website: '',
			},
			groupForm: {
				name: '',
				logo_path: '',
				description: '',
			},
			contextMenu: {
				show: false,
				x: 0,
				y: 0,
				studio: null,
			},
		}
	},
	computed: {
		filteredStudios() {
			let filtered = [...this.studios]

			// Search filter
			if (this.searchQuery) {
				const query = this.searchQuery.toLowerCase()
				filtered = filtered.filter((s) => s.name.toLowerCase().includes(query) || s.description?.toLowerCase().includes(query))
			}

			// Country filter
			if (this.filterCountry) {
				filtered = filtered.filter((s) => s.country === this.filterCountry)
			}

			// Sort
			filtered.sort((a, b) => {
				if (this.sortBy === 'name') {
					return a.name.localeCompare(b.name)
				} else if (this.sortBy === 'created') {
					return new Date(b.created_at) - new Date(a.created_at)
				} else if (this.sortBy === 'videos') {
					return (b.video_count || 0) - (a.video_count || 0)
				}
				return 0
			})

			return filtered
		},
		availableCountries() {
			const countries = new Set()
			this.studios.forEach((s) => {
				if (s.country) countries.add(s.country)
			})
			return Array.from(countries).sort()
		},
		studioGroups() {
			if (!this.selectedStudio) return []
			return this.groups.filter((g) => g.studio_id === this.selectedStudio.id)
		},
	},
	mounted() {
		this.loadStudios()
		this.loadGroups()
		document.addEventListener('click', this.hideContextMenu)
	},
	beforeUnmount() {
		document.removeEventListener('click', this.hideContextMenu)
	},
	methods: {
		getAssetURL,
		async loadStudios() {
			this.loading = true
			try {
				const response = await studiosAPI.getAll()
				this.studios = response.data || []
			} catch (error) {
				console.error('Failed to load studios:', error)
				this.$toast.error('Failed to load studios', 'error')
			} finally {
				this.loading = false
			}
		},
		async loadGroups() {
			try {
				const response = await groupsAPI.getAll()
				this.groups = response.data || []
			} catch (error) {
				console.error('Failed to load groups:', error)
			}
		},
		selectStudio(studio) {
			this.selectedStudio = studio
		},
		selectGroup(group) {
			// Navigate to group details or show modal
			console.log('Selected group:', group)
		},
		openCreateModal() {
			this.editingStudio = null
			this.studioForm = {
				name: '',
				logo_path: '',
				description: '',
				founded_date: '',
				country: '',
				website: '',
			}
			this.showStudioModal = true
		},
		openCreateGroupModal() {
			this.editingGroup = null
			this.groupForm = {
				name: '',
				logo_path: '',
				description: '',
			}
			this.showGroupModal = true
		},
		editStudio(studio) {
			this.editingStudio = studio
			this.studioForm = {
				name: studio.name,
				logo_path: studio.logo_path || '',
				description: studio.description || '',
				founded_date: studio.founded_date || '',
				country: studio.country || '',
				website: studio.metadata?.website || '',
			}
			this.showStudioModal = true
			this.hideContextMenu()
		},
		editGroup(group) {
			this.editingGroup = group
			this.groupForm = {
				name: group.name,
				logo_path: group.logo_path || '',
				description: group.description || '',
			}
			this.showGroupModal = true
		},
		async saveStudio() {
			try {
				const data = {
					name: this.studioForm.name,
					logo_path: this.studioForm.logo_path,
					description: this.studioForm.description,
					founded_date: this.studioForm.founded_date,
					country: this.studioForm.country,
					metadata: {
						website: this.studioForm.website,
					},
				}

				if (this.editingStudio) {
					await studiosAPI.update(this.editingStudio.id, data)
					this.$toast.success('Studio updated successfully', 'success')
				} else {
					await studiosAPI.create(data)
					this.$toast.success('Studio created successfully', 'success')
				}

				this.closeStudioModal()
				this.loadStudios()
			} catch (error) {
				console.error('Failed to save studio:', error)
				this.$toast.error('Failed to save studio', 'error')
			}
		},
		async saveGroup() {
			try {
				const data = {
					studio_id: this.selectedStudio.id,
					name: this.groupForm.name,
					logo_path: this.groupForm.logo_path,
					description: this.groupForm.description,
				}

				if (this.editingGroup) {
					await groupsAPI.update(this.editingGroup.id, data)
					this.$toast.success('Group updated successfully', 'success')
				} else {
					await groupsAPI.create(data)
					this.$toast.success('Group created successfully', 'success')
				}

				this.closeGroupModal()
				this.loadGroups()
			} catch (error) {
				console.error('Failed to save group:', error)
				this.$toast.error('Failed to save group', 'error')
			}
		},
		async deleteStudio(studio) {
			if (!confirm(`Are you sure you want to delete "${studio.name}"?`)) return

			try {
				await studiosAPI.delete(studio.id)
				this.$toast.error('Studio deleted successfully', 'success')
				this.loadStudios()
				if (this.selectedStudio?.id === studio.id) {
					this.selectedStudio = null
				}
			} catch (error) {
				console.error('Failed to delete studio:', error)
				this.$toast.error('Failed to delete studio', 'error')
			}
			this.hideContextMenu()
		},
		async deleteGroup(group) {
			if (!confirm(`Are you sure you want to delete "${group.name}"?`)) return

			try {
				await groupsAPI.delete(group.id)
				this.$toast.success('Group deleted successfully', 'success')
				this.loadGroups()
			} catch (error) {
				console.error('Failed to delete group:', error)
				this.$toast.error('Failed to delete group', 'error')
			}
		},
		closeStudioModal() {
			this.showStudioModal = false
			this.editingStudio = null
		},
		closeGroupModal() {
			this.showGroupModal = false
			this.editingGroup = null
		},
		showContextMenu(event, studio) {
			this.contextMenu = {
				show: true,
				x: event.clientX,
				y: event.clientY,
				studio,
			}
		},
		hideContextMenu() {
			this.contextMenu.show = false
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/studios_page.css';
</style>
