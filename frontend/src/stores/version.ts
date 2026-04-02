import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { Version } from '../types/version'
import * as versionApi from '../api/version'

export const useVersionStore = defineStore('version', () => {
  // 状态
  const versions = ref<Version[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const currentVersion = ref<string | null>(null)
  const primaryVersion = ref<Version | null>(null)
  const downloading = ref<Set<string>>(new Set())
  const installing = ref<Set<string>>(new Set())

  // 计算属性
  const installedVersions = computed(() =>
    versions.value.filter(v => v.installed)
  )

  const versionsByType = computed(() => {
    const grouped: Record<string, Version[]> = {
      api: [] as Version[],
      net: [] as Version[],
      original: [] as Version[]
    }
    versions.value.forEach(v => {
      if (!grouped[v.versionType]) {
        grouped[v.versionType] = []
      }
      grouped[v.versionType].push(v)
    })
    return grouped
  })

  // 操作
  async function fetchVersions() {
    loading.value = true
    error.value = null
    try {
      versions.value = await versionApi.FetchVersions()
    } catch (e) {
      error.value = e as string
      console.error('Failed to fetch versions:', e)
    } finally {
      loading.value = false
    }
  }

  async function getVersions() {
    loading.value = true
    error.value = null
    try {
      versions.value = await versionApi.GetVersions()
    } catch (e) {
      error.value = e as string
      console.error('Failed to get versions:', e)
    } finally {
      loading.value = false
    }
  }

  async function downloadVersion(versionId: string) {
    downloading.value.add(versionId)
    try {
      await versionApi.DownloadVersion(versionId)
    } catch (e) {
      console.error('Failed to download version:', e)
      throw e
    }
  }

  async function downloadVersionWithCustomName(versionId: string, customName: string) {
    downloading.value.add(versionId)
    try {
      await versionApi.DownloadVersionWithCustomName(versionId, customName)
    } catch (e) {
      console.error('Failed to download version:', e)
      throw e
    }
  }

  function finishDownload(versionId: string) {
    downloading.value.delete(versionId)
  }

  async function installVersion(versionId: string) {
    installing.value.add(versionId)
    try {
      await versionApi.InstallVersion(versionId)
      // 更新版本状态
      const version = versions.value.find(v => v.id === versionId)
      if (version) {
        version.installed = true
      }
      // 刷新主要版本（可能自动设置了新的主要版本）
      await getPrimaryVersion()
    } catch (e) {
      console.error('Failed to install version:', e)
      throw e
    } finally {
      installing.value.delete(versionId)
    }
  }

  async function deleteVersion(versionId: string) {
    try {
      await versionApi.DeleteVersion(versionId)
      // 更新版本状态
      const version = versions.value.find(v => v.id === versionId)
      if (version) {
        version.installed = false
      }
      // 刷新主要版本（可能重新选择了新的主要版本）
      await getPrimaryVersion()
    } catch (e) {
      console.error('Failed to delete version:', e)
      throw e
    }
  }

  async function renameVersion(versionId: string, newName: string) {
    try {
      await versionApi.RenameVersion(versionId, newName)
      // 更新版本名称
      const version = versions.value.find(v => v.id === versionId)
      if (version) {
        version.name = newName
      }
      // 更新主要版本名称
      if (primaryVersion.value && primaryVersion.value.id === versionId) {
        primaryVersion.value.name = newName
      }
    } catch (e) {
      console.error('Failed to rename version:', e)
      throw e
    }
  }

  async function setCurrentVersion(versionId: string) {
    try {
      await versionApi.SetCurrentVersion(versionId)
      currentVersion.value = versionId
    } catch (e) {
      console.error('Failed to set current version:', e)
      throw e
    }
  }

  async function getPrimaryVersion() {
    try {
      primaryVersion.value = await versionApi.GetPrimaryVersion()
      // 更新版本列表中的主要标记
      versions.value.forEach(v => {
        v.isPrimary = primaryVersion.value !== null && v.id === primaryVersion.value.id
      })
    } catch (e) {
      console.error('Failed to get primary version:', e)
    }
  }

  async function setPrimaryVersion(versionId: string) {
    try {
      await versionApi.SetPrimaryVersion(versionId)
      // 更新本地状态
      await getPrimaryVersion()
      // 更新版本列表中的主要标记
      versions.value.forEach(v => {
        v.isPrimary = v.id === versionId
      })
    } catch (e) {
      console.error('Failed to set primary version:', e)
      throw e
    }
  }

  return {
    versions,
    loading,
    error,
    currentVersion,
    primaryVersion,
    downloading,
    installing,
    installedVersions,
    versionsByType,
    fetchVersions,
    getVersions,
    downloadVersion,
    downloadVersionWithCustomName,
    finishDownload,
    installVersion,
    deleteVersion,
    renameVersion,
    setCurrentVersion,
    getPrimaryVersion,
    setPrimaryVersion
  }
})
