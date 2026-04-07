<template>
  <n-thing>
    <template #header>
      <n-space align="center">
        <n-text strong>{{ version.subVersion }}</n-text>
        <n-tag :type="getTypeColor(version.versionType)" size="small">
          {{ getTypeText(t, version.versionType) }}
        </n-tag>
      </n-space>
    </template>

    <template #description>
      <n-space vertical size="small">
        <n-text depth="3">
          {{ t('common.size') }}: {{ formatSize(version.size) }}
        </n-text>
        <n-text v-if="version.illustrate" depth="3">
          {{ t('common.description') }}: {{ version.illustrate }}
        </n-text>
      </n-space>
    </template>

    <template #action>
      <n-space>
        <!-- Download button or progress -->
        <n-button
          v-if="!isDownloading"
          type="primary"
          size="medium"
          @click="handleDownload"
        >
          <template #icon>
            <n-icon><DownloadIcon /></n-icon>
          </template>
          {{ t('versions.download') }}
        </n-button>
        <n-progress
          v-else
          type="line"
          :percentage="downloadProgress"
          :indicator-placement="'inside'"
          processing
          style="width: 200px"
        />
      </n-space>
    </template>
  </n-thing>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { CloudDownload as DownloadIcon } from '@vicons/ionicons5'
import { formatSize } from '../../utils/format'
import { getTypeText, getTypeColor } from './versionUtils'
import type { Version } from '../../types/version'

const props = defineProps<{
  version: Version
  isDownloading: boolean
  downloadProgress: number
}>()

const emit = defineEmits<{
  'download': [version: Version]
}>()

const { t } = useI18n()

function handleDownload() {
  emit('download', props.version)
}
</script>
