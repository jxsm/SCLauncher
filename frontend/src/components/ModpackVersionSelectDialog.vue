<template>
  <n-modal
    :show="show"
    :mask-closable="!busy"
    :close-on-esc="!busy"
    @update:show="(val: boolean) => emit('update:show', val)"
  >
    <n-card
      style="width: 560px"
      :title="t('modpacks.selectVersion')"
      :bordered="false"
      size="small"
      role="dialog"
      closable
      @close="handleClose"
    >
      <!-- 整合包声明的适配游戏版本提示 -->
      <n-alert v-if="preferredText" type="info" :bordered="false" style="margin-bottom: 12px">
        {{ t('modpacks.preferredVersionHint', { version: preferredText }) }}
      </n-alert>

      <!-- 模式一：选择已安装版本 -->
      <div v-if="mode === 'select'">
        <p class="dialog-desc">
          {{ installedOptions.length > 0 ? t('modpacks.selectVersionDesc') : t('modpacks.noInstalledButCanDownload') }}
        </p>
        <n-select
          v-if="installedOptions.length > 0"
          v-model:value="selectedInstalledId"
          :options="installedOptions"
          :loading="loadingInstalled"
        />

        <!-- 下载新版本入口 -->
        <n-button
          block
          dashed
          type="info"
          style="margin-top: 16px"
          @click="enterDownloadMode"
        >
          <template #icon>
            <n-icon><CloudDownloadIcon /></n-icon>
          </template>
          {{ t('modpacks.downloadNewVersion') }}
        </n-button>
      </div>

      <!-- 模式二：下载新版本 -->
      <div v-else>
        <n-button text size="small" style="margin-bottom: 8px" :disabled="busy" @click="backToSelectMode">
          <template #icon>
            <n-icon><ArrowBackIcon /></n-icon>
          </template>
          {{ t('modpacks.backToInstalled') }}
        </n-button>

        <n-spin :show="loadingOnline">
          <n-select
            v-model:value="selectedOnlineId"
            :options="onlineOptions"
            :placeholder="t('modpacks.onlineVersions')"
            :disabled="busy"
          />
        </n-spin>
        <n-text v-if="modpackName" depth="3" style="display: block; margin-top: 8px; font-size: 12px">
          {{ t('modpacks.newVersionNamedAsPack', { name: modpackName }) }}
        </n-text>
        <p v-if="!loadingOnline && onlineOptions.length === 0" class="dialog-desc" style="margin-top: 8px">
          {{ t('modpacks.loadOnlineVersionsFailed') }}
        </p>

        <!-- 下载/安装进度 -->
        <div v-if="busy" style="margin-top: 16px">
          <n-text depth="3" style="font-size: 12px">
            {{ phase === 'downloading' ? t('modpacks.downloadingVersion') : t('modpacks.installingVersion') }}
          </n-text>
          <n-progress
            type="line"
            :percentage="progress"
            :indicator-placement="'inside'"
            :status="phase === 'installing' ? 'success' : 'default'"
            processing
          />
        </div>
      </div>

      <template #footer>
        <n-space justify="end">
          <n-button
            v-if="mode === 'download' && busy"
            type="error"
            @click="cancelDownload"
          >
            {{ t('common.cancel') }}
          </n-button>
          <n-button :disabled="busy" @click="handleClose">
            {{ t('common.cancel') }}
          </n-button>
          <n-button
            v-if="mode === 'select'"
            type="primary"
            :disabled="!selectedInstalledId"
            @click="confirmInstalled"
          >
            {{ t('common.confirm') }}
          </n-button>
          <n-button
            v-else
            type="primary"
            :loading="busy"
            :disabled="!selectedOnlineId || loadingOnline"
            @click="startDownload"
          >
            {{ t('modpacks.downloadAndInstall') }}
          </n-button>
        </n-space>
      </template>
    </n-card>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import { CloudDownloadOutline as CloudDownloadIcon, ArrowBackOutline as ArrowBackIcon } from '@vicons/ionicons5'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { useVersionStore } from '../stores/version'
import { CancelDownload } from '../api/version'
import type { Version } from '../types/version'

const props = defineProps<{
  show: boolean
  /** 整合包声明的适配游戏版本名（如 ["2.4"]）；非空时优先预选匹配版本，空则完全由用户选择 */
  preferredGameVersions?: string[]
  /** 整合包名称；下载新版本时新版本默认以它命名，便于区分（空则用清单默认名） */
  modpackName?: string
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  'confirm': [versionId: string]
}>()

const { t } = useI18n()
const message = useMessage()
const versionStore = useVersionStore()

// select = 选已安装版本；download = 下载新版本
const mode = ref<'select' | 'download'>('select')
const selectedInstalledId = ref('')
const selectedOnlineId = ref('')
const loadingInstalled = ref(false)
const loadingOnline = ref(false)
// idle = 未在下载；downloading = 下载 zip；installing = 解压安装
const phase = ref<'idle' | 'downloading' | 'installing'>('idle')
const progress = ref(0)
// 自定义名下载路径的后端新版本 ID（<原ID>-<纳秒时间戳>），从 download:start 事件捕获
const activeDownloadId = ref('')

// 已安装版本快照：打开弹窗时从 DB 拉取后固化，弹窗期间不受下载/安装
// 过程中列表变化影响（fetchVersions 现在写 manifestVersions，不再覆盖 versions）
const installedList = ref<Version[]>([])

const busy = computed(() => phase.value !== 'idle')

const hasPreference = computed(() => (props.preferredGameVersions?.length ?? 0) > 0)
const preferredText = computed(() => (props.preferredGameVersions || []).join(' / '))

const installedOptions = computed(() =>
  installedList.value
    .filter(v => v.pathExists !== false)
    .map(v => ({ label: `${v.name} (${v.id})`, value: v.id }))
)

const onlineOptions = computed(() =>
  versionStore.manifestVersions
    .filter(v => !v.installed)
    .map(v => ({ label: `${v.name} (${v.id})`, value: v.id }))
)

// 版本是否匹配整合包声明的适配版本（gameVersion 相等，或名称/ID 包含）
function matchesPreference(v: Version): boolean {
  if (!hasPreference.value) return false
  const gameVersion = String(v.gameVersion || '').trim().toLowerCase()
  const name = String(v.name || '').toLowerCase()
  const id = String(v.id || '').toLowerCase()
  return (props.preferredGameVersions || []).some(p => {
    const q = p.trim().toLowerCase()
    if (!q) return false
    return (gameVersion !== '' && gameVersion === q) || name.includes(q) || id.includes(q)
  })
}

// 打开时刷新并快照已安装列表；有适配声明时预选匹配版本（无匹配已装则进下载模式）
watch(() => props.show, async (val) => {
  if (!val) return
  mode.value = 'select'
  phase.value = 'idle'
  progress.value = 0

  loadingInstalled.value = true
  try {
    await versionStore.getVersions()
    await versionStore.getPrimaryVersion()
    // 快照固化（pathExists 未检测(undefined) 视为可用）
    installedList.value = versionStore.installedVersions.filter(v => v.pathExists !== false)
  } finally {
    loadingInstalled.value = false
  }

  selectedInstalledId.value =
    versionStore.primaryVersion?.id || installedList.value[0]?.id || ''

  // 整合包声明了适配版本：默认选匹配的已装版本；没有匹配的已装则直接引导下载新版本
  //（enterDownloadMode 会优先预选匹配的在线版本；在线清单拉取失败时留在选择模式由用户自选）
  if (hasPreference.value) {
    const matched = installedList.value.find(matchesPreference)
    if (matched) {
      selectedInstalledId.value = matched.id
    } else {
      await enterDownloadMode()
    }
  } else if (installedList.value.length === 0) {
    // 无声明且无已装版本：直接进入下载模式
    await enterDownloadMode()
  }
})

async function enterDownloadMode() {
  mode.value = 'download'
  // 在线清单走 manifest 缓存（30 分钟）
  loadingOnline.value = true
  try {
    await versionStore.fetchVersions()
  } catch (error) {
    message.error(t('modpacks.loadOnlineVersionsFailed') + '：' + error)
  } finally {
    loadingOnline.value = false
  }
  // 默认选第一个；有适配声明则优先选匹配项
  const preferred = versionStore.manifestVersions.find(v => !v.installed && matchesPreference(v))
  selectedOnlineId.value = preferred?.id || onlineOptions.value[0]?.value || ''
}

function backToSelectMode() {
  if (busy.value) return
  mode.value = 'select'
  // 已装列表可能因安装/下载而变化，重校验选中项
  if (!installedList.value.some(v => v.id === selectedInstalledId.value)) {
    selectedInstalledId.value =
      versionStore.primaryVersion?.id || installedList.value[0]?.id || ''
  }
}

function confirmInstalled() {
  if (!selectedInstalledId.value) return
  const id = selectedInstalledId.value
  emit('update:show', false)
  emit('confirm', id)
}

async function startDownload() {
  if (!selectedOnlineId.value || busy.value) return
  phase.value = 'downloading'
  progress.value = 0
  activeDownloadId.value = ''
  const customName = (props.modpackName || '').trim()
  try {
    if (customName) {
      // 自定义名路径：后端以整合包名新建版本记录（新版本 ID 从 download:start 事件捕获）
      await versionStore.downloadVersionWithCustomName(selectedOnlineId.value, customName)
    } else {
      await versionStore.downloadVersion(selectedOnlineId.value)
    }
    const newId = activeDownloadId.value || selectedOnlineId.value
    phase.value = 'installing'
    progress.value = 0
    // 解压安装为已安装版本（store 内部会刷新主版本）
    await versionStore.installVersion(newId)
    await versionStore.getVersions()

    phase.value = 'idle'
    progress.value = 0
    activeDownloadId.value = ''
    emit('update:show', false)
    emit('confirm', newId)
  } catch (error: any) {
    const failed = phase.value === 'downloading'
      ? t('modpacks.versionDownloadFailed')
      : t('modpacks.versionInstallFailed')
    message.error(failed + '：' + error)
    phase.value = 'idle'
    progress.value = 0
  } finally {
    // 清理 store 下载状态（本组件不监听 download:complete，避免其他页面看到残留的下载中状态）
    versionStore.finishDownload(selectedOnlineId.value)
  }
}

async function cancelDownload() {
  if (!busy.value) return
  try {
    await CancelDownload(activeDownloadId.value || selectedOnlineId.value)
  } catch (error) {
    console.warn('取消下载失败:', error)
    // 即使取消调用失败也重置状态（下载 Promise 可能已 reject）
    phase.value = 'idle'
    progress.value = 0
  }
}

function handleClose() {
  if (busy.value) return
  emit('update:show', false)
}

// 下载/安装进度事件（组件存活期间注册；phase 门槛保证多实例只有发起者响应）
// download:start 捕获自定义名下载路径的新版本 ID（进度/安装事件用它标识）
function handleDownloadStart(data: { originalId?: string; uniqueId?: string }) {
  if (phase.value === 'downloading' && data.originalId === selectedOnlineId.value && data.uniqueId) {
    activeDownloadId.value = data.uniqueId
  }
}

function handleDownloadProgress(data: { versionId?: string; downloaded?: number; total?: number }) {
  if (phase.value !== 'downloading' || (data.total ?? 0) <= 0) return
  // 普通路径 versionId=原ID；自定义名路径 versionId=新ID
  if (data.versionId === activeDownloadId.value || data.versionId === selectedOnlineId.value) {
    progress.value = Math.min(100, Math.floor(((data.downloaded ?? 0) / (data.total ?? 1)) * 100))
  }
}

function handleInstallProgress(data: { versionId?: string; current?: number; total?: number }) {
  if (phase.value !== 'installing' || (data.total ?? 0) <= 0) return
  if (data.versionId === activeDownloadId.value || data.versionId === selectedOnlineId.value) {
    progress.value = Math.min(100, Math.floor(((data.current ?? 0) / (data.total ?? 1)) * 100))
  }
}

onMounted(() => {
  EventsOn('download:start', handleDownloadStart)
  EventsOn('download:progress', handleDownloadProgress)
  EventsOn('install:progress', handleInstallProgress)
})

onUnmounted(() => {
  // EventsOff 只接收事件名（可一次传多个）
  EventsOff('download:start', 'download:progress', 'install:progress')
})
</script>

<style scoped>
.dialog-desc {
  margin: 0 0 12px 0;
  font-size: 14px;
}
</style>
