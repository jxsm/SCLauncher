import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GameStatus, GameProcessInfo } from '../types/game'
import * as gameApi from '../api/game'

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
