class WebSocketService {
	constructor() {
		this.ws = null
		this.reconnectAttempts = 0
		this.maxReconnectAttempts = 5
		this.reconnectDelay = 3000
		this.listeners = new Map()
		this.isConnecting = false
		this.shouldReconnect = true
	}

	connect() {
		if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
			return
		}

		if (this.isConnecting) {
			return
		}

		this.isConnecting = true
		const wsUrl = `ws://localhost:8080/api/v1/ws`

		try {
			this.ws = new WebSocket(wsUrl)

			this.ws.onopen = () => {
				this.reconnectAttempts = 0
				this.isConnecting = false
				this.notifyListeners('connected', { connected: true })
			}

			this.ws.onmessage = (event) => {
				try {
					const message = JSON.parse(event.data)
					this.handleMessage(message)
				} catch (error) {
					// Silently handle parse errors
				}
			}

			this.ws.onerror = () => {
				this.isConnecting = false
			}

			this.ws.onclose = (event) => {
				this.isConnecting = false
				this.notifyListeners('connected', { connected: false })

				// Attempt to reconnect if not a normal closure and we should reconnect
				if (this.shouldReconnect && event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
					this.reconnectAttempts++
					setTimeout(() => this.connect(), this.reconnectDelay)
				} else if (this.reconnectAttempts >= this.maxReconnectAttempts) {
					this.notifyListeners('error', { message: 'Failed to connect to server' })
				}
			}
		} catch (error) {
			this.isConnecting = false
		}
	}

	disconnect() {
		this.shouldReconnect = false
		if (this.ws) {
			this.ws.close(1000, 'Client disconnect')
			this.ws = null
		}
	}

	handleMessage(message) {
		const { type, data } = message

		switch (type) {
			case 'activity_update':
				this.notifyListeners('activity_update', data)
				break
			case 'status_update':
				this.notifyListeners('status_update', data)
				break
			case 'console_log':
				this.notifyListeners('console_log', data)
				break
			default:
				// Silently ignore unknown message types
		}
	}

	// Subscribe to WebSocket events
	on(event, callback) {
		if (!this.listeners.has(event)) {
			this.listeners.set(event, [])
		}
		this.listeners.get(event).push(callback)

		// Return unsubscribe function
		return () => {
			const callbacks = this.listeners.get(event)
			if (callbacks) {
				const index = callbacks.indexOf(callback)
				if (index > -1) {
					callbacks.splice(index, 1)
				}
			}
		}
	}

	// Remove all listeners for an event
	off(event) {
		this.listeners.delete(event)
	}

	// Notify all listeners for an event
	notifyListeners(event, data) {
		const callbacks = this.listeners.get(event)
		if (callbacks) {
			callbacks.forEach((callback) => {
				try {
					callback(data)
				} catch (error) {
					// Silently handle listener errors
				}
			})
		}
	}

	// Check if connected
	isConnected() {
		return this.ws && this.ws.readyState === WebSocket.OPEN
	}
}

// Create singleton instance
const websocketService = new WebSocketService()

export default websocketService
