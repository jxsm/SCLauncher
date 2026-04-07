<template>
  <n-card :title="t('settings.manifest')">
    <n-form-item :label="t('settings.manifestUrl')">
      <n-input
        :value="manifestUrl"
        :placeholder="t('settings.manifestUrlPlaceholder') || '请输入清单文件 URL'"
        @update:value="handleUpdateUrl"
      />
    </n-form-item>
    <n-space>
      <n-button type="primary" @click="handleSave">
        {{ t('settings.saveManifestUrl') }}
      </n-button>
      <n-button @click="handleReset">
        {{ t('settings.resetManifestUrl') }}
      </n-button>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  manifestUrl: string
}>()

const emit = defineEmits<{
  'update:manifestUrl': [value: string]
  save: []
  reset: []
}>()

function handleUpdateUrl(value: string) {
  emit('update:manifestUrl', value)
}

function handleSave() {
  emit('save')
}

function handleReset() {
  emit('reset')
}
</script>
