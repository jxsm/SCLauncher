<template>
  <n-list-item @click="$emit('click', mod)">
    <n-thing>
      <template #header>
        <n-space align="center">
          <n-avatar
            v-if="mod.icon"
            :src="mod.icon"
            :size="48"
            round
          />
          <n-avatar
            v-else-if="mod.authorAvatar"
            :src="mod.authorAvatar"
            :size="48"
            round
          />
          <n-avatar v-else :size="48" round>
            {{ mod.title.charAt(0) }}
          </n-avatar>
          <n-text strong>{{ mod.title }}</n-text>
          <n-tag v-if="mod.versions.length > 0" size="small" type="info">
            v{{ mod.versions[0].version }}
          </n-tag>
        </n-space>
      </template>

      <template #description>
        <n-space vertical size="small">
          <n-text depth="3">
            {{ mod.author }}
          </n-text>
          <n-text depth="3" :line-clamp="1">
            {{ stripHtmlTags(mod.description) }}
          </n-text>
          <n-space>
            <n-tag size="small" type="info">
              👁 {{ mod.views }}
            </n-tag>
            <n-tag v-if="mod.likes > 0" size="small" type="warning">
              👍 {{ mod.likes }}
            </n-tag>
            <n-tag v-if="mod.versions.length > 0" size="small" type="success">
              {{ mod.versions.length }} {{ t('mods.versions') }}
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
  mod: ModSearchResult
}>()

defineEmits<{
  click: [mod: ModSearchResult]
}>()

const { t } = useI18n()

function stripHtmlTags(html: string): string {
  const tmp = document.createElement('div')
  tmp.innerHTML = html
  const text = tmp.textContent || tmp.innerText || ''
  return text.length > 100 ? text.substring(0, 100) + '...' : text
}
</script>
