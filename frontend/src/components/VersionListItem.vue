<template>
  <n-list-item
    :style="isPathMissing ? 'background-color: rgba(255, 0, 0, 0.05);' : ''"
  >
    <n-thing>
      <template #header>
        <n-space align="center">
          <n-text strong style="font-size: 16px;">{{ version.name }}</n-text>
          <n-tag v-if="version.isPrimary" type="success" size="small">
            {{ t('installed.primary') }}
          </n-tag>
          <n-tag v-if="!isImportedVersion" :type="versionTypeColor" size="small">
            {{ versionTypeText }}
          </n-tag>
          <n-tag v-if="!isPathMissing" type="success" size="small">
            {{ t('versions.installed') }}
          </n-tag>
          <n-tag v-if="isPathMissing" type="error" size="small">
            {{ t('installed.pathMissing') }}
          </n-tag>
        </n-space>
      </template>

      <template #description>
        <n-space vertical size="small">
          <n-text depth="3">
            {{ t('common.version') }}: {{ version.gameVersion }} - {{ version.subVersion }}
          </n-text>
          <n-text v-if="isPathMissing" type="error" style="margin-top: 8px;">
            ⚠️ {{ t('installed.pathMissingMessage') }}
          </n-text>
        </n-space>
      </template>

      <template #action>
        <n-space>
          <template v-if="!isPathMissing">
            <n-button
              type="success"
              size="medium"
              :disabled="isGameRunning"
              @click="$emit('launch', version)"
            >
              <template #icon>
                <n-icon><PlayIcon /></n-icon>
              </template>
              {{ t('installed.launchGame') }}
            </n-button>
            <n-button
              size="medium"
              @click="$emit('setPrimary', version)"
              :disabled="version.isPrimary"
              :type="version.isPrimary ? 'success' : 'default'"
            >
              <template #icon>
                <n-icon><StarIcon /></n-icon>
              </template>
              {{ version.isPrimary ? t('installed.alreadyPrimary') : t('versions.setAsPrimary') }}
            </n-button>
            <n-button
              size="medium"
              @click="$emit('openFolder', version)"
            >
              <template #icon>
                <n-icon><FolderIcon /></n-icon>
              </template>
              {{ t('versions.openFolder') }}
            </n-button>
            <n-button
              size="medium"
              @click="$emit('manageResources', version)"
            >
              <template #icon>
                <n-icon><ResourcesIcon /></n-icon>
              </template>
              {{ t('installed.manageResources') }}
            </n-button>
            <n-button
              size="medium"
              @click="$emit('rename', version)"
            >
              <template #icon>
                <n-icon><EditIcon /></n-icon>
              </template>
              {{ t('versions.rename') }}
            </n-button>
          </template>
          <n-popconfirm @positive-click="$emit('delete', version)">
            <template #trigger>
              <n-button type="error" size="medium">
                <template #icon>
                  <n-icon><TrashIcon /></n-icon>
                </template>
                {{ t('common.delete') }}
              </n-button>
            </template>
            {{ t('installed.confirmDeleteVersion', { name: version.name }) }}
          </n-popconfirm>
        </n-space>
      </template>
    </n-thing>
  </n-list-item>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Play as PlayIcon, Star as StarIcon, Trash as TrashIcon, CreateOutline as EditIcon, FolderOpen as FolderIcon, CubeOutline as ResourcesIcon } from '@vicons/ionicons5'
import type { Version } from '../types/version'

const { t } = useI18n()

const props = defineProps<{
  version: Version
  isPathMissing: boolean
  isGameRunning: boolean
}>()

const emit = defineEmits<{
  launch: [version: Version]
  setPrimary: [version: Version]
  openFolder: [version: Version]
  manageResources: [version: Version]
  rename: [version: Version]
  delete: [version: Version]
}>()

const isImportedVersion = computed(() => {
  return props.version.id.startsWith('imported-') || props.version.versionType === 'unknown'
})

const versionTypeText = computed(() => {
  const types = {
    api: t('versions.apiVersion'),
    net: t('versions.netVersion'),
    original: t('versions.originalVersion')
  }
  return types[props.version.versionType as keyof typeof types] || props.version.versionType
})

const versionTypeColor = computed(() => {
  switch (props.version.versionType) {
    case 'api': return 'info'
    case 'net': return 'warning'
    case 'original': return 'success'
    default: return 'default'
  }
})
</script>
