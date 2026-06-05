<template>
  <n-card :title="t('settings.themeSettings')">
    <n-form-item :label="t('settings.theme')">
      <n-radio-group :value="themeMode" @update:value="handleUpdateTheme">
        <n-space>
          <n-radio value="auto">
            {{ t('settings.themeAuto') }}
          </n-radio>
          <n-radio value="light">
            {{ t('settings.themeLight') }}
          </n-radio>
          <n-radio value="dark">
            {{ t('settings.themeDark') }}
          </n-radio>
        </n-space>
      </n-radio-group>
    </n-form-item>
    <n-text depth="3" style="font-size: 13px;">
      {{ t('settings.themeDescription') }}
    </n-text>
  </n-card>
</template>

<script setup lang="ts">
import { inject, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const themeMode = inject<Ref<'light' | 'dark' | 'auto'>>('themeMode')
const setTheme = inject<(mode: 'light' | 'dark' | 'auto') => Promise<void>>('setTheme')

function handleUpdateTheme(value: string) {
  if (setTheme && themeMode) {
    setTheme(value as 'light' | 'dark' | 'auto')
  }
}
</script>
