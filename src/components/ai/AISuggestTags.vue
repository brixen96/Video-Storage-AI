<template>
	<div class="ai-feature-card">
		<div class="feature-icon">
			<font-awesome-icon :icon="['fas', 'tags']" />
		</div>
		<h3>Smart Tagging</h3>
		<p>AI-powered tag suggestions based on video content analysis.</p>

		<div class="feature-stats" v-if="tagStats">
			<div class="stat">
				<span class="stat-label">Videos Analyzed:</span>
				<span class="stat-value">{{ tagStats.videosAnalyzed || 0 }}</span>
			</div>
			<div class="stat">
				<span class="stat-label">Tags Suggested:</span>
				<span class="stat-value">{{ tagStats.tagsSuggested || 0 }}</span>
			</div>
		</div>

		<div class="feature-controls mt-4">
			<button class="btn btn-primary btn-lg" @click="startSmartTagging" :disabled="isTagging">
				<font-awesome-icon :icon="['fas', isTagging ? 'spinner' : 'play']" :spin="isTagging" class="me-2" />
				{{ isTagging ? 'Analyzing...' : 'Start Analysis' }}
			</button>
			<div class="form-check mt-3">
				<input v-model="autoApplyTags" type="checkbox" class="form-check-input" id="autoApplyTags" />
				<label class="form-check-label" for="autoApplyTags"> Auto-apply high confidence tags (85%+) </label>
			</div>
			<div v-if="tagSuggestions.length > 0" class="confidence-slider mt-3">
				<label class="form-label">
					Confidence Threshold: <strong>{{ (tagConfidenceThreshold * 100).toFixed(0) }}%</strong>
				</label>
				<input v-model.number="tagConfidenceThreshold" type="range" class="form-range" min="0" max="1" step="0.05" />
				<div class="slider-labels">
					<span>0%</span>
					<span>100%</span>
				</div>
			</div>
		</div>

		<!-- Tag Suggestions Display -->
		<div v-if="tagSuggestions.length > 0" class="suggestions-panel mt-4">
			<div class="suggestions-header">
				<h4>
					Tag Suggestions ({{ filteredTagSuggestions.length }} videos)
					<span v-if="selectedTagSuggestions.length > 0" class="badge bg-primary ms-2"> {{ selectedTagSuggestions.length }} selected </span>
				</h4>
				<div class="suggestions-actions">
					<button class="btn btn-sm btn-outline-info me-2" @click="exportToCSV(filteredTagSuggestions, 'tag-suggestions')">
						<font-awesome-icon :icon="['fas', 'file-export']" class="me-1" />
						CSV
					</button>
					<button class="btn btn-sm btn-outline-info me-2" @click="exportToJSON(filteredTagSuggestions, 'tag-suggestions')">
						<font-awesome-icon :icon="['fas', 'file-export']" class="me-1" />
						JSON
					</button>
					<button v-if="selectedTagSuggestions.length > 0" class="btn btn-sm btn-success me-2" @click="applySelectedTagSuggestions">
						<font-awesome-icon :icon="['fas', 'check']" class="me-2" />
						Apply Selected ({{ selectedTagSuggestions.length }})
					</button>
					<button v-else class="btn btn-sm btn-success" @click="applyAllTagSuggestions">
						<font-awesome-icon :icon="['fas', 'check']" class="me-2" />
						Apply All
					</button>
				</div>
			</div>

			<!-- Bulk Selection Controls -->
			<div class="bulk-selection-controls mb-3">
				<div class="form-check">
					<input v-model="selectAllTagSuggestions" type="checkbox" class="form-check-input" id="selectAllTags" @change="toggleSelectAllTagSuggestions" />
					<label class="form-check-label" for="selectAllTags"> Select All Visible Tags </label>
				</div>
				<button
					v-if="selectedTagSuggestions.length > 0"
					class="btn btn-sm btn-outline-danger"
					@click=";(selectedTagSuggestions = []), (selectAllTagSuggestions = false)"
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
					<input v-model="tagSuggestionSearch" type="text" class="form-control" placeholder="Search by video title or tag name..." />
					<button v-if="tagSuggestionSearch" class="btn btn-outline-secondary" @click="tagSuggestionSearch = ''">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
			</div>

			<div class="suggestions-list">
				<div v-for="suggestion in filteredTagSuggestions" :key="suggestion.video_id" class="suggestion-item">
					<div class="video-info">
						<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
						<router-link :to="{ path: '/videos', query: { search: suggestion.video_title } }" class="video-link">
							<strong>{{ suggestion.video_title }}</strong>
						</router-link>
					</div>
					<div class="matches">
						<div v-for="tag in suggestion.suggestions" :key="tag.tag_id" class="match-item">
							<div class="match-checkbox">
								<input
									:checked="isTagSuggestionSelected(suggestion.video_id, tag.tag_id)"
									type="checkbox"
									class="form-check-input"
									@change="toggleTagSuggestionSelection(suggestion.video_id, tag)"
								/>
							</div>
							<div class="match-info">
								<font-awesome-icon :icon="['fas', 'tag']" class="me-2" />
								{{ tag.tag_name }}
							</div>
							<div class="match-confidence">
								<span class="confidence-badge" :class="getConfidenceClass(tag.confidence)"> {{ (tag.confidence * 100).toFixed(0) }}% </span>
								<span class="match-type">{{ tag.reason }}</span>
							</div>
							<button class="btn btn-sm btn-outline-success" @click="applyTagToVideo(suggestion.video_id, tag.tag_id, suggestion)">
								<font-awesome-icon :icon="['fas', 'check']" />
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Empty State -->
		<div v-else-if="tagStats && tagSuggestions.length === 0" class="empty-state mt-4">
			<font-awesome-icon :icon="['fas', 'check-circle']" class="empty-state-icon text-success" />
			<p class="empty-state-text">No new tag suggestions. Your videos are well tagged!</p>
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
const isTagging = ref(false)
const autoApplyTags = ref(true)
const tagSuggestions = ref([])
const tagStats = ref(null)
const tagConfidenceThreshold = ref(0.7)
const tagSuggestionSearch = ref('')
const selectedTagSuggestions = ref([])
const selectAllTagSuggestions = ref(false)

// Computed - Filtered tag suggestions based on confidence threshold and search
const filteredTagSuggestions = computed(() => {
	return tagSuggestions.value
		.map((suggestion) => ({
			...suggestion,
			suggestions: suggestion.suggestions.filter((tag) => tag.confidence >= tagConfidenceThreshold.value),
		}))
		.filter((suggestion) => suggestion.suggestions.length > 0)
		.filter((suggestion) => {
			if (!tagSuggestionSearch.value) return true
			const searchLower = tagSuggestionSearch.value.toLowerCase()
			const videoMatch = suggestion.video_title.toLowerCase().includes(searchLower)
			const tagMatch = suggestion.suggestions.some((t) => t.tag_name.toLowerCase().includes(searchLower))
			return videoMatch || tagMatch
		})
})

// Main function - Start smart tagging analysis
const startSmartTagging = async () => {
	isTagging.value = true
	toast.info('Analyzing', 'Scanning videos for tag suggestions...')

	try {
		const response = await aiAPI.suggestTags({
			video_ids: [], // Empty = analyze all videos
			auto_apply: autoApplyTags.value,
			min_confidence: 0.85,
		})

		tagSuggestions.value = response.data.suggestions || []
		tagStats.value = {
			videosAnalyzed: response.data.total || 0,
			tagsSuggested: tagSuggestions.value.reduce((sum, s) => sum + s.suggestions.length, 0),
		}

		if (autoApplyTags.value) {
			const autoApplied = tagSuggestions.value.reduce((sum, s) => sum + s.suggestions.filter((t) => t.confidence >= 0.85).length, 0)

			toast.success(
				'Analysis Complete',
				`Found ${tagStats.value.tagsSuggested} tag suggestions across ${tagStats.value.videosAnalyzed} videos. Auto-applied ${autoApplied} high-confidence tags.`
			)
		} else {
			toast.success('Analysis Complete', `Found ${tagStats.value.tagsSuggested} tag suggestions across ${tagStats.value.videosAnalyzed} videos.`)
		}
	} catch (error) {
		console.error('Smart tagging failed:', error)
		toast.error('Analysis Failed', 'Could not analyze videos for tag suggestions')
	} finally {
		isTagging.value = false
	}
}

// Apply a single tag to a video
const applyTagToVideo = async (videoId, tagId, suggestion) => {
	try {
		await aiAPI.applyTagSuggestions({
			video_id: videoId,
			tag_ids: [tagId],
		})

		const tagName = suggestion.suggestions.find((t) => t.tag_id === tagId)?.tag_name
		toast.success('Tag Applied', `Tag "${tagName}" added to video`)

		// Remove this tag from suggestions
		tagSuggestions.value = tagSuggestions.value
			.map((s) => ({
				...s,
				suggestions: s.suggestions.filter((t) => !(t.tag_id === tagId && s.video_id === videoId)),
			}))
			.filter((s) => s.suggestions.length > 0)

		// Update stats
		if (tagStats.value) {
			tagStats.value.tagsSuggested = Math.max(0, tagStats.value.tagsSuggested - 1)
		}
	} catch (error) {
		console.error('Apply tag failed:', error)
		toast.error('Failed', 'Could not apply tag to video')
	}
}

// Apply all tag suggestions
const applyAllTagSuggestions = async () => {
	const totalTags = tagSuggestions.value.reduce((sum, s) => sum + s.suggestions.length, 0)
	if (!confirm(`Apply all ${totalTags} suggested tags?`)) {
		return
	}

	try {
		// Group tags by video and apply
		for (const suggestion of tagSuggestions.value) {
			const tagIds = suggestion.suggestions.map((t) => t.tag_id)
			await aiAPI.applyTagSuggestions({
				video_id: suggestion.video_id,
				tag_ids: tagIds,
			})
		}

		toast.success('All Tags Applied', `Successfully applied ${totalTags} tags`)
		tagSuggestions.value = []
		if (tagStats.value) {
			tagStats.value.tagsSuggested = 0
		}
	} catch (error) {
		console.error('Apply all tags failed:', error)
		toast.error('Failed', 'Could not apply all tag suggestions')
	}
}

// Apply selected tag suggestions
const applySelectedTagSuggestions = async () => {
	if (selectedTagSuggestions.value.length === 0) {
		toast.warning('No Selection', 'Please select at least one tag to apply')
		return
	}

	if (!confirm(`Apply ${selectedTagSuggestions.value.length} selected tags?`)) {
		return
	}

	try {
		// Group tags by video
		const tagsByVideo = selectedTagSuggestions.value.reduce((acc, tag) => {
			if (!acc[tag.video_id]) {
				acc[tag.video_id] = []
			}
			acc[tag.video_id].push(tag.tag_id)
			return acc
		}, {})

		// Apply tags for each video
		for (const [videoId, tagIds] of Object.entries(tagsByVideo)) {
			await aiAPI.applyTagSuggestions({
				video_id: parseInt(videoId),
				tag_ids: tagIds,
			})
		}

		toast.success('Tags Applied', `Successfully applied ${selectedTagSuggestions.value.length} tags`)

		// Remove applied tags from suggestions
		const appliedKeys = new Set(selectedTagSuggestions.value.map((s) => `${s.video_id}-${s.tag_id}`))
		tagSuggestions.value = tagSuggestions.value
			.map((s) => ({
				...s,
				suggestions: s.suggestions.filter((t) => !appliedKeys.has(`${s.video_id}-${t.tag_id}`)),
			}))
			.filter((s) => s.suggestions.length > 0)

		selectedTagSuggestions.value = []
		selectAllTagSuggestions.value = false

		// Update stats
		if (tagStats.value) {
			tagStats.value.tagsSuggested = Math.max(0, tagStats.value.tagsSuggested - selectedTagSuggestions.value.length)
		}
	} catch (error) {
		console.error('Apply selected tags failed:', error)
		toast.error('Failed', 'Could not apply selected tags')
	}
}

// Helper - Get confidence badge class
const getConfidenceClass = (confidence) => {
	if (confidence >= 0.9) return 'confidence-high'
	if (confidence >= 0.75) return 'confidence-medium'
	return 'confidence-low'
}

// Bulk selection helpers
const toggleSelectAllTagSuggestions = () => {
	if (selectAllTagSuggestions.value) {
		// Select all filtered tag suggestions
		selectedTagSuggestions.value = filteredTagSuggestions.value.flatMap((s) =>
			s.suggestions.map((t) => ({
				video_id: s.video_id,
				tag_id: t.tag_id,
				...t,
			}))
		)
	} else {
		selectedTagSuggestions.value = []
	}
}

const toggleTagSuggestionSelection = (videoId, tag) => {
	const key = `${videoId}-${tag.tag_id}`
	const index = selectedTagSuggestions.value.findIndex((s) => `${s.video_id}-${s.tag_id}` === key)

	if (index > -1) {
		selectedTagSuggestions.value.splice(index, 1)
	} else {
		selectedTagSuggestions.value.push({
			video_id: videoId,
			tag_id: tag.tag_id,
			...tag,
		})
	}

	// Update select all checkbox
	const totalTags = filteredTagSuggestions.value.flatMap((s) => s.suggestions).length
	selectAllTagSuggestions.value = selectedTagSuggestions.value.length === totalTags && totalTags > 0
}

const isTagSuggestionSelected = (videoId, tagId) => {
	const key = `${videoId}-${tagId}`
	return selectedTagSuggestions.value.some((s) => `${s.video_id}-${s.tag_id}` === key)
}
</script>

<style scoped>
/* Component will use styles from parent ai_page.css */
</style>
