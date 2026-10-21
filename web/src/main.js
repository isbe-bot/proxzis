import { createApp } from 'vue'
import { createWebHistory } from 'vue-router'
import '@fontsource-variable/archivo/wdth.css'
import '@fontsource-variable/jetbrains-mono'
import './styles/reset.css'
import './styles/tokens.css'
import './styles/base.css'
import App from './App.vue'
import { createAppRouter } from './router.js'

createApp(App).use(createAppRouter(createWebHistory())).mount('#app')
