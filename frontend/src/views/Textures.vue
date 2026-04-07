<template>
  <div class="textures-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space justify="space-between">
          <n-space>
            <!-- 材质下载视图下显示返回按钮 -->
            <n-button
              v-if="currentView === 'download'"
              @click="switchView('manage')"
            >
              <template #icon>
                <n-icon><ArrowBackIcon /></n-icon>
              </template>
              {{ t('textures.backToManage') }}
            </n-button>
            <!-- 材质管理视图下显示原有按钮 -->
            <template v-if="currentView === 'manage'">
              <n-text strong style="font-size: 18px;">{{ t('textures.title') }}</n-text>
              <n-select
                v-model:value="selectedVersionId"
                :options="versionOptions"
                :placeholder="t('textures.selectVersion')"
                style="width: 300px;"
                @update:value="handleVersionChange"
              />
            </template>
          </n-space>
          <n-space>
            <!-- 材质管理视图下的按钮 -->
            <template v-if="currentView === 'manage'">
              <n-button type="primary" @click="handleImportTexture">
                <template #icon>
                  <n-icon><ImportIcon /></n-icon>
                </template>
                {{ t('textures.importTexture') }}
              </n-button>
              <n-button type="info" @click="switchView('download')">
                <template #icon>
                  <n-icon><DownloadIcon /></n-icon>
                </template>
                {{ t('textures.downloadTexture') }}
              </n-button>
              <n-button @click="handleOpenFolder">
                <template #icon>
                  <n-icon><FolderOpenIcon /></n-icon>
                </template>
                {{ t('textures.openFolder') }}
              </n-button>
              <n-text depth="3">
                {{ t('textures.totalTextures') }} {{ textures.length }}
              </n-text>
            </template>
          </n-space>
        </n-space>
      </n-card>

      <!-- 视图切换区域 -->
      <n-card>
        <transition name="view-fade" mode="out-in">
          <!-- 材质管理视图 -->
          <div v-if="currentView === 'manage'" key="manage" class="view-content">
            <!-- 材质列表 -->
            <n-spin :show="loading">
              <n-list hoverable clickable>
                <TextureListItem
                  v-for="texture in textures"
                  :key="texture.id"
                  :texture="texture"
                  @rename="handleRename"
                  @delete="handleDelete"
                />
              </n-list>
              <n-empty v-if="textures.length === 0 && !loading && !folderNotFound" :description="t('textures.noTextures')">
                <template #extra>
                  <n-button type="primary" @click="handleImportTexture">
                    {{ t('textures.importFirstTexture') }}
                  </n-button>
                </template>
              </n-empty>
              <n-empty v-if="folderNotFound && !loading" :description="t('textures.folderNotFound')">
              </n-empty>
            </n-spin>
          </div>

          <!-- 材质下载视图 -->
          <div v-else-if="currentView === 'download'" key="download" class="view-content">
            <n-space vertical size="large">
              <!-- 版本选择和下载源选择器 -->
              <n-space align="center" justify="space-between">
                <n-space>
                  <!-- 游戏版本选择 -->
                  <n-select
                    v-model:value="selectedVersionId"
                    :options="installedVersionOptions"
                    :placeholder="t('textures.selectVersion')"
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
                  {{ t('textures.manageSources') }}
                </n-button>
              </n-space>

              <!-- 版本提示 -->
              <n-alert v-if="!selectedVersionId" type="warning" :title="t('textures.pleaseSelectVersionFirst')">
                {{ t('textures.selectVersionToDownload') }}
              </n-alert>

              <!-- 搜索框 -->
              <n-input
                v-model:value="downloadSearchText"
                :placeholder="t('textures.searchOnlineTextures')"
                clearable
                size="large"
                @keyup.enter="handleSearchTextures"
              >
                <template #prefix>
                  <n-icon><SearchIcon /></n-icon>
                </template>
                <template #suffix>
                  <n-button type="primary" @click="handleSearchTextures" :loading="searching">
                    {{ t('common.search') }}
                  </n-button>
                </template>
              </n-input>

              <!-- 搜索结果 -->
              <n-spin :show="searching">
                <n-list v-if="searchResults.length > 0" hoverable clickable>
                  <TextureSearchResultItem
                    v-for="texture in searchResults"
                    :key="texture.id"
                    :texture="texture"
                    @click="handleShowTextureDetail"
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
                  :description="t('textures.noSearchResults')"
                />
                <n-empty
                  v-else-if="!searching && searchResults.length === 0 && !hasSearched"
                  :description="t('textures.loadingTextures')"
                />
              </n-spin>
            </n-space>
          </div>
        </transition>
      </n-card>
    </n-space>

    <!-- 材质详情对话框 -->
    <TextureDetailModal
      v-model:show="showTextureDetailModal"
      :texture="selectedTexture"
      :downloading="downloadingTextures"
      @download="handleDownloadTexture"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated, watch, nextTick, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog, NInput } from 'naive-ui'
import { Download as ImportIcon, ArrowBack as ArrowBackIcon, Download as DownloadIcon, CloudDownload as CloudDownloadIcon, Settings as SettingsIcon, GameController as GameControllerIcon, Search as SearchIcon, FolderOpen as FolderOpenIcon } from '@vicons/ionicons5'
import { useVersionStore } from '../stores/version'
import { GetTextures, DeleteTexture, OpenTextureFolder, RenameTexture, ImportTexture, SelectTextureFile, DownloadTextureFromURL } from '../api/texture'
import { ModSourceManager } from '../managers'
import type { Texture } from '../types/texture'
import type { ModSearchResult } from '../types/mod-source'
import { useRouter } from 'vue-router'
import TextureListItem from '../components/texture/TextureListItem.vue'
import TextureSearchResultItem from '../components/texture/TextureSearchResultItem.vue'
import TextureDetailModal from '../components/texture/TextureDetailModal.vue'

const { t } = useI18n()
const versionStore = useVersionStore()
const message = useMessage()
const dialog = useDialog()
const router = useRouter()

const loading = ref(false)
const selectedVersionId = ref<string>('')

// 材质列表
const textures = ref<Texture[]>([])
const folderNotFound = ref(false)

// 视图状态
const currentView = ref<'manage' | 'download'>('manage')

// 材质下载状态
const downloadSearchText = ref<string>('')
const searching = ref(false)
const searchResults = ref<ModSearchResult[]>([])
const downloadingTextures = ref<Set<string>>(new Set())
const hasSearched = ref(false)
const selectedSourceId = ref<string>('')

// 分页状态
const currentPage = ref(1)
const pageSize = 10
const totalPages = ref(0)
const isSearchMode = ref(false)

// 材质详情相关
const showTextureDetailModal = ref(false)
const selectedTexture = ref<ModSearchResult | null>(null)

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

// 材质下载源选项（只显示材质类型的下载源）
const sourceOptions = computed(() => {
  return ModSourceManager.getEnabledSources()
    .filter(source => source.type === 'textures')
    .map(source => ({
      label: source.name,
      value: source.id
    }))
})

// 加载材质列表
async function loadTextures() {
  if (!selectedVersionId.value) {
    return
  }

  loading.value = true
  folderNotFound.value = false
  try {
    const result = await GetTextures(selectedVersionId.value)
    if (result === null) {
      // 文件夹不存在
      textures.value = []
      folderNotFound.value = true
    } else {
      textures.value = result
    }
  } catch (error) {
    message.error(t('textures.loadFailed') + '：' + error)
    textures.value = []
    folderNotFound.value = false
  } finally {
    loading.value = false
  }
}

// 版本切换
function handleVersionChange(versionId: string) {
  selectedVersionId.value = versionId
  loadTextures()
}

// 打开材质文件夹
async function handleOpenFolder() {
  if (!selectedVersionId.value) {
    message.error(t('textures.noVersionSelected'))
    return
  }

  try {
    await OpenTextureFolder(selectedVersionId.value)
  } catch (error) {
    message.error(t('textures.openFolderFailed') + '：' + error)
  }
}

// 导入材质
async function handleImportTexture() {
  if (!selectedVersionId.value) {
    message.error(t('textures.noVersionSelected'))
    return
  }

  try {
    // 选择文件
    const selectedFile = await SelectTextureFile()
    if (!selectedFile) {
      return // 用户取消
    }

    // 执行导入
    await ImportTexture(selectedVersionId.value, selectedFile)
    message.success(t('textures.importSuccess'))
    await loadTextures()
  } catch (error) {
    message.error(t('textures.importFailed') + '：' + error)
  }
}

// 重命名材质
function handleRename(texture: Texture) {
  const newName = ref(texture.name)

  dialog.create({
    title: t('textures.renameTexture'),
    content: () => {
      return h('div', [
        h('div', { style: 'margin-bottom: 8px' }, t('textures.enterNewName')),
        h(NInput, {
          value: newName.value,
          placeholder: texture.name,
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
        message.error(t('textures.nameCannotBeEmpty'))
        return false
      }

      if (newName.value.trim() === texture.name) {
        message.info(t('textures.nameUnchanged'))
        return true
      }

      try {
        await RenameTexture(selectedVersionId.value, texture.id, newName.value.trim())
        message.success(t('textures.renameSuccess'))
        await loadTextures()
        return true
      } catch (error) {
        message.error(t('textures.renameFailed') + '：' + error)
        return false
      }
    }
  })
}

// 删除材质
async function handleDelete(texture: Texture) {
  if (!selectedVersionId.value) {
    message.error(t('textures.noVersionSelected'))
    return
  }

  try {
    await DeleteTexture(selectedVersionId.value, texture.id)
    message.success(t('textures.deleteSuccess'))
    // 重新加载材质列表
    await loadTextures()
  } catch (error) {
    message.error(t('textures.deleteFailed') + '：' + error)
  }
}

// 视图切换函数
function switchView(view: 'manage' | 'download') {
  currentView.value = view

  // 切换到下载视图时自动加载第一页
  if (view === 'download' && !hasSearched.value && !isSearchMode.value) {
    loadTextureList()
  }
}

// 材质下载相关函数

/**
 * 加载材质列表
 */
async function loadTextureList() {
  searching.value = true

  try {
    const response = await ModSourceManager.getModList({
      page: currentPage.value,
      limit: pageSize
    })

    searchResults.value = response.data
    totalPages.value = response.totalPages

    if (response.data.length === 0) {
      message.info(t('textures.noTextures'))
    }
  } catch (error) {
    message.error(t('textures.searchFailed') + '：' + error)
    searchResults.value = []
    totalPages.value = 0
  } finally {
    searching.value = false
  }
}

/**
 * 搜索材质
 */
async function handleSearchTextures() {
  if (!downloadSearchText.value.trim()) {
    // 如果搜索框为空，切换回浏览模式
    isSearchMode.value = false
    currentPage.value = 1
    hasSearched.value = false
    loadTextureList()
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
      message.info(t('textures.noSearchResults'))
    } else {
      message.success(t('textures.searchSuccess'))
    }
  } catch (error) {
    message.error(t('textures.searchFailed') + '：' + error)
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
      message.error(t('textures.searchFailed') + '：' + error)
    } finally {
      searching.value = false
    }
  } else {
    // 浏览模式
    await loadTextureList()
  }

  // 等待DOM更新后滚动到材质列表顶部
  await nextTick()
  const downloadView = document.querySelector('.view-content')
  if (downloadView) {
    downloadView.scrollIntoView({ behavior: 'smooth', block: 'start' })
  } else {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

/**
 * 下载材质
 */
async function handleDownloadTexture(texture: ModSearchResult, versionIndexOrString: string | number = 0) {
  if (!selectedVersionId.value) {
    message.warning(t('textures.pleaseSelectVersionFirst'))
    return
  }

  // 确定要下载的版本
  let versionIndex = 0
  if (typeof versionIndexOrString === 'string') {
    versionIndex = texture.versions.findIndex(v => v.version === versionIndexOrString)
  } else {
    versionIndex = versionIndexOrString
  }

  if (versionIndex < 0 || versionIndex >= texture.versions.length) {
    message.error(t('textures.versionNotFound'))
    return
  }

  const version = texture.versions[versionIndex]
  const downloadKey = `${texture.id}-${versionIndex}`

  downloadingTextures.value.add(downloadKey)

  try {
    await DownloadTextureFromURL(version.downloadUrl, selectedVersionId.value, version.fileName)
    message.success(t('textures.downloadSuccess', { name: texture.title }))

    // 下载成功后刷新材质列表
    await loadTextures()
  } catch (error) {
    message.error(t('textures.downloadFailed') + '：' + error)
  } finally {
    downloadingTextures.value.delete(downloadKey)
  }
}

/**
 * 显示材质详情
 */
function handleShowTextureDetail(texture: ModSearchResult) {
  selectedTexture.value = texture
  showTextureDetailModal.value = true
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
  loadTextureList()
}

/**
 * 打开源设置页面
 */
function openSourceSettings() {
  // 打开设置页面并定位到材质下载源管理
  router.push({
    path: '/settings',
    query: { tab: 'texture-sources' }
  })
}

onMounted(async () => {
  loading.value = true
  try {
    // 初始化下载源（只选择材质类型的源）
    // 优先选择材质类型的默认源
    const defaultTextureSource = ModSourceManager.getAllSources().find(s => s.type === 'textures' && s.isDefault)
    if (defaultTextureSource) {
      selectedSourceId.value = defaultTextureSource.id
      ModSourceManager.setCurrentSource(defaultTextureSource.id)
    } else {
      // 如果没有默认源，选择第一个启用的材质源
      const firstTextureSource = ModSourceManager.getEnabledSources().find(s => s.type === 'textures')
      if (firstTextureSource) {
        selectedSourceId.value = firstTextureSource.id
        ModSourceManager.setCurrentSource(firstTextureSource.id)
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

    // 加载材质列表
    if (selectedVersionId.value) {
      await loadTextures()
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

  // 优先选择材质类型的默认源
  const defaultTextureSource = ModSourceManager.getAllSources().find(s => s.type === 'textures' && s.isDefault)
  if (defaultTextureSource) {
    selectedSourceId.value = defaultTextureSource.id
    ModSourceManager.setCurrentSource(defaultTextureSource.id)
  } else {
    // 如果没有默认源，选择第一个启用的材质源
    const firstTextureSource = ModSourceManager.getEnabledSources().find(s => s.type === 'textures')
    if (firstTextureSource) {
      selectedSourceId.value = firstTextureSource.id
      ModSourceManager.setCurrentSource(firstTextureSource.id)
    }
  }
})

// 监听下载源选项变化，确保当前选中的源ID始终有效
watch(sourceOptions, (newOptions) => {
  if (newOptions.length > 0) {
    const currentIdExists = newOptions.some(opt => opt.value === selectedSourceId.value)
    if (!currentIdExists) {
      // 当前选中的源ID不存在了，切换到该类型的默认源
      const defaultTextureSource = ModSourceManager.getAllSources().find(s => s.type === 'textures' && s.isDefault)
      if (defaultTextureSource) {
        selectedSourceId.value = defaultTextureSource.id
        ModSourceManager.setCurrentSource(defaultTextureSource.id)
      } else {
        // 如果没有默认源，切换到第一个可用的材质源
        const firstSource = ModSourceManager.getEnabledSources().find(s => s.type === 'textures')
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
.textures-view {
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
