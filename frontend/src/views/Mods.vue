<template>
  <div class="mods-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space vertical size="medium">
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
              <n-button
                @click="handleOpenModsFolder"
                :disabled="!selectedVersion"
              >
                <template #icon>
                  <n-icon><FolderIcon /></n-icon>
                </template>
                打开文件夹
              </n-button>
            </n-space>
            <n-text depth="3">
              共 {{ modStore.mods.length }} 个模组，显示 {{ filteredMods.length }} 个
            </n-text>
          </n-space>

          <!-- 搜索和筛选 -->
          <n-space>
            <n-input
              v-model:value="searchText"
              placeholder="搜索模组名称..."
              clearable
              style="width: 300px"
            >
              <template #prefix>
                <n-icon><SearchIcon /></n-icon>
              </template>
            </n-input>
            <n-select
              v-model:value="filterType"
              :options="filterOptions"
              style="width: 150px"
            />
          </n-space>
        </n-space>
      </n-card>

      <!-- 模组列表 -->
      <n-spin :show="modStore.loading">
        <n-list hoverable clickable>
          <n-list-item v-for="mod in filteredMods" :key="mod.id">
            <n-thing>
              <template #header>
                <n-space align="center">
                  <n-checkbox
                    :checked="mod.enabled"
                    @update:checked="handleToggleMod(mod, $event)"
                  >
                    <n-text strong>{{ mod.name }}</n-text>
                  </n-checkbox>
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
          v-if="filteredMods.length === 0 && !modStore.loading"
          :description="searchText || filterType !== 'all' ? '没有找到匹配的模组' : '暂无模组'"
        />
      </n-spin>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useModStore } from '../stores/mod'
import { useVersionStore } from '../stores/version'
import { useMessage } from 'naive-ui'
import { Add as AddIcon, Search as SearchIcon, FolderOpen as FolderIcon } from '@vicons/ionicons5'
import { formatSize } from '../utils/format'
import { OpenVersionModsFolder } from '../api/version'

const modStore = useModStore()
const versionStore = useVersionStore()
const message = useMessage()
const router = useRouter()
const route = useRoute()

const selectedVersion = ref<string>('')
const searchText = ref<string>('')
const filterType = ref<string>('all')

// Filter options
const filterOptions = [
  { label: '全部', value: 'all' },
  { label: '已启用', value: 'enabled' },
  { label: '已禁用', value: 'disabled' }
]

// Installed version options
const installedVersionOptions = computed(() => {
  return versionStore.installedVersions.map(v => ({
    label: v.name,
    value: v.id
  }))
})

// Filtered mods based on search and filter
const filteredMods = computed(() => {
  let mods = modStore.mods

  // Apply status filter
  if (filterType.value === 'enabled') {
    mods = mods.filter(m => m.enabled)
  } else if (filterType.value === 'disabled') {
    mods = mods.filter(m => !m.enabled)
  }

  // Apply search filter
  if (searchText.value.trim()) {
    const searchLower = searchText.value.toLowerCase().trim()
    mods = mods.filter(m =>
      m.name.toLowerCase().includes(searchLower)
    )
  }

  return mods
})

function handleVersionChange() {
  if (selectedVersion.value) {
    modStore.loadMods(selectedVersion.value)
    // Reset filters when changing version
    searchText.value = ''
    filterType.value = 'all'
  }
}

async function handleImportMod() {
  if (!selectedVersion.value) {
    message.warning('请先选择一个版本')
    return
  }

  try {
    // Use Wails file selection dialog
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

async function handleOpenModsFolder() {
  if (!selectedVersion.value) {
    message.warning('请先选择一个版本')
    return
  }

  try {
    await OpenVersionModsFolder(selectedVersion.value)
  } catch (error) {
    message.error('打开文件夹失败：' + error)
  }
}

function handleToggleMod(mod: any, enabled: boolean) {
  modStore.toggleMod(selectedVersion.value, mod.id, enabled)
    .catch((error) => {
      message.error('操作失败：' + error)
      // Revert the UI change on error
      mod.enabled = !enabled
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

    // Check if versionId is provided in route query
    const versionIdFromRoute = route.query.versionId as string
    if (versionIdFromRoute) {
      // Check if the version exists and is installed
      const version = versionStore.installedVersions.find(v => v.id === versionIdFromRoute)
      if (version) {
        selectedVersion.value = versionIdFromRoute
        await modStore.loadMods(selectedVersion.value)
        return
      }
    }

    // Select primary version by default
    if (versionStore.primaryVersion) {
      selectedVersion.value = versionStore.primaryVersion.id
      await modStore.loadMods(selectedVersion.value)
    } else if (versionStore.installedVersions.length > 0) {
      // If no primary version, select the first installed version
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
