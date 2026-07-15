// 通用 HTTP 请求封装
// 前端直连第三方 API 会被 WebView 的 CORS 策略拦截，统一通过后端代理发起请求
import * as AppBindings from '../../wailsjs/go/main/App'

export interface HttpRequestOptions {
  method?: string
  headers?: Record<string, string>
  body?: string
}

// 通过后端发起 HTTP 请求，返回响应体字符串
export async function HttpRequest(url: string, options: HttpRequestOptions = {}): Promise<string> {
  return await AppBindings.HttpRequest(
    url,
    options.method || 'GET',
    options.headers || {},
    options.body || ''
  )
}
