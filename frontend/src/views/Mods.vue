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
                :placeholder="t('mods.selectVersion')"
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
                {{ t('mods.importMod') }}
              </n-button>
              <n-button
                @click="handleOpenModsFolder"
                :disabled="!selectedVersion"
              >
                <template #icon>
                  <n-icon><FolderIcon /></n-icon>
                </template>
                {{ t('mods.openModsFolder') }}
              </n-button>
            </n-space>
            <n-text depth="3">
              {{ t('mods.totalMods', { total: modStore.mods.length, displayed: filteredMods.length }) }}
            </n-text>
          </n-space>

          <!-- 搜索和筛选 -->
          <n-space>
            <n-input
              v-model:value="searchText"
              :placeholder="t('mods.searchPlaceholder')"
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
                    {{ mod.enabled ? t('mods.enabled') : t('mods.disabled') }}
                  </n-tag>
                </n-space>
              </template>

              <template #description>
                <n-space vertical size="small">
                  <n-text depth="3">
                    {{ t('common.size') }}: {{ formatSize(mod.size) }}
                  </n-text>
                  <n-text depth="3">
                    {{ t('mods.installDate') }}: {{ new Date(mod.installDate).toLocaleString() }}
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
                        {{ t('common.delete') }}
                      </n-button>
                    </template>
                    {{ t('mods.confirmDeleteMessage') }}
                  </n-popconfirm>
                </n-space>
              </template>
            </n-thing>
          </n-list-item>
        </n-list>
        <n-empty
          v-if="filteredMods.length === 0 && !modStore.loading"
          :description="searchText || filterType !== 'all' ? t('mods.noMatchingMods') : t('mods.noMods')"
        />
      </n-spin>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n' 
import { useRouter, useRoute } from 'vue-router'
import { useModStore } from '../stores/mod'
import { useVersionStore } from '../stores/version'
import { useMessage } from 'naive-ui'
import { Add as AddIcon, Search as SearchIcon, FolderOpen as FolderIcon } from '@vicons/ionicons5'
import { formatSize } from '../utils/format'
import { OpenVersionModsFolder } from '../api/version'

const { t } = useI18n()
const modStore = useModStore()
const versionStore = useVersionStore()
const message = useMessage()
const router = useRouter()
const route = useRoute()

const selectedVersion = ref<string>('')
const searchText = ref<string>('')
const filterType = ref<string>('all')

// Filter options
const filterOptions = computed(() => [
  { label: t('mods.all'), value: 'all' },
  { label: t('mods.enabled'), value: 'enabled' },
  { label: t('mods.disabled'), value: 'disabled' }
])

// Installed version options (filter out versions with missing paths)
const installedVersionOptions = computed(() => {
  return versionStore.installedVersions
    .filter(v => v.pathExists !== false && v.pathExists !== undefined)
    .map(v => ({
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
    message.warning(t('mods.noVersionSelected'))
    return
  }

  try {
    // Use Wails file selection dialog
    const { SelectModFile } = await import('../api/mod')
    const filePath = await SelectModFile()

    if (filePath) {
      await modStore.importMod(selectedVersion.value, filePath)
      message.success(t('mods.importSuccess'))
    }
  } catch (error) {
    message.error(t('mods.importFailed') + '：' + error)
  }
}

async function handleOpenModsFolder() {
  if (!selectedVersion.value) {
    message.warning(t('mods.noVersionSelected'))
    return
  }

  try {
    await OpenVersionModsFolder(selectedVersion.value)
  } catch (error) {
    message.error(t('mods.openFolderFailed') + '：' + error)
  }
}

function handleToggleMod(mod: any, enabled: boolean) {
  modStore.toggleMod(selectedVersion.value, mod.id, enabled)
    .catch((error) => {
      message.error(t('mods.toggleFailed') + '：' + error)
      // Revert the UI change on error
      mod.enabled = !enabled
    })
}

function handleDeleteMod(mod: any) {
  modStore.deleteMod(selectedVersion.value, mod.id)
    .then(() => {
      message.success(t('mods.deleteSuccess'))
    })
    .catch((error) => {
      message.error(t('mods.deleteFailed') + '：' + error)
    })
}

onMounted(async () => {
  try {
    await versionStore.getVersions()
    await versionStore.getPrimaryVersion()

    // Get valid versions (paths exist)
    const validVersions = versionStore.installedVersions.filter(v => v.pathExists !== false && v.pathExists !== undefined)

    // Check if versionId is provided in route query
    const versionIdFromRoute = route.query.versionId as string
    if (versionIdFromRoute) {
      // Check if the version exists and path exists
      const version = validVersions.find(v => v.id === versionIdFromRoute)
      if (version) {
        selectedVersion.value = versionIdFromRoute
        await modStore.loadMods(selectedVersion.value)
        return
      }
      message.warning(t('mods.versionPathMissing'))
    }

    // Select primary version by default (if path exists)
    if (versionStore.primaryVersion && versionStore.primaryVersion.pathExists !== false && versionStore.primaryVersion.pathExists !== undefined) {
      selectedVersion.value = versionStore.primaryVersion.id
      await modStore.loadMods(selectedVersion.value)
    } else if (validVersions.length > 0) {
      // If no primary version or primary version path missing, select the first valid version
      selectedVersion.value = validVersions[0].id
      await modStore.loadMods(selectedVersion.value)
    } else {
      // No valid versions found
      message.warning(t('mods.noValidVersions'))
    }
  } catch (error) {
    message.error(t('mods.loadVersionsFailed') + '：' + error)
  }
})
</script>

<style scoped>
.mods-view {
  max-width: 1000px;
  margin: 0 auto;
}
</style>
