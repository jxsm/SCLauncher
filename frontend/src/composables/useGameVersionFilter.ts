/**
 * 资源下载页的游戏版本筛选（对齐 scbbs 社区站的版本筛选）：
 * - 下拉选项来自社区版本目录（/api/version/list），默认选中当前主版本对应的游戏版本（如 2.4 → "2.4.x"）
 * - 选中的版本 ID 通过 ModSourceManager 的 filters（gameVersionIds 参数）交给服务端过滤，与分页兼容
 * - 目录拉取失败时静默保持「全部版本」且不显示下拉，不影响原有下载流程
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useVersionStore } from '../stores/version'
import { getCommunityGameVersions, resolveGameVersionId } from '../api/communityVersions'

export function useGameVersionFilter() {
  const { t } = useI18n()
  const versionStore = useVersionStore()

  // 0 = 全部版本
  const gameVersionId = ref(0)
  const gameVersionOptions = ref<Array<{ label: string; value: number }>>([])

  let initPromise: Promise<void> | null = null

  /**
   * 各下载视图在首次拉列表前调用（幂等，二次调用直接返回）：
   * 加载版本目录 + 解析默认值。任何失败都静默降级为「全部版本」。
   */
  function ensureGameVersionFilter(): Promise<void> {
    if (!initPromise) {
      initPromise = (async () => {
        try {
          const versions = await getCommunityGameVersions()
          if (versions.length === 0) return
          gameVersionOptions.value = [
            { label: t('common.allGameVersions'), value: 0 },
            ...versions.map(v => ({ label: v.name, value: v.id }))
          ]
          // 默认选中当前主版本的游戏版本
          if (!versionStore.primaryVersion) {
            await versionStore.getPrimaryVersion().catch(() => {})
          }
          const primaryGameVersion = versionStore.primaryVersion?.gameVersion
          if (primaryGameVersion) {
            gameVersionId.value = (await resolveGameVersionId(primaryGameVersion)) ?? 0
          }
        } catch (error) {
          console.warn('[gameVersionFilter] 初始化社区版本目录失败:', error)
        }
      })()
    }
    return initPromise
  }

  /** 传给 ModSourceManager.getModList/searchMods 的 filters；「全部」时不传 */
  function gameVersionFilters(): { gameVersionIds: number } | undefined {
    return gameVersionId.value ? { gameVersionIds: gameVersionId.value } : undefined
  }

  return { gameVersionId, gameVersionOptions, ensureGameVersionFilter, gameVersionFilters }
}
