<template>
  <div class="save-games-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space justify="space-between">
          <n-space>
            <!-- 存档下载视图下显示返回按钮 -->
            <n-button
              v-if="currentView === 'download'"
              @click="switchView('manage')"
            >
              <template #icon>
                <n-icon><ArrowBackIcon /></n-icon>
              </template>
              {{ t('saveGames.backToManage') }}
            </n-button>
            <!-- 存档管理视图下显示原有按钮 -->
            <template v-if="currentView === 'manage'">
              <n-text strong style="font-size: 18px;">{{ t('saveGames.title') }}</n-text>
              <n-select
                v-model:value="selectedVersionId"
                :options="versionOptions"
                :placeholder="t('saveGames.selectVersion')"
                style="width: 300px;"
                @update:value="handleVersionChange"
              />
            </template>
          </n-space>
          <n-space>
            <!-- 存档管理视图下的按钮 -->
            <template v-if="currentView === 'manage'">
              <n-button type="primary" @click="handleImportSave">
                <template #icon>
                  <n-icon><ImportIcon /></n-icon>
                </template>
                {{ t('saveGames.importSave') }}
              </n-button>
              <n-button type="info" @click="switchView('download')">
                <template #icon>
                  <n-icon><DownloadIcon /></n-icon>
                </template>
                {{ t('saveGames.downloadSave') }}
              </n-button>
              <n-text depth="3">
                {{ t('saveGames.totalSaves') }} {{ saveGames.length }}
              </n-text>
            </template>
          </n-space>
        </n-space>
      </n-card>

      <!-- 视图切换区域 -->
      <n-card>
        <transition name="view-fade" mode="out-in">
          <!-- 存档管理视图 -->
          <div v-if="currentView === 'manage'" key="manage" class="view-content">
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
          </div>

          <!-- 存档下载视图 -->
          <div v-else-if="currentView === 'download'" key="download" class="view-content">
            <n-space vertical size="large">
              <!-- 版本选择和下载源选择器 -->
              <n-space align="center" justify="space-between">
                <n-space>
                  <!-- 游戏版本选择 -->
                  <n-select
                    v-model:value="selectedVersionId"
                    :options="installedVersionOptions"
                    :placeholder="t('saveGames.selectVersion')"
                    style="width: 300px"
                    :disabled="installedVersionOptions.length === 0"
                  >
                    <template #prefix>
                      <n-icon><GameControllerIcon /></n-icon>
                    </template>
                  </n-select>

                  <!-- 下载源选择器 -->
                  <n-select
                    v-model:value="selectedSourceId"
                    :options="sourceOptions"
                    style="width: 300px"
                    @update:value="handleSourceChange"
                  >
                    <template #prefix>
                      <n-icon><CloudDownloadIcon /></n-icon>
                    </template>
                  </n-select>
                </n-space>
                <n-button text @click="openSourceSettings">
                  <template #icon>
                    <n-icon><SettingsIcon /></n-icon>
                  </template>
                  {{ t('saveGames.manageSources') }}
                </n-button>
              </n-space>

              <!-- 版本提示 -->
              <n-alert v-if="!selectedVersionId" type="warning" :title="t('saveGames.pleaseSelectVersionFirst')">
                {{ t('saveGames.selectVersionToDownload') }}
              </n-alert>

              <!-- 搜索框 -->
              <n-input
                v-model:value="downloadSearchText"
                :placeholder="t('saveGames.searchOnlineSaves')"
                clearable
                size="large"
                @keyup.enter="handleSearchSaves"
              >
                <template #prefix>
                  <n-icon><SearchIcon /></n-icon>
                </template>
                <template #suffix>
                  <n-button type="primary" @click="handleSearchSaves" :loading="searching">
                    {{ t('common.search') }}
                  </n-button>
                </template>
              </n-input>

              <!-- 搜索结果 -->
              <n-spin :show="searching">
                <n-list v-if="searchResults.length > 0" hoverable clickable>
                  <n-list-item v-for="save in searchResults" :key="save.id" @click="handleShowSaveDetail(save)">
                    <n-thing>
                      <template #header>
                        <n-space align="center">
                          <n-avatar
                            v-if="save.icon"
                            :src="save.icon"
                            :size="48"
                            round
                          />
                          <n-avatar
                            v-else-if="save.authorAvatar"
                            :src="save.authorAvatar"
                            :size="48"
                            round
                          />
                          <n-avatar v-else :size="48" round>
                            {{ save.title.charAt(0) }}
                          </n-avatar>
                          <n-text strong>{{ save.title }}</n-text>
                          <n-tag v-if="save.versions.length > 0" size="small" type="info">
                            v{{ save.versions[0].version }}
                          </n-tag>
                        </n-space>
                      </template>

                      <template #description>
                        <n-space vertical size="small">
                          <n-text depth="3">
                            {{ save.author }}
                          </n-text>
                          <n-text depth="3" :line-clamp="1">
                            {{ stripHtmlTags(save.description) }}
                          </n-text>
                          <n-space>
                            <n-tag size="small" type="info">
                              👁 {{ save.views }}
                            </n-tag>
                            <n-tag v-if="save.likes > 0" size="small" type="warning">
                              👍 {{ save.likes }}
                            </n-tag>
                            <n-tag size="small" type="success">
                              📦 {{ save.versions.length }} {{ t('mods.versions') }}
                            </n-tag>
                          </n-space>
                        </n-space>
                      </template>
                    </n-thing>
                  </n-list-item>
                </n-list>

                <!-- 分页组件 -->
                <div v-if="searchResults.length > 0 && totalPages > 1" style="margin-top: 16px; display: flex; justify-content: center;">
                  <n-pagination
                    v-model:page="currentPage"
                    :page-count="totalPages"
                    @update:page="handlePageChange"
                  />
                </div>

                <n-empty
                  v-else-if="!searching && searchResults.length === 0 && hasSearched"
                  :description="t('saveGames.noSearchResults')"
                />
                <n-empty
                  v-else-if="!searching && searchResults.length === 0 && !hasSearched"
                  :description="t('saveGames.loadingSaves')"
                />
              </n-spin>
            </n-space>
          </div>
        </transition>
      </n-card>

      <!-- 存档详情对话框 -->
      <n-modal
        v-model:show="showSaveDetailModal"
        preset="card"
        :title="selectedSave?.title || ''"
        style="width: 700px;"
      >
        <n-scrollbar style="max-height: 60vh;">
          <n-space vertical size="large">
            <!-- 基本信息 -->
            <n-space vertical size="small">
              <n-text strong>{{ t('common.author') }}:</n-text>
              <n-text>{{ selectedSave?.author }}</n-text>
            </n-space>

            <!-- 描述 -->
            <n-space vertical size="small">
              <n-text strong>{{ t('common.description') }}:</n-text>
              <div v-if="selectedSave" class="save-description" v-html="selectedSave.description"></div>
            </n-space>

            <!-- 统计信息 -->
            <n-space>
              <n-tag size="small" type="info">
                👁 {{ selectedSave?.views }}
              </n-tag>
              <n-tag v-if="selectedSave && selectedSave.likes > 0" size="small" type="warning">
                👍 {{ selectedSave?.likes }}
              </n-tag>
              <n-tag size="small" type="success">
                📦 {{ selectedSave?.versions.length }} {{ t('mods.versions') }}
              </n-tag>
            </n-space>

            <!-- 版本列表 -->
            <n-divider />
            <n-space vertical size="medium">
              <n-text strong>{{ t('mods.availableVersions') }}</n-text>
              <n-list v-if="selectedSave && selectedSave.versions.length > 0" bordered>
                <n-list-item v-for="(version, index) in selectedSave.versions" :key="index">
                  <n-space justify="space-between" align="center" style="width: 100%;">
                    <n-space vertical size="small">
                      <n-text strong>v{{ version.version }}</n-text>
                      <n-text depth="3">
                        {{ t('common.size') }}: {{ formatSize(Number(version.fileSize)) }}
                      </n-text>
                    </n-space>
                    <n-button
                      type="primary"
                      size="small"
                      @click="handleDownloadSave(selectedSave!, index)"
                      :loading="downloadingSaves.has(`${selectedSave!.id}-${index}`)"
                    >
                      <template #icon>
                        <n-icon><DownloadIcon /></n-icon>
                      </template>
                      {{ t('saveGames.downloadSave') }}
                    </n-button>
                  </n-space>
                </n-list-item>
              </n-list>
              <n-empty v-else :description="t('mods.noVersions')" />
            </n-space>
          </n-space>
        </n-scrollbar>

        <template #footer>
          <n-space justify="end">
            <n-button @click="showSaveDetailModal = false">{{ t('common.close') }}</n-button>
          </n-space>
        </template>
      </n-modal>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated, h, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog, NInput } from 'naive-ui'
import { Trash as TrashIcon, CreateOutline as EditIcon, Download as ImportIcon, CloudUploadOutline as ExportIcon, FolderOpen as FolderIcon, ArrowBack as ArrowBackIcon, Download as DownloadIcon, CloudDownload as CloudDownloadIcon, Settings as SettingsIcon, GameController as GameControllerIcon, Search as SearchIcon } from '@vicons/ionicons5'
import { useVersionStore } from '../stores/version'
import { GetSaveGames, DeleteSaveGame, OpenSaveGameFolder, RenameSaveGame, ExportSaveGame, ImportSaveGame, SelectSaveGameFile, PreviewSaveGame, DownloadSaveGameFromURL } from '../api/savegame'
import { ModSourceManager } from '../managers'
import type { SaveGame } from '../types/savegame'
import type { ModSearchResult } from '../types/mod-source'
import { useRouter } from 'vue-router'
import { formatSize } from '../utils/format'

const { t } = useI18n()
const versionStore = useVersionStore()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const selectedVersionId = ref<string>('')

// 存档列表
const saveGames = ref<SaveGame[]>([])

// 视图状态
const currentView = ref<'manage' | 'download'>('manage')

// 存档下载状态
const downloadSearchText = ref<string>('')
const searching = ref(false)
const searchResults = ref<ModSearchResult[]>([])
const downloadingSaves = ref<Set<string>>(new Set())
const hasSearched = ref(false)
const selectedSourceId = ref<string>('')

// 分页状态
const currentPage = ref(1)
const pageSize = 10
const totalPages = ref(0)
const isSearchMode = ref(false)

// 存档详情相关
const showSaveDetailModal = ref(false)
const selectedSave = ref<ModSearchResult | null>(null)

const router = useRouter()

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

// 已安装版本选项（过滤掉路径不存在的版本）
const installedVersionOptions = computed(() => {
  return versionStore.installedVersions
    .filter(v => v.pathExists !== false && v.pathExists !== undefined)
    .map(v => ({
      label: v.name,
      value: v.id
    }))
})

// 存档下载源选项（只显示存档类型的下载源）
const sourceOptions = computed(() => {
  return ModSourceManager.getEnabledSources()
    .filter(source => source.type === 'savegames')
    .map(source => ({
      label: source.name,
      value: source.id
    }))
})

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

// 视图切换函数
function switchView(view: 'manage' | 'download') {
  currentView.value = view

  // 切换到下载视图时自动加载第一页
  if (view === 'download' && !hasSearched.value && !isSearchMode.value) {
    loadSaveList()
  }
}

// 存档下载相关函数

/**
 * 加载存档列表
 */
async function loadSaveList() {
  searching.value = true

  try {
    const response = await ModSourceManager.getModList({
      page: currentPage.value,
      limit: pageSize
    })

    searchResults.value = response.data
    totalPages.value = response.totalPages

    if (response.data.length === 0) {
      message.info(t('saveGames.noSaves'))
    }
  } catch (error) {
    message.error(t('saveGames.searchFailed') + '：' + error)
    searchResults.value = []
    totalPages.value = 0
  } finally {
    searching.value = false
  }
}

/**
 * 搜索存档
 */
async function handleSearchSaves() {
  if (!downloadSearchText.value.trim()) {
    // 如果搜索框为空，切换回浏览模式
    isSearchMode.value = false
    currentPage.value = 1
    hasSearched.value = false
    loadSaveList()
    return
  }

  searching.value = true
  hasSearched.value = true
  isSearchMode.value = true
  currentPage.value = 1

  try {
    const response = await ModSourceManager.searchMods(downloadSearchText.value, {
      page: 1,
      limit: pageSize
    })

    searchResults.value = response.data
    totalPages.value = response.totalPages

    if (response.data.length === 0) {
      message.info(t('saveGames.noSearchResults'))
    } else {
      message.success(t('saveGames.searchSuccess'))
    }
  } catch (error) {
    message.error(t('saveGames.searchFailed') + '：' + error)
    searchResults.value = []
    totalPages.value = 0
  } finally {
    searching.value = false
  }
}

/**
 * 分页改变
 */
async function handlePageChange(page: number) {
  currentPage.value = page

  if (isSearchMode.value) {
    // 搜索模式
    searching.value = true
    try {
      const response = await ModSourceManager.searchMods(downloadSearchText.value, {
        page,
        limit: pageSize
      })

      searchResults.value = response.data
      totalPages.value = response.totalPages
    } catch (error) {
      message.error(t('saveGames.searchFailed') + '：' + error)
    } finally {
      searching.value = false
    }
  } else {
    // 浏览模式
    await loadSaveList()
  }

  // 等待DOM更新后滚动到存档列表顶部
  await nextTick()
  const downloadView = document.querySelector('.view-content')
  if (downloadView) {
    downloadView.scrollIntoView({ behavior: 'smooth', block: 'start' })
  } else {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

/**
 * 下载存档
 */
async function handleDownloadSave(save: ModSearchResult, versionIndexOrString: string | number = 0) {
  if (!selectedVersionId.value) {
    message.warning(t('saveGames.pleaseSelectVersionFirst'))
    return
  }

  // 确定要下载的版本
  let versionIndex = 0
  if (typeof versionIndexOrString === 'string') {
    versionIndex = save.versions.findIndex(v => v.version === versionIndexOrString)
  } else {
    versionIndex = versionIndexOrString
  }

  if (versionIndex < 0 || versionIndex >= save.versions.length) {
    message.error(t('saveGames.versionNotFound'))
    return
  }

  const version = save.versions[versionIndex]
  const downloadKey = `${save.id}-${versionIndex}`

  downloadingSaves.value.add(downloadKey)

  try {
    await DownloadSaveGameFromURL(version.downloadUrl, selectedVersionId.value, version.fileName)
    message.success(t('saveGames.downloadSuccess', { name: save.title }))

    // 下载成功后刷新存档列表
    await loadSaveGames()
  } catch (error) {
    message.error(t('saveGames.downloadFailed') + '：' + error)
  } finally {
    downloadingSaves.value.delete(downloadKey)
  }
}

/**
 * 显示存档详情
 */
function handleShowSaveDetail(save: ModSearchResult) {
  selectedSave.value = save
  showSaveDetailModal.value = true
}

/**
 * 移除HTML标签（用于列表显示）
 */
function stripHtmlTags(html: string): string {
  const tmp = document.createElement('div')
  tmp.innerHTML = html
  const text = tmp.textContent || tmp.innerText || ''
  // 截取前100个字符
  return text.length > 100 ? text.substring(0, 100) + '...' : text
}

/**
 * 下载源变更
 */
function handleSourceChange(sourceId: string) {
  ModSourceManager.setCurrentSource(sourceId)
  // 重置状态并重新加载数据
  searchResults.value = []
  currentPage.value = 1
  hasSearched.value = false
  isSearchMode.value = false
  downloadSearchText.value = ''
  totalPages.value = 0

  // 重新加载第一页数据
  loadSaveList()
}

/**
 * 打开源设置页面
 */
function openSourceSettings() {
  // 打开设置页面并定位到存档下载源管理
  router.push({
    path: '/settings',
    query: { tab: 'savegame-sources' }
  })
}

onMounted(async () => {
  loading.value = true
  try {
    // 初始化下载源（只选择存档类型的源）
    // 优先选择存档类型的默认源
    const defaultSaveSource = ModSourceManager.getAllSources().find(s => s.type === 'savegames' && s.isDefault)
    if (defaultSaveSource) {
      selectedSourceId.value = defaultSaveSource.id
      ModSourceManager.setCurrentSource(defaultSaveSource.id)
    } else {
      // 如果没有默认源，选择第一个启用的存档源
      const firstSaveSource = ModSourceManager.getEnabledSources().find(s => s.type === 'savegames')
      if (firstSaveSource) {
        selectedSourceId.value = firstSaveSource.id
        ModSourceManager.setCurrentSource(firstSaveSource.id)
      }
    }

    // 加载已安装版本列表
    await versionStore.getVersions()

    // Get valid versions (paths exist)
    const validVersions = versionStore.installedVersions.filter(v => v.pathExists !== false && v.pathExists !== undefined)

    // 默认选择主版本
    if (versionStore.primaryVersion && versionStore.primaryVersion.pathExists !== false && versionStore.primaryVersion.pathExists !== undefined) {
      selectedVersionId.value = versionStore.primaryVersion.id
    } else if (validVersions.length > 0) {
      selectedVersionId.value = validVersions[0].id
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

// 当页面激活时重新加载下载源列表
onActivated(async () => {
  await ModSourceManager.reloadSources()

  // 优先选择存档类型的默认源
  const defaultSaveSource = ModSourceManager.getAllSources().find(s => s.type === 'savegames' && s.isDefault)
  if (defaultSaveSource) {
    selectedSourceId.value = defaultSaveSource.id
    ModSourceManager.setCurrentSource(defaultSaveSource.id)
  } else {
    // 如果没有默认源，选择第一个启用的存档源
    const firstSaveSource = ModSourceManager.getEnabledSources().find(s => s.type === 'savegames')
    if (firstSaveSource) {
      selectedSourceId.value = firstSaveSource.id
      ModSourceManager.setCurrentSource(firstSaveSource.id)
    }
  }
})

// 监听下载源选项变化，确保当前选中的源ID始终有效
watch(sourceOptions, (newOptions) => {
  if (newOptions.length > 0) {
    const currentIdExists = newOptions.some(opt => opt.value === selectedSourceId.value)
    if (!currentIdExists) {
      // 当前选中的源ID不存在了，切换到该类型的默认源
      const defaultSaveSource = ModSourceManager.getAllSources().find(s => s.type === 'savegames' && s.isDefault)
      if (defaultSaveSource) {
        selectedSourceId.value = defaultSaveSource.id
        ModSourceManager.setCurrentSource(defaultSaveSource.id)
      } else {
        // 如果没有默认源，切换到第一个可用的存档源
        const firstSource = ModSourceManager.getEnabledSources().find(s => s.type === 'savegames')
        if (firstSource) {
          selectedSourceId.value = firstSource.id
          ModSourceManager.setCurrentSource(firstSource.id)
        }
      }
      // 重置搜索状态
      searchResults.value = []
      currentPage.value = 1
      hasSearched.value = false
      isSearchMode.value = false
      downloadSearchText.value = ''
      totalPages.value = 0
    }
  }
}, { deep: true })
</script>

<style scoped>
.save-games-view {
  max-width: 1000px;
  margin: 0 auto;
}

.view-content {
  min-height: 400px;
}

/* 视图切换动画 */
.view-fade-enter-active,
.view-fade-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.view-fade-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.view-fade-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

.view-fade-enter-to,
.view-fade-leave-from {
  opacity: 1;
  transform: translateX(0);
}
</style>
