/**
 * useVoiceInput — Web Speech API integration for voice-to-text.
 */
import { ref, onUnmounted } from 'vue'

export function useVoiceInput({ onResult }) {
	const isListening = ref(false)
	const recognition = ref(null)

	const initSpeechRecognition = () => {
		if ('webkitSpeechRecognition' in window || 'SpeechRecognition' in window) {
			const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition
			recognition.value = new SpeechRecognition()
			recognition.value.continuous = false
			recognition.value.interimResults = true
			recognition.value.lang = 'en-US'

			recognition.value.onstart = () => {
				isListening.value = true
			}

			recognition.value.onresult = (event) => {
				const transcript = Array.from(event.results)
					.map((result) => result[0])
					.map((result) => result.transcript)
					.join('')

				if (event.results[0].isFinal) {
					onResult?.(transcript, true)
				} else {
					onResult?.(transcript, false)
				}
			}

			recognition.value.onerror = () => {
				isListening.value = false
			}

			recognition.value.onend = () => {
				isListening.value = false
			}
		}
	}

	const toggleVoiceInput = () => {
		if (!recognition.value) {
			initSpeechRecognition()
		}
		if (!recognition.value) return false // not supported

		if (isListening.value) {
			recognition.value.stop()
		} else {
			recognition.value.start()
		}
		return true
	}

	onUnmounted(() => {
		if (recognition.value && isListening.value) {
			recognition.value.stop()
		}
	})

	return {
		isListening,
		toggleVoiceInput,
	}
}
