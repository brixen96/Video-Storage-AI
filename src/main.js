
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import FontAwesomeIcon from './plugins/fontawesome'
import { consoleLogAPI } from './services/api'
// Import design tokens (must be first CSS import)
import './styles/design-tokens.css'
// Import Bootstrap JS for interactive components
import 'bootstrap/dist/js/bootstrap.bundle'

// Setup frontend console logging to API
let isLogging = false // Prevent feedback loop
const sendConsoleLog = (level, message, details = {}) => {
    if (isLogging) return // Don't send logs about failed log sends
    try {
        isLogging = true
        consoleLogAPI.create({
            source: 'frontend',
            level: level,
            message: message,
            details: details
        }).catch(() => {
            // Silently fail — do NOT log the failure (prevents feedback loop)
        })
    } catch (err) {
        // Ignore errors in logging system
    } finally {
        isLogging = false
    }
}

// Override console methods to capture logs
const originalConsoleError = console.error
const originalConsoleWarn = console.warn

console.error = (...args) => {
    originalConsoleError(...args)
    // Don't send console errors that are about the logging system itself
    const msg = args.map(arg => typeof arg === 'object' ? JSON.stringify(arg) : String(arg)).join(' ')
    if (msg.includes('console log') || msg.includes('sendConsoleLog')) return
    // Debounce: only send one error per second to prevent floods
    if (!console.error._lastSend || Date.now() - console.error._lastSend > 1000) {
        console.error._lastSend = Date.now()
        sendConsoleLog('error', msg.slice(0, 500))
    }
}

console.warn = (...args) => {
    originalConsoleWarn(...args)
    // Don't send console warnings about the logging system itself
    const msg = args.map(arg => typeof arg === 'object' ? JSON.stringify(arg) : String(arg)).join(' ')
    if (msg.includes('console log') || msg.includes('sendConsoleLog')) return
    // Debounce: only send one warning per 5 seconds
    if (!console.warn._lastSend || Date.now() - console.warn._lastSend > 5000) {
        console.warn._lastSend = Date.now()
        sendConsoleLog('warning', msg.slice(0, 500))
    }
}

// Don't override console.info to avoid spam - only capture errors and warnings

const app = createApp(App)

// Setup global error handler for Vue errors
app.config.errorHandler = (err, instance, info) => {
    console.error('Vue error:', err, info)
    sendConsoleLog('error', `Vue error: ${err.message}`, {
        error: err.toString(),
        info: info,
        stack: err.stack
    })
}

// Register global components
app.component('font-awesome-icon', FontAwesomeIcon)

// Use plugins
app.use(router)
app.use(store)

// Mount app and wire up $toast via the App component's ref
const appInstance = app.mount('#app')
app.config.globalProperties.$toast = appInstance.$refs.toast
