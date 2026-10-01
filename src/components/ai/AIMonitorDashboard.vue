<template>
	<div class="ai-monitor-dashboard">
		<div class="monitor-header">
			<h2>
				<font-awesome-icon :icon="['fas', 'chart-line']" class="me-2" />
				AI Companion Monitor
			</h2>
			<button class="btn-refresh" @click="refreshData" :disabled="loading">
				<font-awesome-icon :icon="['fas', 'sync']" :spin="loading" />
				Refresh
			</button>
		</div>

		<!-- Stats Cards -->
		<div class="stats-grid">
			<div class="stat-card">
				<div class="stat-icon blue">
					<font-awesome-icon :icon="['fas', 'exchange-alt']" />
				</div>
				<div class="stat-info">
					<div class="stat-value">{{ metrics.total_calls || 0 }}</div>
					<div class="stat-label">Total API Calls</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon green">
					<font-awesome-icon :icon="['fas', 'check-circle']" />
				</div>
				<div class="stat-info">
					<div class="stat-value">{{ metrics.successful_calls || 0 }}</div>
					<div class="stat-label">Successful</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon red">
					<font-awesome-icon :icon="['fas', 'times-circle']" />
				</div>
				<div class="stat-info">
					<div class="stat-value">{{ metrics.failed_calls || 0 }}</div>
					<div class="stat-label">Failed</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon yellow">
					<font-awesome-icon :icon="['fas', 'clock']" />
				</div>
				<div class="stat-info">
					<div class="stat-value">{{ formatLatency(metrics.avg_latency_ms) }}</div>
					<div class="stat-label">Avg Latency</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon purple">
					<font-awesome-icon :icon="['fas', 'coin']" />
				</div>
				<div class="stat-info">
					<div class="stat-value">${{ formatCost(metrics.total_cost_usd) }}</div>
					<div class="stat-label">Total Cost</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon cyan">
					<font-awesome-icon :icon="['fas', 'microchip']" />
				</div>
				<div class="stat-info">
					<div class="stat-value">{{ formatTokens(metrics.total_tokens_used) }}</div>
					<div class="stat-label">Total Tokens</div>
				</div>
			</div>
		</div>

		<!-- Token Breakdown -->
		<div class="token-breakdown">
			<h3>Token Usage</h3>
			<div class="token-bars">
				<div class="token-bar">
					<span class="token-label">Prompt</span>
					<div class="token-progress">
						<div class="token-fill prompt" :style="{ width: tokenPercentages.prompt + '%' }"></div>
					</div>
					<span class="token-count">{{ formatTokens(metrics.total_prompt_tokens) }}</span>
				</div>
				<div class="token-bar">
					<span class="token-label">Completion</span>
					<div class="token-progress">
						<div class="token-fill completion" :style="{ width: tokenPercentages.completion + '%' }"></div>
					</div>
					<span class="token-count">{{ formatTokens(metrics.total_completion_tokens) }}</span>
				</div>
				<div class="token-bar">
					<span class="token-label">Reasoning</span>
					<div class="token-progress">
						<div class="token-fill reasoning" :style="{ width: tokenPercentages.reasoning + '%' }"></div>
					</div>
					<span class="token-count">{{ formatTokens(metrics.total_reasoning_tokens) }}</span>
				</div>
			</div>
		</div>

		<!-- Model Breakdown & Tool Stats -->
		<div class="two-columns">
			<!-- Model Usage -->
			<div class="panel">
				<h3>Model Usage</h3>
				<div v-if="metrics.model_breakdown && metrics.model_breakdown.length > 0" class="model-list">
					<div v-for="model in metrics.model_breakdown" :key="model.model" class="model-item">
						<div class="model-name">{{ model.model }}</div>
						<div class="model-stats">
							<span>{{ model.call_count }} calls</span>
							<span>{{ formatTokens(model.total_tokens) }} tokens</span>
							<span>${{ formatCost(model.total_cost_usd) }}</span>
						</div>
					</div>
				</div>
				<div v-else class="empty-state">No model data yet</div>
			</div>

			<!-- Tool Usage -->
			<div class="panel">
				<h3>Tool Usage</h3>
				<div v-if="toolStats.length > 0" class="tool-list">
					<div v-for="tool in toolStats" :key="tool.tool_name" class="tool-item">
						<div class="tool-name">{{ tool.tool_name }}</div>
						<div class="tool-stats">
							<span class="tool-success">{{ tool.success_count }} ✓</span>
							<span class="tool-fail" v-if="tool.failure_count > 0">{{ tool.failure_count }} ✗</span>
							<span>{{ formatLatency(tool.avg_latency) }} avg</span>
						</div>
					</div>
				</div>
				<div v-else class="empty-state">No tool data yet</div>
			</div>
		</div>

		<!-- Cost by Day -->
		<div class="panel" v-if="metrics.cost_by_day && metrics.cost_by_day.length > 0">
			<h3>Cost by Day (Last 7 Days)</h3>
			<div class="cost-chart">
				<div v-for="day in metrics.cost_by_day" :key="day.date" class="cost-bar">
					<div class="cost-bar-fill" :style="{ height: getCostBarHeight(day.cost_usd) + '%' }"></div>
					<div class="cost-bar-label">{{ formatShortDate(day.date) }}</div>
					<div class="cost-bar-value">${{ formatCost(day.cost_usd) }}</div>
				</div>
			</div>
		</div>

		<!-- Recent Calls -->
		<div class="panel">
			<h3>Recent API Calls</h3>
			<div v-if="metrics.recent_calls && metrics.recent_calls.length > 0" class="calls-table">
				<table>
					<thead>
						<tr>
							<th>Time</th>
							<th>Model</th>
							<th>Status</th>
							<th>Tokens</th>
							<th>Cost</th>
							<th>Latency</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="call in metrics.recent_calls" :key="call.id" :class="{ 'call-failed': !call.success }">
							<td>{{ formatTime(call.created_at) }}</td>
							<td>{{ call.model }}</td>
							<td>
								<span :class="call.success ? 'status-success' : 'status-error'">
									{{ call.status_code || (call.success ? 'OK' : 'ERR') }}
								</span>
							</td>
							<td>{{ formatTokens(call.total_tokens) }}</td>
							<td>${{ formatCost(call.cost_usd) }}</td>
							<td>{{ formatLatency(call.latency_ms) }}</td>
						</tr>
					</tbody>
				</table>
			</div>
			<div v-else class="empty-state">No API calls recorded yet</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { API_ORIGIN } from '@/services/api'

const loading = ref(false)
const metrics = ref({})
const toolStats = ref([])

const tokenPercentages = computed(() => {
	const total = metrics.value.total_tokens_used || 1
	return {
		prompt: ((metrics.value.total_prompt_tokens || 0) / total) * 100,
		completion: ((metrics.value.total_completion_tokens || 0) / total) * 100,
		reasoning: ((metrics.value.total_reasoning_tokens || 0) / total) * 100,
	}
})

const maxCost = computed(() => {
	if (!metrics.value.cost_by_day || metrics.value.cost_by_day.length === 0) return 1
	return Math.max(...metrics.value.cost_by_day.map(d => d.cost_usd), 0.01)
})

function formatTokens(tokens) {
	if (!tokens) return '0'
	if (tokens >= 1000000) return (tokens / 1000000).toFixed(1) + 'M'
	if (tokens >= 1000) return (tokens / 1000).toFixed(1) + 'K'
	return tokens.toString()
}

function formatCost(cost) {
	if (!cost) return '0.00'
	return cost.toFixed(4)
}

function formatLatency(ms) {
	if (!ms) return '0ms'
	if (ms >= 1000) return (ms / 1000).toFixed(1) + 's'
	return Math.round(ms) + 'ms'
}

function formatTime(dateStr) {
	if (!dateStr) return '-'
	const d = new Date(dateStr)
	return d.toLocaleTimeString()
}

function formatShortDate(dateStr) {
	if (!dateStr) return '-'
	const d = new Date(dateStr)
	return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

function getCostBarHeight(cost) {
	return (cost / maxCost.value) * 100
}

async function refreshData() {
	loading.value = true
	try {
		const [dashboardRes, toolsRes] = await Promise.all([
			fetch(`${API_ORIGIN}/api/v1/ai/monitor/dashboard`),
			fetch(`${API_ORIGIN}/api/v1/ai/monitor/tools`),
		])
		const dashboard = await dashboardRes.json()
		const tools = await toolsRes.json()

		if (dashboard.success) metrics.value = dashboard.data
		if (tools.success) toolStats.value = tools.data || []
	} catch (err) {
		console.error('Failed to load monitor data:', err)
	} finally {
		loading.value = false
	}
}

onMounted(refreshData)
</script>

<style scoped>
.ai-monitor-dashboard {
	padding: 1.5rem;
	max-width: 1400px;
	margin: 0 auto;
}

.monitor-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 1.5rem;
}

.monitor-header h2 {
	margin: 0;
	color: var(--text-primary);
}

.btn-refresh {
	padding: 0.5rem 1rem;
	background: var(--accent);
	color: white;
	border: none;
	border-radius: 8px;
	cursor: pointer;
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.btn-refresh:hover {
	opacity: 0.9;
}

.btn-refresh:disabled {
	opacity: 0.5;
	cursor: not-allowed;
}

.stats-grid {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
	gap: 1rem;
	margin-bottom: 1.5rem;
}

.stat-card {
	display: flex;
	align-items: center;
	gap: 1rem;
	padding: 1rem;
	background: var(--card-bg);
	border-radius: 12px;
	border: 1px solid var(--border);
}

.stat-icon {
	width: 48px;
	height: 48px;
	border-radius: 12px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.25rem;
}

.stat-icon.blue { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.stat-icon.green { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
.stat-icon.red { background: rgba(239, 68, 68, 0.15); color: #ef4444; }
.stat-icon.yellow { background: rgba(234, 179, 8, 0.15); color: #eab308; }
.stat-icon.purple { background: rgba(168, 85, 247, 0.15); color: #a855f7; }
.stat-icon.cyan { background: rgba(6, 182, 212, 0.15); color: #06b6d4; }

.stat-value {
	font-size: 1.5rem;
	font-weight: 700;
	color: var(--text-primary);
}

.stat-label {
	font-size: 0.85rem;
	color: var(--text-secondary);
}

.token-breakdown, .panel {
	background: var(--card-bg);
	border-radius: 12px;
	border: 1px solid var(--border);
	padding: 1.25rem;
	margin-bottom: 1.5rem;
}

.token-breakdown h3, .panel h3 {
	margin: 0 0 1rem 0;
	font-size: 1rem;
	color: var(--text-primary);
}

.token-bars {
	display: flex;
	flex-direction: column;
	gap: 0.75rem;
}

.token-bar {
	display: flex;
	align-items: center;
	gap: 1rem;
}

.token-label {
	width: 80px;
	font-size: 0.85rem;
	color: var(--text-secondary);
}

.token-progress {
	flex: 1;
	height: 8px;
	background: rgba(255, 255, 255, 0.1);
	border-radius: 4px;
	overflow: hidden;
}

.token-fill {
	height: 100%;
	border-radius: 4px;
	transition: width 0.3s ease;
}

.token-fill.prompt { background: #3b82f6; }
.token-fill.completion { background: #22c55e; }
.token-fill.reasoning { background: #a855f7; }

.token-count {
	width: 60px;
	text-align: right;
	font-size: 0.85rem;
	color: var(--text-secondary);
}

.two-columns {
	display: grid;
	grid-template-columns: 1fr 1fr;
	gap: 1.5rem;
	margin-bottom: 1.5rem;
}

@media (max-width: 768px) {
	.two-columns {
		grid-template-columns: 1fr;
	}
}

.model-list, .tool-list {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.model-item, .tool-item {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 0.75rem;
	background: rgba(255, 255, 255, 0.03);
	border-radius: 8px;
}

.model-name, .tool-name {
	font-weight: 500;
	color: var(--text-primary);
}

.model-stats, .tool-stats {
	display: flex;
	gap: 1rem;
	font-size: 0.85rem;
	color: var(--text-secondary);
}

.tool-success { color: #22c55e; }
.tool-fail { color: #ef4444; }

.cost-chart {
	display: flex;
	align-items: flex-end;
	gap: 0.5rem;
	height: 150px;
	padding-top: 1rem;
}

.cost-bar {
	flex: 1;
	display: flex;
	flex-direction: column;
	align-items: center;
	height: 100%;
	justify-content: flex-end;
}

.cost-bar-fill {
	width: 100%;
	background: linear-gradient(180deg, #a855f7, #6366f1);
	border-radius: 4px 4px 0 0;
	min-height: 2px;
	transition: height 0.3s ease;
}

.cost-bar-label {
	font-size: 0.75rem;
	color: var(--text-secondary);
	margin-top: 0.5rem;
}

.cost-bar-value {
	font-size: 0.7rem;
	color: var(--text-secondary);
}

.calls-table {
	overflow-x: auto;
}

.calls-table table {
	width: 100%;
	border-collapse: collapse;
}

.calls-table th, .calls-table td {
	padding: 0.75rem;
	text-align: left;
	border-bottom: 1px solid var(--border);
}

.calls-table th {
	font-size: 0.85rem;
	color: var(--text-secondary);
	font-weight: 500;
}

.calls-table td {
	font-size: 0.9rem;
	color: var(--text-primary);
}

.call-failed {
	background: rgba(239, 68, 68, 0.05);
}

.status-success {
	color: #22c55e;
	font-weight: 500;
}

.status-error {
	color: #ef4444;
	font-weight: 500;
}

.empty-state {
	text-align: center;
	padding: 2rem;
	color: var(--text-secondary);
}
</style>
