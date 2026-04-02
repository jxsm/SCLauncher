<template>
  <div class="versions-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space justify="space-between">
          <n-space>
            <n-button type="primary" @click="handleFetchVersions" :loading="loading">
              <template #icon>
                <n-icon><RefreshIcon /></n-icon>
              </template>
              刷新版本列表
            </n-button>
            <n-select
              v-model:value="filterType"
              :options="typeOptions"
              style="width: 150px"
            />
          </n-space>
          <n-text depth="3">
            共 {{ groupedVersions.reduce((sum, g) => sum + g.versions.length, 0) }} 个版本可下载
          </n-text>
        </n-space>
      </n-card>

      <!-- 版本列表 -->
      <n-spin :show="loading">
        <n-collapse v-if="groupedVersions.length > 0">
          <n-collapse-item v-for="group in groupedVersions" :key="group.gameVersion">
            <template #header>
              <n-space align="center" justify="space-between" style="width: 100%">
                <n-text strong style="font-size: 16px">
                  {{ group.gameVersion }}
                </n-text>
                <n-tag size="small" type="info">
                  {{ group.versions.length }} 个版本
                </n-tag>
              </n-space>
            </template>

            <n-list hoverable clickable>
              <n-list-item v-for="version in group.versions" :key="version.id">
                <n-thing>
                  <template #header>
                    <n-space align="center">
                      <n-text strong>{{ version.subVersion }}</n-text>
                      <n-tag :type="getTypeColor(version.versionType)" size="small">
                        {{ getTypeText(version.versionType) }}
                      </n-tag>
                    </n-space>
                  </template>

                  <template #description>
                    <n-space vertical size="small">
                      <n-text depth="3">
                        大小: {{ formatSize(version.size) }}
                      </n-text>
                      <n-text v-if="version.illustrate" depth="3">
                        说明: {{ version.illustrate }}
                      </n-text>
                    </n-space>
                  </template>

                  <template #action>
                    <n-space>
                      <!-- Download button or progress -->
                      <n-button
                        v-if="!isDownloading(version.id)"
                        type="primary"
                        size="medium"
                        @click="handleDownload(version)"
                      >
                        <template #icon>
                          <n-icon><DownloadIcon /></n-icon>
                        </template>
                        下载
                      </n-button>
                      <n-progress
                        v-else
                        type="line"
                        :percentage="getDownloadProgress(version.id)"
                        :indicator-placement="'inside'"
                        processing
                        style="width: 200px"
                      />
                    </n-space>
                  </template>
                </n-thing>
              </n-list-item>
            </n-list>
          </n-collapse-item>
        </n-collapse>
        <n-empty v-if="groupedVersions.length === 0 && !loading" description="暂无版本" />
      </n-spin>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, h } from 'vue'
import { useVersionStore } from '../stores/version'
import { useMessage, useDialog, NInput } from 'naive-ui'
import { Refresh as RefreshIcon, CloudDownload as DownloadIcon } from '@vicons/ionicons5'
import { formatSize } from '../utils/format'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import type { Version } from '../types/version'

const versionStore = useVersionStore()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const filterType = ref<string>('api') // Default to plugin version
const installingVersions = ref<Set<string>>(new Set())
// Record completed downloads to prevent duplicate processing
const completedDownloads = ref<Set<string>>(new Set())
// Mapping from original ID to unique ID
const originalToUniqueId = ref<Record<string, string>>({})
// Manifest versions list (only contains original versions, not custom named ones)
const manifestVersions = ref<Version[]>([])

const typeOptions = [
  { label: '插件版', value: 'api' },
  { label: '联机版', value: 'net' }
]

// 版本号比较函数，用于正确排序（例如：2.4 > 2.3 > 2.2）
function compareVersion(v1: string, v2: string): number {
  const parts1 = v1.split('.').map(Number)
  const parts2 = v2.split('.').map(Number)

  for (let i = 0; i < Math.max(parts1.length, parts2.length); i++) {
    const num1 = parts1[i] || 0
    const num2 = parts2[i] || 0
    if (num1 !== num2) {
      return num2 - num1 // 降序
    }
  }
  return 0
}

// Group versions by game version (2.31, 2.4, etc.)
const groupedVersions = computed(() => {
  let versions = manifestVersions.value

  // Filter by selected type
  versions = versions.filter(v => v.versionType === filterType.value)

  // Group by gameVersion
  const groups: Record<string, typeof versions> = {}
  versions.forEach(v => {
    if (!groups[v.gameVersion]) {
      groups[v.gameVersion] = []
    }
    groups[v.gameVersion].push(v)
  })

  // Sort versions within each group by subVersion (descending)
  Object.keys(groups).forEach(gameVersion => {
    groups[gameVersion].sort((a, b) => {
      const extractSubVersionNumbers = (subVersion: string): number[] => {
        const matches = subVersion.match(/(\d+(\.\d+)?)/g)
        return matches ? matches.map(v => v.split('.').map(Number)).flat() : [0]
      }

      const subNumbersA = extractSubVersionNumbers(a.subVersion)
      const subNumbersB = extractSubVersionNumbers(b.subVersion)

      for (let i = 0; i < Math.max(subNumbersA.length, subNumbersB.length); i++) {
        const numA = subNumbersA[i] || 0
        const numB = subNumbersB[i] || 0
        if (numA !== numB) {
          return numB - numA // Descending
        }
      }
      return 0
    })
  })

  // Sort groups by gameVersion (descending) and convert to array
  return Object.keys(groups)
    .sort((a, b) => compareVersion(a, b))
    .map(gameVersion => ({
      gameVersion,
      versions: groups[gameVersion]
    }))
})

function getTypeText(type: string): string {
  const types = {
    api: '插件版',
    net: '联机版',
    original: '原版'
  }
  return types[type as keyof typeof types] || type
}

function getTypeColor(type: string): 'info' | 'success' | 'warning' | 'default' {
  switch (type) {
    case 'api': return 'info'
    case 'net': return 'warning'
    case 'original': return 'success'
    default: return 'default'
  }
}

function isDownloading(id: string): boolean {
  const uniqueId = originalToUniqueId.value[id]

  // 如果这个版本已经完成了下载，返回 false
  if (uniqueId) {
    // 有映射时，检查 uniqueId 是否已完成
    if (completedDownloads.value.has(uniqueId)) {
      console.log(`[isDownloading] id=${id}, uniqueId=${uniqueId}, completed=true, returning false`)
      return false
    }
  } else {
    // 没有映射时，检查原始ID是否已完成
    if (completedDownloads.value.has(id)) {
      console.log(`[isDownloading] id=${id}, completed=true, returning false`)
      return false
    }
  }

  // 检查是否正在下载
  const isDownloadingOriginal = versionStore.downloading.has(id)
  const isDownloadingUnique = uniqueId && versionStore.downloading.has(uniqueId)

  // 调试日志
  if (isDownloadingOriginal || isDownloadingUnique) {
    console.log(`[isDownloading] id=${id}, uniqueId=${uniqueId}, isDownloadingOriginal=${isDownloadingOriginal}, isDownloadingUnique=${isDownloadingUnique}`)
  }

  return isDownloadingOriginal || Boolean(isDownloadingUnique)
}

function getDownloadProgress(id: string): number {
  const uniqueId = originalToUniqueId.value[id]

  // 如果这个版本已经完成了下载，返回0（即使有进度数据也不显示）
  if (uniqueId) {
    // 有映射时，检查 uniqueId 是否已完成
    if (completedDownloads.value.has(uniqueId)) {
      console.log(`[getDownloadProgress] id=${id}, uniqueId=${uniqueId}, completed=true, returning 0`)
      return 0
    }
    // 返回 uniqueId 的进度
    const progress = versionStore.downloadProgress[uniqueId] || 0
    if (progress > 0) {
      console.log(`[getDownloadProgress] id=${id}, uniqueId=${uniqueId}, progress=${progress}`)
    }
    return progress
  } else {
    // 没有映射时，检查原始ID是否已完成
    if (completedDownloads.value.has(id)) {
      console.log(`[getDownloadProgress] id=${id}, completed=true, returning 0`)
      return 0
    }
    // 返回原始ID的进度
    const progress = versionStore.downloadProgress[id] || 0
    if (progress > 0) {
      console.log(`[getDownloadProgress] id=${id}, progress=${progress}`)
    }
    return progress
  }
}

async function handleFetchVersions() {
  loading.value = true
  try {
    // 获取清单文件中的版本列表（只包含原始版本）
    const versions = await versionStore.fetchVersions()
    manifestVersions.value = versions
    message.success('版本列表已更新')
  } catch (error) {
    message.error('获取版本列表失败：' + error)
  } finally {
    loading.value = false
  }
}

async function handleDownload(version: Version) {
  const customName = await getCustomVersionName(version.name)
  if (!customName) {
    return
  }

  try {
    await versionStore.downloadVersionWithCustomName(version.id, customName)
  } catch (error) {
    message.error('下载失败：' + error)
  }
}

async function getCustomVersionName(defaultName: string): Promise<string | null> {
  return new Promise((resolve) => {
    let name = defaultName
    let errorMessage = ''

    function checkDuplicate(inputName: string): boolean {
      const trimmed = inputName.trim()
      if (!trimmed) return false
      return versionStore.versions.some(v =>
        v.installed && v.name === trimmed
      )
    }

    const d = dialog.create({
      title: '输入版本名称',
      content: () => {
        return h('div', [
          h('p', { style: 'margin-bottom: 12px;' }, '请输入这个版本的名称（用于区分不同配置）'),
          h(NInput, {
            placeholder: defaultName,
            defaultValue: defaultName,
            status: errorMessage ? 'error' : undefined,
            onUpdateValue: (value: string) => {
              name = value
              if (checkDuplicate(value)) {
                errorMessage = '该名称已存在，请使用其他名称'
              } else {
                errorMessage = ''
              }
            },
            onKeyup: (e: KeyboardEvent) => {
              if (e.key === 'Enter') {
                if (checkDuplicate(name)) {
                  errorMessage = '该名称已存在，请使用其他名称'
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
      positiveText: '确定',
      negativeText: '取消',
      onPositiveClick: () => {
        if (checkDuplicate(name)) {
          errorMessage = '该名称已存在，请使用其他名称'
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

function handleDownloadProgress(data: any) {
  const { versionId, downloaded, total, originalId } = data

  // 如果这个下载已经完成了，忽略后续进度事件
  if (completedDownloads.value.has(versionId)) {
    console.log(`[Download] Ignoring progress for completed download: ${versionId}`)
    return
  }

  const progress = Math.floor((downloaded / total) * 100)

  // 只在未完成时更新进度
  if (!completedDownloads.value.has(versionId)) {
    versionStore.updateDownloadProgress(versionId, progress)
  }
}

function handleDownloadComplete(data: any) {
  const { versionId, originalId } = data

  // 防止重复处理
  if (installingVersions.value.has(versionId)) {
    console.log(`[Download] Already installing: ${versionId}`)
    return
  }

  // 先标记为正在安装，防止重复处理
  installingVersions.value.add(versionId)
  completedDownloads.value.add(versionId)

  // 立即停止显示下载进度（删除 uniqueId）
  versionStore.finishDownload(versionId)

  // 同时也尝试删除原始ID（如果存在）
  if (originalId) {
    versionStore.finishDownload(originalId)
  }

  console.log(`[Download] Version ${versionId} completed, starting installation...`)

  versionStore.installVersion(versionId)
    .then(async () => {
      message.success('安装完成！')

      // 清理所有相关的进度状态
      versionStore.clearDownloadProgress(versionId)

      // 清理映射和原始ID的进度数据
      if (originalId && originalToUniqueId.value[originalId] === versionId) {
        delete originalToUniqueId.value[originalId]
        versionStore.clearDownloadProgress(originalId)
        console.log(`[Download] Cleaned mapping and progress for: ${originalId}`)
      }

      installingVersions.value.delete(versionId)
      completedDownloads.value.delete(versionId)

      // 注意：不需要刷新 manifestVersions，因为它只包含清单文件中的原始版本
      // 用户可以点击"刷新版本列表"按钮来更新清单

      console.log(`[Download] Version ${versionId} installation completed`)
    })
    .catch((error) => {
      console.error(`[Download] Version ${versionId} installation failed:`, error)
      message.error('安装失败：' + error)

      // 清理所有相关的进度状态
      versionStore.clearDownloadProgress(versionId)

      // 清理映射和原始ID的进度数据
      if (originalId && originalToUniqueId.value[originalId] === versionId) {
        delete originalToUniqueId.value[originalId]
        versionStore.clearDownloadProgress(originalId)
      }

      installingVersions.value.delete(versionId)
      completedDownloads.value.delete(versionId)
    })
}

function handleDownloadStart(data: any) {
  const { originalId, uniqueId } = data
  if (originalId && uniqueId) {
    // 清理旧的状态（如果有）
    completedDownloads.value.delete(uniqueId)
    installingVersions.value.delete(uniqueId)
    versionStore.clearDownloadProgress(uniqueId)

    // 清理原始ID的旧数据（如果有）
    versionStore.clearDownloadProgress(originalId)

    originalToUniqueId.value[originalId] = uniqueId
    console.log(`[Download] Mapping ${originalId} -> ${uniqueId}`)
  }
}

onMounted(async () => {
  EventsOn('download:start', handleDownloadStart)
  EventsOn('download:progress', handleDownloadProgress)
  EventsOn('download:complete', handleDownloadComplete)

  // 初始化清单版本列表
  loading.value = true
  try {
    const versions = await versionStore.fetchVersions()
    manifestVersions.value = versions
  } catch (error) {
    message.error('加载版本列表失败：' + error)
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  EventsOff('download:start')
  EventsOff('download:progress')
  EventsOff('download:complete')
})
</script>

<style scoped>
.versions-view {
  max-width: 1000px;
  margin: 0 auto;
}
</style>
