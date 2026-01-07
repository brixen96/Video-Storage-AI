<template>
	<div class="ai-feature-card">
		<div class="feature-icon">
			<font-awesome-icon :icon="['fas', 'image']" />
		</div>
		<h3>Thumbnail Quality</h3>
		<p>Analyze and suggest better thumbnail frames for videos.</p>

		<div class="feature-stats" v-if="thumbnailStats">
			<div class="stat">
				<span class="stat-label">Videos Analyzed:</span>
				<span class="stat-value">{{ thumbnailStats.videosAnalyzed || 0 }}</span>
			</div>
			<div class="stat">
				<span class="stat-label">Improvements:</span>
				<span class="stat-value">{{ thumbnailStats.improvements || 0 }}</span>
			</div>
		</div>

		<div class="feature-controls mt-4">
			<button class="btn btn-primary btn-lg" @click="startThumbnailAnalysis" :disabled="isAnalyzingThumbnails">
				<font-awesome-icon :icon="['fas', isAnalyzingThumbnails ? 'spinner' : 'play']" :spin="isAnalyzingThumbnails" class="me-2" />
				{{ isAnalyzingThumbnails ? 'Analyzing...' : 'Start Analysis' }}
			</button>
		</div>

		<!-- Empty State -->
		<div v-if="thumbnailStats && thumbnailResults.length === 0" class="empty-state mt-4">
			<font-awesome-icon :icon="['fas', 'check-circle']" class="empty-state-icon text-success" />
			<p class="empty-state-text">All thumbnails are of good quality! No improvements needed.</p>
		</div>
	</div>
</template>

<script setup>
import { ref, getCurrentInstance } from 'vue'
import { aiAPI } from '@/services/api'

// Get toast instance
const { proxy } = getCurrentInstance()
const toast = proxy.$toast

// State
const isAnalyzingThumbnails = ref(false)
const thumbnailStats = ref(null)
const thumbnailResults = ref([])

// Main function - Start thumbnail analysis
const startThumbnailAnalysis = async () => {
	isAnalyzingThumbnails.value = true
	toast.info('Analyzing', 'Analyzing thumbnail quality...')

	try {
		const response = await aiAPI.analyzeThumbnailQuality({
			video_ids: [],
		})

		thumbnailResults.value = response.data.results || []
		const needsImprovement = thumbnailResults.value.filter((r) => r.quality_score < 0.7).length

		thumbnailStats.value = {
			videosAnalyzed: thumbnailResults.value.length,
			improvements: needsImprovement,
		}

		toast.success('Thumbnail Analysis Complete', `Analyzed ${thumbnailStats.value.videosAnalyzed} thumbnails. ${thumbnailStats.value.improvements} could be improved.`)
	} catch (error) {
		console.error('Thumbnail analysis failed:', error)
		toast.error('Analysis Failed', 'Could not analyze thumbnail quality')
	} finally {
		isAnalyzingThumbnails.value = false
	}
}
</script>

<style scoped>
/* Component will use styles from parent ai_page.css */
</style>
