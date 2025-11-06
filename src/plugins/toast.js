import Toast from '@/components/Toast.vue'

export default {
	install(app) {
		// Create a global toast instance
		const ToastConstructor = app.extend(Toast)
		const instance = new ToastConstructor()

		// Mount to body
		const el = document.createElement('div')
		document.body.appendChild(el)
		instance.$mount(el)

		// Add to global properties
		app.config.globalProperties.$toast = {
			show: (options) => instance.show(options),
			success: (title, message, duration) => instance.success(title, message, duration),
			error: (title, message, duration) => instance.error(title, message, duration),
			info: (title, message, duration) => instance.info(title, message, duration),
			loading: (title, message) => instance.loading(title, message),
			remove: (id) => instance.removeToast(id),
		}

		// Provide for composition API
		app.provide('toast', app.config.globalProperties.$toast)
	}
}
