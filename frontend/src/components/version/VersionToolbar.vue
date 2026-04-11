<template>
  <n-card>
    <n-space justify="space-between" :vertical="false">
      <n-space vertical :size="12">
        <n-space>
          <n-button type="primary" @click="handleRefresh" :loading="loading">
            <template #icon>
              <n-icon><RefreshIcon /></n-icon>
            </template>
            {{ t('versions.refresh') }}
          </n-button>
          <n-select
            :value="filterType"
            @update:value="handleFilterChange"
            :options="typeOptions"
            style="width: 150px"
          />
        </n-space>
        <!-- Extra content slot -->
        <slot name="extra" />
      </n-space>
      <n-text depth="3">
        {{ t('versions.totalVersions') }} {{ totalVersions }}
      </n-text>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Refresh as RefreshIcon } from '@vicons/ionicons5'

interface TypeOption {
  label: string
  value: string
}

defineProps<{
  loading: boolean
  filterType: string
  typeOptions: TypeOption[]
  totalVersions: number
}>()

const emit = defineEmits<{
  'refresh': []
  'update:filterType': [value: string]
}>()

const { t } = useI18n()

function handleRefresh() {
  emit('refresh')
}

function handleFilterChange(value: string) {
  emit('update:filterType', value)
}
</script>
