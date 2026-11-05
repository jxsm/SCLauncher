<template>
  <n-card :title="t('settings.paths')">
    <n-alert type="info" style="margin-bottom: 16px;">
      {{ t('settings.pathsPortableInfo') }}
    </n-alert>
    <n-descriptions :column="1" bordered>
      <n-descriptions-item :label="t('settings.dataDir')">
        <n-code>{{ config?.dataDir }}</n-code>
      </n-descriptions-item>
      <n-descriptions-item :label="t('settings.versionsDir')">
        <n-code>{{ config?.versionsDir }}</n-code>
      </n-descriptions-item>
      <n-descriptions-item :label="t('settings.downloadsDir')">
        <n-code>{{ config?.downloadsDir }}</n-code>
      </n-descriptions-item>
      <n-descriptions-item :label="t('settings.logsDir')">
        <n-code>.Survivalcraft/Logs</n-code>
      </n-descriptions-item>
    </n-descriptions>

    <!-- 启动器日志：按天存放，保留 7 天 -->
    <n-space align="center" justify="space-between" style="margin-top: 16px;">
      <n-text depth="3" style="font-size: 12px;">{{ t('settings.logsRetentionHint') }}</n-text>
      <n-button size="small" @click="handleOpenLogsFolder">
        <template #icon>
          <n-icon><FolderOpenIcon /></n-icon>
        </template>
        {{ t('settings.openLogsFolder') }}
      </n-button>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import { FolderOpenOutline as FolderOpenIcon } from '@vicons/ionicons5'
import { OpenLogsFolder } from '../../api/logger'
import type { AppConfig } from '../../types/config'

const { t } = useI18n()
const message = useMessage()

defineProps<{
  config: AppConfig | null
}>()

async function handleOpenLogsFolder() {
  try {
    await OpenLogsFolder()
  } catch (err) {
    message.error(`${t('settings.openLogsFolder')}: ${err}`)
  }
}
</script>
