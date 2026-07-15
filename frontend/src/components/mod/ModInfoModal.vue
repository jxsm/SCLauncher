<template>
  <n-modal
    :show="show"
    @update:show="$emit('update:show', $event)"
    preset="card"
    :title="mod?.modInfo?.name || mod?.name || t('mods.modInfoTitle')"
    style="width: 700px;"
  >
    <n-scrollbar style="max-height: 60vh;">
      <n-space vertical size="large" v-if="mod?.modInfo">
        <!-- 旧版 API 警告 -->
        <n-alert
          v-if="isApiVersionPotentiallyUnusable(mod.modInfo.apiVersion)"
          type="warning"
          :show-icon="true"
        >
          {{ t('mods.apiVersionLowFull') }}
        </n-alert>

        <!-- 可能不兼容（仅当能推断主机 API 版本时） -->
        <n-alert
          v-if="hostApiVersion && mod.modInfo.apiVersion && isVersionTextIncompatible(hostApiVersion, mod.modInfo.apiVersion)"
          type="warning"
          :show-icon="true"
        >
          {{ t('mods.mayBeIncompatible') }}
        </n-alert>

        <!-- 徽章 -->
        <n-space>
          <n-tag v-if="impactTag" :type="impactTag.type" size="small">
            {{ t(impactTag.label) }}
          </n-tag>
          <n-tag size="small" type="info">
            {{ mod.modInfo.packageName }}
          </n-tag>
          <n-tag v-if="mod.modInfo.loadOrder !== 0" size="small">
            {{ t('mods.loadOrder') }}: {{ mod.modInfo.loadOrder }}
          </n-tag>
          <n-tag v-if="mod.modInfo.nonPersistentMod" size="small" type="default">
            {{ t('mods.nonPersistent') }}
          </n-tag>
        </n-space>

        <!-- 基本字段 -->
        <n-space vertical size="small" v-if="mod.modInfo.author">
          <n-text strong>{{ t('common.author') }}:</n-text>
          <n-text>{{ mod.modInfo.author }}</n-text>
        </n-space>

        <n-space vertical size="small">
          <n-text strong>{{ t('mods.versionLabel') }}:</n-text>
          <n-text>
            v{{ mod.modInfo.version || '-' }}
            <template v-if="mod.modInfo.apiVersion"> · API {{ mod.modInfo.apiVersion }}</template>
            <template v-if="mod.modInfo.scVersion"> · SC {{ mod.modInfo.scVersion }}</template>
          </n-text>
        </n-space>

        <!-- 描述 -->
        <n-space v-if="mod.modInfo.description" vertical size="small">
          <n-text strong>{{ t('common.description') }}:</n-text>
          <n-text>{{ mod.modInfo.description }}</n-text>
        </n-space>

        <!-- 链接 -->
        <n-space v-if="mod.modInfo.link" vertical size="small">
          <n-text strong>{{ t('mods.link') }}:</n-text>
          <n-button text tag="a" :href="mod.modInfo.link" target="_blank" type="primary">
            {{ mod.modInfo.link }}
          </n-button>
        </n-space>

        <!-- 文件信息 -->
        <n-divider />
        <n-space vertical size="small">
          <n-text depth="3">{{ t('mods.fileName') }}: {{ mod.fileName }}</n-text>
          <n-text depth="3">{{ t('common.size') }}: {{ formatSize(mod.size) }}</n-text>
        </n-space>

        <!-- 依赖 -->
        <n-divider />
        <n-space vertical size="medium">
          <n-text strong>{{ t('mods.dependencies') }}</n-text>
          <n-list v-if="mod.modInfo.dependencies.length > 0" bordered>
            <n-list-item v-for="(dep, i) in mod.modInfo.dependencies" :key="i">
              <n-space justify="space-between" align="center" style="width: 100%;">
                <n-text>{{ dep.displayName || dep.packageName }}</n-text>
                <n-tag v-if="dep.versionRange" size="small" type="info">{{ dep.versionRange }}</n-tag>
                <n-tag v-else size="small">{{ t('mods.anyVersion') }}</n-tag>
              </n-space>
            </n-list-item>
          </n-list>
          <n-empty v-else :description="t('mods.noDependencies')" />
        </n-space>
      </n-space>

      <!-- 无法解析 modinfo -->
      <n-space v-else vertical align="center" style="padding: 40px 0;">
        <n-empty :description="t('mods.modInfoParseFailed')" />
      </n-space>
    </n-scrollbar>

    <template #footer>
      <n-space justify="end">
        <n-button @click="$emit('update:show', false)">{{ t('common.close') }}</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatSize } from '../../utils/format'
import { isApiVersionPotentiallyUnusable, isVersionTextIncompatible } from '../../utils/modVersion'
import type { Mod } from '../../types/mod'

const props = defineProps<{
  show: boolean
  mod: Mod | null
  hostApiVersion?: string
}>()

defineEmits<{
  'update:show': [show: boolean]
}>()

const { t } = useI18n()

// 玩法影响等级 → 标签颜色与文案
const impactTag = computed<{ type: 'default' | 'success' | 'warning' | 'error'; label: string } | null>(() => {
  const level: string = props.mod?.modInfo?.gameplayImpactLevel ?? ''
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
