<template>
  <div class="skins-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space justify="space-between">
          <n-space>
            <!-- 皮肤下载视图下显示返回按钮 -->
            <n-button
              v-if="currentView === 'download'"
              @click="switchView('manage')"
            >
              <template #icon>
                <n-icon><ArrowBackIcon /></n-icon>
              </template>
              {{ t('skins.backToManage') }}
            </n-button>
            <!-- 皮肤管理视图下显示原有按钮 -->
            <template v-if="currentView === 'manage'">
              <n-text strong style="font-size: 18px;">{{ t('skins.title') }}</n-text>
            </template>
          </n-space>
          <n-space>
            <!-- 皮肤管理视图下的按钮 -->
            <template v-if="currentView === 'manage'">
              <n-button type="primary" @click="handleImportSkin">
                <template #icon>
                  <n-icon><AddIcon /></n-icon>
                </template>
                {{ t('skins.import') }}
              </n-button>
              <n-button type="info" @click="switchView('download')">
                <template #icon>
                  <n-icon><DownloadIcon /></n-icon>
                </template>
                {{ t('skins.downloadSkin') }}
              </n-button>
              <n-button @click="handleRefresh">
                <template #icon>
                  <n-icon><RefreshIcon /></n-icon>
                </template>
                {{ t('common.refresh') }}
              </n-button>
              <n-text depth="3">
                {{ t('skins.totalSkins', { count: skinStore.skins.length }) }}
              </n-text>
            </template>
          </n-space>
        </n-space>
      </n-card>

      <!-- 视图切换区域 -->
      <n-card>
        <transition name="view-fade" mode="out-in">
          <!-- 皮肤管理视图 -->
          <div v-if="currentView === 'manage'" key="manage" class="view-content">
            <!-- 皮肤列表 -->
            <n-spin :show="skinStore.loading">
              <n-grid :x-gap="16" :y-gap="16" :cols="3" responsive="screen">
                <n-grid-item v-for="skin in skinStore.skins" :key="skin.fileName">
                  <n-card hoverable class="skin-card">
                    <n-space vertical size="medium">
                      <!-- 皮肤预览区域 -->
                      <div class="skin-preview">
                        <n-spin :show="loadingImages[skin.fileName]" size="small">
                          <img
                            v-if="skinImages[skin.fileName]"
                            :src="skinImages[skin.fileName]"
                            :alt="skin.fileName"
                            class="skin-image"
                          />
                          <n-icon v-else size="64" :component="ImageIcon" />
                        </n-spin>
                      </div>

                      <!-- 皮肤信息 -->
                      <n-space vertical size="small">
                        <n-text strong>{{ skin.fileName }}</n-text>
                        <n-text depth="3">
                          {{ t('skins.fileSize') }}: {{ formatSize(skin.size) }}
                        </n-text>
                        <n-text depth="3">
                          {{ t('skins.importDate') }}: {{ skin.importDate }}
                        </n-text>
                      </n-space>

                      <!-- 操作按钮 -->
                      <n-space>
                        <n-popconfirm @positive-click="handleDeleteSkin(skin)">
                          <template #trigger>
                            <n-button type="error" size="small" block>
                              {{ t('common.delete') }}
                            </n-button>
                          </template>
                          {{ t('skins.confirmDeleteMessage') }}
                        </n-popconfirm>
                      </n-space>
                    </n-space>
                  </n-card>
                </n-grid-item>
              </n-grid>

              <n-empty
                v-if="skinStore.skins.length === 0 && !skinStore.loading"
                :description="t('skins.noSkins')"
              />
            </n-spin>
          </div>

          <!-- 皮肤下载视图 -->
          <div v-else-if="currentView === 'download'" key="download" class="view-content">
            <n-space vertical size="large">
              <!-- 下载源选择器 -->
              <n-space align="center" justify="space-between">
                <n-space>
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
                  {{ t('skins.manageSources') }}
                </n-button>
              </n-space>

              <!-- 搜索框 -->
              <n-input
                v-model:value="downloadSearchText"
                :placeholder="t('skins.searchOnlineSkins')"
                clearable
                size="large"
                @keyup.enter="handleSearchSkins"
              >
                <template #prefix>
                  <n-icon><SearchIcon /></n-icon>
                </template>
                <template #suffix>
                  <n-button type="primary" @click="handleSearchSkins" :loading="searching">
                    {{ t('common.search') }}
                  </n-button>
                </template>
              </n-input>

              <!-- 搜索结果 -->
              <n-spin :show="searching">
                <n-list v-if="searchResults.length > 0" hoverable clickable>
                  <SkinSearchResultItem
                    v-for="skin in searchResults"
                    :key="skin.id"
                    :skin="skin"
                    @click="handleShowSkinDetail"
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
                  :description="t('skins.noSearchResults')"
                />
                <n-empty
                  v-else-if="!searching && searchResults.length === 0 && !hasSearched"
                  :description="t('skins.loadingSkins')"
                />
              </n-spin>
            </n-space>
          </div>
        </transition>
      </n-card>
    </n-space>

    <!-- 皮肤详情对话框 -->
    <SkinDetailModal
      v-model:show="showSkinDetailModal"
      :skin="selectedSkin"
      :downloading="downloadingSkins"
      @download="handleDownloadSkin"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSkinStore } from '../stores/skin'
import { useMessage } from 'naive-ui'
import { Add as AddIcon, Refresh as RefreshIcon, Image as ImageIcon, ArrowBack as ArrowBackIcon, Download as DownloadIcon, CloudDownload as CloudDownloadIcon, Settings as SettingsIcon, Search as SearchIcon } from '@vicons/ionicons5'
import { formatSize } from '../utils/format'
import { ModSourceManager } from '../managers'
import { useRouter } from 'vue-router'
import type { ModSearchResult } from '../types/mod-source'
import SkinSearchResultItem from '../components/skin/SkinSearchResultItem.vue'
import SkinDetailModal from '../components/skin/SkinDetailModal.vue'

// 定义props
const props = defineProps<{
  versionIdFromRoute?: string
}>()

const { t } = useI18n()
const skinStore = useSkinStore()
const message = useMessage()
const router = useRouter()
const skinImages = ref<Record<string, string>>({})
const loadingImages = ref<Record<string, boolean>>({})

// 视图状态
const currentView = ref<'manage' | 'download'>('manage')

// 皮肤下载状态
const downloadSearchText = ref<string>('')
const searching = ref(false)
const searchResults = ref<ModSearchResult[]>([])
const downloadingSkins = ref<Set<string>>(new Set())
const hasSearched = ref(false)
const selectedSourceId = ref<string>('suancaixianyu')

// 分页状态
const currentPage = ref(1)
const pageSize = 10
const totalPages = ref(0)
const isSearchMode = ref(false)

// 皮肤详情相关
const showSkinDetailModal = ref(false)
const selectedSkin = ref<ModSearchResult | null>(null)

// 皮肤下载源选项（只显示皮肤类型的下载源）
const sourceOptions = computed(() => {
  return ModSourceManager.getEnabledSources()
    .filter(source => source.type === 'skins')
    .map(source => ({
      label: source.name,
      value: source.id
    }))
})

async function handleImportSkin() {
  try {
    // 使用 Wails 文件选择对话框
    const { SelectSkinFile } = await import('../api/skin')
    const filePath = await SelectSkinFile()

    if (filePath) {
      await skinStore.importSkin(filePath)
      message.success(t('skins.importSuccess'))
    }
  } catch (error) {
    message.error(t('skins.importFailed') + '：' + error)
  }
}

function handleRefresh() {
  skinStore.loadSkins()
}

function handleDeleteSkin(skin: any) {
  skinStore.deleteSkin(skin.fileName)
    .then(() => {
      message.success(t('skins.deleteSuccess'))
      // 清理缓存的图片
      if (skinImages.value[skin.fileName]) {
        delete skinImages.value[skin.fileName]
      }
    })
    .catch((error) => {
      message.error(t('skins.deleteFailed') + '：' + error)
    })
}

async function loadSkinImage(fileName: string) {
  // 如果已经加载过，直接返回
  if (skinImages.value[fileName]) {
    return skinImages.value[fileName]
  }

  // 标记正在加载
  loadingImages.value[fileName] = true

  try {
    const { GetSkinImage } = await import('../api/skin')
    const base64 = await GetSkinImage(fileName)
    skinImages.value[fileName] = base64
    return base64
  } catch (error) {
    console.error('Failed to load skin image:', error)
    return null
  } finally {
    loadingImages.value[fileName] = false
  }
}

// 当皮肤列表加载完成后，预加载所有皮肤图片
async function preloadSkinImages() {
  for (const skin of skinStore.skins) {
    await loadSkinImage(skin.fileName)
  }
}

// 视图切换函数
function switchView(view: 'manage' | 'download') {
  currentView.value = view

  // 切换到下载视图时自动加载第一页
  if (view === 'download' && !hasSearched.value && !isSearchMode.value) {
    loadSkinList()
  }
}

// 皮肤下载相关函数

/**
 * 加载皮肤列表
 */
async function loadSkinList() {
  searching.value = true

  try {
    const response = await ModSourceManager.getModList({
      page: currentPage.value,
      limit: pageSize
    })

    searchResults.value = response.data
    totalPages.value = response.totalPages

    if (response.data.length === 0) {
      message.info(t('skins.noSkins'))
    }
  } catch (error) {
    message.error(t('skins.searchFailed') + '：' + error)
    searchResults.value = []
    totalPages.value = 0
  } finally {
    searching.value = false
  }
}

/**
 * 搜索皮肤
 */
async function handleSearchSkins() {
  if (!downloadSearchText.value.trim()) {
    // 如果搜索框为空，切换回浏览模式
    isSearchMode.value = false
    currentPage.value = 1
    hasSearched.value = false
    loadSkinList()
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
      message.info(t('skins.noSearchResults'))
    } else {
      message.success(t('skins.searchSuccess'))
    }
  } catch (error) {
    message.error(t('skins.searchFailed') + '：' + error)
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
      message.error(t('skins.searchFailed') + '：' + error)
    } finally {
      searching.value = false
    }
  } else {
    // 浏览模式
    await loadSkinList()
  }

  // 等待DOM更新后滚动到皮肤列表顶部
  await nextTick()
  const downloadView = document.querySelector('.view-content')
  if (downloadView) {
    downloadView.scrollIntoView({ behavior: 'smooth', block: 'start' })
  } else {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

/**
 * 下载皮肤
 */
async function handleDownloadSkin(skin: ModSearchResult, versionIndex: number = 0) {
  if (!skin.versions || skin.versions.length === 0) {
    message.error('没有可下载的文件')
    return
  }

  if (versionIndex < 0 || versionIndex >= skin.versions.length) {
    message.error('版本索引超出范围')
    return
  }

  const version = skin.versions[versionIndex]
  const downloadKey = `${skin.id}-${versionIndex}`

  downloadingSkins.value.add(downloadKey)

  try {
    const { DownloadSkinFromURL } = await import('../api/skin')
    await DownloadSkinFromURL(version.downloadUrl, version.fileName)
    message.success(t('skins.downloadSuccess', { name: version.fileName }))

    // 下载成功后刷新皮肤列表
    await skinStore.loadSkins()
    await preloadSkinImages()
  } catch (error) {
    message.error(t('skins.downloadFailed') + '：' + error)
  } finally {
    downloadingSkins.value.delete(downloadKey)
  }
}

/**
 * 显示皮肤详情
 */
function handleShowSkinDetail(skin: ModSearchResult) {
  selectedSkin.value = skin
  showSkinDetailModal.value = true
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
  loadSkinList()
}

/**
 * 打开源设置页面
 */
function openSourceSettings() {
  // 打开设置页面并定位到皮肤下载源管理
  router.push({
    path: '/settings',
    query: { tab: 'skin-sources' }
  })
}

onMounted(async () => {
  try {
    // 初始化下载源（只选择皮肤类型的源）
    // 优先选择皮肤类型的默认源
    const defaultSkinSource = ModSourceManager.getAllSources().find(s => s.type === 'skins' && s.isDefault)
    if (defaultSkinSource) {
      selectedSourceId.value = defaultSkinSource.id
      ModSourceManager.setCurrentSource(defaultSkinSource.id)
    } else {
      // 如果没有默认源，选择第一个启用的皮肤源
      const firstSkinSource = ModSourceManager.getEnabledSources().find(s => s.type === 'skins')
      if (firstSkinSource) {
        selectedSourceId.value = firstSkinSource.id
        ModSourceManager.setCurrentSource(firstSkinSource.id)
      }
    }

    await skinStore.loadSkins()
    // 预加载所有皮肤图片
    await preloadSkinImages()
  } catch (error) {
    message.error(t('skins.loadFailed') + '：' + error)
  }
})

// 当页面激活时重新加载数据
onActivated(async () => {
  await ModSourceManager.reloadSources()

  // 优先选择皮肤类型的默认源
  const defaultSkinSource = ModSourceManager.getAllSources().find(s => s.type === 'skins' && s.isDefault)
  if (defaultSkinSource) {
    selectedSourceId.value = defaultSkinSource.id
    ModSourceManager.setCurrentSource(defaultSkinSource.id)
  } else {
    // 如果没有默认源，选择第一个启用的皮肤源
    const firstSkinSource = ModSourceManager.getEnabledSources().find(s => s.type === 'skins')
    if (firstSkinSource) {
      selectedSourceId.value = firstSkinSource.id
      ModSourceManager.setCurrentSource(firstSkinSource.id)
    }
  }

  try {
    await skinStore.loadSkins()
    await preloadSkinImages()
  } catch (error) {
    message.error(t('skins.loadFailed') + '：' + error)
  }
})

// 监听下载源选项变化，确保当前选中的源ID始终有效
watch(sourceOptions, (newOptions) => {
  if (newOptions.length > 0) {
    const currentIdExists = newOptions.some(opt => opt.value === selectedSourceId.value)
    if (!currentIdExists) {
      // 当前选中的源ID不存在了，切换到该类型的默认源
      const defaultSkinSource = ModSourceManager.getAllSources().find(s => s.type === 'skins' && s.isDefault)
      if (defaultSkinSource) {
        selectedSourceId.value = defaultSkinSource.id
        ModSourceManager.setCurrentSource(defaultSkinSource.id)
      } else {
        // 如果没有默认源，切换到第一个可用的皮肤源
        const firstSource = ModSourceManager.getEnabledSources().find(s => s.type === 'skins')
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
.skins-view {
  max-width: 1200px;
  margin: 0 auto;
}

.skin-card {
  height: 100%;
}

.skin-preview {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 120px;
  background-color: var(--n-color);
  border-radius: 4px;
  color: var(--n-placeholder-color);
}

.skin-image {
  max-width: 100%;
  max-height: 120px;
  object-fit: contain;
  image-rendering: pixelated; /* 保持像素风格 */
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
