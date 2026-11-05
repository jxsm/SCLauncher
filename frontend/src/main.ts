import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import i18n from './locales'
import './style.css'

// 前端日志收集（console/onerror/promise 拦截上报后端启动器日志），越早安装越好
import { installFrontendLogger, logFrontendError, logFrontendInfo } from './utils/feLogger'
installFrontendLogger()

// Naive UI
import naive from 'naive-ui'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.use(i18n)
app.use(naive)

// Vue 组件内部错误统一上报（Vue 默认会吞掉 handler 外的错误详情）
app.config.errorHandler = (err, _instance, info) => {
  logFrontendError(`Vue 错误[${info}]: ${err instanceof Error ? err.message : String(err)}`)
}

app.mount('#app')

logFrontendInfo('前端已启动')
