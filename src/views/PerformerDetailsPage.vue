<template>
	<div class="performer-details-page">
		<div class="container-fluid py-4">
			<!-- Breadcrumb -->
			<nav aria-label="breadcrumb" class="mb-3">
				<ol class="breadcrumb">
					<li class="breadcrumb-item">
						<router-link to="/performers">
							<font-awesome-icon :icon="['fas', 'users']" class="me-1" />
							Performers
						</router-link>
					</li>
					<li class="breadcrumb-item active" aria-current="page">{{ performer?.name || 'Loading...' }}</li>
				</ol>
			</nav>

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
					<div class="header-content">
						<div class="d-flex align-items-center gap-4">
							<!-- Video Avatar Preview -->
							<div class="performer-avatar" @mouseenter="hoveredAvatar = true" @mouseleave="hoveredAvatar = false">
								<!-- Static Thumbnail (shown when not hovered) -->
								<img
									v-show="!hoveredAvatar"
									v-if="performer.thumbnail_path"
									:key="`thumb-${performer.id}`"
									:src="getAssetURL(performer.thumbnail_path)"
									:alt="performer.name"
									class="avatar-image"
								/>
								<!-- Video Preview (shown on hover) -->
								<video
									v-show="hoveredAvatar"
									v-if="performer.preview_path"
									:key="`video-${performer.id}`"
									:src="getAssetURL(performer.preview_path)"
									autoplay
									loop
									muted
									playsinline
									class="avatar-video"
								></video>
								<!-- Placeholder (shown when no thumbnail and not hovered) -->
								<div v-if="!performer.thumbnail_path && !performer.preview_path" class="avatar-placeholder">
									<font-awesome-icon :icon="['fas', 'user']" size="4x" />
								</div>
								<button class="btn-generate-thumbnail" @click="generateThumbnail" :disabled="generatingThumbnail" title="Generate Thumbnail">
									<font-awesome-icon :icon="['fas', generatingThumbnail ? 'spinner' : 'camera']" :spin="generatingThumbnail" />
								</button>
							</div>

							<!-- Basic Info -->
							<div class="flex-grow-1">
								<div class="d-flex align-items-center gap-3 mb-2">
									<h1 class="mb-0">{{ performer.name }}</h1>
									<button class="btn btn-sm btn-outline-light" @click="editMode = !editMode">
										<font-awesome-icon :icon="['fas', editMode ? 'times' : 'edit']" class="me-1" />
										{{ editMode ? 'Cancel' : 'Edit' }}
									</button>
								</div>

								<div class="performer-meta d-flex gap-3 mt-2">
									<span v-if="performerAge" class="badge bg-secondary">
										<font-awesome-icon :icon="['fas', 'calendar']" class="me-1" />
										{{ performerAge }} years
									</span>
									<span v-if="performerMetadata?.birthplace" class="badge bg-secondary">
										<font-awesome-icon :icon="['fas', 'globe']" class="me-1" />
										{{ performerMetadata.birthplace }}
									</span>
									<span v-if="performer.category === 'zoo'" class="badge bg-danger">
										<font-awesome-icon :icon="['fas', 'dog']" class="me-1" />
										Zoo
									</span>
									<span v-if="performer.category === '3d'" class="badge bg-warning">
										<font-awesome-icon :icon="['fas', 'cube']" class="me-1" />
										3D
									</span>
								</div>

								<!-- Quick Stats -->
								<div class="quick-stats mt-3">
									<div class="stat-item">
										<font-awesome-icon :icon="['fas', 'video']" class="me-1" />
										<strong>{{ performer.video_count || 0 }}</strong> Videos
									</div>
									<div class="stat-item" v-if="stats.totalDuration">
										<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
										<strong>{{ formatDuration(stats.totalDuration) }}</strong> Total
									</div>
									<div class="stat-item" v-if="stats.avgRating">
										<font-awesome-icon :icon="['fas', 'star']" class="me-1 text-warning" />
										<strong>{{ stats.avgRating.toFixed(1) }}</strong> Avg Rating
									</div>
									<div class="stat-item" v-if="previewVideos.length">
										<font-awesome-icon :icon="['fas', 'play-circle']" class="me-1" />
										<strong>{{ previewVideos.length }}</strong> Previews
									</div>
								</div>
							</div>
						</div>

						<!-- Action Buttons -->
						<div class="action-buttons">
							<button class="btn btn-outline-primary" @click="fetchMetadata" :disabled="fetchingMetadata">
								<font-awesome-icon :icon="['fas', fetchingMetadata ? 'spinner' : 'cloud-download-alt']" :spin="fetchingMetadata" class="me-2" />
								Fetch Metadata
							</button>
							<button class="btn btn-outline-warning" @click="resetMetadata">
								<font-awesome-icon :icon="['fas', 'undo']" class="me-2" />
								Reset
							</button>
							<button class="btn btn-outline-secondary" @click="$router.back()">
								<font-awesome-icon :icon="['fas', 'arrow-left']" class="me-2" />
								Back
							</button>
						</div>
					</div>
				</div>

				<!-- Main Content Grid -->
				<div class="row g-4">
					<!-- Left Column - Details & Previews -->
					<div class="col-lg-8">
						<!-- Edit Form (when in edit mode) -->
						<div v-if="editMode" class="detail-card mb-4">
							<h3>
								<font-awesome-icon :icon="['fas', 'edit']" class="me-2" />
								Edit Performer Details
							</h3>
							<form @submit.prevent="saveChanges">
								<div class="row g-3">
									<div class="col-md-6">
										<label class="form-label">Name</label>
										<input v-model="editForm.name" type="text" class="form-control" required />
									</div>
									<div class="col-md-6">
										<label class="form-label">Category</label>
										<select v-model="editForm.category" class="form-select">
											<option value="regular">Regular</option>
											<option value="zoo">Zoo</option>
											<option value="3d">3D</option>
										</select>
									</div>
								</div>
								<div class="d-flex gap-2 mt-4">
									<button type="submit" class="btn btn-primary" :disabled="saving">
										<font-awesome-icon :icon="['fas', saving ? 'spinner' : 'save']" :spin="saving" class="me-2" />
										{{ saving ? 'Saving...' : 'Save Changes' }}
									</button>
									<button type="button" class="btn btn-secondary" @click="editMode = false">Cancel</button>
								</div>
							</form>
						</div>

						<!-- Appearance / Physical Attributes -->
						<div v-else class="detail-card mb-4">
							<h3>
								<font-awesome-icon :icon="['fas', 'user-circle']" class="me-2" />
								Appearance
							</h3>
							<div v-if="appearanceData && Object.keys(appearanceData).length > 0" class="row g-3">
								<div v-for="(value, key) in appearanceData" :key="key" class="col-md-4">
									<div class="attribute-item">
										<label>{{ formatAttributeName(key) }}</label>
										<div class="value">{{ value }}</div>
									</div>
								</div>
							</div>
							<div v-else-if="hasBasicMetadata" class="row g-3">
								<div v-if="performerMetadata?.height" class="col-md-4">
									<div class="attribute-item">
										<label>Height</label>
										<div class="value">{{ performerMetadata.height }}</div>
									</div>
								</div>
								<div v-if="performerMetadata?.weight" class="col-md-4">
									<div class="attribute-item">
										<label>Weight</label>
										<div class="value">{{ performerMetadata.weight }}</div>
									</div>
								</div>
								<div v-if="performerMetadata?.measurements" class="col-md-4">
									<div class="attribute-item">
										<label>Measurements</label>
										<div class="value">{{ performerMetadata.measurements }}</div>
									</div>
								</div>
								<div v-if="performerMetadata?.hair_color" class="col-md-4">
									<div class="attribute-item">
										<label>Hair Color</label>
										<div class="value">{{ performerMetadata.hair_color }}</div>
									</div>
								</div>
								<div v-if="performerMetadata?.eye_color" class="col-md-4">
									<div class="attribute-item">
										<label>Eye Color</label>
										<div class="value">{{ performerMetadata.eye_color }}</div>
									</div>
								</div>
								<div v-if="performerMetadata?.ethnicity" class="col-md-4">
									<div class="attribute-item">
										<label>Ethnicity</label>
										<div class="value">{{ performerMetadata.ethnicity }}</div>
									</div>
								</div>
								<div v-if="performerMetadata?.tattoos" class="col-md-6">
									<div class="attribute-item">
										<label>Tattoos</label>
										<div class="value">{{ performerMetadata.tattoos }}</div>
									</div>
								</div>
								<div v-if="performerMetadata?.piercings" class="col-md-6">
									<div class="attribute-item">
										<label>Piercings</label>
										<div class="value">{{ performerMetadata.piercings }}</div>
									</div>
								</div>
							</div>
							<div v-else class="text-muted text-center py-3">
								<p>No appearance data available</p>
								<button class="btn btn-sm btn-primary" @click="fetchMetadata" :disabled="fetchingMetadata">
									<font-awesome-icon :icon="['fas', fetchingMetadata ? 'spinner' : 'cloud-download-alt']" :spin="fetchingMetadata" class="me-2" />
									Fetch from AdultDataLink
								</button>
							</div>
						</div>

						<!-- Preview Videos Gallery -->
						<div v-if="previewVideos.length > 0" class="detail-card mb-4">
							<div class="d-flex justify-content-between align-items-center mb-3">
								<h3>
									<font-awesome-icon :icon="['fas', 'play-circle']" class="me-2" />
									Preview Videos ({{ previewVideos.length }})
								</h3>
								<div class="btn-group btn-group-sm">
									<button
										:class="['btn', previewViewMode === 'grid' ? 'btn-primary' : 'btn-outline-primary']"
										@click="previewViewMode = 'grid'"
										title="Grid View"
									>
										<font-awesome-icon :icon="['fas', 'th']" />
									</button>
									<button
										:class="['btn', previewViewMode === 'carousel' ? 'btn-primary' : 'btn-outline-primary']"
										@click="previewViewMode = 'carousel'"
										title="Carousel View"
									>
										<font-awesome-icon :icon="['fas', 'ellipsis-h']" />
									</button>
								</div>
							</div>

							<!-- Grid View -->
							<div v-if="previewViewMode === 'grid'" class="preview-grid">
								<div v-for="preview in previewVideos" :key="preview.id" class="preview-card" @click="playPreview(preview)">
									<div class="preview-thumbnail">
										<video :src="getAssetURL(preview.file_path)" preload="metadata" muted></video>
										<div class="play-overlay">
											<font-awesome-icon :icon="['fas', 'play-circle']" size="3x" />
										</div>
										<div class="preview-duration" v-if="preview.duration">{{ formatDuration(preview.duration) }}</div>
									</div>
									<div class="preview-title">{{ preview.title || `Preview ${preview.id}` }}</div>
								</div>
							</div>

							<!-- Carousel View -->
							<div v-else class="preview-carousel">
								<button class="carousel-btn prev" @click="scrollPreviews(-1)" :disabled="previewScroll === 0">
									<font-awesome-icon :icon="['fas', 'chevron-left']" />
								</button>
								<div class="carousel-container" ref="previewCarousel">
									<div
										v-for="preview in previewVideos"
										:key="preview.id"
										class="carousel-preview"
										@click="playPreview(preview)"
									>
										<div class="preview-thumbnail">
											<video :src="getAssetURL(preview.file_path)" preload="metadata" muted></video>
											<div class="play-overlay">
												<font-awesome-icon :icon="['fas', 'play-circle']" size="2x" />
											</div>
										</div>
										<div class="preview-title">{{ preview.title || `Preview ${preview.id}` }}</div>
									</div>
								</div>
								<button class="carousel-btn next" @click="scrollPreviews(1)">
									<font-awesome-icon :icon="['fas', 'chevron-right']" />
								</button>
							</div>
						</div>

						<!-- Videos Section -->
						<div class="detail-card">
							<div class="d-flex justify-content-between align-items-center mb-3">
								<h3>
									<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
									Videos ({{ filteredVideos.length }})
								</h3>
								<div class="d-flex gap-2">
									<select v-model="videoSort" class="form-select form-select-sm" style="width: auto">
										<option value="date">Sort by Date</option>
										<option value="title">Sort by Title</option>
										<option value="rating">Sort by Rating</option>
										<option value="duration">Sort by Duration</option>
									</select>
									<div class="btn-group btn-group-sm">
										<button
											:class="['btn', videoViewMode === 'grid' ? 'btn-primary' : 'btn-outline-primary']"
											@click="videoViewMode = 'grid'"
										>
											<font-awesome-icon :icon="['fas', 'th']" />
										</button>
										<button
											:class="['btn', videoViewMode === 'list' ? 'btn-primary' : 'btn-outline-primary']"
											@click="videoViewMode = 'list'"
										>
											<font-awesome-icon :icon="['fas', 'list']" />
										</button>
									</div>
								</div>
							</div>

							<!-- Loading State -->
							<div v-if="loadingVideos" class="text-center py-4">
								<div class="spinner-border spinner-border-sm text-primary" role="status">
									<span class="visually-hidden">Loading videos...</span>
								</div>
							</div>

							<!-- Grid View -->
							<div v-else-if="filteredVideos.length > 0 && videoViewMode === 'grid'" class="videos-grid">
								<div v-for="video in filteredVideos" :key="video.id" class="video-grid-card" @click="openVideo(video.id)">
									<div class="video-thumbnail">
										<img :src="getThumbnailURL(video)" :alt="video.title || video.filename" loading="lazy" @error="handleThumbnailError" />
										<div class="video-duration" v-if="video.duration">{{ formatDuration(video.duration) }}</div>
										<div class="video-rating" v-if="video.rating">
											<font-awesome-icon :icon="['fas', 'star']" class="text-warning" />
											{{ video.rating }}
										</div>
									</div>
									<div class="video-info">
										<div class="video-title">{{ video.title || video.filename }}</div>
										<div class="video-meta">
											<span v-if="video.date">
												<font-awesome-icon :icon="['fas', 'calendar']" class="me-1" />
												{{ formatDate(video.date) }}
											</span>
										</div>
									</div>
								</div>
							</div>

							<!-- List View -->
							<div v-else-if="filteredVideos.length > 0" class="video-list">
								<div v-for="video in filteredVideos" :key="video.id" class="video-item" @click="openVideo(video.id)">
									<div class="video-thumbnail">
										<img :src="getThumbnailURL(video)" :alt="video.title || video.filename" loading="lazy" @error="handleThumbnailError" />
										<div class="video-duration" v-if="video.duration">{{ formatDuration(video.duration) }}</div>
									</div>
									<div class="video-info">
										<div class="video-title">{{ video.title || video.filename }}</div>
										<div class="video-meta">
											<span v-if="video.date">
												<font-awesome-icon :icon="['fas', 'calendar']" class="me-1" />
												{{ formatDate(video.date) }}
											</span>
											<span v-if="video.rating" class="ms-2">
												<font-awesome-icon :icon="['fas', 'star']" class="me-1 text-warning" />
												{{ video.rating }}/5
											</span>
										</div>
									</div>
								</div>
							</div>

							<!-- Empty State -->
							<div v-else class="text-muted text-center py-4">
								<font-awesome-icon :icon="['fas', 'video-slash']" size="2x" class="mb-2" />
								<p>No videos featuring this performer yet</p>
							</div>
						</div>
					</div>

					<!-- Right Column - Master Tags & Stats -->
					<div class="col-lg-4">
						<!-- Statistics Card -->
						<div class="detail-card mb-4">
							<h3>
								<font-awesome-icon :icon="['fas', 'chart-bar']" class="me-2" />
								Statistics
							</h3>
							<div class="stats-grid">
								<div class="stat-box">
									<div class="stat-value">{{ performer.video_count || 0 }}</div>
									<div class="stat-label">Total Videos</div>
								</div>
								<div class="stat-box" v-if="previewVideos.length">
									<div class="stat-value">{{ previewVideos.length }}</div>
									<div class="stat-label">Preview Videos</div>
								</div>
								<div class="stat-box" v-if="stats.totalDuration">
									<div class="stat-value">{{ formatDuration(stats.totalDuration) }}</div>
									<div class="stat-label">Total Duration</div>
								</div>
								<div class="stat-box" v-if="stats.avgRating">
									<div class="stat-value">
										{{ stats.avgRating.toFixed(1) }}
										<font-awesome-icon :icon="['fas', 'star']" class="text-warning ms-1" size="sm" />
									</div>
									<div class="stat-label">Average Rating</div>
								</div>
							</div>
						</div>

						<!-- Master Tags -->
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
							<div v-else class="text-muted text-center py-3">
								<p class="mb-2">No master tags assigned</p>
								<button class="btn btn-sm btn-outline-primary" @click="showAddTagModal = true">Add Your First Tag</button>
							</div>

							<!-- Sync Button -->
							<div v-if="performerTags.length > 0" class="mt-3">
								<button class="btn btn-outline-primary w-100" @click="syncTagsToVideos" :disabled="syncing">
									<font-awesome-icon :icon="['fas', syncing ? 'spinner' : 'sync']" :spin="syncing" class="me-2" />
									{{ syncing ? 'Syncing...' : 'Sync Tags to All Videos' }}
								</button>
								<small class="text-muted d-block mt-2 text-center">
									Apply these master tags to all {{ performer.video_count || 0 }} videos
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

		<!-- Preview Player Modal -->
		<div v-if="playingPreview" class="modal-overlay" @click.self="closePreview">
			<div class="preview-player-modal">
				<div class="preview-player-header">
					<h5>{{ playingPreview.title || 'Preview Video' }}</h5>
					<button class="btn-close" @click="closePreview"></button>
				</div>
				<div class="preview-player-body">
					<video ref="previewPlayer" :src="getAssetURL(playingPreview.file_path)" controls autoplay></video>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, onMounted, computed, getCurrentInstance, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { performersAPI, tagsAPI, getAssetURL } from '@/services/api'

const route = useRoute()
const router = useRouter()
const { proxy } = getCurrentInstance()
const toast = proxy.$toast

// State
const performer = ref(null)
const performerTags = ref([])
const performerVideos = ref([])
const previewVideos = ref([])
const allTags = ref([])
const loading = ref(true)
const loadingVideos = ref(false)
const syncing = ref(false)
const fetchingMetadata = ref(false)
const generatingThumbnail = ref(false)
const saving = ref(false)
const showAddTagModal = ref(false)
const selectedTagId = ref('')
const editMode = ref(false)
const videoSort = ref('date')
const videoViewMode = ref('grid')
const previewViewMode = ref('grid')
const previewScroll = ref(0)
const playingPreview = ref(null)
const previewCarousel = ref(null)
const hoveredAvatar = ref(false)

// Edit form
const editForm = ref({
	name: '',
	category: 'regular',
})

// Computed
const availableTags = computed(() => {
	const assignedTagIds = new Set(performerTags.value.map((t) => t.id))
	return allTags.value.filter((tag) => !assignedTagIds.has(tag.id))
})

const performerMetadata = computed(() => {
	return performer.value?.metadata || null
})

const performerAge = computed(() => {
	if (!performerMetadata.value?.birthdate) return null
	const birthYear = new Date(performerMetadata.value.birthdate).getFullYear()
	const currentYear = new Date().getFullYear()
	return currentYear - birthYear
})

const appearanceData = computed(() => {
	const adlResponse = performerMetadata.value?.adult_data_link_response
	if (!adlResponse?.appearance) return null
	if (Object.keys(adlResponse.appearance).length === 0) return null
	return adlResponse.appearance
})

const hasBasicMetadata = computed(() => {
	if (!performerMetadata.value) return false
	return !!(
		performerMetadata.value.height ||
		performerMetadata.value.weight ||
		performerMetadata.value.measurements ||
		performerMetadata.value.hair_color ||
		performerMetadata.value.eye_color ||
		performerMetadata.value.ethnicity ||
		performerMetadata.value.tattoos ||
		performerMetadata.value.piercings
	)
})

const filteredVideos = computed(() => {
	let videos = [...performerVideos.value]

	// Sort videos
	switch (videoSort.value) {
		case 'date':
			videos.sort((a, b) => new Date(b.date || 0) - new Date(a.date || 0))
			break
		case 'title':
			videos.sort((a, b) => (a.title || a.filename).localeCompare(b.title || b.filename))
			break
		case 'rating':
			videos.sort((a, b) => (b.rating || 0) - (a.rating || 0))
			break
		case 'duration':
			videos.sort((a, b) => (b.duration || 0) - (a.duration || 0))
			break
	}

	return videos
})

const stats = computed(() => {
	const videos = performerVideos.value
	return {
		totalDuration: videos.reduce((sum, v) => sum + (v.duration || 0), 0),
		avgRating: videos.filter((v) => v.rating).length > 0 ? videos.reduce((sum, v) => sum + (v.rating || 0), 0) / videos.filter((v) => v.rating).length : 0,
	}
})

// Format attribute name (snake_case to Title Case)
const formatAttributeName = (key) => {
	return key
		.split('_')
		.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
		.join(' ')
}

// Load performer details
const loadPerformer = async () => {
	loading.value = true
	try {
		const performerId = route.params.id
		const response = await performersAPI.getById(performerId)
		performer.value = response.data

		// Initialize edit form
		editForm.value = {
			name: performer.value.name || '',
			category: performer.value.category || 'regular',
		}

		await Promise.all([loadPerformerTags(), loadPerformerVideos(), loadPreviewVideos()])
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

// Load videos featuring this performer
const loadPerformerVideos = async () => {
	loadingVideos.value = true
	try {
		const response = await performersAPI.getVideos(route.params.id)
		performerVideos.value = response.data || []
	} catch (error) {
		console.error('Failed to load performer videos:', error)
		performerVideos.value = []
	} finally {
		loadingVideos.value = false
	}
}

// Load preview videos
const loadPreviewVideos = async () => {
	try {
		const response = await performersAPI.getPreviews(route.params.id)
		previewVideos.value = response.data || []
	} catch (error) {
		console.error('Failed to load preview videos:', error)
		previewVideos.value = []
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

// Save edited performer details
const saveChanges = async () => {
	saving.value = true
	try {
		await performersAPI.update(route.params.id, editForm.value)
		toast.success('Success', 'Performer details updated successfully')
		editMode.value = false
		await loadPerformer()
	} catch (error) {
		console.error('Failed to save performer:', error)
		toast.error('Error', 'Failed to save performer details')
	} finally {
		saving.value = false
	}
}

// Fetch metadata from AdultDataLink
const fetchMetadata = async () => {
	if (!confirm('Fetch metadata from AdultDataLink? This will overwrite existing data.')) return

	fetchingMetadata.value = true
	try {
		await performersAPI.fetchMetadata(route.params.id)
		toast.success('Success', 'Metadata fetched successfully')
		await loadPerformer()
	} catch (error) {
		console.error('Failed to fetch metadata:', error)
		toast.error('Error', error.response?.data?.error || 'Failed to fetch metadata')
	} finally {
		fetchingMetadata.value = false
	}
}

// Reset performer metadata
const resetMetadata = async () => {
	if (!confirm('Reset all metadata for this performer? This cannot be undone.')) return

	try {
		await performersAPI.resetMetadata(route.params.id)
		toast.success('Success', 'Performer metadata reset')
		await loadPerformer()
	} catch (error) {
		console.error('Failed to reset metadata:', error)
		toast.error('Error', 'Failed to reset metadata')
	}
}

// Generate thumbnail
const generateThumbnail = async () => {
	generatingThumbnail.value = true
	try {
		await performersAPI.generateThumbnail(route.params.id)
		toast.success('Success', 'Thumbnail generated successfully')
		await loadPerformer()
	} catch (error) {
		console.error('Failed to generate thumbnail:', error)
		toast.error('Error', 'Failed to generate thumbnail')
	} finally {
		generatingThumbnail.value = false
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
	if (!confirm(`Apply master tags to all ${performer.value.video_count || 0} videos featuring this performer?`)) return

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

// Navigate to video player
const openVideo = (videoId) => {
	router.push({ name: 'VideoPlayer', params: { id: videoId } })
}

// Preview carousel scrolling
const scrollPreviews = (direction) => {
	if (previewCarousel.value) {
		const scrollAmount = 300
		previewCarousel.value.scrollBy({ left: direction * scrollAmount, behavior: 'smooth' })
	}
}

// Play preview video
const playPreview = (preview) => {
	playingPreview.value = preview
}

// Close preview player
const closePreview = () => {
	if (playingPreview.value && proxy.$refs.previewPlayer) {
		proxy.$refs.previewPlayer.pause()
	}
	playingPreview.value = null
}

// Get thumbnail URL for video
const getThumbnailURL = (video) => {
	if (video.thumbnail_path) {
		return getAssetURL(video.thumbnail_path)
	}
	return `http://localhost:8080/api/v1/videos/${video.id}/thumbnail`
}

// Format duration (seconds to HH:MM:SS or MM:SS)
const formatDuration = (seconds) => {
	if (!seconds) return '0:00'
	const hours = Math.floor(seconds / 3600)
	const mins = Math.floor((seconds % 3600) / 60)
	const secs = Math.floor(seconds % 60)

	if (hours > 0) {
		return `${hours}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
	}
	return `${mins}:${secs.toString().padStart(2, '0')}`
}

// Format date
const formatDate = (dateString) => {
	if (!dateString) return ''
	const date = new Date(dateString)
	return date.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })
}

// Handle thumbnail error
const handleThumbnailError = (event) => {
	event.target.src = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="160" height="90"%3E%3Crect fill="%23333" width="160" height="90"/%3E%3C/svg%3E'
}

// Watch for route changes (when navigating between performers)
watch(
	() => route.params.id,
	(newId, oldId) => {
		if (newId && newId !== oldId) {
			// Reset hover state when switching performers
			hoveredAvatar.value = false
			// Reload performer data
			loadPerformer()
		}
	}
)

// Lifecycle
onMounted(() => {
	loadPerformer()
	loadAllTags()
})
</script>

<style scoped>
@import '@/styles/pages/performers_details_page.css';

/* Additional styles for new features */
.breadcrumb {
	background: rgba(255, 255, 255, 0.05);
	padding: 0.75rem 1rem;
	border-radius: 8px;
	margin-bottom: 1rem;
}

.breadcrumb-item {
	color: rgba(255, 255, 255, 0.6);
}

.breadcrumb-item.active {
	color: #fff;
}

.breadcrumb-item a {
	color: #667eea;
	text-decoration: none;
	transition: color 0.2s;
}

.breadcrumb-item a:hover {
	color: #764ba2;
}

.header-content {
	display: flex;
	justify-content: space-between;
	align-items: flex-start;
	gap: 2rem;
}

.performer-avatar {
	position: relative;
	width: 150px;
	height: 150px;
	border-radius: 50%;
	overflow: hidden;
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

.avatar-image,
.avatar-video {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.avatar-placeholder {
	width: 100%;
	height: 100%;
	display: flex;
	align-items: center;
	justify-content: center;
	color: rgba(255, 255, 255, 0.5);
}

.btn-generate-thumbnail {
	position: absolute;
	bottom: 0;
	right: 0;
	width: 40px;
	height: 40px;
	border-radius: 50%;
	background: rgba(102, 126, 234, 0.9);
	border: 2px solid rgba(255, 255, 255, 0.2);
	color: #fff;
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	transition: all 0.2s;
}

.btn-generate-thumbnail:hover:not(:disabled) {
	background: #667eea;
	transform: scale(1.1);
}

.btn-generate-thumbnail:disabled {
	opacity: 0.6;
	cursor: not-allowed;
}

.quick-stats {
	display: flex;
	gap: 1.5rem;
	flex-wrap: wrap;
}

.stat-item {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	color: rgba(255, 255, 255, 0.8);
	font-size: 0.9rem;
}

.action-buttons {
	display: flex;
	gap: 0.5rem;
	flex-shrink: 0;
}

.stats-grid {
	display: grid;
	grid-template-columns: repeat(2, 1fr);
	gap: 1rem;
}

.stat-box {
	background: rgba(102, 126, 234, 0.1);
	border: 1px solid rgba(102, 126, 234, 0.3);
	border-radius: 8px;
	padding: 1rem;
	text-align: center;
}

.stat-value {
	font-size: 1.5rem;
	font-weight: 700;
	color: #fff;
	margin-bottom: 0.25rem;
}

.stat-label {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.6);
	text-transform: uppercase;
	letter-spacing: 0.5px;
}

.preview-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
	gap: 1rem;
}

.preview-card {
	cursor: pointer;
	border-radius: 8px;
	overflow: hidden;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	transition: all 0.2s;
}

.preview-card:hover {
	transform: translateY(-4px);
	border-color: rgba(102, 126, 234, 0.5);
	box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
}

.preview-thumbnail {
	position: relative;
	width: 100%;
	aspect-ratio: 16/9;
	background: #000;
	overflow: hidden;
}

.preview-thumbnail video {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.play-overlay {
	position: absolute;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.5);
	display: flex;
	align-items: center;
	justify-content: center;
	color: #fff;
	opacity: 0;
	transition: opacity 0.2s;
}

.preview-card:hover .play-overlay {
	opacity: 1;
}

.preview-duration {
	position: absolute;
	bottom: 8px;
	right: 8px;
	background: rgba(0, 0, 0, 0.8);
	color: #fff;
	padding: 4px 8px;
	border-radius: 4px;
	font-size: 0.75rem;
	font-weight: 600;
}

.preview-title {
	padding: 0.75rem;
	font-size: 0.9rem;
	font-weight: 500;
	color: #fff;
	text-align: center;
}

.preview-carousel {
	position: relative;
	display: flex;
	align-items: center;
	gap: 1rem;
}

.carousel-container {
	display: flex;
	gap: 1rem;
	overflow-x: auto;
	scroll-behavior: smooth;
	padding: 0.5rem 0;
	scrollbar-width: none;
}

.carousel-container::-webkit-scrollbar {
	display: none;
}

.carousel-preview {
	flex-shrink: 0;
	width: 250px;
	cursor: pointer;
	border-radius: 8px;
	overflow: hidden;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	transition: all 0.2s;
}

.carousel-preview:hover {
	border-color: rgba(102, 126, 234, 0.5);
	transform: scale(1.05);
}

.carousel-btn {
	background: rgba(102, 126, 234, 0.8);
	border: none;
	color: #fff;
	width: 40px;
	height: 40px;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	transition: all 0.2s;
	flex-shrink: 0;
}

.carousel-btn:hover:not(:disabled) {
	background: #667eea;
	transform: scale(1.1);
}

.carousel-btn:disabled {
	opacity: 0.3;
	cursor: not-allowed;
}

.videos-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
	gap: 1rem;
}

.video-grid-card {
	cursor: pointer;
	border-radius: 8px;
	overflow: hidden;
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.1);
	transition: all 0.2s;
}

.video-grid-card:hover {
	background: rgba(255, 255, 255, 0.08);
	border-color: rgba(102, 126, 234, 0.5);
	transform: translateY(-4px);
	box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
}

.video-grid-card .video-thumbnail {
	position: relative;
	width: 100%;
	aspect-ratio: 16/9;
	background: rgba(255, 255, 255, 0.05);
}

.video-grid-card .video-thumbnail img {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.video-rating {
	position: absolute;
	top: 8px;
	left: 8px;
	background: rgba(0, 0, 0, 0.8);
	color: #fff;
	padding: 4px 8px;
	border-radius: 4px;
	font-size: 0.75rem;
	font-weight: 600;
	display: flex;
	align-items: center;
	gap: 4px;
}

.video-grid-card .video-info {
	padding: 0.75rem;
}

.preview-player-modal {
	background: #1e1e2e;
	border-radius: 12px;
	max-width: 90vw;
	max-height: 90vh;
	overflow: hidden;
	box-shadow: 0 20px 60px rgba(0, 0, 0, 0.8);
}

.preview-player-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1rem 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.preview-player-header h5 {
	margin: 0;
	font-size: 1.1rem;
	color: #fff;
}

.preview-player-body {
	padding: 0;
}

.preview-player-body video {
	width: 100%;
	max-height: 80vh;
	display: block;
}

.form-check-input {
	background-color: rgba(255, 255, 255, 0.1);
	border-color: rgba(255, 255, 255, 0.3);
}

.form-check-input:checked {
	background-color: #667eea;
	border-color: #667eea;
}

.form-check-label {
	color: rgba(255, 255, 255, 0.9);
}

@media (max-width: 1200px) {
	.header-content {
		flex-direction: column;
	}

	.action-buttons {
		width: 100%;
		justify-content: flex-start;
	}

	.stats-grid {
		grid-template-columns: repeat(2, 1fr);
	}
}

@media (max-width: 768px) {
	.videos-grid,
	.preview-grid {
		grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
	}

	.quick-stats {
		flex-direction: column;
		gap: 0.75rem;
	}

	.action-buttons {
		flex-direction: column;
	}

	.action-buttons .btn {
		width: 100%;
	}
}
</style>
