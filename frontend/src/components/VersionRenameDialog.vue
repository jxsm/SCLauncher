<template>
  <div ref="dialogContainer"></div>
</template>

<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { useDialog, useMessage, NInput } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import type { Version } from '../types/version'

const { t } = useI18n()
const dialog = useDialog()
const message = useMessage()

const props = defineProps<{
  version: Version
  existingVersions: Version[]
}>()

const emit = defineEmits<{
  confirm: [newName: string]
  cancel: []
}>()

const dialogContainer = ref<HTMLElement | null>(null)

function checkDuplicate(inputName: string): boolean {
  const trimmed = inputName.trim()
  if (!trimmed) return false
  return props.existingVersions.some(v =>
    v.name === trimmed && v.id !== props.version.id
  )
}

function show() {
  let newName = props.version.name
  let errorMessage = ''

  const d = dialog.create({
    title: t('installed.renameVersion'),
    content: () => {
      return h('div', [
        h('div', { style: 'margin-bottom: 8px' }, t('installed.enterNewVersionName')),
        h(NInput, {
          value: newName,
          placeholder: t('installed.enterVersionName'),
          status: errorMessage ? 'error' : undefined,
          onUpdateValue: (value: string) => {
            newName = value
            if (checkDuplicate(value)) {
              errorMessage = t('installed.nameAlreadyExists')
            } else {
              errorMessage = ''
            }
          },
          onKeyup: (e: KeyboardEvent) => {
            if (e.key === 'Enter') {
              handleConfirm()
            }
          }
        }),
        errorMessage ? h('p', {
          style: 'margin-top: 8px; color: #f56c6c; font-size: 12px;'
        }, errorMessage) : null
      ])
    },
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: handleConfirm,
    onNegativeClick: () => {
      emit('cancel')
    }
  })

  function handleConfirm() {
    if (!newName.trim()) {
      message.error(t('installed.nameCannotBeEmpty'))
      return false
    }

    if (checkDuplicate(newName)) {
      errorMessage = t('installed.nameAlreadyExists')
      return false
    }

    emit('confirm', newName.trim())
    return true
  }
}

onMounted(() => {
  show()
})

defineExpose({
  show
})
</script>
