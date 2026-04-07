<template>
  <n-list-item>
    <n-thing>
      <template #header>
        <n-space align="center">
          <n-avatar v-if="source.icon" :src="source.icon" :size="32" round />
          <n-avatar v-else :size="32" round>
            {{ source.name.charAt(0) }}
          </n-avatar>
          <n-text strong>{{ source.name }}</n-text>
          <n-tag v-if="source.isDefault" size="small" type="info">
            {{ t('mods.defaultSource') }}
          </n-tag>
          <n-switch
            :value="source.enabled"
            :disabled="source.id === disabledSourceId"
            @update:value="handleToggle"
          />
        </n-space>
      </template>

      <template #description>
        <n-text depth="3">
          {{ source.description }}
        </n-text>
      </template>

      <template #action>
        <n-space>
          <n-button
            size="small"
            :type="source.isDefault ? 'warning' : 'default'"
            @click="handleSetDefault"
          >
            <template #icon>
              <n-icon><StarIcon /></n-icon>
            </template>
            {{ source.isDefault ? t('mods.cancelDefault') : t('mods.setDefault') }}
          </n-button>
          <n-button
            v-if="!source.isDefault"
            size="small"
            type="error"
            :disabled="source.id === disabledSourceId"
            @click="handleDelete"
          >
            <template #icon>
              <n-icon v-if="showDeleteIcon"><TrashIcon /></n-icon>
            </template>
            {{ t('common.delete') }}
          </n-button>
        </n-space>
      </template>
    </n-thing>
  </n-list-item>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Star as StarIcon, TrashOutline as TrashIcon } from '@vicons/ionicons5'
import type { ModSource } from '../../types/mod-source'

const { t } = useI18n()

defineProps<{
  source: ModSource
  disabledSourceId?: string
  showDeleteIcon?: boolean
}>()

const emit = defineEmits<{
  toggle: [enabled: boolean]
  setDefault: []
  delete: []
}>()

function handleToggle(enabled: boolean) {
  emit('toggle', enabled)
}

function handleSetDefault() {
  emit('setDefault')
}

function handleDelete() {
  emit('delete')
}
</script>
