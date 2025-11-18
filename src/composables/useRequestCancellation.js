import { onBeforeUnmount } from 'vue'
import requestManager from '@/utils/requestCancellation'

/**
 * Vue composable for managing request cancellation
 * Automatically cleans up controllers when component is unmounted
 */
export function useRequestCancellation() {
	const activeKeys = new Set()

	/**
	 * Create a cancellable request
	 * @param {string} key - Unique identifier for this request type
	 * @returns {AbortSignal} - Signal to pass to API call
	 */
	const createCancellableRequest = (key) => {
		activeKeys.add(key)
		const controller = requestManager.createController(key)
		return controller.signal
	}

	/**
	 * Cancel a specific request
	 * @param {string} key - Request key to cancel
	 */
	const cancelRequest = (key) => {
		requestManager.cancel(key)
		activeKeys.delete(key)
	}

	/**
	 * Mark a request as completed (for cleanup)
	 * @param {string} key - Request key
	 */
	const completeRequest = (key) => {
		requestManager.cleanup(key)
		activeKeys.delete(key)
	}

	// Cleanup all active requests when component unmounts
	onBeforeUnmount(() => {
		activeKeys.forEach((key) => {
			requestManager.cancel(key)
		})
		activeKeys.clear()
	})

	return {
		createCancellableRequest,
		cancelRequest,
		completeRequest,
	}
}
