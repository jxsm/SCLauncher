<template>
  <div class="home-view">
    <!-- 左侧操作面板 -->
    <div class="left-panel">
      <!-- 版本信息区 - 居中显示 -->
      <div class="version-section">
        <div v-if="primaryVersion" class="version-info">
          <div class="version-name">{{ primaryVersion.name }}</div>
          <div class="version-details">
            <n-tag :type="getVersionTypeColor(primaryVersion.versionType)" size="small">
              {{ getVersionTypeText(primaryVersion.versionType) }}
            </n-tag>
            <n-text depth="3" style="font-size: 13px;">
              {{ primaryVersion.gameVersion }} - {{ primaryVersion.subVersion }}
            </n-text>
          </div>
        </div>
        <n-empty v-else :description="t('home.notSet')" size="small" />
      </div>

      <!-- 核心操作按钮区 - 固定在底部 -->
      <div class="action-section">
        <!-- 启动游戏按钮 -->
        <n-button
          v-if="primaryVersion"
          class="launch-btn"
          :loading="launching"
          :disabled="gameStore.isRunning"
          @click="handleLaunch"
        >
          <div class="launch-btn-content">
            <span class="launch-btn-text">{{ t('home.launchGame') }}</span>
            <span class="launch-btn-subtitle">Star Technology</span>
          </div>
        </n-button>

        <!-- 停止游戏按钮 -->
        <n-button
          v-if="gameStore.isRunning"
          class="launch-btn stop-btn"
          type="error"
          @click="handleStop"
        >
          <div class="launch-btn-content">
            <span class="launch-btn-text">{{ t('installed.stopGame') }}</span>
            <span class="launch-btn-subtitle">{{ t('installed.gameRunning') }}</span>
          </div>
        </n-button>

        <!-- 两个小按钮 -->
        <div class="secondary-actions">
          <n-button class="secondary-btn" @click="$router.push('/installed')">
            {{ t('versions.selectVersion') }}
          </n-button>
          <n-button class="secondary-btn" @click="handleManageResources">
            {{ t('nav.mods') }}
          </n-button>
        </div>
      </div>
    </div>

    <!-- 右侧空白区域 -->
    <div class="right-area">
      <!-- 可以在这里放置背景图或其他内容 -->
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useVersionStore } from '../stores/version'
import { useGameStore } from '../stores/game'
import { useMessage } from 'naive-ui'

const { t } = useI18n()
const router = useRouter()
const versionStore = useVersionStore()
const gameStore = useGameStore()
const message = useMessage()

const launching = ref(false)

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

async function handleLaunch() {
  if (!primaryVersion.value) {
    message.error(t('home.setPrimaryFirst') || '请先设置主要版本')
    return
  }

  launching.value = true
  try {
    await gameStore.launchGame(primaryVersion.value.id)
    message.success(`${t('home.launchSuccess') || '游戏启动成功'}: "${primaryVersion.value.name}"`)
  } catch (error) {
    message.error(t('errors.launchFailed') + '：' + error)
  } finally {
    launching.value = false
  }
}

async function handleStop() {
  try {
    await gameStore.stopGame()
    message.success(t('installed.gameStopped') || '游戏已停止')
  } catch (error) {
    message.error(t('errors.stopFailed') || '停止游戏失败：' + error)
  }
}

function handleManageResources() {
  if (primaryVersion.value) {
    router.push({
      name: 'Resources',
      query: { versionId: primaryVersion.value.id }
    })
  } else {
    router.push('/resources')
  }
}

onMounted(async () => {
  try {
    await versionStore.getVersions()
    await versionStore.getPrimaryVersion()
    await gameStore.updateStatus()
  } catch (error) {
    message.error(t('errors.loadDataFailed') || '加载数据失败：' + error)
  }
})
</script>

<style scoped>
.home-view {
  display: flex;
  height: calc(100vh - 80px);
  gap: 0;
}

/* 左侧操作面板 - Apple 风格 */
.left-panel {
  width: 320px;
  min-width: 320px;
  height: 100%;
  background: var(--color-surface);
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--color-border);
}

/* 版本信息区 - 居中显示 */
.version-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding: 24px;
}

.version-info {
  text-align: center;
}

.version-name {
  font-size: 21px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 8px;
  line-height: 1.19;
  letter-spacing: 0.231px;
}

.version-details {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  flex-wrap: wrap;
}

/* 核心操作按钮区 - Apple 风格 */
.action-section {
  margin-top: auto;
  padding: 24px;
  border-top: 1px solid var(--color-border);
}

/* Apple 风格启动按钮 - 胶囊形 */
.launch-btn {
  width: 100%;
  height: 56px !important;
  background: var(--color-primary) !important;
  color: #ffffff !important;
  font-size: 17px;
  font-weight: 400;
  border-radius: 9999px !important;
  border: none !important;
  transition: all 0.2s ease;
  margin-bottom: 16px;
}

.launch-btn:hover {
  background: var(--color-primary-hover) !important;
}

.launch-btn:active {
  transform: scale(0.95);
}

.launch-btn.stop-btn {
  background: var(--color-error) !important;
  border: none !important;
}

.launch-btn.stop-btn:hover {
  background: var(--color-error-hover) !important;
}

.launch-btn-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 2px;
}

.launch-btn-text {
  font-size: 17px;
  font-weight: 400;
  line-height: 1.2;
}

.launch-btn-subtitle {
  font-size: 12px;
  opacity: 0.7;
}

/* 次要操作按钮 - Apple 幽灵胶囊 */
.secondary-actions {
  display: flex;
  gap: 12px;
}

.secondary-btn {
  flex: 1;
  height: 44px;
  font-size: 14px;
  border: 1px solid var(--color-primary) !important;
  background: transparent !important;
  color: var(--color-primary) !important;
  border-radius: 9999px !important;
  transition: all 0.2s ease;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.secondary-btn:hover {
  background: var(--color-primary-hover-bg) !important;
}

.secondary-btn:active {
  transform: scale(0.95);
}

/* 右侧空白区域 - Apple parchment */
.right-area {
  flex: 1;
  background: var(--color-surface-elevated);
}
</style>
