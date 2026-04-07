<template>
  <div class="settings-view">
    <n-space vertical size="large">
      <!-- 语言设置 -->
      <LanguageSettings
        :language="language"
        @update:language="handleSaveLanguage"
      />

      <!-- 路径信息 -->
      <PathSettings :config="config" />

      <!-- 清单设置 -->
      <ManifestSettings
        :manifest-url="manifestUrl"
        @update:manifest-url="manifestUrl = $event"
        @save="handleSaveManifestUrl"
        @reset="handleResetManifestUrl"
      />

      <!-- 下载源设置 -->
      <SourceSettings
        :sources="modSources"
        @toggle-source="handleToggleSource"
        @set-default-source="handleSetDefaultSource"
        @delete-source="handleDeleteSource"
        @add-source="showAddSourceDialog = true"
      />

      <!-- 背景设置 -->
      <BackgroundSettings
        :has-background="!!config?.backgroundImage"
        :preview="backgroundImagePreview"
        @select="handleSelectBackground"
        @clear="handleClearBackground"
      />

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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog, NAlert } from 'naive-ui'
import { ModSourceManager } from '../managers'
import type { ModSource } from '../types/mod-source'
import { InformationCircleOutline as InformationIcon } from '@vicons/ionicons5'
import { GetConfig, SetManifestURL, SetMaxConcurrent, SetLanguage, GetAppInfo, CheckUpdateForce, SelectBackgroundFile, SetBackground, ClearBackground } from '../api/config'
import { useVersionStore } from '../stores/version'
import type { AppConfig } from '../types/config'
import LanguageSettings from '../components/settings/LanguageSettings.vue'
import PathSettings from '../components/settings/PathSettings.vue'
import ManifestSettings from '../components/settings/ManifestSettings.vue'
import SourceSettings from '../components/settings/SourceSettings.vue'
import BackgroundSettings from '../components/settings/BackgroundSettings.vue'
import AboutDialog from '../components/settings/AboutDialog.vue'
import AddSourceDialog from '../components/settings/AddSourceDialog.vue'

const { t, locale } = useI18n()

const message = useMessage()
const dialog = useDialog()
const versionStore = useVersionStore()

const config = ref<AppConfig | null>(null)
const manifestUrl = ref('')
const maxConcurrent = ref(3)
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

// 加载模组下载源列表
async function loadModSources() {
  // 重新加载源列表以获取最新状态
  await ModSourceManager.reloadSources()
  modSources.value = ModSourceManager.getAllSources()
  console.log('设置页面加载下载源:', modSources.value)
  console.log('各类型源统计:', {
    mods: modSources.value.filter(s => s.type === 'mods').length,
    savegames: modSources.value.filter(s => s.type === 'savegames').length,
    furniture: modSources.value.filter(s => s.type === 'furniture').length,
    textures: modSources.value.filter(s => s.type === 'textures').length,
    skins: modSources.value.filter(s => s.type === 'skins').length
  })
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
async function handleAddSource(source: { name: string; description: string; apiUrl: string; type: 'mods' | 'savegames' | 'furniture' | 'textures' | 'skins' }) {
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
      }
    })

    loadModSources()
    showAddSourceDialog.value = false
    message.success(t('mods.sourceAdded'))
  } catch (error) {
    message.error(t('mods.operationFailed') + '：' + error)
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

async function handleSaveManifestUrl() {
  if (!manifestUrl.value.trim()) {
    message.error(t('settings.manifestUrlEmpty'))
    return
  }

  try {
    await SetManifestURL(manifestUrl.value.trim())
    message.success(t('settings.manifestUrlSaved'))

    // 清除清单缓存，以便下次进入版本页面时重新获取
    versionStore.clearManifestCache()
  } catch (error) {
    message.error(t('settings.saveFailed') + '：' + error)
  }
}

function handleResetManifestUrl() {
  manifestUrl.value = 'https://github.com/jxsm/SCVersionList/raw/refs/heads/main/manifest.json'
}

async function handleSaveSettings() {
  try {
    await SetMaxConcurrent(maxConcurrent.value)
    message.success(t('settings.settingsSaved'))
  } catch (error) {
    message.error(t('settings.saveFailed') + '：' + error)
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
            h('p', { style: 'margin-bottom: 12px; font-weight: bold; color: #18a058;' }, `${t('settings.latestVersion')}: v${updateInfo.latestVersion}`),
            h('p', { style: 'margin-bottom: 12px;' }, `${t('settings.releaseDate')}: ${new Date(updateInfo.publishedAt).toLocaleString()}`),
            h(NAlert, {
              type: 'info',
              title: t('settings.updateContent')
            }, {
              default: () => h('pre', {
                style: 'max-height: 200px; overflow-y: auto; background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; font-size: 12px; white-space: pre-wrap;'
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
      maxConcurrent.value = config.value.maxConcurrent
      language.value = config.value.language
    }

    // 加载背景图片预览
    await loadBackgroundPreview()
  } catch (error) {
    message.error(t('settings.loadConfigFailed') + '：' + error)
  }
})
</script>

<style scoped>
.settings-view {
  max-width: 800px;
  margin: 0 auto;
}

.about-section {
  text-align: center;
  padding: 20px 0;
  opacity: 0.6;
  transition: opacity 0.3s;
}

.about-section:hover {
  opacity: 1;
}
</style>
