/**
 * 资源下载页的游戏版本筛选（仅模组页与整合包页提供）：
 * - 选中的版本 ID 通过 ModSourceManager 的 filters（gameVersionIds 参数）交给服务端过滤，
 *   与分页兼容；默认「全部版本」，不自动匹配当前版本
 * - 整合包页走 ensureGameVersionOptions：选项为整个目录，默认「全部版本」
 * - 模组页走 applyModsVersionContext：单机版只显示插件 API 系列（API1.9.x…），
 *   联机版只显示联机系列（x26.10），系列切换时重置为「全部版本」，
 *   同系列内不重置以保留用户手选
 * - 目录拉取失败时静默保持「全部版本」且不显示下拉，不影响原有下载流程
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getCommunityGameVersions, type CommunityGameVersion } from '../api/communityVersions'

export function useGameVersionFilter() {
  const { t } = useI18n()

  // 0 = 全部版本
  const gameVersionId = ref(0)
  const gameVersionOptions = ref<Array<{ label: string; value: number }>>([])

  const catalog = ref<CommunityGameVersion[]>([])
  let catalogPromise: Promise<void> | null = null

  /** 拉取社区版本目录（幂等，失败静默并允许下次重试） */
  function ensureCatalog(): Promise<void> {
    if (!catalogPromise) {
      catalogPromise = getCommunityGameVersions()
        .then(list => {
          catalog.value = list
        })
        .catch(error => {
          catalogPromise = null
          console.warn('[gameVersionFilter] 拉取社区版本目录失败:', error)
        })
    }
    return catalogPromise
  }

  let initPromise: Promise<void> | null = null

  /**
   * 加载版本目录并填充下拉选项（幂等）：选项为整个目录 + 「全部版本」，
   * 默认保持「全部版本」。供整合包等展示完整目录的视图使用。
   */
  function ensureGameVersionOptions(): Promise<void> {
    if (!initPromise) {
      initPromise = (async () => {
        await ensureCatalog()
        if (catalog.value.length === 0) return
        gameVersionOptions.value = [
          { label: t('common.allGameVersions'), value: 0 },
          ...catalog.value.map(v => ({ label: v.name, value: v.id }))
        ]
      })()
    }
    return initPromise
  }

  /**
   * 模组视图：按当前选中版本的联机/单机重算下拉选项（每次拉列表前调用）。
   * 联机状态没变时跳过（保留用户手选的筛选值）；切换系列时重置为「全部版本」。
   * 单机 → 只显示插件 API 系列；联机 → 只显示联机系列。
   */
  let lastModsContext = ''
  async function applyModsVersionContext(isOnline: boolean): Promise<void> {
    const contextKey = isOnline ? 'online' : 'plugin'
    if (contextKey === lastModsContext) return
    lastModsContext = contextKey
    await ensureCatalog()
    if (catalog.value.length === 0) return
    gameVersionOptions.value = [
      { label: t('common.allGameVersions'), value: 0 },
      ...catalog.value.filter(v => v.type === contextKey).map(v => ({ label: v.name, value: v.id }))
    ]
    gameVersionId.value = 0
  }

  /** 传给 ModSourceManager.getModList/searchMods 的 filters；「全部」时不传 */
  function gameVersionFilters(): { gameVersionIds: number } | undefined {
    return gameVersionId.value ? { gameVersionIds: gameVersionId.value } : undefined
  }

  return {
    gameVersionId,
    gameVersionOptions,
    ensureGameVersionOptions,
    applyModsVersionContext,
    gameVersionFilters
  }
}
