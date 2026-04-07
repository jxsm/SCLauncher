<template>
  <n-card :title="t('settings.languageSettings')">
    <n-form-item :label="t('settings.language')">
      <n-select
        :value="language"
        :options="languageOptions"
        @update:value="handleUpdateLanguage"
      />
    </n-form-item>
  </n-card>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t, locale } = useI18n()

defineProps<{
  language: string
}>()

const emit = defineEmits<{
  'update:language': [value: string]
}>()

const languageOptions = [
  { label: '简体中文', value: 'zh-CN' },
  { label: 'English', value: 'en-US' },
  { label: 'Русский', value: 'ru-RU' },
  { label: 'Português (Brasil)', value: 'pt-BR' },
  { label: 'हिन्दी', value: 'hi-IN' },
  { label: 'Bahasa Indonesia', value: 'id-ID' },
  { label: 'العربية', value: 'ar-SA' },
  { label: 'Español (España)', value: 'es-ES' }
]

function handleUpdateLanguage(value: string) {
  emit('update:language', value)
}
</script>
