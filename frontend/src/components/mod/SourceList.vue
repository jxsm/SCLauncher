<template>
  <div class="source-list">
    <n-list v-if="sources.length > 0" hoverable>
      <SourceListItem
        v-for="source in sources"
        :key="source.id"
        :source="source"
        :disabled-source-id="disabledSourceId"
        @toggle="handleToggleSource"
        @set-default="handleSetDefaultSource"
        @delete="handleDeleteSource"
      />
    </n-list>
    <n-empty v-else :description="t('mods.noSources')" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import SourceListItem from './SourceListItem.vue'
import type { ModSource } from '../../types/mod-source'

defineProps<{
  sources: ModSource[]
  disabledSourceId?: string
}>()

const emit = defineEmits<{
  toggle: [sourceId: string, enabled: boolean]
  'set-default': [source: ModSource]
  delete: [source: ModSource]
}>()

const { t } = useI18n()

function handleToggleSource(sourceId: string, enabled: boolean) {
  emit('toggle', sourceId, enabled)
}

function handleSetDefaultSource(source: ModSource) {
  emit('set-default', source)
}

function handleDeleteSource(source: ModSource) {
  emit('delete', source)
}
</script>

<style scoped>
.source-list {
  width: 100%;
}
</style>
