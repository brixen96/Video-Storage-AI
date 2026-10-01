<template>
	<div class="task-board">
		<div class="task-board-header">
			<h3>
				<font-awesome-icon :icon="['fas', 'clipboard-list']" class="me-2" />
				Shared Task Board
			</h3>
			<div class="task-stats" v-if="stats">
				<span class="stat-badge open">{{ stats.open || 0 }} open</span>
				<span class="stat-badge in-progress">{{ stats.in_progress || 0 }} in progress</span>
				<span class="stat-badge completed">{{ stats.completed || 0 }} done</span>
			</div>
		</div>

		<!-- Create Task Form -->
		<div class="create-task-form" v-if="showCreateForm">
			<input v-model="newTask.title" type="text" class="form-input" placeholder="Task title..." />
			<textarea v-model="newTask.description" class="form-textarea" placeholder="Description (optional)..." rows="2"></textarea>
			<div class="form-row">
				<select v-model="newTask.priority" class="form-select">
					<option :value="1">Low Priority</option>
					<option :value="5">Medium Priority</option>
					<option :value="8">High Priority</option>
					<option :value="10">Critical</option>
				</select>
				<input v-model="newTask.tagsInput" type="text" class="form-input" placeholder="Tags (comma separated)..." />
			</div>
			<div class="form-actions">
				<button class="btn btn-secondary" @click="showCreateForm = false">Cancel</button>
				<button class="btn btn-primary" @click="createTask" :disabled="!newTask.title.trim()">Create Task</button>
			</div>
		</div>

		<!-- Task Filters -->
		<div class="task-filters">
			<button v-for="filter in filters" :key="filter.value" class="filter-btn" :class="{ active: currentFilter === filter.value }" @click="currentFilter = filter.value">
				{{ filter.label }}
			</button>
			<button class="btn btn-primary btn-sm" @click="showCreateForm = true">
				<font-awesome-icon :icon="['fas', 'plus']" class="me-1" />
				New Task
			</button>
		</div>

		<!-- Task List -->
		<div class="task-list" v-if="tasks.length > 0">
			<div v-for="task in filteredTasks" :key="task.id" class="task-card" :class="[`priority-${task.priority}`, `status-${task.status}`]">
				<div class="task-header">
					<span class="task-id">#{{ task.id }}</span>
					<span class="task-title">{{ task.title }}</span>
					<span class="task-priority" :class="getPriorityClass(task.priority)">{{ task.priority }}</span>
				</div>
				<div class="task-meta">
					<span class="task-creator">
						<font-awesome-icon :icon="['fas', task.created_by === 'companion' ? 'robot' : 'user']" class="me-1" />
						{{ task.created_by }}
					</span>
					<span class="task-time">{{ formatTime(task.created_at) }}</span>
				</div>
				<div v-if="task.description" class="task-description">{{ task.description }}</div>
				<div v-if="task.tags && task.tags.length > 0" class="task-tags">
					<span v-for="tag in task.tags" :key="tag" class="tag-badge">{{ tag }}</span>
				</div>
				<div v-if="task.error_context" class="task-error">
					<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="me-1" />
					{{ truncate(task.error_context, 100) }}
				</div>
				<div class="task-actions">
					<button v-if="task.status === 'open'" class="btn btn-sm btn-warning" @click="updateTaskStatus(task.id, 'in_progress')">Start</button>
					<button v-if="task.status === 'in_progress'" class="btn btn-sm btn-success" @click="updateTaskStatus(task.id, 'completed')">Complete</button>
					<button v-if="task.status !== 'archived'" class="btn btn-sm btn-outline-secondary" @click="updateTaskStatus(task.id, 'archived')">Archive</button>
					<button class="btn btn-sm btn-outline-danger" @click="deleteTask(task.id)">
						<font-awesome-icon :icon="['fas', 'trash']" />
					</button>
				</div>
			</div>
		</div>

		<div v-else class="empty-state">
			<font-awesome-icon :icon="['fas', 'clipboard-check']" class="empty-icon" />
			<p>No tasks yet. Create one or ask the AI Companion to add tasks!</p>
		</div>
	</div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'
import { aiAPI } from '@/services/api'

export default {
	name: 'TaskBoard',
	setup() {
		const tasks = ref([])
		const stats = ref(null)
		const loading = ref(false)
		const showCreateForm = ref(false)
		const currentFilter = ref('all')

		const newTask = ref({
			title: '',
			description: '',
			priority: 5,
			tagsInput: '',
		})

		const filters = [
			{ label: 'All', value: 'all' },
			{ label: 'Open', value: 'open' },
			{ label: 'In Progress', value: 'in_progress' },
			{ label: 'Completed', value: 'completed' },
		]

		const filteredTasks = computed(() => {
			if (currentFilter.value === 'all') return tasks.value
			return tasks.value.filter(t => t.status === currentFilter.value)
		})

		const fetchTasks = async () => {
			loading.value = true
			try {
				const response = await aiAPI.getTasks()
				const data = response.data
				if (data.success) tasks.value = data.data || []
			} catch (e) {
				console.error('Failed to fetch tasks:', e)
			}
			loading.value = false
		}

		const fetchStats = async () => {
			try {
				const response = await aiAPI.getTaskStats()
				const data = response.data
				if (data.success) stats.value = data.data
			} catch (e) {
				console.error('Failed to fetch stats:', e)
			}
		}

		const createTask = async () => {
			if (!newTask.value.title.trim()) return

			const tags = newTask.value.tagsInput
				? newTask.value.tagsInput.split(',').map(t => t.trim()).filter(Boolean)
				: []

			try {
				const response = await aiAPI.createTask({
					title: newTask.value.title,
					description: newTask.value.description,
					priority: newTask.value.priority,
					tags,
					created_by: 'developer',
				})
				const data = response.data
				if (data.success) {
					tasks.value.unshift(data.data)
					newTask.value = { title: '', description: '', priority: 5, tagsInput: '' }
					showCreateForm.value = false
					fetchStats()
				}
			} catch (e) {
				console.error('Failed to create task:', e)
			}
		}

		const updateTaskStatus = async (taskId, status) => {
			try {
				const response = await aiAPI.updateTask(taskId, { status })
				const data = response.data
				if (data.success) {
					const idx = tasks.value.findIndex(t => t.id === taskId)
					if (idx !== -1) tasks.value[idx].status = status
					fetchStats()
				}
			} catch (e) {
				console.error('Failed to update task:', e)
			}
		}

		const deleteTask = async (taskId) => {
			try {
				const response = await aiAPI.deleteTask(taskId)
				const data = response.data
				if (data.success) {
					tasks.value = tasks.value.filter(t => t.id !== taskId)
					fetchStats()
				}
			} catch (e) {
				console.error('Failed to delete task:', e)
			}
		}

		const formatTime = (dateStr) => {
			const date = new Date(dateStr)
			return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
		}

		const getPriorityClass = (priority) => {
			if (priority >= 8) return 'critical'
			if (priority >= 5) return 'high'
			return 'low'
		}

		const truncate = (str, len) => str && str.length > len ? str.substring(0, len) + '...' : str

		onMounted(() => {
			fetchTasks()
			fetchStats()
		})

		return {
			tasks, stats, loading, showCreateForm, currentFilter, newTask,
			filters, filteredTasks, createTask, updateTaskStatus, deleteTask,
			formatTime, getPriorityClass, truncate,
		}
	},
}
</script>

<style scoped>
.task-board {
	padding: 1rem;
	max-height: 600px;
	overflow-y: auto;
}

.task-board-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 1rem;
}

.task-board-header h3 {
	margin: 0;
	font-size: 1.2rem;
}

.task-stats {
	display: flex;
	gap: 0.5rem;
}

.stat-badge {
	padding: 0.25rem 0.5rem;
	border-radius: 12px;
	font-size: 0.75rem;
	font-weight: 600;
}

.stat-badge.open { background: #3b82f620; color: #3b82f6; }
.stat-badge.in-progress { background: #f59e0b20; color: #f59e0b; }
.stat-badge.completed { background: #10b98120; color: #10b981; }

.create-task-form {
	background: var(--bg-secondary, #1a1a2e);
	border-radius: 8px;
	padding: 1rem;
	margin-bottom: 1rem;
	border: 1px solid var(--border-color, #333);
}

.form-input, .form-textarea, .form-select {
	width: 100%;
	padding: 0.5rem;
	margin-bottom: 0.5rem;
	background: var(--bg-tertiary, #16213e);
	border: 1px solid var(--border-color, #333);
	border-radius: 6px;
	color: var(--text-primary, #fff);
	font-size: 0.9rem;
}

.form-textarea {
	resize: vertical;
	min-height: 60px;
}

.form-row {
	display: flex;
	gap: 0.5rem;
}

.form-row .form-select,
.form-row .form-input {
	flex: 1;
}

.form-actions {
	display: flex;
	gap: 0.5rem;
	justify-content: flex-end;
	margin-top: 0.5rem;
}

.task-filters {
	display: flex;
	gap: 0.5rem;
	margin-bottom: 1rem;
	align-items: center;
	flex-wrap: wrap;
}

.filter-btn {
	padding: 0.35rem 0.75rem;
	border: 1px solid var(--border-color, #333);
	background: transparent;
	color: var(--text-secondary, #aaa);
	border-radius: 16px;
	cursor: pointer;
	font-size: 0.8rem;
	transition: all 0.2s;
}

.filter-btn.active {
	background: var(--accent-color, #3b82f6);
	color: white;
	border-color: var(--accent-color, #3b82f6);
}

.task-list {
	display: flex;
	flex-direction: column;
	gap: 0.75rem;
}

.task-card {
	background: var(--bg-secondary, #1a1a2e);
	border-radius: 8px;
	padding: 1rem;
	border: 1px solid var(--border-color, #333);
	transition: transform 0.2s, box-shadow 0.2s;
}

.task-card:hover {
	transform: translateY(-1px);
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.task-card.priority-10 { border-left: 3px solid #ef4444; }
.task-card.priority-8 { border-left: 3px solid #f59e0b; }
.task-card.priority-5 { border-left: 3px solid #3b82f6; }
.task-card.priority-1 { border-left: 3px solid #6b7280; }

.task-card.status-completed { opacity: 0.7; }

.task-header {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	margin-bottom: 0.5rem;
}

.task-id {
	color: var(--text-secondary, #888);
	font-size: 0.8rem;
	font-family: monospace;
}

.task-title {
	font-weight: 600;
	flex: 1;
}

.task-priority {
	padding: 0.1rem 0.4rem;
	border-radius: 4px;
	font-size: 0.7rem;
	font-weight: 700;
}

.task-priority.critical { background: #ef4444; color: white; }
.task-priority.high { background: #f59e0b; color: black; }
.task-priority.low { background: #6b7280; color: white; }

.task-meta {
	display: flex;
	gap: 1rem;
	font-size: 0.8rem;
	color: var(--text-secondary, #888);
	margin-bottom: 0.5rem;
}

.task-description {
	font-size: 0.85rem;
	color: var(--text-secondary, #ccc);
	margin-bottom: 0.5rem;
	line-height: 1.4;
}

.task-tags {
	display: flex;
	flex-wrap: wrap;
	gap: 0.25rem;
	margin-bottom: 0.5rem;
}

.tag-badge {
	padding: 0.15rem 0.5rem;
	background: var(--accent-color, #3b82f6)20;
	color: var(--accent-color, #3b82f6);
	border-radius: 12px;
	font-size: 0.7rem;
}

.task-error {
	padding: 0.5rem;
	background: #ef444420;
	color: #ef4444;
	border-radius: 6px;
	font-size: 0.8rem;
	margin-bottom: 0.5rem;
	font-family: monospace;
}

.task-actions {
	display: flex;
	gap: 0.5rem;
	margin-top: 0.5rem;
}

.btn {
	padding: 0.35rem 0.75rem;
	border: none;
	border-radius: 6px;
	cursor: pointer;
	font-size: 0.8rem;
	transition: opacity 0.2s;
}

.btn:hover { opacity: 0.8; }

.btn-primary { background: var(--accent-color, #3b82f6); color: white; }
.btn-secondary { background: #6b7280; color: white; }
.btn-success { background: #10b981; color: white; }
.btn-warning { background: #f59e0b; color: black; }
.btn-danger { background: #ef4444; color: white; }
.btn-outline-secondary { background: transparent; border: 1px solid #6b7280; color: #6b7280; }
.btn-outline-danger { background: transparent; border: 1px solid #ef4444; color: #ef4444; }

.btn-sm { padding: 0.25rem 0.5rem; font-size: 0.75rem; }

.empty-state {
	text-align: center;
	padding: 2rem;
	color: var(--text-secondary, #888);
}

.empty-icon {
	font-size: 3rem;
	margin-bottom: 1rem;
	opacity: 0.5;
}
</style>