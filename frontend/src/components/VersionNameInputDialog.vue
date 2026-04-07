<template>
  <div ref="dialogContainer"></div>
</template>

<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { useDialog, NInput } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import type { Version } from '../types/version'

const { t } = useI18n()
const dialog = useDialog()

const props = defineProps<{
  defaultName: string
  existingVersions: Version[]
}>()

const emit = defineEmits<{
  confirm: [name: string]
  cancel: []
}>()

const dialogContainer = ref<HTMLElement | null>(null)

function checkDuplicate(inputName: string): boolean {
  const trimmed = inputName.trim()
  if (!trimmed) return false
  return props.existingVersions.some(v => v.name === trimmed)
}

function show(): Promise<string | null> {
  return new Promise((resolve) => {
    let name = props.defaultName
    let errorMessage = ''

    const d = dialog.create({
      title: t('installed.enterVersionName'),
      content: () => {
        return h('div', [
          h('p', { style: 'margin-bottom: 12px;' }, t('installed.enterVersionNameDesc')),
          h(NInput, {
            placeholder: props.defaultName,
            defaultValue: props.defaultName,
            status: errorMessage ? 'error' : undefined,
            onUpdateValue: (value: string) => {
              name = value
              if (checkDuplicate(value)) {
                errorMessage = t('installed.nameAlreadyExists')
              } else {
                errorMessage = ''
              }
            },
            onKeyup: (e: KeyboardEvent) => {
              if (e.key === 'Enter') {
                if (checkDuplicate(name)) {
                  errorMessage = t('installed.nameAlreadyExists')
                } else {
                  resolve(name.trim() || null)
                }
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
      onPositiveClick: () => {
        if (checkDuplicate(name)) {
          errorMessage = t('installed.nameAlreadyExists')
        } else {
          resolve(name.trim() || null)
        }
      },
      onNegativeClick: () => {
        resolve(null)
      }
    })
  })
}

onMounted(() => {
  show().then(result => {
    if (result) {
      emit('confirm', result)
    } else {
      emit('cancel')
    }
  })
})

defineExpose({
  show
})
</script>
