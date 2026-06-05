import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { Version } from '../types/version'
import * as versionApi from '../api/version'

// localStorage key
const MANIFEST_CACHE_KEY = 'sc_launcher_manifest_cache'
const MANIFEST_CACHE_TIME_KEY = 'sc_launcher_manifest_cache_time'

export const useVersionStore = defineStore('version', () => {
  // 状态
  const versions = ref<Version[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const currentVersion = ref<string | null>(null)
  const primaryVersion = ref<Version | null>(null)
  const downloading = ref<Set<string>>(new Set())
  const installing = ref<Set<string>>(new Set())
  const downloadProgress = ref<Record<string, number>>({})

  // 当前选中的版本ID（用于资源管理页面）
  const selectedVersionId = ref<string>('')

  // 当前资源管理页面的标签页（用于拖拽导入判断资源类型）
  const activeResourceTab = ref<string>('mods')

  // 下载映射和完成状态（跨页面持久化）
  const originalToUniqueId = ref<Record<string, string>>({})
  const completedDownloads = ref<Set<string>>(new Set())

  // 清单缓存（内存中，用于快速访问）
  const manifestCache = ref<Version[] | null>(null)
  const manifestCacheTime = ref<number | null>(null)
  const MANIFEST_CACHE_DURATION = 30 * 60 * 1000 // 30分钟（毫秒）

  // 从 localStorage 加载清单缓存
  function loadManifestCacheFromStorage() {
    try {
      const cachedData = localStorage.getItem(MANIFEST_CACHE_KEY)
      const cachedTime = localStorage.getItem(MANIFEST_CACHE_TIME_KEY)

      if (cachedData && cachedTime) {
        const parsedVersions = JSON.parse(cachedData) as Version[]
        const parsedTime = parseInt(cachedTime, 10)

        if (Array.isArray(parsedVersions) && !isNaN(parsedTime)) {
          manifestCache.value = parsedVersions
          manifestCacheTime.value = parsedTime
          console.log('[Version] Loaded manifest cache from localStorage, cached at:', new Date(parsedTime).toLocaleString())
          return true
        }
      }
    } catch (error) {
      console.error('[Version] Failed to load manifest cache from localStorage:', error)
    }
    return false
  }

  // 保存清单缓存到 localStorage
  function saveManifestCacheToStorage(versions: Version[]) {
    try {
      localStorage.setItem(MANIFEST_CACHE_KEY, JSON.stringify(versions))
      localStorage.setItem(MANIFEST_CACHE_TIME_KEY, Date.now().toString())
      console.log('[Version] Saved manifest cache to localStorage')
    } catch (error) {
      console.error('[Version] Failed to save manifest cache to localStorage:', error)
    }
  }

  // 清除 localStorage 中的清单缓存
  function clearManifestCacheFromStorage() {
    try {
      localStorage.removeItem(MANIFEST_CACHE_KEY)
      localStorage.removeItem(MANIFEST_CACHE_TIME_KEY)
      console.log('[Version] Cleared manifest cache from localStorage')
    } catch (error) {
      console.error('[Version] Failed to clear manifest cache from localStorage:', error)
    }
  }

  // 初始化时从 localStorage 加载缓存
  loadManifestCacheFromStorage()

  // 计算属性
  const installedVersions = computed(() =>
    versions.value.filter(v => v.installed)
  )

  const versionsByType = computed(() => {
    const grouped: Record<string, Version[]> = {
      api: [] as Version[],
      net: [] as Version[],
      original: [] as Version[]
    }
    versions.value.forEach(v => {
      if (!grouped[v.versionType]) {
        grouped[v.versionType] = []
      }
      grouped[v.versionType].push(v)
    })
    return grouped
  })

  // 检查缓存是否过期（注意：这只是简单检查，实际逻辑在 fetchVersions 中）
  function isManifestCacheExpired(): boolean {
    if (!manifestCacheTime.value) return true
    const now = Date.now()
    return now - manifestCacheTime.value > MANIFEST_CACHE_DURATION
  }

  // 清除清单缓存（包括内存和 localStorage）
  function clearManifestCache() {
    manifestCache.value = null
    manifestCacheTime.value = null
    clearManifestCacheFromStorage()
    console.log('[Version] Manifest cache cleared')
  }

  // 获取缓存状态信息（用于调试）
  function getCacheStatus() {
    if (!manifestCacheTime.value) {
      return { hasCache: false, age: 0, ageMinutes: 0, isExpired: true }
    }
    const now = Date.now()
    const age = now - manifestCacheTime.value
    return {
      hasCache: manifestCache.value !== null,
      age,
      ageMinutes: Math.floor(age / 1000 / 60),
      isExpired: age > MANIFEST_CACHE_DURATION,
      cachedAt: new Date(manifestCacheTime.value).toLocaleString(),
      cacheDuration: MANIFEST_CACHE_DURATION / 1000 / 60 + ' minutes'
    }
  }

  // 操作
  // @param forceRefresh - 是否强制刷新，跳过缓存直接请求服务器
  // @param onBackgroundUpdate - 可选回调，当后台刷新完成时调用（用于同步组件数据）
  async function fetchVersions(forceRefresh?: boolean, onBackgroundUpdate?: (versions: Version[]) => void): Promise<Version[]> {
    const now = Date.now()
    const cacheAge = manifestCacheTime.value ? now - manifestCacheTime.value : Infinity
    const hasCache = manifestCache.value !== null
    const isExpired = cacheAge > MANIFEST_CACHE_DURATION

    // 情况0：强制刷新 - 跳过缓存，直接请求服务器
    if (forceRefresh) {
      console.log('[Version] Force refresh, fetching from server...')
      loading.value = true
      error.value = null
      try {
        const fetchedVersions = await versionApi.FetchVersions()
        versions.value = fetchedVersions

        // 更新缓存
        manifestCache.value = fetchedVersions
        manifestCacheTime.value = Date.now()
        // 保存到 localStorage
        saveManifestCacheToStorage(fetchedVersions)
        console.log('[Version] Force refresh completed, cached at:', new Date(manifestCacheTime.value).toLocaleString())

        return fetchedVersions
      } catch (e) {
        error.value = e as string
        console.error('[Version] Force refresh failed:', e)
        throw e
      } finally {
        loading.value = false
      }
    }

    // 情况1：有缓存且未过期 - 直接使用缓存
    if (hasCache && !isExpired) {
      console.log('[Version] Using fresh cache (age:', Math.floor(cacheAge / 1000 / 60), 'minutes)')
      versions.value = manifestCache.value!
      return manifestCache.value!
    }

    // 情况2：有缓存但已过期 - 先返回缓存，然后后台静默刷新
    if (hasCache && isExpired) {
      console.log('[Version] Cache expired (age:', Math.floor(cacheAge / 1000 / 60), 'minutes), using stale cache and refreshing in background...')
      versions.value = manifestCache.value!

      // 后台静默刷新（不设置 loading，不阻塞界面）
      versionApi.FetchVersions()
        .then(fetchedVersions => {
          console.log('[Version] Background refresh completed, updating cache and UI')
          // 更新缓存
          manifestCache.value = fetchedVersions
          manifestCacheTime.value = Date.now()
          // 保存到 localStorage
          saveManifestCacheToStorage(fetchedVersions)
          // 更新界面
          versions.value = fetchedVersions
          // 通知调用方后台刷新完成
          if (onBackgroundUpdate) {
            onBackgroundUpdate(fetchedVersions)
          }
        })
        .catch(e => {
          console.error('[Version] Background refresh failed:', e)
          // 静默失败，不影响当前显示的缓存数据
        })

      return manifestCache.value!
    }

    // 情况3：没有缓存 - 需要请求
    console.log('[Version] No cache, fetching from server...')
    loading.value = true
    error.value = null
    try {
      const fetchedVersions = await versionApi.FetchVersions()
      versions.value = fetchedVersions

      // 更新缓存
      manifestCache.value = fetchedVersions
      manifestCacheTime.value = Date.now()
      // 保存到 localStorage
      saveManifestCacheToStorage(fetchedVersions)
      console.log('[Version] Manifest cached at:', new Date(manifestCacheTime.value).toLocaleString())

      return fetchedVersions
    } catch (e) {
      error.value = e as string
      console.error('Failed to fetch versions:', e)
      throw e
    } finally {
      loading.value = false
    }
  }

  async function getVersions() {
    loading.value = true
    error.value = null
    try {
      versions.value = await versionApi.GetVersions()
    } catch (e) {
      error.value = e as string
      console.error('Failed to get versions:', e)
    } finally {
      loading.value = false
    }
  }

  async function downloadVersion(versionId: string) {
    downloading.value.add(versionId)
    try {
      await versionApi.DownloadVersion(versionId)
    } catch (e) {
      console.error('Failed to download version:', e)
      throw e
    }
  }

  async function downloadVersionWithCustomName(versionId: string, customName: string) {
    downloading.value.add(versionId)
    try {
      await versionApi.DownloadVersionWithCustomName(versionId, customName)
    } catch (e) {
      console.error('Failed to download version:', e)
      throw e
    }
  }

  function finishDownload(versionId: string) {
    downloading.value.delete(versionId)
  }

  function updateDownloadProgress(versionId: string, progress: number) {
    downloadProgress.value[versionId] = progress
  }

  function clearDownloadProgress(versionId: string) {
    delete downloadProgress.value[versionId]
  }

  // 设置原始ID到唯一ID的映射
  function setDownloadMapping(originalId: string, uniqueId: string) {
    originalToUniqueId.value[originalId] = uniqueId
    console.log(`[VersionStore] Set mapping: ${originalId} -> ${uniqueId}`)
  }

  // 移除下载映射
  function removeDownloadMapping(originalId: string) {
    if (originalToUniqueId.value[originalId]) {
      delete originalToUniqueId.value[originalId]
      console.log(`[VersionStore] Removed mapping: ${originalId}`)
    }
  }

  // 获取唯一ID
  function getUniqueId(originalId: string): string | undefined {
    return originalToUniqueId.value[originalId]
  }

  // 标记下载为已完成
  function markDownloadCompleted(versionId: string) {
    completedDownloads.value.add(versionId)
    console.log(`[VersionStore] Marked as completed: ${versionId}`)
  }

  // 检查下载是否已完成
  function isDownloadCompleted(versionId: string): boolean {
    return completedDownloads.value.has(versionId)
  }

  // 清除完成标记
  function clearDownloadCompleted(versionId: string) {
    completedDownloads.value.delete(versionId)
    console.log(`[VersionStore] Cleared completed mark: ${versionId}`)
  }

  // 获取下载状态（考虑映射）
  function getDownloadingStatus(originalId: string): { isDownloading: boolean; uniqueId?: string; progress: number } {
    const uniqueId = originalToUniqueId.value[originalId]
    const isDownloadingOriginal = downloading.value.has(originalId)
    const isDownloadingUnique = uniqueId ? downloading.value.has(uniqueId) : false
    const isDownloading = isDownloadingOriginal || isDownloadingUnique

    // 获取进度（优先使用 uniqueId）
    let progress = 0
    if (uniqueId) {
      progress = downloadProgress.value[uniqueId] || 0
    }
    if (progress === 0) {
      progress = downloadProgress.value[originalId] || 0
    }

    return {
      isDownloading,
      uniqueId,
      progress
    }
  }

  async function installVersion(versionId: string) {
    installing.value.add(versionId)
    try {
      await versionApi.InstallVersion(versionId)
      // 更新版本状态
      const version = versions.value.find(v => v.id === versionId)
      if (version) {
        version.installed = true
      }
      // 刷新主要版本（可能自动设置了新的主要版本）
      await getPrimaryVersion()
    } catch (e) {
      console.error('Failed to install version:', e)
      throw e
    } finally {
      installing.value.delete(versionId)
    }
  }

  async function deleteVersion(versionId: string) {
    try {
      await versionApi.DeleteVersion(versionId)
      // 更新版本状态
      const version = versions.value.find(v => v.id === versionId)
      if (version) {
        version.installed = false
      }
      // 刷新主要版本（可能重新选择了新的主要版本）
      await getPrimaryVersion()
    } catch (e) {
      console.error('Failed to delete version:', e)
      throw e
    }
  }

  async function renameVersion(versionId: string, newName: string) {
    try {
      await versionApi.RenameVersion(versionId, newName)
      // 更新版本名称
      const version = versions.value.find(v => v.id === versionId)
      if (version) {
        version.name = newName
      }
      // 更新主要版本名称
      if (primaryVersion.value && primaryVersion.value.id === versionId) {
        primaryVersion.value.name = newName
      }
    } catch (e) {
      console.error('Failed to rename version:', e)
      throw e
    }
  }

  async function setCurrentVersion(versionId: string) {
    try {
      await versionApi.SetPrimaryVersion(versionId)
      currentVersion.value = versionId
    } catch (e) {
      console.error('Failed to set current version:', e)
      throw e
    }
  }

  async function getPrimaryVersion() {
    try {
      primaryVersion.value = await versionApi.GetPrimaryVersion()
      // 更新版本列表中的主要标记
      versions.value.forEach(v => {
        v.isPrimary = primaryVersion.value !== null && v.id === primaryVersion.value.id
      })
    } catch (e) {
      console.error('Failed to get primary version:', e)
    }
  }

  async function setPrimaryVersion(versionId: string) {
    try {
      await versionApi.SetPrimaryVersion(versionId)
      // 更新本地状态
      await getPrimaryVersion()
      // 更新版本列表中的主要标记
      versions.value.forEach(v => {
        v.isPrimary = v.id === versionId
      })
    } catch (e) {
      console.error('Failed to set primary version:', e)
      throw e
    }
  }

  return {
    versions,
    loading,
    error,
    currentVersion,
    primaryVersion,
    selectedVersionId,
    activeResourceTab,
    downloading,
    installing,
    downloadProgress,
    originalToUniqueId,
    completedDownloads,
    installedVersions,
    versionsByType,
    fetchVersions,
    getVersions,
    downloadVersion,
    downloadVersionWithCustomName,
    finishDownload,
    updateDownloadProgress,
    clearDownloadProgress,
    setDownloadMapping,
    removeDownloadMapping,
    getUniqueId,
    markDownloadCompleted,
    isDownloadCompleted,
    clearDownloadCompleted,
    getDownloadingStatus,
    installVersion,
    deleteVersion,
    renameVersion,
    setCurrentVersion,
    getPrimaryVersion,
    setPrimaryVersion,
    clearManifestCache,
    getCacheStatus
  }
})
