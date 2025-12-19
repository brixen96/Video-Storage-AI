<template>
	<teleport to="body">
		<div class="toast-container">
			<transition-group name="toast">
				<div v-for="toast in toasts" :key="toast.id" :class="['toast', `toast-${toast.type}`]">
					<div class="toast-icon">
						<font-awesome-icon v-if="toast.type === 'success'" :icon="['fas', 'check-circle']" />
						<font-awesome-icon v-else-if="toast.type === 'error'" :icon="['fas', 'exclamation-triangle']" />
						<font-awesome-icon v-else-if="toast.type === 'warning'" :icon="['fas', 'exclamation-circle']" />
						<font-awesome-icon v-else-if="toast.type === 'info'" :icon="['fas', 'info-circle']" />
						<font-awesome-icon v-else :icon="['fas', 'spinner']" spin />
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
			const index = this.toasts.findIndex((t) => t.id === id)
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
		warning(title, message, duration) {
			return this.show({ type: 'warning', title, message, duration })
		},
		loading(title, message) {
			return this.show({ type: 'loading', title, message, duration: 0 })
		},
	},
}
</script>

<style scoped>
@import '@/styles/components/toast_notification.css';
</style>
