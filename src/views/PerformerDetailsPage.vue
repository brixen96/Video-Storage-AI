<template>
	<div class="performer-details-page">
		<div class="container-fluid py-4">
			<!-- Loading State -->
			<div v-if="loading" class="text-center py-5">
				<div class="spinner-border text-primary" role="status">
					<span class="visually-hidden">Loading...</span>
				</div>
			</div>

			<!-- Content -->
			<div v-else-if="performer" class="performer-content">
				<!-- Header -->
				<div class="page-header mb-4">
					<div class="d-flex justify-content-between align-items-start">
						<div class="d-flex align-items-center gap-4">
							<!-- Profile Image -->
							<div class="performer-avatar">
								<img v-if="performer.profile_image_path" :src="getAssetURL(performer.profile_image_path)" :alt="performer.name" />
								<div v-else class="avatar-placeholder">
									<font-awesome-icon :icon="['fas', 'user']" size="4x" />
								</div>
							</div>

							<!-- Basic Info -->
							<div>
								<h1>{{ performer.name }}</h1>
								<div class="performer-meta d-flex gap-3 mt-2">
									<span v-if="performer.age" class="badge bg-secondary">
										<font-awesome-icon :icon="['fas', 'calendar']" class="me-1" />
										{{ performer.age }} years
									</span>
									<span v-if="performer.country" class="badge bg-secondary">
										<font-awesome-icon :icon="['fas', 'globe']" class="me-1" />
										{{ performer.country }}
									</span>
									<span v-if="performer.zoo" class="badge bg-danger">
										<font-awesome-icon :icon="['fas', 'dog']" class="me-1" />
										Zoo
									</span>
								</div>
							</div>
						</div>

						<!-- Action Buttons -->
						<div class="d-flex gap-2">
							<button class="btn btn-outline-secondary" @click="$router.back()">
								<font-awesome-icon :icon="['fas', 'arrow-left']" class="me-2" />
								Back
							</button>
						</div>
					</div>
				</div>

				<!-- Main Content Grid -->
				<div class="row g-4">
					<!-- Left Column - Details -->
					<div class="col-lg-8">
						<!-- Physical Attributes -->
						<div class="detail-card mb-4">
							<h3>
								<font-awesome-icon :icon="['fas', 'user-circle']" class="me-2" />
								Physical Attributes
							</h3>
							<div class="row g-3">
								<div v-if="performer.height" class="col-md-4">
									<div class="attribute-item">
										<label>Height</label>
										<div class="value">{{ performer.height }} cm</div>
									</div>
								</div>
								<div v-if="performer.weight" class="col-md-4">
									<div class="attribute-item">
										<label>Weight</label>
										<div class="value">{{ performer.weight }} kg</div>
									</div>
								</div>
								<div v-if="performer.breast_size" class="col-md-4">
									<div class="attribute-item">
										<label>Breast Size</label>
										<div class="value">{{ performer.breast_size }}</div>
									</div>
								</div>
								<div v-if="performer.hair_color" class="col-md-4">
									<div class="attribute-item">
										<label>Hair Color</label>
										<div class="value">{{ performer.hair_color }}</div>
									</div>
								</div>
								<div v-if="performer.eye_color" class="col-md-4">
									<div class="attribute-item">
										<label>Eye Color</label>
										<div class="value">{{ performer.eye_color }}</div>
									</div>
								</div>
								<div v-if="performer.ethnicity" class="col-md-4">
									<div class="attribute-item">
										<label>Ethnicity</label>
										<div class="value">{{ performer.ethnicity }}</div>
									</div>
								</div>
							</div>
						</div>

						<!-- Videos -->
						<div class="detail-card">
							<h3>
								<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
								Videos ({{ performer.scene_count || 0 }})
							</h3>
							<div v-if="performer.scene_count > 0" class="text-muted">
								<p>Videos featuring this performer will be listed here</p>
							</div>
							<div v-else class="text-muted">
								<p>No videos featuring this performer yet</p>
							</div>
						</div>
					</div>

					<!-- Right Column - Master Tags -->
					<div class="col-lg-4">
						<div class="detail-card">
							<div class="d-flex justify-content-between align-items-center mb-3">
								<h3>
									<font-awesome-icon :icon="['fas', 'tags']" class="me-2" />
									Master Tags
								</h3>
								<button class="btn btn-sm btn-primary" @click="showAddTagModal = true">
									<font-awesome-icon :icon="['fas', 'plus']" />
								</button>
							</div>

							<div class="tags-info mb-3">
								<small class="text-muted">
									<font-awesome-icon :icon="['fas', 'info-circle']" class="me-1" />
									Master tags automatically apply to all videos featuring this performer
								</small>
							</div>

							<!-- Tag List -->
							<div v-if="performerTags.length > 0" class="tag-list">
								<div v-for="tag in performerTags" :key="tag.id" class="tag-item">
									<span class="tag-name">{{ tag.name }}</span>
									<button class="btn-remove-tag" @click="removeTag(tag.id)" title="Remove tag">
										<font-awesome-icon :icon="['fas', 'times']" />
									</button>
								</div>
							</div>
							<div v-else class="text-muted">
								<p>No master tags assigned</p>
							</div>

							<!-- Sync Button -->
							<div v-if="performerTags.length > 0" class="mt-3">
								<button class="btn btn-outline-primary w-100" @click="syncTagsToVideos" :disabled="syncing">
									<font-awesome-icon :icon="['fas', syncing ? 'spinner' : 'sync']" :spin="syncing" class="me-2" />
									{{ syncing ? 'Syncing...' : 'Sync Tags to All Videos' }}
								</button>
								<small class="text-muted d-block mt-2">
									Apply these master tags to all {{ performer.scene_count || 0 }} videos
								</small>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Error State -->
			<div v-else class="text-center py-5">
				<font-awesome-icon :icon="['fas', 'exclamation-triangle']" size="3x" class="text-warning mb-3" />
				<p class="text-muted">Performer not found</p>
			</div>
		</div>

		<!-- Add Tag Modal -->
		<div v-if="showAddTagModal" class="modal-overlay" @click.self="showAddTagModal = false">
			<div class="modal-dialog">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">Add Master Tag</h5>
						<button type="button" class="btn-close" @click="showAddTagModal = false"></button>
					</div>
					<div class="modal-body">
						<div class="mb-3">
							<label class="form-label">Select Tag</label>
							<select v-model="selectedTagId" class="form-select">
								<option value="">Choose a tag...</option>
								<option v-for="tag in availableTags" :key="tag.id" :value="tag.id">
									{{ tag.name }}
								</option>
							</select>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="showAddTagModal = false">Cancel</button>
						<button type="button" class="btn btn-primary" @click="addTag" :disabled="!selectedTagId">Add Tag</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, onMounted, computed, getCurrentInstance } from 'vue'
import { useRoute } from 'vue-router'
import { performersAPI, tagsAPI, getAssetURL } from '@/services/api'

const route = useRoute()
const { proxy } = getCurrentInstance()
const toast = proxy.$toast

// State
const performer = ref(null)
const performerTags = ref([])
const allTags = ref([])
const loading = ref(true)
const syncing = ref(false)
const showAddTagModal = ref(false)
const selectedTagId = ref('')

// Computed
const availableTags = computed(() => {
	const assignedTagIds = new Set(performerTags.value.map((t) => t.id))
	return allTags.value.filter((tag) => !assignedTagIds.has(tag.id))
})

// Load performer details
const loadPerformer = async () => {
	loading.value = true
	try {
		const performerId = route.params.id
		const response = await performersAPI.getById(performerId)
		performer.value = response.data
		await loadPerformerTags()
	} catch (error) {
		console.error('Failed to load performer:', error)
		toast.error('Error', 'Failed to load performer details')
	} finally {
		loading.value = false
	}
}

// Load performer's master tags
const loadPerformerTags = async () => {
	try {
		const response = await performersAPI.getTags(route.params.id)
		performerTags.value = response.data || []
	} catch (error) {
		console.error('Failed to load performer tags:', error)
		performerTags.value = []
	}
}

// Load all available tags
const loadAllTags = async () => {
	try {
		const response = await tagsAPI.getAll()
		allTags.value = response.data || []
	} catch (error) {
		console.error('Failed to load tags:', error)
		allTags.value = []
	}
}

// Add tag to performer
const addTag = async () => {
	if (!selectedTagId.value) return

	try {
		await performersAPI.addTag(route.params.id, selectedTagId.value)
		toast.success('Tag Added', 'Master tag added to performer')
		await loadPerformerTags()
		showAddTagModal.value = false
		selectedTagId.value = ''
	} catch (error) {
		console.error('Failed to add tag:', error)
		toast.error('Error', 'Failed to add master tag')
	}
}

// Remove tag from performer
const removeTag = async (tagId) => {
	if (!confirm('Remove this master tag from the performer?')) return

	try {
		await performersAPI.removeTag(route.params.id, tagId)
		toast.success('Tag Removed', 'Master tag removed from performer')
		await loadPerformerTags()
	} catch (error) {
		console.error('Failed to remove tag:', error)
		toast.error('Error', 'Failed to remove master tag')
	}
}

// Sync tags to all videos
const syncTagsToVideos = async () => {
	if (!confirm(`Apply master tags to all ${performer.value.scene_count || 0} videos featuring this performer?`)) return

	syncing.value = true
	try {
		const response = await performersAPI.syncTags(route.params.id)
		const videosUpdated = response.data?.videos_updated || 0
		toast.success('Sync Complete', `Applied master tags to ${videosUpdated} videos`)
	} catch (error) {
		console.error('Failed to sync tags:', error)
		toast.error('Error', 'Failed to sync master tags to videos')
	} finally {
		syncing.value = false
	}
}

// Lifecycle
onMounted(() => {
	loadPerformer()
	loadAllTags()
})
</script>

<style scoped>
@import '@/styles/pages/performers_details_page.css';
</style>
