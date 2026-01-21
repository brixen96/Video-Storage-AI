<template>
	<div class="ai-feature-card">
		<div class="feature-icon">
			<font-awesome-icon :icon="['fas', 'exclamation']" />
		</div>
		<h3>Missing Metadata</h3>
		<p>Find videos with incomplete or missing metadata.</p>

		<div class="feature-stats" v-if="metadataStats">
			<div class="stat">
				<span class="stat-label">Videos Checked:</span>
				<span class="stat-value">{{ metadataStats.videosChecked || 0 }}</span>
			</div>
			<div class="stat">
				<span class="stat-label">Issues Found:</span>
				<span class="stat-value">{{ metadataStats.issuesFound || 0 }}</span>
			</div>
		</div>

		<div class="feature-controls mt-4">
			<button class="btn btn-primary btn-lg" @click="startMissingMetadataDetection" :disabled="isDetectingMetadata">
				<font-awesome-icon :icon="['fas', isDetectingMetadata ? 'spinner' : 'play']" :spin="isDetectingMetadata" class="me-2" />
				{{ isDetectingMetadata ? 'Analyzing...' : 'Start Analysis' }}
			</button>
		</div>

		<!-- Missing Metadata Results Display -->
		<div v-if="metadataResults.length > 0" class="suggestions-panel mt-4">
			<div class="suggestions-header">
				<h4>Missing Metadata ({{ filteredMetadataResults.length }} videos)</h4>
				<div class="suggestions-actions">
					<button class="btn btn-sm btn-outline-info me-2" @click="exportToCSV(filteredMetadataResults, 'missing-metadata')">
						<font-awesome-icon :icon="['fas', 'file-export']" class="me-1" />
						CSV
					</button>
					<button class="btn btn-sm btn-outline-info" @click="exportToJSON(filteredMetadataResults, 'missing-metadata')">
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
					<input v-model="metadataSearch" type="text" class="form-control" placeholder="Search by video title or missing field..." />
					<button v-if="metadataSearch" class="btn btn-outline-secondary" @click="metadataSearch = ''">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
			</div>

			<div class="suggestions-list">
				<div v-for="result in filteredMetadataResults.slice(0, 10)" :key="result.video_id" class="suggestion-item">
					<div class="video-info">
						<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
						<router-link :to="{ path: '/videos', query: { search: result.video_title } }" class="video-link">
							<strong>{{ result.video_title }}</strong>
						</router-link>
					</div>
					<div class="metadata-info mt-2">
						<span class="badge me-2" :class="result.severity === 'high' ? 'bg-danger' : result.severity === 'medium' ? 'bg-warning' : 'bg-info'">
							{{ result.severity }} severity
						</span>
						<div class="mt-1">
							<small class="text-light d-block">Missing: {{ result.missing_fields.join(', ') }}</small>
							<small v-for="(suggestion, idx) in result.suggestions.slice(0, 2)" :key="idx" class="text-info d-block">
								<font-awesome-icon :icon="['fas', 'info-circle']" class="me-1" />
								{{ suggestion }}
							</small>
						</div>
					</div>
				</div>
			</div>
			<p v-if="filteredMetadataResults.length > 10" class="text-light text-center mt-2">Showing 10 of {{ filteredMetadataResults.length }} results</p>
		</div>

		<!-- Empty State -->
		<div v-else-if="metadataStats && metadataResults.length === 0" class="empty-state mt-4">
			<font-awesome-icon :icon="['fas', 'check-circle']" class="empty-state-icon text-success" />
			<p class="empty-state-text">All videos have complete metadata! Your library is well organized.</p>
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
const isDetectingMetadata = ref(false)
const metadataStats = ref(null)
const metadataResults = ref([])
const metadataSearch = ref('')

// Computed - Filtered metadata results based on search
const filteredMetadataResults = computed(() => {
	if (!metadataSearch.value) return metadataResults.value
	const searchLower = metadataSearch.value.toLowerCase()
	return metadataResults.value.filter((result) => {
		const videoMatch = result.video_title.toLowerCase().includes(searchLower)
		const fieldMatch = result.missing_fields.some((f) => f.toLowerCase().includes(searchLower))
		return videoMatch || fieldMatch
	})
})

// Main function - Start missing metadata detection
const startMissingMetadataDetection = async () => {
	isDetectingMetadata.value = true
	toast.info('Analyzing', 'Checking for missing metadata...')

	try {
		const response = await aiAPI.detectMissingMetadata({
			video_ids: [],
		})

		metadataResults.value = response.data.results || []
		metadataStats.value = {
			videosChecked: metadataResults.value.length,
			issuesFound: metadataResults.value.length,
		}

		toast.success('Metadata Check Complete', `Checked ${metadataStats.value.videosChecked} videos. Found ${metadataStats.value.issuesFound} with missing metadata.`)
	} catch (error) {
		console.error('Missing metadata detection failed:', error)
		toast.error('Analysis Failed', 'Could not detect missing metadata')
	} finally {
		isDetectingMetadata.value = false
	}
}
</script>

<style scoped>
/* Component will use styles from parent ai_page.css */
</style>
