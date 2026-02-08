<template>
	<div class="scraper-card" :class="{ enabled: config.enabled, [config.color]: true }">
		<div class="scraper-header">
			<div class="scraper-info">
				<font-awesome-icon :icon="['fas', config.icon || 'globe']" class="scraper-icon" />
				<span class="scraper-name">{{ config.display_name }}</span>
			</div>
			<label class="toggle-switch">
				<input type="checkbox" :checked="config.enabled" @change="toggleEnabled" />
				<span class="slider"></span>
			</label>
		</div>
		<div class="scraper-stats">
			<div class="stat-row">
				<span class="stat-label">Last checked:</span>
				<span class="stat-value">{{ formatLastChecked }}</span>
			</div>
			<div class="stat-row">
				<span class="stat-label">Last found:</span>
				<span class="stat-value">{{ config.last_found_count }} items</span>
			</div>
			<div class="stat-row">
				<span class="stat-label">Total items:</span>
				<span class="stat-value">{{ formatNumber(config.total_items) }}</span>
			</div>
			<div class="stat-row" v-if="config.enabled && config.next_scheduled_run">
				<span class="stat-label">Next run:</span>
				<span class="stat-value">{{ formatNextRun }}</span>
			</div>
		</div>
		<div class="scraper-actions">
			<button class="btn-check-now" @click="$emit('check-now')" :disabled="isChecking">
				<font-awesome-icon :icon="['fas', isChecking ? 'spinner' : 'sync']" :spin="isChecking" />
				{{ isChecking ? 'Checking...' : 'Check Now' }}
			</button>
			<button class="btn-settings" @click="$emit('settings')">
				<font-awesome-icon :icon="['fas', 'cog']" />
			</button>
		</div>
	</div>
</template>

<script>
export default {
	name: 'ScraperStatusCard',
	props: {
		config: {
			type: Object,
			required: true,
		},
		isChecking: {
			type: Boolean,
			default: false,
		},
	},
	emits: ['toggle', 'check-now', 'settings'],
	computed: {
		formatLastChecked() {
			if (!this.config.last_checked_at) return 'Never'
			const date = new Date(this.config.last_checked_at)
			const now = new Date()
			const diffMs = now - date
			const diffMins = Math.floor(diffMs / 60000)
			const diffHours = Math.floor(diffMs / 3600000)
			if (diffMins < 1) return 'Just now'
			if (diffMins < 60) return `${diffMins}m ago`
			if (diffHours < 24) return `${diffHours}h ago`
			return date.toLocaleDateString()
		},
		formatNextRun() {
			if (!this.config.next_scheduled_run) return '-'
			const date = new Date(this.config.next_scheduled_run)
			const now = new Date()
			const diffMs = date - now
			if (diffMs < 0) return 'Overdue'
			const diffMins = Math.floor(diffMs / 60000)
			if (diffMins < 60) return `in ${diffMins}m`
			const diffHours = Math.floor(diffMs / 3600000)
			return `in ${diffHours}h`
		},
	},
	methods: {
		toggleEnabled() {
			this.$emit('toggle', !this.config.enabled)
		},
		formatNumber(num) {
			if (!num) return '0'
			return num.toLocaleString()
		},
	},
}
</script>

<style scoped>
.scraper-card {
	background: rgba(30, 30, 40, 0.6);
	border-radius: 12px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	padding: 1rem;
	transition: all 0.3s;
	opacity: 0.6;
}

.scraper-card.enabled {
	opacity: 1;
	border-color: rgba(255, 255, 255, 0.15);
}

.scraper-card.purple.enabled {
	border-color: rgba(168, 85, 247, 0.3);
	box-shadow: 0 0 20px rgba(168, 85, 247, 0.1);
}

.scraper-card.orange.enabled {
	border-color: rgba(249, 115, 22, 0.3);
	box-shadow: 0 0 20px rgba(249, 115, 22, 0.1);
}

.scraper-card.pink.enabled {
	border-color: rgba(236, 72, 153, 0.3);
	box-shadow: 0 0 20px rgba(236, 72, 153, 0.1);
}

.scraper-card.red.enabled {
	border-color: rgba(239, 68, 68, 0.3);
	box-shadow: 0 0 20px rgba(239, 68, 68, 0.1);
}

.scraper-card.blue.enabled {
	border-color: rgba(59, 130, 246, 0.3);
	box-shadow: 0 0 20px rgba(59, 130, 246, 0.1);
}

.scraper-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 0.75rem;
}

.scraper-info {
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.scraper-icon {
	font-size: 1rem;
	opacity: 0.8;
}

.purple .scraper-icon { color: #c084fc; }
.orange .scraper-icon { color: #fb923c; }
.pink .scraper-icon { color: #f472b6; }
.red .scraper-icon { color: #f87171; }
.blue .scraper-icon { color: #60a5fa; }

.scraper-name {
	font-weight: 600;
	font-size: 0.9375rem;
	color: rgba(255, 255, 255, 0.9);
}

.toggle-switch {
	position: relative;
	display: inline-block;
	width: 40px;
	height: 22px;
}

.toggle-switch input {
	opacity: 0;
	width: 0;
	height: 0;
}

.slider {
	position: absolute;
	cursor: pointer;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background-color: rgba(255, 255, 255, 0.1);
	transition: 0.3s;
	border-radius: 22px;
}

.slider:before {
	position: absolute;
	content: '';
	height: 16px;
	width: 16px;
	left: 3px;
	bottom: 3px;
	background-color: rgba(255, 255, 255, 0.5);
	transition: 0.3s;
	border-radius: 50%;
}

input:checked + .slider {
	background-color: rgba(34, 197, 94, 0.5);
}

input:checked + .slider:before {
	transform: translateX(18px);
	background-color: #4ade80;
}

.scraper-stats {
	margin-bottom: 0.75rem;
}

.stat-row {
	display: flex;
	justify-content: space-between;
	font-size: 0.8125rem;
	padding: 0.25rem 0;
}

.stat-label {
	color: rgba(255, 255, 255, 0.5);
}

.stat-value {
	color: rgba(255, 255, 255, 0.8);
	font-weight: 500;
}

.scraper-actions {
	display: flex;
	gap: 0.5rem;
}

.btn-check-now {
	flex: 1;
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 0.5rem;
	padding: 0.5rem;
	background: rgba(255, 255, 255, 0.1);
	border: none;
	border-radius: 6px;
	color: rgba(255, 255, 255, 0.8);
	font-size: 0.8125rem;
	cursor: pointer;
	transition: all 0.2s;
}

.btn-check-now:hover:not(:disabled) {
	background: rgba(255, 255, 255, 0.15);
}

.btn-check-now:disabled {
	opacity: 0.5;
	cursor: not-allowed;
}

.btn-settings {
	width: 36px;
	height: 36px;
	display: flex;
	align-items: center;
	justify-content: center;
	background: rgba(255, 255, 255, 0.05);
	border: none;
	border-radius: 6px;
	color: rgba(255, 255, 255, 0.5);
	cursor: pointer;
	transition: all 0.2s;
}

.btn-settings:hover {
	background: rgba(255, 255, 255, 0.1);
	color: rgba(255, 255, 255, 0.8);
}
</style>
