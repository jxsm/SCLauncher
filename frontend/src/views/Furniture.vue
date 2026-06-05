<template>
  <div class="furniture-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space justify="space-between">
          <n-space>
            <!-- 家具下载视图下显示返回按钮 -->
            <n-button
              v-if="currentView === 'download'"
              @click="switchView('manage')"
            >
              <template #icon>
                <n-icon><ArrowBackIcon /></n-icon>
              </template>
              {{ t('furniture.backToManage') }}
            </n-button>
            <!-- 家具管理视图下显示原有按钮 -->
            <template v-if="currentView === 'manage'">
              <n-text strong style="font-size: 18px;">{{ t('furniture.title') }}</n-text>
              <n-select
                v-model:value="selectedVersionId"
                :options="versionOptions"
                :placeholder="t('furniture.selectVersion')"
                style="width: 300px;"
                @update:value="handleVersionChange"
              />
            </template>
          </n-space>
          <n-space>
            <!-- 家具管理视图下的按钮 -->
            <template v-if="currentView === 'manage'">
              <n-button type="primary" @click="handleImportFurniture">
                <template #icon>
                  <n-icon><ImportIcon /></n-icon>
                </template>
                {{ t('furniture.importFurniture') }}
              </n-button>
              <n-button @click="handleOpenFurnitureFolder">
                <template #icon>
                  <n-icon><FolderIcon /></n-icon>
                </template>
                {{ t('furniture.openFolder') }}
              </n-button>
              <n-button type="info" @click="switchView('download')">
                <template #icon>
                  <n-icon><DownloadIcon /></n-icon>
                </template>
                {{ t('furniture.downloadFurniture') }}
              </n-button>
              <n-text depth="3">
                {{ t('furniture.totalFurnitures') }} {{ furnitures.length }}
              </n-text>
            </template>
          </n-space>
        </n-space>
      </n-card>

      <!-- 视图切换区域 -->
      <n-card>
        <transition name="view-fade" mode="out-in">
          <!-- 家具管理视图 -->
          <div v-if="currentView === 'manage'" key="manage" class="view-content">
            <!-- 家具列表 -->
            <n-spin :show="loading">
              <n-list hoverable clickable>
                <FurnitureListItem
                  v-for="furniture in furnitures"
                  :key="furniture.id"
                  :furniture="furniture"
                  @rename="handleRename"
                  @delete="handleDelete"
                />
              </n-list>
              <n-empty v-if="furnitures.length === 0 && !loading && !folderNotFound" :description="t('furniture.noFurnitures')">
                <template #extra>
                  <n-button type="primary" @click="handleImportFurniture">
                    {{ t('furniture.importFirstFurniture') }}
                  </n-button>
                </template>
              </n-empty>
              <n-empty v-if="folderNotFound && !loading" :description="t('furniture.folderNotFound')">
              </n-empty>
            </n-spin>
          </div>

          <!-- 家具下载视图 -->
          <div v-else-if="currentView === 'download'" key="download" class="view-content">
            <n-space vertical size="large">
              <!-- 版本选择和下载源选择器 -->
              <n-space align="center" justify="space-between">
                <n-space>
                  <!-- 游戏版本选择 -->
                  <n-select
                    v-model:value="selectedVersionId"
                    :options="installedVersionOptions"
                    :placeholder="t('furniture.selectVersion')"
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
                  {{ t('furniture.manageSources') }}
                </n-button>
              </n-space>

              <!-- 版本提示 -->
              <n-alert v-if="!selectedVersionId" type="warning" :title="t('furniture.pleaseSelectVersionFirst')">
                {{ t('furniture.selectVersionToDownload') }}
              </n-alert>

              <!-- 搜索框 -->
              <n-input
                v-model:value="downloadSearchText"
                :placeholder="t('furniture.searchOnlineFurnitures')"
                clearable
                size="large"
                @keyup.enter="handleSearchFurnitures"
              >
                <template #prefix>
                  <n-icon><SearchIcon /></n-icon>
                </template>
                <template #suffix>
                  <n-button type="primary" @click="handleSearchFurnitures" :loading="searching">
                    {{ t('common.search') }}
                  </n-button>
                </template>
              </n-input>

              <!-- 搜索结果 -->
              <n-spin :show="searching">
                <n-list v-if="searchResults.length > 0" hoverable clickable>
                  <FurnitureSearchResultItem
                    v-for="furniture in searchResults"
                    :key="furniture.id"
                    :furniture="furniture"
                    @click="handleShowFurnitureDetail"
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
                  :description="t('furniture.noSearchResults')"
                />
                <n-empty
                  v-else-if="!searching && searchResults.length === 0 && !hasSearched"
                  :description="t('furniture.loadingFurnitures')"
                />
              </n-spin>
            </n-space>
          </div>
        </transition>
      </n-card>
    </n-space>

    <!-- 家具详情对话框 -->
    <FurnitureDetailModal
      v-model:show="showFurnitureDetailModal"
      :furniture="selectedFurniture"
      :downloading="downloadingFurnitures"
      @download="handleDownloadFurniture"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, onActivated, watch, nextTick, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog, NInput } from 'naive-ui'
import {
  FolderOpen as FolderIcon,
  CloudUploadOutline as ImportIcon,
  ArrowBack as ArrowBackIcon,
  Download as DownloadIcon,
  CloudDownload as CloudDownloadIcon,
  Settings as SettingsIcon,
  GameController as GameControllerIcon,
  Search as SearchIcon
} from '@vicons/ionicons5'
import { useVersionStore } from '../stores/version'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import {
  GetFurnitures,
  DeleteFurniture,
  OpenFurnitureFolder,
  RenameFurniture,
  SelectFurnitureFile,
  ImportFurniture,
  DownloadFurnitureFromURL
} from '../api/furniture'
import { ModSourceManager } from '../managers'
import type { Furniture } from '../types/furniture'
import type { ModSearchResult } from '../types/mod-source'
import { useRouter } from 'vue-router'
import FurnitureListItem from '../components/furniture/FurnitureListItem.vue'
import FurnitureSearchResultItem from '../components/furniture/FurnitureSearchResultItem.vue'
import FurnitureDetailModal from '../components/furniture/FurnitureDetailModal.vue'

// 定义props
const props = defineProps<{
  versionIdFromRoute?: string
}>()

const { t } = useI18n()
const versionStore = useVersionStore()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const selectedVersionId = ref<string>('')
const folderNotFound = ref(false)

// 同步选中版本到全局 store
watch(selectedVersionId, (newVal) => {
  versionStore.selectedVersionId = newVal
})

// 家具列表
const furnitures = ref<Furniture[]>([])

// 视图状态
const currentView = ref<'manage' | 'download'>('manage')

// 家具下载状态
const downloadSearchText = ref<string>('')
const searching = ref(false)
const searchResults = ref<ModSearchResult[]>([])
const downloadingFurnitures = ref<Set<string>>(new Set())
const hasSearched = ref(false)
const selectedSourceId = ref<string>('')

// 分页状态
const currentPage = ref(1)
const pageSize = 10
const totalPages = ref(0)
const isSearchMode = ref(false)

// 家具详情相关
const showFurnitureDetailModal = ref(false)
const selectedFurniture = ref<ModSearchResult | null>(null)

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

// 家具下载源选项（只显示家具类型的下载源）
const sourceOptions = computed(() => {
  return ModSourceManager.getEnabledSources()
    .filter(source => source.type === 'furniture')
    .map(source => ({
      label: source.name,
      value: source.id
    }))
})

// 加载家具列表
async function loadFurnitures() {
  if (!selectedVersionId.value) {
    return
  }

  loading.value = true
  folderNotFound.value = false
  try {
    const result = await GetFurnitures(selectedVersionId.value)
    if (result === null) {
      // 文件夹不存在
      furnitures.value = []
      folderNotFound.value = true
    } else {
      furnitures.value = result
    }
  } catch (error) {
    message.error(t('furniture.loadFailed') + '：' + error)
    furnitures.value = []
    folderNotFound.value = false
  } finally {
    loading.value = false
  }
}

// 版本切换
function handleVersionChange(versionId: string) {
  selectedVersionId.value = versionId
  loadFurnitures()
}

// 导入家具
async function handleImportFurniture() {
  if (!selectedVersionId.value) {
    message.error(t('furniture.noVersionSelected'))
    return
  }

  try {
    // 选择文件
    const selectedFile = await SelectFurnitureFile()
    if (!selectedFile) {
      return // 用户取消
    }

    // 执行导入
    await ImportFurniture(selectedVersionId.value, selectedFile)
    message.success(t('furniture.importSuccess'))
    await loadFurnitures()
  } catch (error) {
    message.error(t('furniture.importFailed') + '：' + error)
  }
}

// 重命名家具
function handleRename(furniture: Furniture) {
  const newName = ref(furniture.name)

  dialog.create({
    title: t('furniture.renameFurniture'),
    content: () => {
      return h('div', [
        h('div', { style: 'margin-bottom: 8px' }, t('furniture.enterNewName')),
        h(NInput, {
          value: newName.value,
          placeholder: furniture.name,
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
        message.error(t('furniture.nameCannotBeEmpty'))
        return false
      }

      if (newName.value.trim() === furniture.name) {
        message.info(t('furniture.nameUnchanged'))
        return true
      }

      try {
        await RenameFurniture(selectedVersionId.value, furniture.id, newName.value.trim())
        message.success(t('furniture.renameSuccess'))
        await loadFurnitures()
        return true
      } catch (error) {
        message.error(t('furniture.renameFailed') + '：' + error)
        return false
      }
    }
  })
}

// 删除家具
async function handleDelete(furniture: Furniture) {
  if (!selectedVersionId.value) {
    message.error(t('furniture.noVersionSelected'))
    return
  }

  try {
    await DeleteFurniture(selectedVersionId.value, furniture.id)
    message.success(t('furniture.deleteSuccess'))
    // 重新加载家具列表
    await loadFurnitures()
  } catch (error) {
    message.error(t('furniture.deleteFailed') + '：' + error)
  }
}

// 打开家具文件夹
async function handleOpenFurnitureFolder() {
  if (!selectedVersionId.value) {
    message.error(t('furniture.noVersionSelected'))
    return
  }

  try {
    await OpenFurnitureFolder(selectedVersionId.value)
  } catch (error) {
    message.error(t('furniture.openFolderFailed') + '：' + error)
  }
}

// 视图切换函数
function switchView(view: 'manage' | 'download') {
  currentView.value = view

  // 切换到下载视图时自动加载第一页
  if (view === 'download' && !hasSearched.value && !isSearchMode.value) {
    loadFurnitureList()
  }
}

// 家具下载相关函数

/**
 * 加载家具列表
 */
async function loadFurnitureList() {
  searching.value = true

  try {
    const response = await ModSourceManager.getModList({
      page: currentPage.value,
      limit: pageSize
    })

    searchResults.value = response.data
    totalPages.value = response.totalPages

    if (response.data.length === 0) {
      message.info(t('furniture.noFurnitures'))
    }
  } catch (error) {
    message.error(t('furniture.searchFailed') + '：' + error)
    searchResults.value = []
    totalPages.value = 0
  } finally {
    searching.value = false
  }
}

/**
 * 搜索家具
 */
async function handleSearchFurnitures() {
  if (!downloadSearchText.value.trim()) {
    // 如果搜索框为空，切换回浏览模式
    isSearchMode.value = false
    currentPage.value = 1
    hasSearched.value = false
    loadFurnitureList()
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
      message.info(t('furniture.noSearchResults'))
    } else {
      message.success(t('furniture.searchSuccess'))
    }
  } catch (error) {
    message.error(t('furniture.searchFailed') + '：' + error)
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
      message.error(t('furniture.searchFailed') + '：' + error)
    } finally {
      searching.value = false
    }
  } else {
    // 浏览模式
    await loadFurnitureList()
  }

  // 等待DOM更新后滚动到家具列表顶部
  await nextTick()
  const downloadView = document.querySelector('.view-content')
  if (downloadView) {
    downloadView.scrollIntoView({ behavior: 'smooth', block: 'start' })
  } else {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

/**
 * 下载家具
 */
async function handleDownloadFurniture(furniture: ModSearchResult, versionIndexOrString: string | number = 0) {
  if (!selectedVersionId.value) {
    message.warning(t('furniture.pleaseSelectVersionFirst'))
    return
  }

  // 确定要下载的版本
  let versionIndex = 0
  if (typeof versionIndexOrString === 'string') {
    versionIndex = furniture.versions.findIndex(v => v.version === versionIndexOrString)
  } else {
    versionIndex = versionIndexOrString
  }

  if (versionIndex < 0 || versionIndex >= furniture.versions.length) {
    message.error(t('furniture.versionNotFound'))
    return
  }

  const version = furniture.versions[versionIndex]
  const downloadKey = `${furniture.id}-${versionIndex}`

  downloadingFurnitures.value.add(downloadKey)

  try {
    await DownloadFurnitureFromURL(version.downloadUrl, selectedVersionId.value, version.fileName)
    message.success(t('furniture.downloadSuccess', { name: furniture.title }))

    // 下载成功后刷新家具列表
    await loadFurnitures()
  } catch (error) {
    message.error(t('furniture.downloadFailed') + '：' + error)
  } finally {
    downloadingFurnitures.value.delete(downloadKey)
  }
}

/**
 * 显示家具详情
 */
function handleShowFurnitureDetail(furniture: ModSearchResult) {
  selectedFurniture.value = furniture
  showFurnitureDetailModal.value = true
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
  loadFurnitureList()
}

/**
 * 打开源设置页面
 */
function openSourceSettings() {
  // 打开设置页面并定位到家具下载源管理
  router.push({
    path: '/settings',
    query: { tab: 'furniture-sources' }
  })
}

onMounted(async () => {
  loading.value = true
  try {
    // 初始化下载源（只选择家具类型的源）
    // 优先选择家具类型的默认源
    const defaultFurnitureSource = ModSourceManager.getAllSources().find(s => s.type === 'furniture' && s.isDefault)
    if (defaultFurnitureSource) {
      selectedSourceId.value = defaultFurnitureSource.id
      ModSourceManager.setCurrentSource(defaultFurnitureSource.id)
    } else {
      // 如果没有默认源，选择第一个启用的家具源
      const firstFurnitureSource = ModSourceManager.getEnabledSources().find(s => s.type === 'furniture')
      if (firstFurnitureSource) {
        selectedSourceId.value = firstFurnitureSource.id
        ModSourceManager.setCurrentSource(firstFurnitureSource.id)
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

    // 加载家具列表
    if (selectedVersionId.value) {
      await loadFurnitures()
    }
  } catch (error) {
    message.error(t('errors.loadDataFailed') + '：' + error)
  } finally {
    loading.value = false
  }

  EventsOn('dragdrop:imported', async () => {
    if (selectedVersionId.value) {
      await loadFurnitures()
    }
  })
})

onUnmounted(() => {
  EventsOff('dragdrop:imported')
})

// 当页面激活时重新加载下载源列表
onActivated(async () => {
  await ModSourceManager.reloadSources()

  // 优先选择家具类型的默认源
  const defaultFurnitureSource = ModSourceManager.getAllSources().find(s => s.type === 'furniture' && s.isDefault)
  if (defaultFurnitureSource) {
    selectedSourceId.value = defaultFurnitureSource.id
    ModSourceManager.setCurrentSource(defaultFurnitureSource.id)
  } else {
    // 如果没有默认源，选择第一个启用的家具源
    const firstFurnitureSource = ModSourceManager.getEnabledSources().find(s => s.type === 'furniture')
    if (firstFurnitureSource) {
      selectedSourceId.value = firstFurnitureSource.id
      ModSourceManager.setCurrentSource(firstFurnitureSource.id)
    }
  }
})

// 监听下载源选项变化，确保当前选中的源ID始终有效
watch(sourceOptions, (newOptions) => {
  if (newOptions.length > 0) {
    const currentIdExists = newOptions.some(opt => opt.value === selectedSourceId.value)
    if (!currentIdExists) {
      // 当前选中的源ID不存在了，切换到该类型的默认源
      const defaultFurnitureSource = ModSourceManager.getAllSources().find(s => s.type === 'furniture' && s.isDefault)
      if (defaultFurnitureSource) {
        selectedSourceId.value = defaultFurnitureSource.id
        ModSourceManager.setCurrentSource(defaultFurnitureSource.id)
      } else {
        // 如果没有默认源，切换到第一个可用的家具源
        const firstSource = ModSourceManager.getEnabledSources().find(s => s.type === 'furniture')
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
.furniture-view {
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
