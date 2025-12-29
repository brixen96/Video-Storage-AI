
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import FontAwesomeIcon from './plugins/fontawesome'
import ToastNotification from './components/ToastNotification.vue'
import { consoleLogAPI } from './services/api'
// Import Bootstrap JS for interactive components
import 'bootstrap/dist/js/bootstrap.bundle'

// Setup frontend console logging to API
const sendConsoleLog = (level, message, details = {}) => {
    try {
        consoleLogAPI.create({
            source: 'frontend',
            level: level,
            message: message,
            details: details
        }).catch(err => {
            // Silently fail if console log API is unavailable
            console.warn('Failed to send console log to API:', err)
        })
    } catch (err) {
        // Ignore errors in logging system
    }
}

// Override console methods to capture logs
const originalConsoleError = console.error
const originalConsoleWarn = console.warn

console.error = (...args) => {
    originalConsoleError(...args)
    const message = args.map(arg => typeof arg === 'object' ? JSON.stringify(arg) : String(arg)).join(' ')
    sendConsoleLog('error', message, { args: args.map(a => String(a)) })
}

console.warn = (...args) => {
    originalConsoleWarn(...args)
    const message = args.map(arg => typeof arg === 'object' ? JSON.stringify(arg) : String(arg)).join(' ')
    sendConsoleLog('warning', message, { args: args.map(a => String(a)) })
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
app.component('ToastNotification', ToastNotification)

// Create and mount toast instance
const toastApp = createApp(ToastNotification)
toastApp.component('font-awesome-icon', FontAwesomeIcon) // Register font-awesome-icon in toast app
const toastContainer = document.createElement('div')
document.body.appendChild(toastContainer)
const toastInstance = toastApp.mount(toastContainer)

// Make toast globally available
app.config.globalProperties.$toast = toastInstance

// Use plugins
app.use(router)
app.use(store)

// Mount app
app.mount('#app')
