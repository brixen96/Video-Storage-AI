<template>
	<div class="quick-actions-panel">
		<div class="panel-header">
			<h3>
				<font-awesome-icon :icon="['fas', 'bolt']" class="me-2" />
				Quick Actions
			</h3>
		</div>
		<div class="actions-grid">
			<button class="action-btn" @click="$router.push('/scrapers')">
				<font-awesome-icon :icon="['fas', 'globe']" />
				<span>Scrapers</span>
			</button>
			<button class="action-btn" @click="$router.push('/browser')">
				<font-awesome-icon :icon="['fas', 'folder-open']" />
				<span>Browse Files</span>
			</button>
			<button class="action-btn" @click="$router.push('/videos')">
				<font-awesome-icon :icon="['fas', 'film']" />
				<span>Videos</span>
			</button>
			<button class="action-btn" @click="$router.push('/activity')">
				<font-awesome-icon :icon="['fas', 'tasks']" />
				<span>Activity</span>
			</button>
			<button class="action-btn" @click="$router.push('/performers')">
				<font-awesome-icon :icon="['fas', 'users']" />
				<span>Performers</span>
			</button>
			<button class="action-btn" @click="$router.push('/settings')">
				<font-awesome-icon :icon="['fas', 'cog']" />
				<span>Settings</span>
			</button>
		</div>
	</div>
</template>

<script>
export default {
	name: 'QuickActionsPanel',
}
</script>

<style scoped>
.quick-actions-panel {
	background: rgba(30, 30, 40, 0.6);
	border-radius: 12px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	overflow: hidden;
}

.panel-header {
	padding: 1rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.panel-header h3 {
	margin: 0;
	font-size: 1rem;
	font-weight: 600;
	color: rgba(255, 255, 255, 0.9);
}

.actions-grid {
	display: grid;
	grid-template-columns: repeat(3, 1fr);
	gap: 0.5rem;
	padding: 1rem;
}

.action-btn {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	gap: 0.5rem;
	padding: 1rem 0.5rem;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.08);
	border-radius: 8px;
	color: rgba(255, 255, 255, 0.7);
	cursor: pointer;
	transition: all 0.2s;
}

.action-btn:hover {
	background: rgba(255, 255, 255, 0.1);
	border-color: rgba(255, 255, 255, 0.15);
	color: rgba(255, 255, 255, 0.9);
	transform: translateY(-2px);
}

.action-btn svg {
	font-size: 1.25rem;
	opacity: 0.8;
}

.action-btn span {
	font-size: 0.75rem;
	font-weight: 500;
}
</style>
