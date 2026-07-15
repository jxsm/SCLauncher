<template>
  <n-list-item :class="{ 'mod-incompatible': incompatible }">
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
          <!-- 与当前游戏版本可能不兼容 -->
          <n-tag
            v-if="crossIncompatible"
            type="error"
            size="small"
          >
            {{ t('mods.incompatible') }}
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
import { isApiVersionPotentiallyUnusable, isVersionTextIncompatible } from '../../utils/modVersion'
import type { Mod } from '../../types/mod'

const props = defineProps<{
  mod: Mod
  hostApiVersion?: string
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

// 跨版本不兼容：仅当选中版本能推断出主机 API 版本时才判定
const crossIncompatible = computed(() => {
  const api = props.mod.modInfo?.apiVersion
  return !!(props.hostApiVersion && api && isVersionTextIncompatible(props.hostApiVersion, api))
})

// 是否需要特殊高亮（旧版 API 或 跨版本不兼容）
const incompatible = computed(() => {
  if (!props.mod.modInfo) return false
  return isApiVersionPotentiallyUnusable(props.mod.modInfo.apiVersion) || crossIncompatible.value
})
</script>

<style scoped>
/* 不兼容模组特殊高亮：暖色背景 + 左侧强调条（不挤占布局） */
.mod-incompatible {
  background-color: rgba(232, 108, 26, 0.10);
  box-shadow: inset 4px 0 0 0 #e06c1a;
}

</style>
