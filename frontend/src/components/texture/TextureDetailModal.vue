<template>
  <n-modal
    :show="show"
    @update:show="$emit('update:show', $event)"
    preset="card"
    :title="texture?.title || ''"
    style="width: 700px;"
  >
    <n-scrollbar style="max-height: 60vh;">
      <n-space vertical size="large">
        <!-- 基本信息 -->
        <n-space vertical size="small">
          <n-text strong>{{ t('common.author') }}:</n-text>
          <n-text>{{ texture?.author }}</n-text>
        </n-space>

        <!-- 描述 -->
        <n-space vertical size="small">
          <n-text strong>{{ t('common.description') }}:</n-text>
          <div v-if="texture" class="texture-description" v-html="texture.description"></div>
        </n-space>

        <!-- 统计信息 -->
        <n-space>
          <n-tag size="small" type="info">
            👁 {{ texture?.views }}
          </n-tag>
          <n-tag v-if="texture && texture.likes > 0" size="small" type="warning">
            👍 {{ texture?.likes }}
          </n-tag>
          <n-tag size="small" type="success">
            📦 {{ texture?.versions.length }} {{ t('mods.versions') }}
          </n-tag>
        </n-space>

        <!-- 版本列表 -->
        <n-divider />
        <n-space vertical size="medium">
          <n-text strong>{{ t('mods.availableVersions') }}</n-text>
          <n-list v-if="texture && texture.versions.length > 0" bordered>
            <n-list-item v-for="(version, index) in texture.versions" :key="index">
              <n-space justify="space-between" align="center" style="width: 100%;">
                <n-space vertical size="small">
                  <n-text strong>v{{ version.version }}</n-text>
                  <n-text depth="3">
                    {{ t('common.size') }}: {{ formatSize(Number(version.fileSize)) }}
                  </n-text>
                </n-space>
                <n-button
                  type="primary"
                  size="small"
                  @click="$emit('download', texture!, index)"
                  :loading="downloading.has(`${texture!.id}-${index}`)"
                >
                  <template #icon>
                    <n-icon><DownloadIcon /></n-icon>
                  </template>
                  {{ t('textures.downloadTexture') }}
                </n-button>
              </n-space>
            </n-list-item>
          </n-list>
          <n-empty v-else :description="t('mods.noVersions')" />
        </n-space>
      </n-space>
    </n-scrollbar>

    <template #footer>
      <n-space justify="end">
        <n-button @click="$emit('update:show', false)">{{ t('common.close') }}</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Download as DownloadIcon } from '@vicons/ionicons5'
import { formatSize } from '../../utils/format'
import type { ModSearchResult } from '../../types/mod-source'

defineProps<{
  show: boolean
  texture: ModSearchResult | null
  downloading: Set<string>
}>()

defineEmits<{
  'update:show': [show: boolean]
  download: [texture: ModSearchResult, versionIndex: number]
}>()

const { t } = useI18n()
</script>

<style scoped>
.texture-description {
  line-height: 1.6;
  color: #fff;
}

.texture-description :deep(p) {
  margin-bottom: 8px;
}

.texture-description :deep(img) {
  max-width: 100%;
  border-radius: 4px;
  margin: 8px 0;
}

.texture-description :deep(ul),
.texture-description :deep(ol) {
  margin-left: 20px;
  margin-bottom: 8px;
}

.texture-description :deep(li) {
  margin-bottom: 4px;
}

.texture-description :deep(code) {
  background-color: rgba(255, 255, 255, 0.1);
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
}

.texture-description :deep(pre) {
  background-color: rgba(255, 255, 255, 0.1);
  padding: 12px;
  border-radius: 4px;
  overflow-x: auto;
  margin: 8px 0;
}

.texture-description :deep(blockquote) {
  border-left: 4px solid rgba(255, 255, 255, 0.3);
  padding-left: 12px;
  margin: 8px 0;
  color: rgba(255, 255, 0.8);
}
</style>
