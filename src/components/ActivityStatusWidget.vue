
<template>
    <div class="activity-status-widget px-2 py-2" v-click-outside="closeDetails">
        <div class="status-indicator-container" @click="toggleDetails">
            <div class="status-indicator" :class="statusColor">
                <div class="status-pulse" :class="statusColor" v-if="statusColor !== 'green'"></div>
            </div>
            <div class="status-text">
                <div class="status-label">{{ statusLabel }}</div>
                <div class="status-message" v-if="currentActivity">{{ currentActivity.message }}</div>
                <div class="status-message" v-else>{{ idleMessage }}</div>
            </div>
        </div>

        <!-- Expandable Details -->
        <transition name="slide-fade">
            <div v-if="showDetails" class="status-details position-absolute rounded mt-2 p-2 text-center">
                <div v-if="currentActivity" class="current-task">
                    <div class="task-header">
                        <span class="badge" :class="getTaskTypeBadge(currentActivity.task_type)">
                            {{ formatTaskType(currentActivity.task_type) }}
                        </span>
                        <span class="task-time">{{ formatTime(currentActivity.started_at) }}</span>
                    </div>
                    <div class="progress mt-2" style="height: 6px">
                        <div
                            class="progress-bar progress-bar-striped progress-bar-animated"
                            :class="statusColor === 'yellow' ? 'bg-warning' : 'bg-danger'"
                            :style="{ width: currentActivity.progress + '%' }"
                        ></div>
                    </div>
                    <small class="progress-text">{{ currentActivity.progress }}% complete</small>
                </div>
                <div v-else class="idle-state">
                    <font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
                    <span class="text-info">All systems operational</span>
                </div>
                <router-link to="/activity" class="btn btn-sm btn-outline-primary mt-2 w-100">
                    <font-awesome-icon :icon="['fas', 'chart-line']" class="me-1" />
                    View Full Monitor
                </router-link>
            </div>
        </transition>
    </div>
</template>

<script>
import { activityAPI } from '@/services/api'

export default {
    name: 'ActivityStatusWidget',
    data() {
        return {
            status: {},
            currentActivity: null,
            autoRefresh: null,
            showDetails: false,
        }
    },
    computed: {
        statusColor() {
            // Red: Error (failed tasks)
            if (this.status.failed_tasks > 0 || (this.currentActivity && this.currentActivity.status === 'failed')) {
                return 'red'
            }
            // Yellow: Processing (running or pending tasks)
            if (this.status.running_tasks > 0 || this.status.pending_tasks > 0) {
                return 'yellow'
            }
            // Green: Idle (no active tasks)
            return 'green'
        },
        statusLabel() {
            if (this.statusColor === 'red') return 'Error'
            if (this.statusColor === 'yellow') return 'Processing'
            return 'Idle'
        },
        idleMessage() {
            return 'No active tasks'
        },
    },
    directives: {
        clickOutside: {
            mounted(el, binding) {
                el.clickOutsideEvent = function (event) {
                    // Check if the click is outside the element
                    if (!(el === event.target || el.contains(event.target))) {
                        // Call the provided method
                        binding.value(event)
                    }
                }
                document.addEventListener('click', el.clickOutsideEvent)
            },
            unmounted(el) {
                document.removeEventListener('click', el.clickOutsideEvent)
            },
        },
    },
    async mounted() {
        await this.loadStatus()
        this.startAutoRefresh()
    },
    beforeUnmount() {
        this.stopAutoRefresh()
    },
    methods: {
        async loadStatus() {
            try {
                const response = await activityAPI.getStatus()
                this.status = response.data || {}

                // Get the current running/pending task
                if (this.status.current_tasks && this.status.current_tasks.length > 0) {
                    this.currentActivity = this.status.current_tasks[0]
                } else {
                    // Check for failed tasks
                    try {
                        const failedResponse = await activityAPI.getAll({ status: 'failed', limit: 1 })
                        if (failedResponse.data && failedResponse.data.length > 0) {
                            this.currentActivity = failedResponse.data[0]
                        } else {
                            this.currentActivity = null
                        }
                    } catch (error) {
                        this.currentActivity = null
                    }
                }
            } catch (error) {
                console.error('Failed to load status:', error)
            }
        },
        toggleDetails() {
            this.showDetails = !this.showDetails
        },
        closeDetails() {
            this.showDetails = false
        },
        startAutoRefresh() {
            // Refresh every 3 seconds
            this.autoRefresh = setInterval(() => {
                this.loadStatus()
            }, 3000)
        },
        stopAutoRefresh() {
            if (this.autoRefresh) {
                clearInterval(this.autoRefresh)
            }
        },
        formatTaskType(taskType) {
            return taskType
                .split('_')
                .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
                .join(' ')
        },
        formatTime(dateString) {
            const date = new Date(dateString)
            return date.toLocaleTimeString()
        },
        getTaskTypeBadge(taskType) {
            const badges = {
                scanning: 'bg-info',
                indexing: 'bg-primary',
                ai_tagging: 'bg-purple',
                metadata_fetch: 'bg-cyan',
                thumbnail_generation: 'bg-warning',
                video_analysis: 'bg-success',
                file_operation: 'bg-secondary',
            }
            return badges[taskType] || 'bg-dark'
        },
    },
}
</script>

<style scoped>
.activity-status-widget {
	position: relative;
	background: rgba(255, 255, 255, 0.05);
	border-radius: 0.75rem;
	backdrop-filter: blur(10px);
	transition: all 0.3s ease;
}

.activity-status-widget:hover {
	background: rgba(255, 255, 255, 0.08);
}

.status-indicator-container {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	cursor: pointer;
	user-select: none;
}

.status-indicator {
	position: relative;
	width: 16px;
	height: 16px;
	border-radius: 50%;
	flex-shrink: 0;
	transition: all 0.3s ease;
}

.status-indicator.green {
	background: #28a745;
	box-shadow: 0 0 10px rgba(40, 167, 69, 0.5);
}

.status-indicator.yellow {
	background: #ffc107;
	box-shadow: 0 0 10px rgba(255, 193, 7, 0.5);
}

.status-indicator.red {
	background: #dc3545;
	box-shadow: 0 0 10px rgba(220, 53, 69, 0.5);
}

.status-pulse {
	position: absolute;
	top: 50%;
	left: 50%;
	transform: translate(-50%, -50%);
	width: 100%;
	height: 100%;
	border-radius: 50%;
	animation: pulse-ring 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

.status-pulse.yellow {
	background: rgba(255, 193, 7, 0.5);
}

.status-pulse.red {
	background: rgba(220, 53, 69, 0.5);
}

@keyframes pulse-ring {
	0% {
		transform: translate(-50%, -50%) scale(1);
		opacity: 1;
	}
	100% {
		transform: translate(-50%, -50%) scale(2.5);
		opacity: 0;
	}
}

.status-text {
	flex: 1;
	min-width: 0;
}

.status-label {
	font-weight: 600;
	font-size: 0.875rem;
	color: #00d9ff;
	margin-bottom: 0.125rem;
}

.status-message {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.7);
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.status-details {
	background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%) !important;

	border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.current-task {
	margin-bottom: 0.5rem;
}

.task-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 0.5rem;
}

.task-time {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.6);
}

.progress {
	background: rgba(0, 0, 0, 0.3);
	border-radius: 0.5rem;
}

.progress-text {
	display: block;
	margin-top: 0.25rem;
	font-size: 0.7rem;
	color: rgba(255, 255, 255, 0.6);
}

.idle-state {
	display: flex;
	align-items: center;
	padding: 0.5rem 0;
	font-size: 0.875rem;
	color: rgba(255, 255, 255, 0.8);
}

.badge.bg-purple {
	background-color: #6f42c1 !important;
}

.badge.bg-cyan {
	background-color: #00d9ff !important;
	color: #000 !important;
}

.btn-outline-primary {
	border-color: rgba(0, 217, 255, 0.5);
	color: #00d9ff;
	font-size: 0.75rem;
}

.btn-outline-primary:hover {
	background-color: #00d9ff;
	border-color: #00d9ff;
	color: #000;
}

/* Transition animations */
.slide-fade-enter-active {
	transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
	transition: all 0.2s cubic-bezier(1, 0.5, 0.8, 1);
}

.slide-fade-enter-from {
	transform: translateY(-10px);
	opacity: 0;
}

.slide-fade-leave-to {
	transform: translateY(-5px);
	opacity: 0;
}
</style>
