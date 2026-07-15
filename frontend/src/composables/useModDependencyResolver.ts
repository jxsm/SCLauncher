/**
 * 模组依赖解析器
 * 在模组下载或本地导入后调用：解析其 modinfo.json 的依赖，
 * 询问用户是否自动下载缺失依赖，确认后通过各源的「按包名查询接口」递归安装。
 */
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import { useModStore } from '../stores/mod'
import { ModSourceManager } from '../managers'
import { satisfiesVersionRange } from '../utils/modVersion'
import type { Dependency } from '../types/mod'

const MAX_DEPENDENCY_INSTALLS = 16 // 传递依赖安装上限，防止环/爆炸

export function useModDependencyResolver() {
  const { t } = useI18n()
  const message = useMessage()
  const dialog = useDialog()
  const modStore = useModStore()

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

    const mod = modStore.mods.find(m => m.fileName === fileName)
    if (!mod?.modInfo) return
    const deps: Dependency[] = mod.modInfo.dependencies || []
    if (deps.length === 0) return

    // 某依赖是否已被当前已装模组满足（包名匹配 + 版本范围满足）
    const isSatisfied = (d: Dependency): boolean => {
      return modStore.mods.some(m =>
        !!m.modInfo &&
        m.modInfo.packageName.toLowerCase() === d.packageName.toLowerCase() &&
        satisfiesVersionRange(m.modInfo.version, d.versionRange)
      )
    }

    const initialMissing = deps.filter(d => !isSatisfied(d))
    if (initialMissing.length === 0) return

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
        if (isSatisfied(dep)) continue // 已满足（可能本轮已装）

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
          if (!visited.has(key) && !isSatisfied(td)) {
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
      if (!isSatisfied(dep)) failed.push(dep)
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

  return { resolveDependenciesForFile }
}
