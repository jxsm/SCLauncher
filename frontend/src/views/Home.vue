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
        <n-space class="secondary-actions">
          <n-button class="secondary-btn" @click="$router.push('/installed')">
            {{ t('versions.selectVersion') }}
          </n-button>
          <n-button class="secondary-btn" @click="$router.push('/mods')">
            {{ t('nav.mods') }}
          </n-button>
        </n-space>
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
import { useI18n } from 'vue-i18n'
import { useVersionStore } from '../stores/version'
import { useGameStore } from '../stores/game'
import { useMessage } from 'naive-ui'

const { t } = useI18n()
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
  height: calc(100vh - 100px);
}

/* 左侧操作面板 */
.left-panel {
  width: 300px;
  min-width: 300px;
  height: 100%;
  background: #161f2d;
  border-right: 1px solid var(--n-divider-color);
  display: flex;
  flex-direction: column;
  border-radius: 10px;
}

/* 版本信息区 - 居中显示 */
.version-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding: 20px;
}

.version-info {
  text-align: center;
}

.version-name {
  font-size: 18px;
  font-weight: 600;
  color: var(--n-text-color);
  margin-bottom: 8px;
  line-height: 1.4;
}

.version-details {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  flex-wrap: wrap;
}

/* 核心操作按钮区 - 固定在底部 */
.action-section {
  margin-top: auto;
  padding: 24px;
  border-top: 1px solid var(--n-divider-color);
}

.launch-btn {
  width: 100%;
  height: 60px !important;
  border: 2px solid var(--n-primary-color);
  background: var(--n-color);
  color: var(--n-primary-color);
  font-size: 16px;
  font-weight: bold;
  border-radius: 6px;
  transition: all 0.3s ease;
  margin-bottom: 16px;
}

.launch-btn:hover {
  background: var(--n-primary-color);
  color: #ffffff;
}

.launch-btn.stop-btn {
  border-color: var(--n-error-color);
  color: var(--n-error-color);
}

.launch-btn.stop-btn:hover {
  background: var(--n-error-color);
  color: #ffffff;
}

.launch-btn-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 4px;
}

.launch-btn-text {
  font-size: 16px;
  font-weight: bold;
  line-height: 1.2;
}

.launch-btn-subtitle {
  font-size: 11px;
  opacity: 0.7;
}

/* 次要操作按钮 */
.secondary-actions {
  display: flex;
  gap: 12px;
}

.secondary-btn {
  border: 1px solid red;
  width: 120px;
  height: 48px;
  font-size: 14px;
  border: 1px solid var(--n-border-color);
  background: var(--n-color);
  color: var(--n-text-color-2);
  border-radius: 6px;
  transition: all 0.3s ease;
}

.secondary-btn:hover {
  border-color: var(--n-primary-color);
  color: var(--n-primary-color);
}

/* 右侧空白区域 */
.right-area {
  flex: 1;
  background: var(--n-color-modal);
}
</style>
