import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GameStatus, GameProcessInfo } from '../types/game'
import * as gameApi from '../api/game'
import { useRuntimeStore } from './runtime'
import { dialog as discreteDialog } from '../utils/naive'
import i18n from '../locales'
import type { RuntimeStatus } from '../api/runtime'

export const useGameStore = defineStore('game', () => {
  // 状态
  const status = ref<GameStatus>('stopped')
  const processInfo = ref<GameProcessInfo | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  // 状态检查定时器
  let statusCheckInterval: number | null = null

  // 计算属性
  const isRunning = computed(() => status.value === 'running')
  const isStopped = computed(() => status.value === 'stopped')

  // 启动状态检查定时器
  function startStatusCheck() {
    // 清除旧的定时器
    stopStatusCheck()

    // 每2秒检查一次游戏状态
    statusCheckInterval = window.setInterval(async () => {
      const oldStatus = status.value
      await updateStatus()

      // 如果游戏从运行变为停止，清空进程信息并停止检查
      if (oldStatus === 'running' && (status.value === 'stopped' || status.value === 'crashed')) {
        processInfo.value = null
        stopStatusCheck()
      } else if (status.value !== 'running') {
        // 如果当前不是运行状态，也停止检查
        stopStatusCheck()
      }
    }, 2000)
  }

  // 停止状态检查定时器
  function stopStatusCheck() {
    if (statusCheckInterval !== null) {
      clearInterval(statusCheckInterval)
      statusCheckInterval = null
    }
  }

  // 操作
  async function launchGame(versionId: string) {
    loading.value = true
    error.value = null
    try {
      // 启动前：按需检查并补齐 .NET 运行时
      const proceed = await ensureDotNetRuntime(versionId)
      if (!proceed) {
        return // 用户取消或安装失败
      }

      await gameApi.LaunchGame(versionId)
      status.value = 'running'
      // 获取进程信息
      await updateProcessInfo()
      // 启动状态检查
      startStatusCheck()
    } catch (e) {
      error.value = e as string
      console.error('Failed to launch game:', e)
      throw e
    } finally {
      loading.value = false
    }
  }

  // 启动前运行时环境检查：返回是否可以继续启动游戏。
  // - 关闭了自动检查 → 直接放行
  // - 不需要系统运行时 → 直接放行
  // - 已安装 → 直接放行
  // - 缺失 → 弹 Vue 确认框；确认则安装（带进度），取消或安装失败则中止
  async function ensureDotNetRuntime(versionId: string): Promise<boolean> {
    const runtimeStore = useRuntimeStore()
    const t = i18n.global.t.bind(i18n.global)

    let st: RuntimeStatus | undefined
    try {
      st = await runtimeStore.check(versionId)
    } catch (e) {
      // 检查本身失败不阻塞启动（避免运行时检查 bug 挡住游戏）
      console.error('Runtime check failed, skipping:', e)
      return true
    }

    if (!st || !st.enabled || !st.required || st.installed) {
      return true
    }

    // 需要安装 → 弹确认框
    const majorVersion = st.majorVersion
    const ok = await new Promise<boolean>((resolve) => {
      let settled = false
      const done = (v: boolean) => {
        if (!settled) {
          settled = true
          resolve(v)
        }
      }
      discreteDialog.warning({
        title: t('runtime.title'),
        content: t('runtime.installPrompt', { version: majorVersion }),
        positiveText: t('runtime.install'),
        negativeText: t('common.cancel'),
        onPositiveClick: () => done(true),
        onNegativeClick: () => done(false),
        onMaskClick: () => done(false),
        onClose: () => done(false)
      })
    })

    if (!ok) {
      error.value = t('runtime.userCancelled')
      return false
    }

    try {
      await runtimeStore.install(versionId)
      return true
    } catch (e) {
      error.value = t('runtime.failed')
      console.error('Runtime install failed:', e)
      return false
    }
  }

  async function stopGame() {
    loading.value = true
    error.value = null
    try {
      await gameApi.StopGame()
      status.value = 'stopped'
      processInfo.value = null
      // 停止状态检查
      stopStatusCheck()
    } catch (e) {
      error.value = e as string
      console.error('Failed to stop game:', e)
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateStatus() {
    try {
      status.value = await gameApi.GetGameStatus()
    } catch (e) {
      console.error('Failed to get game status:', e)
    }
  }

  async function updateProcessInfo() {
    try {
      processInfo.value = await gameApi.GetGameProcessInfo()
    } catch (e) {
      // 如果获取失败（游戏已关闭），清空进程信息
      if (status.value === 'stopped' || status.value === 'crashed') {
        processInfo.value = null
      }
    }
  }

  return {
    status,
    processInfo,
    loading,
    error,
    isRunning,
    isStopped,
    launchGame,
    stopGame,
    updateStatus,
    updateProcessInfo,
    startStatusCheck,
    stopStatusCheck
  }
})
