<template>
  <n-modal :show="visible" @update:show="$emit('update:visible', $event)" preset="dialog" :title="t('mods.addSource')">
    <n-form ref="formRef" :model="newSource" :rules="rules" label-placement="left" label-width="100px">
      <n-form-item :label="t('mods.sourceType')" path="type">
        <n-select
          :value="newSource.type"
          :options="typeOptions"
          @update:value="handleUpdateType"
        />
      </n-form-item>
      <n-form-item :label="t('mods.sourceName')" path="name">
        <n-input
          :value="newSource.name"
          :placeholder="t('mods.sourceNamePlaceholder')"
          @update:value="handleUpdateName"
        />
      </n-form-item>
      <n-form-item :label="t('mods.sourceDescription')" path="description">
        <n-input
          :value="newSource.description"
          :placeholder="t('mods.sourceDescriptionPlaceholder')"
          @update:value="handleUpdateDescription"
        />
      </n-form-item>
      <n-form-item :label="t('mods.sourceApiUrl')" path="apiUrl">
        <n-input
          :value="newSource.apiUrl"
          :placeholder="t('mods.sourceApiUrlPlaceholder')"
          @update:value="handleUpdateApiUrl"
        />
      </n-form-item>
    </n-form>
    <template #action>
      <n-button @click="$emit('update:visible', false)">{{ t('common.cancel') }}</n-button>
      <n-button type="primary" @click="$emit('confirm', newSource)">{{ t('common.confirm') }}</n-button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  visible: boolean
}>()

const newSource = ref({
  name: '',
  description: '',
  apiUrl: '',
  type: 'mods' as 'mods' | 'savegames' | 'furniture' | 'textures' | 'skins'
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  confirm: [source: typeof newSource.value]
}>()

const typeOptions = computed(() => [
  { label: t('mods.modSources'), value: 'mods' },
  { label: t('mods.savegameSources'), value: 'savegames' },
  { label: t('mods.furnitureSources'), value: 'furniture' },
  { label: t('mods.textureSources'), value: 'textures' },
  { label: t('mods.skinSources'), value: 'skins' }
])

const rules = {
  name: {
    required: true,
    message: t('mods.sourceNameRequired'),
    trigger: 'blur'
  },
  description: {
    required: true,
    message: t('mods.sourceDescriptionRequired'),
    trigger: 'blur'
  },
  apiUrl: {
    required: true,
    message: t('mods.sourceApiUrlRequired'),
    trigger: 'blur'
  }
}

function handleUpdateType(value: 'mods' | 'savegames' | 'furniture' | 'textures' | 'skins') {
  newSource.value.type = value
}

function handleUpdateName(value: string) {
  newSource.value.name = value
}

function handleUpdateDescription(value: string) {
  newSource.value.description = value
}

function handleUpdateApiUrl(value: string) {
  newSource.value.apiUrl = value
}
</script>
