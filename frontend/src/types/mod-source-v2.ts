/**
 * 新的模组下载源类型定义
 * 支持更灵活的API配置，包括GET/POST请求、URL参数替换、自定义请求体等
 */

// 下载源类型
export type SourceType = 'mods' | 'savegames' | 'furniture' | 'textures' | 'skins'

// HTTP 方法类型
export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'

// 请求体类型
export type BodyType = 'json' | 'form-data' | 'form-urlencoded' | 'raw'

// API 端点配置
export interface ApiEndpointConfig {
  // HTTP 方法：GET, POST 等
  method: HttpMethod

  // URL 路径，支持替换符，如：/api/mods?page={page}&limit={limit}
  // 对于 GET 请求：替换符会被实际值替换
  // 对于 POST 请求：替换符在 URL 中保持不变，实际值通过请求体传递
  url: string

  // 请求头配置
  headers?: Record<string, string>

  // POST 请求的请求体配置（仅用于 POST/PUT/PATCH 请求）
  body?: {
    // 请求体类型
    type: BodyType

    // 请求体模板，支持替换符
    // 对于 json 类型：可以是对象或字符串
    // 对于 form-data/form-urlencoded 类型：键值对对象
    // 对于 raw 类型：字符串
    template?: any

    // 对于 form-data 类型，可以配置文件字段
    fileFields?: string[]
  }

  // 分页参数配置
  pagination?: {
    // 页码参数名
    pageParam?: string
    // 每页数量参数名
    limitParam?: string
    // 搜索关键词参数名
    searchParam?: string
    // 分页信息传递位置：'url' 或 'body'
    paramLocation?: 'url' | 'body'
  }
}

// 模组搜索结果
export interface ModSearchResult {
  id: string
  title: string
  description: string
  author: string
  authorAvatar?: string
  views: number
  likes: number
  cover?: string
  icon?: string
  versions: ModVersion[]
  sourceId: string // 来自哪个下载源
}

// 模组版本
export interface ModVersion {
  version: string
  downloadUrl: string
  fileName: string
  fileSize: string
  icon?: string
}

// 响应映射配置 - 使用 JSONPath 表达式
export interface ResponseMapping {
  // 结果列表的 JSONPath
  results: string

  // 基础字段映射
  id: string
  title: string
  description: string
  author: string
  authorAvatar?: string
  views?: string
  likes?: string
  cover?: string
  icon?: string

  // 版本相关映射
  versions: string // 版本列表的 JSONPath
  version: string // 版本号字段
  downloadUrl: string
  fileName: string
  fileSize: string

  // 分页信息映射（可选）
  total?: string // 总数的 JSONPath
  totalPages?: string // 总页数的 JSONPath
  currentPage?: string // 当前页的 JSONPath
}

// 分页响应数据
export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  limit: number
  totalPages: number
}

// 下载源配置（新版本）
export interface ModSourceV2 {
  id: string
  type: SourceType
  name: string
  description: string
  icon?: string
  enabled: boolean
  isDefault?: boolean

  // API 配置
  api: {
    // 基础 URL
    baseUrl: string

    // 列表接口配置（获取模组列表）
    list?: ApiEndpointConfig

    // 搜索接口配置（搜索模组）
    search?: ApiEndpointConfig

    // 如果 list 和 search 相同，可以只配置 endpoint
    endpoint?: ApiEndpointConfig

    // 响应数据映射配置
    responseMapping: ResponseMapping

    // 可选：请求超时时间（毫秒）
    timeout?: number
  }

  // 可选：其他元数据
  metadata?: {
    website?: string
    author?: string
    version?: string
    tags?: string[]
  }

  // 按包名查询接口（用于依赖解析）。URL 含 {packageName} 与 {versionRange} 占位符
  packageLookup?: { url: string } | null
}

// 搜索选项
export interface SearchOptions {
  page?: number
  limit?: number
  query?: string
  filters?: Record<string, any>
}

// 请求上下文 - 用于替换符解析
export interface RequestContext {
  page: number
  limit: number
  query?: string
  filters?: Record<string, any>
}

// 下载进度
export interface DownloadProgress {
  taskId: string
  modId: string
  fileName: string
  total: number
  downloaded: number
  speed: number
  status: 'downloading' | 'completed' | 'failed' | 'cancelled'
  error?: string
}
