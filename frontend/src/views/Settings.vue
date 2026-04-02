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
            v0.1.0
          </n-descriptions-item>
          <n-descriptions-item label="作者">
            jxsm
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
      </n-space>
      <template #action>
        <n-button @click="showAboutDialog = false">关闭</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { InformationCircleOutline as InformationIcon, LogoGithub as GithubIcon } from '@vicons/ionicons5'
import { GetConfig, SetManifestURL, SetMaxConcurrent } from '../api/config'
import { useVersionStore } from '../stores/version'
import type { AppConfig } from '../types/config'

const message = useMessage()
const versionStore = useVersionStore()

const config = ref<AppConfig | null>(null)
const manifestUrl = ref('')
const maxConcurrent = ref(3)
const showAboutDialog = ref(false)

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

onMounted(async () => {
  try {
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
