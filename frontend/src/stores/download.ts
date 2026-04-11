import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

export interface DownloadProgress {
  downloadId: string
  type: 'mod' | 'skin' | 'texture' | 'furniture' | 'savegame'
  versionId?: string
  fileName: string
  downloaded: number
  total: number
  progress: number // 0-100
  status: 'downloading' | 'completed' | 'error'
  error?: string
}

export const useDownloadStore = defineStore('download', () => {
  // 存储所有下载任务的进度
  const downloads = ref<Map<string, DownloadProgress>>(new Map())

  // 获取所有下载任务
  const allDownloads = computed(() => Array.from(downloads.value.values()))

  // 获取正在下载的任务
  const downloadingItems = computed(() =>
    allDownloads.value.filter(d => d.status === 'downloading')
  )

  // 根据ID获取下载任务
  const getDownload = (downloadId: string) => downloads.value.get(downloadId)

  // 延迟清除下载任务
  function delayClear(downloadId: string, delay: number = 2000) {
    setTimeout(() => {
      downloads.value.delete(downloadId)
    }, delay)
  }

  // 初始化事件监听
  function initEventListeners() {
    // 监听下载开始事件
    EventsOn('resource-download:start', (data: {
      downloadId: string
      type: string
      versionId?: string
      fileName: string
    }) => {
      const progress: DownloadProgress = {
        downloadId: data.downloadId,
        type: data.type as DownloadProgress['type'],
        versionId: data.versionId,
        fileName: data.fileName,
        downloaded: 0,
        total: 0,
        progress: 0,
        status: 'downloading'
      }
      downloads.value.set(data.downloadId, progress)
    })

    // 监听下载进度事件
    EventsOn('resource-download:progress', (data: {
      downloadId: string
      downloaded: number
      total: number
    }) => {
      const download = downloads.value.get(data.downloadId)
      if (download) {
        download.downloaded = data.downloaded
        download.total = data.total
        download.progress = data.total > 0 ? (data.downloaded / data.total) * 100 : 0
      }
    })

    // 监听下载完成事件
    EventsOn('resource-download:complete', (data: {
      downloadId: string
      type: string
      versionId?: string
      fileName: string
    }) => {
      const download = downloads.value.get(data.downloadId)
      if (download) {
        download.status = 'completed'
        download.progress = 100
        // 2秒后自动清除
        delayClear(data.downloadId, 2000)
      }
    })

    // 监听下载错误事件
    EventsOn('resource-download:error', (data: {
      downloadId: string
      error: string
    }) => {
      const download = downloads.value.get(data.downloadId)
      if (download) {
        download.status = 'error'
        download.error = data.error
        // 2秒后自动清除
        delayClear(data.downloadId, 2000)
      }
    })
  }

  // 移除事件监听
  function removeEventListeners() {
    EventsOff('resource-download:start')
    EventsOff('resource-download:progress')
    EventsOff('resource-download:complete')
    EventsOff('resource-download:error')
  }

  // 清除已完成的下载任务（可以延迟清除）
  function clearDownload(downloadId: string) {
    downloads.value.delete(downloadId)
  }

  // 清除所有已完成的任务
  function clearCompleted() {
    for (const [id, download] of downloads.value) {
      if (download.status === 'completed' || download.status === 'error') {
        downloads.value.delete(id)
      }
    }
  }

  return {
    downloads,
    allDownloads,
    downloadingItems,
    getDownload,
    initEventListeners,
    removeEventListeners,
    clearDownload,
    clearCompleted
  }
})
