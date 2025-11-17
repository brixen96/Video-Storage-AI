import axios from 'axios'

const API_BASE_URL = 'http://localhost:8080/api/v1'

export default {
	/**
	 * Convert a video to MP4 format
	 * @param {number} videoId - The video ID to convert
	 * @returns {Promise} - The converted video data
	 */
	async convertToMP4(videoId) {
		const response = await axios.post(`${API_BASE_URL}/videos/${videoId}/convert`)
		return response
	},

	/**
	 * Check if FFmpeg is installed
	 * @returns {Promise} - FFmpeg installation status
	 */
	async checkFFmpegStatus() {
		const response = await axios.get(`${API_BASE_URL}/conversion/status`)
		return response
	},
}
