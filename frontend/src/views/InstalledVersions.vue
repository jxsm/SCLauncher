<template>
  <div class="installed-versions-view">
    <n-space vertical size="large">
      <!-- 工具栏 -->
      <n-card>
        <n-space justify="space-between">
          <n-space>
            <n-text strong style="font-size: 18px;">已安装版本</n-text>
          </n-space>
          <n-text depth="3">
            共 {{ installedVersions.length }} 个已安装版本
          </n-text>
        </n-space>
      </n-card>

      <!-- 版本列表 -->
      <n-spin :show="loading">
        <n-list hoverable clickable>
          <n-list-item v-for="version in installedVersions" :key="version.id">
            <n-thing>
              <template #header>
                <n-space align="center">
                  <n-text strong style="font-size: 16px;">{{ version.name }}</n-text>
                  <n-tag v-if="version.isPrimary" type="success" size="small">
                    主要
                  </n-tag>
                  <n-tag :type="getVersionTypeColor(version.versionType)" size="small">
                    {{ getVersionTypeText(version.versionType) }}
                  </n-tag>
                  <n-tag type="success" size="small">
                    已安装
                  </n-tag>
                </n-space>
              </template>

              <template #description>
                <n-space vertical size="small">
                  <n-text depth="3">
                    版本: {{ version.gameVersion }} - {{ version.subVersion }}
                  </n-text>
                  <n-text depth="3" v-if="version.localPath">
                    路径: {{ version.localPath }}
                  </n-text>
                </n-space>
              </template>

              <template #action>
                <n-space>
                  <n-button
                    type="success"
                    size="medium"
                    :disabled="gameStore.isRunning"
                    @click="handleLaunch(version)"
                  >
                    <template #icon>
                      <n-icon><PlayIcon /></n-icon>
                    </template>
                    启动
                  </n-button>
                  <n-button
                    size="medium"
                    @click="handleSetPrimary(version)"
                    :disabled="version.isPrimary"
                    :type="version.isPrimary ? 'success' : 'default'"
                  >
                    <template #icon>
                      <n-icon><StarIcon /></n-icon>
                    </template>
                    {{ version.isPrimary ? '已设为主要' : '设为主要' }}
                  </n-button>
                  <n-button
                    size="medium"
                    @click="handleRename(version)"
                  >
                    <template #icon>
                      <n-icon><EditIcon /></n-icon>
                    </template>
                    改名
                  </n-button>
                  <n-popconfirm
                    @positive-click="handleDelete(version)"
                  >
                    <template #trigger>
                      <n-button type="error" size="medium">
                        <template #icon>
                          <n-icon><TrashIcon /></n-icon>
                        </template>
                        删除
                      </n-button>
                    </template>
                    确定要删除版本"{{ version.name }}"吗？
                  </n-popconfirm>
                </n-space>
              </template>
            </n-thing>
          </n-list-item>
        </n-list>
        <n-empty v-if="installedVersions.length === 0 && !loading" description="暂无已安装版本">
          <template #extra>
            <n-button type="primary" @click="$router.push('/versions')">
              去下载版本
            </n-button>
          </template>
        </n-empty>
      </n-spin>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useVersionStore } from '../stores/version'
import { useGameStore } from '../stores/game'
import { useMessage, useDialog, NInput } from 'naive-ui'
import { Play as PlayIcon, Star as StarIcon, Trash as TrashIcon, CreateOutline as EditIcon } from '@vicons/ionicons5'
import type { Version } from '../types/version'

const versionStore = useVersionStore()
const gameStore = useGameStore()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const renamingVersion = ref<Version | null>(null)
const newName = ref('')

const installedVersions = computed(() => versionStore.installedVersions)

function getVersionTypeText(type: string): string {
  const types = {
    api: '插件版',
    net: '联机版',
    original: '原版'
  }
  return types[type as keyof typeof types] || type
}

function getVersionTypeColor(type: string): 'info' | 'success' | 'warning' | 'default' {
  switch (type) {
    case 'api': return 'info'
    case 'net': return 'warning'
    case 'original': return 'success'
    default: return 'default'
  }
}

async function handleLaunch(version: Version) {
  try {
    await gameStore.launchGame(version.id)
    message.success(`游戏 "${version.name}" 启动成功！`)
  } catch (error) {
    message.error('游戏启动失败：' + error)
  }
}

async function handleSetPrimary(version: Version) {
  try {
    await versionStore.setPrimaryVersion(version.id)
    message.success(`已将 "${version.name}" 设为主要版本`)
  } catch (error) {
    message.error('设置失败：' + error)
  }
}

function handleRename(version: Version) {
  renamingVersion.value = version
  newName.value = version.name

  dialog.create({
    title: '重命名版本',
    content: () => {
      return h('div', [
        h('div', { style: 'margin-bottom: 8px' }, '请输入新的版本名称：'),
        h(NInput, {
          value: newName.value,
          placeholder: '请输入版本名称',
          onUpdateValue: (value: string) => {
            newName.value = value
          },
          onKeyup: (e: KeyboardEvent) => {
            if (e.key === 'Enter') {
              // 按回车键确认
            }
          }
        })
      ])
    },
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      if (!newName.value.trim()) {
        message.error('版本名称不能为空')
        return false
      }

      // 检查重名
      const exists = versionStore.installedVersions.some(
        v => v.name === newName.value && v.id !== version.id
      )
      if (exists) {
        message.error('版本名称已存在，请使用其他名称')
        return false
      }

      try {
        await versionStore.renameVersion(version.id, newName.value)
        message.success(`版本 "${version.name}" 已重命名为 "${newName.value}"`)
        renamingVersion.value = null
        return true
      } catch (error) {
        message.error('重命名失败：' + error)
        return false
      }
    }
  })
}

function handleDelete(version: Version) {
  versionStore.deleteVersion(version.id)
    .then(() => {
      message.success(`版本 "${version.name}" 已删除`)
    })
    .catch((error) => {
      message.error('删除失败：' + error)
    })
}

onMounted(async () => {
  loading.value = true
  try {
    await versionStore.getVersions()
    await versionStore.getPrimaryVersion()
  } catch (error) {
    message.error('加载数据失败：' + error)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.installed-versions-view {
  max-width: 1000px;
  margin: 0 auto;
}
</style>
