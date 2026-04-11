<template>
  <n-modal
    :show="visible"
    @update:show="$emit('update:visible', $event)"
    preset="dialog"
    :title="t('settings.addManifestSource')"
  >
    <n-form
      ref="formRef"
      :model="newSource"
      :rules="rules"
      label-placement="left"
      label-width="100px"
    >
      <n-form-item :label="t('settings.manifestSourceName')" path="name">
        <n-input
          :value="newSource.name"
          :placeholder="t('settings.manifestSourceNamePlaceholder')"
          @update:value="handleUpdateName"
        />
      </n-form-item>
      <n-form-item :label="t('settings.manifestSourceUrl')" path="url">
        <n-input
          :value="newSource.url"
          :placeholder="t('settings.manifestSourceUrlPlaceholder')"
          @update:value="handleUpdateUrl"
        />
      </n-form-item>
    </n-form>
    <template #action>
      <n-button @click="$emit('update:visible', false)">{{ t('common.cancel') }}</n-button>
      <n-button type="primary" @click="handleConfirm">{{ t('common.confirm') }}</n-button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  visible: boolean
}>()

const newSource = ref({
  name: '',
  url: ''
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  confirm: [source: typeof newSource.value]
}>()

const rules = {
  name: {
    required: true,
    message: t('settings.manifestSourceNameRequired'),
    trigger: 'blur'
  },
  url: {
    required: true,
    message: t('settings.manifestSourceUrlRequired'),
    trigger: 'blur'
  }
}

function handleUpdateName(value: string) {
  newSource.value.name = value
}

function handleUpdateUrl(value: string) {
  newSource.value.url = value
}

function handleConfirm() {
  if (!newSource.value.name || !newSource.value.url) {
    return
  }
  emit('confirm', { ...newSource.value })
  // 重置表单
  newSource.value = {
    name: '',
    url: ''
  }
}
</script>
