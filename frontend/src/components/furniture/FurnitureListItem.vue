<template>
  <n-list-item>
    <n-thing>
      <template #header>
        <n-text strong style="font-size: 16px;">{{ furniture.name }}</n-text>
      </template>

      <template #description>
        <n-space vertical size="small">
          <n-text depth="3">
            {{ t('furniture.fileName') }}: {{ furniture.fileName }}
          </n-text>
        </n-space>
      </template>

      <template #action>
        <n-space>
          <n-button
            size="medium"
            @click="$emit('rename', furniture)"
          >
            <template #icon>
              <n-icon><EditIcon /></n-icon>
            </template>
            {{ t('common.rename') }}
          </n-button>
          <n-popconfirm @positive-click="$emit('delete', furniture)">
            <template #trigger>
              <n-button type="error" size="medium">
                <template #icon>
                  <n-icon><TrashIcon /></n-icon>
                </template>
                {{ t('common.delete') }}
              </n-button>
            </template>
            {{ t('furniture.confirmDelete', { name: furniture.name }) }}
          </n-popconfirm>
        </n-space>
      </template>
    </n-thing>
  </n-list-item>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Trash as TrashIcon, CreateOutline as EditIcon } from '@vicons/ionicons5'
import type { Furniture } from '../../types/furniture'

defineProps<{
  furniture: Furniture
}>()

defineEmits<{
  rename: [furniture: Furniture]
  delete: [furniture: Furniture]
}>()

const { t } = useI18n()
</script>
