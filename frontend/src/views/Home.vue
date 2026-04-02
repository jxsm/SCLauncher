<template>
  <div class="home-view">
    <n-space vertical size="large">
      <!-- 欢迎卡片 -->
      <n-card title="欢迎使用 SCLauncher" hoverable>
        <template #header-extra>
          <n-tag type="info">生存战争启动器</n-tag>
        </template>
        <n-space vertical>
          <p>一个现代化的生存战争游戏启动器</p>
          <n-space>
            <n-button type="primary" @click="$router.push('/installed')">
              已安装版本
            </n-button>
            <n-button @click="$router.push('/versions')">
              版本下载
            </n-button>
            <n-button @click="$router.push('/mods')">
              模组管理
            </n-button>
          </n-space>
        </n-space>
      </n-card>

      <!-- 快速启动主要版本 -->
      <n-card title="快速启动" hoverable>
        <template #header-extra>
          <n-tag v-if="primaryVersion" type="success">主要版本</n-tag>
        </template>
        <n-spin :show="loading">
          <n-space vertical v-if="primaryVersion">
            <n-text strong style="font-size: 20px;">{{ primaryVersion.name }}</n-text>
            <n-space>
              <n-tag :type="getVersionTypeColor(primaryVersion.versionType)" size="small">
                {{ getVersionTypeText(primaryVersion.versionType) }}
              </n-tag>
              <n-text depth="3">版本: {{ primaryVersion.gameVersion }} - {{ primaryVersion.subVersion }}</n-text>
            </n-space>

            <n-space>
              <n-button
                type="success"
                size="large"
                :disabled="gameStore.isRunning"
                :loading="launchingPrimary"
                @click="handleLaunchPrimary"
              >
                <template #icon>
                  <n-icon><PlayIcon /></n-icon>
                </template>
                {{ gameStore.isRunning ? '游戏运行中' : '启动游戏' }}
              </n-button>
              <n-button
                v-if="gameStore.isRunning"
                type="error"
                size="large"
                @click="handleStop"
              >
                <template #icon>
                  <n-icon><StopIcon /></n-icon>
                </template>
                停止游戏
              </n-button>
            </n-space>
          </n-space>
          <n-empty v-else description="未设置主要版本">
            <template #extra>
              <n-button type="primary" @click="$router.push('/installed')">
                去设置
              </n-button>
            </template>
          </n-empty>
        </n-spin>
      </n-card>

      <!-- 游戏状态 -->
      <n-card title="游戏状态" hoverable>
        <n-space vertical>
          <n-space>
            <n-text>状态:</n-text>
            <n-tag :type="getStatusType()">
              {{ getStatusText() }}
            </n-tag>
          </n-space>
          <n-text v-if="gameStore.processInfo" depth="3">
            进程 ID: {{ gameStore.processInfo.pid }}
          </n-text>
        </n-space>
      </n-card>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useVersionStore } from '../stores/version'
import { useGameStore } from '../stores/game'
import { useMessage } from 'naive-ui'
import { Play as PlayIcon, Stop as StopIcon } from '@vicons/ionicons5'

const versionStore = useVersionStore()
const gameStore = useGameStore()
const message = useMessage()

const loading = ref(false)
const launchingPrimary = ref(false)

const primaryVersion = computed(() => versionStore.primaryVersion)

function getVersionTypeText(type: string): string {
  const types = {
    api: '插件版',
    net: '联机版',
    original: '原版'
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

function getStatusType() {
  switch (gameStore.status) {
    case 'running': return 'success'
    case 'stopped': return 'default'
    case 'crashed': return 'error'
    default: return 'default'
  }
}

function getStatusText() {
  switch (gameStore.status) {
    case 'running': return '运行中'
    case 'stopped': return '已停止'
    case 'crashed': return '已崩溃'
    default: return '未知'
  }
}

async function handleLaunchPrimary() {
  if (!primaryVersion.value) {
    message.error('请先设置主要版本')
    return
  }

  launchingPrimary.value = true
  try {
    await gameStore.launchGame(primaryVersion.value.id)
    message.success(`游戏 "${primaryVersion.value.name}" 启动成功！`)
  } catch (error) {
    message.error('游戏启动失败：' + error)
  } finally {
    launchingPrimary.value = false
  }
}

async function handleStop() {
  try {
    await gameStore.stopGame()
    message.success('游戏已停止')
  } catch (error) {
    message.error('停止游戏失败：' + error)
  }
}

onMounted(async () => {
  loading.value = true
  try {
    await versionStore.getVersions()
    await versionStore.getPrimaryVersion()
    await gameStore.updateStatus()
  } catch (error) {
    message.error('加载数据失败：' + error)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.home-view {
  max-width: 800px;
  margin: 0 auto;
}
</style>
