<template>
  <n-list-item @click="$emit('click', saveGame)">
    <n-thing>
      <template #header>
        <n-space align="center">
          <n-avatar
            v-if="saveGame.icon"
            :src="saveGame.icon"
            :size="48"
            round
          />
          <n-avatar
            v-else-if="saveGame.authorAvatar"
            :src="saveGame.authorAvatar"
            :size="48"
            round
          />
          <n-avatar v-else :size="48" round>
            {{ saveGame.title.charAt(0) }}
          </n-avatar>
          <n-text strong>{{ saveGame.title }}</n-text>
          <n-tag v-if="saveGame.versions.length > 0" size="small" type="info">
            v{{ saveGame.versions[0].version }}
          </n-tag>
        </n-space>
      </template>

      <template #description>
        <n-space vertical size="small">
          <n-text depth="3">
            {{ saveGame.author }}
          </n-text>
          <n-text depth="3" :line-clamp="1">
            {{ stripHtmlTags(saveGame.description) }}
          </n-text>
          <n-space>
            <n-tag size="small" type="info">
              👁 {{ saveGame.views }}
            </n-tag>
            <n-tag v-if="saveGame.likes > 0" size="small" type="warning">
              👍 {{ saveGame.likes }}
            </n-tag>
            <n-tag size="small" type="success">
              📦 {{ saveGame.versions.length }} {{ t('mods.versions') }}
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
  saveGame: ModSearchResult
}>()

defineEmits<{
  click: [saveGame: ModSearchResult]
}>()

const { t } = useI18n()

function stripHtmlTags(html: string): string {
  const tmp = document.createElement('div')
  tmp.innerHTML = html
  const text = tmp.textContent || tmp.innerText || ''
  return text.length > 100 ? text.substring(0, 100) + '...' : text
}
</script>
