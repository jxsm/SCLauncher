<template>
  <n-modal
    :show="showModal"
    @update:show="$emit('update:show', $event)"
    preset="card"
    :title="t('installed.installingModpack')"
    style="width: 600px; max-width: 90vw;"
    :mask-closable="false"
    :closable="status !== 'installing'"
  >
    <div class="install-progress">
      <!-- 当前阶段信息 -->
      <n-space vertical size="large">
        <!-- 整合包信息 -->
        <n-card size="small" v-if="modpackInfo">
          <n-space align="center">
            <n-icon size="24" :component="InformationCircle" />
            <div>
              <div class="modpack-name">{{ modpackInfo.name }}</div>
              <div class="modpack-version">{{ modpackInfo.version }}</div>
            </div>
          </n-space>
        </n-card>

        <!-- 总体进度 -->
        <div v-if="status === 'installing'">
          <n-space justify="space-between" :style="{ marginBottom: '8px' }">
            <n-text>{{ t('installed.installProgress') }}</n-text>
            <n-text strong>{{ currentProgress.toFixed(1) }}%</n-text>
          </n-space>
          <n-progress
            type="line"
            :percentage="currentProgress"
            :status="progressStatus"
            :height="20"
            :border-radius="4"
          />
        </div>

        <!-- 当前阶段详情 -->
        <n-card size="small" v-if="status === 'installing'">
          <n-space vertical size="small">
            <n-space justify="space-between">
              <n-text>{{ currentStageText }}</n-text>
              <n-text v-if="currentMessage" type="info" style="font-size: 12px;">
                {{ currentMessage }}
              </n-text>
            </n-space>

            <!-- 阶段进度条 -->
            <n-progress
              type="line"
              :percentage="stageProgress"
              :height="8"
              :border-radius="4"
              :show-indicator="false"
              processing
            />
          </n-space>
        </n-card>

        <!-- 安装阶段列表 -->
        <n-card size="small">
          <n-space vertical size="small">
            <n-text strong>{{ t('installed.installStages') }}</n-text>
            <n-list bordered size="small" style="max-height: 200px; overflow-y: auto;">
              <n-list-item v-for="stage in stages" :key="stage.id">
                <n-space align="center" justify="space-between" :style="{ width: '100%' }">
                  <n-space align="center">
                    <n-icon
                      :size="18"
                      :color="getStageColor(stage.status)"
                      :component="getStageIcon(stage.status)"
                    />
                    <n-text>{{ stage.label }}</n-text>
                  </n-space>
                  <n-tag v-if="stage.status === 'completed'" type="success" size="small">
                    {{ t('common.completed') }}
                  </n-tag>
                  <n-tag v-else-if="stage.status === 'running'" type="info" size="small">
                    {{ t('common.running') }}
                  </n-tag>
                  <n-tag v-else-if="stage.status === 'error'" type="error" size="small">
                    {{ t('common.error') }}
                  </n-tag>
                  <n-tag v-else type="default" size="small">
                    {{ t('common.pending') }}
                  </n-tag>
                </n-space>
              </n-list-item>
            </n-list>
          </n-space>
        </n-card>

        <!-- 错误信息 -->
        <n-alert v-if="status === 'error'" type="error" :title="t('common.error')">
          {{ errorMessage }}
        </n-alert>

        <!-- 成功信息 -->
        <n-alert v-if="status === 'completed'" type="success" :title="t('common.success')">
          {{ t('installed.modpackInstallSuccess') }}
        </n-alert>
      </n-space>
    </div>

    <template #footer>
      <n-space justify="end">
        <n-button v-if="status === 'error'" @click="handleClose">
          {{ t('common.close') }}
        </n-button>
        <n-button v-if="status === 'completed'" type="primary" @click="handleClose">
          {{ t('common.confirm') }}
        </n-button>
        <n-button v-if="status === 'installing'" type="error" @click="handleCancel">
          {{ t('common.cancel') }}
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { InformationCircle, CheckmarkCircle, Time as TimeCircle, AlertCircle, ReloadOutline as Loader } from '@vicons/ionicons5'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { CancelModpackInstall } from '../api/version'

const props = defineProps<{
  show: boolean
  modpackInfo?: any
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  'completed': [versionId: string]
  'error': [error: string]
}>()

const { t } = useI18n()

const showModal = ref(props.show)
const status = ref<'pending' | 'installing' | 'completed' | 'error'>('pending')
const currentProgress = ref(0)
const stageProgress = ref(0)
const currentStage = ref('')
const previousStage = ref('')
const currentMessage = ref('')
const errorMessage = ref('')
const versionId = ref('')

// 阶段定义
const stages = ref([
  { id: 'prepare', label: t('installed.stagePrepare'), status: 'pending' },
  { id: 'download_game', label: t('installed.stageDownloadGame'), status: 'pending' },
  { id: 'install_game', label: t('installed.stageInstallGame'), status: 'pending' },
  { id: 'download_mods', label: t('installed.stageDownloadMods'), status: 'pending' },
  { id: 'copy_overrides', label: t('installed.stageCopyOverrides'), status: 'pending' },
  { id: 'complete', label: t('installed.stageComplete'), status: 'pending' }
])

// 当前阶段文本
const currentStageText = computed(() => {
  const stage = stages.value.find(s => s.id === currentStage.value)
  return stage?.label || currentStage.value
})

// 进度状态
const progressStatus = computed(() => {
  if (status.value === 'error') return 'error'
  if (status.value === 'completed') return 'success'
  return 'default'
})

// 获取阶段图标
function getStageIcon(stageStatus: string) {
  switch (stageStatus) {
    case 'completed':
      return CheckmarkCircle
    case 'running':
      return Loader
    case 'error':
      return AlertCircle
    default:
      return TimeCircle
  }
}

// 获取阶段颜色
function getStageColor(stageStatus: string) {
  switch (stageStatus) {
    case 'completed':
      return '#34c759'
    case 'running':
      return '#0066cc'
    case 'error':
      return '#ff3b30'
    default:
      return '#7a7a7a'
  }
}

// 更新阶段状态
function updateStageStatus(stageId: string, newStatus: 'pending' | 'running' | 'completed' | 'error') {
  const stage = stages.value.find(s => s.id === stageId)
  if (stage) {
    stage.status = newStatus
  }
}

// 监听显示状态
watch(() => props.show, (newVal) => {
  showModal.value = newVal
  if (newVal) {
    // 重置状态
    status.value = 'installing'
    currentProgress.value = 0
    stageProgress.value = 0
    currentStage.value = ''
    previousStage.value = ''
    currentMessage.value = ''
    errorMessage.value = ''
    stages.value.forEach(s => s.status = 'pending')
  }
})

// 处理进度更新
function handleProgress(data: { stage: string; progress: number; message: string }) {
  // 如果是新阶段，将上一个阶段标记为完成
  if (previousStage.value && previousStage.value !== data.stage) {
    updateStageStatus(previousStage.value, 'completed')
  }
  previousStage.value = data.stage

  currentStage.value = data.stage
  stageProgress.value = data.progress
  currentMessage.value = data.message

  // 更新阶段状态
  updateStageStatus(data.stage, 'running')

  // 如果当前阶段进度达到100%，标记为完成
  if (data.progress >= 100) {
    updateStageStatus(data.stage, 'completed')
  }

  // 改进的总体进度计算：基于阶段的实际权重
  // 不同阶段的时间差异很大，使用动态权重
  const stageWeights: Record<string, number> = {
    prepare: 5,        // 准备阶段：5%
    download_game: 50,  // 下载游戏：50%（最耗时）
    install_game: 10,   // 安装游戏：10%
    download_mods: 25,  // 下载模组：25%
    install_mods: 5,    // 安装模组：5%
    copy_overrides: 3,  // 复制覆盖文件：3%
    complete: 2         // 完成：2%
  }

  const stageIndex = stages.value.findIndex(s => s.id === data.stage)
  if (stageIndex !== -1) {
    // 计算当前阶段之前所有阶段的总权重
    let baseWeight = 0
    for (let i = 0; i < stageIndex; i++) {
      const stageId = stages.value[i].id
      baseWeight += stageWeights[stageId] || (100 / stages.value.length)
    }

    // 获取当前阶段的权重
    const currentWeight = stageWeights[data.stage] || (100 / stages.value.length)

    // 计算总体进度
    currentProgress.value = baseWeight + (data.progress / 100) * currentWeight

    // 确保进度不超过100%
    if (currentProgress.value > 100) {
      currentProgress.value = 100
    }
  }
}

// 处理安装完成
function handleComplete(data: { versionID: string; name: string }) {
  status.value = 'completed'
  versionId.value = data.versionID
  currentProgress.value = 100
  updateStageStatus('complete', 'completed')
  emit('completed', data.versionID)
}

// 处理安装错误
function handleError(data: { error: string }) {
  status.value = 'error'
  errorMessage.value = data.error

  // 标记当前阶段为错误
  if (currentStage.value) {
    updateStageStatus(currentStage.value, 'error')
  }
  emit('error', data.error)
}

// 处理安装取消
function handleCancelled() {
  status.value = 'error'
  errorMessage.value = t('installed.installCancelled')
}

// 关闭对话框
function handleClose() {
  emit('update:show', false)
}

// 取消安装
async function handleCancel() {
  try {
    await CancelModpackInstall()
  } catch (error: any) {
    console.error('取消安装失败:', error)
  }
}

// 注册事件监听
onMounted(() => {
  console.log('[ModpackInstallDialog] 注册事件监听器')
  EventsOn('modpack:install:start', (data: any) => {
    console.log('[ModpackInstallDialog] 收到 start 事件:', data)
    status.value = 'installing'
    versionId.value = data.versionID
  })
  EventsOn('modpack:install:progress', (data: any) => {
    console.log('[ModpackInstallDialog] 收到 progress 事件:', data)
    handleProgress(data)
  })
  EventsOn('modpack:install:complete', (data: any) => {
    console.log('[ModpackInstallDialog] 收到 complete 事件:', data)
    handleComplete(data)
  })
  EventsOn('modpack:install:error', (data: any) => {
    console.error('[ModpackInstallDialog] 收到 error 事件:', data)
    handleError(data)
  })
  EventsOn('modpack:install:cancelled', (data: any) => {
    console.log('[ModpackInstallDialog] 收到 cancelled 事件:', data)
    handleCancelled()
  })
})

// 清理事件监听
onUnmounted(() => {
  EventsOff('modpack:install:progress')
  EventsOff('modpack:install:complete')
  EventsOff('modpack:install:error')
  EventsOff('modpack:install:cancelled')
})
</script>

<style scoped>
.install-progress {
  min-height: 300px;
}

.modpack-name {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 4px;
}

.modpack-version {
  font-size: 12px;
  color: var(--n-text-color-2);
}
</style>
