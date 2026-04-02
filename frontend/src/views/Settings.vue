<template>
  <div class="settings-view">
    <n-space vertical size="large">
      <!-- 路径信息 -->
      <n-card title="路径信息">
        <n-descriptions :column="1" bordered>
          <n-descriptions-item label="数据目录">
            {{ config?.dataDir }}
          </n-descriptions-item>
          <n-descriptions-item label="版本目录">
            {{ config?.versionsDir }}
          </n-descriptions-item>
          <n-descriptions-item label="下载目录">
            {{ config?.downloadsDir }}
          </n-descriptions-item>
        </n-descriptions>
      </n-card>

      <!-- 清单设置 -->
      <n-card title="清单设置">
        <n-form-item label="清单文件 URL">
          <n-input
            v-model:value="manifestUrl"
            placeholder="请输入清单文件 URL"
          />
        </n-form-item>
        <n-space>
          <n-button type="primary" @click="handleSaveManifestUrl">
            保存
          </n-button>
          <n-button @click="handleResetManifestUrl">
            重置
          </n-button>
        </n-space>
      </n-card>

      <!-- 其他设置 -->
      <n-card title="其他设置">
        <n-form-item label="最大并发下载数">
          <n-input-number
            v-model:value="maxConcurrent"
            :min="1"
            :max="10"
          />
        </n-form-item>
        <n-button type="primary" @click="handleSaveSettings">
          保存设置
        </n-button>
      </n-card>

      <!-- 关于 -->
      <div class="about-section">
        <n-button text @click="showAboutDialog = true">
          <template #icon>
            <n-icon><InformationIcon /></n-icon>
          </template>
          关于
        </n-button>
      </div>
    </n-space>

    <!-- 关于对话框 -->
    <n-modal v-model:show="showAboutDialog" preset="dialog" title="关于 SCLauncher">
      <n-space vertical>
        <n-descriptions :column="1" bordered label-placement="left" label-style="width: 80px;">
          <n-descriptions-item label="版本">
            v{{ appInfo.version }}
          </n-descriptions-item>
          <n-descriptions-item label="作者">
            {{ appInfo.repoOwner }}
          </n-descriptions-item>
          <n-descriptions-item label="开源协议">
            MIT License
          </n-descriptions-item>
        </n-descriptions>
        <n-divider />
        <n-text>
          SCLauncher 是一个开源的生存战争游戏启动器，支持版本管理、模组安装等功能。
        </n-text>
        <n-button type="primary" block @click="openGitHub">
          <template #icon>
            <n-icon><GithubIcon /></n-icon>
          </template>
          在 GitHub 上查看项目
        </n-button>
        <n-button block @click="handleCheckUpdate">
          <template #icon>
            <n-icon><UpdateIcon /></n-icon>
          </template>
          检查更新
        </n-button>
      </n-space>
      <template #action>
        <n-button @click="showAboutDialog = false">关闭</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useMessage, useDialog, NAlert } from 'naive-ui'
import { InformationCircleOutline as InformationIcon, LogoGithub as GithubIcon, RefreshOutline as UpdateIcon } from '@vicons/ionicons5'
import { GetConfig, SetManifestURL, SetMaxConcurrent, GetAppInfo, CheckUpdate } from '../api/config'
import { useVersionStore } from '../stores/version'
import type { AppConfig } from '../types/config'

const message = useMessage()
const dialog = useDialog()
const versionStore = useVersionStore()

const config = ref<AppConfig | null>(null)
const manifestUrl = ref('')
const maxConcurrent = ref(3)
const showAboutDialog = ref(false)
const appInfo = ref<{ version: string; repoOwner: string; repoName: string }>({
  version: '0.0.1',
  repoOwner: 'jxsm',
  repoName: 'SCLauncher'
})

async function handleSaveManifestUrl() {
  if (!manifestUrl.value.trim()) {
    message.error('清单 URL 不能为空')
    return
  }

  try {
    await SetManifestURL(manifestUrl.value.trim())
    message.success('清单 URL 已保存')

    // 清除清单缓存，以便下次进入版本页面时重新获取
    versionStore.clearManifestCache()
  } catch (error) {
    message.error('保存失败：' + error)
  }
}

function handleResetManifestUrl() {
  manifestUrl.value = 'https://github.com/jxsm/SCVersionList/raw/refs/heads/main/manifest.json'
}

async function handleSaveSettings() {
  try {
    await SetMaxConcurrent(maxConcurrent.value)
    message.success('设置已保存')
  } catch (error) {
    message.error('保存失败：' + error)
  }
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
        title: '发现新版本',
        content: () => {
          return h('div', [
            h('p', { style: 'margin-bottom: 12px;' }, `当前版本: v${updateInfo.currentVersion}`),
            h('p', { style: 'margin-bottom: 12px; font-weight: bold; color: #18a058;' }, `最新版本: v${updateInfo.latestVersion}`),
            h('p', { style: 'margin-bottom: 12px;' }, `发布时间: ${new Date(updateInfo.publishedAt).toLocaleString()}`),
            h(NAlert, {
              type: 'info',
              title: '更新内容'
            }, {
              default: () => h('pre', {
                style: 'max-height: 200px; overflow-y: auto; background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; font-size: 12px; white-space: pre-wrap;'
              }, updateInfo.body || '暂无更新说明')
            })
          ])
        },
        positiveText: '前往下载',
        negativeText: '关闭',
        onPositiveClick: () => {
          window.open(updateInfo.url, '_blank')
        }
      })
    } else {
      message.success('当前已是最新版本')
    }
  } catch (error) {
    message.error('检查更新失败：' + error)
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
    }
  } catch (error) {
    message.error('加载配置失败：' + error)
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
