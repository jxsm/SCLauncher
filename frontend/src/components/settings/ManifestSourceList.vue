<template>
  <n-list v-if="sources.length > 0" hoverable clickable>
    <ManifestSourceListItem
      v-for="source in sources"
      :key="source.id"
      :source="source"
      :is-current="source.id === currentSourceId"
      @set-current="handleSetCurrent"
      @delete="handleDelete"
    />
  </n-list>
  <n-empty v-else :description="t('settings.noManifestSources')" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ManifestSource } from '../../types/manifest-source'
import ManifestSourceListItem from './ManifestSourceListItem.vue'

const { t } = useI18n()

defineProps<{
  sources: ManifestSource[]
  currentSourceId: string
}>()

const emit = defineEmits<{
  'set-current': [source: ManifestSource]
  delete: [source: ManifestSource]
}>()

function handleSetCurrent(source: ManifestSource) {
  emit('set-current', source)
}

function handleDelete(source: ManifestSource) {
  emit('delete', source)
}
</script>
