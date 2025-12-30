/**
 * Performance monitoring utilities for tracking app performance
 * Provides timing, metrics collection, and performance analysis
 */

class PerformanceMonitor {
	constructor() {
		this.metrics = new Map()
		this.marks = new Map()
		this.enabled = process.env.NODE_ENV === 'development'
	}

	/**
	 * Start a performance measurement
	 */
	start(name) {
		if (!this.enabled) return

		const mark = `${name}-start`
		if (window.performance && window.performance.mark) {
			window.performance.mark(mark)
		}
		this.marks.set(name, performance.now())
	}

	/**
	 * End a performance measurement and log the result
	 */
	end(name, log = true) {
		if (!this.enabled) return 0

		const startTime = this.marks.get(name)
		if (!startTime) {
			console.warn(`Performance mark "${name}" not found`)
			return 0
		}

		const duration = performance.now() - startTime
		this.marks.delete(name)

		// Store metric
		if (!this.metrics.has(name)) {
			this.metrics.set(name, [])
		}
		this.metrics.get(name).push(duration)

		if (log) {
			const color = duration < 100 ? 'green' : duration < 500 ? 'orange' : 'red'
			console.log(
				`%c⏱ ${name}: ${duration.toFixed(2)}ms`,
				`color: ${color}; font-weight: bold`
			)
		}

		// Use Performance API if available
		if (window.performance && window.performance.mark && window.performance.measure) {
			const startMark = `${name}-start`
			const endMark = `${name}-end`
			window.performance.mark(endMark)
			try {
				window.performance.measure(name, startMark, endMark)
			} catch (e) {
				// Ignore if marks don't exist
			}
		}

		return duration
	}

	/**
	 * Measure a function execution time
	 */
	async measure(name, fn) {
		this.start(name)
		try {
			const result = await fn()
			return result
		} finally {
			this.end(name)
		}
	}

	/**
	 * Get statistics for a metric
	 */
	getStats(name) {
		const values = this.metrics.get(name)
		if (!values || values.length === 0) {
			return null
		}

		const sorted = [...values].sort((a, b) => a - b)
		const sum = values.reduce((a, b) => a + b, 0)
		const avg = sum / values.length
		const min = sorted[0]
		const max = sorted[sorted.length - 1]
		const median = sorted[Math.floor(sorted.length / 2)]
		const p95 = sorted[Math.floor(sorted.length * 0.95)]
		const p99 = sorted[Math.floor(sorted.length * 0.99)]

		return {
			name,
			count: values.length,
			min: min.toFixed(2),
			max: max.toFixed(2),
			avg: avg.toFixed(2),
			median: median.toFixed(2),
			p95: p95.toFixed(2),
			p99: p99.toFixed(2),
		}
	}

	/**
	 * Get all collected metrics
	 */
	getAllStats() {
		const stats = {}
		for (const name of this.metrics.keys()) {
			stats[name] = this.getStats(name)
		}
		return stats
	}

	/**
	 * Clear all metrics
	 */
	clear() {
		this.metrics.clear()
		this.marks.clear()
		if (window.performance && window.performance.clearMarks) {
			window.performance.clearMarks()
			window.performance.clearMeasures()
		}
	}

	/**
	 * Log all statistics to console
	 */
	report() {
		if (!this.enabled) return

		console.group('%c📊 Performance Report', 'font-size: 14px; font-weight: bold')
		const stats = this.getAllStats()

		if (Object.keys(stats).length === 0) {
			console.log('No metrics collected')
		} else {
			console.table(stats)
		}

		// Log web vitals if available
		this.logWebVitals()

		console.groupEnd()
	}

	/**
	 * Log Core Web Vitals
	 */
	logWebVitals() {
		if (!window.performance || !window.performance.getEntriesByType) return

		console.group('🎯 Core Web Vitals')

		// First Contentful Paint
		const paintEntries = window.performance.getEntriesByType('paint')
		const fcp = paintEntries.find((entry) => entry.name === 'first-contentful-paint')
		if (fcp) {
			console.log(`FCP: ${fcp.startTime.toFixed(2)}ms`)
		}

		// Navigation Timing
		const navTiming = window.performance.getEntriesByType('navigation')[0]
		if (navTiming) {
			console.log(`DOM Content Loaded: ${navTiming.domContentLoadedEventEnd.toFixed(2)}ms`)
			console.log(`Load Complete: ${navTiming.loadEventEnd.toFixed(2)}ms`)
			console.log(`DOM Interactive: ${navTiming.domInteractive.toFixed(2)}ms`)
		}

		// Resource Timing Summary
		const resources = window.performance.getEntriesByType('resource')
		if (resources.length > 0) {
			const totalSize = resources.reduce((sum, r) => sum + (r.transferSize || 0), 0)
			const totalDuration = resources.reduce((sum, r) => sum + r.duration, 0)
			console.log(`Resources: ${resources.length} total`)
			console.log(`Total Size: ${(totalSize / 1024 / 1024).toFixed(2)} MB`)
			console.log(`Avg Duration: ${(totalDuration / resources.length).toFixed(2)}ms`)
		}

		console.groupEnd()
	}

	/**
	 * Enable or disable monitoring
	 */
	setEnabled(enabled) {
		this.enabled = enabled
	}
}

// Create singleton instance
const performanceMonitor = new PerformanceMonitor()

// Expose globally for debugging
if (typeof window !== 'undefined') {
	window.performanceMonitor = performanceMonitor
}

// Auto-report on page unload in development
if (process.env.NODE_ENV === 'development') {
	window.addEventListener('beforeunload', () => {
		performanceMonitor.report()
	})
}

/**
 * Decorator for measuring Vue component methods
 */
export function measurePerformance(target, propertyKey, descriptor) {
	const originalMethod = descriptor.value

	descriptor.value = async function (...args) {
		const name = `${target.constructor.name}.${propertyKey}`
		performanceMonitor.start(name)
		try {
			const result = await originalMethod.apply(this, args)
			return result
		} finally {
			performanceMonitor.end(name)
		}
	}

	return descriptor
}

/**
 * Mark a component render
 */
export function markRender(componentName) {
	if (process.env.NODE_ENV === 'development') {
		performanceMonitor.start(`render:${componentName}`)
	}
}

/**
 * End component render mark
 */
export function endRender(componentName) {
	if (process.env.NODE_ENV === 'development') {
		performanceMonitor.end(`render:${componentName}`, false)
	}
}

/**
 * Measure API call performance
 */
export async function measureAPI(endpoint, fn) {
	return performanceMonitor.measure(`API: ${endpoint}`, fn)
}

export default performanceMonitor
