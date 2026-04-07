<template>
  <n-modal :show="visible" @update:show="$emit('update:visible', $event)" preset="dialog" :title="t('settings.aboutSCLauncher')">
    <n-space vertical>
      <n-descriptions :column="1" bordered label-placement="left" label-style="width: 80px;">
        <n-descriptions-item :label="t('common.version')">
          v{{ appInfo.version }}
        </n-descriptions-item>
        <n-descriptions-item :label="t('settings.author')">
          {{ appInfo.repoOwner }}
        </n-descriptions-item>
        <n-descriptions-item :label="t('settings.license')">
          MIT License
        </n-descriptions-item>
      </n-descriptions>
      <n-divider />
      <n-text>
        {{ t('settings.description') }}
      </n-text>
      <n-button type="primary" block @click="$emit('openGitHub')">
        <template #icon>
          <n-icon><GithubIcon /></n-icon>
        </template>
        {{ t('settings.viewOnGitHub') }}
      </n-button>
      <n-button block @click="$emit('checkUpdate')">
        <template #icon>
          <n-icon><UpdateIcon /></n-icon>
        </template>
        {{ t('settings.checkUpdate') }}
      </n-button>
    </n-space>
    <template #action>
      <n-button @click="$emit('update:visible', false)">{{ t('common.close') }}</n-button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { LogoGithub as GithubIcon, RefreshOutline as UpdateIcon } from '@vicons/ionicons5'

const { t } = useI18n()

defineProps<{
  visible: boolean
  appInfo: {
    version: string
    repoOwner: string
    repoName: string
  }
}>()

defineEmits<{
  'update:visible': [value: boolean]
  openGitHub: []
  checkUpdate: []
}>()
</script>
