import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as runtimeApi from '../api/runtime'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import type { RuntimeStatus } from '../api/runtime'

// 运行时安装相关的全局状态：供 RuntimeInstallDialog 显示进度。
export const useRuntimeStore = defineStore('runtime', () => {
  const installing = ref(false)
  const downloaded = ref(0)
  const total = ref(0)
  const versionId = ref<string>('')
  const installError = ref<string | null>(null)

  let bound = false
  function bindEvents() {
    if (bound) return
    bound = true
    EventsOn('runtime:install:start', (d: any) => {
      installing.value = true
      installError.value = null
      versionId.value = d?.versionId || ''
      downloaded.value = 0
      total.value = 0
    })
    EventsOn('runtime:install:progress', (d: any) => {
      downloaded.value = d?.downloaded || 0
      total.value = d?.total || 0
    })
    EventsOn('runtime:install:complete', () => {
      installing.value = false
    })
    EventsOn('runtime:install:error', (d: any) => {
      installError.value = d?.error || 'error'
      installing.value = false
    })
  }
  bindEvents()

  // 检查指定版本所需的运行时状态
  async function check(vid: string): Promise<RuntimeStatus> {
    return await runtimeApi.CheckDotNetRuntime(vid)
  }

  // 安装指定版本所需的运行时（进度通过事件回流到本 store）
  async function install(vid: string): Promise<void> {
    installing.value = true
    installError.value = null
    versionId.value = vid
    downloaded.value = 0
    total.value = 0
    try {
      await runtimeApi.InstallDotNetRuntime(vid)
    } finally {
      installing.value = false
    }
  }

  return { installing, downloaded, total, versionId, installError, check, install }
})
