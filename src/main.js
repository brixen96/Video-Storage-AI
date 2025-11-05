import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import FontAwesomeIcon from './plugins/fontawesome'

// Import Bootstrap JS for interactive components
import 'bootstrap/dist/js/bootstrap.bundle'

const app = createApp(App)

// Register global components
app.component('font-awesome-icon', FontAwesomeIcon)

// Use plugins
app.use(router)

// Mount app
app.mount('#app')
