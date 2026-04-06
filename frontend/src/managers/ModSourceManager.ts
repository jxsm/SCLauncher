/**
 * 模组下载源管理器
 * 负责管理多个模组下载源，提供统一的搜索和下载接口
 */

import { ref } from 'vue'
import type { ModSource, ModSearchResult, SearchOptions, PaginatedResponse } from '../types/mod-source'

class ModSourceManagerClass {
  // 状态
  private sources = ref<ModSource[]>([])
  private currentSourceId = ref<string>('')
  private loading = ref(false)

  constructor() {
    this.loadSources()
  }

  /**
   * 重新加载所有下载源配置
   */
  async reloadSources(): Promise<void> {
    // 清空当前源列表
    this.sources.value = []
    this.currentSourceId.value = ''
    // 重新加载
    await this.loadSources()
  }

  /**
   * 加载所有下载源配置
   */
  private async loadSources() {
    try {
      // TODO: 从启动器目录的 mod-sources 文件夹加载配置
      // 暂时使用内置的默认源（不设置为默认，等加载完所有源后再判断）
      this.sources.value = [
        {
          id: 'suancaixianyu',
          type: 'mods',
          name: '生存战争中文社区',
          description: '生存战争中文社区模组仓库',
          icon: '',
          enabled: true,
          isDefault: false, // 初始不设置为默认
          api: {
            baseUrl: 'https://m.suancaixianyu.cn',
            searchPath: '/api/post/list',
            searchParams: {
              type: 2,
              fileTypes: 5
            },
            responseMapping: {
              results: '$.data.list',
              id: '$.id',
              title: '$.title',
              description: '$.content',
              author: '$.creator.nickname',
              authorAvatar: '$.creator.headImg',
              views: '$.views',
              likes: '$.likeCount',
              cover: '$.cover',
              versions: '$.postVersions',
              version: '$.version',
              downloadUrl: '$.files[0].url',
              fileName: '$.files[0].filename',
              fileSize: '$.files[0].size',
              icon: '$.files[0].icon'
            }
          }
        }
      ]

      // 尝试从启动器目录加载自定义下载源
      try {
        const { GetModSources } = await import('../api/config')
        const customSources = await GetModSources()

        if (customSources && Array.isArray(customSources)) {
          console.log('加载到自定义下载源:', customSources)
          // 合并内置源和自定义源（自定义源优先）
          const existingIds = this.sources.value.map(s => s.id)
          customSources.forEach((source: any) => {
            if (!existingIds.includes(source.id)) {
              // 如果旧源没有 type 字段，默认为 'mods'
              if (!source.type) {
                console.log(`源 ${source.id} 缺少 type 字段，默认设为 'mods'`)
                source.type = 'mods'
              }
              this.sources.value.push(source)
              console.log('添加自定义源:', source)
            }
          })
        }
      } catch (error) {
        // 如果加载失败，只使用内置源
        console.log('未找到自定义下载源，使用默认配置')
      }

      // 检查是否有默认源，如果没有则将内置源设为默认
      const hasDefaultSource = this.sources.value.some(s => s.isDefault)
      if (!hasDefaultSource) {
        // 没有默认源，将内置源设为默认
        const builtinSource = this.sources.value.find(s => s.id === 'suancaixianyu')
        if (builtinSource) {
          builtinSource.isDefault = true
          console.log('没有默认源，将内置源设为默认')
        }
      } else {
        // 有默认源（自定义源），确保内置源不是默认源
        const builtinSource = this.sources.value.find(s => s.id === 'suancaixianyu')
        if (builtinSource && builtinSource.isDefault) {
          builtinSource.isDefault = false
          console.log('有自定义默认源，取消内置源的默认状态')
        }
      }

      // 设置当前默认源
      const defaultSource = this.sources.value.find(s => s.isDefault)
      if (defaultSource) {
        this.currentSourceId.value = defaultSource.id
      }
    } catch (error) {
      console.error('Failed to load mod sources:', error)
    }
  }

  /**
   * 获取所有下载源
   */
  getAllSources(): ModSource[] {
    return this.sources.value
  }

  /**
   * 获取启用的下载源
   */
  getEnabledSources(): ModSource[] {
    return this.sources.value.filter(s => s.enabled)
  }

  /**
   * 获取当前下载源
   */
  getCurrentSource(): ModSource | undefined {
    return this.sources.value.find(s => s.id === this.currentSourceId.value)
  }

  /**
   * 设置当前下载源
   */
  setCurrentSource(sourceId: string): boolean {
    const source = this.sources.value.find(s => s.id === sourceId)
    if (source && source.enabled) {
      this.currentSourceId.value = sourceId
      return true
    }
    return false
  }

  /**
   * 获取模组列表（带分页）
   */
  async getModList(options: SearchOptions = {}): Promise<PaginatedResponse<ModSearchResult>> {
    const currentSource = this.getCurrentSource()
    if (!currentSource) {
      throw new Error('No mod source selected')
    }

    this.loading.value = true

    try {
      const { api } = currentSource

      // 构建请求参数
      const params = new URLSearchParams()
      if (api.searchParams) {
        Object.entries({ ...api.searchParams, ...options.filters }).forEach(([key, value]) => {
          params.append(key, String(value))
        })
      }

      // 添加分页参数
      const page = options.page || 1
      const limit = options.limit || 10
      params.append('page', String(page))
      params.append('limit', String(limit))

      // 发起请求
      const url = `${api.baseUrl}${api.searchPath}?${params.toString()}`
      const response = await fetch(url, {
        headers: api.headers || {}
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const data = await response.json()

      // 解析响应数据
      const mods = this.parseResponseData(data, api.responseMapping, currentSource.id)

      // 从响应中获取分页信息
      const total = Number(this.getValueByPath(data, '$.data.total') || 0)
      const totalPages = Math.ceil(total / limit)

      return {
        data: mods,
        total,
        page,
        limit,
        totalPages
      }
    } catch (error) {
      console.error('Failed to get mod list:', error)
      throw error
    } finally {
      this.loading.value = false
    }
  }

  /**
   * 搜索模组（使用关键词）
   */
  async searchMods(query: string, options: SearchOptions = {}): Promise<PaginatedResponse<ModSearchResult>> {
    const currentSource = this.getCurrentSource()
    if (!currentSource) {
      throw new Error('No mod source selected')
    }

    this.loading.value = true

    try {
      const { api } = currentSource

      // 构建请求参数
      const params = new URLSearchParams()
      if (api.searchParams) {
        Object.entries({ ...api.searchParams, ...options.filters }).forEach(([key, value]) => {
          params.append(key, String(value))
        })
      }

      // 添加分页参数
      const page = options.page || 1
      const limit = options.limit || 10
      params.append('page', String(page))
      params.append('limit', String(limit))

      // 添加搜索关键词（蒜菜闲鱼使用 title 参数）
      if (query.trim()) {
        params.append('title', query.trim())
      }

      // 发起请求
      const url = `${api.baseUrl}${api.searchPath}?${params.toString()}`
      const response = await fetch(url, {
        headers: api.headers || {}
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const data = await response.json()

      // 解析响应数据
      const mods = this.parseResponseData(data, api.responseMapping, currentSource.id)

      // 从响应中获取分页信息
      const total = Number(this.getValueByPath(data, '$.data.total') || 0)
      const totalPages = Math.ceil(total / limit)

      return {
        data: mods,
        total,
        page,
        limit,
        totalPages
      }
    } catch (error) {
      console.error('Failed to search mods:', error)
      throw error
    } finally {
      this.loading.value = false
    }
  }

  /**
   * 解析响应数据
   */
  private parseResponseData(data: any, mapping: any, sourceId: string): ModSearchResult[] {
    try {
      // 使用简单的路径访问代替JSONPath
      const results = this.getValueByPath(data, mapping.results)

      if (!Array.isArray(results)) {
        return []
      }

      return results.map((item: any) => {
        const versions = this.getValueByPath(item, mapping.versions) || []

        return {
          id: String(this.getValueByPath(item, mapping.id) || ''),
          title: String(this.getValueByPath(item, mapping.title) || ''),
          description: this.stripHtmlTags(String(this.getValueByPath(item, mapping.description) || '')),
          author: String(this.getValueByPath(item, mapping.author) || ''),
          authorAvatar: this.getValueByPath(item, mapping.authorAvatar),
          views: Number(this.getValueByPath(item, mapping.views) || 0),
          likes: Number(this.getValueByPath(item, mapping.likes) || 0),
          cover: this.getValueByPath(item, mapping.cover),
          icon: this.getValueByPath(item, mapping.icon),
          sourceId,
          versions: versions.map((v: any) => ({
            version: String(this.getValueByPath(v, mapping.version) || ''),
            downloadUrl: String(this.getValueByPath(v, mapping.downloadUrl) || ''),
            fileName: String(this.getValueByPath(v, mapping.fileName) || ''),
            fileSize: String(this.getValueByPath(v, mapping.fileSize) || ''),
            icon: this.getValueByPath(v, mapping.icon)
          })).filter((v: any) => v.downloadUrl) // 过滤掉没有下载链接的版本
        }
      }).filter((mod: ModSearchResult) => mod.id && mod.title) // 过滤掉无效数据
    } catch (error) {
      console.error('Failed to parse response data:', error)
      return []
    }
  }

  /**
   * 根据路径获取对象值
   * 支持 $.data.list 或 data.list 格式
   */
  private getValueByPath(obj: any, path: string): any {
    if (!path) return obj

    // 移除 $. 前缀
    const cleanPath = path.replace(/^\$\./, '')

    // 分割路径
    const keys = cleanPath.split('.')

    // 遍历路径获取值
    let value = obj
    for (const key of keys) {
      if (value && typeof value === 'object') {
        // 处理数组索引 [0]
        const arrayMatch = key.match(/^(.+)\[(\d+)\]$/)
        if (arrayMatch) {
          const [, arrayKey, index] = arrayMatch
          value = value[arrayKey]?.[parseInt(index)]
        } else {
          value = value[key]
        }
      } else {
        return undefined
      }
    }

    return value
  }

  /**
   * 移除HTML标签
   */
  private stripHtmlTags(html: string): string {
    const tmp = document.createElement('div')
    tmp.innerHTML = html
    return tmp.textContent || tmp.innerText || ''
  }

  /**
   * 添加下载源
   */
  async addSource(source: ModSource): Promise<void> {
    // 检查ID是否已存在
    if (this.sources.value.some(s => s.id === source.id)) {
      throw new Error('Source ID already exists')
    }

    this.sources.value.push(source)

    // 保存到配置文件
    await this.saveSources()
  }

  /**
   * 删除下载源
   */
  async removeSource(sourceId: string): Promise<void> {
    // 不允许删除默认源
    if (this.sources.value.find(s => s.id === sourceId)?.isDefault) {
      throw new Error('Cannot remove default source')
    }

    this.sources.value = this.sources.value.filter(s => s.id !== sourceId)

    // 如果删除的是当前源，切换到默认源
    if (this.currentSourceId.value === sourceId) {
      const defaultSource = this.sources.value.find(s => s.isDefault)
      if (defaultSource) {
        this.currentSourceId.value = defaultSource.id
      }
    }

    // 保存到配置文件
    await this.saveSources()
  }

  /**
   * 启用/禁用下载源
   */
  async toggleSource(sourceId: string, enabled: boolean): Promise<void> {
    const source = this.sources.value.find(s => s.id === sourceId)
    if (source) {
      source.enabled = enabled

      // 如果禁用的是当前源，切换到其他启用的源
      if (!enabled && this.currentSourceId.value === sourceId) {
        const enabledSource = this.sources.value.find(s => s.enabled && s.id !== sourceId)
        if (enabledSource) {
          this.currentSourceId.value = enabledSource.id
        }
      }

      // 保存到配置文件
      await this.saveSources()
    }
  }

  /**
   * 保存下载源配置到文件
   */
  async saveSources(): Promise<void> {
    try {
      const { SaveModSources } = await import('../api/config')

      // 内置源ID列表
      const builtinSourceIds = ['suancaixianyu']

      // 只保存自定义源（排除内置源）
      const customSources = this.sources.value
        .filter(s => !builtinSourceIds.includes(s.id))
        .map(s => ({
          id: s.id,
          type: s.type,
          name: s.name,
          description: s.description,
          icon: s.icon || '',
          enabled: s.enabled,
          isDefault: s.isDefault, // 保存默认源标记
          api: s.api
        }))

      console.log('保存自定义下载源:', customSources)
      await SaveModSources(customSources as any)

      // 注意：不要在这里修改内置源的 isDefault 状态
      // 内置源的默认状态应该由 loadSources() 在加载时动态决定
    } catch (error) {
      console.error('Failed to save mod sources:', error)
      throw error
    }
  }

  /**
   * 获取加载状态
   */
  isLoading(): boolean {
    return this.loading.value
  }
}

// 导出单例
export const ModSourceManager = new ModSourceManagerClass()
