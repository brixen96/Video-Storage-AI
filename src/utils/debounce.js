/**
 * Creates a debounced function that delays invoking func until after wait milliseconds
 * have elapsed since the last time the debounced function was invoked.
 *
 * @param {Function} func - The function to debounce
 * @param {number} wait - The number of milliseconds to delay
 * @param {boolean} immediate - If true, trigger the function on the leading edge instead of trailing
 * @returns {Function} The debounced function
 */
export function debounce(func, wait = 300, immediate = false) {
	let timeout

	return function executedFunction(...args) {
		const context = this

		const later = function () {
			timeout = null
			if (!immediate) func.apply(context, args)
		}

		const callNow = immediate && !timeout

		clearTimeout(timeout)
		timeout = setTimeout(later, wait)

		if (callNow) func.apply(context, args)
	}
}

/**
 * Creates a throttled function that only invokes func at most once per every wait milliseconds.
 *
 * @param {Function} func - The function to throttle
 * @param {number} wait - The number of milliseconds to throttle invocations to
 * @returns {Function} The throttled function
 */
export function throttle(func, wait = 300) {
	let inThrottle
	let lastTime

	return function executedFunction(...args) {
		const context = this

		if (!inThrottle) {
			func.apply(context, args)
			lastTime = Date.now()
			inThrottle = true

			setTimeout(() => {
				inThrottle = false
			}, wait)
		}
	}
}

/**
 * Delays execution of a function until it has been a specified amount of time
 * since the function was last called.
 *
 * @param {Function} func - The function to delay
 * @param {number} delay - The delay in milliseconds
 * @returns {Function} The delayed function with a cancel method
 */
export function delay(func, delay = 0) {
	let timeoutId

	const delayed = function (...args) {
		const context = this
		clearTimeout(timeoutId)
		timeoutId = setTimeout(() => {
			func.apply(context, args)
		}, delay)
	}

	delayed.cancel = function () {
		clearTimeout(timeoutId)
	}

	return delayed
}

export default { debounce, throttle, delay }
