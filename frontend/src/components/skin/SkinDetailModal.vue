<template>
  <n-modal
    :show="show"
    @update:show="$emit('update:show', $event)"
    preset="card"
    :title="skin?.title || ''"
    style="width: 700px;"
  >
    <n-scrollbar style="max-height: 60vh;">
      <n-space vertical size="large">
        <!-- 预览图 -->
        <div v-if="skin?.icon" class="skin-icon-preview">
          <img :src="skin.icon" :alt="skin.title" class="skin-icon-image" />
        </div>

        <!-- 基本信息 -->
        <n-space vertical size="small">
          <n-text strong>{{ t('common.author') }}:</n-text>
          <n-text>{{ skin?.author }}</n-text>
        </n-space>

        <!-- 描述 -->
        <n-space vertical size="small">
          <n-text strong>{{ t('common.description') }}:</n-text>
          <div v-if="skin" class="skin-description" v-html="skin.description"></div>
        </n-space>

        <!-- 统计信息 -->
        <n-space>
          <n-tag size="small" type="info">
            👁 {{ skin?.views }}
          </n-tag>
          <n-tag v-if="skin && skin.likes > 0" size="small" type="warning">
            👍 {{ skin?.likes }}
          </n-tag>
          <n-tag v-if="skin?.versions" size="small" type="success">
            📦 {{ skin.versions.length }} {{ t('common.unitCount') }} {{ t('skins.files') }}
          </n-tag>
        </n-space>

        <!-- 文件列表 -->
        <n-divider />
        <n-space vertical size="medium">
          <n-text strong>{{ t('skins.downloadableFiles') }}</n-text>
          <n-list v-if="skin && skin.versions && skin.versions.length > 0" bordered>
            <n-list-item v-for="(version, index) in skin.versions" :key="index">
              <n-space justify="space-between" align="center" style="width: 100%;">
                <n-space align="center" size="medium">
                  <!-- 文件预览图 -->
                  <img
                    v-if="version.icon"
                    :src="version.icon"
                    :alt="version.fileName"
                    class="file-preview-icon"
                  />
                  <n-icon v-else size="32" :component="ImageIcon" />

                  <n-space vertical size="small">
                    <n-text strong>{{ version.fileName }}</n-text>
                    <n-text depth="3">
                      {{ t('common.size') }}: {{ formatSize(Number(version.fileSize)) }}
                    </n-text>
                  </n-space>
                </n-space>
                <n-button
                  type="primary"
                  size="small"
                  @click="$emit('download', skin!, index)"
                  :loading="downloading.has(`${skin!.id}-${index}`)"
                >
                  <template #icon>
                    <n-icon><DownloadIcon /></n-icon>
                  </template>
                  {{ t('skins.downloadSkin') }}
                </n-button>
              </n-space>
            </n-list-item>
          </n-list>
          <n-empty v-else :description="t('skins.noDownloadableFiles')" />
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
import { Download as DownloadIcon, Image as ImageIcon } from '@vicons/ionicons5'
import { formatSize } from '../../utils/format'
import type { ModSearchResult } from '../../types/mod-source'

defineProps<{
  show: boolean
  skin: ModSearchResult | null
  downloading: Set<string>
}>()

defineEmits<{
  'update:show': [show: boolean]
  download: [skin: ModSearchResult, versionIndex: number]
}>()

const { t } = useI18n()
</script>

<style scoped>
.skin-icon-preview {
  text-align: center;
  padding: 16px;
  background-color: var(--n-code-color, #f5f5f7);
  border-radius: 8px;
}

.skin-icon-image {
  max-width: 100%;
  max-height: 300px;
  object-fit: contain;
  image-rendering: pixelated;
  border-radius: 8px;
}

.file-preview-icon {
  width: 32px;
  height: 32px;
  object-fit: cover;
  image-rendering: pixelated;
  border-radius: 5px;
}

.skin-description {
  line-height: 1.6;
  color: #1d1d1f;
}

.skin-description :deep(p) {
  margin-bottom: 8px;
}

.skin-description :deep(img) {
  max-width: 100%;
  border-radius: 8px;
  margin: 8px 0;
}

.skin-description :deep(ul),
.skin-description :deep(ol) {
  margin-left: 20px;
  margin-bottom: 8px;
}

.skin-description :deep(li) {
  margin-bottom: 4px;
}

.skin-description :deep(code) {
  background-color: var(--n-code-color, #f5f5f7);
  padding: 2px 6px;
  border-radius: 5px;
  font-family: 'SF Mono', SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 14px;
}

.skin-description :deep(pre) {
  background-color: var(--n-code-color, #f5f5f7);
  padding: 12px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 8px 0;
  border: 1px solid var(--color-border, #e0e0e0);
}

.skin-description :deep(pre) code {
  background-color: transparent;
  padding: 0;
  border: none;
}

.skin-description :deep(blockquote) {
  border-left: 4px solid #0066cc;
  padding-left: 12px;
  margin: 8px 0;
  color: #7a7a7a;
}
</style>
