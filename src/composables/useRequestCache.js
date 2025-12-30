import { ref, onUnmounted } from 'vue'
import requestCache from '@/utils/requestCache'

/**
 * Composable for cached API requests
 * Automatically caches GET requests and provides cache invalidation
 *
 * @param {Function} apiFunction - The API function to call
 * @param {Object} options - Configuration options
 * @returns {Object} - Reactive data and methods
 */
export function useCachedRequest(apiFunction, options = {}) {
	const {
		cacheKey = null,
		ttl = 5 * 60 * 1000, // 5 minutes default
		enableCache = true,
		onSuccess = null,
		onError = null,
	} = options

	const data = ref(null)
	const loading = ref(false)
	const error = ref(null)

	/**
	 * Execute the API request with caching
	 */
	const execute = async (...args) => {
		const key = cacheKey || apiFunction.name

		// Check cache first if enabled
		if (enableCache) {
			const cached = requestCache.get(key, args)
			if (cached) {
				data.value = cached
				return cached
			}
		}

		loading.value = true
		error.value = null

		try {
			const result = await apiFunction(...args)
			data.value = result

			// Cache the result if enabled
			if (enableCache) {
				requestCache.set(key, args, result, ttl)
			}

			if (onSuccess) {
				onSuccess(result)
			}

			return result
		} catch (err) {
			error.value = err
			if (onError) {
				onError(err)
			}
			throw err
		} finally {
			loading.value = false
		}
	}

	/**
	 * Invalidate cache for this request
	 */
	const invalidate = () => {
		const key = cacheKey || apiFunction.name
		requestCache.invalidatePattern(key)
	}

	/**
	 * Refresh data (bypass cache)
	 */
	const refresh = async (...args) => {
		invalidate()
		return await execute(...args)
	}

	return {
		data,
		loading,
		error,
		execute,
		invalidate,
		refresh,
	}
}

/**
 * Composable for debounced search
 *
 * @param {Function} searchFunction - The search function to debounce
 * @param {number} delay - Debounce delay in ms
 * @returns {Object} - Search state and methods
 */
export function useDebouncedSearch(searchFunction, delay = 300) {
	const results = ref([])
	const loading = ref(false)
	const searchTerm = ref('')
	let timeoutId = null

	const search = (term) => {
		searchTerm.value = term

		// Clear previous timeout
		if (timeoutId) {
			clearTimeout(timeoutId)
		}

		// Return early if search term is empty
		if (!term || term.trim() === '') {
			results.value = []
			loading.value = false
			return
		}

		loading.value = true

		// Set new timeout
		timeoutId = setTimeout(async () => {
			try {
				const data = await searchFunction(term)
				results.value = data
			} catch (error) {
				console.error('Search error:', error)
				results.value = []
			} finally {
				loading.value = false
			}
		}, delay)
	}

	const clear = () => {
		if (timeoutId) {
			clearTimeout(timeoutId)
		}
		searchTerm.value = ''
		results.value = []
		loading.value = false
	}

	// Cleanup on unmount
	onUnmounted(() => {
		if (timeoutId) {
			clearTimeout(timeoutId)
		}
	})

	return {
		search,
		clear,
		results,
		loading,
		searchTerm,
	}
}

export default { useCachedRequest, useDebouncedSearch }
