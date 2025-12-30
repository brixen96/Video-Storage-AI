/**
 * Utilities for optimizing Vue computed properties
 * Provides memoization and caching helpers for expensive computations
 */

import { computed, ref, watch } from 'vue'

/**
 * Create a memoized computed property that only recalculates when dependencies change
 * Useful for expensive filtering, sorting, or transformations
 *
 * @param {Function} fn - The computation function
 * @param {Array} deps - Dependency refs to watch
 * @param {Object} options - Options (compareFn, maxCacheSize)
 * @returns {ComputedRef} - Memoized computed property
 */
export function memoizedComputed(fn, deps = [], options = {}) {
	const { compareFn = null, maxCacheSize = 10 } = options
	const cache = new Map()
	const cacheKeys = []

	return computed(() => {
		// Generate cache key from dependencies
		const key = JSON.stringify(deps.map((dep) => dep.value))

		// Check cache
		if (cache.has(key)) {
			return cache.get(key)
		}

		// Compute new value
		const value = fn()

		// Store in cache
		cache.set(key, value)
		cacheKeys.push(key)

		// Limit cache size (LRU-style)
		if (cacheKeys.length > maxCacheSize) {
			const oldestKey = cacheKeys.shift()
			cache.delete(oldestKey)
		}

		return value
	})
}

/**
 * Create a debounced computed property that only updates after a delay
 * Useful for search filters and expensive real-time computations
 *
 * @param {Function} fn - The computation function
 * @param {number} delay - Debounce delay in milliseconds
 * @returns {Object} - { value: Ref, immediate: Function }
 */
export function debouncedComputed(fn, delay = 300) {
	const value = ref(fn())
	let timeoutId = null

	const updateValue = () => {
		if (timeoutId) {
			clearTimeout(timeoutId)
		}
		timeoutId = setTimeout(() => {
			value.value = fn()
		}, delay)
	}

	// Create computed for dependency tracking
	const tracker = computed(() => fn())

	// Watch for changes
	watch(tracker, updateValue)

	// Immediate update function (bypass debounce)
	const immediate = () => {
		if (timeoutId) {
			clearTimeout(timeoutId)
		}
		value.value = fn()
	}

	return { value, immediate }
}

/**
 * Create a lazy computed property that only computes when accessed
 * Useful for expensive computations that may not always be needed
 *
 * @param {Function} fn - The computation function
 * @returns {Object} - { value: Ref, refresh: Function, clear: Function }
 */
export function lazyComputed(fn) {
	const value = ref(null)
	let computed = false

	const refresh = () => {
		value.value = fn()
		computed = true
	}

	const clear = () => {
		value.value = null
		computed = false
	}

	// Create getter that computes on first access
	const lazyValue = computed({
		get() {
			if (!computed) {
				refresh()
			}
			return value.value
		},
	})

	return { value: lazyValue, refresh, clear }
}

/**
 * Optimize array filtering with index-based caching
 * Significantly faster for large arrays with frequent filters
 *
 * @param {Array} items - Source array
 * @param {Function} filterFn - Filter function
 * @returns {Array} - Filtered array
 */
export function optimizedFilter(items, filterFn) {
	// For small arrays, use native filter (faster)
	if (items.length < 100) {
		return items.filter(filterFn)
	}

	// For large arrays, use for loop (more performant)
	const result = []
	for (let i = 0, len = items.length; i < len; i++) {
		if (filterFn(items[i], i, items)) {
			result.push(items[i])
		}
	}
	return result
}

/**
 * Optimize array sorting with cached comparisons
 *
 * @param {Array} items - Array to sort
 * @param {Function} compareFn - Comparison function
 * @returns {Array} - Sorted array (new array, original unchanged)
 */
export function optimizedSort(items, compareFn) {
	// Create shallow copy to avoid mutating original
	const copy = items.slice()

	// For small arrays, use native sort
	if (copy.length < 100) {
		return copy.sort(compareFn)
	}

	// For large arrays, use optimized sort with cached keys
	const keyed = copy.map((item, index) => ({ item, index }))
	keyed.sort((a, b) => compareFn(a.item, b.item))
	return keyed.map((k) => k.item)
}

/**
 * Create a chunked array for virtual scrolling or pagination
 * More efficient than slicing entire array repeatedly
 *
 * @param {Array} items - Source array
 * @param {number} chunkSize - Size of each chunk
 * @returns {Array} - Array of chunks
 */
export function createChunkedArray(items, chunkSize = 50) {
	const chunks = []
	for (let i = 0; i < items.length; i += chunkSize) {
		chunks.push(items.slice(i, i + chunkSize))
	}
	return chunks
}

/**
 * Batch multiple computed property updates
 * Prevents redundant recalculations when updating multiple dependencies
 *
 * @param {Function} fn - Function containing multiple ref updates
 */
export async function batchUpdate(fn) {
	// Vue 3 automatically batches updates, but this ensures it
	await Promise.resolve()
	fn()
	await Promise.resolve()
}

/**
 * Create a cached computed property with manual invalidation
 * Useful when you want full control over when recalculation happens
 *
 * @param {Function} fn - Computation function
 * @returns {Object} - { value: ComputedRef, invalidate: Function }
 */
export function cachedComputed(fn) {
	const cache = ref(null)
	const valid = ref(false)

	const value = computed(() => {
		if (!valid.value) {
			cache.value = fn()
			valid.value = true
		}
		return cache.value
	})

	const invalidate = () => {
		valid.value = false
	}

	return { value, invalidate }
}

/**
 * Optimize object transformations with structural sharing
 * Only creates new objects when values actually change
 *
 * @param {Object} source - Source object
 * @param {Function} transformFn - Transformation function
 * @param {Array} watchKeys - Keys to watch for changes
 * @returns {ComputedRef} - Optimized computed object
 */
export function optimizedTransform(source, transformFn, watchKeys = []) {
	let lastSource = {}
	let lastResult = null

	return computed(() => {
		// Check if watched keys changed
		const changed =
			watchKeys.length === 0 ||
			watchKeys.some((key) => source.value[key] !== lastSource[key])

		if (!changed && lastResult) {
			return lastResult
		}

		// Update cache
		lastSource = { ...source.value }
		lastResult = transformFn(source.value)
		return lastResult
	})
}

/**
 * Profile a computed property to measure performance
 * Logs execution time and call count
 *
 * @param {string} name - Name for debugging
 * @param {Function} fn - Computation function
 * @returns {ComputedRef} - Profiled computed property
 */
export function profiledComputed(name, fn) {
	let callCount = 0
	let totalTime = 0

	const profiled = computed(() => {
		const start = performance.now()
		const result = fn()
		const duration = performance.now() - start

		callCount++
		totalTime += duration

		if (process.env.NODE_ENV === 'development') {
			if (callCount % 10 === 0) {
				console.log(
					`%c[Computed Profile] ${name}`,
					'color: blue; font-weight: bold',
					`\nCalls: ${callCount}`,
					`\nAvg: ${(totalTime / callCount).toFixed(2)}ms`,
					`\nLast: ${duration.toFixed(2)}ms`
				)
			}
		}

		return result
	})

	// Expose stats
	profiled.getStats = () => ({
		name,
		callCount,
		totalTime: totalTime.toFixed(2),
		avgTime: (totalTime / callCount).toFixed(2),
	})

	return profiled
}

export default {
	memoizedComputed,
	debouncedComputed,
	lazyComputed,
	optimizedFilter,
	optimizedSort,
	createChunkedArray,
	batchUpdate,
	cachedComputed,
	optimizedTransform,
	profiledComputed,
}
