<template>
  <div class="modpacks-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space vertical size="medium">
          <n-space align="center" justify="space-between">
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
            <n-button text @click="openSourceSettings">
              <template #icon>
                <n-icon><SettingsIcon /></n-icon>
              </template>
              {{ t('mods.manageSources') }}
            </n-button>
          </n-space>

          <!-- 搜索框 -->
          <n-input
            v-model:value="searchText"
            :placeholder="t('modpacks.searchPlaceholder')"
            clearable
            size="large"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <n-icon><SearchIcon /></n-icon>
            </template>
            <template #suffix>
              <n-button type="primary" @click="handleSearch" :loading="searching">
                {{ t('common.search') }}
              </n-button>
            </template>
          </n-input>

          <!-- 列表 -->
          <n-spin :show="searching">
            <n-list v-if="results.length > 0" hoverable clickable>
              <ModSearchResultItem
                v-for="modpack in results"
                :key="modpack.id"
                :mod="modpack"
                @click="handleShowDetail"
              />
            </n-list>

            <!-- 分页组件 -->
            <div v-if="results.length > 0 && totalPages > 1" style="margin-top: 16px; display: flex; justify-content: center;">
              <n-pagination
                v-model:page="currentPage"
                :page-count="totalPages"
                @update:page="handlePageChange"
              />
            </div>

            <n-empty
              v-else-if="!searching && results.length === 0 && hasSearched"
              :description="t('modpacks.noResults')"
            />
            <n-empty
              v-else-if="!searching && results.length === 0 && !hasSearched"
              :description="t('modpacks.loading')"
            />
          </n-spin>
        </n-space>
      </n-card>
    </n-space>

    <!-- 整合包详情对话框（社区帖子：版本列表 + 下载按钮） -->
    <ModDetailModal
      v-model:show="showDetailModal"
      :mod="selectedModpack"
      :downloading="downloadingModpacks"
      @download="handleDownloadModpack"
    />

    <!-- 整合包信息对话框（解析结果预览 + 安装） -->
    <ModpackInfoDialog
      v-model:show="showModpackDialog"
      :modpack-data="parsedModpackInfo"
      @install="handleModpackInstall"
    />

    <!-- 整合包安装进度对话框 -->
    <ModpackInstallDialog
      v-model:show="showInstallDialog"
      :modpack-info="parsedModpackInfo"
      @completed="handleInstallCompleted"
      @error="handleInstallError"
    />

    <!-- 目标版本选择（已安装版本 + 下载新版本） -->
    <ModpackVersionSelectDialog
      v-model:show="showVersionSelectDialog"
      :preferred-game-versions="preferredGameVersions"
      :modpack-name="parsedModpackInfo?.name || ''"
      @confirm="handleVersionConfirm"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { h } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useVersionStore } from '../stores/version'
import { useMessage, useDialog } from 'naive-ui'
import { Search as SearchIcon, CloudDownload as CloudDownloadIcon, Settings as SettingsIcon } from '@vicons/ionicons5'
import { ModSourceManager } from '../managers'
import type { ModSearchResult } from '../types/mod-source'
import ModSearchResultItem from '../components/mod/ModSearchResultItem.vue'
import ModDetailModal from '../components/mod/ModDetailModal.vue'
import ModpackInfoDialog from '../components/ModpackInfoDialog.vue'
import ModpackInstallDialog from '../components/ModpackInstallDialog.vue'
import ModpackVersionSelectDialog from '../components/ModpackVersionSelectDialog.vue'
import { ParseModpack, InstallModpackWithProgress, InstallScbbsModpack, DownloadModpackFromUrl, DeleteModpackTempFile } from '../api/version'
import { resolveGameVersionNames } from '../api/communityVersions'

const { t } = useI18n()
const versionStore = useVersionStore()
const message = useMessage()
const dialog = useDialog()
const router = useRouter()

// 列表状态
const searchText = ref('')
const searching = ref(false)
const hasSearched = ref(false)
const results = ref<ModSearchResult[]>([])
const selectedSourceId = ref('')
const currentPage = ref(1)
const pageSize = 10
const totalPages = ref(0)
const isSearchMode = ref(false)

// 详情弹窗
const showDetailModal = ref(false)
const selectedModpack = ref<ModSearchResult | null>(null)
const downloadingModpacks = ref<Set<string>>(new Set())

// 下载 → 解析 → 安装流程
const showModpackDialog = ref(false)
const showInstallDialog = ref(false)
const parsedModpackInfo = ref<any>(null)
// 下载产生的临时文件（新下载开始 / 安装会话结束 / 组件卸载时清理）
const modpackTempPath = ref('')

// 社区帖子声明的适配游戏版本（gameVersionIds → /api/version/list 版本名），
// 安装时传给版本选择弹窗做默认预选；解析失败为空数组，交由用户自选
const preferredGameVersions = ref<string[]>([])

// 整合包类型的下载源
const sourceOptions = computed(() =>
  ModSourceManager.getEnabledSources()
    .filter(s => s.type === 'modpacks')
    .map(s => ({ label: s.name, value: s.id }))
)

async function loadList() {
  searching.value = true
  try {
    const response = await ModSourceManager.getModList({
      page: currentPage.value,
      limit: pageSize
    })
    results.value = response.data
    totalPages.value = response.totalPages
  } catch (error) {
    message.error(t('mods.searchFailed') + '：' + error)
    results.value = []
    totalPages.value = 0
  } finally {
    searching.value = false
  }
}

async function handleSearch() {
  if (!searchText.value.trim()) {
    isSearchMode.value = false
    currentPage.value = 1
    hasSearched.value = false
    loadList()
    return
  }

  searching.value = true
  hasSearched.value = true
  isSearchMode.value = true
  currentPage.value = 1

  try {
    const response = await ModSourceManager.searchMods(searchText.value, {
      page: 1,
      limit: pageSize
    })
    results.value = response.data
    totalPages.value = response.totalPages
    if (response.data.length === 0) {
      message.info(t('modpacks.noResults'))
    }
  } catch (error) {
    message.error(t('mods.searchFailed') + '：' + error)
    results.value = []
    totalPages.value = 0
  } finally {
    searching.value = false
  }
}

async function handlePageChange(page: number) {
  currentPage.value = page
  searching.value = true
  try {
    const response = isSearchMode.value
      ? await ModSourceManager.searchMods(searchText.value, { page, limit: pageSize })
      : await ModSourceManager.getModList({ page, limit: pageSize })
    results.value = response.data
    totalPages.value = response.totalPages
  } catch (error) {
    message.error(t('mods.searchFailed') + '：' + error)
  } finally {
    searching.value = false
  }
}

function handleSourceChange(sourceId: string) {
  ModSourceManager.setCurrentSource(sourceId)
  results.value = []
  currentPage.value = 1
  hasSearched.value = false
  isSearchMode.value = false
  searchText.value = ''
  totalPages.value = 0
  loadList()
}

function openSourceSettings() {
  router.push({
    path: '/settings',
    query: { tab: 'mod-sources' }
  })
}

function handleShowDetail(modpack: ModSearchResult) {
  selectedModpack.value = modpack
  showDetailModal.value = true
}

// 清理上一次下载的临时文件
async function cleanupTempFile() {
  const path = modpackTempPath.value
  modpackTempPath.value = ''
  if (!path) return
  try {
    await DeleteModpackTempFile(path)
  } catch (error) {
    console.warn('清理整合包临时文件失败:', error)
  }
}

// 临时文件清理时机说明：
// 不在信息弹窗关闭时清理——scbbs/手动模式点「安装」后信息弹窗会立即自关（组件自身
// emit update:show false），而安装要等用户在版本选择对话框里确认才开始；若在此间隙
// 清理会删掉即将安装的文件（曾导致「文件不存在」报错）。
// 进度弹窗关闭即安装会话结束（完成/出错/取消之后）→ 此时清理是安全的。
watch(showInstallDialog, (val) => {
  if (!val) {
    cleanupTempFile()
  }
})

/**
 * 下载整合包并进入「解析 → 信息弹窗」流程
 * （对齐手机版：下载到临时目录 → 自动解析 → 弹信息/安装弹窗）
 */
async function handleDownloadModpack(modpack: ModSearchResult, versionIndex: number = 0) {
  const version = modpack.versions[versionIndex]
  if (!version?.downloadUrl) {
    message.error(t('modpacks.downloadFailed'))
    return
  }

  const downloadKey = `${modpack.id}-${versionIndex}`
  downloadingModpacks.value.add(downloadKey)

  // 新一次下载开始，上一个临时文件不再需要
  await cleanupTempFile()
  // 并行解析帖子声明的适配游戏版本（目录有会话缓存，失败则交由用户自选）
  preferredGameVersions.value = []
  resolveGameVersionNames(version.gameVersionIds)
    .then(names => { preferredGameVersions.value = names })
    .catch(() => { preferredGameVersions.value = [] })

  try {
    const tempPath = await DownloadModpackFromUrl(version.downloadUrl, version.fileName || `${modpack.title}.zip`)

    const loadingMsg = message.loading(t('modpacks.parsing'), { duration: 0 })
    try {
      const info = await ParseModpack(tempPath)
      loadingMsg.destroy()

      parsedModpackInfo.value = info
      modpackTempPath.value = tempPath
      showModpackDialog.value = true
    } catch (error: any) {
      loadingMsg.destroy()
      await cleanupTempFile()
      showParseError(error)
    }
  } catch (error) {
    message.error(t('modpacks.downloadFailed') + '：' + error)
  } finally {
    downloadingModpacks.value.delete(downloadKey)
  }
}

// 解析失败弹窗（与已安装页一致：错误详情 monospace 展示）
function showParseError(error: any) {
  const errorMsg = error?.message || error?.toString() || '未知错误'
  console.error('解析整合包失败:', error)
  dialog.create({
    title: t('installed.modpackParseFailed'),
    content: () => {
      return h('div', [
        h('p', { style: 'margin-bottom: 12px;' }, t('modpacks.parseFailedDesc')),
        h('div', {
          style: 'background: var(--n-code-color); padding: 12px; border-radius: 4px; font-family: monospace; font-size: 12px; white-space: pre-wrap; word-break: break-all;'
        }, errorMsg)
      ])
    },
    positiveText: t('common.confirm')
  })
}

async function handleModpackInstall() {
  if (!parsedModpackInfo.value) {
    return
  }

  try {
    // scbbs（ScbbsPlus）格式不带游戏本体，必须安装到已安装版本，与手动模式一致走版本选择
    const isScbbs = parsedModpackInfo.value?.format === 'scbbs'
    const isManualMode = isScbbs || parsedModpackInfo.value.survivalcraft?.version?.manual === true

    if (isManualMode) {
      // 手动/scbbs 模式：弹版本选择（可选已安装版本或下载新版本）
      showVersionSelectDialog.value = true
    } else {
      await startModpackInstall()
    }
  } catch (error) {
    message.error(t('installed.modpackInstallFailed') + '：' + error)
    showInstallDialog.value = false
  }
}

// 版本选择对话框（已安装版本 + 下载新版本，scbbs / 手动模式共用）
const showVersionSelectDialog = ref(false)

function handleVersionConfirm(versionId: string) {
  if (parsedModpackInfo.value?.format === 'scbbs') {
    startScbbsInstall(versionId)
  } else {
    startModpackInstall(versionId)
  }
}

// 开始安装整合包（v0.1 自动/手动模式）
async function startModpackInstall(baseVersionID?: string) {
  if (!parsedModpackInfo.value) {
    return
  }

  try {
    showModpackDialog.value = false
    showInstallDialog.value = true

    await InstallModpackWithProgress(parsedModpackInfo.value.filePath, baseVersionID)
  } catch (error) {
    message.error(t('installed.modpackInstallFailed') + '：' + error)
    showInstallDialog.value = false
  }
}

// 开始安装 scbbs（ScbbsPlus）格式整合包（装到已安装版本，不创建新版本）
async function startScbbsInstall(targetVersionID: string) {
  if (!parsedModpackInfo.value) {
    return
  }

  try {
    showModpackDialog.value = false
    showInstallDialog.value = true

    await InstallScbbsModpack(parsedModpackInfo.value.filePath, targetVersionID)
  } catch (error) {
    message.error(t('installed.modpackInstallFailed') + '：' + error)
    showInstallDialog.value = false
  }
}

// 安装完成回调
async function handleInstallCompleted(_versionId: string) {
  message.success(t('installed.modpackInstallSuccess'))
  await cleanupTempFile()

  // 刷新版本列表（v0.1 会创建新版本）
  await versionStore.getVersions()
  await versionStore.getPrimaryVersion()
}

// 安装错误回调
function handleInstallError(_error: string) {
  cleanupTempFile()
}

// 组件卸载时清理未处理的临时文件
onUnmounted(() => {
  cleanupTempFile()
})

onMounted(async () => {
  // 初始化下载源（只选择整合包类型的源）
  const currentSource = ModSourceManager.getCurrentSource()
  if (currentSource && currentSource.type === 'modpacks') {
    selectedSourceId.value = currentSource.id
  } else {
    const firstSource = ModSourceManager.getEnabledSources().find(s => s.type === 'modpacks')
    if (firstSource) {
      selectedSourceId.value = firstSource.id
      ModSourceManager.setCurrentSource(firstSource.id)
    }
  }

  await loadList()
})
</script>

<style scoped>
.modpacks-view {
  width: 100%;
}
</style>
