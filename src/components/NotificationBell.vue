<template>
	<div class="notification-bell">
		<button class="bell-button" @click="togglePanel" :class="{ 'has-unread': unreadCount > 0 }">
			<font-awesome-icon :icon="['fas', 'bell']" />
			<span v-if="unreadCount > 0" class="badge">{{ unreadCount > 99 ? '99+' : unreadCount }}</span>
		</button>

		<transition name="slide-fade">
			<div v-if="showPanel" class="notification-panel">
				<div class="panel-header">
					<h3>Notifications</h3>
					<div class="header-actions">
						<button v-if="unreadCount > 0" class="btn-link" @click="markAllAsRead">
							Mark all read
						</button>
						<button class="close-btn" @click="showPanel = false">
							<font-awesome-icon :icon="['fas', 'times']" />
						</button>
					</div>
				</div>

				<div class="panel-filters">
					<button
						v-for="filter in filters"
						:key="filter.value"
						class="filter-btn"
						:class="{ active: activeFilter === filter.value }"
						@click="activeFilter = filter.value"
					>
						{{ filter.label }}
					</button>
				</div>

				<div v-if="loading" class="panel-loading">
					<font-awesome-icon :icon="['fas', 'spinner']" spin />
					<span>Loading notifications...</span>
				</div>

				<div v-else-if="filteredNotifications.length === 0" class="panel-empty">
					<font-awesome-icon :icon="['fas', 'inbox']" size="3x" />
					<p>No notifications</p>
				</div>

				<div v-else class="notifications-list">
					<div
						v-for="notification in filteredNotifications"
						:key="notification.id"
						class="notification-item"
						:class="{
							'unread': !notification.is_read,
							[`priority-${notification.priority}`]: true
						}"
						@click="handleNotificationClick(notification)"
					>
						<div class="notification-icon" :class="`icon-${notification.category}`">
							<font-awesome-icon :icon="getNotificationIcon(notification.category)" />
						</div>
						<div class="notification-content">
							<div class="notification-title">{{ notification.title }}</div>
							<div class="notification-message">{{ notification.message }}</div>
							<div class="notification-time">{{ formatTime(notification.created_at) }}</div>
						</div>
						<button class="notification-action" @click.stop="archiveNotification(notification.id)">
							<font-awesome-icon :icon="['fas', 'times']" />
						</button>
					</div>
				</div>
			</div>
		</transition>
	</div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, getCurrentInstance } from 'vue'
import { useRouter } from 'vue-router'
import websocketService from '@/services/websocket'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast
const router = useRouter()

const showPanel = ref(false)
const loading = ref(false)
const notifications = ref([])
const unreadCount = ref(0)
const activeFilter = ref('all')

const filters = [
	{ label: 'All', value: 'all' },
	{ label: 'Unread', value: 'unread' },
	{ label: 'System', value: 'system' },
	{ label: 'Scraper', value: 'scraper' },
	{ label: 'Downloads', value: 'downloads' },
]

const filteredNotifications = computed(() => {
	let filtered = notifications.value

	if (activeFilter.value === 'unread') {
		filtered = filtered.filter((n) => !n.is_read)
	} else if (activeFilter.value !== 'all') {
		filtered = filtered.filter((n) => n.category === activeFilter.value)
	}

	return filtered
})

let refreshInterval = null
let unsubscribeNotification = null

const loadNotifications = async () => {
	loading.value = true
	try {
		const response = await fetch('http://localhost:8080/api/v1/notifications?limit=50')
		const data = await response.json()
		if (data.success) {
			notifications.value = data.data.notifications || []
		}
	} catch (error) {
		console.error('Failed to load notifications:', error)
	} finally {
		loading.value = false
	}
}

const loadStats = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/notifications/stats')
		const data = await response.json()
		if (data.success) {
			unreadCount.value = data.data.unread || 0
		}
	} catch (error) {
		console.error('Failed to load notification stats:', error)
	}
}

const togglePanel = async () => {
	showPanel.value = !showPanel.value
	if (showPanel.value) {
		await loadNotifications()
	}
}

const markAllAsRead = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/notifications/read-all', {
			method: 'POST',
		})
		const data = await response.json()
		if (data.success) {
			notifications.value.forEach((n) => (n.is_read = true))
			unreadCount.value = 0
			toast.success('All notifications marked as read')
		}
	} catch (error) {
		console.error('Failed to mark all as read:', error)
		toast.error('Failed to mark all as read')
	}
}

const handleNotificationClick = async (notification) => {
	// Mark as read
	if (!notification.is_read) {
		try {
			await fetch(`http://localhost:8080/api/v1/notifications/${notification.id}/read`, {
				method: 'POST',
			})
			notification.is_read = true
			unreadCount.value = Math.max(0, unreadCount.value - 1)
		} catch (error) {
			console.error('Failed to mark as read:', error)
		}
	}

	// Navigate if has action URL
	if (notification.action_url) {
		showPanel.value = false
		router.push(notification.action_url)
	}
}

const archiveNotification = async (id) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/notifications/${id}/archive`, {
			method: 'POST',
		})
		const data = await response.json()
		if (data.success) {
			notifications.value = notifications.value.filter((n) => n.id !== id)
			const notification = notifications.value.find((n) => n.id === id)
			if (notification && !notification.is_read) {
				unreadCount.value = Math.max(0, unreadCount.value - 1)
			}
		}
	} catch (error) {
		console.error('Failed to archive notification:', error)
	}
}

const getNotificationIcon = (category) => {
	const icons = {
		system: ['fas', 'server'],
		scraper: ['fas', 'spider'],
		downloads: ['fas', 'download'],
		scheduler: ['fas', 'clock'],
		ai: ['fas', 'brain'],
	}
	return icons[category] || ['fas', 'bell']
}

const formatTime = (dateStr) => {
	const date = new Date(dateStr)
	const now = new Date()
	const diffMs = now - date
	const diffMins = Math.floor(diffMs / 60000)
	const diffHours = Math.floor(diffMs / 3600000)
	const diffDays = Math.floor(diffMs / 86400000)

	if (diffMins < 1) return 'Just now'
	if (diffMins < 60) return `${diffMins}m ago`
	if (diffHours < 24) return `${diffHours}h ago`
	if (diffDays < 7) return `${diffDays}d ago`
	return date.toLocaleDateString()
}

onMounted(() => {
	loadStats()
	// Refresh stats every 30 seconds
	refreshInterval = setInterval(loadStats, 30000)

	// Subscribe to WebSocket notification events
	unsubscribeNotification = websocketService.on('notification', (data) => {
		const notification = data.data
		// Add notification to the list
		notifications.value.unshift(notification)
		// Increment unread count
		unreadCount.value++
		// Show toast notification
		toast.info(notification.title)
	})
})

onBeforeUnmount(() => {
	if (refreshInterval) {
		clearInterval(refreshInterval)
	}
	if (unsubscribeNotification) {
		unsubscribeNotification()
	}
})
</script>

<style scoped>
.notification-bell {
	position: relative;
}

.bell-button {
	position: relative;
	background: none;
	border: none;
	color: rgba(255, 255, 255, 0.8);
	font-size: 1.2rem;
	cursor: pointer;
	padding: 0.5rem;
	transition: all 0.3s ease;
}

.bell-button:hover {
	color: #fff;
	transform: scale(1.1);
}

.bell-button.has-unread {
	color: #4caf50;
}

.badge {
	position: absolute;
	top: 0;
	right: 0;
	background: #f44336;
	color: white;
	font-size: 0.65rem;
	font-weight: 700;
	padding: 0.15rem 0.4rem;
	border-radius: 10px;
	min-width: 18px;
	text-align: center;
}

.notification-panel {
	position: fixed;
	top: 60px;
	right: 0;
	width: 420px;
	max-width: 100vw;
	height: calc(100vh - 60px);
	background: rgba(20, 20, 30, 0.98);
	backdrop-filter: blur(20px);
	border-left: 1px solid rgba(255, 255, 255, 0.1);
	z-index: 1000;
	display: flex;
	flex-direction: column;
	box-shadow: -5px 0 30px rgba(0, 0, 0, 0.5);
}

.panel-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.panel-header h3 {
	margin: 0;
	color: #e0e0e0;
	font-size: 1.3rem;
}

.header-actions {
	display: flex;
	align-items: center;
	gap: 1rem;
}

.btn-link {
	background: none;
	border: none;
	color: #667eea;
	cursor: pointer;
	font-size: 0.9rem;
	padding: 0;
}

.btn-link:hover {
	text-decoration: underline;
}

.close-btn {
	background: none;
	border: none;
	color: #aaa;
	font-size: 1.2rem;
	cursor: pointer;
	padding: 0;
	width: 30px;
	height: 30px;
	display: flex;
	align-items: center;
	justify-content: center;
}

.close-btn:hover {
	color: #fff;
}

.panel-filters {
	display: flex;
	gap: 0.5rem;
	padding: 1rem 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
	overflow-x: auto;
}

.filter-btn {
	padding: 0.4rem 1rem;
	border-radius: 20px;
	border: 1px solid rgba(255, 255, 255, 0.2);
	background: rgba(255, 255, 255, 0.05);
	color: #aaa;
	cursor: pointer;
	font-size: 0.85rem;
	white-space: nowrap;
	transition: all 0.2s ease;
}

.filter-btn:hover {
	background: rgba(255, 255, 255, 0.1);
	border-color: rgba(255, 255, 255, 0.3);
}

.filter-btn.active {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	border-color: transparent;
	color: white;
}

.panel-loading,
.panel-empty {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	flex: 1;
	color: #aaa;
	gap: 1rem;
}

.notifications-list {
	flex: 1;
	overflow-y: auto;
}

.notification-item {
	display: flex;
	gap: 1rem;
	padding: 1rem 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.05);
	cursor: pointer;
	transition: all 0.2s ease;
}

.notification-item:hover {
	background: rgba(255, 255, 255, 0.05);
}

.notification-item.unread {
	background: rgba(103, 126, 234, 0.05);
	border-left: 3px solid #667eea;
}

.notification-item.priority-urgent {
	border-left: 3px solid #f44336;
}

.notification-item.priority-high {
	border-left: 3px solid #ff9800;
}

.notification-icon {
	width: 40px;
	height: 40px;
	border-radius: 10px;
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0;
	font-size: 1.2rem;
}

.icon-system {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.icon-scraper {
	background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.icon-downloads {
	background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.icon-scheduler {
	background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
}

.icon-ai {
	background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
}

.notification-content {
	flex: 1;
	min-width: 0;
}

.notification-title {
	font-weight: 600;
	color: #e0e0e0;
	margin-bottom: 0.25rem;
	font-size: 0.95rem;
}

.notification-message {
	color: #aaa;
	font-size: 0.85rem;
	margin-bottom: 0.5rem;
	overflow: hidden;
	text-overflow: ellipsis;
	display: -webkit-box;
	-webkit-line-clamp: 2;
	-webkit-box-orient: vertical;
}

.notification-time {
	font-size: 0.75rem;
	color: #888;
}

.notification-action {
	background: none;
	border: none;
	color: #888;
	cursor: pointer;
	padding: 0;
	width: 24px;
	height: 24px;
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0;
	border-radius: 4px;
	transition: all 0.2s ease;
}

.notification-action:hover {
	background: rgba(255, 255, 255, 0.1);
	color: #fff;
}

.slide-fade-enter-active {
	transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
	transition: all 0.3s ease-in;
}

.slide-fade-enter-from {
	transform: translateX(100%);
	opacity: 0;
}

.slide-fade-leave-to {
	transform: translateX(100%);
	opacity: 0;
}
</style>
