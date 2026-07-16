import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Mod } from '../types/mod'
import * as modApi from '../api/mod'

export const useModStore = defineStore('mod', () => {
  // 状态
  const mods = ref<Mod[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function loadMods(versionId: string) {
    loading.value = true
    error.value = null
    try {
      mods.value = await modApi.GetMods(versionId)
    } catch (e) {
      error.value = e as string
      console.error('Failed to load mods:', e)
    } finally {
      loading.value = false
    }
  }

  async function importMod(versionId: string, filePath: string) {
    try {
      await modApi.ImportMod(versionId, filePath)
      await loadMods(versionId)
    } catch (e) {
      error.value = e as string
      console.error('Failed to import mod:', e)
      throw e
    }
  }

  async function toggleMod(versionId: string, modId: string, enabled: boolean) {
    try {
      await modApi.ToggleMod(versionId, modId, enabled)
      // 切换只会改 enabled 一个字段，就地更新即可。
      // 不调用 loadMods 整表重载：那会把 loading 置 true，导致列表被替换为转圈占位、
      // 页面滚动位置丢失（模组管理页切换启用状态时“瞬间回到顶部”的根因）。
      const target = mods.value.find(m => m.id === modId)
      if (target) {
        target.enabled = enabled
      }
    } catch (e) {
      error.value = e as string
      console.error('Failed to toggle mod:', e)
      throw e
    }
  }

  async function deleteMod(versionId: string, modId: string) {
    try {
      await modApi.DeleteMod(versionId, modId)
      // 后端删除成功后就地移除该项，避免整表重载触发 loading（同 toggleMod，
      // 防止列表被替换为转圈占位导致滚动位置丢失）。
      mods.value = mods.value.filter(m => m.id !== modId)
    } catch (e) {
      error.value = e as string
      console.error('Failed to delete mod:', e)
      throw e
    }
  }

  return {
    mods,
    loading,
    error,
    loadMods,
    importMod,
    toggleMod,
    deleteMod
  }
})
