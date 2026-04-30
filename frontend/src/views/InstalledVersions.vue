<template>
  <div class="installed-versions-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <VersionToolbar
        :total-count="filteredVersions.length"
        :filter-type="filterType"
        :type-options="typeOptions"
        @update:filter-type="filterType = $event"
        @import-game="handleImportGame"
        @local-install="handleLocalInstall"
        @install-modpack="handleInstallModpack"
      />

      <!-- 版本列表 -->
      <n-spin :show="loading">
        <n-list hoverable clickable>
          <VersionListItem
            v-for="version in filteredVersions"
            :key="version.id"
            :version="version"
            :is-path-missing="isPathMissing(version)"
            :is-game-running="gameStore.isRunning"
            @launch="handleLaunch"
            @set-primary="handleSetPrimary"
            @open-folder="handleOpenFolder"
            @manage-resources="handleManageResources"
            @rename="handleRename"
            @delete="handleDelete"
          />
        </n-list>
        <n-empty v-if="filteredVersions.length === 0 && !loading" :description="t('installed.noVersions')">
          <template #extra>
            <n-button type="primary" @click="$router.push('/versions')">
              {{ t('installed.goToVersions') }}
            </n-button>
          </template>
        </n-empty>
      </n-spin>

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
      @completed="handleInstallCompleted"
      @error="handleInstallError"
    />
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useVersionStore } from '../stores/version'
import { useGameStore } from '../stores/game'
import { useMessage, useDialog, NInput } from 'naive-ui'
import { OpenVersionFolder, SelectGameFolder, ImportGameVersion, SelectArchiveFile, InstallFromArchive, SelectModpackFile, InstallModpack, ParseModpack, InstallModpackWithProgress } from '../api/version'
import ModpackInfoDialog from '../components/ModpackInfoDialog.vue'
import ModpackInstallDialog from '../components/ModpackInstallDialog.vue'
import VersionToolbar from '../components/VersionToolbar.vue'
import VersionListItem from '../components/VersionListItem.vue'
import type { Version } from '../types/version'

const { t } = useI18n()
const versionStore = useVersionStore()
const gameStore = useGameStore()
const message = useMessage()
const dialog = useDialog()
const router = useRouter()

const loading = ref(false)
const renamingVersion = ref<Version | null>(null)
const newName = ref('')
const filterType = ref<string>('all')
const showModpackDialog = ref(false)
const showInstallDialog = ref(false)
const parsedModpackInfo = ref<any>(null)

const installedVersions = computed(() => versionStore.installedVersions)

const typeOptions = computed(() => [
  { label: t('versions.all'), value: 'all' },
  { label: t('versions.apiVersion'), value: 'api' },
  { label: t('versions.netVersion'), value: 'net' },
  { label: t('versions.originalVersion'), value: 'original' },
  { label: t('versions.modifiedVersion'), value: 'modified' },
  { label: t('versions.customVersion'), value: 'custom' }
])

const filteredVersions = computed(() => {
  if (filterType.value === 'all') {
    return installedVersions.value
  }
  if (filterType.value === 'custom') {
    return installedVersions.value.filter(v => v.isCustomName)
  }
  return installedVersions.value.filter(v => v.versionType === filterType.value)
})

// 判断路径是否缺失
function isPathMissing(version: Version): boolean {
  // 调试日志
  console.log('[isPathMissing] Checking version:', version.id, 'pathExists:', version.pathExists, 'type:', typeof version.pathExists)

  // pathExists 为 false 或 undefined 表示路径不存在（如果字段不存在，默认认为不存在）
  return version.pathExists === false || version.pathExists === undefined
}

async function handleLaunch(version: Version) {
  try {
    await gameStore.launchGame(version.id)
    message.success(`${t('installed.launchSuccess')}: "${version.name}"`)
  } catch (error) {
    message.error(t('installed.launchFailed') + '：' + error)
  }
}

async function handleSetPrimary(version: Version) {
  try {
    await versionStore.setPrimaryVersion(version.id)
    message.success(`${t('installed.setPrimarySuccess')}: "${version.name}"`)
  } catch (error) {
    message.error(t('installed.setPrimaryFailed') + '：' + error)
  }
}

function handleRename(version: Version) {
  renamingVersion.value = version
  newName.value = version.name

  dialog.create({
    title: t('installed.renameVersion'),
    content: () => {
      return h('div', [
        h('div', { style: 'margin-bottom: 8px' }, t('installed.enterNewVersionName')),
        h(NInput, {
          value: newName.value,
          placeholder: t('installed.enterVersionName'),
          onUpdateValue: (value: string) => {
            newName.value = value
          },
          onKeyup: (e: KeyboardEvent) => {
            if (e.key === 'Enter') {
              // 按回车键确认
            }
          }
        })
      ])
    },
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      if (!newName.value.trim()) {
        message.error(t('installed.nameCannotBeEmpty'))
        return false
      }

      // 检查重名
      const exists = versionStore.installedVersions.some(
        v => v.name === newName.value && v.id !== version.id
      )
      if (exists) {
        message.error(t('installed.nameAlreadyExists'))
        return false
      }

      try {
        await versionStore.renameVersion(version.id, newName.value)
        message.success(`${t('installed.renameSuccess')}: "${version.name}" → "${newName.value}"`)
        renamingVersion.value = null
        return true
      } catch (error) {
        message.error(t('installed.renameFailed') + '：' + error)
        return false
      }
    }
  })
}

function handleDelete(version: Version) {
  versionStore.deleteVersion(version.id)
    .then(() => {
      message.success(`${t('installed.deleteSuccess')}: "${version.name}"`)
    })
    .catch((error) => {
      message.error(t('installed.deleteFailed') + '：' + error)
    })
}

async function handleOpenFolder(version: Version) {
  try {
    await OpenVersionFolder(version.id)
  } catch (error) {
    message.error(t('installed.openFolderFailed') + '：' + error)
  }
}

function handleManageResources(version: Version) {
  router.push({
    name: 'Resources',
    query: { versionId: version.id }
  })
}

async function handleImportGame() {
  try {
    // 选择游戏文件夹
    const folderPath = await SelectGameFolder()
    if (!folderPath) {
      return
    }

    // 显示正在导入的消息
    const loadingMsg = message.loading(t('installed.importing'), { duration: 0 })

    try {
      // 导入游戏版本
      const versionId = await ImportGameVersion(folderPath)

      loadingMsg.destroy()
      message.success(t('installed.importSuccess'))

      // 重新加载版本列表
      await versionStore.getVersions()
      await versionStore.getPrimaryVersion()
    } catch (error) {
      loadingMsg.destroy()
      message.error(t('installed.importFailed') + '：' + error)
    }
  } catch (error) {
    message.error(t('installed.selectFolderFailed') + '：' + error)
  }
}

async function handleLocalInstall() {
  try {
    // 选择压缩包文件
    const archivePath = await SelectArchiveFile()
    if (!archivePath) {
      return
    }

    // 获取自定义名称
    const defaultName = archivePath.split('\\').pop()?.split('/').pop()?.replace(/\.(zip|7z|rar)$/i, '') || '本地安装的游戏'
    const customName = await getCustomVersionName(defaultName)
    if (!customName) {
      return
    }

    // 显示正在安装的消息
    const loadingMsg = message.loading(t('installed.installing'), { duration: 0 })

    try {
      // 从压缩包安装游戏
      const versionId = await InstallFromArchive(archivePath, customName)

      loadingMsg.destroy()
      message.success(t('installed.installSuccess'))

      // 重新加载版本列表
      await versionStore.getVersions()
      await versionStore.getPrimaryVersion()
    } catch (error) {
      loadingMsg.destroy()
      message.error(t('installed.installFailed') + '：' + error)
    }
  } catch (error) {
    message.error(t('installed.selectArchiveFailed') + '：' + error)
  }
}

async function handleInstallModpack() {
  try {
    // 选择整合包文件
    const modpackPath = await SelectModpackFile()
    if (!modpackPath) {
      return
    }

    // 先在后台解析，不要立即打开弹窗
    const loadingMsg = message.loading(t('installed.installingModpack'), { duration: 0 })

    try {
      // 解析整合包
      const info = await ParseModpack(modpackPath)

      loadingMsg.destroy()

      // 解析成功，保存解析的数据并打开弹窗
      parsedModpackInfo.value = info
      showModpackDialog.value = true
    } catch (error: any) {
      loadingMsg.destroy()

      // 解析失败，显示错误对话框
      const errorMsg = error?.message || error?.toString() || '未知错误'
      console.error('解析整合包失败:', error)

      // 检查是否是平台不支持错误
      const isPlatformError = errorMsg.includes('暂不支持') || errorMsg.includes('does not support')

      dialog.create({
        title: isPlatformError ? t('installed.modpackNotSupportedWindows') : t('installed.modpackParseFailed'),
        content: () => {
          return h('div', [
            isPlatformError
              ? h('p', { style: 'margin-bottom: 12px; font-size: 14px; padding: 16px; background: var(--n-error-color); color: white; border-radius: 4px;' }, '该整合包仅支持 Android 平台，无法在 Windows 版启动器中安装。')
              : h('div', [
                  h('p', { style: 'margin-bottom: 12px;' }, '解析整合包时发生错误：'),
                  h('div', {
                    style: 'background: var(--n-code-color); padding: 12px; border-radius: 4px; font-family: monospace; font-size: 12px; white-space: pre-wrap; word-break: break-all;'
                  }, errorMsg)
                ])
          ])
        },
        positiveText: t('common.confirm')
      })
    }
  } catch (error) {
    message.error(t('installed.selectModpackFailed') + '：' + error)
  }
}

async function handleModpackInstall() {
  if (!parsedModpackInfo.value) {
    return
  }

  try {
    // 关闭信息对话框
    showModpackDialog.value = false

    // 打开安装进度对话框
    showInstallDialog.value = true

    // 使用新的安装方法（带进度反馈）
    await InstallModpackWithProgress(parsedModpackInfo.value.filePath)
  } catch (error) {
    message.error(t('installed.modpackInstallFailed') + '：' + error)
    showInstallDialog.value = false
  }
}

// 安装完成回调
async function handleInstallCompleted(versionId: string) {
  message.success(t('installed.modpackInstallSuccess'))

  // 重新加载版本列表
  await versionStore.getVersions()
  await versionStore.getPrimaryVersion()
}

// 安装错误回调
function handleInstallError(error: string) {
  console.error('整合包安装失败:', error)
  // 对话框会显示错误信息，用户可以手动关闭
}

async function getCustomVersionName(defaultName: string): Promise<string | null> {
  return new Promise((resolve) => {
    let name = defaultName
    let errorMessage = ''

    function checkDuplicate(inputName: string): boolean {
      const trimmed = inputName.trim()
      if (!trimmed) return false
      return versionStore.installedVersions.some(v =>
        v.name === trimmed
      )
    }

    const d = dialog.create({
      title: t('installed.enterVersionName'),
      content: () => {
        return h('div', [
          h('p', { style: 'margin-bottom: 12px;' }, t('installed.enterVersionNameDesc')),
          h(NInput, {
            placeholder: defaultName,
            defaultValue: defaultName,
            status: errorMessage ? 'error' : undefined,
            onUpdateValue: (value: string) => {
              name = value
              if (checkDuplicate(value)) {
                errorMessage = t('installed.nameAlreadyExists')
              } else {
                errorMessage = ''
              }
            },
            onKeyup: (e: KeyboardEvent) => {
              if (e.key === 'Enter') {
                if (checkDuplicate(name)) {
                  errorMessage = t('installed.nameAlreadyExists')
                } else {
                  resolve(name.trim() || null)
                }
              }
            }
          }),
          errorMessage ? h('p', {
            style: 'margin-top: 8px; color: #f56c6c; font-size: 12px;'
          }, errorMessage) : null
        ])
      },
      positiveText: t('common.confirm'),
      negativeText: t('common.cancel'),
      onPositiveClick: () => {
        if (checkDuplicate(name)) {
          errorMessage = t('installed.nameAlreadyExists')
        } else {
          resolve(name.trim() || null)
        }
      },
      onNegativeClick: () => {
        resolve(null)
      }
    })
  })
}

onMounted(async () => {
  loading.value = true
  try {
    await versionStore.getVersions()
    await versionStore.getPrimaryVersion()
  } catch (error) {
    message.error(t('errors.loadDataFailed') + '：' + error)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.installed-versions-view {
  max-width: 1000px;
  margin: 0 auto;
}
</style>
