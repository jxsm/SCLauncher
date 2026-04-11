<template>
  <n-list-item>
    <n-thing :title="source.name" :description="source.url">
      <template #header-extra>
        <n-space>
          <n-tag v-if="source.isDefault" type="success" size="small">
            {{ t('settings.default') }}
          </n-tag>
          <n-tag v-if="isCurrent" type="primary" size="small">
            {{ t('settings.current') }}
          </n-tag>
        </n-space>
      </template>
      <template #action>
        <n-space>
          <n-button
            v-if="!isCurrent"
            size="small"
            type="primary"
            @click="$emit('set-current', source)"
          >
            {{ t('settings.setAsCurrent') }}
          </n-button>
          <n-button
            v-if="!source.isDefault"
            size="small"
            type="error"
            @click="$emit('delete', source)"
          >
            {{ t('common.delete') }}
          </n-button>
        </n-space>
      </template>
    </n-thing>
  </n-list-item>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ManifestSource } from '../../types/manifest-source'

const { t } = useI18n()

defineProps<{
  source: ManifestSource
  isCurrent: boolean
}>()

defineEmits<{
  'set-current': [source: ManifestSource]
  delete: [source: ManifestSource]
}>()
</script>
