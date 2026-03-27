/**
 * Parses video.metadata JSON and returns the performers array.
 * Each entry: { id: number, name: string }
 * id > 0 means matched to a DB performer; id === 0 means unmatched.
 *
 * @param {object} video
 * @returns {Array<{id: number, name: string}>}
 */
export function getVideoPerformers(video) {
	if (!video || !video.metadata) return []
	try {
		const meta = typeof video.metadata === 'string' ? JSON.parse(video.metadata) : video.metadata
		return meta.performers || []
	} catch {
		return []
	}
}

/**
 * Parses video.metadata JSON and returns matched tags from the DB.
 * Each entry: { id: number, name: string, color: string, icon: string }
 *
 * @param {object} video
 * @returns {Array<{id: number, name: string, color: string, icon: string}>}
 */
export function getVideoMatchedTags(video) {
	if (!video || !video.metadata) return []
	try {
		const meta = typeof video.metadata === 'string' ? JSON.parse(video.metadata) : video.metadata
		return meta.matched_tags || []
	} catch {
		return []
	}
}
