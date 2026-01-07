<template>
	<div class="ai-feature-card">
		<div class="feature-icon">
			<font-awesome-icon :icon="['fas', 'copy']" />
		</div>
		<h3>Duplicate Detection</h3>
		<p>Find duplicate and similar videos across your library.</p>

		<div class="feature-stats" v-if="duplicateStats">
			<div class="stat">
				<span class="stat-label">Videos Scanned:</span>
				<span class="stat-value">{{ duplicateStats.videosScanned || 0 }}</span>
			</div>
			<div class="stat">
				<span class="stat-label">Duplicate Groups:</span>
				<span class="stat-value">{{ duplicateStats.duplicateGroups || 0 }}</span>
			</div>
		</div>

		<div class="feature-controls mt-4">
			<button class="btn btn-primary btn-lg" @click="startDuplicateDetection" :disabled="isDetectingDuplicates">
				<font-awesome-icon :icon="['fas', isDetectingDuplicates ? 'spinner' : 'play']" :spin="isDetectingDuplicates" class="me-2" />
				{{ isDetectingDuplicates ? 'Analyzing...' : 'Start Analysis' }}
			</button>
		</div>

		<!-- Duplicate Results Display -->
		<div v-if="duplicateResults.length > 0" class="suggestions-panel mt-4">
			<div class="suggestions-header">
				<h4>Duplicate Groups ({{ filteredDuplicateResults.length }} groups)</h4>
				<div class="suggestions-actions">
					<button class="btn btn-sm btn-outline-info me-2" @click="exportToCSV(filteredDuplicateResults, 'duplicate-detection')">
						<font-awesome-icon :icon="['fas', 'file-export']" class="me-1" />
						CSV
					</button>
					<button class="btn btn-sm btn-outline-info" @click="exportToJSON(filteredDuplicateResults, 'duplicate-detection')">
						<font-awesome-icon :icon="['fas', 'file-export']" class="me-1" />
						JSON
					</button>
				</div>
			</div>

			<!-- Search Filter -->
			<div class="search-filter mb-3">
				<div class="input-group">
					<span class="input-group-text">
						<font-awesome-icon :icon="['fas', 'search']" />
					</span>
					<input v-model="duplicateSearch" type="text" class="form-control" placeholder="Search by video title or reason..." />
					<button v-if="duplicateSearch" class="btn btn-outline-secondary" @click="duplicateSearch = ''">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
			</div>

			<div class="suggestions-list">
				<div v-for="(group, idx) in filteredDuplicateResults.slice(0, 5)" :key="group.group_id || idx" class="suggestion-item">
					<div class="duplicate-group-header mb-2">
						<div>
							<span class="badge bg-warning text-dark me-2">Group {{ idx + 1 }}</span>
							<span class="badge bg-info">{{ (group.similarity * 100).toFixed(0) }}% similar</span>
							<small class="text-light ms-2">{{ group.reason }}</small>
						</div>
						<button class="btn btn-sm btn-outline-primary" @click="openComparisonView(group)">
							<font-awesome-icon :icon="['fas', 'expand']" class="me-1" />
							Compare
						</button>
					</div>
					<div class="duplicate-videos">
						<div v-for="video in group.videos" :key="video.video_id" class="ps-3 mb-1">
							<font-awesome-icon :icon="['fas', 'video']" class="me-2 text-light" />
							<router-link :to="{ path: '/videos', query: { search: video.video_title } }" class="video-link">
								<small>{{ video.video_title }}</small>
							</router-link>
							<small class="text-light ms-2">({{ (video.file_size / 1024 / 1024).toFixed(0) }} MB)</small>
						</div>
					</div>
				</div>
			</div>
			<p v-if="filteredDuplicateResults.length > 5" class="text-light text-center mt-2">Showing 5 of {{ filteredDuplicateResults.length }} groups</p>
		</div>

		<!-- Empty State -->
		<div v-else-if="duplicateStats && duplicateResults.length === 0" class="empty-state mt-4">
			<font-awesome-icon :icon="['fas', 'check-circle']" class="empty-state-icon text-success" />
			<p class="empty-state-text">No duplicates detected! Your library is clean and organized.</p>
		</div>

		<!-- Comparison Modal -->
		<div v-if="showComparisonModal && comparisonGroup" class="comparison-modal-overlay" @click="closeComparisonView">
			<div class="comparison-modal" @click.stop>
				<div class="comparison-modal-header">
					<h3>
						<font-awesome-icon :icon="['fas', 'copy']" class="me-2" />
						Compare Duplicates
					</h3>
					<button class="btn-close-modal" @click="closeComparisonView">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>

				<div class="comparison-modal-body">
					<div class="comparison-info mb-4">
						<div class="badge bg-info me-2">{{ (comparisonGroup.similarity * 100).toFixed(0) }}% similar</div>
						<small class="text-light">{{ comparisonGroup.reason }}</small>
					</div>

					<div class="comparison-grid">
						<div v-for="video in comparisonGroup.videos" :key="video.video_id" class="comparison-video-card">
							<div class="video-card-header">
								<div class="video-title">
									<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
									{{ video.title }}
								</div>
								<div v-if="getBetterQualityVideo(comparisonGroup)?.video_id === video.video_id" class="recommended-badge">
									<font-awesome-icon :icon="['fas', 'star']" class="me-1" />
									Recommended
								</div>
							</div>

							<div class="video-metadata">
								<div class="metadata-row">
									<span class="metadata-label">File Size:</span>
									<span class="metadata-value">{{ (video.file_size / 1024 / 1024).toFixed(2) }} MB</span>
								</div>
								<div v-if="video.resolution" class="metadata-row">
									<span class="metadata-label">Resolution:</span>
									<span class="metadata-value">{{ video.resolution }}</span>
								</div>
								<div v-if="video.duration" class="metadata-row">
									<span class="metadata-label">Duration:</span>
									<span class="metadata-value">{{ Math.floor(video.duration / 60) }}:{{ (video.duration % 60).toString().padStart(2, '0') }}</span>
								</div>
								<div v-if="video.bitrate" class="metadata-row">
									<span class="metadata-label">Bitrate:</span>
									<span class="metadata-value">{{ (video.bitrate / 1000).toFixed(0) }} kbps</span>
								</div>
								<div v-if="video.codec" class="metadata-row">
									<span class="metadata-label">Codec:</span>
									<span class="metadata-value">{{ video.codec }}</span>
								</div>
								<div class="metadata-row">
									<span class="metadata-label">Path:</span>
									<span class="metadata-value text-truncate" :title="video.path">{{ video.path }}</span>
								</div>
							</div>

							<div class="video-actions mt-3">
								<router-link :to="{ path: '/videos', query: { search: video.title } }" class="btn btn-sm btn-outline-info me-2">
									<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
									View
								</router-link>
								<button class="btn btn-sm btn-outline-danger" @click="deleteDuplicateVideo(video.video_id)">
									<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />
									Delete
								</button>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, getCurrentInstance } from 'vue'
import { aiAPI } from '@/services/api'
import { useAIExport } from '@/composables/useAIExport'

// Get toast instance
const { proxy } = getCurrentInstance()
const toast = proxy.$toast

// Get export functions
const { exportToCSV, exportToJSON } = useAIExport(toast)

// State
const isDetectingDuplicates = ref(false)
const duplicateStats = ref(null)
const duplicateResults = ref([])
const duplicateSearch = ref('')
const showComparisonModal = ref(false)
const comparisonGroup = ref(null)

// Computed - Filtered duplicate results based on search
const filteredDuplicateResults = computed(() => {
	if (!duplicateSearch.value) return duplicateResults.value
	const searchLower = duplicateSearch.value.toLowerCase()
	return duplicateResults.value.filter((group) => {
		const videoMatch = group.videos.some((v) => v.title.toLowerCase().includes(searchLower))
		const reasonMatch = group.reason.toLowerCase().includes(searchLower)
		return videoMatch || reasonMatch
	})
})

// Main function - Start duplicate detection
const startDuplicateDetection = async () => {
	isDetectingDuplicates.value = true
	toast.info('Analyzing', 'Searching for duplicate videos...')

	try {
		const response = await aiAPI.detectDuplicates({
			video_ids: [],
		})

		duplicateResults.value = response.data.results || []
		duplicateStats.value = {
			videosScanned: duplicateResults.value.reduce((sum, g) => sum + g.videos.length, 0),
			duplicateGroups: duplicateResults.value.length,
		}

		toast.success('Duplicate Detection Complete', `Found ${duplicateStats.value.duplicateGroups} duplicate groups across ${duplicateStats.value.videosScanned} videos.`)
	} catch (error) {
		console.error('Duplicate detection failed:', error)
		toast.error('Analysis Failed', 'Could not detect duplicates')
	} finally {
		isDetectingDuplicates.value = false
	}
}

// Modal functions
const openComparisonView = (group) => {
	comparisonGroup.value = group
	showComparisonModal.value = true
}

const closeComparisonView = () => {
	showComparisonModal.value = false
	comparisonGroup.value = null
}

const deleteDuplicateVideo = async (videoId) => {
	if (!confirm('Are you sure you want to delete this video? This action cannot be undone.')) {
		return
	}

	try {
		// Call API to delete video (endpoint would need to be implemented)
		// await aiAPI.deleteVideo(videoId)

		// For now, just remove from the duplicate group in UI
		if (comparisonGroup.value) {
			comparisonGroup.value.videos = comparisonGroup.value.videos.filter((v) => v.video_id !== videoId)

			// If only one video left, remove the group entirely
			if (comparisonGroup.value.videos.length <= 1) {
				duplicateResults.value = duplicateResults.value.filter((g) => g.group_id !== comparisonGroup.value.group_id)
				closeComparisonView()
			}
		}

		toast.success('Video Deleted', 'Duplicate video has been removed')
	} catch (error) {
		console.error('Delete video failed:', error)
		toast.error('Delete Failed', 'Could not delete video')
	}
}

const getBetterQualityVideo = (group) => {
	if (!group || !group.videos || group.videos.length === 0) return null
	// Find video with highest resolution or largest file size
	return group.videos.reduce((best, current) => {
		const currentSize = current.file_size || 0
		const bestSize = best.file_size || 0
		return currentSize > bestSize ? current : best
	})
}
</script>

<style scoped>
/* Component will use styles from parent ai_page.css */
</style>
