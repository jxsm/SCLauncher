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
        <n-alert type="info" style="margin-bottom: 16px;">
          {{ t('settings.pathsPortableInfo') }}
        </n-alert>
        <n-descriptions :column="1" bordered>
          <n-descriptions-item :label="t('settings.dataDir')">
            <n-code>{{ config?.dataDir }}</n-code>
          </n-descriptions-item>
          <n-descriptions-item :label="t('settings.versionsDir')">
            <n-code>{{ config?.versionsDir }}</n-code>
          </n-descriptions-item>
          <n-descriptions-item :label="t('settings.downloadsDir')">
            <n-code>{{ config?.downloadsDir }}</n-code>
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

      <!-- 下载源设置 -->
      <n-card :title="t('mods.manageSources')">
        <template #header-extra>
          <n-button size="small" type="primary" @click="showAddSourceDialog = true">
            <template #icon>
              <n-icon><AddIcon /></n-icon>
            </template>
            {{ t('mods.addSource') }}
          </n-button>
        </template>

        <n-tabs v-model:value="currentSourceType" type="line">
          <n-tab-pane name="mods" :tab="t('mods.modSources')">
            <n-list v-if="getSourcesByType('mods').length > 0" hoverable>
              <n-list-item v-for="source in getSourcesByType('mods')" :key="source.id">
                <n-thing>
                  <template #header>
                    <n-space align="center">
                      <n-avatar v-if="source.icon" :src="source.icon" :size="32" round />
                      <n-avatar v-else :size="32" round>
                        {{ source.name.charAt(0) }}
                      </n-avatar>
                      <n-text strong>{{ source.name }}</n-text>
                      <n-tag v-if="source.isDefault" size="small" type="info">
                        {{ t('mods.defaultSource') }}
                      </n-tag>
                      <n-switch
                        v-model:value="source.enabled"
                        :disabled="source.id === 'suancaixianyu'"
                        @update:value="(enabled: boolean) => handleToggleSource(source.id, enabled)"
                      />
                    </n-space>
                  </template>

                  <template #description>
                    <n-text depth="3">
                      {{ source.description }}
                    </n-text>
                  </template>

                  <template #action>
                    <n-space>
                      <n-button
                        size="small"
                        :type="source.isDefault ? 'warning' : 'default'"
                        @click="handleSetDefaultSource(source)"
                      >
                        <template #icon>
                          <n-icon><StarIcon /></n-icon>
                        </template>
                        {{ source.isDefault ? t('mods.cancelDefault') : t('mods.setDefault') }}
                      </n-button>
                      <n-button
                        v-if="!source.isDefault"
                        size="small"
                        type="error"
                        @click="handleDeleteSource(source)"
                      >
                        {{ t('common.delete') }}
                      </n-button>
                    </n-space>
                  </template>
                </n-thing>
              </n-list-item>
            </n-list>
            <n-empty v-else :description="t('mods.noSources')" />
          </n-tab-pane>

          <n-tab-pane name="savegames" :tab="t('mods.savegameSources')">
            <n-list v-if="getSourcesByType('savegames').length > 0" hoverable>
              <n-list-item v-for="source in getSourcesByType('savegames')" :key="source.id">
                <n-thing>
                  <template #header>
                    <n-space align="center">
                      <n-avatar v-if="source.icon" :src="source.icon" :size="32" round />
                      <n-avatar v-else :size="32" round>
                        {{ source.name.charAt(0) }}
                      </n-avatar>
                      <n-text strong>{{ source.name }}</n-text>
                      <n-tag v-if="source.isDefault" size="small" type="info">
                        {{ t('mods.defaultSource') }}
                      </n-tag>
                      <n-switch
                        v-model:value="source.enabled"
                        :disabled="source.id === 'suancaixianyu-saves'"
                        @update:value="(enabled: boolean) => handleToggleSource(source.id, enabled)"
                      />
                    </n-space>
                  </template>

                  <template #description>
                    <n-text depth="3">
                      {{ source.description }}
                    </n-text>
                  </template>

                  <template #action>
                    <n-space>
                      <n-button
                        size="small"
                        :type="source.isDefault ? 'warning' : 'default'"
                        @click="handleSetDefaultSource(source)"
                      >
                        <template #icon>
                          <n-icon><StarIcon /></n-icon>
                        </template>
                        {{ source.isDefault ? t('mods.cancelDefault') : t('mods.setDefault') }}
                      </n-button>
                      <n-button
                        v-if="!source.isDefault"
                        size="small"
                        type="error"
                        @click="handleDeleteSource(source)"
                        :disabled="source.id === 'suancaixianyu-saves'"
                      >
                        <template #icon>
                          <n-icon><TrashIcon /></n-icon>
                        </template>
                        {{ t('common.delete') }}
                      </n-button>
                    </n-space>
                  </template>
                </n-thing>
              </n-list-item>
            </n-list>
            <n-empty v-else :description="t('mods.noSources')" />
          </n-tab-pane>

          <n-tab-pane name="furniture" :tab="t('mods.furnitureSources')">
            <n-list v-if="getSourcesByType('furniture').length > 0" hoverable>
              <n-list-item v-for="source in getSourcesByType('furniture')" :key="source.id">
                <n-thing>
                  <template #header>
                    <n-space align="center">
                      <n-avatar v-if="source.icon" :src="source.icon" :size="32" round />
                      <n-avatar v-else :size="32" round>
                        {{ source.name.charAt(0) }}
                      </n-avatar>
                      <n-text strong>{{ source.name }}</n-text>
                      <n-tag v-if="source.isDefault" size="small" type="info">
                        {{ t('mods.defaultSource') }}
                      </n-tag>
                      <n-switch
                        v-model:value="source.enabled"
                        :disabled="source.id === 'suancaixianyu'"
                        @update:value="(enabled: boolean) => handleToggleSource(source.id, enabled)"
                      />
                    </n-space>
                  </template>

                  <template #description>
                    <n-text depth="3">
                      {{ source.description }}
                    </n-text>
                  </template>

                  <template #action>
                    <n-space>
                      <n-button
                        size="small"
                        :type="source.isDefault ? 'warning' : 'default'"
                        @click="handleSetDefaultSource(source)"
                      >
                        <template #icon>
                          <n-icon><StarIcon /></n-icon>
                        </template>
                        {{ source.isDefault ? t('mods.cancelDefault') : t('mods.setDefault') }}
                      </n-button>
                      <n-button
                        v-if="!source.isDefault"
                        size="small"
                        type="error"
                        @click="handleDeleteSource(source)"
                      >
                        {{ t('common.delete') }}
                      </n-button>
                    </n-space>
                  </template>
                </n-thing>
              </n-list-item>
            </n-list>
            <n-empty v-else :description="t('mods.noSources')" />
          </n-tab-pane>

          <n-tab-pane name="textures" :tab="t('mods.textureSources')">
            <n-list v-if="getSourcesByType('textures').length > 0" hoverable>
              <n-list-item v-for="source in getSourcesByType('textures')" :key="source.id">
                <n-thing>
                  <template #header>
                    <n-space align="center">
                      <n-avatar v-if="source.icon" :src="source.icon" :size="32" round />
                      <n-avatar v-else :size="32" round>
                        {{ source.name.charAt(0) }}
                      </n-avatar>
                      <n-text strong>{{ source.name }}</n-text>
                      <n-tag v-if="source.isDefault" size="small" type="info">
                        {{ t('mods.defaultSource') }}
                      </n-tag>
                      <n-switch
                        v-model:value="source.enabled"
                        :disabled="source.id === 'suancaixianyu'"
                        @update:value="(enabled: boolean) => handleToggleSource(source.id, enabled)"
                      />
                    </n-space>
                  </template>

                  <template #description>
                    <n-text depth="3">
                      {{ source.description }}
                    </n-text>
                  </template>

                  <template #action>
                    <n-space>
                      <n-button
                        size="small"
                        :type="source.isDefault ? 'warning' : 'default'"
                        @click="handleSetDefaultSource(source)"
                      >
                        <template #icon>
                          <n-icon><StarIcon /></n-icon>
                        </template>
                        {{ source.isDefault ? t('mods.cancelDefault') : t('mods.setDefault') }}
                      </n-button>
                      <n-button
                        v-if="!source.isDefault"
                        size="small"
                        type="error"
                        @click="handleDeleteSource(source)"
                      >
                        {{ t('common.delete') }}
                      </n-button>
                    </n-space>
                  </template>
                </n-thing>
              </n-list-item>
            </n-list>
            <n-empty v-else :description="t('mods.noSources')" />
          </n-tab-pane>

          <n-tab-pane name="skins" :tab="t('mods.skinSources')">
            <n-list v-if="getSourcesByType('skins').length > 0" hoverable>
              <n-list-item v-for="source in getSourcesByType('skins')" :key="source.id">
                <n-thing>
                  <template #header>
                    <n-space align="center">
                      <n-avatar v-if="source.icon" :src="source.icon" :size="32" round />
                      <n-avatar v-else :size="32" round>
                        {{ source.name.charAt(0) }}
                      </n-avatar>
                      <n-text strong>{{ source.name }}</n-text>
                      <n-tag v-if="source.isDefault" size="small" type="info">
                        {{ t('mods.defaultSource') }}
                      </n-tag>
                      <n-switch
                        v-model:value="source.enabled"
                        :disabled="source.id === 'suancaixianyu'"
                        @update:value="(enabled: boolean) => handleToggleSource(source.id, enabled)"
                      />
                    </n-space>
                  </template>

                  <template #description>
                    <n-text depth="3">
                      {{ source.description }}
                    </n-text>
                  </template>

                  <template #action>
                    <n-space>
                      <n-button
                        size="small"
                        :type="source.isDefault ? 'warning' : 'default'"
                        @click="handleSetDefaultSource(source)"
                      >
                        <template #icon>
                          <n-icon><StarIcon /></n-icon>
                        </template>
                        {{ source.isDefault ? t('mods.cancelDefault') : t('mods.setDefault') }}
                      </n-button>
                      <n-button
                        v-if="!source.isDefault"
                        size="small"
                        type="error"
                        @click="handleDeleteSource(source)"
                      >
                        {{ t('common.delete') }}
                      </n-button>
                    </n-space>
                  </template>
                </n-thing>
              </n-list-item>
            </n-list>
            <n-empty v-else :description="t('mods.noSources')" />
          </n-tab-pane>
        </n-tabs>
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

    <!-- 添加下载源对话框 -->
    <n-modal v-model:show="showAddSourceDialog" preset="dialog" :title="t('mods.addSource')">
      <n-form ref="addSourceFormRef" :model="newSource" :rules="sourceRules" label-placement="left" label-width="100px">
        <n-form-item :label="t('mods.sourceType')" path="type">
          <n-select
            v-model:value="newSource.type"
            :options="[
              { label: t('mods.modSources'), value: 'mods' },
              { label: t('mods.savegameSources'), value: 'savegames' },
              { label: t('mods.furnitureSources'), value: 'furniture' },
              { label: t('mods.textureSources'), value: 'textures' },
              { label: t('mods.skinSources'), value: 'skins' }
            ]"
          />
        </n-form-item>
        <n-form-item :label="t('mods.sourceName')" path="name">
          <n-input v-model:value="newSource.name" :placeholder="t('mods.sourceNamePlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('mods.sourceDescription')" path="description">
          <n-input v-model:value="newSource.description" :placeholder="t('mods.sourceDescriptionPlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('mods.sourceApiUrl')" path="apiUrl">
          <n-input v-model:value="newSource.apiUrl" :placeholder="t('mods.sourceApiUrlPlaceholder')" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-button @click="showAddSourceDialog = false">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" @click="handleAddSource">{{ t('common.confirm') }}</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog, NAlert } from 'naive-ui'
import { ModSourceManager } from '../managers'
import type { ModSource } from '../types/mod-source'
import { Add as AddIcon, Star as StarIcon } from '@vicons/ionicons5'
import { InformationCircleOutline as InformationIcon, LogoGithub as GithubIcon, RefreshOutline as UpdateIcon, ImageOutline as ImageIcon, TrashOutline as TrashIcon } from '@vicons/ionicons5'
import { GetConfig, SetManifestURL, SetMaxConcurrent, SetLanguage, GetAppInfo, CheckUpdateForce, SelectBackgroundFile, SetBackground, ClearBackground } from '../api/config'
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

// 模组下载源相关
const modSources = ref<ModSource[]>([])
const currentSourceType = ref<string>('mods')
const showAddSourceDialog = ref(false)
const addSourceFormRef = ref()
const newSource = ref({
  name: '',
  description: '',
  apiUrl: '',
  type: 'mods' as 'mods' | 'savegames' | 'furniture' | 'textures' | 'skins'
})

// 根据类型过滤下载源
const getSourcesByType = (type: string) => {
  return modSources.value.filter(s => s.type === type)
}

// 下载源表单验证规则
const sourceRules = {
  name: {
    required: true,
    message: t('mods.sourceNameRequired'),
    trigger: 'blur'
  },
  description: {
    required: true,
    message: t('mods.sourceDescriptionRequired'),
    trigger: 'blur'
  },
  apiUrl: {
    required: true,
    message: t('mods.sourceApiUrlRequired'),
    trigger: 'blur'
  }
}

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
async function handleAddSource() {
  if (!newSource.value.name || !newSource.value.description || !newSource.value.apiUrl) {
    message.warning(t('mods.pleaseFillAllFields'))
    return
  }

  try {
    // 生成唯一ID
    const id = 'custom-' + Date.now()

    await ModSourceManager.addSource({
      id,
      type: newSource.value.type,
      name: newSource.value.name,
      description: newSource.value.description,
      enabled: true,
      api: {
        baseUrl: newSource.value.apiUrl,
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
    newSource.value = { name: '', description: '', apiUrl: '', type: 'mods' }
    message.success(t('mods.sourceAdded'))
  } catch (error) {
    message.error(t('mods.operationFailed') + '：' + error)
  }
}

// 语言选项
const languageOptions = [
  { label: '简体中文', value: 'zh-CN' },
  { label: 'English', value: 'en-US' },
  { label: 'Русский', value: 'ru-RU' },
  { label: 'Português (Brasil)', value: 'pt-BR' },
  { label: 'हिन्दी', value: 'hi-IN' },
  { label: 'Bahasa Indonesia', value: 'id-ID' },
  { label: 'العربية', value: 'ar-SA' }
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

.background-preview {
  width: 100%;
  border: 1px solid #e0e0e0;
  border-radius: 4px;
  overflow: hidden;
  background-color: #f5f5f5;
}
</style>
