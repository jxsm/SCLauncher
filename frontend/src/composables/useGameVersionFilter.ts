/**
 * 资源下载页的游戏版本筛选（对齐 scbbs 社区站 / scbbs-plus-mod 的版本筛选语义）：
 * - 选中的版本 ID 通过 ModSourceManager 的 filters（gameVersionIds 参数）交给服务端过滤，与分页兼容
 * - 非模组资源（皮肤/家具/材质/存档/整合包）适配游戏大版本：ensureGameVersionFilter
 *   选项为整个目录，默认按主版本的 gameVersion（如 2.4）匹配原版系列（"2.4.x"）
 * - 模组走 applyModsVersionContext（参考 scbbs-plus-mod FindBestPluginVersion）：
 *   单机版按 subVersion（如 "api-1.9.5.1"）匹配插件 API 系列（API1.9.x），
 *   联机版按 subVersion（如 "26.07.01.01"）匹配联机系列（x26.10），
 *   且下拉只显示对应系列（联机系列仅在联机版时出现）
 * - 目录拉取失败时静默保持「全部版本」且不显示下拉，不影响原有下载流程
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useVersionStore } from '../stores/version'
import {
  getCommunityGameVersions,
  resolveGameVersionId,
  findBestCommunityVersion,
  type CommunityGameVersion
} from '../api/communityVersions'

export function useGameVersionFilter() {
  const { t } = useI18n()
  const versionStore = useVersionStore()

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
   * 非模组视图在首次拉列表前调用（幂等）：加载版本目录 + 按主版本 gameVersion
   * 解析默认值。任何失败都静默降级为「全部版本」。
   */
  function ensureGameVersionFilter(): Promise<void> {
    if (!initPromise) {
      initPromise = (async () => {
        await ensureCatalog()
        if (catalog.value.length === 0) return
        gameVersionOptions.value = [
          { label: t('common.allGameVersions'), value: 0 },
          ...catalog.value.map(v => ({ label: v.name, value: v.id }))
        ]
        // 默认选中当前主版本的游戏版本（如 2.4 → "2.4.x"）
        if (!versionStore.primaryVersion) {
          await versionStore.getPrimaryVersion().catch(() => {})
        }
        const primaryGameVersion = versionStore.primaryVersion?.gameVersion
        if (primaryGameVersion) {
          gameVersionId.value = (await resolveGameVersionId(primaryGameVersion)) ?? 0
        }
      })()
    }
    return initPromise
  }

  /**
   * 模组视图：按当前选中版本的联机/单机重算筛选项与默认值（每次拉列表前调用，
   * 上下文（subVersion + 联机状态）没变时跳过，避免覆盖用户手选的筛选值）。
   * 单机 → 只显示插件 API 系列；联机 → 只显示联机系列；
   * 匹配不到（如原版/导入版本）→ 默认「全部版本」。
   */
  let lastModsContext = ''
  async function applyModsVersionContext(subVersion: string, isOnline: boolean): Promise<void> {
    const contextKey = `${isOnline}|${subVersion}`
    if (contextKey === lastModsContext) return
    lastModsContext = contextKey
    await ensureCatalog()
    if (catalog.value.length === 0) return
    const type = isOnline ? 'online' : 'plugin'
    gameVersionOptions.value = [
      { label: t('common.allGameVersions'), value: 0 },
      ...catalog.value.filter(v => v.type === type).map(v => ({ label: v.name, value: v.id }))
    ]
    // 主版本校验，避免不可信的 subVersion 产生误导性匹配（对齐 inferApiVersionFromVersionId
    // 的约束）：插件 API 恒为 1.x（"api-1.9.5.1"/"API1.60"）；联机版本号是年份系列（x26.10 → 26）。
    // 原版（"Survivalcraft2.4.10.4"）等不满足时保持「全部版本」。
    const nums = (subVersion.match(/\d+/g) ?? []).map(Number)
    const majorOk = type === 'plugin'
      ? nums.length > 0 && nums[0] === 1
      : nums.length > 0 && nums[0] >= 20
    const best = majorOk ? await findBestCommunityVersion(type, subVersion) : null
    gameVersionId.value = best?.id ?? 0
  }

  /** 传给 ModSourceManager.getModList/searchMods 的 filters；「全部」时不传 */
  function gameVersionFilters(): { gameVersionIds: number } | undefined {
    return gameVersionId.value ? { gameVersionIds: gameVersionId.value } : undefined
  }

  return {
    gameVersionId,
    gameVersionOptions,
    ensureGameVersionFilter,
    applyModsVersionContext,
    gameVersionFilters
  }
}
