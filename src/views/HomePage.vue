<template>
	<div class="dashboard-page">
		<!-- Header -->
		<div class="dashboard-header">
			<div class="header-content">
				<h1 class="dashboard-title">Dashboard</h1>
				<p class="dashboard-subtitle">Video Storage AI Control Center</p>
			</div>
			<div class="header-actions">
				<button class="btn-refresh" @click="refreshAll" :disabled="isRefreshing">
					<font-awesome-icon :icon="['fas', 'sync']" :spin="isRefreshing" />
					{{ isRefreshing ? 'Refreshing...' : 'Refresh' }}
				</button>
			</div>
		</div>

		<!-- Stats Row -->
		<div class="stats-row">
			<DashboardStatCard
				:value="stats.total_videos"
				label="Local Videos"
				icon="video"
				color="blue"
			/>
			<DashboardStatCard
				:value="stats.total_scraped_videos"
				label="Scraped Videos"
				icon="download"
				color="purple"
			/>
			<DashboardStatCard
				:value="stats.total_threads"
				label="Threads"
				icon="comments"
				color="pink"
			/>
			<DashboardStatCard
				:value="stats.total_performers"
				label="Performers"
				icon="users"
				color="green"
			/>
			<DashboardStatCard
				:value="stats.running_tasks"
				label="Running"
				icon="spinner"
				color="yellow"
			/>
			<DashboardStatCard
				:value="stats.recent_new_items"
				label="New (24h)"
				icon="bell"
				color="orange"
			/>
		</div>

		<!-- Main Content -->
		<div class="dashboard-content">
			<!-- Left Column -->
			<div class="left-column">
				<RunningTasksWidget :tasks="runningTasks" />
				<DashboardFeed
					:items="feedItems"
					:hasMore="feedHasMore"
					@load-more="loadMoreFeed"
				/>
			</div>

			<!-- Right Column -->
			<div class="right-column">
				<!-- Enabled Scrapers Section -->
				<div class="scrapers-section" v-if="enabledScrapers.length > 0">
					<div class="section-header">
						<h3>
							<font-awesome-icon :icon="['fas', 'robot']" class="me-2" />
							Active Auto-Update Scrapers
						</h3>
					</div>
					<div class="scrapers-list">
						<ScraperStatusCard
							v-for="scraper in enabledScrapers"
							:key="scraper.scraper_type"
							:config="scraper"
							:isChecking="checkingScrapers[scraper.scraper_type]"
							@toggle="(enabled) => toggleScraper(scraper.scraper_type, enabled)"
							@check-now="checkScraperNow(scraper.scraper_type)"
							@settings="openScraperSettings(scraper)"
						/>
					</div>
				</div>

				<!-- No Active Scrapers Message -->
				<div class="no-scrapers-card" v-else>
					<font-awesome-icon :icon="['fas', 'robot']" class="empty-icon" />
					<h4>No Active Scrapers</h4>
					<p>Enable auto-update on scrapers from the Scrapers page to see them here.</p>
					<router-link to="/scrapers" class="btn btn-primary btn-sm">
						<font-awesome-icon :icon="['fas', 'cog']" class="me-2" />
						Configure Scrapers
					</router-link>
				</div>
			</div>
		</div>

		<!-- Scraper Settings Modal -->
		<div v-if="selectedScraper" class="modal-overlay" @click.self="selectedScraper = null">
			<div class="scraper-settings-modal">
				<div class="modal-header">
					<h4>
						<font-awesome-icon :icon="['fas', selectedScraper.icon || 'globe']" class="me-2" />
						{{ selectedScraper.display_name }} Settings
					</h4>
					<button class="btn-close" @click="selectedScraper = null">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
				<div class="modal-body">
					<div class="setting-row">
						<label>Check Interval (minutes)</label>
						<input
							type="number"
							v-model.number="scraperSettings.interval_minutes"
							min="5"
							max="1440"
							class="form-control"
						/>
					</div>
					<div class="setting-row">
						<label>Max Pages to Check</label>
						<input
							type="number"
							v-model.number="scraperSettings.max_pages_to_check"
							min="1"
							max="20"
							class="form-control"
						/>
					</div>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="selectedScraper = null">Cancel</button>
					<button class="btn btn-primary" @click="saveScraperSettings">Save Changes</button>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import { dashboardAPI, autoUpdateAPI } from '@/services/api'
import DashboardStatCard from '@/components/dashboard/DashboardStatCard.vue'
import DashboardFeed from '@/components/dashboard/DashboardFeed.vue'
import RunningTasksWidget from '@/components/dashboard/RunningTasksWidget.vue'
import ScraperStatusCard from '@/components/dashboard/ScraperStatusCard.vue'

export default {
	name: 'HomePage',
	components: {
		DashboardStatCard,
		DashboardFeed,
		RunningTasksWidget,
		ScraperStatusCard,
	},
	data() {
		return {
			isRefreshing: false,
			stats: {
				total_videos: 0,
				total_scraped_videos: 0,
				total_threads: 0,
				total_performers: 0,
				watched_threads: 0,
				running_tasks: 0,
				pending_tasks: 0,
				recent_new_items: 0,
			},
			feedItems: [],
			feedOffset: 0,
			feedLimit: 20,
			feedTotal: 0,
			runningTasks: [],
			scraperConfigs: [],
			checkingScrapers: {},
			selectedScraper: null,
			scraperSettings: {
				interval_minutes: 60,
				max_pages_to_check: 3,
			},
			refreshInterval: null,
		}
	},
	computed: {
		feedHasMore() {
			return this.feedOffset + this.feedItems.length < this.feedTotal
		},
		enabledScrapers() {
			return this.scraperConfigs.filter((s) => s.enabled)
		},
	},
	async mounted() {
		await this.loadAll()
		// Auto-refresh every 30 seconds
		this.refreshInterval = setInterval(() => {
			this.loadRunningTasks()
			this.loadStats()
		}, 30000)
	},
	beforeUnmount() {
		if (this.refreshInterval) {
			clearInterval(this.refreshInterval)
		}
	},
	methods: {
		async loadAll() {
			await Promise.all([
				this.loadStats(),
				this.loadFeed(),
				this.loadRunningTasks(),
				this.loadScraperConfigs(),
			])
		},
		async refreshAll() {
			this.isRefreshing = true
			try {
				await this.loadAll()
			} finally {
				this.isRefreshing = false
			}
		},
		async loadStats() {
			try {
				const response = await dashboardAPI.getStats()
				if (response?.data) {
					this.stats = response.data
				}
			} catch (error) {
				console.error('Failed to load dashboard stats:', error)
			}
		},
		async loadFeed() {
			try {
				const response = await dashboardAPI.getFeed(this.feedLimit, this.feedOffset)
				if (response) {
					this.feedItems = response.data || []
					this.feedTotal = response.pagination?.total || 0
				}
			} catch (error) {
				console.error('Failed to load feed:', error)
			}
		},
		async loadMoreFeed() {
			this.feedOffset += this.feedLimit
			try {
				const response = await dashboardAPI.getFeed(this.feedLimit, this.feedOffset)
				if (response?.data) {
					this.feedItems = [...this.feedItems, ...response.data]
				}
			} catch (error) {
				console.error('Failed to load more feed items:', error)
			}
		},
		async loadRunningTasks() {
			try {
				const response = await dashboardAPI.getRunningTasks()
				if (response?.data) {
					this.runningTasks = response.data
				}
			} catch (error) {
				console.error('Failed to load running tasks:', error)
			}
		},
		async loadScraperConfigs() {
			try {
				const response = await dashboardAPI.getScrapers()
				if (response?.data) {
					this.scraperConfigs = response.data
				}
			} catch (error) {
				console.error('Failed to load scraper configs:', error)
			}
		},
		async toggleScraper(scraperType, enabled) {
			try {
				await autoUpdateAPI.toggle(scraperType, enabled)
				// Update local state
				const config = this.scraperConfigs.find((c) => c.scraper_type === scraperType)
				if (config) {
					config.enabled = enabled
				}
			} catch (error) {
				console.error('Failed to toggle scraper:', error)
			}
		},
		async checkScraperNow(scraperType) {
			this.checkingScrapers = { ...this.checkingScrapers, [scraperType]: true }
			try {
				await autoUpdateAPI.checkNow(scraperType)
				// Reload after a short delay to get updated stats
				setTimeout(() => {
					this.loadScraperConfigs()
					this.loadFeed()
					this.loadStats()
				}, 2000)
			} catch (error) {
				console.error('Failed to check scraper:', error)
			} finally {
				this.checkingScrapers = { ...this.checkingScrapers, [scraperType]: false }
			}
		},
		openScraperSettings(scraper) {
			this.selectedScraper = scraper
			this.scraperSettings = {
				interval_minutes: scraper.interval_minutes || 60,
				max_pages_to_check: scraper.max_pages_to_check || 3,
			}
		},
		async saveScraperSettings() {
			if (!this.selectedScraper) return
			try {
				await autoUpdateAPI.updateConfig(this.selectedScraper.scraper_type, this.scraperSettings)
				// Update local state
				const config = this.scraperConfigs.find(
					(c) => c.scraper_type === this.selectedScraper.scraper_type
				)
				if (config) {
					config.interval_minutes = this.scraperSettings.interval_minutes
					config.max_pages_to_check = this.scraperSettings.max_pages_to_check
				}
				this.selectedScraper = null
			} catch (error) {
				console.error('Failed to save scraper settings:', error)
			}
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/dashboard_page.css';
</style>
