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

      <!-- 背景特效（设置背景图片后可用，调整立即生效） -->
      <template v-if="hasBackground">
        <n-divider title-placement="left" style="margin: 4px 0;">
          {{ t('settings.backgroundEffects') }}
        </n-divider>
        <n-form-item :label="t('settings.backgroundBlur')" :show-feedback="false">
          <n-slider
            v-model:value="blur"
            :min="0"
            :max="40"
            :step="1"
            :format-tooltip="formatPx"
          />
        </n-form-item>
        <n-form-item :label="t('settings.backgroundDim')" :show-feedback="false">
          <n-slider
            v-model:value="dim"
            :min="0"
            :max="100"
            :step="5"
            :format-tooltip="formatPercent"
          />
        </n-form-item>
        <n-form-item :label="t('settings.backgroundSurfaceOpacity')" :show-feedback="false">
          <n-slider
            v-model:value="surfaceOpacity"
            :min="50"
            :max="100"
            :step="1"
            :format-tooltip="formatPercent"
          />
        </n-form-item>
        <n-text depth="3" style="font-size: 12px;">{{ t('settings.backgroundEffectsHint') }}</n-text>
      </template>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue'
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

// 背景特效状态与更新入口由 App.vue provide（拖动滑块即时预览，停止后由 App 防抖持久化）
const effects = inject<{ blur: number; dim: number; surfaceOpacity: number } | null>('backgroundEffects', null)
const updateEffects = inject<
  (patch: Partial<{ blur: number; dim: number; surfaceOpacity: number }>) => void
>('updateBackgroundEffects', () => {})

const blur = computed({
  get: () => effects?.blur ?? 0,
  set: (value: number) => updateEffects({ blur: value })
})
const dim = computed({
  get: () => effects?.dim ?? 40,
  set: (value: number) => updateEffects({ dim: value })
})
const surfaceOpacity = computed({
  get: () => effects?.surfaceOpacity ?? 85,
  set: (value: number) => updateEffects({ surfaceOpacity: value })
})

function formatPx(value: number): string {
  return `${value}px`
}

function formatPercent(value: number): string {
  return `${value}%`
}
</script>

<style scoped>
.background-preview {
  width: 100%;
  border: 1px solid var(--color-border, #e0e0e0);
  border-radius: 8px;
  overflow: hidden;
  background-color: var(--color-surface-elevated, #f5f5f7);
}
</style>
