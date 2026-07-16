<template>
  <n-list-item>
    <n-thing>
      <template #header>
        <n-space align="center">
          <n-text strong style="font-size: 16px;">{{ saveGame.name }}</n-text>
          <n-tag v-if="saveGame.isAutoSave" type="info" size="small">
            {{ t('saveGames.autoSave') }}
          </n-tag>
        </n-space>
      </template>

      <template #description>
        <n-space vertical size="small">
          <n-text depth="3">
            {{ t('saveGames.lastModified') }}: {{ formatDate(saveGame.lastModified) }}
          </n-text>
          <n-text depth="3">
            {{ t('saveGames.gameVersion') }}:
            <n-tag v-if="saveGame.gameVersion" size="tiny" :type="saveGame.gameVersion ? 'info' : 'default'">
              {{ saveGame.gameVersion || t('common.unknown') }}
            </n-tag>
          </n-text>
          <n-text v-if="saveGame.gameMode" depth="3">
            {{ t('saveGames.gameMode') }}:
            <n-tag size="tiny" type="success">
              {{ translateGameMode(saveGame.gameMode) }}
            </n-tag>
          </n-text>
        </n-space>
      </template>

      <template #action>
        <n-space>
          <n-button
            size="medium"
            @click="$emit('open-folder', saveGame)"
          >
            <template #icon>
              <n-icon><FolderIcon /></n-icon>
            </template>
            {{ t('saveGames.openFolder') }}
          </n-button>
          <n-button
            size="medium"
            @click="$emit('export', saveGame)"
          >
            <template #icon>
              <n-icon><ExportIcon /></n-icon>
            </template>
            {{ t('saveGames.exportSave') }}
          </n-button>
          <n-button
            size="medium"
            @click="$emit('rename', saveGame)"
          >
            <template #icon>
              <n-icon><EditIcon /></n-icon>
            </template>
            {{ t('common.rename') }}
          </n-button>
          <n-popconfirm @positive-click="$emit('delete', saveGame)">
            <template #trigger>
              <n-button type="error" size="medium">
                <template #icon>
                  <n-icon><TrashIcon /></n-icon>
                </template>
                {{ t('common.delete') }}
              </n-button>
            </template>
            {{ t('saveGames.confirmDelete', { name: saveGame.name }) }}
          </n-popconfirm>
        </n-space>
      </template>
    </n-thing>
  </n-list-item>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { FolderOpen as FolderIcon, CloudUploadOutline as ExportIcon, CreateOutline as EditIcon, Trash as TrashIcon } from '@vicons/ionicons5'
import type { SaveGame } from '../../types/savegame'

defineProps<{
  saveGame: SaveGame
}>()

defineEmits<{
  'open-folder': [saveGame: SaveGame]
  export: [saveGame: SaveGame]
  rename: [saveGame: SaveGame]
  delete: [saveGame: SaveGame]
}>()

const { t, te } = useI18n()

function formatDate(date: string | Date): string {
  const d = typeof date === 'string' ? new Date(date) : date
  return d.toLocaleString()
}

function translateGameMode(mode: string): string {
  const key = `saveGames.gameModes.${mode}`
  // 未匹配到翻译时直接显示原始模式名（不翻译）
  return te(key) ? t(key) : mode
}
</script>
