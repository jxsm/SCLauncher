<template>
  <n-modal
    :show="runtimeStore.installing"
    preset="card"
    :title="t('runtime.installing')"
    :mask-closable="false"
    :closable="false"
    style="max-width: 480px;"
    role="dialog"
    aria-modal="true"
  >
    <n-space vertical :size="12">
      <n-progress
        type="line"
        :percentage="percent"
        :status="runtimeStore.installError ? 'error' : 'success'"
        :processing="!hasTotal"
        indicator-placement="inside"
      />
      <n-text depth="2" style="word-break: break-word;">
        {{
          runtimeStore.installError
            ? runtimeStore.installError
            : hasTotal
            ? `${formatSize(runtimeStore.downloaded)} / ${formatSize(runtimeStore.total)}`
            : t('runtime.installingTip')
        }}
      </n-text>
    </n-space>
  </n-modal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRuntimeStore } from '../stores/runtime'

const { t } = useI18n()
const runtimeStore = useRuntimeStore()

const hasTotal = computed(() => runtimeStore.total > 0)

const percent = computed(() => {
  if (!hasTotal.value) return 100 // 走 winget 时无下载进度，显示满条 + processing 动画
  return Math.min(100, Math.round((runtimeStore.downloaded / runtimeStore.total) * 100))
})

function formatSize(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let v = bytes
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(1)} ${units[i]}`
}
</script>
