/**
 * useAIExport Composable
 *
 * Shared export functionality for AI features
 * Provides CSV and JSON export capabilities with proper formatting
 *
 * Usage:
 * const { exportToCSV, exportToJSON } = useAIExport(toast)
 * exportToCSV(data, 'filename')
 */

import { getCurrentInstance } from 'vue'

export function useAIExport(toastInstance = null) {
	// Get toast instance if not provided
	let toast = toastInstance
	if (!toast) {
		const instance = getCurrentInstance()
		if (instance) {
			toast = instance.proxy.$toast
		}
	}

	/**
	 * Download a file to the user's computer
	 * @param {string} content - File content
	 * @param {string} filename - Name of the file
	 * @param {string} contentType - MIME type
	 */
	const downloadFile = (content, filename, contentType) => {
		const blob = new Blob([content], { type: contentType })
		const url = URL.createObjectURL(blob)
		const link = document.createElement('a')
		link.href = url
		link.download = filename
		link.click()
		URL.revokeObjectURL(url)
	}

	/**
	 * Export data to CSV format
	 * @param {Array} data - Array of objects to export
	 * @param {string} filename - Base filename (without extension)
	 */
	const exportToCSV = (data, filename) => {
		if (!data || data.length === 0) {
			if (toast) {
				toast.warning('No Data', 'No data available to export')
			}
			return
		}

		try {
			// Get headers from first object
			const headers = Object.keys(data[0])
			const csvContent =
				headers.join(',') +
				'\n' +
				data
					.map((row) =>
						headers
							.map((header) => {
								const value = row[header]
								// Handle nested objects and arrays
								if (typeof value === 'object' && value !== null) {
									return '"' + JSON.stringify(value).replace(/"/g, '""') + '"'
								}
								// Escape quotes and wrap in quotes if contains comma
								const stringValue = String(value || '')
								if (stringValue.includes(',') || stringValue.includes('"') || stringValue.includes('\n')) {
									return '"' + stringValue.replace(/"/g, '""') + '"'
								}
								return stringValue
							})
							.join(',')
					)
					.join('\n')

			downloadFile(csvContent, filename + '.csv', 'text/csv')

			if (toast) {
				toast.success('Export Complete', `Exported ${data.length} items to ${filename}.csv`)
			}
		} catch (error) {
			console.error('CSV export failed:', error)
			if (toast) {
				toast.error('Export Failed', 'Could not export data to CSV')
			}
		}
	}

	/**
	 * Export data to JSON format
	 * @param {Array} data - Array of objects to export
	 * @param {string} filename - Base filename (without extension)
	 */
	const exportToJSON = (data, filename) => {
		if (!data || data.length === 0) {
			if (toast) {
				toast.warning('No Data', 'No data available to export')
			}
			return
		}

		try {
			const jsonContent = JSON.stringify(data, null, 2)
			downloadFile(jsonContent, filename + '.json', 'application/json')

			if (toast) {
				toast.success('Export Complete', `Exported ${data.length} items to ${filename}.json`)
			}
		} catch (error) {
			console.error('JSON export failed:', error)
			if (toast) {
				toast.error('Export Failed', 'Could not export data to JSON')
			}
		}
	}

	return {
		exportToCSV,
		exportToJSON,
	}
}
