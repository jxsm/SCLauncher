<template>
  <n-modal
    :show="show"
    @update:show="$emit('update:show', $event)"
    preset="card"
    :title="saveGame?.title || ''"
    style="width: 700px;"
  >
    <n-scrollbar style="max-height: 60vh;">
      <n-space vertical size="large">
        <!-- 基本信息 -->
        <n-space vertical size="small">
          <n-text strong>{{ t('common.author') }}:</n-text>
          <n-text>{{ saveGame?.author }}</n-text>
        </n-space>

        <!-- 描述 -->
        <n-space vertical size="small">
          <n-text strong>{{ t('common.description') }}:</n-text>
          <div v-if="saveGame" class="save-description" v-html="saveGame.description"></div>
        </n-space>

        <!-- 统计信息 -->
        <n-space>
          <n-tag size="small" type="info">
            👁 {{ saveGame?.views }}
          </n-tag>
          <n-tag v-if="saveGame && saveGame.likes > 0" size="small" type="warning">
            👍 {{ saveGame?.likes }}
          </n-tag>
          <n-tag size="small" type="success">
            📦 {{ saveGame?.versions.length }} {{ t('mods.versions') }}
          </n-tag>
        </n-space>

        <!-- 版本列表 -->
        <n-divider />
        <n-space vertical size="medium">
          <n-text strong>{{ t('mods.availableVersions') }}</n-text>
          <n-list v-if="saveGame && saveGame.versions.length > 0" bordered>
            <n-list-item v-for="(version, index) in saveGame.versions" :key="index">
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
                  @click="$emit('download', saveGame!, index)"
                  :loading="downloading.has(`${saveGame!.id}-${index}`)"
                >
                  <template #icon>
                    <n-icon><DownloadIcon /></n-icon>
                  </template>
                  {{ t('saveGames.downloadSave') }}
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
  saveGame: ModSearchResult | null
  downloading: Set<string>
}>()

defineEmits<{
  'update:show': [show: boolean]
  download: [saveGame: ModSearchResult, versionIndex: number]
}>()

const { t } = useI18n()
</script>

<style scoped>
.save-description {
  line-height: 1.6;
  color: var(--color-text-primary, #1d1d1f);
}

.save-description :deep(p) {
  margin-bottom: 8px;
}

.save-description :deep(img) {
  max-width: 100%;
  border-radius: 8px;
  margin: 8px 0;
}

.save-description :deep(ul),
.save-description :deep(ol) {
  margin-left: 20px;
  margin-bottom: 8px;
}

.save-description :deep(li) {
  margin-bottom: 4px;
}

.save-description :deep(code) {
  background-color: rgba(128, 128, 140, 0.13);
  padding: 2px 6px;
  border-radius: 5px;
  font-family: 'SF Mono', SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 14px;
}

.save-description :deep(pre) {
  background-color: rgba(128, 128, 140, 0.13);
  padding: 12px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 8px 0;
  border: 1px solid var(--color-border, #e0e0e0);
}

.save-description :deep(pre) code {
  background-color: transparent;
  padding: 0;
  border: none;
}

.save-description :deep(blockquote) {
  border-left: 4px solid #0066cc;
  padding-left: 12px;
  margin: 8px 0;
  color: var(--color-text-tertiary, #7a7a7a);
}
</style>
