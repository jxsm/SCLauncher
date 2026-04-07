<template>
  <n-list-item @click="$emit('click', furniture)">
    <n-thing>
      <template #header>
        <n-space align="center">
          <n-avatar
            v-if="furniture.icon"
            :src="furniture.icon"
            :size="48"
            round
          />
          <n-avatar
            v-else-if="furniture.authorAvatar"
            :src="furniture.authorAvatar"
            :size="48"
            round
          />
          <n-avatar v-else :size="48" round>
            {{ furniture.title.charAt(0) }}
          </n-avatar>
          <n-text strong>{{ furniture.title }}</n-text>
          <n-tag v-if="furniture.versions.length > 0" size="small" type="info">
            v{{ furniture.versions[0].version }}
          </n-tag>
        </n-space>
      </template>

      <template #description>
        <n-space vertical size="small">
          <n-text depth="3">
            {{ furniture.author }}
          </n-text>
          <n-text depth="3" :line-clamp="1">
            {{ stripHtmlTags(furniture.description) }}
          </n-text>
          <n-space>
            <n-tag size="small" type="info">
              👁 {{ furniture.views }}
            </n-tag>
            <n-tag v-if="furniture.likes > 0" size="small" type="warning">
              👍 {{ furniture.likes }}
            </n-tag>
            <n-tag size="small" type="success">
              📦 {{ furniture.versions.length }} {{ t('mods.versions') }}
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
  furniture: ModSearchResult
}>()

defineEmits<{
  click: [furniture: ModSearchResult]
}>()

const { t } = useI18n()

function stripHtmlTags(html: string): string {
  const tmp = document.createElement('div')
  tmp.innerHTML = html
  const text = tmp.textContent || tmp.innerText || ''
  return text.length > 100 ? text.substring(0, 100) + '...' : text
}
</script>
