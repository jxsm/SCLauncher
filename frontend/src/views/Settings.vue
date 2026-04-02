<template>
  <div class="settings-view">
    <n-space vertical size="large">
      <!-- 应用信息 -->
      <n-card title="应用信息">
        <n-descriptions :column="1" bordered>
          <n-descriptions-item label="应用名称">
            SCLauncher
          </n-descriptions-item>
          <n-descriptions-item label="版本">
            v0.1.0
          </n-descriptions-item>
          <n-descriptions-item label="作者">
            jxsm
          </n-descriptions-item>
        </n-descriptions>
      </n-card>

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
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { GetConfig } from '../api/config'
import type { AppConfig } from '../types/config'

const message = useMessage()

const config = ref<AppConfig | null>(null)
const manifestUrl = ref('')
const maxConcurrent = ref(3)

async function handleSaveManifestUrl() {
  // TODO: 实现保存清单 URL 的逻辑
  message.success('清单 URL 已保存（功能待实现）')
}

function handleResetManifestUrl() {
  manifestUrl.value = 'https://github.com/jxsm/SCVersionList/raw/refs/heads/main/manifest.json'
}

async function handleSaveSettings() {
  // TODO: 实现保存设置的逻辑
  message.success('设置已保存（功能待实现）')
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
</style>
