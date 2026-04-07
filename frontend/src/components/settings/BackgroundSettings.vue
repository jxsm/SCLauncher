<template>
  <n-card :title="t('settings.background')">
    <n-space vertical>
      <n-form-item :label="t('settings.backgroundImage')">
        <n-space>
          <n-button @click="$emit('select')">
            <template #icon>
              <n-icon><ImageIcon /></n-icon>
            </template>
            {{ t('settings.selectImage') }}
          </n-button>
          <n-button v-if="hasBackground" type="error" @click="$emit('clear')">
            <template #icon>
              <n-icon><TrashIcon /></n-icon>
            </template>
            {{ t('settings.clearBackground') }}
          </n-button>
        </n-space>
      </n-form-item>

      <!-- 背景预览 -->
      <div v-if="preview" class="background-preview">
        <n-image
          :src="preview"
          object-fit="cover"
          style="width: 100%; height: 200px; border-radius: 4px;"
        />
      </div>
      <n-text v-else depth="3">{{ t('settings.noBackground') }}</n-text>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ImageOutline as ImageIcon, TrashOutline as TrashIcon } from '@vicons/ionicons5'

const { t } = useI18n()

defineProps<{
  hasBackground?: boolean
  preview?: string
}>()

defineEmits<{
  select: []
  clear: []
}>()
</script>

<style scoped>
.background-preview {
  width: 100%;
  border: 1px solid #e0e0e0;
  border-radius: 4px;
  overflow: hidden;
  background-color: #f5f5f5;
}
</style>
