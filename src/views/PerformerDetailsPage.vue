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
			<LoadingState v-if="loading" />

			<!-- Content -->
			<div v-else-if="performer" class="performer-content">
				<!-- Hero Section -->
				<div class="performer-hero">
					<div class="hero-content">
						<!-- Large Avatar -->
						<div class="performer-avatar" @mouseenter="startAvatarVideo" @mouseleave="stopAvatarVideo">
							<video
								v-if="performer.preview_path"
								ref="avatarVideo"
								:key="`video-${performer.id}`"
								:src="getAssetURL(performer.preview_path)"
								loop
								muted
								playsinline
								autoplay
								class="avatar-video"
								:poster="performer.thumbnail_path ? getAssetURL(performer.thumbnail_path) : ''"
							></video>
							<div v-else class="avatar-placeholder">
								<font-awesome-icon :icon="['fas', 'user']" size="4x" />
								<button class="btn-generate-thumbnail" @click="generateThumbnail" :disabled="generatingThumbnail" title="Generate Thumbnail">
									<font-awesome-icon :icon="['fas', generatingThumbnail ? 'spinner' : 'camera']" :spin="generatingThumbnail" />
								</button>
							</div>
						</div>

						<!-- Hero Info -->
						<div class="hero-info">
							<div class="col">
								<div class="d-flex align-items-center gap-3 mb-3">
									<h1>{{ performer.name }}</h1>
									<button class="btn btn-outline-light" @click="editMode = !editMode">
										<font-awesome-icon :icon="['fas', editMode ? 'times' : 'edit']" class="me-2" />
										{{ editMode ? 'Cancel' : 'Edit' }}
									</button>
								</div>

								<!-- Badges -->
								<div class="d-flex gap-2 mb-3 flex-wrap">
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
								<div class="quick-stats">
									<div class="stat-item">
										<font-awesome-icon :icon="['fas', 'video']" />
										<strong>{{ performer.video_count || 0 }}</strong> Videos
									</div>
									<div class="stat-item" v-if="stats.totalDuration">
										<font-awesome-icon :icon="['fas', 'clock']" />
										<strong>{{ formatDuration(stats.totalDuration) }}</strong> Total
									</div>
									<div class="stat-item" v-if="stats.avgRating">
										<font-awesome-icon :icon="['fas', 'star']" class="text-warning" />
										<strong>{{ stats.avgRating.toFixed(1) }}</strong> Avg Rating
									</div>
									<div class="stat-item" v-if="previewVideos.length">
										<font-awesome-icon :icon="['fas', 'play-circle']" />
										<strong>{{ previewVideos.length }}</strong> Previews
									</div>
								</div>

								<!-- Action Buttons -->
								<div class="action-buttons mt-3">
									<button class="btn btn-primary" @click="fetchMetadata" :disabled="fetchingMetadata">
										<font-awesome-icon :icon="['fas', fetchingMetadata ? 'spinner' : 'cloud-download-alt']" :spin="fetchingMetadata" class="me-2" />
										Fetch Metadata
									</button>
									<button class="btn btn-danger" @click="resetMetadata">
										<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
										Reset Metadata
									</button>
								</div>
							</div>
							<div class="col"></div>
						</div>
					</div>
				</div>

				<!-- Stats Grid -->
				<div class="stats-grid">
					<StatCard :value="performer.video_count || 0" label="Total Videos" />
					<StatCard :value="previewVideos.length" label="Preview Videos" />
					<StatCard :value="stats.totalDuration ? formatDuration(stats.totalDuration) : '0:00'" label="Total Duration" />
					<StatCard :value="stats.avgRating ? stats.avgRating.toFixed(1) : 'N/A'" label="Avg Rating" />
				</div>

				<!-- Preview Videos Section - Full Width & Prominent -->
				<div v-if="previewVideos.length > 0" class="preview-section">
					<div class="view-controls">
						<h3>
							<font-awesome-icon :icon="['fas', 'play-circle']" class="me-2" />
							Preview Videos ({{ previewVideos.length }})
						</h3>
						<div class="btn-group">
							<button :class="['btn', previewViewMode === 'grid' ? 'btn-primary' : 'btn-outline-primary']" @click="previewViewMode = 'grid'">
								<font-awesome-icon :icon="['fas', 'th']" />
								Grid
							</button>
							<button :class="['btn', previewViewMode === 'carousel' ? 'btn-primary' : 'btn-outline-primary']" @click="previewViewMode = 'carousel'">
								<font-awesome-icon :icon="['fas', 'ellipsis-h']" />
								Carousel
							</button>
						</div>
					</div>

					<!-- Grid View -->
					<div v-if="previewViewMode === 'grid'" class="preview-grid">
						<div v-for="preview in previewVideos" :key="preview.id" class="preview-card" @click="playPreview(preview)">
							<div class="preview-thumbnail">
								<video :src="getAssetURL(preview.file_path)" preload="metadata" muted playsinline></video>
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
						<button class="carousel-btn prev" @click="scrollPreviews(-1)">
							<font-awesome-icon :icon="['fas', 'chevron-left']" />
						</button>
						<div class="carousel-container" ref="previewCarousel">
							<div v-for="preview in previewVideos" :key="preview.id" class="carousel-preview" @click="playPreview(preview)">
								<div class="preview-thumbnail">
									<video :src="getAssetURL(preview.file_path)" preload="metadata" muted playsinline></video>
									<div class="play-overlay">
										<font-awesome-icon :icon="['fas', 'play-circle']" size="2x" />
									</div>
									<div class="preview-duration" v-if="preview.duration">{{ formatDuration(preview.duration) }}</div>
								</div>
								<div class="preview-title">{{ preview.title || `Preview ${preview.id}` }}</div>
							</div>
						</div>
						<button class="carousel-btn next" @click="scrollPreviews(1)">
							<font-awesome-icon :icon="['fas', 'chevron-right']" />
						</button>
					</div>
				</div>

				<!-- Main Content Grid -->
				<div class="content-grid">
					<!-- Left Column - Master Tags -->
					<div>
						<!-- Edit Form -->
						<div v-if="editMode" class="detail-card">
							<h3>
								<font-awesome-icon :icon="['fas', 'edit']" />
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

						<!-- Appearance Data -->
						<div v-else class="detail-card">
							<h3>
								<font-awesome-icon :icon="['fas', 'user-circle']" />
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
						<!-- Master Tags -->
					</div>

					<!-- Right Column - Details & Videos -->
					<div>
						<!-- Videos Section -->
						<div class="detail-card videos-section">
							<h3>
								<font-awesome-icon :icon="['fas', 'video']" />
								Videos ({{ performerVideos.length }})
								<div class="d-flex gap-2 ms-auto">
									<select v-model="videoSort" class="form-select form-select-sm" style="width: auto">
										<option value="date">Sort by Date</option>
										<option value="title">Sort by Title</option>
										<option value="rating">Sort by Rating</option>
										<option value="duration">Sort by Duration</option>
									</select>
									<div class="btn-group btn-group-sm">
										<button :class="['btn', videoViewMode === 'grid' ? 'btn-primary' : 'btn-outline-primary']" @click="videoViewMode = 'grid'">
											<font-awesome-icon :icon="['fas', 'th']" />
										</button>
										<button :class="['btn', videoViewMode === 'list' ? 'btn-primary' : 'btn-outline-primary']" @click="videoViewMode = 'list'">
											<font-awesome-icon :icon="['fas', 'list']" />
										</button>
									</div>
								</div>
							</h3>

							<!-- Loading State -->
							<LoadingState v-if="loadingVideos" :padding="4" />

							<!-- Grid View -->
							<div v-else-if="videoViewMode === 'grid'" class="videos-grid">
								<div v-for="video in filteredVideos" :key="video.id" class="video-grid-card" @click="openVideo(video.id)">
									<div class="video-thumbnail">
										<img :src="getThumbnailURL(video)" :alt="video.title || video.filename" @error="handleThumbnailError" />
										<div class="video-rating" v-if="video.rating">
											<font-awesome-icon :icon="['fas', 'star']" />
											{{ video.rating }}
										</div>
										<div class="video-duration" v-if="video.duration">{{ formatDuration(video.duration) }}</div>
									</div>
									<div class="video-info">
										<div class="video-title">{{ video.title || video.filename }}</div>
										<div class="video-meta">
											<span v-if="video.date">{{ formatDate(video.date) }}</span>
											<span v-if="video.duration">{{ formatDuration(video.duration) }}</span>
										</div>
									</div>
								</div>
							</div>

							<!-- List View -->
							<div v-else class="video-list">
								<div v-for="video in filteredVideos" :key="video.id" class="video-item" @click="openVideo(video.id)">
									<div class="video-thumbnail">
										<img :src="getThumbnailURL(video)" :alt="video.title || video.filename" @error="handleThumbnailError" />
										<div class="video-duration" v-if="video.duration">{{ formatDuration(video.duration) }}</div>
									</div>
									<div class="video-info">
										<div class="video-title">{{ video.title || video.filename }}</div>
										<div class="video-meta">
											<span v-if="video.date">
												<font-awesome-icon :icon="['fas', 'calendar']" />
												{{ formatDate(video.date) }}
											</span>
											<span v-if="video.duration">
												<font-awesome-icon :icon="['fas', 'clock']" />
												{{ formatDuration(video.duration) }}
											</span>
											<span v-if="video.rating">
												<font-awesome-icon :icon="['fas', 'star']" />
												{{ video.rating }} / 5
											</span>
										</div>
									</div>
								</div>
							</div>

							<!-- Empty State -->
							<EmptyState v-if="!loadingVideos && performerVideos.length === 0" :icon="['fas', 'video-slash']" message="No videos found for this performer" />
						</div>

						<div class="detail-card">
							<h3>
								<font-awesome-icon :icon="['fas', 'tags']" />
								Master Tags
							</h3>
							<div class="tags-info mb-3">
								<p class="small mb-0">Master tags are automatically applied to all videos featuring this performer</p>
							</div>

							<!-- Tag List -->
							<div v-if="performerTags.length > 0" class="tag-list mb-3">
								<div v-for="tag in performerTags" :key="tag.id" class="tag-item">
									<span class="tag-name">
										<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" class="me-2" />
										{{ tag.name }}
									</span>
									<button class="btn-remove-tag" @click="removeTag(tag.id)" title="Remove tag">
										<font-awesome-icon :icon="['fas', 'times']" />
									</button>
								</div>
							</div>
							<div v-else class="text-muted text-center py-3">
								<p class="mb-0">No master tags assigned</p>
							</div>

							<button class="btn btn-outline-primary w-100 mb-3" @click="showAddTagModal = true">
								<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
								Add Master Tag
							</button>

							<!-- Sync Button -->
							<div v-if="performerTags.length > 0">
								<button class="btn btn-outline-primary w-100" @click="syncTagsToVideos" :disabled="syncing">
									<font-awesome-icon :icon="['fas', syncing ? 'spinner' : 'sync']" :spin="syncing" class="me-2" />
									{{ syncing ? 'Syncing...' : 'Sync Tags to All Videos' }}
								</button>
								<small class="text-muted d-block mt-2 text-center"> Apply these master tags to all {{ performer.video_count || 0 }} videos </small>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Error State -->
			<div v-else class="text-center py-5">
				<p class="text-danger">Failed to load performer details</p>
				<button class="btn btn-primary" @click="loadPerformer">Try Again</button>
			</div>
		</div>

		<!-- Add Tag Modal -->
		<div v-if="showAddTagModal" class="modal-overlay" @click.self="showAddTagModal = false">
			<div class="modal-dialog">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							<font-awesome-icon :icon="['fas', 'tag']" class="me-2" />
							Add Master Tag
						</h5>
						<button class="btn-close" @click="showAddTagModal = false"></button>
					</div>
					<div class="modal-body">
						<label class="form-label">Select Tag</label>
						<select v-model="selectedTagId" class="form-select">
							<option value="">Choose a tag...</option>
							<option v-for="tag in availableTags" :key="tag.id" :value="tag.id">{{ tag.name }}</option>
						</select>
					</div>
					<div class="modal-footer">
						<button class="btn btn-secondary" @click="showAddTagModal = false">Cancel</button>
						<button class="btn btn-primary" @click="addTag" :disabled="!selectedTagId">Add Tag</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Preview Player Modal -->
		<div v-if="playingPreview" class="modal-overlay" @click.self="closePreview">
			<div class="preview-player-modal">
				<div class="preview-player-header">
					<h5>{{ playingPreview.title || `Preview ${playingPreview.id}` }}</h5>
					<button class="btn-close" @click="closePreview"></button>
				</div>
				<div class="preview-player-body">
					<video ref="previewPlayer" :src="getAssetURL(playingPreview.file_path)" controls></video>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, onMounted, computed, getCurrentInstance, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { performersAPI, tagsAPI, getAssetURL } from '@/services/api'
import { useFormatters } from '@/composables/useFormatters'
import { LoadingState, EmptyState, StatCard } from '@/components/shared'

const route = useRoute()
const router = useRouter()
const { proxy } = getCurrentInstance()
const toast = proxy.$toast
const { formatDuration, formatDate } = useFormatters()

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
const playingPreview = ref(null)
const previewCarousel = ref(null)
const avatarVideo = ref(null)

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

// formatDuration and formatDate now provided by useFormatters composable

// Handle thumbnail error
const handleThumbnailError = (event) => {
	event.target.src = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="160" height="90"%3E%3Crect fill="%23333" width="160" height="90"/%3E%3C/svg%3E'
}

// Start avatar video on hover
const startAvatarVideo = () => {
	if (avatarVideo.value) {
		avatarVideo.value.play()
	}
}

// Stop avatar video when not hovering
const stopAvatarVideo = () => {
	if (avatarVideo.value) {
		avatarVideo.value.pause()
		avatarVideo.value.currentTime = 0
	}
}

// Watch for route changes (when navigating between performers)
watch(
	() => route.params.id,
	(newId, oldId) => {
		if (newId && newId !== oldId) {
			// Stop avatar video when switching performers
			stopAvatarVideo()
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
</style>
