<template>
	<div class="running-tasks-widget">
		<div class="widget-header">
			<h3>
				<font-awesome-icon v-if="tasks.length > 0" :icon="['fas', 'spinner']" spin class="me-2" />
				<font-awesome-icon v-else :icon="['fas', 'check-circle']" class="me-2 text-success" />
				Running Tasks
				<span class="task-count" v-if="tasks.length > 0">({{ tasks.length }})</span>
			</h3>
		</div>
		<div class="task-list" v-if="tasks.length > 0">
			<div v-for="task in tasks" :key="task.id" class="task-item">
				<div class="task-info">
					<span class="task-type">{{ formatTaskType(task.task_type) }}</span>
					<span class="task-message">{{ task.message }}</span>
				</div>
				<div class="task-progress-container">
					<div class="task-progress-bar">
						<div class="task-progress-fill" :style="{ width: (task.progress || 0) + '%' }"></div>
					</div>
					<span class="task-progress-text">{{ task.progress || 0 }}%</span>
				</div>
			</div>
		</div>
		<div class="no-tasks" v-else>
			<p>All systems idle</p>
		</div>
	</div>
</template>

<script>
export default {
	name: 'RunningTasksWidget',
	props: {
		tasks: {
			type: Array,
			default: () => [],
		},
	},
	methods: {
		formatTaskType(type) {
			if (!type) return 'Task'
			return type
				.split('_')
				.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
				.join(' ')
		},
	},
}
</script>

<style scoped>
.running-tasks-widget {
	background: rgba(30, 30, 40, 0.6);
	border-radius: 12px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	overflow: hidden;
}

.widget-header {
	padding: 1rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.widget-header h3 {
	margin: 0;
	font-size: 1rem;
	font-weight: 600;
	color: rgba(255, 255, 255, 0.9);
	display: flex;
	align-items: center;
}

.task-count {
	font-weight: 400;
	color: rgba(255, 255, 255, 0.5);
	margin-left: 0.25rem;
}

.task-list {
	max-height: 200px;
	overflow-y: auto;
}

.task-item {
	padding: 0.75rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.task-item:last-child {
	border-bottom: none;
}

.task-info {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	margin-bottom: 0.5rem;
}

.task-type {
	font-size: 0.75rem;
	font-weight: 600;
	text-transform: uppercase;
	letter-spacing: 0.5px;
	color: #60a5fa;
	background: rgba(59, 130, 246, 0.15);
	padding: 0.125rem 0.5rem;
	border-radius: 4px;
}

.task-message {
	font-size: 0.8125rem;
	color: rgba(255, 255, 255, 0.7);
	flex: 1;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.task-progress-container {
	display: flex;
	align-items: center;
	gap: 0.75rem;
}

.task-progress-bar {
	flex: 1;
	height: 6px;
	background: rgba(255, 255, 255, 0.1);
	border-radius: 3px;
	overflow: hidden;
}

.task-progress-fill {
	height: 100%;
	background: linear-gradient(90deg, #3b82f6, #60a5fa);
	border-radius: 3px;
	transition: width 0.3s ease;
}

.task-progress-text {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.5);
	min-width: 35px;
	text-align: right;
}

.no-tasks {
	padding: 1.5rem;
	text-align: center;
}

.no-tasks p {
	margin: 0;
	color: rgba(255, 255, 255, 0.4);
	font-size: 0.875rem;
}

.text-success {
	color: #4ade80 !important;
}
</style>
