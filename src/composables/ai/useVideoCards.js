/**
 * useVideoCards — inline video card actions (rating, favorites, navigation).
 */
import { useRouter } from 'vue-router'
import { videosAPI } from '@/services/api'
import { getAssetURL } from '@/services/api'

export function useVideoCards() {
	const router = useRouter()

	const getVideoThumbURL = (thumbnailPath) => {
		if (!thumbnailPath) return null
		return getAssetURL(thumbnailPath)
	}

	const navigateToVideo = (videoId, closeFn) => {
		router.push({ path: '/videos', query: { highlight: videoId } })
		closeFn?.()
	}

	const compareCardsInAnalyzer = (videos, closeFn) => {
		const ids = videos.filter((v) => v.id > 0).map((v) => v.id).join(',')
		if (!ids) return
		router.push({ path: '/analyzer', query: { ids } })
		closeFn?.()
	}

	const rateCardVideo = async (video, rating) => {
		const newRating = video.rating === rating ? 0 : rating
		try {
			await videosAPI.update(video.id, { rating: newRating })
			video.rating = newRating
		} catch { /* best-effort */ }
	}

	const toggleCardFavorite = async (video) => {
		const newFav = !video.is_favorite
		try {
			await videosAPI.update(video.id, { is_favorite: newFav })
			video.is_favorite = newFav
		} catch { /* best-effort */ }
	}

	return {
		getVideoThumbURL,
		navigateToVideo,
		compareCardsInAnalyzer,
		rateCardVideo,
		toggleCardFavorite,
	}
}
