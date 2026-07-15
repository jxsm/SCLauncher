/**
 * 模组下载源类型定义
 */

// 下载源类型
export type SourceType = 'mods' | 'online-mods' | 'savegames' | 'furniture' | 'textures' | 'skins'

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

// 模组详情
export interface ModDetails extends ModSearchResult {
  fullDescription: string
  publishDate: string
  updateDate: string
  tags: string[]
  screenshots: string[]
}

// 下载源配置
export interface ModSource {
  id: string
  type: SourceType
  name: string
  description: string
  icon?: string
  enabled: boolean
  isDefault?: boolean
  api: ModSourceAPI
  // 按包名查询接口（用于依赖解析）。URL 含 {packageName} 与 {versionRange} 占位符
  packageLookup?: { url: string } | null
}

// 下载源API配置
export interface ModSourceAPI {
  baseUrl: string
  searchPath: string
  searchParams?: Record<string, any>
  headers?: Record<string, string>
  responseMapping: ResponseMapping
}

// 响应映射配置
export interface ResponseMapping {
  results: string // JSONPath to results array
  id: string
  title: string
  description: string
  author: string
  authorAvatar?: string
  views?: string
  likes?: string
  cover?: string
  versions: string
  version: string
  downloadUrl: string
  fileName: string
  fileSize: string
  icon?: string
}

// 搜索选项
export interface SearchOptions {
  page?: number
  limit?: number
  filters?: Record<string, any>
}

// 分页响应数据
export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  limit: number
  totalPages: number
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
