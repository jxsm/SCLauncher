<template>
  <n-list-item>
    <n-thing>
      <template #header>
        <n-space align="center">
          <n-checkbox
            :checked="mod.enabled"
            @update:checked="$emit('toggle', mod, $event)"
          >
            <n-text strong>{{ mod.name }}</n-text>
          </n-checkbox>
          <n-tag :type="mod.enabled ? 'success' : 'default'" size="small">
            {{ mod.enabled ? t('mods.enabled') : t('mods.disabled') }}
          </n-tag>
        </n-space>
      </template>

      <template #description>
        <n-space vertical size="small">
          <n-text depth="3">
            {{ t('common.size') }}: {{ formatSize(mod.size) }}
          </n-text>
          <n-text depth="3">
            {{ t('mods.installDate') }}: {{ new Date(mod.installDate).toLocaleString() }}
          </n-text>
        </n-space>
      </template>

      <template #action>
        <n-space>
          <n-popconfirm @positive-click="$emit('delete', mod)">
            <template #trigger>
              <n-button type="error" size="small">
                {{ t('common.delete') }}
              </n-button>
            </template>
            {{ t('mods.confirmDeleteMessage') }}
          </n-popconfirm>
        </n-space>
      </template>
    </n-thing>
  </n-list-item>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { formatSize } from '../../utils/format'
import type { Mod } from '../../types/mod'

defineProps<{
  mod: Mod
}>()

defineEmits<{
  toggle: [mod: Mod, enabled: boolean]
  delete: [mod: Mod]
}>()

const { t } = useI18n()
</script>
