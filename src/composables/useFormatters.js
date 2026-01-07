/**
 * Composable for common formatting utilities
 *
 * Provides reusable formatting functions for:
 * - Duration (seconds to HH:MM:SS or MM:SS)
 * - File size (bytes to KB/MB/GB)
 * - Dates (various formats)
 * - Total duration (total time in videos)
 *
 * Usage:
 * import { useFormatters } from '@/composables/useFormatters'
 * const { formatDuration, formatFileSize, formatDate } = useFormatters()
 */

export function useFormatters() {
	/**
	 * Format duration in seconds to human-readable time
	 * @param {number} seconds - Duration in seconds
	 * @returns {string} Formatted duration (HH:MM:SS or MM:SS)
	 * @example
	 * formatDuration(90) // "1:30"
	 * formatDuration(3665) // "1:01:05"
	 */
	const formatDuration = (seconds) => {
		if (!seconds) return '00:00'
		const hours = Math.floor(seconds / 3600)
		const minutes = Math.floor((seconds % 3600) / 60)
		const secs = Math.floor(seconds % 60)

		if (hours > 0) {
			return `${hours}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
		}
		return `${minutes}:${String(secs).padStart(2, '0')}`
	}

	/**
	 * Format file size in bytes to human-readable format
	 * @param {number} bytes - File size in bytes
	 * @returns {string} Formatted file size (e.g., "2.5 GB")
	 * @example
	 * formatFileSize(1024) // "1.00 KB"
	 * formatFileSize(2500000000) // "2.33 GB"
	 */
	const formatFileSize = (bytes) => {
		if (!bytes) return '0 B'
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
		const i = Math.floor(Math.log(bytes) / Math.log(1024))
		return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${sizes[i]}`
	}

	/**
	 * Format date to localized date string
	 * @param {string|Date} date - Date to format
	 * @returns {string} Formatted date string
	 * @example
	 * formatDate(new Date()) // "12/30/2025"
	 */
	const formatDate = (date) => {
		if (!date) return 'N/A'
		return new Date(date).toLocaleDateString()
	}

	/**
	 * Format date to localized date and time string
	 * @param {string|Date} date - Date to format
	 * @returns {string} Formatted date and time string
	 * @example
	 * formatDateTime(new Date()) // "12/30/2025, 3:45:30 PM"
	 */
	const formatDateTime = (date) => {
		if (!date) return 'N/A'
		return new Date(date).toLocaleString()
	}

	/**
	 * Format total duration with hours and minutes
	 * @param {number} seconds - Total duration in seconds
	 * @returns {string} Formatted total duration (e.g., "2h 15m")
	 * @example
	 * formatTotalDuration(8100) // "2h 15m"
	 * formatTotalDuration(45) // "0h 0m"
	 */
	const formatTotalDuration = (seconds) => {
		if (!seconds) return '0h 0m'
		const hours = Math.floor(seconds / 3600)
		const minutes = Math.floor((seconds % 3600) / 60)
		return `${hours}h ${minutes}m`
	}

	/**
	 * Format number with thousands separators
	 * @param {number} num - Number to format
	 * @returns {string} Formatted number with commas
	 * @example
	 * formatNumber(1234567) // "1,234,567"
	 */
	const formatNumber = (num) => {
		if (num === null || num === undefined) return '0'
		return num.toLocaleString()
	}

	/**
	 * Format percentage
	 * @param {number} value - Value to format as percentage
	 * @param {number} total - Total value for calculating percentage
	 * @param {number} decimals - Number of decimal places (default: 1)
	 * @returns {string} Formatted percentage
	 * @example
	 * formatPercentage(25, 100) // "25.0%"
	 * formatPercentage(1, 3, 2) // "33.33%"
	 */
	const formatPercentage = (value, total, decimals = 1) => {
		if (!total) return '0%'
		const percentage = (value / total) * 100
		return `${percentage.toFixed(decimals)}%`
	}

	return {
		formatDuration,
		formatFileSize,
		formatDate,
		formatDateTime,
		formatTotalDuration,
		formatNumber,
		formatPercentage,
	}
}
