<template>
	<div class="auto-update-panel" :class="{ enabled: config?.enabled }">
		<div class="panel-content" v-if="config">
			<!-- Left: Toggle and Status -->
			<div class="panel-left">
				<label class="toggle-switch" :title="config.enabled ? 'Disable auto-update' : 'Enable auto-update'">
					<input type="checkbox" :checked="config.enabled" @change="toggleEnabled" />
					<span class="slider"></span>
				</label>
				<div class="status-info">
					<span class="status-label">Auto-Update</span>
					<span class="status-text" :class="config.enabled ? 'text-success' : 'text-muted'">
						{{ config.enabled ? 'Enabled' : 'Disabled' }}
					</span>
				</div>
			</div>

			<!-- Center: Stats -->
			<div class="panel-center" v-if="config.enabled">
				<div class="stat-item">
					<span class="stat-value">{{ formatLastChecked }}</span>
					<span class="stat-label">Last Check</span>
				</div>
				<div class="stat-item">
					<span class="stat-value" :class="nextCheckUrgent ? 'text-warning' : ''">{{ formatNextCheck }}</span>
					<span class="stat-label">Next Check</span>
				</div>
				<div class="stat-item">
					<span class="stat-value">{{ config.interval_minutes }}m</span>
					<span class="stat-label">Interval</span>
				</div>
				<div class="stat-item" v-if="config.total_found_count > 0">
					<span class="stat-value">{{ config.total_found_count.toLocaleString() }}</span>
					<span class="stat-label">Total Found</span>
				</div>
			</div>

			<!-- Right: Actions -->
			<div class="panel-right">
				<button class="btn-icon" @click="showSettingsModal = true" title="Settings">
					<font-awesome-icon :icon="['fas', 'cog']" />
				</button>
				<button class="btn-check" @click="checkNow" :disabled="isChecking || !config.enabled" title="Check now">
					<font-awesome-icon :icon="['fas', isChecking ? 'spinner' : 'sync']" :spin="isChecking" />
					<span>Check</span>
				</button>
			</div>
		</div>

		<div class="panel-loading" v-else>
			<font-awesome-icon :icon="['fas', 'spinner']" spin />
		</div>

		<!-- Settings Modal -->
		<div v-if="showSettingsModal" class="modal-overlay" @click.self="showSettingsModal = false">
			<div class="settings-modal">
				<div class="modal-header">
					<h4>
						<font-awesome-icon :icon="['fas', 'cog']" class="me-2" />
						Auto-Update Settings
					</h4>
					<button class="btn-close" @click="showSettingsModal = false">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
				<div class="modal-body">
					<!-- Interval -->
					<div class="setting-group">
						<label>Check Interval</label>
						<select v-model.number="intervalMinutes" class="form-select">
							<option :value="15">Every 15 minutes</option>
							<option :value="30">Every 30 minutes</option>
							<option :value="60">Every 1 hour</option>
							<option :value="120">Every 2 hours</option>
							<option :value="180">Every 3 hours</option>
							<option :value="360">Every 6 hours</option>
							<option :value="720">Every 12 hours</option>
							<option :value="1440">Every 24 hours</option>
						</select>
					</div>

					<!-- Max Pages -->
					<div class="setting-group">
						<label>Max Pages to Check</label>
						<select v-model.number="maxPages" class="form-select">
							<option :value="1">1 page</option>
							<option :value="2">2 pages</option>
							<option :value="3">3 pages</option>
							<option :value="5">5 pages</option>
							<option :value="10">10 pages</option>
						</select>
					</div>

					<!-- Endpoint Configuration -->
					<div v-if="hasEndpointConfig" class="setting-group">
						<label>{{ endpointLabel }}</label>
						<div class="endpoint-input-group">
							<input
								type="text"
								v-model="newEndpoint"
								:placeholder="endpointPlaceholder"
								class="form-control"
								@keyup.enter="addEndpoint"
							/>
							<button class="btn btn-primary btn-sm" @click="addEndpoint" :disabled="!newEndpoint.trim()">
								<font-awesome-icon :icon="['fas', 'plus']" />
							</button>
						</div>
						<div class="endpoint-tags" v-if="endpoints.length > 0">
							<span v-for="(endpoint, idx) in endpoints" :key="idx" class="endpoint-tag">
								{{ endpoint }}
								<button class="tag-remove" @click="removeEndpoint(idx)">
									<font-awesome-icon :icon="['fas', 'times']" />
								</button>
							</span>
						</div>
						<p v-else class="no-endpoints">No {{ endpointLabel.toLowerCase() }} configured yet</p>
					</div>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="showSettingsModal = false">Cancel</button>
					<button class="btn btn-primary" @click="saveSettings">Save Settings</button>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import { autoUpdateAPI } from '@/services/api'

export default {
	name: 'AutoUpdatePanel',
	props: {
		scraperType: {
			type: String,
			required: true,
		},
		endpointType: {
			type: String,
			default: null,
		},
	},
	emits: ['updated', 'check-started'],
	data() {
		return {
			config: null,
			isChecking: false,
			showSettingsModal: false,
			newEndpoint: '',
			endpoints: [],
			intervalMinutes: 60,
			maxPages: 3,
			now: Date.now(),
			tickerInterval: null,
		}
	},
	computed: {
		hasEndpointConfig() {
			return this.endpointType !== null
		},
		endpointLabel() {
			if (this.endpointType === 'keywords') return 'Keywords'
			if (this.endpointType === 'categories') return 'Categories'
			if (this.endpointType === 'urls') return 'Forum URLs'
			return 'Endpoints'
		},
		endpointPlaceholder() {
			if (this.endpointType === 'keywords') return 'Add keyword or search term...'
			if (this.endpointType === 'categories') return 'Add category URL...'
			if (this.endpointType === 'urls') return 'Add forum/listing URL to monitor...'
			return 'Add endpoint...'
		},
		formatLastChecked() {
			if (!this.config?.last_checked_at) return 'Never'
			const date = new Date(this.config.last_checked_at)
			const diffMs = this.now - date.getTime()
			const diffMins = Math.floor(diffMs / 60000)
			const diffHours = Math.floor(diffMs / 3600000)
			if (diffMins < 1) return 'Just now'
			if (diffMins < 60) return `${diffMins}m ago`
			if (diffHours < 24) return `${diffHours}h ago`
			return date.toLocaleDateString()
		},
		formatNextCheck() {
			if (!this.config?.last_checked_at || !this.config?.interval_minutes) return '—'
			const last = new Date(this.config.last_checked_at).getTime()
			const nextMs = last + this.config.interval_minutes * 60000
			const diffMs = nextMs - this.now
			if (diffMs <= 0) return 'Soon'
			const diffMins = Math.ceil(diffMs / 60000)
			if (diffMins < 60) return `${diffMins}m`
			const h = Math.floor(diffMins / 60)
			const m = diffMins % 60
			return m > 0 ? `${h}h ${m}m` : `${h}h`
		},
		nextCheckUrgent() {
			if (!this.config?.last_checked_at || !this.config?.interval_minutes) return false
			const last = new Date(this.config.last_checked_at).getTime()
			const nextMs = last + this.config.interval_minutes * 60000
			return nextMs - this.now <= 5 * 60000 // within 5 minutes
		},
	},
	async mounted() {
		await this.loadConfig()
		this.tickerInterval = setInterval(() => { this.now = Date.now() }, 30000)
	},
	beforeUnmount() {
		if (this.tickerInterval) clearInterval(this.tickerInterval)
	},
	methods: {
		async loadConfig() {
			try {
				const response = await autoUpdateAPI.getConfig(this.scraperType)
				if (response?.data) {
					this.config = response.data
					this.intervalMinutes = this.config.interval_minutes || 60
					this.maxPages = this.config.max_pages_to_check || 3

					if (this.config.config) {
						if (this.endpointType === 'keywords') {
							this.endpoints = this.config.config.keywords || []
						} else if (this.endpointType === 'categories') {
							this.endpoints = this.config.config.categories || []
						} else if (this.endpointType === 'urls') {
							this.endpoints = this.config.config.urls || []
						}
					}
				}
			} catch (error) {
				console.error('Failed to load auto-update config:', error)
			}
		},
		async toggleEnabled() {
			try {
				const newState = !this.config.enabled
				await autoUpdateAPI.toggle(this.scraperType, newState)
				this.config.enabled = newState
				this.$emit('updated', this.config)
			} catch (error) {
				console.error('Failed to toggle auto-update:', error)
			}
		},
		async saveSettings() {
			try {
				const configJSON = JSON.stringify({
					keywords: this.endpointType === 'keywords' ? this.endpoints : undefined,
					categories: this.endpointType === 'categories' ? this.endpoints : undefined,
					urls: this.endpointType === 'urls' ? this.endpoints : undefined,
				})

				await autoUpdateAPI.updateConfig(this.scraperType, {
					interval_minutes: this.intervalMinutes,
					max_pages_to_check: this.maxPages,
					config_json: configJSON,
				})

				this.config.interval_minutes = this.intervalMinutes
				this.config.max_pages_to_check = this.maxPages
				this.$emit('updated', this.config)
				this.showSettingsModal = false
			} catch (error) {
				console.error('Failed to save settings:', error)
			}
		},
		async checkNow() {
			this.isChecking = true
			this.$emit('check-started')
			try {
				await autoUpdateAPI.checkNow(this.scraperType)
				setTimeout(() => {
					this.loadConfig()
					this.isChecking = false
				}, 3000)
			} catch (error) {
				console.error('Failed to check now:', error)
				this.isChecking = false
			}
		},
		addEndpoint() {
			const endpoint = this.newEndpoint.trim()
			if (endpoint && !this.endpoints.includes(endpoint)) {
				this.endpoints.push(endpoint)
				this.newEndpoint = ''
			}
		},
		removeEndpoint(index) {
			this.endpoints.splice(index, 1)
		},
	},
}
</script>

<style scoped>
.auto-update-panel {
	background: rgba(30, 30, 50, 0.6);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 10px;
	margin-bottom: 1rem;
}

.auto-update-panel.enabled {
	border-color: rgba(34, 197, 94, 0.3);
}

.panel-content {
	display: flex;
	align-items: center;
	padding: 0.75rem 1rem;
	gap: 1.5rem;
}

.panel-left {
	display: flex;
	align-items: center;
	gap: 0.75rem;
}

.toggle-switch {
	position: relative;
	display: inline-block;
	width: 40px;
	height: 22px;
	flex-shrink: 0;
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
	background-color: rgba(100, 100, 120, 0.5);
	border: 1px solid rgba(255, 255, 255, 0.2);
	transition: 0.3s;
	border-radius: 22px;
}

.slider:before {
	position: absolute;
	content: '';
	height: 16px;
	width: 16px;
	left: 2px;
	bottom: 2px;
	background-color: rgba(255, 255, 255, 0.7);
	transition: 0.3s;
	border-radius: 50%;
	box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}

input:checked + .slider {
	background-color: rgba(34, 197, 94, 0.6);
	border-color: rgba(34, 197, 94, 0.5);
}

input:checked + .slider:before {
	transform: translateX(18px);
	background-color: #4ade80;
	box-shadow: 0 0 8px rgba(74, 222, 128, 0.5);
}

.status-info {
	display: flex;
	flex-direction: column;
}

.status-label {
	font-size: 0.8125rem;
	font-weight: 600;
	color: rgba(255, 255, 255, 0.9);
}

.status-text {
	font-size: 0.75rem;
}

.text-success {
	color: #4ade80 !important;
	font-weight: 500;
}

.text-muted {
	color: rgba(180, 180, 200, 0.7) !important;
}

.panel-center {
	display: flex;
	gap: 1.5rem;
	flex: 1;
}

.stat-item {
	display: flex;
	flex-direction: column;
	align-items: center;
}

.stat-item .stat-value {
	font-size: 0.875rem;
	font-weight: 600;
	color: rgba(255, 255, 255, 0.9);
}

.stat-item .stat-label {
	font-size: 0.6875rem;
	color: rgba(255, 255, 255, 0.5);
	text-transform: uppercase;
}

.panel-right {
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.btn-icon {
	width: 32px;
	height: 32px;
	display: flex;
	align-items: center;
	justify-content: center;
	background: rgba(255, 255, 255, 0.08);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 6px;
	color: rgba(255, 255, 255, 0.7);
	cursor: pointer;
	transition: all 0.2s;
}

.btn-icon:hover {
	background: rgba(255, 255, 255, 0.12);
	color: rgba(255, 255, 255, 0.9);
}

/* Prevent icon rotation */
.btn-icon svg {
	transition: none !important;
	transform: none !important;
}

.btn-check {
	display: flex;
	align-items: center;
	gap: 0.375rem;
	padding: 0.375rem 0.75rem;
	background: linear-gradient(135deg, #6366f1, #8b5cf6);
	border: none;
	border-radius: 6px;
	color: #fff;
	font-size: 0.8125rem;
	font-weight: 500;
	cursor: pointer;
	transition: all 0.2s;
}

.btn-check:hover:not(:disabled) {
	transform: translateY(-1px);
	box-shadow: 0 4px 12px rgba(99, 102, 241, 0.4);
}

.btn-check:disabled {
	opacity: 0.5;
	cursor: not-allowed;
}

.panel-loading {
	padding: 1rem;
	text-align: center;
	color: rgba(255, 255, 255, 0.5);
}

/* Modal */
.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.7);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 1000;
	backdrop-filter: blur(4px);
}

.settings-modal {
	background: linear-gradient(135deg, rgba(30, 30, 50, 0.98), rgba(20, 20, 35, 0.98));
	border-radius: 12px;
	border: 1px solid rgba(255, 255, 255, 0.15);
	width: 100%;
	max-width: 450px;
	max-height: 80vh;
	overflow: hidden;
	display: flex;
	flex-direction: column;
}

.modal-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.modal-header h4 {
	margin: 0;
	font-size: 1rem;
	font-weight: 600;
	color: rgba(255, 255, 255, 0.95);
}

.modal-header .btn-close {
	background: none;
	border: none;
	color: rgba(255, 255, 255, 0.5);
	cursor: pointer;
	padding: 0.25rem;
}

.modal-header .btn-close:hover {
	color: rgba(255, 255, 255, 0.9);
}

.modal-body {
	padding: 1.25rem;
	overflow-y: auto;
}

.setting-group {
	margin-bottom: 1.25rem;
}

.setting-group:last-child {
	margin-bottom: 0;
}

.setting-group label {
	display: block;
	font-size: 0.875rem;
	font-weight: 500;
	color: rgba(255, 255, 255, 0.8);
	margin-bottom: 0.5rem;
}

.setting-group .form-select,
.setting-group .form-control {
	width: 100%;
	padding: 0.5rem 0.75rem;
	background-color: rgba(25, 25, 40, 0.95);
	border: 1px solid rgba(255, 255, 255, 0.2);
	border-radius: 6px;
	color: #fff;
	font-size: 0.875rem;
	-webkit-appearance: none;
	-moz-appearance: none;
	appearance: none;
}

.setting-group .form-select {
	background-image: url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 20 20'%3e%3cpath stroke='%23a0a0a0' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.5' d='M6 8l4 4 4-4'/%3e%3c/svg%3e");
	background-position: right 0.5rem center;
	background-repeat: no-repeat;
	background-size: 1.5em 1.5em;
	padding-right: 2.5rem;
}

.setting-group .form-select option {
	background-color: #1a1a2e;
	color: #fff;
	padding: 0.5rem;
}

.setting-group .form-select:focus,
.setting-group .form-control:focus {
	outline: none;
	border-color: rgba(99, 102, 241, 0.5);
	box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.15);
	background-color: rgba(30, 30, 50, 0.98);
}

.setting-group .form-control::placeholder {
	color: rgba(255, 255, 255, 0.4);
}

.endpoint-input-group {
	display: flex;
	gap: 0.5rem;
}

.endpoint-input-group .btn {
	background: linear-gradient(135deg, #6366f1, #8b5cf6);
	border: none;
	color: #fff;
	padding: 0.5rem 0.75rem;
	border-radius: 6px;
	cursor: pointer;
	transition: all 0.2s;
}

.endpoint-input-group .btn:hover:not(:disabled) {
	box-shadow: 0 4px 12px rgba(99, 102, 241, 0.4);
}

.endpoint-input-group .btn:disabled {
	opacity: 0.5;
	cursor: not-allowed;
}

.endpoint-tags {
	display: flex;
	flex-wrap: wrap;
	gap: 0.5rem;
	margin-top: 0.75rem;
}

.endpoint-tag {
	display: inline-flex;
	align-items: center;
	gap: 0.375rem;
	padding: 0.25rem 0.5rem 0.25rem 0.625rem;
	background: rgba(99, 102, 241, 0.2);
	border: 1px solid rgba(99, 102, 241, 0.3);
	border-radius: 16px;
	font-size: 0.8125rem;
	color: rgba(255, 255, 255, 0.9);
}

.tag-remove {
	background: none;
	border: none;
	color: rgba(255, 255, 255, 0.5);
	cursor: pointer;
	padding: 0;
	font-size: 0.6875rem;
	display: flex;
}

.tag-remove:hover {
	color: #f87171;
}

.no-endpoints {
	margin: 0.5rem 0 0 0;
	font-size: 0.8125rem;
	color: rgba(255, 255, 255, 0.4);
	font-style: italic;
}

.modal-footer {
	display: flex;
	justify-content: flex-end;
	gap: 0.75rem;
	padding: 1rem 1.25rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.modal-footer .btn {
	padding: 0.5rem 1rem;
	border-radius: 6px;
	font-size: 0.875rem;
	font-weight: 500;
	cursor: pointer;
}

.modal-footer .btn-secondary {
	background: rgba(255, 255, 255, 0.1);
	border: 1px solid rgba(255, 255, 255, 0.15);
	color: rgba(255, 255, 255, 0.8);
}

.modal-footer .btn-secondary:hover {
	background: rgba(255, 255, 255, 0.15);
}

.modal-footer .btn-primary {
	background: linear-gradient(135deg, #6366f1, #8b5cf6);
	border: none;
	color: #fff;
}

.modal-footer .btn-primary:hover {
	box-shadow: 0 4px 12px rgba(99, 102, 241, 0.4);
}

.me-2 {
	margin-right: 0.5rem;
}
</style>
