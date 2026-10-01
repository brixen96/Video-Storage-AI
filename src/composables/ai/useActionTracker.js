/**
 * useActionTracker — reports meaningful UI actions to the AI Companion supervisor.
 * Module-level queue shared across all callers; debounced flush to keep the
 * telemetry pipe quiet (fire-and-forget, never blocks the UI).
 */
import { aiAPI } from '@/services/api'

let queue = []
let flushTimer = null

const flush = () => {
	if (flushTimer) {
		clearTimeout(flushTimer)
		flushTimer = null
	}
	if (queue.length === 0) return
	const batch = queue.slice(0, 20)
	queue = queue.slice(20)
	aiAPI.trackActions(batch).catch(() => { /* best-effort telemetry */ })
}

export function useActionTracker() {
	/**
	 * Queue an action for supervisor observation.
	 * @param {string} type - action type (e.g. scrape_started, settings_saved)
	 * @param {string} summary - human-readable one-liner
	 * @param {object|null} details - optional structured context
	 */
	const trackAction = (type, summary, details = null) => {
		if (!type || !summary) return
		queue.push({ type, summary, details: details ? JSON.stringify(details) : '' })
		if (queue.length >= 10) {
			flush()
			return
		}
		if (!flushTimer) flushTimer = setTimeout(flush, 3000)
	}

	return { trackAction }
}
