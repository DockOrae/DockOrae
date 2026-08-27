import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import i18n from './i18n'
import { initTheme } from './store'
import './styles/main.css'

initTheme()

createApp(App).use(router).use(i18n).mount('#app')
