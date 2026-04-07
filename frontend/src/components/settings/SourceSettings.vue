<template>
  <n-card :title="t('mods.manageSources')">
    <template #header-extra>
      <n-button size="small" type="primary" @click="$emit('addSource')">
        <template #icon>
          <n-icon><AddIcon /></n-icon>
        </template>
        {{ t('mods.addSource') }}
      </n-button>
    </template>

    <n-tabs v-model:value="currentSourceType" type="line">
      <n-tab-pane name="mods" :tab="t('mods.modSources')">
        <SourceList
          :sources="getSourcesByType('mods')"
          disabled-source-id="suancaixianyu"
          @toggle="handleToggleSource"
          @set-default="handleSetDefaultSource"
          @delete="handleDeleteSource"
        />
      </n-tab-pane>

      <n-tab-pane name="savegames" :tab="t('mods.savegameSources')">
        <SourceList
          :sources="getSourcesByType('savegames')"
          disabled-source-id="suancaixianyu-saves"
          show-delete-icon
          @toggle="handleToggleSource"
          @set-default="handleSetDefaultSource"
          @delete="handleDeleteSource"
        />
      </n-tab-pane>

      <n-tab-pane name="furniture" :tab="t('mods.furnitureSources')">
        <SourceList
          :sources="getSourcesByType('furniture')"
          disabled-source-id="suancaixianyu"
          @toggle="handleToggleSource"
          @set-default="handleSetDefaultSource"
          @delete="handleDeleteSource"
        />
      </n-tab-pane>

      <n-tab-pane name="textures" :tab="t('mods.textureSources')">
        <SourceList
          :sources="getSourcesByType('textures')"
          disabled-source-id="suancaixianyu"
          @toggle="handleToggleSource"
          @set-default="handleSetDefaultSource"
          @delete="handleDeleteSource"
        />
      </n-tab-pane>

      <n-tab-pane name="skins" :tab="t('mods.skinSources')">
        <SourceList
          :sources="getSourcesByType('skins')"
          disabled-source-id="suancaixianyu"
          @toggle="handleToggleSource"
          @set-default="handleSetDefaultSource"
          @delete="handleDeleteSource"
        />
      </n-tab-pane>
    </n-tabs>
  </n-card>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Add as AddIcon } from '@vicons/ionicons5'
import type { ModSource } from '../../types/mod-source'
import SourceList from './SourceList.vue'

const { t } = useI18n()

const props = defineProps<{
  sources: ModSource[]
}>()

const emit = defineEmits<{
  toggleSource: [sourceId: string, enabled: boolean]
  setDefaultSource: [source: ModSource]
  deleteSource: [source: ModSource]
  addSource: []
}>()

const currentSourceType = ref('mods')

function getSourcesByType(type: string) {
  return props.sources.filter(s => s.type === type)
}

function handleToggleSource(sourceId: string, enabled: boolean) {
  emit('toggleSource', sourceId, enabled)
}

function handleSetDefaultSource(source: ModSource) {
  emit('setDefaultSource', source)
}

function handleDeleteSource(source: ModSource) {
  emit('deleteSource', source)
}
</script>
