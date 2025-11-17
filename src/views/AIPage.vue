<template>
	<div class="ai-page">
		<div class="container-fluid mt-3">
			<!-- AI Features Grid -->
			<div class="row g-4">
				<!-- Auto-Link Performers Card -->
				<div class="col-md-6">
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
								<label class="form-check-label" for="autoApply"> Auto-apply high confidence matches (90%+) </label>
							</div>
						</div>

						<!-- Suggestions Display -->
						<div v-if="suggestions.length > 0" class="suggestions-panel mt-4">
							<div class="suggestions-header">
								<h4>Suggested Links ({{ suggestions.length }} videos)</h4>
								<button class="btn btn-sm btn-success" @click="applyAllSuggestions">
									<font-awesome-icon :icon="['fas', 'check']" class="me-2" />
									Apply All
								</button>
							</div>

							<div class="suggestions-list">
								<div v-for="suggestion in suggestions" :key="suggestion.video_id" class="suggestion-item">
									<div class="video-info">
										<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
										<strong>{{ suggestion.video_title }}</strong>
									</div>
									<div class="matches">
										<div v-for="match in suggestion.matches" :key="match.performer_id" class="match-item">
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
					</div>
				</div>

				<!-- Smart Tagging Card -->
				<div class="col-md-6">
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
						</div>

						<!-- Tag Suggestions Display -->
						<div v-if="tagSuggestions.length > 0" class="suggestions-panel mt-4">
							<div class="suggestions-header">
								<h4>Tag Suggestions ({{ tagSuggestions.length }} videos)</h4>
								<button class="btn btn-sm btn-success" @click="applyAllTagSuggestions">
									<font-awesome-icon :icon="['fas', 'check']" class="me-2" />
									Apply All
								</button>
							</div>

							<div class="suggestions-list">
								<div v-for="suggestion in tagSuggestions" :key="suggestion.video_id" class="suggestion-item">
									<div class="video-info">
										<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
										<strong>{{ suggestion.video_title }}</strong>
									</div>
									<div class="matches">
										<div v-for="tag in suggestion.suggestions" :key="tag.tag_id" class="match-item">
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
					</div>
				</div>

				<!-- Coming Soon Features -->
				<div class="col-md-6">
					<div class="ai-feature-card disabled">
						<div class="feature-icon">
							<font-awesome-icon :icon="['fas', 'search']" />
						</div>
						<h3>Duplicate Detection</h3>
						<p>Find duplicate and similar videos across your library.</p>
						<div class="coming-soon-badge">Coming Soon</div>
					</div>
				</div>

				<div class="col-md-6">
					<div class="ai-feature-card disabled">
						<div class="feature-icon">
							<font-awesome-icon :icon="['fas', 'search']" />
						</div>
						<h3>Duplicate Detection</h3>
						<p>Find duplicate and similar videos across your library.</p>
						<div class="coming-soon-badge">Coming Soon</div>
					</div>
				</div>

				<div class="col-md-6">
					<div class="ai-feature-card disabled">
						<div class="feature-icon">
							<font-awesome-icon :icon="['fas', 'chart-line']" />
						</div>
						<h3>Library Analytics</h3>
						<p>Intelligent insights and recommendations for your collection.</p>
						<div class="coming-soon-badge">Coming Soon</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, getCurrentInstance } from 'vue'
import { aiAPI } from '@/services/api'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast

// State
const isAnalyzing = ref(false)
const autoApplyLinks = ref(true)
const suggestions = ref([])
const linkStats = ref(null)

// Smart Tagging State
const isTagging = ref(false)
const autoApplyTags = ref(true)
const tagSuggestions = ref([])
const tagStats = ref(null)

// Auto-link performers
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
			const autoApplied = suggestions.value.filter((s) => s.matches.some((m) => m.confidence >= 0.9)).length

			toast.success(
				'Analysis Complete',
				`Found ${linkStats.value.matchesFound} matches across ${linkStats.value.videosAnalyzed} videos. Auto-applied ${autoApplied} high-confidence links.`
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

// Get confidence badge class
const getConfidenceClass = (confidence) => {
	if (confidence >= 0.9) return 'confidence-high'
	if (confidence >= 0.75) return 'confidence-medium'
	return 'confidence-low'
}

// Smart Tagging functions
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
</script>

<style scoped>
@import '@/styles/pages/ai_page.css';
</style>
