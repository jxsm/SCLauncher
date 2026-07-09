<template>
  <n-card :title="t('runtime.title')">
    <n-form-item :label="t('runtime.autoCheckLabel')">
      <n-switch :value="enabled" :loading="loading" @update:value="handleToggle" />
    </n-form-item>
    <n-text depth="3" style="font-size: 13px;">
      {{ t('runtime.autoCheckDesc') }}
    </n-text>
  </n-card>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import * as runtimeApi from '../../api/runtime'
import * as AppBindings from '../../../wailsjs/go/main/App'

const { t } = useI18n()
const message = useMessage()

const enabled = ref(true)
const loading = ref(false)

onMounted(async () => {
  try {
    const cfg: any = await AppBindings.GetConfig()
    enabled.value = !!cfg?.autoCheckRuntime
  } catch (e) {
    console.error('load autoCheckRuntime failed', e)
  }
})

async function handleToggle(val: boolean) {
  loading.value = true
  try {
    await runtimeApi.SetAutoCheckRuntime(val)
    enabled.value = val
  } catch (e) {
    message.error(t('runtime.failed'))
    console.error('set autoCheckRuntime failed', e)
  } finally {
    loading.value = false
  }
}
</script>
