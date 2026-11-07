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
    // 默认名可能已与已装版本重名（如重复下载同一版本），打开时就提示，
    // 而不是等用户点确认后弹窗一闪而过地关闭
    errorMessage.value = checkDuplicate(effectiveName())
      ? t('versions.nameAlreadyExists')
      : ''
  }
})

// 实际生效的名字：输入为空时回退到默认名
function effectiveName(): string {
  return inputName.value.trim() || props.defaultName
}

function checkDuplicate(name: string): boolean {
  const trimmed = name.trim()
  if (!trimmed) return false
  return props.existingNames.some(n => n === trimmed)
}

function handleInputChange(value: string) {
  inputName.value = value
  // 校验实际生效的名字（清空输入会回退默认名，同样可能重名）
  errorMessage.value = checkDuplicate(effectiveName())
    ? t('versions.nameAlreadyExists')
    : ''
}

function handleKeyup(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    handleConfirm()
  }
}

function handleConfirm(): boolean {
  const name = effectiveName()
  if (checkDuplicate(name)) {
    errorMessage.value = t('versions.nameAlreadyExists')
    // naive-ui 对话框预设：确认回调返回 false 才不自动关闭弹窗，
    // 这里重名时保持弹窗打开，等待用户重新输入
    return false
  }

  emit('confirm', name)
  emit('update:visible', false)
  return true
}

function handleVisibleChange(value: boolean) {
  emit('update:visible', value)
}
</script>
