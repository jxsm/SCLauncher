<template>
  <div class="settings-view">
    <!-- 常用/高级两级分区：技术性配置收进「高级」，降低新手认知负担 -->
    <n-tabs v-model:value="settingsTab" type="line" animated>
      <n-tab-pane name="basic" :tab="t('settings.tabBasic')">
        <n-space vertical size="large">
          <!-- 主题设置 -->
          <div id="settings-card-theme">
            <ThemeSettings />
          </div>

          <!-- 语言设置 -->
          <LanguageSettings
            :language="language"
            @update:language="handleSaveLanguage"
          />

          <!-- 背景设置 -->
          <div id="settings-card-background">
            <BackgroundSettings
              :has-background="!!config?.backgroundImage"
              :preview="backgroundImagePreview"
              @select="handleSelectBackground"
              @clear="handleClearBackground"
            />
          </div>

          <!-- 关于 -->
          <div class="about-section">
            <n-button text @click="showAboutDialog = true">
              <template #icon>
                <n-icon><InformationIcon /></n-icon>
              </template>
              {{ t('common.about') }}
            </n-button>
          </div>
        </n-space>
      </n-tab-pane>

      <n-tab-pane name="advanced" :tab="t('settings.tabAdvanced')">
        <n-space vertical size="large">
          <!-- 路径信息 -->
          <div id="settings-card-path">
            <PathSettings :config="config" />
          </div>

          <!-- 清单设置 -->
          <div id="settings-card-manifest">
            <ManifestSettings
              :sources="manifestSources"
              :current-source-id="currentManifestSourceId"
              :speed-info="defaultManifestInfo"
              :testing="speedTesting"
              @add-source="showAddManifestSourceDialog = true"
              @set-current="handleSetCurrentManifestSource"
              @delete="handleDeleteManifestSource"
              @retest="handleRetestDefaultManifest"
            />
          </div>

          <!-- 下载源设置 -->
          <div id="settings-card-sources">
            <SourceSettings
              :sources="modSources"
              :initial-type="initialSourceType"
              @toggle-source="handleToggleSource"
              @set-default-source="handleSetDefaultSource"
              @delete-source="handleDeleteSource"
              @add-source="showAddSourceDialog = true"
            />
          </div>

          <!-- .NET 运行时自动检查 -->
          <div id="settings-card-runtime">
            <RuntimeSettings />
          </div>
        </n-space>
      </n-tab-pane>
    </n-tabs>

    <!-- 关于对话框 -->
    <AboutDialog
      v-model:visible="showAboutDialog"
      :app-info="appInfo"
      @open-github="openGitHub"
      @check-update="handleCheckUpdate"
    />

    <!-- 添加下载源对话框 -->
    <AddSourceDialog
      v-model:visible="showAddSourceDialog"
      @confirm="handleAddSource"
    />

    <!-- 添加清单源对话框 -->
    <AddManifestSourceDialog
      v-model:visible="showAddManifestSourceDialog"
      @confirm="handleAddManifestSource"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, h, computed, inject } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog, NAlert } from 'naive-ui'
import { ModSourceManager } from '../managers'
import { ManifestSourceManager } from '../managers/ManifestSourceManager'
import type { ModSource } from '../types/mod-source'
import type { ManifestSource } from '../types/manifest-source'
import { InformationCircleOutline as InformationIcon } from '@vicons/ionicons5'
import { GetConfig, SetLanguage, GetAppInfo, CheckUpdateForce, SelectBackgroundFile, SetBackground, ClearBackground, TestDefaultManifestSpeed } from '../api/config'
import { useVersionStore } from '../stores/version'
import type { AppConfig, DefaultManifestInfo, ManifestSpeedTestResponse } from '../types/config'
import LanguageSettings from '../components/settings/LanguageSettings.vue'
import ThemeSettings from '../components/settings/ThemeSettings.vue'
import PathSettings from '../components/settings/PathSettings.vue'
import ManifestSettings from '../components/settings/ManifestSettings.vue'
import SourceSettings from '../components/settings/SourceSettings.vue'
import BackgroundSettings from '../components/settings/BackgroundSettings.vue'
import RuntimeSettings from '../components/settings/RuntimeSettings.vue'
import AboutDialog from '../components/settings/AboutDialog.vue'
import AddSourceDialog from '../components/settings/AddSourceDialog.vue'
import AddManifestSourceDialog from '../components/settings/AddManifestSourceDialog.vue'

const { t, locale } = useI18n()

const route = useRoute()
const message = useMessage()
const dialog = useDialog()
const versionStore = useVersionStore()

// App.vue 提供：设置/清除背景后立即重载背景层与特效
const reloadBackground = inject<() => Promise<void>>('reloadBackground', async () => {})

// 常用/高级分区 + 深链定位（资源页「管理下载源」通过 /settings?tab=xxx 直达）
const settingsTab = ref<'basic' | 'advanced'>('basic')
const initialSourceType = ref('mods')

const DEEP_LINK_TABS: Record<string, { pane: 'basic' | 'advanced'; card?: string; sourceType?: string }> = {
  'mod-sources': { pane: 'advanced', card: 'settings-card-sources', sourceType: 'mods' },
  'online-mod-sources': { pane: 'advanced', card: 'settings-card-sources', sourceType: 'online-mods' },
  'savegame-sources': { pane: 'advanced', card: 'settings-card-sources', sourceType: 'savegames' },
  'furniture-sources': { pane: 'advanced', card: 'settings-card-sources', sourceType: 'furniture' },
  'texture-sources': { pane: 'advanced', card: 'settings-card-sources', sourceType: 'textures' },
  'skin-sources': { pane: 'advanced', card: 'settings-card-sources', sourceType: 'skins' },
  manifest: { pane: 'advanced', card: 'settings-card-manifest' },
  path: { pane: 'advanced', card: 'settings-card-path' },
  runtime: { pane: 'advanced', card: 'settings-card-runtime' },
  theme: { pane: 'basic', card: 'settings-card-theme' },
  background: { pane: 'basic', card: 'settings-card-background' }
}

const deepLink = DEEP_LINK_TABS[String(route.query.tab || '')]
if (deepLink) {
  settingsTab.value = deepLink.pane
  if (deepLink.sourceType) {
    initialSourceType.value = deepLink.sourceType
  }
}

const config = ref<AppConfig | null>(null)
const manifestUrl = ref('')
const language = ref('zh-CN')
const showAboutDialog = ref(false)
const backgroundImagePreview = ref('')
const appInfo = ref<{ version: string; repoOwner: string; repoName: string }>({
  version: '0.0.1',
  repoOwner: 'jxsm',
  repoName: 'SCLauncher'
})

// 模组下载源相关
const modSources = ref<ModSource[]>([])
const showAddSourceDialog = ref(false)

// 清单源相关
const manifestSources = ref<ManifestSource[]>([])
const currentManifestSourceId = ref('')
const showAddManifestSourceDialog = ref(false)

// 默认清单源线路测速相关
const defaultManifestInfo = ref<DefaultManifestInfo | null>(null)
const speedTesting = ref(false)

// 加载模组下载源列表
async function loadModSources() {
  // 重新加载源列表以获取最新状态
  await ModSourceManager.reloadSources()
  modSources.value = ModSourceManager.getAllSources()
  console.log('设置页面加载下载源:', modSources.value)
  console.log('各类型源统计:', {
    mods: modSources.value.filter(s => s.type === 'mods').length,
    'online-mods': modSources.value.filter(s => s.type === 'online-mods').length,
    savegames: modSources.value.filter(s => s.type === 'savegames').length,
    furniture: modSources.value.filter(s => s.type === 'furniture').length,
    textures: modSources.value.filter(s => s.type === 'textures').length,
    skins: modSources.value.filter(s => s.type === 'skins').length
  })
}

// 加载清单源列表
async function loadManifestSources() {
  if (config.value?.manifestSources) {
    manifestSources.value = config.value.manifestSources
    currentManifestSourceId.value = config.value.currentManifestSourceId || 'default'
    // 同步管理器内存列表，避免其使用过期的源列表
    await ManifestSourceManager.loadSources(config.value.manifestSources)
  }
  defaultManifestInfo.value = config.value?.defaultManifest || null
}

// 重新测速默认清单线路（哪个快用哪个）
async function handleRetestDefaultManifest() {
  if (speedTesting.value) return
  speedTesting.value = true
  try {
    const res = await TestDefaultManifestSpeed() as ManifestSpeedTestResponse

    // 更新测速信息展示
    const mainResult = res.results?.find(r => r.line === 'main')
    const mirrorResult = res.results?.find(r => r.line === 'mirror')
    const prevInfo = defaultManifestInfo.value
    defaultManifestInfo.value = {
      mainUrl: prevInfo?.mainUrl || mainResult?.url || '',
      mirrorUrl: prevInfo?.mirrorUrl || mirrorResult?.url || '',
      selectedUrl: res.selectedUrl,
      mainLatencyMs: mainResult?.latencyMs ?? -1,
      mirrorLatencyMs: mirrorResult?.latencyMs ?? -1,
      lastTest: res.lastTest
    }

    // 同步默认源条目显示的 URL
    const defaultSource = manifestSources.value.find(s => s.id === 'default')
    if (defaultSource) {
      defaultSource.url = res.selectedUrl
    }

    // 当前使用的是默认源时，清除清单缓存以便下次用新线路获取
    if (currentManifestSourceId.value === 'default') {
      versionStore.clearManifestCache()
    }

    if (res.results?.some(r => r.ok)) {
      message.success(t('settings.manifestSpeedTestDone'))
    } else {
      message.warning(t('settings.manifestSpeedTestAllFailed'))
    }
  } catch (error) {
    message.error(t('settings.operationFailed') + '：' + error)
  } finally {
    speedTesting.value = false
  }
}

// 切换下载源启用状态
async function handleToggleSource(sourceId: string, enabled: boolean) {
  try {
    await ModSourceManager.toggleSource(sourceId, enabled)
    loadModSources()
    message.success(enabled ? t('mods.sourceEnabled') : t('mods.sourceDisabled'))
  } catch (error) {
    message.error(t('mods.operationFailed') + '：' + error)
  }
}

// 删除下载源
function handleDeleteSource(source: ModSource) {
  dialog.warning({
    title: t('mods.deleteSourceTitle'),
    content: t('mods.deleteSourceConfirm', { name: source.name }),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await ModSourceManager.removeSource(source.id)
        loadModSources()
        message.success(t('mods.sourceDeleted'))
      } catch (error) {
        message.error(t('mods.operationFailed') + '：' + error)
      }
    }
  })
}

// 设置默认下载源（支持切换）
async function handleSetDefaultSource(source: ModSource) {
  try {
    const sources = ModSourceManager.getAllSources()

    // 如果该源已经是默认源，取消其默认状态
    if (source.isDefault) {
      source.isDefault = false
      await ModSourceManager.saveSources()
      loadModSources()
      message.info(t('mods.defaultSourceCancelled') + ': ' + source.name)
      return
    }

    // 取消当前同类型的默认源
    sources.forEach(s => {
      if (s.type === source.type && s.isDefault) {
        s.isDefault = false
      }
    })

    // 设置新的默认源
    const targetSource = sources.find(s => s.id === source.id)
    if (targetSource) {
      targetSource.isDefault = true
    }

    // 保存更改
    await ModSourceManager.saveSources()
    loadModSources()
    message.success(t('mods.defaultSourceSet') + ': ' + source.name)
  } catch (error) {
    message.error(t('mods.operationFailed') + '：' + error)
  }
}

// 添加下载源
async function handleAddSource(source: { name: string; description: string; apiUrl: string; packageLookupUrl?: string; type: 'mods' | 'savegames' | 'furniture' | 'textures' | 'skins' }) {
  if (!source.name || !source.description || !source.apiUrl) {
    message.warning(t('mods.pleaseFillAllFields'))
    return
  }

  try {
    // 生成唯一ID
    const id = 'custom-' + Date.now()

    await ModSourceManager.addSource({
      id,
      type: source.type,
      name: source.name,
      description: source.description,
      enabled: true,
      api: {
        baseUrl: source.apiUrl,
        searchPath: '/api/mods/search',
        responseMapping: {
          // 这里需要用户提供正确的映射配置
          // 暂时使用默认配置
          results: '$.data',
          id: '$.id',
          title: '$.name',
          description: '$.description',
          author: '$.author',
          versions: '$.versions',
          version: '$.version',
          downloadUrl: '$.downloadUrl',
          fileName: '$.fileName',
          fileSize: '$.fileSize'
        } as any
      },
      // 按包名查询接口（仅模组类型源有意义，用于依赖解析）
      ...(source.type === 'mods' && source.packageLookupUrl?.trim()
        ? { packageLookup: { url: source.packageLookupUrl.trim() } }
        : {})
    })

    loadModSources()
    showAddSourceDialog.value = false
    message.success(t('mods.sourceAdded'))
  } catch (error) {
    message.error(t('mods.operationFailed') + '：' + error)
  }
}

// 设置当前清单源
async function handleSetCurrentManifestSource(source: ManifestSource) {
  try {
    await ManifestSourceManager.setCurrentSource(source.id)
    currentManifestSourceId.value = source.id
    manifestUrl.value = source.url

    // 清除清单缓存，以便下次进入版本页面时重新获取
    versionStore.clearManifestCache()

    message.success(t('settings.currentManifestSourceSet') + ': ' + source.name)
  } catch (error) {
    message.error(t('settings.operationFailed') + '：' + error)
  }
}

// 删除清单源
function handleDeleteManifestSource(source: ManifestSource) {
  dialog.warning({
    title: t('settings.deleteManifestSourceTitle'),
    content: t('settings.deleteManifestSourceConfirm', { name: source.name }),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
      onPositiveClick: async () => {
      try {
        await ManifestSourceManager.removeSource(source.id)

        // 重新加载配置
        config.value = await GetConfig()
        await loadManifestSources()

        // 删除的是当前选中的源时，后端会自动切回默认源，这里同步显示状态
        if (currentManifestSourceId.value === source.id) {
          currentManifestSourceId.value = config.value?.currentManifestSourceId || 'default'
          const currentSource = manifestSources.value.find(s => s.id === currentManifestSourceId.value)
          if (currentSource) {
            manifestUrl.value = currentSource.url
          }
        }

        message.success(t('settings.manifestSourceDeleted'))
      } catch (error) {
        message.error(t('settings.operationFailed') + '：' + error)
      }
    }
  })
}

// 添加清单源
async function handleAddManifestSource(source: { name: string; url: string }) {
  if (!source.name || !source.url) {
    message.warning(t('settings.pleaseFillAllFields'))
    return
  }

  try {
    // 生成唯一ID
    const id = 'manifest-custom-' + Date.now()

    await ManifestSourceManager.addSource({
      id,
      name: source.name,
      url: source.url
    })

    // 重新加载配置
    config.value = await GetConfig()
    await loadManifestSources()

    showAddManifestSourceDialog.value = false
    message.success(t('settings.manifestSourceAdded'))
  } catch (error) {
    message.error(t('settings.operationFailed') + '：' + error)
  }
}

// 加载背景图片预览
async function loadBackgroundPreview() {
  if (!config.value?.backgroundImage) {
    backgroundImagePreview.value = ''
    return
  }

  try {
    const { GetBackgroundImageBase64 } = await import('../api/config')
    const base64 = await GetBackgroundImageBase64()
    backgroundImagePreview.value = base64
  } catch (error) {
    console.error('Failed to load background image:', error)
    backgroundImagePreview.value = ''
  }
}

async function handleSaveLanguage(newLanguage: string) {
  try {
    // 更新语言值
    language.value = newLanguage
    await SetLanguage(newLanguage)
    // 立即切换应用语言
    locale.value = newLanguage
    message.success(t('settings.languageSaved'))
  } catch (error) {
    message.error(t('settings.saveFailed') + '：' + error)
  }
}

async function handleSelectBackground() {
  try {
    const filename = await SelectBackgroundFile()
    if (!filename) {
      return
    }

    message.info(t('settings.setBackground'))
    await SetBackground(filename)

    // 重新加载配置
    config.value = await GetConfig()
    // 加载背景预览
    await loadBackgroundPreview()
    // 立即应用背景与特效（无需离开设置页）
    await reloadBackground()
    message.success(t('settings.backgroundSetSuccess'))
  } catch (error) {
    message.error(t('settings.backgroundSetFailed') + '：' + error)
  }
}

async function handleClearBackground() {
  dialog.warning({
    title: t('settings.confirmClear'),
    content: t('settings.confirmClearMessage'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await ClearBackground()
        // 重新加载配置
        config.value = await GetConfig()
        // 清除预览
        backgroundImagePreview.value = ''
        // 立即移除背景层与半透明表面
        await reloadBackground()
        message.success(t('settings.backgroundCleared'))
      } catch (error) {
        message.error(t('settings.saveFailed') + '：' + error)
      }
    }
  })
}

function openGitHub() {
  window.open('https://github.com/jxsm/SCLauncher', '_blank')
}

async function handleCheckUpdate() {
  try {
    // 手动检查更新时使用强制检查，忽略30天不再提醒的限制
    const updateInfo = await CheckUpdateForce()
    console.log('[Update Check] Update info:', updateInfo)

    if (updateInfo.hasUpdate) {
      // 有新版本，显示更新对话框
      dialog.create({
        title: t('settings.updateAvailable'),
        content: () => {
          return h('div', [
            h('p', { style: 'margin-bottom: 12px;' }, `${t('settings.currentVersion')}: v${updateInfo.currentVersion}`),
            h('p', { style: 'margin-bottom: 12px; font-weight: bold; color: var(--n-primary-color);' }, `${t('settings.latestVersion')}: v${updateInfo.latestVersion}`),
            h('p', { style: 'margin-bottom: 12px;' }, `${t('settings.releaseDate')}: ${new Date(updateInfo.publishedAt).toLocaleString()}`),
            h(NAlert, {
              type: 'info',
              title: t('settings.updateContent')
            }, {
              default: () => h('pre', {
                style: 'max-height: 200px; overflow-y: auto; background: var(--n-code-color); color: var(--n-text-color); padding: 12px; border-radius: 8px; font-size: 12px; white-space: pre-wrap; border: 1px solid var(--n-border-color); font-family: "SF Mono", SFMono-Regular, Menlo, Monaco, Consolas, monospace;'
              }, updateInfo.body || t('settings.noUpdateContent'))
            })
          ])
        },
        positiveText: t('settings.goToDownload'),
        negativeText: t('common.close'),
        onPositiveClick: () => {
          window.open(updateInfo.url, '_blank')
        }
      })
    } else {
      message.success(t('settings.noUpdate'))
    }
  } catch (error) {
    message.error(t('settings.checkUpdateFailed') + '：' + error)
  }
}

onMounted(async () => {
  try {
    // 加载模组下载源列表
    await loadModSources()

    // 获取应用信息
    const info = await GetAppInfo()
    if (info) {
      appInfo.value = info
    }

    // 获取配置
    config.value = await GetConfig()
    if (config.value) {
      manifestUrl.value = config.value.manifestUrl
      language.value = config.value.language
    }

    // 加载清单源列表
    await loadManifestSources()

    // 加载背景图片预览
    await loadBackgroundPreview()
  } catch (error) {
    message.error(t('settings.loadConfigFailed') + '：' + error)
  }

  // 深链定位：滚动到对应设置卡片
  if (deepLink?.card) {
    await nextTick()
    setTimeout(() => {
      document.getElementById(deepLink.card!)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }, 200)
  }
})
</script>

<style scoped>
.settings-view {
  max-width: 800px;
  margin: 0 auto;
  padding: 0 24px;
}

.about-section {
  text-align: center;
  padding: 20px 0;
  opacity: 0.6;
  transition: opacity 0.3s ease;
}

.about-section:hover {
  opacity: 1;
}
</style>
