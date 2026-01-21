<template>
	<div class="ai-feature-card">
		<div class="feature-icon">
			<font-awesome-icon :icon="['fas', 'magic']" />
		</div>
		<h3>Quality Analysis</h3>
		<p>Analyze video quality metrics and detect issues.</p>

		<div class="feature-stats" v-if="qualityStats">
			<div class="stat">
				<span class="stat-label">Videos Analyzed:</span>
				<span class="stat-value">{{ qualityStats.videosAnalyzed || 0 }}</span>
			</div>
			<div class="stat">
				<span class="stat-label">Issues Found:</span>
				<span class="stat-value">{{ qualityStats.issuesFound || 0 }}</span>
			</div>
		</div>

		<div class="feature-controls mt-4">
			<button class="btn btn-primary btn-lg" @click="startQualityAnalysis" :disabled="isAnalyzingQuality">
				<font-awesome-icon :icon="['fas', isAnalyzingQuality ? 'spinner' : 'play']" :spin="isAnalyzingQuality" class="me-2" />
				{{ isAnalyzingQuality ? 'Analyzing...' : 'Start Analysis' }}
			</button>
		</div>

		<!-- Quality Results Display -->
		<div v-if="qualityResults.length > 0" class="suggestions-panel mt-4">
			<div class="suggestions-header">
				<h4>Quality Issues Found ({{ filteredQualityResults.filter((r) => r.issues.length > 0).length }} videos)</h4>
				<div class="suggestions-actions">
					<button class="btn btn-sm btn-outline-info me-2" @click="exportToCSV(filteredQualityResults, 'quality-analysis')">
						<font-awesome-icon :icon="['fas', 'file-export']" class="me-1" />
						CSV
					</button>
					<button class="btn btn-sm btn-outline-info" @click="exportToJSON(filteredQualityResults, 'quality-analysis')">
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
					<input v-model="qualitySearch" type="text" class="form-control" placeholder="Search by video title or issue..." />
					<button v-if="qualitySearch" class="btn btn-outline-secondary" @click="qualitySearch = ''">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
			</div>

			<div class="suggestions-list">
				<div v-for="result in filteredQualityResults.filter((r) => r.issues.length > 0).slice(0, 10)" :key="result.video_id" class="suggestion-item">
					<div class="video-info">
						<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
						<router-link :to="{ path: '/videos', query: { search: result.video_title } }" class="video-link">
							<strong>{{ result.video_title }}</strong>
						</router-link>
					</div>
					<div class="quality-info mt-2">
						<span class="badge bg-primary me-2">{{ result.resolution }}</span>
						<span class="badge me-2" :class="result.quality_score >= 0.8 ? 'bg-success' : result.quality_score >= 0.6 ? 'bg-warning' : 'bg-danger'">
							Score: {{ (result.quality_score * 100).toFixed(0) }}%
						</span>
						<div class="mt-1">
							<small v-for="(issue, idx) in result.issues" :key="idx" class="text-warning d-block">
								<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="me-1" />
								{{ issue }}
							</small>
						</div>
					</div>
				</div>
			</div>
			<p v-if="filteredQualityResults.filter((r) => r.issues.length > 0).length > 10" class="text-light text-center mt-2">
				Showing 10 of {{ filteredQualityResults.filter((r) => r.issues.length > 0).length }} results
			</p>
		</div>

		<!-- Empty State -->
		<div v-else-if="qualityStats && qualityResults.length === 0" class="empty-state mt-4">
			<font-awesome-icon :icon="['fas', 'magic']" class="empty-state-icon" />
			<p class="empty-state-text">No quality issues found. Your videos are in good condition!</p>
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
const isAnalyzingQuality = ref(false)
const qualityStats = ref(null)
const qualityResults = ref([])
const qualitySearch = ref('')

// Computed - Filtered quality results based on search
const filteredQualityResults = computed(() => {
	if (!qualitySearch.value) return qualityResults.value
	const searchLower = qualitySearch.value.toLowerCase()
	return qualityResults.value.filter((result) => {
		const videoMatch = result.video_title.toLowerCase().includes(searchLower)
		const issueMatch = result.issues.some((i) => i.toLowerCase().includes(searchLower))
		return videoMatch || issueMatch
	})
})

// Main function - Start quality analysis
const startQualityAnalysis = async () => {
	isAnalyzingQuality.value = true
	toast.info('Analyzing', 'Analyzing video quality...')

	try {
		const response = await aiAPI.analyzeQuality({
			video_ids: [],
		})

		qualityResults.value = response.data.results || []
		qualityStats.value = {
			videosAnalyzed: qualityResults.value.length,
			issuesFound: qualityResults.value.reduce((sum, r) => sum + r.issues.length, 0),
		}

		toast.success('Quality Analysis Complete', `Analyzed ${qualityStats.value.videosAnalyzed} videos. Found ${qualityStats.value.issuesFound} quality issues.`)
	} catch (error) {
		console.error('Quality analysis failed:', error)
		toast.error('Analysis Failed', 'Could not analyze video quality')
	} finally {
		isAnalyzingQuality.value = false
	}
}
</script>

<style scoped>
/* Component will use styles from parent ai_page.css */
</style>
