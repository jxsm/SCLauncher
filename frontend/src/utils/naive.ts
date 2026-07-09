import { createDiscreteApi } from 'naive-ui'

// 在组件外（如 Pinia store / 工具函数）使用 Naive UI 的 dialog / message。
// createDiscreteApi 会创建独立的服务实例，不依赖组件内的 provider 注入。
const { dialog, message } = createDiscreteApi(['dialog', 'message'])

export { dialog, message }
