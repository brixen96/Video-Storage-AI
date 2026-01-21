<template>
	<div class="ai-feature-card">
		<div class="feature-icon">
			<font-awesome-icon :icon="['fas', 'layer-group']" />
		</div>
		<h3>Scene Detection</h3>
		<p>Detect and timestamp different scenes in your videos.</p>

		<div class="feature-stats" v-if="sceneStats">
			<div class="stat">
				<span class="stat-label">Videos Analyzed:</span>
				<span class="stat-value">{{ sceneStats.videosAnalyzed || 0 }}</span>
			</div>
			<div class="stat">
				<span class="stat-label">Scenes Found:</span>
				<span class="stat-value">{{ sceneStats.scenesFound || 0 }}</span>
			</div>
		</div>

		<div class="feature-controls mt-4">
			<button class="btn btn-primary btn-lg" @click="startSceneDetection" :disabled="isDetectingScenes">
				<font-awesome-icon :icon="['fas', isDetectingScenes ? 'spinner' : 'play']" :spin="isDetectingScenes" class="me-2" />
				{{ isDetectingScenes ? 'Analyzing...' : 'Start Analysis' }}
			</button>
		</div>

		<!-- Scene Results Display -->
		<div v-if="sceneResults.length > 0" class="suggestions-panel mt-4">
			<div class="suggestions-header">
				<h4>Scene Detection Results ({{ filteredSceneResults.length }} videos)</h4>
				<div class="suggestions-actions">
					<button class="btn btn-sm btn-outline-info me-2" @click="exportToCSV(filteredSceneResults, 'scene-detection')">
						<font-awesome-icon :icon="['fas', 'file-export']" class="me-1" />
						CSV
					</button>
					<button class="btn btn-sm btn-outline-info" @click="exportToJSON(filteredSceneResults, 'scene-detection')">
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
					<input v-model="sceneSearch" type="text" class="form-control" placeholder="Search by video title or scene type..." />
					<button v-if="sceneSearch" class="btn btn-outline-secondary" @click="sceneSearch = ''">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
			</div>

			<div class="suggestions-list">
				<div v-for="result in filteredSceneResults.slice(0, 10)" :key="result.video_id" class="suggestion-item">
					<div class="video-info">
						<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
						<router-link :to="{ path: '/videos', query: { search: result.video_title } }" class="video-link">
							<strong>{{ result.video_title }}</strong>
						</router-link>
					</div>
					<div class="scene-info mt-2">
						<span class="badge bg-info me-2">{{ result.total_scenes }} scenes</span>
						<span v-for="(scene, idx) in result.scenes.slice(0, 3)" :key="idx" class="badge bg-secondary me-1">
							{{ scene.scene_type }} ({{ Math.floor(scene.start_time / 60) }}:{{ (scene.start_time % 60).toString().padStart(2, '0') }})
						</span>
					</div>
				</div>
			</div>
			<p v-if="filteredSceneResults.length > 10" class="text-light text-center mt-2">Showing 10 of {{ filteredSceneResults.length }} results</p>
		</div>

		<!-- Empty State -->
		<div v-else-if="sceneStats && sceneResults.length === 0" class="empty-state mt-4">
			<font-awesome-icon :icon="['fas', 'layer-group']" class="empty-state-icon" />
			<p class="empty-state-text">No scenes detected. Try running analysis on videos with distinct segments.</p>
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
const isDetectingScenes = ref(false)
const sceneStats = ref(null)
const sceneResults = ref([])
const sceneSearch = ref('')

// Computed - Filtered scene results based on search
const filteredSceneResults = computed(() => {
	if (!sceneSearch.value) return sceneResults.value
	const searchLower = sceneSearch.value.toLowerCase()
	return sceneResults.value.filter((result) => {
		const videoMatch = result.video_title.toLowerCase().includes(searchLower)
		const sceneTypeMatch = result.scenes.some((s) => s.scene_type.toLowerCase().includes(searchLower))
		return videoMatch || sceneTypeMatch
	})
})

// Main function - Start scene detection
const startSceneDetection = async () => {
	isDetectingScenes.value = true
	toast.info('Analyzing', 'Detecting scenes in videos...')

	try {
		const response = await aiAPI.detectScenes({
			video_ids: [],
		})

		sceneResults.value = response.data.results || []
		sceneStats.value = {
			videosAnalyzed: sceneResults.value.length,
			scenesFound: sceneResults.value.reduce((sum, r) => sum + r.total_scenes, 0),
		}

		toast.success('Scene Detection Complete', `Detected ${sceneStats.value.scenesFound} scenes across ${sceneStats.value.videosAnalyzed} videos.`)
	} catch (error) {
		console.error('Scene detection failed:', error)
		toast.error('Analysis Failed', 'Could not detect scenes in videos')
	} finally {
		isDetectingScenes.value = false
	}
}
</script>

<style scoped>
/* Component will use styles from parent ai_page.css */
</style>
