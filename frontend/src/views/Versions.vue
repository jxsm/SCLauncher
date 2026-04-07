<template>
  <div class="versions-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <VersionToolbar
        :loading="loading"
        :filter-type="filterType"
        :type-options="typeOptions"
        :total-versions="totalVersions"
        @refresh="handleFetchVersions"
        @update:filter-type="filterType = $event"
      />

      <!-- 版本列表 -->
      <n-spin :show="loading">
        <n-collapse v-if="groupedVersions.length > 0">
          <VersionGroup
            v-for="group in groupedVersions"
            :key="group.gameVersion"
            :game-version="group.gameVersion"
            :versions="group.versions"
            :is-downloading="isDownloading"
            :get-download-progress="getDownloadProgress"
            @download="handleDownload"
          />
        </n-collapse>
        <n-empty v-if="groupedVersions.length === 0 && !loading" :description="t('versions.noVersions')" />
      </n-spin>
    </n-space>

    <!-- 自定义版本名称对话框 -->
    <CustomVersionNameDialog
      v-model:visible="showCustomNameDialog"
      :default-name="customNameDefault"
      :existing-names="existingVersionNames"
      @confirm="handleCustomNameConfirm"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useVersionStore } from '../stores/version'
import { useMessage } from 'naive-ui'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import type { Version } from '../types/version'
import VersionToolbar from '../components/version/VersionToolbar.vue'
import VersionGroup from '../components/version/VersionGroup.vue'
import CustomVersionNameDialog from '../components/version/CustomVersionNameDialog.vue'

const { t } = useI18n()
const versionStore = useVersionStore()
const message = useMessage()

const loading = ref(false)
const filterType = ref<string>('api') // Default to plugin version
const installingVersions = ref<Set<string>>(new Set())
// Record completed downloads to prevent duplicate processing
const completedDownloads = ref<Set<string>>(new Set())
// Mapping from original ID to unique ID
const originalToUniqueId = ref<Record<string, string>>({})
// Manifest versions list (only contains original versions, not custom named ones)
const manifestVersions = ref<Version[]>([])

// Custom version name dialog state
const showCustomNameDialog = ref(false)
const customNameDefault = ref('')
const pendingDownloadVersion = ref<Version | null>(null)

const typeOptions = computed(() => [
  { label: t('versions.apiVersion'), value: 'api' },
  { label: t('versions.netVersion'), value: 'net' }
])

// Total versions count
const totalVersions = computed(() =>
  groupedVersions.value.reduce((sum, g) => sum + g.versions.length, 0)
)

// Helper function to check if a version is downloading
function isDownloading(id: string): boolean {
  const uniqueId = originalToUniqueId.value[id]

  // 如果这个版本已经完成了下载，返回 false
  if (uniqueId) {
    if (completedDownloads.value.has(uniqueId)) {
      return false
    }
  } else {
    if (completedDownloads.value.has(id)) {
      return false
    }
  }

  // 检查是否正在下载
  const isDownloadingOriginal = versionStore.downloading.has(id)
  const isDownloadingUnique = uniqueId && versionStore.downloading.has(uniqueId)

  return isDownloadingOriginal || Boolean(isDownloadingUnique)
}

// Helper function to get download progress for a version
function getDownloadProgress(id: string): number {
  const uniqueId = originalToUniqueId.value[id]

  // 如果这个版本已经完成了下载，返回0
  if (uniqueId) {
    if (completedDownloads.value.has(uniqueId)) {
      return 0
    }
    const progress = versionStore.downloadProgress[uniqueId] || 0
    return progress
  } else {
    if (completedDownloads.value.has(id)) {
      return 0
    }
    const progress = versionStore.downloadProgress[id] || 0
    return progress
  }
}

// Existing version names for validation
const existingVersionNames = computed(() =>
  versionStore.versions.filter(v => v.installed).map(v => v.name)
)

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

async function handleFetchVersions() {
  loading.value = true
  try {
    // 获取清单文件中的版本列表（只包含原始版本）
    const versions = await versionStore.fetchVersions()
    manifestVersions.value = versions
    message.success(t('versions.versionListUpdated'))
  } catch (error) {
    message.error(t('versions.loadFailed') + '：' + error)
  } finally {
    loading.value = false
  }
}

async function handleDownload(version: Version) {
  // Show custom name dialog
  pendingDownloadVersion.value = version
  customNameDefault.value = version.name
  showCustomNameDialog.value = true
}

async function handleCustomNameConfirm(customName: string) {
  const version = pendingDownloadVersion.value
  if (!version) {
    return
  }

  try {
    await versionStore.downloadVersionWithCustomName(version.id, customName)
  } catch (error) {
    message.error(t('versions.downloadFailed') + '：' + error)
  } finally {
    pendingDownloadVersion.value = null
  }
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
      message.success(t('versions.installComplete'))

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
      message.error(t('versions.installFailed') + '：' + error)

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
    message.error(t('versions.loadFailed') + '：' + error)
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
