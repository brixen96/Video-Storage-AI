/**
 * Simple in-memory cache for API requests
 * Helps reduce redundant network calls and improve performance
 */

class RequestCache {
	constructor(defaultTTL = 5 * 60 * 1000) {
		// 5 minutes default
		this.cache = new Map()
		this.defaultTTL = defaultTTL
	}

	/**
	 * Generate cache key from request parameters
	 */
	generateKey(endpoint, params = {}) {
		const paramString = Object.keys(params)
			.sort()
			.map((key) => `${key}=${JSON.stringify(params[key])}`)
			.join('&')
		return `${endpoint}?${paramString}`
	}

	/**
	 * Get cached response if available and not expired
	 */
	get(endpoint, params = {}) {
		const key = this.generateKey(endpoint, params)
		const cached = this.cache.get(key)

		if (!cached) return null

		// Check if expired
		if (Date.now() > cached.expiry) {
			this.cache.delete(key)
			return null
		}

		return cached.data
	}

	/**
	 * Set cache entry with optional TTL
	 */
	set(endpoint, params = {}, data, ttl = this.defaultTTL) {
		const key = this.generateKey(endpoint, params)
		this.cache.set(key, {
			data,
			expiry: Date.now() + ttl,
		})
	}

	/**
	 * Clear specific cache entry
	 */
	clear(endpoint, params = {}) {
		const key = this.generateKey(endpoint, params)
		this.cache.delete(key)
	}

	/**
	 * Clear all cache entries
	 */
	clearAll() {
		this.cache.clear()
	}

	/**
	 * Clear expired entries
	 */
	clearExpired() {
		const now = Date.now()
		for (const [key, value] of this.cache.entries()) {
			if (now > value.expiry) {
				this.cache.delete(key)
			}
		}
	}

	/**
	 * Invalidate cache entries matching a pattern
	 */
	invalidatePattern(pattern) {
		const regex = new RegExp(pattern)
		for (const key of this.cache.keys()) {
			if (regex.test(key)) {
				this.cache.delete(key)
			}
		}
	}

	/**
	 * Get cache statistics
	 */
	getStats() {
		return {
			size: this.cache.size,
			keys: Array.from(this.cache.keys()),
		}
	}
}

// Create singleton instance
const requestCache = new RequestCache()

// Auto-clear expired entries every 5 minutes
setInterval(() => {
	requestCache.clearExpired()
}, 5 * 60 * 1000)

export default requestCache
