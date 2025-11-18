// Request cancellation utility to prevent race conditions
class RequestCancellationManager {
	constructor() {
		// Map of request keys to AbortControllers
		this.controllers = new Map()
	}

	// Cancel any existing request with the given key
	cancel(key) {
		const controller = this.controllers.get(key)
		if (controller) {
			controller.abort()
			this.controllers.delete(key)
		}
	}

	// Create a new AbortController for a request
	// Automatically cancels any previous request with the same key
	createController(key) {
		// Cancel previous request if exists
		this.cancel(key)

		// Create new controller
		const controller = new AbortController()
		this.controllers.set(key, controller)
		return controller
	}

	// Get the signal for a request key
	getSignal(key) {
		const controller = this.controllers.get(key)
		return controller ? controller.signal : null
	}

	// Clean up after request completes
	cleanup(key) {
		this.controllers.delete(key)
	}

	// Cancel all requests
	cancelAll() {
		this.controllers.forEach((controller) => controller.abort())
		this.controllers.clear()
	}
}

// Export singleton instance
export default new RequestCancellationManager()
