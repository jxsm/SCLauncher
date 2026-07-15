<template>
  <n-list-item>
    <n-thing>
      <template #header>
        <n-space align="center">
          <n-checkbox
            :checked="mod.enabled"
            @update:checked="$emit('toggle', mod, $event)"
          >
            <n-text strong>{{ mod.modInfo?.name || mod.name }}</n-text>
          </n-checkbox>
          <n-tag :type="mod.enabled ? 'success' : 'default'" size="small">
            {{ mod.enabled ? t('mods.enabled') : t('mods.disabled') }}
          </n-tag>
          <!-- 玩法影响等级 -->
          <n-tag
            v-if="impactTag"
            :type="impactTag.type"
            size="small"
          >
            {{ t(impactTag.label) }}
          </n-tag>
          <!-- 旧版 API 警告（ApiVersion < 1.8） -->
          <n-tag
            v-if="mod.modInfo && isApiVersionPotentiallyUnusable(mod.modInfo.apiVersion)"
            type="warning"
            size="small"
          >
            {{ t('mods.apiVersionLow') }}
          </n-tag>
        </n-space>
      </template>

      <template #description>
        <n-space vertical size="small">
          <n-text depth="3">
            {{ t('common.size') }}: {{ formatSize(mod.size) }}
          </n-text>
          <n-text v-if="mod.modInfo?.version" depth="3">
            {{ t('mods.versionLabel') }}: v{{ mod.modInfo.version }}
            <template v-if="mod.modInfo.apiVersion"> · API {{ mod.modInfo.apiVersion }}</template>
          </n-text>
          <n-text depth="3">
            {{ t('mods.installDate') }}: {{ new Date(mod.installDate).toLocaleString() }}
          </n-text>
        </n-space>
      </template>

      <template #action>
        <n-space>
          <n-button
            size="small"
            :disabled="!mod.modInfo"
            @click="$emit('show-info', mod)"
          >
            {{ t('mods.viewModInfo') }}
          </n-button>
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
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatSize } from '../../utils/format'
import { isApiVersionPotentiallyUnusable } from '../../utils/modVersion'
import type { Mod } from '../../types/mod'

const props = defineProps<{
  mod: Mod
}>()

defineEmits<{
  toggle: [mod: Mod, enabled: boolean]
  delete: [mod: Mod]
  'show-info': [mod: Mod]
}>()

const { t } = useI18n()

// 玩法影响等级 → 标签颜色与文案
const impactTag = computed<{ type: 'default' | 'success' | 'warning' | 'error'; label: string } | null>(() => {
  const level: string = props.mod.modInfo?.gameplayImpactLevel ?? ''
  switch (level) {
    case 'Assist':
      return { type: 'success', label: 'mods.impactAssist' }
    case 'Turbo':
      return { type: 'warning', label: 'mods.impactTurbo' }
    case 'Break':
      return { type: 'error', label: 'mods.impactBreak' }
    case 'Godmode':
      return { type: 'error', label: 'mods.impactGodmode' }
    case 'Cosmetic':
      return { type: 'default', label: 'mods.impactCosmetic' }
    default:
      return null
  }
})
</script>
