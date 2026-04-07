<template>
  <n-list-item @click="$emit('click', skin)">
    <n-thing>
      <template #header>
        <n-space align="center">
          <!-- 皮肤预览图 -->
          <div class="skin-preview-container">
            <img
              v-if="skin.icon || skin.versions[0]?.icon"
              :src="skin.icon || skin.versions[0]?.icon"
              :alt="skin.title"
              class="skin-preview-image"
            />
            <n-avatar v-else-if="skin.authorAvatar" :src="skin.authorAvatar" :size="48" round />
            <n-avatar v-else :size="48" round>
              {{ skin.title.charAt(0) }}
            </n-avatar>
          </div>
          <n-text strong>{{ skin.title }}</n-text>
          <n-tag v-if="skin.versions && skin.versions.length > 0" size="small" type="success">
            {{ skin.versions.length }} 个文件
          </n-tag>
        </n-space>
      </template>

      <template #description>
        <n-space vertical size="small">
          <n-text depth="3">
            {{ skin.author }}
          </n-text>
          <n-text depth="3" :line-clamp="1">
            {{ stripHtmlTags(skin.description) }}
          </n-text>
          <n-space>
            <n-tag size="small" type="info">
              👁 {{ skin.views }}
            </n-tag>
            <n-tag v-if="skin.likes > 0" size="small" type="warning">
              👍 {{ skin.likes }}
            </n-tag>
          </n-space>
        </n-space>
      </template>
    </n-thing>
  </n-list-item>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ModSearchResult } from '../../types/mod-source'

defineProps<{
  skin: ModSearchResult
}>()

defineEmits<{
  click: [skin: ModSearchResult]
}>()

const { t } = useI18n()

function stripHtmlTags(html: string): string {
  const tmp = document.createElement('div')
  tmp.innerHTML = html
  const text = tmp.textContent || tmp.innerText || ''
  return text.length > 100 ? text.substring(0, 100) + '...' : text
}
</script>

<style scoped>
.skin-preview-container {
  width: 48px;
  height: 48px;
  border-radius: 4px;
  overflow: hidden;
  background-color: var(--n-color);
  display: flex;
  align-items: center;
  justify-content: center;
}

.skin-preview-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  image-rendering: pixelated;
}
</style>
