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

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { studiosAPI, groupsAPI, getAssetURL } from '@/services/api'

// State
const studios = ref([])
const groups = ref([])
const loading = ref(false)
const searchQuery = ref('')
const sortBy = ref('name')
const filterCountry = ref('')
const selectedStudio = ref(null)
const showStudioModal = ref(false)
const showGroupModal = ref(false)
const editingStudio = ref(null)
const editingGroup = ref(null)
const studioForm = ref({
	name: '',
	logo_path: '',
	description: '',
	founded_date: '',
	country: '',
	website: '',
})
const groupForm = ref({
	name: '',
	logo_path: '',
	description: '',
})
const contextMenu = ref({
	show: false,
	x: 0,
	y: 0,
	studio: null,
})

// Computed
const filteredStudios = computed(() => {
	let filtered = [...studios.value]

	// Search filter
	if (searchQuery.value) {
		const query = searchQuery.value.toLowerCase()
		filtered = filtered.filter((s) => s.name.toLowerCase().includes(query) || s.description?.toLowerCase().includes(query))
	}

	// Country filter
	if (filterCountry.value) {
		filtered = filtered.filter((s) => s.country === filterCountry.value)
	}

	// Sort
	filtered.sort((a, b) => {
		if (sortBy.value === 'name') {
			return a.name.localeCompare(b.name)
		} else if (sortBy.value === 'created') {
			return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
		} else if (sortBy.value === 'videos') {
			return (b.video_count || 0) - (a.video_count || 0)
		}
		return 0
	})

	return filtered
})

const availableCountries = computed(() => {
	const countries = new Set()
	studios.value.forEach((s) => {
		if (s.country) countries.add(s.country)
	})
	return Array.from(countries).sort()
})

const studioGroups = computed(() => {
	if (!selectedStudio.value) return []
	return groups.value.filter((g) => g.studio_id === selectedStudio.value.id)
})

// Methods
const loadStudios = async () => {
	loading.value = true
	try {
		const response = await studiosAPI.getAll()
		studios.value = response || []
	} catch (error) {
		console.error('Failed to load studios:', error)
	} finally {
		loading.value = false
	}
}

const loadGroups = async () => {
	try {
		const response = await groupsAPI.getAll()
		groups.value = response || []
	} catch (error) {
		console.error('Failed to load groups:', error)
	}
}

const selectStudio = (studio) => {
	selectedStudio.value = studio
}

const selectGroup = (group) => {
	// Navigate to group details or show modal
	console.log('Selected group:', group)
}

const openCreateModal = () => {
	editingStudio.value = null
	studioForm.value = {
		name: '',
		logo_path: '',
		description: '',
		founded_date: '',
		country: '',
		website: '',
	}
	showStudioModal.value = true
}

const openCreateGroupModal = () => {
	editingGroup.value = null
	groupForm.value = {
		name: '',
		logo_path: '',
		description: '',
	}
	showGroupModal.value = true
}

const editStudio = (studio) => {
	editingStudio.value = studio
	studioForm.value = {
		name: studio.name,
		logo_path: studio.logo_path || '',
		description: studio.description || '',
		founded_date: studio.founded_date || '',
		country: studio.country || '',
		website: studio.metadata?.website || '',
	}
	showStudioModal.value = true
	hideContextMenu()
}

const editGroup = (group) => {
	editingGroup.value = group
	groupForm.value = {
		name: group.name,
		logo_path: group.logo_path || '',
		description: group.description || '',
	}
	showGroupModal.value = true
}

const saveStudio = async () => {
	try {
		const data = {
			name: studioForm.value.name,
			logo_path: studioForm.value.logo_path,
			description: studioForm.value.description,
			founded_date: studioForm.value.founded_date,
			country: studioForm.value.country,
			metadata: {
				website: studioForm.value.website,
			},
		}

		if (editingStudio.value) {
			await studiosAPI.update(editingStudio.value.id, data)
			console.log('Studio updated successfully')
		} else {
			await studiosAPI.create(data)
			console.log('Studio created successfully')
		}

		closeStudioModal()
		loadStudios()
	} catch (error) {
		console.error('Failed to save studio:', error)
	}
}

const saveGroup = async () => {
	if (!selectedStudio.value) return
	try {
		const data = {
			studio_id: selectedStudio.value.id,
			name: groupForm.value.name,
			logo_path: groupForm.value.logo_path,
			description: groupForm.value.description,
		}

		if (editingGroup.value) {
			await groupsAPI.update(editingGroup.value.id, data)
			console.log('Group updated successfully')
		} else {
			await groupsAPI.create(data)
			console.log('Group created successfully')
		}

		closeGroupModal()
		loadGroups()
	} catch (error) {
		console.error('Failed to save group:', error)
	}
}

const deleteStudio = async (studio) => {
	if (!confirm(`Are you sure you want to delete "${studio.name}"?`)) return

	try {
		await studiosAPI.delete(studio.id)
		console.log('Studio deleted successfully')
		loadStudios()
		if (selectedStudio.value?.id === studio.id) {
			selectedStudio.value = null
		}
	} catch (error) {
		console.error('Failed to delete studio:', error)
	}
	hideContextMenu()
}

const deleteGroup = async (group) => {
	if (!confirm(`Are you sure you want to delete "${group.name}"?`)) return

	try {
		await groupsAPI.delete(group.id)
		console.log('Group deleted successfully')
		loadGroups()
	} catch (error) {
		console.error('Failed to delete group:', error)
	}
}

const closeStudioModal = () => {
	showStudioModal.value = false
	editingStudio.value = null
}

const closeGroupModal = () => {
	showGroupModal.value = false
	editingGroup.value = null
}

const showContextMenu = (event, studio) => {
	contextMenu.value = {
		show: true,
		x: event.clientX,
		y: event.clientY,
		studio,
	}
}

const hideContextMenu = () => {
	contextMenu.value.show = false
}

// Lifecycle
onMounted(() => {
	loadStudios()
	loadGroups()
	document.addEventListener('click', hideContextMenu)
})

onBeforeUnmount(() => {
	document.removeEventListener('click', hideContextMenu)
})
</script>

<style scoped>
@import '@/styles/pages/studios_page.css';
</style>
