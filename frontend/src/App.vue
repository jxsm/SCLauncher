<template>
  <n-config-provider :theme="darkTheme">
    <n-message-provider>
      <n-dialog-provider ref="dialogProviderInst">
        <n-notification-provider>
          <div class="app-container">
            <!-- 顶部标签页 -->
            <n-tabs
              v-model:value="activeTab"
              type="line"
              animated
              @update:value="handleTabChange"
            >
              <n-tab-pane name="home" tab="首页">
                <HomeView />
              </n-tab-pane>

              <n-tab-pane name="installed" tab="已安装版本">
                <InstalledVersionsView />
              </n-tab-pane>

              <n-tab-pane name="versions" tab="版本下载">
                <VersionsView />
              </n-tab-pane>

              <n-tab-pane name="mods" tab="模组管理">
                <ModsView />
              </n-tab-pane>

              <n-tab-pane name="settings" tab="设置">
                <SettingsView />
              </n-tab-pane>
            </n-tabs>
          </div>
        </n-notification-provider>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { ref, h, onMounted, onUnmounted, getCurrentInstance } from 'vue'
import { useRouter } from 'vue-router'
import { darkTheme, NAlert, NDialogProvider, useDialog } from 'naive-ui'
import { useGameStore } from './stores/game'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import HomeView from './views/Home.vue'
import InstalledVersionsView from './views/InstalledVersions.vue'
import VersionsView from './views/Versions.vue'
import ModsView from './views/Mods.vue'
import SettingsView from './views/Settings.vue'

const router = useRouter()
const gameStore = useGameStore()
const activeTab = ref('home')

// 不要在根组件的 setup 中直接调用 useDialog
// 而是通过 getCurrentInstance 或者动态获取
const dialogProviderInst = ref<InstanceType<typeof NDialogProvider> | null>(null)

function handleTabChange(value: string) {
  router.push({ name: value.charAt(0).toUpperCase() + value.slice(1) })
}

// 处理游戏崩溃事件
function handleGameCrash(data: any) {
  const { versionName, exitCode, log, crashTime } = data

  // 使用 dialogProvider 的实例
  if (dialogProviderInst.value) {
    const dialog = dialogProviderInst.value
    // 通过 create 方法创建对话框
    // @ts-ignore
    dialog.create({
      title: '游戏崩溃',
      content: () => {
        return h('div', [
          h('p', { style: 'margin-bottom: 12px; font-weight: bold;' }, `版本: ${versionName}`),
          h('p', { style: 'margin-bottom: 12px;' }, `退出码: ${exitCode}`),
          h('p', { style: 'margin-bottom: 12px;' }, `崩溃时间: ${crashTime}`),
          h(NAlert, {
            type: 'error',
            title: '运行日志'
          }, {
            default: () => h('pre', {
              style: 'max-height: 300px; overflow-y: auto; background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; font-size: 12px; white-space: pre-wrap; word-wrap: break-word;'
            }, log)
          })
        ])
      },
      positiveText: '确定'
    })
  }
}

onMounted(async () => {
  // 监听游戏崩溃事件
  EventsOn('game:crashed', handleGameCrash)

  // 监听路由变化
  router.afterEach((to) => {
    activeTab.value = to.name?.toString().toLowerCase() || 'home'
  })

  // 初始化时检查游戏状态
  try {
    await gameStore.updateStatus()
    await gameStore.updateProcessInfo()

    // 如果游戏正在运行，启动状态检查
    if (gameStore.status === 'running') {
      gameStore.startStatusCheck?.()
    }
  } catch (e) {
    console.error('Failed to initialize game status:', e)
  }
})

onUnmounted(() => {
  // 移除崩溃事件监听
  EventsOff('game:crashed')
})
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Helvetica Neue', sans-serif;
}

.app-container {
  width: 100vw;
  height: 100vh;
  padding: 20px;
  overflow: auto;
}

/* 隐藏滚动条但保持滚动功能 */
.app-container::-webkit-scrollbar {
  display: none;
}

/* Firefox */
.app-container {
  scrollbar-width: none;
  -ms-overflow-style: none;
}
</style>
