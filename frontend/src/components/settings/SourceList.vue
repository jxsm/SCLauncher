<template>
  <n-list v-if="sources.length > 0" hoverable>
    <SourceListItem
      v-for="source in sources"
      :key="source.id"
      :source="source"
      :disabled-source-id="disabledSourceId"
      :show-delete-icon="showDeleteIcon"
      @toggle="(enabled) => $emit('toggle', source.id, enabled)"
      @set-default="$emit('setDefault', source)"
      @delete="$emit('delete', source)"
    />
  </n-list>
  <n-empty v-else :description="t('mods.noSources')" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import SourceListItem from './SourceListItem.vue'
import type { ModSource } from '../../types/mod-source'

const { t } = useI18n()

defineProps<{
  sources: ModSource[]
  disabledSourceId?: string
  showDeleteIcon?: boolean
}>()

defineEmits<{
  toggle: [sourceId: string, enabled: boolean]
  setDefault: [source: ModSource]
  delete: [source: ModSource]
}>()
</script>
