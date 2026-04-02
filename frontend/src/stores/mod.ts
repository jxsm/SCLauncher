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
      await loadMods(versionId)
    } catch (e) {
      error.value = e as string
      console.error('Failed to toggle mod:', e)
      throw e
    }
  }

  async function deleteMod(versionId: string, modId: string) {
    try {
      await modApi.DeleteMod(versionId, modId)
      await loadMods(versionId)
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
