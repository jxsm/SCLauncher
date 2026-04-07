<template>
  <n-modal
    v-model:show="showModal"
    preset="card"
    :title="texture?.title || ''"
    style="width: 600px;"
  >
    <n-space vertical size="large" v-if="texture">
      <!-- 基本信息 -->
      <n-space vertical>
        <n-text strong>{{ t('textures.author') }}: {{ texture.author }}</n-text>
        <n-text v-if="texture.description">{{ t('textures.description') }}: {{ texture.description }}</n-text>
        <n-text type="info" v-if="texture.views">{{ t('textures.views') }}: {{ texture.views }}</n-text>
        <n-text type="info" v-if="texture.likes">{{ t('textures.likes') }}: {{ texture.likes }}</n-text>
      </n-space>

      <n-divider />

      <!-- 版本列表 -->
      <n-space vertical>
        <n-text strong>{{ t('textures.availableVersions') }}:</n-text>
        <n-list>
          <n-list-item v-for="(version, index) in texture.versions" :key="index">
            <n-space justify="space-between" align="center">
              <n-space>
                <n-text strong>{{ version.version }}</n-text>
                <n-text depth="3">({{ version.fileName }})</n-text>
                <n-text depth="3">{{ version.fileSize }}</n-text>
              </n-space>
              <n-button
                type="primary"
                :loading="downloading.has(`${texture.id}-${index}`)"
                @click="handleDownload(texture, index)"
              >
                {{ t('common.download') }}
              </n-button>
            </n-space>
          </n-list-item>
        </n-list>
      </n-space>
    </n-space>
  </n-modal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModSearchResult } from '../../types/mod-source'

const props = defineProps<{
  show: boolean
  texture: ModSearchResult | null
  downloading: Set<string>
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  download: [texture: ModSearchResult, versionIndex: number]
}>()

const { t } = useI18n()

const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value)
})

function handleDownload(texture: ModSearchResult, versionIndex: number) {
  emit('download', texture, versionIndex)
}
</script>
