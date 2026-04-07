<template>
  <div class="installed-versions-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <VersionToolbar
        :total-count="installedVersions.length"
        @import-game="handleImportGame"
        @local-install="handleLocalInstall"
      />

      <!-- 版本列表 -->
      <n-spin :show="loading">
        <n-list hoverable clickable>
          <VersionListItem
            v-for="version in installedVersions"
            :key="version.id"
            :version="version"
            :is-path-missing="isPathMissing(version)"
            :is-game-running="gameStore.isRunning"
            @launch="handleLaunch"
            @set-primary="handleSetPrimary"
            @open-folder="handleOpenFolder"
            @manage-mods="handleManageMods"
            @rename="handleRename"
            @delete="handleDelete"
          />
        </n-list>
        <n-empty v-if="installedVersions.length === 0 && !loading" :description="t('installed.noVersions')">
          <template #extra>
            <n-button type="primary" @click="$router.push('/versions')">
              {{ t('installed.goToVersions') }}
            </n-button>
          </template>
        </n-empty>
      </n-spin>
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
import { OpenVersionFolder, SelectGameFolder, ImportGameVersion, SelectArchiveFile, InstallFromArchive } from '../api/version'
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

const installedVersions = computed(() => versionStore.installedVersions)

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

function handleManageMods(version: Version) {
  router.push({
    name: 'Mods',
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
