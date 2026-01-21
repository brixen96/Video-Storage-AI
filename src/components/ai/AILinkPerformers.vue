<template>
	<div class="ai-feature-card">
		<div class="feature-icon">
			<font-awesome-icon :icon="['fas', 'user-plus']" />
		</div>
		<h3>Auto-Link Performers</h3>
		<p>Automatically detect and link performers to videos based on filename analysis.</p>

		<div class="feature-stats" v-if="linkStats">
			<div class="stat">
				<span class="stat-label">Videos Analyzed:</span>
				<span class="stat-value">{{ linkStats.videosAnalyzed || 0 }}</span>
			</div>
			<div class="stat">
				<span class="stat-label">Matches Found:</span>
				<span class="stat-value">{{ linkStats.matchesFound || 0 }}</span>
			</div>
		</div>

		<div class="feature-controls mt-4">
			<button class="btn btn-primary btn-lg" @click="startAutoLink" :disabled="isAnalyzing">
				<font-awesome-icon :icon="['fas', isAnalyzing ? 'spinner' : 'play']" :spin="isAnalyzing" class="me-2" />
				{{ isAnalyzing ? 'Analyzing...' : 'Start Analysis' }}
			</button>
			<div class="form-check mt-3">
				<input v-model="autoApplyLinks" type="checkbox" class="form-check-input" id="autoApply" />
				<label class="form-check-label" for="autoApply"> Auto-apply 100% matches </label>
			</div>
			<div v-if="suggestions.length > 0" class="confidence-slider mt-3">
				<label class="form-label">
					Confidence Threshold: <strong>{{ (confidenceThreshold * 100).toFixed(0) }}%</strong>
				</label>
				<input v-model.number="confidenceThreshold" type="range" class="form-range" min="0" max="1" step="0.05" />
				<div class="slider-labels">
					<span>0%</span>
					<span>100%</span>
				</div>
			</div>
		</div>

		<!-- Suggestions Display -->
		<div v-if="suggestions.length > 0" class="suggestions-panel mt-4">
			<div class="suggestions-header">
				<h4>
					Suggested Links ({{ filteredSuggestions.length }} videos)
					<span v-if="selectedPerformerLinks.length > 0" class="badge bg-primary ms-2"> {{ selectedPerformerLinks.length }} selected </span>
				</h4>
				<div class="suggestions-actions">
					<button class="btn btn-sm btn-outline-info me-2" @click="exportToCSV(filteredSuggestions, 'performer-links')">
						<font-awesome-icon :icon="['fas', 'file-export']" class="me-1" />
						CSV
					</button>
					<button class="btn btn-sm btn-outline-info me-2" @click="exportToJSON(filteredSuggestions, 'performer-links')">
						<font-awesome-icon :icon="['fas', 'file-export']" class="me-1" />
						JSON
					</button>
					<button v-if="selectedPerformerLinks.length > 0" class="btn btn-sm btn-success me-2" @click="applySelectedPerformerLinks">
						<font-awesome-icon :icon="['fas', 'check']" class="me-2" />
						Apply Selected ({{ selectedPerformerLinks.length }})
					</button>
					<button v-else class="btn btn-sm btn-success" @click="applyAllSuggestions">
						<font-awesome-icon :icon="['fas', 'check']" class="me-2" />
						Apply All
					</button>
				</div>
			</div>

			<!-- Bulk Selection Controls -->
			<div class="bulk-selection-controls mb-3">
				<div class="form-check">
					<input
						v-model="selectAllPerformerLinks"
						type="checkbox"
						class="form-check-input"
						id="selectAllPerformers"
						@change="toggleSelectAllPerformerLinks"
					/>
					<label class="form-check-label" for="selectAllPerformers"> Select All Visible Matches </label>
				</div>
				<button
					v-if="selectedPerformerLinks.length > 0"
					class="btn btn-sm btn-outline-danger"
					@click=";(selectedPerformerLinks = []), (selectAllPerformerLinks = false)"
				>
					<font-awesome-icon :icon="['fas', 'times']" class="me-1" />
					Clear Selection
				</button>
			</div>

			<!-- Search Filter -->
			<div class="search-filter mb-3">
				<div class="input-group">
					<span class="input-group-text">
						<font-awesome-icon :icon="['fas', 'search']" />
					</span>
					<input v-model="performerLinkSearch" type="text" class="form-control" placeholder="Search by video title or performer name..." />
					<button v-if="performerLinkSearch" class="btn btn-outline-secondary" @click="performerLinkSearch = ''">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
			</div>

			<div class="suggestions-list">
				<div v-for="suggestion in filteredSuggestions" :key="suggestion.video_id" class="suggestion-item">
					<div class="video-info">
						<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
						<router-link :to="{ path: '/videos', query: { search: suggestion.video_title } }" class="video-link">
							<strong>{{ suggestion.video_title }}</strong>
						</router-link>
					</div>
					<div class="matches">
						<div v-for="match in suggestion.matches" :key="match.performer_id" class="match-item">
							<div class="match-checkbox">
								<input
									:checked="isPerformerLinkSelected(suggestion.video_id, match.performer_id)"
									type="checkbox"
									class="form-check-input"
									@change="togglePerformerLinkSelection(suggestion.video_id, match)"
								/>
							</div>
							<div class="match-info">
								<font-awesome-icon :icon="['fas', 'user']" class="me-2" />
								{{ match.performer_name }}
							</div>
							<div class="match-confidence">
								<span class="confidence-badge" :class="getConfidenceClass(match.confidence)"> {{ (match.confidence * 100).toFixed(0) }}% </span>
								<span class="match-type">{{ match.match_type }}</span>
							</div>
							<button class="btn btn-sm btn-outline-success" @click="applyMatch(match)">
								<font-awesome-icon :icon="['fas', 'check']" />
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Empty State -->
		<div v-else-if="linkStats && suggestions.length === 0" class="empty-state mt-4">
			<font-awesome-icon :icon="['fas', 'check-circle']" class="empty-state-icon text-success" />
			<p class="empty-state-text">No new performer links to suggest. Your videos are properly linked!</p>
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
const isAnalyzing = ref(false)
const autoApplyLinks = ref(true)
const suggestions = ref([])
const linkStats = ref(null)
const confidenceThreshold = ref(0.75)
const performerLinkSearch = ref('')
const selectedPerformerLinks = ref([])
const selectAllPerformerLinks = ref(false)

// Computed - Filtered suggestions based on confidence threshold and search
const filteredSuggestions = computed(() => {
	return suggestions.value
		.map((suggestion) => ({
			...suggestion,
			matches: suggestion.matches.filter((match) => match.confidence >= confidenceThreshold.value),
		}))
		.filter((suggestion) => suggestion.matches.length > 0)
		.filter((suggestion) => {
			if (!performerLinkSearch.value) return true
			const searchLower = performerLinkSearch.value.toLowerCase()
			const videoMatch = suggestion.video_title.toLowerCase().includes(searchLower)
			const performerMatch = suggestion.matches.some((m) => m.performer_name.toLowerCase().includes(searchLower))
			return videoMatch || performerMatch
		})
})

// Main function - Start auto-link analysis
const startAutoLink = async () => {
	isAnalyzing.value = true
	toast.info('Analyzing', 'Scanning videos for performer matches...')

	try {
		const response = await aiAPI.linkPerformers({
			video_ids: [], // Empty = analyze all videos
			auto_apply: autoApplyLinks.value,
		})

		suggestions.value = response.data.suggestions || []
		linkStats.value = {
			videosAnalyzed: response.data.total || 0,
			matchesFound: suggestions.value.reduce((sum, s) => sum + s.matches.length, 0),
		}

		if (autoApplyLinks.value) {
			const autoApplied = suggestions.value.filter((s) => s.matches.some((m) => m.confidence === 1.0)).length

			toast.success(
				'Analysis Complete',
				`Found ${linkStats.value.matchesFound} matches across ${linkStats.value.videosAnalyzed} videos. Auto-applied ${autoApplied} 100% matches.`
			)
		} else {
			toast.success('Analysis Complete', `Found ${linkStats.value.matchesFound} potential matches across ${linkStats.value.videosAnalyzed} videos.`)
		}
	} catch (error) {
		console.error('Auto-link failed:', error)
		toast.error('Analysis Failed', 'Could not analyze videos for performer matches')
	} finally {
		isAnalyzing.value = false
	}
}

// Apply a single match
const applyMatch = async (match) => {
	try {
		await aiAPI.applyLinks({
			matches: [match],
		})

		toast.success('Link Applied', `${match.performer_name} linked to video`)

		// Remove this match from suggestions
		suggestions.value = suggestions.value
			.map((s) => ({
				...s,
				matches: s.matches.filter((m) => m.performer_id !== match.performer_id || m.video_id !== match.video_id),
			}))
			.filter((s) => s.matches.length > 0)
	} catch (error) {
		console.error('Apply match failed:', error)
		toast.error('Failed', 'Could not apply performer link')
	}
}

// Apply all suggestions
const applyAllSuggestions = async () => {
	if (!confirm(`Apply all ${suggestions.value.reduce((sum, s) => sum + s.matches.length, 0)} suggested links?`)) {
		return
	}

	try {
		// Flatten all matches
		const allMatches = suggestions.value.flatMap((s) => s.matches)

		await aiAPI.applyLinks({
			matches: allMatches,
		})

		toast.success('All Links Applied', `Successfully linked ${allMatches.length} performers`)
		suggestions.value = []
		linkStats.value.matchesFound = 0
	} catch (error) {
		console.error('Apply all failed:', error)
		toast.error('Failed', 'Could not apply all performer links')
	}
}

// Apply selected performer links
const applySelectedPerformerLinks = async () => {
	if (selectedPerformerLinks.value.length === 0) {
		toast.warning('No Selection', 'Please select at least one performer link to apply')
		return
	}

	if (!confirm(`Apply ${selectedPerformerLinks.value.length} selected performer links?`)) {
		return
	}

	try {
		await aiAPI.applyLinks({
			matches: selectedPerformerLinks.value,
		})

		toast.success('Links Applied', `Successfully linked ${selectedPerformerLinks.value.length} performers`)

		// Remove applied matches from suggestions
		const appliedKeys = new Set(selectedPerformerLinks.value.map((s) => `${s.video_id}-${s.performer_id}`))
		suggestions.value = suggestions.value
			.map((s) => ({
				...s,
				matches: s.matches.filter((m) => !appliedKeys.has(`${s.video_id}-${m.performer_id}`)),
			}))
			.filter((s) => s.matches.length > 0)

		selectedPerformerLinks.value = []
		selectAllPerformerLinks.value = false
	} catch (error) {
		console.error('Apply selected links failed:', error)
		toast.error('Failed', 'Could not apply selected performer links')
	}
}

// Helper - Get confidence badge class
const getConfidenceClass = (confidence) => {
	if (confidence >= 0.9) return 'confidence-high'
	if (confidence >= 0.75) return 'confidence-medium'
	return 'confidence-low'
}

// Bulk selection helpers
const toggleSelectAllPerformerLinks = () => {
	if (selectAllPerformerLinks.value) {
		// Select all filtered matches
		selectedPerformerLinks.value = filteredSuggestions.value.flatMap((s) =>
			s.matches.map((m) => ({
				video_id: s.video_id,
				performer_id: m.performer_id,
				...m,
			}))
		)
	} else {
		selectedPerformerLinks.value = []
	}
}

const togglePerformerLinkSelection = (videoId, match) => {
	const key = `${videoId}-${match.performer_id}`
	const index = selectedPerformerLinks.value.findIndex((s) => `${s.video_id}-${s.performer_id}` === key)

	if (index > -1) {
		selectedPerformerLinks.value.splice(index, 1)
	} else {
		selectedPerformerLinks.value.push({
			video_id: videoId,
			performer_id: match.performer_id,
			...match,
		})
	}

	// Update select all checkbox
	const totalMatches = filteredSuggestions.value.flatMap((s) => s.matches).length
	selectAllPerformerLinks.value = selectedPerformerLinks.value.length === totalMatches && totalMatches > 0
}

const isPerformerLinkSelected = (videoId, performerId) => {
	const key = `${videoId}-${performerId}`
	return selectedPerformerLinks.value.some((s) => `${s.video_id}-${s.performer_id}` === key)
}
</script>

<style scoped>
/* Component will use styles from parent ai_page.css */
</style>
