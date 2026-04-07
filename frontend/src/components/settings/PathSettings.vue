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
    </n-descriptions>
  </n-card>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { AppConfig } from '../../types/config'

const { t } = useI18n()

defineProps<{
  config: AppConfig | null
}>()
</script>
