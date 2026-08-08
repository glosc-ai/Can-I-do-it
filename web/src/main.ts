import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import ElementPlusMessage from 'element-plus-message'

import './assets/index.css'
import 'element-plus-message/dist/index.css'
import 'element-plus-message/theme-chalk/dark/css-vars.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(ElementPlusMessage)
app.mount('#app')
