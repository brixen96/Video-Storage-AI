<template>
	<div v-if="showDebug" class="cache-debug-panel">
		<div class="cache-debug-header">
			<h6>Cache Storage Debug</h6>
			<button class="btn btn-sm btn-outline-light" @click="refreshStats">
				<font-awesome-icon :icon="['fas', 'sync']" />
			</button>
			<button class="btn btn-sm btn-outline-danger" @click="clearCache">
				<font-awesome-icon :icon="['fas', 'trash']" />
			</button>
			<button class="btn btn-sm btn-outline-secondary" @click="showDebug = false">
				<font-awesome-icon :icon="['fas', 'times']" />
			</button>
		</div>
		<div class="cache-debug-content">
			<div v-if="stats.available">
				<div class="cache-stat">
					<strong>Total Entries:</strong> {{ stats.totalEntries }}
				</div>
				<div v-for="entry in stats.entries" :key="entry.url" class="cache-entry">
					<div class="cache-entry-url">{{ extractPath(entry.url) }}</div>
					<div class="cache-entry-info">
						<span :class="['cache-status', entry.expired ? 'expired' : 'valid']">
							{{ entry.expired ? 'Expired' : 'Valid' }}
						</span>
						<span class="cache-age">Age: {{ formatAge(entry.age) }}</span>
						<span class="cache-ttl">Expires in: {{ formatAge(entry.expiresIn) }}</span>
					</div>
				</div>
			</div>
			<div v-else class="cache-unavailable">Cache Storage API not available</div>
		</div>
	</div>
	<button v-else class="cache-debug-toggle" @click="showDebugPanel" title="Show Cache Debug">
		<font-awesome-icon :icon="['fas', 'database']" />
	</button>
</template>

<script>
import cacheService from '@/services/cacheService'

export default {
	name: 'CacheDebugPanel',
	data() {
		return {
			showDebug: false,
			stats: {
				available: false,
				totalEntries: 0,
				entries: [],
			},
		}
	},
	methods: {
		async showDebugPanel() {
			this.showDebug = true
			await this.refreshStats()
		},
		async refreshStats() {
			this.stats = await cacheService.getStats()
		},
		async clearCache() {
			if (confirm('Are you sure you want to clear the cache?')) {
				await cacheService.clear()
				this.$toast?.success('Cache cleared successfully')
				await this.refreshStats()
			}
		},
		extractPath(url) {
			try {
				const urlObj = new URL(url)
				return urlObj.pathname
			} catch {
				return url
			}
		},
		formatAge(ms) {
			if (ms < 1000) return `${ms}ms`
			const seconds = Math.floor(ms / 1000)
			if (seconds < 60) return `${seconds}s`
			const minutes = Math.floor(seconds / 60)
			const remainingSeconds = seconds % 60
			return `${minutes}m ${remainingSeconds}s`
		},
	},
}
</script>

<style scoped>
.cache-debug-panel {
	position: fixed;
	bottom: 20px;
	right: 20px;
	width: 400px;
	max-height: 500px;
	background: rgba(0, 0, 0, 0.9);
	border: 1px solid rgba(255, 255, 255, 0.2);
	border-radius: 8px;
	color: white;
	font-size: 12px;
	z-index: 9999;
	box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
}

.cache-debug-header {
	display: flex;
	align-items: center;
	gap: 8px;
	padding: 12px;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.cache-debug-header h6 {
	margin: 0;
	flex: 1;
	font-size: 14px;
	font-weight: 600;
}

.cache-debug-content {
	padding: 12px;
	max-height: 400px;
	overflow-y: auto;
}

.cache-stat {
	margin-bottom: 12px;
	padding-bottom: 12px;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.cache-entry {
	margin-bottom: 12px;
	padding: 8px;
	background: rgba(255, 255, 255, 0.05);
	border-radius: 4px;
}

.cache-entry-url {
	font-family: monospace;
	margin-bottom: 4px;
	word-break: break-all;
	color: #4fc3f7;
}

.cache-entry-info {
	display: flex;
	gap: 12px;
	font-size: 11px;
}

.cache-status {
	padding: 2px 6px;
	border-radius: 3px;
	font-weight: 600;
}

.cache-status.valid {
	background: rgba(76, 175, 80, 0.3);
	color: #81c784;
}

.cache-status.expired {
	background: rgba(244, 67, 54, 0.3);
	color: #e57373;
}

.cache-age,
.cache-ttl {
	color: rgba(255, 255, 255, 0.6);
}

.cache-unavailable {
	color: rgba(255, 255, 255, 0.5);
	text-align: center;
	padding: 20px;
}

.cache-debug-toggle {
	position: fixed;
	bottom: 20px;
	right: 20px;
	width: 50px;
	height: 50px;
	background: rgba(0, 0, 0, 0.8);
	border: 1px solid rgba(255, 255, 255, 0.2);
	border-radius: 50%;
	color: white;
	cursor: pointer;
	z-index: 9999;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 20px;
	transition: all 0.3s;
}

.cache-debug-toggle:hover {
	background: rgba(0, 0, 0, 0.9);
	transform: scale(1.1);
}
</style>
