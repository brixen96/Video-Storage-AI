// Settings Service - Centralized settings management with localStorage persistence

const SETTINGS_KEY = 'videoStorageAI_settings'

// Default settings
const DEFAULT_SETTINGS = {
	// General
	defaultViewMode: 'grid',
	theme: 'dark',
	itemsPerPage: 30,

	// Library
	videoLibraryPath: '',
	autoScanOnStartup: false,
	watchForChanges: true,
	videoFormats: 'mp4,mkv,avi,mov,wmv,flv,webm',

	// Display
	thumbnailQuality: 'medium',
	gridCardSize: 'medium',
	showFileExtensions: false,
	animatedPreviews: true,

	// Privacy
	defaultZooFilter: 'all',
	blurThumbnails: false,
	showMetadata: true,
}

class SettingsService {
	constructor() {
		this.settings = this.loadSettings()
		this.listeners = []
	}

	/**
	 * Load settings from localStorage
	 */
	loadSettings() {
		try {
			const saved = localStorage.getItem(SETTINGS_KEY)
			if (saved) {
				const parsed = JSON.parse(saved)
				return { ...DEFAULT_SETTINGS, ...parsed }
			}
		} catch (e) {
			console.error('Failed to load settings:', e)
		}
		return { ...DEFAULT_SETTINGS }
	}

	/**
	 * Save settings to localStorage
	 */
	saveSettings(newSettings) {
		try {
			this.settings = { ...this.settings, ...newSettings }
			localStorage.setItem(SETTINGS_KEY, JSON.stringify(this.settings))
			this.notifyListeners(this.settings)
			return true
		} catch (e) {
			console.error('Failed to save settings:', e)
			return false
		}
	}

	/**
	 * Get all settings
	 */
	getSettings() {
		return { ...this.settings }
	}

	/**
	 * Get a specific setting
	 */
	getSetting(key) {
		return this.settings[key]
	}

	/**
	 * Update a specific setting
	 */
	updateSetting(key, value) {
		return this.saveSettings({ [key]: value })
	}

	/**
	 * Reset to default settings
	 */
	resetSettings() {
		this.settings = { ...DEFAULT_SETTINGS }
		localStorage.removeItem(SETTINGS_KEY)
		this.notifyListeners(this.settings)
		return true
	}

	/**
	 * Subscribe to settings changes
	 */
	subscribe(listener) {
		this.listeners.push(listener)
		// Return unsubscribe function
		return () => {
			this.listeners = this.listeners.filter((l) => l !== listener)
		}
	}

	/**
	 * Notify all listeners of settings changes
	 */
	notifyListeners(settings) {
		this.listeners.forEach((listener) => {
			try {
				listener(settings)
			} catch (e) {
				console.error('Settings listener error:', e)
			}
		})
	}

	/**
	 * Apply theme setting
	 */
	applyTheme() {
		const theme = this.settings.theme
		const root = document.documentElement

		if (theme === 'dark') {
			root.classList.add('dark-theme')
			root.classList.remove('light-theme')
		} else if (theme === 'light') {
			root.classList.add('light-theme')
			root.classList.remove('dark-theme')
		} else if (theme === 'auto') {
			// Use system preference
			const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
			if (prefersDark) {
				root.classList.add('dark-theme')
				root.classList.remove('light-theme')
			} else {
				root.classList.add('light-theme')
				root.classList.remove('dark-theme')
			}
		}
	}

	/**
	 * Get supported video formats as array
	 */
	getVideoFormatsArray() {
		return this.settings.videoFormats.split(',').map((f) => f.trim().toLowerCase())
	}

	/**
	 * Check if file extension is supported
	 */
	isSupportedFormat(filename) {
		const ext = filename.split('.').pop().toLowerCase()
		return this.getVideoFormatsArray().includes(ext)
	}
}

// Create singleton instance
const settingsService = new SettingsService()

// Apply theme on initialization
settingsService.applyTheme()

// Listen for system theme changes when in auto mode
if (window.matchMedia) {
	window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
		if (settingsService.getSetting('theme') === 'auto') {
			settingsService.applyTheme()
		}
	})
}

export default settingsService
