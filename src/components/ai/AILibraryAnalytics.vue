<template>
	<div class="ai-feature-card">
		<div class="feature-icon">
			<font-awesome-icon :icon="['fas', 'chart-line']" />
		</div>
		<h3>Library Analytics</h3>
		<p>Comprehensive statistics and insights about your library.</p>

		<div class="feature-stats" v-if="analyticsData">
			<div class="stat">
				<span class="stat-label">Total Videos:</span>
				<span class="stat-value">{{ analyticsData.total_videos || 0 }}</span>
			</div>
			<div class="stat">
				<span class="stat-label">Total Performers:</span>
				<span class="stat-value">{{ analyticsData.total_performers || 0 }}</span>
			</div>
		</div>

		<div class="feature-controls mt-4">
			<button class="btn btn-primary btn-lg" @click="loadLibraryAnalytics" :disabled="isLoadingAnalytics">
				<font-awesome-icon :icon="['fas', isLoadingAnalytics ? 'spinner' : 'chart-line']" :spin="isLoadingAnalytics" class="me-2" />
				{{ isLoadingAnalytics ? 'Loading...' : 'View Analytics' }}
			</button>
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
const isLoadingAnalytics = ref(false)
const analyticsData = ref(null)

// Main function - Load library analytics
const loadLibraryAnalytics = async () => {
	isLoadingAnalytics.value = true
	toast.info('Loading', 'Loading library analytics...')

	try {
		const response = await aiAPI.getLibraryAnalytics()

		analyticsData.value = response.data || {}

		toast.success('Analytics Loaded', 'Library analytics loaded successfully.')
	} catch (error) {
		console.error('Library analytics failed:', error)
		toast.error('Load Failed', 'Could not load library analytics')
	} finally {
		isLoadingAnalytics.value = false
	}
}
</script>

<style scoped>
/* Component will use styles from parent ai_page.css */
</style>
