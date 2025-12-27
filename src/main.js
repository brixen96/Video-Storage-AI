
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import FontAwesomeIcon from './plugins/fontawesome'
import ToastNotification from './components/ToastNotification.vue'
// Import Bootstrap JS for interactive components
import 'bootstrap/dist/js/bootstrap.bundle'

const app = createApp(App)

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
