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
                <SaveGameListItem
                  v-for="saveGame in saveGames"
                  :key="saveGame.id"
                  :save-game="saveGame"
                  @open-folder="handleOpenFolder"
                  @export="handleExportSave"
                  @rename="handleRename"
                  @delete="handleDelete"
                />
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
                  <SaveGameSearchResultItem
                    v-for="saveGame in searchResults"
                    :key="saveGame.id"
                    :save-game="saveGame"
                    @click="handleShowSaveDetail"
                  />
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
    </n-space>

    <!-- 存档详情对话框 -->
    <SaveGameDetailModal
      v-model:show="showSaveDetailModal"
      :save-game="selectedSave"
      :downloading="downloadingSaves"
      @download="handleDownloadSave"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, onActivated, watch, nextTick, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog, NInput } from 'naive-ui'
import { Download as ImportIcon, ArrowBack as ArrowBackIcon, Download as DownloadIcon, CloudDownload as CloudDownloadIcon, Settings as SettingsIcon, GameController as GameControllerIcon, Search as SearchIcon } from '@vicons/ionicons5'
import { useVersionStore } from '../stores/version'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { GetSaveGames, DeleteSaveGame, OpenSaveGameFolder, RenameSaveGame, ExportSaveGame, ExportSaveGameAsModpack, ImportSaveGame, SelectSaveGameFile, PreviewSaveGame, DownloadSaveGameFromURL, PreviewSaveRequiredMods, GetSaveRequiredMods } from '../api/savegame'
import { IsOnlineVersion } from '../api/version'
import { useModDependencyResolver } from '../composables/useModDependencyResolver'
import { ModSourceManager } from '../managers'
import type { SaveGame } from '../types/savegame'
import type { ModSearchResult } from '../types/mod-source'
import { useRouter } from 'vue-router'
import SaveGameListItem from '../components/savegame/SaveGameListItem.vue'
import SaveGameSearchResultItem from '../components/savegame/SaveGameSearchResultItem.vue'
import SaveGameDetailModal from '../components/savegame/SaveGameDetailModal.vue'

// 定义props
const props = defineProps<{
  versionIdFromRoute?: string
}>()

const { t } = useI18n()
const versionStore = useVersionStore()
const message = useMessage()
const dialog = useDialog()
const { resolveDependenciesForSave } = useModDependencyResolver()

const isOnlineVersion = ref(false) // 当前版本是否是联机版（决定依赖解析优先 NetMods 源）

// 刷新当前版本是否为联机版（缓存，供依赖解析使用）
async function refreshIsOnlineVersion() {
  if (!selectedVersionId.value) {
    isOnlineVersion.value = false
    return
  }
  try {
    isOnlineVersion.value = await IsOnlineVersion(selectedVersionId.value)
  } catch {
    isOnlineVersion.value = false
  }
}

// 存档所需模组依赖解析（非阻塞：失败仅 console，不影响导入/下载成功提示）
async function resolveSaveDependenciesFromArchive(sourcePath: string) {
  try {
    const required = await PreviewSaveRequiredMods(sourcePath)
    await resolveDependenciesForSave(required, selectedVersionId.value, isOnlineVersion.value)
  } catch (e) {
    console.error('[SaveGames] 解析存档所需模组失败:', e)
  }
}

async function resolveSaveDependenciesFromInstalled(saveId: string) {
  try {
    const required = await GetSaveRequiredMods(selectedVersionId.value, saveId)
    await resolveDependenciesForSave(required, selectedVersionId.value, isOnlineVersion.value)
  } catch (e) {
    console.error('[SaveGames] 解析存档所需模组失败:', e)
  }
}

const loading = ref(false)
const selectedVersionId = ref<string>('')

// 同步选中版本到全局 store，并刷新联机版状态
watch(selectedVersionId, (newVal) => {
  versionStore.selectedVersionId = newVal
  refreshIsOnlineVersion()
})

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
          h('div', { style: 'margin-top: 16px; padding-top: 12px; border-top: 1px solid var(--n-border-color);' }, [
            h('span', { style: 'color: #ff9500;' }, t('saveGames.confirmImportMessage'))
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
          // 导入对话框关闭后再解析所需模组（不阻塞 return，避免对话框卡住/子框叠加）
          // 此时仍持有源文件路径，直接从归档解析
          resolveSaveDependenciesFromArchive(selectedFile)
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

  // 导出格式选择
  const exportFormat = ref<'scworld' | 'modpack'>('scworld')

  // 显示导出格式选择对话框
  dialog.create({
    title: t('saveGames.selectExportFormat'),
    content: () => {
      return h('div', { style: 'padding: 8px 0;' }, [
        h('div', { style: 'margin-bottom: 16px;' }, t('saveGames.selectExportFormatDescription')),
        h('div', { style: 'display: flex; flex-direction: column; gap: 12px;' }, [
          h('label', { style: 'display: flex; align-items: flex-start; gap: 8px; cursor: pointer; padding: 12px; border: 1px solid #e0e0e0; border-radius: 8px; transition: all 0.2s;' }, [
            h('input', {
              type: 'radio',
              value: 'scworld',
              checked: exportFormat.value === 'scworld',
              onChange: () => { exportFormat.value = 'scworld' },
              style: 'margin-top: 2px;'
            }),
            h('div', { style: 'flex: 1;' }, [
              h('strong', { style: 'display: block; margin-bottom: 4px; color: #0066cc;' }, '.scworld'),
              h('div', { style: 'font-size: 12px; color: #7a7a7a;' }, t('saveGames.scworldFormatDescription'))
            ])
          ]),
          h('label', { style: 'display: flex; align-items: flex-start; gap: 8px; cursor: pointer; padding: 12px; border: 1px solid #e0e0e0; border-radius: 8px; transition: all 0.2s;' }, [
            h('input', {
              type: 'radio',
              value: 'modpack',
              checked: exportFormat.value === 'modpack',
              onChange: () => { exportFormat.value = 'modpack' },
              style: 'margin-top: 2px;'
            }),
            h('div', { style: 'flex: 1;' }, [
              h('strong', { style: 'display: block; margin-bottom: 4px; color: #34c759;' }, '整合包 (.scmodpack)'),
              h('div', { style: 'font-size: 12px; color: #7a7a7a;' }, t('saveGames.modpackFormatDescription'))
            ])
          ])
        ])
      ])
    },
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        if (exportFormat.value === 'scworld') {
          const result = await ExportSaveGame(selectedVersionId.value, save.id)
          if (result) {
            message.success(t('saveGames.exportSuccess'))
          }
        } else {
          const result = await ExportSaveGameAsModpack(selectedVersionId.value, save.id)
          if (result) {
            message.success(t('saveGames.exportSuccess'))
          }
        }
        return true
      } catch (error) {
        message.error(t('saveGames.exportFailed') + '：' + error)
        return false
      }
    }
  })
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

  let saveId = ''
  try {
    // 返回落地存档目录名，供后续解析所需模组
    saveId = await DownloadSaveGameFromURL(version.downloadUrl, selectedVersionId.value, version.fileName)
    message.success(t('saveGames.downloadSuccess', { name: save.title }))

    // 下载成功后刷新存档列表
    await loadSaveGames()
  } catch (error) {
    message.error(t('saveGames.downloadFailed') + '：' + error)
    return // 下载失败则不解析依赖
  } finally {
    downloadingSaves.value.delete(downloadKey)
  }

  // 存档本身已下载完成（下载按钮已恢复），再解析所需模组
  if (saveId) {
    await resolveSaveDependenciesFromInstalled(saveId)
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

// 翻译游戏模式
function translateGameMode(mode: string): string {
  return t(`saveGames.gameModes.${mode}`)
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
    await versionStore.getPrimaryVersion()

    // Get valid versions (paths exist)
    const validVersions = versionStore.installedVersions.filter(v => v.pathExists !== false && v.pathExists !== undefined)

    // 优先使用从路由传入的版本ID
    if (props.versionIdFromRoute) {
      const version = validVersions.find(v => v.id === props.versionIdFromRoute)
      if (version) {
        selectedVersionId.value = props.versionIdFromRoute
      } else {
        // 如果指定版本无效，选择主要版本
        const primaryInValid = validVersions.find(v => v.isPrimary)
        if (primaryInValid) {
          selectedVersionId.value = primaryInValid.id
        } else if (validVersions.length > 0) {
          selectedVersionId.value = validVersions[0].id
        }
      }
    } else {
      // 默认选择主版本
      const primaryInValid = validVersions.find(v => v.isPrimary)
      if (primaryInValid) {
        selectedVersionId.value = primaryInValid.id
      } else if (validVersions.length > 0) {
        selectedVersionId.value = validVersions[0].id
      }
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

  EventsOn('dragdrop:imported', async () => {
    if (selectedVersionId.value) {
      await loadSaveGames()
    }
  })
})

onUnmounted(() => {
  EventsOff('dragdrop:imported')
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

/* 视图切换动画 - Apple 风格 */
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
