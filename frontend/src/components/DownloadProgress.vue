<template>
  <div class="download-progress-wrapper">
    <div
      v-for="(download, index) in downloadStore.allDownloads"
      :key="download.downloadId"
      class="download-progress-container"
      :style="{ bottom: `${120 + index * 140}px` }"
    >
      <n-card size="small" :bordered="false" class="progress-card">
        <template #header>
          <n-space align="center">
            <n-icon size="20" :component="DownloadIcon" />
            <n-text strong>{{ getDownloadTitle(download) }}</n-text>
          </n-space>
        </template>

        <n-space vertical size="small">
          <!-- 进度条 -->
          <n-progress
            type="line"
            :percentage="getProgressPercentage(download)"
            :status="getProgressStatus(download)"
            :show-indicator="false"
          />

          <!-- 下载信息 -->
          <n-space justify="space-between">
            <n-text depth="3">
              {{ formatSize(download.downloaded) }} / {{ formatSize(download.total) }}
            </n-text>
            <n-text depth="3" v-if="getProgressPercentage(download) > 0">
              {{ getProgressPercentage(download).toFixed(1) }}%
            </n-text>
          </n-space>

          <!-- 状态信息 -->
          <n-text v-if="download.status === 'downloading'" depth="3" type="info">
            {{ t('common.downloading') }}...
          </n-text>
          <n-text v-else-if="download.status === 'completed'" depth="3" type="success">
            {{ t('common.downloadComplete') }}
          </n-text>
          <n-text v-else-if="download.status === 'error'" depth="3" type="error">
            {{ download.error || t('common.downloadFailed') }}
          </n-text>
        </n-space>
      </n-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { CloudDownload as DownloadIcon } from '@vicons/ionicons5'
import { useDownloadStore } from '../stores/download'

const { t } = useI18n()
const downloadStore = useDownloadStore()

// 获取下载标题
function getDownloadTitle(download: any): string {
  if (download.status === 'completed') {
    return t('common.downloadComplete')
  }
  if (download.status === 'error') {
    return t('common.downloadFailed')
  }
  return download.fileName || t('common.downloading')
}

// 获取进度百分比
function getProgressPercentage(download: any): number {
  return Math.min(100, Math.max(0, download.progress || 0))
}

// 获取进度条状态
function getProgressStatus(download: any): 'success' | 'error' | 'info' {
  if (download.status === 'completed') return 'success'
  if (download.status === 'error') return 'error'
  return 'info'
}

// 格式化文件大小
function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}
</script>

<style scoped>
.download-progress-wrapper {
  position: fixed;
  right: 40px;
  bottom: 120px;
  z-index: 999;
  pointer-events: none;
}

.download-progress-container {
  position: fixed;
  right: 40px;
  min-width: 300px;
  max-width: 400px;
  pointer-events: auto;
  transition: bottom 0.3s ease;
}

.progress-card {
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
  border-radius: 18px;
  border: 1px solid var(--color-border, #e0e0e0);
  background: var(--color-nav-bg, rgba(255, 255, 255, 0.95));
  backdrop-filter: saturate(180%) blur(20px);
  -webkit-backdrop-filter: saturate(180%) blur(20px);
}

/* 动画效果 */
.download-progress-container {
  animation: slideIn 0.3s ease-out;
}

@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}
</style>
