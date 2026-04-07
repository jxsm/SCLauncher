<template>
  <n-modal
    :show="visible"
    @update:show="handleVisibleChange"
    preset="dialog"
    :title="t('versions.enterVersionName')"
    :positive-text="t('common.confirm')"
    :negative-text="t('common.cancel')"
    @positive-click="handleConfirm"
  >
    <n-space vertical>
      <n-text>{{ t('versions.enterVersionNameDesc') }}</n-text>
      <n-input
        :placeholder="defaultName"
        :value="inputName"
        @update:value="handleInputChange"
        @keyup="handleKeyup"
        :status="errorMessage ? 'error' : undefined"
      />
      <n-text v-if="errorMessage" type="error" style="font-size: 12px">
        {{ errorMessage }}
      </n-text>
    </n-space>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  visible: boolean
  defaultName: string
  existingNames: string[]
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'confirm': [name: string]
}>()

const { t } = useI18n()

const inputName = ref('')
const errorMessage = ref('')

// Reset input when dialog opens
watch(() => props.visible, (newValue) => {
  if (newValue) {
    inputName.value = props.defaultName
    errorMessage.value = ''
  }
})

function checkDuplicate(name: string): boolean {
  const trimmed = name.trim()
  if (!trimmed) return false
  return props.existingNames.some(n => n === trimmed)
}

function handleInputChange(value: string) {
  inputName.value = value
  if (checkDuplicate(value)) {
    errorMessage.value = t('versions.nameAlreadyExists')
  } else {
    errorMessage.value = ''
  }
}

function handleKeyup(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    handleConfirm()
  }
}

function handleConfirm() {
  if (checkDuplicate(inputName.value)) {
    errorMessage.value = t('versions.nameAlreadyExists')
    return
  }

  const name = inputName.value.trim() || props.defaultName
  emit('confirm', name)
  emit('update:visible', false)
}

function handleVisibleChange(value: boolean) {
  emit('update:visible', value)
}
</script>
