import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Skin } from '../types/skin'
import * as skinApi from '../api/skin'

export const useSkinStore = defineStore('skin', () => {
  // 状态
  const skins = ref<Skin[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function loadSkins() {
    loading.value = true
    error.value = null
    try {
      const result = await skinApi.GetSkins()
      // 确保返回的是数组，即使后端返回 null 或 undefined
      skins.value = Array.isArray(result) ? result : []
    } catch (e) {
      error.value = e as string
      console.error('Failed to load skins:', e)
      // 发生错误时也要设置为空数组，避免渲染问题
      skins.value = []
    } finally {
      loading.value = false
    }
  }

  async function importSkin(filePath: string) {
    try {
      await skinApi.ImportSkin(filePath)
      await loadSkins()
    } catch (e) {
      error.value = e as string
      console.error('Failed to import skin:', e)
      throw e
    }
  }

  async function deleteSkin(fileName: string) {
    try {
      await skinApi.DeleteSkin(fileName)
      await loadSkins()
    } catch (e) {
      error.value = e as string
      console.error('Failed to delete skin:', e)
      throw e
    }
  }

  return {
    skins,
    loading,
    error,
    loadSkins,
    importSkin,
    deleteSkin
  }
})
