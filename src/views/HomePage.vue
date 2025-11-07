<template>
	<div class="home-page">
		<div class="container-fluid py-5">
			<!-- Hero Section -->
			<div class="row mb-5">
				<div class="col-12 text-center">
					<h1 class="display-3 fw-bold mb-3 gradient-text">Welcome to Video Storage AI</h1>
					<p class="lead mb-4">Your intelligent video library manager with AI-powered organization</p>
				</div>
			</div>

			<!-- Stats Cards -->
			<div class="row g-4 mb-5">
				<div class="col-md-3">
					<div class="stat-card card bg-gradient-primary text-white">
						<div class="card-body">
							<div class="d-flex justify-content-between align-items-center">
								<div>
									<h3 class="mb-0">{{ stats.performers }}</h3>
									<p class="mb-0">Performers</p>
								</div>
								<font-awesome-icon :icon="['fas', 'users']" class="stat-icon" />
							</div>
						</div>
					</div>
				</div>

				<div class="col-md-3">
					<div class="stat-card card bg-gradient-success text-white">
						<div class="card-body">
							<div class="d-flex justify-content-between align-items-center">
								<div>
									<h3 class="mb-0">{{ stats.videos }}</h3>
									<p class="mb-0">Videos</p>
								</div>
								<font-awesome-icon :icon="['fas', 'video']" class="stat-icon" />
							</div>
						</div>
					</div>
				</div>

				<div class="col-md-3">
					<div class="stat-card card bg-gradient-info text-white">
						<div class="card-body">
							<div class="d-flex justify-content-between align-items-center">
								<div>
									<h3 class="mb-0">{{ stats.studios }}</h3>
									<p class="mb-0">Studios</p>
								</div>
								<font-awesome-icon :icon="['fas', 'building']" class="stat-icon" />
							</div>
						</div>
					</div>
				</div>

				<div class="col-md-3">
					<div class="stat-card card bg-gradient-warning text-white">
						<div class="card-body">
							<div class="d-flex justify-content-between align-items-center">
								<div>
									<h3 class="mb-0">{{ stats.tags }}</h3>
									<p class="mb-0">Tags</p>
								</div>
								<font-awesome-icon :icon="['fas', 'tags']" class="stat-icon" />
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Quick Actions -->
			<div class="row mb-5">
				<div class="col-12">
					<h2 class="mb-4">Quick Actions</h2>
				</div>
				<div class="col-md-4 mb-3">
					<router-link to="/performers" class="action-card card h-100">
						<div class="card-body text-center">
							<font-awesome-icon :icon="['fas', 'users']" class="action-icon mb-3" />
							<h5 class="card-title">Browse Performers</h5>
							<p class="card-text">View and manage your performer collection</p>
						</div>
					</router-link>
				</div>

				<div class="col-md-4 mb-3">
					<router-link to="/videos" class="action-card card h-100">
						<div class="card-body text-center">
							<font-awesome-icon :icon="['fas', 'video']" class="action-icon mb-3" />
							<h5 class="card-title">Browse Videos</h5>
							<p class="card-text">Explore your video library</p>
						</div>
					</router-link>
				</div>

				<div class="col-md-4 mb-3">
					<router-link to="/activity" class="action-card card h-100">
						<div class="card-body text-center">
							<font-awesome-icon :icon="['fas', 'chart-line']" class="action-icon mb-3" />
							<h5 class="card-title">Activity Monitor</h5>
							<p class="card-text">Track scanning and AI operations</p>
						</div>
					</router-link>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import { performersAPI, videosAPI, studiosAPI, tagsAPI } from '@/services/api'

export default {
	name: 'HomePage',
	data() {
		return {
			stats: {
				performers: 0,
				videos: 0,
				studios: 0,
				tags: 0,
			},
		}
	},
	async mounted() {
		await this.loadStats()
	},
	methods: {
		async loadStats() {
			try {
				const [performers, videos, studios, tags] = await Promise.all([
					performersAPI.getAll().catch(() => ({ data: [] })),
					videosAPI.getAll().catch(() => ({ data: [] })),
					studiosAPI.getAll().catch(() => ({ data: [] })),
					tagsAPI.getAll().catch(() => ({ data: [] })),
				])

				this.stats.performers = performers.data?.length || 0
				this.stats.videos = videos.data?.length || 0
				this.stats.studios = studios.data?.length || 0
				this.stats.tags = tags.data?.length || 0
			} catch (error) {
				console.error('Failed to load stats:', error)
			}
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/home_page.css';
</style>
