/**
 * Persistent Cache Service using Cache Storage API
 * Provides persistent caching that survives page reloads
 * Visible in DevTools > Application > Cache Storage
 */

const CACHE_NAME = 'video-storage-ai-v1'
const CACHE_TTL = 5 * 60 * 1000 // 5 minutes

class CacheService {
	constructor() {
		this.cacheAvailable = 'caches' in window
	}

	/**
	 * Generate cache key with timestamp metadata
	 */
	getCacheKey(url) {
		return new URL(url, window.location.origin).href
	}

	/**
	 * Store data in cache with TTL metadata
	 */
	async set(url, data) {
		if (!this.cacheAvailable) {
			console.warn('Cache Storage API not available')
			return false
		}

		try {
			const cache = await caches.open(CACHE_NAME)
			const cacheKey = this.getCacheKey(url)

			// Create response with TTL metadata in headers
			const response = new Response(JSON.stringify(data), {
				headers: {
					'Content-Type': 'application/json',
					'X-Cache-Timestamp': Date.now().toString(),
					'X-Cache-TTL': CACHE_TTL.toString(),
				},
			})

			await cache.put(cacheKey, response)
			return true
		} catch (error) {
			console.error('Failed to cache data:', error)
			return false
		}
	}

	/**
	 * Get data from cache if valid (not expired)
	 */
	async get(url) {
		if (!this.cacheAvailable) {
			return null
		}

		try {
			const cache = await caches.open(CACHE_NAME)
			const cacheKey = this.getCacheKey(url)
			const cachedResponse = await cache.match(cacheKey)

			if (!cachedResponse) {
				return null
			}

			// Check if cache is expired
			const timestamp = parseInt(cachedResponse.headers.get('X-Cache-Timestamp') || '0')
			const ttl = parseInt(cachedResponse.headers.get('X-Cache-TTL') || CACHE_TTL.toString())
			const age = Date.now() - timestamp

			if (age > ttl) {
				// Cache expired, delete it
				await cache.delete(cacheKey)
				return null
			}

			// Parse and return cached data
			const data = await cachedResponse.json()
			return data
		} catch (error) {
			console.error('Failed to get cached data:', error)
			return null
		}
	}

	/**
	 * Invalidate (delete) cached data for a specific URL
	 */
	async invalidate(url) {
		if (!this.cacheAvailable) {
			return false
		}

		try {
			const cache = await caches.open(CACHE_NAME)
			const cacheKey = this.getCacheKey(url)
			return await cache.delete(cacheKey)
		} catch (error) {
			console.error('Failed to invalidate cache:', error)
			return false
		}
	}

	/**
	 * Clear all cached data
	 */
	async clear() {
		if (!this.cacheAvailable) {
			return false
		}

		try {
			return await caches.delete(CACHE_NAME)
		} catch (error) {
			console.error('Failed to clear cache:', error)
			return false
		}
	}

	/**
	 * Get cache statistics
	 */
	async getStats() {
		if (!this.cacheAvailable) {
			return { available: false }
		}

		try {
			const cache = await caches.open(CACHE_NAME)
			const keys = await cache.keys()

			const entries = []
			for (const request of keys) {
				const response = await cache.match(request)
				if (response) {
					const timestamp = parseInt(response.headers.get('X-Cache-Timestamp') || '0')
					const ttl = parseInt(response.headers.get('X-Cache-TTL') || CACHE_TTL.toString())
					const age = Date.now() - timestamp

					entries.push({
						url: request.url,
						age: age,
						ttl: ttl,
						expired: age > ttl,
						expiresIn: Math.max(0, ttl - age),
					})
				}
			}

			return {
				available: true,
				cacheName: CACHE_NAME,
				totalEntries: entries.length,
				entries: entries,
			}
		} catch (error) {
			console.error('Failed to get cache stats:', error)
			return { available: false, error: error.message }
		}
	}

	/**
	 * Clean up expired entries
	 */
	async cleanup() {
		if (!this.cacheAvailable) {
			return 0
		}

		try {
			const cache = await caches.open(CACHE_NAME)
			const keys = await cache.keys()
			let deletedCount = 0

			for (const request of keys) {
				const response = await cache.match(request)
				if (response) {
					const timestamp = parseInt(response.headers.get('X-Cache-Timestamp') || '0')
					const ttl = parseInt(response.headers.get('X-Cache-TTL') || CACHE_TTL.toString())
					const age = Date.now() - timestamp

					if (age > ttl) {
						await cache.delete(request)
						deletedCount++
					}
				}
			}

			return deletedCount
		} catch (error) {
			console.error('Failed to cleanup cache:', error)
			return 0
		}
	}
}

// Export singleton instance
export default new CacheService()
