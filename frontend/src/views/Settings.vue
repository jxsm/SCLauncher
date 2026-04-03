<template>
  <div class="settings-view">
    <n-space vertical size="large">
      <!-- 语言设置 -->
      <n-card :title="t('settings.languageSettings')">
        <n-form-item :label="t('settings.language')">
          <n-select
            v-model:value="language"
            :options="languageOptions"
            @update:value="handleSaveLanguage"
          />
        </n-form-item>
      </n-card>

      <!-- 路径信息 -->
      <n-card :title="t('settings.paths')">
        <n-descriptions :column="1" bordered>
          <n-descriptions-item :label="t('settings.dataDir')">
            {{ config?.dataDir }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('settings.versionsDir')">
            {{ config?.versionsDir }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('settings.downloadsDir')">
            {{ config?.downloadsDir }}
          </n-descriptions-item>
        </n-descriptions>
      </n-card>

      <!-- 清单设置 -->
      <n-card :title="t('settings.manifest')">
        <n-form-item :label="t('settings.manifestUrl')">
          <n-input
            v-model:value="manifestUrl"
            :placeholder="t('settings.manifestUrlPlaceholder') || '请输入清单文件 URL'"
          />
        </n-form-item>
        <n-space>
          <n-button type="primary" @click="handleSaveManifestUrl">
            {{ t('settings.saveManifestUrl') }}
          </n-button>
          <n-button @click="handleResetManifestUrl">
            {{ t('settings.resetManifestUrl') }}
          </n-button>
        </n-space>
      </n-card>

      <!-- 其他设置 -->
      <n-card :title="t('settings.other')">
        <n-form-item :label="t('settings.maxConcurrent')">
          <n-input-number
            v-model:value="maxConcurrent"
            :min="1"
            :max="10"
          />
        </n-form-item>
        <n-button type="primary" @click="handleSaveSettings">
          {{ t('settings.saveSettings') }}
        </n-button>
      </n-card>

      <!-- 背景设置 -->
      <n-card :title="t('settings.background')">
        <n-space vertical>
          <n-form-item :label="t('settings.backgroundImage')">
            <n-space>
              <n-button @click="handleSelectBackground">
                <template #icon>
                  <n-icon><ImageIcon /></n-icon>
                </template>
                {{ t('settings.selectImage') }}
              </n-button>
              <n-button v-if="config?.backgroundImage" type="error" @click="handleClearBackground">
                <template #icon>
                  <n-icon><TrashIcon /></n-icon>
                </template>
                {{ t('settings.clearBackground') }}
              </n-button>
            </n-space>
          </n-form-item>

          <!-- 背景预览 -->
          <div v-if="backgroundImagePreview" class="background-preview">
            <n-image
              :src="backgroundImagePreview"
              object-fit="cover"
              style="width: 100%; height: 200px; border-radius: 4px;"
            />
          </div>
          <n-text v-else depth="3">{{ t('settings.noBackground') }}</n-text>
        </n-space>
      </n-card>

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
    <n-modal v-model:show="showAboutDialog" preset="dialog" :title="t('settings.aboutSCLauncher')">
      <n-space vertical>
        <n-descriptions :column="1" bordered label-placement="left" label-style="width: 80px;">
          <n-descriptions-item :label="t('common.version')">
            v{{ appInfo.version }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('settings.author')">
            {{ appInfo.repoOwner }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('settings.license')">
            MIT License
          </n-descriptions-item>
        </n-descriptions>
        <n-divider />
        <n-text>
          {{ t('settings.description') }}
        </n-text>
        <n-button type="primary" block @click="openGitHub">
          <template #icon>
            <n-icon><GithubIcon /></n-icon>
          </template>
          {{ t('settings.viewOnGitHub') }}
        </n-button>
        <n-button block @click="handleCheckUpdate">
          <template #icon>
            <n-icon><UpdateIcon /></n-icon>
          </template>
          {{ t('settings.checkUpdate') }}
        </n-button>
      </n-space>
      <template #action>
        <n-button @click="showAboutDialog = false">{{ t('common.close') }}</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog, NAlert } from 'naive-ui'
import { InformationCircleOutline as InformationIcon, LogoGithub as GithubIcon, RefreshOutline as UpdateIcon, ImageOutline as ImageIcon, TrashOutline as TrashIcon } from '@vicons/ionicons5'
import { GetConfig, SetManifestURL, SetMaxConcurrent, SetLanguage, GetAppInfo, CheckUpdate, SelectBackgroundFile, SetBackground, ClearBackground } from '../api/config'
import { useVersionStore } from '../stores/version'
import type { AppConfig } from '../types/config'

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

// 语言选项
const languageOptions = [
  { label: '简体中文', value: 'zh-CN' },
  { label: 'English', value: 'en-US' }
]

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

async function handleSaveLanguage() {
  try {
    await SetLanguage(language.value)
    // 立即切换应用语言
    locale.value = language.value
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
    const updateInfo = await CheckUpdate()
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

.background-preview {
  width: 100%;
  border: 1px solid #e0e0e0;
  border-radius: 4px;
  overflow: hidden;
  background-color: #f5f5f5;
}
</style>
