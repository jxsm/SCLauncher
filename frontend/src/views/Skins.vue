<template>
  <div class="skins-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space justify="space-between">
          <n-space>
            <n-button type="primary" @click="handleImportSkin">
              <template #icon>
                <n-icon><AddIcon /></n-icon>
              </template>
              {{ t('skins.import') }}
            </n-button>
            <n-button @click="handleRefresh">
              <template #icon>
                <n-icon><RefreshIcon /></n-icon>
              </template>
              {{ t('common.refresh') }}
            </n-button>
          </n-space>
          <n-text depth="3">
            共 {{ skinStore.skins.length }} 个皮肤
          </n-text>
        </n-space>
      </n-card>

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
                    大小: {{ formatSize(skin.size) }}
                  </n-text>
                  <n-text depth="3">
                    导入日期: {{ skin.importDate }}
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
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSkinStore } from '../stores/skin'
import { useMessage } from 'naive-ui'
import { Add as AddIcon, Refresh as RefreshIcon, Image as ImageIcon } from '@vicons/ionicons5'
import { formatSize } from '../utils/format'

const { t } = useI18n()
const skinStore = useSkinStore()
const message = useMessage()
const skinImages = ref<Record<string, string>>({})
const loadingImages = ref<Record<string, boolean>>({})

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

onMounted(async () => {
  try {
    await skinStore.loadSkins()
    // 预加载所有皮肤图片
    await preloadSkinImages()
  } catch (error) {
    message.error(t('skins.loadFailed') + '：' + error)
  }
})
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
</style>
