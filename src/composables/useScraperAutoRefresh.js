import { onMounted, onUnmounted } from 'vue'
import websocketService from '@/services/websocket'

/**
 * Subscribes to feed_update WebSocket events and calls loadFn() when a
 * new-content event for the given scraperType arrives.
 *
 * @param {string} scraperType - e.g. "pornhub", "luxuretv", "sexuria"
 * @param {Function} loadFn    - the tab's loadVideos() function
 */
export function useScraperAutoRefresh(scraperType, loadFn) {
	let unsubscribe = null

	onMounted(() => {
		unsubscribe = websocketService.on('feed_update', (data) => {
			if (data && data.scraper_type === scraperType && data.new_count > 0) {
				loadFn()
			}
		})
	})

	onUnmounted(() => {
		if (unsubscribe) unsubscribe()
	})
}
