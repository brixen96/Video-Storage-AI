import { ref, onBeforeUnmount } from 'vue'

/**
 * Composable for scraper tab sidebar + polling timeout cleanup.
 * Provides sidebarCollapsed, activeTimeouts, and auto-clears timeouts on unmount.
 *
 * @param {boolean} initialCollapsed - sidebar starts collapsed (default true)
 */
export function useScraperSidebar(initialCollapsed = true) {
	const sidebarCollapsed = ref(initialCollapsed)
	const activeTimeouts = ref([])

	onBeforeUnmount(() => {
		activeTimeouts.value.forEach((id) => clearInterval(id))
		activeTimeouts.value = []
	})

	return { sidebarCollapsed, activeTimeouts }
}
