import { createApp } from 'vue'
import App from './App.vue'
import { createAppRouter } from './router'
import i18n from './i18n'
import { initTheme } from './store'
import './styles/main.css'

initTheme()

// router 需要先获取安全入口(webBasePath)确定 base,再挂载应用
createAppRouter().then((router) => {
  createApp(App).use(router).use(i18n).mount('#app')
})
