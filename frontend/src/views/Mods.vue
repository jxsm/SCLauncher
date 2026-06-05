<template>
  <div class="mods-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space vertical size="medium">
          <n-space justify="space-between">
            <n-space>
              <!-- 模组下载视图下显示返回按钮 -->
              <n-button
                v-if="currentView === 'download'"
                @click="switchView('manage')"
              >
                <template #icon>
                  <n-icon><ArrowBackIcon /></n-icon>
                </template>
                {{ t('mods.backToManage') }}
              </n-button>
              <!-- 模组管理视图下显示原有按钮 -->
              <template v-if="currentView === 'manage'">
                <n-select
                  v-model:value="selectedVersion"
                  :options="installedVersionOptions"
                  :placeholder="t('mods.selectVersion')"
                  style="width: 300px"
                  @update:value="handleVersionChange"
                />
                <n-button
                  type="primary"
                  @click="handleImportMod"
                  :disabled="!selectedVersion"
                >
                  <template #icon>
                    <n-icon><AddIcon /></n-icon>
                  </template>
                  {{ t('mods.importMod') }}
                </n-button>
                <n-button
                  @click="handleOpenModsFolder"
                  :disabled="!selectedVersion"
                >
                  <template #icon>
                    <n-icon><FolderIcon /></n-icon>
                  </template>
                  {{ t('mods.openModsFolder') }}
                </n-button>
                <n-button
                  type="info"
                  @click="switchView('download')"
                >
                  <template #icon>
                    <n-icon><DownloadIcon /></n-icon>
                  </template>
                  {{ t('mods.downloadMod') }}
                </n-button>
              </template>
            </n-space>
            <n-text depth="3" v-if="currentView === 'manage'">
              {{ t('mods.totalMods', { total: modStore.mods?.length || 0, displayed: filteredMods.length }) }}
            </n-text>
          </n-space>

          <!-- 搜索和筛选 (仅在管理视图显示) -->
          <n-space v-if="currentView === 'manage'">
            <n-input
              v-model:value="searchText"
              :placeholder="t('mods.searchPlaceholder')"
              clearable
              style="width: 300px"
            >
              <template #prefix>
                <n-icon><SearchIcon /></n-icon>
              </template>
            </n-input>
            <n-select
              v-model:value="filterType"
              :options="filterOptions"
              style="width: 150px"
            />
          </n-space>
        </n-space>
      </n-card>

      <!-- 视图切换区域 -->
      <n-card>
        <transition name="view-fade" mode="out-in">
          <!-- 模组管理视图 -->
          <div v-if="currentView === 'manage'" key="manage" class="view-content">

            <!-- 模组列表 -->
            <div v-if="modStore.loading" style="min-height: 200px; display: flex; align-items: center; justify-content: center;">
              <n-spin size="large" />
            </div>
            <template v-else>
              <n-list v-if="filteredMods.length > 0" hoverable clickable>
                <ModListItem
                  v-for="mod in filteredMods"
                  :key="mod.id"
                  :mod="mod"
                  @toggle="handleToggleMod"
                  @delete="handleDeleteMod"
                />
              </n-list>
              <n-empty
                v-else
                :description="searchText || filterType !== 'all' ? t('mods.noMatchingMods') : t('mods.noMods')"
                style="padding: 60px 0;"
              />
            </template>
          </div>

          <!-- 模组下载视图 -->
          <div v-else-if="currentView === 'download'" key="download" class="view-content">
            <n-space vertical size="large">
              <!-- 版本选择和下载源选择器 -->
              <n-space align="center" justify="space-between">
                <n-space>
                  <!-- 游戏版本选择 -->
                  <n-select
                    v-model:value="selectedVersion"
                    :options="installedVersionOptions"
                    :placeholder="t('mods.selectVersion')"
                    style="width: 300px"
                    :disabled="installedVersionOptions.length === 0"
                    @update:value="handleVersionChange"
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
                  {{ t('mods.manageSources') }}
                </n-button>
              </n-space>

              <!-- 版本提示 -->
              <n-alert v-if="!selectedVersion" type="warning" :title="t('mods.pleaseSelectVersionFirst')">
                {{ t('mods.selectVersionToDownload') }}
              </n-alert>

              <!-- 搜索框 -->
              <n-input
                v-model:value="downloadSearchText"
                :placeholder="t('mods.searchOnlineMods')"
                clearable
                size="large"
                @keyup.enter="handleSearchMods"
              >
                <template #prefix>
                  <n-icon><SearchIcon /></n-icon>
                </template>
                <template #suffix>
                  <n-button type="primary" @click="handleSearchMods" :loading="searching">
                    {{ t('common.search') }}
                  </n-button>
                </template>
              </n-input>

              <!-- 搜索结果 -->
              <n-spin :show="searching">
                <n-list v-if="searchResults.length > 0" hoverable clickable>
                  <ModSearchResultItem
                    v-for="mod in searchResults"
                    :key="mod.id"
                    :mod="mod"
                    @click="handleShowModDetail"
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
                  :description="t('mods.noSearchResults')"
                />
                <n-empty
                  v-else-if="!searching && searchResults.length === 0 && !hasSearched"
                  :description="t('mods.loadingMods')"
                />
              </n-spin>
            </n-space>
          </div>
        </transition>
      </n-card>
    </n-space>

    <!-- 模组详情对话框 -->
    <ModDetailModal
      v-model:show="showModDetailModal"
      :mod="selectedMod"
      :downloading="downloadingMods"
      @download="handleDownloadMod"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import { useModStore } from '../stores/mod'
import { useVersionStore } from '../stores/version'
import { useMessage } from 'naive-ui'
import { Add as AddIcon, Search as SearchIcon, FolderOpen as FolderIcon, ArrowBack as ArrowBackIcon, Download as DownloadIcon, CloudDownload as CloudDownloadIcon, Settings as SettingsIcon, GameController as GameControllerIcon } from '@vicons/ionicons5'
import { OpenVersionModsFolder } from '../api/version'
import { ModSourceManager } from '../managers'
import type { ModSearchResult, ModSource } from '../types/mod-source'
import ModListItem from '../components/mod/ModListItem.vue'
import ModSearchResultItem from '../components/mod/ModSearchResultItem.vue'
import ModDetailModal from '../components/mod/ModDetailModal.vue'

const { t } = useI18n()
const modStore = useModStore()
const versionStore = useVersionStore()
const message = useMessage()
const router = useRouter()
const route = useRoute()

// 视图状态
const currentView = ref<'manage' | 'download'>('manage')

// 模组管理状态
const selectedVersion = ref<string>('')
const searchText = ref<string>('')
const filterType = ref<string>('all')

// 模组下载状态
const downloadSearchText = ref<string>('')
const searching = ref(false)
const searchResults = ref<ModSearchResult[]>([])
const downloadingMods = ref<Set<string>>(new Set())
const hasSearched = ref(false)
const selectedSourceId = ref<string>('')

// 版本类型检测
const isOnlineVersion = ref(false) // 当前版本是否是联机版
const isInitialized = ref(false) // 是否已完成初始化

// 分页状态
const currentPage = ref(1)
const pageSize = 10 // 固定每页10条
const totalPages = ref(0)
const isSearchMode = ref(false) // 是否处于搜索模式

// 模组详情相关
const showModDetailModal = ref(false)
const selectedMod = ref<ModSearchResult | null>(null)

// Filter options
const filterOptions = computed(() => [
  { label: t('mods.all'), value: 'all' },
  { label: t('mods.enabled'), value: 'enabled' },
  { label: t('mods.disabled'), value: 'disabled' }
])

// Installed version options (filter out versions with missing paths)
const installedVersionOptions = computed(() => {
  return versionStore.installedVersions
    .filter(v => v.pathExists !== false && v.pathExists !== undefined)
    .map(v => ({
      label: v.name,
      value: v.id
    }))
})

// Filtered mods based on search and filter
const filteredMods = computed(() => {
  let mods = modStore.mods || []

  // Apply status filter
  if (filterType.value === 'enabled') {
    mods = mods.filter(m => m.enabled)
  } else if (filterType.value === 'disabled') {
    mods = mods.filter(m => !m.enabled)
  }

  // Apply search filter
  if (searchText.value.trim()) {
    const searchLower = searchText.value.toLowerCase().trim()
    mods = mods.filter(m =>
      m.name.toLowerCase().includes(searchLower)
    )
  }

  return mods
})

// 下载源选项（根据版本类型显示相应的下载源）
const sourceOptions = computed(() => {
  const sourceType = isOnlineVersion.value ? 'online-mods' : 'mods'
  return ModSourceManager.getEnabledSources()
    .filter(source => source.type === sourceType)
    .map(source => ({
      label: source.name,
      value: source.id
    }))
})

// 检测版本是否是联机版
async function checkVersionType() {
  if (!selectedVersion.value) {
    isOnlineVersion.value = false
    return
  }

  try {
    const { IsOnlineVersion } = await import('../api/version')
    isOnlineVersion.value = await IsOnlineVersion(selectedVersion.value)
  } catch (error) {
    console.error('检测版本类型失败:', error)
    isOnlineVersion.value = false
  }
}

async function handleVersionChange() {
  if (selectedVersion.value) {
    await modStore.loadMods(selectedVersion.value)
    // Reset filters when changing version
    searchText.value = ''
    filterType.value = 'all'

    // 检测版本类型（会触发 watch(isOnlineVersion) 自动切换源）
    const oldIsOnline = isOnlineVersion.value
    await checkVersionType()

    // 如果版本类型改变了，watch 会自动处理
    // 如果版本类型没变，但当前在下载视图，需要手动重新加载
    if (oldIsOnline === isOnlineVersion.value && currentView.value === 'download') {
      // 重置搜索状态
      searchResults.value = []
      currentPage.value = 1
      hasSearched.value = false
      isSearchMode.value = false
      downloadSearchText.value = ''
      totalPages.value = 0

      // 重新加载第一页数据
      await loadModList()
    }
  }
}

async function handleImportMod() {
  if (!selectedVersion.value) {
    message.warning(t('mods.noVersionSelected'))
    return
  }

  try {
    // Use Wails file selection dialog
    const { SelectModFile } = await import('../api/mod')
    const filePath = await SelectModFile()

    if (filePath) {
      await modStore.importMod(selectedVersion.value, filePath)
      message.success(t('mods.importSuccess'))
    }
  } catch (error) {
    message.error(t('mods.importFailed') + '：' + error)
  }
}

async function handleOpenModsFolder() {
  if (!selectedVersion.value) {
    message.warning(t('mods.noVersionSelected'))
    return
  }

  try {
    await OpenVersionModsFolder(selectedVersion.value)
  } catch (error) {
    message.error(t('mods.openFolderFailed') + '：' + error)
  }
}

function handleToggleMod(mod: any, enabled: boolean) {
  modStore.toggleMod(selectedVersion.value, mod.id, enabled)
    .catch((error) => {
      message.error(t('mods.toggleFailed') + '：' + error)
      // Revert the UI change on error
      mod.enabled = !enabled
    })
}

function handleDeleteMod(mod: any) {
  modStore.deleteMod(selectedVersion.value, mod.id)
    .then(() => {
      message.success(t('mods.deleteSuccess'))
    })
    .catch((error) => {
      message.error(t('mods.deleteFailed') + '：' + error)
    })
}

// 视图切换函数
async function switchView(view: 'manage' | 'download') {
  currentView.value = view

  // 切换到下载视图时自动加载第一页
  if (view === 'download' && !hasSearched.value && !isSearchMode.value) {
    await loadModList()
  }
}

// 模组下载相关函数

/**
 * 加载模组列表
 */
async function loadModList() {
  searching.value = true

  try {
    const response = await ModSourceManager.getModList({
      page: currentPage.value,
      limit: pageSize
    })

    searchResults.value = response.data
    totalPages.value = response.totalPages

    if (response.data.length === 0) {
      message.info(t('mods.noMods'))
    }
  } catch (error) {
    message.error(t('mods.searchFailed') + '：' + error)
    searchResults.value = []
    totalPages.value = 0
  } finally {
    searching.value = false
  }
}

/**
 * 搜索模组
 */
async function handleSearchMods() {
  if (!downloadSearchText.value.trim()) {
    // 如果搜索框为空，切换回浏览模式
    isSearchMode.value = false
    currentPage.value = 1
    hasSearched.value = false
    loadModList()
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
      message.info(t('mods.noSearchResults'))
    } else {
      message.success(t('mods.searchSuccess'))
    }
  } catch (error) {
    message.error(t('mods.searchFailed') + '：' + error)
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
      message.error(t('mods.searchFailed') + '：' + error)
    } finally {
      searching.value = false
    }
  } else {
    // 浏览模式
    await loadModList()
  }

  // 等待DOM更新后滚动到模组列表顶部
  await nextTick()
  const downloadView = document.querySelector('.view-content')
  if (downloadView) {
    downloadView.scrollIntoView({ behavior: 'smooth', block: 'start' })
  } else {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

async function handleDownloadMod(mod: ModSearchResult, versionIndexOrString: string | number = 0) {
  if (!selectedVersion.value) {
    message.warning(t('mods.pleaseSelectVersionFirst'))
    return
  }

  // 确定要下载的版本
  let versionIndex = 0
  if (typeof versionIndexOrString === 'string') {
    versionIndex = mod.versions.findIndex(v => v.version === versionIndexOrString)
  } else {
    versionIndex = versionIndexOrString
  }

  if (versionIndex < 0 || versionIndex >= mod.versions.length) {
    message.error(t('mods.versionNotFound'))
    return
  }

  const version = mod.versions[versionIndex]
  const downloadKey = `${mod.id}-${versionIndex}`

  downloadingMods.value.add(downloadKey)

  try {
    // 调用Go后端下载接口，传递文件名
    const { DownloadModFromUrl } = await import('../api/mod')
    await DownloadModFromUrl(version.downloadUrl, selectedVersion.value, version.fileName)

    message.success(t('mods.downloadSuccess', { name: mod.title }))

    // 下载成功后刷新模组列表
    await modStore.loadMods(selectedVersion.value)
  } catch (error) {
    message.error(t('mods.downloadFailed') + '：' + error)
  } finally {
    downloadingMods.value.delete(downloadKey)
  }
}

function getVersionOptions(mod: ModSearchResult) {
  return mod.versions.map((v, index) => ({
    label: `v${v.version}`,
    key: index
  }))
}

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
  loadModList()
}

function openSourceSettings() {
  // 打开设置页面并定位到模组下载源管理
  router.push({
    path: '/settings',
    query: { tab: 'mod-sources' }
  })
}

/**
 * 显示模组详情
 */
function handleShowModDetail(mod: ModSearchResult) {
  selectedMod.value = mod
  showModDetailModal.value = true
}

onMounted(async () => {
  try {
    // 初始化下载源（只选择模组类型的源）
    const currentSource = ModSourceManager.getCurrentSource()
    if (currentSource && currentSource.type === 'mods') {
      selectedSourceId.value = currentSource.id
    } else {
      // 如果当前源不是模组类型，选择第一个模组类型的源
      const firstModSource = ModSourceManager.getEnabledSources().find(s => s.type === 'mods')
      if (firstModSource) {
        selectedSourceId.value = firstModSource.id
        ModSourceManager.setCurrentSource(firstModSource.id)
      }
    }

    await versionStore.getVersions()
    await versionStore.getPrimaryVersion()

    // Get valid versions (paths exist)
    const validVersions = versionStore.installedVersions.filter(v => v.pathExists !== false && v.pathExists !== undefined)

    // Check if versionId is provided in route query
    const versionIdFromRoute = route.query.versionId as string
    if (versionIdFromRoute) {
      // Check if the version exists and path exists
      const version = validVersions.find(v => v.id === versionIdFromRoute)
      if (version) {
        selectedVersion.value = versionIdFromRoute
        await modStore.loadMods(selectedVersion.value)
        // 检测版本类型
        await checkVersionType()
        // 标记初始化完成
        isInitialized.value = true
        return
      }
      message.warning(t('mods.versionPathMissing'))
    }

    // Select primary version by default (if it exists in valid versions)
    const primaryInValid = validVersions.find(v => v.isPrimary)
    if (primaryInValid) {
      selectedVersion.value = primaryInValid.id
      await modStore.loadMods(selectedVersion.value)
      // 检测版本类型
      await checkVersionType()
    } else if (validVersions.length > 0) {
      // If no primary version in valid versions, select the first valid version
      selectedVersion.value = validVersions[0].id
      await modStore.loadMods(selectedVersion.value)
      // 检测版本类型
      await checkVersionType()
    } else {
      // No valid versions found
      message.warning(t('mods.noValidVersions'))
    }

    // 标记初始化完成
    isInitialized.value = true
  } catch (error) {
    message.error(t('mods.loadVersionsFailed') + '：' + error)
  }
})

// 当页面激活时重新加载下载源列表
onActivated(async () => {
  await ModSourceManager.reloadSources()

  // 根据版本类型选择相应的默认源
  const sourceType = isOnlineVersion.value ? 'online-mods' : 'mods'
  const defaultSource = ModSourceManager.getAllSources().find(s => s.type === sourceType && s.isDefault)
  if (defaultSource) {
    selectedSourceId.value = defaultSource.id
    ModSourceManager.setCurrentSource(defaultSource.id)
  } else {
    // 如果没有默认源，选择第一个启用的相应类型源
    const firstSource = ModSourceManager.getEnabledSources().find(s => s.type === sourceType)
    if (firstSource) {
      selectedSourceId.value = firstSource.id
      ModSourceManager.setCurrentSource(firstSource.id)
    }
  }

  // 如果当前在下载视图，重新加载模组列表
  if (currentView.value === 'download') {
    // 重置搜索状态
    searchResults.value = []
    currentPage.value = 1
    hasSearched.value = false
    isSearchMode.value = false
    downloadSearchText.value = ''
    totalPages.value = 0

    // 重新加载第一页数据
    await loadModList()
  }
})

// 监听下载源选项变化，确保当前选中的源ID始终有效
watch(sourceOptions, async (newOptions) => {
  if (newOptions.length > 0) {
    const currentIdExists = newOptions.some(opt => opt.value === selectedSourceId.value)
    if (!currentIdExists) {
      // 当前选中的源ID不存在了，切换到该类型的默认源
      const sourceType = isOnlineVersion.value ? 'online-mods' : 'mods'
      const defaultSource = ModSourceManager.getAllSources().find(s => s.type === sourceType && s.isDefault)
      if (defaultSource) {
        selectedSourceId.value = defaultSource.id
        ModSourceManager.setCurrentSource(defaultSource.id)
      } else {
        // 如果没有默认源，切换到第一个可用的相应类型源
        const firstSource = ModSourceManager.getEnabledSources().find(s => s.type === sourceType)
        if (firstSource) {
          selectedSourceId.value = firstSource.id
          ModSourceManager.setCurrentSource(firstSource.id)
        }
      }

      // 仅在当前是下载视图时，才重置搜索状态并重新加载模组列表
      if (currentView.value === 'download') {
        // 重置搜索状态
        searchResults.value = []
        currentPage.value = 1
        hasSearched.value = false
        isSearchMode.value = false
        downloadSearchText.value = ''
        totalPages.value = 0

        // 重新加载第一页数据
        await loadModList()
      }
    }
  }
}, { deep: true })

// 监听版本类型变化，自动切换下载源并重新加载模组列表（仅在下载视图和初始化完成后）
watch(isOnlineVersion, async (newValue, oldValue) => {
  // 如果版本类型没有实际变化，不执行操作
  if (newValue === oldValue) {
    return
  }

  // 如果还未完成初始化，不执行操作（避免初始化时的重复触发）
  if (!isInitialized.value) {
    return
  }

  // 根据版本类型切换下载源
  const sourceType = newValue ? 'online-mods' : 'mods'
  const defaultSource = ModSourceManager.getAllSources().find(s => s.type === sourceType && s.isDefault)

  if (defaultSource) {
    selectedSourceId.value = defaultSource.id
    ModSourceManager.setCurrentSource(defaultSource.id)
  } else {
    // 如果没有默认源，选择第一个启用的相应类型源
    const firstSource = ModSourceManager.getEnabledSources().find(s => s.type === sourceType)
    if (firstSource) {
      selectedSourceId.value = firstSource.id
      ModSourceManager.setCurrentSource(firstSource.id)
    }
  }

  // 仅在当前是下载视图时，才重新加载模组列表
  if (currentView.value === 'download') {
    // 重置搜索状态
    searchResults.value = []
    currentPage.value = 1
    hasSearched.value = false
    isSearchMode.value = false
    downloadSearchText.value = ''
    totalPages.value = 0

    // 重新加载第一页数据
    await loadModList()
  }
})
</script>

<style scoped>
.mods-view {
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
