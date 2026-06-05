<template>
  <!-- 拖拽视觉反馈覆盖层 -->
  <Transition name="fade">
    <div v-if="isDragging" class="drag-overlay">
      <div class="drag-content">
        <n-icon size="64" :component="CloudUploadOutline" />
        <p class="drag-title">{{ dragTitle }}</p>
        <p class="drag-hint">{{ dragHint }}</p>
      </div>
    </div>
  </Transition>

  <!-- 版本选择弹窗（选择已安装的游戏版本） -->
  <VersionSelectDialog
    v-model:show="showVersionSelect"
    :resource-type="pendingResourceType"
    :file-name="pendingFileName"
    @select="handleVersionSelect"
    @cancel="handleCancel"
  />

  <!-- 整合包信息对话框 -->
  <ModpackInfoDialog
    v-model:show="showModpackDialog"
    :modpack-data="parsedModpackInfo"
    @install="handleModpackInstall"
  />

  <!-- 整合包安装进度对话框 -->
  <ModpackInstallDialog
    v-model:show="showInstallDialog"
    :modpack-info="parsedModpackInfo"
    @completed="handleModpackCompleted"
    @error="handleModpackError"
  />
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useMessage, useDialog, NInput } from 'naive-ui'
import { CloudUploadOutline } from '@vicons/ionicons5'
import { OnFileDrop, OnFileDropOff, EventsEmit } from '../../wailsjs/runtime/runtime'
import { ImportMod } from '../api/mod'
import { ImportTexture } from '../api/texture'
import { ImportFurniture } from '../api/furniture'
import { ImportSaveGame } from '../api/savegame'
import { ImportSkin } from '../api/skin'
import { InspectArchive } from '../api/config'
import { ParseModpack, InstallModpackWithProgress, InstallFromArchive } from '../api/version'
import VersionSelectDialog from './VersionSelectDialog.vue'
import ModpackInfoDialog from './ModpackInfoDialog.vue'
import ModpackInstallDialog from './ModpackInstallDialog.vue'
import { useVersionStore } from '../stores/version'

const { t } = useI18n()
const route = useRoute()
const message = useMessage()
const dialog = useDialog()
const versionStore = useVersionStore()

// 整合包安装状态
const showModpackDialog = ref(false)
const showInstallDialog = ref(false)
const parsedModpackInfo = ref<any>(null)

// 拖拽状态
const isDragging = ref(false)
const dragCounter = ref(0)

// 版本选择弹窗状态
const showVersionSelect = ref(false)
const pendingResourceType = ref<ResourceType>('')
const pendingFilePath = ref('')
const pendingFileName = ref('')

// 资源类型定义
type ResourceType = 'mod' | 'texture' | 'furniture' | 'savegame' | 'skin' | 'modpack' | ''

// 文件后缀到资源类型的映射
const EXTENSION_MAP: Record<string, ResourceType> = {
  '.scmod': 'mod',
  '.disable': 'mod',
  '.scbtex': 'texture',
  '.scfpack': 'furniture',
  '.scworld': 'savegame',
  '.scword': 'savegame',
  '.scskin': 'skin',
  '.scmodpack': 'modpack',
}

// 资源管理页面路由名称到资源类型的映射
const ROUTE_RESOURCE_MAP: Record<string, ResourceType> = {
  'Resources': '',
  'Mods': 'mod',
  'SaveGames': 'savegame',
  'Skins': 'skin',
}

// Resources 页面标签页名称到资源类型的映射
const TAB_RESOURCE_MAP: Record<string, ResourceType> = {
  'mods': 'mod',
  'furniture': 'furniture',
  'textures': 'texture',
  'savegames': 'savegame',
  'skins': 'skin',
}

// 当前是否在资源管理页面
const isResourcePage = computed(() => {
  const routeName = route.name as string
  return routeName in ROUTE_RESOURCE_MAP
})

// 当前资源管理页面的类型
const currentResourceType = computed<ResourceType>(() => {
  const routeName = route.name as string
  // Resources 总页面：由当前标签页决定
  if (routeName === 'Resources') {
    return TAB_RESOURCE_MAP[versionStore.activeResourceTab] || ''
  }
  return ROUTE_RESOURCE_MAP[routeName] || ''
})

// 拖拽提示文本
const dragTitle = computed(() => {
  if (isResourcePage.value && currentResourceType.value) {
    const typeNames: Record<ResourceType, string> = {
      'mod': t('dragDrop.types.mod'),
      'texture': t('dragDrop.types.texture'),
      'furniture': t('dragDrop.types.furniture'),
      'savegame': t('dragDrop.types.savegame'),
      'skin': t('dragDrop.types.skin'),
      'modpack': t('dragDrop.types.modpack'),
      '': '',
    }
    return t('dragDrop.dropToImport', { type: typeNames[currentResourceType.value] })
  }
  return t('dragDrop.dropFile')
})

const dragHint = computed(() => {
  if (isResourcePage.value && currentResourceType.value) {
    const extensions: Record<ResourceType, string> = {
      'mod': '.scmod, .zip, .7z, .disable',
      'texture': '.scbtex',
      'furniture': '.scfpack',
      'savegame': '.scworld, .scword, .zip',
      'skin': '.scskin',
      'modpack': '.scmodpack, .zip',
      '': '',
    }
    return t('dragDrop.supportedFormats', { formats: extensions[currentResourceType.value] })
  }
  return t('dragDrop.supportedFormatsAll')
})

// 获取文件后缀
function getFileExtension(filePath: string): string {
  const lastDot = filePath.lastIndexOf('.')
  if (lastDot === -1) return ''
  return filePath.substring(lastDot).toLowerCase()
}

// 获取文件名
function getFileName(filePath: string): string {
  const parts = filePath.replace(/\\/g, '/').split('/')
  return parts[parts.length - 1] || filePath
}

// 判断文件是否为当前资源类型支持的格式
function isFileSupportedForCurrentType(filePath: string): boolean {
  const ext = getFileExtension(filePath)
  const fileType = EXTENSION_MAP[ext]

  // .zip 和 .7z 由 InspectArchive 单独处理，这里总是返回 true
  if (ext === '.zip' || ext === '.7z') {
    return true
  }

  // Resources 总页面（无特定类型）：接受任何支持的格式
  if (!currentResourceType.value) {
    return !!fileType
  }

  return fileType === currentResourceType.value
}

// 根据类型调用导入 API
async function importByType(resourceType: ResourceType, versionId: string, filePath: string) {
  console.log('[DragDrop] importByType called:', { resourceType, versionId, filePath })
  switch (resourceType) {
    case 'mod':
      console.log('[DragDrop] → Calling ImportMod')
      await ImportMod(versionId, filePath)
      break
    case 'texture':
      console.log('[DragDrop] → Calling ImportTexture')
      await ImportTexture(versionId, filePath)
      break
    case 'furniture':
      console.log('[DragDrop] → Calling ImportFurniture')
      await ImportFurniture(versionId, filePath)
      break
    case 'savegame':
      console.log('[DragDrop] → Calling ImportSaveGame')
      await ImportSaveGame(versionId, filePath)
      break
    case 'modpack':
      console.log('[DragDrop] → Calling ImportMod (modpack)')
      await ImportMod(versionId, filePath)
      break
    default:
      console.error('[DragDrop] ❌ Unknown resource type:', resourceType)
      throw new Error(`Unknown resource type: ${resourceType}`)
  }
  console.log('[DragDrop] importByType completed')
}

// 关闭遮罩
function closeOverlay() {
  isDragging.value = false
  dragCounter.value = 0
}

// 处理文件拖入（OnFileDrop 回调，有完整文件路径）
async function handleFileDrop(paths: string[]) {
  console.log('[DragDrop] ===== handleFileDrop START =====')
  console.log('[DragDrop] paths:', paths)

  closeOverlay()

  if (!paths || paths.length === 0) {
    console.log('[DragDrop] ❌ No paths provided, aborting')
    return
  }

  const filePath = paths[0]
  const ext = getFileExtension(filePath)
  const fileName = getFileName(filePath)

  console.log('[DragDrop] filePath:', filePath)
  console.log('[DragDrop] ext:', ext)
  console.log('[DragDrop] fileName:', fileName)

  // 判断资源类型
  let resourceType = EXTENSION_MAP[ext]

  // .scmodpack 是整合包
  if (ext === '.scmodpack') {
    console.log('[DragDrop] → .scmodpack, starting modpack install')
    await startModpackInstall(filePath)
    return
  }

  // .zip 和 .7z 需要检查压缩包内容来判断类型
  if (ext === '.zip' || ext === '.7z') {
    console.log('[DragDrop] Archive file, inspecting contents...')
    try {
      const archiveType = await InspectArchive(filePath)
      console.log('[DragDrop] Archive type:', archiveType)

      if (archiveType === 'game') {
        console.log('[DragDrop] → Game archive, starting game install')
        await startGameInstall(filePath)
        return
      } else if (archiveType === 'modpack') {
        console.log('[DragDrop] → Modpack, starting modpack install')
        await startModpackInstall(filePath)
        return
      } else {
        message.warning(t('dragDrop.unsupportedArchive'))
        return
      }
    } catch (error) {
      console.error('[DragDrop] ❌ InspectArchive failed:', error)
      message.error(t('dragDrop.importFailed') + '：' + error)
      return
    }
  }

  console.log('[DragDrop] resourceType:', resourceType)
  console.log('[DragDrop] isResourcePage:', isResourcePage.value)
  console.log('[DragDrop] currentResourceType:', currentResourceType.value)
  console.log('[DragDrop] route.name:', route.name)

  // 不支持的文件类型 → 忽略
  if (!resourceType) {
    console.log('[DragDrop] ❌ Unsupported format, ignoring:', ext)
    return
  }

  // 在资源管理页面
  if (isResourcePage.value) {
    const supported = isFileSupportedForCurrentType(filePath)
    console.log('[DragDrop] isFileSupportedForCurrentType:', supported)

    if (!supported) {
      console.log('[DragDrop] ❌ File type mismatch for current page')
      const typeNames: Record<ResourceType, string> = {
        'mod': t('dragDrop.types.mod'),
        'texture': t('dragDrop.types.texture'),
        'furniture': t('dragDrop.types.furniture'),
        'savegame': t('dragDrop.types.savegame'),
        'skin': t('dragDrop.types.skin'),
        'modpack': t('dragDrop.types.modpack'),
        '': '',
      }
      message.warning(t('dragDrop.typeMismatch', {
        current: typeNames[currentResourceType.value],
        file: typeNames[resourceType]
      }))
      return
    }

    // 资源管理页面 → 直接导入
    console.log('[DragDrop] → Calling importToCurrentVersion')
    await importToCurrentVersion(resourceType, filePath, fileName)
  } else {
    // 非资源管理页面 → 弹窗选择已安装的游戏版本
    console.log('[DragDrop] → Not in resource page, showing version select dialog')
    pendingResourceType.value = resourceType
    pendingFilePath.value = filePath
    pendingFileName.value = fileName
    showVersionSelect.value = true
    console.log('[DragDrop] showVersionSelect set to:', showVersionSelect.value)
  }

  console.log('[DragDrop] ===== handleFileDrop END =====')
}

// 导入到当前版本（资源管理页面调用，用户已在页面上选了版本）
async function importToCurrentVersion(resourceType: ResourceType, filePath: string, fileName: string) {
  console.log('[DragDrop] importToCurrentVersion called:', { resourceType, filePath, fileName })

  if (resourceType === 'skin') {
    try {
      await ImportSkin(filePath)
      message.success(t('dragDrop.importSuccess', { file: fileName }))
      EventsEmit('dragdrop:imported', { resourceType, versionId: '' })
    } catch (error) {
      message.error(t('dragDrop.importFailed') + '：' + error)
    }
    return
  }

  const versionId = versionStore.selectedVersionId
  console.log('[DragDrop] selectedVersionId:', versionId)

  if (!versionId) {
    message.warning(t('dragDrop.noVersionSelected'))
    return
  }

  try {
    await importByType(resourceType, versionId, filePath)
    message.success(t('dragDrop.importSuccess', { file: fileName }))
    EventsEmit('dragdrop:imported', { resourceType, versionId })
  } catch (error) {
    message.error(t('dragDrop.importFailed') + '：' + error)
  }
}

// 用户选择了版本（从版本选择弹窗）
async function handleVersionSelect(versionId: string) {
  console.log('[DragDrop] handleVersionSelect called with versionId:', versionId)
  showVersionSelect.value = false

  if (!pendingFilePath.value) {
    console.log('[DragDrop] ❌ No pending file path')
    return
  }

  const fileName = pendingFileName.value
  const resourceType = pendingResourceType.value
  console.log('[DragDrop] pending:', { resourceType, filePath: pendingFilePath.value, fileName })

  if (resourceType === 'skin') {
    console.log('[DragDrop] → Calling ImportSkin')
    try {
      await ImportSkin(pendingFilePath.value)
      console.log('[DragDrop] ✅ ImportSkin success')
      message.success(t('dragDrop.importSuccess', { file: fileName }))
      EventsEmit('dragdrop:imported', { resourceType, versionId: '' })
    } catch (error) {
      console.error('[DragDrop] ❌ ImportSkin failed:', error)
      message.error(t('dragDrop.importFailed') + '：' + error)
    }
  } else {
    console.log('[DragDrop] → Calling importByType')
    try {
      await importByType(resourceType, versionId, pendingFilePath.value)
      console.log('[DragDrop] ✅ importByType success')
      message.success(t('dragDrop.importSuccess', { file: fileName }))
      EventsEmit('dragdrop:imported', { resourceType, versionId })
    } catch (error) {
      console.error('[DragDrop] ❌ importByType failed:', error)
      message.error(t('dragDrop.importFailed') + '：' + error)
    }
  }

  // 清理状态
  pendingFilePath.value = ''
  pendingFileName.value = ''
  pendingResourceType.value = ''
}

// 用户取消选择
function handleCancel() {
  showVersionSelect.value = false
  pendingFilePath.value = ''
  pendingFileName.value = ''
  pendingResourceType.value = ''
}

// ==================== 整合包安装 ====================

// 解析并显示整合包信息
async function startModpackInstall(filePath: string) {
  console.log('[DragDrop] startModpackInstall:', filePath)
  const loadingMsg = message.loading(t('installed.installingModpack') || '正在解析整合包...', { duration: 0 })
  try {
    const info = await ParseModpack(filePath)
    loadingMsg.destroy()
    parsedModpackInfo.value = info
    showModpackDialog.value = true
  } catch (error: any) {
    loadingMsg.destroy()
    const errorMsg = error?.message || error?.toString() || '未知错误'
    console.error('[DragDrop] ParseModpack failed:', error)
    dialog.error({
      title: t('installed.modpackParseFailed') || '解析整合包失败',
      content: errorMsg,
      positiveText: t('common.confirm'),
    })
  }
}

// 用户确认安装整合包
async function handleModpackInstall() {
  if (!parsedModpackInfo.value) return
  showModpackDialog.value = false
  showInstallDialog.value = true
  try {
    await InstallModpackWithProgress(parsedModpackInfo.value.filePath)
  } catch (error) {
    console.error('[DragDrop] Modpack install failed:', error)
    // 不关闭弹窗，让 ModpackInstallDialog 显示错误状态
  }
}

// 整合包安装完成
async function handleModpackCompleted(versionId: string) {
  message.success(t('installed.modpackInstallSuccess') || '整合包安装成功')
  showInstallDialog.value = false
  await versionStore.getVersions()
  await versionStore.getPrimaryVersion()
}

// 整合包安装失败
function handleModpackError(error: string) {
  console.error('[DragDrop] Modpack install error:', error)
  // 不关闭弹窗，让 ModpackInstallDialog 显示错误状态
}

// ==================== 游戏安装 ====================

// 弹窗输入别名后安装游戏
async function startGameInstall(archivePath: string) {
  const defaultName = archivePath.split('\\').pop()?.split('/').pop()?.replace(/\.(zip|7z|rar)$/i, '') || '本地安装的游戏'

  const customName = await new Promise<string | null>((resolve) => {
    let name = defaultName
    dialog.create({
      title: t('installed.enterVersionName') || '输入版本别名',
      content: () => h('div', [
        h('p', { style: 'margin-bottom: 12px;' }, t('installed.enterVersionNameDesc') || '为这个游戏版本起一个别名：'),
        h(NInput, {
          placeholder: defaultName,
          defaultValue: defaultName,
          onUpdateValue: (v: string) => { name = v },
        }),
      ]),
      positiveText: t('common.confirm'),
      negativeText: t('common.cancel'),
      onPositiveClick: () => resolve(name.trim() || defaultName),
      onNegativeClick: () => resolve(null),
      onClose: () => resolve(null),
    })
  })

  if (!customName) return

  const loadingMsg = message.loading(t('installed.installing') || '正在安装...', { duration: 0 })
  try {
    await InstallFromArchive(archivePath, customName)
    loadingMsg.destroy()
    message.success(t('installed.installSuccess') || '安装成功')
    await versionStore.getVersions()
    await versionStore.getPrimaryVersion()
  } catch (error) {
    loadingMsg.destroy()
    message.error((t('installed.installFailed') || '安装失败') + '：' + error)
  }
}

// 拖拽事件处理（视觉反馈）
function handleDragEnter(e: DragEvent) {
  e.preventDefault()
  dragCounter.value++
  if (dragCounter.value === 1) {
    isDragging.value = true
  }
}

function handleDragLeave(e: DragEvent) {
  e.preventDefault()
  dragCounter.value--
  if (dragCounter.value === 0) {
    isDragging.value = false
  }
}

function handleDragOver(e: DragEvent) {
  e.preventDefault()
}

// 阻止浏览器默认处理拖拽文件（如打开图片、触发下载）
function handleDrop(e: DragEvent) {
  console.log('[DragDrop] ⚡ HTML5 drop event (browser handler)')
  e.preventDefault()
  closeOverlay()
}

// 初始化
onMounted(() => {
  console.log('[DragDrop] Initializing drag and drop listeners')

  // 检查 Wails 运行时是否可用
  const hasRuntime = typeof window !== 'undefined' && 'runtime' in window
  console.log('[DragDrop] Wails runtime available:', hasRuntime)
  if (hasRuntime) {
    const rt = (window as any).runtime
    console.log('[DragDrop] window.runtime keys:', Object.keys(rt))
    console.log('[DragDrop] runtime.OnFileDrop type:', typeof rt.OnFileDrop)
    console.log('[DragDrop] runtime.OnFileDropOff type:', typeof rt.OnFileDropOff)
    console.log('[DragDrop] runtime.CanResolveFilePaths:', typeof rt.CanResolveFilePaths)
  }

  // useDropTarget = true：需要元素带有 --wails-drop-target CSS（在 App.vue 的 .app-box 上设置）
  // 前提：main.go 中必须启用 DragAndDrop: &options.DragAndDrop{EnableFileDrop: true}
  try {
    OnFileDrop((x, y, paths) => {
      console.log('[DragDrop] ✅ OnFileDrop triggered!', { x, y, paths })
      handleFileDrop(paths)
    }, true)
    console.log('[DragDrop] OnFileDrop registered (useDropTarget=true)')
  } catch (err) {
    console.error('[DragDrop] ❌ OnFileDrop registration failed:', err)
  }

  // HTML5 拖拽事件：视觉反馈 + 阻止浏览器默认行为
  document.addEventListener('dragenter', handleDragEnter)
  document.addEventListener('dragleave', handleDragLeave)
  document.addEventListener('dragover', handleDragOver)
  document.addEventListener('drop', handleDrop)

  console.log('[DragDrop] All listeners registered')
})

onUnmounted(() => {
  OnFileDropOff()
  document.removeEventListener('dragenter', handleDragEnter)
  document.removeEventListener('dragleave', handleDragLeave)
  document.removeEventListener('dragover', handleDragOver)
  document.removeEventListener('drop', handleDrop)
})
</script>

<style scoped>
.drag-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background-color: rgba(88, 101, 242, 0.15);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  pointer-events: none;
}

.drag-content {
  text-align: center;
  padding: 48px;
  background-color: var(--color-surface, #2B2D31);
  border: 2px dashed var(--color-primary, #5865F2);
  border-radius: 16px;
  max-width: 400px;
}

.drag-content :deep(.n-icon) {
  color: var(--color-primary, #5865F2);
  margin-bottom: 16px;
}

.drag-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--color-text-primary, #F2F3F5);
  margin-bottom: 8px;
}

.drag-hint {
  font-size: 14px;
  color: var(--color-text-tertiary, #949BA4);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
