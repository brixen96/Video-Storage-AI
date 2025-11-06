<template>
	<teleport to="body">
		<div class="toast-container">
			<transition-group name="toast">
				<div
					v-for="toast in toasts"
					:key="toast.id"
					:class="['toast', `toast-${toast.type}`]"
				>
					<div class="toast-icon">
						<font-awesome-icon
							v-if="toast.type === 'success'"
							:icon="['fas', 'check-circle']"
						/>
						<font-awesome-icon
							v-else-if="toast.type === 'error'"
							:icon="['fas', 'exclamation-triangle']"
						/>
						<font-awesome-icon
							v-else-if="toast.type === 'info'"
							:icon="['fas', 'info-circle']"
						/>
						<font-awesome-icon
							v-else
							:icon="['fas', 'spinner']"
							spin
						/>
					</div>
					<div class="toast-content">
						<div class="toast-title">{{ toast.title }}</div>
						<div v-if="toast.message" class="toast-message">{{ toast.message }}</div>
					</div>
					<button class="toast-close" @click="removeToast(toast.id)">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
			</transition-group>
		</div>
	</teleport>
</template>

<script>
export default {
	name: 'ToastNotification',
	data() {
		return {
			toasts: [],
			nextId: 1,
		}
	},
	methods: {
		show(options) {
			const toast = {
				id: this.nextId++,
				type: options.type || 'info',
				title: options.title || '',
				message: options.message || '',
				duration: options.duration || 3000,
			}
			this.toasts.push(toast)

			if (toast.duration > 0) {
				setTimeout(() => {
					this.removeToast(toast.id)
				}, toast.duration)
			}

			return toast.id
		},
		removeToast(id) {
			const index = this.toasts.findIndex(t => t.id === id)
			if (index !== -1) {
				this.toasts.splice(index, 1)
			}
		},
		success(title, message, duration) {
			return this.show({ type: 'success', title, message, duration })
		},
		error(title, message, duration) {
			return this.show({ type: 'error', title, message, duration })
		},
		info(title, message, duration) {
			return this.show({ type: 'info', title, message, duration })
		},
		loading(title, message) {
			return this.show({ type: 'loading', title, message, duration: 0 })
		},
	},
}
</script>

<style scoped>
.toast-container {
	position: fixed;
	bottom: 2rem;
	left: 50%;
	transform: translateX(-50%);
	z-index: 9999;
	display: flex;
	flex-direction: column;
	gap: 0.75rem;
	max-width: 500px;
	width: 100%;
	padding: 0 1rem;
}

.toast {
	display: flex;
	align-items: flex-start;
	gap: 0.75rem;
	padding: 1rem;
	background: #1a1a2e;
	border-radius: 0.5rem;
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
	border-left: 4px solid;
	min-width: 300px;
}

.toast-success {
	border-left-color: #28a745;
}

.toast-error {
	border-left-color: #dc3545;
}

.toast-info {
	border-left-color: #00d9ff;
}

.toast-loading {
	border-left-color: #ffc107;
}

.toast-icon {
	font-size: 1.25rem;
	flex-shrink: 0;
}

.toast-success .toast-icon {
	color: #28a745;
}

.toast-error .toast-icon {
	color: #dc3545;
}

.toast-info .toast-icon {
	color: #00d9ff;
}

.toast-loading .toast-icon {
	color: #ffc107;
}

.toast-content {
	flex: 1;
}

.toast-title {
	font-weight: 600;
	color: #fff;
	margin-bottom: 0.25rem;
}

.toast-message {
	font-size: 0.875rem;
	color: rgba(255, 255, 255, 0.7);
}

.toast-close {
	background: none;
	border: none;
	color: rgba(255, 255, 255, 0.5);
	cursor: pointer;
	padding: 0;
	font-size: 1rem;
	transition: color 0.2s;
}

.toast-close:hover {
	color: #fff;
}

/* Transitions */
.toast-enter-active,
.toast-leave-active {
	transition: all 0.3s ease;
}

.toast-enter-from {
	opacity: 0;
	transform: translateY(100%);
}

.toast-leave-to {
	opacity: 0;
	transform: translateY(100%);
}
</style>
