/**
 * useScraperJobPolling
 *
 * Generic composable for polling a scraper job's progress.
 * Eliminates the duplicated pollXxxJobProgress pattern (8 copies) in ScraperPage.vue.
 *
 * @example
 * const { pollJob } = useScraperJobPolling()
 *
 * pollJob(jobId, {
 *   activeJob:    hqPornerActiveJob,   // Ref<job|null> — updated each poll
 *   isScraping:   hqPornerScraping,   // Ref<bool>     — set false when done
 *   activeTimeouts,                    // Ref<[]>       — interval id is registered here
 *   onSuccess: async (job) => {
 *     toast.success(`Done! Found ${job.videosFound} videos`)
 *     await loadVideos()
 *     await loadStats()
 *   },
 *   onFail: (job) => toast.error('Scrape failed'),
 * })
 */
import { scrapersAPI } from '@/services/api'

export function useScraperJobPolling() {
	const pollJob = (
		jobId,
		{ activeJob, isScraping, activeTimeouts, onSuccess, onFail, interval = 2000 },
	) => {
		const pollInterval = setInterval(async () => {
			try {
				const data = await scrapersAPI.getJob(jobId)

				if (data.success && data.data) {
					const job = data.data
					activeJob.value = {
						id:             job.id,
						status:         job.status,
						pagesCompleted: job.pagesCompleted || 0,
						videosFound:    job.videosFound    || 0,
						videosSaved:    job.videosSaved    || 0,
						errorsCount:    job.errorsCount    || 0,
					}

					const isDone = ['completed', 'failed', 'cancelled'].includes(job.status)
					if (isDone) {
						clearInterval(pollInterval)
						isScraping.value = false

						if (job.status === 'completed' && onSuccess) {
							await onSuccess(job)
						} else if (job.status !== 'completed' && onFail) {
							await onFail(job)
						}

						setTimeout(() => { activeJob.value = null }, 5000)
					}
				}
			} catch (error) {
				console.error('Error polling scraper job:', error)
			}
		}, interval)

		if (activeTimeouts) {
			activeTimeouts.value.push(pollInterval)
		}

		return pollInterval
	}

	return { pollJob }
}
