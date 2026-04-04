<template>
  <div class="installed-versions-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space justify="space-between">
          <n-space>
            <n-text strong style="font-size: 18px;">{{ t('installed.title') }}</n-text>
          </n-space>
          <n-space>
            <n-button type="primary" @click="handleImportGame">
              <template #icon>
                <n-icon><ImportIcon /></n-icon>
              </template>
              {{ t('installed.importGame') }}
            </n-button>
            <n-button type="success" @click="handleLocalInstall">
              <template #icon>
                <n-icon><ArchiveIcon /></n-icon>
              </template>
              {{ t('installed.localInstall') }}
            </n-button>
            <n-text depth="3">
              {{ t('installed.totalInstalled') }} {{ installedVersions.length }}
            </n-text>
          </n-space>
        </n-space>
      </n-card>

      <!-- 版本列表 -->
      <n-spin :show="loading">
        <n-list hoverable clickable>
          <n-list-item
            v-for="version in installedVersions"
            :key="version.id"
            :style="isPathMissing(version) ? 'background-color: rgba(255, 0, 0, 0.05);' : ''"
          >
            <n-thing>
              <template #header>
                <n-space align="center">
                  <n-text strong style="font-size: 16px;">{{ version.name }}</n-text>
                  <n-tag v-if="version.isPrimary" type="success" size="small">
                    {{ t('installed.primary') }}
                  </n-tag>
                  <n-tag v-if="!isImportedVersion(version)" :type="getVersionTypeColor(version.versionType)" size="small">
                    {{ getVersionTypeText(version.versionType) }}
                  </n-tag>
                  <n-tag v-if="!isPathMissing(version)" type="success" size="small">
                    {{ t('versions.installed') }}
                  </n-tag>
                  <n-tag v-if="isPathMissing(version)" type="error" size="small">
                    {{ t('installed.pathMissing') }}
                  </n-tag>
                </n-space>
              </template>

              <template #description>
                <n-space vertical size="small">
                  <n-text depth="3">
                    {{ t('common.version') }}: {{ version.gameVersion }} - {{ version.subVersion }}
                  </n-text>
                  <n-text v-if="isPathMissing(version)" type="error" style="margin-top: 8px;">
                    ⚠️ {{ t('installed.pathMissingMessage') }}
                  </n-text>
                </n-space>
              </template>

              <template #action>
                <n-space>
                  <!-- 路径不存在时，隐藏所有按钮除了删除按钮 -->
                  <template v-if="!isPathMissing(version)">
                    <n-button
                      type="success"
                      size="medium"
                      :disabled="gameStore.isRunning"
                      @click="handleLaunch(version)"
                    >
                      <template #icon>
                        <n-icon><PlayIcon /></n-icon>
                      </template>
                      {{ t('installed.launchGame') }}
                    </n-button>
                    <n-button
                      size="medium"
                      @click="handleSetPrimary(version)"
                      :disabled="version.isPrimary"
                      :type="version.isPrimary ? 'success' : 'default'"
                    >
                      <template #icon>
                        <n-icon><StarIcon /></n-icon>
                      </template>
                      {{ version.isPrimary ? t('installed.alreadyPrimary') : t('versions.setAsPrimary') }}
                    </n-button>
                    <n-button
                      size="medium"
                      @click="handleOpenFolder(version)"
                    >
                      <template #icon>
                        <n-icon><FolderIcon /></n-icon>
                      </template>
                      {{ t('versions.openFolder') }}
                    </n-button>
                    <n-button
                      size="medium"
                      @click="handleManageMods(version)"
                    >
                      <template #icon>
                        <n-icon><ModsIcon /></n-icon>
                      </template>
                      {{ t('installed.manageMods') }}
                    </n-button>
                    <n-button
                      size="medium"
                      @click="handleRename(version)"
                    >
                      <template #icon>
                        <n-icon><EditIcon /></n-icon>
                      </template>
                      {{ t('versions.rename') }}
                    </n-button>
                  </template>
                  <n-popconfirm
                    @positive-click="handleDelete(version)"
                  >
                    <template #trigger>
                      <n-button type="error" size="medium">
                        <template #icon>
                          <n-icon><TrashIcon /></n-icon>
                        </template>
                        {{ t('common.delete') }}
                      </n-button>
                    </template>
                    {{ t('installed.confirmDeleteVersion', { name: version.name }) }}
                  </n-popconfirm>
                </n-space>
              </template>
            </n-thing>
          </n-list-item>
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
import { Play as PlayIcon, Star as StarIcon, Trash as TrashIcon, CreateOutline as EditIcon, FolderOpen as FolderIcon, ExtensionPuzzle as ModsIcon, Download as ImportIcon, Archive as ArchiveIcon } from '@vicons/ionicons5'
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

function getVersionTypeText(type: string): string {
  const types = {
    api: t('versions.apiVersion'),
    net: t('versions.netVersion'),
    original: t('versions.originalVersion')
  }
  return types[type as keyof typeof types] || type
}

function getVersionTypeColor(type: string): 'info' | 'success' | 'warning' | 'default' {
  switch (type) {
    case 'api': return 'info'
    case 'net': return 'warning'
    case 'original': return 'success'
    default: return 'default'
  }
}

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

// 判断是否为导入的版本
function isImportedVersion(version: Version): boolean {
  // 导入的版本ID以"imported-"开头，或者版本类型为"unknown"
  return version.id.startsWith('imported-') || version.versionType === 'unknown'
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
