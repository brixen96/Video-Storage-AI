/**
 * useVideoSelection
 *
 * Generic composable for managing per-scraper-tab video selection state.
 * Replaces the duplicated selection refs + toggle functions across all 7 scraper tabs.
 *
 * @example
 * const {
 *   selectionMode,
 *   selectedVideos,
 *   toggleSelectionMode,
 *   toggleVideoSelection,
 *   selectAll,
 *   clearSelection,
 * } = useVideoSelection()
 *
 * // With aliasing to keep existing names intact:
 * const {
 *   selectionMode:       hqPornerSelectionMode,
 *   selectedVideos:      hqPornerSelectedVideos,
 *   toggleSelectionMode: toggleHQPornerSelectionMode,
 *   toggleVideoSelection:toggleHQPornerVideoSelection,
 *   selectAll:           _hqPornerSelectAll,
 * } = useVideoSelection()
 * const selectAllHQPornerVideos = () => _hqPornerSelectAll(hqPornerVideos.value)
 */
import { ref } from 'vue'

export function useVideoSelection() {
	const selectionMode  = ref(false)
	const selectedVideos = ref(new Set())

	const toggleSelectionMode = () => {
		selectionMode.value = !selectionMode.value
		if (!selectionMode.value) {
			selectedVideos.value = new Set()
		}
	}

	/**
	 * Toggle a single video in/out of the selection.
	 * Creates a new Set to ensure Vue reactivity triggers.
	 */
	const toggleVideoSelection = (videoId) => {
		const next = new Set(selectedVideos.value)
		if (next.has(videoId)) {
			next.delete(videoId)
		} else {
			next.add(videoId)
		}
		selectedVideos.value = next
	}

	/**
	 * Add all videos in the given array to the selection.
	 * @param {Array} videos – array of video objects with .id
	 */
	const selectAll = (videos) => {
		const next = new Set(selectedVideos.value)
		videos.forEach((v) => next.add(v.id))
		selectedVideos.value = next
	}

	/** Clear selection and exit selection mode. */
	const clearSelection = () => {
		selectedVideos.value = new Set()
		selectionMode.value  = false
	}

	return {
		selectionMode,
		selectedVideos,
		toggleSelectionMode,
		toggleVideoSelection,
		selectAll,
		clearSelection,
	}
}
