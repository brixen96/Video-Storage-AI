class CacheService {
	constructor() {
		this.cache = new Map()
		this.timestamps = new Map()
		this.defaultTTL = 5 * 60 * 1000 // 5 minutes
	}

	set(key, data, ttl = this.defaultTTL) {
		this.cache.set(key, data)
		this.timestamps.set(key, Date.now() + ttl)
	}

	get(key) {
		const timestamp = this.timestamps.get(key)
		if (!timestamp || Date.now() > timestamp) {
			this.cache.delete(key)
			this.timestamps.delete(key)
			return null
		}
		return this.cache.get(key)
	}

	invalidate(pattern) {
		// Invalidate cache entries matching pattern
		for (const key of this.cache.keys()) {
			if (key.includes(pattern)) {
				this.cache.delete(key)
				this.timestamps.delete(key)
			}
		}
	}

	clear() {
		this.cache.clear()
		this.timestamps.clear()
	}
}

export default new CacheService()
