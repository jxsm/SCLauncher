<template>
  <n-card>
    <n-space justify="space-between">
      <n-space>
        <n-text strong style="font-size: 18px;">{{ t('installed.title') }}</n-text>
      </n-space>
      <n-space>
        <n-button type="primary" @click="$emit('importGame')">
          <template #icon>
            <n-icon><ImportIcon /></n-icon>
          </template>
          {{ t('installed.importGame') }}
        </n-button>
        <n-button type="success" @click="$emit('localInstall')">
          <template #icon>
            <n-icon><ArchiveIcon /></n-icon>
          </template>
          {{ t('installed.localInstall') }}
        </n-button>
        <n-text depth="3">
          {{ t('installed.totalInstalled') }} {{ totalCount }}
        </n-text>
      </n-space>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Download as ImportIcon, Archive as ArchiveIcon } from '@vicons/ionicons5'

const { t } = useI18n()

defineProps<{
  totalCount: number
}>()

defineEmits<{
  importGame: []
  localInstall: []
}>()
</script>
