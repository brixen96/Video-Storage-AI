<template>
	<div class="ai-feature-card">
		<div class="feature-icon">
			<font-awesome-icon :icon="['fas', 'file-signature']" />
		</div>
		<h3>Auto-Naming</h3>
		<p>Generate better file names based on video metadata.</p>

		<div class="feature-stats" v-if="namingStats">
			<div class="stat">
				<span class="stat-label">Videos Analyzed:</span>
				<span class="stat-value">{{ namingStats.videosAnalyzed || 0 }}</span>
			</div>
			<div class="stat">
				<span class="stat-label">Suggestions:</span>
				<span class="stat-value">{{ namingStats.suggestions || 0 }}</span>
			</div>
		</div>

		<div class="feature-controls mt-4">
			<button class="btn btn-primary btn-lg" @click="startAutoNaming" :disabled="isGeneratingNames">
				<font-awesome-icon :icon="['fas', isGeneratingNames ? 'spinner' : 'play']" :spin="isGeneratingNames" class="me-2" />
				{{ isGeneratingNames ? 'Analyzing...' : 'Start Analysis' }}
			</button>
		</div>

		<!-- Empty State -->
		<div v-if="namingStats && namingResults.length === 0" class="empty-state mt-4">
			<font-awesome-icon :icon="['fas', 'check-circle']" class="empty-state-icon text-success" />
			<p class="empty-state-text">All videos have good filenames! No naming suggestions needed.</p>
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
const isGeneratingNames = ref(false)
const namingStats = ref(null)
const namingResults = ref([])

// Main function - Start auto-naming
const startAutoNaming = async () => {
	isGeneratingNames.value = true
	toast.info('Analyzing', 'Generating naming suggestions...')

	try {
		const response = await aiAPI.suggestNaming({
			video_ids: [],
		})

		namingResults.value = response.data.results || []
		namingStats.value = {
			videosAnalyzed: namingResults.value.length,
			suggestions: namingResults.value.length,
		}

		toast.success('Naming Suggestions Complete', `Generated ${namingStats.value.suggestions} naming suggestions for ${namingStats.value.videosAnalyzed} videos.`)
	} catch (error) {
		console.error('Auto-naming failed:', error)
		toast.error('Analysis Failed', 'Could not generate naming suggestions')
	} finally {
		isGeneratingNames.value = false
	}
}
</script>

<style scoped>
/* Component will use styles from parent ai_page.css */
</style>
