<template>
  <n-list-item @click="$emit('click', texture)" style="cursor: pointer;">
    <n-thing>
      <template #header>
        <n-space align="center">
          <n-avatar
            v-if="texture.icon"
            :src="texture.icon"
            :size="48"
            round
          />
          <n-avatar
            v-else-if="texture.authorAvatar"
            :src="texture.authorAvatar"
            :size="48"
            round
          />
          <n-avatar v-else :size="48" round>
            {{ texture.title.charAt(0) }}
          </n-avatar>
          <n-text strong>{{ texture.title }}</n-text>
          <n-tag v-if="texture.versions.length > 0" size="small" type="info">
            v{{ texture.versions[0].version }}
          </n-tag>
        </n-space>
      </template>

      <template #description>
        <n-space vertical size="small">
          <n-text depth="3">
            {{ texture.author }}
          </n-text>
          <n-text depth="3" :line-clamp="1">
            {{ stripHtmlTags(texture.description) }}
          </n-text>
          <n-space>
            <n-tag size="small" type="info">
              <n-icon><EyeIcon /></n-icon> {{ texture.views }}
            </n-tag>
            <n-tag v-if="texture.likes > 0" size="small" type="warning">
              <n-icon><ThumbUpIcon /></n-icon> {{ texture.likes }}
            </n-tag>
            <n-tag size="small" type="success">
              <n-icon><LayersIcon /></n-icon> {{ texture.versions.length }} {{ t('mods.versions') }}
            </n-tag>
          </n-space>
        </n-space>
      </template>
    </n-thing>
  </n-list-item>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { EyeOutline as EyeIcon, ThumbsUpOutline as ThumbUpIcon, LayersOutline as LayersIcon } from '@vicons/ionicons5'
import type { ModSearchResult } from '../../types/mod-source'

defineProps<{
  texture: ModSearchResult
}>()

defineEmits<{
  click: [texture: ModSearchResult]
}>()

const { t } = useI18n()

function stripHtmlTags(html: string): string {
  const tmp = document.createElement('div')
  tmp.innerHTML = html
  const text = tmp.textContent || tmp.innerText || ''
  return text.length > 100 ? text.substring(0, 100) + '...' : text
}
</script>
