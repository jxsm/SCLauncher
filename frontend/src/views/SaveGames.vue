<template>
  <div class="save-games-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space justify="space-between">
          <n-space>
            <n-text strong style="font-size: 18px;">{{ t('saveGames.title') }}</n-text>
            <n-select
              v-model:value="selectedVersionId"
              :options="versionOptions"
              :placeholder="t('saveGames.selectVersion')"
              style="width: 300px;"
              @update:value="handleVersionChange"
            />
          </n-space>
          <n-space>
            <n-button type="primary" @click="handleImportSave">
              <template #icon>
                <n-icon><ImportIcon /></n-icon>
              </template>
              {{ t('saveGames.importSave') }}
            </n-button>
            <n-text depth="3">
              {{ t('saveGames.totalSaves') }} {{ saveGames.length }}
            </n-text>
          </n-space>
        </n-space>
      </n-card>

      <!-- 存档列表 -->
      <n-spin :show="loading">
        <n-list hoverable clickable>
          <n-list-item
            v-for="save in saveGames"
            :key="save.id"
          >
            <n-thing>
              <template #header>
                <n-space align="center">
                  <n-text strong style="font-size: 16px;">{{ save.name }}</n-text>
                  <n-tag v-if="save.isAutoSave" type="info" size="small">
                    {{ t('saveGames.autoSave') }}
                  </n-tag>
                </n-space>
              </template>

              <template #description>
                <n-space vertical size="small">
                  <n-text depth="3">
                    {{ t('saveGames.lastModified') }}: {{ formatDate(save.lastModified) }}
                  </n-text>
                  <n-text depth="3">
                    {{ t('saveGames.gameVersion') }}:
                    <n-tag v-if="save.gameVersion" size="tiny" :type="save.gameVersion ? 'info' : 'default'">
                      {{ save.gameVersion || t('common.unknown') }}
                    </n-tag>
                  </n-text>
                  <n-text v-if="save.gameMode" depth="3">
                    {{ t('saveGames.gameMode') }}:
                    <n-tag size="tiny" type="success">
                      {{ translateGameMode(save.gameMode) }}
                    </n-tag>
                  </n-text>
                </n-space>
              </template>

              <template #action>
                <n-space>
                  <n-button
                    size="medium"
                    @click="handleOpenFolder(save)"
                  >
                    <template #icon>
                      <n-icon><FolderIcon /></n-icon>
                    </template>
                    {{ t('saveGames.openFolder') }}
                  </n-button>
                  <n-button
                    size="medium"
                    @click="handleExportSave(save)"
                  >
                    <template #icon>
                      <n-icon><ExportIcon /></n-icon>
                    </template>
                    {{ t('saveGames.exportSave') }}
                  </n-button>
                  <n-button
                    size="medium"
                    @click="handleRename(save)"
                  >
                    <template #icon>
                      <n-icon><EditIcon /></n-icon>
                    </template>
                    {{ t('common.rename') }}
                  </n-button>
                  <n-popconfirm
                    @positive-click="handleDelete(save)"
                  >
                    <template #trigger>
                      <n-button type="error" size="medium">
                        <template #icon>
                          <n-icon><TrashIcon /></n-icon>
                        </template>
                        {{ t('common.delete') }}
                      </n-button>
                    </template>
                    {{ t('saveGames.confirmDelete', { name: save.name }) }}
                  </n-popconfirm>
                </n-space>
              </template>
            </n-thing>
          </n-list-item>
        </n-list>
        <n-empty v-if="saveGames.length === 0 && !loading" :description="t('saveGames.noSaves')">
          <template #extra>
            <n-button type="primary" @click="handleImportSave">
              {{ t('saveGames.importFirstSave') }}
            </n-button>
          </template>
        </n-empty>
      </n-spin>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog, NInput } from 'naive-ui'
import { Trash as TrashIcon, CreateOutline as EditIcon, Download as ImportIcon, CloudUploadOutline as ExportIcon, FolderOpen as FolderIcon } from '@vicons/ionicons5'
import { useVersionStore } from '../stores/version'
import { GetSaveGames, DeleteSaveGame, OpenSaveGameFolder, RenameSaveGame, ExportSaveGame, ImportSaveGame, SelectSaveGameFile, PreviewSaveGame } from '../api/savegame'
import type { SaveGame } from '../types/savegame'

const { t } = useI18n()
const versionStore = useVersionStore()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const selectedVersionId = ref<string>('')

// 存档列表
const saveGames = ref<SaveGame[]>([])

// 版本选项
const versionOptions = computed(() => {
  return versionStore.installedVersions.map(v => ({
    label: v.name,
    value: v.id
  }))
})

// 格式化日期
function formatDate(date: string | Date): string {
  const d = typeof date === 'string' ? new Date(date) : date
  return d.toLocaleString()
}

// 翻译游戏模式
function translateGameMode(mode: string): string {
  return t(`saveGames.gameModes.${mode}`)
}

// 加载存档列表
async function loadSaveGames() {
  if (!selectedVersionId.value) {
    return
  }

  loading.value = true
  try {
    saveGames.value = await GetSaveGames(selectedVersionId.value)
  } catch (error) {
    message.error(t('saveGames.loadFailed') + '：' + error)
    saveGames.value = []
  } finally {
    loading.value = false
  }
}

// 版本切换
function handleVersionChange(versionId: string) {
  selectedVersionId.value = versionId
  loadSaveGames()
}

// 导入存档
async function handleImportSave() {
  if (!selectedVersionId.value) {
    message.error(t('saveGames.noVersionSelected'))
    return
  }

  try {
    // 选择文件
    const selectedFile = await SelectSaveGameFile()
    if (!selectedFile) {
      return // 用户取消
    }

    // 预览存档信息
    const preview = await PreviewSaveGame(selectedFile)

    // 显示确认对话框
    dialog.create({
      title: t('saveGames.importSave'),
      content: () => {
        return h('div', { style: 'padding: 8px 0;' }, [
          h('div', { style: 'margin-bottom: 12px;' }, [
            h('strong', { style: 'display: inline-block; width: 100px;' }, t('saveGames.saveName') + ':'),
            h('span', preview.name || t('common.unknown'))
          ]),
          h('div', { style: 'margin-bottom: 12px;' }, [
            h('strong', { style: 'display: inline-block; width: 100px;' }, t('saveGames.gameVersion') + ':'),
            h('span', preview.gameVersion || t('common.unknown'))
          ]),
          h('div', { style: 'margin-bottom: 12px;' }, [
            h('strong', { style: 'display: inline-block; width: 100px;' }, t('saveGames.gameMode') + ':'),
            h('span', preview.gameMode ? translateGameMode(preview.gameMode) : t('common.unknown'))
          ]),
          h('div', { style: 'margin-top: 16px; padding-top: 12px; border-top: 1px solid #e0e0e0;' }, [
            h('span', { style: 'color: #f0a020;' }, t('saveGames.confirmImportMessage'))
          ])
        ])
      },
      positiveText: t('common.confirm'),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        try {
          // 执行导入
          await ImportSaveGame(selectedVersionId.value, selectedFile)
          message.success(t('saveGames.importSuccess'))
          await loadSaveGames()
          return true
        } catch (error) {
          message.error(t('saveGames.importFailed') + '：' + error)
          return false
        }
      }
    })
  } catch (error) {
    message.error(t('saveGames.importFailed') + '：' + error)
  }
}

// 导出存档
async function handleExportSave(save: SaveGame) {
  if (!selectedVersionId.value) {
    message.error(t('saveGames.noVersionSelected'))
    return
  }

  try {
    const result = await ExportSaveGame(selectedVersionId.value, save.id)
    // 只有在真正导出时才显示成功消息
    if (result) {
      message.success(t('saveGames.exportSuccess'))
    }
    // 如果 result 为 false/undefined，说明用户取消了，不显示任何消息
  } catch (error) {
    message.error(t('saveGames.exportFailed') + '：' + error)
  }
}

// 重命名存档
function handleRename(save: SaveGame) {
  const newName = ref(save.name)

  dialog.create({
    title: t('saveGames.renameSave'),
    content: () => {
      return h('div', [
        h('div', { style: 'margin-bottom: 8px' }, t('saveGames.enterNewSaveName')),
        h(NInput, {
          value: newName.value,
          placeholder: save.name,
          onUpdateValue: (value: string) => {
            newName.value = value
          }
        })
      ])
    },
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      if (!newName.value.trim()) {
        message.error(t('saveGames.nameCannotBeEmpty'))
        return false
      }

      if (newName.value.trim() === save.name) {
        message.info(t('saveGames.nameUnchanged'))
        return true
      }

      try {
        await RenameSaveGame(selectedVersionId.value, save.id, newName.value.trim())
        message.success(t('saveGames.renameSuccess'))
        await loadSaveGames()
        return true
      } catch (error) {
        message.error(t('saveGames.renameFailed') + '：' + error)
        return false
      }
    }
  })
}

// 删除存档
async function handleDelete(save: SaveGame) {
  if (!selectedVersionId.value) {
    message.error(t('saveGames.noVersionSelected'))
    return
  }

  try {
    await DeleteSaveGame(selectedVersionId.value, save.id)
    message.success(t('saveGames.deleteSuccess'))
    // 重新加载存档列表
    await loadSaveGames()
  } catch (error) {
    message.error(t('saveGames.deleteFailed') + '：' + error)
  }
}

// 打开存档文件夹
async function handleOpenFolder(save: SaveGame) {
  if (!selectedVersionId.value) {
    message.error(t('saveGames.noVersionSelected'))
    return
  }

  try {
    await OpenSaveGameFolder(selectedVersionId.value, save.id)
  } catch (error) {
    message.error(t('saveGames.openFolderFailed') + '：' + error)
  }
}

onMounted(async () => {
  loading.value = true
  try {
    // 加载已安装版本列表
    await versionStore.getVersions()

    // 默认选择主版本
    if (versionStore.primaryVersion) {
      selectedVersionId.value = versionStore.primaryVersion.id
    } else if (versionStore.installedVersions.length > 0) {
      selectedVersionId.value = versionStore.installedVersions[0].id
    }

    // 加载存档列表
    if (selectedVersionId.value) {
      await loadSaveGames()
    }
  } catch (error) {
    message.error(t('errors.loadDataFailed') + '：' + error)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.save-games-view {
  max-width: 1000px;
  margin: 0 auto;
}
</style>
