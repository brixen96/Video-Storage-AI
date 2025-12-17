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
				</div>

				<!-- Scene Detection Card -->
				<div class="col-md-6">
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
				</div>

				<!-- Content Classification Card -->
				<div class="col-md-6">
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
				</div>

				<!-- Quality Analysis Card -->
				<div class="col-md-6">
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
				</div>

				<!-- Missing Metadata Card -->
				<div class="col-md-6">
					<div class="ai-feature-card">
						<div class="feature-icon">
							<font-awesome-icon :icon="['fas', 'exclamation']" />
						</div>
						<h3>Missing Metadata</h3>
						<p>Find videos with incomplete or missing metadata.</p>

						<div class="feature-stats" v-if="metadataStats">
							<div class="stat">
								<span class="stat-label">Videos Checked:</span>
								<span class="stat-value">{{ metadataStats.videosChecked || 0 }}</span>
							</div>
							<div class="stat">
								<span class="stat-label">Issues Found:</span>
								<span class="stat-value">{{ metadataStats.issuesFound || 0 }}</span>
							</div>
						</div>

						<div class="feature-controls mt-4">
							<button class="btn btn-primary btn-lg" @click="startMissingMetadataDetection" :disabled="isDetectingMetadata">
								<font-awesome-icon :icon="['fas', isDetectingMetadata ? 'spinner' : 'play']" :spin="isDetectingMetadata" class="me-2" />
								{{ isDetectingMetadata ? 'Analyzing...' : 'Start Analysis' }}
							</button>
						</div>

						<!-- Missing Metadata Results Display -->
						<div v-if="metadataResults.length > 0" class="suggestions-panel mt-4">
							<div class="suggestions-header">
								<h4>Missing Metadata ({{ filteredMetadataResults.length }} videos)</h4>
								<div class="suggestions-actions">
									<button class="btn btn-sm btn-outline-info me-2" @click="exportToCSV(filteredMetadataResults, 'missing-metadata')">
										<font-awesome-icon :icon="['fas', 'file-export']" class="me-1" />
										CSV
									</button>
									<button class="btn btn-sm btn-outline-info" @click="exportToJSON(filteredMetadataResults, 'missing-metadata')">
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
									<input v-model="metadataSearch" type="text" class="form-control" placeholder="Search by video title or missing field..." />
									<button v-if="metadataSearch" class="btn btn-outline-secondary" @click="metadataSearch = ''">
										<font-awesome-icon :icon="['fas', 'times']" />
									</button>
								</div>
							</div>

							<div class="suggestions-list">
								<div v-for="result in filteredMetadataResults.slice(0, 10)" :key="result.video_id" class="suggestion-item">
									<div class="video-info">
										<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
										<router-link :to="{ path: '/videos', query: { search: result.video_title } }" class="video-link">
											<strong>{{ result.video_title }}</strong>
										</router-link>
									</div>
									<div class="metadata-info mt-2">
										<span class="badge me-2" :class="result.severity === 'high' ? 'bg-danger' : result.severity === 'medium' ? 'bg-warning' : 'bg-info'">
											{{ result.severity }} severity
										</span>
										<div class="mt-1">
											<small class="text-light d-block">Missing: {{ result.missing_fields.join(', ') }}</small>
											<small v-for="(suggestion, idx) in result.suggestions.slice(0, 2)" :key="idx" class="text-info d-block">
												<font-awesome-icon :icon="['fas', 'info-circle']" class="me-1" />
												{{ suggestion }}
											</small>
										</div>
									</div>
								</div>
							</div>
							<p v-if="filteredMetadataResults.length > 10" class="text-light text-center mt-2">Showing 10 of {{ filteredMetadataResults.length }} results</p>
						</div>

						<!-- Empty State -->
						<div v-else-if="metadataStats && metadataResults.length === 0" class="empty-state mt-4">
							<font-awesome-icon :icon="['fas', 'check-circle']" class="empty-state-icon text-success" />
							<p class="empty-state-text">All videos have complete metadata! Your library is well organized.</p>
						</div>
					</div>
				</div>

				<!-- Duplicate Detection Card -->
				<div class="col-md-6">
					<div class="ai-feature-card">
						<div class="feature-icon">
							<font-awesome-icon :icon="['fas', 'copy']" />
						</div>
						<h3>Duplicate Detection</h3>
						<p>Find duplicate and similar videos across your library.</p>

						<div class="feature-stats" v-if="duplicateStats">
							<div class="stat">
								<span class="stat-label">Videos Scanned:</span>
								<span class="stat-value">{{ duplicateStats.videosScanned || 0 }}</span>
							</div>
							<div class="stat">
								<span class="stat-label">Duplicate Groups:</span>
								<span class="stat-value">{{ duplicateStats.duplicateGroups || 0 }}</span>
							</div>
						</div>

						<div class="feature-controls mt-4">
							<button class="btn btn-primary btn-lg" @click="startDuplicateDetection" :disabled="isDetectingDuplicates">
								<font-awesome-icon :icon="['fas', isDetectingDuplicates ? 'spinner' : 'play']" :spin="isDetectingDuplicates" class="me-2" />
								{{ isDetectingDuplicates ? 'Analyzing...' : 'Start Analysis' }}
							</button>
						</div>

						<!-- Duplicate Results Display -->
						<div v-if="duplicateResults.length > 0" class="suggestions-panel mt-4">
							<div class="suggestions-header">
								<h4>Duplicate Groups ({{ filteredDuplicateResults.length }} groups)</h4>
								<div class="suggestions-actions">
									<button class="btn btn-sm btn-outline-info me-2" @click="exportToCSV(filteredDuplicateResults, 'duplicate-detection')">
										<font-awesome-icon :icon="['fas', 'file-export']" class="me-1" />
										CSV
									</button>
									<button class="btn btn-sm btn-outline-info" @click="exportToJSON(filteredDuplicateResults, 'duplicate-detection')">
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
									<input v-model="duplicateSearch" type="text" class="form-control" placeholder="Search by video title or reason..." />
									<button v-if="duplicateSearch" class="btn btn-outline-secondary" @click="duplicateSearch = ''">
										<font-awesome-icon :icon="['fas', 'times']" />
									</button>
								</div>
							</div>

							<div class="suggestions-list">
								<div v-for="(group, idx) in filteredDuplicateResults.slice(0, 5)" :key="group.group_id || idx" class="suggestion-item">
									<div class="duplicate-group-header mb-2">
										<div>
											<span class="badge bg-warning text-dark me-2">Group {{ idx + 1 }}</span>
											<span class="badge bg-info">{{ (group.similarity * 100).toFixed(0) }}% similar</span>
											<small class="text-light ms-2">{{ group.reason }}</small>
										</div>
										<button class="btn btn-sm btn-outline-primary" @click="openComparisonView(group)">
											<font-awesome-icon :icon="['fas', 'expand']" class="me-1" />
											Compare
										</button>
									</div>
									<div class="duplicate-videos">
										<div v-for="video in group.videos" :key="video.video_id" class="ps-3 mb-1">
											<font-awesome-icon :icon="['fas', 'video']" class="me-2 text-light" />
											<router-link :to="{ path: '/videos', query: { search: video.video_title } }" class="video-link">
												<small>{{ video.video_title }}</small>
											</router-link>
											<small class="text-light ms-2">({{ (video.file_size / 1024 / 1024).toFixed(0) }} MB)</small>
										</div>
									</div>
								</div>
							</div>
							<p v-if="filteredDuplicateResults.length > 5" class="text-light text-center mt-2">Showing 5 of {{ filteredDuplicateResults.length }} groups</p>
						</div>

						<!-- Empty State -->
						<div v-else-if="duplicateStats && duplicateResults.length === 0" class="empty-state mt-4">
							<font-awesome-icon :icon="['fas', 'check-circle']" class="empty-state-icon text-success" />
							<p class="empty-state-text">No duplicates detected! Your library is clean and organized.</p>
						</div>
					</div>
				</div>

				<!-- Auto-Naming Card -->
				<div class="col-md-6">
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
				</div>

				<!-- Library Analytics Card -->
				<div class="col-md-6">
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
				</div>

				<!-- Thumbnail Quality Card -->
				<div class="col-md-6">
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
				</div>
			</div>
		</div>

		<!-- Duplicate Comparison Modal -->
		<div v-if="showComparisonModal && comparisonGroup" class="comparison-modal-overlay" @click="closeComparisonView">
			<div class="comparison-modal" @click.stop>
				<div class="comparison-modal-header">
					<h3>
						<font-awesome-icon :icon="['fas', 'copy']" class="me-2" />
						Compare Duplicates
					</h3>
					<button class="btn-close-modal" @click="closeComparisonView">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>

				<div class="comparison-modal-body">
					<div class="comparison-info mb-4">
						<div class="badge bg-info me-2">{{ (comparisonGroup.similarity * 100).toFixed(0) }}% similar</div>
						<small class="text-light">{{ comparisonGroup.reason }}</small>
					</div>

					<div class="comparison-grid">
						<div v-for="video in comparisonGroup.videos" :key="video.video_id" class="comparison-video-card">
							<div class="video-card-header">
								<div class="video-title">
									<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
									{{ video.title }}
								</div>
								<div v-if="getBetterQualityVideo(comparisonGroup)?.video_id === video.video_id" class="recommended-badge">
									<font-awesome-icon :icon="['fas', 'star']" class="me-1" />
									Recommended
								</div>
							</div>

							<div class="video-metadata">
								<div class="metadata-row">
									<span class="metadata-label">File Size:</span>
									<span class="metadata-value">{{ (video.file_size / 1024 / 1024).toFixed(2) }} MB</span>
								</div>
								<div v-if="video.resolution" class="metadata-row">
									<span class="metadata-label">Resolution:</span>
									<span class="metadata-value">{{ video.resolution }}</span>
								</div>
								<div v-if="video.duration" class="metadata-row">
									<span class="metadata-label">Duration:</span>
									<span class="metadata-value">{{ Math.floor(video.duration / 60) }}:{{ (video.duration % 60).toString().padStart(2, '0') }}</span>
								</div>
								<div v-if="video.bitrate" class="metadata-row">
									<span class="metadata-label">Bitrate:</span>
									<span class="metadata-value">{{ (video.bitrate / 1000).toFixed(0) }} kbps</span>
								</div>
								<div v-if="video.codec" class="metadata-row">
									<span class="metadata-label">Codec:</span>
									<span class="metadata-value">{{ video.codec }}</span>
								</div>
								<div class="metadata-row">
									<span class="metadata-label">Path:</span>
									<span class="metadata-value text-truncate" :title="video.path">{{ video.path }}</span>
								</div>
							</div>

							<div class="video-actions mt-3">
								<router-link :to="{ path: '/videos', query: { search: video.title } }" class="btn btn-sm btn-outline-info me-2">
									<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
									View
								</router-link>
								<button class="btn btn-sm btn-outline-danger" @click="deleteDuplicateVideo(video.video_id)">
									<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />
									Delete
								</button>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, getCurrentInstance } from 'vue'
import { aiAPI } from '@/services/api'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast

// State
const isAnalyzing = ref(false)
const autoApplyLinks = ref(true)
const suggestions = ref([])
const linkStats = ref(null)
const confidenceThreshold = ref(0.75)

// Smart Tagging State
const isTagging = ref(false)
const autoApplyTags = ref(true)
const tagSuggestions = ref([])
const tagStats = ref(null)
const tagConfidenceThreshold = ref(0.7)

// Scene Detection State
const isDetectingScenes = ref(false)
const sceneStats = ref(null)
const sceneResults = ref([])

// Content Classification State
const isClassifyingContent = ref(false)
const classificationStats = ref(null)
const classificationResults = ref([])

// Quality Analysis State
const isAnalyzingQuality = ref(false)
const qualityStats = ref(null)
const qualityResults = ref([])

// Missing Metadata State
const isDetectingMetadata = ref(false)
const metadataStats = ref(null)
const metadataResults = ref([])

// Duplicate Detection State
const isDetectingDuplicates = ref(false)
const duplicateStats = ref(null)
const duplicateResults = ref([])

// Auto-Naming State
const isGeneratingNames = ref(false)
const namingStats = ref(null)
const namingResults = ref([])

// Library Analytics State
const isLoadingAnalytics = ref(false)
const analyticsData = ref(null)

// Thumbnail Quality State
const isAnalyzingThumbnails = ref(false)
const thumbnailStats = ref(null)
const thumbnailResults = ref([])

// Search filters
const performerLinkSearch = ref('')
const tagSuggestionSearch = ref('')
const sceneSearch = ref('')
const qualitySearch = ref('')
const metadataSearch = ref('')
const duplicateSearch = ref('')

// Bulk selection state
const selectedPerformerLinks = ref([])
const selectedTagSuggestions = ref([])
const selectAllPerformerLinks = ref(false)
const selectAllTagSuggestions = ref(false)

// Duplicate comparison state
const showComparisonModal = ref(false)
const comparisonGroup = ref(null)

// Filtered suggestions based on confidence threshold and search
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

const filteredSceneResults = computed(() => {
	if (!sceneSearch.value) return sceneResults.value
	const searchLower = sceneSearch.value.toLowerCase()
	return sceneResults.value.filter((result) => {
		const videoMatch = result.video_title.toLowerCase().includes(searchLower)
		const sceneTypeMatch = result.scenes.some((s) => s.scene_type.toLowerCase().includes(searchLower))
		return videoMatch || sceneTypeMatch
	})
})

const filteredQualityResults = computed(() => {
	if (!qualitySearch.value) return qualityResults.value
	const searchLower = qualitySearch.value.toLowerCase()
	return qualityResults.value.filter((result) => {
		const videoMatch = result.video_title.toLowerCase().includes(searchLower)
		const issueMatch = result.issues.some((i) => i.toLowerCase().includes(searchLower))
		return videoMatch || issueMatch
	})
})

const filteredMetadataResults = computed(() => {
	if (!metadataSearch.value) return metadataResults.value
	const searchLower = metadataSearch.value.toLowerCase()
	return metadataResults.value.filter((result) => {
		const videoMatch = result.video_title.toLowerCase().includes(searchLower)
		const fieldMatch = result.missing_fields.some((f) => f.toLowerCase().includes(searchLower))
		return videoMatch || fieldMatch
	})
})

const filteredDuplicateResults = computed(() => {
	if (!duplicateSearch.value) return duplicateResults.value
	const searchLower = duplicateSearch.value.toLowerCase()
	return duplicateResults.value.filter((group) => {
		const videoMatch = group.videos.some((v) => v.title.toLowerCase().includes(searchLower))
		const reasonMatch = group.reason.toLowerCase().includes(searchLower)
		return videoMatch || reasonMatch
	})
})

// Dashboard Summary Computed Properties
const totalVideosProcessed = computed(() => {
	return (linkStats.value?.videosAnalyzed || 0) + (tagStats.value?.videosAnalyzed || 0) + (sceneStats.value?.videosAnalyzed || 0) + (qualityStats.value?.videosAnalyzed || 0)
})

const totalMatches = computed(() => {
	return (linkStats.value?.matchesFound || 0) + (tagStats.value?.tagsSuggested || 0) + (sceneStats.value?.scenesFound || 0)
})

const totalIssues = computed(() => {
	return (qualityStats.value?.issuesFound || 0) + (metadataStats.value?.issuesFound || 0) + (duplicateStats.value?.duplicateGroups || 0)
})

const libraryHealthScore = computed(() => {
	// Calculate health score based on metadata completeness and quality
	const videosWithIssues = totalIssues.value
	const videosProcessed = totalVideosProcessed.value || 1 // Avoid division by zero

	// Simple health calculation: 100% - (issues / videos * 100)
	const healthPercentage = Math.max(0, Math.min(100, 100 - (videosWithIssues / videosProcessed) * 100))
	return Math.round(healthPercentage)
})

// Helper functions for dashboard
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

// Export functions
const exportToCSV = (data, filename) => {
	if (!data || data.length === 0) {
		toast.warning('No Data', 'No data available to export')
		return
	}

	// Get headers from first object
	const headers = Object.keys(data[0])
	const csvContent =
		headers.join(',') +
		'\n' +
		data
			.map((row) =>
				headers
					.map((header) => {
						const value = row[header]
						// Handle nested objects and arrays
						if (typeof value === 'object' && value !== null) {
							return '"' + JSON.stringify(value).replace(/"/g, '""') + '"'
						}
						// Escape quotes and wrap in quotes if contains comma
						const stringValue = String(value || '')
						if (stringValue.includes(',') || stringValue.includes('"') || stringValue.includes('\n')) {
							return '"' + stringValue.replace(/"/g, '""') + '"'
						}
						return stringValue
					})
					.join(',')
			)
			.join('\n')

	downloadFile(csvContent, filename + '.csv', 'text/csv')
	toast.success('Export Complete', `Exported ${data.length} items to ${filename}.csv`)
}

const exportToJSON = (data, filename) => {
	if (!data || data.length === 0) {
		toast.warning('No Data', 'No data available to export')
		return
	}

	const jsonContent = JSON.stringify(data, null, 2)
	downloadFile(jsonContent, filename + '.json', 'application/json')
	toast.success('Export Complete', `Exported ${data.length} items to ${filename}.json`)
}

const downloadFile = (content, filename, contentType) => {
	const blob = new Blob([content], { type: contentType })
	const url = URL.createObjectURL(blob)
	const link = document.createElement('a')
	link.href = url
	link.download = filename
	link.click()
	URL.revokeObjectURL(url)
}

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

// Scene Detection
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

// Content Classification
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

// Quality Analysis
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

// Missing Metadata Detection
const startMissingMetadataDetection = async () => {
	isDetectingMetadata.value = true
	toast.info('Analyzing', 'Checking for missing metadata...')

	try {
		const response = await aiAPI.detectMissingMetadata({
			video_ids: [],
		})

		metadataResults.value = response.data.results || []
		metadataStats.value = {
			videosChecked: metadataResults.value.length,
			issuesFound: metadataResults.value.length,
		}

		toast.success('Metadata Check Complete', `Checked ${metadataStats.value.videosChecked} videos. Found ${metadataStats.value.issuesFound} with missing metadata.`)
	} catch (error) {
		console.error('Missing metadata detection failed:', error)
		toast.error('Analysis Failed', 'Could not detect missing metadata')
	} finally {
		isDetectingMetadata.value = false
	}
}

// Duplicate Detection
const startDuplicateDetection = async () => {
	isDetectingDuplicates.value = true
	toast.info('Analyzing', 'Searching for duplicate videos...')

	try {
		const response = await aiAPI.detectDuplicates({
			video_ids: [],
		})

		duplicateResults.value = response.data.results || []
		duplicateStats.value = {
			videosScanned: duplicateResults.value.reduce((sum, g) => sum + g.videos.length, 0),
			duplicateGroups: duplicateResults.value.length,
		}

		toast.success('Duplicate Detection Complete', `Found ${duplicateStats.value.duplicateGroups} duplicate groups across ${duplicateStats.value.videosScanned} videos.`)
	} catch (error) {
		console.error('Duplicate detection failed:', error)
		toast.error('Analysis Failed', 'Could not detect duplicates')
	} finally {
		isDetectingDuplicates.value = false
	}
}

// Duplicate comparison functions
const openComparisonView = (group) => {
	comparisonGroup.value = group
	showComparisonModal.value = true
}

const closeComparisonView = () => {
	showComparisonModal.value = false
	comparisonGroup.value = null
}

const deleteDuplicateVideo = async (videoId) => {
	if (!confirm('Are you sure you want to delete this video? This action cannot be undone.')) {
		return
	}

	try {
		// Call API to delete video (endpoint would need to be implemented)
		// await aiAPI.deleteVideo(videoId)

		// For now, just remove from the duplicate group in UI
		if (comparisonGroup.value) {
			comparisonGroup.value.videos = comparisonGroup.value.videos.filter((v) => v.video_id !== videoId)

			// If only one video left, remove the group entirely
			if (comparisonGroup.value.videos.length <= 1) {
				duplicateResults.value = duplicateResults.value.filter((g) => g.group_id !== comparisonGroup.value.group_id)
				closeComparisonView()
			}
		}

		toast.success('Video Deleted', 'Duplicate video has been removed')
	} catch (error) {
		console.error('Delete video failed:', error)
		toast.error('Delete Failed', 'Could not delete video')
	}
}

const getBetterQualityVideo = (group) => {
	if (!group || !group.videos || group.videos.length === 0) return null
	// Find video with highest resolution or largest file size
	return group.videos.reduce((best, current) => {
		const currentSize = current.file_size || 0
		const bestSize = best.file_size || 0
		return currentSize > bestSize ? current : best
	})
}

// Auto-Naming
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

// Library Analytics
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

// Thumbnail Quality
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
@import '@/styles/pages/ai_page.css';
</style>
