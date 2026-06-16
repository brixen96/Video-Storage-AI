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
			<DashboardStatCard :value="stats.total_videos" label="Local Videos" icon="video" color="blue" to="/videos" />
			<DashboardStatCard :value="stats.total_scraped_videos" label="Scraped Videos" icon="film" color="purple" to="/scraper" />
			<DashboardStatCard :value="stats.total_downloaded_scraped" label="Downloaded" icon="download" color="yellow" to="/scraper" />
			<DashboardStatCard :value="stats.total_threads" label="Threads" icon="comments" color="pink" to="/scraper" />
			<DashboardStatCard :value="stats.total_performers" label="Performers" icon="users" color="green" to="/performers" />
			<DashboardStatCard :value="stats.recent_new_items" label="New (24h)" icon="bell" color="orange" />
			<DashboardStatCard :value="stats.videos_without_performer" label="Unlinked Videos" icon="user-slash" color="pink" to="/videos?no_performer=true" />
		</div>

		<!-- Top Bar: Running Tasks + Active Scrapers -->
		<div class="top-bar">
			<!-- Running Tasks (compact) -->
			<RunningTasksWidget :tasks="runningTasks" compact />

			<!-- Active Scrapers (compact inline list) -->
			<div class="active-scrapers-bar" v-if="enabledScrapers.length > 0">
				<span class="bar-label">
					<font-awesome-icon :icon="['fas', 'robot']" />
					Active Scrapers
				</span>
				<div class="scraper-chips-row">
					<div
						v-for="scraper in enabledScrapers"
						:key="scraper.scraper_type"
						class="scraper-chip"
					>
						<span class="chip-dot" :class="scraper.is_checking ? 'checking' : 'active'"></span>
						<span class="chip-name">{{ scraper.display_name || scraper.scraper_type }}</span>
						<button class="chip-btn" @click="checkScraperNow(scraper.scraper_type)" :disabled="checkingScrapers[scraper.scraper_type]" title="Check now">
							<font-awesome-icon :icon="['fas', checkingScrapers[scraper.scraper_type] ? 'spinner' : 'sync']" :spin="checkingScrapers[scraper.scraper_type]" />
						</button>
						<button class="chip-btn" @click="openScraperSettings(scraper)" title="Settings">
							<font-awesome-icon :icon="['fas', 'cog']" />
						</button>
					</div>
				</div>
			</div>
			<div class="active-scrapers-bar" v-else>
				<span class="bar-label muted">
					<font-awesome-icon :icon="['fas', 'robot']" />
					No Active Scrapers
				</span>
				<router-link to="/scraper" class="btn-configure">Configure</router-link>
			</div>
		</div>

		<!-- Video Shelves -->
		<div v-if="recentVideos.length > 0 || mostWatched.length > 0" class="video-shelves">
			<!-- Recently Added -->
			<div v-if="recentVideos.length > 0" class="video-shelf">
				<div class="shelf-header">
					<span class="shelf-title">
						<font-awesome-icon :icon="['fas', 'clock']" class="me-2" />
						Recently Added
					</span>
					<router-link to="/videos" class="shelf-more">View all</router-link>
				</div>
				<div class="shelf-track">
					<div
						v-for="video in recentVideos"
						:key="video.id"
						class="shelf-card"
						@click="openVideo(video)"
					>
						<div class="shelf-thumb">
							<img v-if="video.thumbnail_path" :src="getShelfThumb(video)" loading="lazy" />
							<div v-else class="shelf-thumb-placeholder">
								<font-awesome-icon :icon="['fas', 'video']" />
							</div>
							<div v-if="video.duration" class="shelf-duration">{{ formatDuration(video.duration) }}</div>
						</div>
						<div class="shelf-info">
							<div class="shelf-video-title" :title="video.title">{{ video.title }}</div>
							<div v-if="video.performers && video.performers.length" class="shelf-performer">
								{{ video.performers[0].name }}
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Most Watched -->
			<div v-if="mostWatched.length > 0" class="video-shelf">
				<div class="shelf-header">
					<span class="shelf-title">
						<font-awesome-icon :icon="['fas', 'eye']" class="me-2" />
						Most Watched
					</span>
					<router-link to="/videos?sort=play_count" class="shelf-more">View all</router-link>
				</div>
				<div class="shelf-track">
					<div
						v-for="video in mostWatched"
						:key="video.id"
						class="shelf-card"
						@click="openVideo(video)"
					>
						<div class="shelf-thumb">
							<img v-if="video.thumbnail_path" :src="getShelfThumb(video)" loading="lazy" />
							<div v-else class="shelf-thumb-placeholder">
								<font-awesome-icon :icon="['fas', 'video']" />
							</div>
							<div v-if="video.duration" class="shelf-duration">{{ formatDuration(video.duration) }}</div>
							<div class="shelf-play-count">
								<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
								{{ video.play_count }}
							</div>
						</div>
						<div class="shelf-info">
							<div class="shelf-video-title" :title="video.title">{{ video.title }}</div>
							<div v-if="video.performers && video.performers.length" class="shelf-performer">
								{{ video.performers[0].name }}
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Activity Feed — Full Width -->
		<DashboardFeed
			:items="feedItems"
			:hasMore="feedHasMore"
			:scraperTypes="feedScraperTypes"
			:activeFilter="feedScraperFilter"
			:allScraperTypes="enabledScrapers"
			@load-more="loadMoreFeed"
			@filter-change="onFeedFilterChange"
			@clear-feed="clearFeed"
		/>

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
						<input type="number" v-model.number="scraperSettings.interval_minutes" min="5" max="1440" class="form-control" />
					</div>
					<div class="setting-row">
						<label>Max Pages to Check</label>
						<input type="number" v-model.number="scraperSettings.max_pages_to_check" min="1" max="20" class="form-control" />
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
import { dashboardAPI, autoUpdateAPI, videosAPI, getAssetURL } from '@/services/api'
import { useFormatters } from '@/composables/useFormatters'
import websocketService from '@/services/websocket'
import DashboardStatCard from '@/components/dashboard/DashboardStatCard.vue'
import DashboardFeed from '@/components/dashboard/DashboardFeed.vue'
import RunningTasksWidget from '@/components/dashboard/RunningTasksWidget.vue'

export default {
	name: 'HomePage',
	components: {
		DashboardStatCard,
		DashboardFeed,
		RunningTasksWidget,
	},
	data() {
		return {
			isRefreshing: false,
			stats: {
				total_videos: 0,
				total_scraped_videos: 0,
				total_downloaded_scraped: 0,
				total_threads: 0,
				total_performers: 0,
				watched_threads: 0,
				running_tasks: 0,
				pending_tasks: 0,
				recent_new_items: 0,
			},
			recentVideos: [],
			mostWatched: [],
			feedItems: [],
			feedOffset: 0,
			feedLimit: 20,
			feedTotal: 0,
			feedScraperFilter: '',
			feedScraperTypes: [],
			runningTasks: [],
			scraperConfigs: [],
			checkingScrapers: {},
			selectedScraper: null,
			scraperSettings: {
				interval_minutes: 60,
				max_pages_to_check: 3,
			},
			refreshInterval: null,
			wsUnsubscribers: [],
		}
	},
	created() {
		const { formatDuration } = useFormatters()
		this.formatDuration = formatDuration
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
		this.refreshInterval = setInterval(() => {
			this.loadRunningTasks()
			this.loadStats()
		}, 30000)

		websocketService.connect()
		this.wsUnsubscribers.push(
			websocketService.on('feed_update', () => {
				this.loadFeed()
				this.loadStats()
				this.loadScraperConfigs()
				this.loadRunningTasks()
			}),
			websocketService.on('activity_update', () => {
				this.loadRunningTasks()
			}),
			websocketService.on('status_update', () => {
				this.loadRunningTasks()
				this.loadStats()
			})
		)
	},
	onActivated() {
		this.loadAll()
	},
	beforeUnmount() {
		if (this.refreshInterval) clearInterval(this.refreshInterval)
		this.wsUnsubscribers.forEach((unsub) => unsub())
		this.wsUnsubscribers = []
	},
	methods: {
		async loadAll() {
			await Promise.all([
				this.loadStats(),
				this.loadFeed(),
				this.loadRunningTasks(),
				this.loadScraperConfigs(),
				this.loadVideoShelves(),
			])
		},
		async refreshAll() {
			this.isRefreshing = true
			try { await this.loadAll() }
			finally { this.isRefreshing = false }
		},
		async loadStats() {
			try {
				const response = await dashboardAPI.getStats()
				if (response?.data) this.stats = response.data
			} catch (error) { console.error('Failed to load stats:', error) }
		},
		async loadFeed() {
			try {
				const response = await dashboardAPI.getFeed(this.feedLimit, this.feedOffset, this.feedScraperFilter)
				if (response) {
					this.feedItems = response.data || []
					this.feedTotal = response.pagination?.total || 0
					if (response.scraper_types) this.feedScraperTypes = response.scraper_types
				}
			} catch (error) { console.error('Failed to load feed:', error) }
		},
		async loadMoreFeed() {
			this.feedOffset += this.feedLimit
			try {
				const response = await dashboardAPI.getFeed(this.feedLimit, this.feedOffset, this.feedScraperFilter)
				if (response?.data) this.feedItems = [...this.feedItems, ...response.data]
			} catch (error) { console.error('Failed to load more feed:', error) }
		},
		async onFeedFilterChange(scraperType) {
			this.feedScraperFilter = scraperType
			this.feedOffset = 0
			await this.loadFeed()
		},
		async clearFeed() {
			try {
				await dashboardAPI.clearFeed()
				this.feedItems = []
				this.feedTotal = 0
				this.feedOffset = 0
			} catch (error) { console.error('Failed to clear feed:', error) }
		},
		async loadRunningTasks() {
			try {
				const response = await dashboardAPI.getRunningTasks()
				if (response?.data) this.runningTasks = response.data
			} catch (error) { console.error('Failed to load tasks:', error) }
		},
		async loadScraperConfigs() {
			try {
				const response = await dashboardAPI.getScrapers()
				if (response?.data) this.scraperConfigs = response.data
			} catch (error) { console.error('Failed to load scrapers:', error) }
		},
		async toggleScraper(scraperType, enabled) {
			try {
				await autoUpdateAPI.toggle(scraperType, enabled)
				const config = this.scraperConfigs.find((c) => c.scraper_type === scraperType)
				if (config) config.enabled = enabled
			} catch (error) { console.error('Failed to toggle scraper:', error) }
		},
		async checkScraperNow(scraperType) {
			this.checkingScrapers = { ...this.checkingScrapers, [scraperType]: true }
			try {
				await autoUpdateAPI.checkNow(scraperType)
				setTimeout(() => {
					this.loadScraperConfigs()
					this.loadFeed()
					this.loadStats()
				}, 2000)
			} catch (error) { console.error('Failed to check scraper:', error) }
			finally { this.checkingScrapers = { ...this.checkingScrapers, [scraperType]: false } }
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
				const config = this.scraperConfigs.find((c) => c.scraper_type === this.selectedScraper.scraper_type)
				if (config) {
					config.interval_minutes = this.scraperSettings.interval_minutes
					config.max_pages_to_check = this.scraperSettings.max_pages_to_check
				}
				this.selectedScraper = null
			} catch (error) { console.error('Failed to save settings:', error) }
		},
		async loadVideoShelves() {
			try {
				const [recentRes, watchedRes] = await Promise.allSettled([
					videosAPI.getAll({ sort_by: 'created_at', sort_order: 'desc', limit: 10, page: 1 }),
					videosAPI.getAll({ sort_by: 'play_count', sort_order: 'desc', limit: 10, page: 1 }),
				])
				if (recentRes.status === 'fulfilled') {
					const raw = recentRes.value
					this.recentVideos = (Array.isArray(raw) ? raw : raw?.data ?? []).slice(0, 10)
				}
				if (watchedRes.status === 'fulfilled') {
					const raw = watchedRes.value
					const videos = (Array.isArray(raw) ? raw : raw?.data ?? [])
					// Only show most-watched shelf if there are actually watched videos
					this.mostWatched = videos.filter(v => v.play_count > 0).slice(0, 10)
				}
			} catch (error) { console.error('Failed to load video shelves:', error) }
		},
		getShelfThumb(video) {
			if (!video.thumbnail_path) return ''
			if (video.thumbnail_path.startsWith('http')) return video.thumbnail_path
			return getAssetURL(video.thumbnail_path)
		},
		openVideo(video) {
			const route = this.$router.resolve(`/watch/${video.id}`)
			window.open(route.href, '_blank')
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/dashboard_page.css';
</style>
