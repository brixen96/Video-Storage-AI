<template>
	<div class="ai-page">
		<div class="container-fluid mt-3">
			<!-- Page Header -->
			<div class="page-header text-center mb-5">
				<h1>
					<font-awesome-icon :icon="['fas', 'robot']" class="me-3" />
					AI Assistant
				</h1>
				<p class="lead">Intelligent tools to organize, analyze, and optimize your video library</p>
			</div>

			<!-- Dashboard Summary Card -->
			<div class="dashboard-summary mb-5">
				<div class="row g-4">
					<div class="col-md-3">
						<div class="summary-card">
							<div class="summary-icon">
								<font-awesome-icon :icon="['fas', 'heart']" :class="getHealthIconClass()" />
							</div>
							<div class="summary-content">
								<div class="summary-value" :class="getHealthScoreClass()">{{ libraryHealthScore }}%</div>
								<div class="summary-label">Library Health</div>
							</div>
						</div>
					</div>
					<div class="col-md-3">
						<div class="summary-card">
							<div class="summary-icon bg-primary">
								<font-awesome-icon :icon="['fas', 'video']" />
							</div>
							<div class="summary-content">
								<div class="summary-value">{{ totalVideosProcessed }}</div>
								<div class="summary-label">Videos Processed</div>
							</div>
						</div>
					</div>
					<div class="col-md-3">
						<div class="summary-card">
							<div class="summary-icon bg-success">
								<font-awesome-icon :icon="['fas', 'check-circle']" />
							</div>
							<div class="summary-content">
								<div class="summary-value">{{ totalMatches }}</div>
								<div class="summary-label">AI Matches Found</div>
							</div>
						</div>
					</div>
					<div class="col-md-3">
						<div class="summary-card">
							<div class="summary-icon bg-warning">
								<font-awesome-icon :icon="['fas', 'exclamation-triangle']" />
							</div>
							<div class="summary-content">
								<div class="summary-value">{{ totalIssues }}</div>
								<div class="summary-label">Issues Detected</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- AI Features Grid -->
			<div class="row g-4">
				<!-- Auto-Link Performers -->
				<div class="col-md-6">
					<AILinkPerformers ref="linkPerformersRef" @statsUpdated="updateDashboard" />
				</div>

				<!-- Smart Tagging -->
				<div class="col-md-6">
					<AISuggestTags ref="suggestTagsRef" @statsUpdated="updateDashboard" />
				</div>

				<!-- Scene Detection -->
				<div class="col-md-6">
					<AIDetectScenes ref="detectScenesRef" @statsUpdated="updateDashboard" />
				</div>

				<!-- Content Classification -->
				<div class="col-md-6">
					<AIClassifyContent ref="classifyContentRef" @statsUpdated="updateDashboard" />
				</div>

				<!-- Quality Analysis -->
				<div class="col-md-6">
					<AIAnalyzeQuality ref="analyzeQualityRef" @statsUpdated="updateDashboard" />
				</div>

				<!-- Missing Metadata -->
				<div class="col-md-6">
					<AIDetectMissingMetadata ref="detectMetadataRef" @statsUpdated="updateDashboard" />
				</div>

				<!-- Duplicate Detection -->
				<div class="col-md-6">
					<AIDetectDuplicates ref="detectDuplicatesRef" @statsUpdated="updateDashboard" />
				</div>

				<!-- Auto-Naming -->
				<div class="col-md-6">
					<AISuggestNaming ref="suggestNamingRef" @statsUpdated="updateDashboard" />
				</div>

				<!-- Library Analytics -->
				<div class="col-md-6">
					<AILibraryAnalytics ref="libraryAnalyticsRef" @statsUpdated="updateDashboard" />
				</div>

				<!-- Thumbnail Quality -->
				<div class="col-md-6">
					<AIAnalyzeThumbnails ref="analyzeThumbnailsRef" @statsUpdated="updateDashboard" />
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed } from 'vue'

// Import all AI feature components
import AILinkPerformers from '@/components/ai/AILinkPerformers.vue'
import AISuggestTags from '@/components/ai/AISuggestTags.vue'
import AIDetectScenes from '@/components/ai/AIDetectScenes.vue'
import AIClassifyContent from '@/components/ai/AIClassifyContent.vue'
import AIAnalyzeQuality from '@/components/ai/AIAnalyzeQuality.vue'
import AIDetectMissingMetadata from '@/components/ai/AIDetectMissingMetadata.vue'
import AIDetectDuplicates from '@/components/ai/AIDetectDuplicates.vue'
import AISuggestNaming from '@/components/ai/AISuggestNaming.vue'
import AILibraryAnalytics from '@/components/ai/AILibraryAnalytics.vue'
import AIAnalyzeThumbnails from '@/components/ai/AIAnalyzeThumbnails.vue'

// Component refs for accessing child component data
const linkPerformersRef = ref(null)
const suggestTagsRef = ref(null)
const detectScenesRef = ref(null)
const classifyContentRef = ref(null)
const analyzeQualityRef = ref(null)
const detectMetadataRef = ref(null)
const detectDuplicatesRef = ref(null)
const suggestNamingRef = ref(null)
const libraryAnalyticsRef = ref(null)
const analyzeThumbnailsRef = ref(null)

// Dashboard aggregation state
const dashboardStats = ref({
	videosProcessed: 0,
	matches: 0,
	issues: 0,
})

// Dashboard computed properties
const totalVideosProcessed = computed(() => {
	// Aggregate from all components that track videos processed
	let total = 0
	if (linkPerformersRef.value?.linkStats) total += linkPerformersRef.value.linkStats.videosAnalyzed || 0
	if (suggestTagsRef.value?.tagStats) total += suggestTagsRef.value.tagStats.videosAnalyzed || 0
	if (detectScenesRef.value?.sceneStats) total += detectScenesRef.value.sceneStats.videosAnalyzed || 0
	if (analyzeQualityRef.value?.qualityStats) total += analyzeQualityRef.value.qualityStats.videosAnalyzed || 0
	return total
})

const totalMatches = computed(() => {
	// Aggregate matches/suggestions from components
	let total = 0
	if (linkPerformersRef.value?.linkStats) total += linkPerformersRef.value.linkStats.matchesFound || 0
	if (suggestTagsRef.value?.tagStats) total += suggestTagsRef.value.tagStats.tagsSuggested || 0
	if (detectScenesRef.value?.sceneStats) total += detectScenesRef.value.sceneStats.scenesFound || 0
	return total
})

const totalIssues = computed(() => {
	// Aggregate issues from quality, metadata, and duplicate detection
	let total = 0
	if (analyzeQualityRef.value?.qualityStats) total += analyzeQualityRef.value.qualityStats.issuesFound || 0
	if (detectMetadataRef.value?.metadataStats) total += detectMetadataRef.value.metadataStats.issuesFound || 0
	if (detectDuplicatesRef.value?.duplicateStats) total += detectDuplicatesRef.value.duplicateStats.duplicateGroups || 0
	return total
})

const libraryHealthScore = computed(() => {
	// Calculate health score based on metadata completeness and quality
	const videosWithIssues = totalIssues.value
	const videosProcessed = totalVideosProcessed.value || 1 // Avoid division by zero

	// Simple health calculation: 100% - (issues / videos * 100)
	const healthPercentage = Math.max(0, Math.min(100, 100 - (videosWithIssues / videosProcessed) * 100))
	return Math.round(healthPercentage)
})

// Helper functions for dashboard display
const getHealthScoreClass = () => {
	const score = libraryHealthScore.value
	if (score >= 80) return 'text-success'
	if (score >= 60) return 'text-warning'
	return 'text-danger'
}

const getHealthIconClass = () => {
	const score = libraryHealthScore.value
	if (score >= 80) return 'text-success pulse-animation'
	if (score >= 60) return 'text-warning'
	return 'text-danger'
}

// Event handler for when child components update their stats
const updateDashboard = () => {
	// This function can be called by child components when they update
	// The computed properties will automatically recalculate
	console.log('Dashboard stats updated')
}
</script>

<style scoped>
@import '@/styles/pages/ai_page.css';
</style>
