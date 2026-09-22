import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { Icon } from '@iconify/vue'

import App from './App.vue'
import router from './router'
import HelpButton from './components/HelpButton.vue'

const app = createApp(App)
app.component('IconVue', Icon)
app.component('HelpButton', HelpButton)

app.use(createPinia())
app.use(router)

app.mount('#app')
