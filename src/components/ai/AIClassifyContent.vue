<template>
	<div class="ai-feature-card">
		<div class="feature-icon">
			<font-awesome-icon :icon="['fas', 'file-video']" />
		</div>
		<h3>Content Classification</h3>
		<p>Categorize videos by content type and quality.</p>

		<div class="feature-stats" v-if="classificationStats">
			<div class="stat">
				<span class="stat-label">Videos Classified:</span>
				<span class="stat-value">{{ classificationStats.videosClassified || 0 }}</span>
			</div>
			<div class="stat">
				<span class="stat-label">Categories Found:</span>
				<span class="stat-value">{{ classificationStats.categoriesFound || 0 }}</span>
			</div>
		</div>

		<div class="feature-controls mt-4">
			<button class="btn btn-primary btn-lg" @click="startContentClassification" :disabled="isClassifyingContent">
				<font-awesome-icon :icon="['fas', isClassifyingContent ? 'spinner' : 'play']" :spin="isClassifyingContent" class="me-2" />
				{{ isClassifyingContent ? 'Analyzing...' : 'Start Analysis' }}
			</button>
		</div>

		<!-- Empty State -->
		<div v-if="classificationStats && classificationResults.length === 0" class="empty-state mt-4">
			<font-awesome-icon :icon="['fas', 'file-video']" class="empty-state-icon" />
			<p class="empty-state-text">No content classifications found. Run analysis to categorize your videos.</p>
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
const isClassifyingContent = ref(false)
const classificationStats = ref(null)
const classificationResults = ref([])

// Main function - Start content classification
const startContentClassification = async () => {
	isClassifyingContent.value = true
	toast.info('Analyzing', 'Classifying video content...')

	try {
		const response = await aiAPI.classifyContent({
			video_ids: [],
		})

		classificationResults.value = response.data.results || []
		classificationStats.value = {
			videosClassified: classificationResults.value.length,
			categoriesFound: classificationResults.value.reduce((sum, r) => sum + r.categories.length, 0),
		}

		toast.success('Classification Complete', `Classified ${classificationStats.value.videosClassified} videos with ${classificationStats.value.categoriesFound} category tags.`)
	} catch (error) {
		console.error('Content classification failed:', error)
		toast.error('Analysis Failed', 'Could not classify video content')
	} finally {
		isClassifyingContent.value = false
	}
}
</script>

<style scoped>
/* Component will use styles from parent ai_page.css */
</style>
