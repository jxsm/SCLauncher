<template>
  <n-collapse-item>
    <template #header>
      <n-space align="center" justify="space-between" style="width: 100%">
        <n-text strong style="font-size: 16px">
          {{ gameVersion }}
        </n-text>
        <n-tag size="small" type="info">
          {{ versions.length }} {{ t('versions.versionsCount') }}
        </n-tag>
      </n-space>
    </template>

    <n-list hoverable clickable>
      <n-list-item v-for="version in versions" :key="version.id">
        <VersionListItem
          :version="version"
          :is-downloading="isDownloading(version.id)"
          :download-progress="getDownloadProgress(version.id)"
          @download="handleDownload"
        />
      </n-list-item>
    </n-list>
  </n-collapse-item>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import VersionListItem from './VersionListItem.vue'
import type { Version } from '../../types/version'

const props = defineProps<{
  gameVersion: string
  versions: Version[]
  isDownloading: (id: string) => boolean
  getDownloadProgress: (id: string) => number
}>()

const emit = defineEmits<{
  'download': [version: Version]
}>()

const { t } = useI18n()

function handleDownload(version: Version) {
  emit('download', version)
}
</script>
