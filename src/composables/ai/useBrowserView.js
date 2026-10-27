import { ref, onMounted, onUnmounted } from 'vue'
import { browserAPI } from '@/services/api'
import websocketService from '@/services/websocket'

// Live view of what the Companion sees: bridge connection state, open tabs and
// screenshot frames of the active Firefox tab. Frames are captured on demand
// (fresh screenshot on tab change + periodic refresh while the Browser tab is open).
export function useBrowserView() {
	const connected = ref(false)
	const activeTab = ref(null)
	const tabs = ref([])
	const frameUrl = ref('')
	const frameAt = ref('')
	const capturing = ref(false)
	const statusLoaded = ref(false)

	let liveTimer = null
	let pendingCapture = null

	const stopCapture = websocketService.on('browser_state', (state) => {
		connected.value = !!state?.connected
		activeTab.value = state?.active || null
		tabs.value = state?.tabs || []
		scheduleCapture()
	})

	// Re-sync status when the websocket itself (re)connects
	const offConnected = websocketService.on('connected', ({ connected: up }) => {
		if (up) loadStatus()
	})

	async function loadStatus() {
		try {
			const res = await browserAPI.getStatus()
			const st = res?.data
			if (!st) return
			connected.value = !!st.connected
			activeTab.value = st.state?.active || null
			tabs.value = st.state?.tabs || []
			statusLoaded.value = true
			capture()
		} catch {
			// backend down - keep whatever state we already have
		}
	}

	async function capture() {
		if (capturing.value || !connected.value) return
		capturing.value = true
		try {
			const res = await browserAPI.captureScreenshot()
			if (res?.data?.image) {
				frameUrl.value = res.data.image
				frameAt.value = res.data.captured_at || ''
			}
		} catch {
			// privileged page (about:*) or bridge hiccup - keep the last good frame
		} finally {
			capturing.value = false
		}
	}

	function scheduleCapture() {
		if (pendingCapture) clearTimeout(pendingCapture)
		pendingCapture = setTimeout(() => {
			pendingCapture = null
			capture()
		}, 600)
	}

	function startLive() {
		if (liveTimer) return
		capture()
		liveTimer = setInterval(capture, 3500)
	}

	function stopLive() {
		if (liveTimer) {
			clearInterval(liveTimer)
			liveTimer = null
		}
		if (pendingCapture) {
			clearTimeout(pendingCapture)
			pendingCapture = null
		}
	}

	onMounted(() => {
		loadStatus()
	})

	onUnmounted(() => {
		stopLive()
		stopCapture?.()
		offConnected?.()
	})

	return {
		connected,
		activeTab,
		tabs,
		frameUrl,
		frameAt,
		capturing,
		statusLoaded,
		loadStatus,
		capture,
		startLive,
		stopLive,
	}
}
