import { ref } from 'vue'
import { jdownloaderAPI } from '@/services/api'

/**
 * useSendToJD — bulk-send selected scraped videos to JDownloader.
 *
 * @param {object} toast       – The $toast proxy (proxy.$toast)
 * @param {string} scraperType – e.g. 'pornhub', used for /output/<scraperType> download path
 */
export function useSendToJD(toast, scraperType) {
	const sendingToJD = ref(false)

	/**
	 * @param {Array}  videos      – full array of video objects in the current tab
	 * @param {Set}    selectedIds – Set of selected video IDs
	 */
	const sendToJD = async (videos, selectedIds) => {
		if (!selectedIds || selectedIds.size === 0) return

		const links = [...selectedIds]
			.map((id) => videos.find((v) => v.id === id))
			.filter(Boolean)
			.map((v) => ({
				url: v.source_url || v.url,
				packageName: v.title || '',
				downloadPath: scraperType ? `/output/${scraperType}` : '/output',
				videoId: v.id ?? null,
			}))
			.filter((l) => l.url)

		if (links.length === 0) {
			toast?.warning?.('No valid URLs found for selected videos')
			return
		}

		sendingToJD.value = true
		try {
			const result = await jdownloaderAPI.sendLinks(links)
			const sent = result?.successCount ?? links.length
			toast?.success?.(`Sent ${sent} link${sent === 1 ? '' : 's'} to JDownloader`)
		} catch (e) {
			console.error('Failed to send to JDownloader:', e)
			toast?.error?.('Failed to send to JDownloader')
		} finally {
			sendingToJD.value = false
		}
	}

	return { sendingToJD, sendToJD }
}
