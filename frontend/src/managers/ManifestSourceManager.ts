import type { ManifestSource } from '../types/manifest-source'

const MANIFEST_SOURCES_KEY = 'manifest_sources'

export class ManifestSourceManager {
  private static sources: ManifestSource[] = []

  /**
   * 初始化管理器，从配置文件加载清单源
   */
  static async loadSources(configSources: any[]): Promise<void> {
    if (Array.isArray(configSources) && configSources.length > 0) {
      this.sources = configSources.map(s => ({
        id: s.id,
        name: s.name,
        url: s.url,
        isDefault: s.isDefault || false
      }))

      // 检查是否存在默认源
      const hasDefaultSource = this.sources.some(s => s.id === 'default')

      // 如果不存在默认源，添加它
      if (!hasDefaultSource) {
        this.sources.unshift({
          id: 'default',
          name: 'BTOS',
          url: 'https://sc.btos.top/api/manifest.json',
          isDefault: true
        })
      }
    } else {
      // 如果配置中没有清单源，使用默认源
      this.sources = [{
        id: 'default',
        name: 'BTOS',
        url: 'https://sc.btos.top/api/manifest.json',
        isDefault: true
      }]
    }
  }

  /**
   * 获取所有清单源
   */
  static getAllSources(): ManifestSource[] {
    return [...this.sources]
  }

  /**
   * 根据ID获取清单源
   */
  static getSourceById(id: string): ManifestSource | undefined {
    return this.sources.find(s => s.id === id)
  }

  /**
   * 添加清单源
   */
  static async addSource(source: Omit<ManifestSource, 'isDefault'>): Promise<void> {
    // 检查ID是否已存在
    if (this.sources.some(s => s.id === source.id)) {
      throw new Error('清单源ID已存在')
    }

    // 添加新源
    this.sources.push({
      ...source,
      isDefault: false
    })

    // 保存到后端
    await this.saveSources()
  }

  /**
   * 删除清单源
   */
  static async removeSource(id: string): Promise<void> {
    // 不允许删除默认源
    if (id === 'default') {
      throw new Error('不能删除默认清单源')
    }

    const index = this.sources.findIndex(s => s.id === id)
    if (index === -1) {
      throw new Error('清单源不存在')
    }

    this.sources.splice(index, 1)

    // 保存到后端
    await this.saveSources()
  }

  /**
   * 更新清单源
   */
  static async updateSource(source: ManifestSource): Promise<void> {
    const index = this.sources.findIndex(s => s.id === source.id)
    if (index === -1) {
      throw new Error('清单源不存在')
    }

    this.sources[index] = source

    // 保存到后端
    await this.saveSources()
  }

  /**
   * 保存清单源到后端配置
   */
  static async saveSources(): Promise<void> {
    const { SaveManifestSources } = await import('../api/config')
    await SaveManifestSources(this.sources)
  }

  /**
   * 设置当前选中的清单源
   */
  static async setCurrentSource(id: string): Promise<void> {
    const source = this.sources.find(s => s.id === id)
    if (!source) {
      throw new Error('清单源不存在')
    }

    const { SetCurrentManifestSource } = await import('../api/config')
    await SetCurrentManifestSource(id)
  }

  /**
   * 获取当前选中的清单源
   */
  static getCurrentSource(): ManifestSource | undefined {
    // 这里需要从后端获取当前选中的源ID
    // 暂时返回第一个源或默认源
    return this.sources.find(s => s.isDefault) || this.sources[0]
  }

  /**
   * 重新加载源列表
   */
  static async reloadSources(): Promise<void> {
    const { GetConfig } = await import('../api/config')
    const config = await GetConfig()
    await this.loadSources(config.manifestSources || [])
  }
}
