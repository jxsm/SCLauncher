// 前端日志收集：拦截 console 警告/错误、window.onerror、未处理的 Promise 拒绝，
// 防抖批量上报到后端，由后端写入 .Survivalcraft/Logs 下的启动器日志（按天分隔，保留 7 天）。
// 收集器自身绝不抛错、不改变 console 原有行为；Wails 环境不可用时静默丢弃。

type Level = 'debug' | 'info' | 'warn' | 'error'

interface Entry {
  level: Level
  message: string
}

// 队列与批量上限：防止错误风暴（如组件循环报错）刷爆 IPC 与日志文件
const QUEUE_LIMIT = 200
const BATCH_LIMIT = 50
const FLUSH_INTERVAL = 500
// 单条消息最大长度，超长截断（如完整堆栈）
const MAX_MESSAGE_LENGTH = 2000

let installed = false
let queue: Entry[] = []
let timer: ReturnType<typeof setTimeout> | null = null
// 连续相同消息去重计数（Vue/naive-ui 的重复警告很常见）
let lastMessage = ''
let repeatCount = 0

function formatValue(value: unknown): string {
  if (value instanceof Error) {
    // 只保留首行堆栈位置，完整堆栈太长
    const location = value.stack?.split('\n')[1]?.trim()
    return location ? `${value.message} (${location})` : value.message
  }
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function formatArgs(args: unknown[]): string {
  return args.map(formatValue).join(' ')
}

function truncate(message: string): string {
  return message.length > MAX_MESSAGE_LENGTH
    ? message.slice(0, MAX_MESSAGE_LENGTH) + '…(截断)'
    : message
}

function enqueue(level: Level, message: string): void {
  try {
    const msg = truncate(message)
    // 连续相同消息只记一条，标注重复次数
    if (msg === lastMessage) {
      repeatCount++
      return
    }
    flushRepeatCount()
    lastMessage = msg
    repeatCount = 0
    queue.push({ level, message: msg })
    if (queue.length > QUEUE_LIMIT) {
      queue.splice(0, queue.length - QUEUE_LIMIT)
    }
    scheduleFlush()
  } catch {
    // 日志收集失败必须静默
  }
}

function flushRepeatCount(): void {
  if (repeatCount > 1 && lastMessage) {
    queue.push({ level: 'info', message: `(上一条消息重复 ${repeatCount} 次)` })
  }
  repeatCount = 0
}

function scheduleFlush(): void {
  if (timer !== null) return
  timer = setTimeout(() => {
    timer = null
    flush()
  }, FLUSH_INTERVAL)
}

function flush(): void {
  if (queue.length === 0) return
  const app = (window as unknown as { go?: { main?: { App?: Record<string, (level: string, message: string) => Promise<void>> } } }).go?.main?.App
  if (!app?.WriteFrontendLog) return // 非 Wails 环境（纯浏览器开发）不上报

  flushRepeatCount()
  const batch = queue.splice(0, BATCH_LIMIT)
  for (const entry of batch) {
    try {
      app.WriteFrontendLog(entry.level, entry.message).catch(() => {})
    } catch {
      // 上报失败静默
    }
  }
  if (queue.length > 0) scheduleFlush() // 剩余的下一轮再发
}

// installFrontendLogger 安装全局拦截钩子（幂等，main.ts 最早调用）
export function installFrontendLogger(): void {
  if (installed || typeof window === 'undefined') return
  installed = true

  // 拦截 console.warn / console.error：保留原行为，镜像上报
  const originalWarn = console.warn.bind(console)
  const originalError = console.error.bind(console)
  console.warn = (...args: unknown[]) => {
    enqueue('warn', formatArgs(args))
    originalWarn(...args)
  }
  console.error = (...args: unknown[]) => {
    enqueue('error', formatArgs(args))
    originalError(...args)
  }

  // 未捕获的同步错误
  window.addEventListener('error', (event) => {
    if (event.message) {
      enqueue('error', `未捕获错误: ${event.message} @ ${event.filename}:${event.lineno}:${event.colno}`)
    }
  })

  // 未处理的 Promise 拒绝
  window.addEventListener('unhandledrejection', (event) => {
    enqueue('error', `未处理的 Promise 拒绝: ${formatValue(event.reason)}`)
  })
}

// logFrontendError 供 Vue errorHandler 等主动上报
export function logFrontendError(message: string): void {
  enqueue('error', message)
}

// logFrontendInfo 供关键流程打点（页面加载、路由切换等）
export function logFrontendInfo(message: string): void {
  enqueue('info', message)
}
