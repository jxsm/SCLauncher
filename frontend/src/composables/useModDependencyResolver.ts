/**
 * 模组依赖解析器
 * 在模组下载或本地导入后调用：解析其 modinfo.json 的依赖，
 * 询问用户是否自动下载缺失依赖，确认后通过各源的「按包名查询接口」递归安装。
 *
 * 同时用于存档：存档的 Project.xml/json 记录了 UsedMods（所需模组），
 * 导入/下载存档后可调用 resolveDependenciesForSave 自动补齐缺失模组。
 */
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import { useModStore } from '../stores/mod'
import { ModSourceManager } from '../managers'
import { satisfiesVersionRange } from '../utils/modVersion'
import type { Dependency } from '../types/mod'
import type { SaveRequiredMod } from '../types/savegame'

const MAX_DEPENDENCY_INSTALLS = 16 // 传递依赖安装上限，防止环/爆炸

export function useModDependencyResolver() {
  const { t } = useI18n()
  const message = useMessage()
  const dialog = useDialog()
  const modStore = useModStore()

  /**
   * 共享安装核心：给定「初始缺失依赖列表」，弹窗确认后递归下载安装。
   * 调用方负责先把 modStore.mods 刷新到最新（各自前导里 loadMods）。
   */
  async function installMissingDependencies(
    initialMissing: Dependency[],
    versionId: string,
    preferOnline: boolean,
    presenceMode: boolean
  ): Promise<void> {
    if (initialMissing.length === 0) return

    // 判断某依赖是否「已存在、无需下载」：
    // presenceMode（存档）→ 同包名已装即算（任意版本，符合“已装就别重下”）；
    // 否则（模组）→ 包名 + 版本范围都满足（尊重作者声明的版本范围）。
    const isHandled = (dep: Dependency): boolean =>
      presenceMode ? isSaveModPresent(dep) : isSatisfied(dep)

    // 弹窗询问是否自动下载
    const confirmed = await askUserConfirm(initialMissing)
    if (!confirmed) return

    const loadingMsg = message.loading(t('mods.dependencyResolving'), { duration: 0 })

    const visited = new Set<string>()
    initialMissing.forEach(d => visited.add(d.packageName.toLowerCase()))
    const queue: Dependency[] = [...initialMissing]
    let installed = 0
    const failed: Dependency[] = []
    let totalInstalls = 0

    const { DownloadModFromUrl } = await import('../api/mod')

    try {
      while (queue.length > 0 && totalInstalls < MAX_DEPENDENCY_INSTALLS) {
        const dep = queue.shift()!
        if (isHandled(dep)) continue // 已满足（可能本轮已装）

        const hit = await ModSourceManager.resolveDependency(dep.packageName, dep.versionRange, preferOnline)
        if (!hit) {
          failed.push(dep)
          continue
        }

        // 目标文件名：若与服务端返回的文件名冲突且属不同包，合成唯一文件名避免覆盖既有模组
        const destFileName = pickDestFileName(hit.fileName, dep.packageName)

        try {
          await DownloadModFromUrl(hit.downloadUrl, versionId, destFileName)
          await modStore.loadMods(versionId)
          totalInstalls++
        } catch (e) {
          console.error(`[depResolver] 下载依赖 ${dep.packageName} 失败:`, e)
          failed.push(dep)
          continue
        }

        const newMod = modStore.mods.find(m => m.fileName === destFileName)
        // 防御性校验：包名必须匹配（按包名查询接口本应精确命中）
        if (!newMod?.modInfo || newMod.modInfo.packageName.toLowerCase() !== dep.packageName.toLowerCase()) {
          if (newMod) {
            try { await modStore.deleteMod(versionId, newMod.id) } catch { /* 忽略清理失败 */ }
          }
          failed.push(dep)
          continue
        }

        installed++
        // 把新装模组的依赖入队（传递依赖）
        for (const td of newMod.modInfo.dependencies || []) {
          const key = td.packageName.toLowerCase()
          if (!visited.has(key) && !isHandled(td)) {
            visited.add(key)
            queue.push(td)
          }
        }
      }
    } finally {
      loadingMsg.destroy()
    }

    // 预算耗尽时，队列里尚未处理且仍未满足的依赖标记为未解决（避免静默丢弃）
    for (const dep of queue) {
      if (!isHandled(dep)) failed.push(dep)
    }

    // 汇总
    if (installed > 0) {
      message.success(t('mods.dependencyResolved', { count: installed }))
    }
    if (failed.length > 0) {
      message.warning(t('mods.dependencyUnresolved', {
        names: failed.map(f => f.displayName || f.packageName).join(', ')
      }))
    }
  }

  /**
   * 解析并安装 fileName 对应模组的缺失依赖。
   * 调用前应已把该模组导入版本目录（导入/下载流程会刷新 modStore.mods）。
   */
  async function resolveDependenciesForFile(
    fileName: string,
    versionId: string,
    preferOnline: boolean
  ): Promise<void> {
    await modStore.loadMods(versionId)
    // loadMods 会吞掉错误（仅写入 modStore.error）。列表加载失败时绝不能继续：
    // 否则所有已装模组都会被当成缺失而重复下载。
    if (modStore.error) {
      console.warn('[depResolver] 模组列表加载失败，跳过依赖解析以避免重复下载:', modStore.error)
      return
    }

    const mod = modStore.mods.find(m => m.fileName === fileName)
    if (!mod?.modInfo) return
    const deps: Dependency[] = mod.modInfo.dependencies || []
    if (deps.length === 0) return

    const initialMissing = deps.filter(d => !isSatisfied(d))
    await installMissingDependencies(initialMissing, versionId, preferOnline, false)
  }

  /**
   * 解析并安装存档所需模组的缺失项。
   * requiredMods 来自存档的 UsedMods（PackageName + 记录版本）。
   * 记录版本作为版本约束传入（裸版本 = ">= 该版本"，见 modVersion.ts）。
   */
  async function resolveDependenciesForSave(
    requiredMods: SaveRequiredMod[],
    versionId: string,
    preferOnline: boolean
  ): Promise<void> {
    if (!requiredMods || requiredMods.length === 0) return

    await modStore.loadMods(versionId)
    // loadMods 会吞掉错误（仅写入 modStore.error）。列表加载失败时绝不能继续：
    // 否则所有已装模组都会被当成缺失而重复下载。
    if (modStore.error) {
      console.warn('[saveDeps] 模组列表加载失败，跳过依赖解析以避免重复下载:', modStore.error)
      return
    }

    // 映射为依赖：过滤空包名；versionRange 用记录版本（供下载时按版本偏好查询），displayName 用模组名
    const deps: Dependency[] = requiredMods
      .filter(m => m && m.packageName && m.packageName.trim())
      .map(m => ({
        packageName: m.packageName,
        versionRange: (m.version || '').trim(),
        displayName: m.name || m.packageName
      }))

    // 存档所需模组：只要同包名已安装就跳过（不比较版本）。
    // 记录的版本仅用于下载缺失项时的版本偏好（见 resolveDependency）。
    const initialMissing = deps.filter(d => !isSaveModPresent(d))

    // 诊断日志：便于排查“已装仍被下载”的问题（按包名或模组名匹配）
    const installed = modStore.mods.map(m =>
      m.modInfo
        ? `${m.modInfo.name || '?'}/${m.modInfo.packageName || '?'}@${m.modInfo.version || '?'}`
        : `[无modInfo]${m.fileName}`
    )
    console.log('[saveDeps] 所需模组:', deps.map(d => `${d.displayName || d.packageName}@${d.versionRange || '*'}`))
    console.log('[saveDeps] 已装模组:', installed)
    console.log('[saveDeps] 缺失(将下载):', initialMissing.map(d => d.displayName || d.packageName))

    await installMissingDependencies(initialMissing, versionId, preferOnline, true)
  }

  // 某依赖是否已被当前已装模组满足（包名匹配 + 版本范围满足）
  function isSatisfied(d: Dependency): boolean {
    return modStore.mods.some(m =>
      !!m.modInfo &&
      m.modInfo.packageName.toLowerCase() === d.packageName.toLowerCase() &&
      satisfiesVersionRange(m.modInfo.version, d.versionRange)
    )
  }

  // 存档所需模组是否「已安装、可跳过」：
  // 优先按包名匹配；包名对不上时按模组名兜底（不同版本/分支的同一模组包名可能不同，但名通常一致——
  // 这正是“手动下载的某个模组”应被跳过的关键）。dep.displayName 在存档流程里是模组名。
  function isSaveModPresent(dep: Dependency): boolean {
    const pkg = dep.packageName.trim().toLowerCase()
    const name = (dep.displayName || '').trim().toLowerCase()
    if (!pkg && !name) return false
    // 包名最后一段（如 "com.author.foo" -> "foo"）：modInfo 缺失时按文件名兜底用。
    const pkgTail = pkg ? (pkg.split('.').pop() || '').trim() : ''
    return modStore.mods.some(m => {
      if (m.modInfo) {
        if (pkg && m.modInfo.packageName.trim().toLowerCase() === pkg) return true
        if (name && m.modInfo.name.trim().toLowerCase() === name) return true
        return false
      }
      // modInfo 解析失败的已装模组：只能按文件名兜底。模组名常是中文而文件名常是拼音/英文，
      // 单靠名字往往匹配不上；故同时用「包名末段」兜底（包名通常含可匹配的英文标识）。
      const stem = m.fileName.toLowerCase().replace(/\.disable$/, '').replace(/\.[^.]+$/, '')
      if (name && stem.includes(name)) return true
      if (pkgTail && pkgTail.length > 1 && stem.includes(pkgTail)) return true
      return false
    })
  }

  // 选择下载目标文件名：若服务端返回的文件名已被一个“不同包名”的已装模组占用，
  // 则合成基于包名的唯一文件名，避免 os.Create 覆盖既有模组文件。
  function pickDestFileName(serverFileName: string, packageName: string): string {
    const collision = modStore.mods.find(m => m.fileName === serverFileName)
    if (collision?.modInfo && collision.modInfo.packageName.toLowerCase() !== packageName.toLowerCase()) {
      const dotIdx = serverFileName.lastIndexOf('.')
      const ext = dotIdx >= 0 ? serverFileName.slice(dotIdx) : '.scmod'
      const safeBase = packageName.replace(/[^\w.-]/g, '_')
      return `${safeBase}${ext}`
    }
    return serverFileName
  }

  function askUserConfirm(missing: Dependency[]): Promise<boolean> {
    return new Promise<boolean>(resolve => {
      let done = false
      const settle = (val: boolean) => {
        if (!done) { done = true; resolve(val) }
      }
      const list = missing
        .map(d => `• ${d.displayName || d.packageName}${d.versionRange ? ' (' + d.versionRange + ')' : ''}`)
        .join('\n')
      dialog.warning({
        title: t('mods.dependencyDialogTitle'),
        content: `${t('mods.dependencyDialogPrompt')}\n\n${list}`,
        positiveText: t('mods.dependencyAutoDownload'),
        negativeText: t('common.cancel'),
        onPositiveClick: () => settle(true),
        onNegativeClick: () => settle(false),
        onClose: () => settle(false),
        onMaskClick: () => settle(false),
      })
    })
  }

  return { resolveDependenciesForFile, resolveDependenciesForSave }
}
