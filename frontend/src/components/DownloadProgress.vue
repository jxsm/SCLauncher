<template>
  <Teleport to="body">
    <Transition name="dp-slide">
      <div
        v-if="hasTasks"
        class="download-panel"
        :class="{ 'is-collapsed': !expanded, 'is-downloading': isDownloading }"
        role="region"
        :aria-label="panelLabel"
      >
        <n-card size="small" :bordered="false" class="panel-card">
          <!-- 可点击的折叠头部：默认收起，只显示“{n} 个下载任务” -->
          <div
            class="panel-header"
            role="button"
            tabindex="0"
            :aria-expanded="expanded"
            aria-controls="download-panel-body"
            :aria-label="panelLabel"
            @click="toggle"
            @keydown.enter.prevent="toggle"
            @keydown.space.prevent="toggle"
          >
            <div class="panel-leading">
              <n-icon size="18" :component="DownloadIcon" />
              <!-- 展开时显示“{n} 个下载任务”；收起时只显示数量，二者淡入淡出过渡 -->
              <Transition name="dp-label" mode="out-in">
                <n-text v-if="expanded" key="full" strong class="panel-title">{{ panelLabel }}</n-text>
                <span v-else key="count" class="panel-count">{{ taskCount }}</span>
              </Transition>
            </div>
            <n-icon
              size="16"
              :component="ChevronDown"
              class="panel-chevron"
              :class="{ 'is-expanded': expanded }"
            />
          </div>

          <!-- 展开后：任务名 + 进度 -->
          <n-collapse-transition :show="expanded">
            <div id="download-panel-body" class="panel-body">
              <div
                v-for="download in downloadStore.allDownloads"
                :key="download.downloadId"
                class="task-row"
              >
                <div class="task-row-top">
                  <n-text class="task-name" :title="download.fileName" depth="2">
                    {{ download.fileName || t('common.downloading') }}
                  </n-text>
                  <n-text depth="3" class="task-pct">
                    {{ getProgressPercentage(download).toFixed(0) }}%
                  </n-text>
                </div>

                <n-progress
                  type="line"
                  :percentage="getProgressPercentage(download)"
                  :status="getProgressStatus(download)"
                  :show-indicator="false"
                  :height="6"
                  :border-radius="3"
                />

                <div class="task-row-bottom">
                  <n-text depth="3" class="task-size">
                    {{ formatSize(download.downloaded) }} / {{ formatSize(download.total) }}
                  </n-text>
                  <n-text v-if="download.status === 'downloading'" type="info">
                    {{ t('common.downloading') }}…
                  </n-text>
                  <n-text v-else-if="download.status === 'completed'" type="success">
                    {{ t('common.downloadComplete') }}
                  </n-text>
                  <n-text v-else-if="download.status === 'error'" type="error" class="task-error">
                    {{ download.error || t('common.downloadFailed') }}
                  </n-text>
                </div>
              </div>
            </div>
          </n-collapse-transition>
        </n-card>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CloudDownload as DownloadIcon,
  ChevronDown,
} from '@vicons/ionicons5'
import { useDownloadStore, type DownloadProgress } from '../stores/download'

const { t } = useI18n()
const downloadStore = useDownloadStore()

// 默认收起
const expanded = ref(false)

// 是否有下载任务（决定整个面板是否显示）
const hasTasks = computed(() => downloadStore.allDownloads.length > 0)

// 当前下载任务数量
const taskCount = computed(() => downloadStore.allDownloads.length)

// 是否有正在下载的任务（用于胶囊徽标的呼吸动画）
const isDownloading = computed(() => downloadStore.downloadingItems.length > 0)

// 展开时折叠头部展示的文案：“{n} 个下载任务”
const panelLabel = computed(() =>
  t('common.downloadTasks', { count: taskCount.value })
)

// 从无任务变为有任务时（面板重新出现），恢复默认收起状态
watch(hasTasks, (now, prev) => {
  if (now && !prev) {
    expanded.value = false
  }
})

function toggle() {
  expanded.value = !expanded.value
}

// 获取进度百分比
function getProgressPercentage(download: DownloadProgress): number {
  return Math.min(100, Math.max(0, download.progress || 0))
}

// 获取进度条状态
function getProgressStatus(download: DownloadProgress): 'success' | 'error' | 'info' {
  if (download.status === 'completed') return 'success'
  if (download.status === 'error') return 'error'
  return 'info'
}

// 格式化文件大小
function formatSize(bytes: number): string {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}
</script>

<style scoped>
.download-panel {
  position: fixed;
  right: 24px;
  /* 避开右下角的“回到顶部”按钮 */
  bottom: 96px;
  /* 高于 naive-ui 弹窗/抽屉（~2000），保证下载进行时即便打开详情弹窗也能看到进度 */
  z-index: 3000;
  width: 320px;
  max-width: calc(100vw - 48px);
  pointer-events: auto;
  transition: width 0.25s ease;
}

/* 收起时变成固定宽度的紧凑胶囊（大小接近“回到顶部”按钮），固定宽度才能与展开态平滑过渡 */
.download-panel.is-collapsed {
  width: 88px;
}

.panel-card {
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
  border-radius: 14px;
  border: 1px solid var(--color-border, #e0e0e0);
  background: var(--color-nav-bg, rgba(255, 255, 255, 0.95));
  backdrop-filter: saturate(180%) blur(20px);
  -webkit-backdrop-filter: saturate(180%) blur(20px);
  overflow: hidden;
  transition: box-shadow 0.2s ease;
}

.download-panel:hover .panel-card {
  box-shadow: 0 6px 26px rgba(0, 0, 0, 0.18);
}

/* 收起态：胶囊按钮风格，紧凑到接近“回到顶部”按钮的大小 */
.download-panel.is-collapsed :deep(.n-card__content) {
  padding: 0;
}

.download-panel.is-collapsed .panel-card {
  border-radius: 9999px;
}

.download-panel.is-collapsed .panel-header {
  padding: 8px 16px;
  margin: 0;
  justify-content: center;
}

.panel-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9999px;
  background: var(--color-primary, #5865F2);
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
  box-shadow: 0 2px 6px rgba(88, 101, 242, 0.35);
}

/* 有正在下载的任务时，徽标轻微呼吸提示 */
.download-panel.is-downloading .panel-count {
  animation: dp-pulse 1.8s ease-in-out infinite;
}

@keyframes dp-pulse {
  0%, 100% {
    transform: scale(1);
    box-shadow: 0 2px 6px rgba(88, 101, 242, 0.35);
  }
  50% {
    transform: scale(1.12);
    box-shadow: 0 2px 10px rgba(88, 101, 242, 0.55);
  }
}

/* 折叠头部（可点击） */
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 8px;
  margin: -4px;
  cursor: pointer;
  user-select: none;
  border-radius: 10px;
  outline: none;
  transition: background-color 0.17s ease;
}

.panel-header:hover {
  background: var(--hover-color, rgba(0, 0, 0, 0.04));
}

.panel-header:focus-visible {
  box-shadow: 0 0 0 2px var(--color-primary, #5865F2);
}

.panel-title {
  font-size: 14px;
}

.panel-leading {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

/* 数量 ⇄ 完整文案 之间的淡入淡出 */
.dp-label-enter-active,
.dp-label-leave-active {
  transition: opacity 0.14s ease;
}

.dp-label-enter-from,
.dp-label-leave-to {
  opacity: 0;
}

/* 箭头随展开/收起平滑出现并旋转（宽度+透明度+旋转一起过渡） */
.panel-chevron {
  flex-shrink: 0;
  color: var(--color-text-tertiary, #999);
  width: 0;
  opacity: 0;
  overflow: hidden;
  transform: rotate(0deg);
  transition: width 0.25s ease, opacity 0.2s ease, transform 0.25s ease;
}

.panel-chevron.is-expanded {
  width: 16px;
  opacity: 1;
  transform: rotate(180deg);
}

/* 展开后的任务列表 */
.panel-body {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  /* 高窗口下限制为 280px；矮窗口下随视口收缩，避免顶部（折叠按钮）被挤出可视区 */
  max-height: min(280px, calc(100vh - 240px));
  overflow-y: auto;
}

.task-row {
  padding: 8px 0;
}

.task-row + .task-row {
  border-top: 1px solid var(--color-border, rgba(0, 0, 0, 0.05));
}

.task-row-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
}

.task-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  min-width: 0;
}

.task-pct {
  flex-shrink: 0;
  font-size: 12px;
}

.task-row-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 4px;
  font-size: 12px;
}

.task-size {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.task-error {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 60%;
  text-align: right;
}

/* 面板进入/离开动画 */
.dp-slide-enter-active,
.dp-slide-leave-active {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.dp-slide-enter-from,
.dp-slide-leave-to {
  transform: translateX(120%);
  opacity: 0;
}

/* 任务列表滚动条 */
.panel-body::-webkit-scrollbar {
  width: 6px;
}

.panel-body::-webkit-scrollbar-thumb {
  background: var(--color-scroll-thumb, rgba(0, 0, 0, 0.2));
  border-radius: 3px;
}
</style>
