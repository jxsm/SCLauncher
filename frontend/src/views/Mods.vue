<template>
  <div class="mods-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space justify="space-between">
          <n-space>
            <n-select
              v-model:value="selectedVersion"
              :options="installedVersionOptions"
              placeholder="选择版本"
              style="width: 300px"
              @update:value="handleVersionChange"
            />
            <n-button
              type="primary"
              @click="handleImportMod"
              :disabled="!selectedVersion"
            >
              <template #icon>
                <n-icon><AddIcon /></n-icon>
              </template>
              导入模组
            </n-button>
          </n-space>
          <n-text depth="3">
            共 {{ modStore.mods.length }} 个模组
          </n-text>
        </n-space>
      </n-card>

      <!-- 模组列表 -->
      <n-spin :show="modStore.loading">
        <n-list hoverable clickable>
          <n-list-item v-for="mod in modStore.mods" :key="mod.id">
            <n-thing>
              <template #header>
                <n-space align="center">
                  <n-text strong>{{ mod.name }}</n-text>
                  <n-tag :type="mod.enabled ? 'success' : 'default'" size="small">
                    {{ mod.enabled ? '已启用' : '已禁用' }}
                  </n-tag>
                </n-space>
              </template>

              <template #description>
                <n-space vertical size="small">
                  <n-text depth="3">
                    大小: {{ formatSize(mod.size) }}
                  </n-text>
                  <n-text depth="3">
                    安装日期: {{ new Date(mod.installDate).toLocaleString() }}
                  </n-text>
                </n-space>
              </template>

              <template #action>
                <n-space>
                  <n-button
                    size="small"
                    @click="handleToggleMod(mod)"
                  >
                    {{ mod.enabled ? '禁用' : '启用' }}
                  </n-button>
                  <n-popconfirm
                    @positive-click="handleDeleteMod(mod)"
                  >
                    <template #trigger>
                      <n-button type="error" size="small">
                        删除
                      </n-button>
                    </template>
                    确定要删除这个模组吗？
                  </n-popconfirm>
                </n-space>
              </template>
            </n-thing>
          </n-list-item>
        </n-list>
        <n-empty
          v-if="modStore.mods.length === 0 && !modStore.loading"
          description="暂无模组"
        />
      </n-spin>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useModStore } from '../stores/mod'
import { useVersionStore } from '../stores/version'
import { useMessage } from 'naive-ui'
import { Add as AddIcon } from '@vicons/ionicons5'
import { formatSize } from '../utils/format'

const modStore = useModStore()
const versionStore = useVersionStore()
const message = useMessage()

const selectedVersion = ref<string>('')

const installedVersionOptions = computed(() => {
  return versionStore.installedVersions.map(v => ({
    label: v.name,
    value: v.id
  }))
})

function handleVersionChange() {
  if (selectedVersion.value) {
    modStore.loadMods(selectedVersion.value)
  }
}

async function handleImportMod() {
  if (!selectedVersion.value) {
    message.warning('请先选择一个版本')
    return
  }

  try {
    // 使用 Wails 文件选择对话框
    const { SelectModFile } = await import('../api/mod')
    const filePath = await SelectModFile()

    if (filePath) {
      await modStore.importMod(selectedVersion.value, filePath)
      message.success('模组导入成功')
    }
  } catch (error) {
    message.error('模组导入失败：' + error)
  }
}

function handleFileSelected(event: Event) {
  // 这个函数不再使用，保留是为了兼容性
  event.preventDefault()
}

function handleToggleMod(mod: any) {
  modStore.toggleMod(selectedVersion.value, mod.id, !mod.enabled)
    .then(() => {
      message.success('模组状态已更新')
    })
    .catch((error) => {
      message.error('操作失败：' + error)
    })
}

function handleDeleteMod(mod: any) {
  modStore.deleteMod(selectedVersion.value, mod.id)
    .then(() => {
      message.success('模组已删除')
    })
    .catch((error) => {
      message.error('删除失败：' + error)
    })
}

onMounted(async () => {
  try {
    await versionStore.getVersions()
    await versionStore.getPrimaryVersion()

    // 默认选择主要版本
    if (versionStore.primaryVersion) {
      selectedVersion.value = versionStore.primaryVersion.id
      await modStore.loadMods(selectedVersion.value)
    } else if (versionStore.installedVersions.length > 0) {
      // 如果没有主要版本，选择第一个已安装版本
      selectedVersion.value = versionStore.installedVersions[0].id
      await modStore.loadMods(selectedVersion.value)
    }
  } catch (error) {
    message.error('加载版本列表失败：' + error)
  }
})
</script>

<style scoped>
.mods-view {
  max-width: 1000px;
  margin: 0 auto;
}
</style>
