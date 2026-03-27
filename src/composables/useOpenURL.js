import { scrapersAPI } from '@/services/api'

/**
 * Composable that provides openInFirefox — opens a URL via the backend
 * (which launches Firefox) and falls back to window.open if unavailable.
 */
export function useOpenURL() {
	const openInFirefox = async (url) => {
		if (!url) return
		try {
			await scrapersAPI.openURL(url)
		} catch {
			window.open(url, '_blank')
		}
	}

	return { openInFirefox }
}
