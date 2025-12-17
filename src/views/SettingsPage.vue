<template>
	<div class="settings-page">
		<div class="container-fluid mt-3">
			<!-- Settings Navigation Tabs -->
			<ul class="nav nav-tabs mb-4" role="tablist">
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'general' }]" @click="activeTab = 'general'" type="button">
						<font-awesome-icon :icon="['fas', 'cog']" class="me-2" />
						General
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'library' }]" @click="activeTab = 'library'" type="button">
						<font-awesome-icon :icon="['fas', 'folder']" class="me-2" />
						Library
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'display' }]" @click="activeTab = 'display'" type="button">
						<font-awesome-icon :icon="['fas', 'eye']" class="me-2" />
						Display
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'privacy' }]" @click="activeTab = 'privacy'" type="button">
						<font-awesome-icon :icon="['fas', 'user']" class="me-2" />
						Privacy
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'about' }]" @click="activeTab = 'about'" type="button">
						<font-awesome-icon :icon="['fas', 'info-circle']" class="me-2" />
						About
					</button>
				</li>
			</ul>

			<!-- Tab Content -->
			<div class="tab-content">
				<!-- General Settings -->
				<div v-if="activeTab === 'general'" class="settings-section">
					<div class="setting-item">
						<div class="setting-info">
							<h4>Default View Mode</h4>
							<p>Choose the default view for browsing videos and performers</p>
						</div>
						<div class="setting-control">
							<select v-model="settings.defaultViewMode" class="form-select">
								<option value="grid">Grid View</option>
								<option value="list">List View</option>
							</select>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Theme</h4>
							<p>Select your preferred color theme</p>
						</div>
						<div class="setting-control">
							<select v-model="settings.theme" class="form-select">
								<option value="dark">Dark Mode</option>
								<option value="light">Light Mode</option>
								<option value="auto">Auto (System Default)</option>
							</select>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Items Per Page</h4>
							<p>Number of items to display per page</p>
						</div>
						<div class="setting-control">
							<input v-model.number="settings.itemsPerPage" type="number" class="form-control" min="10" max="100" step="10" />
						</div>
					</div>
				</div>

				<!-- Library Settings -->
				<div v-if="activeTab === 'library'" class="settings-section">
					<div class="setting-item">
						<div class="setting-info">
							<h4>Video Library Path</h4>
							<p>Default location for video files</p>
						</div>
						<div class="setting-control">
							<input v-model="settings.videoLibraryPath" type="text" class="form-control" placeholder="C:\Videos" />
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Auto-Scan on Startup</h4>
							<p>Automatically scan for new videos when the application starts</p>
						</div>
						<div class="setting-control">
							<div class="form-check form-switch">
								<input v-model="settings.autoScanOnStartup" class="form-check-input" type="checkbox" id="autoScanSwitch" />
								<label class="form-check-label" for="autoScanSwitch">
									{{ settings.autoScanOnStartup ? 'Enabled' : 'Disabled' }}
								</label>
							</div>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Watch for Changes</h4>
							<p>Monitor library folders for new or deleted files</p>
						</div>
						<div class="setting-control">
							<div class="form-check form-switch">
								<input v-model="settings.watchForChanges" class="form-check-input" type="checkbox" id="watchChangesSwitch" />
								<label class="form-check-label" for="watchChangesSwitch">
									{{ settings.watchForChanges ? 'Enabled' : 'Disabled' }}
								</label>
							</div>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Supported Video Formats</h4>
							<p>File extensions to include when scanning (comma-separated)</p>
						</div>
						<div class="setting-control">
							<input v-model="settings.videoFormats" type="text" class="form-control" placeholder="mp4,mkv,avi,mov,wmv" />
						</div>
					</div>
				</div>

				<!-- Display Settings -->
				<div v-if="activeTab === 'display'" class="settings-section">
					<div class="setting-item">
						<div class="setting-info">
							<h4>Thumbnail Quality</h4>
							<p>Quality of generated thumbnails (higher quality = larger file size)</p>
						</div>
						<div class="setting-control">
							<select v-model="settings.thumbnailQuality" class="form-select">
								<option value="low">Low (Fast)</option>
								<option value="medium">Medium (Balanced)</option>
								<option value="high">High (Best Quality)</option>
							</select>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Grid Card Size</h4>
							<p>Size of cards in grid view</p>
						</div>
						<div class="setting-control">
							<select v-model="settings.gridCardSize" class="form-select">
								<option value="small">Small</option>
								<option value="medium">Medium</option>
								<option value="large">Large</option>
							</select>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Show File Extensions</h4>
							<p>Display file extensions in video titles</p>
						</div>
						<div class="setting-control">
							<div class="form-check form-switch">
								<input v-model="settings.showFileExtensions" class="form-check-input" type="checkbox" id="extensionsSwitch" />
								<label class="form-check-label" for="extensionsSwitch">
									{{ settings.showFileExtensions ? 'Show' : 'Hide' }}
								</label>
							</div>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Animated Previews</h4>
							<p>Show animated previews on hover (requires preview generation)</p>
						</div>
						<div class="setting-control">
							<div class="form-check form-switch">
								<input v-model="settings.animatedPreviews" class="form-check-input" type="checkbox" id="previewsSwitch" />
								<label class="form-check-label" for="previewsSwitch">
									{{ settings.animatedPreviews ? 'Enabled' : 'Disabled' }}
								</label>
							</div>
						</div>
					</div>
				</div>

				<!-- Privacy Settings -->
				<div v-if="activeTab === 'privacy'" class="settings-section">
					<div class="setting-item">
						<div class="setting-info">
							<h4>
								<font-awesome-icon :icon="['fas', 'dog']" class="me-2 text-danger" />
								Default Zoo Filter
							</h4>
							<p>Default filter setting for zoo content when viewing performers</p>
						</div>
						<div class="setting-control">
							<select v-model="settings.defaultZooFilter" class="form-select">
								<option value="all">Show All</option>
								<option value="zoo-only">Zoo Only</option>
								<option value="non-zoo">Non-Zoo Only</option>
							</select>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Blur Thumbnails</h4>
							<p>Blur thumbnails until hovered over for privacy</p>
						</div>
						<div class="setting-control">
							<div class="form-check form-switch">
								<input v-model="settings.blurThumbnails" class="form-check-input" type="checkbox" id="blurSwitch" />
								<label class="form-check-label" for="blurSwitch">
									{{ settings.blurThumbnails ? 'Enabled' : 'Disabled' }}
								</label>
							</div>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Show Metadata</h4>
							<p>Display performer metadata (age, measurements, etc.)</p>
						</div>
						<div class="setting-control">
							<div class="form-check form-switch">
								<input v-model="settings.showMetadata" class="form-check-input" type="checkbox" id="metadataSwitch" />
								<label class="form-check-label" for="metadataSwitch">
									{{ settings.showMetadata ? 'Show' : 'Hide' }}
								</label>
							</div>
						</div>
					</div>
				</div>

				<!-- About -->
				<div v-if="activeTab === 'about'" class="settings-section">
					<div class="about-content">
						<div class="app-info">
							<h3>
								<font-awesome-icon :icon="['fas', 'video']" class="me-3 text-primary" />
								Video Storage AI
							</h3>
							<p class="version">Version 1.0.0</p>
							<p class="description">
								A powerful video library management system with AI-powered organization, metadata management, and intelligent search capabilities.
							</p>
						</div>

						<div class="features-list">
							<h4>Features</h4>
							<ul>
								<li>
									<font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
									Automatic video cataloging and organization
								</li>
								<li>
									<font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
									Performer and studio database management
								</li>
								<li>
									<font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
									Advanced filtering and search capabilities
								</li>
								<li>
									<font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
									Metadata integration and enrichment
								</li>
								<li>
									<font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
									Tag-based organization system
								</li>
								<li>
									<font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
									Thumbnail generation and preview support
								</li>
							</ul>
						</div>

						<div class="tech-stack">
							<h4>Technology</h4>
							<div class="tech-badges">
								<span class="badge bg-primary">Vue.js 3</span>
								<span class="badge bg-success">Go + Gin</span>
								<span class="badge bg-info">SQLite</span>
								<span class="badge bg-warning text-dark">FFmpeg</span>
							</div>
						</div>

						<div class="links-section mt-4">
							<h4>Links</h4>
							<div class="link-buttons">
								<button class="btn btn-outline-primary">
									<font-awesome-icon :icon="['fas', 'book']" class="me-2" />
									Documentation
								</button>
								<button class="btn btn-outline-secondary">
									<font-awesome-icon :icon="['fas', 'code-branch']" class="me-2" />
									GitHub
								</button>
								<button class="btn btn-outline-info">
									<font-awesome-icon :icon="['fas', 'question-circle']" class="me-2" />
									Support
								</button>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Save Button -->
			<div v-if="activeTab !== 'about'" class="settings-footer">
				<button class="btn btn-primary btn-lg" @click="saveSettings">
					<font-awesome-icon :icon="['fas', 'save']" class="me-2" />
					Save Settings
				</button>
				<button class="btn btn-secondary btn-lg ms-3" @click="resetSettings">
					<font-awesome-icon :icon="['fas', 'sync']" class="me-2" />
					Reset to Defaults
				</button>
			</div>
		</div>
	</div>
</template>

<script>
import settingsService from '@/services/settingsService'

export default {
	name: 'SettingsPage',
	data() {
		return {
			activeTab: 'general',
			settings: settingsService.getSettings(),
		}
	},
	watch: {
		// Watch for theme changes and apply immediately
		'settings.theme'(newTheme) {
			settingsService.updateSetting('theme', newTheme)
			settingsService.applyTheme()
		},
	},
	methods: {
		saveSettings() {
			const success = settingsService.saveSettings(this.settings)
			if (success) {
				settingsService.applyTheme()
				this.$toast.success('Settings Saved', 'Your preferences have been updated')
			} else {
				this.$toast.error('Save Failed', 'Could not save settings')
			}
		},
		resetSettings() {
			if (confirm('Are you sure you want to reset all settings to their default values?')) {
				settingsService.resetSettings()
				this.settings = settingsService.getSettings()
				settingsService.applyTheme()
				this.$toast.info('Settings Reset', 'All settings have been restored to defaults')
			}
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/settings_page.css';
</style>
