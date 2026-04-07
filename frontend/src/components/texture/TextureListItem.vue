<template>
  <n-list-item>
    <n-thing>
      <template #header>
        <n-text strong style="font-size: 16px;">{{ texture.name }}</n-text>
      </template>

      <template #description>
        <n-space vertical size="small">
          <n-text depth="3">
            {{ t('textures.fileName') }}: {{ texture.fileName }}
          </n-text>
        </n-space>
      </template>

      <template #action>
        <n-space>
          <n-button
            size="medium"
            @click="$emit('rename', texture)"
          >
            <template #icon>
              <n-icon><EditIcon /></n-icon>
            </template>
            {{ t('common.rename') }}
          </n-button>
          <n-popconfirm @positive-click="$emit('delete', texture)">
            <template #trigger>
              <n-button type="error" size="medium">
                <template #icon>
                  <n-icon><TrashIcon /></n-icon>
                </template>
                {{ t('common.delete') }}
              </n-button>
            </template>
            {{ t('textures.confirmDelete', { name: texture.name }) }}
          </n-popconfirm>
        </n-space>
      </template>
    </n-thing>
  </n-list-item>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Trash as TrashIcon, CreateOutline as EditIcon } from '@vicons/ionicons5'
import type { Texture } from '../../types/texture'

defineProps<{
  texture: Texture
}>()

defineEmits<{
  rename: [texture: Texture]
  delete: [texture: Texture]
}>()

const { t } = useI18n()
</script>
